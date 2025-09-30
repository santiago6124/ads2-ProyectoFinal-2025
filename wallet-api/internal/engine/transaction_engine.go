package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"wallet-api/internal/models"
	"wallet-api/internal/repository"
)

type TransactionEngine interface {
	ProcessTransaction(ctx context.Context, req *TransactionRequest) (*TransactionResult, error)
	LockFunds(ctx context.Context, req *LockFundsRequest) (*LockFundsResult, error)
	ReleaseFunds(ctx context.Context, req *ReleaseFundsRequest) (*ReleaseFundsResult, error)
	ExecuteLock(ctx context.Context, req *ExecuteLockRequest) (*ExecuteLockResult, error)
	ReverseTransaction(ctx context.Context, req *ReverseTransactionRequest) (*ReverseTransactionResult, error)
	GetTransactionStatus(ctx context.Context, transactionID string) (*TransactionStatusResult, error)
}

type transactionEngine struct {
	walletRepo      repository.WalletRepository
	transactionRepo repository.TransactionRepository
	lockManager     *repository.WalletLockManager
	idempotencyRepo repository.IdempotencyRepository
	db              *mongo.Database
}

func NewTransactionEngine(
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	lockManager *repository.WalletLockManager,
	idempotencyRepo repository.IdempotencyRepository,
	db *mongo.Database,
) TransactionEngine {
	return &transactionEngine{
		walletRepo:      walletRepo,
		transactionRepo: transactionRepo,
		lockManager:     lockManager,
		idempotencyRepo: idempotencyRepo,
		db:              db,
	}
}

type TransactionRequest struct {
	UserID         int64                 `json:"user_id"`
	Type           string                `json:"type"`
	Amount         decimal.Decimal       `json:"amount"`
	Fee            decimal.Decimal       `json:"fee"`
	Currency       string                `json:"currency"`
	Reference      models.Reference      `json:"reference"`
	IdempotencyKey string                `json:"idempotency_key"`
	Metadata       map[string]interface{} `json:"metadata"`
	AuditInfo      models.AuditInfo      `json:"audit_info"`
}

type TransactionResult struct {
	Transaction   *models.Transaction `json:"transaction"`
	Wallet        *models.Wallet      `json:"wallet"`
	Success       bool                `json:"success"`
	ErrorMessage  string              `json:"error_message,omitempty"`
	WasIdempotent bool                `json:"was_idempotent"`
}

type LockFundsRequest struct {
	UserID         int64           `json:"user_id"`
	Amount         decimal.Decimal `json:"amount"`
	OrderID        string          `json:"order_id"`
	Reason         string          `json:"reason"`
	ExpirationTime time.Duration   `json:"expiration_time"`
	IdempotencyKey string          `json:"idempotency_key"`
	AuditInfo      models.AuditInfo `json:"audit_info"`
}

type LockFundsResult struct {
	LockID        string          `json:"lock_id"`
	Success       bool            `json:"success"`
	ErrorMessage  string          `json:"error_message,omitempty"`
	WasIdempotent bool            `json:"was_idempotent"`
}

type ReleaseFundsRequest struct {
	UserID        int64            `json:"user_id"`
	LockID        string           `json:"lock_id"`
	AuditInfo     models.AuditInfo `json:"audit_info"`
}

type ReleaseFundsResult struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type ExecuteLockRequest struct {
	UserID         int64                 `json:"user_id"`
	LockID         string                `json:"lock_id"`
	ActualAmount   decimal.Decimal       `json:"actual_amount"`
	TransactionType string               `json:"transaction_type"`
	Reference      models.Reference      `json:"reference"`
	IdempotencyKey string                `json:"idempotency_key"`
	Metadata       map[string]interface{} `json:"metadata"`
	AuditInfo      models.AuditInfo      `json:"audit_info"`
}

type ExecuteLockResult struct {
	Transaction   *models.Transaction `json:"transaction"`
	Success       bool                `json:"success"`
	ErrorMessage  string              `json:"error_message,omitempty"`
	WasIdempotent bool                `json:"was_idempotent"`
}

type ReverseTransactionRequest struct {
	TransactionID string           `json:"transaction_id"`
	Reason        string           `json:"reason"`
	ReversedBy    string           `json:"reversed_by"`
	AuditInfo     models.AuditInfo `json:"audit_info"`
}

type ReverseTransactionResult struct {
	ReversalTransaction *models.Transaction `json:"reversal_transaction"`
	Success             bool                `json:"success"`
	ErrorMessage        string              `json:"error_message,omitempty"`
}

