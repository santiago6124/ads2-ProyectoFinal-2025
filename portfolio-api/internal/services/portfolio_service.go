package services

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"portfolio-api/internal/analytics"
	"portfolio-api/internal/calculator"
	"portfolio-api/internal/clients"
	"portfolio-api/internal/models"
	"portfolio-api/internal/repositories"
	"portfolio-api/pkg/cache"
)

// Interfaces for testing
type CacheInterface interface {
	GetPortfolio(ctx context.Context, userID int64, portfolio interface{}) error
	SetPortfolio(ctx context.Context, userID int64, portfolio interface{}) error
	DeletePortfolio(ctx context.Context, userID int64) error
	InvalidatePortfolio(ctx context.Context, userID int64) error
	Get(ctx context.Context, key string, value interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

type UserClientInterface interface {
	GetUserBalance(ctx context.Context, userID int64) (decimal.Decimal, error)
}

type PnLCalculatorInterface interface {
	CalculatePnL(ctx context.Context, portfolio *models.Portfolio) error
}

type RiskCalculatorInterface interface {
	CalculateRisk(ctx context.Context, portfolio *models.Portfolio) error
	CalculateRiskMetrics(ctx context.Context, portfolio *models.Portfolio) error
}

type ROICalculatorInterface interface {
	CalculateROI(ctx context.Context, portfolio *models.Portfolio) error
	CalculatePortfolioROI(ctx context.Context, portfolio *models.Portfolio) (*models.ROIData, error)
}

type AnalyzerInterface interface {
	Analyze(ctx context.Context, portfolio *models.Portfolio) (*models.AnalysisResult, error)
	PerformComprehensiveAnalysis(ctx context.Context, portfolio *models.Portfolio) (*models.AnalysisResult, error)
}

type OptimizerInterface interface {
	Optimize(ctx context.Context, portfolio *models.Portfolio, constraints *analytics.OptimizationConstraints) (*analytics.OptimizationResult, error)
}

type PortfolioService struct {
	portfolioRepo   repositories.PortfolioRepository
	snapshotRepo    repositories.SnapshotRepository
	cache           CacheInterface
	userClient      UserClientInterface
	pnlCalculator   PnLCalculatorInterface
	riskCalculator  RiskCalculatorInterface
	roiCalculator   ROICalculatorInterface
	analyzer        AnalyzerInterface
	optimizer       OptimizerInterface
}

func NewPortfolioService(
	portfolioRepo repositories.PortfolioRepository,
	snapshotRepo repositories.SnapshotRepository,
	cache CacheInterface,
	userClient UserClientInterface,
	pnlCalculator PnLCalculatorInterface,
	riskCalculator RiskCalculatorInterface,
	roiCalculator ROICalculatorInterface,
	analyzer AnalyzerInterface,
	optimizer OptimizerInterface,
) *PortfolioService {
	return &PortfolioService{
		portfolioRepo:  portfolioRepo,
		snapshotRepo:   snapshotRepo,
		cache:          cache,
		userClient:     userClient,
		pnlCalculator:  pnlCalculator,
		riskCalculator: riskCalculator,
		roiCalculator:  roiCalculator,
		analyzer:       analyzer,
		optimizer:      optimizer,
	}
}

// GetPortfolio retrieves a portfolio by user ID with caching
func (ps *PortfolioService) GetPortfolio(ctx context.Context, userID int64) (*models.Portfolio, error) {
	// Try cache first
	var portfolio models.Portfolio
	err := ps.cache.GetPortfolio(ctx, userID, &portfolio)
	if err == nil {
		// Update cash balance from user API
		userBalance, err := ps.userClient.GetUserBalance(ctx, userID)
		if err == nil {
			portfolio.TotalCash = userBalance
		}
		return &portfolio, nil
	}

	// Cache miss, get from database
	portfolioPtr, err := ps.portfolioRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	// Get current user balance
	userBalance, err := ps.userClient.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user balance: %w", err)
	}

	// Update portfolio with current cash balance
	if portfolioPtr != nil {
		portfolioPtr.TotalCash = userBalance
		// Cache the updated result
		_ = ps.cache.SetPortfolio(ctx, userID, portfolioPtr)
		return portfolioPtr, nil
	}

	return nil, fmt.Errorf("portfolio not found for user %d", userID)
}

// CreatePortfolio creates a new portfolio for a user
func (ps *PortfolioService) CreatePortfolio(ctx context.Context, userID int64) (*models.Portfolio, error) {
	// Check if portfolio already exists
	existingPortfolio, err := ps.portfolioRepo.GetByUserID(ctx, userID)
	if err == nil && existingPortfolio != nil {
		return nil, fmt.Errorf("portfolio already exists for user %d", userID)
	}

	// Get user balance
	userBalance, err := ps.userClient.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user balance: %w", err)
	}

	// Create new portfolio with user's cash balance
	portfolio := models.NewPortfolio(userID)
	portfolio.TotalCash = userBalance

	err = ps.portfolioRepo.Create(ctx, portfolio)
	if err != nil {
		return nil, fmt.Errorf("failed to create portfolio: %w", err)
	}

	// Cache the new portfolio
	_ = ps.cache.SetPortfolio(ctx, userID, portfolio)

	return portfolio, nil
}

