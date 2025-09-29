package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"market-data-api/internal/models"
)

// Manager provides high-level cache management functionality
type Manager struct {
	cache      Cache
	priceCache PriceCache
	config     *ManagerConfig
	metrics    *ManagerMetrics
	mu         sync.RWMutex

	// Background processes
	backgroundCtx    context.Context
	backgroundCancel context.CancelFunc
	wg               sync.WaitGroup
}

// ManagerConfig represents cache manager configuration
type ManagerConfig struct {
	// Cache configuration
	CacheConfig *CacheConfig `json:"cache_config"`

	// TTL settings for different data types
	PriceTTL              time.Duration `json:"price_ttl"`
	MarketDataTTL         time.Duration `json:"market_data_ttl"`
	HistoricalDataTTL     time.Duration `json:"historical_data_ttl"`
	OrderBookTTL          time.Duration `json:"order_book_ttl"`
	StatisticsTTL         time.Duration `json:"statistics_ttl"`
	TechnicalIndicatorsTTL time.Duration `json:"technical_indicators_ttl"`
	VolatilityDataTTL     time.Duration `json:"volatility_data_ttl"`

	// Cache warming
	EnableWarmup      bool          `json:"enable_warmup"`
	WarmupSymbols     []string      `json:"warmup_symbols"`
	WarmupInterval    time.Duration `json:"warmup_interval"`

	// Cache maintenance
	EnableMaintenance    bool          `json:"enable_maintenance"`
	MaintenanceInterval  time.Duration `json:"maintenance_interval"`
	CleanupThreshold     float64       `json:"cleanup_threshold"`   // Memory usage threshold for cleanup
	MaxEntries           int64         `json:"max_entries"`          // Maximum number of entries before cleanup

	// Performance optimization
	EnablePrefetch       bool          `json:"enable_prefetch"`
	PrefetchBatchSize    int           `json:"prefetch_batch_size"`
	EnableCompression    bool          `json:"enable_compression"`
	CompressionLevel     int           `json:"compression_level"`

	// Monitoring and alerting
	EnableMonitoring     bool          `json:"enable_monitoring"`
	MonitoringInterval   time.Duration `json:"monitoring_interval"`
	AlertThresholds      *AlertThresholds `json:"alert_thresholds"`
}

// AlertThresholds defines thresholds for cache alerts
type AlertThresholds struct {
	MemoryUsagePercent   float64       `json:"memory_usage_percent"`
	HitRatioThreshold    float64       `json:"hit_ratio_threshold"`
	ErrorRateThreshold   float64       `json:"error_rate_threshold"`
	LatencyThreshold     time.Duration `json:"latency_threshold"`
}

// ManagerMetrics tracks cache manager performance
type ManagerMetrics struct {
	// Operation metrics
	TotalOperations   int64         `json:"total_operations"`
	SuccessfulOps     int64         `json:"successful_operations"`
	FailedOps         int64         `json:"failed_operations"`
	CacheHitRatio     float64       `json:"cache_hit_ratio"`
	AverageLatency    time.Duration `json:"average_latency"`

	// Data type metrics
	PriceOperations      int64 `json:"price_operations"`
	MarketDataOps        int64 `json:"market_data_operations"`
	HistoricalDataOps    int64 `json:"historical_data_operations"`
	OrderBookOps         int64 `json:"order_book_operations"`
	StatisticsOps        int64 `json:"statistics_operations"`
	TechnicalOps         int64 `json:"technical_operations"`
	VolatilityOps        int64 `json:"volatility_operations"`

	// Background process metrics
	WarmupRuns           int64     `json:"warmup_runs"`
	MaintenanceRuns      int64     `json:"maintenance_runs"`
	CleanupOperations    int64     `json:"cleanup_operations"`
	PrefetchOperations   int64     `json:"prefetch_operations"`
	LastMaintenanceRun   time.Time `json:"last_maintenance_run"`
	LastWarmupRun        time.Time `json:"last_warmup_run"`

	// Error tracking
	ConnectionErrors     int64     `json:"connection_errors"`
	SerializationErrors  int64     `json:"serialization_errors"`
	TimeoutErrors        int64     `json:"timeout_errors"`
	LastError            string    `json:"last_error"`
	LastErrorTime        time.Time `json:"last_error_time"`

	LastUpdated          time.Time `json:"last_updated"`
}

