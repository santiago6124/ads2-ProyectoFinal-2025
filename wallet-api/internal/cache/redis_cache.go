package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"wallet-api/internal/models"
)

type CacheService interface {
	// Generic cache operations
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Wallet specific cache operations
	CacheWallet(ctx context.Context, wallet *models.Wallet, expiration time.Duration) error
	GetCachedWallet(ctx context.Context, walletID string) (*models.Wallet, error)
	InvalidateWallet(ctx context.Context, walletID string) error

	// Transaction specific cache operations
	CacheTransaction(ctx context.Context, transaction *models.Transaction, expiration time.Duration) error
	GetCachedTransaction(ctx context.Context, transactionID string) (*models.Transaction, error)
	InvalidateTransaction(ctx context.Context, transactionID string) error

	// User specific cache operations
	CacheUserWallets(ctx context.Context, userID int64, wallets []*models.Wallet, expiration time.Duration) error
	GetCachedUserWallets(ctx context.Context, userID int64) ([]*models.Wallet, error)
	InvalidateUserWallets(ctx context.Context, userID int64) error

	// Session and rate limiting
	SetSession(ctx context.Context, sessionID string, userID int64, expiration time.Duration) error
	GetSession(ctx context.Context, sessionID string) (int64, error)
	InvalidateSession(ctx context.Context, sessionID string) error

	// Counter operations for rate limiting
	Increment(ctx context.Context, key string, expiration time.Duration) (int64, error)
	IncrementBy(ctx context.Context, key string, value int64, expiration time.Duration) (int64, error)

	// List operations
	ListPush(ctx context.Context, key string, values ...interface{}) error
	ListPop(ctx context.Context, key string) (string, error)
	ListLength(ctx context.Context, key string) (int64, error)
	ListRange(ctx context.Context, key string, start, stop int64) ([]string, error)

	// Set operations
	SetAdd(ctx context.Context, key string, members ...interface{}) error
	SetMembers(ctx context.Context, key string) ([]string, error)
	SetIsMember(ctx context.Context, key string, member interface{}) (bool, error)

	// Health check
	Ping(ctx context.Context) error
	Close() error
}

type redisCache struct {
	client *redis.Client
	config *CacheConfig
}

type CacheConfig struct {
	Host         string
	Port         int
	Password     string
	Database     int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	Timeout      time.Duration
	KeyPrefix    string
}

func NewRedisCache(config *CacheConfig) (CacheService, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:     config.Password,
		DB:           config.Database,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.Timeout,
		ReadTimeout:  config.Timeout,
		WriteTimeout: config.Timeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &redisCache{
		client: rdb,
		config: config,
	}, nil
}

func (r *redisCache) buildKey(key string) string {
	if r.config.KeyPrefix != "" {
		return fmt.Sprintf("%s:%s", r.config.KeyPrefix, key)
	}
	return key
}

// Generic cache operations
func (r *redisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return r.client.Set(ctx, r.buildKey(key), data, expiration).Err()
}

func (r *redisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, r.buildKey(key)).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("failed to get value: %w", err)
	}

	return json.Unmarshal([]byte(data), dest)
}

func (r *redisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.buildKey(key)).Err()
}

func (r *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, r.buildKey(key)).Result()
	return result > 0, err
}

// Wallet specific cache operations
func (r *redisCache) CacheWallet(ctx context.Context, wallet *models.Wallet, expiration time.Duration) error {
	key := fmt.Sprintf("wallet:%s", wallet.ID.Hex())
	return r.Set(ctx, key, wallet, expiration)
}

func (r *redisCache) GetCachedWallet(ctx context.Context, walletID string) (*models.Wallet, error) {
	key := fmt.Sprintf("wallet:%s", walletID)
	var wallet models.Wallet
	err := r.Get(ctx, key, &wallet)
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *redisCache) InvalidateWallet(ctx context.Context, walletID string) error {
	key := fmt.Sprintf("wallet:%s", walletID)
	return r.Delete(ctx, key)
}

// Transaction specific cache operations
func (r *redisCache) CacheTransaction(ctx context.Context, transaction *models.Transaction, expiration time.Duration) error {
	key := fmt.Sprintf("transaction:%s", transaction.TransactionID)
	return r.Set(ctx, key, transaction, expiration)
}

func (r *redisCache) GetCachedTransaction(ctx context.Context, transactionID string) (*models.Transaction, error) {
	key := fmt.Sprintf("transaction:%s", transactionID)
	var transaction models.Transaction
	err := r.Get(ctx, key, &transaction)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *redisCache) InvalidateTransaction(ctx context.Context, transactionID string) error {
	key := fmt.Sprintf("transaction:%s", transactionID)
	return r.Delete(ctx, key)
}

// User specific cache operations
func (r *redisCache) CacheUserWallets(ctx context.Context, userID int64, wallets []*models.Wallet, expiration time.Duration) error {
	key := fmt.Sprintf("user:%d:wallets", userID)
	return r.Set(ctx, key, wallets, expiration)
}

func (r *redisCache) GetCachedUserWallets(ctx context.Context, userID int64) ([]*models.Wallet, error) {
	key := fmt.Sprintf("user:%d:wallets", userID)
	var wallets []*models.Wallet
	err := r.Get(ctx, key, &wallets)
	if err != nil {
		return nil, err
	}
	return wallets, nil
}

