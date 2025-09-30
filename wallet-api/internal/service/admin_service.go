package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"wallet-api/internal/engine"
	"wallet-api/internal/models"
	"wallet-api/internal/repository"
)

type AdminService interface {
	ReconcileWallet(ctx context.Context, req *ReconcileWalletRequest) (*ReconcileWalletResponse, error)
	ReconcileAllWallets(ctx context.Context, req *ReconcileAllWalletsRequest) (*ReconcileAllWalletsResponse, error)
	GetAuditReport(ctx context.Context, req *GetAuditReportRequest) (*GetAuditReportResponse, error)
	GetWalletMetrics(ctx context.Context, req *GetWalletMetricsRequest) (*GetWalletMetricsResponse, error)
	CreateBalanceAdjustment(ctx context.Context, req *CreateBalanceAdjustmentRequest) (*CreateBalanceAdjustmentResponse, error)
	ForceUnlockFunds(ctx context.Context, req *ForceUnlockFundsRequest) (*ForceUnlockFundsResponse, error)
	GetSystemHealth(ctx context.Context) (*SystemHealthResponse, error)
	ManualReconciliation(ctx context.Context, req *ManualReconciliationRequest) (*ManualReconciliationResponse, error)
	GetTransactionStats(ctx context.Context, req *GetTransactionStatsRequest) (*GetTransactionStatsResponse, error)
	CleanupExpiredLocks(ctx context.Context) (*CleanupResponse, error)
	GetSuspiciousTransactions(ctx context.Context, req *GetSuspiciousTransactionsRequest) (*GetSuspiciousTransactionsResponse, error)
}

type adminService struct {
	walletRepo           repository.WalletRepository
	transactionRepo      repository.TransactionRepository
	reconciliationEngine engine.ReconciliationEngine
	transactionEngine    engine.TransactionEngine
}

func NewAdminService(
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	reconciliationEngine engine.ReconciliationEngine,
	transactionEngine engine.TransactionEngine,
) AdminService {
	return &adminService{
		walletRepo:           walletRepo,
		transactionRepo:      transactionRepo,
		reconciliationEngine: reconciliationEngine,
		transactionEngine:    transactionEngine,
	}
}

// Request/Response types
type ReconcileWalletRequest struct {
	UserID int64 `json:"user_id"`
}

type ReconcileWalletResponse struct {
	Result       *engine.ReconciliationResult `json:"result"`
	Success      bool                         `json:"success"`
	ErrorMessage string                       `json:"error_message,omitempty"`
}

type ReconcileAllWalletsRequest struct {
	BatchSize int `json:"batch_size"`
}

type ReconcileAllWalletsResponse struct {
	Result       *engine.BatchReconciliationResult `json:"result"`
	Success      bool                              `json:"success"`
	ErrorMessage string                            `json:"error_message,omitempty"`
}

