package aggregator

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	"market-data-api/internal/models"
	"market-data-api/internal/providers"
)

// TechnicalAnalyzer provides technical analysis capabilities for the aggregation engine
type TechnicalAnalyzer struct {
	providerManager *providers.ProviderManager
}

// NewTechnicalAnalyzer creates a new technical analyzer
func NewTechnicalAnalyzer(providerManager *providers.ProviderManager) *TechnicalAnalyzer {
	return &TechnicalAnalyzer{
		providerManager: providerManager,
	}
}

// AnalyzeTechnicalIndicators calculates technical indicators for price data
func (ta *TechnicalAnalyzer) AnalyzeTechnicalIndicators(ctx context.Context, symbol string, period string) (*models.TechnicalIndicators, error) {
	// Get historical data for analysis
	candles, err := ta.getHistoricalCandles(ctx, symbol, "1h", period, 200) // Get enough data for calculations
	if err != nil {
		return nil, fmt.Errorf("failed to get historical data: %w", err)
	}

	if len(candles) < 20 {
		return nil, fmt.Errorf("insufficient data for technical analysis: %d candles", len(candles))
	}

	indicators := &models.TechnicalIndicators{
		Symbol:    symbol,
		Timestamp: time.Now(),
	}

	// Calculate various technical indicators
	ta.calculateMovingAverages(candles, indicators)
	ta.calculateRSI(candles, indicators, 14)
	ta.calculateMACD(candles, indicators)
	ta.calculateBollingerBands(candles, indicators, 20, 2.0)
	ta.calculateStochastic(candles, indicators, 14)
	ta.calculateWilliamsR(candles, indicators, 14)
	ta.calculateCCI(candles, indicators, 20)
	ta.calculateADX(candles, indicators, 14)
	ta.calculateOBV(candles, indicators)

	return indicators, nil
}

