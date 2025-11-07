package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"market-data-api/internal/models"
)

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	client  redis.UniversalClient
	config  *CacheConfig
	metrics *CacheMetrics
	mu      sync.RWMutex
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(config *CacheConfig) (*RedisCache, error) {
	if config == nil {
		config = getDefaultRedisConfig()
	}

	var client redis.UniversalClient

	if config.EnableCluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:       config.ClusterNodes,
			Password:    config.Password,
			MaxRetries:  config.MaxRetries,
			PoolSize:    config.PoolSize,
			MinIdleConns: config.MinIdleConnections,
			DialTimeout:  config.DialTimeout,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			PoolTimeout:  config.PoolTimeout,
			ConnMaxIdleTime: config.IdleTimeout,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
			Password:     config.Password,
			DB:          config.DB,
			MaxRetries:  config.MaxRetries,
			PoolSize:    config.PoolSize,
			MinIdleConns: config.MinIdleConnections,
			DialTimeout:  config.DialTimeout,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			PoolTimeout:  config.PoolTimeout,
			ConnMaxIdleTime: config.IdleTimeout,
		})
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, NewCacheError("connect", "", ErrCodeConnectionFailed, err)
	}

	cache := &RedisCache{
		client:  client,
		config:  config,
		metrics: &CacheMetrics{},
	}

	// Start metrics collection if enabled
	if config.EnableMetrics {
		go cache.collectMetrics()
	}

	return cache, nil
}

// Basic operations

func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()
	defer r.recordOperation("get", start)

	result, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			r.recordMiss()
			return nil, NewCacheError("get", key, ErrCodeKeyNotFound, err)
		}
		r.recordError()
		return nil, NewCacheError("get", key, ErrCodeConnectionFailed, err)
	}

	r.recordHit()
	return result, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	start := time.Now()
	defer r.recordOperation("set", start)

	err := r.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("set", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) Del(ctx context.Context, keys ...string) error {
	start := time.Now()
	defer r.recordOperation("del", start)

	err := r.client.Del(ctx, keys...).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("del", "", ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	defer r.recordOperation("exists", start)

	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		r.recordError()
		return false, NewCacheError("exists", key, ErrCodeConnectionFailed, err)
	}

	return count > 0, nil
}

func (r *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	start := time.Now()
	defer r.recordOperation("ttl", start)

	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		r.recordError()
		return 0, NewCacheError("ttl", key, ErrCodeConnectionFailed, err)
	}

	return ttl, nil
}

// Advanced operations

func (r *RedisCache) GetSet(ctx context.Context, key string, value []byte, ttl time.Duration) ([]byte, error) {
	start := time.Now()
	defer r.recordOperation("getset", start)

	oldValue, err := r.client.GetSet(ctx, key, value).Bytes()
	if err != nil && err != redis.Nil {
		r.recordError()
		return nil, NewCacheError("getset", key, ErrCodeConnectionFailed, err)
	}

	// Set TTL separately since GetSet doesn't support TTL
	if ttl > 0 {
		r.client.Expire(ctx, key, ttl)
	}

	return oldValue, nil
}

func (r *RedisCache) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	start := time.Now()
	defer r.recordOperation("setnx", start)

	success, err := r.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		r.recordError()
		return false, NewCacheError("setnx", key, ErrCodeConnectionFailed, err)
	}

	return success, nil
}

func (r *RedisCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	start := time.Now()
	defer r.recordOperation("mget", start)

	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		r.recordError()
		return nil, NewCacheError("mget", "", ErrCodeConnectionFailed, err)
	}

	result := make(map[string][]byte)
	for i, key := range keys {
		if i < len(values) && values[i] != nil {
			if str, ok := values[i].(string); ok {
				result[key] = []byte(str)
				r.recordHit()
			} else {
				r.recordMiss()
			}
		} else {
			r.recordMiss()
		}
	}

	return result, nil
}

