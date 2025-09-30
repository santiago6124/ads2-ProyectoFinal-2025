package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"wallet-api/internal/config"
	"wallet-api/internal/engine"
	"wallet-api/internal/models"
	"wallet-api/internal/repository"
)

type WalletService interface {
	CreateWallet(ctx context.Context, req *CreateWalletRequest) (*CreateWalletResponse, error)
	GetWallet(ctx context.Context, userID int64) (*GetWalletResponse, error)
	GetBalance(ctx context.Context, userID int64) (*GetBalanceResponse, error)
	Deposit(ctx context.Context, req *DepositRequest) (*DepositResponse, error)
	Withdraw(ctx context.Context, req *WithdrawRequest) (*WithdrawResponse, error)
	LockFunds(ctx context.Context, req *LockFundsRequest) (*LockFundsResponse, error)
	ReleaseFunds(ctx context.Context, req *ReleaseFundsRequest) (*ReleaseFundsResponse, error)
	ExecuteLock(ctx context.Context, req *ExecuteLockRequest) (*ExecuteLockResponse, error)
	GetTransactionHistory(ctx context.Context, req *GetTransactionHistoryRequest) (*GetTransactionHistoryResponse, error)
	GetTransaction(ctx context.Context, req *GetTransactionRequest) (*GetTransactionResponse, error)
	ReverseTransaction(ctx context.Context, req *ReverseTransactionRequest) (*ReverseTransactionResponse, error)
	SuspendWallet(ctx context.Context, userID int64, reason string) error
	ReactivateWallet(ctx context.Context, userID int64) error
	GetWalletStats(ctx context.Context, userID int64) (*WalletStatsResponse, error)
}

type walletService struct {
	walletRepo         repository.WalletRepository
	transactionRepo    repository.TransactionRepository
	transactionEngine  engine.TransactionEngine
	reconciliationEngine engine.ReconciliationEngine
	idempotencyManager engine.IdempotencyManager
	config             *config.Config
}

func NewWalletService(
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	transactionEngine engine.TransactionEngine,
	reconciliationEngine engine.ReconciliationEngine,
	idempotencyManager engine.IdempotencyManager,
	config *config.Config,
) WalletService {
	return &walletService{
		walletRepo:           walletRepo,
		transactionRepo:      transactionRepo,
		transactionEngine:    transactionEngine,
		reconciliationEngine: reconciliationEngine,
		idempotencyManager:   idempotencyManager,
		config:               config,
	}
}

// Request/Response types
type CreateWalletRequest struct {
	UserID         int64           `json:"user_id"`
	InitialBalance decimal.Decimal `json:"initial_balance"`
	Currency       string          `json:"currency"`
	Limits         *models.Limits  `json:"limits,omitempty"`
}

