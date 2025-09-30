package models

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Transaction represents a wallet transaction
type Transaction struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	TransactionID   string             `bson:"transaction_id" json:"transaction_id"`
	WalletID        primitive.ObjectID `bson:"wallet_id" json:"wallet_id"`
	UserID          int64              `bson:"user_id" json:"user_id"`
	IdempotencyKey  string             `bson:"idempotency_key" json:"idempotency_key"`

	Type   string `bson:"type" json:"type"`     // "deposit", "withdrawal", "order_lock", "order_release", "order_execute", "fee", "refund", "adjustment"
	Status string `bson:"status" json:"status"` // "pending", "processing", "completed", "failed", "reversed"

	Amount       TransactionAmount `bson:"amount" json:"amount"`
	Balance      BalanceSnapshot   `bson:"balance" json:"balance"`
	Reference    Reference         `bson:"reference" json:"reference"`
	Processing   ProcessingInfo    `bson:"processing" json:"processing"`
	Audit        AuditInfo         `bson:"audit" json:"audit"`
	Reversal     ReversalInfo      `bson:"reversal" json:"reversal"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// TransactionAmount represents transaction amount details
type TransactionAmount struct {
	Value    decimal.Decimal `bson:"value" json:"value"`
	Currency string          `bson:"currency" json:"currency"`
	Fee      decimal.Decimal `bson:"fee" json:"fee"`
	Net      decimal.Decimal `bson:"net" json:"net"`
}

// BalanceSnapshot represents balance before and after transaction
type BalanceSnapshot struct {
	Before          decimal.Decimal `bson:"before" json:"before"`
	After           decimal.Decimal `bson:"after" json:"after"`
	AvailableBefore decimal.Decimal `bson:"available_before" json:"available_before"`
	AvailableAfter  decimal.Decimal `bson:"available_after" json:"available_after"`
	LockedBefore    decimal.Decimal `bson:"locked_before" json:"locked_before"`
	LockedAfter     decimal.Decimal `bson:"locked_after" json:"locked_after"`
}

// Reference represents transaction reference information
type Reference struct {
	Type        string                 `bson:"type" json:"type"` // "order", "manual", "system", "external"
	ID          string                 `bson:"id" json:"id"`
	Description string                 `bson:"description" json:"description"`
	Metadata    map[string]interface{} `bson:"metadata" json:"metadata"`
}

// ProcessingInfo represents transaction processing details
type ProcessingInfo struct {
	InitiatedAt    time.Time `bson:"initiated_at" json:"initiated_at"`
	CompletedAt    time.Time `bson:"completed_at" json:"completed_at"`
	ProcessingTime int64     `bson:"processing_time_ms" json:"processing_time_ms"`
	Attempts       int       `bson:"attempts" json:"attempts"`
	Errors         []string  `bson:"errors" json:"errors"`
}

// AuditInfo represents audit trail information
type AuditInfo struct {
	IPAddress  string `bson:"ip_address" json:"ip_address"`
	UserAgent  string `bson:"user_agent" json:"user_agent"`
	SessionID  string `bson:"session_id" json:"session_id"`
	APIVersion string `bson:"api_version" json:"api_version"`
}

// ReversalInfo represents transaction reversal information
type ReversalInfo struct {
	IsReversed            bool   `bson:"is_reversed" json:"is_reversed"`
	ReversedBy            string `bson:"reversed_by" json:"reversed_by"`
	ReversalTransactionID string `bson:"reversal_transaction_id" json:"reversal_transaction_id"`
	ReversalReason        string `bson:"reversal_reason" json:"reversal_reason"`
}

// TransactionRequest represents a transaction request
type TransactionRequest struct {
	WalletID       primitive.ObjectID    `json:"wallet_id"`
	UserID         int64                 `json:"user_id"`
	Type           string                `json:"type"`
	Amount         decimal.Decimal       `json:"amount"`
	Fee            decimal.Decimal       `json:"fee"`
	Currency       string                `json:"currency"`
	Reference      Reference             `json:"reference"`
	IdempotencyKey string                `json:"idempotency_key"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewTransaction creates a new transaction
