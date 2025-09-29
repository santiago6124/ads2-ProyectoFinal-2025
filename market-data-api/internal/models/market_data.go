package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// MarketData represents comprehensive market data for a cryptocurrency
type MarketData struct {
	Symbol                    string          `json:"symbol"`
	Name                      string          `json:"name,omitempty"`
	CurrentPrice              decimal.Decimal `json:"current_price"`
	MarketCap                 decimal.Decimal `json:"market_cap"`
	FullyDilutedValuation     decimal.Decimal `json:"fully_diluted_valuation,omitempty"`
	TotalVolume               decimal.Decimal `json:"total_volume"`

	// 24h data
	High24h                   decimal.Decimal `json:"high_24h"`
	Low24h                    decimal.Decimal `json:"low_24h"`
	PriceChange24h            decimal.Decimal `json:"price_change_24h"`
	PriceChangePercentage24h  decimal.Decimal `json:"price_change_percentage_24h"`
	MarketCapChange24h        decimal.Decimal `json:"market_cap_change_24h,omitempty"`
	MarketCapChangePercentage24h decimal.Decimal `json:"market_cap_change_percentage_24h,omitempty"`

	// Extended period changes
	PriceChangePercentage7d   decimal.Decimal `json:"price_change_percentage_7d,omitempty"`
	PriceChangePercentage30d  decimal.Decimal `json:"price_change_percentage_30d,omitempty"`
	PriceChangePercentage1y   decimal.Decimal `json:"price_change_percentage_1y,omitempty"`

	// All-time data
	ATH                       decimal.Decimal `json:"ath"`
	ATHChangePercentage       decimal.Decimal `json:"ath_change_percentage"`
	ATHDate                   *time.Time      `json:"ath_date,omitempty"`
	ATL                       decimal.Decimal `json:"atl"`
	ATLChangePercentage       decimal.Decimal `json:"atl_change_percentage"`
	ATLDate                   *time.Time      `json:"atl_date,omitempty"`

	// Supply information
	CirculatingSupply         decimal.Decimal `json:"circulating_supply,omitempty"`
	TotalSupply               decimal.Decimal `json:"total_supply,omitempty"`
	MaxSupply                 decimal.Decimal `json:"max_supply,omitempty"`

	// Market metrics
	MarketMetrics             *MarketMetrics  `json:"market_metrics,omitempty"`

	// Metadata
	LastUpdated               time.Time       `json:"last_updated"`
	DataSource                string          `json:"data_source,omitempty"`
	Confidence                float64         `json:"confidence,omitempty"`
}

// MarketMetrics represents advanced market metrics
type MarketMetrics struct {
	// Volatility measures
	Volatility24h             decimal.Decimal `json:"volatility_24h"`
	Volatility7d              decimal.Decimal `json:"volatility_7d"`
	Volatility30d             decimal.Decimal `json:"volatility_30d,omitempty"`

	// Risk metrics
	SharpeRatio               decimal.Decimal `json:"sharpe_ratio,omitempty"`
	Beta                      decimal.Decimal `json:"beta,omitempty"`
	CorrelationWithMarket     decimal.Decimal `json:"correlation_with_market,omitempty"`
	VaR95                     decimal.Decimal `json:"var_95,omitempty"` // Value at Risk 95%

	// Market structure
	MarketDominance           decimal.Decimal `json:"market_dominance,omitempty"`
	LiquidityScore            decimal.Decimal `json:"liquidity_score,omitempty"`
	AverageSpread             decimal.Decimal `json:"average_spread,omitempty"`

	// Trading metrics
	TurnoverRate              decimal.Decimal `json:"turnover_rate,omitempty"`
	AverageTradingVolume7d    decimal.Decimal `json:"avg_trading_volume_7d,omitempty"`
	AverageTradingVolume30d   decimal.Decimal `json:"avg_trading_volume_30d,omitempty"`

	// Technical indicators
	RSI                       decimal.Decimal `json:"rsi,omitempty"`
	MACD                      decimal.Decimal `json:"macd,omitempty"`
	MovingAverage50           decimal.Decimal `json:"ma_50,omitempty"`
	MovingAverage200          decimal.Decimal `json:"ma_200,omitempty"`
}

// PriceAlert represents a price alert configuration
type PriceAlert struct {
	ID           string          `json:"id"`
	Symbol       string          `json:"symbol"`
	UserID       string          `json:"user_id,omitempty"`
	AlertType    AlertType       `json:"alert_type"`
	TargetPrice  decimal.Decimal `json:"target_price"`
	CurrentPrice decimal.Decimal `json:"current_price,omitempty"`
	Condition    AlertCondition  `json:"condition"`
	IsActive     bool            `json:"is_active"`
	CreatedAt    time.Time       `json:"created_at"`
	TriggeredAt  *time.Time      `json:"triggered_at,omitempty"`
	ExpiresAt    *time.Time      `json:"expires_at,omitempty"`
}