type CreateWalletResponse struct {
	Wallet       *models.Wallet `json:"wallet"`
	Success      bool           `json:"success"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

type GetWalletResponse struct {
	Wallet       *models.Wallet `json:"wallet"`
	Success      bool           `json:"success"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

type GetBalanceResponse struct {
	Available    decimal.Decimal `json:"available"`
	Locked       decimal.Decimal `json:"locked"`
	Total        decimal.Decimal `json:"total"`
	Currency     string          `json:"currency"`
	Success      bool            `json:"success"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

type DepositRequest struct {
	UserID         int64                 `json:"user_id"`
	Amount         decimal.Decimal       `json:"amount"`
	Currency       string                `json:"currency"`
	Reference      models.Reference      `json:"reference"`
	IdempotencyKey string                `json:"idempotency_key"`
	Metadata       map[string]interface{} `json:"metadata"`
	AuditInfo      models.AuditInfo      `json:"audit_info"`
}

type DepositResponse struct {
	Transaction  *models.Transaction `json:"transaction"`
	NewBalance   decimal.Decimal     `json:"new_balance"`
	Success      bool                `json:"success"`
	ErrorMessage string              `json:"error_message,omitempty"`
}

type WithdrawRequest struct {
	UserID         int64                 `json:"user_id"`
	Amount         decimal.Decimal       `json:"amount"`
	Currency       string                `json:"currency"`
	Reference      models.Reference      `json:"reference"`
	IdempotencyKey string                `json:"idempotency_key"`
	Metadata       map[string]interface{} `json:"metadata"`
	AuditInfo      models.AuditInfo      `json:"audit_info"`
}

type WithdrawResponse struct {
	Transaction  *models.Transaction `json:"transaction"`
	NewBalance   decimal.Decimal     `json:"new_balance"`
	Success      bool                `json:"success"`
	ErrorMessage string              `json:"error_message,omitempty"`
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

type LockFundsResponse struct {
	LockID       string          `json:"lock_id"`
	ExpiresAt    time.Time       `json:"expires_at"`
	Success      bool            `json:"success"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

type ReleaseFundsRequest struct {
	UserID    int64            `json:"user_id"`
	LockID    string           `json:"lock_id"`
	AuditInfo models.AuditInfo `json:"audit_info"`
}

type ReleaseFundsResponse struct {
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

type ExecuteLockResponse struct {
	Transaction  *models.Transaction `json:"transaction"`
	Success      bool                `json:"success"`
	ErrorMessage string              `json:"error_message,omitempty"`
}

type GetTransactionHistoryRequest struct {
	UserID          int64     `json:"user_id"`
	Limit           int       `json:"limit"`
	Offset          int       `json:"offset"`
	TransactionType string    `json:"transaction_type,omitempty"`
	StartDate       time.Time `json:"start_date,omitempty"`
	EndDate         time.Time `json:"end_date,omitempty"`
}

type GetTransactionHistoryResponse struct {
	Transactions []*models.Transaction `json:"transactions"`
	Total        int64                 `json:"total"`
	Success      bool                  `json:"success"`
	ErrorMessage string                `json:"error_message,omitempty"`
}

type GetTransactionRequest struct {
	UserID        int64  `json:"user_id"`
	TransactionID string `json:"transaction_id"`
}

type GetTransactionResponse struct {
	Transaction  *models.Transaction `json:"transaction"`
	Success      bool                `json:"success"`
	ErrorMessage string              `json:"error_message,omitempty"`
}

type ReverseTransactionRequest struct {
	TransactionID string           `json:"transaction_id"`
	Reason        string           `json:"reason"`
	ReversedBy    string           `json:"reversed_by"`
	AuditInfo     models.AuditInfo `json:"audit_info"`
}

type ReverseTransactionResponse struct {
	ReversalTransaction *models.Transaction `json:"reversal_transaction"`
	Success             bool                `json:"success"`
	ErrorMessage        string              `json:"error_message,omitempty"`
}

type WalletStatsResponse struct {
	TotalDeposits     decimal.Decimal `json:"total_deposits"`
	TotalWithdrawals  decimal.Decimal `json:"total_withdrawals"`
	TotalFeesPaid     decimal.Decimal `json:"total_fees_paid"`
	TransactionCount  int64           `json:"transaction_count"`
	AccountAgeDays    int             `json:"account_age_days"`
	LastActivity      time.Time       `json:"last_activity"`
	Success           bool            `json:"success"`
	ErrorMessage      string          `json:"error_message,omitempty"`
}

func (s *walletService) CreateWallet(ctx context.Context, req *CreateWalletRequest) (*CreateWalletResponse, error) {
	// Check if wallet already exists for this user
	existingWallet, err := s.walletRepo.GetByUserID(ctx, req.UserID)
	if err == nil && existingWallet != nil {
		return &CreateWalletResponse{
			Success:      false,
			ErrorMessage: "Wallet already exists for this user",
		}, nil
	}

	// Use default limits if not provided
	limits := req.Limits
	if limits == nil {
		limits = &models.Limits{
			DailyWithdrawal:   decimal.NewFromFloat(s.config.Limits.DefaultDailyWithdrawal),
			DailyDeposit:      decimal.NewFromFloat(s.config.Limits.DefaultDailyDeposit),
			SingleTransaction: decimal.NewFromFloat(s.config.Limits.DefaultSingleTransaction),
			MonthlyVolume:     decimal.NewFromFloat(s.config.Limits.DefaultMonthlyVolume),
		}
	}

	// Create new wallet
	wallet := models.NewWallet(req.UserID, req.InitialBalance, *limits)
	if req.Currency != "" {
		wallet.Balance.Currency = req.Currency
	}

	// Validate wallet
	if err := wallet.Validate(); err != nil {
		return &CreateWalletResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Wallet validation failed: %v", err),
		}, nil
	}

	// Save wallet
	if err := s.walletRepo.Create(ctx, wallet); err != nil {
		return &CreateWalletResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to create wallet: %v", err),
		}, nil
	}

	return &CreateWalletResponse{
		Wallet:  wallet,
		Success: true,
	}, nil
}

func (s *walletService) GetWallet(ctx context.Context, userID int64) (*GetWalletResponse, error) {
	wallet, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return &GetWalletResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
		}, nil
	}

	return &GetWalletResponse{
		Wallet:  wallet,
		Success: true,
	}, nil
}

