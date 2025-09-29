package services

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"orders-api/internal/models"
)

type feeCalculator struct {
	config *FeeConfig
}

type FeeConfig struct {
	BaseFeePercentage decimal.Decimal
	MakerFee         decimal.Decimal
	TakerFee         decimal.Decimal
	MinimumFee       decimal.Decimal
	MaximumFee       decimal.Decimal
	TierStructure    []models.FeeTier
	VIPDiscounts     map[string]decimal.Decimal
}

func NewFeeCalculator(config *FeeConfig) FeeCalculator {
	if config == nil {
		config = DefaultFeeConfig()
	}
	return &feeCalculator{
		config: config,
	}
}

func (fc *feeCalculator) Calculate(ctx context.Context, order *models.Order) (*models.FeeResult, error) {
	if order == nil {
		return nil, fmt.Errorf("order cannot be nil")
	}

	feeType := models.DetermineFeeType(order.OrderKind)
	orderValue := order.Quantity.Mul(order.GetEffectivePrice())

	calculation := models.NewFeeCalculation(orderValue, feeType)

	var feePercentage decimal.Decimal
	switch feeType {
	case models.FeeTypeMaker:
		feePercentage = fc.config.MakerFee
	case models.FeeTypeTaker:
		feePercentage = fc.config.TakerFee
	default:
		feePercentage = fc.config.BaseFeePercentage
	}

	calculation.CalculateWithPercentage(feePercentage, fc.config.MinimumFee)

	if calculation.TotalFee.GreaterThan(fc.config.MaximumFee) {
		calculation.TotalFee = fc.config.MaximumFee
		calculation.AppliedFee = fc.config.MaximumFee
	}

	userProfile := fc.getUserFeeProfile(order.UserID)
	if userProfile != nil {
		fc.applyUserDiscounts(calculation, userProfile)
	}

	return &models.FeeResult{
		BaseFee:       calculation.BaseFeeAmount,
		PercentageFee: calculation.PercentageFee,
		TotalFee:      calculation.TotalFee,
		FeePercentage: calculation.FeePercentage,
		FeeType:       string(feeType),
	}, nil
}

func (fc *feeCalculator) CalculateForAmount(ctx context.Context, amount decimal.Decimal, orderType models.OrderKind) (*models.FeeCalculation, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	feeType := models.DetermineFeeType(orderType)
	calculation := models.NewFeeCalculation(amount, feeType)

	var feePercentage decimal.Decimal
	switch feeType {
	case models.FeeTypeMaker:
		feePercentage = fc.config.MakerFee
	case models.FeeTypeTaker:
		feePercentage = fc.config.TakerFee
	default:
		feePercentage = fc.config.BaseFeePercentage
	}

	calculation.CalculateWithPercentage(feePercentage, fc.config.MinimumFee)

	if calculation.TotalFee.GreaterThan(fc.config.MaximumFee) {
		calculation.TotalFee = fc.config.MaximumFee
		calculation.AppliedFee = fc.config.MaximumFee
	}

	return calculation, nil
}

func (fc *feeCalculator) getUserFeeProfile(userID int) *models.UserFeeProfile {
	return nil
}

func (fc *feeCalculator) applyUserDiscounts(calculation *models.FeeCalculation, profile *models.UserFeeProfile) {
	if profile.DiscountPercent.GreaterThan(decimal.Zero) {
		discountAmount := calculation.TotalFee.Mul(profile.DiscountPercent.Div(decimal.NewFromFloat(100)))
		calculation.ApplyDiscount(discountAmount, fmt.Sprintf("User tier %d discount", profile.TierLevel))
	}

	if profile.FeeMultiplier.GreaterThan(decimal.Zero) && !profile.FeeMultiplier.Equal(decimal.NewFromFloat(1)) {
		calculation.TotalFee = calculation.TotalFee.Mul(profile.FeeMultiplier)
		calculation.AppliedFee = calculation.TotalFee
	}

	if profile.SpecialRates != nil {
		switch calculation.FeeType {
		case models.FeeTypeMaker:
			if profile.SpecialRates.MakerFee.GreaterThan(decimal.Zero) {
				newFee := calculation.OrderValue.Mul(profile.SpecialRates.MakerFee)
				if newFee.LessThan(calculation.TotalFee) {
					calculation.TotalFee = newFee
					calculation.AppliedFee = newFee
					calculation.PercentageFee = profile.SpecialRates.MakerFee
				}
			}
		case models.FeeTypeTaker:
			if profile.SpecialRates.TakerFee.GreaterThan(decimal.Zero) {
				newFee := calculation.OrderValue.Mul(profile.SpecialRates.TakerFee)
				if newFee.LessThan(calculation.TotalFee) {
					calculation.TotalFee = newFee
					calculation.AppliedFee = newFee
					calculation.PercentageFee = profile.SpecialRates.TakerFee
				}
			}
		}
	}

	if calculation.TotalFee.LessThan(fc.config.MinimumFee) {
		calculation.TotalFee = fc.config.MinimumFee
		calculation.AppliedFee = fc.config.MinimumFee
	}
}

