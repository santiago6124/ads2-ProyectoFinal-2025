package analytics

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"portfolio-api/internal/models"
)

type PortfolioAnalyzer struct {
	correlationAnalyzer *CorrelationAnalyzer
	portfolioOptimizer  *PortfolioOptimizer
}

func NewPortfolioAnalyzer(riskFreeRate decimal.Decimal) *PortfolioAnalyzer {
	return &PortfolioAnalyzer{
		correlationAnalyzer: NewCorrelationAnalyzer(),
		portfolioOptimizer:  NewPortfolioOptimizer(riskFreeRate),
	}
}

type ComprehensiveAnalysis struct {
	Portfolio              *models.Portfolio      `json:"portfolio"`
	PerformanceAnalysis    PerformanceAnalysis    `json:"performance_analysis"`
	RiskAnalysis           RiskAnalysis           `json:"risk_analysis"`
	DiversificationAnalysis DiversificationAnalysis `json:"diversification_analysis"`
	TrendAnalysis          TrendAnalysis          `json:"trend_analysis"`
	BenchmarkComparison    BenchmarkComparison    `json:"benchmark_comparison"`
	Recommendations        []Recommendation       `json:"recommendations"`
	OverallScore           OverallScore           `json:"overall_score"`
	LastUpdated            time.Time              `json:"last_updated"`
}

type PerformanceAnalysis struct {
	Returns            PeriodReturns       `json:"returns"`
	Consistency        ConsistencyMetrics  `json:"consistency"`
	WinLossRatio       WinLossRatio        `json:"win_loss_ratio"`
	DrawdownAnalysis   DrawdownAnalysis    `json:"drawdown_analysis"`
	PerformanceRanking PerformanceRanking  `json:"performance_ranking"`
}

type PeriodReturns struct {
	Daily      decimal.Decimal `json:"daily"`
	Weekly     decimal.Decimal `json:"weekly"`
	Monthly    decimal.Decimal `json:"monthly"`
	Quarterly  decimal.Decimal `json:"quarterly"`
	YearToDate decimal.Decimal `json:"year_to_date"`
	OneYear    decimal.Decimal `json:"one_year"`
	ThreeYear  decimal.Decimal `json:"three_year"`
	FiveYear   decimal.Decimal `json:"five_year"`
	Inception  decimal.Decimal `json:"inception"`
}

type ConsistencyMetrics struct {
	ConsistencyRatio   decimal.Decimal `json:"consistency_ratio"`
	PositiveMonths     int             `json:"positive_months"`
	NegativeMonths     int             `json:"negative_months"`
	LongestWinStreak   int             `json:"longest_win_streak"`
	LongestLossStreak  int             `json:"longest_loss_streak"`
	AverageWinStreak   decimal.Decimal `json:"average_win_streak"`
	AverageLossStreak  decimal.Decimal `json:"average_loss_streak"`
}

type WinLossRatio struct {
	WinRate           decimal.Decimal `json:"win_rate"`
	LossRate          decimal.Decimal `json:"loss_rate"`
	AverageWin        decimal.Decimal `json:"average_win"`
	AverageLoss       decimal.Decimal `json:"average_loss"`
	WinLossRatio      decimal.Decimal `json:"win_loss_ratio"`
	ProfitFactor      decimal.Decimal `json:"profit_factor"`
	ExpectedValue     decimal.Decimal `json:"expected_value"`
}

type DrawdownAnalysis struct {
	CurrentDrawdown    decimal.Decimal    `json:"current_drawdown"`
	MaxDrawdown        decimal.Decimal    `json:"max_drawdown"`
	AverageDrawdown    decimal.Decimal    `json:"average_drawdown"`
	DrawdownFrequency  decimal.Decimal    `json:"drawdown_frequency"`
	RecoveryTime       RecoveryMetrics    `json:"recovery_time"`
	DrawdownPeriods    []DrawdownPeriod   `json:"drawdown_periods"`
}

type RecoveryMetrics struct {
	AverageRecoveryTime int `json:"average_recovery_time"`
	MaxRecoveryTime     int `json:"max_recovery_time"`
	CurrentRecoveryTime int `json:"current_recovery_time"`
}

type DrawdownPeriod struct {
	StartDate    time.Time       `json:"start_date"`
	EndDate      time.Time       `json:"end_date"`
	RecoveryDate time.Time       `json:"recovery_date"`
	Magnitude    decimal.Decimal `json:"magnitude"`
	Duration     int             `json:"duration"`
	RecoveryTime int             `json:"recovery_time"`
}

type PerformanceRanking struct {
	Percentile        decimal.Decimal `json:"percentile"`
	Rank              int             `json:"rank"`
	TotalPortfolios   int             `json:"total_portfolios"`
	Category          string          `json:"category"`
	BenchmarkOutperformance decimal.Decimal `json:"benchmark_outperformance"`
}

type RiskAnalysis struct {
	RiskMetrics        RiskMetrics         `json:"risk_metrics"`
	RiskProfile        RiskProfile         `json:"risk_profile"`
	ConcentrationRisk  ConcentrationRisk   `json:"concentration_risk"`
	LiquidityRisk      LiquidityRisk       `json:"liquidity_risk"`
	CurrencyRisk       CurrencyRisk        `json:"currency_risk"`
	RiskAttributions   []RiskAttribution   `json:"risk_attributions"`
}

type RiskMetrics struct {
	TotalRisk          decimal.Decimal `json:"total_risk"`
	SystematicRisk     decimal.Decimal `json:"systematic_risk"`
	IdiosyncraticRisk  decimal.Decimal `json:"idiosyncratic_risk"`
	ActiveRisk         decimal.Decimal `json:"active_risk"`
	RiskAdjustedReturn decimal.Decimal `json:"risk_adjusted_return"`
}

type RiskProfile struct {
	RiskCapacity   decimal.Decimal `json:"risk_capacity"`
	RiskTolerance  decimal.Decimal `json:"risk_tolerance"`
	RiskBudget     decimal.Decimal `json:"risk_budget"`
	RiskUtilization decimal.Decimal `json:"risk_utilization"`
	RecommendedRisk string         `json:"recommended_risk"`
}

