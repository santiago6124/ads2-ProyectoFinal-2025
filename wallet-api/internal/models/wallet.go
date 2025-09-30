package models

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Wallet represents a user's virtual wallet
type Wallet struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID       int64              `bson:"user_id" json:"user_id"`
	WalletNumber string             `bson:"wallet_number" json:"wallet_number"`
	Status       string             `bson:"status" json:"status"` // "active", "suspended", "closed"

	Balance      Balance      `bson:"balance" json:"balance"`
	Limits       Limits       `bson:"limits" json:"limits"`
	UsageToday   UsageToday   `bson:"usage_today" json:"usage_today"`
	Locks        []FundsLock  `bson:"locks" json:"locks"`
	Verification Verification `bson:"verification" json:"verification"`
	Metadata     Metadata     `bson:"metadata" json:"metadata"`

	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at" json:"updated_at"`
	LastActivity time.Time `bson:"last_activity" json:"last_activity"`
}

// Balance represents wallet balance information
type Balance struct {
	Available decimal.Decimal `bson:"available" json:"available"`
	Locked    decimal.Decimal `bson:"locked" json:"locked"`
	Total     decimal.Decimal `bson:"total" json:"total"`
	Currency  string          `bson:"currency" json:"currency"`
}

// Limits represents transaction limits for the wallet
type Limits struct {
	DailyWithdrawal    decimal.Decimal `bson:"daily_withdrawal" json:"daily_withdrawal"`
	DailyDeposit       decimal.Decimal `bson:"daily_deposit" json:"daily_deposit"`
	SingleTransaction  decimal.Decimal `bson:"single_transaction" json:"single_transaction"`
	MonthlyVolume      decimal.Decimal `bson:"monthly_volume" json:"monthly_volume"`
}

// UsageToday represents today's transaction usage
type UsageToday struct {
	Withdrawn         decimal.Decimal `bson:"withdrawn" json:"withdrawn"`
	Deposited         decimal.Decimal `bson:"deposited" json:"deposited"`
	TransactionsCount int             `bson:"transactions_count" json:"transactions_count"`
	LastTransaction   time.Time       `bson:"last_transaction" json:"last_transaction"`
}

// FundsLock represents a locked amount for pending transactions
type FundsLock struct {
	LockID    string          `bson:"lock_id" json:"lock_id"`
	OrderID   string          `bson:"order_id" json:"order_id"`
	Amount    decimal.Decimal `bson:"amount" json:"amount"`
	LockedAt  time.Time       `bson:"locked_at" json:"locked_at"`
	ExpiresAt time.Time       `bson:"expires_at" json:"expires_at"`
	Status    string          `bson:"status" json:"status"` // "active", "released", "executed", "expired"
	Reason    string          `bson:"reason" json:"reason"`
}

// Verification represents wallet verification and integrity data
type Verification struct {
	LastReconciled   time.Time `bson:"last_reconciled" json:"last_reconciled"`
	BalanceHash      string    `bson:"balance_hash" json:"balance_hash"`
	TransactionCount int64     `bson:"transaction_count" json:"transaction_count"`
	Checksum         string    `bson:"checksum" json:"checksum"`
}

// Metadata represents additional wallet metadata
type Metadata struct {
	InitialBalance    decimal.Decimal `bson:"initial_balance" json:"initial_balance"`
	TotalDeposits     decimal.Decimal `bson:"total_deposits" json:"total_deposits"`
	TotalWithdrawals  decimal.Decimal `bson:"total_withdrawals" json:"total_withdrawals"`
	TotalFeesPaid     decimal.Decimal `bson:"total_fees_paid" json:"total_fees_paid"`
	AccountAgeDays    int             `bson:"account_age_days" json:"account_age_days"`
}

// NewWallet creates a new wallet for a user
func NewWallet(userID int64, initialBalance decimal.Decimal, limits Limits) *Wallet {
	now := time.Now()
	walletNumber := fmt.Sprintf("WAL-%d-%06d", now.Year(), userID)

	return &Wallet{
		UserID:       userID,
		WalletNumber: walletNumber,
		Status:       "active",
		Balance: Balance{
			Available: initialBalance,
			Locked:    decimal.Zero,
			Total:     initialBalance,
			Currency:  "USD",
		},
		Limits: limits,
		UsageToday: UsageToday{
			Withdrawn:         decimal.Zero,
			Deposited:         decimal.Zero,
			TransactionsCount: 0,
			LastTransaction:   time.Time{},
		},
		Locks: make([]FundsLock, 0),
		Verification: Verification{
			LastReconciled:   now,
			TransactionCount: 0,
		},
		Metadata: Metadata{
			InitialBalance:   initialBalance,
			TotalDeposits:    decimal.Zero,
			TotalWithdrawals: decimal.Zero,
			TotalFeesPaid:    decimal.Zero,
			AccountAgeDays:   0,
		},
		CreatedAt:    now,
		UpdatedAt:    now,
		LastActivity: now,
	}
}