func (r *RedisCache) MSet(ctx context.Context, keyValues map[string][]byte, ttl time.Duration) error {
	start := time.Now()
	defer r.recordOperation("mset", start)

	// Convert to interface{} for Redis
	pairs := make([]interface{}, 0, len(keyValues)*2)
	for key, value := range keyValues {
		pairs = append(pairs, key, value)
	}

	pipe := r.client.Pipeline()
	pipe.MSet(ctx, pairs...)

	// Set TTL for each key if specified
	if ttl > 0 {
		for key := range keyValues {
			pipe.Expire(ctx, key, ttl)
		}
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.recordError()
		return NewCacheError("mset", "", ErrCodeConnectionFailed, err)
	}

	return nil
}

// List operations

func (r *RedisCache) LPush(ctx context.Context, key string, values ...[]byte) error {
	start := time.Now()
	defer r.recordOperation("lpush", start)

	// Convert []byte to interface{}
	interfaceValues := make([]interface{}, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}

	err := r.client.LPush(ctx, key, interfaceValues...).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("lpush", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) RPush(ctx context.Context, key string, values ...[]byte) error {
	start := time.Now()
	defer r.recordOperation("rpush", start)

	// Convert []byte to interface{}
	interfaceValues := make([]interface{}, len(values))
	for i, v := range values {
		interfaceValues[i] = v
	}

	err := r.client.RPush(ctx, key, interfaceValues...).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("rpush", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) LPop(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()
	defer r.recordOperation("lpop", start)

	result, err := r.client.LPop(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, NewCacheError("lpop", key, ErrCodeKeyNotFound, err)
		}
		r.recordError()
		return nil, NewCacheError("lpop", key, ErrCodeConnectionFailed, err)
	}

	return result, nil
}

func (r *RedisCache) RPop(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()
	defer r.recordOperation("rpop", start)

	result, err := r.client.RPop(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, NewCacheError("rpop", key, ErrCodeKeyNotFound, err)
		}
		r.recordError()
		return nil, NewCacheError("rpop", key, ErrCodeConnectionFailed, err)
	}

	return result, nil
}

func (r *RedisCache) LRange(ctx context.Context, key string, start, stop int64) ([][]byte, error) {
	startTime := time.Now()
	defer r.recordOperation("lrange", startTime)

	results, err := r.client.LRange(ctx, key, start, stop).Result()
	if err != nil {
		r.recordError()
		return nil, NewCacheError("lrange", key, ErrCodeConnectionFailed, err)
	}

	bytes := make([][]byte, len(results))
	for i, result := range results {
		bytes[i] = []byte(result)
	}

	return bytes, nil
}