// UpdatePortfolio updates an existing portfolio
func (ps *PortfolioService) UpdatePortfolio(ctx context.Context, portfolio *models.Portfolio) error {
	if portfolio == nil {
		return fmt.Errorf("portfolio cannot be nil")
	}

	// Validate portfolio
	if err := portfolio.Validate(); err != nil {
		return fmt.Errorf("portfolio validation failed: %w", err)
	}

	// Update timestamp
	portfolio.UpdatedAt = time.Now()

	// Update in database
	err := ps.portfolioRepo.Update(ctx, portfolio)
	if err != nil {
		return fmt.Errorf("failed to update portfolio: %w", err)
	}

	// Update cache
	_ = ps.cache.SetPortfolio(ctx, portfolio.UserID, portfolio)

	// Invalidate related caches
	_ = ps.cache.InvalidatePortfolio(ctx, portfolio.UserID)

	return nil
}

// AddHolding adds a new holding to a portfolio
func (ps *PortfolioService) AddHolding(ctx context.Context, userID int64, holding *models.Holding) error {
	if holding == nil {
		return fmt.Errorf("holding cannot be nil")
	}

	portfolio, err := ps.GetPortfolio(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get portfolio: %w", err)
	}

	// Check if holding already exists
	existingHolding, found := portfolio.GetHoldingBySymbol(holding.Symbol)
	if found {
		// Update existing holding
		existingHolding.Quantity = existingHolding.Quantity.Add(holding.Quantity)
		existingHolding.AverageBuyPrice = ps.calculateNewAverageCost(
			existingHolding.AverageBuyPrice, existingHolding.Quantity.Sub(holding.Quantity),
			holding.AverageBuyPrice, holding.Quantity,
		)
		existingHolding.LastPurchaseDate = time.Now()
	} else {
		// Add new holding
		holding.FirstPurchaseDate = time.Now()
		holding.LastPurchaseDate = time.Now()
		portfolio.Holdings = append(portfolio.Holdings, *holding)
	}

	// Recalculate portfolio
	err = ps.RecalculatePortfolio(ctx, portfolio)
	if err != nil {
		return fmt.Errorf("failed to recalculate portfolio: %w", err)
	}

	return ps.UpdatePortfolio(ctx, portfolio)
}

// RemoveHolding removes a holding from a portfolio
func (ps *PortfolioService) RemoveHolding(ctx context.Context, userID int64, symbol string) error {
	portfolio, err := ps.GetPortfolio(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get portfolio: %w", err)
	}

	// Find and remove holding
	for i, holding := range portfolio.Holdings {
		if holding.Symbol == symbol {
			portfolio.Holdings = append(portfolio.Holdings[:i], portfolio.Holdings[i+1:]...)
			break
		}
	}

	// Recalculate portfolio
	err = ps.RecalculatePortfolio(ctx, portfolio)
	if err != nil {
		return fmt.Errorf("failed to recalculate portfolio: %w", err)
	}

	return ps.UpdatePortfolio(ctx, portfolio)
}

