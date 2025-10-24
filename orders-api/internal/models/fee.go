package models

import (
	"github.com/shopspring/decimal"
)

// FeeCalculation represents a simple 0.1% fee calculation
type FeeCalculation struct {
	OrderValue    decimal.Decimal `json:"order_value"`
	TotalFee      decimal.Decimal `json:"total_fee"`
	FeePercentage decimal.Decimal `json:"fee_percentage"`
	MinimumFee    decimal.Decimal `json:"minimum_fee"`
}

// FeeResult represents the result of a fee calculation
type FeeResult struct {
	Fee           decimal.Decimal `json:"fee"`
	FeePercentage decimal.Decimal `json:"fee_percentage"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
}

// NewFeeCalculation creates a new fee calculation with default 0.1% fee
func NewFeeCalculation(orderValue decimal.Decimal) *FeeCalculation {
	feePercentage := decimal.NewFromFloat(0.001) // 0.1%
	minimumFee := decimal.NewFromFloat(0.01)     // $0.01 minimum

	calculatedFee := orderValue.Mul(feePercentage)

	// Apply minimum fee if calculated fee is less
	totalFee := calculatedFee
	if calculatedFee.LessThan(minimumFee) {
		totalFee = minimumFee
	}

	return &FeeCalculation{
		OrderValue:    orderValue,
		TotalFee:      totalFee,
		FeePercentage: feePercentage,
		MinimumFee:    minimumFee,
	}
}

// Calculate returns the fee for a given order value
func Calculate(orderValue decimal.Decimal) *FeeResult {
	calc := NewFeeCalculation(orderValue)

	return &FeeResult{
		Fee:           calc.TotalFee,
		FeePercentage: calc.FeePercentage,
		TotalAmount:   orderValue.Add(calc.TotalFee),
	}
}
