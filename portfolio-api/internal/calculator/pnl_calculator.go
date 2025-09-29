package calculator

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"

	"portfolio-api/internal/models"
	"portfolio-api/internal/clients"
)

// PnLCalculator handles profit and loss calculations
type PnLCalculator struct {
	marketClient *clients.MarketClient
	logger       *logrus.Logger
}

// PnLResult represents the result of P&L calculation
type PnLResult struct {
	TotalValue            decimal.Decimal `json:"total_value"`
	TotalInvested         decimal.Decimal `json:"total_invested"`
	TotalCash             decimal.Decimal `json:"total_cash"`
	RealizedPnL           decimal.Decimal `json:"realized_pnl"`
	UnrealizedPnL         decimal.Decimal `json:"unrealized_pnl"`
	TotalPnL              decimal.Decimal `json:"total_pnl"`
	PnLPercentage         decimal.Decimal `json:"pnl_percentage"`
	DailyPnL              decimal.Decimal `json:"daily_pnl"`
	DailyPnLPercentage    decimal.Decimal `json:"daily_pnl_percentage"`
	WeeklyPnL             decimal.Decimal `json:"weekly_pnl"`
	WeeklyPnLPercentage   decimal.Decimal `json:"weekly_pnl_percentage"`
	MonthlyPnL            decimal.Decimal `json:"monthly_pnl"`
	MonthlyPnLPercentage  decimal.Decimal `json:"monthly_pnl_percentage"`
	YearlyPnL             decimal.Decimal `json:"yearly_pnl"`
	YearlyPnLPercentage   decimal.Decimal `json:"yearly_pnl_percentage"`
}

// HoldingPnL represents P&L for a specific holding
type HoldingPnL struct {
	Symbol               string          `json:"symbol"`
	Quantity             decimal.Decimal `json:"quantity"`
	AverageBuyPrice      decimal.Decimal `json:"average_buy_price"`
	CurrentPrice         decimal.Decimal `json:"current_price"`
	TotalInvested        decimal.Decimal `json:"total_invested"`
	CurrentValue         decimal.Decimal `json:"current_value"`
	ProfitLoss           decimal.Decimal `json:"profit_loss"`
	ProfitLossPercentage decimal.Decimal `json:"profit_loss_percentage"`
	PercentageOfPortfolio decimal.Decimal `json:"percentage_of_portfolio"`
	DailyChange          decimal.Decimal `json:"daily_change"`
	DailyChangePercentage decimal.Decimal `json:"daily_change_percentage"`
}

// Transaction represents a buy/sell transaction
type Transaction struct {
	ID        string          `json:"id"`
	Symbol    string          `json:"symbol"`
	Type      string          `json:"type"` // "buy" or "sell"
	Quantity  decimal.Decimal `json:"quantity"`
	Price     decimal.Decimal `json:"price"`
	Value     decimal.Decimal `json:"value"`
	Fee       decimal.Decimal `json:"fee"`
	Timestamp time.Time       `json:"timestamp"`
}

// CostBasisMethod represents different cost basis calculation methods
type CostBasisMethod string

const (
	CostBasisFIFO    CostBasisMethod = "FIFO"    // First In, First Out
	CostBasisLIFO    CostBasisMethod = "LIFO"    // Last In, First Out
	CostBasisAverage CostBasisMethod = "AVERAGE" // Weighted Average
)

// NewPnLCalculator creates a new P&L calculator
func NewPnLCalculator(marketClient *clients.MarketClient) *PnLCalculator {
	return &PnLCalculator{
		marketClient: marketClient,
		logger:       logrus.WithField("component", "pnl_calculator"),
	}
}

