package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"search-api/internal/cache"
	"search-api/internal/dto"
	"search-api/internal/models"
	"search-api/internal/repositories"
)

// Mock implementations
type MockSearchRepository struct {
	mock.Mock
}

func (m *MockSearchRepository) Search(ctx context.Context, req *dto.SearchRequest) (*repositories.SearchResult, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*repositories.SearchResult), args.Error(1)
}

func (m *MockSearchRepository) SearchTrending(ctx context.Context, period string, limit int) ([]models.TrendingCrypto, error) {
	args := m.Called(ctx, period, limit)
	return args.Get(0).([]models.TrendingCrypto), args.Error(1)
}

func (m *MockSearchRepository) GetSuggestions(ctx context.Context, query string, limit int) ([]models.Suggestion, error) {
	args := m.Called(ctx, query, limit)
	return args.Get(0).([]models.Suggestion), args.Error(1)
}

func (m *MockSearchRepository) GetByID(ctx context.Context, id string) (*models.Crypto, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Crypto), args.Error(1)
}

func (m *MockSearchRepository) GetFacets(ctx context.Context) (*models.Filter, error) {
	args := m.Called(ctx)
	return args.Get(0).(*models.Filter), args.Error(1)
}

func (m *MockSearchRepository) IndexCrypto(ctx context.Context, crypto *models.Crypto) error {
	args := m.Called(ctx, crypto)
	return args.Error(0)
}

func (m *MockSearchRepository) IndexBatch(ctx context.Context, cryptos []*models.Crypto) error {
	args := m.Called(ctx, cryptos)
	return args.Error(0)
}

func (m *MockSearchRepository) UpdateTrendingScore(ctx context.Context, cryptoID string, score float32) error {
	args := m.Called(ctx, cryptoID, score)
	return args.Error(0)
}

func (m *MockSearchRepository) DeleteCrypto(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSearchRepository) GetDocumentCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSearchRepository) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockCachedSearchRepository struct {
	mock.Mock
}

func (m *MockCachedSearchRepository) GetSearchResults(ctx context.Context, req *dto.SearchRequest) (*repositories.SearchResult, bool) {
	args := m.Called(ctx, req)
	return args.Get(0).(*repositories.SearchResult), args.Bool(1)
}

func (m *MockCachedSearchRepository) SetSearchResults(ctx context.Context, req *dto.SearchRequest, result *repositories.SearchResult) error {
	args := m.Called(ctx, req, result)
	return args.Error(0)
}

func (m *MockCachedSearchRepository) GetTrendingResults(ctx context.Context, period string, limit int) ([]models.TrendingCrypto, bool) {
	args := m.Called(ctx, period, limit)
	return args.Get(0).([]models.TrendingCrypto), args.Bool(1)
}

func (m *MockCachedSearchRepository) SetTrendingResults(ctx context.Context, period string, limit int, trending []models.TrendingCrypto) error {
	args := m.Called(ctx, period, limit, trending)
	return args.Error(0)
}

func (m *MockCachedSearchRepository) GetSuggestions(ctx context.Context, query string, limit int) ([]models.Suggestion, bool) {
	args := m.Called(ctx, query, limit)
	return args.Get(0).([]models.Suggestion), args.Bool(1)
}

func (m *MockCachedSearchRepository) SetSuggestions(ctx context.Context, query string, limit int, suggestions []models.Suggestion) error {
	args := m.Called(ctx, query, limit, suggestions)
	return args.Error(0)
}

func (m *MockCachedSearchRepository) GetCrypto(ctx context.Context, id string) (*models.Crypto, bool) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Crypto), args.Bool(1)
}

func (m *MockCachedSearchRepository) SetCrypto(ctx context.Context, crypto *models.Crypto) error {
	args := m.Called(ctx, crypto)
	return args.Error(0)
}

func (m *MockCachedSearchRepository) GetFilters(ctx context.Context) (*models.Filter, bool) {
	args := m.Called(ctx)
	return args.Get(0).(*models.Filter), args.Bool(1)
}

func (m *MockCachedSearchRepository) SetFilters(ctx context.Context, filters *models.Filter) error {
	args := m.Called(ctx, filters)
	return args.Error(0)
}

func (m *MockCachedSearchRepository) InvalidateSearch(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockCachedSearchRepository) InvalidateAll(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCachedSearchRepository) GetStats() *cache.CacheStats {
	args := m.Called()
	return args.Get(0).(*cache.CacheStats)
}

