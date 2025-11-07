package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"market-data-api/internal/models"
	"market-data-api/internal/providers"
)

// Mock ProviderManager
type MockProviderManager struct {
	mock.Mock
}

func (m *MockProviderManager) GetPrice(ctx context.Context, symbol string, provider string) (*models.Price, error) {
	args := m.Called(ctx, symbol, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Price), args.Error(1)
}

func (m *MockProviderManager) GetMultiplePrices(ctx context.Context, symbol string) (map[string]*models.Price, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*models.Price), args.Error(1)
}

func (m *MockProviderManager) GetActiveProviders() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

// Test NewService
func TestNewService(t *testing.T) {
	t.Run("create service with default config", func(t *testing.T) {
		mockProviderManager := new(MockProviderManager)
		service := NewService(mockProviderManager, nil)

		assert.NotNil(t, service)
		assert.NotNil(t, service.aggregator)
		assert.NotNil(t, service.technicalAnalyzer)
		assert.NotNil(t, service.config)
		assert.NotNil(t, service.metrics)
	})

	t.Run("create service with custom config", func(t *testing.T) {
		mockProviderManager := new(MockProviderManager)
		config := &ServiceConfig{
			EnableBackgroundProcessing: false,
			EnablePrecomputation:       false,
			EnableQualityChecks:        false,
			MaxConcurrentRequests:      20,
		}
		service := NewService(mockProviderManager, config)

		assert.NotNil(t, service)
		assert.Equal(t, 20, service.config.MaxConcurrentRequests)
	})
}

// Test GetAggregatedPrice
func TestService_GetAggregatedPrice(t *testing.T) {
	ctx := context.Background()

	t.Run("successful price aggregation", func(t *testing.T) {
		mockProviderManager := new(MockProviderManager)
		config := &ServiceConfig{
			EnableBackgroundProcessing: false,
			MaxConcurrentRequests:      10,
		}
		service := NewService(mockProviderManager, config)

		// Mock price data from multiple providers
		priceMap := map[string]*models.Price{
			"binance": {
				Price:     decimal.NewFromInt(50000),
				Volume24h:    decimal.NewFromInt(1000),
				Timestamp: time.Now(),
			},
			"coinbase": {
				Price:     decimal.NewFromInt(50100),
				Volume24h:    decimal.NewFromInt(900),
				Timestamp: time.Now(),
			},
			"coingecko": {
				Price:     decimal.NewFromInt(49900),
				Volume24h:    decimal.NewFromInt(1100),
				Timestamp: time.Now(),
			},
		}

		mockProviderManager.On("GetMultiplePrices", ctx, "BTC").Return(priceMap, nil)
		mockProviderManager.On("GetActiveProviders").Return([]string{"binance", "coinbase", "coingecko"})

		options := &PriceOptions{
			IncludeTechnicalAnalysis: false,
			IncludeMarketSentiment:   false,
			IncludeVolatility:        false,
		}

		result, err := service.GetAggregatedPrice(ctx, "BTC", options)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "BTC", result.Symbol)
		assert.True(t, result.Price.GreaterThan(decimal.Zero))
		assert.True(t, result.QualityScore > 0)

		mockProviderManager.AssertExpectations(t)
	})

	t.Run("no providers available", func(t *testing.T) {
		mockProviderManager := new(MockProviderManager)
		config := &ServiceConfig{
			EnableBackgroundProcessing: false,
			MaxConcurrentRequests:      10,
		}
		service := NewService(mockProviderManager, config)

		emptyMap := make(map[string]*models.Price)
		mockProviderManager.On("GetMultiplePrices", ctx, "INVALID").Return(emptyMap, errors.New("no providers"))

		options := &PriceOptions{}
		result, err := service.GetAggregatedPrice(ctx, "INVALID", options)

		assert.Error(t, err)
		assert.Nil(t, result)

		mockProviderManager.AssertExpectations(t)
	})

	t.Run("with technical analysis", func(t *testing.T) {
		mockProviderManager := new(MockProviderManager)
		config := &ServiceConfig{
			EnableBackgroundProcessing: false,
			MaxConcurrentRequests:      10,
		}
		service := NewService(mockProviderManager, config)

		priceMap := map[string]*models.Price{
			"binance": {
				Price:     decimal.NewFromInt(50000),
				Volume24h:    decimal.NewFromInt(1000),
				Timestamp: time.Now(),
			},
		}

		mockProviderManager.On("GetMultiplePrices", ctx, "BTC").Return(priceMap, nil)
		mockProviderManager.On("GetActiveProviders").Return([]string{"binance"})

		options := &PriceOptions{
			IncludeTechnicalAnalysis: true,
			TechnicalPeriod:         "24h",
		}

		result, err := service.GetAggregatedPrice(ctx, "BTC", options)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Technical analysis might fail without historical data, but should not error the main call

		mockProviderManager.AssertExpectations(t)
	})
}

