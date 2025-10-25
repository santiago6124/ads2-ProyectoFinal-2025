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
}

// UserClient interface para validar usuarios
type UserClient interface {
	VerifyUser(ctx context.Context, userID int) (*models.ValidationResult, error)
}

// UserBalanceClient interface para verificar saldos
type UserBalanceClient interface {
	CheckBalance(ctx context.Context, userID int, amount decimal.Decimal) (*models.BalanceResult, error)
}

// MarketClient interface para obtener precios
type MarketClient interface {
	GetCurrentPrice(ctx context.Context, symbol string) (*models.PriceResult, error)
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
	}
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

	// 5. Verificar balance (para compras)
	if order.Type == models.OrderTypeBuy {
		requiredAmount := totalAmount.Add(fee)
		balanceResult, err := s.userBalanceClient.CheckBalance(ctx, order.UserID, requiredAmount)
		if err != nil {
			return nil, fmt.Errorf("balance check failed: %w", err)
		}

		if !balanceResult.HasSufficient {
			return nil, fmt.Errorf("insufficient balance: required %s, available %s",
				requiredAmount.String(), balanceResult.Available.String())
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
