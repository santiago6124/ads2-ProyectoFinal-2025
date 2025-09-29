package services

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"search-api/internal/models"
)

type MockSearchRepositoryForTrending struct {
	mock.Mock
}

func (m *MockSearchRepositoryForTrending) UpdateTrendingScore(ctx context.Context, cryptoID string, score float32) error {
	args := m.Called(ctx, cryptoID, score)
	return args.Error(0)
}

func TestTrendingService_UpdateTrendingScore(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockRepo := new(MockSearchRepositoryForTrending)
	config := DefaultTrendingConfig()
	service := NewTrendingService(mockRepo, config, logger)

	t.Run("search event", func(t *testing.T) {
		mockRepo.On("UpdateTrendingScore", mock.Anything, "bitcoin", mock.AnythingOfType("float32")).Return(nil)

		service.UpdateTrendingScore("bitcoin", "search", 1.0)

		score, exists := service.GetTrendingScore("bitcoin")
		assert.True(t, exists)
		assert.Greater(t, score, float32(0))

		mockRepo.AssertExpectations(t)
	})

	t.Run("order executed event", func(t *testing.T) {
		mockRepo.On("UpdateTrendingScore", mock.Anything, "ethereum", mock.AnythingOfType("float32")).Return(nil)

		service.UpdateTrendingScore("ethereum", "order_executed", 1000000.0) // $1M order

		score, exists := service.GetTrendingScore("ethereum")
		assert.True(t, exists)
		assert.Greater(t, score, float32(0))

		mockRepo.AssertExpectations(t)
	})

	t.Run("price change event", func(t *testing.T) {
		mockRepo.On("UpdateTrendingScore", mock.Anything, "cardano", mock.AnythingOfType("float32")).Return(nil)

		service.UpdateTrendingScore("cardano", "price_change", 15.5) // 15.5% price change

		score, exists := service.GetTrendingScore("cardano")
		assert.True(t, exists)
		assert.Greater(t, score, float32(0))

		mockRepo.AssertExpectations(t)
	})

	t.Run("mention event", func(t *testing.T) {
		mockRepo.On("UpdateTrendingScore", mock.Anything, "solana", mock.AnythingOfType("float32")).Return(nil)

		service.UpdateTrendingScore("solana", "mention", 1.0)

		score, exists := service.GetTrendingScore("solana")
		assert.True(t, exists)
		assert.Greater(t, score, float32(0))

		mockRepo.AssertExpectations(t)
	})
}

func TestTrendingService_GetTrendingScore(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockRepo := new(MockSearchRepositoryForTrending)
	config := DefaultTrendingConfig()
	service := NewTrendingService(mockRepo, config, logger)

	t.Run("existing score", func(t *testing.T) {
		mockRepo.On("UpdateTrendingScore", mock.Anything, "bitcoin", mock.AnythingOfType("float32")).Return(nil)

		// Add some activity to create a score
		service.UpdateTrendingScore("bitcoin", "search", 1.0)

		score, exists := service.GetTrendingScore("bitcoin")
		assert.True(t, exists)
		assert.Greater(t, score, float32(0))
	})

	t.Run("non-existing score", func(t *testing.T) {
		score, exists := service.GetTrendingScore("nonexistent-coin")
		assert.False(t, exists)
		assert.Equal(t, float32(0), score)
	})
}

func TestTrendingService_GetTopTrending(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockRepo := new(MockSearchRepositoryForTrending)
	config := DefaultTrendingConfig()
	service := NewTrendingService(mockRepo, config, logger)

	// Mock multiple update calls
	mockRepo.On("UpdateTrendingScore", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("float32")).Return(nil).Times(6)

	// Create some trending data
	service.UpdateTrendingScore("bitcoin", "search", 1.0)
	service.UpdateTrendingScore("bitcoin", "order_executed", 5000000.0)
	service.UpdateTrendingScore("ethereum", "search", 1.0)
	service.UpdateTrendingScore("ethereum", "price_change", 10.0)
	service.UpdateTrendingScore("cardano", "search", 1.0)
	service.UpdateTrendingScore("solana", "mention", 1.0)

	t.Run("get top 3 trending", func(t *testing.T) {
		trending := service.GetTopTrending(3, "24h")

		assert.LessOrEqual(t, len(trending), 3)

		// Check that results are sorted by score (descending)
		if len(trending) > 1 {
			for i := 1; i < len(trending); i++ {
				assert.GreaterOrEqual(t, trending[i-1].TrendingScore, trending[i].TrendingScore)
			}
		}

		// Check ranking
		for i, item := range trending {
			assert.Equal(t, i+1, item.Rank)
		}
	})

	t.Run("get all trending", func(t *testing.T) {
		trending := service.GetTopTrending(0, "24h") // 0 means no limit

		// Should return items with score > 10 (threshold)
		for _, item := range trending {
			assert.Greater(t, item.TrendingScore, float32(10))
		}
	})

	t.Run("different time periods", func(t *testing.T) {
		trending1h := service.GetTopTrending(10, "1h")
		trending24h := service.GetTopTrending(10, "24h")
		trending7d := service.GetTopTrending(10, "7d")

		// All should be valid (might be different lengths due to time filtering)
		assert.GreaterOrEqual(t, len(trending1h), 0)
		assert.GreaterOrEqual(t, len(trending24h), 0)
		assert.GreaterOrEqual(t, len(trending7d), 0)
	})

	mockRepo.AssertExpectations(t)
}

