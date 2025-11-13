package models

import "time"

// BalanceTransaction representa una transacción de saldo procesada
type BalanceTransaction struct {
	ID              int32     `json:"id" gorm:"primaryKey;autoIncrement"`
	OrderID         string    `json:"order_id" gorm:"uniqueIndex;not null;size:100"`
	UserID          int32     `json:"user_id" gorm:"not null;index"`
	Amount          float64   `json:"amount" gorm:"type:decimal(15,2);not null"`
	TransactionType string    `json:"transaction_type" gorm:"size:10;not null"` // "buy" o "sell"
	CryptoSymbol    string    `json:"crypto_symbol" gorm:"size:10"`
	PreviousBalance float64   `json:"previous_balance" gorm:"type:decimal(15,2)"`
	NewBalance      float64   `json:"new_balance" gorm:"type:decimal(15,2)"`
	ProcessedAt     time.Time `json:"processed_at" gorm:"autoCreateTime"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (bt *BalanceTransaction) TableName() string {
	return "balance_transactions"
}
