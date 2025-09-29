package monitoring

import (
	"sync"
	"time"
)

// MetricsCollector collects and aggregates application metrics
type MetricsCollector struct {
	httpRequests     map[string]*RequestMetrics
	searchMetrics    *SearchMetrics
	cacheMetrics     *CacheMetrics
	solrMetrics      *SolrMetrics
	systemMetrics    *SystemMetrics
	mu               sync.RWMutex
	startTime        time.Time
}

// RequestMetrics tracks HTTP request metrics
type RequestMetrics struct {
	Count            int64
	TotalDuration    time.Duration
	MinDuration      time.Duration
	MaxDuration      time.Duration
	ErrorCount       int64
	LastRequestTime  time.Time
}

// SearchMetrics tracks search-specific metrics
type SearchMetrics struct {
	TotalSearches       int64
	SuccessfulSearches  int64
	FailedSearches      int64
	AverageResponseTime time.Duration
	CacheHitRate        float64
	PopularQueries      map[string]int64
	TrendingRequests    int64
	SuggestionRequests  int64
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	LocalHits         int64
	LocalMisses       int64
	DistributedHits   int64
	DistributedMisses int64
	EvictionCount     int64
	ErrorCount        int64
}

// SolrMetrics tracks Solr performance
type SolrMetrics struct {
	QueryCount        int64
	SuccessfulQueries int64
	FailedQueries     int64
	AverageLatency    time.Duration
	TimeoutCount      int64
	RetryCount        int64
}

// SystemMetrics tracks system-level metrics
type SystemMetrics struct {
	GoroutineCount   int
	MemoryUsage      int64
	CPUUsage         float64
	ConnectionCount  int64
	UptimeSeconds    int64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		httpRequests:  make(map[string]*RequestMetrics),
		searchMetrics: &SearchMetrics{
			PopularQueries: make(map[string]int64),
		},
		cacheMetrics:  &CacheMetrics{},
		solrMetrics:   &SolrMetrics{},
		systemMetrics: &SystemMetrics{},
		startTime:     time.Now(),
	}
}

// RecordHTTPRequest records metrics for an HTTP request
func (mc *MetricsCollector) RecordHTTPRequest(method, path string, duration time.Duration, statusCode int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := method + " " + path
	metrics, exists := mc.httpRequests[key]
	if !exists {
		metrics = &RequestMetrics{
			MinDuration: duration,
			MaxDuration: duration,
		}
		mc.httpRequests[key] = metrics
	}

	metrics.Count++
	metrics.TotalDuration += duration
	metrics.LastRequestTime = time.Now()

	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}

	if statusCode >= 400 {
		metrics.ErrorCount++
	}
}

// RecordSearchRequest records metrics for a search request
func (mc *MetricsCollector) RecordSearchRequest(query string, duration time.Duration, cacheHit bool, success bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.searchMetrics.TotalSearches++
	if success {
		mc.searchMetrics.SuccessfulSearches++
	} else {
		mc.searchMetrics.FailedSearches++
	}

	// Update average response time
	if mc.searchMetrics.TotalSearches > 0 {
		totalTime := time.Duration(mc.searchMetrics.AverageResponseTime.Nanoseconds()*int64(mc.searchMetrics.TotalSearches-1)) + duration
		mc.searchMetrics.AverageResponseTime = totalTime / time.Duration(mc.searchMetrics.TotalSearches)
	}

	// Track popular queries
	if query != "" {
		mc.searchMetrics.PopularQueries[query]++
	}

	// Update cache hit rate
	if mc.searchMetrics.TotalSearches > 0 {
		hits := mc.cacheMetrics.LocalHits + mc.cacheMetrics.DistributedHits
		total := hits + mc.cacheMetrics.LocalMisses + mc.cacheMetrics.DistributedMisses
		if total > 0 {
			mc.searchMetrics.CacheHitRate = float64(hits) / float64(total)
		}
	}
}

// RecordTrendingRequest records a trending request
func (mc *MetricsCollector) RecordTrendingRequest() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.searchMetrics.TrendingRequests++
}

// RecordSuggestionRequest records a suggestion request
func (mc *MetricsCollector) RecordSuggestionRequest() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.searchMetrics.SuggestionRequests++
}

// RecordCacheHit records a cache hit
func (mc *MetricsCollector) RecordCacheHit(isLocal bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if isLocal {
		mc.cacheMetrics.LocalHits++
	} else {
		mc.cacheMetrics.DistributedHits++
	}
}

// RecordCacheMiss records a cache miss
func (mc *MetricsCollector) RecordCacheMiss(isLocal bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if isLocal {
		mc.cacheMetrics.LocalMisses++
	} else {
		mc.cacheMetrics.DistributedMisses++
	}
}

// RecordCacheEviction records a cache eviction
func (mc *MetricsCollector) RecordCacheEviction() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.cacheMetrics.EvictionCount++
}

// RecordCacheError records a cache error
func (mc *MetricsCollector) RecordCacheError() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.cacheMetrics.ErrorCount++
}