type TransactionStatusResult struct {
	Transaction  *models.Transaction `json:"transaction"`
	Success      bool                `json:"success"`
	ErrorMessage string              `json:"error_message,omitempty"`
}

const (
	lockTimeout = 30 * time.Second
)

func (e *transactionEngine) ProcessTransaction(ctx context.Context, req *TransactionRequest) (*TransactionResult, error) {
	// Check idempotency first
	if req.IdempotencyKey != "" {
		if existing, exists, err := e.idempotencyRepo.GetIdempotencyResponse(ctx, req.IdempotencyKey); err == nil && exists {
			// Return cached result
			if result, ok := existing.(*TransactionResult); ok {
				result.WasIdempotent = true
				return result, nil
			}
		}
	}

	// Acquire user lock for the transaction
	userLock, err := e.lockManager.LockUser(ctx, req.UserID, "transaction", lockTimeout)
	if err != nil {
		return &TransactionResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to acquire user lock: %v", err),
		}, nil
	}
	defer e.lockManager.ReleaseLock(ctx, userLock)

	// Get user's wallet
	wallet, err := e.walletRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return &TransactionResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
		}, nil
	}

	// Validate wallet status
	if !wallet.IsActive() {
		return &TransactionResult{
			Success:      false,
			ErrorMessage: "Wallet is not active",
		}, nil
	}

	// Create transaction request
	transactionReq := &models.TransactionRequest{
		WalletID:       wallet.ID,
		UserID:         req.UserID,
		Type:           req.Type,
		Amount:         req.Amount,
		Fee:            req.Fee,
		Currency:       req.Currency,
		Reference:      req.Reference,
		IdempotencyKey: req.IdempotencyKey,
		Metadata:       req.Metadata,
	}

	// Start MongoDB transaction
	var result *TransactionResult
	err = e.withMongoTransaction(ctx, func(sc mongo.SessionContext) error {
		// Create the transaction record
		transaction := models.NewTransaction(transactionReq)
		transaction.SetAuditInfo(req.AuditInfo)

		// Validate transaction
		if err := transaction.Validate(); err != nil {
			result = &TransactionResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Transaction validation failed: %v", err),
			}
			return nil
		}

		// Check business rules based on transaction type
		if err := e.validateTransactionRules(ctx, wallet, transaction); err != nil {
			result = &TransactionResult{
				Success:      false,
				ErrorMessage: err.Error(),
			}
			return nil
		}

		// Mark transaction as processing
		transaction.MarkProcessing()

		// Save transaction
		if err := e.transactionRepo.Create(sc, transaction); err != nil {
			result = &TransactionResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to create transaction: %v", err),
			}
			return nil
		}

		// Calculate balance snapshots
		balanceBefore := models.BalanceSnapshot{
			Before:          wallet.Balance.Total,
			AvailableBefore: wallet.Balance.Available,
			LockedBefore:    wallet.Balance.Locked,
		}

		// Update wallet balance
		wallet.UpdateBalance(transaction.Amount.Net, transaction.Type)

		balanceAfter := models.BalanceSnapshot{
			After:           wallet.Balance.Total,
			AvailableAfter:  wallet.Balance.Available,
			LockedAfter:     wallet.Balance.Locked,
		}

		// Save updated wallet
		if err := e.walletRepo.Update(sc, wallet); err != nil {
			result = &TransactionResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to update wallet: %v", err),
			}
			return nil
		}

		// Mark transaction as completed
		transaction.MarkCompleted(balanceBefore, balanceAfter)

		// Update transaction
		if err := e.transactionRepo.Update(sc, transaction); err != nil {
			result = &TransactionResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to update transaction: %v", err),
			}
			return nil
		}

		result = &TransactionResult{
			Transaction: transaction,
			Wallet:      wallet,
			Success:     true,
		}

		return nil
	})

	if err != nil {
		return &TransactionResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Transaction failed: %v", err),
		}, nil
	}

	// Cache result for idempotency
	if req.IdempotencyKey != "" && result.Success {
		e.idempotencyRepo.SetIdempotencyKey(ctx, req.IdempotencyKey, result, 24*time.Hour)
	}

	return result, nil
}