// Test GetBatchAggregatedPrices
func TestService_GetBatchAggregatedPrices(t *testing.T) {
	ctx := context.Background()

	t.Run("successful batch price retrieval", func(t *testing.T) {
		mockProviderManager := new(MockProviderManager)
		config := &ServiceConfig{
			EnableBackgroundProcessing: false,
			MaxConcurrentRequests:      10,
		}
		service := NewService(mockProviderManager, config)

		symbols := []string{"BTC", "ETH", "ADA"}

		btcPriceMap := map[string]*models.Price{
			"binance": {
				Price:     decimal.NewFromInt(50000),
				Volume24h:    decimal.NewFromInt(1000),
				Timestamp: time.Now(),
			},
		}

		ethPriceMap := map[string]*models.Price{
			"binance": {
				Price:     decimal.NewFromInt(3000),
				Volume24h:    decimal.NewFromInt(5000),
				Timestamp: time.Now(),
			},
		}

		adaPriceMap := map[string]*models.Price{
			"binance": {
				Price:     decimal.NewFromFloat(1.5),
				Volume24h:    decimal.NewFromInt(10000),
				Timestamp: time.Now(),
			},
		}

		mockProviderManager.On("GetMultiplePrices", ctx, "BTC").Return(btcPriceMap, nil)
		mockProviderManager.On("GetMultiplePrices", ctx, "ETH").Return(ethPriceMap, nil)
		mockProviderManager.On("GetMultiplePrices", ctx, "ADA").Return(adaPriceMap, nil)
		mockProviderManager.On("GetActiveProviders").Return([]string{"binance"})

		options := &PriceOptions{}
		results, err := service.GetBatchAggregatedPrices(ctx, symbols, options)

		assert.NoError(t, err)
		assert.NotNil(t, results)
		assert.Equal(t, 3, len(results))
		assert.Contains(t, results, "BTC")
		assert.Contains(t, results, "ETH")
		assert.Contains(t, results, "ADA")

		mockProviderManager.AssertExpectations(t)
	})

	t.Run("empty symbols list", func(t *testing.T) {
		mockProviderManager := new(MockProviderManager)
		config := &ServiceConfig{
			EnableBackgroundProcessing: false,
			MaxConcurrentRequests:      10,
		}
		service := NewService(mockProviderManager, config)

		symbols := []string{}
		options := &PriceOptions{}

		results, err := service.GetBatchAggregatedPrices(ctx, symbols, options)

		assert.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("partial success with some failures", func(t *testing.T) {
		mockProviderManager := new(MockProviderManager)
		config := &ServiceConfig{
			EnableBackgroundProcessing: false,
			MaxConcurrentRequests:      10,
		}
		service := NewService(mockProviderManager, config)

		symbols := []string{"BTC", "INVALID"}

		btcPriceMap := map[string]*models.Price{
			"binance": {
				Price:     decimal.NewFromInt(50000),
				Volume24h:    decimal.NewFromInt(1000),
				Timestamp: time.Now(),
			},
		}

		mockProviderManager.On("GetMultiplePrices", ctx, "BTC").Return(btcPriceMap, nil)
		mockProviderManager.On("GetMultiplePrices", ctx, "INVALID").Return(nil, errors.New("not found"))
		mockProviderManager.On("GetActiveProviders").Return([]string{"binance"})

		options := &PriceOptions{}
		results, err := service.GetBatchAggregatedPrices(ctx, symbols, options)

		assert.NoError(t, err)
		assert.NotNil(t, results)
		assert.Equal(t, 1, len(results))
		assert.Contains(t, results, "BTC")

		mockProviderManager.AssertExpectations(t)
	})
}

// Test GetMarketOverview
func TestService_GetMarketOverview(t *testing.T) {
	ctx := context.Background()

	t.Run("successful market overview", func(t *testing.T) {
		mockProviderManager := new(MockProviderManager)
		config := &ServiceConfig{
			EnableBackgroundProcessing: false,
			PopularSymbols:             []string{"BTC", "ETH"},
			MaxConcurrentRequests:      10,
		}
		service := NewService(mockProviderManager, config)

		btcPriceMap := map[string]*models.Price{
			"binance": {
				Price:     decimal.NewFromInt(50000),
				Volume24h:    decimal.NewFromInt(1000),
				Timestamp: time.Now(),
			},
		}

		ethPriceMap := map[string]*models.Price{
			"binance": {
				Price:     decimal.NewFromInt(3000),
				Volume24h:    decimal.NewFromInt(5000),
				Timestamp: time.Now(),
			},
		}

		mockProviderManager.On("GetMultiplePrices", ctx, "BTC").Return(btcPriceMap, nil)
		mockProviderManager.On("GetMultiplePrices", ctx, "ETH").Return(ethPriceMap, nil)
		mockProviderManager.On("GetActiveProviders").Return([]string{"binance"})

		overview, err := service.GetMarketOverview(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, overview)
		assert.Equal(t, 2, overview.TotalSymbols)
		assert.NotNil(t, overview.MarketPrices)
		assert.NotNil(t, overview.Statistics)

		mockProviderManager.AssertExpectations(t)
	})
}

// Test calculateQualityScore
func TestService_calculateQualityScore(t *testing.T) {
	mockProviderManager := new(MockProviderManager)
	config := &ServiceConfig{
		EnableBackgroundProcessing: false,
	}
	service := NewService(mockProviderManager, config)

	t.Run("high quality score with multiple providers", func(t *testing.T) {
		price := &models.AggregatedPrice{
			Symbol:     "BTC",
			Price:      decimal.NewFromInt(50000),
			Volume24h:     decimal.NewFromInt(1000000),
			Confidence: 0.95,
			Timestamp:  time.Now(),
			Metadata: &models.AggregationMetadata{
				ProvidersUsed: 5,
			},
		}

		score := service.calculateQualityScore(price)

		assert.True(t, score > 0.7)
		assert.True(t, score <= 1.0)
	})

	t.Run("low quality score with single provider", func(t *testing.T) {
		price := &models.AggregatedPrice{
			Symbol:     "BTC",
			Price:      decimal.NewFromInt(50000),
			Volume24h:     decimal.NewFromInt(100),
			Confidence: 0.5,
			Timestamp:  time.Now().Add(-2 * time.Hour),
			Metadata: &models.AggregationMetadata{
				ProvidersUsed: 1,
			},
		}

		score := service.calculateQualityScore(price)

		assert.True(t, score < 0.7)
		assert.True(t, score >= 0)
	})
}

// Test calculateMarketSentiment
func TestService_calculateMarketSentiment(t *testing.T) {
	mockProviderManager := new(MockProviderManager)
	config := &ServiceConfig{
		EnableBackgroundProcessing: false,
	}
	service := NewService(mockProviderManager, config)

	t.Run("bullish sentiment with low variance", func(t *testing.T) {
		price := &models.AggregatedPrice{
			Symbol: "BTC",
			Price:  decimal.NewFromInt(50000),
			ProviderPrices: []models.ProviderPrice{
				{Provider: "binance", Price: decimal.NewFromInt(50000)},
				{Provider: "coinbase", Price: decimal.NewFromInt(50010)},
				{Provider: "coingecko", Price: decimal.NewFromInt(49990)},
			},
			Timestamp: time.Now(),
		}

		sentiment := service.calculateMarketSentiment(price)

		assert.NotNil(t, sentiment)
		assert.Equal(t, "BTC", sentiment.Symbol)
		assert.Contains(t, []string{"BULLISH", "NEUTRAL", "BEARISH"}, sentiment.Sentiment)
		assert.True(t, sentiment.Score >= 0 && sentiment.Score <= 1)
	})

	t.Run("neutral sentiment with single provider", func(t *testing.T) {
		price := &models.AggregatedPrice{
			Symbol: "ETH",
			Price:  decimal.NewFromInt(3000),
			ProviderPrices: []models.ProviderPrice{
				{Provider: "binance", Price: decimal.NewFromInt(3000)},
			},
			Timestamp: time.Now(),
		}

		sentiment := service.calculateMarketSentiment(price)

		assert.NotNil(t, sentiment)
		assert.Equal(t, "NEUTRAL", sentiment.Sentiment)
		assert.Equal(t, 0.5, sentiment.Score)
	})
}

// Test GetMetrics
func TestService_GetMetrics(t *testing.T) {
	mockProviderManager := new(MockProviderManager)
	config := &ServiceConfig{
		EnableBackgroundProcessing: false,
	}
	service := NewService(mockProviderManager, config)

	t.Run("get initial metrics", func(t *testing.T) {
		metrics := service.GetMetrics()

		assert.NotNil(t, metrics)
		assert.Equal(t, int64(0), metrics.TotalRequests)
		assert.Equal(t, int64(0), metrics.SuccessfulRequests)
		assert.Equal(t, int64(0), metrics.FailedRequests)
	})

	t.Run("metrics update after requests", func(t *testing.T) {
		ctx := context.Background()

		priceMap := map[string]*models.Price{
			"binance": {
				Price:     decimal.NewFromInt(50000),
				Volume24h:    decimal.NewFromInt(1000),
				Timestamp: time.Now(),
			},
		}

		mockProviderManager.On("GetMultiplePrices", ctx, "BTC").Return(priceMap, nil)
		mockProviderManager.On("GetActiveProviders").Return([]string{"binance"})

		options := &PriceOptions{}
		_, _ = service.GetAggregatedPrice(ctx, "BTC", options)

		metrics := service.GetMetrics()

		assert.NotNil(t, metrics)
		assert.True(t, metrics.TotalRequests > 0)
	})
}

// Test Stop
func TestService_Stop(t *testing.T) {
	t.Run("stop service gracefully", func(t *testing.T) {
		mockProviderManager := new(MockProviderManager)
		config := &ServiceConfig{
			EnableBackgroundProcessing: false,
		}
		service := NewService(mockProviderManager, config)

		// Should not panic
		assert.NotPanics(t, func() {
			service.Stop()
		})
	})
}
