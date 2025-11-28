package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"search-api/internal/clients"
	"search-api/internal/dto"
	"search-api/internal/models"
	"search-api/internal/repositories"
)

// SearchService handles search business logic with caching
type SearchService struct {
	solrRepo        repositories.SearchRepository
	cacheRepo       repositories.CachedSearchRepository
	trendingService *TrendingService
	ordersClient    *clients.OrdersClient
	logger          *logrus.Logger
}

// NewSearchService creates a new search service
func NewSearchService(
	solrRepo repositories.SearchRepository,
	cacheRepo repositories.CachedSearchRepository,
	trendingService *TrendingService,
	ordersClient *clients.OrdersClient,
	logger *logrus.Logger,
) *SearchService {
	return &SearchService{
		solrRepo:        solrRepo,
		cacheRepo:       cacheRepo,
		trendingService: trendingService,
		ordersClient:    ordersClient,
		logger:          logger,
	}
}

// Search performs a comprehensive search with caching and fallback
func (s *SearchService) Search(ctx context.Context, req *dto.SearchRequest) (*dto.SearchResponse, error) {
	startTime := time.Now()

	// Validate and set defaults
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid search request: %w", err)
	}
	req.SetDefaults()

	// Level 1: Try fresh cache first (5 min TTL)
	if result, found := s.cacheRepo.GetSearchResults(ctx, req); found {
		s.logger.WithFields(logrus.Fields{
			"query": req.Query,
			"page":  req.Page,
			"cache": "hit",
		}).Debug("Search cache hit")

		return s.buildSearchResponse(result, req, true, time.Since(startTime)), nil
	}

	// Level 2: Execute search against Solr with shorter timeout for faster fallback
	solrCtx, solrCancel := context.WithTimeout(ctx, 3*time.Second)
	defer solrCancel()

	result, err := s.solrRepo.Search(solrCtx, req)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"query": req.Query,
			"error": err,
		}).Warn("Search execution failed, attempting fallback")

		// Level 3: Try stale cache (30 min TTL)
		if staleResult, found := s.cacheRepo.GetStaleSearchResults(ctx, req); found {
			s.logger.WithFields(logrus.Fields{
				"query": req.Query,
				"page":  req.Page,
				"cache": "stale",
			}).Warn("Returning stale cache due to Solr failure")

			return s.buildSearchResponse(staleResult, req, true, time.Since(startTime)), nil
		}

		// Level 4: Fallback to Orders API (direct database query)
		if s.ordersClient != nil {
			s.logger.WithFields(logrus.Fields{
				"query": req.Query,
			}).Warn("Attempting fallback to Orders API")

			fallbackResult, fallbackErr := s.fallbackToOrdersAPI(ctx, req)
			if fallbackErr == nil {
				s.logger.WithFields(logrus.Fields{
					"query":   req.Query,
					"results": len(fallbackResult.Results),
					"total":   fallbackResult.Total,
					"source":  "orders_api",
				}).Info("Fallback to Orders API successful")

				// Cache the fallback result for future use
				go func() {
					cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					if err := s.cacheRepo.SetSearchResults(cacheCtx, req, fallbackResult); err != nil {
						s.logger.WithFields(logrus.Fields{
							"query": req.Query,
							"error": err,
						}).Warn("Failed to cache fallback results")
					}
				}()

				return s.buildSearchResponse(fallbackResult, req, false, time.Since(startTime)), nil
			}

			s.logger.WithFields(logrus.Fields{
				"query": req.Query,
				"error": fallbackErr,
			}).Error("Fallback to Orders API also failed")
		}

		// All fallbacks failed
		return nil, fmt.Errorf("search failed and all fallbacks exhausted: %w", err)
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

