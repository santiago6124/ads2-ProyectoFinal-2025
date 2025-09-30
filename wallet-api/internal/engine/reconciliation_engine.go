package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"wallet-api/internal/models"
	"wallet-api/internal/repository"
)

type ReconciliationEngine interface {
	ReconcileWallet(ctx context.Context, walletID primitive.ObjectID) (*ReconciliationResult, error)
	ReconcileAllWallets(ctx context.Context, batchSize int) (*BatchReconciliationResult, error)
	VerifyWalletIntegrity(ctx context.Context, walletID primitive.ObjectID) (*IntegrityResult, error)
	GenerateBalanceChecksum(wallet *models.Wallet) string
	DetectBalanceDiscrepancies(ctx context.Context, walletID primitive.ObjectID) (*DiscrepancyResult, error)
}

type reconciliationEngine struct {
	walletRepo      repository.WalletRepository
	transactionRepo repository.TransactionRepository
	lockManager     *repository.WalletLockManager
}

func NewReconciliationEngine(
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	lockManager *repository.WalletLockManager,
) ReconciliationEngine {
	return &reconciliationEngine{
		walletRepo:      walletRepo,
		transactionRepo: transactionRepo,
		lockManager:     lockManager,
	}
}

type ReconciliationResult struct {
	WalletID                 primitive.ObjectID `json:"wallet_id"`
	BalanceDiscrepancy       decimal.Decimal    `json:"balance_discrepancy"`
	CalculatedBalance        decimal.Decimal    `json:"calculated_balance"`
	StoredBalance            decimal.Decimal    `json:"stored_balance"`
	TransactionCount         int64              `json:"transaction_count"`
	LastTransactionProcessed time.Time          `json:"last_transaction_processed"`
	ReconciliationTime       time.Time          `json:"reconciliation_time"`
	Status                   string             `json:"status"` // "success", "discrepancy_found", "error"
	ErrorMessage             string             `json:"error_message,omitempty"`
	BalanceAdjustment        *models.Transaction `json:"balance_adjustment,omitempty"`
}

type BatchReconciliationResult struct {
	TotalWallets            int                    `json:"total_wallets"`
	ReconciledWallets       int                    `json:"reconciled_wallets"`
	DiscrepanciesFound      int                    `json:"discrepancies_found"`
	ErrorsEncountered       int                    `json:"errors_encountered"`
	Results                 []*ReconciliationResult `json:"results"`
	BatchStartTime          time.Time              `json:"batch_start_time"`
	BatchEndTime            time.Time              `json:"batch_end_time"`
	TotalProcessingTime     time.Duration          `json:"total_processing_time"`
}

type IntegrityResult struct {
	WalletID            primitive.ObjectID `json:"wallet_id"`
	ChecksumMatch       bool               `json:"checksum_match"`
	ExpectedChecksum    string             `json:"expected_checksum"`
	ActualChecksum      string             `json:"actual_checksum"`
	TransactionCount    int64              `json:"transaction_count"`
	LastVerified        time.Time          `json:"last_verified"`
	IntegrityStatus     string             `json:"integrity_status"`
	RecommendedAction   string             `json:"recommended_action"`
}

type DiscrepancyResult struct {
	WalletID              primitive.ObjectID  `json:"wallet_id"`
	DiscrepanciesFound    bool                `json:"discrepancies_found"`
	AvailableDiscrepancy  decimal.Decimal     `json:"available_discrepancy"`
	LockedDiscrepancy     decimal.Decimal     `json:"locked_discrepancy"`
	TotalDiscrepancy      decimal.Decimal     `json:"total_discrepancy"`
	SuspiciousTransactions []*models.Transaction `json:"suspicious_transactions"`
	DetectedAt            time.Time           `json:"detected_at"`
}

const (
	reconciliationTimeout = 60 * time.Second
	maxDiscrepancyThreshold = "0.01" // Maximum allowed discrepancy in USD
)