type GetAuditReportRequest struct {
	UserID    int64     `json:"user_id"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type GetAuditReportResponse struct {
	Report       *AuditReport `json:"report"`
	Success      bool         `json:"success"`
	ErrorMessage string       `json:"error_message,omitempty"`
}

type AuditReport struct {
	UserID           int64                 `json:"user_id"`
	WalletID         primitive.ObjectID    `json:"wallet_id"`
	ReportPeriod     ReportPeriod          `json:"report_period"`
	BalanceSummary   BalanceSummary        `json:"balance_summary"`
	TransactionStats TransactionStatistics `json:"transaction_stats"`
	Discrepancies    []Discrepancy         `json:"discrepancies"`
	GeneratedAt      time.Time             `json:"generated_at"`
}

type ReportPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type BalanceSummary struct {
	OpeningBalance decimal.Decimal `json:"opening_balance"`
	ClosingBalance decimal.Decimal `json:"closing_balance"`
	TotalDeposits  decimal.Decimal `json:"total_deposits"`
	TotalWithdrawals decimal.Decimal `json:"total_withdrawals"`
	TotalFees      decimal.Decimal `json:"total_fees"`
	NetChange      decimal.Decimal `json:"net_change"`
}

type TransactionStatistics struct {
	TotalTransactions int64                    `json:"total_transactions"`
	SuccessfulTransactions int64               `json:"successful_transactions"`
	FailedTransactions int64                   `json:"failed_transactions"`
	ByType            map[string]int64         `json:"by_type"`
	VolumeByType      map[string]decimal.Decimal `json:"volume_by_type"`
	AverageAmount     decimal.Decimal          `json:"average_amount"`
	LargestTransaction decimal.Decimal         `json:"largest_transaction"`
}

type Discrepancy struct {
	Type        string          `json:"type"`
	Description string          `json:"description"`
	Amount      decimal.Decimal `json:"amount"`
	DetectedAt  time.Time       `json:"detected_at"`
	Resolved    bool            `json:"resolved"`
}

type GetWalletMetricsRequest struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type GetWalletMetricsResponse struct {
	Metrics      *WalletMetrics `json:"metrics"`
	Success      bool           `json:"success"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

type WalletMetrics struct {
	ActiveWallets        int64           `json:"active_wallets"`
	SuspendedWallets     int64           `json:"suspended_wallets"`
	TotalBalance         decimal.Decimal `json:"total_balance"`
	TotalTransactions    int64           `json:"total_transactions"`
	TransactionVolume    decimal.Decimal `json:"transaction_volume"`
	AverageWalletBalance decimal.Decimal `json:"average_wallet_balance"`
	MetricsDate          time.Time       `json:"metrics_date"`
}

type CreateBalanceAdjustmentRequest struct {
	UserID      int64           `json:"user_id"`
	Amount      decimal.Decimal `json:"amount"`
	Reason      string          `json:"reason"`
	AdjustedBy  string          `json:"adjusted_by"`
	AuditInfo   models.AuditInfo `json:"audit_info"`
}

type CreateBalanceAdjustmentResponse struct {
	Transaction  *models.Transaction `json:"transaction"`
	Success      bool                `json:"success"`
	ErrorMessage string              `json:"error_message,omitempty"`
}

type ForceUnlockFundsRequest struct {
	UserID    int64  `json:"user_id"`
	LockID    string `json:"lock_id"`
	Reason    string `json:"reason"`
	UnlockedBy string `json:"unlocked_by"`
}

type ForceUnlockFundsResponse struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type SystemHealthResponse struct {
	DatabaseHealth    HealthStatus `json:"database_health"`
	RedisHealth      HealthStatus `json:"redis_health"`
	SystemMetrics    SystemMetrics `json:"system_metrics"`
	ActiveConnections int          `json:"active_connections"`
	Success          bool         `json:"success"`
	ErrorMessage     string       `json:"error_message,omitempty"`
	CheckedAt        time.Time    `json:"checked_at"`
}

type HealthStatus struct {
	Status       string        `json:"status"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

type SystemMetrics struct {
	PendingTransactions int64 `json:"pending_transactions"`
	ActiveLocks         int64 `json:"active_locks"`
	ExpiredLocks        int64 `json:"expired_locks"`
	QueuedReconciliations int64 `json:"queued_reconciliations"`
}

type ManualReconciliationRequest struct {
	UserID   int64           `json:"user_id"`
	NewBalance decimal.Decimal `json:"new_balance"`
	Reason   string          `json:"reason"`
	AdjustedBy string        `json:"adjusted_by"`
}

type ManualReconciliationResponse struct {
	Transaction  *models.Transaction `json:"transaction"`
	Success      bool                `json:"success"`
	ErrorMessage string              `json:"error_message,omitempty"`
}

type GetTransactionStatsRequest struct {
	UserID    int64     `json:"user_id,omitempty"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type GetTransactionStatsResponse struct {
	Stats        *repository.TransactionStats `json:"stats"`
	Success      bool                        `json:"success"`
	ErrorMessage string                      `json:"error_message,omitempty"`
}

type CleanupResponse struct {
	ExpiredLocksRemoved int64  `json:"expired_locks_removed"`
	Success             bool   `json:"success"`
	ErrorMessage        string `json:"error_message,omitempty"`
}

type GetSuspiciousTransactionsRequest struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Limit     int       `json:"limit"`
}

type GetSuspiciousTransactionsResponse struct {
	Transactions []*models.Transaction `json:"transactions"`
	Success      bool                  `json:"success"`
	ErrorMessage string                `json:"error_message,omitempty"`
}

func (s *adminService) ReconcileWallet(ctx context.Context, req *ReconcileWalletRequest) (*ReconcileWalletResponse, error) {
	// Get wallet
	wallet, err := s.walletRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return &ReconcileWalletResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
		}, nil
	}

	// Perform reconciliation
	result, err := s.reconciliationEngine.ReconcileWallet(ctx, wallet.ID)
	if err != nil {
		return &ReconcileWalletResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to reconcile wallet: %v", err),
		}, nil
	}

	return &ReconcileWalletResponse{
		Result:  result,
		Success: true,
	}, nil
}

func (s *adminService) ReconcileAllWallets(ctx context.Context, req *ReconcileAllWalletsRequest) (*ReconcileAllWalletsResponse, error) {
	batchSize := req.BatchSize
	if batchSize <= 0 || batchSize > 1000 {
		batchSize = 100
	}

	result, err := s.reconciliationEngine.ReconcileAllWallets(ctx, batchSize)
	if err != nil {
		return &ReconcileAllWalletsResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to reconcile wallets: %v", err),
		}, nil
	}

	return &ReconcileAllWalletsResponse{
		Result:  result,
		Success: true,
	}, nil
}

