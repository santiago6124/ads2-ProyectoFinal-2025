package services

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"search-api/internal/dto"
	"search-api/internal/models"
	"search-api/internal/repositories"
)

// SearchService handles search business logic with caching
type SearchService struct {
	solrRepo        repositories.SearchRepository
	cacheRepo       repositories.CachedSearchRepository
	trendingService *TrendingService
	logger          *logrus.Logger
}

// NewSearchService creates a new search service
func NewSearchService(
	solrRepo repositories.SearchRepository,
	cacheRepo repositories.CachedSearchRepository,
	trendingService *TrendingService,
	logger *logrus.Logger,
) *SearchService {
	return &SearchService{
		solrRepo:        solrRepo,
		cacheRepo:       cacheRepo,
		trendingService: trendingService,
		logger:          logger,
	}
}

// Search performs a comprehensive search with caching
func (s *SearchService) Search(ctx context.Context, req *dto.SearchRequest) (*dto.SearchResponse, error) {
	startTime := time.Now()

	// Validate and set defaults
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid search request: %w", err)
	}
	req.SetDefaults()

	// Try cache first
	if result, found := s.cacheRepo.GetSearchResults(ctx, req); found {
		s.logger.WithFields(logrus.Fields{
			"query": req.Query,
			"page":  req.Page,
			"cache": "hit",
		}).Debug("Search cache hit")

		return s.buildSearchResponse(result, req, true, time.Since(startTime)), nil
	}

	// Execute search against Solr
	result, err := s.solrRepo.Search(ctx, req)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"query": req.Query,
			"error": err,
		}).Error("Search execution failed")
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Cache the results asynchronously
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.cacheRepo.SetSearchResults(cacheCtx, req, result); err != nil {
			s.logger.WithFields(logrus.Fields{
				"query": req.Query,
				"error": err,
			}).Warn("Failed to cache search results")
		}
	}()

	s.logger.WithFields(logrus.Fields{
		"query":   req.Query,
		"results": len(result.Results),
		"total":   result.Total,
		"time":    time.Since(startTime),
	}).Info("Search executed")

	return s.buildSearchResponse(result, req, false, time.Since(startTime)), nil
}

// GetTrending gets trending cryptocurrencies with caching
func (s *SearchService) GetTrending(ctx context.Context, req *dto.TrendingRequest) (*dto.TrendingResponse, error) {
	req.SetDefaults()

	// Try cache first
	if trending, found := s.cacheRepo.GetTrendingResults(ctx, req.Period, req.Limit); found {
		s.logger.WithFields(logrus.Fields{
			"period": req.Period,
			"limit":  req.Limit,
			"cache":  "hit",
		}).Debug("Trending cache hit")

		return dto.BuildTrendingResponse(trending, req.Period), nil
	}

	// Get from Solr
	trending, err := s.solrRepo.SearchTrending(ctx, req.Period, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("trending search failed: %w", err)
	}

	// Update trending scores based on recent activity
	s.enhanceTrendingWithRealtimeData(trending)

	// Cache results asynchronously
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.cacheRepo.SetTrendingResults(cacheCtx, req.Period, req.Limit, trending); err != nil {
			s.logger.WithFields(logrus.Fields{
				"period": req.Period,
				"error":  err,
			}).Warn("Failed to cache trending results")
		}
	}()

	s.logger.WithFields(logrus.Fields{
		"period":  req.Period,
		"results": len(trending),
	}).Info("Trending search executed")

	return dto.BuildTrendingResponse(trending, req.Period), nil
}