func NewTransaction(req *TransactionRequest) *Transaction {
	now := time.Now()
	transactionID := fmt.Sprintf("TXN-%d-%d", now.Unix(), req.UserID)

	// Calculate net amount
	netAmount := req.Amount
	if req.Amount.LessThan(decimal.Zero) {
		// For withdrawals, subtract fee from amount
		netAmount = req.Amount.Sub(req.Fee)
	}

	return &Transaction{
		TransactionID:  transactionID,
		WalletID:       req.WalletID,
		UserID:         req.UserID,
		IdempotencyKey: req.IdempotencyKey,
		Type:           req.Type,
		Status:         "pending",
		Amount: TransactionAmount{
			Value:    req.Amount,
			Currency: req.Currency,
			Fee:      req.Fee,
			Net:      netAmount,
		},
		Reference: req.Reference,
		Processing: ProcessingInfo{
			InitiatedAt: now,
			Attempts:    0,
			Errors:      make([]string, 0),
		},
		Reversal: ReversalInfo{
			IsReversed: false,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsDebit returns true if this is a debit transaction
func (t *Transaction) IsDebit() bool {
	return t.Amount.Value.LessThan(decimal.Zero)
}

// IsCredit returns true if this is a credit transaction
func (t *Transaction) IsCredit() bool {
	return t.Amount.Value.GreaterThan(decimal.Zero)
}

// GetAbsoluteAmount returns the absolute amount of the transaction
func (t *Transaction) GetAbsoluteAmount() decimal.Decimal {
	return t.Amount.Value.Abs()
}

// MarkProcessing marks the transaction as processing
func (t *Transaction) MarkProcessing() {
	t.Status = "processing"
	t.Processing.Attempts++
	t.UpdatedAt = time.Now()
}

// MarkCompleted marks the transaction as completed
func (t *Transaction) MarkCompleted(balanceBefore, balanceAfter BalanceSnapshot) {
	now := time.Now()
	t.Status = "completed"
	t.Balance = BalanceSnapshot{
		Before:          balanceBefore.Before,
		After:           balanceAfter.After,
		AvailableBefore: balanceBefore.AvailableBefore,
		AvailableAfter:  balanceAfter.AvailableAfter,
		LockedBefore:    balanceBefore.LockedBefore,
		LockedAfter:     balanceAfter.LockedAfter,
	}
	t.Processing.CompletedAt = now
	t.Processing.ProcessingTime = now.Sub(t.Processing.InitiatedAt).Milliseconds()
	t.UpdatedAt = now
}

// MarkFailed marks the transaction as failed
func (t *Transaction) MarkFailed(errorMsg string) {
	t.Status = "failed"
	t.Processing.Errors = append(t.Processing.Errors, errorMsg)
	t.UpdatedAt = time.Now()
}

// CanBeReversed checks if the transaction can be reversed
func (t *Transaction) CanBeReversed() bool {
	// Only completed transactions can be reversed
	if t.Status != "completed" {
		return false
	}

	// Already reversed
	if t.Reversal.IsReversed {
		return false
	}

	// Check if transaction type supports reversal
	reversibleTypes := []string{"deposit", "withdrawal", "refund", "adjustment"}
	for _, rType := range reversibleTypes {
		if t.Type == rType {
			return true
		}
	}

	return false
}

// Reverse creates a reversal transaction
func (t *Transaction) Reverse(reason string, reversedBy string) *Transaction {
	if !t.CanBeReversed() {
		return nil
	}

	now := time.Now()
	reversalID := fmt.Sprintf("REV-%s", t.TransactionID)

	// Create reversal transaction with opposite amount
	reversal := &Transaction{
		TransactionID:  reversalID,
		WalletID:       t.WalletID,
		UserID:         t.UserID,
		IdempotencyKey: fmt.Sprintf("reversal-%s", t.TransactionID),
		Type:           "reversal",
		Status:         "pending",
		Amount: TransactionAmount{
			Value:    t.Amount.Value.Neg(), // Opposite amount
			Currency: t.Amount.Currency,
			Fee:      decimal.Zero, // No fee for reversals
			Net:      t.Amount.Value.Neg(),
		},
		Reference: Reference{
			Type:        "reversal",
			ID:          t.TransactionID,
			Description: fmt.Sprintf("Reversal of transaction %s", t.TransactionID),
			Metadata: map[string]interface{}{
				"original_transaction_id": t.TransactionID,
				"reversal_reason":        reason,
				"reversed_by":            reversedBy,
			},
		},
		Processing: ProcessingInfo{
			InitiatedAt: now,
			Attempts:    0,
			Errors:      make([]string, 0),
		},
		Reversal: ReversalInfo{
			IsReversed: false,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Mark original transaction as reversed
	t.Reversal.IsReversed = true
	t.Reversal.ReversedBy = reversedBy
	t.Reversal.ReversalTransactionID = reversalID
	t.Reversal.ReversalReason = reason
	t.UpdatedAt = now

	return reversal
}

// AddError adds an error to the transaction
func (t *Transaction) AddError(errorMsg string) {
	t.Processing.Errors = append(t.Processing.Errors, errorMsg)
	t.UpdatedAt = time.Now()
}

// SetAuditInfo sets audit information for the transaction
func (t *Transaction) SetAuditInfo(audit AuditInfo) {
	t.Audit = audit
	t.UpdatedAt = time.Now()
}

// Validate validates the transaction data
func (t *Transaction) Validate() error {
	if t.TransactionID == "" {
		return fmt.Errorf("transaction ID is required")
	}

	if t.WalletID.IsZero() {
		return fmt.Errorf("wallet ID is required")
	}

	if t.UserID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	if t.Type == "" {
		return fmt.Errorf("transaction type is required")
	}

	if t.Amount.Value.IsZero() {
		return fmt.Errorf("transaction amount cannot be zero")
	}

	if t.Amount.Currency == "" {
		return fmt.Errorf("currency is required")
	}

	// Validate status
	validStatuses := []string{"pending", "processing", "completed", "failed", "reversed"}
	isValidStatus := false
	for _, status := range validStatuses {
		if t.Status == status {
			isValidStatus = true
			break
		}
	}
	if !isValidStatus {
		return fmt.Errorf("invalid transaction status: %s", t.Status)
	}

	// Validate type
	validTypes := []string{"deposit", "withdrawal", "order_lock", "order_release", "order_execute", "fee", "refund", "adjustment", "reversal"}
	isValidType := false
	for _, tType := range validTypes {
		if t.Type == tType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return fmt.Errorf("invalid transaction type: %s", t.Type)
	}

	return nil
}

// GetDescription returns a human-readable description of the transaction
func (t *Transaction) GetDescription() string {
	if t.Reference.Description != "" {
		return t.Reference.Description
	}

	switch t.Type {
	case "deposit":
		return fmt.Sprintf("Deposit of %s %s", t.Amount.Value.String(), t.Amount.Currency)
	case "withdrawal":
		return fmt.Sprintf("Withdrawal of %s %s", t.Amount.Value.Abs().String(), t.Amount.Currency)
	case "order_execute":
		return fmt.Sprintf("Order execution: %s %s", t.Amount.Value.String(), t.Amount.Currency)
	case "order_lock":
		return fmt.Sprintf("Funds locked for order: %s %s", t.Amount.Value.Abs().String(), t.Amount.Currency)
	case "order_release":
		return fmt.Sprintf("Funds released from order: %s %s", t.Amount.Value.String(), t.Amount.Currency)
	case "fee":
		return fmt.Sprintf("Transaction fee: %s %s", t.Amount.Value.Abs().String(), t.Amount.Currency)
	case "refund":
		return fmt.Sprintf("Refund: %s %s", t.Amount.Value.String(), t.Amount.Currency)
	case "adjustment":
		return fmt.Sprintf("Balance adjustment: %s %s", t.Amount.Value.String(), t.Amount.Currency)
	case "reversal":
		return fmt.Sprintf("Reversal: %s %s", t.Amount.Value.String(), t.Amount.Currency)
	default:
		return fmt.Sprintf("Transaction: %s %s", t.Amount.Value.String(), t.Amount.Currency)
	}
}