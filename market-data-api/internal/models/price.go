package models

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// Price represents a cryptocurrency price from a specific provider
type Price struct {
	Symbol    string          `json:"symbol" validate:"required"`
	Price     decimal.Decimal `json:"price" validate:"required,gte=0"`
	PriceUSD  decimal.Decimal `json:"price_usd"`
	Timestamp time.Time       `json:"timestamp" validate:"required"`
	Source    string          `json:"source" validate:"required"`
	Provider  string          `json:"provider,omitempty"`

	// Market data
	Volume24h     decimal.Decimal `json:"volume_24h,omitempty"`
	MarketCap     decimal.Decimal `json:"market_cap,omitempty"`
	Change24h     decimal.Decimal `json:"change_24h,omitempty"`
	ChangePercent decimal.Decimal `json:"change_percent_24h,omitempty"`

	// Quality indicators
	Confidence float64 `json:"confidence,omitempty"`
	Latency    int64   `json:"latency_ms,omitempty"`
}

// AggregatedPrice represents a price aggregated from multiple providers
type AggregatedPrice struct {
	Symbol         string                     `json:"symbol"`
	Price          decimal.Decimal            `json:"price"`
	PriceUSD       decimal.Decimal            `json:"price_usd"`
	Timestamp      time.Time                  `json:"timestamp"`
	Source         string                     `json:"source"`
	Confidence     float64                    `json:"confidence_score"`

	// Aggregation metadata
	ProviderPrices map[string]*ProviderPrice  `json:"provider_prices,omitempty"`
	Metadata       *AggregationMetadata       `json:"metadata,omitempty"`

	// Market data
	Volume         decimal.Decimal            `json:"volume,omitempty"`
	Volume24h      decimal.Decimal            `json:"volume_24h,omitempty"`
	MarketCap      decimal.Decimal            `json:"market_cap,omitempty"`
	Change24h      decimal.Decimal            `json:"change_24h,omitempty"`
	ChangePercent  decimal.Decimal            `json:"change_percent_24h,omitempty"`
}

// ProviderPrice represents price data from a specific provider
type ProviderPrice struct {
	Price     decimal.Decimal `json:"price"`
	Timestamp time.Time       `json:"timestamp"`
	Latency   time.Duration   `json:"latency_ms"`
	Weight    float64         `json:"weight"`
	IsOutlier bool            `json:"is_outlier"`
	Error     string          `json:"error,omitempty"`
}

// AggregationMetadata contains metadata about price aggregation
type AggregationMetadata struct {
	Method          string            `json:"aggregation_method"`
	ProvidersUsed   []string          `json:"providers_used"`
	OutliersRemoved int               `json:"outliers_removed"`
	LastUpdate      time.Time         `json:"last_update"`
	ProcessingTime  time.Duration     `json:"processing_time_ms"`
	Weights         map[string]float64 `json:"weights,omitempty"`
}

// PriceHistory represents historical price data
type PriceHistory struct {
	Symbol   string     `json:"symbol"`
	Interval string     `json:"interval"`
	Candles  []*Candle  `json:"candles"`
	Metadata *Metadata  `json:"metadata,omitempty"`
}

