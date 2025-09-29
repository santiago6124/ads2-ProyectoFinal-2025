package calculator

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"

	"portfolio-api/internal/models"
)

type ROICalculator struct{}

func NewROICalculator() *ROICalculator {
	return &ROICalculator{}
}

type ROIMetrics struct {
	SimpleROI                decimal.Decimal `json:"simple_roi"`
	AnnualizedROI            decimal.Decimal `json:"annualized_roi"`
	MoneyWeightedReturn      decimal.Decimal `json:"money_weighted_return"`
	TimeWeightedReturn       decimal.Decimal `json:"time_weighted_return"`
	CompoundAnnualGrowthRate decimal.Decimal `json:"compound_annual_growth_rate"`
	HoldingPeriodReturn      decimal.Decimal `json:"holding_period_return"`
	RealizedROI              decimal.Decimal `json:"realized_roi"`
	UnrealizedROI            decimal.Decimal `json:"unrealized_roi"`
	TotalROI                 decimal.Decimal `json:"total_roi"`
}

type PeriodROI struct {
	Period        string          `json:"period"`
	StartDate     time.Time       `json:"start_date"`
	EndDate       time.Time       `json:"end_date"`
	StartValue    decimal.Decimal `json:"start_value"`
	EndValue      decimal.Decimal `json:"end_value"`
	ROI           decimal.Decimal `json:"roi"`
	AnnualizedROI decimal.Decimal `json:"annualized_roi"`
	NetCashFlow   decimal.Decimal `json:"net_cash_flow"`
}

type HoldingROI struct {
	Symbol               string          `json:"symbol"`
	SimpleROI            decimal.Decimal `json:"simple_roi"`
	AnnualizedROI        decimal.Decimal `json:"annualized_roi"`
	HoldingPeriodReturn  decimal.Decimal `json:"holding_period_return"`
	RealizedGains        decimal.Decimal `json:"realized_gains"`
	UnrealizedGains      decimal.Decimal `json:"unrealized_gains"`
	TotalInvested        decimal.Decimal `json:"total_invested"`
	CurrentValue         decimal.Decimal `json:"current_value"`
	DividendsReceived    decimal.Decimal `json:"dividends_received"`
	FirstPurchaseDate    time.Time       `json:"first_purchase_date"`
	AverageHoldingPeriod time.Duration   `json:"average_holding_period"`
}

func (roi *ROICalculator) CalculatePortfolioROI(ctx context.Context, portfolio *models.Portfolio, snapshots []models.Snapshot) (*ROIMetrics, error) {
	if portfolio == nil {
		return nil, fmt.Errorf("portfolio is required")
	}

	metrics := &ROIMetrics{}

	// Calculate simple ROI
	if !portfolio.TotalInvested.IsZero() {
		metrics.SimpleROI = portfolio.ProfitLoss.Div(portfolio.TotalInvested)
	}

	// Calculate total ROI (including unrealized gains)
	totalInvestment := portfolio.TotalInvested
	totalValue := portfolio.TotalValue
	if !totalInvestment.IsZero() {
		metrics.TotalROI = totalValue.Sub(totalInvestment).Div(totalInvestment)
	}

	// Calculate realized vs unrealized ROI
	realizedGains := roi.calculateRealizedGains(portfolio)
	unrealizedGains := portfolio.ProfitLoss.Sub(realizedGains)

	if !totalInvestment.IsZero() {
		metrics.RealizedROI = realizedGains.Div(totalInvestment)
		metrics.UnrealizedROI = unrealizedGains.Div(totalInvestment)
	}

	// Calculate time-weighted metrics if snapshots are available
	if len(snapshots) >= 2 {
		var err error
		metrics.TimeWeightedReturn, err = roi.calculateTimeWeightedReturn(snapshots)
		if err != nil {
			metrics.TimeWeightedReturn = decimal.Zero
		}

		metrics.CompoundAnnualGrowthRate, err = roi.calculateCAGR(snapshots)
		if err != nil {
			metrics.CompoundAnnualGrowthRate = decimal.Zero
		}

		metrics.AnnualizedROI, err = roi.calculateAnnualizedROI(snapshots)
		if err != nil {
			metrics.AnnualizedROI = decimal.Zero
		}

		metrics.HoldingPeriodReturn, err = roi.calculateHoldingPeriodReturn(snapshots)
		if err != nil {
			metrics.HoldingPeriodReturn = decimal.Zero
		}
	}

	return metrics, nil
}

