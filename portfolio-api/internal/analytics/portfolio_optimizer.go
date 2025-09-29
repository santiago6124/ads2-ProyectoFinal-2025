package analytics

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/shopspring/decimal"

	"portfolio-api/internal/models"
)

type PortfolioOptimizer struct {
	riskFreeRate decimal.Decimal
}

func NewPortfolioOptimizer(riskFreeRate decimal.Decimal) *PortfolioOptimizer {
	return &PortfolioOptimizer{
		riskFreeRate: riskFreeRate,
	}
}

type OptimizationStrategy string

const (
	StrategyMaxSharpe      OptimizationStrategy = "max_sharpe"
	StrategyMinVariance    OptimizationStrategy = "min_variance"
	StrategyMaxReturn      OptimizationStrategy = "max_return"
	StrategyRiskParity     OptimizationStrategy = "risk_parity"
	StrategyEqualWeight    OptimizationStrategy = "equal_weight"
	StrategyCustomTarget   OptimizationStrategy = "custom_target"
)

type OptimizationConstraints struct {
	MaxWeight         decimal.Decimal            `json:"max_weight"`
	MinWeight         decimal.Decimal            `json:"min_weight"`
	MaxSectorWeight   decimal.Decimal            `json:"max_sector_weight"`
	MinSectorWeight   decimal.Decimal            `json:"min_sector_weight"`
	RequiredHoldings  []string                   `json:"required_holdings"`
	ExcludedHoldings  []string                   `json:"excluded_holdings"`
	SectorLimits      map[string]decimal.Decimal `json:"sector_limits"`
	TurnoverLimit     decimal.Decimal            `json:"turnover_limit"`
	TransactionCosts  decimal.Decimal            `json:"transaction_costs"`
}

type OptimizationResult struct {
	Strategy           OptimizationStrategy    `json:"strategy"`
	TargetWeights      map[string]decimal.Decimal `json:"target_weights"`
	CurrentWeights     map[string]decimal.Decimal `json:"current_weights"`
	RebalancingActions []RebalancingAction     `json:"rebalancing_actions"`
	ExpectedReturn     decimal.Decimal         `json:"expected_return"`
	ExpectedVolatility decimal.Decimal         `json:"expected_volatility"`
	ExpectedSharpe     decimal.Decimal         `json:"expected_sharpe"`
	TotalTurnover      decimal.Decimal         `json:"total_turnover"`
	EstimatedCosts     decimal.Decimal         `json:"estimated_costs"`
	Recommendation     string                  `json:"recommendation"`
}

type RebalancingAction struct {
	Symbol      string          `json:"symbol"`
	Action      string          `json:"action"` // "buy", "sell", "hold"
	CurrentQty  decimal.Decimal `json:"current_quantity"`
	TargetQty   decimal.Decimal `json:"target_quantity"`
	DeltaQty    decimal.Decimal `json:"delta_quantity"`
	CurrentValue decimal.Decimal `json:"current_value"`
	TargetValue  decimal.Decimal `json:"target_value"`
	DeltaValue   decimal.Decimal `json:"delta_value"`
	Priority     int             `json:"priority"`
}

type AssetAllocation struct {
	Symbols           []string               `json:"symbols"`
	ExpectedReturns   []decimal.Decimal      `json:"expected_returns"`
	CovarianceMatrix  [][]decimal.Decimal    `json:"covariance_matrix"`
	OptimalWeights    []decimal.Decimal      `json:"optimal_weights"`
	EfficientFrontier []EfficientPoint       `json:"efficient_frontier"`
}

type EfficientPoint struct {
	ExpectedReturn decimal.Decimal `json:"expected_return"`
	Volatility     decimal.Decimal `json:"volatility"`
	SharpeRatio    decimal.Decimal `json:"sharpe_ratio"`
	Weights        []decimal.Decimal `json:"weights"`
}

