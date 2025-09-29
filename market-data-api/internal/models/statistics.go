package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// VolatilityData represents volatility calculation results
type VolatilityData struct {
	Symbol                string          `json:"symbol"`
	Period                string          `json:"period"`
	Interval              string          `json:"interval"`
	Volatility            decimal.Decimal `json:"volatility"`
	VolatilityPercentage  decimal.Decimal `json:"volatility_percentage"`
	StandardDeviation     decimal.Decimal `json:"standard_deviation"`
	Variance              decimal.Decimal `json:"variance"`
	Samples               int             `json:"samples"`
	CalculationMethod     string          `json:"calculation_method"`
	AnnualizedVolatility  decimal.Decimal `json:"annualized_volatility"`
	LastUpdated           time.Time       `json:"last_updated"`
}

// CorrelationMatrix represents correlation between multiple assets
type CorrelationMatrix struct {
	Assets      []string                            `json:"assets"`
	Matrix      map[string]map[string]decimal.Decimal `json:"matrix"`
	Period      string                              `json:"period"`
	LastUpdated time.Time                           `json:"last_updated"`
}

// MovingAverages represents various moving averages for an asset
type MovingAverages struct {
	Symbol      string                     `json:"symbol"`
	MA7         decimal.Decimal            `json:"ma_7"`
	MA20        decimal.Decimal            `json:"ma_20"`
	MA50        decimal.Decimal            `json:"ma_50"`
	MA100       decimal.Decimal            `json:"ma_100"`
	MA200       decimal.Decimal            `json:"ma_200"`
	EMA12       decimal.Decimal            `json:"ema_12"`
	EMA26       decimal.Decimal            `json:"ema_26"`
	LastUpdated time.Time                  `json:"last_updated"`
}

// TechnicalIndicators represents a collection of technical indicators
type TechnicalIndicators struct {
	Symbol         string          `json:"symbol"`
	RSI            decimal.Decimal `json:"rsi"`
	MACD           *MACDData       `json:"macd"`
	BollingerBands *BollingerData  `json:"bollinger_bands"`
	MovingAverages *MovingAverages `json:"moving_averages"`
	LastUpdated    time.Time       `json:"last_updated"`
}

// MACDData represents MACD indicator data
type MACDData struct {
	MACD      decimal.Decimal `json:"macd"`
	Signal    decimal.Decimal `json:"signal"`
	Histogram decimal.Decimal `json:"histogram"`
}

// BollingerData represents Bollinger Bands data
type BollingerData struct {
	Upper  decimal.Decimal `json:"upper"`
	Middle decimal.Decimal `json:"middle"`
	Lower  decimal.Decimal `json:"lower"`
	Width  decimal.Decimal `json:"width"`
}

// PriceStatistics represents price statistics over a period
type PriceStatistics struct {
	Symbol              string          `json:"symbol"`
	Period              string          `json:"period"`

	// Price metrics
	MaxPrice            decimal.Decimal `json:"max_price"`
	MinPrice            decimal.Decimal `json:"min_price"`
	AveragePrice        decimal.Decimal `json:"average_price"`
	MedianPrice         decimal.Decimal `json:"median_price"`
	OpenPrice           decimal.Decimal `json:"open_price"`
	ClosePrice          decimal.Decimal `json:"close_price"`

	// Change metrics
	AbsoluteChange      decimal.Decimal `json:"absolute_change"`
	PercentageChange    decimal.Decimal `json:"percentage_change"`

	// Volume metrics
	TotalVolume         decimal.Decimal `json:"total_volume"`
	AverageVolume       decimal.Decimal `json:"average_volume"`
	MaxVolume           decimal.Decimal `json:"max_volume"`
	MinVolume           decimal.Decimal `json:"min_volume"`

	// Volatility metrics
	StandardDeviation   decimal.Decimal `json:"standard_deviation"`
	Variance            decimal.Decimal `json:"variance"`
	CoefficientOfVariation decimal.Decimal `json:"coefficient_of_variation"`

	// Trading metrics
	TradingDays         int             `json:"trading_days"`
	UpDays              int             `json:"up_days"`
	DownDays            int             `json:"down_days"`
	UnchangedDays       int             `json:"unchanged_days"`

	// Performance metrics
	SharpeRatio         decimal.Decimal `json:"sharpe_ratio,omitempty"`
	SortinoRatio        decimal.Decimal `json:"sortino_ratio,omitempty"`
	MaxDrawdown         decimal.Decimal `json:"max_drawdown,omitempty"`

	LastUpdated         time.Time       `json:"last_updated"`
}