func (fc *feeCalculator) CalculateFeeForTier(amount decimal.Decimal, tierLevel int, feeType models.FeeType) (*models.FeeCalculation, error) {
	tiers := models.GetFeeTiers()

	var tier *models.FeeTier
	for i := len(tiers) - 1; i >= 0; i-- {
		if tierLevel >= tiers[i].Level {
			tier = &tiers[i]
			break
		}
	}

	if tier == nil {
		tier = &tiers[0]
	}

	calculation := models.NewFeeCalculation(amount, feeType)

	var feePercentage decimal.Decimal
	switch feeType {
	case models.FeeTypeMaker:
		feePercentage = tier.MakerFee
	case models.FeeTypeTaker:
		feePercentage = tier.TakerFee
	default:
		feePercentage = fc.config.BaseFeePercentage
	}

	calculation.CalculateWithPercentage(feePercentage, fc.config.MinimumFee)

	return calculation, nil
}

func (fc *feeCalculator) EstimateFee(amount decimal.Decimal, orderType models.OrderKind, userID int) (*models.FeeCalculation, error) {
	feeType := models.DetermineFeeType(orderType)
	calculation := models.NewFeeCalculation(amount, feeType)

	var feePercentage decimal.Decimal
	switch feeType {
	case models.FeeTypeMaker:
		feePercentage = fc.config.MakerFee
	case models.FeeTypeTaker:
		feePercentage = fc.config.TakerFee
	default:
		feePercentage = fc.config.BaseFeePercentage
	}

	calculation.CalculateWithPercentage(feePercentage, fc.config.MinimumFee)

	userProfile := fc.getUserFeeProfile(userID)
	if userProfile != nil {
		fc.applyUserDiscounts(calculation, userProfile)
	}

	return calculation, nil
}

func (fc *feeCalculator) GetFeeStructure() *models.FeeStructure {
	return &models.FeeStructure{
		MakerFee:   fc.config.MakerFee,
		TakerFee:   fc.config.TakerFee,
		MinimumFee: fc.config.MinimumFee,
		MaximumFee: fc.config.MaximumFee,
		FlatFee:    decimal.Zero,
	}
}

func DefaultFeeConfig() *FeeConfig {
	return &FeeConfig{
		BaseFeePercentage: decimal.NewFromFloat(0.001), // 0.1%
		MakerFee:         decimal.NewFromFloat(0.0008), // 0.08%
		TakerFee:         decimal.NewFromFloat(0.0012), // 0.12%
		MinimumFee:       decimal.NewFromFloat(0.01),   // $0.01
		MaximumFee:       decimal.NewFromFloat(1000.0), // $1000
		TierStructure:    models.GetFeeTiers(),
		VIPDiscounts: map[string]decimal.Decimal{
			"bronze":   decimal.NewFromFloat(0.05),  // 5% discount
			"silver":   decimal.NewFromFloat(0.10),  // 10% discount
			"gold":     decimal.NewFromFloat(0.15),  // 15% discount
			"platinum": decimal.NewFromFloat(0.25),  // 25% discount
		},
	}
}

func LoadFeeConfigFromEnv() *FeeConfig {
	config := DefaultFeeConfig()

	return config
}

func (fc *feeCalculator) ValidateFeeCalculation(calculation *models.FeeCalculation) error {
	if calculation.OrderValue.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("order value must be greater than zero")
	}

	if calculation.TotalFee.LessThan(decimal.Zero) {
		return fmt.Errorf("total fee cannot be negative")
	}

	if calculation.TotalFee.GreaterThan(calculation.OrderValue) {
		return fmt.Errorf("total fee cannot exceed order value")
	}

	if calculation.FeePercentage.LessThan(decimal.Zero) || calculation.FeePercentage.GreaterThan(decimal.NewFromFloat(1)) {
		return fmt.Errorf("fee percentage must be between 0 and 1")
	}

	return nil
}

func (fc *feeCalculator) GetEffectiveFeeRate(userID int, orderType models.OrderKind) decimal.Decimal {
	feeType := models.DetermineFeeType(orderType)

	var baseFee decimal.Decimal
	switch feeType {
	case models.FeeTypeMaker:
		baseFee = fc.config.MakerFee
	case models.FeeTypeTaker:
		baseFee = fc.config.TakerFee
	default:
		baseFee = fc.config.BaseFeePercentage
	}

	userProfile := fc.getUserFeeProfile(userID)
	if userProfile != nil && userProfile.FeeMultiplier.GreaterThan(decimal.Zero) {
		baseFee = baseFee.Mul(userProfile.FeeMultiplier)
	}

	return baseFee
}