// GetAvailableBalance returns the available balance
func (w *Wallet) GetAvailableBalance() decimal.Decimal {
	return w.Balance.Available
}

// GetLockedBalance returns the locked balance
func (w *Wallet) GetLockedBalance() decimal.Decimal {
	return w.Balance.Locked
}

// GetTotalBalance returns the total balance
func (w *Wallet) GetTotalBalance() decimal.Decimal {
	return w.Balance.Available.Add(w.Balance.Locked)
}

// HasSufficientBalance checks if wallet has sufficient available balance
func (w *Wallet) HasSufficientBalance(amount decimal.Decimal) bool {
	return w.Balance.Available.GreaterThanOrEqual(amount)
}

// GetActiveLocks returns all active locks
func (w *Wallet) GetActiveLocks() []FundsLock {
	var activeLocks []FundsLock
	for _, lock := range w.Locks {
		if lock.Status == "active" && time.Now().Before(lock.ExpiresAt) {
			activeLocks = append(activeLocks, lock)
		}
	}
	return activeLocks
}

// GetLockByID returns a lock by its ID
func (w *Wallet) GetLockByID(lockID string) (*FundsLock, bool) {
	for i, lock := range w.Locks {
		if lock.LockID == lockID {
			return &w.Locks[i], true
		}
	}
	return nil, false
}

// AddLock adds a new fund lock
func (w *Wallet) AddLock(lock FundsLock) error {
	// Check if we have sufficient available balance
	if !w.HasSufficientBalance(lock.Amount) {
		return fmt.Errorf("insufficient available balance: required %s, available %s",
			lock.Amount.String(), w.Balance.Available.String())
	}

	// Move funds from available to locked
	w.Balance.Available = w.Balance.Available.Sub(lock.Amount)
	w.Balance.Locked = w.Balance.Locked.Add(lock.Amount)

	// Add the lock
	w.Locks = append(w.Locks, lock)
	w.UpdatedAt = time.Now()

	return nil
}

// ReleaseLock releases a fund lock and returns funds to available balance
func (w *Wallet) ReleaseLock(lockID string) error {
	lock, exists := w.GetLockByID(lockID)
	if !exists {
		return fmt.Errorf("lock not found: %s", lockID)
	}

	if lock.Status != "active" {
		return fmt.Errorf("lock is not active: %s", lock.Status)
	}

	// Move funds from locked back to available
	w.Balance.Locked = w.Balance.Locked.Sub(lock.Amount)
	w.Balance.Available = w.Balance.Available.Add(lock.Amount)

	// Update lock status
	lock.Status = "released"
	w.UpdatedAt = time.Now()

	return nil
}

// ExecuteLock executes a fund lock (used when transaction is completed)
func (w *Wallet) ExecuteLock(lockID string, actualAmount decimal.Decimal) error {
	lock, exists := w.GetLockByID(lockID)
	if !exists {
		return fmt.Errorf("lock not found: %s", lockID)
	}

	if lock.Status != "active" {
		return fmt.Errorf("lock is not active: %s", lock.Status)
	}

	// Remove from locked balance
	w.Balance.Locked = w.Balance.Locked.Sub(lock.Amount)

	// If actual amount is different from locked amount, adjust available balance
	difference := lock.Amount.Sub(actualAmount)
	if difference.GreaterThan(decimal.Zero) {
		// Return excess to available balance
		w.Balance.Available = w.Balance.Available.Add(difference)
	} else if difference.LessThan(decimal.Zero) {
		// Additional amount needed from available balance
		additionalRequired := difference.Abs()
		if !w.HasSufficientBalance(additionalRequired) {
			return fmt.Errorf("insufficient balance for lock execution")
		}
		w.Balance.Available = w.Balance.Available.Sub(additionalRequired)
	}

	// Update total balance
	w.Balance.Total = w.Balance.Available.Add(w.Balance.Locked)

	// Update lock status
	lock.Status = "executed"
	w.UpdatedAt = time.Now()

	return nil
}

