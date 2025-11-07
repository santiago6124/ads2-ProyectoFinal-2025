package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"portfolio-api/internal/models"
)

// Mock implementations
type MockPortfolioRepository struct {
	mock.Mock
}

func (m *MockPortfolioRepository) GetByUserID(ctx context.Context, userID int64) (*models.Portfolio, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Portfolio), args.Error(1)
}

func (m *MockPortfolioRepository) Create(ctx context.Context, portfolio *models.Portfolio) error {
	args := m.Called(ctx, portfolio)
	return args.Error(0)
}

func (m *MockPortfolioRepository) Update(ctx context.Context, portfolio *models.Portfolio) error {
	args := m.Called(ctx, portfolio)
	return args.Error(0)
}

func (m *MockPortfolioRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPortfolioRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockPortfolioRepository) GetNeedingRecalculation(ctx context.Context, limit int) ([]*models.Portfolio, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Portfolio), args.Error(1)
}

func (m *MockPortfolioRepository) BulkUpdate(ctx context.Context, portfolios []*models.Portfolio) error {
	args := m.Called(ctx, portfolios)
	return args.Error(0)
}

type MockSnapshotRepository struct {
	mock.Mock
}

func (m *MockSnapshotRepository) GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.Snapshot, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Snapshot), args.Error(1)
}

func (m *MockSnapshotRepository) GetByInterval(ctx context.Context, userID int64, interval string, limit, offset int) ([]models.Snapshot, error) {
	args := m.Called(ctx, userID, interval, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Snapshot), args.Error(1)
}

func (m *MockSnapshotRepository) Create(ctx context.Context, snapshot *models.Snapshot) error {
	args := m.Called(ctx, snapshot)
	return args.Error(0)
}

func (m *MockSnapshotRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSnapshotRepository) BulkCreate(ctx context.Context, snapshots []models.Snapshot) error {
	args := m.Called(ctx, snapshots)
	return args.Error(0)
}

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) GetPortfolio(ctx context.Context, userID int64, portfolio interface{}) error {
	args := m.Called(ctx, userID, portfolio)
	return args.Error(0)
}

func (m *MockRedisClient) SetPortfolio(ctx context.Context, userID int64, portfolio interface{}) error {
	args := m.Called(ctx, userID, portfolio)
	return args.Error(0)
}

func (m *MockRedisClient) InvalidatePortfolio(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRedisClient) Get(ctx context.Context, key string, value interface{}) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

type MockUserClient struct {
	mock.Mock
}

