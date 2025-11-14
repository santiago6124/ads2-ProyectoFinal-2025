package models

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Snapshot represents a historical snapshot of a portfolio
type Snapshot struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id,omitempty"`
	PortfolioID  primitive.ObjectID  `bson:"portfolio_id" json:"portfolio_id"`
	UserID       int64               `bson:"user_id" json:"user_id"`
	Timestamp    time.Time           `bson:"timestamp" json:"timestamp"`
	Interval     string              `bson:"interval" json:"interval"` // "hourly", "daily", "weekly", "monthly", "manual"

	Value        SnapshotValue       `bson:"value" json:"value"`
	Holdings     []HoldingSnapshot   `bson:"holdings_snapshot" json:"holdings_snapshot"`
	Metrics      SnapshotMetrics     `bson:"metrics" json:"metrics"`
	Comparison   MarketComparison    `bson:"market_comparison" json:"market_comparison"`

	// Optional fields for manual snapshots
	Note         string              `bson:"note,omitempty" json:"note,omitempty"`
	Tags         []string            `bson:"tags,omitempty" json:"tags,omitempty"`

	CreatedAt    time.Time           `bson:"created_at" json:"created_at"`
}

// SnapshotValue represents the value information in a snapshot
type SnapshotValue struct {
	Total                decimal.Decimal `bson:"total" json:"total"`
	Invested             decimal.Decimal `bson:"invested" json:"invested"`
	Cash                 decimal.Decimal `bson:"cash" json:"cash"`
	ProfitLoss           decimal.Decimal `bson:"profit_loss" json:"profit_loss"`
	ProfitLossPercentage decimal.Decimal `bson:"profit_loss_percentage" json:"profit_loss_percentage"`
	DailyChange          decimal.Decimal `bson:"daily_change,omitempty" json:"daily_change,omitempty"`
	DailyChangePercent   decimal.Decimal `bson:"daily_change_percentage,omitempty" json:"daily_change_percentage,omitempty"`
}

// HoldingSnapshot represents a holding at a specific point in time
type HoldingSnapshot struct {
	Symbol      string          `bson:"symbol" json:"symbol"`
	Name        string          `bson:"name,omitempty" json:"name,omitempty"`
	Quantity    decimal.Decimal `bson:"quantity" json:"quantity"`
	Price       decimal.Decimal `bson:"price" json:"price"`
	Value       decimal.Decimal `bson:"value" json:"value"`
	Percentage  decimal.Decimal `bson:"percentage" json:"percentage"`
	ProfitLoss  decimal.Decimal `bson:"profit_loss,omitempty" json:"profit_loss,omitempty"`
	ProfitLossPercentage decimal.Decimal `bson:"profit_loss_percentage,omitempty" json:"profit_loss_percentage,omitempty"`
	Category    string          `bson:"category,omitempty" json:"category,omitempty"`
}

// SnapshotMetrics represents key metrics at snapshot time
type SnapshotMetrics struct {
	Volatility           decimal.Decimal `bson:"volatility" json:"volatility"`
	SharpeRatio          decimal.Decimal `bson:"sharpe_ratio" json:"sharpe_ratio"`
	SortinoRatio         decimal.Decimal `bson:"sortino_ratio,omitempty" json:"sortino_ratio,omitempty"`
	MaxDrawdown          decimal.Decimal `bson:"max_drawdown,omitempty" json:"max_drawdown,omitempty"`
	DiversificationIndex decimal.Decimal `bson:"diversification_index" json:"diversification_index"`
	HoldingsCount        int             `bson:"holdings_count" json:"holdings_count"`
	ConcentrationRisk    decimal.Decimal `bson:"concentration_risk,omitempty" json:"concentration_risk,omitempty"`
}

// MarketComparison represents market comparison metrics
type MarketComparison struct {
	BTCPerformance       decimal.Decimal `bson:"btc_performance" json:"btc_performance"`
	MarketAvgPerformance decimal.Decimal `bson:"market_avg_performance" json:"market_avg_performance"`
	Outperformance       decimal.Decimal `bson:"outperformance" json:"outperformance"`
	Beta                 decimal.Decimal `bson:"beta,omitempty" json:"beta,omitempty"`
	Alpha                decimal.Decimal `bson:"alpha,omitempty" json:"alpha,omitempty"`
}

