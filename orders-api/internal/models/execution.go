package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ExecutionResult contiene el resultado simplificado de ejecutar una orden
type ExecutionResult struct {
	Success       bool            `json:"success"`
	OrderID       string          `json:"order_id"`
	ExecutedPrice decimal.Decimal `json:"executed_price"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	Fee           decimal.Decimal `json:"fee"`
	ExecutionTime time.Duration   `json:"execution_time"`
	Error         string          `json:"error,omitempty"`
}

// ValidationResult resultado de validar un usuario
type ValidationResult struct {
	IsValid bool   `json:"is_valid"`
	UserID  int    `json:"user_id"`
	Message string `json:"message"`
}

// BalanceResult resultado de verificar balance
type BalanceResult struct {
	HasSufficient bool            `json:"has_sufficient"`
	Available     decimal.Decimal `json:"available"`
	Required      decimal.Decimal `json:"required"`
	Currency      string          `json:"currency"`
	Message       string          `json:"message"`
}

// PriceResult resultado de obtener precio de mercado
type PriceResult struct {
	Symbol      string          `json:"symbol"`
	MarketPrice decimal.Decimal `json:"market_price"`
	Timestamp   time.Time       `json:"timestamp"`
}

// FeeResult resultado del cálculo de comisiones
type FeeResult struct {
	TotalFee      decimal.Decimal `json:"total_fee"`
	FeePercentage decimal.Decimal `json:"fee_percentage"` // 0.1% = 0.001
	FeeType       string          `json:"fee_type"`       // "taker"
}