func TestTrendingService_GetTrendingMetrics(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockRepo := new(MockSearchRepositoryForTrending)
	config := DefaultTrendingConfig()
	service := NewTrendingService(mockRepo, config, logger)

	mockRepo.On("UpdateTrendingScore", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("float32")).Return(nil).Times(3)

	// Create some test data with different score levels
	service.UpdateTrendingScore("bitcoin", "search", 1.0)
	service.UpdateTrendingScore("bitcoin", "order_executed", 10000000.0) // High volume to get high score
	service.UpdateTrendingScore("ethereum", "search", 1.0)
	service.UpdateTrendingScore("cardano", "mention", 1.0)

	metrics := service.GetTrendingMetrics()

	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "total_tracked")
	assert.Contains(t, metrics, "total_trending")
	assert.Contains(t, metrics, "high_score_trending")
	assert.Contains(t, metrics, "average_score")

	totalTracked := metrics["total_tracked"].(int)
	assert.GreaterOrEqual(t, totalTracked, 0)

	averageScore := metrics["average_score"].(float32)
	assert.GreaterOrEqual(t, averageScore, float32(0))

	mockRepo.AssertExpectations(t)
}

func TestTrendingService_ScoreCalculations(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockRepo := new(MockSearchRepositoryForTrending)
	config := DefaultTrendingConfig()
	service := NewTrendingService(mockRepo, config, logger)

	t.Run("search score calculation", func(t *testing.T) {
		score1 := service.calculateSearchScore(1)
		score10 := service.calculateSearchScore(10)
		score100 := service.calculateSearchScore(100)

		// Logarithmic scaling means more searches should give higher scores
		assert.Greater(t, score10, score1)
		assert.Greater(t, score100, score10)

		// But diminishing returns
		diff1 := score10 - score1
		diff2 := score100 - score10
		assert.Greater(t, diff1, diff2)
	})

	t.Run("volume score calculation", t *testing.T) {
		score1M := service.calculateVolumeScore(1000000.0)   // $1M
		score10M := service.calculateVolumeScore(10000000.0) // $10M
		score100M := service.calculateVolumeScore(100000000.0) // $100M

		assert.Greater(t, score10M, score1M)
		assert.Greater(t, score100M, score10M)

		// Negative volume should return 0
		scoreNeg := service.calculateVolumeScore(-1000.0)
		assert.Equal(t, float32(0), scoreNeg)
	})

	t.Run("price score calculation", func(t *testing.T) {
		scorePos := service.calculatePriceScore(10.0)  // +10%
		scoreNeg := service.calculatePriceScore(-10.0) // -10%

		// Both positive and negative changes should increase trending score
		assert.Greater(t, scorePos, float32(0))
		assert.Greater(t, scoreNeg, float32(0))

		// Same magnitude changes should give same score
		assert.Equal(t, scorePos, scoreNeg)

		// Larger changes should give higher scores
		scoreLarge := service.calculatePriceScore(20.0)
		assert.Greater(t, scoreLarge, scorePos)
	})

	t.Run("mention score calculation", func(t *testing.T) {
		score1 := service.calculateMentionScore(1)
		score10 := service.calculateMentionScore(10)
		score100 := service.calculateMentionScore(100)

		assert.Greater(t, score10, score1)
		assert.Greater(t, score100, score10)

		// Zero mentions should return 0
		scoreZero := service.calculateMentionScore(0)
		assert.Equal(t, float32(0), scoreZero)
	})
}