func (roi *ROICalculator) CalculateHoldingROI(ctx context.Context, holding *models.Holding, transactions []models.Transaction) (*HoldingROI, error) {
	if holding == nil {
		return nil, fmt.Errorf("holding is required")
	}

	holdingROI := &HoldingROI{
		Symbol:       holding.Symbol,
		CurrentValue: holding.CurrentValue,
	}

	// Calculate total invested and realized gains from transactions
	totalInvested := decimal.Zero
	realizedGains := decimal.Zero
	dividendsReceived := decimal.Zero
	var firstPurchase time.Time
	var lastPurchase time.Time
	purchaseCount := 0

	for _, tx := range transactions {
		if tx.Symbol != holding.Symbol {
			continue
		}

		switch tx.Type {
		case "buy":
			totalInvested = totalInvested.Add(tx.Amount)
			if firstPurchase.IsZero() || tx.Date.Before(firstPurchase) {
				firstPurchase = tx.Date
			}
			if tx.Date.After(lastPurchase) {
				lastPurchase = tx.Date
			}
			purchaseCount++

		case "sell":
			// Calculate realized gain/loss for this sale
			avgCostBasis := totalInvested.Div(holding.Quantity.Add(tx.Quantity))
			saleGainLoss := tx.Amount.Sub(avgCostBasis.Mul(tx.Quantity))
			realizedGains = realizedGains.Add(saleGainLoss)

		case "dividend":
			dividendsReceived = dividendsReceived.Add(tx.Amount)
		}
	}

	holdingROI.TotalInvested = totalInvested
	holdingROI.RealizedGains = realizedGains
	holdingROI.DividendsReceived = dividendsReceived
	holdingROI.FirstPurchaseDate = firstPurchase

	// Calculate unrealized gains
	if !totalInvested.IsZero() {
		currentInvestmentValue := holding.CurrentValue
		costBasis := totalInvested
		holdingROI.UnrealizedGains = currentInvestmentValue.Sub(costBasis)
	}

	// Calculate simple ROI
	if !totalInvested.IsZero() {
		totalGains := realizedGains.Add(holdingROI.UnrealizedGains).Add(dividendsReceived)
		holdingROI.SimpleROI = totalGains.Div(totalInvested)
	}

	// Calculate annualized ROI
	if !firstPurchase.IsZero() && !totalInvested.IsZero() {
		holdingPeriod := time.Since(firstPurchase)
		if holdingPeriod > 0 {
			holdingROI.AverageHoldingPeriod = holdingPeriod / time.Duration(purchaseCount)

			years := decimal.NewFromFloat(holdingPeriod.Hours() / (24 * 365.25))
			if years.GreaterThan(decimal.Zero) {
				// Annualized ROI = (Current Value / Total Invested)^(1/years) - 1
				currentTotal := holding.CurrentValue.Add(realizedGains).Add(dividendsReceived)
				if currentTotal.GreaterThan(decimal.Zero) && totalInvested.GreaterThan(decimal.Zero) {
					ratio := currentTotal.Div(totalInvested)
					ratioFloat, _ := ratio.Float64()
					yearsFloat, _ := years.Float64()

					if ratioFloat > 0 && yearsFloat > 0 {
						annualizedFloat := math.Pow(ratioFloat, 1.0/yearsFloat) - 1
						holdingROI.AnnualizedROI = decimal.NewFromFloat(annualizedFloat)
					}
				}
			}
		}
	}

	// Calculate holding period return
	if !totalInvested.IsZero() {
		currentTotal := holding.CurrentValue.Add(realizedGains).Add(dividendsReceived)
		holdingROI.HoldingPeriodReturn = currentTotal.Sub(totalInvested).Div(totalInvested)
	}

	return holdingROI, nil
}