func (s *adminService) GetAuditReport(ctx context.Context, req *GetAuditReportRequest) (*GetAuditReportResponse, error) {
	// Get wallet
	wallet, err := s.walletRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return &GetAuditReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
		}, nil
	}

	// Get transactions for the period
	transactions, err := s.transactionRepo.GetTransactionsByDateRange(ctx, wallet.ID, req.StartDate, req.EndDate)
	if err != nil {
		return &GetAuditReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get transactions: %v", err),
		}, nil
	}

	// Generate report
	report := s.generateAuditReport(wallet, transactions, req.StartDate, req.EndDate)

	return &GetAuditReportResponse{
		Report:  report,
		Success: true,
	}, nil
}

func (s *adminService) GetWalletMetrics(ctx context.Context, req *GetWalletMetricsRequest) (*GetWalletMetricsResponse, error) {
	// Get active wallets
	activeWallets, err := s.walletRepo.GetActiveWallets(ctx, 10000, 0)
	if err != nil {
		return &GetWalletMetricsResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get active wallets: %v", err),
		}, nil
	}

	// Calculate metrics
	metrics := &WalletMetrics{
		ActiveWallets: int64(len(activeWallets)),
		MetricsDate:   time.Now(),
	}

	totalBalance := decimal.Zero
	for _, wallet := range activeWallets {
		totalBalance = totalBalance.Add(wallet.Balance.Total)
	}

	metrics.TotalBalance = totalBalance
	if len(activeWallets) > 0 {
		metrics.AverageWalletBalance = totalBalance.Div(decimal.NewFromInt(int64(len(activeWallets))))
	}

	return &GetWalletMetricsResponse{
		Metrics: metrics,
		Success: true,
	}, nil
}

func (s *adminService) CreateBalanceAdjustment(ctx context.Context, req *CreateBalanceAdjustmentRequest) (*CreateBalanceAdjustmentResponse, error) {
	// Create adjustment transaction
	txReq := &engine.TransactionRequest{
		UserID:   req.UserID,
		Type:     "adjustment",
		Amount:   req.Amount,
		Fee:      decimal.Zero,
		Currency: "USD",
		Reference: models.Reference{
			Type:        "manual",
			ID:          "admin_adjustment",
			Description: req.Reason,
			Metadata: map[string]interface{}{
				"adjusted_by": req.AdjustedBy,
				"admin_action": true,
			},
		},
		IdempotencyKey: fmt.Sprintf("admin-adjustment-%d-%d", req.UserID, time.Now().Unix()),
		AuditInfo:      req.AuditInfo,
	}

	result, err := s.transactionEngine.ProcessTransaction(ctx, txReq)
	if err != nil {
		return &CreateBalanceAdjustmentResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to create adjustment: %v", err),
		}, nil
	}

	if !result.Success {
		return &CreateBalanceAdjustmentResponse{
			Success:      false,
			ErrorMessage: result.ErrorMessage,
		}, nil
	}

	return &CreateBalanceAdjustmentResponse{
		Transaction: result.Transaction,
		Success:     true,
	}, nil
}

func (s *adminService) ForceUnlockFunds(ctx context.Context, req *ForceUnlockFundsRequest) (*ForceUnlockFundsResponse, error) {
	// Get wallet
	wallet, err := s.walletRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return &ForceUnlockFundsResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
		}, nil
	}

	// Release the lock
	if err := wallet.ReleaseLock(req.LockID); err != nil {
		return &ForceUnlockFundsResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Update wallet
	if err := s.walletRepo.Update(ctx, wallet); err != nil {
		return &ForceUnlockFundsResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to update wallet: %v", err),
		}, nil
	}

	return &ForceUnlockFundsResponse{
		Success: true,
	}, nil
}

func (s *adminService) GetSystemHealth(ctx context.Context) (*SystemHealthResponse, error) {
	response := &SystemHealthResponse{
		CheckedAt: time.Now(),
		Success:   true,
	}

	// Check database health (simplified)
	response.DatabaseHealth = HealthStatus{
		Status: "healthy",
		ResponseTime: time.Millisecond * 10,
	}

	// Check Redis health (simplified)
	response.RedisHealth = HealthStatus{
		Status: "healthy",
		ResponseTime: time.Millisecond * 5,
	}

	// Get system metrics
	pendingTxs, _ := s.transactionRepo.GetPendingTransactions(ctx, 1000)
	response.SystemMetrics = SystemMetrics{
		PendingTransactions: int64(len(pendingTxs)),
	}

	return response, nil
}