// UpdateHolding updates a specific holding in a portfolio
func (ps *PortfolioService) UpdateHolding(ctx context.Context, userID int64, holding *models.Holding) error {
	if holding == nil {
		return fmt.Errorf("holding cannot be nil")
	}

	portfolio, err := ps.GetPortfolio(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get portfolio: %w", err)
	}

	// Find and update holding
	existingHolding, found := portfolio.GetHoldingBySymbol(holding.Symbol)
	if !found {
		return fmt.Errorf("holding %s not found", holding.Symbol)
	}

	// Update holding fields
	existingHolding.Quantity = holding.Quantity
	existingHolding.AverageBuyPrice = holding.AverageBuyPrice
	existingHolding.CurrentPrice = holding.CurrentPrice
	existingHolding.LastPurchaseDate = time.Now()

	// Recalculate portfolio
	err = ps.RecalculatePortfolio(ctx, portfolio)
	if err != nil {
		return fmt.Errorf("failed to recalculate portfolio: %w", err)
	}

	return ps.UpdatePortfolio(ctx, portfolio)
}

// RecalculatePortfolio recalculates all portfolio metrics
func (ps *PortfolioService) RecalculatePortfolio(ctx context.Context, portfolio *models.Portfolio) error {
	if portfolio == nil {
		return fmt.Errorf("portfolio cannot be nil")
	}

	// Calculate P&L for all holdings
	for i := range portfolio.Holdings {
		holding := &portfolio.Holdings[i]

		// Calculate current value
		holding.CurrentValue = holding.Quantity.Mul(holding.CurrentPrice)

		// Calculate P&L
		holding.ProfitLoss = holding.CurrentValue.Sub(holding.Quantity.Mul(holding.AverageBuyPrice))

		// Calculate P&L percentage
		costBasis := holding.Quantity.Mul(holding.AverageBuyPrice)
		if !costBasis.IsZero() {
			holding.ProfitLossPercentage = holding.ProfitLoss.Div(costBasis)
		}
	}

	// Calculate portfolio totals
	ps.calculatePortfolioTotals(portfolio)

	// Calculate performance metrics
	err := ps.calculatePerformanceMetrics(ctx, portfolio)
	if err != nil {
		return fmt.Errorf("failed to calculate performance metrics: %w", err)
	}

	// Calculate risk metrics
	err = ps.calculateRiskMetrics(ctx, portfolio)
	if err != nil {
		return fmt.Errorf("failed to calculate risk metrics: %w", err)
	}

	// Calculate diversification metrics
	ps.calculateDiversificationMetrics(portfolio)

	// Update metadata
	portfolio.Metadata.LastCalculated = time.Now()
	portfolio.Metadata.NeedsRecalculation = false
	portfolio.Metadata.CalculationVersion = "1.0"

	return nil
}

// calculatePortfolioTotals calculates basic portfolio totals
func (ps *PortfolioService) calculatePortfolioTotals(portfolio *models.Portfolio) {
	totalValue := decimal.Zero
	totalInvested := decimal.Zero
	totalPnL := decimal.Zero

	for _, holding := range portfolio.Holdings {
		totalValue = totalValue.Add(holding.CurrentValue)
		totalInvested = totalInvested.Add(holding.Quantity.Mul(holding.AverageBuyPrice))
		totalPnL = totalPnL.Add(holding.ProfitLoss)
	}

	portfolio.TotalValue = totalValue.Add(portfolio.TotalCash)
	portfolio.TotalInvested = totalInvested.Add(portfolio.TotalCash)
	portfolio.ProfitLoss = totalPnL

	// Calculate percentage of portfolio for each holding
	for i := range portfolio.Holdings {
		holding := &portfolio.Holdings[i]
		if !portfolio.TotalValue.IsZero() {
			holding.PercentageOfPortfolio = holding.CurrentValue.Div(portfolio.TotalValue)
		}
	}

	// Calculate overall P&L percentage
	if !portfolio.TotalInvested.IsZero() {
		portfolio.ProfitLossPercentage = portfolio.ProfitLoss.Div(portfolio.TotalInvested)
	}
}