func (roi *ROICalculator) CalculatePeriodROI(ctx context.Context, snapshots []models.Snapshot, period string) ([]PeriodROI, error) {
	if len(snapshots) < 2 {
		return nil, fmt.Errorf("insufficient snapshots for period calculation")
	}

	periods := roi.groupSnapshotsByPeriod(snapshots, period)
	results := make([]PeriodROI, 0, len(periods))

	for _, periodSnapshots := range periods {
		if len(periodSnapshots) < 2 {
			continue
		}

		startSnapshot := periodSnapshots[0]
		endSnapshot := periodSnapshots[len(periodSnapshots)-1]

		periodROI := PeriodROI{
			Period:     period,
			StartDate:  startSnapshot.Timestamp,
			EndDate:    endSnapshot.Timestamp,
			StartValue: startSnapshot.Value.Total,
			EndValue:   endSnapshot.Value.Total,
		}

		// Calculate net cash flow during period
		netCashFlow := decimal.Zero
		for i := 1; i < len(periodSnapshots); i++ {
			// This would need to be enhanced with actual transaction data
			// For now, we'll estimate based on value changes that exceed market movements
			valueChange := periodSnapshots[i].Value.Total.Sub(periodSnapshots[i-1].Value.Total)
			marketChange := periodSnapshots[i-1].Value.Total.Mul(periodSnapshots[i].Value.DailyChangePercent)

			if valueChange.Sub(marketChange).Abs().GreaterThan(marketChange.Abs().Mul(decimal.NewFromFloat(0.1))) {
				netCashFlow = netCashFlow.Add(valueChange.Sub(marketChange))
			}
		}

		periodROI.NetCashFlow = netCashFlow

		// Calculate simple ROI for the period
		if !startSnapshot.Value.Total.IsZero() {
			periodROI.ROI = endSnapshot.Value.Total.Sub(startSnapshot.Value.Total).Sub(netCashFlow).
				Div(startSnapshot.Value.Total)
		}

		// Calculate annualized ROI
		duration := endSnapshot.Timestamp.Sub(startSnapshot.Timestamp)
		if duration > 0 {
			years := decimal.NewFromFloat(duration.Hours() / (24 * 365.25))
			if years.GreaterThan(decimal.Zero) && !periodROI.ROI.IsZero() {
				roiPlusOne := periodROI.ROI.Add(decimal.NewFromInt(1))
				roiFloat, _ := roiPlusOne.Float64()
				yearsFloat, _ := years.Float64()

				if roiFloat > 0 && yearsFloat > 0 {
					annualizedFloat := math.Pow(roiFloat, 1.0/yearsFloat) - 1
					periodROI.AnnualizedROI = decimal.NewFromFloat(annualizedFloat)
				}
			}
		}

		results = append(results, periodROI)
	}

	return results, nil
}

func (roi *ROICalculator) calculateRealizedGains(portfolio *models.Portfolio) decimal.Decimal {
	// This would be calculated from actual transaction history
	// For now, we'll estimate based on portfolio metadata
	if portfolio.Metadata != nil {
		if realizedGains, exists := portfolio.Metadata["realized_gains"]; exists {
			if gains, ok := realizedGains.(string); ok {
				if decimal, err := decimal.NewFromString(gains); err == nil {
					return decimal
				}
			}
		}
	}

	// Default estimation: assume 20% of profit/loss is realized
	return portfolio.ProfitLoss.Mul(decimal.NewFromFloat(0.2))
}

func (roi *ROICalculator) calculateTimeWeightedReturn(snapshots []models.Snapshot) (decimal.Decimal, error) {
	if len(snapshots) < 2 {
		return decimal.Zero, fmt.Errorf("insufficient snapshots")
	}

	cumulativeReturn := decimal.NewFromInt(1)

	for i := 1; i < len(snapshots); i++ {
		prevValue := snapshots[i-1].Value.Total
		currentValue := snapshots[i].Value.Total

		if prevValue.IsZero() {
			continue
		}

		periodReturn := currentValue.Div(prevValue)
		cumulativeReturn = cumulativeReturn.Mul(periodReturn)
	}

	return cumulativeReturn.Sub(decimal.NewFromInt(1)), nil
}