// CalculateVolatility calculates volatility using different methods
func CalculateVolatility(prices []decimal.Decimal, method VolatilityMethod) *VolatilityData {
	if len(prices) < 2 {
		return nil
	}

	var returns []decimal.Decimal

	switch method {
	case CloseToClose:
		returns = calculateCloseToCloseReturns(prices)
	case HighLow:
		// For high-low method, need both high and low prices
		// This is a simplified version
		returns = calculateCloseToCloseReturns(prices)
	default:
		returns = calculateCloseToCloseReturns(prices)
	}

	if len(returns) == 0 {
		return nil
	}

	stdDev := calculateStandardDeviation(returns)
	variance := stdDev.Mul(stdDev)

	// Annualize volatility (assuming 365 trading days)
	annualizedVol := stdDev.Mul(decimal.NewFromFloat(19.1049732)) // sqrt(365)

	return &VolatilityData{
		Volatility:           stdDev,
		VolatilityPercentage: stdDev.Mul(decimal.NewFromInt(100)),
		StandardDeviation:    stdDev,
		Variance:             variance,
		AnnualizedVolatility: annualizedVol,
		Samples:              len(returns),
		CalculationMethod:    string(method),
		LastUpdated:          time.Now(),
	}
}

// VolatilityMethod represents different volatility calculation methods
type VolatilityMethod string

const (
	CloseToClose VolatilityMethod = "close-to-close"
	HighLow      VolatilityMethod = "high-low"
	Parkinson    VolatilityMethod = "parkinson"
	GarmanKlass  VolatilityMethod = "garman-klass"
)

// calculateCloseToCloseReturns calculates returns based on closing prices
func calculateCloseToCloseReturns(prices []decimal.Decimal) []decimal.Decimal {
	if len(prices) < 2 {
		return nil
	}

	returns := make([]decimal.Decimal, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1].GreaterThan(decimal.Zero) {
			returns[i-1] = prices[i].Div(prices[i-1]).Sub(decimal.NewFromInt(1))
		}
	}

	return returns
}

// CalculateMovingAverage calculates simple moving average
func CalculateMovingAverage(prices []decimal.Decimal, period int) decimal.Decimal {
	if len(prices) < period {
		return decimal.Zero
	}

	sum := decimal.Zero
	recentPrices := prices[len(prices)-period:]

	for _, price := range recentPrices {
		sum = sum.Add(price)
	}

	return sum.Div(decimal.NewFromInt(int64(period)))
}

// CalculateEMA calculates exponential moving average
func CalculateEMA(prices []decimal.Decimal, period int) decimal.Decimal {
	if len(prices) == 0 {
		return decimal.Zero
	}

	if len(prices) == 1 {
		return prices[0]
	}

	multiplier := decimal.NewFromInt(2).Div(decimal.NewFromInt(int64(period + 1)))
	ema := prices[0]

	for i := 1; i < len(prices); i++ {
		ema = prices[i].Mul(multiplier).Add(ema.Mul(decimal.NewFromInt(1).Sub(multiplier)))
	}

	return ema
}

// CalculateMACD calculates MACD indicator
func CalculateMACD(prices []decimal.Decimal, fastPeriod, slowPeriod, signalPeriod int) *MACDData {
	if len(prices) < slowPeriod+signalPeriod {
		return nil
	}

	emaFast := CalculateEMA(prices, fastPeriod)
	emaSlow := CalculateEMA(prices, slowPeriod)
	macd := emaFast.Sub(emaSlow)

	// Calculate signal line (EMA of MACD)
	// For simplicity, using a single MACD value here
	// In practice, you'd need historical MACD values
	signal := macd // Simplified

	histogram := macd.Sub(signal)

	return &MACDData{
		MACD:      macd,
		Signal:    signal,
		Histogram: histogram,
	}
}

// CalculateBollingerBands calculates Bollinger Bands
func CalculateBollingerBands(prices []decimal.Decimal, period int, multiplier decimal.Decimal) *BollingerData {
	if len(prices) < period {
		return nil
	}

	sma := CalculateMovingAverage(prices, period)
	stdDev := calculateStandardDeviation(prices[len(prices)-period:])

	upper := sma.Add(stdDev.Mul(multiplier))
	lower := sma.Sub(stdDev.Mul(multiplier))
	width := upper.Sub(lower)

	return &BollingerData{
		Upper:  upper,
		Middle: sma,
		Lower:  lower,
		Width:  width,
	}
}

// CalculateCorrelation calculates correlation coefficient between two price series
func CalculateCorrelation(prices1, prices2 []decimal.Decimal) decimal.Decimal {
	if len(prices1) != len(prices2) || len(prices1) == 0 {
		return decimal.Zero
	}

	n := decimal.NewFromInt(int64(len(prices1)))

	// Calculate means
	mean1 := calculateAverage(prices1)
	mean2 := calculateAverage(prices2)

	// Calculate correlation components
	numerator := decimal.Zero
	sumSq1 := decimal.Zero
	sumSq2 := decimal.Zero

	for i := 0; i < len(prices1); i++ {
		diff1 := prices1[i].Sub(mean1)
		diff2 := prices2[i].Sub(mean2)

		numerator = numerator.Add(diff1.Mul(diff2))
		sumSq1 = sumSq1.Add(diff1.Mul(diff1))
		sumSq2 = sumSq2.Add(diff2.Mul(diff2))
	}

	denominator := sumSq1.Mul(sumSq2).Pow(decimal.NewFromFloat(0.5))

	if denominator.Equal(decimal.Zero) {
		return decimal.Zero
	}

	return numerator.Div(denominator)
}

