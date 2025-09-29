package calculator

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"portfolio-api/internal/models"
)

type RiskCalculator struct {
	riskFreeRate decimal.Decimal
}

type RiskCalculatorConfig struct {
	RiskFreeRate float64 `json:"risk_free_rate" default:"0.02"`
}

func NewRiskCalculator(config RiskCalculatorConfig) *RiskCalculator {
	return &RiskCalculator{
		riskFreeRate: decimal.NewFromFloat(config.RiskFreeRate),
	}
}

type VaRParams struct {
	ConfidenceLevel float64 `json:"confidence_level" default:"0.95"`
	TimeHorizon     int     `json:"time_horizon" default:"1"`
}

type RiskMetricsResult struct {
	Volatility30d      decimal.Decimal `json:"volatility_30d"`
	Volatility90d      decimal.Decimal `json:"volatility_90d"`
	SharpeRatio        decimal.Decimal `json:"sharpe_ratio"`
	SortinoRatio       decimal.Decimal `json:"sortino_ratio"`
	MaxDrawdown        decimal.Decimal `json:"max_drawdown"`
	MaxDrawdownDays    int             `json:"max_drawdown_days"`
	VaR95              decimal.Decimal `json:"var_95"`
	VaR99              decimal.Decimal `json:"var_99"`
	CVaR95             decimal.Decimal `json:"cvar_95"`
	CVaR99             decimal.Decimal `json:"cvar_99"`
	Beta               decimal.Decimal `json:"beta"`
	Alpha              decimal.Decimal `json:"alpha"`
	CalmarRatio        decimal.Decimal `json:"calmar_ratio"`
	InformationRatio   decimal.Decimal `json:"information_ratio"`
	TreynorRatio       decimal.Decimal `json:"treynor_ratio"`
	UpsideDeviation    decimal.Decimal `json:"upside_deviation"`
	DownsideDeviation  decimal.Decimal `json:"downside_deviation"`
}

func (rc *RiskCalculator) CalculateRiskMetrics(ctx context.Context, snapshots []models.Snapshot, benchmarkReturns []decimal.Decimal) (*RiskMetricsResult, error) {
	if len(snapshots) < 2 {
		return nil, fmt.Errorf("insufficient data: need at least 2 snapshots")
	}

	returns := rc.calculateReturns(snapshots)
	if len(returns) == 0 {
		return nil, fmt.Errorf("no valid returns calculated")
	}

	result := &RiskMetricsResult{}

	// Calculate volatilities
	if len(returns) >= 30 {
		result.Volatility30d = rc.calculateVolatility(returns[len(returns)-30:])
	}
	if len(returns) >= 90 {
		result.Volatility90d = rc.calculateVolatility(returns[len(returns)-90:])
	} else {
		result.Volatility30d = rc.calculateVolatility(returns)
		result.Volatility90d = result.Volatility30d
	}

	// Calculate Sharpe ratio
	result.SharpeRatio = rc.calculateSharpeRatio(returns)

	// Calculate Sortino ratio
	result.SortinoRatio = rc.calculateSortinoRatio(returns)

	// Calculate maximum drawdown
	result.MaxDrawdown, result.MaxDrawdownDays = rc.calculateMaxDrawdown(snapshots)

	// Calculate VaR and CVaR
	result.VaR95 = rc.calculateVaR(returns, 0.95)
	result.VaR99 = rc.calculateVaR(returns, 0.99)
	result.CVaR95 = rc.calculateCVaR(returns, 0.95)
	result.CVaR99 = rc.calculateCVaR(returns, 0.99)

	// Calculate Calmar ratio
	result.CalmarRatio = rc.calculateCalmarRatio(returns, result.MaxDrawdown)

	// Calculate upside and downside deviations
	result.UpsideDeviation, result.DownsideDeviation = rc.calculateUpsideDownsideDeviation(returns)

	// Calculate beta and alpha if benchmark data is available
	if len(benchmarkReturns) == len(returns) {
		result.Beta = rc.calculateBeta(returns, benchmarkReturns)
		result.Alpha = rc.calculateAlpha(returns, benchmarkReturns, result.Beta)
		result.TreynorRatio = rc.calculateTreynorRatio(returns, result.Beta)
		result.InformationRatio = rc.calculateInformationRatio(returns, benchmarkReturns)
	}

	return result, nil
}