func (m *MockCachedSearchRepository) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockTrendingService struct {
	mock.Mock
}

func (m *MockTrendingService) GetTrendingScore(cryptoID string) (float32, bool) {
	args := m.Called(cryptoID)
	return args.Get(0).(float32), args.Bool(1)
}

func (m *MockTrendingService) UpdateTrendingScore(cryptoID string, eventType string, value float64) {
	m.Called(cryptoID, eventType, value)
}

func TestSearchService_Search(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce log noise in tests

	t.Run("cache hit", func(t *testing.T) {
		mockSolrRepo := new(MockSearchRepository)
		mockCacheRepo := new(MockCachedSearchRepository)
		mockTrendingService := new(MockTrendingService)

		service := NewSearchService(mockSolrRepo, mockCacheRepo, mockTrendingService, logger)

		req := &dto.SearchRequest{
			Query: "bitcoin",
			Page:  1,
			Limit: 20,
		}
		req.SetDefaults()

		expectedResult := &repositories.SearchResult{
			Results: []*models.Crypto{
				{
					ID:     "bitcoin",
					Symbol: "BTC",
					Name:   "Bitcoin",
				},
			},
			Total: 1,
		}

		mockCacheRepo.On("GetSearchResults", mock.Anything, req).Return(expectedResult, true)

		result, err := service.Search(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, len(result.Results))
		assert.Equal(t, "bitcoin", result.Results[0].ID)
		assert.True(t, result.QueryInfo.CacheHit)

		mockCacheRepo.AssertExpectations(t)
		mockSolrRepo.AssertNotCalled(t, "Search")
	})

	t.Run("cache miss", func(t *testing.T) {
		mockSolrRepo := new(MockSearchRepository)
		mockCacheRepo := new(MockCachedSearchRepository)
		mockTrendingService := new(MockTrendingService)

		service := NewSearchService(mockSolrRepo, mockCacheRepo, mockTrendingService, logger)

		req := &dto.SearchRequest{
			Query: "ethereum",
			Page:  1,
			Limit: 20,
		}
		req.SetDefaults()

		solrResult := &repositories.SearchResult{
			Results: []*models.Crypto{
				{
					ID:     "ethereum",
					Symbol: "ETH",
					Name:   "Ethereum",
				},
			},
			Total: 1,
		}

		mockCacheRepo.On("GetSearchResults", mock.Anything, req).Return((*repositories.SearchResult)(nil), false)
		mockSolrRepo.On("Search", mock.Anything, req).Return(solrResult, nil)
		mockCacheRepo.On("SetSearchResults", mock.Anything, req, solrResult).Return(nil)

		result, err := service.Search(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, len(result.Results))
		assert.Equal(t, "ethereum", result.Results[0].ID)
		assert.False(t, result.QueryInfo.CacheHit)

		mockCacheRepo.AssertExpectations(t)
		mockSolrRepo.AssertExpectations(t)
	})

	t.Run("search error", func(t *testing.T) {
		mockSolrRepo := new(MockSearchRepository)
		mockCacheRepo := new(MockCachedSearchRepository)
		mockTrendingService := new(MockTrendingService)

		service := NewSearchService(mockSolrRepo, mockCacheRepo, mockTrendingService, logger)

		req := &dto.SearchRequest{
			Query: "error",
			Page:  1,
			Limit: 20,
		}
		req.SetDefaults()

		mockCacheRepo.On("GetSearchResults", mock.Anything, req).Return((*repositories.SearchResult)(nil), false)
		mockSolrRepo.On("Search", mock.Anything, req).Return((*repositories.SearchResult)(nil), errors.New("solr error"))

		result, err := service.Search(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "search failed")

		mockCacheRepo.AssertExpectations(t)
		mockSolrRepo.AssertExpectations(t)
	})

	t.Run("invalid request", func(t *testing.T) {
		mockSolrRepo := new(MockSearchRepository)
		mockCacheRepo := new(MockCachedSearchRepository)
		mockTrendingService := new(MockTrendingService)

		service := NewSearchService(mockSolrRepo, mockCacheRepo, mockTrendingService, logger)

		req := &dto.SearchRequest{
			Query: "bitcoin",
			Page:  0, // Invalid page
			Limit: 20,
		}

		result, err := service.Search(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid search request")

		// No repository methods should be called for invalid requests
		mockCacheRepo.AssertNotCalled(t, "GetSearchResults")
		mockSolrRepo.AssertNotCalled(t, "Search")
	})
}

func TestSearchService_GetTrending(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("successful trending request", func(t *testing.T) {
		mockSolrRepo := new(MockSearchRepository)
		mockCacheRepo := new(MockCachedSearchRepository)
		mockTrendingService := new(MockTrendingService)

		service := NewSearchService(mockSolrRepo, mockCacheRepo, mockTrendingService, logger)

		req := &dto.TrendingRequest{
			Period: "24h",
			Limit:  10,
		}

		expectedTrending := []models.TrendingCrypto{
			{
				ID:            "bitcoin",
				Symbol:        "BTC",
				Name:          "Bitcoin",
				Rank:          1,
				TrendingScore: 95.5,
			},
		}

		mockCacheRepo.On("GetTrendingResults", mock.Anything, "24h", 10).Return([]models.TrendingCrypto(nil), false)
		mockSolrRepo.On("SearchTrending", mock.Anything, "24h", 10).Return(expectedTrending, nil)
		mockCacheRepo.On("SetTrendingResults", mock.Anything, "24h", 10, expectedTrending).Return(nil)

		// Mock trending service calls for enhancement
		mockTrendingService.On("GetTrendingScore", "bitcoin").Return(float32(95.5), true)

		result, err := service.GetTrending(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, len(result.Results))
		assert.Equal(t, "bitcoin", result.Results[0].ID)

		mockCacheRepo.AssertExpectations(t)
		mockSolrRepo.AssertExpectations(t)
		mockTrendingService.AssertExpectations(t)
	})
}

func TestSearchService_GetSuggestions(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("successful suggestions request", func(t *testing.T) {
		mockSolrRepo := new(MockSearchRepository)
		mockCacheRepo := new(MockCachedSearchRepository)
		mockTrendingService := new(MockTrendingService)

		service := NewSearchService(mockSolrRepo, mockCacheRepo, mockTrendingService, logger)

		req := &dto.SuggestionRequest{
			Query: "bit",
			Limit: 5,
		}

		expectedSuggestions := []models.Suggestion{
			{
				ID:     "bitcoin",
				Symbol: "BTC",
				Name:   "Bitcoin",
				Score:  100.0,
				Type:   "cryptocurrency",
			},
		}

		mockCacheRepo.On("GetSuggestions", mock.Anything, "bit", 5).Return([]models.Suggestion(nil), false)
		mockSolrRepo.On("GetSuggestions", mock.Anything, "bit", 5).Return(expectedSuggestions, nil)
		mockCacheRepo.On("SetSuggestions", mock.Anything, "bit", 5, mock.AnythingOfType("[]models.Suggestion")).Return(nil)

		result, err := service.GetSuggestions(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, len(result.Suggestions))
		assert.Equal(t, "Bitcoin", result.Suggestions[0].Name)

		mockCacheRepo.AssertExpectations(t)
		mockSolrRepo.AssertExpectations(t)
	})
}

func TestSearchService_GetCryptoByID(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("successful crypto lookup", func(t *testing.T) {
		mockSolrRepo := new(MockSearchRepository)
		mockCacheRepo := new(MockCachedSearchRepository)
		mockTrendingService := new(MockTrendingService)

		service := NewSearchService(mockSolrRepo, mockCacheRepo, mockTrendingService, logger)

		expectedCrypto := &models.Crypto{
			ID:     "bitcoin",
			Symbol: "BTC",
			Name:   "Bitcoin",
		}

		mockCacheRepo.On("GetCrypto", mock.Anything, "bitcoin").Return((*models.Crypto)(nil), false)
		mockSolrRepo.On("GetByID", mock.Anything, "bitcoin").Return(expectedCrypto, nil)
		mockCacheRepo.On("SetCrypto", mock.Anything, expectedCrypto).Return(nil)

		result, err := service.GetCryptoByID(context.Background(), "bitcoin")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "bitcoin", result.ID)
		assert.Equal(t, "BTC", result.Symbol)

		mockCacheRepo.AssertExpectations(t)
		mockSolrRepo.AssertExpectations(t)
	})

	t.Run("crypto not found", func(t *testing.T) {
		mockSolrRepo := new(MockSearchRepository)
		mockCacheRepo := new(MockCachedSearchRepository)
		mockTrendingService := new(MockTrendingService)

		service := NewSearchService(mockSolrRepo, mockCacheRepo, mockTrendingService, logger)

		mockCacheRepo.On("GetCrypto", mock.Anything, "nonexistent").Return((*models.Crypto)(nil), false)
		mockSolrRepo.On("GetByID", mock.Anything, "nonexistent").Return((*models.Crypto)(nil), errors.New("not found"))

		result, err := service.GetCryptoByID(context.Background(), "nonexistent")

		assert.Error(t, err)
		assert.Nil(t, result)

		mockCacheRepo.AssertExpectations(t)
		mockSolrRepo.AssertExpectations(t)
	})
}