func (roi *ROICalculator) calculateCAGR(snapshots []models.Snapshot) (decimal.Decimal, error) {
	if len(snapshots) < 2 {
		return decimal.Zero, fmt.Errorf("insufficient snapshots")
	}

	startSnapshot := snapshots[0]
	endSnapshot := snapshots[len(snapshots)-1]

	if startSnapshot.Value.Total.IsZero() {
		return decimal.Zero, fmt.Errorf("start value cannot be zero")
	}

	duration := endSnapshot.Timestamp.Sub(startSnapshot.Timestamp)
	years := decimal.NewFromFloat(duration.Hours() / (24 * 365.25))

	if years.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, fmt.Errorf("invalid time period")
	}

	// CAGR = (End Value / Start Value)^(1/years) - 1
	ratio := endSnapshot.Value.Total.Div(startSnapshot.Value.Total)
	ratioFloat, _ := ratio.Float64()
	yearsFloat, _ := years.Float64()

	if ratioFloat <= 0 || yearsFloat <= 0 {
		return decimal.Zero, nil
	}

	cagrFloat := math.Pow(ratioFloat, 1.0/yearsFloat) - 1
	return decimal.NewFromFloat(cagrFloat), nil
}

func (roi *ROICalculator) calculateAnnualizedROI(snapshots []models.Snapshot) (decimal.Decimal, error) {
	if len(snapshots) < 2 {
		return decimal.Zero, fmt.Errorf("insufficient snapshots")
	}

	startSnapshot := snapshots[0]
	endSnapshot := snapshots[len(snapshots)-1]

	duration := endSnapshot.Timestamp.Sub(startSnapshot.Timestamp)
	years := decimal.NewFromFloat(duration.Hours() / (24 * 365.25))

	if years.LessThanOrEqual(decimal.Zero) || startSnapshot.Value.Total.IsZero() {
		return decimal.Zero, fmt.Errorf("invalid time period or start value")
	}

	totalReturn := endSnapshot.Value.Total.Sub(startSnapshot.Value.Total).Div(startSnapshot.Value.Total)

	// Annualized ROI = (1 + Total Return)^(1/years) - 1
	totalReturnPlusOne := totalReturn.Add(decimal.NewFromInt(1))
	totalReturnFloat, _ := totalReturnPlusOne.Float64()
	yearsFloat, _ := years.Float64()

	if totalReturnFloat <= 0 || yearsFloat <= 0 {
		return decimal.Zero, nil
	}

	annualizedFloat := math.Pow(totalReturnFloat, 1.0/yearsFloat) - 1
	return decimal.NewFromFloat(annualizedFloat), nil
}

func (roi *ROICalculator) calculateHoldingPeriodReturn(snapshots []models.Snapshot) (decimal.Decimal, error) {
	if len(snapshots) < 2 {
		return decimal.Zero, fmt.Errorf("insufficient snapshots")
	}

	startSnapshot := snapshots[0]
	endSnapshot := snapshots[len(snapshots)-1]

	if startSnapshot.Value.Total.IsZero() {
		return decimal.Zero, fmt.Errorf("start value cannot be zero")
	}

	// HPR = (End Value - Start Value) / Start Value
	return endSnapshot.Value.Total.Sub(startSnapshot.Value.Total).Div(startSnapshot.Value.Total), nil
}

func (roi *ROICalculator) groupSnapshotsByPeriod(snapshots []models.Snapshot, period string) [][]models.Snapshot {
	if len(snapshots) == 0 {
		return nil
	}

	groups := make([][]models.Snapshot, 0)
	currentGroup := make([]models.Snapshot, 0)

	var lastPeriodKey string

	for _, snapshot := range snapshots {
		var periodKey string

		switch period {
		case "daily":
			periodKey = snapshot.Timestamp.Format("2006-01-02")
		case "weekly":
			year, week := snapshot.Timestamp.ISOWeek()
			periodKey = fmt.Sprintf("%d-W%d", year, week)
		case "monthly":
			periodKey = snapshot.Timestamp.Format("2006-01")
		case "yearly":
			periodKey = snapshot.Timestamp.Format("2006")
		default:
			periodKey = snapshot.Timestamp.Format("2006-01-02")
		}

		if lastPeriodKey == "" {
			lastPeriodKey = periodKey
		}

		if periodKey != lastPeriodKey {
			if len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
				currentGroup = make([]models.Snapshot, 0)
			}
			lastPeriodKey = periodKey
		}

		currentGroup = append(currentGroup, snapshot)
	}

	// Add the last group
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