// CalculatePortfolioPnL calculates P&L for entire portfolio
func (calc *PnLCalculator) CalculatePortfolioPnL(ctx context.Context, portfolio *models.Portfolio, historicalSnapshots []*models.Snapshot) (*PnLResult, error) {
	calc.logger.WithField("user_id", portfolio.UserID).Info("Calculating portfolio P&L")

	result := &PnLResult{
		TotalInvested: decimal.Zero,
		TotalValue:    decimal.Zero,
		TotalCash:     portfolio.TotalCash,
		RealizedPnL:   decimal.Zero,
		UnrealizedPnL: decimal.Zero,
	}

	// Calculate P&L for each holding
	var totalCryptoValue decimal.Decimal
	for i, holding := range portfolio.Holdings {
		if holding.Quantity.IsZero() {
			continue
		}

		// Get current price for the holding
		currentPrice, err := calc.marketClient.GetCurrentPrice(ctx, holding.Symbol)
		if err != nil {
			calc.logger.WithError(err).WithField("symbol", holding.Symbol).Warn("Failed to get current price")
			currentPrice = holding.CurrentPrice // Use cached price
		}

		// Calculate holding P&L
		holdingPnL := calc.calculateHoldingPnL(&holding, currentPrice)

		// Update portfolio holdings with latest data
		portfolio.Holdings[i].CurrentPrice = currentPrice
		portfolio.Holdings[i].CurrentValue = holdingPnL.CurrentValue
		portfolio.Holdings[i].ProfitLoss = holdingPnL.ProfitLoss
		portfolio.Holdings[i].ProfitLossPercentage = holdingPnL.ProfitLossPercentage

		// Accumulate totals
		result.TotalInvested = result.TotalInvested.Add(holdingPnL.TotalInvested)
		result.UnrealizedPnL = result.UnrealizedPnL.Add(holdingPnL.ProfitLoss)
		totalCryptoValue = totalCryptoValue.Add(holdingPnL.CurrentValue)
	}

	// Calculate total portfolio value
	result.TotalValue = totalCryptoValue.Add(result.TotalCash)

	// Calculate total P&L and percentage
	result.TotalPnL = result.RealizedPnL.Add(result.UnrealizedPnL)
	if result.TotalInvested.GreaterThan(decimal.Zero) {
		result.PnLPercentage = result.TotalPnL.Div(result.TotalInvested).Mul(decimal.NewFromInt(100))
	}

	// Calculate percentage of portfolio for each holding
	if result.TotalValue.GreaterThan(decimal.Zero) {
		for i := range portfolio.Holdings {
			if portfolio.Holdings[i].Quantity.IsZero() {
				continue
			}
			percentage := portfolio.Holdings[i].CurrentValue.Div(result.TotalValue).Mul(decimal.NewFromInt(100))
			portfolio.Holdings[i].PercentageOfPortfolio = percentage
		}
	}

	// Calculate periodic changes
	calc.calculatePeriodicChanges(result, historicalSnapshots)

	calc.logger.WithFields(logrus.Fields{
		"user_id":      portfolio.UserID,
		"total_value":  result.TotalValue,
		"total_pnl":    result.TotalPnL,
		"pnl_percent":  result.PnLPercentage,
	}).Info("Portfolio P&L calculation completed")

	return result, nil
}

// calculateHoldingPnL calculates P&L for a single holding
func (calc *PnLCalculator) calculateHoldingPnL(holding *models.Holding, currentPrice decimal.Decimal) *HoldingPnL {
	currentValue := holding.Quantity.Mul(currentPrice)
	totalInvested := holding.Quantity.Mul(holding.AverageBuyPrice)
	profitLoss := currentValue.Sub(totalInvested)

	var profitLossPercentage decimal.Decimal
	if totalInvested.GreaterThan(decimal.Zero) {
		profitLossPercentage = profitLoss.Div(totalInvested).Mul(decimal.NewFromInt(100))
	}

	return &HoldingPnL{
		Symbol:               holding.Symbol,
		Quantity:             holding.Quantity,
		AverageBuyPrice:      holding.AverageBuyPrice,
		CurrentPrice:         currentPrice,
		TotalInvested:        totalInvested,
		CurrentValue:         currentValue,
		ProfitLoss:           profitLoss,
		ProfitLossPercentage: profitLossPercentage,
	}
}