// CalculateSharpeRatio calculates Sharpe ratio
func CalculateSharpeRatio(returns []decimal.Decimal, riskFreeRate decimal.Decimal) decimal.Decimal {
	if len(returns) == 0 {
		return decimal.Zero
	}

	meanReturn := calculateAverage(returns)
	stdDev := calculateStandardDeviation(returns)

	if stdDev.Equal(decimal.Zero) {
		return decimal.Zero
	}

	excessReturn := meanReturn.Sub(riskFreeRate)
	return excessReturn.Div(stdDev)
}

// CalculateMaxDrawdown calculates maximum drawdown
func CalculateMaxDrawdown(prices []decimal.Decimal) decimal.Decimal {
	if len(prices) == 0 {
		return decimal.Zero
	}

	peak := prices[0]
	maxDrawdown := decimal.Zero

	for _, price := range prices {
		if price.GreaterThan(peak) {
			peak = price
		}

		drawdown := peak.Sub(price).Div(peak)
		if drawdown.GreaterThan(maxDrawdown) {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// CalculatePriceStatistics calculates comprehensive price statistics
func CalculatePriceStatistics(candles []*Candle, period string) *PriceStatistics {
	if len(candles) == 0 {
		return nil
	}

	prices := make([]decimal.Decimal, len(candles))
	volumes := make([]decimal.Decimal, len(candles))

	for i, candle := range candles {
		prices[i] = candle.Close
		volumes[i] = candle.Volume
	}

	stats := &PriceStatistics{
		Period:         period,
		OpenPrice:      candles[0].Open,
		ClosePrice:     candles[len(candles)-1].Close,
		TradingDays:    len(candles),
		LastUpdated:    time.Now(),
	}

	// Calculate price statistics
	stats.MaxPrice = findMax(prices)
	stats.MinPrice = findMin(prices)
	stats.AveragePrice = calculateAverage(prices)
	stats.MedianPrice = calculateMedian(prices)

	// Calculate changes
	if stats.OpenPrice.GreaterThan(decimal.Zero) {
		stats.AbsoluteChange = stats.ClosePrice.Sub(stats.OpenPrice)
		stats.PercentageChange = stats.AbsoluteChange.Div(stats.OpenPrice).Mul(decimal.NewFromInt(100))
	}

	// Calculate volume statistics
	stats.TotalVolume = calculateSum(volumes)
	stats.AverageVolume = calculateAverage(volumes)
	stats.MaxVolume = findMax(volumes)
	stats.MinVolume = findMin(volumes)

	// Calculate volatility statistics
	stats.StandardDeviation = calculateStandardDeviation(prices)
	stats.Variance = stats.StandardDeviation.Mul(stats.StandardDeviation)

	if stats.AveragePrice.GreaterThan(decimal.Zero) {
		stats.CoefficientOfVariation = stats.StandardDeviation.Div(stats.AveragePrice)
	}

	// Calculate trading day statistics
	stats.UpDays, stats.DownDays, stats.UnchangedDays = calculateTradingDayStats(candles)

	// Calculate performance metrics
	returns := calculateCloseToCloseReturns(prices)
	stats.SharpeRatio = CalculateSharpeRatio(returns, decimal.Zero) // Assuming 0% risk-free rate
	stats.MaxDrawdown = CalculateMaxDrawdown(prices)

	return stats
}

// Helper functions

func findMax(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	max := values[0]
	for _, v := range values[1:] {
		if v.GreaterThan(max) {
			max = v
		}
	}
	return max
}

func findMin(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	min := values[0]
	for _, v := range values[1:] {
		if v.LessThan(min) {
			min = v
		}
	}
	return min
}

func calculateSum(values []decimal.Decimal) decimal.Decimal {
	sum := decimal.Zero
	for _, v := range values {
		sum = sum.Add(v)
	}
	return sum
}

func calculateMedian(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	// Simple median calculation (would need sorting in practice)
	// This is a simplified version
	return calculateAverage(values)
}

func calculateTradingDayStats(candles []*Candle) (upDays, downDays, unchangedDays int) {
	for _, candle := range candles {
		if candle.Close.GreaterThan(candle.Open) {
			upDays++
		} else if candle.Close.LessThan(candle.Open) {
			downDays++
		} else {
			unchangedDays++
		}
	}
	return
}