func (po *PortfolioOptimizer) OptimizePortfolio(ctx context.Context, portfolio *models.Portfolio, strategy OptimizationStrategy, constraints *OptimizationConstraints) (*OptimizationResult, error) {
	if portfolio == nil || len(portfolio.Holdings) == 0 {
		return nil, fmt.Errorf("portfolio or holdings cannot be empty")
	}

	if constraints == nil {
		constraints = po.getDefaultConstraints()
	}

	// Calculate current weights
	currentWeights := po.calculateCurrentWeights(portfolio)

	// Calculate expected returns and covariance matrix
	expectedReturns, covarianceMatrix := po.estimateReturnsAndCovariance(portfolio)

	// Optimize based on strategy
	targetWeights, err := po.optimizeWeights(strategy, expectedReturns, covarianceMatrix, constraints)
	if err != nil {
		return nil, fmt.Errorf("optimization failed: %w", err)
	}

	// Generate rebalancing actions
	actions := po.generateRebalancingActions(portfolio, currentWeights, targetWeights)

	// Calculate metrics
	expectedReturn := po.calculateExpectedReturn(targetWeights, expectedReturns)
	expectedVolatility := po.calculateExpectedVolatility(targetWeights, covarianceMatrix)
	expectedSharpe := po.calculateSharpeRatio(expectedReturn, expectedVolatility)
	totalTurnover := po.calculateTurnover(currentWeights, targetWeights)
	estimatedCosts := totalTurnover.Mul(constraints.TransactionCosts)

	result := &OptimizationResult{
		Strategy:           strategy,
		TargetWeights:      targetWeights,
		CurrentWeights:     currentWeights,
		RebalancingActions: actions,
		ExpectedReturn:     expectedReturn,
		ExpectedVolatility: expectedVolatility,
		ExpectedSharpe:     expectedSharpe,
		TotalTurnover:      totalTurnover,
		EstimatedCosts:     estimatedCosts,
		Recommendation:     po.generateRecommendation(strategy, totalTurnover, expectedSharpe),
	}

	return result, nil
}

func (po *PortfolioOptimizer) getDefaultConstraints() *OptimizationConstraints {
	return &OptimizationConstraints{
		MaxWeight:        decimal.NewFromFloat(0.4),  // Max 40% in any single asset
		MinWeight:        decimal.NewFromFloat(0.01), // Min 1% in any asset
		MaxSectorWeight:  decimal.NewFromFloat(0.6),  // Max 60% in any sector
		MinSectorWeight:  decimal.NewFromFloat(0.05), // Min 5% in any sector
		TurnoverLimit:    decimal.NewFromFloat(0.5),  // Max 50% turnover
		TransactionCosts: decimal.NewFromFloat(0.001), // 0.1% transaction costs
	}
}

func (po *PortfolioOptimizer) calculateCurrentWeights(portfolio *models.Portfolio) map[string]decimal.Decimal {
	weights := make(map[string]decimal.Decimal)
	totalValue := portfolio.TotalValue

	if totalValue.IsZero() {
		return weights
	}

	for _, holding := range portfolio.Holdings {
		weight := holding.CurrentValue.Div(totalValue)
		weights[holding.Symbol] = weight
	}

	return weights
}

func (po *PortfolioOptimizer) estimateReturnsAndCovariance(portfolio *models.Portfolio) ([]decimal.Decimal, [][]decimal.Decimal) {
	n := len(portfolio.Holdings)
	expectedReturns := make([]decimal.Decimal, n)
	covarianceMatrix := make([][]decimal.Decimal, n)

	for i := range covarianceMatrix {
		covarianceMatrix[i] = make([]decimal.Decimal, n)
	}

	// Estimate expected returns based on historical performance
	for i, holding := range portfolio.Holdings {
		// Simple estimation: use recent performance or market cap weighted return
		expectedReturns[i] = holding.ProfitLossPercentage.Div(decimal.NewFromInt(252)) // Annualized daily return

		// If no historical data, use risk-free rate + risk premium
		if expectedReturns[i].IsZero() {
			expectedReturns[i] = po.riskFreeRate.Add(decimal.NewFromFloat(0.05)).Div(decimal.NewFromInt(252))
		}
	}

	// Estimate covariance matrix (simplified)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				// Variance estimation based on asset volatility
				variance := decimal.NewFromFloat(0.16) // Default 40% annual volatility squared
				if portfolio.Holdings[i].Category == "crypto" {
					variance = decimal.NewFromFloat(0.64) // Higher volatility for crypto
				}
				covarianceMatrix[i][j] = variance.Div(decimal.NewFromInt(252)) // Daily variance
			} else {
				// Correlation estimation (simplified)
				correlation := decimal.NewFromFloat(0.3) // Default moderate correlation
				if portfolio.Holdings[i].Category == portfolio.Holdings[j].Category {
					correlation = decimal.NewFromFloat(0.6) // Higher correlation within same category
				}

				// Covariance = correlation * sqrt(var_i * var_j)
				var_i := covarianceMatrix[i][i]
				var_j := covarianceMatrix[j][j]
				covariance := correlation.Mul(var_i.Mul(var_j).Sqrt())
				covarianceMatrix[i][j] = covariance
			}
		}
	}

	return expectedReturns, covarianceMatrix
}