type ConcentrationRisk struct {
	TopHoldingWeight    decimal.Decimal      `json:"top_holding_weight"`
	Top5HoldingsWeight  decimal.Decimal      `json:"top5_holdings_weight"`
	Top10HoldingsWeight decimal.Decimal      `json:"top10_holdings_weight"`
	HerfindahlIndex     decimal.Decimal      `json:"herfindahl_index"`
	EffectiveAssets     decimal.Decimal      `json:"effective_assets"`
	ConcentrationScore  decimal.Decimal      `json:"concentration_score"`
	SectorConcentration map[string]decimal.Decimal `json:"sector_concentration"`
}

type LiquidityRisk struct {
	OverallLiquidity   decimal.Decimal            `json:"overall_liquidity"`
	DaysToLiquidate    int                        `json:"days_to_liquidate"`
	LiquidityTiers     map[string]decimal.Decimal `json:"liquidity_tiers"`
	IlliquidPercentage decimal.Decimal            `json:"illiquid_percentage"`
}

type CurrencyRisk struct {
	BaseCurrencyExposure decimal.Decimal            `json:"base_currency_exposure"`
	ForeignExposure      decimal.Decimal            `json:"foreign_exposure"`
	CurrencyBreakdown    map[string]decimal.Decimal `json:"currency_breakdown"`
	HedgingRatio         decimal.Decimal            `json:"hedging_ratio"`
}

type RiskAttribution struct {
	Source      string          `json:"source"`
	Contribution decimal.Decimal `json:"contribution"`
	Percentage  decimal.Decimal `json:"percentage"`
}

type DiversificationAnalysis struct {
	DiversificationScore  *DiversificationScore  `json:"diversification_score"`
	CorrelationMatrix     *CorrelationMatrix     `json:"correlation_matrix"`
	AssetClassExposure    AssetClassExposure     `json:"asset_class_exposure"`
	GeographicExposure    GeographicExposure     `json:"geographic_exposure"`
	SectorExposure        SectorExposure         `json:"sector_exposure"`
	StyleExposure         StyleExposure          `json:"style_exposure"`
}

type AssetClassExposure struct {
	Equities      decimal.Decimal `json:"equities"`
	Bonds         decimal.Decimal `json:"bonds"`
	Commodities   decimal.Decimal `json:"commodities"`
	RealEstate    decimal.Decimal `json:"real_estate"`
	Crypto        decimal.Decimal `json:"crypto"`
	Cash          decimal.Decimal `json:"cash"`
	Alternatives  decimal.Decimal `json:"alternatives"`
}

type GeographicExposure struct {
	Domestic       decimal.Decimal `json:"domestic"`
	International  decimal.Decimal `json:"international"`
	EmergingMarkets decimal.Decimal `json:"emerging_markets"`
	RegionBreakdown map[string]decimal.Decimal `json:"region_breakdown"`
}

type SectorExposure struct {
	Technology     decimal.Decimal `json:"technology"`
	Healthcare     decimal.Decimal `json:"healthcare"`
	Financials     decimal.Decimal `json:"financials"`
	ConsumerDiscretionary decimal.Decimal `json:"consumer_discretionary"`
	Communication  decimal.Decimal `json:"communication"`
	Industrials    decimal.Decimal `json:"industrials"`
	Energy         decimal.Decimal `json:"energy"`
	Materials      decimal.Decimal `json:"materials"`
	Utilities      decimal.Decimal `json:"utilities"`
	RealEstate     decimal.Decimal `json:"real_estate"`
	Other          decimal.Decimal `json:"other"`
}

type StyleExposure struct {
	Growth         decimal.Decimal `json:"growth"`
	Value          decimal.Decimal `json:"value"`
	LargeCap       decimal.Decimal `json:"large_cap"`
	MidCap         decimal.Decimal `json:"mid_cap"`
	SmallCap       decimal.Decimal `json:"small_cap"`
	Quality        decimal.Decimal `json:"quality"`
	Momentum       decimal.Decimal `json:"momentum"`
	LowVolatility  decimal.Decimal `json:"low_volatility"`
}

type TrendAnalysis struct {
	ShortTermTrend    TrendDirection   `json:"short_term_trend"`
	MediumTermTrend   TrendDirection   `json:"medium_term_trend"`
	LongTermTrend     TrendDirection   `json:"long_term_trend"`
	TrendStrength     TrendStrength    `json:"trend_strength"`
	SupportResistance SupportResistance `json:"support_resistance"`
	Momentum          MomentumIndicators `json:"momentum"`
}

type TrendDirection struct {
	Direction    string          `json:"direction"`
	Confidence   decimal.Decimal `json:"confidence"`
	Duration     int             `json:"duration"`
	Slope        decimal.Decimal `json:"slope"`
}

type TrendStrength struct {
	Overall      decimal.Decimal `json:"overall"`
	ShortTerm    decimal.Decimal `json:"short_term"`
	MediumTerm   decimal.Decimal `json:"medium_term"`
	LongTerm     decimal.Decimal `json:"long_term"`
}

type SupportResistance struct {
	NearestSupport     decimal.Decimal `json:"nearest_support"`
	NearestResistance  decimal.Decimal `json:"nearest_resistance"`
	SupportStrength    decimal.Decimal `json:"support_strength"`
	ResistanceStrength decimal.Decimal `json:"resistance_strength"`
}

type MomentumIndicators struct {
	RSI              decimal.Decimal `json:"rsi"`
	MACD             decimal.Decimal `json:"macd"`
	MACDSignal       decimal.Decimal `json:"macd_signal"`
	StochasticK      decimal.Decimal `json:"stochastic_k"`
	StochasticD      decimal.Decimal `json:"stochastic_d"`
	WilliamsR        decimal.Decimal `json:"williams_r"`
}

type BenchmarkComparison struct {
	PrimaryBenchmark   BenchmarkMetrics            `json:"primary_benchmark"`
	SecondaryBenchmarks []BenchmarkMetrics         `json:"secondary_benchmarks"`
	RelativePerformance RelativePerformance        `json:"relative_performance"`
	Attribution        PerformanceAttribution     `json:"attribution"`
}

