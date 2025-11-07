package services

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"orders-api/internal/dto"
	"orders-api/internal/models"
)

// Mock implementations
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) Create(ctx context.Context, order *models.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) Update(ctx context.Context, order *models.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) GetByID(ctx context.Context, orderID string) (*models.Order, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockOrderRepository) ListByUser(ctx context.Context, userID int, filter *dto.OrderFilterRequest) ([]models.Order, int64, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Order), args.Get(1).(int64), args.Error(2)
}

func (m *MockOrderRepository) GetOrdersSummary(ctx context.Context, userID int) (*dto.OrdersSummary, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.OrdersSummary), args.Error(1)
}

func (m *MockOrderRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOrderRepository) GetByOrderNumber(ctx context.Context, orderNumber string) (*models.Order, error) {
	args := m.Called(ctx, orderNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockOrderRepository) UpdateStatus(ctx context.Context, id string, status models.OrderStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockOrderRepository) GetPendingOrders(ctx context.Context, limit int) ([]models.Order, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Order), args.Error(1)
}

func (m *MockOrderRepository) GetOrdersByStatus(ctx context.Context, status models.OrderStatus, limit int) ([]models.Order, error) {
	args := m.Called(ctx, status, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Order), args.Error(1)
}

func (m *MockOrderRepository) BulkUpdateStatus(ctx context.Context, orderIDs []string, status models.OrderStatus) error {
	args := m.Called(ctx, orderIDs, status)
	return args.Error(0)
}

type MockMarketService struct {
	mock.Mock
}

func (m *MockMarketService) GetCurrentPrice(ctx context.Context, symbol string) (decimal.Decimal, error) {
	args := m.Called(ctx, symbol)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockMarketService) ValidateSymbol(ctx context.Context, symbol string) (*CryptoInfo, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CryptoInfo), args.Error(1)
}

type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) PublishOrderCreated(ctx context.Context, order *models.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockEventPublisher) PublishOrderExecuted(ctx context.Context, order *models.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockEventPublisher) PublishOrderCancelled(ctx context.Context, order *models.Order, reason string) error {
	args := m.Called(ctx, order, reason)
	return args.Error(0)
}

func (m *MockEventPublisher) PublishOrderFailed(ctx context.Context, order *models.Order, reason string) error {
	args := m.Called(ctx, order, reason)
	return args.Error(0)
}

// Helper function to create a test execution service with mocked dependencies
func createMockExecutionService() *ExecutionService {
	return &ExecutionService{}
}