func (po *PortfolioOptimizer) optimizeWeights(strategy OptimizationStrategy, expectedReturns []decimal.Decimal, covarianceMatrix [][]decimal.Decimal, constraints *OptimizationConstraints) (map[string]decimal.Decimal, error) {
	n := len(expectedReturns)
	weights := make(map[string]decimal.Decimal)

	switch strategy {
	case StrategyEqualWeight:
		// Equal weight allocation
		equalWeight := decimal.NewFromInt(1).Div(decimal.NewFromInt(int64(n)))
		for i := 0; i < n; i++ {
			weights[fmt.Sprintf("asset_%d", i)] = equalWeight
		}

	case StrategyMinVariance:
		// Minimum variance portfolio (simplified)
		weights = po.calculateMinVarianceWeights(covarianceMatrix, constraints)

	case StrategyMaxSharpe:
		// Maximum Sharpe ratio portfolio (simplified)
		weights = po.calculateMaxSharpeWeights(expectedReturns, covarianceMatrix, constraints)

	case StrategyRiskParity:
		// Risk parity allocation
		weights = po.calculateRiskParityWeights(covarianceMatrix, constraints)

	default:
		// Default to equal weight
		equalWeight := decimal.NewFromInt(1).Div(decimal.NewFromInt(int64(n)))
		for i := 0; i < n; i++ {
			weights[fmt.Sprintf("asset_%d", i)] = equalWeight
		}
	}

	return weights, nil
}

func (po *PortfolioOptimizer) calculateMinVarianceWeights(covarianceMatrix [][]decimal.Decimal, constraints *OptimizationConstraints) map[string]decimal.Decimal {
	n := len(covarianceMatrix)
	weights := make(map[string]decimal.Decimal)

	// Simplified minimum variance: inverse variance weighting
	variances := make([]decimal.Decimal, n)
	sumInverseVar := decimal.Zero

	for i := 0; i < n; i++ {
		variance := covarianceMatrix[i][i]
		if variance.GreaterThan(decimal.Zero) {
			inverseVar := decimal.NewFromInt(1).Div(variance)
			variances[i] = inverseVar
			sumInverseVar = sumInverseVar.Add(inverseVar)
		}
	}

	// Normalize weights
	for i := 0; i < n; i++ {
		if sumInverseVar.GreaterThan(decimal.Zero) {
			weight := variances[i].Div(sumInverseVar)

			// Apply constraints
			if weight.GreaterThan(constraints.MaxWeight) {
				weight = constraints.MaxWeight
			} else if weight.LessThan(constraints.MinWeight) {
				weight = constraints.MinWeight
			}

			weights[fmt.Sprintf("asset_%d", i)] = weight
		}
	}

	return po.normalizeWeights(weights)
}

func (po *PortfolioOptimizer) calculateMaxSharpeWeights(expectedReturns []decimal.Decimal, covarianceMatrix [][]decimal.Decimal, constraints *OptimizationConstraints) map[string]decimal.Decimal {
	n := len(expectedReturns)
	weights := make(map[string]decimal.Decimal)

	// Simplified max Sharpe: weight by excess return / variance
	excessReturns := make([]decimal.Decimal, n)
	sumWeighted := decimal.Zero

	dailyRiskFreeRate := po.riskFreeRate.Div(decimal.NewFromInt(252))

	for i := 0; i < n; i++ {
		excessReturn := expectedReturns[i].Sub(dailyRiskFreeRate)
		variance := covarianceMatrix[i][i]

		if variance.GreaterThan(decimal.Zero) && excessReturn.GreaterThan(decimal.Zero) {
			weightedReturn := excessReturn.Div(variance)
			excessReturns[i] = weightedReturn
			sumWeighted = sumWeighted.Add(weightedReturn)
		}
	}

	// Normalize weights
	for i := 0; i < n; i++ {
		if sumWeighted.GreaterThan(decimal.Zero) {
			weight := excessReturns[i].Div(sumWeighted)

			// Apply constraints
			if weight.GreaterThan(constraints.MaxWeight) {
				weight = constraints.MaxWeight
			} else if weight.LessThan(constraints.MinWeight) {
				weight = constraints.MinWeight
			}

			weights[fmt.Sprintf("asset_%d", i)] = weight
		}
	}

	return po.normalizeWeights(weights)
}

