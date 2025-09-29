package analytics

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/shopspring/decimal"

	"portfolio-api/internal/models"
)

type CorrelationAnalyzer struct{}

func NewCorrelationAnalyzer() *CorrelationAnalyzer {
	return &CorrelationAnalyzer{}
}

type CorrelationMatrix struct {
	Symbols     []string                       `json:"symbols"`
	Matrix      [][]decimal.Decimal            `json:"matrix"`
	Heatmap     map[string]map[string]decimal.Decimal `json:"heatmap"`
	Summary     CorrelationSummary             `json:"summary"`
	LastUpdated string                         `json:"last_updated"`
}

type CorrelationSummary struct {
	AverageCorrelation    decimal.Decimal `json:"average_correlation"`
	MaxCorrelation        decimal.Decimal `json:"max_correlation"`
	MinCorrelation        decimal.Decimal `json:"min_correlation"`
	HighlyCorrelatedPairs []CorrelationPair `json:"highly_correlated_pairs"`
	LowCorrelationPairs   []CorrelationPair `json:"low_correlation_pairs"`
}

type CorrelationPair struct {
	Symbol1     string          `json:"symbol1"`
	Symbol2     string          `json:"symbol2"`
	Correlation decimal.Decimal `json:"correlation"`
	Strength    string          `json:"strength"`
}

type HoldingPrice struct {
	Symbol string          `json:"symbol"`
	Date   string          `json:"date"`
	Price  decimal.Decimal `json:"price"`
}

func (ca *CorrelationAnalyzer) AnalyzeCorrelations(ctx context.Context, holdings []models.Holding, priceHistory [][]HoldingPrice) (*CorrelationMatrix, error) {
	if len(holdings) < 2 {
		return nil, fmt.Errorf("need at least 2 holdings for correlation analysis")
	}

	symbols := make([]string, len(holdings))
	for i, holding := range holdings {
		symbols[i] = holding.Symbol
	}

	// Create correlation matrix
	matrix := make([][]decimal.Decimal, len(symbols))
	heatmap := make(map[string]map[string]decimal.Decimal)

	for i := range symbols {
		matrix[i] = make([]decimal.Decimal, len(symbols))
		heatmap[symbols[i]] = make(map[string]decimal.Decimal)
	}

	// Calculate correlations
	correlations := make([]CorrelationPair, 0)
	correlationSum := decimal.Zero
	correlationCount := 0

	for i := 0; i < len(symbols); i++ {
		for j := 0; j < len(symbols); j++ {
			if i == j {
				// Self correlation is always 1
				matrix[i][j] = decimal.NewFromInt(1)
				heatmap[symbols[i]][symbols[j]] = decimal.NewFromInt(1)
			} else if i < j {
				// Calculate correlation between symbols[i] and symbols[j]
				correlation := ca.calculateCorrelation(symbols[i], symbols[j], priceHistory)
				matrix[i][j] = correlation
				matrix[j][i] = correlation // Symmetric matrix
				heatmap[symbols[i]][symbols[j]] = correlation
				heatmap[symbols[j]][symbols[i]] = correlation

				pair := CorrelationPair{
					Symbol1:     symbols[i],
					Symbol2:     symbols[j],
					Correlation: correlation,
					Strength:    ca.getCorrelationStrength(correlation),
				}
				correlations = append(correlations, pair)

				correlationSum = correlationSum.Add(correlation.Abs())
				correlationCount++
			}
		}
	}

	// Calculate summary statistics
	summary := ca.calculateSummary(correlations, correlationSum, correlationCount)

	return &CorrelationMatrix{
		Symbols: symbols,
		Matrix:  matrix,
		Heatmap: heatmap,
		Summary: summary,
	}, nil
}

func (ca *CorrelationAnalyzer) calculateCorrelation(symbol1, symbol2 string, priceHistory [][]HoldingPrice) decimal.Decimal {
	// Extract price series for both symbols
	prices1 := ca.extractPriceSeriesForSymbol(symbol1, priceHistory)
	prices2 := ca.extractPriceSeriesForSymbol(symbol2, priceHistory)

	if len(prices1) != len(prices2) || len(prices1) < 2 {
		return decimal.Zero
	}

	// Calculate returns
	returns1 := ca.calculateReturns(prices1)
	returns2 := ca.calculateReturns(prices2)

	if len(returns1) != len(returns2) || len(returns1) < 2 {
		return decimal.Zero
	}

	return ca.calculatePearsonCorrelation(returns1, returns2)
}

