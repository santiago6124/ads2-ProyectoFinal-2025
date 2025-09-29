package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/karlseguin/ccache/v2"
	"github.com/sirupsen/logrus"
)

// CacheManager manages multi-level caching with local CCache and distributed Memcached
type CacheManager struct {
	localCache       *ccache.Cache
	distributedCache *memcache.Client
	config           *Config
	metrics          *CacheMetrics
	logger           *logrus.Logger
	keyPrefix        string
}

// Config represents cache configuration
type Config struct {
	LocalTTL            time.Duration
	DistributedTTL      time.Duration
	MaxLocalSize        int64
	LocalItemsToPrune   uint32
	MemcachedHosts      []string
	MemcachedTimeout    time.Duration
	MemcachedMaxIdleConns int
	KeyPrefix           string
	EnableMetrics       bool
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	LocalHits           int64
	LocalMisses         int64
	DistributedHits     int64
	DistributedMisses   int64
	LocalEvictions      int64
	Errors              int64
	TotalOperations     int64
	mu                  sync.RWMutex
}

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	TTL       time.Duration `json:"ttl"`
	CreatedAt time.Time   `json:"created_at"`
	HitCount  int64       `json:"hit_count"`
	Source    string      `json:"source"` // "local" or "distributed"
}

// NewCacheManager creates a new cache manager instance
func NewCacheManager(config *Config, logger *logrus.Logger) *CacheManager {
	if config == nil {
		config = DefaultConfig()
	}

	// Configure local cache (CCache)
	localCache := ccache.New(ccache.Configure().
		MaxSize(config.MaxLocalSize).
		ItemsToPrune(config.LocalItemsToPrune).
		DeleteBuffer(256).
		PromoteBuffer(256).
		GetsPerPromote(3))

	// Configure distributed cache (Memcached)
	var distributedCache *memcache.Client
	if len(config.MemcachedHosts) > 0 {
		distributedCache = memcache.New(config.MemcachedHosts...)
		distributedCache.Timeout = config.MemcachedTimeout
		distributedCache.MaxIdleConns = config.MemcachedMaxIdleConns
	}

	metrics := &CacheMetrics{}

	return &CacheManager{
		localCache:       localCache,
		distributedCache: distributedCache,
		config:           config,
		metrics:          metrics,
		logger:           logger,
		keyPrefix:        config.KeyPrefix,
	}
}

// Get attempts to retrieve a value from cache (local first, then distributed)
func (cm *CacheManager) Get(ctx context.Context, key string) (interface{}, bool) {
	cm.incrementTotalOperations()

	fullKey := cm.buildKey(key)

	// 1. Try local cache first
	if item := cm.localCache.Get(fullKey); item != nil && !item.Expired() {
		cm.incrementLocalHits()
		cm.logger.WithFields(logrus.Fields{
			"key":    key,
			"source": "local",
		}).Debug("Cache hit")

		return item.Value(), true
	}

	cm.incrementLocalMisses()

	// 2. Try distributed cache
	if cm.distributedCache != nil {
		select {
		case <-ctx.Done():
			return nil, false
		default:
		}

		if item, err := cm.distributedCache.Get(fullKey); err == nil {
			var cacheEntry CacheEntry
			if err := json.Unmarshal(item.Value, &cacheEntry); err == nil {
				// Store in local cache for faster subsequent access
				cm.localCache.Set(fullKey, cacheEntry.Value, cm.config.LocalTTL)

				cm.incrementDistributedHits()
				cm.logger.WithFields(logrus.Fields{
					"key":    key,
					"source": "distributed",
				}).Debug("Cache hit")

				return cacheEntry.Value, true
			} else {
				cm.incrementErrors()
				cm.logger.WithFields(logrus.Fields{
					"key":   key,
					"error": err,
				}).Error("Failed to unmarshal cache entry")
			}
		}
	}

	cm.incrementDistributedMisses()
	cm.logger.WithField("key", key).Debug("Cache miss")
	return nil, false
}

// Set stores a value in both local and distributed cache
func (cm *CacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	cm.incrementTotalOperations()

	fullKey := cm.buildKey(key)

	// Store in local cache
	cm.localCache.Set(fullKey, value, ttl)

	// Store in distributed cache if available
	if cm.distributedCache != nil {
		cacheEntry := CacheEntry{
			Key:       key,
			Value:     value,
			TTL:       ttl,
			CreatedAt: time.Now(),
			Source:    "distributed",
		}

		data, err := json.Marshal(cacheEntry)
		if err != nil {
			cm.incrementErrors()
			cm.logger.WithFields(logrus.Fields{
				"key":   key,
				"error": err,
			}).Error("Failed to marshal cache entry")
			return fmt.Errorf("failed to marshal cache entry: %w", err)
		}

		memcacheItem := &memcache.Item{
			Key:        fullKey,
			Value:      data,
			Expiration: int32(ttl.Seconds()),
		}

		if err := cm.distributedCache.Set(memcacheItem); err != nil {
			cm.incrementErrors()
			cm.logger.WithFields(logrus.Fields{
				"key":   key,
				"error": err,
			}).Error("Failed to set distributed cache")
			// Don't return error - local cache still works
		}
	}

	cm.logger.WithFields(logrus.Fields{
		"key": key,
		"ttl": ttl,
	}).Debug("Cache set")

	return nil
}

