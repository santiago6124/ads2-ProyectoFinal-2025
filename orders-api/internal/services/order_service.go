package services

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"orders-api/internal/concurrent"
	"orders-api/internal/dto"
	"orders-api/internal/models"
	"orders-api/internal/repositories"
)

type OrderService interface {
	CreateOrder(ctx context.Context, req *dto.CreateOrderRequest, userID int, metadata *models.OrderMetadata) (*models.Order, error)
	GetOrder(ctx context.Context, orderID string, userID int) (*models.Order, error)
	UpdateOrder(ctx context.Context, orderID string, req *dto.UpdateOrderRequest, userID int) (*models.Order, error)
	CancelOrder(ctx context.Context, orderID string, userID int, reason string) error
	ListUserOrders(ctx context.Context, userID int, filter *dto.OrderFilterRequest) ([]models.Order, int64, *dto.OrdersSummary, error)
	ExecuteOrder(ctx context.Context, orderID string, forceExecution bool) (*models.ExecutionResult, error)
	ListAllOrders(ctx context.Context, filter *dto.AdminOrderFilterRequest) ([]models.Order, int64, *dto.AdminStatistics, error)
	ReprocessOrder(ctx context.Context, orderID string, reason string) error
	BulkCancelOrders(ctx context.Context, orderIDs []string, reason string) error
	GetOrdersSummary(ctx context.Context, userID int) (*dto.OrdersSummary, error)
}

type orderService struct {
	orderRepo     repositories.OrderRepository
	orchestrator  *concurrent.OrderOrchestrator
	executor      *concurrent.ExecutionService
	feeCalculator FeeCalculator
	marketService MarketService
	publisher     EventPublisher
}

type FeeCalculator interface {
	Calculate(ctx context.Context, order *models.Order) (*models.FeeResult, error)
	CalculateForAmount(ctx context.Context, amount decimal.Decimal, orderType models.OrderKind) (*models.FeeCalculation, error)
}

type MarketService interface {
	GetCurrentPrice(ctx context.Context, symbol string) (decimal.Decimal, error)
	ValidateSymbol(ctx context.Context, symbol string) (*CryptoInfo, error)
	IsMarketOpen(ctx context.Context) bool
}