// HistoryPoint represents a single point in portfolio history
type HistoryPoint struct {
	Date                   string          `json:"date"`
	TotalValue             decimal.Decimal `json:"total_value"`
	ProfitLoss             decimal.Decimal `json:"profit_loss"`
	DailyChange            decimal.Decimal `json:"daily_change"`
	DailyChangePercentage  decimal.Decimal `json:"daily_change_percentage"`
	HoldingsCount          int             `json:"holdings_count,omitempty"`
	Volatility             decimal.Decimal `json:"volatility,omitempty"`
}

// HistorySummary represents summary statistics for a history period
type HistorySummary struct {
	StartValue            decimal.Decimal `json:"start_value"`
	EndValue              decimal.Decimal `json:"end_value"`
	TotalChange           decimal.Decimal `json:"total_change"`
	TotalChangePercentage decimal.Decimal `json:"total_change_percentage"`
	BestDay               HistoryExtreme  `json:"best_day"`
	WorstDay              HistoryExtreme  `json:"worst_day"`
	AverageDaily          decimal.Decimal `json:"average_daily,omitempty"`
	Volatility            decimal.Decimal `json:"volatility,omitempty"`
	MaxDrawdown           decimal.Decimal `json:"max_drawdown,omitempty"`
	MaxDrawdownDays       int             `json:"max_drawdown_days,omitempty"`
}

// HistoryExtreme represents best/worst performance days
type HistoryExtreme struct {
	Date   string          `json:"date"`
	Value  decimal.Decimal `json:"value"`
	Change decimal.Decimal `json:"change"`
	ChangePercentage decimal.Decimal `json:"change_percentage"`
}