// UpdateBalance updates the wallet balance
func (w *Wallet) UpdateBalance(amount decimal.Decimal, transactionType string) {
	w.Balance.Available = w.Balance.Available.Add(amount)
	w.Balance.Total = w.Balance.Available.Add(w.Balance.Locked)

	// Update usage tracking
	now := time.Now()
	if isSameDay(w.UsageToday.LastTransaction, now) {
		// Same day, update current usage
		if amount.GreaterThan(decimal.Zero) {
			w.UsageToday.Deposited = w.UsageToday.Deposited.Add(amount)
		} else {
			w.UsageToday.Withdrawn = w.UsageToday.Withdrawn.Add(amount.Abs())
		}
		w.UsageToday.TransactionsCount++
	} else {
		// New day, reset usage
		if amount.GreaterThan(decimal.Zero) {
			w.UsageToday.Deposited = amount
			w.UsageToday.Withdrawn = decimal.Zero
		} else {
			w.UsageToday.Withdrawn = amount.Abs()
			w.UsageToday.Deposited = decimal.Zero
		}
		w.UsageToday.TransactionsCount = 1
	}

	w.UsageToday.LastTransaction = now
	w.LastActivity = now
	w.UpdatedAt = now

	// Update metadata
	if amount.GreaterThan(decimal.Zero) {
		w.Metadata.TotalDeposits = w.Metadata.TotalDeposits.Add(amount)
	} else {
		w.Metadata.TotalWithdrawals = w.Metadata.TotalWithdrawals.Add(amount.Abs())
	}
}

// GetRemainingDailyWithdrawal returns remaining daily withdrawal limit
func (w *Wallet) GetRemainingDailyWithdrawal() decimal.Decimal {
	remaining := w.Limits.DailyWithdrawal.Sub(w.UsageToday.Withdrawn)
	if remaining.LessThan(decimal.Zero) {
		return decimal.Zero
	}
	return remaining
}

// GetRemainingDailyDeposit returns remaining daily deposit limit
func (w *Wallet) GetRemainingDailyDeposit() decimal.Decimal {
	remaining := w.Limits.DailyDeposit.Sub(w.UsageToday.Deposited)
	if remaining.LessThan(decimal.Zero) {
		return decimal.Zero
	}
	return remaining
}

// CanWithdraw checks if a withdrawal amount is allowed
func (w *Wallet) CanWithdraw(amount decimal.Decimal) error {
	// Check available balance
	if !w.HasSufficientBalance(amount) {
		return fmt.Errorf("insufficient balance")
	}

	// Check single transaction limit
	if amount.GreaterThan(w.Limits.SingleTransaction) {
		return fmt.Errorf("amount exceeds single transaction limit")
	}

	// Check daily withdrawal limit
	if amount.GreaterThan(w.GetRemainingDailyWithdrawal()) {
		return fmt.Errorf("amount exceeds daily withdrawal limit")
	}

	return nil
}

// CanDeposit checks if a deposit amount is allowed
func (w *Wallet) CanDeposit(amount decimal.Decimal) error {
	// Check single transaction limit
	if amount.GreaterThan(w.Limits.SingleTransaction) {
		return fmt.Errorf("amount exceeds single transaction limit")
	}

	// Check daily deposit limit
	if amount.GreaterThan(w.GetRemainingDailyDeposit()) {
		return fmt.Errorf("amount exceeds daily deposit limit")
	}

	return nil
}

// IsActive checks if the wallet is active
func (w *Wallet) IsActive() bool {
	return w.Status == "active"
}

// Validate validates the wallet data
func (w *Wallet) Validate() error {
	if w.UserID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	if w.WalletNumber == "" {
		return fmt.Errorf("wallet number is required")
	}

	if w.Balance.Available.LessThan(decimal.Zero) {
		return fmt.Errorf("available balance cannot be negative")
	}

	if w.Balance.Locked.LessThan(decimal.Zero) {
		return fmt.Errorf("locked balance cannot be negative")
	}

	if w.Balance.Currency == "" {
		return fmt.Errorf("currency is required")
	}

	// Validate balance consistency
	calculatedTotal := w.Balance.Available.Add(w.Balance.Locked)
	if !calculatedTotal.Equal(w.Balance.Total) {
		return fmt.Errorf("balance inconsistency: total does not match available + locked")
	}

	return nil
}

// CleanupExpiredLocks removes expired locks and returns funds to available balance
func (w *Wallet) CleanupExpiredLocks() {
	now := time.Now()
	var activeLocks []FundsLock

	for _, lock := range w.Locks {
		if lock.Status == "active" && now.After(lock.ExpiresAt) {
			// Release expired lock
			w.Balance.Locked = w.Balance.Locked.Sub(lock.Amount)
			w.Balance.Available = w.Balance.Available.Add(lock.Amount)
			lock.Status = "expired"
		}

		// Keep all locks for audit trail, but only active ones affect balance
		activeLocks = append(activeLocks, lock)
	}

	w.Locks = activeLocks
	w.Balance.Total = w.Balance.Available.Add(w.Balance.Locked)
	w.UpdatedAt = now
}

// Helper function to check if two times are on the same day
func isSameDay(t1, t2 time.Time) bool {
	if t1.IsZero() || t2.IsZero() {
		return false
	}
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}