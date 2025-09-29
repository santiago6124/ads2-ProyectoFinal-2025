package dto

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// HistoryRequest represents a request for historical price data
type HistoryRequest struct {
	Symbol   string `json:"symbol" validate:"required"`
	Interval string `json:"interval" validate:"required"`
	From     int64  `json:"from,omitempty"`
	To       int64  `json:"to,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// BatchPricesRequest represents a request for multiple cryptocurrency prices
type BatchPricesRequest struct {
	Symbols         []string `json:"symbols" validate:"required,min=1,max=100"`
	Include24hChange bool    `json:"include_24h_change,omitempty"`
	IncludeVolume   bool    `json:"include_volume,omitempty"`
	IncludeMarketCap bool   `json:"include_market_cap,omitempty"`
	Currency        string  `json:"currency,omitempty"`
}

// VolatilityRequest represents a request for volatility calculation
type VolatilityRequest struct {
	Symbol   string `json:"symbol" validate:"required"`
	Period   string `json:"period,omitempty"`
	Interval string `json:"interval,omitempty"`
	Method   string `json:"method,omitempty"`
}

// PriceRequest represents a request for current price data
type PriceRequest struct {
	Symbol          string `json:"symbol" validate:"required"`
	Source          string `json:"source,omitempty"`
	IncludeMetadata bool   `json:"include_metadata,omitempty"`
	Currency        string `json:"currency,omitempty"`
}

// OrderBookRequest represents a request for order book data
type OrderBookRequest struct {
	Symbol string `json:"symbol" validate:"required"`
	Depth  int    `json:"depth,omitempty"`
}

// StatsRequest represents a request for market statistics
type StatsRequest struct {
	Symbol          string `json:"symbol" validate:"required"`
	IncludeMetrics  bool   `json:"include_metrics,omitempty"`
	Period          string `json:"period,omitempty"`
}

// Validate validates the history request
func (hr *HistoryRequest) Validate() error {
	if hr.Symbol == "" {
		return errors.New("symbol is required")
	}

	// Validate interval
	validIntervals := []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d", "1w", "1M"}
	if !contains(validIntervals, hr.Interval) {
		return errors.New("invalid interval, must be one of: " + strings.Join(validIntervals, ", "))
	}

	// Validate time range
	if hr.From > 0 && hr.To > 0 && hr.From >= hr.To {
		return errors.New("from timestamp must be less than to timestamp")
	}

	// Validate limit
	if hr.Limit < 0 || hr.Limit > 1000 {
		return errors.New("limit must be between 0 and 1000")
	}

	// Check if time range is too large
	if hr.From > 0 && hr.To > 0 {
		duration := time.Unix(hr.To, 0).Sub(time.Unix(hr.From, 0))
		maxDuration := getMaxDurationForInterval(hr.Interval)
		if duration > maxDuration {
			return errors.New("time range too large for the specified interval")
		}
	}

	return nil
}

// SetDefaults sets default values for the history request
func (hr *HistoryRequest) SetDefaults() {
	if hr.Interval == "" {
		hr.Interval = "1h"
	}

	if hr.Limit == 0 {
		hr.Limit = 100
	}

	// Set default time range if not specified
	if hr.From == 0 && hr.To == 0 {
		now := time.Now()
		hr.To = now.Unix()
		hr.From = now.Add(-24 * time.Hour).Unix()
	}
}

// GetTimeRange returns the time range as time.Time objects
func (hr *HistoryRequest) GetTimeRange() (time.Time, time.Time) {
	from := time.Unix(hr.From, 0)
	to := time.Unix(hr.To, 0)
	return from, to
}

// GetIntervalDuration returns the interval as a time.Duration
func (hr *HistoryRequest) GetIntervalDuration() time.Duration {
	switch hr.Interval {
	case "1m":
		return time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return time.Hour
	case "4h":
		return 4 * time.Hour
	case "1d":
		return 24 * time.Hour
	case "1w":
		return 7 * 24 * time.Hour
	case "1M":
		return 30 * 24 * time.Hour
	default:
		return time.Hour
	}
}

// Validate validates the batch prices request
func (bpr *BatchPricesRequest) Validate() error {
	if len(bpr.Symbols) == 0 {
		return errors.New("symbols array cannot be empty")
	}

	if len(bpr.Symbols) > 100 {
		return errors.New("maximum 100 symbols allowed per request")
	}

	// Validate each symbol
	for _, symbol := range bpr.Symbols {
		if symbol == "" {
			return errors.New("empty symbol not allowed")
		}
		if len(symbol) > 20 {
			return errors.New("symbol too long (max 20 characters)")
		}
	}

	// Validate currency if specified
	if bpr.Currency != "" {
		validCurrencies := []string{"USD", "EUR", "GBP", "JPY", "BTC", "ETH"}
		if !contains(validCurrencies, strings.ToUpper(bpr.Currency)) {
			return errors.New("unsupported currency")
		}
	}

	return nil
}

// SetDefaults sets default values for the batch prices request
func (bpr *BatchPricesRequest) SetDefaults() {
	if bpr.Currency == "" {
		bpr.Currency = "USD"
	}

	// Remove duplicates and normalize symbols
	symbolMap := make(map[string]bool)
	uniqueSymbols := make([]string, 0, len(bpr.Symbols))

	for _, symbol := range bpr.Symbols {
		normalizedSymbol := strings.ToUpper(strings.TrimSpace(symbol))
		if normalizedSymbol != "" && !symbolMap[normalizedSymbol] {
			symbolMap[normalizedSymbol] = true
			uniqueSymbols = append(uniqueSymbols, normalizedSymbol)
		}
	}

	bpr.Symbols = uniqueSymbols
}

// Validate validates the volatility request
func (vr *VolatilityRequest) Validate() error {
	if vr.Symbol == "" {
		return errors.New("symbol is required")
	}

	// Validate period
	if vr.Period != "" {
		validPeriods := []string{"24h", "7d", "30d", "90d", "1y"}
		if !contains(validPeriods, vr.Period) {
			return errors.New("invalid period, must be one of: " + strings.Join(validPeriods, ", "))
		}
	}

	// Validate interval
	if vr.Interval != "" {
		validIntervals := []string{"5m", "15m", "1h", "4h", "1d"}
		if !contains(validIntervals, vr.Interval) {
			return errors.New("invalid interval, must be one of: " + strings.Join(validIntervals, ", "))
		}
	}

	// Validate method
	if vr.Method != "" {
		validMethods := []string{"close-to-close", "high-low", "parkinson", "garman-klass"}
		if !contains(validMethods, vr.Method) {
			return errors.New("invalid method, must be one of: " + strings.Join(validMethods, ", "))
		}
	}

	return nil
}

// SetDefaults sets default values for the volatility request
func (vr *VolatilityRequest) SetDefaults() {
	if vr.Period == "" {
		vr.Period = "24h"
	}

	if vr.Interval == "" {
		vr.Interval = "1h"
	}

	if vr.Method == "" {
		vr.Method = "close-to-close"
	}
}

// Validate validates the price request
func (pr *PriceRequest) Validate() error {
	if pr.Symbol == "" {
		return errors.New("symbol is required")
	}

	// Validate source if specified
	if pr.Source != "" {
		validSources := []string{"aggregated", "coingecko", "binance", "coinbase"}
		if !contains(validSources, pr.Source) {
			return errors.New("invalid source, must be one of: " + strings.Join(validSources, ", "))
		}
	}

	// Validate currency if specified
	if pr.Currency != "" {
		validCurrencies := []string{"USD", "EUR", "GBP", "JPY", "BTC", "ETH"}
		if !contains(validCurrencies, strings.ToUpper(pr.Currency)) {
			return errors.New("unsupported currency")
		}
	}

	return nil
}

// SetDefaults sets default values for the price request
func (pr *PriceRequest) SetDefaults() {
	if pr.Source == "" {
		pr.Source = "aggregated"
	}

	if pr.Currency == "" {
		pr.Currency = "USD"
	}

	pr.Symbol = strings.ToUpper(strings.TrimSpace(pr.Symbol))
}

// Validate validates the order book request
func (obr *OrderBookRequest) Validate() error {
	if obr.Symbol == "" {
		return errors.New("symbol is required")
	}

	if obr.Depth < 0 || obr.Depth > 100 {
		return errors.New("depth must be between 0 and 100")
	}

	return nil
}

// SetDefaults sets default values for the order book request
func (obr *OrderBookRequest) SetDefaults() {
	if obr.Depth == 0 {
		obr.Depth = 20
	}

	obr.Symbol = strings.ToUpper(strings.TrimSpace(obr.Symbol))
}

// Validate validates the stats request
func (sr *StatsRequest) Validate() error {
	if sr.Symbol == "" {
		return errors.New("symbol is required")
	}

	// Validate period if specified
	if sr.Period != "" {
		validPeriods := []string{"24h", "7d", "30d", "90d", "1y"}
		if !contains(validPeriods, sr.Period) {
			return errors.New("invalid period, must be one of: " + strings.Join(validPeriods, ", "))
		}
	}

	return nil
}

// SetDefaults sets default values for the stats request
func (sr *StatsRequest) SetDefaults() {
	if sr.Period == "" {
		sr.Period = "24h"
	}

	sr.Symbol = strings.ToUpper(strings.TrimSpace(sr.Symbol))
}

// BuildCacheKey builds a cache key for the request
func (hr *HistoryRequest) BuildCacheKey() string {
	return "history:" + hr.Symbol + ":" + hr.Interval + ":" +
		   strconv.FormatInt(hr.From, 10) + ":" +
		   strconv.FormatInt(hr.To, 10) + ":" +
		   strconv.Itoa(hr.Limit)
}

// BuildCacheKey builds a cache key for the batch prices request
func (bpr *BatchPricesRequest) BuildCacheKey() string {
	symbolsStr := strings.Join(bpr.Symbols, ",")
	flags := strconv.FormatBool(bpr.Include24hChange) + ":" +
			 strconv.FormatBool(bpr.IncludeVolume) + ":" +
			 strconv.FormatBool(bpr.IncludeMarketCap)
	return "batch:" + symbolsStr + ":" + bpr.Currency + ":" + flags
}

// BuildCacheKey builds a cache key for the volatility request
func (vr *VolatilityRequest) BuildCacheKey() string {
	return "volatility:" + vr.Symbol + ":" + vr.Period + ":" + vr.Interval + ":" + vr.Method
}

// BuildCacheKey builds a cache key for the price request
func (pr *PriceRequest) BuildCacheKey() string {
	metadata := strconv.FormatBool(pr.IncludeMetadata)
	return "price:" + pr.Symbol + ":" + pr.Source + ":" + pr.Currency + ":" + metadata
}

// BuildCacheKey builds a cache key for the order book request
func (obr *OrderBookRequest) BuildCacheKey() string {
	return "orderbook:" + obr.Symbol + ":" + strconv.Itoa(obr.Depth)
}

// BuildCacheKey builds a cache key for the stats request
func (sr *StatsRequest) BuildCacheKey() string {
	metrics := strconv.FormatBool(sr.IncludeMetrics)
	return "stats:" + sr.Symbol + ":" + sr.Period + ":" + metrics
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getMaxDurationForInterval(interval string) time.Duration {
	switch interval {
	case "1m":
		return 24 * time.Hour
	case "5m":
		return 5 * 24 * time.Hour
	case "15m":
		return 15 * 24 * time.Hour
	case "30m":
		return 30 * 24 * time.Hour
	case "1h":
		return 30 * 24 * time.Hour
	case "4h":
		return 120 * 24 * time.Hour
	case "1d":
		return 365 * 24 * time.Hour
	case "1w":
		return 2 * 365 * 24 * time.Hour
	case "1M":
		return 5 * 365 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}