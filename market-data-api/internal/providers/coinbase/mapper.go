package coinbase

import (
	"strconv"
	"strings"
	"time"
)

// TickerResponse represents the response from Coinbase Pro ticker endpoint
type TickerResponse struct {
	TradeId int64  `json:"trade_id"`
	Price   string `json:"price"`
	Size    string `json:"size"`
	Time    string `json:"time"`
	Bid     string `json:"bid"`
	Ask     string `json:"ask"`
	Volume  string `json:"volume"`
}

// StatsResponse represents the response from Coinbase Pro stats endpoint
type StatsResponse struct {
	Open        string `json:"open"`
	High        string `json:"high"`
	Low         string `json:"low"`
	Volume      string `json:"volume"`
	Last        string `json:"last"`
	Volume30Day string `json:"volume_30day"`
}

// OrderBookResponse represents the response from Coinbase Pro book endpoint
type OrderBookResponse struct {
	Sequence int64      `json:"sequence"`
	Bids     [][]string `json:"bids"`
	Asks     [][]string `json:"asks"`
}

// ProductResponse represents a product from Coinbase Pro
type ProductResponse struct {
	ID              string  `json:"id"`
	DisplayName     string  `json:"display_name"`
	BaseCurrency    string  `json:"base_currency"`
	QuoteCurrency   string  `json:"quote_currency"`
	BaseIncrement   string  `json:"base_increment"`
	QuoteIncrement  string  `json:"quote_increment"`
	BaseMinSize     string  `json:"base_min_size"`
	BaseMaxSize     string  `json:"base_max_size"`
	MinMarketFunds  string  `json:"min_market_funds"`
	MaxMarketFunds  string  `json:"max_market_funds"`
	Status          string  `json:"status"`
	StatusMessage   string  `json:"status_message"`
	CancelOnly      bool    `json:"cancel_only"`
	LimitOnly       bool    `json:"limit_only"`
	PostOnly        bool    `json:"post_only"`
	TradingDisabled bool    `json:"trading_disabled"`
	FxStablecoin    bool    `json:"fx_stablecoin"`
}

// TradeResponse represents a trade from Coinbase Pro
type TradeResponse struct {
	TradeId int64  `json:"trade_id"`
	Price   string `json:"price"`
	Size    string `json:"size"`
	Time    string `json:"time"`
	Side    string `json:"side"`
}

// TimeResponse represents the response from Coinbase Pro time endpoint
type TimeResponse struct {
	ISO   string  `json:"iso"`
	Epoch float64 `json:"epoch"`
}

// ErrorResponse represents error responses from Coinbase Pro
type ErrorResponse struct {
	Message string `json:"message"`
}

// WebSocket message types
type WebSocketMessage struct {
	Type      string      `json:"type"`
	ProductId string      `json:"product_id,omitempty"`
	Sequence  int64       `json:"sequence,omitempty"`
	Time      string      `json:"time,omitempty"`
	Price     string      `json:"price,omitempty"`
	Size      string      `json:"size,omitempty"`
	Side      string      `json:"side,omitempty"`
	TradeId   int64       `json:"trade_id,omitempty"`
	MakerOrderId string   `json:"maker_order_id,omitempty"`
	TakerOrderId string   `json:"taker_order_id,omitempty"`
	Changes   [][]string  `json:"changes,omitempty"`
	Bids      [][]string  `json:"bids,omitempty"`
	Asks      [][]string  `json:"asks,omitempty"`
}

// WebSocketSubscriptionMessage represents subscription message
type WebSocketSubscriptionMessage struct {
	Type       string             `json:"type"`
	ProductIds []string           `json:"product_ids"`
	Channels   []ChannelSubscription `json:"channels"`
}

// ChannelSubscription represents a channel subscription
type ChannelSubscription struct {
	Name       string   `json:"name"`
	ProductIds []string `json:"product_ids"`
}

