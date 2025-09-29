package models

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Portfolio represents a user's investment portfolio
type Portfolio struct {
	ID                    primitive.ObjectID    `bson:"_id,omitempty" json:"id,omitempty"`
	UserID                int64                 `bson:"user_id" json:"user_id"`
	TotalValue            decimal.Decimal       `bson:"total_value" json:"total_value"`
	TotalInvested         decimal.Decimal       `bson:"total_invested" json:"total_invested"`
	TotalCash             decimal.Decimal       `bson:"total_cash" json:"total_cash"`
	ProfitLoss            decimal.Decimal       `bson:"profit_loss" json:"profit_loss"`
	ProfitLossPercentage  decimal.Decimal       `bson:"profit_loss_percentage" json:"profit_loss_percentage"`
	Currency              string                `bson:"currency" json:"currency"`
	Holdings              []Holding             `bson:"holdings" json:"holdings"`
	Performance           Performance           `bson:"performance" json:"performance"`
	RiskMetrics           RiskMetrics           `bson:"risk_metrics" json:"risk_metrics"`
	Diversification       Diversification       `bson:"diversification" json:"diversification"`
	Metadata              PortfolioMetadata     `bson:"metadata" json:"metadata"`
	CreatedAt             time.Time             `bson:"created_at" json:"created_at"`
	UpdatedAt             time.Time             `bson:"updated_at" json:"updated_at"`
}

// Holding represents a cryptocurrency holding in the portfolio
type Holding struct {
	CryptoID               string          `bson:"crypto_id" json:"crypto_id"`
	Symbol                 string          `bson:"symbol" json:"symbol"`
	Name                   string          `bson:"name" json:"name"`
	Quantity               decimal.Decimal `bson:"quantity" json:"quantity"`
	AverageBuyPrice        decimal.Decimal `bson:"average_buy_price" json:"average_buy_price"`
	TotalInvested          decimal.Decimal `bson:"total_invested" json:"total_invested"`
	CurrentPrice           decimal.Decimal `bson:"current_price" json:"current_price"`
	CurrentValue           decimal.Decimal `bson:"current_value" json:"current_value"`
	ProfitLoss             decimal.Decimal `bson:"profit_loss" json:"profit_loss"`
	ProfitLossPercentage   decimal.Decimal `bson:"profit_loss_percentage" json:"profit_loss_percentage"`
	PercentageOfPortfolio  decimal.Decimal `bson:"percentage_of_portfolio" json:"percentage_of_portfolio"`
	FirstPurchaseDate      time.Time       `bson:"first_purchase_date" json:"first_purchase_date"`
	LastPurchaseDate       time.Time       `bson:"last_purchase_date" json:"last_purchase_date"`
	TransactionsCount      int             `bson:"transactions_count" json:"transactions_count"`

	// Additional fields for detailed analysis
	CostBasis              []CostBasisEntry `bson:"cost_basis,omitempty" json:"cost_basis,omitempty"`
	DailyChange            decimal.Decimal  `bson:"daily_change,omitempty" json:"daily_change,omitempty"`
	DailyChangePercentage  decimal.Decimal  `bson:"daily_change_percentage,omitempty" json:"daily_change_percentage,omitempty"`
	Category               string           `bson:"category,omitempty" json:"category,omitempty"`
}

// CostBasisEntry represents a cost basis entry for FIFO/LIFO calculations
type CostBasisEntry struct {
	Date     time.Time       `bson:"date" json:"date"`
	Quantity decimal.Decimal `bson:"quantity" json:"quantity"`
	Price    decimal.Decimal `bson:"price" json:"price"`
	OrderID  string          `bson:"order_id,omitempty" json:"order_id,omitempty"`
}

