package cache

import (
	"context"
	"fmt"
	"time"

	"market-data-api/internal/models"
)

// Cache defines the interface for caching operations
type Cache interface {
	// Basic operations
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Advanced operations
	GetSet(ctx context.Context, key string, value []byte, ttl time.Duration) ([]byte, error)
	SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)
	MSet(ctx context.Context, keyValues map[string][]byte, ttl time.Duration) error

	// List operations (for time-series data)
	LPush(ctx context.Context, key string, values ...[]byte) error
	RPush(ctx context.Context, key string, values ...[]byte) error
	LPop(ctx context.Context, key string) ([]byte, error)
	RPop(ctx context.Context, key string) ([]byte, error)
	LRange(ctx context.Context, key string, start, stop int64) ([][]byte, error)
	LTrim(ctx context.Context, key string, start, stop int64) error
	LLen(ctx context.Context, key string) (int64, error)

	// Set operations (for collections)
	SAdd(ctx context.Context, key string, members ...[]byte) error
	SRem(ctx context.Context, key string, members ...[]byte) error
	SMembers(ctx context.Context, key string) ([][]byte, error)
	SIsMember(ctx context.Context, key string, member []byte) (bool, error)
	SCard(ctx context.Context, key string) (int64, error)

	// Hash operations (for structured data)
	HSet(ctx context.Context, key string, field string, value []byte) error
	HGet(ctx context.Context, key string, field string) ([]byte, error)
	HMSet(ctx context.Context, key string, fieldValues map[string][]byte) error
	HMGet(ctx context.Context, key string, fields []string) (map[string][]byte, error)
	HGetAll(ctx context.Context, key string) (map[string][]byte, error)
	HDel(ctx context.Context, key string, fields ...string) error
	HExists(ctx context.Context, key string, field string) (bool, error)
	HKeys(ctx context.Context, key string) ([]string, error)

	// Sorted set operations (for rankings, time-series with scores)
	ZAdd(ctx context.Context, key string, score float64, member []byte) error
	ZRem(ctx context.Context, key string, members ...[]byte) error
	ZRange(ctx context.Context, key string, start, stop int64) ([][]byte, error)
	ZRangeByScore(ctx context.Context, key string, min, max float64, limit int64) ([][]byte, error)
	ZRevRange(ctx context.Context, key string, start, stop int64) ([][]byte, error)
	ZCard(ctx context.Context, key string) (int64, error)
	ZScore(ctx context.Context, key string, member []byte) (float64, error)

	// Expiration operations
	Expire(ctx context.Context, key string, ttl time.Duration) error
	ExpireAt(ctx context.Context, key string, at time.Time) error
	Persist(ctx context.Context, key string) error

	// Pattern operations
	Keys(ctx context.Context, pattern string) ([]string, error)
	Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error)

	// Pipeline operations for batch processing
	Pipeline() Pipeline

	// Health and monitoring
	Ping(ctx context.Context) error
	Info(ctx context.Context) (map[string]string, error)
	FlushAll(ctx context.Context) error
	Close() error
}

// Pipeline defines the interface for pipelined operations
type Pipeline interface {
	Get(key string) *StringCmd
	Set(key string, value []byte, ttl time.Duration) *StatusCmd
	Del(keys ...string) *IntCmd
	HSet(key string, field string, value []byte) *BoolCmd
	HGet(key string, field string) *StringCmd
	ZAdd(key string, score float64, member []byte) *IntCmd
	ZRange(key string, start, stop int64) *StringSliceCmd
	Expire(key string, ttl time.Duration) *BoolCmd
	Exec(ctx context.Context) ([]Cmd, error)
	Discard() error
}

// Command result interfaces
type Cmd interface {
	Err() error
}

type StringCmd interface {
	Cmd
	Result() ([]byte, error)
	Val() []byte
}

type StatusCmd interface {
	Cmd
	Result() (string, error)
	Val() string
}

type IntCmd interface {
	Cmd
	Result() (int64, error)
	Val() int64
}

type BoolCmd interface {
	Cmd
	Result() (bool, error)
	Val() bool
}

type StringSliceCmd interface {
	Cmd
	Result() ([][]byte, error)
	Val() [][]byte
}