func (r *RedisCache) LTrim(ctx context.Context, key string, start, stop int64) error {
	startTime := time.Now()
	defer r.recordOperation("ltrim", startTime)

	err := r.client.LTrim(ctx, key, start, stop).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("ltrim", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) LLen(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	defer r.recordOperation("llen", start)

	length, err := r.client.LLen(ctx, key).Result()
	if err != nil {
		r.recordError()
		return 0, NewCacheError("llen", key, ErrCodeConnectionFailed, err)
	}

	return length, nil
}

// Set operations

func (r *RedisCache) SAdd(ctx context.Context, key string, members ...[]byte) error {
	start := time.Now()
	defer r.recordOperation("sadd", start)

	// Convert []byte to interface{}
	interfaceMembers := make([]interface{}, len(members))
	for i, m := range members {
		interfaceMembers[i] = m
	}

	err := r.client.SAdd(ctx, key, interfaceMembers...).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("sadd", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) SRem(ctx context.Context, key string, members ...[]byte) error {
	start := time.Now()
	defer r.recordOperation("srem", start)

	// Convert []byte to interface{}
	interfaceMembers := make([]interface{}, len(members))
	for i, m := range members {
		interfaceMembers[i] = m
	}

	err := r.client.SRem(ctx, key, interfaceMembers...).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("srem", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) SMembers(ctx context.Context, key string) ([][]byte, error) {
	start := time.Now()
	defer r.recordOperation("smembers", start)

	members, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		r.recordError()
		return nil, NewCacheError("smembers", key, ErrCodeConnectionFailed, err)
	}

	result := make([][]byte, len(members))
	for i, member := range members {
		result[i] = []byte(member)
	}

	return result, nil
}

func (r *RedisCache) SIsMember(ctx context.Context, key string, member []byte) (bool, error) {
	start := time.Now()
	defer r.recordOperation("sismember", start)

	isMember, err := r.client.SIsMember(ctx, key, member).Result()
	if err != nil {
		r.recordError()
		return false, NewCacheError("sismember", key, ErrCodeConnectionFailed, err)
	}

	return isMember, nil
}

func (r *RedisCache) SCard(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	defer r.recordOperation("scard", start)

	count, err := r.client.SCard(ctx, key).Result()
	if err != nil {
		r.recordError()
		return 0, NewCacheError("scard", key, ErrCodeConnectionFailed, err)
	}

	return count, nil
}

// Hash operations

func (r *RedisCache) HSet(ctx context.Context, key string, field string, value []byte) error {
	start := time.Now()
	defer r.recordOperation("hset", start)

	err := r.client.HSet(ctx, key, field, value).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("hset", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) HGet(ctx context.Context, key string, field string) ([]byte, error) {
	start := time.Now()
	defer r.recordOperation("hget", start)

	result, err := r.client.HGet(ctx, key, field).Bytes()
	if err != nil {
		if err == redis.Nil {
			r.recordMiss()
			return nil, NewCacheError("hget", key, ErrCodeKeyNotFound, err)
		}
		r.recordError()
		return nil, NewCacheError("hget", key, ErrCodeConnectionFailed, err)
	}

	r.recordHit()
	return result, nil
}

func (r *RedisCache) HMSet(ctx context.Context, key string, fieldValues map[string][]byte) error {
	start := time.Now()
	defer r.recordOperation("hmset", start)

	// Convert to interface{}
	values := make(map[string]interface{})
	for field, value := range fieldValues {
		values[field] = value
	}

	err := r.client.HMSet(ctx, key, values).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("hmset", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) HMGet(ctx context.Context, key string, fields []string) (map[string][]byte, error) {
	start := time.Now()
	defer r.recordOperation("hmget", start)

	values, err := r.client.HMGet(ctx, key, fields...).Result()
	if err != nil {
		r.recordError()
		return nil, NewCacheError("hmget", key, ErrCodeConnectionFailed, err)
	}

	result := make(map[string][]byte)
	for i, field := range fields {
		if i < len(values) && values[i] != nil {
			if str, ok := values[i].(string); ok {
				result[field] = []byte(str)
				r.recordHit()
			} else {
				r.recordMiss()
			}
		} else {
			r.recordMiss()
		}
	}

	return result, nil
}

func (r *RedisCache) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	start := time.Now()
	defer r.recordOperation("hgetall", start)

	values, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		r.recordError()
		return nil, NewCacheError("hgetall", key, ErrCodeConnectionFailed, err)
	}

	result := make(map[string][]byte)
	for field, value := range values {
		result[field] = []byte(value)
	}

	if len(result) > 0 {
		r.recordHit()
	} else {
		r.recordMiss()
	}

	return result, nil
}

func (r *RedisCache) HDel(ctx context.Context, key string, fields ...string) error {
	start := time.Now()
	defer r.recordOperation("hdel", start)

	err := r.client.HDel(ctx, key, fields...).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("hdel", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) HExists(ctx context.Context, key string, field string) (bool, error) {
	start := time.Now()
	defer r.recordOperation("hexists", start)

	exists, err := r.client.HExists(ctx, key, field).Result()
	if err != nil {
		r.recordError()
		return false, NewCacheError("hexists", key, ErrCodeConnectionFailed, err)
	}

	return exists, nil
}

func (r *RedisCache) HKeys(ctx context.Context, key string) ([]string, error) {
	start := time.Now()
	defer r.recordOperation("hkeys", start)

	keys, err := r.client.HKeys(ctx, key).Result()
	if err != nil {
		r.recordError()
		return nil, NewCacheError("hkeys", key, ErrCodeConnectionFailed, err)
	}

	return keys, nil
}

// Sorted set operations

func (r *RedisCache) ZAdd(ctx context.Context, key string, score float64, member []byte) error {
	start := time.Now()
	defer r.recordOperation("zadd", start)

	err := r.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("zadd", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) ZRem(ctx context.Context, key string, members ...[]byte) error {
	start := time.Now()
	defer r.recordOperation("zrem", start)

	// Convert []byte to interface{}
	interfaceMembers := make([]interface{}, len(members))
	for i, m := range members {
		interfaceMembers[i] = m
	}

	err := r.client.ZRem(ctx, key, interfaceMembers...).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("zrem", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) ZRange(ctx context.Context, key string, start, stop int64) ([][]byte, error) {
	startTime := time.Now()
	defer r.recordOperation("zrange", startTime)

	members, err := r.client.ZRange(ctx, key, start, stop).Result()
	if err != nil {
		r.recordError()
		return nil, NewCacheError("zrange", key, ErrCodeConnectionFailed, err)
	}

	result := make([][]byte, len(members))
	for i, member := range members {
		result[i] = []byte(member)
	}

	return result, nil
}

func (r *RedisCache) ZRangeByScore(ctx context.Context, key string, min, max float64, limit int64) ([][]byte, error) {
	start := time.Now()
	defer r.recordOperation("zrangebyscore", start)

	opt := &redis.ZRangeBy{
		Min: strconv.FormatFloat(min, 'f', -1, 64),
		Max: strconv.FormatFloat(max, 'f', -1, 64),
	}

	if limit > 0 {
		opt.Count = limit
	}

	members, err := r.client.ZRangeByScore(ctx, key, opt).Result()
	if err != nil {
		r.recordError()
		return nil, NewCacheError("zrangebyscore", key, ErrCodeConnectionFailed, err)
	}

	result := make([][]byte, len(members))
	for i, member := range members {
		result[i] = []byte(member)
	}

	return result, nil
}

func (r *RedisCache) ZRevRange(ctx context.Context, key string, start, stop int64) ([][]byte, error) {
	startTime := time.Now()
	defer r.recordOperation("zrevrange", startTime)

	members, err := r.client.ZRevRange(ctx, key, start, stop).Result()
	if err != nil {
		r.recordError()
		return nil, NewCacheError("zrevrange", key, ErrCodeConnectionFailed, err)
	}

	result := make([][]byte, len(members))
	for i, member := range members {
		result[i] = []byte(member)
	}

	return result, nil
}

func (r *RedisCache) ZCard(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	defer r.recordOperation("zcard", start)

	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		r.recordError()
		return 0, NewCacheError("zcard", key, ErrCodeConnectionFailed, err)
	}

	return count, nil
}

func (r *RedisCache) ZScore(ctx context.Context, key string, member []byte) (float64, error) {
	start := time.Now()
	defer r.recordOperation("zscore", start)

	score, err := r.client.ZScore(ctx, key, string(member)).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, NewCacheError("zscore", key, ErrCodeKeyNotFound, err)
		}
		r.recordError()
		return 0, NewCacheError("zscore", key, ErrCodeConnectionFailed, err)
	}

	return score, nil
}

// Expiration operations

func (r *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	start := time.Now()
	defer r.recordOperation("expire", start)

	err := r.client.Expire(ctx, key, ttl).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("expire", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) ExpireAt(ctx context.Context, key string, at time.Time) error {
	start := time.Now()
	defer r.recordOperation("expireat", start)

	err := r.client.ExpireAt(ctx, key, at).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("expireat", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) Persist(ctx context.Context, key string) error {
	start := time.Now()
	defer r.recordOperation("persist", start)

	err := r.client.Persist(ctx, key).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("persist", key, ErrCodeConnectionFailed, err)
	}

	return nil
}

// Pattern operations

func (r *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	start := time.Now()
	defer r.recordOperation("keys", start)

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		r.recordError()
		return nil, NewCacheError("keys", "", ErrCodeConnectionFailed, err)
	}

	return keys, nil
}

func (r *RedisCache) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	start := time.Now()
	defer r.recordOperation("scan", start)

	keys, newCursor, err := r.client.Scan(ctx, cursor, match, count).Result()
	if err != nil {
		r.recordError()
		return nil, 0, NewCacheError("scan", "", ErrCodeConnectionFailed, err)
	}

	return keys, newCursor, nil
}

// Pipeline operations

func (r *RedisCache) Pipeline() Pipeline {
	return &RedisPipeline{
		pipe: r.client.Pipeline(),
	}
}

// Health and monitoring

func (r *RedisCache) Ping(ctx context.Context) error {
	start := time.Now()
	defer r.recordOperation("ping", start)

	err := r.client.Ping(ctx).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("ping", "", ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) Info(ctx context.Context) (map[string]string, error) {
	start := time.Now()
	defer r.recordOperation("info", start)

	info, err := r.client.Info(ctx).Result()
	if err != nil {
		r.recordError()
		return nil, NewCacheError("info", "", ErrCodeConnectionFailed, err)
	}

	// Parse info string into map
	result := make(map[string]string)
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	return result, nil
}

func (r *RedisCache) FlushAll(ctx context.Context) error {
	start := time.Now()
	defer r.recordOperation("flushall", start)

	err := r.client.FlushAll(ctx).Err()
	if err != nil {
		r.recordError()
		return NewCacheError("flushall", "", ErrCodeConnectionFailed, err)
	}

	return nil
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

// Metrics methods

func (r *RedisCache) recordOperation(op string, start time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	latency := time.Since(start)

	// Update operation counts
	switch op {
	case "get", "mget", "hget", "hmget":
		r.metrics.GetCount++
	case "set", "mset", "hset", "hmset":
		r.metrics.SetCount++
	case "del", "hdel":
		r.metrics.DelCount++
	}

	// Update latency metrics
	if r.metrics.MinLatency == 0 || latency < r.metrics.MinLatency {
		r.metrics.MinLatency = latency
	}
	if latency > r.metrics.MaxLatency {
		r.metrics.MaxLatency = latency
	}

	// Update average latency
	if r.metrics.AvgLatency == 0 {
		r.metrics.AvgLatency = latency
	} else {
		r.metrics.AvgLatency = (r.metrics.AvgLatency + latency) / 2
	}

	r.metrics.LastUpdated = time.Now()
}

func (r *RedisCache) recordHit() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics.HitCount++
	r.updateHitRatio()
}

func (r *RedisCache) recordMiss() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics.MissCount++
	r.updateHitRatio()
}

func (r *RedisCache) recordError() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics.ErrorCount++
}

