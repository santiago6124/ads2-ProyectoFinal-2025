package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"orders-api/internal/dto"
	"orders-api/internal/models"
	"orders-api/internal/repositories"
)

// OrderServiceSimple servicio simplificado de órdenes (sin concurrencia compleja)
type OrderServiceSimple struct {
	orderRepo        repositories.OrderRepository
	executionService *ExecutionService
	marketService    MarketService
	publisher        EventPublisher
}

// MarketService interface para servicios de mercado
type MarketService interface {
	GetCurrentPrice(ctx context.Context, symbol string) (decimal.Decimal, error)
	ValidateSymbol(ctx context.Context, symbol string) (*CryptoInfo, error)
}

// CryptoInfo información de una criptomoneda
type CryptoInfo struct {
	Symbol       string          `json:"symbol"`
	Name         string          `json:"name"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	IsActive     bool            `json:"is_active"`
}

// EventPublisher interface para publicar eventos
type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, order *models.Order) error
	PublishOrderExecuted(ctx context.Context, order *models.Order) error
	PublishOrderCancelled(ctx context.Context, order *models.Order, reason string) error
	PublishOrderFailed(ctx context.Context, order *models.Order, reason string) error
}

// NewOrderServiceSimple crea una instancia del servicio simplificado
func NewOrderServiceSimple(
	orderRepo repositories.OrderRepository,
	executionService *ExecutionService,
	marketService MarketService,
	publisher EventPublisher,
) *OrderServiceSimple {
	return &OrderServiceSimple{
		orderRepo:        orderRepo,
		executionService: executionService,
		marketService:    marketService,
		publisher:        publisher,
	}
}

// CreateOrder crea y ejecuta una orden de forma simplificada
func (s *OrderServiceSimple) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest, userID int) (*models.Order, error) {
	// DEBUG: Log request
	log.Printf("🔍 CreateOrder received - Symbol: %s, MarketPrice field: '%s', OrderKind: %s",
		req.CryptoSymbol, req.MarketPrice, req.OrderKind)

	// 1. Validar request y parsear valores
	quantity, limitPrice, marketPrice, err := req.Validate()
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// DEBUG: Log parsed values
	if marketPrice != nil {
		log.Printf("🔍 Parsed marketPrice: %s", marketPrice.String())
	} else {
		log.Printf("🔍 Parsed marketPrice is nil")
	}

	// 2. Validar símbolo de crypto
	cryptoInfo, err := s.marketService.ValidateSymbol(ctx, req.CryptoSymbol)
	if err != nil {
		return nil, fmt.Errorf("invalid crypto symbol: %w", err)
	}

	if !cryptoInfo.IsActive {
		return nil, fmt.Errorf("trading is suspended for %s", req.CryptoSymbol)
	}

	// 3. Determinar precio de la orden
	var orderPrice decimal.Decimal
	if req.OrderKind == models.OrderKindLimit {
		// Para limit orders, usar el precio límite
		orderPrice = *limitPrice
	} else if marketPrice != nil {
		// Para market orders, usar el precio del frontend si está disponible
		log.Printf("📊 Using market price from frontend: %s for %s", marketPrice.String(), req.CryptoSymbol)
		orderPrice = *marketPrice
	} else {
		// Si no viene precio del frontend, obtener del backend
		log.Printf("⚠️ No market price from frontend, fetching from backend for %s", req.CryptoSymbol)
		currentPrice, err := s.marketService.GetCurrentPrice(ctx, req.CryptoSymbol)
		if err != nil {
			return nil, fmt.Errorf("failed to get current price: %w", err)
		}
		orderPrice = currentPrice
	}

	// 4. Calcular monto total y comisión
	totalAmount := quantity.Mul(orderPrice)
	fee := totalAmount.Mul(decimal.NewFromFloat(0.001)) // 0.1%
	minFee := decimal.NewFromFloat(0.01)
	if fee.LessThan(minFee) {
		fee = minFee
	}

	// 5. Crear orden
	order := &models.Order{
		ID:           primitive.NewObjectID(),
		OrderNumber:  models.NewOrderNumber(),
		UserID:       userID,
		Type:         req.Type,
		Status:       models.OrderStatusPending,
		CryptoSymbol: req.CryptoSymbol,
		CryptoName:   cryptoInfo.Name,
		Quantity:     quantity,
		OrderKind:    req.OrderKind,
		Price:        orderPrice,
		TotalAmount:  totalAmount,
		Fee:          fee,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 6. Guardar en base de datos
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// 7. Publicar evento de creación (no bloquea si falla)
	if err := s.publisher.PublishOrderCreated(ctx, order); err != nil {
		log.Printf("Warning: failed to publish order created event: %v", err)
	}

	// 8. Si es market order, ejecutar inmediatamente de forma síncrona
	if req.OrderKind == models.OrderKindMarket {
		// Get user token from context if available
		execCtx := ctx
		if userToken := ctx.Value("user_token"); userToken != nil {
			execCtx = context.WithValue(execCtx, "user_token", userToken)
		}

		if err := s.executeOrderSync(execCtx, order); err != nil {
			log.Printf("Warning: failed to execute market order: %v", err)
			// La orden queda en pending, el usuario puede ver el error
		}
	}

	return order, nil
}

// executeOrderSync ejecuta una orden de forma síncrona
func (s *OrderServiceSimple) executeOrderSync(ctx context.Context, order *models.Order) error {
	// Ejecutar orden con timeout
	execCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := s.executionService.ExecuteOrder(execCtx, order)
	if err != nil {
		// Marcar orden como fallida
		order.Status = models.OrderStatusFailed
		order.ErrorMessage = err.Error()
		order.UpdatedAt = time.Now()

		s.orderRepo.Update(ctx, order)
		s.publisher.PublishOrderFailed(ctx, order, err.Error())
		return err
	}

	// Actualizar orden con resultado exitoso
	order.Status = models.OrderStatusExecuted
	order.Price = result.ExecutedPrice
	order.TotalAmount = result.TotalAmount
	order.Fee = result.Fee
	now := time.Now()
	order.ExecutedAt = &now
	order.UpdatedAt = now

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("failed to update executed order: %w", err)
	}

	// Publicar evento de ejecución
	if err := s.publisher.PublishOrderExecuted(ctx, order); err != nil {
		log.Printf("Warning: failed to publish order executed event: %v", err)
	}

	return nil
}

// GetOrder obtiene una orden por ID
func (s *OrderServiceSimple) GetOrder(ctx context.Context, orderID string, userID int) (*models.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	if order.UserID != userID {
		return nil, fmt.Errorf("access denied: order does not belong to user")
	}

	return order, nil
}

// ListUserOrders lista las órdenes de un usuario con filtros
func (s *OrderServiceSimple) ListUserOrders(ctx context.Context, userID int, filter *dto.OrderFilterRequest) ([]models.Order, int64, *dto.OrdersSummary, error) {
	filter.SetDefaults()

	orders, total, err := s.orderRepo.ListByUser(ctx, userID, filter)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to list orders: %w", err)
	}

	summary, err := s.orderRepo.GetOrdersSummary(ctx, userID)
	if err != nil {
		log.Printf("Warning: failed to get orders summary: %v", err)
		summary = &dto.OrdersSummary{}
	}

	return orders, total, summary, nil
}

// CancelOrder cancela una orden pendiente
func (s *OrderServiceSimple) CancelOrder(ctx context.Context, orderID string, userID int, reason string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	if order.UserID != userID {
		return fmt.Errorf("access denied: order does not belong to user")
	}

	if !order.IsCancellable() {
		return fmt.Errorf("order cannot be cancelled (status: %s)", order.Status)
	}

	order.Status = models.OrderStatusCancelled
	order.UpdatedAt = time.Now()

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	// Publicar evento de cancelación
	if err := s.publisher.PublishOrderCancelled(ctx, order, reason); err != nil {
		log.Printf("Warning: failed to publish order cancelled event: %v", err)
	}

	return nil
}