func (s *walletService) GetBalance(ctx context.Context, userID int64) (*GetBalanceResponse, error) {
	wallet, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return &GetBalanceResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
		}, nil
	}

	return &GetBalanceResponse{
		Available: wallet.Balance.Available,
		Locked:    wallet.Balance.Locked,
		Total:     wallet.Balance.Total,
		Currency:  wallet.Balance.Currency,
		Success:   true,
	}, nil
}

func (s *walletService) Deposit(ctx context.Context, req *DepositRequest) (*DepositResponse, error) {
	// Validate amount
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return &DepositResponse{
			Success:      false,
			ErrorMessage: "Deposit amount must be positive",
		}, nil
	}

	// Create transaction request
	txReq := &engine.TransactionRequest{
		UserID:         req.UserID,
		Type:           "deposit",
		Amount:         req.Amount,
		Fee:            decimal.Zero,
		Currency:       req.Currency,
		Reference:      req.Reference,
		IdempotencyKey: req.IdempotencyKey,
		Metadata:       req.Metadata,
		AuditInfo:      req.AuditInfo,
	}

	// Process transaction
	result, err := s.transactionEngine.ProcessTransaction(ctx, txReq)
	if err != nil {
		return &DepositResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to process deposit: %v", err),
		}, nil
	}

	if !result.Success {
		return &DepositResponse{
			Success:      false,
			ErrorMessage: result.ErrorMessage,
		}, nil
	}

	return &DepositResponse{
		Transaction: result.Transaction,
		NewBalance:  result.Wallet.Balance.Total,
		Success:     true,
	}, nil
}

func (s *walletService) Withdraw(ctx context.Context, req *WithdrawRequest) (*WithdrawResponse, error) {
	// Validate amount
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return &WithdrawResponse{
			Success:      false,
			ErrorMessage: "Withdrawal amount must be positive",
		}, nil
	}

	// Create transaction request (negative amount for withdrawal)
	txReq := &engine.TransactionRequest{
		UserID:         req.UserID,
		Type:           "withdrawal",
		Amount:         req.Amount.Neg(),
		Fee:            decimal.Zero,
		Currency:       req.Currency,
		Reference:      req.Reference,
		IdempotencyKey: req.IdempotencyKey,
		Metadata:       req.Metadata,
		AuditInfo:      req.AuditInfo,
	}

	// Process transaction
	result, err := s.transactionEngine.ProcessTransaction(ctx, txReq)
	if err != nil {
		return &WithdrawResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to process withdrawal: %v", err),
		}, nil
	}

	if !result.Success {
		return &WithdrawResponse{
			Success:      false,
			ErrorMessage: result.ErrorMessage,
		}, nil
	}

	return &WithdrawResponse{
		Transaction: result.Transaction,
		NewBalance:  result.Wallet.Balance.Total,
		Success:     true,
	}, nil
}