func TestSearchService_GetHealthStatus(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("all services healthy", func(t *testing.T) {
		mockSolrRepo := new(MockSearchRepository)
		mockCacheRepo := new(MockCachedSearchRepository)
		mockTrendingService := new(MockTrendingService)

		service := NewSearchService(mockSolrRepo, mockCacheRepo, mockTrendingService, logger)

		mockSolrRepo.On("Ping", mock.Anything).Return(nil)
		mockSolrRepo.On("GetDocumentCount", mock.Anything).Return(int64(1000), nil)
		mockCacheRepo.On("Ping", mock.Anything).Return(nil)
		mockCacheRepo.On("GetStats").Return(&cache.CacheStats{
			LocalHits:   100,
			LocalMisses: 20,
		})

		status := service.GetHealthStatus(context.Background())

		assert.NotNil(t, status)
		assert.True(t, status["overall_healthy"].(bool))
		assert.True(t, status["solr_healthy"].(bool))
		assert.True(t, status["cache_healthy"].(bool))
		assert.Equal(t, int64(1000), status["document_count"])

		mockSolrRepo.AssertExpectations(t)
		mockCacheRepo.AssertExpectations(t)
	})

	t.Run("solr unhealthy", func(t *testing.T) {
		mockSolrRepo := new(MockSearchRepository)
		mockCacheRepo := new(MockCachedSearchRepository)
		mockTrendingService := new(MockTrendingService)

		service := NewSearchService(mockSolrRepo, mockCacheRepo, mockTrendingService, logger)

		mockSolrRepo.On("Ping", mock.Anything).Return(errors.New("connection failed"))
		mockCacheRepo.On("Ping", mock.Anything).Return(nil)
		mockCacheRepo.On("GetStats").Return(&cache.CacheStats{})

		status := service.GetHealthStatus(context.Background())

		assert.NotNil(t, status)
		assert.False(t, status["overall_healthy"].(bool))
		assert.False(t, status["solr_healthy"].(bool))
		assert.True(t, status["cache_healthy"].(bool))

		mockSolrRepo.AssertExpectations(t)
		mockCacheRepo.AssertExpectations(t)
	})
}