// CandleData represents historical candle data
type CandleData struct {
	Timestamp time.Time `json:"timestamp"`
	Low       float64   `json:"low"`
	High      float64   `json:"high"`
	Open      float64   `json:"open"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

// NormalizeSymbol converts standard symbol format to Coinbase Pro format
// e.g., BTC -> BTC-USD, ETH -> ETH-USD
func NormalizeSymbol(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// If already in Coinbase format, return as is
	if strings.Contains(symbol, "-") {
		return symbol
	}

	// Common symbol mappings
	symbolMappings := map[string]string{
		"BTC":  "BTC-USD",
		"ETH":  "ETH-USD",
		"LTC":  "LTC-USD",
		"BCH":  "BCH-USD",
		"ETC":  "ETC-USD",
		"ZRX":  "ZRX-USD",
		"BAT":  "BAT-USD",
		"MANA": "MANA-USD",
		"DNT":  "DNT-USD",
		"LOOM": "LOOM-USD",
		"CVC":  "CVC-USD",
		"MKR":  "MKR-USD",
		"ZEC":  "ZEC-USD",
		"XLM":  "XLM-USD",
		"ADA":  "ADA-USD",
		"XTZ":  "XTZ-USD",
		"ATOM": "ATOM-USD",
		"DASH": "DASH-USD",
		"EOS":  "EOS-USD",
		"LINK": "LINK-USD",
		"OXT":  "OXT-USD",
		"COMP": "COMP-USD",
		"BAND": "BAND-USD",
		"NMR":  "NMR-USD",
		"CGT":  "CGT-USD",
		"UMA":  "UMA-USD",
		"LRC":  "LRC-USD",
		"YFI":  "YFI-USD",
		"UNI":  "UNI-USD",
		"REN":  "REN-USD",
		"BAL":  "BAL-USD",
		"FIL":  "FIL-USD",
		"GRT":  "GRT-USD",
		"AAVE": "AAVE-USD",
		"BNT":  "BNT-USD",
		"SNX":  "SNX-USD",
		"STORJ": "STORJ-USD",
		"SYN":  "SYN-USD",
		"BADGER": "BADGER-USD",
		"CTSI": "CTSI-USD",
		"RLC":  "RLC-USD",
		"WBTC": "WBTC-USD",
		"LPT":  "LPT-USD",
		"NU":   "NU-USD",
		"API3": "API3-USD",
		"ANKR": "ANKR-USD",
		"CRV":  "CRV-USD",
		"QUICK": "QUICK-USD",
		"POLY": "POLY-USD",
		"LTC":  "LTC-USD",
		"MATIC": "MATIC-USD",
		"SUSHI": "SUSHI-USD",
		"ALGO": "ALGO-USD",
		"FORTH": "FORTH-USD",
		"SKL":  "SKL-USD",
		"MASK": "MASK-USD",
		"NKN":  "NKN-USD",
		"OGN":  "OGN-USD",
		"1INCH": "1INCH-USD",
		"IOTX": "IOTX-USD",
		"FETCH": "FETCH-USD",
		"AMP":  "AMP-USD",
		"SHIB": "SHIB-USD",
		"DOT":  "DOT-USD",
		"DOGE": "DOGE-USD",
	}

	if normalized, exists := symbolMappings[symbol]; exists {
		return normalized
	}

	// Default to USD pair
	return symbol + "-USD"
}

// DenormalizeSymbol converts Coinbase Pro format back to standard format
// e.g., BTC-USD -> BTC
func DenormalizeSymbol(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// Split by hyphen and return base currency
	parts := strings.Split(symbol, "-")
	if len(parts) > 1 {
		return parts[0]
	}

	return symbol
}

// ValidateSymbol checks if a symbol is valid for Coinbase Pro
func ValidateSymbol(symbol string) bool {
	if len(symbol) < 3 || len(symbol) > 20 {
		return false
	}

	// Coinbase Pro symbols are alphanumeric with hyphens
	for _, char := range symbol {
		if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') ||
			 (char >= '0' && char <= '9') || char == '-') {
			return false
		}
	}

	return true
}

// NormalizeInterval converts common interval formats to Coinbase Pro granularity
func NormalizeInterval(interval string) string {
	switch strings.ToLower(interval) {
	case "1m", "1min", "1minute":
		return "60"
	case "5m", "5min", "5minutes":
		return "300"
	case "15m", "15min", "15minutes":
		return "900"
	case "1h", "1hour", "hourly":
		return "3600"
	case "6h", "6hour", "6hours":
		return "21600"
	case "1d", "1day", "daily":
		return "86400"
	default:
		return "3600" // Default to 1 hour
	}
}

// IntervalToSeconds converts interval string to seconds
func IntervalToSeconds(interval string) int {
	seconds, err := strconv.Atoi(interval)
	if err != nil {
		return 3600 // Default to 1 hour
	}
	return seconds
}

// GetSupportedIntervals returns list of supported intervals in seconds
func GetSupportedIntervals() []string {
	return []string{
		"60",    // 1 minute
		"300",   // 5 minutes
		"900",   // 15 minutes
		"3600",  // 1 hour
		"21600", // 6 hours
		"86400", // 1 day
	}
}

// IsValidInterval checks if an interval is supported
func IsValidInterval(interval string) bool {
	supported := GetSupportedIntervals()
	normalized := NormalizeInterval(interval)

	for _, supportedInterval := range supported {
		if supportedInterval == normalized {
			return true
		}
	}

	return false
}

// GetSupportedChannels returns list of supported WebSocket channels
func GetSupportedChannels() []string {
	return []string{
		"heartbeat",
		"ticker",
		"level2",
		"user",
		"matches",
		"full",
	}
}

// IsValidChannel checks if a WebSocket channel is supported
func IsValidChannel(channel string) bool {
	supported := GetSupportedChannels()

	for _, supportedChannel := range supported {
		if supportedChannel == channel {
			return true
		}
	}

	return false
}

// ParseTime parses Coinbase Pro time format
func ParseTime(timeStr string) (time.Time, error) {
	// Coinbase Pro uses ISO 8601 format
	return time.Parse(time.RFC3339, timeStr)
}

// FormatTime formats time to Coinbase Pro format
func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// GetProductID gets the full product ID for a symbol
func GetProductID(symbol string) string {
	return NormalizeSymbol(symbol)
}

// ExtractBaseAsset extracts the base asset from a product ID
func ExtractBaseAsset(productID string) string {
	return DenormalizeSymbol(productID)
}

// ExtractQuoteAsset extracts the quote asset from a product ID
func ExtractQuoteAsset(productID string) string {
	parts := strings.Split(strings.ToUpper(productID), "-")
	if len(parts) > 1 {
		return parts[1]
	}
	return "USD"
}

// Helper functions for data conversion

// ParseFloat64 safely parses a float64 from string
func ParseFloat64(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

// ParseInt64 safely parses an int64 from string
func ParseInt64(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

// FormatFloat64 formats a float64 to string
func FormatFloat64(f float64, precision int) string {
	return strconv.FormatFloat(f, 'f', precision, 64)
}

// ValidateProductID validates a Coinbase Pro product ID
func ValidateProductID(productID string) bool {
	parts := strings.Split(strings.ToUpper(productID), "-")
	if len(parts) != 2 {
		return false
	}

	baseAsset := parts[0]
	quoteAsset := parts[1]

	// Check if base and quote assets are valid
	if len(baseAsset) < 1 || len(baseAsset) > 10 {
		return false
	}

	if len(quoteAsset) < 1 || len(quoteAsset) > 10 {
		return false
	}

	// Common quote assets
	commonQuotes := []string{"USD", "EUR", "GBP", "USDC", "BTC", "ETH"}
	validQuote := false
	for _, quote := range commonQuotes {
		if quoteAsset == quote {
			validQuote = true
			break
		}
	}

	return validQuote
}

// GetWebSocketURL returns the appropriate WebSocket URL
func GetWebSocketURL(sandbox bool) string {
	if sandbox {
		return "wss://ws-feed-public.sandbox.exchange.coinbase.com"
	}
	return "wss://ws-feed.exchange.coinbase.com"
}

// GetRestURL returns the appropriate REST API URL
func GetRestURL(sandbox bool) string {
	if sandbox {
		return "https://api-public.sandbox.exchange.coinbase.com"
	}
	return "https://api.exchange.coinbase.com"
}

// CreateSubscriptionMessage creates a WebSocket subscription message
func CreateSubscriptionMessage(productIds []string, channels []string) *WebSocketSubscriptionMessage {
	channelSubs := make([]ChannelSubscription, len(channels))
	for i, channel := range channels {
		channelSubs[i] = ChannelSubscription{
			Name:       channel,
			ProductIds: productIds,
		}
	}

	return &WebSocketSubscriptionMessage{
		Type:       "subscribe",
		ProductIds: productIds,
		Channels:   channelSubs,
	}
}

// CreateUnsubscriptionMessage creates a WebSocket unsubscription message
func CreateUnsubscriptionMessage(productIds []string, channels []string) *WebSocketSubscriptionMessage {
	channelSubs := make([]ChannelSubscription, len(channels))
	for i, channel := range channels {
		channelSubs[i] = ChannelSubscription{
			Name:       channel,
			ProductIds: productIds,
		}
	}

	return &WebSocketSubscriptionMessage{
		Type:       "unsubscribe",
		ProductIds: productIds,
		Channels:   channelSubs,
	}
}

// GetGranularityFromInterval converts time duration to Coinbase Pro granularity
func GetGranularityFromInterval(interval string) int {
	seconds, _ := strconv.Atoi(NormalizeInterval(interval))
	return seconds
}

// Constants for Coinbase Pro API limits
const (
	MaxRequestsPerSecond = 10
	MaxProductsPerRequest = 100
	MaxCandlesPerRequest = 300
	DefaultTimeout = 30 * time.Second
)

// Rate limiting constants
const (
	PublicEndpointRateLimit = 3  // requests per second
	PrivateEndpointRateLimit = 5 // requests per second
)

// Common error messages
const (
	ErrInvalidSymbol    = "invalid symbol format"
	ErrInvalidInterval  = "invalid interval"
	ErrInvalidChannel   = "invalid WebSocket channel"
	ErrRateLimitExceeded = "rate limit exceeded"
	ErrInvalidCredentials = "invalid API credentials"
	ErrProductNotFound   = "product not found"
	ErrInvalidRequest    = "invalid request parameters"
)