func (rc *RiskCalculator) calculateReturns(snapshots []models.Snapshot) []decimal.Decimal {
	if len(snapshots) < 2 {
		return nil
	}

	returns := make([]decimal.Decimal, 0, len(snapshots)-1)

	for i := 1; i < len(snapshots); i++ {
		prevValue := snapshots[i-1].Value.Total
		currentValue := snapshots[i].Value.Total

		if prevValue.IsZero() {
			continue
		}

		// Calculate daily return: (current - previous) / previous
		ret := currentValue.Sub(prevValue).Div(prevValue)
		returns = append(returns, ret)
	}

	return returns
}

func (rc *RiskCalculator) calculateVolatility(returns []decimal.Decimal) decimal.Decimal {
	if len(returns) < 2 {
		return decimal.Zero
	}

	// Calculate mean
	sum := decimal.Zero
	for _, ret := range returns {
		sum = sum.Add(ret)
	}
	mean := sum.Div(decimal.NewFromInt(int64(len(returns))))

	// Calculate variance
	variance := decimal.Zero
	for _, ret := range returns {
		diff := ret.Sub(mean)
		variance = variance.Add(diff.Mul(diff))
	}
	variance = variance.Div(decimal.NewFromInt(int64(len(returns) - 1)))

	// Convert to float for sqrt calculation
	varianceFloat, _ := variance.Float64()
	if varianceFloat < 0 {
		return decimal.Zero
	}

	// Calculate standard deviation and annualize (assuming daily returns)
	stdDev := decimal.NewFromFloat(math.Sqrt(varianceFloat))
	annualizedVol := stdDev.Mul(decimal.NewFromFloat(math.Sqrt(252))) // 252 trading days

	return annualizedVol
}

func (rc *RiskCalculator) calculateSharpeRatio(returns []decimal.Decimal) decimal.Decimal {
	if len(returns) < 2 {
		return decimal.Zero
	}

	// Calculate excess returns
	excessReturns := make([]decimal.Decimal, len(returns))
	dailyRiskFreeRate := rc.riskFreeRate.Div(decimal.NewFromInt(252)) // Daily risk-free rate

	for i, ret := range returns {
		excessReturns[i] = ret.Sub(dailyRiskFreeRate)
	}

	// Calculate mean excess return
	sum := decimal.Zero
	for _, ret := range excessReturns {
		sum = sum.Add(ret)
	}
	meanExcessReturn := sum.Div(decimal.NewFromInt(int64(len(excessReturns))))

	// Calculate standard deviation of excess returns
	variance := decimal.Zero
	for _, ret := range excessReturns {
		diff := ret.Sub(meanExcessReturn)
		variance = variance.Add(diff.Mul(diff))
	}
	variance = variance.Div(decimal.NewFromInt(int64(len(excessReturns) - 1)))

	varianceFloat, _ := variance.Float64()
	if varianceFloat <= 0 {
		return decimal.Zero
	}

	stdDev := decimal.NewFromFloat(math.Sqrt(varianceFloat))
	if stdDev.IsZero() {
		return decimal.Zero
	}

	// Annualized Sharpe ratio
	annualizedMean := meanExcessReturn.Mul(decimal.NewFromInt(252))
	annualizedStdDev := stdDev.Mul(decimal.NewFromFloat(math.Sqrt(252)))

	return annualizedMean.Div(annualizedStdDev)
}

