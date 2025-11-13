package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"orders-api/internal/models"
)

// ExecutionService servicio con procesamiento concurrente usando goroutines, channels y WaitGroup
type ExecutionService struct {
	userClient        UserClient
	userBalanceClient UserBalanceClient
	marketClient      MarketClient
	portfolioClient   PortfolioClient
}

// UserClient interface para validar usuarios
type UserClient interface {
	VerifyUser(ctx context.Context, userID int) (*models.ValidationResult, error)
}

// UserBalanceClient interface para verificar saldos
type UserBalanceClient interface {
	CheckBalance(ctx context.Context, userID int, amount decimal.Decimal, userToken string) (*models.BalanceResult, error)
	ProcessTransaction(ctx context.Context, userID int, amount decimal.Decimal, transactionType, orderID, description string) (string, error)
}

// MarketClient interface para obtener precios
type MarketClient interface {
	GetCurrentPrice(ctx context.Context, symbol string) (*models.PriceResult, error)
}

// PortfolioClient interface para actualizar holdings
type PortfolioClient interface {
	UpdateHoldings(ctx context.Context, userID int64, symbol string, quantity, price decimal.Decimal, orderType string) error
}

// Resultados de cálculos concurrentes
type validationResult struct {
	userValid *models.ValidationResult
	userErr   error
}

type priceResult struct {
	price *models.PriceResult
	err   error
}

type balanceResult struct {
	balance *models.BalanceResult
	err     error
}

type feeResult struct {
	fee        decimal.Decimal
	totalAmount decimal.Decimal
}

// NewExecutionService crea una nueva instancia del servicio de ejecución
func NewExecutionService(
	userClient UserClient,
	userBalanceClient UserBalanceClient,
	marketClient MarketClient,
	feeCalculator interface{}, // No se usa, pero lo dejamos para compatibilidad con main.go
) *ExecutionService {
	return &ExecutionService{
		userClient:        userClient,
		userBalanceClient: userBalanceClient,
		marketClient:      marketClient,
		portfolioClient:   nil, // Will be set later
	}
}

// SetPortfolioClient sets the portfolio client
func (s *ExecutionService) SetPortfolioClient(pc PortfolioClient) {
	s.portfolioClient = pc
}

