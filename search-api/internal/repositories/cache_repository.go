package repositories

import (
	"context"
	"fmt"
	"time"

	"search-api/internal/cache"
	"search-api/internal/dto"
	"search-api/internal/models"
)

// CacheRepository handles cached search operations
type CacheRepository struct {
	cacheManager *cache.CacheManager
	keyBuilder   *cache.CacheKeyBuilder
}

// CachedSearchRepository defines the cached search repository interface
type CachedSearchRepository interface {
	GetSearchResults(ctx context.Context, req *dto.SearchRequest) (*SearchResult, bool)
	SetSearchResults(ctx context.Context, req *dto.SearchRequest, result *SearchResult) error
	GetTrendingResults(ctx context.Context, period string, limit int) ([]models.TrendingCrypto, bool)
	SetTrendingResults(ctx context.Context, period string, limit int, trending []models.TrendingCrypto) error
	GetSuggestions(ctx context.Context, query string, limit int) ([]models.Suggestion, bool)
	SetSuggestions(ctx context.Context, query string, limit int, suggestions []models.Suggestion) error
	GetCrypto(ctx context.Context, id string) (*models.Crypto, bool)
	SetCrypto(ctx context.Context, crypto *models.Crypto) error
	GetFilters(ctx context.Context) (*models.Filter, bool)
	SetFilters(ctx context.Context, filters *models.Filter) error
	InvalidateSearch(ctx context.Context, pattern string) error
	InvalidateAll(ctx context.Context) error
	GetStats() *cache.CacheStats
	Ping(ctx context.Context) error
}

// NewCacheRepository creates a new cache repository
func NewCacheRepository(cacheManager *cache.CacheManager) CachedSearchRepository {
	return &CacheRepository{
		cacheManager: cacheManager,
		keyBuilder:   cache.NewCacheKeyBuilder("search"),
	}
}

// GetSearchResults retrieves search results from cache
func (r *CacheRepository) GetSearchResults(ctx context.Context, req *dto.SearchRequest) (*SearchResult, bool) {
	key := r.buildSearchKey(req)

	if value, found := r.cacheManager.Get(ctx, key); found {
		if result, ok := value.(*SearchResult); ok {
			return result, true
		}
	}

	return nil, false
}

// SetSearchResults stores search results in cache
func (r *CacheRepository) SetSearchResults(ctx context.Context, req *dto.SearchRequest, result *SearchResult) error {
	key := r.buildSearchKey(req)
	ttl := r.getSearchCacheTTL(req)

	return r.cacheManager.Set(ctx, key, result, ttl)
}

// GetTrendingResults retrieves trending results from cache
func (r *CacheRepository) GetTrendingResults(ctx context.Context, period string, limit int) ([]models.TrendingCrypto, bool) {
	key := r.keyBuilder.TrendingKey(period, limit)

	if value, found := r.cacheManager.Get(ctx, key); found {
		if trending, ok := value.([]models.TrendingCrypto); ok {
			return trending, true
		}
	}

	return nil, false
}

// SetTrendingResults stores trending results in cache
func (r *CacheRepository) SetTrendingResults(ctx context.Context, period string, limit int, trending []models.TrendingCrypto) error {
	key := r.keyBuilder.TrendingKey(period, limit)
	ttl := r.getTrendingCacheTTL(period)

	return r.cacheManager.Set(ctx, key, trending, ttl)
}

// GetSuggestions retrieves suggestions from cache
func (r *CacheRepository) GetSuggestions(ctx context.Context, query string, limit int) ([]models.Suggestion, bool) {
	key := r.keyBuilder.SuggestionsKey(query, limit)

	if value, found := r.cacheManager.Get(ctx, key); found {
		if suggestions, ok := value.([]models.Suggestion); ok {
			return suggestions, true
		}
	}

	return nil, false
}