func (rc *RiskCalculator) calculateSortinoRatio(returns []decimal.Decimal) decimal.Decimal {
	if len(returns) < 2 {
		return decimal.Zero
	}

	// Calculate mean return
	sum := decimal.Zero
	for _, ret := range returns {
		sum = sum.Add(ret)
	}
	meanReturn := sum.Div(decimal.NewFromInt(int64(len(returns))))

	// Calculate downside deviation (only negative returns)
	dailyRiskFreeRate := rc.riskFreeRate.Div(decimal.NewFromInt(252))
	downsideVariance := decimal.Zero
	downsideCount := 0

	for _, ret := range returns {
		if ret.LessThan(dailyRiskFreeRate) {
			diff := ret.Sub(dailyRiskFreeRate)
			downsideVariance = downsideVariance.Add(diff.Mul(diff))
			downsideCount++
		}
	}

	if downsideCount == 0 {
		return decimal.Zero
	}

	downsideVariance = downsideVariance.Div(decimal.NewFromInt(int64(downsideCount)))
	downsideVarianceFloat, _ := downsideVariance.Float64()
	if downsideVarianceFloat <= 0 {
		return decimal.Zero
	}

	downsideStdDev := decimal.NewFromFloat(math.Sqrt(downsideVarianceFloat))

	// Annualized Sortino ratio
	annualizedMean := meanReturn.Sub(dailyRiskFreeRate).Mul(decimal.NewFromInt(252))
	annualizedDownsideStdDev := downsideStdDev.Mul(decimal.NewFromFloat(math.Sqrt(252)))

	if annualizedDownsideStdDev.IsZero() {
		return decimal.Zero
	}

	return annualizedMean.Div(annualizedDownsideStdDev)
}

func (rc *RiskCalculator) calculateMaxDrawdown(snapshots []models.Snapshot) (decimal.Decimal, int) {
	if len(snapshots) < 2 {
		return decimal.Zero, 0
	}

	maxDrawdown := decimal.Zero
	maxDrawdownDays := 0
	peak := snapshots[0].Value.Total
	peakIndex := 0

	for i, snapshot := range snapshots {
		currentValue := snapshot.Value.Total

		// Update peak if current value is higher
		if currentValue.GreaterThan(peak) {
			peak = currentValue
			peakIndex = i
		}

		// Calculate drawdown from peak
		if peak.GreaterThan(decimal.Zero) {
			drawdown := peak.Sub(currentValue).Div(peak)
			if drawdown.GreaterThan(maxDrawdown) {
				maxDrawdown = drawdown
				maxDrawdownDays = i - peakIndex
			}
		}
	}

	return maxDrawdown, maxDrawdownDays
}

func (rc *RiskCalculator) calculateVaR(returns []decimal.Decimal, confidenceLevel float64) decimal.Decimal {
	if len(returns) == 0 {
		return decimal.Zero
	}

	// Sort returns in ascending order
	sortedReturns := make([]decimal.Decimal, len(returns))
	copy(sortedReturns, returns)

	sort.Slice(sortedReturns, func(i, j int) bool {
		return sortedReturns[i].LessThan(sortedReturns[j])
	})

	// Calculate VaR percentile
	percentile := 1.0 - confidenceLevel
	index := int(float64(len(sortedReturns)) * percentile)
	if index >= len(sortedReturns) {
		index = len(sortedReturns) - 1
	}

	// Return negative value (VaR is typically expressed as a positive loss)
	return sortedReturns[index].Neg()
}

func (rc *RiskCalculator) calculateCVaR(returns []decimal.Decimal, confidenceLevel float64) decimal.Decimal {
	if len(returns) == 0 {
		return decimal.Zero
	}

	// Sort returns in ascending order
	sortedReturns := make([]decimal.Decimal, len(returns))
	copy(sortedReturns, returns)

	sort.Slice(sortedReturns, func(i, j int) bool {
		return sortedReturns[i].LessThan(sortedReturns[j])
	})

	// Calculate CVaR (average of worst returns beyond VaR)
	percentile := 1.0 - confidenceLevel
	cutoffIndex := int(float64(len(sortedReturns)) * percentile)
	if cutoffIndex >= len(sortedReturns) {
		cutoffIndex = len(sortedReturns) - 1
	}

	if cutoffIndex == 0 {
		return sortedReturns[0].Neg()
	}

	sum := decimal.Zero
	for i := 0; i <= cutoffIndex; i++ {
		sum = sum.Add(sortedReturns[i])
	}

	avgTailLoss := sum.Div(decimal.NewFromInt(int64(cutoffIndex + 1)))
	return avgTailLoss.Neg()
}