// calculatePerformanceMetrics calculates portfolio performance metrics
func (ps *PortfolioService) calculatePerformanceMetrics(ctx context.Context, portfolio *models.Portfolio) error {
	// Get recent snapshots for calculations
	snapshots, err := ps.snapshotRepo.GetByUserID(ctx, portfolio.UserID, 90, 0) // Last 90 snapshots
	if err != nil {
		// If no snapshots available, set default values
		portfolio.Performance = models.Performance{
			DailyChange:           decimal.Zero,
			DailyChangePercentage: decimal.Zero,
			WeeklyChange:          decimal.Zero,
			WeeklyChangePercentage: decimal.Zero,
			MonthlyChange:         decimal.Zero,
			MonthlyChangePercentage: decimal.Zero,
			YearlyChange:          decimal.Zero,
			YearlyChangePercentage: decimal.Zero,
		}
		return nil
	}

	if len(snapshots) > 0 {
		// Calculate daily change
		if len(snapshots) >= 1 {
			yesterday := snapshots[0]
			dailyChange := portfolio.TotalValue.Sub(yesterday.Value.Total)
			portfolio.Performance.DailyChange = dailyChange

			if !yesterday.Value.Total.IsZero() {
				portfolio.Performance.DailyChangePercentage = dailyChange.Div(yesterday.Value.Total)
			}
		}

		// Calculate weekly change
		if len(snapshots) >= 7 {
			weekAgo := snapshots[6]
			weeklyChange := portfolio.TotalValue.Sub(weekAgo.Value.Total)
			portfolio.Performance.WeeklyChange = weeklyChange

			if !weekAgo.Value.Total.IsZero() {
				portfolio.Performance.WeeklyChangePercentage = weeklyChange.Div(weekAgo.Value.Total)
			}
		}

		// Calculate monthly change
		if len(snapshots) >= 30 {
			monthAgo := snapshots[29]
			monthlyChange := portfolio.TotalValue.Sub(monthAgo.Value.Total)
			portfolio.Performance.MonthlyChange = monthlyChange

			if !monthAgo.Value.Total.IsZero() {
				portfolio.Performance.MonthlyChangePercentage = monthlyChange.Div(monthAgo.Value.Total)
			}
		}

		// Calculate yearly change
		if len(snapshots) >= 365 {
			yearAgo := snapshots[364]
			yearlyChange := portfolio.TotalValue.Sub(yearAgo.Value.Total)
			portfolio.Performance.YearlyChange = yearlyChange

			if !yearAgo.Value.Total.IsZero() {
				portfolio.Performance.YearlyChangePercentage = yearlyChange.Div(yearAgo.Value.Total)
			}
		}
	}

	return nil
}

// calculateRiskMetrics calculates portfolio risk metrics
func (ps *PortfolioService) calculateRiskMetrics(ctx context.Context, portfolio *models.Portfolio) error {
	// Get snapshots for risk calculations
	snapshots, err := ps.snapshotRepo.GetByUserID(ctx, portfolio.UserID, 90, 0)
	if err != nil || len(snapshots) < 30 {
		// Set default risk metrics if insufficient data
		portfolio.RiskMetrics = models.RiskMetrics{
			Volatility30d:      decimal.NewFromFloat(0.2),
			SharpeRatio:        decimal.Zero,
			SortinoRatio:       decimal.Zero,
			MaxDrawdown:        decimal.Zero,
			ValueAtRisk95:      decimal.Zero,
			ConditionalVaR95:   decimal.Zero,
			Beta:               decimal.NewFromInt(1),
			Alpha:              decimal.Zero,
		}
		return nil
	}

	// Calculate risk metrics using risk calculator
	riskMetrics, err := ps.riskCalculator.CalculateRiskMetrics(ctx, snapshots, nil)
	if err != nil {
		return fmt.Errorf("failed to calculate risk metrics: %w", err)
	}

	// Map to portfolio risk metrics
	portfolio.RiskMetrics = models.RiskMetrics{
		Volatility30d:      riskMetrics.Volatility30d,
		SharpeRatio:        riskMetrics.SharpeRatio,
		SortinoRatio:       riskMetrics.SortinoRatio,
		MaxDrawdown:        riskMetrics.MaxDrawdown,
		ValueAtRisk95:      riskMetrics.VaR95,
		ConditionalVaR95:   riskMetrics.CVaR95,
		Beta:               riskMetrics.Beta,
		Alpha:              riskMetrics.Alpha,
	}

	return nil
}