// Performance represents portfolio performance metrics
type Performance struct {
	DailyChange                decimal.Decimal `bson:"daily_change" json:"daily_change"`
	DailyChangePercentage      decimal.Decimal `bson:"daily_change_percentage" json:"daily_change_percentage"`
	WeeklyChange               decimal.Decimal `bson:"weekly_change" json:"weekly_change"`
	WeeklyChangePercentage     decimal.Decimal `bson:"weekly_change_percentage" json:"weekly_change_percentage"`
	MonthlyChange              decimal.Decimal `bson:"monthly_change" json:"monthly_change"`
	MonthlyChangePercentage    decimal.Decimal `bson:"monthly_change_percentage" json:"monthly_change_percentage"`
	YearlyChange               decimal.Decimal `bson:"yearly_change" json:"yearly_change"`
	YearlyChangePercentage     decimal.Decimal `bson:"yearly_change_percentage" json:"yearly_change_percentage"`

	AllTimeHigh                decimal.Decimal `bson:"all_time_high" json:"all_time_high"`
	AllTimeHighDate            time.Time       `bson:"all_time_high_date" json:"all_time_high_date"`
	AllTimeLow                 decimal.Decimal `bson:"all_time_low" json:"all_time_low"`
	AllTimeLowDate             time.Time       `bson:"all_time_low_date" json:"all_time_low_date"`

	BestPerformingAsset        string          `bson:"best_performing_asset" json:"best_performing_asset"`
	WorstPerformingAsset       string          `bson:"worst_performing_asset" json:"worst_performing_asset"`

	ROI                        decimal.Decimal `bson:"roi" json:"roi"`
	AnnualizedReturn           decimal.Decimal `bson:"annualized_return" json:"annualized_return"`
	TimeWeightedReturn         decimal.Decimal `bson:"time_weighted_return,omitempty" json:"time_weighted_return,omitempty"`
	MoneyWeightedReturn        decimal.Decimal `bson:"money_weighted_return,omitempty" json:"money_weighted_return,omitempty"`
}

// RiskMetrics represents risk analysis metrics
type RiskMetrics struct {
	Volatility24h      decimal.Decimal `bson:"volatility_24h" json:"volatility_24h"`
	Volatility7d       decimal.Decimal `bson:"volatility_7d" json:"volatility_7d"`
	Volatility30d      decimal.Decimal `bson:"volatility_30d" json:"volatility_30d"`

	SharpeRatio        decimal.Decimal `bson:"sharpe_ratio" json:"sharpe_ratio"`
	SortinoRatio       decimal.Decimal `bson:"sortino_ratio" json:"sortino_ratio"`
	CalmarRatio        decimal.Decimal `bson:"calmar_ratio,omitempty" json:"calmar_ratio,omitempty"`

	MaxDrawdown        decimal.Decimal `bson:"max_drawdown" json:"max_drawdown"`
	MaxDrawdownDate    time.Time       `bson:"max_drawdown_date" json:"max_drawdown_date"`
	RecoveryTimeDays   int             `bson:"recovery_time_days,omitempty" json:"recovery_time_days,omitempty"`

	Beta               decimal.Decimal `bson:"beta" json:"beta"`
	Alpha              decimal.Decimal `bson:"alpha" json:"alpha"`

	ValueAtRisk95      decimal.Decimal `bson:"value_at_risk_95" json:"value_at_risk_95"`
	ConditionalVaR95   decimal.Decimal `bson:"conditional_var_95" json:"conditional_var_95"`

	DownsideDeviation  decimal.Decimal `bson:"downside_deviation,omitempty" json:"downside_deviation,omitempty"`
	UpsideCapture      decimal.Decimal `bson:"upside_capture,omitempty" json:"upside_capture,omitempty"`
	DownsideCapture    decimal.Decimal `bson:"downside_capture,omitempty" json:"downside_capture,omitempty"`
	TrackingError      decimal.Decimal `bson:"tracking_error,omitempty" json:"tracking_error,omitempty"`
	InformationRatio   decimal.Decimal `bson:"information_ratio,omitempty" json:"information_ratio,omitempty"`
}