type BenchmarkMetrics struct {
	Name               string          `json:"name"`
	Return             decimal.Decimal `json:"return"`
	Volatility         decimal.Decimal `json:"volatility"`
	SharpeRatio        decimal.Decimal `json:"sharpe_ratio"`
	MaxDrawdown        decimal.Decimal `json:"max_drawdown"`
	Correlation        decimal.Decimal `json:"correlation"`
	Beta               decimal.Decimal `json:"beta"`
	Alpha              decimal.Decimal `json:"alpha"`
	TrackingError      decimal.Decimal `json:"tracking_error"`
	InformationRatio   decimal.Decimal `json:"information_ratio"`
}

type RelativePerformance struct {
	Outperformance     decimal.Decimal `json:"outperformance"`
	HitRate            decimal.Decimal `json:"hit_rate"`
	UpCapture          decimal.Decimal `json:"up_capture"`
	DownCapture        decimal.Decimal `json:"down_capture"`
	BestPeriod         decimal.Decimal `json:"best_period"`
	WorstPeriod        decimal.Decimal `json:"worst_period"`
}

type PerformanceAttribution struct {
	AssetAllocation    decimal.Decimal               `json:"asset_allocation"`
	SecuritySelection  decimal.Decimal               `json:"security_selection"`
	InteractionEffect  decimal.Decimal               `json:"interaction_effect"`
	SectorAttribution  map[string]decimal.Decimal    `json:"sector_attribution"`
}