// NewManager creates a new cache manager
func NewManager(config *ManagerConfig) (*Manager, error) {
	if config == nil {
		config = GetDefaultManagerConfig()
	}

	// Create base cache
	cache, err := NewRedisCache(config.CacheConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis cache: %w", err)
	}

	// Create specialized price cache
	priceCache := NewRedisPriceCache(cache)

	backgroundCtx, backgroundCancel := context.WithCancel(context.Background())

	manager := &Manager{
		cache:            cache,
		priceCache:       priceCache,
		config:          config,
		metrics:         &ManagerMetrics{},
		backgroundCtx:   backgroundCtx,
		backgroundCancel: backgroundCancel,
	}

	// Start background processes
	if config.EnableWarmup {
		manager.wg.Add(1)
		go manager.warmupLoop()
	}

	if config.EnableMaintenance {
		manager.wg.Add(1)
		go manager.maintenanceLoop()
	}

	if config.EnableMonitoring {
		manager.wg.Add(1)
		go manager.monitoringLoop()
	}

	return manager, nil
}

// High-level cache operations

func (m *Manager) GetPrice(ctx context.Context, symbol string) (*models.AggregatedPrice, error) {
	start := time.Now()
	defer m.recordOperation("get_price", start)

	price, err := m.priceCache.GetPrice(ctx, symbol)
	if err != nil {
		m.recordError("get_price", err)
		return nil, err
	}

	m.recordSuccess("get_price")
	return price, nil
}

func (m *Manager) SetPrice(ctx context.Context, symbol string, price *models.AggregatedPrice) error {
	start := time.Now()
	defer m.recordOperation("set_price", start)

	err := m.priceCache.SetPrice(ctx, symbol, price, m.config.PriceTTL)
	if err != nil {
		m.recordError("set_price", err)
		return err
	}

	m.recordSuccess("set_price")
	return nil
}

func (m *Manager) GetPrices(ctx context.Context, symbols []string) (map[string]*models.AggregatedPrice, error) {
	start := time.Now()
	defer m.recordOperation("get_prices", start)

	prices, err := m.priceCache.GetPrices(ctx, symbols)
	if err != nil {
		m.recordError("get_prices", err)
		return nil, err
	}

	m.recordSuccess("get_prices")
	return prices, nil
}

func (m *Manager) SetPrices(ctx context.Context, prices map[string]*models.AggregatedPrice) error {
	start := time.Now()
	defer m.recordOperation("set_prices", start)

	err := m.priceCache.SetPrices(ctx, prices, m.config.PriceTTL)
	if err != nil {
		m.recordError("set_prices", err)
		return err
	}

	m.recordSuccess("set_prices")
	return nil
}

func (m *Manager) GetMarketData(ctx context.Context, symbol string) (*models.MarketData, error) {
	start := time.Now()
	defer m.recordOperation("get_market_data", start)

	data, err := m.priceCache.GetMarketData(ctx, symbol)
	if err != nil {
		m.recordError("get_market_data", err)
		return nil, err
	}

	m.recordSuccess("get_market_data")
	return data, nil
}

func (m *Manager) SetMarketData(ctx context.Context, symbol string, data *models.MarketData) error {
	start := time.Now()
	defer m.recordOperation("set_market_data", start)

	err := m.priceCache.SetMarketData(ctx, symbol, data, m.config.MarketDataTTL)
	if err != nil {
		m.recordError("set_market_data", err)
		return err
	}

	m.recordSuccess("set_market_data")
	return nil
}

func (m *Manager) GetHistoricalData(ctx context.Context, symbol, interval string) ([]*models.Candle, error) {
	start := time.Now()
	defer m.recordOperation("get_historical_data", start)

	data, err := m.priceCache.GetHistoricalData(ctx, symbol, interval)
	if err != nil {
		m.recordError("get_historical_data", err)
		return nil, err
	}

	m.recordSuccess("get_historical_data")
	return data, nil
}

func (m *Manager) SetHistoricalData(ctx context.Context, symbol, interval string, data []*models.Candle) error {
	start := time.Now()
	defer m.recordOperation("set_historical_data", start)

	err := m.priceCache.SetHistoricalData(ctx, symbol, interval, data, m.config.HistoricalDataTTL)
	if err != nil {
		m.recordError("set_historical_data", err)
		return err
	}

	m.recordSuccess("set_historical_data")
	return nil
}

func (m *Manager) GetOrderBook(ctx context.Context, symbol string) (*models.OrderBook, error) {
	start := time.Now()
	defer m.recordOperation("get_order_book", start)

	orderBook, err := m.priceCache.GetOrderBook(ctx, symbol)
	if err != nil {
		m.recordError("get_order_book", err)
		return nil, err
	}

	m.recordSuccess("get_order_book")
	return orderBook, nil
}