// fallbackToOrdersAPI queries Orders API directly when Solr is unavailable
func (s *SearchService) fallbackToOrdersAPI(ctx context.Context, req *dto.SearchRequest) (*repositories.SearchResult, error) {
	// Convert SearchRequest to Orders API parameters
	// Note: Orders API has limited filtering compared to Solr
	var status, orderType, symbol string
	if len(req.Status) > 0 {
		status = req.Status[0] // Take first status if multiple
	}
	if len(req.Type) > 0 {
		orderType = req.Type[0] // Take first type if multiple
	}
	if len(req.CryptoSymbol) > 0 {
		symbol = req.CryptoSymbol[0] // Take first symbol if multiple
	}

	// Call Orders API
	ordersResp, err := s.ordersClient.SearchOrders(ctx, req.UserID, status, orderType, symbol, req.Page, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("orders API search failed: %w", err)
	}

	// Convert Orders API response to SearchResult
	result := s.convertOrdersAPIResponseToSearchResult(ordersResp, req)

	return result, nil
}

// convertOrdersAPIResponseToSearchResult converts Orders API response to SearchResult format
func (s *SearchService) convertOrdersAPIResponseToSearchResult(ordersResp *clients.SearchOrdersResponse, req *dto.SearchRequest) *repositories.SearchResult {
	results := make([]interface{}, 0, len(ordersResp.Orders))

	for _, orderResp := range ordersResp.Orders {
		// Convert OrderResponse to Order model
		order := s.convertOrderResponseToOrderModel(&orderResp)

		// Convert to OrderSearchResult
		orderSearchResult := models.OrderSearchResult{
			Order:      *order,
			Score:      1.0, // Default score for fallback results
			MatchType:  "fallback",
			Highlighting: make(map[string][]string),
		}

		// Apply text matching if query is provided (simple contains check)
		if req.Query != "" {
			queryLower := strings.ToLower(req.Query)
			if strings.Contains(strings.ToLower(order.CryptoSymbol), queryLower) ||
				strings.Contains(strings.ToLower(order.CryptoName), queryLower) {
				orderSearchResult.MatchType = "symbol_match"
				orderSearchResult.Score = 0.8
			}
		}

		results = append(results, orderSearchResult)
	}

	// Build facets from results (simplified)
	facets := s.buildFacetsFromOrders(ordersResp.Orders)

	return &repositories.SearchResult{
		Results:   results,
		Total:     ordersResp.Total,
		Facets:    facets,
		QueryTime: 0, // Not measured for fallback
	}
}

// convertOrderResponseToOrderModel converts OrderResponse to Order model
func (s *SearchService) convertOrderResponseToOrderModel(resp *clients.OrderResponse) *models.Order {
	order := &models.Order{
		ID:           resp.ID,
		UserID:       resp.UserID,
		Type:         resp.Type,
		Status:       resp.Status,
		OrderKind:    resp.OrderKind,
		CryptoSymbol: resp.CryptoSymbol,
		CryptoName:   resp.CryptoName,
		Quantity:     resp.Quantity,
		Price:        resp.OrderPrice,
		TotalAmount:  resp.TotalAmount,
		Fee:          resp.Fee,
		CreatedAt:    resp.CreatedAt,
		UpdatedAt:    resp.UpdatedAt,
	}

	if resp.ExecutedAt != nil {
		order.ExecutedAt = resp.ExecutedAt
	}

	if resp.CancelledAt != nil {
		order.CancelledAt = resp.CancelledAt
	}

	// Build search text
	order.SearchText = s.buildSearchTextFromOrder(order)

	return order
}

// buildSearchTextFromOrder builds searchable text from order
func (s *SearchService) buildSearchTextFromOrder(order *models.Order) string {
	parts := []string{
		order.ID,
		order.CryptoSymbol,
		order.CryptoName,
		order.Type,
		order.Status,
		order.OrderKind,
		order.Quantity,
		order.Price,
		order.TotalAmount,
	}
	return strings.Join(parts, " ")
}

// buildFacetsFromOrders builds facets from orders list
func (s *SearchService) buildFacetsFromOrders(orders []clients.OrderResponse) models.OrderFacets {
	facets := models.OrderFacets{
		Statuses:      make(map[string]int64),
		Types:         make(map[string]int64),
		OrderKinds:    make(map[string]int64),
		CryptoSymbols: make(map[string]int64),
	}

	for _, order := range orders {
		facets.Statuses[order.Status]++
		facets.Types[order.Type]++
		facets.OrderKinds[order.OrderKind]++
		facets.CryptoSymbols[order.CryptoSymbol]++
	}

	return facets
}