func (s *walletService) LockFunds(ctx context.Context, req *LockFundsRequest) (*LockFundsResponse, error) {
	// Validate amount
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return &LockFundsResponse{
			Success:      false,
			ErrorMessage: "Lock amount must be positive",
		}, nil
	}

	// Use default expiration if not provided
	expiration := req.ExpirationTime
	if expiration == 0 {
		expiration = s.config.Limits.LockDuration
	}

	// Create lock request
	lockReq := &engine.LockFundsRequest{
		UserID:         req.UserID,
		Amount:         req.Amount,
		OrderID:        req.OrderID,
		Reason:         req.Reason,
		ExpirationTime: expiration,
		IdempotencyKey: req.IdempotencyKey,
		AuditInfo:      req.AuditInfo,
	}

	// Process lock
	result, err := s.transactionEngine.LockFunds(ctx, lockReq)
	if err != nil {
		return &LockFundsResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to lock funds: %v", err),
		}, nil
	}

	if !result.Success {
		return &LockFundsResponse{
			Success:      false,
			ErrorMessage: result.ErrorMessage,
		}, nil
	}

	return &LockFundsResponse{
		LockID:    result.LockID,
		ExpiresAt: time.Now().Add(expiration),
		Success:   true,
	}, nil
}

func (s *walletService) ReleaseFunds(ctx context.Context, req *ReleaseFundsRequest) (*ReleaseFundsResponse, error) {
	// Create release request
	releaseReq := &engine.ReleaseFundsRequest{
		UserID:    req.UserID,
		LockID:    req.LockID,
		AuditInfo: req.AuditInfo,
	}

	// Process release
	result, err := s.transactionEngine.ReleaseFunds(ctx, releaseReq)
	if err != nil {
		return &ReleaseFundsResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to release funds: %v", err),
		}, nil
	}

	if !result.Success {
		return &ReleaseFundsResponse{
			Success:      false,
			ErrorMessage: result.ErrorMessage,
		}, nil
	}

	return &ReleaseFundsResponse{
		Success: true,
	}, nil
}

func (s *walletService) ExecuteLock(ctx context.Context, req *ExecuteLockRequest) (*ExecuteLockResponse, error) {
	// Validate amount
	if req.ActualAmount.LessThanOrEqual(decimal.Zero) {
		return &ExecuteLockResponse{
			Success:      false,
			ErrorMessage: "Execution amount must be positive",
		}, nil
	}

	// Create execution request
	execReq := &engine.ExecuteLockRequest{
		UserID:          req.UserID,
		LockID:          req.LockID,
		ActualAmount:    req.ActualAmount,
		TransactionType: req.TransactionType,
		Reference:       req.Reference,
		IdempotencyKey:  req.IdempotencyKey,
		Metadata:        req.Metadata,
		AuditInfo:       req.AuditInfo,
	}

	// Process execution
	result, err := s.transactionEngine.ExecuteLock(ctx, execReq)
	if err != nil {
		return &ExecuteLockResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to execute lock: %v", err),
		}, nil
	}

	if !result.Success {
		return &ExecuteLockResponse{
			Success:      false,
			ErrorMessage: result.ErrorMessage,
		}, nil
	}

	return &ExecuteLockResponse{
		Transaction: result.Transaction,
		Success:     true,
	}, nil
}