// calculatePeriodicChanges calculates daily, weekly, monthly, yearly changes
func (calc *PnLCalculator) calculatePeriodicChanges(result *PnLResult, snapshots []*models.Snapshot) {
	if len(snapshots) == 0 {
		return
	}

	// Sort snapshots by timestamp (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.After(snapshots[j].Timestamp)
	})

	now := time.Now()

	// Find snapshots for different periods
	var dailySnapshot, weeklySnapshot, monthlySnapshot, yearlySnapshot *models.Snapshot

	for _, snapshot := range snapshots {
		age := now.Sub(snapshot.Timestamp)

		if dailySnapshot == nil && age >= 24*time.Hour && age < 48*time.Hour {
			dailySnapshot = snapshot
		}
		if weeklySnapshot == nil && age >= 7*24*time.Hour && age < 14*24*time.Hour {
			weeklySnapshot = snapshot
		}
		if monthlySnapshot == nil && age >= 30*24*time.Hour && age < 60*24*time.Hour {
			monthlySnapshot = snapshot
		}
		if yearlySnapshot == nil && age >= 365*24*time.Hour && age < 730*24*time.Hour {
			yearlySnapshot = snapshot
		}
	}

	// Calculate daily change
	if dailySnapshot != nil {
		result.DailyPnL = result.TotalValue.Sub(dailySnapshot.Value.Total)
		if dailySnapshot.Value.Total.GreaterThan(decimal.Zero) {
			result.DailyPnLPercentage = result.DailyPnL.Div(dailySnapshot.Value.Total).Mul(decimal.NewFromInt(100))
		}
	}

	// Calculate weekly change
	if weeklySnapshot != nil {
		result.WeeklyPnL = result.TotalValue.Sub(weeklySnapshot.Value.Total)
		if weeklySnapshot.Value.Total.GreaterThan(decimal.Zero) {
			result.WeeklyPnLPercentage = result.WeeklyPnL.Div(weeklySnapshot.Value.Total).Mul(decimal.NewFromInt(100))
		}
	}

	// Calculate monthly change
	if monthlySnapshot != nil {
		result.MonthlyPnL = result.TotalValue.Sub(monthlySnapshot.Value.Total)
		if monthlySnapshot.Value.Total.GreaterThan(decimal.Zero) {
			result.MonthlyPnLPercentage = result.MonthlyPnL.Div(monthlySnapshot.Value.Total).Mul(decimal.NewFromInt(100))
		}
	}

	// Calculate yearly change
	if yearlySnapshot != nil {
		result.YearlyPnL = result.TotalValue.Sub(yearlySnapshot.Value.Total)
		if yearlySnapshot.Value.Total.GreaterThan(decimal.Zero) {
			result.YearlyPnLPercentage = result.YearlyPnL.Div(yearlySnapshot.Value.Total).Mul(decimal.NewFromInt(100))
		}
	}
}

// CalculateCostBasis calculates cost basis using specified method
func (calc *PnLCalculator) CalculateCostBasis(transactions []Transaction, method CostBasisMethod) (decimal.Decimal, decimal.Decimal, error) {
	if len(transactions) == 0 {
		return decimal.Zero, decimal.Zero, nil
	}

	// Sort transactions by timestamp
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Timestamp.Before(transactions[j].Timestamp)
	})

	switch method {
	case CostBasisFIFO:
		return calc.calculateFIFOCostBasis(transactions)
	case CostBasisLIFO:
		return calc.calculateLIFOCostBasis(transactions)
	case CostBasisAverage:
		return calc.calculateAverageCostBasis(transactions)
	default:
		return calc.calculateFIFOCostBasis(transactions)
	}
}

// calculateFIFOCostBasis calculates cost basis using First In, First Out method
func (calc *PnLCalculator) calculateFIFOCostBasis(transactions []Transaction) (decimal.Decimal, decimal.Decimal, error) {
	var queue []Transaction
	totalCost := decimal.Zero
	totalQuantity := decimal.Zero

	for _, tx := range transactions {
		if tx.Type == "buy" {
			queue = append(queue, tx)
			totalCost = totalCost.Add(tx.Value)
			totalQuantity = totalQuantity.Add(tx.Quantity)
		} else if tx.Type == "sell" {
			remaining := tx.Quantity

			for len(queue) > 0 && remaining.GreaterThan(decimal.Zero) {
				if queue[0].Quantity.LessThanOrEqual(remaining) {
					// Consume entire first entry
					soldCost := queue[0].Quantity.Mul(queue[0].Price)
					totalCost = totalCost.Sub(soldCost)
					totalQuantity = totalQuantity.Sub(queue[0].Quantity)
					remaining = remaining.Sub(queue[0].Quantity)
					queue = queue[1:]
				} else {
					// Partially consume first entry
					soldCost := remaining.Mul(queue[0].Price)
					totalCost = totalCost.Sub(soldCost)
					totalQuantity = totalQuantity.Sub(remaining)
					queue[0].Quantity = queue[0].Quantity.Sub(remaining)
					remaining = decimal.Zero
				}
			}
		}
	}

	var averagePrice decimal.Decimal
	if totalQuantity.GreaterThan(decimal.Zero) {
		averagePrice = totalCost.Div(totalQuantity)
	}

	return totalQuantity, averagePrice, nil
}