// Delete removes a key from both caches
func (cm *CacheManager) Delete(ctx context.Context, key string) error {
	cm.incrementTotalOperations()

	fullKey := cm.buildKey(key)

	// Delete from local cache
	cm.localCache.Delete(fullKey)

	// Delete from distributed cache
	if cm.distributedCache != nil {
		if err := cm.distributedCache.Delete(fullKey); err != nil && err != memcache.ErrCacheMiss {
			cm.incrementErrors()
			cm.logger.WithFields(logrus.Fields{
				"key":   key,
				"error": err,
			}).Error("Failed to delete from distributed cache")
		}
	}

	cm.logger.WithField("key", key).Debug("Cache delete")
	return nil
}

// InvalidatePattern removes all keys matching a pattern
func (cm *CacheManager) InvalidatePattern(ctx context.Context, pattern string) error {
	cm.incrementTotalOperations()

	// For local cache, we can use prefix deletion
	fullPattern := cm.buildKey(pattern)
	cm.localCache.DeletePrefix(fullPattern)

	// For distributed cache, we need to track keys or use versioning
	// This is a simplified approach - in production, you might want to use
	// a more sophisticated key tracking mechanism
	if cm.distributedCache != nil {
		// Note: Memcached doesn't support pattern deletion natively
		// You would need to maintain a key index or use cache versioning
		cm.logger.WithField("pattern", pattern).Warn("Pattern invalidation on distributed cache is limited")
	}

	cm.logger.WithField("pattern", pattern).Info("Cache pattern invalidated")
	return nil
}

// Clear removes all entries from both caches
func (cm *CacheManager) Clear(ctx context.Context) error {
	cm.incrementTotalOperations()

	// Clear local cache
	cm.localCache.Clear()

	// Clear distributed cache (flush all)
	if cm.distributedCache != nil {
		if err := cm.distributedCache.FlushAll(); err != nil {
			cm.incrementErrors()
			cm.logger.WithError(err).Error("Failed to flush distributed cache")
			return fmt.Errorf("failed to flush distributed cache: %w", err)
		}
	}

	cm.logger.Info("All caches cleared")
	return nil
}

// WarmCache pre-populates cache with popular data
func (cm *CacheManager) WarmCache(ctx context.Context, warmupData map[string]interface{}) error {
	cm.logger.Info("Starting cache warmup")

	for key, value := range warmupData {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Use longer TTL for warmup data
		ttl := cm.config.DistributedTTL
		if err := cm.Set(ctx, key, value, ttl); err != nil {
			cm.logger.WithFields(logrus.Fields{
				"key":   key,
				"error": err,
			}).Error("Failed to warm cache entry")
		}
	}

	cm.logger.WithField("entries", len(warmupData)).Info("Cache warmup completed")
	return nil
}

// GetStats returns cache statistics
func (cm *CacheManager) GetStats() *CacheStats {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()

	localSize := cm.localCache.Size()

	var localHitRate, distributedHitRate float64
	totalLocalRequests := cm.metrics.LocalHits + cm.metrics.LocalMisses
	if totalLocalRequests > 0 {
		localHitRate = float64(cm.metrics.LocalHits) / float64(totalLocalRequests)
	}

	totalDistributedRequests := cm.metrics.DistributedHits + cm.metrics.DistributedMisses
	if totalDistributedRequests > 0 {
		distributedHitRate = float64(cm.metrics.DistributedHits) / float64(totalDistributedRequests)
	}

	return &CacheStats{
		LocalHits:              cm.metrics.LocalHits,
		LocalMisses:            cm.metrics.LocalMisses,
		LocalHitRate:           localHitRate,
		LocalSize:              int64(localSize),
		LocalMaxSize:           cm.config.MaxLocalSize,
		DistributedHits:        cm.metrics.DistributedHits,
		DistributedMisses:      cm.metrics.DistributedMisses,
		DistributedHitRate:     distributedHitRate,
		LocalEvictions:         cm.metrics.LocalEvictions,
		Errors:                 cm.metrics.Errors,
		TotalOperations:        cm.metrics.TotalOperations,
		MemcachedConnected:     cm.distributedCache != nil,
	}
}

// Ping checks if both caches are available
func (cm *CacheManager) Ping(ctx context.Context) error {
	// Test local cache
	testKey := cm.buildKey("ping_test")
	testValue := "ping"

	cm.localCache.Set(testKey, testValue, time.Minute)
	if item := cm.localCache.Get(testKey); item == nil {
		return fmt.Errorf("local cache ping failed")
	}
	cm.localCache.Delete(testKey)

	// Test distributed cache
	if cm.distributedCache != nil {
		testItem := &memcache.Item{
			Key:        testKey,
			Value:      []byte(testValue),
			Expiration: 60,
		}

		if err := cm.distributedCache.Set(testItem); err != nil {
			return fmt.Errorf("distributed cache ping failed: %w", err)
		}

		if _, err := cm.distributedCache.Get(testKey); err != nil {
			return fmt.Errorf("distributed cache ping read failed: %w", err)
		}

		cm.distributedCache.Delete(testKey)
	}

	return nil
}