// Diversification represents portfolio diversification metrics
type Diversification struct {
	HoldingsCount             int                        `bson:"holdings_count" json:"holdings_count"`
	ConcentrationIndex        decimal.Decimal            `bson:"concentration_index" json:"concentration_index"`
	HerfindahlIndex           decimal.Decimal            `bson:"herfindahl_index" json:"herfindahl_index"`
	EffectiveHoldings         decimal.Decimal            `bson:"effective_holdings,omitempty" json:"effective_holdings,omitempty"`
	Categories                map[string]decimal.Decimal `bson:"categories" json:"categories"`
	LargestPositionPercentage decimal.Decimal            `bson:"largest_position_percentage" json:"largest_position_percentage"`
	Top3Concentration         decimal.Decimal            `bson:"top_3_concentration" json:"top_3_concentration"`
}

// PortfolioMetadata represents metadata about the portfolio
type PortfolioMetadata struct {
	LastCalculated       time.Time `bson:"last_calculated" json:"last_calculated"`
	LastOrderProcessed   time.Time `bson:"last_order_processed" json:"last_order_processed"`
	CalculationVersion   string    `bson:"calculation_version" json:"calculation_version"`
	NeedsRecalculation   bool      `bson:"needs_recalculation" json:"needs_recalculation"`
	CalculationDuration  int64     `bson:"calculation_duration,omitempty" json:"calculation_duration,omitempty"` // milliseconds
	LastSnapshotDate     time.Time `bson:"last_snapshot_date,omitempty" json:"last_snapshot_date,omitempty"`
}

// BenchmarkComparison represents comparison with market benchmarks
type BenchmarkComparison struct {
	BenchmarkSymbol        string          `bson:"benchmark_symbol" json:"benchmark_symbol"`
	PortfolioReturn        decimal.Decimal `bson:"portfolio_return" json:"portfolio_return"`
	BenchmarkReturn        decimal.Decimal `bson:"benchmark_return" json:"benchmark_return"`
	Alpha                  decimal.Decimal `bson:"alpha" json:"alpha"`
	TrackingError          decimal.Decimal `bson:"tracking_error" json:"tracking_error"`
	InformationRatio       decimal.Decimal `bson:"information_ratio" json:"information_ratio"`
	Outperformance         decimal.Decimal `bson:"outperformance" json:"outperformance"`
	CorrelationCoefficient decimal.Decimal `bson:"correlation_coefficient" json:"correlation_coefficient"`
}

// PortfolioAllocation represents asset allocation breakdown
type PortfolioAllocation struct {
	Crypto            decimal.Decimal `json:"crypto"`
	Cash              decimal.Decimal `json:"cash"`
	CryptoPercentage  decimal.Decimal `json:"crypto_percentage"`
	CashPercentage    decimal.Decimal `json:"cash_percentage"`
}

// TransactionSummary represents transaction summary for a holding
type TransactionSummary struct {
	Total         int       `json:"total"`
	Buys          int       `json:"buys"`
	Sells         int       `json:"sells"`
	FirstPurchase time.Time `json:"first_purchase"`
	LastActivity  time.Time `json:"last_activity"`
}

// PortfolioSummary represents a summary view of the portfolio
type PortfolioSummary struct {
	UserID                int64                `json:"user_id"`
	TotalValue            decimal.Decimal      `json:"total_value"`
	TotalInvested         decimal.Decimal      `json:"total_invested"`
	TotalCash             decimal.Decimal      `json:"total_cash"`
	ProfitLoss            decimal.Decimal      `json:"profit_loss"`
	ProfitLossPercentage  decimal.Decimal      `json:"profit_loss_percentage"`
	Currency              string               `json:"currency"`
	Allocation            PortfolioAllocation  `json:"allocation"`
	LastUpdated           time.Time            `json:"last_updated"`
}

// Holdings returns non-zero holdings
func (p *Portfolio) GetNonZeroHoldings() []Holding {
	var nonZeroHoldings []Holding
	for _, holding := range p.Holdings {
		if holding.Quantity.GreaterThan(decimal.Zero) {
			nonZeroHoldings = append(nonZeroHoldings, holding)
		}
	}
	return nonZeroHoldings
}