// SetSuggestions stores suggestions in cache
func (r *CacheRepository) SetSuggestions(ctx context.Context, query string, limit int, suggestions []models.Suggestion) error {
	key := r.keyBuilder.SuggestionsKey(query, limit)
	ttl := 2 * time.Minute // Short TTL for suggestions

	return r.cacheManager.Set(ctx, key, suggestions, ttl)
}

// GetCrypto retrieves individual crypto data from cache
func (r *CacheRepository) GetCrypto(ctx context.Context, id string) (*models.Crypto, bool) {
	key := r.keyBuilder.CryptoKey(id)

	if value, found := r.cacheManager.Get(ctx, key); found {
		if crypto, ok := value.(*models.Crypto); ok {
			return crypto, true
		}
	}

	return nil, false
}

// SetCrypto stores individual crypto data in cache
func (r *CacheRepository) SetCrypto(ctx context.Context, crypto *models.Crypto) error {
	key := r.keyBuilder.CryptoKey(crypto.ID)
	ttl := 5 * time.Minute // Medium TTL for individual crypto data

	return r.cacheManager.Set(ctx, key, crypto, ttl)
}

// GetFilters retrieves filter data from cache
func (r *CacheRepository) GetFilters(ctx context.Context) (*models.Filter, bool) {
	key := r.keyBuilder.FiltersKey()

	if value, found := r.cacheManager.Get(ctx, key); found {
		if filters, ok := value.(*models.Filter); ok {
			return filters, true
		}
	}

	return nil, false
}

// SetFilters stores filter data in cache
func (r *CacheRepository) SetFilters(ctx context.Context, filters *models.Filter) error {
	key := r.keyBuilder.FiltersKey()
	ttl := 10 * time.Minute // Longer TTL for filter data

	return r.cacheManager.Set(ctx, key, filters, ttl)
}

// InvalidateSearch invalidates search-related cache entries
func (r *CacheRepository) InvalidateSearch(ctx context.Context, pattern string) error {
	return r.cacheManager.InvalidatePattern(ctx, "search:"+pattern)
}

// InvalidateAll clears all cache entries
func (r *CacheRepository) InvalidateAll(ctx context.Context) error {
	return r.cacheManager.Clear(ctx)
}

// GetStats returns cache statistics
func (r *CacheRepository) GetStats() *cache.CacheStats {
	return r.cacheManager.GetStats()
}

// Ping checks cache health
func (r *CacheRepository) Ping(ctx context.Context) error {
	return r.cacheManager.Ping(ctx)
}

// Helper methods

func (r *CacheRepository) buildSearchKey(req *dto.SearchRequest) string {
	filters := make(map[string]interface{})

	if req.Sort != "" {
		filters["sort"] = req.Sort
	}

	if len(req.Category) > 0 {
		filters["category"] = req.Category
	}

	if req.MinPrice != nil {
		filters["min_price"] = *req.MinPrice
	}

	if req.MaxPrice != nil {
		filters["max_price"] = *req.MaxPrice
	}

	if req.MinMarketCap != nil {
		filters["min_mc"] = *req.MinMarketCap
	}

	if req.MaxMarketCap != nil {
		filters["max_mc"] = *req.MaxMarketCap
	}

	if req.PriceChange24h != "" {
		filters["change"] = req.PriceChange24h
	}

	if req.IsTrending != nil {
		filters["trending"] = *req.IsTrending
	}

	if req.Platform != "" {
		filters["platform"] = req.Platform
	}

	if len(req.Tags) > 0 {
		filters["tags"] = req.Tags
	}

	return r.keyBuilder.SearchKey(req.Query, req.Page, req.Limit, filters)
}

func (r *CacheRepository) getSearchCacheTTL(req *dto.SearchRequest) time.Duration {
	// Dynamic TTL based on search characteristics

	// Empty queries (homepage) get longer cache
	if req.Query == "" && len(req.Category) == 0 {
		return 10 * time.Minute
	}

	// Trending searches get medium cache
	if req.IsTrending != nil && *req.IsTrending {
		return 5 * time.Minute
	}

	// Specific searches get shorter cache
	if req.Query != "" {
		return 3 * time.Minute
	}

	// Filtered searches get shortest cache
	if r.hasFilters(req) {
		return 2 * time.Minute
	}

	// Default cache duration
	return 5 * time.Minute
}