type ComparisonROI struct {
	PortfolioROI  decimal.Decimal `json:"portfolio_roi"`
	BenchmarkROI  decimal.Decimal `json:"benchmark_roi"`
	Outperformance decimal.Decimal `json:"outperformance"`
	TrackingError decimal.Decimal `json:"tracking_error"`
}

func (roi *ROICalculator) CalculateBenchmarkComparison(ctx context.Context, portfolioSnapshots []models.Snapshot, benchmarkPrices []decimal.Decimal) (*ComparisonROI, error) {
	if len(portfolioSnapshots) != len(benchmarkPrices) || len(portfolioSnapshots) < 2 {
		return nil, fmt.Errorf("mismatched data lengths or insufficient data")
	}

	// Calculate portfolio ROI
	startValue := portfolioSnapshots[0].Value.Total
	endValue := portfolioSnapshots[len(portfolioSnapshots)-1].Value.Total

	var portfolioROI decimal.Decimal
	if !startValue.IsZero() {
		portfolioROI = endValue.Sub(startValue).Div(startValue)
	}

	// Calculate benchmark ROI
	startBenchmark := benchmarkPrices[0]
	endBenchmark := benchmarkPrices[len(benchmarkPrices)-1]

	var benchmarkROI decimal.Decimal
	if !startBenchmark.IsZero() {
		benchmarkROI = endBenchmark.Sub(startBenchmark).Div(startBenchmark)
	}

	// Calculate outperformance
	outperformance := portfolioROI.Sub(benchmarkROI)

	// Calculate tracking error
	portfolioReturns := make([]decimal.Decimal, 0, len(portfolioSnapshots)-1)
	benchmarkReturns := make([]decimal.Decimal, 0, len(benchmarkPrices)-1)

	for i := 1; i < len(portfolioSnapshots); i++ {
		// Portfolio return
		prevPortfolio := portfolioSnapshots[i-1].Value.Total
		currentPortfolio := portfolioSnapshots[i].Value.Total
		if !prevPortfolio.IsZero() {
			portfolioReturns = append(portfolioReturns, currentPortfolio.Sub(prevPortfolio).Div(prevPortfolio))
		}

		// Benchmark return
		prevBenchmark := benchmarkPrices[i-1]
		currentBenchmark := benchmarkPrices[i]
		if !prevBenchmark.IsZero() {
			benchmarkReturns = append(benchmarkReturns, currentBenchmark.Sub(prevBenchmark).Div(prevBenchmark))
		}
	}

	trackingError := roi.calculateTrackingError(portfolioReturns, benchmarkReturns)

	return &ComparisonROI{
		PortfolioROI:   portfolioROI,
		BenchmarkROI:   benchmarkROI,
		Outperformance: outperformance,
		TrackingError:  trackingError,
	}, nil
}

func (roi *ROICalculator) calculateTrackingError(portfolioReturns, benchmarkReturns []decimal.Decimal) decimal.Decimal {
	if len(portfolioReturns) != len(benchmarkReturns) || len(portfolioReturns) < 2 {
		return decimal.Zero
	}

	// Calculate tracking differences
	trackingDiffs := make([]decimal.Decimal, len(portfolioReturns))
	for i := 0; i < len(portfolioReturns); i++ {
		trackingDiffs[i] = portfolioReturns[i].Sub(benchmarkReturns[i])
	}

	// Calculate standard deviation of tracking differences
	sum := decimal.Zero
	for _, diff := range trackingDiffs {
		sum = sum.Add(diff)
	}
	mean := sum.Div(decimal.NewFromInt(int64(len(trackingDiffs))))

	variance := decimal.Zero
	for _, diff := range trackingDiffs {
		devFromMean := diff.Sub(mean)
		variance = variance.Add(devFromMean.Mul(devFromMean))
	}
	variance = variance.Div(decimal.NewFromInt(int64(len(trackingDiffs) - 1)))

	varianceFloat, _ := variance.Float64()
	if varianceFloat <= 0 {
		return decimal.Zero
	}

	// Annualize the tracking error
	stdDev := decimal.NewFromFloat(math.Sqrt(varianceFloat))
	annualizedTrackingError := stdDev.Mul(decimal.NewFromFloat(math.Sqrt(252))) // 252 trading days

	return annualizedTrackingError
}