// Candle represents OHLCV data for a specific time period
type Candle struct {
	Timestamp time.Time       `json:"timestamp"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
	Volume    decimal.Decimal `json:"volume"`

	// Additional metrics
	VWAP      decimal.Decimal `json:"vwap,omitempty"`      // Volume Weighted Average Price
	Trades    int64           `json:"trades,omitempty"`    // Number of trades
	QuoteVol  decimal.Decimal `json:"quote_volume,omitempty"` // Quote asset volume
}

// Metadata contains additional information about the data
type Metadata struct {
	TotalCandles int         `json:"total_candles"`
	TimeRange    *TimeRange  `json:"time_range"`
	Source       string      `json:"source,omitempty"`
	Quality      float64     `json:"quality,omitempty"`
}

// TimeRange represents a time range
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// OrderBook represents order book data
type OrderBook struct {
	Symbol      string        `json:"symbol"`
	Bids        []*OrderLevel `json:"bids"`
	Asks        []*OrderLevel `json:"asks"`
	Timestamp   time.Time     `json:"timestamp"`
	Spread      decimal.Decimal `json:"spread"`
	SpreadPct   decimal.Decimal `json:"spread_percentage"`
	LastUpdate  time.Time     `json:"last_update"`
	Source      string        `json:"source,omitempty"`
}

// OrderLevel represents a single level in the order book
type OrderLevel struct {
	Price  decimal.Decimal `json:"price"`
	Amount decimal.Decimal `json:"amount"`
	Total  decimal.Decimal `json:"total,omitempty"`
	Count  int             `json:"count,omitempty"`
}

// ProviderStatus represents the status of a data provider
type ProviderStatus struct {
	Name        string        `json:"name"`
	Status      string        `json:"status"` // healthy, degraded, down
	Latency     time.Duration `json:"latency"`
	LastUpdate  time.Time     `json:"last_update"`
	ErrorCount  int           `json:"error_count"`
	SuccessRate float64       `json:"success_rate"`

	// Additional metrics
	ResponseTime time.Duration `json:"avg_response_time"`
	RateLimit    int           `json:"rate_limit_remaining,omitempty"`
	Weight       float64       `json:"weight"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// Error implements the error interface
func (ve ValidationError) Error() string {
	return ve.Message
}

// Validate validates the price model
func (p *Price) Validate() error {
	if p.Symbol == "" {
		return ValidationError{Field: "symbol", Message: "symbol is required"}
	}

	if p.Price.LessThan(decimal.Zero) {
		return ValidationError{Field: "price", Message: "price must be non-negative", Value: p.Price}
	}

	if p.Timestamp.IsZero() {
		return ValidationError{Field: "timestamp", Message: "timestamp is required"}
	}

	if p.Source == "" {
		return ValidationError{Field: "source", Message: "source is required"}
	}

	return nil
}

// IsStale checks if the price data is stale based on a threshold
func (p *Price) IsStale(threshold time.Duration) bool {
	return time.Since(p.Timestamp) > threshold
}

// ToJSON converts the price to JSON bytes
func (p *Price) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// FromJSON creates a Price from JSON bytes
func (p *Price) FromJSON(data []byte) error {
	return json.Unmarshal(data, p)
}

// CalculateSpread calculates the bid-ask spread
func (ob *OrderBook) CalculateSpread() {
	if len(ob.Bids) > 0 && len(ob.Asks) > 0 {
		bestBid := ob.Bids[0].Price
		bestAsk := ob.Asks[0].Price

		ob.Spread = bestAsk.Sub(bestBid)
		if bestBid.GreaterThan(decimal.Zero) {
			ob.SpreadPct = ob.Spread.Div(bestBid).Mul(decimal.NewFromInt(100))
		}
	}
}

// GetMidPrice returns the mid price from the order book
func (ob *OrderBook) GetMidPrice() decimal.Decimal {
	if len(ob.Bids) > 0 && len(ob.Asks) > 0 {
		bestBid := ob.Bids[0].Price
		bestAsk := ob.Asks[0].Price
		return bestBid.Add(bestAsk).Div(decimal.NewFromInt(2))
	}
	return decimal.Zero
}

// GetBestBid returns the best bid price
func (ob *OrderBook) GetBestBid() decimal.Decimal {
	if len(ob.Bids) > 0 {
		return ob.Bids[0].Price
	}
	return decimal.Zero
}

// GetBestAsk returns the best ask price
func (ob *OrderBook) GetBestAsk() decimal.Decimal {
	if len(ob.Asks) > 0 {
		return ob.Asks[0].Price
	}
	return decimal.Zero
}

// CalculateVWAP calculates Volume Weighted Average Price for a candle
func (c *Candle) CalculateVWAP() {
	if c.Volume.GreaterThan(decimal.Zero) {
		typicalPrice := c.High.Add(c.Low).Add(c.Close).Div(decimal.NewFromInt(3))
		c.VWAP = typicalPrice.Mul(c.Volume).Div(c.Volume)
	}
}

// GetReturn calculates the return from open to close
func (c *Candle) GetReturn() decimal.Decimal {
	if c.Open.GreaterThan(decimal.Zero) {
		return c.Close.Sub(c.Open).Div(c.Open)
	}
	return decimal.Zero
}

// GetTrueRange calculates the true range for volatility calculations
func (c *Candle) GetTrueRange(prevClose decimal.Decimal) decimal.Decimal {
	hl := c.High.Sub(c.Low)
	hc := c.High.Sub(prevClose).Abs()
	lc := c.Low.Sub(prevClose).Abs()

	tr := hl
	if hc.GreaterThan(tr) {
		tr = hc
	}
	if lc.GreaterThan(tr) {
		tr = lc
	}

	return tr
}

// NewPrice creates a new Price instance
func NewPrice(symbol, source string, price decimal.Decimal) *Price {
	return &Price{
		Symbol:    symbol,
		Price:     price,
		PriceUSD:  price,
		Timestamp: time.Now(),
		Source:    source,
		Confidence: 1.0,
	}
}

// NewAggregatedPrice creates a new AggregatedPrice instance
func NewAggregatedPrice(symbol string, price decimal.Decimal, confidence float64) *AggregatedPrice {
	return &AggregatedPrice{
		Symbol:         symbol,
		Price:          price,
		PriceUSD:       price,
		Timestamp:      time.Now(),
		Source:         "aggregated",
		Confidence:     confidence,
		ProviderPrices: make(map[string]*ProviderPrice),
	}
}

// NewOrderBook creates a new OrderBook instance
func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		Symbol:    symbol,
		Bids:      make([]*OrderLevel, 0),
		Asks:      make([]*OrderLevel, 0),
		Timestamp: time.Now(),
	}
}