// GetHoldingBySymbol returns a holding by symbol
func (p *Portfolio) GetHoldingBySymbol(symbol string) (*Holding, bool) {
	for i, holding := range p.Holdings {
		if holding.Symbol == symbol {
			return &p.Holdings[i], true
		}
	}
	return nil, false
}

// AddOrUpdateHolding adds a new holding or updates existing one
func (p *Portfolio) AddOrUpdateHolding(holding Holding) {
	for i, existingHolding := range p.Holdings {
		if existingHolding.Symbol == holding.Symbol {
			p.Holdings[i] = holding
			return
		}
	}
	p.Holdings = append(p.Holdings, holding)
}

// RemoveHolding removes a holding by symbol
func (p *Portfolio) RemoveHolding(symbol string) bool {
	for i, holding := range p.Holdings {
		if holding.Symbol == symbol {
			p.Holdings = append(p.Holdings[:i], p.Holdings[i+1:]...)
			return true
		}
	}
	return false
}

// GetTotalCryptoValue returns total value of crypto holdings
func (p *Portfolio) GetTotalCryptoValue() decimal.Decimal {
	total := decimal.Zero
	for _, holding := range p.Holdings {
		total = total.Add(holding.CurrentValue)
	}
	return total
}

// GetAllocation returns portfolio allocation breakdown
func (p *Portfolio) GetAllocation() PortfolioAllocation {
	cryptoValue := p.GetTotalCryptoValue()
	totalValue := cryptoValue.Add(p.TotalCash)

	allocation := PortfolioAllocation{
		Crypto: cryptoValue,
		Cash:   p.TotalCash,
	}

	if totalValue.GreaterThan(decimal.Zero) {
		allocation.CryptoPercentage = cryptoValue.Div(totalValue).Mul(decimal.NewFromInt(100))
		allocation.CashPercentage = p.TotalCash.Div(totalValue).Mul(decimal.NewFromInt(100))
	}

	return allocation
}

// GetSummary returns a portfolio summary
func (p *Portfolio) GetSummary() PortfolioSummary {
	return PortfolioSummary{
		UserID:               p.UserID,
		TotalValue:           p.TotalValue,
		TotalInvested:        p.TotalInvested,
		TotalCash:            p.TotalCash,
		ProfitLoss:           p.ProfitLoss,
		ProfitLossPercentage: p.ProfitLossPercentage,
		Currency:             p.Currency,
		Allocation:           p.GetAllocation(),
		LastUpdated:          p.UpdatedAt,
	}
}

// MarkForRecalculation marks the portfolio as needing recalculation
func (p *Portfolio) MarkForRecalculation() {
	p.Metadata.NeedsRecalculation = true
	p.UpdatedAt = time.Now()
}

// MarkCalculated marks the portfolio as calculated
func (p *Portfolio) MarkCalculated(duration int64) {
	p.Metadata.LastCalculated = time.Now()
	p.Metadata.NeedsRecalculation = false
	p.Metadata.CalculationDuration = duration
	p.UpdatedAt = time.Now()
}

// IsStale checks if portfolio data is stale and needs recalculation
func (p *Portfolio) IsStale(maxAge time.Duration) bool {
	return p.Metadata.NeedsRecalculation ||
		   time.Since(p.Metadata.LastCalculated) > maxAge
}

// Validate validates the portfolio data
func (p *Portfolio) Validate() error {
	if p.UserID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	if p.Currency == "" {
		return fmt.Errorf("currency is required")
	}

	// Validate holdings
	for i, holding := range p.Holdings {
		if holding.Symbol == "" {
			return fmt.Errorf("holding %d: symbol is required", i)
		}

		if holding.Quantity.LessThan(decimal.Zero) {
			return fmt.Errorf("holding %s: quantity cannot be negative", holding.Symbol)
		}

		if holding.AverageBuyPrice.LessThan(decimal.Zero) {
			return fmt.Errorf("holding %s: average buy price cannot be negative", holding.Symbol)
		}
	}

	return nil
}