// GetSuggestions gets autocomplete suggestions with caching
func (s *SearchService) GetSuggestions(ctx context.Context, req *dto.SuggestionRequest) (*dto.SuggestionsResponse, error) {
	startTime := time.Now()
	req.SetDefaults()

	// Try cache first
	if suggestions, found := s.cacheRepo.GetSuggestions(ctx, req.Query, req.Limit); found {
		s.logger.WithFields(logrus.Fields{
			"query": req.Query,
			"cache": "hit",
		}).Debug("Suggestions cache hit")

		return dto.BuildSuggestionsResponse(suggestions, req.Query, time.Since(startTime)), nil
	}

	// Get from Solr
	suggestions, err := s.solrRepo.GetSuggestions(ctx, req.Query, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("suggestions search failed: %w", err)
	}

	// Enhance suggestions with additional data
	s.enhanceSuggestions(suggestions)

	// Cache results asynchronously
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.cacheRepo.SetSuggestions(cacheCtx, req.Query, req.Limit, suggestions); err != nil {
			s.logger.WithFields(logrus.Fields{
				"query": req.Query,
				"error": err,
			}).Warn("Failed to cache suggestions")
		}
	}()

	return dto.BuildSuggestionsResponse(suggestions, req.Query, time.Since(startTime)), nil
}

// GetCryptoByID gets a single cryptocurrency by ID with caching
func (s *SearchService) GetCryptoByID(ctx context.Context, id string) (*models.Crypto, error) {
	// Try cache first
	if crypto, found := s.cacheRepo.GetCrypto(ctx, id); found {
		s.logger.WithFields(logrus.Fields{
			"id":    id,
			"cache": "hit",
		}).Debug("Crypto cache hit")

		return crypto, nil
	}

	// Get from Solr
	crypto, err := s.solrRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get crypto %s: %w", id, err)
	}

	// Cache result asynchronously
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.cacheRepo.SetCrypto(cacheCtx, crypto); err != nil {
			s.logger.WithFields(logrus.Fields{
				"id":    id,
				"error": err,
			}).Warn("Failed to cache crypto")
		}
	}()

	return crypto, nil
}

// GetOrderByID gets a single order by ID from SolR
func (s *SearchService) GetOrderByID(ctx context.Context, orderID string) (*models.Order, error) {
	// Get from Solr (orders are typically not cached individually, only search results)
	order, err := s.solrRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order %s: %w", orderID, err)
	}

	return order, nil
}

// GetFilters gets available search filters with caching
func (s *SearchService) GetFilters(ctx context.Context) (*models.OrderFilter, error) {
	// Try cache first
	if filters, found := s.cacheRepo.GetFilters(ctx); found {
		s.logger.WithField("cache", "hit").Debug("Filters cache hit")
		return filters, nil
	}

	// Get from Solr
	filters, err := s.solrRepo.GetOrderFilters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get filters: %w", err)
	}

	// Cache result asynchronously
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.cacheRepo.SetFilters(cacheCtx, filters); err != nil {
			s.logger.WithError(err).Warn("Failed to cache filters")
		}
	}()

	return filters, nil
}

// InvalidateCache invalidates search cache for a pattern
func (s *SearchService) InvalidateCache(ctx context.Context, pattern string) error {
	if err := s.cacheRepo.InvalidateSearch(ctx, pattern); err != nil {
		return fmt.Errorf("cache invalidation failed: %w", err)
	}

	s.logger.WithField("pattern", pattern).Info("Cache invalidated")
	return nil
}

// GetHealthStatus returns the health status of search service dependencies
func (s *SearchService) GetHealthStatus(ctx context.Context) map[string]interface{} {
	status := make(map[string]interface{})

	// Check Solr health
	solrHealthy := true
	if err := s.solrRepo.Ping(ctx); err != nil {
		solrHealthy = false
		status["solr_error"] = err.Error()
	}
	status["solr_healthy"] = solrHealthy

	// Check cache health
	cacheHealthy := true
	if err := s.cacheRepo.Ping(ctx); err != nil {
		cacheHealthy = false
		status["cache_error"] = err.Error()
	}
	status["cache_healthy"] = cacheHealthy

	// Get cache statistics
	status["cache_stats"] = s.cacheRepo.GetStats()

	// Get document count
	if docCount, err := s.solrRepo.GetDocumentCount(ctx); err == nil {
		status["document_count"] = docCount
	}

	status["overall_healthy"] = solrHealthy && cacheHealthy

	return status
}