func (r *RedisCache) updateHitRatio() {
	total := r.metrics.HitCount + r.metrics.MissCount
	if total > 0 {
		r.metrics.HitRatio = float64(r.metrics.HitCount) / float64(total)
	}
}

func (r *RedisCache) collectMetrics() {
	ticker := time.NewTicker(r.config.MetricsInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		// Get Redis info
		if info, err := r.Info(ctx); err == nil {
			r.mu.Lock()
			if usedMemory, exists := info["used_memory"]; exists {
				if val, err := strconv.ParseInt(usedMemory, 10, 64); err == nil {
					r.metrics.UsedMemory = val
				}
			}
			if maxMemory, exists := info["maxmemory"]; exists {
				if val, err := strconv.ParseInt(maxMemory, 10, 64); err == nil {
					r.metrics.MaxMemory = val
				}
			}
			if r.metrics.MaxMemory > 0 {
				r.metrics.MemoryUsage = float64(r.metrics.UsedMemory) / float64(r.metrics.MaxMemory) * 100
			}
			r.mu.Unlock()
		}

		cancel()
	}
}

// GetMetrics returns current cache metrics
func (r *RedisCache) GetMetrics() *CacheMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *r.metrics
	return &metrics
}

// Helper functions

func getDefaultRedisConfig() *CacheConfig {
	return &CacheConfig{
		Host:               "localhost",
		Port:               6379,
		DB:                 0,
		PoolSize:           10,
		MinIdleConnections: 5,
		MaxRetries:         3,
		RetryDelay:         100 * time.Millisecond,
		DialTimeout:        5 * time.Second,
		ReadTimeout:        3 * time.Second,
		WriteTimeout:       3 * time.Second,
		PoolTimeout:        4 * time.Second,
		IdleTimeout:        5 * time.Minute,
		EnableMetrics:      true,
		MetricsInterval:    30 * time.Second,
	}
}