// Test CreateOrder
func TestOrderServiceSimple_CreateOrder(t *testing.T) {
	ctx := context.Background()

	t.Run("successful buy limit order creation", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		// Setup request for LIMIT order (not executed immediately)
		req := &dto.CreateOrderRequest{
			Type:         models.OrderTypeBuy,
			CryptoSymbol: "BTC",
			Quantity:     "0.1",
			OrderKind:    models.OrderKindLimit,
			LimitPrice:   "50000.00",
		}

		cryptoInfo := &CryptoInfo{
			Symbol:       "BTC",
			Name:         "Bitcoin",
			CurrentPrice: decimal.NewFromInt(49000),
			IsActive:     true,
		}

		// Setup mocks - Limit orders don't execute immediately
		mockMarket.On("ValidateSymbol", ctx, "BTC").Return(cryptoInfo, nil)
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Order")).Return(nil)
		mockPublisher.On("PublishOrderCreated", ctx, mock.AnythingOfType("*models.Order")).Return(nil)

		// Execute
		order, err := service.CreateOrder(ctx, req, 1)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, models.OrderTypeBuy, order.Type)
		assert.Equal(t, "BTC", order.CryptoSymbol)
		assert.Equal(t, models.OrderStatusPending, order.Status) // Limit orders stay pending
		assert.True(t, order.Quantity.Equal(decimal.NewFromFloat(0.1)))
		assert.True(t, order.Price.Equal(decimal.NewFromInt(50000)))

		mockRepo.AssertExpectations(t)
		mockMarket.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})

	t.Run("successful limit order creation", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		req := &dto.CreateOrderRequest{
			Type:         models.OrderTypeSell,
			CryptoSymbol: "ETH",
			Quantity:     "1.5",
			OrderKind:    models.OrderKindLimit,
			LimitPrice:   "3000.00",
		}

		cryptoInfo := &CryptoInfo{
			Symbol:       "ETH",
			Name:         "Ethereum",
			CurrentPrice: decimal.NewFromInt(2800),
			IsActive:     true,
		}

		mockMarket.On("ValidateSymbol", ctx, "ETH").Return(cryptoInfo, nil)
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Order")).Return(nil)
		mockPublisher.On("PublishOrderCreated", ctx, mock.AnythingOfType("*models.Order")).Return(nil)

		order, err := service.CreateOrder(ctx, req, 1)

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, models.OrderTypeSell, order.Type)
		assert.Equal(t, "ETH", order.CryptoSymbol)
		assert.Equal(t, models.OrderStatusPending, order.Status)
		assert.True(t, order.Price.Equal(decimal.NewFromInt(3000)))

		mockRepo.AssertExpectations(t)
		mockMarket.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})

	t.Run("invalid crypto symbol", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		req := &dto.CreateOrderRequest{
			Type:         models.OrderTypeBuy,
			CryptoSymbol: "INVALID",
			Quantity:     "1.0",
			OrderKind:    models.OrderKindMarket,
			MarketPrice:  "1000.00",
		}

		mockMarket.On("ValidateSymbol", ctx, "INVALID").Return(nil, errors.New("symbol not found"))

		order, err := service.CreateOrder(ctx, req, 1)

		assert.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "invalid crypto symbol")

		mockMarket.AssertExpectations(t)
	})

	t.Run("inactive crypto symbol", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		req := &dto.CreateOrderRequest{
			Type:         models.OrderTypeBuy,
			CryptoSymbol: "SUSPENDED",
			Quantity:     "1.0",
			OrderKind:    models.OrderKindMarket,
			MarketPrice:  "100.00",
		}

		cryptoInfo := &CryptoInfo{
			Symbol:       "SUSPENDED",
			Name:         "Suspended Coin",
			CurrentPrice: decimal.NewFromInt(100),
			IsActive:     false,
		}

		mockMarket.On("ValidateSymbol", ctx, "SUSPENDED").Return(cryptoInfo, nil)

		order, err := service.CreateOrder(ctx, req, 1)

		assert.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "trading is suspended")

		mockMarket.AssertExpectations(t)
	})

	t.Run("repository error on create", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		req := &dto.CreateOrderRequest{
			Type:         models.OrderTypeBuy,
			CryptoSymbol: "BTC",
			Quantity:     "0.5",
			OrderKind:    models.OrderKindMarket,
			MarketPrice:  "50000.00",
		}

		cryptoInfo := &CryptoInfo{
			Symbol:       "BTC",
			Name:         "Bitcoin",
			CurrentPrice: decimal.NewFromInt(50000),
			IsActive:     true,
		}

		mockMarket.On("ValidateSymbol", ctx, "BTC").Return(cryptoInfo, nil)
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Order")).Return(errors.New("database error"))

		order, err := service.CreateOrder(ctx, req, 1)

		assert.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "failed to save order")

		mockRepo.AssertExpectations(t)
		mockMarket.AssertExpectations(t)
	})
}

// Test GetOrder
func TestOrderServiceSimple_GetOrder(t *testing.T) {
	ctx := context.Background()

	t.Run("successful order retrieval", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		orderID := primitive.NewObjectID().Hex()
		expectedOrder := &models.Order{
			ID:           primitive.NewObjectID(),
			OrderNumber:  "ORD-123",
			UserID:       1,
			Type:         models.OrderTypeBuy,
			CryptoSymbol: "BTC",
			Status:       models.OrderStatusExecuted,
		}

		mockRepo.On("GetByID", ctx, orderID).Return(expectedOrder, nil)

		order, err := service.GetOrder(ctx, orderID, 1)

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, expectedOrder.OrderNumber, order.OrderNumber)

		mockRepo.AssertExpectations(t)
	})

	t.Run("order not found", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		orderID := primitive.NewObjectID().Hex()

		mockRepo.On("GetByID", ctx, orderID).Return(nil, errors.New("order not found"))

		order, err := service.GetOrder(ctx, orderID, 1)

		assert.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "order not found")

		mockRepo.AssertExpectations(t)
	})

	t.Run("access denied - wrong user", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		orderID := primitive.NewObjectID().Hex()
		expectedOrder := &models.Order{
			ID:           primitive.NewObjectID(),
			OrderNumber:  "ORD-123",
			UserID:       2,
			Type:         models.OrderTypeBuy,
			CryptoSymbol: "BTC",
			Status:       models.OrderStatusExecuted,
		}

		mockRepo.On("GetByID", ctx, orderID).Return(expectedOrder, nil)

		order, err := service.GetOrder(ctx, orderID, 1)

		assert.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "access denied")

		mockRepo.AssertExpectations(t)
	})
}

