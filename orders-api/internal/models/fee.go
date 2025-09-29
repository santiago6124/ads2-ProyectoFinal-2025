package models

import (
	"github.com/shopspring/decimal"
)

type FeeCalculation struct {
	OrderValue       decimal.Decimal `json:"order_value"`
	BaseFeeAmount    decimal.Decimal `json:"base_fee_amount"`
	PercentageFee    decimal.Decimal `json:"percentage_fee"`
	TotalFee         decimal.Decimal `json:"total_fee"`
	FeePercentage    decimal.Decimal `json:"fee_percentage"`
	FeeType          FeeType         `json:"fee_type"`
	MinimumFee       decimal.Decimal `json:"minimum_fee"`
	AppliedFee       decimal.Decimal `json:"applied_fee"`
	Discount         decimal.Decimal `json:"discount"`
	DiscountReason   string          `json:"discount_reason,omitempty"`
}

type FeeType string

const (
	FeeTypeMaker FeeType = "maker"
	FeeTypeTaker FeeType = "taker"
	FeeTypeFlat  FeeType = "flat"
)

type FeeStructure struct {
	MakerFee     decimal.Decimal `json:"maker_fee"`
	TakerFee     decimal.Decimal `json:"taker_fee"`
	MinimumFee   decimal.Decimal `json:"minimum_fee"`
	MaximumFee   decimal.Decimal `json:"maximum_fee"`
	FlatFee      decimal.Decimal `json:"flat_fee"`
}

type UserFeeProfile struct {
	UserID          int             `json:"user_id"`
	TierLevel       int             `json:"tier_level"`
	VolumeThisMonth decimal.Decimal `json:"volume_this_month"`
	FeeMultiplier   decimal.Decimal `json:"fee_multiplier"`
	SpecialRates    *FeeStructure   `json:"special_rates,omitempty"`
	DiscountPercent decimal.Decimal `json:"discount_percent"`
}

type FeeTier struct {
	Level           int             `json:"level"`
	MinVolume       decimal.Decimal `json:"min_volume"`
	MakerFee        decimal.Decimal `json:"maker_fee"`
	TakerFee        decimal.Decimal `json:"taker_fee"`
	Description     string          `json:"description"`
}

func NewFeeCalculation(orderValue decimal.Decimal, feeType FeeType) *FeeCalculation {
	return &FeeCalculation{
		OrderValue:    orderValue,
		FeeType:       feeType,
		BaseFeeAmount: decimal.Zero,
		PercentageFee: decimal.Zero,
		TotalFee:      decimal.Zero,
		FeePercentage: decimal.Zero,
		MinimumFee:    decimal.Zero,
		Discount:      decimal.Zero,
	}
}

func (fc *FeeCalculation) CalculateWithPercentage(percentage decimal.Decimal, minimum decimal.Decimal) {
	fc.PercentageFee = percentage
	fc.MinimumFee = minimum

	calculatedFee := fc.OrderValue.Mul(percentage)

	if calculatedFee.LessThan(minimum) {
		fc.AppliedFee = minimum
	} else {
		fc.AppliedFee = calculatedFee
	}

	fc.TotalFee = fc.AppliedFee.Sub(fc.Discount)
	if fc.TotalFee.LessThan(decimal.Zero) {
		fc.TotalFee = decimal.Zero
	}

	if fc.OrderValue.GreaterThan(decimal.Zero) {
		fc.FeePercentage = fc.TotalFee.Div(fc.OrderValue)
	}
}

func (fc *FeeCalculation) ApplyDiscount(discountAmount decimal.Decimal, reason string) {
	fc.Discount = discountAmount
	fc.DiscountReason = reason

	fc.TotalFee = fc.AppliedFee.Sub(fc.Discount)
	if fc.TotalFee.LessThan(decimal.Zero) {
		fc.TotalFee = decimal.Zero
	}

	if fc.OrderValue.GreaterThan(decimal.Zero) {
		fc.FeePercentage = fc.TotalFee.Div(fc.OrderValue)
	}
}

func (fc *FeeCalculation) AddBaseFee(amount decimal.Decimal) {
	fc.BaseFeeAmount = amount
	fc.TotalFee = fc.TotalFee.Add(amount)
}

func GetDefaultFeeStructure() *FeeStructure {
	return &FeeStructure{
		MakerFee:   decimal.NewFromFloat(0.0008), // 0.08%
		TakerFee:   decimal.NewFromFloat(0.0012), // 0.12%
		MinimumFee: decimal.NewFromFloat(0.01),   // $0.01
		MaximumFee: decimal.NewFromFloat(1000.0), // $1000
		FlatFee:    decimal.Zero,
	}
}

func GetFeeTiers() []FeeTier {
	return []FeeTier{
		{
			Level:       0,
			MinVolume:   decimal.Zero,
			MakerFee:    decimal.NewFromFloat(0.0012),
			TakerFee:    decimal.NewFromFloat(0.0015),
			Description: "Standard",
		},
		{
			Level:       1,
			MinVolume:   decimal.NewFromFloat(10000),
			MakerFee:    decimal.NewFromFloat(0.0010),
			TakerFee:    decimal.NewFromFloat(0.0012),
			Description: "Bronze",
		},
		{
			Level:       2,
			MinVolume:   decimal.NewFromFloat(50000),
			MakerFee:    decimal.NewFromFloat(0.0008),
			TakerFee:    decimal.NewFromFloat(0.0010),
			Description: "Silver",
		},
		{
			Level:       3,
			MinVolume:   decimal.NewFromFloat(100000),
			MakerFee:    decimal.NewFromFloat(0.0006),
			TakerFee:    decimal.NewFromFloat(0.0008),
			Description: "Gold",
		},
		{
			Level:       4,
			MinVolume:   decimal.NewFromFloat(500000),
			MakerFee:    decimal.NewFromFloat(0.0004),
			TakerFee:    decimal.NewFromFloat(0.0006),
			Description: "Platinum",
		},
	}
}

func DetermineFeeType(orderKind OrderKind) FeeType {
	switch orderKind {
	case OrderKindLimit:
		return FeeTypeMaker
	case OrderKindMarket:
		return FeeTypeTaker
	default:
		return FeeTypeTaker
	}
}