func (ca *CorrelationAnalyzer) extractPriceSeriesForSymbol(symbol string, priceHistory [][]HoldingPrice) []decimal.Decimal {
	var prices []decimal.Decimal

	for _, dayPrices := range priceHistory {
		for _, holdingPrice := range dayPrices {
			if holdingPrice.Symbol == symbol {
				prices = append(prices, holdingPrice.Price)
				break
			}
		}
	}

	return prices
}

func (ca *CorrelationAnalyzer) calculateReturns(prices []decimal.Decimal) []decimal.Decimal {
	if len(prices) < 2 {
		return nil
	}

	returns := make([]decimal.Decimal, 0, len(prices)-1)

	for i := 1; i < len(prices); i++ {
		if prices[i-1].IsZero() {
			continue
		}

		ret := prices[i].Sub(prices[i-1]).Div(prices[i-1])
		returns = append(returns, ret)
	}

	return returns
}

func (ca *CorrelationAnalyzer) calculatePearsonCorrelation(x, y []decimal.Decimal) decimal.Decimal {
	if len(x) != len(y) || len(x) < 2 {
		return decimal.Zero
	}

	n := decimal.NewFromInt(int64(len(x)))

	// Calculate means
	sumX := decimal.Zero
	sumY := decimal.Zero
	for i := 0; i < len(x); i++ {
		sumX = sumX.Add(x[i])
		sumY = sumY.Add(y[i])
	}
	meanX := sumX.Div(n)
	meanY := sumY.Div(n)

	// Calculate numerator and denominators
	numerator := decimal.Zero
	sumSquaredX := decimal.Zero
	sumSquaredY := decimal.Zero

	for i := 0; i < len(x); i++ {
		diffX := x[i].Sub(meanX)
		diffY := y[i].Sub(meanY)

		numerator = numerator.Add(diffX.Mul(diffY))
		sumSquaredX = sumSquaredX.Add(diffX.Mul(diffX))
		sumSquaredY = sumSquaredY.Add(diffY.Mul(diffY))
	}

	// Calculate correlation coefficient
	denominator := sumSquaredX.Mul(sumSquaredY)
	denominatorFloat, _ := denominator.Float64()

	if denominatorFloat <= 0 {
		return decimal.Zero
	}

	denominatorSqrt := decimal.NewFromFloat(math.Sqrt(denominatorFloat))
	if denominatorSqrt.IsZero() {
		return decimal.Zero
	}

	correlation := numerator.Div(denominatorSqrt)
	return correlation
}

func (ca *CorrelationAnalyzer) getCorrelationStrength(correlation decimal.Decimal) string {
	abs := correlation.Abs()
	absFloat, _ := abs.Float64()

	switch {
	case absFloat >= 0.9:
		return "Very Strong"
	case absFloat >= 0.7:
		return "Strong"
	case absFloat >= 0.5:
		return "Moderate"
	case absFloat >= 0.3:
		return "Weak"
	default:
		return "Very Weak"
	}
}

func (ca *CorrelationAnalyzer) calculateSummary(correlations []CorrelationPair, correlationSum decimal.Decimal, correlationCount int) CorrelationSummary {
	summary := CorrelationSummary{
		HighlyCorrelatedPairs: make([]CorrelationPair, 0),
		LowCorrelationPairs:   make([]CorrelationPair, 0),
	}

	if correlationCount > 0 {
		summary.AverageCorrelation = correlationSum.Div(decimal.NewFromInt(int64(correlationCount)))
	}

	if len(correlations) > 0 {
		// Sort correlations to find min and max
		sort.Slice(correlations, func(i, j int) bool {
			return correlations[i].Correlation.LessThan(correlations[j].Correlation)
		})

		summary.MinCorrelation = correlations[0].Correlation
		summary.MaxCorrelation = correlations[len(correlations)-1].Correlation

		// Find highly correlated pairs (correlation > 0.7)
		for _, pair := range correlations {
			absCorr := pair.Correlation.Abs()
			if absCorr.GreaterThan(decimal.NewFromFloat(0.7)) {
				summary.HighlyCorrelatedPairs = append(summary.HighlyCorrelatedPairs, pair)
			} else if absCorr.LessThan(decimal.NewFromFloat(0.3)) {
				summary.LowCorrelationPairs = append(summary.LowCorrelationPairs, pair)
			}
		}
	}

	return summary
}

type DiversificationScore struct {
	OverallScore          decimal.Decimal            `json:"overall_score"`
	ConcentrationRisk     decimal.Decimal            `json:"concentration_risk"`
	CorrelationRisk       decimal.Decimal            `json:"correlation_risk"`
	SectorDiversification SectorDiversification      `json:"sector_diversification"`
	Recommendations       []string                   `json:"recommendations"`
	RiskLevel            string                     `json:"risk_level"`
}