// getHistoricalCandles retrieves historical candle data from providers
func (ta *TechnicalAnalyzer) getHistoricalCandles(ctx context.Context, symbol, interval, period string, limit int) ([]*models.Candle, error) {
	// Try to get data from the best available provider
	healthyProviders := ta.providerManager.GetHealthyProviders()
	if len(healthyProviders) == 0 {
		return nil, fmt.Errorf("no healthy providers available")
	}

	// Calculate time range based on period
	to := time.Now()
	var from time.Time

	switch period {
	case "24h":
		from = to.Add(-24 * time.Hour)
	case "7d":
		from = to.Add(-7 * 24 * time.Hour)
	case "30d":
		from = to.Add(-30 * 24 * time.Hour)
	case "90d":
		from = to.Add(-90 * 24 * time.Hour)
	case "1y":
		from = to.Add(-365 * 24 * time.Hour)
	default:
		from = to.Add(-30 * 24 * time.Hour) // Default to 30 days
	}

	// Try providers in order of preference
	preferredOrder := []string{"binance", "coinbase", "coingecko"}

	for _, providerName := range preferredOrder {
		if provider, exists := healthyProviders[providerName]; exists {
			candles, err := provider.GetHistoricalData(ctx, symbol, interval, from, to, limit)
			if err == nil && len(candles) > 0 {
				// Sort candles by timestamp
				sort.Slice(candles, func(i, j int) bool {
					return candles[i].Timestamp.Before(candles[j].Timestamp)
				})
				return candles, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to retrieve historical data from any provider")
}

// calculateMovingAverages calculates various moving averages
func (ta *TechnicalAnalyzer) calculateMovingAverages(candles []*models.Candle, indicators *models.TechnicalIndicators) {
	if len(candles) < 50 {
		return
	}

	// Simple Moving Averages
	indicators.SMA20 = ta.calculateSMA(candles, 20)
	indicators.SMA50 = ta.calculateSMA(candles, 50)
	indicators.SMA200 = ta.calculateSMA(candles, 200)

	// Exponential Moving Averages
	indicators.EMA12 = ta.calculateEMA(candles, 12)
	indicators.EMA26 = ta.calculateEMA(candles, 26)
	indicators.EMA50 = ta.calculateEMA(candles, 50)
}

// calculateSMA calculates Simple Moving Average
func (ta *TechnicalAnalyzer) calculateSMA(candles []*models.Candle, period int) decimal.Decimal {
	if len(candles) < period {
		return decimal.Zero
	}

	sum := decimal.Zero
	for i := len(candles) - period; i < len(candles); i++ {
		sum = sum.Add(candles[i].Close)
	}

	return sum.Div(decimal.NewFromInt(int64(period)))
}

// calculateEMA calculates Exponential Moving Average
func (ta *TechnicalAnalyzer) calculateEMA(candles []*models.Candle, period int) decimal.Decimal {
	if len(candles) < period {
		return decimal.Zero
	}

	multiplier := decimal.NewFromFloat(2.0).Div(decimal.NewFromInt(int64(period + 1)))

	// Start with SMA for the first EMA value
	sma := ta.calculateSMA(candles[:period], period)
	ema := sma

	// Calculate EMA for remaining periods
	for i := period; i < len(candles); i++ {
		ema = candles[i].Close.Mul(multiplier).Add(ema.Mul(decimal.NewFromInt(1).Sub(multiplier)))
	}

	return ema
}

// calculateRSI calculates Relative Strength Index
func (ta *TechnicalAnalyzer) calculateRSI(candles []*models.Candle, indicators *models.TechnicalIndicators, period int) {
	if len(candles) < period+1 {
		return
	}

	gains := decimal.Zero
	losses := decimal.Zero

	// Calculate initial average gain and loss
	for i := 1; i <= period; i++ {
		change := candles[i].Close.Sub(candles[i-1].Close)
		if change.GreaterThan(decimal.Zero) {
			gains = gains.Add(change)
		} else {
			losses = losses.Add(change.Abs())
		}
	}

	avgGain := gains.Div(decimal.NewFromInt(int64(period)))
	avgLoss := losses.Div(decimal.NewFromInt(int64(period)))

	// Calculate RSI for remaining periods
	for i := period + 1; i < len(candles); i++ {
		change := candles[i].Close.Sub(candles[i-1].Close)

		var currentGain, currentLoss decimal.Decimal
		if change.GreaterThan(decimal.Zero) {
			currentGain = change
			currentLoss = decimal.Zero
		} else {
			currentGain = decimal.Zero
			currentLoss = change.Abs()
		}

		// Smooth the averages
		avgGain = avgGain.Mul(decimal.NewFromInt(int64(period - 1))).Add(currentGain).Div(decimal.NewFromInt(int64(period)))
		avgLoss = avgLoss.Mul(decimal.NewFromInt(int64(period - 1))).Add(currentLoss).Div(decimal.NewFromInt(int64(period)))
	}

	if avgLoss.IsZero() {
		indicators.RSI = decimal.NewFromInt(100)
	} else {
		rs := avgGain.Div(avgLoss)
		indicators.RSI = decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))
	}
}

// calculateMACD calculates Moving Average Convergence Divergence
func (ta *TechnicalAnalyzer) calculateMACD(candles []*models.Candle, indicators *models.TechnicalIndicators) {
	if len(candles) < 26 {
		return
	}

	ema12 := ta.calculateEMA(candles, 12)
	ema26 := ta.calculateEMA(candles, 26)

	indicators.MACDLine = ema12.Sub(ema26)

	// Calculate signal line (9-period EMA of MACD line)
	// For simplicity, we'll use the current MACD value
	// In a full implementation, you'd need historical MACD values
	indicators.MACDSignal = indicators.MACDLine.Mul(decimal.NewFromFloat(0.2)) // Simplified

	indicators.MACDHistogram = indicators.MACDLine.Sub(indicators.MACDSignal)
}

// calculateBollingerBands calculates Bollinger Bands
func (ta *TechnicalAnalyzer) calculateBollingerBands(candles []*models.Candle, indicators *models.TechnicalIndicators, period int, stdDev float64) {
	if len(candles) < period {
		return
	}

	// Calculate SMA
	sma := ta.calculateSMA(candles, period)

	// Calculate standard deviation
	sumSquaredDiff := decimal.Zero
	for i := len(candles) - period; i < len(candles); i++ {
		diff := candles[i].Close.Sub(sma)
		sumSquaredDiff = sumSquaredDiff.Add(diff.Mul(diff))
	}

	variance := sumSquaredDiff.Div(decimal.NewFromInt(int64(period)))
	standardDeviation := decimal.NewFromFloat(math.Sqrt(variance.InexactFloat64()))

	multiplier := decimal.NewFromFloat(stdDev)

	indicators.BBUpper = sma.Add(standardDeviation.Mul(multiplier))
	indicators.BBMiddle = sma
	indicators.BBLower = sma.Sub(standardDeviation.Mul(multiplier))
}

// calculateStochastic calculates Stochastic Oscillator
func (ta *TechnicalAnalyzer) calculateStochastic(candles []*models.Candle, indicators *models.TechnicalIndicators, period int) {
	if len(candles) < period {
		return
	}

	// Find highest high and lowest low in the period
	var highestHigh, lowestLow decimal.Decimal
	recentCandles := candles[len(candles)-period:]

	if len(recentCandles) > 0 {
		highestHigh = recentCandles[0].High
		lowestLow = recentCandles[0].Low
	}

	for _, candle := range recentCandles {
		if candle.High.GreaterThan(highestHigh) {
			highestHigh = candle.High
		}
		if candle.Low.LessThan(lowestLow) {
			lowestLow = candle.Low
		}
	}

	currentClose := candles[len(candles)-1].Close
	if !highestHigh.Equal(lowestLow) {
		indicators.StochK = currentClose.Sub(lowestLow).Div(highestHigh.Sub(lowestLow)).Mul(decimal.NewFromInt(100))
	}

	// %D is typically a 3-period SMA of %K
	// For simplicity, we'll set it equal to %K
	indicators.StochD = indicators.StochK
}

// calculateWilliamsR calculates Williams %R
func (ta *TechnicalAnalyzer) calculateWilliamsR(candles []*models.Candle, indicators *models.TechnicalIndicators, period int) {
	if len(candles) < period {
		return
	}

	// Find highest high and lowest low in the period
	var highestHigh, lowestLow decimal.Decimal
	recentCandles := candles[len(candles)-period:]

	if len(recentCandles) > 0 {
		highestHigh = recentCandles[0].High
		lowestLow = recentCandles[0].Low
	}

	for _, candle := range recentCandles {
		if candle.High.GreaterThan(highestHigh) {
			highestHigh = candle.High
		}
		if candle.Low.LessThan(lowestLow) {
			lowestLow = candle.Low
		}
	}

	currentClose := candles[len(candles)-1].Close
	if !highestHigh.Equal(lowestLow) {
		indicators.WilliamsR = highestHigh.Sub(currentClose).Div(highestHigh.Sub(lowestLow)).Mul(decimal.NewFromInt(-100))
	}
}

// calculateCCI calculates Commodity Channel Index
func (ta *TechnicalAnalyzer) calculateCCI(candles []*models.Candle, indicators *models.TechnicalIndicators, period int) {
	if len(candles) < period {
		return
	}

	// Calculate typical prices
	typicalPrices := make([]decimal.Decimal, len(candles))
	for i, candle := range candles {
		typicalPrices[i] = candle.High.Add(candle.Low).Add(candle.Close).Div(decimal.NewFromInt(3))
	}

	// Calculate SMA of typical prices
	sum := decimal.Zero
	recentPrices := typicalPrices[len(typicalPrices)-period:]
	for _, price := range recentPrices {
		sum = sum.Add(price)
	}
	smaTP := sum.Div(decimal.NewFromInt(int64(period)))

	// Calculate mean deviation
	sumDeviations := decimal.Zero
	for _, price := range recentPrices {
		sumDeviations = sumDeviations.Add(price.Sub(smaTP).Abs())
	}
	meanDeviation := sumDeviations.Div(decimal.NewFromInt(int64(period)))

	currentTP := typicalPrices[len(typicalPrices)-1]
	if !meanDeviation.IsZero() {
		indicators.CCI = currentTP.Sub(smaTP).Div(meanDeviation.Mul(decimal.NewFromFloat(0.015)))
	}
}

// calculateADX calculates Average Directional Index
func (ta *TechnicalAnalyzer) calculateADX(candles []*models.Candle, indicators *models.TechnicalIndicators, period int) {
	if len(candles) < period+1 {
		return
	}

	// This is a simplified ADX calculation
	// Full implementation would require calculating DI+, DI-, and DX values

	trueRanges := make([]decimal.Decimal, 0, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		high := candles[i].High
		low := candles[i].Low
		prevClose := candles[i-1].Close

		tr1 := high.Sub(low)
		tr2 := high.Sub(prevClose).Abs()
		tr3 := low.Sub(prevClose).Abs()

		trueRange := tr1
		if tr2.GreaterThan(trueRange) {
			trueRange = tr2
		}
		if tr3.GreaterThan(trueRange) {
			trueRange = tr3
		}

		trueRanges = append(trueRanges, trueRange)
	}

	// Calculate ATR (Average True Range) as a proxy for ADX
	if len(trueRanges) >= period {
		sum := decimal.Zero
		recentTR := trueRanges[len(trueRanges)-period:]
		for _, tr := range recentTR {
			sum = sum.Add(tr)
		}
		indicators.ADX = sum.Div(decimal.NewFromInt(int64(period)))
	}
}

// calculateOBV calculates On-Balance Volume
func (ta *TechnicalAnalyzer) calculateOBV(candles []*models.Candle, indicators *models.TechnicalIndicators) {
	if len(candles) < 2 {
		return
	}

	obv := decimal.Zero
	for i := 1; i < len(candles); i++ {
		if candles[i].Close.GreaterThan(candles[i-1].Close) {
			obv = obv.Add(candles[i].Volume)
		} else if candles[i].Close.LessThan(candles[i-1].Close) {
			obv = obv.Sub(candles[i].Volume)
		}
		// If closes are equal, OBV remains unchanged
	}

	indicators.OBV = obv
}

// GetTechnicalSignals generates trading signals based on technical indicators
func (ta *TechnicalAnalyzer) GetTechnicalSignals(indicators *models.TechnicalIndicators) *TechnicalSignals {
	signals := &TechnicalSignals{
		Symbol:    indicators.Symbol,
		Timestamp: time.Now(),
		Signals:   make(map[string]string),
		Strength:  make(map[string]float64),
	}

	// RSI signals
	if !indicators.RSI.IsZero() {
		rsi := indicators.RSI.InexactFloat64()
		if rsi > 70 {
			signals.Signals["RSI"] = "SELL"
			signals.Strength["RSI"] = (rsi - 70) / 30 // Strength from 0 to 1
		} else if rsi < 30 {
			signals.Signals["RSI"] = "BUY"
			signals.Strength["RSI"] = (30 - rsi) / 30
		} else {
			signals.Signals["RSI"] = "HOLD"
			signals.Strength["RSI"] = 0.5
		}
	}

	// Moving Average signals
	if !indicators.SMA20.IsZero() && !indicators.SMA50.IsZero() {
		if indicators.SMA20.GreaterThan(indicators.SMA50) {
			signals.Signals["MA"] = "BUY"
			signals.Strength["MA"] = 0.7
		} else {
			signals.Signals["MA"] = "SELL"
			signals.Strength["MA"] = 0.7
		}
	}

	// MACD signals
	if !indicators.MACDLine.IsZero() && !indicators.MACDSignal.IsZero() {
		if indicators.MACDLine.GreaterThan(indicators.MACDSignal) {
			signals.Signals["MACD"] = "BUY"
		} else {
			signals.Signals["MACD"] = "SELL"
		}
		signals.Strength["MACD"] = 0.6
	}

	// Stochastic signals
	if !indicators.StochK.IsZero() {
		stochK := indicators.StochK.InexactFloat64()
		if stochK > 80 {
			signals.Signals["STOCH"] = "SELL"
			signals.Strength["STOCH"] = (stochK - 80) / 20
		} else if stochK < 20 {
			signals.Signals["STOCH"] = "BUY"
			signals.Strength["STOCH"] = (20 - stochK) / 20
		} else {
			signals.Signals["STOCH"] = "HOLD"
			signals.Strength["STOCH"] = 0.3
		}
	}

	// Calculate overall signal
	signals.calculateOverallSignal()

	return signals
}

// TechnicalSignals represents technical analysis signals
type TechnicalSignals struct {
	Symbol         string             `json:"symbol"`
	Timestamp      time.Time          `json:"timestamp"`
	Signals        map[string]string  `json:"signals"`        // Indicator -> BUY/SELL/HOLD
	Strength       map[string]float64 `json:"strength"`       // Signal strength 0-1
	OverallSignal  string             `json:"overall_signal"`
	OverallStrength float64           `json:"overall_strength"`
}

// calculateOverallSignal calculates the overall signal based on individual indicators
func (ts *TechnicalSignals) calculateOverallSignal() {
	buyScore := 0.0
	sellScore := 0.0
	totalWeight := 0.0

	for indicator, signal := range ts.Signals {
		strength := ts.Strength[indicator]
		weight := ta.getIndicatorWeight(indicator)
		totalWeight += weight

		switch signal {
		case "BUY":
			buyScore += strength * weight
		case "SELL":
			sellScore += strength * weight
		}
	}

	if totalWeight == 0 {
		ts.OverallSignal = "HOLD"
		ts.OverallStrength = 0.0
		return
	}

	buyScore /= totalWeight
	sellScore /= totalWeight

	if buyScore > sellScore && buyScore > 0.5 {
		ts.OverallSignal = "BUY"
		ts.OverallStrength = buyScore
	} else if sellScore > buyScore && sellScore > 0.5 {
		ts.OverallSignal = "SELL"
		ts.OverallStrength = sellScore
	} else {
		ts.OverallSignal = "HOLD"
		ts.OverallStrength = math.Max(buyScore, sellScore)
	}
}

// getIndicatorWeight returns the weight for different indicators
func (ta *TechnicalAnalyzer) getIndicatorWeight(indicator string) float64 {
	weights := map[string]float64{
		"RSI":   0.8,
		"MA":    1.0,
		"MACD":  0.9,
		"STOCH": 0.7,
		"CCI":   0.6,
		"ADX":   0.5,
	}

	if weight, exists := weights[indicator]; exists {
		return weight
	}
	return 0.5 // Default weight
}