func (m *Manager) SetOrderBook(ctx context.Context, symbol string, orderBook *models.OrderBook) error {
	start := time.Now()
	defer m.recordOperation("set_order_book", start)

	err := m.priceCache.SetOrderBook(ctx, symbol, orderBook, m.config.OrderBookTTL)
	if err != nil {
		m.recordError("set_order_book", err)
		return err
	}

	m.recordSuccess("set_order_book")
	return nil
}

func (m *Manager) GetTechnicalIndicators(ctx context.Context, symbol string) (*models.TechnicalIndicators, error) {
	start := time.Now()
	defer m.recordOperation("get_technical_indicators", start)

	indicators, err := m.priceCache.GetTechnicalIndicators(ctx, symbol)
	if err != nil {
		m.recordError("get_technical_indicators", err)
		return nil, err
	}

	m.recordSuccess("get_technical_indicators")
	return indicators, nil
}

func (m *Manager) SetTechnicalIndicators(ctx context.Context, symbol string, indicators *models.TechnicalIndicators) error {
	start := time.Now()
	defer m.recordOperation("set_technical_indicators", start)

	err := m.priceCache.SetTechnicalIndicators(ctx, symbol, indicators, m.config.TechnicalIndicatorsTTL)
	if err != nil {
		m.recordError("set_technical_indicators", err)
		return err
	}

	m.recordSuccess("set_technical_indicators")
	return nil
}

// Advanced cache operations

func (m *Manager) WarmupCache(ctx context.Context, symbols []string) error {
	if len(symbols) == 0 {
		symbols = m.config.WarmupSymbols
	}

	// This is a placeholder for cache warmup logic
	// In a real implementation, you would fetch data from providers
	// and populate the cache

	m.recordWarmupRun()
	return nil
}

func (m *Manager) InvalidateSymbol(ctx context.Context, symbol string) error {
	patterns := []string{
		fmt.Sprintf("price:%s", symbol),
		fmt.Sprintf("market:%s", symbol),
		fmt.Sprintf("historical:%s:*", symbol),
		fmt.Sprintf("orderbook:%s", symbol),
		fmt.Sprintf("stats:%s", symbol),
		fmt.Sprintf("technical:%s", symbol),
		fmt.Sprintf("volatility:%s", symbol),
	}

	for _, pattern := range patterns {
		keys, err := m.cache.Keys(ctx, pattern)
		if err != nil {
			continue
		}
		if len(keys) > 0 {
			m.cache.Del(ctx, keys...)
		}
	}

	return nil
}

func (m *Manager) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	// Get base cache metrics
	cacheMetrics := m.cache.(*RedisCache).GetMetrics()

	// Get price cache stats
	priceCacheStats, err := m.priceCache.(*RedisPriceCache).GetCacheStats(ctx)
	if err != nil {
		return nil, err
	}

	stats := &CacheStats{
		CacheMetrics:    cacheMetrics,
		PriceCacheStats: priceCacheStats,
		ManagerMetrics:  m.GetMetrics(),
		Timestamp:       time.Now(),
	}

	return stats, nil
}

// Background processes

func (m *Manager) warmupLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.WarmupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.backgroundCtx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(m.backgroundCtx, 30*time.Second)
			m.WarmupCache(ctx, nil)
			cancel()
		}
	}
}

func (m *Manager) maintenanceLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.MaintenanceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.backgroundCtx.Done():
			return
		case <-ticker.C:
			m.performMaintenance()
		}
	}
}

func (m *Manager) monitoringLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.backgroundCtx.Done():
			return
		case <-ticker.C:
			m.performMonitoring()
		}
	}
}

func (m *Manager) performMaintenance() {
	m.recordMaintenanceRun()

	ctx, cancel := context.WithTimeout(m.backgroundCtx, 5*time.Minute)
	defer cancel()

	// Get current cache metrics
	if cacheMetrics := m.cache.(*RedisCache).GetMetrics(); cacheMetrics != nil {
		// Check if cleanup is needed
		if cacheMetrics.MemoryUsage > m.config.CleanupThreshold {
			m.performCleanup(ctx)
		}
	}
}

func (m *Manager) performCleanup(ctx context.Context) {
	// Implement cache cleanup logic
	// - Remove expired entries
	// - Remove least recently used entries
	// - Compact data structures

	m.recordCleanupOperation()
}