func (e *reconciliationEngine) ReconcileWallet(ctx context.Context, walletID primitive.ObjectID) (*ReconciliationResult, error) {
	// Acquire wallet lock for reconciliation
	walletLock, err := e.lockManager.LockWallet(ctx, walletID.Hex(), "reconciliation", reconciliationTimeout)
	if err != nil {
		return &ReconciliationResult{
			WalletID:     walletID,
			Status:       "error",
			ErrorMessage: fmt.Sprintf("Failed to acquire wallet lock: %v", err),
		}, nil
	}
	defer e.lockManager.ReleaseLock(ctx, walletLock)

	startTime := time.Now()

	// Get wallet
	wallet, err := e.walletRepo.GetByID(ctx, walletID)
	if err != nil {
		return &ReconciliationResult{
			WalletID:     walletID,
			Status:       "error",
			ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
		}, nil
	}

	// Get all completed transactions for this wallet
	transactions, err := e.transactionRepo.GetByWalletID(ctx, walletID, 10000, 0) // Large limit for all transactions
	if err != nil {
		return &ReconciliationResult{
			WalletID:     walletID,
			Status:       "error",
			ErrorMessage: fmt.Sprintf("Failed to get transactions: %v", err),
		}, nil
	}

	// Calculate balance from transactions
	calculatedBalance := e.calculateBalanceFromTransactions(transactions)

	// Compare with stored balance
	storedBalance := wallet.Balance.Available.Add(wallet.Balance.Locked)
	discrepancy := calculatedBalance.Sub(storedBalance)

	result := &ReconciliationResult{
		WalletID:                 walletID,
		CalculatedBalance:        calculatedBalance,
		StoredBalance:            storedBalance,
		BalanceDiscrepancy:       discrepancy,
		TransactionCount:         int64(len(transactions)),
		ReconciliationTime:       startTime,
	}

	// Set last transaction processed
	if len(transactions) > 0 {
		result.LastTransactionProcessed = transactions[0].CreatedAt
	}

	// Check if discrepancy is within acceptable threshold
	threshold, _ := decimal.NewFromString(maxDiscrepancyThreshold)
	if discrepancy.Abs().LessThanOrEqual(threshold) {
		result.Status = "success"

		// Update wallet verification info
		verification := wallet.Verification
		verification.LastReconciled = startTime
		verification.TransactionCount = int64(len(transactions))
		verification.BalanceHash = e.GenerateBalanceChecksum(wallet)
		verification.Checksum = e.generateWalletChecksum(wallet, transactions)

		if err := e.walletRepo.UpdateVerificationInfo(ctx, walletID, verification); err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to update verification info: %v", err)
		}
	} else {
		result.Status = "discrepancy_found"

		// Create balance adjustment transaction if discrepancy is significant
		adjustment, err := e.createBalanceAdjustment(ctx, wallet, discrepancy, "Reconciliation adjustment")
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to create balance adjustment: %v", err)
		} else {
			result.BalanceAdjustment = adjustment
		}
	}

	return result, nil
}

func (e *reconciliationEngine) ReconcileAllWallets(ctx context.Context, batchSize int) (*BatchReconciliationResult, error) {
	startTime := time.Now()

	result := &BatchReconciliationResult{
		BatchStartTime: startTime,
		Results:        make([]*ReconciliationResult, 0),
	}

	// Get wallets that need reconciliation
	wallets, err := e.walletRepo.GetWalletsForReconciliation(ctx, batchSize)
	if err != nil {
		return result, fmt.Errorf("failed to get wallets for reconciliation: %w", err)
	}

	result.TotalWallets = len(wallets)

	// Process each wallet
	for _, wallet := range wallets {
		walletResult, err := e.ReconcileWallet(ctx, wallet.ID)
		if err != nil {
			result.ErrorsEncountered++
			continue
		}

		result.Results = append(result.Results, walletResult)

		switch walletResult.Status {
		case "success":
			result.ReconciledWallets++
		case "discrepancy_found":
			result.DiscrepanciesFound++
		case "error":
			result.ErrorsEncountered++
		}
	}

	result.BatchEndTime = time.Now()
	result.TotalProcessingTime = result.BatchEndTime.Sub(result.BatchStartTime)

	return result, nil
}

func (e *reconciliationEngine) VerifyWalletIntegrity(ctx context.Context, walletID primitive.ObjectID) (*IntegrityResult, error) {
	// Get wallet
	wallet, err := e.walletRepo.GetByID(ctx, walletID)
	if err != nil {
		return &IntegrityResult{
			WalletID:        walletID,
			IntegrityStatus: "error",
			RecommendedAction: "Unable to retrieve wallet",
		}, nil
	}

	// Get transactions
	transactions, err := e.transactionRepo.GetByWalletID(ctx, walletID, 10000, 0)
	if err != nil {
		return &IntegrityResult{
			WalletID:        walletID,
			IntegrityStatus: "error",
			RecommendedAction: "Unable to retrieve transactions",
		}, nil
	}

	// Generate expected checksum
	expectedChecksum := e.generateWalletChecksum(wallet, transactions)
	actualChecksum := wallet.Verification.Checksum

	result := &IntegrityResult{
		WalletID:         walletID,
		ExpectedChecksum: expectedChecksum,
		ActualChecksum:   actualChecksum,
		ChecksumMatch:    expectedChecksum == actualChecksum,
		TransactionCount: int64(len(transactions)),
		LastVerified:     time.Now(),
	}

	if result.ChecksumMatch {
		result.IntegrityStatus = "verified"
		result.RecommendedAction = "none"
	} else {
		result.IntegrityStatus = "compromised"
		result.RecommendedAction = "immediate_reconciliation"
	}

	return result, nil
}