func (po *PortfolioOptimizer) calculateRiskParityWeights(covarianceMatrix [][]decimal.Decimal, constraints *OptimizationConstraints) map[string]decimal.Decimal {
	n := len(covarianceMatrix)
	weights := make(map[string]decimal.Decimal)

	// Simplified risk parity: inverse volatility weighting
	volatilities := make([]decimal.Decimal, n)
	sumInverseVol := decimal.Zero

	for i := 0; i < n; i++ {
		variance := covarianceMatrix[i][i]
		if variance.GreaterThan(decimal.Zero) {
			volatility := variance.Sqrt()
			inverseVol := decimal.NewFromInt(1).Div(volatility)
			volatilities[i] = inverseVol
			sumInverseVol = sumInverseVol.Add(inverseVol)
		}
	}

	// Normalize weights
	for i := 0; i < n; i++ {
		if sumInverseVol.GreaterThan(decimal.Zero) {
			weight := volatilities[i].Div(sumInverseVol)

			// Apply constraints
			if weight.GreaterThan(constraints.MaxWeight) {
				weight = constraints.MaxWeight
			} else if weight.LessThan(constraints.MinWeight) {
				weight = constraints.MinWeight
			}

			weights[fmt.Sprintf("asset_%d", i)] = weight
		}
	}

	return po.normalizeWeights(weights)
}

func (po *PortfolioOptimizer) normalizeWeights(weights map[string]decimal.Decimal) map[string]decimal.Decimal {
	total := decimal.Zero
	for _, weight := range weights {
		total = total.Add(weight)
	}

	if total.IsZero() {
		return weights
	}

	normalized := make(map[string]decimal.Decimal)
	for symbol, weight := range weights {
		normalized[symbol] = weight.Div(total)
	}

	return normalized
}

func (po *PortfolioOptimizer) generateRebalancingActions(portfolio *models.Portfolio, currentWeights, targetWeights map[string]decimal.Decimal) []RebalancingAction {
	actions := make([]RebalancingAction, 0)
	totalValue := portfolio.TotalValue

	// Create map of holdings by symbol
	holdingsMap := make(map[string]*models.Holding)
	for i := range portfolio.Holdings {
		holdingsMap[portfolio.Holdings[i].Symbol] = &portfolio.Holdings[i]
	}

	// Generate actions for existing holdings
	for symbol, currentWeight := range currentWeights {
		targetWeight := targetWeights[symbol]
		if targetWeight.IsZero() {
			continue
		}

		holding := holdingsMap[symbol]
		if holding == nil {
			continue
		}

		currentValue := holding.CurrentValue
		targetValue := targetWeight.Mul(totalValue)
		deltaValue := targetValue.Sub(currentValue)

		action := "hold"
		if deltaValue.GreaterThan(decimal.Zero) {
			action = "buy"
		} else if deltaValue.LessThan(decimal.Zero) {
			action = "sell"
		}

		// Calculate quantity changes
		currentQty := holding.Quantity
		var targetQty, deltaQty decimal.Decimal

		if !holding.CurrentPrice.IsZero() {
			targetQty = targetValue.Div(holding.CurrentPrice)
			deltaQty = targetQty.Sub(currentQty)
		}

		actions = append(actions, RebalancingAction{
			Symbol:       symbol,
			Action:       action,
			CurrentQty:   currentQty,
			TargetQty:    targetQty,
			DeltaQty:     deltaQty,
			CurrentValue: currentValue,
			TargetValue:  targetValue,
			DeltaValue:   deltaValue,
			Priority:     po.calculateActionPriority(deltaValue, currentValue),
		})
	}

	// Sort actions by priority (largest trades first)
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Priority > actions[j].Priority
	})

	return actions
}

func (po *PortfolioOptimizer) calculateActionPriority(deltaValue, currentValue decimal.Decimal) int {
	if currentValue.IsZero() {
		return 0
	}

	changePercent := deltaValue.Abs().Div(currentValue)
	changePercentFloat, _ := changePercent.Float64()

	return int(changePercentFloat * 100)
}