func (e *transactionEngine) LockFunds(ctx context.Context, req *LockFundsRequest) (*LockFundsResult, error) {
	// Check idempotency
	if req.IdempotencyKey != "" {
		if existing, exists, err := e.idempotencyRepo.GetIdempotencyResponse(ctx, req.IdempotencyKey); err == nil && exists {
			if result, ok := existing.(*LockFundsResult); ok {
				result.WasIdempotent = true
				return result, nil
			}
		}
	}

	// Acquire user lock
	userLock, err := e.lockManager.LockUser(ctx, req.UserID, "lock_funds", lockTimeout)
	if err != nil {
		return &LockFundsResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to acquire user lock: %v", err),
		}, nil
	}
	defer e.lockManager.ReleaseLock(ctx, userLock)

	// Get wallet
	wallet, err := e.walletRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return &LockFundsResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
		}, nil
	}

	// Validate wallet status
	if !wallet.IsActive() {
		return &LockFundsResult{
			Success:      false,
			ErrorMessage: "Wallet is not active",
		}, nil
	}

	// Check sufficient balance
	if !wallet.HasSufficientBalance(req.Amount) {
		return &LockFundsResult{
			Success:      false,
			ErrorMessage: "Insufficient balance",
		}, nil
	}

	// Create lock
	lockID := fmt.Sprintf("LOCK-%d-%s", time.Now().Unix(), req.OrderID)
	lock := models.FundsLock{
		LockID:    lockID,
		OrderID:   req.OrderID,
		Amount:    req.Amount,
		LockedAt:  time.Now(),
		ExpiresAt: time.Now().Add(req.ExpirationTime),
		Status:    "active",
		Reason:    req.Reason,
	}

	// Add lock to wallet
	if err := wallet.AddLock(lock); err != nil {
		return &LockFundsResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Update wallet
	if err := e.walletRepo.Update(ctx, wallet); err != nil {
		return &LockFundsResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to update wallet: %v", err),
		}, nil
	}

	result := &LockFundsResult{
		LockID:  lockID,
		Success: true,
	}

	// Cache for idempotency
	if req.IdempotencyKey != "" {
		e.idempotencyRepo.SetIdempotencyKey(ctx, req.IdempotencyKey, result, 24*time.Hour)
	}

	return result, nil
}

func (e *transactionEngine) ReleaseFunds(ctx context.Context, req *ReleaseFundsRequest) (*ReleaseFundsResult, error) {
	// Acquire user lock
	userLock, err := e.lockManager.LockUser(ctx, req.UserID, "release_funds", lockTimeout)
	if err != nil {
		return &ReleaseFundsResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to acquire user lock: %v", err),
		}, nil
	}
	defer e.lockManager.ReleaseLock(ctx, userLock)

	// Get wallet
	wallet, err := e.walletRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return &ReleaseFundsResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
		}, nil
	}

	// Release the lock
	if err := wallet.ReleaseLock(req.LockID); err != nil {
		return &ReleaseFundsResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Update wallet
	if err := e.walletRepo.Update(ctx, wallet); err != nil {
		return &ReleaseFundsResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to update wallet: %v", err),
		}, nil
	}

	return &ReleaseFundsResult{
		Success: true,
	}, nil
}

func (e *transactionEngine) ExecuteLock(ctx context.Context, req *ExecuteLockRequest) (*ExecuteLockResult, error) {
	// Check idempotency
	if req.IdempotencyKey != "" {
		if existing, exists, err := e.idempotencyRepo.GetIdempotencyResponse(ctx, req.IdempotencyKey); err == nil && exists {
			if result, ok := existing.(*ExecuteLockResult); ok {
				result.WasIdempotent = true
				return result, nil
			}
		}
	}

	// Acquire user lock
	userLock, err := e.lockManager.LockUser(ctx, req.UserID, "execute_lock", lockTimeout)
	if err != nil {
		return &ExecuteLockResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to acquire user lock: %v", err),
		}, nil
	}
	defer e.lockManager.ReleaseLock(ctx, userLock)

	var result *ExecuteLockResult
	err = e.withMongoTransaction(ctx, func(sc mongo.SessionContext) error {
		// Get wallet
		wallet, err := e.walletRepo.GetByUserID(sc, req.UserID)
		if err != nil {
			result = &ExecuteLockResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
			}
			return nil
		}

		// Execute the lock
		if err := wallet.ExecuteLock(req.LockID, req.ActualAmount); err != nil {
			result = &ExecuteLockResult{
				Success:      false,
				ErrorMessage: err.Error(),
			}
			return nil
		}

		// Create transaction for the execution
		transactionReq := &models.TransactionRequest{
			WalletID:       wallet.ID,
			UserID:         req.UserID,
			Type:           req.TransactionType,
			Amount:         req.ActualAmount.Neg(), // Negative for debit
			Fee:            decimal.Zero,
			Currency:       wallet.Balance.Currency,
			Reference:      req.Reference,
			IdempotencyKey: req.IdempotencyKey,
			Metadata:       req.Metadata,
		}

		transaction := models.NewTransaction(transactionReq)
		transaction.SetAuditInfo(req.AuditInfo)
		transaction.MarkCompleted(models.BalanceSnapshot{}, models.BalanceSnapshot{})

		// Save transaction
		if err := e.transactionRepo.Create(sc, transaction); err != nil {
			result = &ExecuteLockResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to create transaction: %v", err),
			}
			return nil
		}

		// Update wallet
		if err := e.walletRepo.Update(sc, wallet); err != nil {
			result = &ExecuteLockResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to update wallet: %v", err),
			}
			return nil
		}

		result = &ExecuteLockResult{
			Transaction: transaction,
			Success:     true,
		}

		return nil
	})

	if err != nil {
		return &ExecuteLockResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Lock execution failed: %v", err),
		}, nil
	}

	// Cache for idempotency
	if req.IdempotencyKey != "" && result.Success {
		e.idempotencyRepo.SetIdempotencyKey(ctx, req.IdempotencyKey, result, 24*time.Hour)
	}

	return result, nil
}