type Recommendation struct {
	Type        string          `json:"type"`
	Priority    int             `json:"priority"`
	Category    string          `json:"category"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Action      string          `json:"action"`
	Impact      decimal.Decimal `json:"impact"`
	Confidence  decimal.Decimal `json:"confidence"`
	Timeline    string          `json:"timeline"`
}

type OverallScore struct {
	TotalScore       decimal.Decimal `json:"total_score"`
	PerformanceScore decimal.Decimal `json:"performance_score"`
	RiskScore        decimal.Decimal `json:"risk_score"`
	DiversificationScore decimal.Decimal `json:"diversification_score"`
	EfficiencyScore  decimal.Decimal `json:"efficiency_score"`
	Grade           string          `json:"grade"`
	Ranking         string          `json:"ranking"`
}

func (pa *PortfolioAnalyzer) PerformComprehensiveAnalysis(ctx context.Context, portfolio *models.Portfolio, snapshots []models.Snapshot, benchmarkData []decimal.Decimal) (*ComprehensiveAnalysis, error) {
	if portfolio == nil {
		return nil, fmt.Errorf("portfolio is required")
	}

	analysis := &ComprehensiveAnalysis{
		Portfolio:   portfolio,
		LastUpdated: time.Now(),
	}

	// Performance Analysis
	performanceAnalysis, err := pa.analyzePerformance(ctx, portfolio, snapshots)
	if err != nil {
		return nil, fmt.Errorf("performance analysis failed: %w", err)
	}
	analysis.PerformanceAnalysis = *performanceAnalysis

	// Risk Analysis
	riskAnalysis, err := pa.analyzeRisk(ctx, portfolio, snapshots)
	if err != nil {
		return nil, fmt.Errorf("risk analysis failed: %w", err)
	}
	analysis.RiskAnalysis = *riskAnalysis

	// Diversification Analysis
	diversificationAnalysis, err := pa.analyzeDiversification(ctx, portfolio)
	if err != nil {
		return nil, fmt.Errorf("diversification analysis failed: %w", err)
	}
	analysis.DiversificationAnalysis = *diversificationAnalysis

	// Trend Analysis
	trendAnalysis, err := pa.analyzeTrends(ctx, snapshots)
	if err != nil {
		return nil, fmt.Errorf("trend analysis failed: %w", err)
	}
	analysis.TrendAnalysis = *trendAnalysis

	// Benchmark Comparison
	if len(benchmarkData) > 0 {
		benchmarkComparison, err := pa.analyzeBenchmarkComparison(ctx, snapshots, benchmarkData)
		if err == nil {
			analysis.BenchmarkComparison = *benchmarkComparison
		}
	}

	// Generate Recommendations
	analysis.Recommendations = pa.generateRecommendations(analysis)

	// Calculate Overall Score
	analysis.OverallScore = pa.calculateOverallScore(analysis)

	return analysis, nil
}

func (pa *PortfolioAnalyzer) analyzePerformance(ctx context.Context, portfolio *models.Portfolio, snapshots []models.Snapshot) (*PerformanceAnalysis, error) {
	analysis := &PerformanceAnalysis{}

	// Calculate period returns
	analysis.Returns = pa.calculatePeriodReturns(snapshots)

	// Calculate consistency metrics
	analysis.Consistency = pa.calculateConsistencyMetrics(snapshots)

	// Calculate win/loss ratios
	analysis.WinLossRatio = pa.calculateWinLossRatio(snapshots)

	// Analyze drawdowns
	analysis.DrawdownAnalysis = pa.analyzeDrawdowns(snapshots)

	// Performance ranking (simplified)
	analysis.PerformanceRanking = PerformanceRanking{
		Percentile:      decimal.NewFromFloat(75), // Placeholder
		Rank:            250,                       // Placeholder
		TotalPortfolios: 1000,                     // Placeholder
		Category:        "Mixed Allocation",
	}

	return analysis, nil
}

func (pa *PortfolioAnalyzer) calculatePeriodReturns(snapshots []models.Snapshot) PeriodReturns {
	returns := PeriodReturns{}

	if len(snapshots) < 2 {
		return returns
	}

	// Sort snapshots by date
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.Before(snapshots[j].Timestamp)
	})

	latest := snapshots[len(snapshots)-1]
	earliest := snapshots[0]

	// Calculate inception return
	if !earliest.Value.Total.IsZero() {
		returns.Inception = latest.Value.Total.Sub(earliest.Value.Total).Div(earliest.Value.Total)
	}

	// Calculate other period returns (simplified - would need proper date filtering)
	now := time.Now()

	// Find snapshots for different periods
	for i := len(snapshots) - 1; i >= 0; i-- {
		snapshot := snapshots[i]
		daysDiff := int(now.Sub(snapshot.Timestamp).Hours() / 24)

		switch {
		case daysDiff >= 1 && daysDiff <= 7 && returns.Daily.IsZero():
			if !snapshot.Value.Total.IsZero() {
				returns.Daily = latest.Value.Total.Sub(snapshot.Value.Total).Div(snapshot.Value.Total)
			}
		case daysDiff >= 7 && daysDiff <= 14 && returns.Weekly.IsZero():
			if !snapshot.Value.Total.IsZero() {
				returns.Weekly = latest.Value.Total.Sub(snapshot.Value.Total).Div(snapshot.Value.Total)
			}
		case daysDiff >= 30 && daysDiff <= 37 && returns.Monthly.IsZero():
			if !snapshot.Value.Total.IsZero() {
				returns.Monthly = latest.Value.Total.Sub(snapshot.Value.Total).Div(snapshot.Value.Total)
			}
		case daysDiff >= 90 && daysDiff <= 97 && returns.Quarterly.IsZero():
			if !snapshot.Value.Total.IsZero() {
				returns.Quarterly = latest.Value.Total.Sub(snapshot.Value.Total).Div(snapshot.Value.Total)
			}
		case daysDiff >= 365 && daysDiff <= 372 && returns.OneYear.IsZero():
			if !snapshot.Value.Total.IsZero() {
				returns.OneYear = latest.Value.Total.Sub(snapshot.Value.Total).Div(snapshot.Value.Total)
			}
		}
	}

	return returns
}

func (pa *PortfolioAnalyzer) calculateConsistencyMetrics(snapshots []models.Snapshot) ConsistencyMetrics {
	metrics := ConsistencyMetrics{}

	if len(snapshots) < 2 {
		return metrics
	}

	// Calculate monthly returns and streaks
	positiveCount := 0
	negativeCount := 0
	currentStreak := 0
	currentStreakType := ""
	longestWinStreak := 0
	longestLossStreak := 0
	winStreaks := make([]int, 0)
	lossStreaks := make([]int, 0)

	for i := 1; i < len(snapshots); i++ {
		prevValue := snapshots[i-1].Value.Total
		currentValue := snapshots[i].Value.Total

		if prevValue.IsZero() {
			continue
		}

		change := currentValue.Sub(prevValue).Div(prevValue)

		if change.GreaterThan(decimal.Zero) {
			positiveCount++
			if currentStreakType == "win" {
				currentStreak++
			} else {
				if currentStreakType == "loss" && currentStreak > 0 {
					lossStreaks = append(lossStreaks, currentStreak)
					if currentStreak > longestLossStreak {
						longestLossStreak = currentStreak
					}
				}
				currentStreak = 1
				currentStreakType = "win"
			}
		} else if change.LessThan(decimal.Zero) {
			negativeCount++
			if currentStreakType == "loss" {
				currentStreak++
			} else {
				if currentStreakType == "win" && currentStreak > 0 {
					winStreaks = append(winStreaks, currentStreak)
					if currentStreak > longestWinStreak {
						longestWinStreak = currentStreak
					}
				}
				currentStreak = 1
				currentStreakType = "loss"
			}
		}
	}

	// Close final streak
	if currentStreakType == "win" && currentStreak > 0 {
		winStreaks = append(winStreaks, currentStreak)
		if currentStreak > longestWinStreak {
			longestWinStreak = currentStreak
		}
	} else if currentStreakType == "loss" && currentStreak > 0 {
		lossStreaks = append(lossStreaks, currentStreak)
		if currentStreak > longestLossStreak {
			longestLossStreak = currentStreak
		}
	}

	metrics.PositiveMonths = positiveCount
	metrics.NegativeMonths = negativeCount
	metrics.LongestWinStreak = longestWinStreak
	metrics.LongestLossStreak = longestLossStreak

	// Calculate average streaks
	if len(winStreaks) > 0 {
		sum := 0
		for _, streak := range winStreaks {
			sum += streak
		}
		metrics.AverageWinStreak = decimal.NewFromInt(int64(sum)).Div(decimal.NewFromInt(int64(len(winStreaks))))
	}

	if len(lossStreaks) > 0 {
		sum := 0
		for _, streak := range lossStreaks {
			sum += streak
		}
		metrics.AverageLossStreak = decimal.NewFromInt(int64(sum)).Div(decimal.NewFromInt(int64(len(lossStreaks))))
	}

	// Calculate consistency ratio
	totalPeriods := positiveCount + negativeCount
	if totalPeriods > 0 {
		metrics.ConsistencyRatio = decimal.NewFromInt(int64(positiveCount)).Div(decimal.NewFromInt(int64(totalPeriods)))
	}

	return metrics
}

func (pa *PortfolioAnalyzer) calculateWinLossRatio(snapshots []models.Snapshot) WinLossRatio {
	ratio := WinLossRatio{}

	if len(snapshots) < 2 {
		return ratio
	}

	wins := make([]decimal.Decimal, 0)
	losses := make([]decimal.Decimal, 0)

	for i := 1; i < len(snapshots); i++ {
		prevValue := snapshots[i-1].Value.Total
		currentValue := snapshots[i].Value.Total

		if prevValue.IsZero() {
			continue
		}

		change := currentValue.Sub(prevValue).Div(prevValue)

		if change.GreaterThan(decimal.Zero) {
			wins = append(wins, change)
		} else if change.LessThan(decimal.Zero) {
			losses = append(losses, change.Abs())
		}
	}

	totalPeriods := len(wins) + len(losses)
	if totalPeriods > 0 {
		ratio.WinRate = decimal.NewFromInt(int64(len(wins))).Div(decimal.NewFromInt(int64(totalPeriods)))
		ratio.LossRate = decimal.NewFromInt(int64(len(losses))).Div(decimal.NewFromInt(int64(totalPeriods)))
	}

	// Calculate average win/loss
	if len(wins) > 0 {
		sum := decimal.Zero
		for _, win := range wins {
			sum = sum.Add(win)
		}
		ratio.AverageWin = sum.Div(decimal.NewFromInt(int64(len(wins))))
	}

	if len(losses) > 0 {
		sum := decimal.Zero
		for _, loss := range losses {
			sum = sum.Add(loss)
		}
		ratio.AverageLoss = sum.Div(decimal.NewFromInt(int64(len(losses))))
	}

	// Calculate win/loss ratio
	if !ratio.AverageLoss.IsZero() {
		ratio.WinLossRatio = ratio.AverageWin.Div(ratio.AverageLoss)
	}

	// Calculate profit factor
	totalWins := ratio.AverageWin.Mul(decimal.NewFromInt(int64(len(wins))))
	totalLosses := ratio.AverageLoss.Mul(decimal.NewFromInt(int64(len(losses))))
	if !totalLosses.IsZero() {
		ratio.ProfitFactor = totalWins.Div(totalLosses)
	}

	// Calculate expected value
	winProbability := ratio.WinRate
	lossProbability := ratio.LossRate
	ratio.ExpectedValue = winProbability.Mul(ratio.AverageWin).Sub(lossProbability.Mul(ratio.AverageLoss))

	return ratio
}

func (pa *PortfolioAnalyzer) analyzeDrawdowns(snapshots []models.Snapshot) DrawdownAnalysis {
	analysis := DrawdownAnalysis{
		DrawdownPeriods: make([]DrawdownPeriod, 0),
	}

	if len(snapshots) < 2 {
		return analysis
	}

	// Sort snapshots by date
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.Before(snapshots[j].Timestamp)
	})

	peak := snapshots[0].Value.Total
	peakDate := snapshots[0].Timestamp
	inDrawdown := false
	var currentDrawdown DrawdownPeriod
	drawdowns := make([]decimal.Decimal, 0)

	for _, snapshot := range snapshots {
		currentValue := snapshot.Value.Total

		if currentValue.GreaterThan(peak) {
			// New peak
			if inDrawdown {
				// End current drawdown
				currentDrawdown.RecoveryDate = snapshot.Timestamp
				currentDrawdown.RecoveryTime = int(snapshot.Timestamp.Sub(currentDrawdown.EndDate).Hours() / 24)
				analysis.DrawdownPeriods = append(analysis.DrawdownPeriods, currentDrawdown)
				inDrawdown = false
			}
			peak = currentValue
			peakDate = snapshot.Timestamp
		} else if currentValue.LessThan(peak) {
			// In drawdown
			drawdown := peak.Sub(currentValue).Div(peak)
			drawdowns = append(drawdowns, drawdown)

			if !inDrawdown {
				// Start new drawdown
				currentDrawdown = DrawdownPeriod{
					StartDate: peakDate,
					EndDate:   snapshot.Timestamp,
					Magnitude: drawdown,
					Duration:  int(snapshot.Timestamp.Sub(peakDate).Hours() / 24),
				}
				inDrawdown = true
			} else {
				// Update current drawdown
				if drawdown.GreaterThan(currentDrawdown.Magnitude) {
					currentDrawdown.Magnitude = drawdown
					currentDrawdown.EndDate = snapshot.Timestamp
				}
				currentDrawdown.Duration = int(snapshot.Timestamp.Sub(currentDrawdown.StartDate).Hours() / 24)
			}
		}
	}

	// Calculate current drawdown
	if inDrawdown {
		analysis.CurrentDrawdown = currentDrawdown.Magnitude
	}

	// Calculate statistics
	if len(drawdowns) > 0 {
		// Max drawdown
		maxDD := decimal.Zero
		for _, dd := range drawdowns {
			if dd.GreaterThan(maxDD) {
				maxDD = dd
			}
		}
		analysis.MaxDrawdown = maxDD

		// Average drawdown
		sum := decimal.Zero
		for _, dd := range drawdowns {
			sum = sum.Add(dd)
		}
		analysis.AverageDrawdown = sum.Div(decimal.NewFromInt(int64(len(drawdowns))))

		// Drawdown frequency
		totalPeriods := len(snapshots)
		analysis.DrawdownFrequency = decimal.NewFromInt(int64(len(analysis.DrawdownPeriods))).Div(decimal.NewFromInt(int64(totalPeriods)))
	}

	// Calculate recovery time metrics
	if len(analysis.DrawdownPeriods) > 0 {
		totalRecoveryTime := 0
		maxRecoveryTime := 0
		recoveredDrawdowns := 0

		for _, dd := range analysis.DrawdownPeriods {
			if !dd.RecoveryDate.IsZero() {
				totalRecoveryTime += dd.RecoveryTime
				if dd.RecoveryTime > maxRecoveryTime {
					maxRecoveryTime = dd.RecoveryTime
				}
				recoveredDrawdowns++
			}
		}

		if recoveredDrawdowns > 0 {
			analysis.RecoveryTime.AverageRecoveryTime = totalRecoveryTime / recoveredDrawdowns
		}
		analysis.RecoveryTime.MaxRecoveryTime = maxRecoveryTime

		// Current recovery time
		if inDrawdown {
			analysis.RecoveryTime.CurrentRecoveryTime = int(time.Since(currentDrawdown.StartDate).Hours() / 24)
		}
	}

	return analysis
}

func (pa *PortfolioAnalyzer) analyzeRisk(ctx context.Context, portfolio *models.Portfolio, snapshots []models.Snapshot) (*RiskAnalysis, error) {
	analysis := &RiskAnalysis{}

	// Calculate basic risk metrics
	analysis.RiskMetrics = RiskMetrics{
		TotalRisk:          portfolio.RiskMetrics.Volatility30d,
		SystematicRisk:     portfolio.RiskMetrics.Volatility30d.Mul(decimal.NewFromFloat(0.7)), // Simplified
		IdiosyncraticRisk:  portfolio.RiskMetrics.Volatility30d.Mul(decimal.NewFromFloat(0.3)), // Simplified
		RiskAdjustedReturn: portfolio.RiskMetrics.SharpeRatio,
	}

	// Calculate concentration risk
	analysis.ConcentrationRisk = pa.calculateConcentrationRisk(portfolio)

	// Calculate liquidity risk (simplified)
	analysis.LiquidityRisk = LiquidityRisk{
		OverallLiquidity:   decimal.NewFromFloat(0.8), // Placeholder
		DaysToLiquidate:    5,                         // Placeholder
		IlliquidPercentage: decimal.NewFromFloat(0.1), // Placeholder
	}

	// Risk profile assessment
	analysis.RiskProfile = RiskProfile{
		RiskCapacity:    decimal.NewFromFloat(0.8),
		RiskTolerance:   decimal.NewFromFloat(0.7),
		RiskUtilization: decimal.NewFromFloat(0.6),
		RecommendedRisk: "Moderate",
	}

	return analysis, nil
}

func (pa *PortfolioAnalyzer) calculateConcentrationRisk(portfolio *models.Portfolio) ConcentrationRisk {
	risk := ConcentrationRisk{
		SectorConcentration: make(map[string]decimal.Decimal),
	}

	if portfolio.TotalValue.IsZero() || len(portfolio.Holdings) == 0 {
		return risk
	}

	// Sort holdings by value
	holdings := make([]models.Holding, len(portfolio.Holdings))
	copy(holdings, portfolio.Holdings)
	sort.Slice(holdings, func(i, j int) bool {
		return holdings[i].CurrentValue.GreaterThan(holdings[j].CurrentValue)
	})

	// Calculate top holdings weights
	if len(holdings) > 0 {
		risk.TopHoldingWeight = holdings[0].CurrentValue.Div(portfolio.TotalValue)
	}

	top5Value := decimal.Zero
	top10Value := decimal.Zero
	for i, holding := range holdings {
		if i < 5 {
			top5Value = top5Value.Add(holding.CurrentValue)
		}
		if i < 10 {
			top10Value = top10Value.Add(holding.CurrentValue)
		}
	}

	risk.Top5HoldingsWeight = top5Value.Div(portfolio.TotalValue)
	risk.Top10HoldingsWeight = top10Value.Div(portfolio.TotalValue)

	// Calculate Herfindahl Index
	hhi := decimal.Zero
	for _, holding := range holdings {
		weight := holding.CurrentValue.Div(portfolio.TotalValue)
		hhi = hhi.Add(weight.Mul(weight))
	}
	risk.HerfindahlIndex = hhi

	// Effective number of assets
	if !hhi.IsZero() {
		risk.EffectiveAssets = decimal.NewFromInt(1).Div(hhi)
	}

	// Sector concentration
	sectorValues := make(map[string]decimal.Decimal)
	for _, holding := range holdings {
		sector := holding.Category
		if sector == "" {
			sector = "Unknown"
		}
		if existing, exists := sectorValues[sector]; exists {
			sectorValues[sector] = existing.Add(holding.CurrentValue)
		} else {
			sectorValues[sector] = holding.CurrentValue
		}
	}

	for sector, value := range sectorValues {
		risk.SectorConcentration[sector] = value.Div(portfolio.TotalValue)
	}

	// Overall concentration score (0-100, lower is better)
	concentrationScore := hhi.Mul(decimal.NewFromInt(100))
	risk.ConcentrationScore = concentrationScore

	return risk
}

func (pa *PortfolioAnalyzer) analyzeDiversification(ctx context.Context, portfolio *models.Portfolio) (*DiversificationAnalysis, error) {
	analysis := &DiversificationAnalysis{}

	// Calculate diversification score
	diversificationScore, err := pa.correlationAnalyzer.CalculateDiversificationScore(ctx, portfolio.Holdings, nil)
	if err == nil {
		analysis.DiversificationScore = diversificationScore
	}

	// Calculate asset class exposure
	analysis.AssetClassExposure = pa.calculateAssetClassExposure(portfolio)

	// Calculate sector exposure
	analysis.SectorExposure = pa.calculateSectorExposure(portfolio)

	return analysis, nil
}

func (pa *PortfolioAnalyzer) calculateAssetClassExposure(portfolio *models.Portfolio) AssetClassExposure {
	exposure := AssetClassExposure{}

	if portfolio.TotalValue.IsZero() {
		return exposure
	}

	for _, holding := range portfolio.Holdings {
		weight := holding.CurrentValue.Div(portfolio.TotalValue)

		// Categorize holdings (simplified)
		switch holding.Category {
		case "crypto", "cryptocurrency":
			exposure.Crypto = exposure.Crypto.Add(weight)
		case "equity", "stock":
			exposure.Equities = exposure.Equities.Add(weight)
		case "bond", "fixed_income":
			exposure.Bonds = exposure.Bonds.Add(weight)
		case "commodity":
			exposure.Commodities = exposure.Commodities.Add(weight)
		case "real_estate", "reit":
			exposure.RealEstate = exposure.RealEstate.Add(weight)
		case "cash":
			exposure.Cash = exposure.Cash.Add(weight)
		default:
			exposure.Alternatives = exposure.Alternatives.Add(weight)
		}
	}

	return exposure
}

func (pa *PortfolioAnalyzer) calculateSectorExposure(portfolio *models.Portfolio) SectorExposure {
	exposure := SectorExposure{}

	if portfolio.TotalValue.IsZero() {
		return exposure
	}

	for _, holding := range portfolio.Holdings {
		weight := holding.CurrentValue.Div(portfolio.TotalValue)

		// Map categories to sectors (simplified)
		switch holding.Category {
		case "technology", "tech":
			exposure.Technology = exposure.Technology.Add(weight)
		case "healthcare", "health":
			exposure.Healthcare = exposure.Healthcare.Add(weight)
		case "financial", "finance":
			exposure.Financials = exposure.Financials.Add(weight)
		case "consumer":
			exposure.ConsumerDiscretionary = exposure.ConsumerDiscretionary.Add(weight)
		case "communication":
			exposure.Communication = exposure.Communication.Add(weight)
		case "industrial":
			exposure.Industrials = exposure.Industrials.Add(weight)
		case "energy":
			exposure.Energy = exposure.Energy.Add(weight)
		case "materials":
			exposure.Materials = exposure.Materials.Add(weight)
		case "utilities":
			exposure.Utilities = exposure.Utilities.Add(weight)
		case "real_estate":
			exposure.RealEstate = exposure.RealEstate.Add(weight)
		default:
			exposure.Other = exposure.Other.Add(weight)
		}
	}

	return exposure
}

func (pa *PortfolioAnalyzer) analyzeTrends(ctx context.Context, snapshots []models.Snapshot) (*TrendAnalysis, error) {
	analysis := &TrendAnalysis{}

	if len(snapshots) < 10 {
		return analysis, nil
	}

	// Sort snapshots by date
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.Before(snapshots[j].Timestamp)
	})

	// Analyze short, medium, and long-term trends
	analysis.ShortTermTrend = pa.analyzeTrendDirection(snapshots, 10)  // Last 10 periods
	analysis.MediumTermTrend = pa.analyzeTrendDirection(snapshots, 30) // Last 30 periods
	analysis.LongTermTrend = pa.analyzeTrendDirection(snapshots, 90)   // Last 90 periods

	// Calculate trend strength
	analysis.TrendStrength = pa.calculateTrendStrength(snapshots)

	// Calculate momentum indicators (simplified)
	analysis.Momentum = pa.calculateMomentumIndicators(snapshots)

	return analysis, nil
}

func (pa *PortfolioAnalyzer) analyzeTrendDirection(snapshots []models.Snapshot, periods int) TrendDirection {
	direction := TrendDirection{}

	if len(snapshots) < periods {
		periods = len(snapshots)
	}

	if periods < 2 {
		return direction
	}

	// Take last N periods
	recentSnapshots := snapshots[len(snapshots)-periods:]

	// Calculate linear regression slope (simplified)
	n := len(recentSnapshots)
	sumX := decimal.Zero
	sumY := decimal.Zero
	sumXY := decimal.Zero
	sumX2 := decimal.Zero

	for i, snapshot := range recentSnapshots {
		x := decimal.NewFromInt(int64(i))
		y := snapshot.Value.Total

		sumX = sumX.Add(x)
		sumY = sumY.Add(y)
		sumXY = sumXY.Add(x.Mul(y))
		sumX2 = sumX2.Add(x.Mul(x))
	}

	nDecimal := decimal.NewFromInt(int64(n))
	numerator := nDecimal.Mul(sumXY).Sub(sumX.Mul(sumY))
	denominator := nDecimal.Mul(sumX2).Sub(sumX.Mul(sumX))

	if !denominator.IsZero() {
		slope := numerator.Div(denominator)
		direction.Slope = slope

		if slope.GreaterThan(decimal.Zero) {
			direction.Direction = "up"
		} else if slope.LessThan(decimal.Zero) {
			direction.Direction = "down"
		} else {
			direction.Direction = "sideways"
		}

		// Calculate confidence based on R-squared (simplified)
		direction.Confidence = slope.Abs().Mul(decimal.NewFromFloat(100))
		if direction.Confidence.GreaterThan(decimal.NewFromInt(100)) {
			direction.Confidence = decimal.NewFromInt(100)
		}
	}

	direction.Duration = periods

	return direction
}

func (pa *PortfolioAnalyzer) calculateTrendStrength(snapshots []models.Snapshot) TrendStrength {
	strength := TrendStrength{}

	// Calculate trend strength based on consistency of direction
	// This is a simplified implementation
	shortTerm := pa.analyzeTrendDirection(snapshots, 10)
	mediumTerm := pa.analyzeTrendDirection(snapshots, 30)
	longTerm := pa.analyzeTrendDirection(snapshots, 90)

	strength.ShortTerm = shortTerm.Confidence.Div(decimal.NewFromInt(100))
	strength.MediumTerm = mediumTerm.Confidence.Div(decimal.NewFromInt(100))
	strength.LongTerm = longTerm.Confidence.Div(decimal.NewFromInt(100))

	// Overall strength is weighted average
	strength.Overall = strength.ShortTerm.Mul(decimal.NewFromFloat(0.5)).
		Add(strength.MediumTerm.Mul(decimal.NewFromFloat(0.3))).
		Add(strength.LongTerm.Mul(decimal.NewFromFloat(0.2)))

	return strength
}

func (pa *PortfolioAnalyzer) calculateMomentumIndicators(snapshots []models.Snapshot) MomentumIndicators {
	indicators := MomentumIndicators{}

	if len(snapshots) < 14 {
		return indicators
	}

	// Calculate RSI (simplified)
	indicators.RSI = pa.calculateRSI(snapshots, 14)

	return indicators
}

func (pa *PortfolioAnalyzer) calculateRSI(snapshots []models.Snapshot, periods int) decimal.Decimal {
	if len(snapshots) < periods+1 {
		return decimal.NewFromInt(50) // Neutral RSI
	}

	gains := make([]decimal.Decimal, 0)
	losses := make([]decimal.Decimal, 0)

	for i := 1; i <= periods; i++ {
		prev := snapshots[len(snapshots)-i-1].Value.Total
		current := snapshots[len(snapshots)-i].Value.Total

		if prev.IsZero() {
			continue
		}

		change := current.Sub(prev)
		if change.GreaterThan(decimal.Zero) {
			gains = append(gains, change)
			losses = append(losses, decimal.Zero)
		} else if change.LessThan(decimal.Zero) {
			gains = append(gains, decimal.Zero)
			losses = append(losses, change.Abs())
		} else {
			gains = append(gains, decimal.Zero)
			losses = append(losses, decimal.Zero)
		}
	}

	// Calculate average gains and losses
	avgGain := decimal.Zero
	avgLoss := decimal.Zero

	if len(gains) > 0 {
		sum := decimal.Zero
		for _, gain := range gains {
			sum = sum.Add(gain)
		}
		avgGain = sum.Div(decimal.NewFromInt(int64(len(gains))))
	}

	if len(losses) > 0 {
		sum := decimal.Zero
		for _, loss := range losses {
			sum = sum.Add(loss)
		}
		avgLoss = sum.Div(decimal.NewFromInt(int64(len(losses))))
	}

	// Calculate RSI
	if avgLoss.IsZero() {
		return decimal.NewFromInt(100)
	}

	rs := avgGain.Div(avgLoss)
	rsi := decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))

	return rsi
}

func (pa *PortfolioAnalyzer) analyzeBenchmarkComparison(ctx context.Context, snapshots []models.Snapshot, benchmarkData []decimal.Decimal) (*BenchmarkComparison, error) {
	comparison := &BenchmarkComparison{}

	// This would be expanded with actual benchmark comparison logic
	comparison.PrimaryBenchmark = BenchmarkMetrics{
		Name:        "Market Index",
		Return:      decimal.NewFromFloat(0.08), // Placeholder
		Volatility:  decimal.NewFromFloat(0.15), // Placeholder
		SharpeRatio: decimal.NewFromFloat(0.53), // Placeholder
		Correlation: decimal.NewFromFloat(0.85), // Placeholder
		Beta:        decimal.NewFromFloat(1.1),  // Placeholder
		Alpha:       decimal.NewFromFloat(0.02), // Placeholder
	}

	return comparison, nil
}

func (pa *PortfolioAnalyzer) generateRecommendations(analysis *ComprehensiveAnalysis) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// Performance-based recommendations
	if analysis.PerformanceAnalysis.Returns.OneYear.LessThan(decimal.Zero) {
		recommendations = append(recommendations, Recommendation{
			Type:        "performance",
			Priority:    1,
			Category:    "Underperformance",
			Title:       "Address Negative Returns",
			Description: "Portfolio has negative returns over the past year",
			Action:      "Review holdings and consider rebalancing",
			Impact:      decimal.NewFromFloat(0.8),
			Confidence:  decimal.NewFromFloat(0.9),
			Timeline:    "immediate",
		})
	}

	// Risk-based recommendations
	if analysis.RiskAnalysis.ConcentrationRisk.TopHoldingWeight.GreaterThan(decimal.NewFromFloat(0.3)) {
		recommendations = append(recommendations, Recommendation{
			Type:        "risk",
			Priority:    2,
			Category:    "Concentration Risk",
			Title:       "Reduce Position Concentration",
			Description: "Top holding represents more than 30% of portfolio",
			Action:      "Reduce largest position and diversify",
			Impact:      decimal.NewFromFloat(0.7),
			Confidence:  decimal.NewFromFloat(0.8),
			Timeline:    "short_term",
		})
	}

	// Diversification recommendations
	if analysis.DiversificationAnalysis.DiversificationScore != nil &&
		analysis.DiversificationAnalysis.DiversificationScore.OverallScore.LessThan(decimal.NewFromFloat(60)) {
		recommendations = append(recommendations, Recommendation{
			Type:        "diversification",
			Priority:    3,
			Category:    "Poor Diversification",
			Title:       "Improve Portfolio Diversification",
			Description: "Portfolio diversification score is below recommended levels",
			Action:      "Add holdings from different sectors and asset classes",
			Impact:      decimal.NewFromFloat(0.6),
			Confidence:  decimal.NewFromFloat(0.7),
			Timeline:    "medium_term",
		})
	}

	// Sort recommendations by priority
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority < recommendations[j].Priority
	})

	return recommendations
}

func (pa *PortfolioAnalyzer) calculateOverallScore(analysis *ComprehensiveAnalysis) OverallScore {
	score := OverallScore{}

	// Performance score (0-100)
	performanceScore := decimal.NewFromFloat(70) // Default neutral score
	if analysis.PerformanceAnalysis.Returns.OneYear.GreaterThan(decimal.Zero) {
		performanceScore = decimal.NewFromFloat(80)
	}
	if analysis.PerformanceAnalysis.Returns.OneYear.GreaterThan(decimal.NewFromFloat(0.1)) {
		performanceScore = decimal.NewFromFloat(90)
	}
	score.PerformanceScore = performanceScore

	// Risk score (0-100, higher is better risk-adjusted performance)
	riskScore := decimal.NewFromFloat(70) // Default
	if analysis.RiskAnalysis.ConcentrationRisk.ConcentrationScore.LessThan(decimal.NewFromFloat(30)) {
		riskScore = decimal.NewFromFloat(80)
	}
	score.RiskScore = riskScore

	// Diversification score
	diversificationScore := decimal.NewFromFloat(70) // Default
	if analysis.DiversificationAnalysis.DiversificationScore != nil {
		diversificationScore = analysis.DiversificationAnalysis.DiversificationScore.OverallScore
	}
	score.DiversificationScore = diversificationScore

	// Efficiency score (based on Sharpe ratio and other metrics)
	efficiencyScore := decimal.NewFromFloat(70) // Default
	score.EfficiencyScore = efficiencyScore

	// Calculate total score (weighted average)
	score.TotalScore = performanceScore.Mul(decimal.NewFromFloat(0.3)).
		Add(riskScore.Mul(decimal.NewFromFloat(0.25))).
		Add(diversificationScore.Mul(decimal.NewFromFloat(0.25))).
		Add(efficiencyScore.Mul(decimal.NewFromFloat(0.2)))

	// Assign grade
	totalScoreFloat, _ := score.TotalScore.Float64()
	switch {
	case totalScoreFloat >= 90:
		score.Grade = "A+"
		score.Ranking = "Excellent"
	case totalScoreFloat >= 80:
		score.Grade = "A"
		score.Ranking = "Very Good"
	case totalScoreFloat >= 70:
		score.Grade = "B"
		score.Ranking = "Good"
	case totalScoreFloat >= 60:
		score.Grade = "C"
		score.Ranking = "Average"
	case totalScoreFloat >= 50:
		score.Grade = "D"
		score.Ranking = "Below Average"
	default:
		score.Grade = "F"
		score.Ranking = "Poor"
	}

	return score
}