func (rc *RiskCalculator) calculateBeta(portfolioReturns, benchmarkReturns []decimal.Decimal) decimal.Decimal {
	if len(portfolioReturns) != len(benchmarkReturns) || len(portfolioReturns) < 2 {
		return decimal.NewFromInt(1) // Default beta of 1
	}

	// Calculate means
	portfolioSum := decimal.Zero
	benchmarkSum := decimal.Zero
	for i := 0; i < len(portfolioReturns); i++ {
		portfolioSum = portfolioSum.Add(portfolioReturns[i])
		benchmarkSum = benchmarkSum.Add(benchmarkReturns[i])
	}

	portfolioMean := portfolioSum.Div(decimal.NewFromInt(int64(len(portfolioReturns))))
	benchmarkMean := benchmarkSum.Div(decimal.NewFromInt(int64(len(benchmarkReturns))))

	// Calculate covariance and variance
	covariance := decimal.Zero
	benchmarkVariance := decimal.Zero

	for i := 0; i < len(portfolioReturns); i++ {
		portfolioDiff := portfolioReturns[i].Sub(portfolioMean)
		benchmarkDiff := benchmarkReturns[i].Sub(benchmarkMean)

		covariance = covariance.Add(portfolioDiff.Mul(benchmarkDiff))
		benchmarkVariance = benchmarkVariance.Add(benchmarkDiff.Mul(benchmarkDiff))
	}

	if benchmarkVariance.IsZero() {
		return decimal.NewFromInt(1)
	}

	beta := covariance.Div(benchmarkVariance)
	return beta
}

func (rc *RiskCalculator) calculateAlpha(portfolioReturns, benchmarkReturns []decimal.Decimal, beta decimal.Decimal) decimal.Decimal {
	if len(portfolioReturns) != len(benchmarkReturns) || len(portfolioReturns) == 0 {
		return decimal.Zero
	}

	// Calculate average returns
	portfolioSum := decimal.Zero
	benchmarkSum := decimal.Zero
	for i := 0; i < len(portfolioReturns); i++ {
		portfolioSum = portfolioSum.Add(portfolioReturns[i])
		benchmarkSum = benchmarkSum.Add(benchmarkReturns[i])
	}

	portfolioAvg := portfolioSum.Div(decimal.NewFromInt(int64(len(portfolioReturns))))
	benchmarkAvg := benchmarkSum.Div(decimal.NewFromInt(int64(len(benchmarkReturns))))

	// Alpha = Portfolio Return - (Risk-free rate + Beta * (Benchmark Return - Risk-free rate))
	dailyRiskFreeRate := rc.riskFreeRate.Div(decimal.NewFromInt(252))
	expectedReturn := dailyRiskFreeRate.Add(beta.Mul(benchmarkAvg.Sub(dailyRiskFreeRate)))

	alpha := portfolioAvg.Sub(expectedReturn)

	// Annualize alpha
	return alpha.Mul(decimal.NewFromInt(252))
}

func (rc *RiskCalculator) calculateCalmarRatio(returns []decimal.Decimal, maxDrawdown decimal.Decimal) decimal.Decimal {
	if len(returns) == 0 || maxDrawdown.IsZero() {
		return decimal.Zero
	}

	// Calculate annualized return
	sum := decimal.Zero
	for _, ret := range returns {
		sum = sum.Add(ret)
	}
	annualizedReturn := sum.Div(decimal.NewFromInt(int64(len(returns)))).Mul(decimal.NewFromInt(252))

	return annualizedReturn.Div(maxDrawdown)
}

func (rc *RiskCalculator) calculateTreynorRatio(returns []decimal.Decimal, beta decimal.Decimal) decimal.Decimal {
	if len(returns) == 0 || beta.IsZero() {
		return decimal.Zero
	}

	// Calculate excess return
	sum := decimal.Zero
	for _, ret := range returns {
		sum = sum.Add(ret)
	}
	annualizedReturn := sum.Div(decimal.NewFromInt(int64(len(returns)))).Mul(decimal.NewFromInt(252))
	excessReturn := annualizedReturn.Sub(rc.riskFreeRate)

	return excessReturn.Div(beta)
}