// AlertType represents different types of price alerts
type AlertType string

const (
	AlertTypePrice      AlertType = "price"
	AlertTypePercentage AlertType = "percentage"
	AlertTypeVolume     AlertType = "volume"
	AlertTypeMarketCap  AlertType = "market_cap"
)

// AlertCondition represents alert trigger conditions
type AlertCondition string

const (
	ConditionAbove AlertCondition = "above"
	ConditionBelow AlertCondition = "below"
	ConditionEqual AlertCondition = "equal"
)

// MarketSummary represents a summary of overall market conditions
type MarketSummary struct {
	TotalMarketCap           decimal.Decimal            `json:"total_market_cap"`
	TotalVolume24h           decimal.Decimal            `json:"total_volume_24h"`
	BTCDominance             decimal.Decimal            `json:"btc_dominance"`
	ETHDominance             decimal.Decimal            `json:"eth_dominance"`
	ActiveCryptocurrencies   int                        `json:"active_cryptocurrencies"`
	Markets                  int                        `json:"markets"`
	MarketCapChange24h       decimal.Decimal            `json:"market_cap_change_24h"`
	VolumeChange24h          decimal.Decimal            `json:"volume_change_24h"`
	TopGainers               []MarketData               `json:"top_gainers,omitempty"`
	TopLosers                []MarketData               `json:"top_losers,omitempty"`
	TrendingCoins            []string                   `json:"trending_coins,omitempty"`
	LastUpdated              time.Time                  `json:"last_updated"`
}

// ExchangeData represents data from a specific exchange
type ExchangeData struct {
	ExchangeID    string                     `json:"exchange_id"`
	ExchangeName  string                     `json:"exchange_name"`
	Symbol        string                     `json:"symbol"`
	Price         decimal.Decimal            `json:"price"`
	Volume24h     decimal.Decimal            `json:"volume_24h"`
	LastTrade     time.Time                  `json:"last_trade"`
	Spread        decimal.Decimal            `json:"spread,omitempty"`
	OrderBook     *OrderBook                 `json:"order_book,omitempty"`
	TradingPairs  []string                   `json:"trading_pairs,omitempty"`
}

// HistoricalStats represents historical statistical data
type HistoricalStats struct {
	Symbol              string          `json:"symbol"`
	Period              string          `json:"period"`
	StartDate           time.Time       `json:"start_date"`
	EndDate             time.Time       `json:"end_date"`

	// Price statistics
	MaxPrice            decimal.Decimal `json:"max_price"`
	MinPrice            decimal.Decimal `json:"min_price"`
	AveragePrice        decimal.Decimal `json:"average_price"`
	MedianPrice         decimal.Decimal `json:"median_price"`

	// Volume statistics
	MaxVolume           decimal.Decimal `json:"max_volume"`
	MinVolume           decimal.Decimal `json:"min_volume"`
	AverageVolume       decimal.Decimal `json:"average_volume"`
	TotalVolume         decimal.Decimal `json:"total_volume"`

	// Volatility metrics
	StandardDeviation   decimal.Decimal `json:"standard_deviation"`
	Variance            decimal.Decimal `json:"variance"`
	AnnualizedVolatility decimal.Decimal `json:"annualized_volatility"`

	// Return metrics
	TotalReturn         decimal.Decimal `json:"total_return"`
	AnnualizedReturn    decimal.Decimal `json:"annualized_return"`
	MaxDrawdown         decimal.Decimal `json:"max_drawdown"`

	// Trading metrics
	TradingDays         int             `json:"trading_days"`
	PositiveDays        int             `json:"positive_days"`
	NegativeDays        int             `json:"negative_days"`
	WinRate             decimal.Decimal `json:"win_rate"`
}

// CalculateMarketMetrics calculates advanced market metrics
func (md *MarketData) CalculateMarketMetrics(historicalData []*Candle) {
	if md.MarketMetrics == nil {
		md.MarketMetrics = &MarketMetrics{}
	}

	// Calculate volatility
	md.MarketMetrics.Volatility24h = calculateVolatility(historicalData, 24)
	md.MarketMetrics.Volatility7d = calculateVolatility(historicalData, 24*7)

	// Calculate turnover rate
	if md.MarketCap.GreaterThan(decimal.Zero) {
		md.MarketMetrics.TurnoverRate = md.TotalVolume.Div(md.MarketCap).Mul(decimal.NewFromInt(100))
	}
}