// SnapshotFilter represents filters for querying snapshots
type SnapshotFilter struct {
	UserID    int64     `json:"user_id,omitempty"`
	From      time.Time `json:"from,omitempty"`
	To        time.Time `json:"to,omitempty"`
	Interval  string    `json:"interval,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}

// NewSnapshot creates a new snapshot from a portfolio
func NewSnapshot(portfolio *Portfolio, interval string) *Snapshot {
	now := time.Now()

	snapshot := &Snapshot{
		PortfolioID: portfolio.ID,
		UserID:      portfolio.UserID,
		Timestamp:   now,
		Interval:    interval,
		CreatedAt:   now,
	}

	// Copy value information (cash is managed by Users API, not stored in snapshot)
	snapshot.Value = SnapshotValue{
		Total:                portfolio.TotalValue,
		Invested:             portfolio.TotalInvested,
		Cash:                 decimal.Zero,  // Cash managed by Users API
		ProfitLoss:           portfolio.ProfitLoss,
		ProfitLossPercentage: portfolio.ProfitLossPercentage,
		DailyChange:          portfolio.Performance.DailyChange,
		DailyChangePercent:   portfolio.Performance.DailyChangePercentage,
	}

	// Copy holdings
	snapshot.Holdings = make([]HoldingSnapshot, len(portfolio.Holdings))
	for i, holding := range portfolio.Holdings {
		snapshot.Holdings[i] = HoldingSnapshot{
			Symbol:               holding.Symbol,
			Name:                 holding.Name,
			Quantity:             holding.Quantity,
			Price:                holding.CurrentPrice,
			Value:                holding.CurrentValue,
			Percentage:           holding.PercentageOfPortfolio,
			ProfitLoss:           holding.ProfitLoss,
			ProfitLossPercentage: holding.ProfitLossPercentage,
			Category:             holding.Category,
		}
	}

	// Copy metrics
	snapshot.Metrics = SnapshotMetrics{
		Volatility:           portfolio.RiskMetrics.Volatility30d,
		SharpeRatio:          portfolio.RiskMetrics.SharpeRatio,
		SortinoRatio:         portfolio.RiskMetrics.SortinoRatio,
		MaxDrawdown:          portfolio.RiskMetrics.MaxDrawdown,
		DiversificationIndex: portfolio.Diversification.ConcentrationIndex,
		HoldingsCount:        portfolio.Diversification.HoldingsCount,
		ConcentrationRisk:    portfolio.Diversification.LargestPositionPercentage,
	}

	return snapshot
}

// NewManualSnapshot creates a manual snapshot with note and tags
func NewManualSnapshot(portfolio *Portfolio, note string, tags []string) *Snapshot {
	snapshot := NewSnapshot(portfolio, "manual")
	snapshot.Note = note
	snapshot.Tags = tags
	return snapshot
}

// ToHistoryPoint converts a snapshot to a history point
func (s *Snapshot) ToHistoryPoint() HistoryPoint {
	return HistoryPoint{
		Date:                  s.Timestamp.Format("2006-01-02"),
		TotalValue:            s.Value.Total,
		ProfitLoss:            s.Value.ProfitLoss,
		DailyChange:           s.Value.DailyChange,
		DailyChangePercentage: s.Value.DailyChangePercent,
		HoldingsCount:         s.Metrics.HoldingsCount,
		Volatility:            s.Metrics.Volatility,
	}
}

// GetHoldingBySymbol returns a holding snapshot by symbol
func (s *Snapshot) GetHoldingBySymbol(symbol string) (*HoldingSnapshot, bool) {
	for i, holding := range s.Holdings {
		if holding.Symbol == symbol {
			return &s.Holdings[i], true
		}
	}
	return nil, false
}

// GetTopHoldings returns top N holdings by value
func (s *Snapshot) GetTopHoldings(n int) []HoldingSnapshot {
	if n <= 0 || n > len(s.Holdings) {
		n = len(s.Holdings)
	}

	// Create a copy and sort by value
	holdings := make([]HoldingSnapshot, len(s.Holdings))
	copy(holdings, s.Holdings)

	// Sort by value descending
	for i := 0; i < len(holdings)-1; i++ {
		for j := i + 1; j < len(holdings); j++ {
			if holdings[j].Value.GreaterThan(holdings[i].Value) {
				holdings[i], holdings[j] = holdings[j], holdings[i]
			}
		}
	}

	return holdings[:n]
}

// GetHoldingsValue returns total value of holdings
func (s *Snapshot) GetHoldingsValue() decimal.Decimal {
	total := decimal.Zero
	for _, holding := range s.Holdings {
		total = total.Add(holding.Value)
	}
	return total
}

// IsExpired checks if snapshot is older than given duration
func (s *Snapshot) IsExpired(maxAge time.Duration) bool {
	return time.Since(s.Timestamp) > maxAge
}

// AddTag adds a tag to the snapshot
func (s *Snapshot) AddTag(tag string) {
	for _, existingTag := range s.Tags {
		if existingTag == tag {
			return // Tag already exists
		}
	}
	s.Tags = append(s.Tags, tag)
}

// RemoveTag removes a tag from the snapshot
func (s *Snapshot) RemoveTag(tag string) {
	for i, existingTag := range s.Tags {
		if existingTag == tag {
			s.Tags = append(s.Tags[:i], s.Tags[i+1:]...)
			return
		}
	}
}

// HasTag checks if snapshot has a specific tag
func (s *Snapshot) HasTag(tag string) bool {
	for _, existingTag := range s.Tags {
		if existingTag == tag {
			return true
		}
	}
	return false
}

// Validate validates the snapshot data
func (s *Snapshot) Validate() error {
	if s.UserID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	if s.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	validIntervals := map[string]bool{
		"hourly":  true,
		"daily":   true,
		"weekly":  true,
		"monthly": true,
		"manual":  true,
	}

	if !validIntervals[s.Interval] {
		return fmt.Errorf("invalid interval: %s", s.Interval)
	}

	// Validate holdings
	for i, holding := range s.Holdings {
		if holding.Symbol == "" {
			return fmt.Errorf("holding %d: symbol is required", i)
		}

		if holding.Quantity.LessThan(decimal.Zero) {
			return fmt.Errorf("holding %s: quantity cannot be negative", holding.Symbol)
		}

		if holding.Price.LessThan(decimal.Zero) {
			return fmt.Errorf("holding %s: price cannot be negative", holding.Symbol)
		}
	}

	return nil
}