func (rc *RiskCalculator) calculateInformationRatio(portfolioReturns, benchmarkReturns []decimal.Decimal) decimal.Decimal {
	if len(portfolioReturns) != len(benchmarkReturns) || len(portfolioReturns) < 2 {
		return decimal.Zero
	}

	// Calculate tracking error (active returns)
	activeReturns := make([]decimal.Decimal, len(portfolioReturns))
	for i := 0; i < len(portfolioReturns); i++ {
		activeReturns[i] = portfolioReturns[i].Sub(benchmarkReturns[i])
	}

	// Calculate mean active return
	sum := decimal.Zero
	for _, ret := range activeReturns {
		sum = sum.Add(ret)
	}
	meanActiveReturn := sum.Div(decimal.NewFromInt(int64(len(activeReturns))))

	// Calculate tracking error (standard deviation of active returns)
	variance := decimal.Zero
	for _, ret := range activeReturns {
		diff := ret.Sub(meanActiveReturn)
		variance = variance.Add(diff.Mul(diff))
	}
	variance = variance.Div(decimal.NewFromInt(int64(len(activeReturns) - 1)))

	varianceFloat, _ := variance.Float64()
	if varianceFloat <= 0 {
		return decimal.Zero
	}

	trackingError := decimal.NewFromFloat(math.Sqrt(varianceFloat))
	if trackingError.IsZero() {
		return decimal.Zero
	}

	// Annualized Information Ratio
	annualizedActiveReturn := meanActiveReturn.Mul(decimal.NewFromInt(252))
	annualizedTrackingError := trackingError.Mul(decimal.NewFromFloat(math.Sqrt(252)))

	return annualizedActiveReturn.Div(annualizedTrackingError)
}

func (rc *RiskCalculator) calculateUpsideDownsideDeviation(returns []decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	if len(returns) < 2 {
		return decimal.Zero, decimal.Zero
	}

	// Calculate mean return
	sum := decimal.Zero
	for _, ret := range returns {
		sum = sum.Add(ret)
	}
	meanReturn := sum.Div(decimal.NewFromInt(int64(len(returns))))

	upsideVariance := decimal.Zero
	downsideVariance := decimal.Zero
	upsideCount := 0
	downsideCount := 0

	for _, ret := range returns {
		if ret.GreaterThan(meanReturn) {
			diff := ret.Sub(meanReturn)
			upsideVariance = upsideVariance.Add(diff.Mul(diff))
			upsideCount++
		} else if ret.LessThan(meanReturn) {
			diff := ret.Sub(meanReturn)
			downsideVariance = downsideVariance.Add(diff.Mul(diff))
			downsideCount++
		}
	}

	var upsideDeviation, downsideDeviation decimal.Decimal

	if upsideCount > 0 {
		upsideVariance = upsideVariance.Div(decimal.NewFromInt(int64(upsideCount)))
		upsideVarianceFloat, _ := upsideVariance.Float64()
		if upsideVarianceFloat > 0 {
			upsideDeviation = decimal.NewFromFloat(math.Sqrt(upsideVarianceFloat))
		}
	}

	if downsideCount > 0 {
		downsideVariance = downsideVariance.Div(decimal.NewFromInt(int64(downsideCount)))
		downsideVarianceFloat, _ := downsideVariance.Float64()
		if downsideVarianceFloat > 0 {
			downsideDeviation = decimal.NewFromFloat(math.Sqrt(downsideVarianceFloat))
		}
	}

	// Annualize deviations
	upsideDeviation = upsideDeviation.Mul(decimal.NewFromFloat(math.Sqrt(252)))
	downsideDeviation = downsideDeviation.Mul(decimal.NewFromFloat(math.Sqrt(252)))

	return upsideDeviation, downsideDeviation
}

func (rc *RiskCalculator) SetRiskFreeRate(rate decimal.Decimal) {
	rc.riskFreeRate = rate
}

type PortfolioRiskProfile struct {
	RiskLevel      string          `json:"risk_level"`
	RiskScore      decimal.Decimal `json:"risk_score"`
	Description    string          `json:"description"`
	Recommendations []string       `json:"recommendations"`
}