type FloatCmd interface {
	Cmd
	Result() (float64, error)
	Val() float64
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	// Connection
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`

	// Connection pool
	PoolSize           int           `json:"pool_size"`
	MinIdleConnections int           `json:"min_idle_connections"`
	MaxRetries         int           `json:"max_retries"`
	RetryDelay         time.Duration `json:"retry_delay"`
	DialTimeout        time.Duration `json:"dial_timeout"`
	ReadTimeout        time.Duration `json:"read_timeout"`
	WriteTimeout       time.Duration `json:"write_timeout"`
	PoolTimeout        time.Duration `json:"pool_timeout"`
	IdleTimeout        time.Duration `json:"idle_timeout"`

	// Clustering (for Redis Cluster)
	EnableCluster bool     `json:"enable_cluster"`
	ClusterNodes  []string `json:"cluster_nodes"`

	// SSL/TLS
	EnableTLS     bool   `json:"enable_tls"`
	TLSCertFile   string `json:"tls_cert_file"`
	TLSKeyFile    string `json:"tls_key_file"`
	TLSCAFile     string `json:"tls_ca_file"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`

	// Performance tuning
	EnablePipelining bool `json:"enable_pipelining"`
	PipelineSize     int  `json:"pipeline_size"`

	// Monitoring
	EnableMetrics   bool          `json:"enable_metrics"`
	MetricsInterval time.Duration `json:"metrics_interval"`
}

// CacheMetrics represents cache performance metrics
type CacheMetrics struct {
	// Operation counts
	GetCount   int64 `json:"get_count"`
	SetCount   int64 `json:"set_count"`
	DelCount   int64 `json:"del_count"`
	HitCount   int64 `json:"hit_count"`
	MissCount  int64 `json:"miss_count"`
	ErrorCount int64 `json:"error_count"`

	// Performance metrics
	AvgLatency time.Duration `json:"avg_latency"`
	MaxLatency time.Duration `json:"max_latency"`
	MinLatency time.Duration `json:"min_latency"`

	// Connection metrics
	ActiveConnections int `json:"active_connections"`
	IdleConnections   int `json:"idle_connections"`
	TotalConnections  int `json:"total_connections"`

	// Memory usage
	UsedMemory  int64   `json:"used_memory"`
	MaxMemory   int64   `json:"max_memory"`
	MemoryUsage float64 `json:"memory_usage_percent"`

	// Cache effectiveness
	HitRatio    float64   `json:"hit_ratio"`
	LastUpdated time.Time `json:"last_updated"`
}

// SpecializedCache interfaces for specific data types

// PriceCache provides specialized caching for price data
type PriceCache interface {
	// Price operations
	SetPrice(ctx context.Context, symbol string, price *models.AggregatedPrice, ttl time.Duration) error
	GetPrice(ctx context.Context, symbol string) (*models.AggregatedPrice, error)
	SetPrices(ctx context.Context, prices map[string]*models.AggregatedPrice, ttl time.Duration) error
	GetPrices(ctx context.Context, symbols []string) (map[string]*models.AggregatedPrice, error)
	DelPrice(ctx context.Context, symbols ...string) error

	// Historical data operations
	SetHistoricalData(ctx context.Context, symbol string, interval string, data []*models.Candle, ttl time.Duration) error
	GetHistoricalData(ctx context.Context, symbol string, interval string) ([]*models.Candle, error)
	AppendHistoricalData(ctx context.Context, symbol string, interval string, data []*models.Candle) error

	// Market data operations
	SetMarketData(ctx context.Context, symbol string, data *models.MarketData, ttl time.Duration) error
	GetMarketData(ctx context.Context, symbol string) (*models.MarketData, error)

	// Order book operations
	SetOrderBook(ctx context.Context, symbol string, orderBook *models.OrderBook, ttl time.Duration) error
	GetOrderBook(ctx context.Context, symbol string) (*models.OrderBook, error)

	// Statistical data operations (commented out - StatisticalData not implemented)
	// SetStatistics(ctx context.Context, symbol string, stats *models.StatisticalData, ttl time.Duration) error
	// GetStatistics(ctx context.Context, symbol string) (*models.StatisticalData, error)

	// Technical indicators operations
	SetTechnicalIndicators(ctx context.Context, symbol string, indicators *models.TechnicalIndicators, ttl time.Duration) error
	GetTechnicalIndicators(ctx context.Context, symbol string) (*models.TechnicalIndicators, error)

	// Volatility data operations
	SetVolatilityData(ctx context.Context, symbol string, volatility *models.VolatilityData, ttl time.Duration) error
	GetVolatilityData(ctx context.Context, symbol string) (*models.VolatilityData, error)
}

// TimeSeriesCache provides time-series specific caching operations
type TimeSeriesCache interface {
	// Time-series operations
	AddDataPoint(ctx context.Context, key string, timestamp time.Time, value []byte) error
	GetDataPoints(ctx context.Context, key string, from, to time.Time) ([]TimeSeriesPoint, error)
	GetLatestDataPoints(ctx context.Context, key string, count int64) ([]TimeSeriesPoint, error)
	DeleteDataPoints(ctx context.Context, key string, from, to time.Time) error
	GetTimeSeriesInfo(ctx context.Context, key string) (*TimeSeriesInfo, error)

	// Aggregation operations
	GetAggregatedData(ctx context.Context, key string, from, to time.Time, aggregation string, bucketSize time.Duration) ([]AggregatedPoint, error)

	// Retention management
	SetRetention(ctx context.Context, key string, retention time.Duration) error
	CompactData(ctx context.Context, key string, from, to time.Time, bucketSize time.Duration) error
}

// TimeSeriesPoint represents a point in time-series data
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     []byte    `json:"value"`
}

// AggregatedPoint represents an aggregated time-series point
type AggregatedPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     []byte    `json:"value"`
	Count     int64     `json:"count"`
	Min       []byte    `json:"min,omitempty"`
	Max       []byte    `json:"max,omitempty"`
	Avg       []byte    `json:"avg,omitempty"`
}

// TimeSeriesInfo provides metadata about time-series data
type TimeSeriesInfo struct {
	Key         string        `json:"key"`
	FirstTS     time.Time     `json:"first_timestamp"`
	LastTS      time.Time     `json:"last_timestamp"`
	TotalPoints int64         `json:"total_points"`
	Retention   time.Duration `json:"retention"`
}

// SessionCache provides session management capabilities
type SessionCache interface {
	// Session operations
	CreateSession(ctx context.Context, sessionID string, data interface{}, ttl time.Duration) error
	GetSession(ctx context.Context, sessionID string) (interface{}, error)
	UpdateSession(ctx context.Context, sessionID string, data interface{}, ttl time.Duration) error
	DeleteSession(ctx context.Context, sessionID string) error
	RefreshSession(ctx context.Context, sessionID string, ttl time.Duration) error

	// Session queries
	GetActiveSessions(ctx context.Context) ([]string, error)
	GetSessionTTL(ctx context.Context, sessionID string) (time.Duration, error)
	SessionExists(ctx context.Context, sessionID string) (bool, error)
}

// DistributedLockCache provides distributed locking capabilities
type DistributedLockCache interface {
	// Lock operations
	AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key string) error
	RefreshLock(ctx context.Context, key string, ttl time.Duration) error
	IsLocked(ctx context.Context, key string) (bool, error)
	GetLockTTL(ctx context.Context, key string) (time.Duration, error)

	// Lock with retry
	AcquireLockWithRetry(ctx context.Context, key string, ttl time.Duration, maxRetries int, retryDelay time.Duration) (bool, error)
}

// CacheFactory creates cache instances
type CacheFactory interface {
	CreateCache(config *CacheConfig) (Cache, error)
	CreatePriceCache(config *CacheConfig) (PriceCache, error)
	CreateTimeSeriesCache(config *CacheConfig) (TimeSeriesCache, error)
	CreateSessionCache(config *CacheConfig) (SessionCache, error)
	CreateDistributedLockCache(config *CacheConfig) (DistributedLockCache, error)
}

// CacheError represents cache-specific errors
type CacheError struct {
	Operation string
	Key       string
	Err       error
	Code      string
}

func (e *CacheError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("cache %s operation failed for key '%s': %v", e.Operation, e.Key, e.Err)
	}
	return fmt.Sprintf("cache %s operation failed: %v", e.Operation, e.Err)
}

// Common error codes
const (
	ErrCodeKeyNotFound      = "KEY_NOT_FOUND"
	ErrCodeConnectionFailed = "CONNECTION_FAILED"
	ErrCodeTimeout          = "TIMEOUT"
	ErrCodeSerialization    = "SERIALIZATION_ERROR"
	ErrCodeInvalidKey       = "INVALID_KEY"
	ErrCodeCacheFull        = "CACHE_FULL"
)

// NewCacheError creates a new cache error
func NewCacheError(operation, key, code string, err error) *CacheError {
	return &CacheError{
		Operation: operation,
		Key:       key,
		Err:       err,
		Code:      code,
	}
}

// IsNotFound checks if error is a "not found" error
func IsNotFound(err error) bool {
	if cacheErr, ok := err.(*CacheError); ok {
		return cacheErr.Code == ErrCodeKeyNotFound
	}
	return false
}

// IsTimeout checks if error is a timeout error
func IsTimeout(err error) bool {
	if cacheErr, ok := err.(*CacheError); ok {
		return cacheErr.Code == ErrCodeTimeout
	}
	return false
}

// IsConnectionFailed checks if error is a connection error
func IsConnectionFailed(err error) bool {
	if cacheErr, ok := err.(*CacheError); ok {
		return cacheErr.Code == ErrCodeConnectionFailed
	}
	return false
}