func (po *PortfolioOptimizer) calculateExpectedReturn(weights map[string]decimal.Decimal, expectedReturns []decimal.Decimal) decimal.Decimal {
	expectedReturn := decimal.Zero
	i := 0

	for _, weight := range weights {
		if i < len(expectedReturns) {
			expectedReturn = expectedReturn.Add(weight.Mul(expectedReturns[i]))
			i++
		}
	}

	return expectedReturn.Mul(decimal.NewFromInt(252)) // Annualize
}

func (po *PortfolioOptimizer) calculateExpectedVolatility(weights map[string]decimal.Decimal, covarianceMatrix [][]decimal.Decimal) decimal.Decimal {
	n := len(weights)
	if n != len(covarianceMatrix) {
		return decimal.Zero
	}

	weightsSlice := make([]decimal.Decimal, 0, len(weights))
	for _, weight := range weights {
		weightsSlice = append(weightsSlice, weight)
	}

	// Calculate portfolio variance: w^T * Σ * w
	variance := decimal.Zero

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i < len(weightsSlice) && j < len(weightsSlice) && i < len(covarianceMatrix) && j < len(covarianceMatrix[i]) {
				variance = variance.Add(weightsSlice[i].Mul(weightsSlice[j]).Mul(covarianceMatrix[i][j]))
			}
		}
	}

	varianceFloat, _ := variance.Float64()
	if varianceFloat <= 0 {
		return decimal.Zero
	}

	// Annualize volatility
	dailyVolatility := decimal.NewFromFloat(math.Sqrt(varianceFloat))
	return dailyVolatility.Mul(decimal.NewFromFloat(math.Sqrt(252)))
}

func (po *PortfolioOptimizer) calculateSharpeRatio(expectedReturn, expectedVolatility decimal.Decimal) decimal.Decimal {
	if expectedVolatility.IsZero() {
		return decimal.Zero
	}

	excessReturn := expectedReturn.Sub(po.riskFreeRate)
	return excessReturn.Div(expectedVolatility)
}

func (po *PortfolioOptimizer) calculateTurnover(currentWeights, targetWeights map[string]decimal.Decimal) decimal.Decimal {
	turnover := decimal.Zero

	// Calculate sum of absolute weight differences
	for symbol := range currentWeights {
		currentWeight := currentWeights[symbol]
		targetWeight := targetWeights[symbol]

		diff := targetWeight.Sub(currentWeight).Abs()
		turnover = turnover.Add(diff)
	}

	// Add weights for new positions
	for symbol := range targetWeights {
		if _, exists := currentWeights[symbol]; !exists {
			turnover = turnover.Add(targetWeights[symbol])
		}
	}

	return turnover.Div(decimal.NewFromInt(2)) // Divide by 2 as turnover counts both buys and sells
}

func (po *PortfolioOptimizer) generateRecommendation(strategy OptimizationStrategy, turnover, expectedSharpe decimal.Decimal) string {
	turnoverFloat, _ := turnover.Float64()
	sharpeFloat, _ := expectedSharpe.Float64()

	if turnoverFloat < 0.05 {
		return "Portfolio is well-balanced. Minor adjustments recommended."
	}

	if turnoverFloat > 0.3 {
		recommendation := "Significant rebalancing required. "
		if sharpeFloat > 1.0 {
			recommendation += "High expected Sharpe ratio justifies the rebalancing costs."
		} else {
			recommendation += "Consider the transaction costs before implementing all changes."
		}
		return recommendation
	}

	switch strategy {
	case StrategyMaxSharpe:
		return "Rebalancing to maximize risk-adjusted returns. Monitor implementation costs."
	case StrategyMinVariance:
		return "Conservative rebalancing to reduce portfolio risk."
	case StrategyRiskParity:
		return "Risk parity allocation to balance risk contributions across holdings."
	default:
		return "Portfolio optimization completed. Review suggested changes carefully."
	}
}

type RebalancingSchedule struct {
	Frequency    string                     `json:"frequency"`
	NextDate     string                     `json:"next_date"`
	ThresholdPct decimal.Decimal            `json:"threshold_percentage"`
	Actions      []ScheduledAction          `json:"actions"`
	Triggers     []RebalancingTrigger       `json:"triggers"`
}

