package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"portfolio-api/internal/config"
)

// ErrNotFound is returned when a key is not found in cache
var ErrNotFound = errors.New("key not found in cache")

// RedisClient represents Redis cache client
type RedisClient struct {
	client *redis.Client
	config config.CacheConfig
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg config.CacheConfig) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:              cfg.DB,
		MaxRetries:      cfg.MaxRetries,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConnections,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolTimeout:     cfg.PoolTimeout,
		ConnMaxIdleTime: cfg.IdleTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: rdb,
		config: cfg,
	}, nil
}

// Set stores a value with TTL
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}

// Get retrieves a value and unmarshals it
func (r *RedisClient) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get key %s: %w", key, err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// Delete removes a key
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	return count > 0, err
}

// TTL returns the time to live for a key
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// Expire sets an expiration on a key
func (r *RedisClient) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

// Keys returns keys matching a pattern
func (r *RedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.client.Keys(ctx, pattern).Result()
}

// FlushAll removes all keys
func (r *RedisClient) FlushAll(ctx context.Context) error {
	return r.client.FlushAll(ctx).Err()
}

// Pipeline operations for batch processing
func (r *RedisClient) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

// Hash operations for structured data
func (r *RedisClient) HSet(ctx context.Context, key, field string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return r.client.HSet(ctx, key, field, data).Err()
}

func (r *RedisClient) HGet(ctx context.Context, key, field string, dest interface{}) error {
	data, err := r.client.HGet(ctx, key, field).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get hash field %s:%s: %w", key, field, err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

func (r *RedisClient) HDel(ctx context.Context, key string, fields ...string) error {
	return r.client.HDel(ctx, key, fields...).Err()
}

// List operations for time-series data
func (r *RedisClient) LPush(ctx context.Context, key string, values ...interface{}) error {
	serializedValues := make([]interface{}, len(values))
	for i, v := range values {
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value at index %d: %w", i, err)
		}
		serializedValues[i] = data
	}
	return r.client.LPush(ctx, key, serializedValues...).Err()
}

func (r *RedisClient) RPush(ctx context.Context, key string, values ...interface{}) error {
	serializedValues := make([]interface{}, len(values))
	for i, v := range values {
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value at index %d: %w", i, err)
		}
		serializedValues[i] = data
	}
	return r.client.RPush(ctx, key, serializedValues...).Err()
}

func (r *RedisClient) LRange(ctx context.Context, key string, start, stop int64, dest interface{}) error {
	data, err := r.client.LRange(ctx, key, start, stop).Result()
	if err != nil {
		return fmt.Errorf("failed to get list range: %w", err)
	}

	return json.Unmarshal([]byte(fmt.Sprintf("[%s]",
		strings.Join(data, ","))), dest)
}

func (r *RedisClient) LLen(ctx context.Context, key string) (int64, error) {
	return r.client.LLen(ctx, key).Result()
}

func (r *RedisClient) LTrim(ctx context.Context, key string, start, stop int64) error {
	return r.client.LTrim(ctx, key, start, stop).Err()
}

// Sorted set operations for rankings
func (r *RedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return r.client.ZAdd(ctx, key, members...).Err()
}

func (r *RedisClient) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.ZRange(ctx, key, start, stop).Result()
}

func (r *RedisClient) ZRangeByScore(ctx context.Context, key string, min, max string) ([]string, error) {
	return r.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
}

func (r *RedisClient) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.ZRem(ctx, key, members...).Err()
}

// Increment operations
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

func (r *RedisClient) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.client.IncrBy(ctx, key, value).Result()
}

func (r *RedisClient) IncrByFloat(ctx context.Context, key string, value float64) (float64, error) {
	return r.client.IncrByFloat(ctx, key, value).Result()
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Ping checks the Redis connection
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// GetInfo returns Redis server information
func (r *RedisClient) GetInfo(ctx context.Context) (string, error) {
	return r.client.Info(ctx).Result()
}

// GetStats returns cache statistics
func (r *RedisClient) GetStats(ctx context.Context) (map[string]string, error) {
	info, err := r.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]string)
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				stats[parts[0]] = parts[1]
			}
		}
	}

	return stats, nil
}

// Portfolio-specific cache methods

// SetPortfolio caches a portfolio
func (r *RedisClient) SetPortfolio(ctx context.Context, userID int64, portfolio interface{}) error {
	key := fmt.Sprintf("portfolio:%d", userID)
	return r.Set(ctx, key, portfolio, r.config.PortfolioTTL)
}

// GetPortfolio retrieves a cached portfolio
func (r *RedisClient) GetPortfolio(ctx context.Context, userID int64, dest interface{}) error {
	key := fmt.Sprintf("portfolio:%d", userID)
	return r.Get(ctx, key, dest)
}

// SetPerformance caches performance data
func (r *RedisClient) SetPerformance(ctx context.Context, userID int64, period string, performance interface{}) error {
	key := fmt.Sprintf("performance:%d:%s", userID, period)
	return r.Set(ctx, key, performance, r.config.PerformanceTTL)
}

// GetPerformance retrieves cached performance data
func (r *RedisClient) GetPerformance(ctx context.Context, userID int64, period string, dest interface{}) error {
	key := fmt.Sprintf("performance:%d:%s", userID, period)
	return r.Get(ctx, key, dest)
}

// SetSnapshot caches a snapshot
func (r *RedisClient) SetSnapshot(ctx context.Context, snapshotID string, snapshot interface{}) error {
	key := fmt.Sprintf("snapshot:%s", snapshotID)
	return r.Set(ctx, key, snapshot, r.config.SnapshotTTL)
}

// GetSnapshot retrieves a cached snapshot
func (r *RedisClient) GetSnapshot(ctx context.Context, snapshotID string, dest interface{}) error {
	key := fmt.Sprintf("snapshot:%s", snapshotID)
	return r.Get(ctx, key, dest)
}

// SetCalculation caches calculation results
func (r *RedisClient) SetCalculation(ctx context.Context, calculationKey string, result interface{}) error {
	key := fmt.Sprintf("calc:%s", calculationKey)
	return r.Set(ctx, key, result, r.config.CalculationTTL)
}

// GetCalculation retrieves cached calculation results
func (r *RedisClient) GetCalculation(ctx context.Context, calculationKey string, dest interface{}) error {
	key := fmt.Sprintf("calc:%s", calculationKey)
	return r.Get(ctx, key, dest)
}

// InvalidatePortfolio removes portfolio cache
func (r *RedisClient) InvalidatePortfolio(ctx context.Context, userID int64) error {
	pattern := fmt.Sprintf("*:%d*", userID)
	keys, err := r.Keys(ctx, pattern)
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return r.Delete(ctx, keys...)
	}

	return nil
}