// calculateVolatility calculates price volatility over a specified number of periods
func calculateVolatility(candles []*Candle, periods int) decimal.Decimal {
	if len(candles) < periods {
		return decimal.Zero
	}

	// Take last N candles
	recentCandles := candles[len(candles)-periods:]
	returns := make([]decimal.Decimal, len(recentCandles)-1)

	// Calculate returns
	for i := 1; i < len(recentCandles); i++ {
		if recentCandles[i-1].Close.GreaterThan(decimal.Zero) {
			returns[i-1] = recentCandles[i].Close.Div(recentCandles[i-1].Close).Sub(decimal.NewFromInt(1))
		}
	}

	// Calculate standard deviation of returns
	return calculateStandardDeviation(returns)
}

// calculateStandardDeviation calculates the standard deviation of a slice of decimals
func calculateStandardDeviation(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	// Calculate mean
	sum := decimal.Zero
	for _, v := range values {
		sum = sum.Add(v)
	}
	mean := sum.Div(decimal.NewFromInt(int64(len(values))))

	// Calculate variance
	sumSquaredDiffs := decimal.Zero
	for _, v := range values {
		diff := v.Sub(mean)
		sumSquaredDiffs = sumSquaredDiffs.Add(diff.Mul(diff))
	}
	variance := sumSquaredDiffs.Div(decimal.NewFromInt(int64(len(values))))

	// Return square root of variance (standard deviation)
	return variance.Pow(decimal.NewFromFloat(0.5))
}

// IsTriggered checks if a price alert should be triggered
func (pa *PriceAlert) IsTriggered(currentPrice decimal.Decimal) bool {
	if !pa.IsActive {
		return false
	}

	switch pa.Condition {
	case ConditionAbove:
		return currentPrice.GreaterThan(pa.TargetPrice)
	case ConditionBelow:
		return currentPrice.LessThan(pa.TargetPrice)
	case ConditionEqual:
		// Use a small threshold for equality comparison
		threshold := pa.TargetPrice.Mul(decimal.NewFromFloat(0.001)) // 0.1% threshold
		diff := currentPrice.Sub(pa.TargetPrice).Abs()
		return diff.LessThanOrEqual(threshold)
	default:
		return false
	}
}

// Validate validates the market data
func (md *MarketData) Validate() error {
	if md.Symbol == "" {
		return ValidationError{Field: "symbol", Message: "symbol is required"}
	}

	if md.CurrentPrice.LessThan(decimal.Zero) {
		return ValidationError{Field: "current_price", Message: "current price must be non-negative"}
	}

	if md.MarketCap.LessThan(decimal.Zero) {
		return ValidationError{Field: "market_cap", Message: "market cap must be non-negative"}
	}

	return nil
}

// GetPriceChangeDirection returns the direction of price change
func (md *MarketData) GetPriceChangeDirection() string {
	if md.PriceChange24h.GreaterThan(decimal.Zero) {
		return "up"
	} else if md.PriceChange24h.LessThan(decimal.Zero) {
		return "down"
	}
	return "neutral"
}

// CalculateRSI calculates the Relative Strength Index
func CalculateRSI(candles []*Candle, period int) decimal.Decimal {
	if len(candles) < period+1 {
		return decimal.Zero
	}

	gains := make([]decimal.Decimal, 0)
	losses := make([]decimal.Decimal, 0)

	// Calculate gains and losses
	for i := 1; i < len(candles); i++ {
		change := candles[i].Close.Sub(candles[i-1].Close)
		if change.GreaterThan(decimal.Zero) {
			gains = append(gains, change)
			losses = append(losses, decimal.Zero)
		} else {
			gains = append(gains, decimal.Zero)
			losses = append(losses, change.Abs())
		}
	}

	// Calculate average gains and losses
	avgGain := calculateAverage(gains[len(gains)-period:])
	avgLoss := calculateAverage(losses[len(losses)-period:])

	if avgLoss.Equal(decimal.Zero) {
		return decimal.NewFromInt(100)
	}

	rs := avgGain.Div(avgLoss)
	rsi := decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))

	return rsi
}

// calculateAverage calculates the average of a slice of decimals
func calculateAverage(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	sum := decimal.Zero
	for _, v := range values {
		sum = sum.Add(v)
	}

	return sum.Div(decimal.NewFromInt(int64(len(values))))
}

// NewMarketData creates a new MarketData instance
func NewMarketData(symbol string, currentPrice decimal.Decimal) *MarketData {
	return &MarketData{
		Symbol:       symbol,
		CurrentPrice: currentPrice,
		LastUpdated:  time.Now(),
		Confidence:   1.0,
	}
}

// NewPriceAlert creates a new PriceAlert instance
func NewPriceAlert(symbol, userID string, alertType AlertType, targetPrice decimal.Decimal, condition AlertCondition) *PriceAlert {
	return &PriceAlert{
		Symbol:      symbol,
		UserID:      userID,
		AlertType:   alertType,
		TargetPrice: targetPrice,
		Condition:   condition,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}
}