func (e *transactionEngine) ReverseTransaction(ctx context.Context, req *ReverseTransactionRequest) (*ReverseTransactionResult, error) {
	// Get original transaction
	originalTx, err := e.transactionRepo.GetByTransactionID(ctx, req.TransactionID)
	if err != nil {
		return &ReverseTransactionResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get original transaction: %v", err),
		}, nil
	}

	// Check if transaction can be reversed
	if !originalTx.CanBeReversed() {
		return &ReverseTransactionResult{
			Success:      false,
			ErrorMessage: "Transaction cannot be reversed",
		}, nil
	}

	// Acquire user lock
	userLock, err := e.lockManager.LockUser(ctx, originalTx.UserID, "reverse_transaction", lockTimeout)
	if err != nil {
		return &ReverseTransactionResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to acquire user lock: %v", err),
		}, nil
	}
	defer e.lockManager.ReleaseLock(ctx, userLock)

	var result *ReverseTransactionResult
	err = e.withMongoTransaction(ctx, func(sc mongo.SessionContext) error {
		// Create reversal transaction
		reversalTx := originalTx.Reverse(req.Reason, req.ReversedBy)
		if reversalTx == nil {
			result = &ReverseTransactionResult{
				Success:      false,
				ErrorMessage: "Failed to create reversal transaction",
			}
			return nil
		}

		reversalTx.SetAuditInfo(req.AuditInfo)

		// Get wallet
		wallet, err := e.walletRepo.GetByID(sc, originalTx.WalletID)
		if err != nil {
			result = &ReverseTransactionResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
			}
			return nil
		}

		// Update wallet balance
		wallet.UpdateBalance(reversalTx.Amount.Net, reversalTx.Type)

		// Save reversal transaction
		if err := e.transactionRepo.Create(sc, reversalTx); err != nil {
			result = &ReverseTransactionResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to create reversal transaction: %v", err),
			}
			return nil
		}

		// Update original transaction
		if err := e.transactionRepo.Update(sc, originalTx); err != nil {
			result = &ReverseTransactionResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to update original transaction: %v", err),
			}
			return nil
		}

		// Update wallet
		if err := e.walletRepo.Update(sc, wallet); err != nil {
			result = &ReverseTransactionResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to update wallet: %v", err),
			}
			return nil
		}

		result = &ReverseTransactionResult{
			ReversalTransaction: reversalTx,
			Success:             true,
		}

		return nil
	})

	if err != nil {
		return &ReverseTransactionResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Transaction reversal failed: %v", err),
		}, nil
	}

	return result, nil
}

func (e *transactionEngine) GetTransactionStatus(ctx context.Context, transactionID string) (*TransactionStatusResult, error) {
	transaction, err := e.transactionRepo.GetByTransactionID(ctx, transactionID)
	if err != nil {
		return &TransactionStatusResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get transaction: %v", err),
		}, nil
	}

	return &TransactionStatusResult{
		Transaction: transaction,
		Success:     true,
	}, nil
}

func (e *transactionEngine) validateTransactionRules(ctx context.Context, wallet *models.Wallet, transaction *models.Transaction) error {
	switch transaction.Type {
	case "deposit":
		return wallet.CanDeposit(transaction.GetAbsoluteAmount())
	case "withdrawal":
		return wallet.CanWithdraw(transaction.GetAbsoluteAmount())
	default:
		return nil // Other transaction types have different validation rules
	}
}

func (e *transactionEngine) withMongoTransaction(ctx context.Context, fn func(mongo.SessionContext) error) error {
	session, err := e.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		return nil, fn(sc)
	})

	return err
}