func (rc *RiskCalculator) AssessRiskProfile(metrics *RiskMetricsResult) *PortfolioRiskProfile {
	profile := &PortfolioRiskProfile{
		Recommendations: make([]string, 0),
	}

	// Calculate composite risk score (0-100)
	volatilityScore := rc.normalizeVolatility(metrics.Volatility30d)
	drawdownScore := rc.normalizeDrawdown(metrics.MaxDrawdown)
	varScore := rc.normalizeVaR(metrics.VaR95)

	// Weighted composite score
	riskScore := volatilityScore.Mul(decimal.NewFromFloat(0.4)).
		Add(drawdownScore.Mul(decimal.NewFromFloat(0.3))).
		Add(varScore.Mul(decimal.NewFromFloat(0.3)))

	profile.RiskScore = riskScore

	// Determine risk level and recommendations
	riskScoreFloat, _ := riskScore.Float64()
	switch {
	case riskScoreFloat <= 20:
		profile.RiskLevel = "Conservative"
		profile.Description = "Low risk portfolio with stable returns and minimal drawdowns"
		profile.Recommendations = append(profile.Recommendations,
			"Portfolio shows conservative risk profile",
			"Consider increasing allocation to growth assets for higher returns",
			"Maintain current diversification strategy")

	case riskScoreFloat <= 40:
		profile.RiskLevel = "Moderate"
		profile.Description = "Balanced risk portfolio with moderate volatility"
		profile.Recommendations = append(profile.Recommendations,
			"Well-balanced risk/return profile",
			"Monitor correlation between holdings",
			"Consider rebalancing if concentration risk increases")

	case riskScoreFloat <= 60:
		profile.RiskLevel = "Moderate-Aggressive"
		profile.Description = "Higher volatility with potential for greater returns"
		profile.Recommendations = append(profile.Recommendations,
			"Above-average risk levels detected",
			"Consider reducing position sizes in volatile assets",
			"Implement stop-loss strategies")

	case riskScoreFloat <= 80:
		profile.RiskLevel = "Aggressive"
		profile.Description = "High volatility portfolio with significant risk exposure"
		profile.Recommendations = append(profile.Recommendations,
			"High risk portfolio requires active monitoring",
			"Consider diversifying across asset classes",
			"Implement risk management strategies")

	default:
		profile.RiskLevel = "Very Aggressive"
		profile.Description = "Very high risk with extreme volatility and potential for large losses"
		profile.Recommendations = append(profile.Recommendations,
			"Extremely high risk levels detected",
			"Immediate risk reduction recommended",
			"Consider reducing portfolio concentration",
			"Implement strict risk management protocols")
	}

	// Add specific recommendations based on metrics
	if metrics.MaxDrawdown.GreaterThan(decimal.NewFromFloat(0.2)) {
		profile.Recommendations = append(profile.Recommendations,
			fmt.Sprintf("Maximum drawdown of %.1f%% is concerning", metrics.MaxDrawdown.Mul(decimal.NewFromInt(100))))
	}

	if metrics.SharpeRatio.LessThan(decimal.NewFromFloat(0.5)) {
		profile.Recommendations = append(profile.Recommendations,
			"Low Sharpe ratio indicates poor risk-adjusted returns")
	}

	if metrics.Volatility30d.GreaterThan(decimal.NewFromFloat(0.5)) {
		profile.Recommendations = append(profile.Recommendations,
			"High volatility detected - consider reducing position sizes")
	}

	return profile
}

func (rc *RiskCalculator) normalizeVolatility(volatility decimal.Decimal) decimal.Decimal {
	// Normalize volatility to 0-100 scale (assuming 0-100% volatility range)
	normalized := volatility.Mul(decimal.NewFromInt(100))
	if normalized.GreaterThan(decimal.NewFromInt(100)) {
		return decimal.NewFromInt(100)
	}
	return normalized
}

func (rc *RiskCalculator) normalizeDrawdown(drawdown decimal.Decimal) decimal.Decimal {
	// Normalize drawdown to 0-100 scale
	normalized := drawdown.Mul(decimal.NewFromInt(100))
	if normalized.GreaterThan(decimal.NewFromInt(100)) {
		return decimal.NewFromInt(100)
	}
	return normalized
}

func (rc *RiskCalculator) normalizeVaR(var95 decimal.Decimal) decimal.Decimal {
	// Normalize VaR to 0-100 scale (assuming max 50% daily VaR)
	normalized := var95.Mul(decimal.NewFromInt(200))
	if normalized.GreaterThan(decimal.NewFromInt(100)) {
		return decimal.NewFromInt(100)
	}
	if normalized.LessThan(decimal.Zero) {
		return decimal.Zero
	}
	return normalized
}