func TestTrendingService_FinalScoreCalculation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockRepo := new(MockSearchRepositoryForTrending)
	config := DefaultTrendingConfig()
	service := NewTrendingService(mockRepo, config, logger)

	// Create test data with known values
	data := &TrendingData{
		ID:            "test",
		SearchScore:   10.0,
		VolumeScore:   20.0,
		PriceScore:    5.0,
		MentionScore:  8.0,
		LastUpdated:   time.Now(),
	}

	finalScore := service.calculateFinalScore(data)

	// Final score should be weighted combination
	expectedScore := data.SearchScore*config.SearchWeight +
		data.VolumeScore*config.VolumeWeight +
		data.PriceScore*config.PriceWeight +
		data.MentionScore*config.MentionWeight

	assert.InDelta(t, expectedScore, finalScore, 0.1)

	// Test time decay
	data.LastUpdated = time.Now().Add(-2 * time.Hour) // 2 hours ago
	decayedScore := service.calculateFinalScore(data)
	assert.Less(t, decayedScore, finalScore) // Should be lower due to decay
}

func TestTrendingService_GetPeriodCutoff(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockRepo := new(MockSearchRepositoryForTrending)
	config := DefaultTrendingConfig()
	service := NewTrendingService(mockRepo, config, logger)

	now := time.Now()

	cutoff1h := service.getPeriodCutoff("1h")
	cutoff24h := service.getPeriodCutoff("24h")
	cutoff7d := service.getPeriodCutoff("7d")
	cutoff30d := service.getPeriodCutoff("30d")
	cutoffDefault := service.getPeriodCutoff("invalid")

	// All cutoffs should be in the past
	assert.True(t, cutoff1h.Before(now))
	assert.True(t, cutoff24h.Before(now))
	assert.True(t, cutoff7d.Before(now))
	assert.True(t, cutoff30d.Before(now))

	// Longer periods should have earlier cutoffs
	assert.True(t, cutoff30d.Before(cutoff7d))
	assert.True(t, cutoff7d.Before(cutoff24h))
	assert.True(t, cutoff24h.Before(cutoff1h))

	// Invalid period should default to 24h
	assert.Equal(t, cutoff24h.Unix(), cutoffDefault.Unix())
}

func TestDefaultTrendingConfig(t *testing.T) {
	config := DefaultTrendingConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 5*time.Minute, config.UpdateInterval)
	assert.Equal(t, 100, config.MaxTrendingItems)
	assert.Equal(t, float32(0.95), config.ScoreDecayRate)

	// Check weights sum to reasonable value
	totalWeight := config.VolumeWeight + config.SearchWeight + config.MentionWeight + config.PriceWeight
	assert.InDelta(t, 1.0, totalWeight, 0.1) // Should be close to 1.0

	assert.Greater(t, config.MinSearchThreshold, int64(0))
}

func TestTrendingEventHandler(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockRepo := new(MockSearchRepositoryForTrending)
	config := DefaultTrendingConfig()
	trendingService := NewTrendingService(mockRepo, config, logger)
	handler := NewTrendingEventHandler(trendingService, logger)

	t.Run("handle order event", func(t *testing.T) {
		mockRepo.On("UpdateTrendingScore", mock.Anything, "bitcoin", mock.AnythingOfType("float32")).Return(nil)

		handler.HandleOrderEvent("bitcoin", 5000000.0) // $5M order

		score, exists := trendingService.GetTrendingScore("bitcoin")
		assert.True(t, exists)
		assert.Greater(t, score, float32(0))

		mockRepo.AssertExpectations(t)
	})

	t.Run("handle search event", func(t *testing.T) {
		mockRepo.On("UpdateTrendingScore", mock.Anything, "ethereum", mock.AnythingOfType("float32")).Return(nil)

		handler.HandleSearchEvent("ethereum")

		score, exists := trendingService.GetTrendingScore("ethereum")
		assert.True(t, exists)
		assert.Greater(t, score, float32(0))

		mockRepo.AssertExpectations(t)
	})

	t.Run("handle price change event", func(t *testing.T) {
		mockRepo.On("UpdateTrendingScore", mock.Anything, "cardano", mock.AnythingOfType("float32")).Return(nil)

		handler.HandlePriceChangeEvent("cardano", 12.5)

		score, exists := trendingService.GetTrendingScore("cardano")
		assert.True(t, exists)
		assert.Greater(t, score, float32(0))

		mockRepo.AssertExpectations(t)
	})
}