// RecordSolrQuery records a Solr query
func (mc *MetricsCollector) RecordSolrQuery(duration time.Duration, success bool, timeout bool, retried bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.solrMetrics.QueryCount++
	if success {
		mc.solrMetrics.SuccessfulQueries++
	} else {
		mc.solrMetrics.FailedQueries++
	}

	if timeout {
		mc.solrMetrics.TimeoutCount++
	}

	if retried {
		mc.solrMetrics.RetryCount++
	}

	// Update average latency
	if mc.solrMetrics.QueryCount > 0 {
		totalTime := time.Duration(mc.solrMetrics.AverageLatency.Nanoseconds()*int64(mc.solrMetrics.QueryCount-1)) + duration
		mc.solrMetrics.AverageLatency = totalTime / time.Duration(mc.solrMetrics.QueryCount)
	}
}

// UpdateSystemMetrics updates system-level metrics
func (mc *MetricsCollector) UpdateSystemMetrics(goroutines int, memoryUsage int64, cpuUsage float64, connections int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.systemMetrics.GoroutineCount = goroutines
	mc.systemMetrics.MemoryUsage = memoryUsage
	mc.systemMetrics.CPUUsage = cpuUsage
	mc.systemMetrics.ConnectionCount = connections
	mc.systemMetrics.UptimeSeconds = int64(time.Since(mc.startTime).Seconds())
}

// GetHTTPMetrics returns HTTP request metrics
func (mc *MetricsCollector) GetHTTPMetrics() map[string]*RequestMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*RequestMetrics)
	for k, v := range mc.httpRequests {
		result[k] = &RequestMetrics{
			Count:           v.Count,
			TotalDuration:   v.TotalDuration,
			MinDuration:     v.MinDuration,
			MaxDuration:     v.MaxDuration,
			ErrorCount:      v.ErrorCount,
			LastRequestTime: v.LastRequestTime,
		}
	}
	return result
}

// GetSearchMetrics returns search metrics
func (mc *MetricsCollector) GetSearchMetrics() *SearchMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	popularQueries := make(map[string]int64)
	for k, v := range mc.searchMetrics.PopularQueries {
		popularQueries[k] = v
	}

	return &SearchMetrics{
		TotalSearches:       mc.searchMetrics.TotalSearches,
		SuccessfulSearches:  mc.searchMetrics.SuccessfulSearches,
		FailedSearches:      mc.searchMetrics.FailedSearches,
		AverageResponseTime: mc.searchMetrics.AverageResponseTime,
		CacheHitRate:        mc.searchMetrics.CacheHitRate,
		PopularQueries:      popularQueries,
		TrendingRequests:    mc.searchMetrics.TrendingRequests,
		SuggestionRequests:  mc.searchMetrics.SuggestionRequests,
	}
}

// GetCacheMetrics returns cache metrics
func (mc *MetricsCollector) GetCacheMetrics() *CacheMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return &CacheMetrics{
		LocalHits:         mc.cacheMetrics.LocalHits,
		LocalMisses:       mc.cacheMetrics.LocalMisses,
		DistributedHits:   mc.cacheMetrics.DistributedHits,
		DistributedMisses: mc.cacheMetrics.DistributedMisses,
		EvictionCount:     mc.cacheMetrics.EvictionCount,
		ErrorCount:        mc.cacheMetrics.ErrorCount,
	}
}

// GetSolrMetrics returns Solr metrics
func (mc *MetricsCollector) GetSolrMetrics() *SolrMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return &SolrMetrics{
		QueryCount:        mc.solrMetrics.QueryCount,
		SuccessfulQueries: mc.solrMetrics.SuccessfulQueries,
		FailedQueries:     mc.solrMetrics.FailedQueries,
		AverageLatency:    mc.solrMetrics.AverageLatency,
		TimeoutCount:      mc.solrMetrics.TimeoutCount,
		RetryCount:        mc.solrMetrics.RetryCount,
	}
}

// GetSystemMetrics returns system metrics
func (mc *MetricsCollector) GetSystemMetrics() *SystemMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return &SystemMetrics{
		GoroutineCount:  mc.systemMetrics.GoroutineCount,
		MemoryUsage:     mc.systemMetrics.MemoryUsage,
		CPUUsage:        mc.systemMetrics.CPUUsage,
		ConnectionCount: mc.systemMetrics.ConnectionCount,
		UptimeSeconds:   mc.systemMetrics.UptimeSeconds,
	}
}

// Reset resets all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.httpRequests = make(map[string]*RequestMetrics)
	mc.searchMetrics = &SearchMetrics{
		PopularQueries: make(map[string]int64),
	}
	mc.cacheMetrics = &CacheMetrics{}
	mc.solrMetrics = &SolrMetrics{}
	mc.systemMetrics = &SystemMetrics{}
	mc.startTime = time.Now()
}

// GetSummary returns a summary of all metrics
func (mc *MetricsCollector) GetSummary() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return map[string]interface{}{
		"http":   mc.GetHTTPMetrics(),
		"search": mc.GetSearchMetrics(),
		"cache":  mc.GetCacheMetrics(),
		"solr":   mc.GetSolrMetrics(),
		"system": mc.GetSystemMetrics(),
		"uptime_seconds": time.Since(mc.startTime).Seconds(),
	}
}