func TestSearchService_GetMetrics(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockSolrRepo := new(MockSearchRepository)
	mockCacheRepo := new(MockCachedSearchRepository)
	mockTrendingService := new(MockTrendingService)

	service := NewSearchService(mockSolrRepo, mockCacheRepo, mockTrendingService, logger)

	mockCacheRepo.On("GetStats").Return(&cache.CacheStats{
		LocalHits:         100,
		LocalMisses:       25,
		DistributedHits:   50,
		DistributedMisses: 10,
	})

	metrics, err := service.GetMetrics(context.Background())

	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(185), metrics.TotalSearches) // Total requests
	assert.InDelta(t, 0.81, metrics.CacheHitRate, 0.01) // Hit rate calculation

	mockCacheRepo.AssertExpectations(t)
}

func TestSearchService_InvalidateCache(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockSolrRepo := new(MockSearchRepository)
	mockCacheRepo := new(MockCachedSearchRepository)
	mockTrendingService := new(MockTrendingService)

	service := NewSearchService(mockSolrRepo, mockCacheRepo, mockTrendingService, logger)

	mockCacheRepo.On("InvalidateSearch", mock.Anything, "bitcoin*").Return(nil)

	err := service.InvalidateCache(context.Background(), "bitcoin*")

	assert.NoError(t, err)
	mockCacheRepo.AssertExpectations(t)
}