// calculateDiversificationMetrics calculates portfolio diversification metrics
func (ps *PortfolioService) calculateDiversificationMetrics(portfolio *models.Portfolio) {
	holdingsCount := len(portfolio.Holdings)

	// Calculate Herfindahl-Hirschman Index
	hhi := decimal.Zero
	largestPosition := decimal.Zero

	for _, holding := range portfolio.Holdings {
		if holding.PercentageOfPortfolio.GreaterThan(largestPosition) {
			largestPosition = holding.PercentageOfPortfolio
		}
		hhi = hhi.Add(holding.PercentageOfPortfolio.Mul(holding.PercentageOfPortfolio))
	}

	// Calculate effective number of holdings
	effectiveHoldings := decimal.Zero
	if !hhi.IsZero() {
		effectiveHoldings = decimal.NewFromInt(1).Div(hhi)
	}

	// Calculate sector diversification
	sectorWeights := make(map[string]decimal.Decimal)
	for _, holding := range portfolio.Holdings {
		sector := holding.Category
		if sector == "" {
			sector = "Unknown"
		}

		if weight, exists := sectorWeights[sector]; exists {
			sectorWeights[sector] = weight.Add(holding.PercentageOfPortfolio)
		} else {
			sectorWeights[sector] = holding.PercentageOfPortfolio
		}
	}

	portfolio.Diversification = models.Diversification{
		HoldingsCount:             holdingsCount,
		EffectiveHoldings:         effectiveHoldings,
		HerfindahlIndex:           hhi,
		ConcentrationIndex:        hhi,
		LargestPositionPercentage: largestPosition,
		Categories:                sectorWeights,
	}
}

// calculateNewAverageCost calculates new average cost when adding to existing position
func (ps *PortfolioService) calculateNewAverageCost(oldCost, oldQty, newCost, newQty decimal.Decimal) decimal.Decimal {
	totalQty := oldQty.Add(newQty)
	if totalQty.IsZero() {
		return decimal.Zero
	}

	totalCost := oldCost.Mul(oldQty).Add(newCost.Mul(newQty))
	return totalCost.Div(totalQty)
}

// GetPortfolioPerformance gets detailed performance analysis
func (ps *PortfolioService) GetPortfolioPerformance(ctx context.Context, userID int64, period string) (*calculator.ROIMetrics, error) {
	portfolio, err := ps.GetPortfolio(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	snapshots, err := ps.snapshotRepo.GetByUserID(ctx, userID, 365, 0) // Get up to 1 year of data
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}

	return ps.roiCalculator.CalculatePortfolioROI(ctx, portfolio, snapshots)
}