type SectorDiversification struct {
	SectorWeights map[string]decimal.Decimal `json:"sector_weights"`
	HerfindahlIndex decimal.Decimal         `json:"herfindahl_index"`
	EffectiveAssets decimal.Decimal         `json:"effective_assets"`
}

func (ca *CorrelationAnalyzer) CalculateDiversificationScore(ctx context.Context, holdings []models.Holding, correlationMatrix *CorrelationMatrix) (*DiversificationScore, error) {
	if len(holdings) == 0 {
		return nil, fmt.Errorf("no holdings to analyze")
	}

	score := &DiversificationScore{
		Recommendations: make([]string, 0),
	}

	// Calculate concentration risk
	score.ConcentrationRisk = ca.calculateConcentrationRisk(holdings)

	// Calculate correlation risk
	if correlationMatrix != nil {
		score.CorrelationRisk = ca.calculateCorrelationRisk(correlationMatrix)
	}

	// Calculate sector diversification
	score.SectorDiversification = ca.calculateSectorDiversification(holdings)

	// Calculate overall diversification score (0-100)
	concentrationScore := decimal.NewFromInt(100).Sub(score.ConcentrationRisk.Mul(decimal.NewFromInt(100)))
	correlationScore := decimal.NewFromInt(100).Sub(score.CorrelationRisk.Mul(decimal.NewFromInt(100)))
	sectorScore := decimal.NewFromInt(100).Sub(score.SectorDiversification.HerfindahlIndex.Mul(decimal.NewFromInt(100)))

	// Weighted average
	score.OverallScore = concentrationScore.Mul(decimal.NewFromFloat(0.4)).
		Add(correlationScore.Mul(decimal.NewFromFloat(0.3))).
		Add(sectorScore.Mul(decimal.NewFromFloat(0.3)))

	// Generate recommendations
	score.Recommendations = ca.generateDiversificationRecommendations(score)

	// Determine risk level
	score.RiskLevel = ca.getDiversificationRiskLevel(score.OverallScore)

	return score, nil
}

func (ca *CorrelationAnalyzer) calculateConcentrationRisk(holdings []models.Holding) decimal.Decimal {
	if len(holdings) == 0 {
		return decimal.NewFromInt(1) // Maximum concentration risk
	}

	// Calculate total portfolio value
	totalValue := decimal.Zero
	for _, holding := range holdings {
		totalValue = totalValue.Add(holding.CurrentValue)
	}

	if totalValue.IsZero() {
		return decimal.NewFromInt(1)
	}

	// Calculate Herfindahl-Hirschman Index for concentration
	hhi := decimal.Zero
	for _, holding := range holdings {
		weight := holding.CurrentValue.Div(totalValue)
		hhi = hhi.Add(weight.Mul(weight))
	}

	return hhi
}

func (ca *CorrelationAnalyzer) calculateCorrelationRisk(correlationMatrix *CorrelationMatrix) decimal.Decimal {
	if correlationMatrix == nil || len(correlationMatrix.Summary.HighlyCorrelatedPairs) == 0 {
		return decimal.Zero
	}

	// Average absolute correlation as risk measure
	return correlationMatrix.Summary.AverageCorrelation.Abs()
}

func (ca *CorrelationAnalyzer) calculateSectorDiversification(holdings []models.Holding) SectorDiversification {
	sectorWeights := make(map[string]decimal.Decimal)
	totalValue := decimal.Zero

	// Calculate total value
	for _, holding := range holdings {
		totalValue = totalValue.Add(holding.CurrentValue)
	}

	// Calculate sector weights
	for _, holding := range holdings {
		sector := holding.Category
		if sector == "" {
			sector = "Unknown"
		}

		weight := decimal.Zero
		if !totalValue.IsZero() {
			weight = holding.CurrentValue.Div(totalValue)
		}

		if existingWeight, exists := sectorWeights[sector]; exists {
			sectorWeights[sector] = existingWeight.Add(weight)
		} else {
			sectorWeights[sector] = weight
		}
	}

	// Calculate Herfindahl Index for sectors
	hhi := decimal.Zero
	for _, weight := range sectorWeights {
		hhi = hhi.Add(weight.Mul(weight))
	}

	// Calculate effective number of assets
	effectiveAssets := decimal.Zero
	if !hhi.IsZero() {
		effectiveAssets = decimal.NewFromInt(1).Div(hhi)
	}

	return SectorDiversification{
		SectorWeights:   sectorWeights,
		HerfindahlIndex: hhi,
		EffectiveAssets: effectiveAssets,
	}
}