type ScheduledAction struct {
	Date        string          `json:"date"`
	Symbol      string          `json:"symbol"`
	Action      string          `json:"action"`
	Amount      decimal.Decimal `json:"amount"`
	Reason      string          `json:"reason"`
}

type RebalancingTrigger struct {
	Type        string          `json:"type"`
	Threshold   decimal.Decimal `json:"threshold"`
	CurrentValue decimal.Decimal `json:"current_value"`
	Triggered   bool            `json:"triggered"`
	Description string          `json:"description"`
}

func (po *PortfolioOptimizer) CreateRebalancingSchedule(ctx context.Context, portfolio *models.Portfolio, frequency string, thresholdPct decimal.Decimal) (*RebalancingSchedule, error) {
	schedule := &RebalancingSchedule{
		Frequency:    frequency,
		ThresholdPct: thresholdPct,
		Actions:      make([]ScheduledAction, 0),
		Triggers:     make([]RebalancingTrigger, 0),
	}

	// Calculate triggers
	triggers := po.calculateRebalancingTriggers(portfolio, thresholdPct)
	schedule.Triggers = triggers

	// Determine if immediate rebalancing is needed
	needsRebalancing := false
	for _, trigger := range triggers {
		if trigger.Triggered {
			needsRebalancing = true
			break
		}
	}

	if needsRebalancing {
		schedule.NextDate = "immediate"

		// Generate optimization with equal weight strategy for scheduled rebalancing
		result, err := po.OptimizePortfolio(ctx, portfolio, StrategyEqualWeight, po.getDefaultConstraints())
		if err == nil {
			for _, action := range result.RebalancingActions {
				if action.Action != "hold" {
					schedule.Actions = append(schedule.Actions, ScheduledAction{
						Date:   "immediate",
						Symbol: action.Symbol,
						Action: action.Action,
						Amount: action.DeltaValue.Abs(),
						Reason: "Threshold-based rebalancing",
					})
				}
			}
		}
	} else {
		// Schedule next rebalancing based on frequency
		schedule.NextDate = po.calculateNextRebalancingDate(frequency)
	}

	return schedule, nil
}

func (po *PortfolioOptimizer) calculateRebalancingTriggers(portfolio *models.Portfolio, thresholdPct decimal.Decimal) []RebalancingTrigger {
	triggers := make([]RebalancingTrigger, 0)

	if portfolio.TotalValue.IsZero() {
		return triggers
	}

	// Equal weight target for trigger calculation
	targetWeight := decimal.NewFromInt(1).Div(decimal.NewFromInt(int64(len(portfolio.Holdings))))

	for _, holding := range portfolio.Holdings {
		currentWeight := holding.CurrentValue.Div(portfolio.TotalValue)
		deviation := currentWeight.Sub(targetWeight).Abs()
		deviationPct := deviation.Div(targetWeight)

		triggered := deviationPct.GreaterThan(thresholdPct)

		triggers = append(triggers, RebalancingTrigger{
			Type:         "weight_deviation",
			Threshold:    thresholdPct,
			CurrentValue: deviationPct,
			Triggered:    triggered,
			Description:  fmt.Sprintf("%s weight deviation: %.2f%%", holding.Symbol, deviationPct.Mul(decimal.NewFromInt(100))),
		})
	}

	// Add portfolio-level triggers
	if portfolio.RiskMetrics.Volatility30d.GreaterThan(decimal.NewFromFloat(0.4)) {
		triggers = append(triggers, RebalancingTrigger{
			Type:         "high_volatility",
			Threshold:    decimal.NewFromFloat(0.4),
			CurrentValue: portfolio.RiskMetrics.Volatility30d,
			Triggered:    true,
			Description:  "High portfolio volatility detected",
		})
	}

	return triggers
}

func (po *PortfolioOptimizer) calculateNextRebalancingDate(frequency string) string {
	// Simplified date calculation - in production, use proper date/time libraries
	switch frequency {
	case "weekly":
		return "next_week"
	case "monthly":
		return "next_month"
	case "quarterly":
		return "next_quarter"
	default:
		return "next_month"
	}
}

// Extension for decimal sqrt operation
func (d decimal.Decimal) Sqrt() decimal.Decimal {
	f, _ := d.Float64()
	if f < 0 {
		return decimal.Zero
	}
	return decimal.NewFromFloat(math.Sqrt(f))
}