// Close closes the cache manager
func (cm *CacheManager) Close() error {
	if cm.localCache != nil {
		cm.localCache.Stop()
	}
	return nil
}

// Helper methods

func (cm *CacheManager) buildKey(key string) string {
	if cm.keyPrefix == "" {
		return key
	}
	return cm.keyPrefix + ":" + key
}

func (cm *CacheManager) incrementLocalHits() {
	if cm.config.EnableMetrics {
		cm.metrics.mu.Lock()
		cm.metrics.LocalHits++
		cm.metrics.mu.Unlock()
	}
}

func (cm *CacheManager) incrementLocalMisses() {
	if cm.config.EnableMetrics {
		cm.metrics.mu.Lock()
		cm.metrics.LocalMisses++
		cm.metrics.mu.Unlock()
	}
}

func (cm *CacheManager) incrementDistributedHits() {
	if cm.config.EnableMetrics {
		cm.metrics.mu.Lock()
		cm.metrics.DistributedHits++
		cm.metrics.mu.Unlock()
	}
}

func (cm *CacheManager) incrementDistributedMisses() {
	if cm.config.EnableMetrics {
		cm.metrics.mu.Lock()
		cm.metrics.DistributedMisses++
		cm.metrics.mu.Unlock()
	}
}

func (cm *CacheManager) incrementErrors() {
	if cm.config.EnableMetrics {
		cm.metrics.mu.Lock()
		cm.metrics.Errors++
		cm.metrics.mu.Unlock()
	}
}

func (cm *CacheManager) incrementTotalOperations() {
	if cm.config.EnableMetrics {
		cm.metrics.mu.Lock()
		cm.metrics.TotalOperations++
		cm.metrics.mu.Unlock()
	}
}

// CacheStats represents cache statistics
type CacheStats struct {
	LocalHits              int64   `json:"local_hits"`
	LocalMisses            int64   `json:"local_misses"`
	LocalHitRate           float64 `json:"local_hit_rate"`
	LocalSize              int64   `json:"local_size"`
	LocalMaxSize           int64   `json:"local_max_size"`
	DistributedHits        int64   `json:"distributed_hits"`
	DistributedMisses      int64   `json:"distributed_misses"`
	DistributedHitRate     float64 `json:"distributed_hit_rate"`
	LocalEvictions         int64   `json:"local_evictions"`
	Errors                 int64   `json:"errors"`
	TotalOperations        int64   `json:"total_operations"`
	MemcachedConnected     bool    `json:"memcached_connected"`
}

// DefaultConfig returns default cache configuration
func DefaultConfig() *Config {
	return &Config{
		LocalTTL:              5 * time.Minute,
		DistributedTTL:        15 * time.Minute,
		MaxLocalSize:          1000000, // 1M items
		LocalItemsToPrune:     100,
		MemcachedHosts:        []string{"localhost:11211"},
		MemcachedTimeout:      5 * time.Second,
		MemcachedMaxIdleConns: 100,
		KeyPrefix:             "search",
		EnableMetrics:         true,
	}
}

// CacheKeyBuilder helps build consistent cache keys
type CacheKeyBuilder struct {
	prefix string
}

// NewCacheKeyBuilder creates a new cache key builder
func NewCacheKeyBuilder(prefix string) *CacheKeyBuilder {
	return &CacheKeyBuilder{prefix: prefix}
}

// SearchKey builds a cache key for search results
func (ckb *CacheKeyBuilder) SearchKey(query string, page, limit int, filters map[string]interface{}) string {
	parts := []string{ckb.prefix, "search", "q:" + query, fmt.Sprintf("p:%d", page), fmt.Sprintf("l:%d", limit)}

	// Add filters to key
	if len(filters) > 0 {
		for k, v := range filters {
			parts = append(parts, fmt.Sprintf("%s:%v", k, v))
		}
	}

	return strings.Join(parts, ":")
}

// TrendingKey builds a cache key for trending results
func (ckb *CacheKeyBuilder) TrendingKey(period string, limit int) string {
	return fmt.Sprintf("%s:trending:%s:limit:%d", ckb.prefix, period, limit)
}

// SuggestionsKey builds a cache key for suggestions
func (ckb *CacheKeyBuilder) SuggestionsKey(query string, limit int) string {
	return fmt.Sprintf("%s:suggestions:q:%s:limit:%d", ckb.prefix, query, limit)
}

// CryptoKey builds a cache key for individual crypto data
func (ckb *CacheKeyBuilder) CryptoKey(id string) string {
	return fmt.Sprintf("%s:crypto:%s", ckb.prefix, id)
}

// FiltersKey builds a cache key for filter data
func (ckb *CacheKeyBuilder) FiltersKey() string {
	return fmt.Sprintf("%s:filters:all", ckb.prefix)
}