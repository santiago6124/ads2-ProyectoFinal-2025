package dto

import (
	"time"
	"search-api/internal/models"
)

// APIResponse represents the standard API response format
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta represents response metadata
type Meta struct {
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
	Version   string    `json:"version,omitempty"`
}

// SearchResponse represents the search API response
type SearchResponse struct {
	Results    []models.SearchResult `json:"results"`
	Pagination models.Pagination     `json:"pagination"`
	Facets     models.Facets         `json:"facets"`
	QueryInfo  models.QueryInfo      `json:"query_info"`
}

// TrendingResponse represents the trending API response
type TrendingResponse struct {
	Trending  []models.TrendingCrypto `json:"trending"`
	Period    string                  `json:"period"`
	UpdatedAt time.Time               `json:"updated_at"`
}

// SuggestionsResponse represents the suggestions API response
type SuggestionsResponse struct {
	Suggestions     []models.Suggestion `json:"suggestions"`
	Query           string              `json:"query"`
	ExecutionTimeMS int64               `json:"execution_time_ms"`
}

// CryptoDetailsResponse represents a single crypto details response
type CryptoDetailsResponse struct {
	models.Crypto
	RelatedCryptos []models.Crypto `json:"related_cryptos,omitempty"`
}

// FiltersResponse represents the filters API response
type FiltersResponse struct {
	Filters models.Filter `json:"filters"`
}

// ReindexResponse represents the reindex job response
type ReindexResponse struct {
	JobID           string        `json:"job_id"`
	EstimatedTime   string        `json:"estimated_time"`
	TotalDocuments  int64         `json:"total_documents"`
	BatchSize       int           `json:"batch_size"`
	Status          string        `json:"status"`
	StartedAt       time.Time     `json:"started_at"`
	Progress        float64       `json:"progress,omitempty"`
}

// CacheClearResponse represents the cache clear response
type CacheClearResponse struct {
	LocalCacheCleared       bool  `json:"local_cache_cleared"`
	DistributedCacheCleared bool  `json:"distributed_cache_cleared"`
	EntriesRemoved          int64 `json:"entries_removed"`
	ClearType               string `json:"clear_type"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                  `json:"status"`
	Timestamp time.Time               `json:"timestamp"`
	Services  map[string]ServiceHealth `json:"services"`
	Uptime    time.Duration           `json:"uptime"`
}

// ServiceHealth represents individual service health
type ServiceHealth struct {
	Status       string        `json:"status"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
	Error        string        `json:"error,omitempty"`
	LastCheck    time.Time     `json:"last_check"`
	Details      interface{}   `json:"details,omitempty"`
}

// MetricsResponse represents the metrics response
type MetricsResponse struct {
	SearchMetrics SearchMetrics `json:"search_metrics"`
	CacheMetrics  CacheMetrics  `json:"cache_metrics"`
	SolrMetrics   SolrMetrics   `json:"solr_metrics"`
	SystemMetrics SystemMetrics `json:"system_metrics"`
}

// SearchMetrics represents search-related metrics
type SearchMetrics struct {
	TotalSearches        int64             `json:"total_searches"`
	SearchesLast24h      int64             `json:"searches_last_24h"`
	AverageResponseTime  time.Duration     `json:"average_response_time"`
	PopularQueries       []PopularQuery    `json:"popular_queries"`
	SearchesByCategory   map[string]int64  `json:"searches_by_category"`
}

// PopularQuery represents a popular search query
type PopularQuery struct {
	Query string `json:"query"`
	Count int64  `json:"count"`
}

// CacheMetrics represents cache-related metrics
type CacheMetrics struct {
	LocalCacheHitRate       float64 `json:"local_cache_hit_rate"`
	DistributedCacheHitRate float64 `json:"distributed_cache_hit_rate"`
	LocalCacheSize          int64   `json:"local_cache_size"`
	LocalCacheMaxSize       int64   `json:"local_cache_max_size"`
	CacheEvictions          int64   `json:"cache_evictions"`
	TotalCacheHits          int64   `json:"total_cache_hits"`
	TotalCacheMisses        int64   `json:"total_cache_misses"`
}

// SolrMetrics represents Solr-related metrics
type SolrMetrics struct {
	DocumentCount       int64         `json:"document_count"`
	IndexSize           string        `json:"index_size"`
	AverageQueryTime    time.Duration `json:"average_query_time"`
	TotalQueries        int64         `json:"total_queries"`
	FailedQueries       int64         `json:"failed_queries"`
	LastIndexUpdate     time.Time     `json:"last_index_update"`
}