// Helper methods

func (s *SearchService) buildSearchResponse(result *repositories.SearchResult, req *dto.SearchRequest, cacheHit bool, executionTime time.Duration) *dto.SearchResponse {
	// Calculate pagination
	totalPages := (result.Total + int64(req.Limit) - 1) / int64(req.Limit)
	hasNext := int64(req.Page) < totalPages
	hasPrev := req.Page > 1

	pagination := models.OrderPagination{
		Total:      result.Total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}

	queryInfo := models.OrderQueryInfo{
		Query:           req.Query,
		ExecutionTimeMS: executionTime.Milliseconds(),
		CacheHit:        cacheHit,
		TotalFound:      result.Total,
	}

	// Convert results to OrderSearchResult models
	orderResults := make([]models.OrderSearchResult, 0, len(result.Results))
	for _, r := range result.Results {
		if or, ok := r.(models.OrderSearchResult); ok {
			orderResults = append(orderResults, or)
		}
	}

	// Extract facets for orders
	var facets models.OrderFacets
	if result.Facets != nil {
		if f, ok := result.Facets.(models.OrderFacets); ok {
			facets = f
		}
	}

	return &dto.SearchResponse{
		Results:    orderResults,
		Pagination: pagination,
		Facets:     facets,
		QueryInfo:  queryInfo,
	}
}

func (s *SearchService) enhanceTrendingWithRealtimeData(trending []models.TrendingCrypto) {
	// In a real implementation, this would fetch real-time data from external sources
	// For now, we'll simulate some enhancements

	for i := range trending {
		// Add simulated search volume increase
		if s.trendingService != nil {
			if score, exists := s.trendingService.GetTrendingScore(trending[i].ID); exists {
				trending[i].TrendingScore = score
			}
		}
	}
}

func (s *SearchService) enhanceSuggestions(suggestions []models.Suggestion) {
	// Enhance suggestions with additional scoring or filtering
	// This could include popularity scoring, recent search frequency, etc.

	for i := range suggestions {
		// Boost score for popular cryptocurrencies
		if suggestions[i].Symbol == "BTC" || suggestions[i].Symbol == "ETH" {
			suggestions[i].Score += 50
		}
	}
}

// SearchMetrics represents search performance metrics
type SearchMetrics struct {
	TotalSearches       int64
	CacheHitRate        float64
	AverageResponseTime time.Duration
	PopularQueries      []string
	ErrorRate           float64
}

// GetMetrics returns search service metrics
func (s *SearchService) GetMetrics(ctx context.Context) (*SearchMetrics, error) {
	cacheStats := s.cacheRepo.GetStats()

	totalRequests := cacheStats.LocalHits + cacheStats.LocalMisses +
		cacheStats.DistributedHits + cacheStats.DistributedMisses

	var hitRate float64
	if totalRequests > 0 {
		totalHits := cacheStats.LocalHits + cacheStats.DistributedHits
		hitRate = float64(totalHits) / float64(totalRequests)
	}

	return &SearchMetrics{
		TotalSearches:       totalRequests,
		CacheHitRate:        hitRate,
		AverageResponseTime: 0,          // Would be tracked by middleware
		PopularQueries:      []string{}, // Would be tracked by analytics
		ErrorRate:           0,          // Would be tracked by error monitoring
	}, nil
}

// WarmCache pre-populates cache with popular searches
func (s *SearchService) WarmCache(ctx context.Context) error {
	warmer := repositories.NewCacheWarmer(s.cacheRepo, s.solrRepo)

	if err := warmer.WarmAll(ctx); err != nil {
		s.logger.WithError(err).Error("Cache warming failed")
		return fmt.Errorf("cache warming failed: %w", err)
	}

	s.logger.Info("Cache warming completed successfully")
	return nil
}