// Test CancelOrder
func TestOrderServiceSimple_CancelOrder(t *testing.T) {
	ctx := context.Background()

	t.Run("successful order cancellation", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		orderID := primitive.NewObjectID().Hex()
		order := &models.Order{
			ID:           primitive.NewObjectID(),
			OrderNumber:  "ORD-123",
			UserID:       1,
			Type:         models.OrderTypeBuy,
			CryptoSymbol: "BTC",
			Status:       models.OrderStatusPending,
		}

		mockRepo.On("GetByID", ctx, orderID).Return(order, nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Order")).Return(nil)
		mockPublisher.On("PublishOrderCancelled", ctx, mock.AnythingOfType("*models.Order"), "user requested").Return(nil)

		err := service.CancelOrder(ctx, orderID, 1, "user requested")

		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
		mockPublisher.AssertExpectations(t)
	})

	t.Run("order not found", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		orderID := primitive.NewObjectID().Hex()

		mockRepo.On("GetByID", ctx, orderID).Return(nil, errors.New("order not found"))

		err := service.CancelOrder(ctx, orderID, 1, "user requested")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "order not found")

		mockRepo.AssertExpectations(t)
	})

	t.Run("access denied - wrong user", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		orderID := primitive.NewObjectID().Hex()
		order := &models.Order{
			ID:           primitive.NewObjectID(),
			OrderNumber:  "ORD-123",
			UserID:       2,
			Type:         models.OrderTypeBuy,
			CryptoSymbol: "BTC",
			Status:       models.OrderStatusPending,
		}

		mockRepo.On("GetByID", ctx, orderID).Return(order, nil)

		err := service.CancelOrder(ctx, orderID, 1, "user requested")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")

		mockRepo.AssertExpectations(t)
	})

	t.Run("cannot cancel non-pending order", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		orderID := primitive.NewObjectID().Hex()
		order := &models.Order{
			ID:           primitive.NewObjectID(),
			OrderNumber:  "ORD-123",
			UserID:       1,
			Type:         models.OrderTypeBuy,
			CryptoSymbol: "BTC",
			Status:       models.OrderStatusExecuted,
		}

		mockRepo.On("GetByID", ctx, orderID).Return(order, nil)

		err := service.CancelOrder(ctx, orderID, 1, "user requested")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be cancelled")

		mockRepo.AssertExpectations(t)
	})
}

// Test ListUserOrders
func TestOrderServiceSimple_ListUserOrders(t *testing.T) {
	ctx := context.Background()

	t.Run("successful order listing", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		filter := &dto.OrderFilterRequest{
			Page:  1,
			Limit: 10,
		}

		expectedOrders := []models.Order{
			{
				ID:           primitive.NewObjectID(),
				OrderNumber:  "ORD-123",
				UserID:       1,
				Type:         models.OrderTypeBuy,
				CryptoSymbol: "BTC",
				Status:       models.OrderStatusExecuted,
			},
			{
				ID:           primitive.NewObjectID(),
				OrderNumber:  "ORD-124",
				UserID:       1,
				Type:         models.OrderTypeSell,
				CryptoSymbol: "ETH",
				Status:       models.OrderStatusPending,
			},
		}

		summary := &dto.OrdersSummary{
			TotalOrders:     2,
			ExecutedOrders:  1,
			PendingOrders:   1,
			CancelledOrders: 0,
			FailedOrders:    0,
			TotalVolume:     decimal.NewFromInt(100000),
		}

		mockRepo.On("ListByUser", ctx, 1, filter).Return(expectedOrders, int64(2), nil)
		mockRepo.On("GetOrdersSummary", ctx, 1).Return(summary, nil)

		orders, total, resultSummary, err := service.ListUserOrders(ctx, 1, filter)

		assert.NoError(t, err)
		assert.NotNil(t, orders)
		assert.Equal(t, 2, len(orders))
		assert.Equal(t, int64(2), total)
		assert.Equal(t, summary.TotalOrders, resultSummary.TotalOrders)

		mockRepo.AssertExpectations(t)
	})

	t.Run("empty order list", func(t *testing.T) {
		mockRepo := new(MockOrderRepository)
		mockExec := createMockExecutionService()
		mockMarket := new(MockMarketService)
		mockPublisher := new(MockEventPublisher)

		service := NewOrderServiceSimple(mockRepo, mockExec, mockMarket, mockPublisher)

		filter := &dto.OrderFilterRequest{
			Page:  1,
			Limit: 10,
		}

		emptyOrders := []models.Order{}
		summary := &dto.OrdersSummary{
			TotalOrders:     0,
			ExecutedOrders:  0,
			PendingOrders:   0,
			CancelledOrders: 0,
			FailedOrders:    0,
			TotalVolume:     decimal.Zero,
		}

		mockRepo.On("ListByUser", ctx, 1, filter).Return(emptyOrders, int64(0), nil)
		mockRepo.On("GetOrdersSummary", ctx, 1).Return(summary, nil)

		orders, total, resultSummary, err := service.ListUserOrders(ctx, 1, filter)

		assert.NoError(t, err)
		assert.NotNil(t, orders)
		assert.Equal(t, 0, len(orders))
		assert.Equal(t, int64(0), total)
		assert.Equal(t, int64(0), resultSummary.TotalOrders)

		mockRepo.AssertExpectations(t)
	})
}