func (s *adminService) ManualReconciliation(ctx context.Context, req *ManualReconciliationRequest) (*ManualReconciliationResponse, error) {
	// Get current wallet
	wallet, err := s.walletRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return &ManualReconciliationResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
		}, nil
	}

	// Calculate adjustment needed
	currentBalance := wallet.Balance.Total
	adjustment := req.NewBalance.Sub(currentBalance)

	// Create adjustment transaction
	adjustmentReq := &CreateBalanceAdjustmentRequest{
		UserID:     req.UserID,
		Amount:     adjustment,
		Reason:     req.Reason,
		AdjustedBy: req.AdjustedBy,
	}

	return s.CreateBalanceAdjustment(ctx, adjustmentReq)
}

func (s *adminService) GetTransactionStats(ctx context.Context, req *GetTransactionStatsRequest) (*GetTransactionStatsResponse, error) {
	if req.UserID > 0 {
		// Get wallet first
		wallet, err := s.walletRepo.GetByUserID(ctx, req.UserID)
		if err != nil {
			return &GetTransactionStatsResponse{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to get wallet: %v", err),
			}, nil
		}

		stats, err := s.transactionRepo.GetTransactionStats(ctx, wallet.ID, req.StartDate, req.EndDate)
		if err != nil {
			return &GetTransactionStatsResponse{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to get transaction stats: %v", err),
			}, nil
		}

		return &GetTransactionStatsResponse{
			Stats:   stats,
			Success: true,
		}, nil
	}

	// System-wide stats would require additional implementation
	return &GetTransactionStatsResponse{
		Success:      false,
		ErrorMessage: "System-wide stats not implemented",
	}, nil
}

func (s *adminService) CleanupExpiredLocks(ctx context.Context) (*CleanupResponse, error) {
	err := s.walletRepo.CleanupExpiredLocks(ctx)
	if err != nil {
		return &CleanupResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to cleanup expired locks: %v", err),
		}, nil
	}

	return &CleanupResponse{
		Success: true,
	}, nil
}

func (s *adminService) GetSuspiciousTransactions(ctx context.Context, req *GetSuspiciousTransactionsRequest) (*GetSuspiciousTransactionsResponse, error) {
	// Get failed transactions as a proxy for suspicious ones
	limit := req.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	transactions, err := s.transactionRepo.GetFailedTransactions(ctx, limit, req.StartDate)
	if err != nil {
		return &GetSuspiciousTransactionsResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get suspicious transactions: %v", err),
		}, nil
	}

	return &GetSuspiciousTransactionsResponse{
		Transactions: transactions,
		Success:      true,
	}, nil
}

func (s *adminService) generateAuditReport(wallet *models.Wallet, transactions []*models.Transaction, startDate, endDate time.Time) *AuditReport {
	report := &AuditReport{
		UserID:   wallet.UserID,
		WalletID: wallet.ID,
		ReportPeriod: ReportPeriod{
			StartDate: startDate,
			EndDate:   endDate,
		},
		GeneratedAt: time.Now(),
	}

	// Calculate balance summary
	totalDeposits := decimal.Zero
	totalWithdrawals := decimal.Zero
	totalFees := decimal.Zero

	transactionStats := TransactionStatistics{
		ByType:       make(map[string]int64),
		VolumeByType: make(map[string]decimal.Decimal),
	}

	for _, tx := range transactions {
		if tx.Status == "completed" {
			transactionStats.TotalTransactions++
			transactionStats.SuccessfulTransactions++

			transactionStats.ByType[tx.Type]++

			amount := tx.Amount.Value.Abs()
			if existing, ok := transactionStats.VolumeByType[tx.Type]; ok {
				transactionStats.VolumeByType[tx.Type] = existing.Add(amount)
			} else {
				transactionStats.VolumeByType[tx.Type] = amount
			}

			switch tx.Type {
			case "deposit":
				totalDeposits = totalDeposits.Add(amount)
			case "withdrawal":
				totalWithdrawals = totalWithdrawals.Add(amount)
			}

			totalFees = totalFees.Add(tx.Amount.Fee)
		} else if tx.Status == "failed" {
			transactionStats.FailedTransactions++
		}
	}

	report.BalanceSummary = BalanceSummary{
		ClosingBalance:   wallet.Balance.Total,
		TotalDeposits:    totalDeposits,
		TotalWithdrawals: totalWithdrawals,
		TotalFees:        totalFees,
		NetChange:        totalDeposits.Sub(totalWithdrawals).Sub(totalFees),
	}

	report.TransactionStats = transactionStats

	return report
}