// SystemMetrics represents system-related metrics
type SystemMetrics struct {
	MemoryUsage    string        `json:"memory_usage"`
	CPUUsage       float64       `json:"cpu_usage"`
	GoroutineCount int           `json:"goroutine_count"`
	Uptime         time.Duration `json:"uptime"`
}

// NewSuccessResponse creates a successful API response
func NewSuccessResponse(data interface{}) *APIResponse {
	return &APIResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Timestamp: time.Now(),
			Version:   "1.0",
		},
	}
}

// NewErrorResponse creates an error API response
func NewErrorResponse(code, message, details string) *APIResponse {
	return &APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: &Meta{
			Timestamp: time.Now(),
			Version:   "1.0",
		},
	}
}

// NewValidationErrorResponse creates a validation error response
func NewValidationErrorResponse(message string) *APIResponse {
	return NewErrorResponse("VALIDATION_ERROR", message, "")
}

// NewNotFoundResponse creates a not found error response
func NewNotFoundResponse(resource string) *APIResponse {
	return NewErrorResponse("NOT_FOUND", resource+" not found", "")
}

// NewInternalErrorResponse creates an internal server error response
func NewInternalErrorResponse(details string) *APIResponse {
	return NewErrorResponse("INTERNAL_ERROR", "Internal server error", details)
}

// NewServiceUnavailableResponse creates a service unavailable error response
func NewServiceUnavailableResponse(service string) *APIResponse {
	return NewErrorResponse("SERVICE_UNAVAILABLE", service+" service is currently unavailable", "")
}

// NewRateLimitResponse creates a rate limit error response
func NewRateLimitResponse() *APIResponse {
	return NewErrorResponse("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", "Please try again later")
}

// WithRequestID adds request ID to the response
func (r *APIResponse) WithRequestID(requestID string) *APIResponse {
	if r.Meta == nil {
		r.Meta = &Meta{
			Timestamp: time.Now(),
			Version:   "1.0",
		}
	}
	r.Meta.RequestID = requestID
	return r
}

// BuildSearchResponse builds a search response with pagination and facets
func BuildSearchResponse(results []models.SearchResult, request *SearchRequest, total int64, facets models.Facets, queryInfo models.QueryInfo) *SearchResponse {
	// Calculate pagination
	totalPages := (total + int64(request.Limit) - 1) / int64(request.Limit)
	hasNext := int64(request.Page) < totalPages
	hasPrev := request.Page > 1

	pagination := models.Pagination{
		Total:      total,
		Page:       request.Page,
		Limit:      request.Limit,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}

	return &SearchResponse{
		Results:    results,
		Pagination: pagination,
		Facets:     facets,
		QueryInfo:  queryInfo,
	}
}

// BuildTrendingResponse builds a trending response
func BuildTrendingResponse(trending []models.TrendingCrypto, period string) *TrendingResponse {
	return &TrendingResponse{
		Trending:  trending,
		Period:    period,
		UpdatedAt: time.Now(),
	}
}

// BuildSuggestionsResponse builds a suggestions response
func BuildSuggestionsResponse(suggestions []models.Suggestion, query string, executionTime time.Duration) *SuggestionsResponse {
	return &SuggestionsResponse{
		Suggestions:     suggestions,
		Query:           query,
		ExecutionTimeMS: executionTime.Milliseconds(),
	}
}

// BuildHealthResponse builds a health response
func BuildHealthResponse(services map[string]ServiceHealth, uptime time.Duration) *HealthResponse {
	status := "healthy"
	for _, service := range services {
		if service.Status != "healthy" {
			status = "unhealthy"
			break
		}
	}

	return &HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Services:  services,
		Uptime:    uptime,
	}
}

// ErrorCode constants
const (
	ErrorCodeValidation        = "VALIDATION_ERROR"
	ErrorCodeNotFound          = "NOT_FOUND"
	ErrorCodeInternalError     = "INTERNAL_ERROR"
	ErrorCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrorCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	ErrorCodeUnauthorized      = "UNAUTHORIZED"
	ErrorCodeForbidden         = "FORBIDDEN"
	ErrorCodeBadRequest        = "BAD_REQUEST"
	ErrorCodeConflict          = "CONFLICT"
	ErrorCodeTimeout           = "TIMEOUT"
)