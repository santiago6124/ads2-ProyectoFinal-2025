package services

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"orders-api/internal/models"
)

// ExecutionService servicio simplificado para ejecutar órdenes
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

// ExecuteOrder ejecuta una orden de manera síncrona y simplificada
func (s *ExecutionService) ExecuteOrder(ctx context.Context, order *models.Order) (*models.ExecutionResult, error) {
	start := time.Now()

	// 1. Verificar usuario
	_, err := s.userClient.VerifyUser(ctx, order.UserID)
	if err != nil {
		return nil, fmt.Errorf("user validation failed: %w", err)
	}

	// 2. Obtener precio de mercado
	priceResult, err := s.marketClient.GetCurrentPrice(ctx, order.CryptoSymbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get market price: %w", err)
	}

	// 3. Calcular monto total
	totalAmount := order.Quantity.Mul(priceResult.MarketPrice)

	// 4. Calcular comisión
	fee := totalAmount.Mul(decimal.NewFromFloat(0.001)) // 0.1%
	minFee := decimal.NewFromFloat(0.01)
	if fee.LessThan(minFee) {
		fee = minFee
	}

	// Get user token from context
	userToken := ""
	if token := ctx.Value("user_token"); token != nil {
		userToken = token.(string)
	}

	// 5. Verificar balance y procesar transacción según el tipo
	if order.Type == models.OrderTypeBuy {
		// Para COMPRAS: verificar si tiene suficiente dinero
		requiredAmount := totalAmount.Add(fee)
		balanceResult, err := s.userBalanceClient.CheckBalance(ctx, order.UserID, requiredAmount, userToken)
		if err != nil {
			return nil, fmt.Errorf("balance check failed: %w", err)
		}

		if !balanceResult.HasSufficient {
			return nil, fmt.Errorf("insufficient balance: required %s, available %s",
				requiredAmount.String(), balanceResult.Available.String())
		}

		// Deduct del balance (comprar)
		_, err = s.userBalanceClient.ProcessTransaction(ctx, order.UserID, requiredAmount, "buy", order.ID.Hex(), fmt.Sprintf("Buy %s %s at %s", order.Quantity.String(), order.CryptoSymbol, priceResult.MarketPrice.String()))
		if err != nil {
			return nil, fmt.Errorf("failed to process transaction: %w", err)
		}

		// Actualizar holdings en el portfolio
		if s.portfolioClient != nil {
			err = s.portfolioClient.UpdateHoldings(ctx, int64(order.UserID), order.CryptoSymbol, order.Quantity, priceResult.MarketPrice, "buy")
			if err != nil {
				// Log error but don't fail the order execution
				fmt.Printf("⚠️ Failed to update portfolio holdings: %v\n", err)
			}
		}
	} else if order.Type == models.OrderTypeSell {
		// Para VENTAS: agregar dinero al balance (después de descontar fee)
		netAmount := totalAmount.Sub(fee)
		
		// Para ventas, pasamos el monto positivo y ProcessTransaction lo suma al balance
		_, err = s.userBalanceClient.ProcessTransaction(ctx, order.UserID, netAmount, "sell", order.ID.Hex(), fmt.Sprintf("Sell %s %s at %s", order.Quantity.String(), order.CryptoSymbol, priceResult.MarketPrice.String()))
		if err != nil {
			return nil, fmt.Errorf("failed to process transaction: %w", err)
		}

		// Actualizar holdings en el portfolio
		if s.portfolioClient != nil {
			err = s.portfolioClient.UpdateHoldings(ctx, int64(order.UserID), order.CryptoSymbol, order.Quantity, priceResult.MarketPrice, "sell")
			if err != nil {
				// Log error but don't fail the order execution
				fmt.Printf("⚠️ Failed to update portfolio holdings: %v\n", err)
			}
		}
	}

	// 6. Crear resultado exitoso
	result := &models.ExecutionResult{
		Success:       true,
		OrderID:       order.ID.Hex(),
		ExecutedPrice: priceResult.MarketPrice,
		TotalAmount:   totalAmount,
		Fee:           fee,
		ExecutionTime: time.Since(start),
	}

	return result, nil
}