func (s *walletService) GetTransactionHistory(ctx context.Context, req *GetTransactionHistoryRequest) (*GetTransactionHistoryResponse, error) {
	// Set default limits
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	var transactions []*models.Transaction
	var err error

	if req.TransactionType != "" {
		// Get wallet first
		wallet, err := s.walletRepo.GetByUserID(ctx, req.UserID)
		if err != nil {
			return &GetTransactionHistoryResponse{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
			}, nil
		}

		transactions, err = s.transactionRepo.GetTransactionsByType(ctx, wallet.ID, req.TransactionType, limit, offset)
	} else if !req.StartDate.IsZero() && !req.EndDate.IsZero() {
		// Get wallet first
		wallet, err := s.walletRepo.GetByUserID(ctx, req.UserID)
		if err != nil {
			return &GetTransactionHistoryResponse{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
			}, nil
		}

		transactions, err = s.transactionRepo.GetTransactionsByDateRange(ctx, wallet.ID, req.StartDate, req.EndDate)
	} else {
		transactions, err = s.transactionRepo.GetByUserID(ctx, req.UserID, limit, offset)
	}

	if err != nil {
		return &GetTransactionHistoryResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get transactions: %v", err),
		}, nil
	}

	return &GetTransactionHistoryResponse{
		Transactions: transactions,
		Total:        int64(len(transactions)),
		Success:      true,
	}, nil
}

func (s *walletService) GetTransaction(ctx context.Context, req *GetTransactionRequest) (*GetTransactionResponse, error) {
	transaction, err := s.transactionRepo.GetByTransactionID(ctx, req.TransactionID)
	if err != nil {
		return &GetTransactionResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get transaction: %v", err),
		}, nil
	}

	// Verify transaction belongs to the user
	if transaction.UserID != req.UserID {
		return &GetTransactionResponse{
			Success:      false,
			ErrorMessage: "Transaction not found",
		}, nil
	}

	return &GetTransactionResponse{
		Transaction: transaction,
		Success:     true,
	}, nil
}

func (s *walletService) ReverseTransaction(ctx context.Context, req *ReverseTransactionRequest) (*ReverseTransactionResponse, error) {
	// Create reversal request
	reverseReq := &engine.ReverseTransactionRequest{
		TransactionID: req.TransactionID,
		Reason:        req.Reason,
		ReversedBy:    req.ReversedBy,
		AuditInfo:     req.AuditInfo,
	}

	// Process reversal
	result, err := s.transactionEngine.ReverseTransaction(ctx, reverseReq)
	if err != nil {
		return &ReverseTransactionResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to reverse transaction: %v", err),
		}, nil
	}

	if !result.Success {
		return &ReverseTransactionResponse{
			Success:      false,
			ErrorMessage: result.ErrorMessage,
		}, nil
	}

	return &ReverseTransactionResponse{
		ReversalTransaction: result.ReversalTransaction,
		Success:             true,
	}, nil
}

func (s *walletService) SuspendWallet(ctx context.Context, userID int64, reason string) error {
	wallet, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get wallet: %w", err)
	}

	return s.walletRepo.SetWalletStatus(ctx, wallet.ID, "suspended")
}

func (s *walletService) ReactivateWallet(ctx context.Context, userID int64) error {
	wallet, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get wallet: %w", err)
	}

	return s.walletRepo.SetWalletStatus(ctx, wallet.ID, "active")
}

func (s *walletService) GetWalletStats(ctx context.Context, userID int64) (*WalletStatsResponse, error) {
	wallet, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return &WalletStatsResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
		}, nil
	}

	// Calculate account age
	accountAge := int(time.Since(wallet.CreatedAt).Hours() / 24)

	return &WalletStatsResponse{
		TotalDeposits:    wallet.Metadata.TotalDeposits,
		TotalWithdrawals: wallet.Metadata.TotalWithdrawals,
		TotalFeesPaid:    wallet.Metadata.TotalFeesPaid,
		TransactionCount: wallet.Verification.TransactionCount,
		AccountAgeDays:   accountAge,
		LastActivity:     wallet.LastActivity,
		Success:          true,
	}, nil
}