// ExecuteOrder ejecuta una orden usando procesamiento concurrente con goroutines, channels y WaitGroup
func (s *ExecutionService) ExecuteOrder(ctx context.Context, order *models.Order) (*models.ExecutionResult, error) {
	start := time.Now()

	// Channels para comunicación entre goroutines
	userValidationChan := make(chan validationResult, 1)
	priceChan := make(chan priceResult, 1)
	balanceChan := make(chan balanceResult, 1)
	feeChan := make(chan feeResult, 1)

	// WaitGroup para sincronizar todas las goroutines
	var wg sync.WaitGroup

	// Get user token from context
	userToken := ""
	if token := ctx.Value("user_token"); token != nil {
		userToken = token.(string)
	}

	// Goroutine 1: Validar usuario
	wg.Add(1)
	go func() {
		defer wg.Done()
		userValid, err := s.userClient.VerifyUser(ctx, order.UserID)
		userValidationChan <- validationResult{userValid: userValid, userErr: err}
	}()

	// Goroutine 2: Obtener precio de mercado
	wg.Add(1)
	go func() {
		defer wg.Done()
		price, err := s.marketClient.GetCurrentPrice(ctx, order.CryptoSymbol)
		priceChan <- priceResult{price: price, err: err}
	}()

	// Esperar a que terminen las validaciones básicas antes de calcular balance
	wg.Wait()

	// Leer resultados de las goroutines
	userValidation := <-userValidationChan
	if userValidation.userErr != nil {
		return nil, fmt.Errorf("user validation failed: %w", userValidation.userErr)
	}
	if userValidation.userValid != nil && !userValidation.userValid.IsValid {
		return nil, fmt.Errorf("user validation failed: %s", userValidation.userValid.Message)
	}

	priceRes := <-priceChan
	if priceRes.err != nil {
		return nil, fmt.Errorf("failed to get market price: %w", priceRes.err)
	}
	if priceRes.price == nil {
		return nil, fmt.Errorf("price result is nil")
	}

	// Goroutine 3: Calcular fee y total amount (cálculo local, rápido)
	wg.Add(1)
	go func() {
		defer wg.Done()
		totalAmount := order.Quantity.Mul(priceRes.price.MarketPrice)
		fee := totalAmount.Mul(decimal.NewFromFloat(0.001)) // 0.1%
		minFee := decimal.NewFromFloat(0.01)
		if fee.LessThan(minFee) {
			fee = minFee
		}
		feeChan <- feeResult{fee: fee, totalAmount: totalAmount}
	}()

	// Esperar cálculo de fee
	wg.Wait()
	feeRes := <-feeChan

	// Goroutine 4: Verificar balance (solo para órdenes de compra)
	if order.Type == models.OrderTypeBuy {
		wg.Add(1)
		go func() {
			defer wg.Done()
			requiredAmount := feeRes.totalAmount.Add(feeRes.fee)
			balance, err := s.userBalanceClient.CheckBalance(ctx, order.UserID, requiredAmount, userToken)
			balanceChan <- balanceResult{balance: balance, err: err}
		}()
		wg.Wait()

		balanceRes := <-balanceChan
		if balanceRes.err != nil {
			return nil, fmt.Errorf("balance check failed: %w", balanceRes.err)
		}
		if balanceRes.balance == nil || !balanceRes.balance.HasSufficient {
			return nil, fmt.Errorf("insufficient balance: required %s, available %s",
				feeRes.totalAmount.Add(feeRes.fee).String(),
				func() string {
					if balanceRes.balance != nil {
						return balanceRes.balance.Available.String()
					}
					return "0"
				}())
		}
	}

	// Procesar transacción según el tipo (operación síncrona crítica)
	if order.Type == models.OrderTypeBuy {
		requiredAmount := feeRes.totalAmount.Add(feeRes.fee)
		_, err := s.userBalanceClient.ProcessTransaction(ctx, order.UserID, requiredAmount, "buy", order.ID.Hex(), fmt.Sprintf("Buy %s %s at %s", order.Quantity.String(), order.CryptoSymbol, priceRes.price.MarketPrice.String()))
		if err != nil {
			return nil, fmt.Errorf("failed to process transaction: %w", err)
		}

		// Actualizar holdings en el portfolio (asíncrono, no bloquea)
		if s.portfolioClient != nil {
			go func() {
				_ = s.portfolioClient.UpdateHoldings(ctx, int64(order.UserID), order.CryptoSymbol, order.Quantity, priceRes.price.MarketPrice, "buy")
			}()
		}
	} else if order.Type == models.OrderTypeSell {
		netAmount := feeRes.totalAmount.Sub(feeRes.fee)
		_, err := s.userBalanceClient.ProcessTransaction(ctx, order.UserID, netAmount, "sell", order.ID.Hex(), fmt.Sprintf("Sell %s %s at %s", order.Quantity.String(), order.CryptoSymbol, priceRes.price.MarketPrice.String()))
		if err != nil {
			return nil, fmt.Errorf("failed to process transaction: %w", err)
		}

		// Actualizar holdings en el portfolio (asíncrono, no bloquea)
		if s.portfolioClient != nil {
			go func() {
				_ = s.portfolioClient.UpdateHoldings(ctx, int64(order.UserID), order.CryptoSymbol, order.Quantity, priceRes.price.MarketPrice, "sell")
			}()
		}
	}

	// Crear resultado exitoso
	result := &models.ExecutionResult{
		Success:       true,
		OrderID:       order.ID.Hex(),
		ExecutedPrice: priceRes.price.MarketPrice,
		TotalAmount:   feeRes.totalAmount,
		Fee:           feeRes.fee,
		ExecutionTime: time.Since(start),
	}

	return result, nil
}