func (m *Manager) performMonitoring() {
	ctx, cancel := context.WithTimeout(m.backgroundCtx, time.Minute)
	defer cancel()

	// Check cache health
	if err := m.cache.Ping(ctx); err != nil {
		m.recordError("monitoring_ping", err)
		return
	}

	// Get current metrics
	cacheMetrics := m.cache.(*RedisCache).GetMetrics()

	// Check thresholds and trigger alerts if necessary
	if m.config.AlertThresholds != nil {
		if cacheMetrics.MemoryUsage > m.config.AlertThresholds.MemoryUsagePercent {
			// Trigger memory usage alert
		}

		if cacheMetrics.HitRatio < m.config.AlertThresholds.HitRatioThreshold {
			// Trigger hit ratio alert
		}

		if cacheMetrics.AvgLatency > m.config.AlertThresholds.LatencyThreshold {
			// Trigger latency alert
		}
	}
}

// Metrics methods

func (m *Manager) recordOperation(operation string, start time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	latency := time.Since(start)
	m.metrics.TotalOperations++

	// Update operation-specific counters
	switch operation {
	case "get_price", "set_price", "get_prices", "set_prices":
		m.metrics.PriceOperations++
	case "get_market_data", "set_market_data":
		m.metrics.MarketDataOps++
	case "get_historical_data", "set_historical_data":
		m.metrics.HistoricalDataOps++
	case "get_order_book", "set_order_book":
		m.metrics.OrderBookOps++
	case "get_technical_indicators", "set_technical_indicators":
		m.metrics.TechnicalOps++
	}

	// Update average latency
	if m.metrics.AverageLatency == 0 {
		m.metrics.AverageLatency = latency
	} else {
		m.metrics.AverageLatency = (m.metrics.AverageLatency + latency) / 2
	}

	m.metrics.LastUpdated = time.Now()
}

func (m *Manager) recordSuccess(operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics.SuccessfulOps++
}

func (m *Manager) recordError(operation string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.FailedOps++
	m.metrics.LastError = err.Error()
	m.metrics.LastErrorTime = time.Now()

	// Categorize errors
	if IsConnectionFailed(err) {
		m.metrics.ConnectionErrors++
	} else if IsTimeout(err) {
		m.metrics.TimeoutErrors++
	} else if err.Error() == ErrCodeSerialization {
		m.metrics.SerializationErrors++
	}
}

func (m *Manager) recordWarmupRun() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics.WarmupRuns++
	m.metrics.LastWarmupRun = time.Now()
}

func (m *Manager) recordMaintenanceRun() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics.MaintenanceRuns++
	m.metrics.LastMaintenanceRun = time.Now()
}

func (m *Manager) recordCleanupOperation() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics.CleanupOperations++
}

func (m *Manager) GetMetrics() *ManagerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *m.metrics
	return &metrics
}

// Stop stops the cache manager and all background processes
func (m *Manager) Stop() {
	m.backgroundCancel()
	m.wg.Wait()

	if m.cache != nil {
		m.cache.Close()
	}
}

// Helper structures

type CacheStats struct {
	CacheMetrics    *CacheMetrics    `json:"cache_metrics"`
	PriceCacheStats *PriceCacheStats `json:"price_cache_stats"`
	ManagerMetrics  *ManagerMetrics  `json:"manager_metrics"`
	Timestamp       time.Time        `json:"timestamp"`
}

// Default configurations

func GetDefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		CacheConfig:            getDefaultRedisConfig(),
		PriceTTL:              5 * time.Minute,
		MarketDataTTL:         10 * time.Minute,
		HistoricalDataTTL:     time.Hour,
		OrderBookTTL:          30 * time.Second,
		StatisticsTTL:         15 * time.Minute,
		TechnicalIndicatorsTTL: 10 * time.Minute,
		VolatilityDataTTL:     30 * time.Minute,
		EnableWarmup:          true,
		WarmupSymbols:         []string{"BTC", "ETH", "ADA", "DOT", "LINK"},
		WarmupInterval:        10 * time.Minute,
		EnableMaintenance:     true,
		MaintenanceInterval:   30 * time.Minute,
		CleanupThreshold:      80.0, // 80% memory usage
		MaxEntries:            1000000,
		EnablePrefetch:        false,
		PrefetchBatchSize:     10,
		EnableCompression:     false,
		CompressionLevel:      6,
		EnableMonitoring:      true,
		MonitoringInterval:    time.Minute,
		AlertThresholds: &AlertThresholds{
			MemoryUsagePercent: 90.0,
			HitRatioThreshold:  0.8,
			ErrorRateThreshold: 0.05,
			LatencyThreshold:   100 * time.Millisecond,
		},
	}
}