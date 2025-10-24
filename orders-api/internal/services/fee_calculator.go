package services

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"orders-api/internal/models"
)

// FeeCalculator interface defines fee calculation methods
type FeeCalculator interface {
	Calculate(ctx context.Context, order *models.Order) (*models.FeeResult, error)
	CalculateForAmount(ctx context.Context, amount decimal.Decimal) (*models.FeeCalculation, error)
}

// feeCalculator implements simple 0.1% fee calculation
type feeCalculator struct {
	feePercentage decimal.Decimal
	minimumFee    decimal.Decimal
}

// NewFeeCalculator creates a new fee calculator with fixed 0.1% fee
func NewFeeCalculator() FeeCalculator {
	return &feeCalculator{
		feePercentage: decimal.NewFromFloat(0.001), // 0.1%
		minimumFee:    decimal.NewFromFloat(0.01),  // $0.01 minimum
	}
}

// Calculate calculates the fee for an order
func (fc *feeCalculator) Calculate(ctx context.Context, order *models.Order) (*models.FeeResult, error) {
	if order == nil {
		return nil, fmt.Errorf("order cannot be nil")
	}

	orderValue := order.Quantity.Mul(order.GetEffectivePrice())

	if orderValue.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("order value must be greater than zero")
	}

	// Calculate fee (0.1% of order value)
	calculatedFee := orderValue.Mul(fc.feePercentage)

	// Apply minimum fee if necessary
	fee := calculatedFee
	if calculatedFee.LessThan(fc.minimumFee) {
		fee = fc.minimumFee
	}

	return &models.FeeResult{
		Fee:           fee,
		FeePercentage: fc.feePercentage,
		TotalAmount:   orderValue.Add(fee),
	}, nil
}

// CalculateForAmount calculates the fee for a given amount
func (fc *feeCalculator) CalculateForAmount(ctx context.Context, amount decimal.Decimal) (*models.FeeCalculation, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	return models.NewFeeCalculation(amount), nil
}
