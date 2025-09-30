package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type LockRepository interface {
	AcquireLock(ctx context.Context, key string, ttl time.Duration) (*DistributedLock, error)
	ReleaseLock(ctx context.Context, lock *DistributedLock) error
	ExtendLock(ctx context.Context, lock *DistributedLock, ttl time.Duration) error
	IsLocked(ctx context.Context, key string) (bool, error)
	CleanupExpiredLocks(ctx context.Context) error
}

type DistributedLock struct {
	Key       string
	Value     string
	TTL       time.Duration
	AcquiredAt time.Time
}

type lockRepository struct {
	client *redis.Client
}

func NewLockRepository(client *redis.Client) LockRepository {
	return &lockRepository{
		client: client,
	}
}

const (
	lockPrefix = "lock:"
	lockScript = `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`
)

func (r *lockRepository) AcquireLock(ctx context.Context, key string, ttl time.Duration) (*DistributedLock, error) {
	lockKey := lockPrefix + key
	lockValue := uuid.New().String()

	// Try to acquire the lock with SET NX EX
	result, err := r.client.SetNX(ctx, lockKey, lockValue, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !result {
		return nil, fmt.Errorf("lock already acquired for key: %s", key)
	}

	return &DistributedLock{
		Key:        lockKey,
		Value:      lockValue,
		TTL:        ttl,
		AcquiredAt: time.Now(),
	}, nil
}

func (r *lockRepository) ReleaseLock(ctx context.Context, lock *DistributedLock) error {
	// Use Lua script to ensure we only delete our own lock
	result, err := r.client.Eval(ctx, lockScript, []string{lock.Key}, lock.Value).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result.(int64) == 0 {
		return fmt.Errorf("lock not found or already released: %s", lock.Key)
	}

	return nil
}

func (r *lockRepository) ExtendLock(ctx context.Context, lock *DistributedLock, ttl time.Duration) error {
	// Check if we still own the lock and extend it
	extendScript := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("EXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := r.client.Eval(ctx, extendScript, []string{lock.Key}, lock.Value, int(ttl.Seconds())).Result()
	if err != nil {
		return fmt.Errorf("failed to extend lock: %w", err)
	}

	if result.(int64) == 0 {
		return fmt.Errorf("lock not found or not owned: %s", lock.Key)
	}

	lock.TTL = ttl
	return nil
}

func (r *lockRepository) IsLocked(ctx context.Context, key string) (bool, error) {
	lockKey := lockPrefix + key
	exists, err := r.client.Exists(ctx, lockKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check lock existence: %w", err)
	}

	return exists > 0, nil
}

func (r *lockRepository) CleanupExpiredLocks(ctx context.Context) error {
	// Redis automatically expires keys, but we can scan for any orphaned locks
	pattern := lockPrefix + "*"

	iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()

		// Check TTL
		ttl, err := r.client.TTL(ctx, key).Result()
		if err != nil {
			continue
		}

		// If TTL is -1, the key exists but has no expiration
		if ttl == -1 {
			// Delete keys without expiration (shouldn't happen, but cleanup)
			r.client.Del(ctx, key)
		}
	}

	return iter.Err()
}

// WalletLockManager provides high-level wallet locking operations
type WalletLockManager struct {
	lockRepo LockRepository
}

func NewWalletLockManager(lockRepo LockRepository) *WalletLockManager {
	return &WalletLockManager{
		lockRepo: lockRepo,
	}
}

func (m *WalletLockManager) LockWallet(ctx context.Context, walletID string, operation string, ttl time.Duration) (*DistributedLock, error) {
	lockKey := fmt.Sprintf("wallet:%s:%s", walletID, operation)
	return m.lockRepo.AcquireLock(ctx, lockKey, ttl)
}

func (m *WalletLockManager) LockTransaction(ctx context.Context, transactionID string, ttl time.Duration) (*DistributedLock, error) {
	lockKey := fmt.Sprintf("transaction:%s", transactionID)
	return m.lockRepo.AcquireLock(ctx, lockKey, ttl)
}

func (m *WalletLockManager) LockIdempotency(ctx context.Context, idempotencyKey string, ttl time.Duration) (*DistributedLock, error) {
	lockKey := fmt.Sprintf("idempotency:%s", idempotencyKey)
	return m.lockRepo.AcquireLock(ctx, lockKey, ttl)
}

func (m *WalletLockManager) LockUser(ctx context.Context, userID int64, operation string, ttl time.Duration) (*DistributedLock, error) {
	lockKey := fmt.Sprintf("user:%d:%s", userID, operation)
	return m.lockRepo.AcquireLock(ctx, lockKey, ttl)
}

func (m *WalletLockManager) ReleaseLock(ctx context.Context, lock *DistributedLock) error {
	return m.lockRepo.ReleaseLock(ctx, lock)
}

func (m *WalletLockManager) ExtendLock(ctx context.Context, lock *DistributedLock, ttl time.Duration) error {
	return m.lockRepo.ExtendLock(ctx, lock, ttl)
}

// IdempotencyRepository manages idempotency keys
type IdempotencyRepository interface {
	SetIdempotencyKey(ctx context.Context, key string, response interface{}, ttl time.Duration) error
	GetIdempotencyResponse(ctx context.Context, key string) (interface{}, bool, error)
	DeleteIdempotencyKey(ctx context.Context, key string) error
}

type idempotencyRepository struct {
	client *redis.Client
}

func NewIdempotencyRepository(client *redis.Client) IdempotencyRepository {
	return &idempotencyRepository{
		client: client,
	}
}

const idempotencyPrefix = "idempotency:"

func (r *idempotencyRepository) SetIdempotencyKey(ctx context.Context, key string, response interface{}, ttl time.Duration) error {
	idempotencyKey := idempotencyPrefix + key

	err := r.client.Set(ctx, idempotencyKey, response, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set idempotency key: %w", err)
	}

	return nil
}

func (r *idempotencyRepository) GetIdempotencyResponse(ctx context.Context, key string) (interface{}, bool, error) {
	idempotencyKey := idempotencyPrefix + key

	result, err := r.client.Get(ctx, idempotencyKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to get idempotency response: %w", err)
	}

	return result, true, nil
}

func (r *idempotencyRepository) DeleteIdempotencyKey(ctx context.Context, key string) error {
	idempotencyKey := idempotencyPrefix + key

	err := r.client.Del(ctx, idempotencyKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete idempotency key: %w", err)
	}

	return nil
}