package services

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"orders-api/internal/models"
)

// FeeConfig represents configuration for fee calculation
type FeeConfig struct {
	BaseFeePercentage decimal.Decimal `json:"base_fee_percentage"`
	MakerFee          decimal.Decimal `json:"maker_fee"`
	TakerFee          decimal.Decimal `json:"taker_fee"`
	MinimumFee        decimal.Decimal `json:"minimum_fee"`
	MaximumFee        decimal.Decimal `json:"maximum_fee"`
	VIPDiscounts      map[string]decimal.Decimal `json:"vip_discounts"`
}

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

// NewFeeCalculator creates a new fee calculator with configuration
func NewFeeCalculator(config *FeeConfig) FeeCalculator {
	if config == nil {
		// Default configuration
		config = &FeeConfig{
			BaseFeePercentage: decimal.NewFromFloat(0.001), // 0.1%
			MakerFee:          decimal.NewFromFloat(0.0005), // 0.05%
			TakerFee:          decimal.NewFromFloat(0.001),  // 0.1%
			MinimumFee:        decimal.NewFromFloat(0.01),   // $0.01
		}
	}
	
	return &feeCalculator{
		feePercentage: config.TakerFee,
		minimumFee:    config.MinimumFee,
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
		BaseFee:       fee,
		PercentageFee: fee,
		TotalFee:      fee,
		FeePercentage: fc.feePercentage,
		FeeType:       "taker",
	}, nil
}

// CalculateForAmount calculates the fee for a given amount
func (fc *feeCalculator) CalculateForAmount(ctx context.Context, amount decimal.Decimal) (*models.FeeCalculation, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	return models.NewFeeCalculation(amount), nil
}