func (e *reconciliationEngine) GenerateBalanceChecksum(wallet *models.Wallet) string {
	data := fmt.Sprintf("%s:%s:%s:%s",
		wallet.Balance.Available.String(),
		wallet.Balance.Locked.String(),
		wallet.Balance.Total.String(),
		wallet.Balance.Currency,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (e *reconciliationEngine) DetectBalanceDiscrepancies(ctx context.Context, walletID primitive.ObjectID) (*DiscrepancyResult, error) {
	// Get wallet
	wallet, err := e.walletRepo.GetByID(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	// Get recent transactions (last 30 days)
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	transactions, err := e.transactionRepo.GetTransactionsByDateRange(ctx, walletID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Calculate expected balance from transactions
	calculatedBalance := e.calculateBalanceFromTransactions(transactions)
	storedBalance := wallet.Balance.Available.Add(wallet.Balance.Locked)

	availableDiscrepancy := decimal.Zero
	lockedDiscrepancy := decimal.Zero
	totalDiscrepancy := calculatedBalance.Sub(storedBalance)

	// Detect suspicious transactions
	var suspiciousTransactions []*models.Transaction
	for _, tx := range transactions {
		if e.isTransactionSuspicious(tx) {
			suspiciousTransactions = append(suspiciousTransactions, tx)
		}
	}

	result := &DiscrepancyResult{
		WalletID:               walletID,
		DiscrepanciesFound:     !totalDiscrepancy.IsZero(),
		AvailableDiscrepancy:   availableDiscrepancy,
		LockedDiscrepancy:      lockedDiscrepancy,
		TotalDiscrepancy:       totalDiscrepancy,
		SuspiciousTransactions: suspiciousTransactions,
		DetectedAt:             time.Now(),
	}

	return result, nil
}

func (e *reconciliationEngine) calculateBalanceFromTransactions(transactions []*models.Transaction) decimal.Decimal {
	balance := decimal.Zero

	for _, tx := range transactions {
		if tx.Status == "completed" {
			balance = balance.Add(tx.Amount.Net)
		}
	}

	return balance
}

func (e *reconciliationEngine) generateWalletChecksum(wallet *models.Wallet, transactions []*models.Transaction) string {
	// Create a deterministic checksum based on wallet state and transactions
	data := fmt.Sprintf("wallet:%s:balance:%s:txcount:%d:updated:%d",
		wallet.ID.Hex(),
		e.GenerateBalanceChecksum(wallet),
		len(transactions),
		wallet.UpdatedAt.Unix(),
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (e *reconciliationEngine) createBalanceAdjustment(ctx context.Context, wallet *models.Wallet, discrepancy decimal.Decimal, reason string) (*models.Transaction, error) {
	// Create adjustment transaction
	req := &models.TransactionRequest{
		WalletID: wallet.ID,
		UserID:   wallet.UserID,
		Type:     "adjustment",
		Amount:   discrepancy,
		Fee:      decimal.Zero,
		Currency: wallet.Balance.Currency,
		Reference: models.Reference{
			Type:        "system",
			ID:          "reconciliation",
			Description: reason,
			Metadata: map[string]interface{}{
				"reconciliation_type": "balance_adjustment",
				"original_balance":    wallet.Balance.Total.String(),
				"discrepancy":         discrepancy.String(),
			},
		},
		IdempotencyKey: fmt.Sprintf("adjustment-%s-%d", wallet.ID.Hex(), time.Now().Unix()),
	}

	adjustment := models.NewTransaction(req)
	adjustment.MarkCompleted(models.BalanceSnapshot{}, models.BalanceSnapshot{})

	if err := e.transactionRepo.Create(ctx, adjustment); err != nil {
		return nil, fmt.Errorf("failed to create adjustment transaction: %w", err)
	}

	return adjustment, nil
}

func (e *reconciliationEngine) isTransactionSuspicious(tx *models.Transaction) bool {
	// Define criteria for suspicious transactions

	// Large amounts
	threshold, _ := decimal.NewFromString("10000")
	if tx.Amount.Value.Abs().GreaterThan(threshold) {
		return true
	}

	// Multiple failed attempts
	if len(tx.Processing.Errors) > 3 {
		return true
	}

	// Very old pending transactions
	if tx.Status == "pending" && time.Since(tx.CreatedAt) > 24*time.Hour {
		return true
	}

	// Unusual processing times
	if tx.Processing.ProcessingTime > 60000 { // More than 1 minute
		return true
	}

	return false
}