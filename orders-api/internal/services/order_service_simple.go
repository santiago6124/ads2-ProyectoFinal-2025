package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"orders-api/internal/dto"
	"orders-api/internal/models"
	"orders-api/internal/repositories"
)

// OrderServiceSimple servicio simplificado de órdenes con procesamiento concurrente
type OrderServiceSimple struct {
	orderRepo        repositories.OrderRepository
	executionService *ExecutionService
	marketService    MarketService
	publisher        EventPublisher
	userClient       UserClient // Para validar owner contra API de usuarios (usa la interfaz de ExecutionService)
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
	userClient UserClient, // Agregado para validación de owner
) *OrderServiceSimple {
	return &OrderServiceSimple{
		orderRepo:        orderRepo,
		executionService: executionService,
		marketService:    marketService,
		publisher:        publisher,
		userClient:       userClient,
	}
}

// CreateOrder crea una orden usando procesamiento concurrente con goroutines, channels y WaitGroup
// Valida la existencia del owner contra la API de usuarios antes de crear
func (s *OrderServiceSimple) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest, userID int) (*models.Order, error) {
	// Validación de owner: Invocar al endpoint de obtención por ID mediante HTTP
	if s.userClient != nil {
		_, err := s.userClient.GetUserProfile(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("owner validation failed: user %d does not exist or is not accessible: %w", userID, err)
		}
	}
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

	// Channels para comunicación entre goroutines
	cryptoInfoChan := make(chan *CryptoInfo, 1)
	cryptoInfoErrChan := make(chan error, 1)
	priceChan := make(chan decimal.Decimal, 1)
	priceErrChan := make(chan error, 1)

	// WaitGroup para sincronizar todas las goroutines
	var wg sync.WaitGroup

	// Goroutine 1: Validar símbolo de crypto
	wg.Add(1)
	go func() {
		defer wg.Done()
		cryptoInfo, err := s.marketService.ValidateSymbol(ctx, req.CryptoSymbol)
		if err != nil {
			cryptoInfoErrChan <- err
			return
		}
		cryptoInfoChan <- cryptoInfo
	}()

	// Goroutine 2: Obtener precio (si es necesario)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if req.OrderKind == models.OrderKindLimit {
			// Para limit orders, usar el precio límite (ya validado)
			priceChan <- *limitPrice
			return
		} else if marketPrice != nil {
			// Para market orders, usar el precio del frontend si está disponible
			log.Printf("📊 Using market price from frontend: %s for %s", marketPrice.String(), req.CryptoSymbol)
			priceChan <- *marketPrice
			return
		} else {
			// Si no viene precio del frontend, obtener del backend
			log.Printf("⚠️ No market price from frontend, fetching from backend for %s", req.CryptoSymbol)
			currentPrice, err := s.marketService.GetCurrentPrice(ctx, req.CryptoSymbol)
			if err != nil {
				priceErrChan <- err
				return
			}
			priceChan <- currentPrice
		}
	}()

	// Esperar a que terminen ambas goroutines
	wg.Wait()

	// Leer resultados
	var cryptoInfo *CryptoInfo
	select {
	case cryptoInfo = <-cryptoInfoChan:
	case err := <-cryptoInfoErrChan:
		return nil, fmt.Errorf("invalid crypto symbol: %w", err)
	default:
		return nil, fmt.Errorf("crypto validation did not complete")
	}

	if !cryptoInfo.IsActive {
		return nil, fmt.Errorf("trading is suspended for %s", req.CryptoSymbol)
	}

	var orderPrice decimal.Decimal
	select {
	case orderPrice = <-priceChan:
	case err := <-priceErrChan:
		return nil, fmt.Errorf("failed to get current price: %w", err)
	default:
		return nil, fmt.Errorf("price calculation did not complete")
	}

	// Calcular monto total y comisión (cálculo local, rápido)
	totalAmount := quantity.Mul(orderPrice)
	fee := totalAmount.Mul(decimal.NewFromFloat(0.001)) // 0.1%
	minFee := decimal.NewFromFloat(0.01)
	if fee.LessThan(minFee) {
		fee = minFee
	}

	// Crear orden - dejar que MongoDB genere el ID automáticamente
	order := &models.Order{
		ID:           primitive.NilObjectID, // MongoDB generará el ID automáticamente
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

	// Guardar en base de datos
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Publicar evento de creación a RabbitMQ con operación e ID de entidad
	if err := s.publisher.PublishOrderCreated(ctx, order); err != nil {
		log.Printf("Warning: failed to publish order created event: %v", err)
	}

	// Si es market order, ejecutar inmediatamente usando procesamiento concurrente
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

// UpdateOrder actualiza una orden existente con validación de owner
func (s *OrderServiceSimple) UpdateOrder(ctx context.Context, orderID string, req *dto.UpdateOrderRequest, userID int) (*models.Order, error) {
	// 1. Obtener orden existente
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// 2. Validar que el usuario es el owner (validación de owner)
	if order.UserID != userID {
		return nil, fmt.Errorf("access denied: order does not belong to user")
	}

	// 3. Validar existencia del owner contra la API de usuarios invocando al endpoint de obtención por ID mediante HTTP
	if s.userClient != nil {
		_, err := s.userClient.GetUserProfile(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("owner validation failed: user %d does not exist or is not accessible: %w", userID, err)
		}
	}

	// 4. Actualizar campos si se proporcionan
	if req.Quantity != nil {
		order.Quantity = *req.Quantity
		// Recalcular total amount
		order.TotalAmount = order.Quantity.Mul(order.Price)
		// Recalcular fee
		fee := order.TotalAmount.Mul(decimal.NewFromFloat(0.001))
		minFee := decimal.NewFromFloat(0.01)
		if fee.LessThan(minFee) {
			fee = minFee
		}
		order.Fee = fee
	}

	if req.LimitPrice != nil {
		order.Price = *req.LimitPrice
		// Recalcular total amount
		order.TotalAmount = order.Quantity.Mul(order.Price)
		// Recalcular fee
		fee := order.TotalAmount.Mul(decimal.NewFromFloat(0.001))
		minFee := decimal.NewFromFloat(0.01)
		if fee.LessThan(minFee) {
			fee = minFee
		}
		order.Fee = fee
	}

	order.UpdatedAt = time.Now()

	// 5. Guardar cambios
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	// 6. Publicar evento de actualización a RabbitMQ con operación e ID de entidad
	// Nota: Necesitamos agregar este método al publisher si no existe
	// Por ahora usamos el evento de creación como base
	if err := s.publisher.PublishOrderCreated(ctx, order); err != nil {
		log.Printf("Warning: failed to publish order updated event: %v", err)
	}

	return order, nil
}

// DeleteOrder elimina una orden con validación de owner
func (s *OrderServiceSimple) DeleteOrder(ctx context.Context, orderID string, userID int) error {
	// 1. Obtener orden existente
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	// 2. Validar que el usuario es el owner (validación de owner)
	if order.UserID != userID {
		return fmt.Errorf("access denied: order does not belong to user")
	}

	// 3. Validar existencia del owner contra la API de usuarios invocando al endpoint de obtención por ID mediante HTTP
	if s.userClient != nil {
		_, err := s.userClient.GetUserProfile(ctx, userID)
		if err != nil {
			return fmt.Errorf("owner validation failed: user %d does not exist or is not accessible: %w", userID, err)
		}
	}

	// 4. Eliminar orden
	if err := s.orderRepo.Delete(ctx, orderID); err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	// 5. Publicar evento de eliminación a RabbitMQ con operación e ID de entidad
	// Usamos evento de cancelación como base para la eliminación
	if err := s.publisher.PublishOrderCancelled(ctx, order, "deleted by user"); err != nil {
		log.Printf("Warning: failed to publish order deleted event: %v", err)
	}

	return nil
}

// CancelOrder cancela una orden pendiente con validación de owner
func (s *OrderServiceSimple) CancelOrder(ctx context.Context, orderID string, userID int, reason string) error {
	// 1. Obtener orden existente
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	// 2. Validar que el usuario es el owner (validación de owner)
	if order.UserID != userID {
		return fmt.Errorf("access denied: order does not belong to user")
	}

	// 3. Validar existencia del owner contra la API de usuarios invocando al endpoint de obtención por ID mediante HTTP
	if s.userClient != nil {
		_, err := s.userClient.GetUserProfile(ctx, userID)
		if err != nil {
			return fmt.Errorf("owner validation failed: user %d does not exist or is not accessible: %w", userID, err)
		}
	}

	if !order.IsCancellable() {
		return fmt.Errorf("order cannot be cancelled (status: %s)", order.Status)
	}

	order.Status = models.OrderStatusCancelled
	order.UpdatedAt = time.Now()

	// 4. Guardar cambios
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	// 5. Publicar evento de cancelación a RabbitMQ con operación e ID de entidad
	if err := s.publisher.PublishOrderCancelled(ctx, order, reason); err != nil {
		log.Printf("Warning: failed to publish order cancelled event: %v", err)
	}

	return nil
}

// ExecuteOrder ejecuta una orden pendiente (endpoint de acción)
func (s *OrderServiceSimple) ExecuteOrder(ctx context.Context, orderID string, userID int) (*models.ExecutionResult, error) {
	// 1. Obtener orden existente
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// 2. Validar que el usuario es el owner (validación de owner)
	if order.UserID != userID {
		return nil, fmt.Errorf("access denied: order does not belong to user")
	}

	// 3. Validar existencia del owner contra la API de usuarios
	// (el userID ya viene validado del token JWT)

	// 4. Verificar que la orden puede ser ejecutada
	if order.Status != models.OrderStatusPending {
		return nil, fmt.Errorf("order cannot be executed (status: %s)", order.Status)
	}

	// 5. Ejecutar orden usando el servicio de ejecución (que usa goroutines, channels y WaitGroup)
	execCtx := ctx
	if userToken := ctx.Value("user_token"); userToken != nil {
		execCtx = context.WithValue(execCtx, "user_token", userToken)
	}

	result, err := s.executionService.ExecuteOrder(execCtx, order)
	if err != nil {
		// Marcar orden como fallida
		order.Status = models.OrderStatusFailed
		order.ErrorMessage = err.Error()
		order.UpdatedAt = time.Now()
		s.orderRepo.Update(ctx, order)
		s.publisher.PublishOrderFailed(ctx, order, err.Error())
		return nil, fmt.Errorf("order execution failed: %w", err)
	}

	// 6. Actualizar orden con resultado exitoso
	order.Status = models.OrderStatusExecuted
	order.Price = result.ExecutedPrice
	order.TotalAmount = result.TotalAmount
	order.Fee = result.Fee
	now := time.Now()
	order.ExecutedAt = &now
	order.UpdatedAt = now

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to update executed order: %w", err)
	}

	// 7. Publicar evento de ejecución a RabbitMQ con operación e ID de entidad
	if err := s.publisher.PublishOrderExecuted(ctx, order); err != nil {
		log.Printf("Warning: failed to publish order executed event: %v", err)
	}

	return result, nil
}