type CryptoInfo struct {
	Symbol      string          `json:"symbol"`
	Name        string          `json:"name"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	IsActive    bool            `json:"is_active"`
	MinQuantity decimal.Decimal `json:"min_quantity"`
	MaxQuantity decimal.Decimal `json:"max_quantity"`
}

type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, order *models.Order) error
	PublishOrderExecuted(ctx context.Context, order *models.Order) error
	PublishOrderCancelled(ctx context.Context, order *models.Order, reason string) error
	PublishOrderFailed(ctx context.Context, order *models.Order, reason string) error
}

func (s *orderService) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest, userID int, metadata *models.OrderMetadata) (*models.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	cryptoInfo, err := s.marketService.ValidateSymbol(ctx, req.CryptoSymbol)
	if err != nil {
		return nil, fmt.Errorf("invalid crypto symbol: %w", err)
	}

	if !cryptoInfo.IsActive {
		return nil, fmt.Errorf("trading is suspended for %s", req.CryptoSymbol)
	}

	if req.Quantity.LessThan(cryptoInfo.MinQuantity) || req.Quantity.GreaterThan(cryptoInfo.MaxQuantity) {
		return nil, fmt.Errorf("quantity must be between %s and %s",
			cryptoInfo.MinQuantity.String(), cryptoInfo.MaxQuantity.String())
	}

	if !s.marketService.IsMarketOpen(ctx) && req.OrderType == models.OrderKindMarket {
		return nil, fmt.Errorf("market orders are not allowed when market is closed")
	}

	var orderPrice decimal.Decimal
	if req.OrderType == models.OrderKindLimit {
		if req.LimitPrice == nil {
			return nil, fmt.Errorf("limit price is required for limit orders")
		}
		orderPrice = *req.LimitPrice
	} else {
		currentPrice, err := s.marketService.GetCurrentPrice(ctx, req.CryptoSymbol)
		if err != nil {
			return nil, fmt.Errorf("failed to get current price: %w", err)
		}
		orderPrice = currentPrice
	}

	totalAmount := req.Quantity.Mul(orderPrice)

	feeCalculation, err := s.feeCalculator.CalculateForAmount(ctx, totalAmount, req.OrderType)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate fees: %w", err)
	}

	order := &models.Order{
		ID:           primitive.NewObjectID(),
		OrderNumber:  models.NewOrderNumber(),
		UserID:       userID,
		Type:         req.Type,
		Status:       models.OrderStatusPending,
		CryptoSymbol: req.CryptoSymbol,
		CryptoName:   cryptoInfo.Name,
		Quantity:     req.Quantity,
		OrderKind:    req.OrderType,
		LimitPrice:   req.LimitPrice,
		OrderPrice:   orderPrice,
		TotalAmount:  totalAmount,
		Fee:          feeCalculation.TotalFee,
		FeePercentage: feeCalculation.FeePercentage,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Validation: &models.OrderValidation{
			UserVerified:   true,
			BalanceChecked: false,
			MarketHours:    s.marketService.IsMarketOpen(ctx),
			RiskAssessment: "pending",
		},
		Metadata: metadata,
		Audit: &models.OrderAudit{
			CreatedBy: userID,
		},
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	if err := s.publisher.PublishOrderCreated(ctx, order); err != nil {
		// Log but don't fail the order creation
		fmt.Printf("Warning: Failed to publish order created event: %v\n", err)
	}

	if req.OrderType == models.OrderKindMarket {
		go s.processOrderAsync(context.Background(), order)
	}

	return order, nil
}

func (s *orderService) processOrderAsync(ctx context.Context, order *models.Order) {
	callback := func(result *concurrent.OrderResult, err error) {
		if err != nil {
			s.handleOrderExecutionError(ctx, order, err)
			return
		}

		s.handleOrderExecutionSuccess(ctx, order, result.ExecutionResult)
	}

	err := s.orchestrator.SubmitOrder(order, ctx, callback)
	if err != nil {
		s.handleOrderExecutionError(ctx, order, err)
	}
}

func (s *orderService) handleOrderExecutionSuccess(ctx context.Context, order *models.Order, result *models.ExecutionResult) {
	order.Status = models.OrderStatusExecuted
	order.ExecutionPrice = &result.MarketPrice.ExecutionPrice
	order.TotalAmount = order.Quantity.Mul(*order.ExecutionPrice)
	order.Fee = result.FeeCalculation.TotalFee
	now := time.Now()
	order.ExecutedAt = &now
	order.UpdatedAt = now

	order.ExecutionDetails = &models.ExecutionDetails{
		MarketPriceAtExecution: result.MarketPrice.MarketPrice,
		Slippage:              result.MarketPrice.Slippage,
		SlippagePercentage:    result.MarketPrice.SlippagePerc,
		ExecutionTimeMs:       result.ExecutionTime.Milliseconds(),
		Retries:               0,
		ExecutionID:           result.ExecutionID,
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		fmt.Printf("Error updating executed order: %v\n", err)
		return
	}

	if err := s.publisher.PublishOrderExecuted(ctx, order); err != nil {
		fmt.Printf("Warning: Failed to publish order executed event: %v\n", err)
	}
}

func (s *orderService) handleOrderExecutionError(ctx context.Context, order *models.Order, err error) {
	order.Status = models.OrderStatusFailed
	order.UpdatedAt = time.Now()

	if order.Validation == nil {
		order.Validation = &models.OrderValidation{}
	}
	order.Validation.ValidationErrors = append(order.Validation.ValidationErrors, err.Error())

	if updateErr := s.orderRepo.Update(ctx, order); updateErr != nil {
		fmt.Printf("Error updating failed order: %v\n", updateErr)
		return
	}

	if publishErr := s.publisher.PublishOrderFailed(ctx, order, err.Error()); publishErr != nil {
		fmt.Printf("Warning: Failed to publish order failed event: %v\n", publishErr)
	}
}

func (s *orderService) GetOrder(ctx context.Context, orderID string, userID int) (*models.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order.UserID != userID {
		return nil, fmt.Errorf("order not found or access denied")
	}

	return order, nil
}

func (s *orderService) UpdateOrder(ctx context.Context, orderID string, req *dto.UpdateOrderRequest, userID int) (*models.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order.UserID != userID {
		return nil, fmt.Errorf("order not found or access denied")
	}

	if !order.IsEditable() {
		return nil, fmt.Errorf("order cannot be modified in current status: %s", order.Status)
	}

	modifications := []models.OrderModification{}

	if req.Quantity != nil && !req.Quantity.Equal(order.Quantity) {
		modifications = append(modifications, models.OrderModification{
			Field:      "quantity",
			OldValue:   order.Quantity,
			NewValue:   *req.Quantity,
			ModifiedBy: userID,
			ModifiedAt: time.Now(),
		})
		order.Quantity = *req.Quantity
	}

	if req.LimitPrice != nil && (order.LimitPrice == nil || !req.LimitPrice.Equal(*order.LimitPrice)) {
		modifications = append(modifications, models.OrderModification{
			Field:      "limit_price",
			OldValue:   order.LimitPrice,
			NewValue:   *req.LimitPrice,
			ModifiedBy: userID,
			ModifiedAt: time.Now(),
		})
		order.LimitPrice = req.LimitPrice
		order.OrderPrice = *req.LimitPrice
	}

	if len(modifications) > 0 {
		order.TotalAmount = order.Quantity.Mul(order.OrderPrice)
		order.UpdatedAt = time.Now()

		if order.Audit == nil {
			order.Audit = &models.OrderAudit{}
		}
		order.Audit.Modifications = append(order.Audit.Modifications, modifications...)
		order.Audit.ModifiedBy = &userID

		feeCalculation, err := s.feeCalculator.CalculateForAmount(ctx, order.TotalAmount, order.OrderKind)
		if err != nil {
			return nil, fmt.Errorf("failed to recalculate fees: %w", err)
		}
		order.Fee = feeCalculation.TotalFee
		order.FeePercentage = feeCalculation.FeePercentage

		if err := s.orderRepo.Update(ctx, order); err != nil {
			return nil, fmt.Errorf("failed to update order: %w", err)
		}
	}

	return order, nil
}

func (s *orderService) CancelOrder(ctx context.Context, orderID string, userID int, reason string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.UserID != userID {
		return fmt.Errorf("order not found or access denied")
	}

	if !order.IsCancellable() {
		return fmt.Errorf("order cannot be cancelled in current status: %s", order.Status)
	}

	order.Status = models.OrderStatusCancelled
	now := time.Now()
	order.CancelledAt = &now
	order.UpdatedAt = now

	if order.Audit == nil {
		order.Audit = &models.OrderAudit{}
	}
	order.Audit.Modifications = append(order.Audit.Modifications, models.OrderModification{
		Field:      "status",
		OldValue:   models.OrderStatusPending,
		NewValue:   models.OrderStatusCancelled,
		ModifiedBy: userID,
		ModifiedAt: now,
		Reason:     reason,
	})

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	if err := s.publisher.PublishOrderCancelled(ctx, order, reason); err != nil {
		fmt.Printf("Warning: Failed to publish order cancelled event: %v\n", err)
	}

	return nil
}

func (s *orderService) ListUserOrders(ctx context.Context, userID int, filter *dto.OrderFilterRequest) ([]models.Order, int64, *dto.OrdersSummary, error) {
	filter.SetDefaults()

	if !filter.IsValidSort() {
		return nil, 0, nil, fmt.Errorf("invalid sort field")
	}

	orders, total, err := s.orderRepo.ListByUser(ctx, userID, filter)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to list orders: %w", err)
	}

	summary, err := s.orderRepo.GetOrdersSummary(ctx, userID)
	if err != nil {
		fmt.Printf("Warning: Failed to get orders summary: %v\n", err)
		summary = &dto.OrdersSummary{}
	}

	return orders, total, summary, nil
}

func (s *orderService) ExecuteOrder(ctx context.Context, orderID string, forceExecution bool) (*models.ExecutionResult, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if !forceExecution && order.Status != models.OrderStatusPending {
		return nil, fmt.Errorf("order cannot be executed in current status: %s", order.Status)
	}

	order.Status = models.OrderStatusProcessing
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	result, err := s.executor.ExecuteOrderConcurrent(ctx, order)
	if err != nil {
		order.Status = models.OrderStatusFailed
		s.orderRepo.Update(ctx, order)
		return nil, fmt.Errorf("order execution failed: %w", err)
	}

	s.handleOrderExecutionSuccess(ctx, order, result)
	return result, nil
}

func (s *orderService) ListAllOrders(ctx context.Context, filter *dto.AdminOrderFilterRequest) ([]models.Order, int64, *dto.AdminStatistics, error) {
	filter.SetDefaults()

	if !filter.IsValidSort() {
		return nil, 0, nil, fmt.Errorf("invalid sort field")
	}

	orders, total, err := s.orderRepo.ListAll(ctx, filter)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to list orders: %w", err)
	}

	stats, err := s.orderRepo.GetAdminStatistics(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to get admin statistics: %v\n", err)
		stats = &dto.AdminStatistics{}
	}

	return orders, total, stats, nil
}

func (s *orderService) ReprocessOrder(ctx context.Context, orderID string, reason string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.Status != models.OrderStatusFailed {
		return fmt.Errorf("only failed orders can be reprocessed")
	}

	order.Status = models.OrderStatusPending
	order.UpdatedAt = time.Now()

	if order.Validation != nil {
		order.Validation.ValidationErrors = nil
	}

	if order.Audit == nil {
		order.Audit = &models.OrderAudit{}
	}
	order.Audit.Modifications = append(order.Audit.Modifications, models.OrderModification{
		Field:      "status",
		OldValue:   models.OrderStatusFailed,
		NewValue:   models.OrderStatusPending,
		ModifiedAt: time.Now(),
		Reason:     "reprocessed: " + reason,
	})

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	go s.processOrderAsync(context.Background(), order)

	return nil
}

func (s *orderService) BulkCancelOrders(ctx context.Context, orderIDs []string, reason string) error {
	if len(orderIDs) == 0 {
		return fmt.Errorf("no orders specified")
	}

	if len(orderIDs) > 50 {
		return fmt.Errorf("cannot cancel more than 50 orders at once")
	}

	err := s.orderRepo.BulkUpdateStatus(ctx, orderIDs, models.OrderStatusCancelled)
	if err != nil {
		return fmt.Errorf("failed to bulk cancel orders: %w", err)
	}

	return nil
}

func (s *orderService) GetOrdersSummary(ctx context.Context, userID int) (*dto.OrdersSummary, error) {
	summary, err := s.orderRepo.GetOrdersSummary(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders summary: %w", err)
	}

	return summary, nil
}

func NewOrderService(
	orderRepo repositories.OrderRepository,
	orchestrator *concurrent.OrderOrchestrator,
	executor *concurrent.ExecutionService,
	feeCalculator FeeCalculator,
	marketService MarketService,
	publisher EventPublisher,
) OrderService {
	return &orderService{
		orderRepo:     orderRepo,
		orchestrator:  orchestrator,
		executor:      executor,
		feeCalculator: feeCalculator,
		marketService: marketService,
		publisher:     publisher,
	}
}