// calculateLIFOCostBasis calculates cost basis using Last In, First Out method
func (calc *PnLCalculator) calculateLIFOCostBasis(transactions []Transaction) (decimal.Decimal, decimal.Decimal, error) {
	var stack []Transaction
	totalCost := decimal.Zero
	totalQuantity := decimal.Zero

	for _, tx := range transactions {
		if tx.Type == "buy" {
			stack = append(stack, tx)
			totalCost = totalCost.Add(tx.Value)
			totalQuantity = totalQuantity.Add(tx.Quantity)
		} else if tx.Type == "sell" {
			remaining := tx.Quantity

			for len(stack) > 0 && remaining.GreaterThan(decimal.Zero) {
				lastIndex := len(stack) - 1
				if stack[lastIndex].Quantity.LessThanOrEqual(remaining) {
					// Consume entire last entry
					soldCost := stack[lastIndex].Quantity.Mul(stack[lastIndex].Price)
					totalCost = totalCost.Sub(soldCost)
					totalQuantity = totalQuantity.Sub(stack[lastIndex].Quantity)
					remaining = remaining.Sub(stack[lastIndex].Quantity)
					stack = stack[:lastIndex]
				} else {
					// Partially consume last entry
					soldCost := remaining.Mul(stack[lastIndex].Price)
					totalCost = totalCost.Sub(soldCost)
					totalQuantity = totalQuantity.Sub(remaining)
					stack[lastIndex].Quantity = stack[lastIndex].Quantity.Sub(remaining)
					remaining = decimal.Zero
				}
			}
		}
	}

	var averagePrice decimal.Decimal
	if totalQuantity.GreaterThan(decimal.Zero) {
		averagePrice = totalCost.Div(totalQuantity)
	}

	return totalQuantity, averagePrice, nil
}

// calculateAverageCostBasis calculates cost basis using weighted average method
func (calc *PnLCalculator) calculateAverageCostBasis(transactions []Transaction) (decimal.Decimal, decimal.Decimal, error) {
	totalCost := decimal.Zero
	totalQuantity := decimal.Zero

	for _, tx := range transactions {
		if tx.Type == "buy" {
			totalCost = totalCost.Add(tx.Value)
			totalQuantity = totalQuantity.Add(tx.Quantity)
		} else if tx.Type == "sell" {
			// For sells, reduce quantity but maintain average price
			if totalQuantity.GreaterThan(decimal.Zero) {
				averagePrice := totalCost.Div(totalQuantity)
				soldCost := tx.Quantity.Mul(averagePrice)
				totalCost = totalCost.Sub(soldCost)
				totalQuantity = totalQuantity.Sub(tx.Quantity)
			}
		}
	}

	var averagePrice decimal.Decimal
	if totalQuantity.GreaterThan(decimal.Zero) {
		averagePrice = totalCost.Div(totalQuantity)
	}

	return totalQuantity, averagePrice, nil
}

// UpdateHoldingFromTransactions updates a holding based on transaction history
func (calc *PnLCalculator) UpdateHoldingFromTransactions(holding *models.Holding, transactions []Transaction, method CostBasisMethod) error {
	quantity, averagePrice, err := calc.CalculateCostBasis(transactions, method)
	if err != nil {
		return fmt.Errorf("failed to calculate cost basis: %w", err)
	}

	holding.Quantity = quantity
	holding.AverageBuyPrice = averagePrice
	holding.TotalInvested = quantity.Mul(averagePrice)
	holding.TransactionsCount = len(transactions)

	// Update first and last purchase dates
	if len(transactions) > 0 {
		// Sort transactions by timestamp
		sort.Slice(transactions, func(i, j int) bool {
			return transactions[i].Timestamp.Before(transactions[j].Timestamp)
		})

		var firstBuy, lastBuy *Transaction
		for i := range transactions {
			if transactions[i].Type == "buy" {
				if firstBuy == nil {
					firstBuy = &transactions[i]
				}
				lastBuy = &transactions[i]
			}
		}

		if firstBuy != nil {
			holding.FirstPurchaseDate = firstBuy.Timestamp
		}
		if lastBuy != nil {
			holding.LastPurchaseDate = lastBuy.Timestamp
		}
	}

	return nil
}

// ValidateTransactions validates a slice of transactions
func (calc *PnLCalculator) ValidateTransactions(transactions []Transaction) error {
	for i, tx := range transactions {
		if tx.Symbol == "" {
			return fmt.Errorf("transaction %d: symbol is required", i)
		}

		if tx.Type != "buy" && tx.Type != "sell" {
			return fmt.Errorf("transaction %d: invalid type %s, must be 'buy' or 'sell'", i, tx.Type)
		}

		if tx.Quantity.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("transaction %d: quantity must be positive", i)
		}

		if tx.Price.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("transaction %d: price must be positive", i)
		}

		if tx.Timestamp.IsZero() {
			return fmt.Errorf("transaction %d: timestamp is required", i)
		}
	}

	return nil
}