func (r *CacheRepository) getTrendingCacheTTL(period string) time.Duration {
	switch period {
	case "1h":
		return 2 * time.Minute
	case "24h":
		return 10 * time.Minute
	case "7d":
		return 30 * time.Minute
	case "30d":
		return 1 * time.Hour
	default:
		return 10 * time.Minute
	}
}

func (r *CacheRepository) hasFilters(req *dto.SearchRequest) bool {
	return len(req.Category) > 0 ||
		req.MinPrice != nil ||
		req.MaxPrice != nil ||
		req.MinMarketCap != nil ||
		req.MaxMarketCap != nil ||
		req.PriceChange24h != "" ||
		req.Platform != "" ||
		len(req.Tags) > 0
}

// CacheWarmer provides cache warming functionality
type CacheWarmer struct {
	cacheRepo   CachedSearchRepository
	searchRepo  SearchRepository
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(cacheRepo CachedSearchRepository, searchRepo SearchRepository) *CacheWarmer {
	return &CacheWarmer{
		cacheRepo:  cacheRepo,
		searchRepo: searchRepo,
	}
}

// WarmPopularSearches pre-loads popular search queries into cache
func (cw *CacheWarmer) WarmPopularSearches(ctx context.Context) error {
	popularQueries := []string{
		"",           // Empty query (homepage)
		"bitcoin",
		"ethereum",
		"dogecoin",
		"cardano",
		"solana",
	}

	for _, query := range popularQueries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Create search request
		req := &dto.SearchRequest{
			Query: query,
			Page:  1,
			Limit: 20,
		}
		req.SetDefaults()

		// Check if already cached
		if _, found := cw.cacheRepo.GetSearchResults(ctx, req); found {
			continue
		}

		// Execute search and cache result
		result, err := cw.searchRepo.Search(ctx, req)
		if err != nil {
			continue // Skip on error, don't fail entire warmup
		}

		if err := cw.cacheRepo.SetSearchResults(ctx, req, result); err != nil {
			continue // Skip on cache error
		}
	}

	return nil
}

// WarmTrendingData pre-loads trending data into cache
func (cw *CacheWarmer) WarmTrendingData(ctx context.Context) error {
	periods := []string{"1h", "24h", "7d", "30d"}

	for _, period := range periods {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if already cached
		if _, found := cw.cacheRepo.GetTrendingResults(ctx, period, 10); found {
			continue
		}

		// Execute trending search and cache result
		trending, err := cw.searchRepo.SearchTrending(ctx, period, 10)
		if err != nil {
			continue
		}

		if err := cw.cacheRepo.SetTrendingResults(ctx, period, 10, trending); err != nil {
			continue
		}
	}

	return nil
}

// WarmFilters pre-loads filter data into cache
func (cw *CacheWarmer) WarmFilters(ctx context.Context) error {
	// Check if already cached
	if _, found := cw.cacheRepo.GetFilters(ctx); found {
		return nil
	}

	// Execute filters query and cache result
	filters, err := cw.searchRepo.GetFacets(ctx)
	if err != nil {
		return fmt.Errorf("failed to warm filters cache: %w", err)
	}

	return cw.cacheRepo.SetFilters(ctx, filters)
}

// WarmAll executes all cache warming operations
func (cw *CacheWarmer) WarmAll(ctx context.Context) error {
	operations := []func(context.Context) error{
		cw.WarmPopularSearches,
		cw.WarmTrendingData,
		cw.WarmFilters,
	}

	for _, op := range operations {
		if err := op(ctx); err != nil {
			return err
		}
	}

	return nil
}