// GetPortfolioAnalysis gets comprehensive portfolio analysis
func (ps *PortfolioService) GetPortfolioAnalysis(ctx context.Context, userID int64) (*analytics.ComprehensiveAnalysis, error) {
	portfolio, err := ps.GetPortfolio(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	snapshots, err := ps.snapshotRepo.GetByUserID(ctx, userID, 365, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}

	// Use cached analysis if available
	cacheKey := fmt.Sprintf("analysis:%d", userID)
	var analysis analytics.ComprehensiveAnalysis
	err = ps.cache.Get(ctx, cacheKey, &analysis)
	if err == nil && time.Since(analysis.LastUpdated) < 1*time.Hour {
		return &analysis, nil
	}

	// Generate new analysis
	analysisPtr, err := ps.analyzer.PerformComprehensiveAnalysis(ctx, portfolio, snapshots, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to perform analysis: %w", err)
	}

	// Cache the result
	_ = ps.cache.Set(ctx, cacheKey, analysisPtr, 1*time.Hour)

	return analysisPtr, nil
}

// GetOptimizationSuggestions gets portfolio optimization suggestions
func (ps *PortfolioService) GetOptimizationSuggestions(ctx context.Context, userID int64, strategy analytics.OptimizationStrategy) (*analytics.OptimizationResult, error) {
	portfolio, err := ps.GetPortfolio(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	return ps.optimizer.OptimizePortfolio(ctx, portfolio, strategy, nil)
}

// CreateSnapshot creates a portfolio snapshot
func (ps *PortfolioService) CreateSnapshot(ctx context.Context, userID int64, interval string) (*models.Snapshot, error) {
	portfolio, err := ps.GetPortfolio(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	snapshot := models.NewSnapshot(portfolio, interval)

	err = ps.snapshotRepo.Create(ctx, snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Cache snapshot
	cacheKey := fmt.Sprintf("snapshot:%s", snapshot.ID.Hex())
	_ = ps.cache.Set(ctx, cacheKey, snapshot, 24*time.Hour)

	return snapshot, nil
}

// CreateManualSnapshot creates a manual snapshot with note and tags
func (ps *PortfolioService) CreateManualSnapshot(ctx context.Context, userID int64, note string, tags []string) (*models.Snapshot, error) {
	portfolio, err := ps.GetPortfolio(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	snapshot := models.NewManualSnapshot(portfolio, note, tags)

	err = ps.snapshotRepo.Create(ctx, snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to create manual snapshot: %w", err)
	}

	return snapshot, nil
}

// GetPortfolioHistory gets portfolio history with specified parameters
func (ps *PortfolioService) GetPortfolioHistory(ctx context.Context, userID int64, interval string, limit int) ([]models.HistoryPoint, *models.HistorySummary, error) {
	snapshots, err := ps.snapshotRepo.GetByInterval(ctx, userID, interval, limit, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		return nil, nil, fmt.Errorf("no snapshots found")
	}

	// Convert snapshots to history points
	historyPoints := make([]models.HistoryPoint, len(snapshots))
	for i, snapshot := range snapshots {
		historyPoints[i] = snapshot.ToHistoryPoint()
	}

	// Calculate summary
	summary := ps.calculateHistorySummary(snapshots)

	return historyPoints, summary, nil
}

// calculateHistorySummary calculates summary statistics for portfolio history
func (ps *PortfolioService) calculateHistorySummary(snapshots []models.Snapshot) *models.HistorySummary {
	if len(snapshots) == 0 {
		return &models.HistorySummary{}
	}

	startSnapshot := snapshots[0]
	endSnapshot := snapshots[len(snapshots)-1]

	summary := &models.HistorySummary{
		StartValue: startSnapshot.Value.Total,
		EndValue:   endSnapshot.Value.Total,
	}

	// Calculate total change
	if !startSnapshot.Value.Total.IsZero() {
		summary.TotalChange = endSnapshot.Value.Total.Sub(startSnapshot.Value.Total)
		summary.TotalChangePercentage = summary.TotalChange.Div(startSnapshot.Value.Total)
	}

	// Find best and worst days
	bestChange := decimal.Zero
	worstChange := decimal.Zero
	var bestSnapshot, worstSnapshot models.Snapshot

	for _, snapshot := range snapshots {
		dailyChange := snapshot.Value.DailyChangePercent
		if dailyChange.GreaterThan(bestChange) {
			bestChange = dailyChange
			bestSnapshot = snapshot
		}
		if dailyChange.LessThan(worstChange) {
			worstChange = dailyChange
			worstSnapshot = snapshot
		}
	}

	summary.BestDay = models.HistoryExtreme{
		Date:             bestSnapshot.Timestamp.Format("2006-01-02"),
		Value:            bestSnapshot.Value.Total,
		Change:           bestSnapshot.Value.DailyChange,
		ChangePercentage: bestSnapshot.Value.DailyChangePercent,
	}

	summary.WorstDay = models.HistoryExtreme{
		Date:             worstSnapshot.Timestamp.Format("2006-01-02"),
		Value:            worstSnapshot.Value.Total,
		Change:           worstSnapshot.Value.DailyChange,
		ChangePercentage: worstSnapshot.Value.DailyChangePercent,
	}

	return summary
}

// DeletePortfolio deletes a portfolio and all associated data
func (ps *PortfolioService) DeletePortfolio(ctx context.Context, userID int64) error {
	// Delete snapshots first
	err := ps.snapshotRepo.DeleteByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete snapshots: %w", err)
	}

	// Delete portfolio
	err = ps.portfolioRepo.DeleteByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete portfolio: %w", err)
	}

	// Clear cache
	_ = ps.cache.InvalidatePortfolio(ctx, userID)

	return nil
}

// MarkForRecalculation marks a portfolio as needing recalculation
func (ps *PortfolioService) MarkForRecalculation(ctx context.Context, userID int64) error {
	portfolio, err := ps.GetPortfolio(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get portfolio: %w", err)
	}

	portfolio.Metadata.NeedsRecalculation = true
	portfolio.UpdatedAt = time.Now()

	return ps.UpdatePortfolio(ctx, portfolio)
}

// GetPortfoliosNeedingRecalculation gets portfolios that need recalculation
func (ps *PortfolioService) GetPortfoliosNeedingRecalculation(ctx context.Context, limit int) ([]*models.Portfolio, error) {
	return ps.portfolioRepo.GetNeedingRecalculation(ctx, limit)
}