func (m *MockUserClient) GetUserBalance(ctx context.Context, userID int64) (decimal.Decimal, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

type MockPnLCalculator struct {
	mock.Mock
}

type MockRiskCalculator struct {
	mock.Mock
}

type MockROICalculator struct {
	mock.Mock
}

type MockPortfolioAnalyzer struct {
	mock.Mock
}

type MockPortfolioOptimizer struct {
	mock.Mock
}

// Test GetPortfolio
func TestPortfolioService_GetPortfolio(t *testing.T) {
	ctx := context.Background()

	t.Run("successful retrieval from cache", func(t *testing.T) {
		mockPortfolioRepo := new(MockPortfolioRepository)
		mockSnapshotRepo := new(MockSnapshotRepository)
		mockCache := new(MockRedisClient)
		mockUserClient := new(MockUserClient)
		mockPnL := new(MockPnLCalculator)
		mockRisk := new(MockRiskCalculator)
		mockROI := new(MockROICalculator)
		mockAnalyzer := new(MockPortfolioAnalyzer)
		mockOptimizer := new(MockPortfolioOptimizer)

		service := NewPortfolioService(
			mockPortfolioRepo,
			mockSnapshotRepo,
			mockCache,
			mockUserClient,
			mockPnL,
			mockRisk,
			mockROI,
			mockAnalyzer,
			mockOptimizer,
		)

		userID := int64(1)
		expectedBalance := decimal.NewFromInt(10000)

		// Cache returns portfolio
		mockCache.On("GetPortfolio", ctx, userID, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			portfolio := args.Get(2).(*models.Portfolio)
			portfolio.UserID = userID
			portfolio.TotalCash = decimal.NewFromInt(5000)
		})
		mockUserClient.On("GetUserBalance", ctx, userID).Return(expectedBalance, nil)

		portfolio, err := service.GetPortfolio(ctx, userID)

		assert.NoError(t, err)
		assert.NotNil(t, portfolio)
		assert.Equal(t, userID, portfolio.UserID)
		assert.Equal(t, expectedBalance, portfolio.TotalCash)

		mockCache.AssertExpectations(t)
		mockUserClient.AssertExpectations(t)
	})

	t.Run("successful retrieval from database", func(t *testing.T) {
		mockPortfolioRepo := new(MockPortfolioRepository)
		mockSnapshotRepo := new(MockSnapshotRepository)
		mockCache := new(MockRedisClient)
		mockUserClient := new(MockUserClient)
		mockPnL := new(MockPnLCalculator)
		mockRisk := new(MockRiskCalculator)
		mockROI := new(MockROICalculator)
		mockAnalyzer := new(MockPortfolioAnalyzer)
		mockOptimizer := new(MockPortfolioOptimizer)

		service := NewPortfolioService(
			mockPortfolioRepo,
			mockSnapshotRepo,
			mockCache,
			mockUserClient,
			mockPnL,
			mockRisk,
			mockROI,
			mockAnalyzer,
			mockOptimizer,
		)

		userID := int64(1)
		expectedBalance := decimal.NewFromInt(10000)
		expectedPortfolio := &models.Portfolio{
			ID:         primitive.NewObjectID(),
			UserID:     userID,
			TotalCash:  decimal.NewFromInt(5000),
			TotalValue: decimal.NewFromInt(15000),
			Holdings:   []models.Holding{},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		// Cache miss
		mockCache.On("GetPortfolio", ctx, userID, mock.Anything).Return(errors.New("cache miss"))
		mockPortfolioRepo.On("GetByUserID", ctx, userID).Return(expectedPortfolio, nil)
		mockUserClient.On("GetUserBalance", ctx, userID).Return(expectedBalance, nil)
		mockCache.On("SetPortfolio", ctx, userID, mock.Anything).Return(nil)

		portfolio, err := service.GetPortfolio(ctx, userID)

		assert.NoError(t, err)
		assert.NotNil(t, portfolio)
		assert.Equal(t, userID, portfolio.UserID)
		assert.Equal(t, expectedBalance, portfolio.TotalCash)

		mockCache.AssertExpectations(t)
		mockPortfolioRepo.AssertExpectations(t)
		mockUserClient.AssertExpectations(t)
	})

	t.Run("portfolio not found", func(t *testing.T) {
		mockPortfolioRepo := new(MockPortfolioRepository)
		mockSnapshotRepo := new(MockSnapshotRepository)
		mockCache := new(MockRedisClient)
		mockUserClient := new(MockUserClient)
		mockPnL := new(MockPnLCalculator)
		mockRisk := new(MockRiskCalculator)
		mockROI := new(MockROICalculator)
		mockAnalyzer := new(MockPortfolioAnalyzer)
		mockOptimizer := new(MockPortfolioOptimizer)

		service := NewPortfolioService(
			mockPortfolioRepo,
			mockSnapshotRepo,
			mockCache,
			mockUserClient,
			mockPnL,
			mockRisk,
			mockROI,
			mockAnalyzer,
			mockOptimizer,
		)

		userID := int64(1)

		mockCache.On("GetPortfolio", ctx, userID, mock.Anything).Return(errors.New("cache miss"))
		mockPortfolioRepo.On("GetByUserID", ctx, userID).Return(nil, errors.New("not found"))

		portfolio, err := service.GetPortfolio(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, portfolio)

		mockCache.AssertExpectations(t)
		mockPortfolioRepo.AssertExpectations(t)
	})
}

// Test CreatePortfolio
func TestPortfolioService_CreatePortfolio(t *testing.T) {
	ctx := context.Background()

	t.Run("successful portfolio creation", func(t *testing.T) {
		mockPortfolioRepo := new(MockPortfolioRepository)
		mockSnapshotRepo := new(MockSnapshotRepository)
		mockCache := new(MockRedisClient)
		mockUserClient := new(MockUserClient)
		mockPnL := new(MockPnLCalculator)
		mockRisk := new(MockRiskCalculator)
		mockROI := new(MockROICalculator)
		mockAnalyzer := new(MockPortfolioAnalyzer)
		mockOptimizer := new(MockPortfolioOptimizer)

		service := NewPortfolioService(
			mockPortfolioRepo,
			mockSnapshotRepo,
			mockCache,
			mockUserClient,
			mockPnL,
			mockRisk,
			mockROI,
			mockAnalyzer,
			mockOptimizer,
		)

		userID := int64(1)
		userBalance := decimal.NewFromInt(10000)

		mockPortfolioRepo.On("GetByUserID", ctx, userID).Return(nil, errors.New("not found"))
		mockUserClient.On("GetUserBalance", ctx, userID).Return(userBalance, nil)
		mockPortfolioRepo.On("Create", ctx, mock.AnythingOfType("*models.Portfolio")).Return(nil)
		mockCache.On("SetPortfolio", ctx, userID, mock.Anything).Return(nil)

		portfolio, err := service.CreatePortfolio(ctx, userID)

		assert.NoError(t, err)
		assert.NotNil(t, portfolio)
		assert.Equal(t, userID, portfolio.UserID)
		assert.Equal(t, userBalance, portfolio.TotalCash)

		mockPortfolioRepo.AssertExpectations(t)
		mockUserClient.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("portfolio already exists", func(t *testing.T) {
		mockPortfolioRepo := new(MockPortfolioRepository)
		mockSnapshotRepo := new(MockSnapshotRepository)
		mockCache := new(MockRedisClient)
		mockUserClient := new(MockUserClient)
		mockPnL := new(MockPnLCalculator)
		mockRisk := new(MockRiskCalculator)
		mockROI := new(MockROICalculator)
		mockAnalyzer := new(MockPortfolioAnalyzer)
		mockOptimizer := new(MockPortfolioOptimizer)

		service := NewPortfolioService(
			mockPortfolioRepo,
			mockSnapshotRepo,
			mockCache,
			mockUserClient,
			mockPnL,
			mockRisk,
			mockROI,
			mockAnalyzer,
			mockOptimizer,
		)

		userID := int64(1)
		existingPortfolio := &models.Portfolio{
			ID:        primitive.NewObjectID(),
			UserID:    userID,
			TotalCash: decimal.NewFromInt(5000),
		}

		mockPortfolioRepo.On("GetByUserID", ctx, userID).Return(existingPortfolio, nil)

		portfolio, err := service.CreatePortfolio(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, portfolio)
		assert.Contains(t, err.Error(), "already exists")

		mockPortfolioRepo.AssertExpectations(t)
	})
}

// Test AddHolding
func TestPortfolioService_AddHolding(t *testing.T) {
	ctx := context.Background()

	t.Run("add new holding successfully", func(t *testing.T) {
		mockPortfolioRepo := new(MockPortfolioRepository)
		mockSnapshotRepo := new(MockSnapshotRepository)
		mockCache := new(MockRedisClient)
		mockUserClient := new(MockUserClient)
		mockPnL := new(MockPnLCalculator)
		mockRisk := new(MockRiskCalculator)
		mockROI := new(MockROICalculator)
		mockAnalyzer := new(MockPortfolioAnalyzer)
		mockOptimizer := new(MockPortfolioOptimizer)

		service := NewPortfolioService(
			mockPortfolioRepo,
			mockSnapshotRepo,
			mockCache,
			mockUserClient,
			mockPnL,
			mockRisk,
			mockROI,
			mockAnalyzer,
			mockOptimizer,
		)

		userID := int64(1)
		portfolio := &models.Portfolio{
			ID:        primitive.NewObjectID(),
			UserID:    userID,
			TotalCash: decimal.NewFromInt(10000),
			Holdings:  []models.Holding{},
		}

		newHolding := &models.Holding{
			Symbol:       "BTC",
			Name:         "Bitcoin",
			Quantity:     decimal.NewFromFloat(0.5),
			AverageCost:  decimal.NewFromInt(50000),
			CurrentPrice: decimal.NewFromInt(52000),
		}

		mockCache.On("GetPortfolio", ctx, userID, mock.Anything).Return(errors.New("cache miss"))
		mockPortfolioRepo.On("GetByUserID", ctx, userID).Return(portfolio, nil)
		mockUserClient.On("GetUserBalance", ctx, userID).Return(portfolio.TotalCash, nil)
		mockCache.On("SetPortfolio", ctx, userID, mock.Anything).Return(nil).Once()
		mockSnapshotRepo.On("GetByUserID", ctx, userID, 90, 0).Return([]models.Snapshot{}, nil)
		mockPortfolioRepo.On("Update", ctx, mock.AnythingOfType("*models.Portfolio")).Return(nil)
		mockCache.On("SetPortfolio", ctx, userID, mock.Anything).Return(nil).Once()
		mockCache.On("InvalidatePortfolio", ctx, userID).Return(nil)

		err := service.AddHolding(ctx, userID, newHolding)

		assert.NoError(t, err)

		mockPortfolioRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})
}

// Test RemoveHolding
func TestPortfolioService_RemoveHolding(t *testing.T) {
	ctx := context.Background()

	t.Run("remove holding successfully", func(t *testing.T) {
		mockPortfolioRepo := new(MockPortfolioRepository)
		mockSnapshotRepo := new(MockSnapshotRepository)
		mockCache := new(MockRedisClient)
		mockUserClient := new(MockUserClient)
		mockPnL := new(MockPnLCalculator)
		mockRisk := new(MockRiskCalculator)
		mockROI := new(MockROICalculator)
		mockAnalyzer := new(MockPortfolioAnalyzer)
		mockOptimizer := new(MockPortfolioOptimizer)

		service := NewPortfolioService(
			mockPortfolioRepo,
			mockSnapshotRepo,
			mockCache,
			mockUserClient,
			mockPnL,
			mockRisk,
			mockROI,
			mockAnalyzer,
			mockOptimizer,
		)

		userID := int64(1)
		portfolio := &models.Portfolio{
			ID:        primitive.NewObjectID(),
			UserID:    userID,
			TotalCash: decimal.NewFromInt(10000),
			Holdings: []models.Holding{
				{
					Symbol:       "BTC",
					Name:         "Bitcoin",
					Quantity:     decimal.NewFromFloat(0.5),
					AverageCost:  decimal.NewFromInt(50000),
					CurrentPrice: decimal.NewFromInt(52000),
				},
			},
		}

		mockCache.On("GetPortfolio", ctx, userID, mock.Anything).Return(errors.New("cache miss"))
		mockPortfolioRepo.On("GetByUserID", ctx, userID).Return(portfolio, nil)
		mockUserClient.On("GetUserBalance", ctx, userID).Return(portfolio.TotalCash, nil)
		mockCache.On("SetPortfolio", ctx, userID, mock.Anything).Return(nil).Twice()
		mockSnapshotRepo.On("GetByUserID", ctx, userID, 90, 0).Return([]models.Snapshot{}, nil)
		mockPortfolioRepo.On("Update", ctx, mock.AnythingOfType("*models.Portfolio")).Return(nil)
		mockCache.On("InvalidatePortfolio", ctx, userID).Return(nil)

		err := service.RemoveHolding(ctx, userID, "BTC")

		assert.NoError(t, err)

		mockPortfolioRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})
}

// Test DeletePortfolio
func TestPortfolioService_DeletePortfolio(t *testing.T) {
	ctx := context.Background()

	t.Run("successful portfolio deletion", func(t *testing.T) {
		mockPortfolioRepo := new(MockPortfolioRepository)
		mockSnapshotRepo := new(MockSnapshotRepository)
		mockCache := new(MockRedisClient)
		mockUserClient := new(MockUserClient)
		mockPnL := new(MockPnLCalculator)
		mockRisk := new(MockRiskCalculator)
		mockROI := new(MockROICalculator)
		mockAnalyzer := new(MockPortfolioAnalyzer)
		mockOptimizer := new(MockPortfolioOptimizer)

		service := NewPortfolioService(
			mockPortfolioRepo,
			mockSnapshotRepo,
			mockCache,
			mockUserClient,
			mockPnL,
			mockRisk,
			mockROI,
			mockAnalyzer,
			mockOptimizer,
		)

		userID := int64(1)

		mockSnapshotRepo.On("DeleteByUserID", ctx, userID).Return(nil)
		mockPortfolioRepo.On("DeleteByUserID", ctx, userID).Return(nil)
		mockCache.On("InvalidatePortfolio", ctx, userID).Return(nil)

		err := service.DeletePortfolio(ctx, userID)

		assert.NoError(t, err)

		mockSnapshotRepo.AssertExpectations(t)
		mockPortfolioRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("error deleting snapshots", func(t *testing.T) {
		mockPortfolioRepo := new(MockPortfolioRepository)
		mockSnapshotRepo := new(MockSnapshotRepository)
		mockCache := new(MockRedisClient)
		mockUserClient := new(MockUserClient)
		mockPnL := new(MockPnLCalculator)
		mockRisk := new(MockRiskCalculator)
		mockROI := new(MockROICalculator)
		mockAnalyzer := new(MockPortfolioAnalyzer)
		mockOptimizer := new(MockPortfolioOptimizer)

		service := NewPortfolioService(
			mockPortfolioRepo,
			mockSnapshotRepo,
			mockCache,
			mockUserClient,
			mockPnL,
			mockRisk,
			mockROI,
			mockAnalyzer,
			mockOptimizer,
		)

		userID := int64(1)

		mockSnapshotRepo.On("DeleteByUserID", ctx, userID).Return(errors.New("database error"))

		err := service.DeletePortfolio(ctx, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete snapshots")

		mockSnapshotRepo.AssertExpectations(t)
	})
}