func (ca *CorrelationAnalyzer) generateDiversificationRecommendations(score *DiversificationScore) []string {
	recommendations := make([]string, 0)

	concentrationFloat, _ := score.ConcentrationRisk.Float64()
	correlationFloat, _ := score.CorrelationRisk.Float64()

	// Concentration risk recommendations
	if concentrationFloat > 0.4 {
		recommendations = append(recommendations, "High concentration risk detected - consider reducing position sizes")
	}

	if concentrationFloat > 0.25 {
		recommendations = append(recommendations, "Consider adding more holdings to reduce concentration")
	}

	// Correlation risk recommendations
	if correlationFloat > 0.7 {
		recommendations = append(recommendations, "Many holdings are highly correlated - consider diversifying into different asset classes")
	}

	// Sector diversification recommendations
	if len(score.SectorDiversification.SectorWeights) < 3 {
		recommendations = append(recommendations, "Limited sector diversification - consider adding holdings from different sectors")
	}

	effectiveAssetsFloat, _ := score.SectorDiversification.EffectiveAssets.Float64()
	if effectiveAssetsFloat < 5 {
		recommendations = append(recommendations, "Low effective number of assets - portfolio may not be well diversified")
	}

	// Overall score recommendations
	overallScoreFloat, _ := score.OverallScore.Float64()
	if overallScoreFloat < 50 {
		recommendations = append(recommendations, "Overall diversification is poor - consider a comprehensive portfolio review")
	} else if overallScoreFloat < 70 {
		recommendations = append(recommendations, "Diversification could be improved - focus on reducing concentration and correlation risks")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Portfolio shows good diversification characteristics")
	}

	return recommendations
}

func (ca *CorrelationAnalyzer) getDiversificationRiskLevel(overallScore decimal.Decimal) string {
	scoreFloat, _ := overallScore.Float64()

	switch {
	case scoreFloat >= 80:
		return "Low Risk - Well Diversified"
	case scoreFloat >= 60:
		return "Moderate Risk - Adequately Diversified"
	case scoreFloat >= 40:
		return "High Risk - Poorly Diversified"
	default:
		return "Very High Risk - Concentrated Portfolio"
	}
}

type VolatilityClustering struct {
	Periods         []VolatilityPeriod `json:"periods"`
	CurrentCluster  string             `json:"current_cluster"`
	ClusterAnalysis ClusterAnalysis    `json:"cluster_analysis"`
}

type VolatilityPeriod struct {
	StartDate   string          `json:"start_date"`
	EndDate     string          `json:"end_date"`
	Volatility  decimal.Decimal `json:"volatility"`
	ClusterType string          `json:"cluster_type"`
}

type ClusterAnalysis struct {
	LowVolatilityPeriods    int `json:"low_volatility_periods"`
	HighVolatilityPeriods   int `json:"high_volatility_periods"`
	AverageClusterDuration  int `json:"average_cluster_duration"`
	VolatilityPersistence   decimal.Decimal `json:"volatility_persistence"`
}

func (ca *CorrelationAnalyzer) AnalyzeVolatilityClustering(ctx context.Context, snapshots []models.Snapshot) (*VolatilityClustering, error) {
	if len(snapshots) < 10 {
		return nil, fmt.Errorf("insufficient data for volatility clustering analysis")
	}

	// Calculate rolling volatilities
	volatilities := ca.calculateRollingVolatilities(snapshots, 7) // 7-day rolling volatility

	// Identify clusters
	periods := ca.identifyVolatilityClusters(volatilities, snapshots)

	// Analyze clusters
	analysis := ca.analyzeVolatilityClusters(periods)

	// Determine current cluster
	currentCluster := "Normal"
	if len(periods) > 0 {
		currentCluster = periods[len(periods)-1].ClusterType
	}

	return &VolatilityClustering{
		Periods:         periods,
		CurrentCluster:  currentCluster,
		ClusterAnalysis: analysis,
	}, nil
}

func (ca *CorrelationAnalyzer) calculateRollingVolatilities(snapshots []models.Snapshot, window int) []decimal.Decimal {
	if len(snapshots) < window {
		return nil
	}

	volatilities := make([]decimal.Decimal, 0, len(snapshots)-window+1)

	for i := window - 1; i < len(snapshots); i++ {
		windowSnapshots := snapshots[i-window+1 : i+1]
		volatility := ca.calculateWindowVolatility(windowSnapshots)
		volatilities = append(volatilities, volatility)
	}

	return volatilities
}

