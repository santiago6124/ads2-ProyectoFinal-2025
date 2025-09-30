package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"wallet-api/internal/repository"
)

type IdempotencyManager interface {
	ProcessIdempotentOperation(ctx context.Context, key string, operation func() (interface{}, error)) (interface{}, bool, error)
	GenerateIdempotencyKey(userID int64, operation string, params interface{}) string
	InvalidateIdempotencyKey(ctx context.Context, key string) error
}

type idempotencyManager struct {
	repo repository.IdempotencyRepository
	lockManager *repository.WalletLockManager
}

func NewIdempotencyManager(repo repository.IdempotencyRepository, lockManager *repository.WalletLockManager) IdempotencyManager {
	return &idempotencyManager{
		repo: repo,
		lockManager: lockManager,
	}
}

type IdempotentResult struct {
	Result    interface{} `json:"result"`
	Timestamp time.Time   `json:"timestamp"`
	Status    string      `json:"status"`
}

func (m *idempotencyManager) ProcessIdempotentOperation(ctx context.Context, key string, operation func() (interface{}, error)) (interface{}, bool, error) {
	// First, check if we already have a result for this key
	if existing, exists, err := m.repo.GetIdempotencyResponse(ctx, key); err == nil && exists {
		if result, ok := existing.(string); ok {
			var idempotentResult IdempotentResult
			if err := json.Unmarshal([]byte(result), &idempotentResult); err == nil {
				return idempotentResult.Result, true, nil
			}
		}
	}

	// Acquire lock for this idempotency key to prevent concurrent processing
	lock, err := m.lockManager.LockIdempotency(ctx, key, 30*time.Second)
	if err != nil {
		return nil, false, fmt.Errorf("failed to acquire idempotency lock: %w", err)
	}
	defer m.lockManager.ReleaseLock(ctx, lock)

	// Check again after acquiring lock (double-checked locking pattern)
	if existing, exists, err := m.repo.GetIdempotencyResponse(ctx, key); err == nil && exists {
		if result, ok := existing.(string); ok {
			var idempotentResult IdempotentResult
			if err := json.Unmarshal([]byte(result), &idempotentResult); err == nil {
				return idempotentResult.Result, true, nil
			}
		}
	}

	// Execute the operation
	result, err := operation()
	if err != nil {
		// Store failed result for a shorter time to allow retries
		failedResult := IdempotentResult{
			Result:    nil,
			Timestamp: time.Now(),
			Status:    "failed",
		}

		if data, marshalErr := json.Marshal(failedResult); marshalErr == nil {
			m.repo.SetIdempotencyKey(ctx, key, string(data), 5*time.Minute)
		}

		return nil, false, err
	}

	// Store successful result
	successResult := IdempotentResult{
		Result:    result,
		Timestamp: time.Now(),
		Status:    "success",
	}

	if data, err := json.Marshal(successResult); err == nil {
		// Store successful results for 24 hours
		m.repo.SetIdempotencyKey(ctx, key, string(data), 24*time.Hour)
	}

	return result, false, nil
}

func (m *idempotencyManager) GenerateIdempotencyKey(userID int64, operation string, params interface{}) string {
	// Create a deterministic key based on user, operation, and parameters
	data := struct {
		UserID    int64       `json:"user_id"`
		Operation string      `json:"operation"`
		Params    interface{} `json:"params"`
		Date      string      `json:"date"` // Include date to allow same operation on different days
	}{
		UserID:    userID,
		Operation: operation,
		Params:    params,
		Date:      time.Now().Format("2006-01-02"),
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

func (m *idempotencyManager) InvalidateIdempotencyKey(ctx context.Context, key string) error {
	return m.repo.DeleteIdempotencyKey(ctx, key)
}

// IdempotencyFilter provides middleware-like functionality for idempotent operations
type IdempotencyFilter struct {
	manager IdempotencyManager
}

func NewIdempotencyFilter(manager IdempotencyManager) *IdempotencyFilter {
	return &IdempotencyFilter{
		manager: manager,
	}
}

func (f *IdempotencyFilter) WithIdempotency(ctx context.Context, key string, operation func() (interface{}, error)) (interface{}, bool, error) {
	return f.manager.ProcessIdempotentOperation(ctx, key, operation)
}

// TransactionIdempotencyHandler handles idempotency for transaction operations
type TransactionIdempotencyHandler struct {
	manager IdempotencyManager
}

func NewTransactionIdempotencyHandler(manager IdempotencyManager) *TransactionIdempotencyHandler {
	return &TransactionIdempotencyHandler{
		manager: manager,
	}
}

func (h *TransactionIdempotencyHandler) ProcessDeposit(ctx context.Context, userID int64, amount string, currency string, operation func() (interface{}, error)) (interface{}, bool, error) {
	params := map[string]interface{}{
		"amount":   amount,
		"currency": currency,
	}
	key := h.manager.GenerateIdempotencyKey(userID, "deposit", params)
	return h.manager.ProcessIdempotentOperation(ctx, key, operation)
}

func (h *TransactionIdempotencyHandler) ProcessWithdrawal(ctx context.Context, userID int64, amount string, currency string, operation func() (interface{}, error)) (interface{}, bool, error) {
	params := map[string]interface{}{
		"amount":   amount,
		"currency": currency,
	}
	key := h.manager.GenerateIdempotencyKey(userID, "withdrawal", params)
	return h.manager.ProcessIdempotentOperation(ctx, key, operation)
}

func (h *TransactionIdempotencyHandler) ProcessLockFunds(ctx context.Context, userID int64, amount string, orderID string, operation func() (interface{}, error)) (interface{}, bool, error) {
	params := map[string]interface{}{
		"amount":   amount,
		"order_id": orderID,
	}
	key := h.manager.GenerateIdempotencyKey(userID, "lock_funds", params)
	return h.manager.ProcessIdempotentOperation(ctx, key, operation)
}

func (h *TransactionIdempotencyHandler) ProcessWithCustomKey(ctx context.Context, idempotencyKey string, operation func() (interface{}, error)) (interface{}, bool, error) {
	return h.manager.ProcessIdempotentOperation(ctx, idempotencyKey, operation)
}