func (r *redisCache) InvalidateUserWallets(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("user:%d:wallets", userID)
	return r.Delete(ctx, key)
}

// Session and rate limiting
func (r *redisCache) SetSession(ctx context.Context, sessionID string, userID int64, expiration time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return r.Set(ctx, key, userID, expiration)
}

func (r *redisCache) GetSession(ctx context.Context, sessionID string) (int64, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	var userID int64
	err := r.Get(ctx, key, &userID)
	return userID, err
}

func (r *redisCache) InvalidateSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return r.Delete(ctx, key)
}

// Counter operations for rate limiting
func (r *redisCache) Increment(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	return r.IncrementBy(ctx, key, 1, expiration)
}

func (r *redisCache) IncrementBy(ctx context.Context, key string, value int64, expiration time.Duration) (int64, error) {
	pipe := r.client.TxPipeline()

	incrCmd := pipe.IncrBy(ctx, r.buildKey(key), value)
	pipe.Expire(ctx, r.buildKey(key), expiration)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment counter: %w", err)
	}

	return incrCmd.Val(), nil
}

// List operations
func (r *redisCache) ListPush(ctx context.Context, key string, values ...interface{}) error {
	return r.client.LPush(ctx, r.buildKey(key), values...).Err()
}

func (r *redisCache) ListPop(ctx context.Context, key string) (string, error) {
	result, err := r.client.RPop(ctx, r.buildKey(key)).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("list is empty")
	}
	return result, err
}

func (r *redisCache) ListLength(ctx context.Context, key string) (int64, error) {
	return r.client.LLen(ctx, r.buildKey(key)).Result()
}

func (r *redisCache) ListRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.LRange(ctx, r.buildKey(key), start, stop).Result()
}

// Set operations
func (r *redisCache) SetAdd(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SAdd(ctx, r.buildKey(key), members...).Err()
}

func (r *redisCache) SetMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, r.buildKey(key)).Result()
}

func (r *redisCache) SetIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.client.SIsMember(ctx, r.buildKey(key), member).Result()
}

// Health check
func (r *redisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *redisCache) Close() error {
	return r.client.Close()
}

// Cache strategies and patterns
type CacheStrategy struct {
	cache CacheService
}

func NewCacheStrategy(cache CacheService) *CacheStrategy {
	return &CacheStrategy{cache: cache}
}

// Cache-aside pattern for wallets
func (cs *CacheStrategy) GetWalletWithCache(ctx context.Context, walletID string, fetcher func() (*models.Wallet, error)) (*models.Wallet, error) {
	// Try to get from cache first
	wallet, err := cs.cache.GetCachedWallet(ctx, walletID)
	if err == nil {
		return wallet, nil
	}

	// Cache miss - fetch from source
	wallet, err = fetcher()
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := cs.cache.CacheWallet(ctx, wallet, 15*time.Minute); err != nil {
		// Log cache error but don't fail the request
		fmt.Printf("Failed to cache wallet %s: %v\n", walletID, err)
	}

	return wallet, nil
}

// Write-through pattern for transactions
func (cs *CacheStrategy) SaveTransactionWithCache(ctx context.Context, transaction *models.Transaction, saver func(*models.Transaction) error) error {
	// Save to primary store
	if err := saver(transaction); err != nil {
		return err
	}

	// Update cache
	if err := cs.cache.CacheTransaction(ctx, transaction, 30*time.Minute); err != nil {
		// Log cache error but don't fail the request
		fmt.Printf("Failed to cache transaction %s: %v\n", transaction.TransactionID, err)
	}

	return nil
}

// Cache invalidation helper
func (cs *CacheStrategy) InvalidateUserData(ctx context.Context, userID int64) error {
	// This would invalidate all user-related cache entries
	if err := cs.cache.InvalidateUserWallets(ctx, userID); err != nil {
		return fmt.Errorf("failed to invalidate user wallets cache: %w", err)
	}

	// Additional invalidations can be added here
	return nil
}

// Distributed cache lock implementation
type DistributedLock struct {
	cache CacheService
	key   string
	value string
	ttl   time.Duration
}

func NewDistributedLock(cache CacheService, key string, ttl time.Duration) *DistributedLock {
	return &DistributedLock{
		cache: cache,
		key:   fmt.Sprintf("lock:%s", key),
		value: fmt.Sprintf("%d", time.Now().UnixNano()),
		ttl:   ttl,
	}
}

func (dl *DistributedLock) Acquire(ctx context.Context) error {
	exists, err := dl.cache.Exists(ctx, dl.key)
	if err != nil {
		return fmt.Errorf("failed to check lock existence: %w", err)
	}

	if exists {
		return fmt.Errorf("lock already exists")
	}

	return dl.cache.Set(ctx, dl.key, dl.value, dl.ttl)
}

func (dl *DistributedLock) Release(ctx context.Context) error {
	return dl.cache.Delete(ctx, dl.key)
}

func (dl *DistributedLock) Extend(ctx context.Context, additionalTTL time.Duration) error {
	return dl.cache.Set(ctx, dl.key, dl.value, dl.ttl+additionalTTL)
}