func (ca *CorrelationAnalyzer) calculateWindowVolatility(snapshots []models.Snapshot) decimal.Decimal {
	if len(snapshots) < 2 {
		return decimal.Zero
	}

	returns := make([]decimal.Decimal, 0, len(snapshots)-1)

	for i := 1; i < len(snapshots); i++ {
		prevValue := snapshots[i-1].Value.Total
		currentValue := snapshots[i].Value.Total

		if prevValue.IsZero() {
			continue
		}

		ret := currentValue.Sub(prevValue).Div(prevValue)
		returns = append(returns, ret)
	}

	if len(returns) < 2 {
		return decimal.Zero
	}

	// Calculate standard deviation
	sum := decimal.Zero
	for _, ret := range returns {
		sum = sum.Add(ret)
	}
	mean := sum.Div(decimal.NewFromInt(int64(len(returns))))

	variance := decimal.Zero
	for _, ret := range returns {
		diff := ret.Sub(mean)
		variance = variance.Add(diff.Mul(diff))
	}
	variance = variance.Div(decimal.NewFromInt(int64(len(returns) - 1)))

	varianceFloat, _ := variance.Float64()
	if varianceFloat <= 0 {
		return decimal.Zero
	}

	return decimal.NewFromFloat(math.Sqrt(varianceFloat))
}

func (ca *CorrelationAnalyzer) identifyVolatilityClusters(volatilities []decimal.Decimal, snapshots []models.Snapshot) []VolatilityPeriod {
	if len(volatilities) == 0 {
		return nil
	}

	// Calculate threshold for high/low volatility
	sum := decimal.Zero
	for _, vol := range volatilities {
		sum = sum.Add(vol)
	}
	avgVolatility := sum.Div(decimal.NewFromInt(int64(len(volatilities))))

	periods := make([]VolatilityPeriod, 0)
	currentPeriod := VolatilityPeriod{}
	currentCluster := ""

	for i, volatility := range volatilities {
		snapshotIndex := i + 6 // Adjust for rolling window offset

		clusterType := "Normal"
		if volatility.GreaterThan(avgVolatility.Mul(decimal.NewFromFloat(1.5))) {
			clusterType = "High"
		} else if volatility.LessThan(avgVolatility.Mul(decimal.NewFromFloat(0.5))) {
			clusterType = "Low"
		}

		if currentCluster == "" {
			// Start first period
			currentPeriod = VolatilityPeriod{
				StartDate:   snapshots[snapshotIndex].Timestamp.Format("2006-01-02"),
				Volatility:  volatility,
				ClusterType: clusterType,
			}
			currentCluster = clusterType
		} else if clusterType != currentCluster {
			// End current period and start new one
			currentPeriod.EndDate = snapshots[snapshotIndex-1].Timestamp.Format("2006-01-02")
			periods = append(periods, currentPeriod)

			currentPeriod = VolatilityPeriod{
				StartDate:   snapshots[snapshotIndex].Timestamp.Format("2006-01-02"),
				Volatility:  volatility,
				ClusterType: clusterType,
			}
			currentCluster = clusterType
		}

		// Update current period volatility (could be average or latest)
		currentPeriod.Volatility = volatility
	}

	// Close last period
	if len(snapshots) > 0 {
		currentPeriod.EndDate = snapshots[len(snapshots)-1].Timestamp.Format("2006-01-02")
		periods = append(periods, currentPeriod)
	}

	return periods
}

func (ca *CorrelationAnalyzer) analyzeVolatilityClusters(periods []VolatilityPeriod) ClusterAnalysis {
	analysis := ClusterAnalysis{}

	lowCount := 0
	highCount := 0
	totalDuration := 0

	for _, period := range periods {
		switch period.ClusterType {
		case "Low":
			lowCount++
		case "High":
			highCount++
		}

		// Calculate duration (simplified - could be enhanced with actual date parsing)
		totalDuration++
	}

	analysis.LowVolatilityPeriods = lowCount
	analysis.HighVolatilityPeriods = highCount

	if len(periods) > 0 {
		analysis.AverageClusterDuration = totalDuration / len(periods)
	}

	// Calculate volatility persistence (simplified measure)
	if len(periods) > 1 {
		persistentClusters := 0
		for i := 1; i < len(periods); i++ {
			if periods[i].ClusterType == periods[i-1].ClusterType {
				persistentClusters++
			}
		}
		analysis.VolatilityPersistence = decimal.NewFromInt(int64(persistentClusters)).Div(decimal.NewFromInt(int64(len(periods) - 1)))
	}

	return analysis
}