package binance

import (
	"strconv"
	"strings"
	"time"
)

// PriceResponse represents the response from Binance price endpoint
type PriceResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

// TickerResponse represents the response from Binance 24hr ticker endpoint
type TickerResponse struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	WeightedAvgPrice   string `json:"weightedAvgPrice"`
	PrevClosePrice     string `json:"prevClosePrice"`
	LastPrice          string `json:"lastPrice"`
	LastQty            string `json:"lastQty"`
	BidPrice           string `json:"bidPrice"`
	BidQty             string `json:"bidQty"`
	AskPrice           string `json:"askPrice"`
	AskQty             string `json:"askQty"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
	FirstId            int64  `json:"firstId"`
	LastId             int64  `json:"lastId"`
	Count              int64  `json:"count"`
}

// KlineResponse represents a single kline from Binance
type KlineResponse []interface{}

// OrderBookResponse represents the response from Binance depth endpoint
type OrderBookResponse struct {
	LastUpdateId int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

// ExchangeInfoResponse represents the exchange info response
type ExchangeInfoResponse struct {
	Timezone   string               `json:"timezone"`
	ServerTime int64                `json:"serverTime"`
	Symbols    []ExchangeInfoSymbol `json:"symbols"`
}

// ExchangeInfoSymbol represents symbol info from exchange info
type ExchangeInfoSymbol struct {
	Symbol              string                    `json:"symbol"`
	Status              string                    `json:"status"`
	BaseAsset           string                    `json:"baseAsset"`
	BaseAssetPrecision  int                       `json:"baseAssetPrecision"`
	QuoteAsset          string                    `json:"quoteAsset"`
	QuoteAssetPrecision int                       `json:"quoteAssetPrecision"`
	OrderTypes          []string                  `json:"orderTypes"`
	IcebergAllowed      bool                      `json:"icebergAllowed"`
	OcoAllowed          bool                      `json:"ocoAllowed"`
	IsSpotTradingAllowed bool                     `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed bool                   `json:"isMarginTradingAllowed"`
	Filters             []map[string]interface{}  `json:"filters"`
	Permissions         []string                  `json:"permissions"`
}

// WebSocketTickerResponse represents ticker data from WebSocket
type WebSocketTickerResponse struct {
	EventType       string `json:"e"`  // Event type
	EventTime       int64  `json:"E"`  // Event time
	Symbol          string `json:"s"`  // Symbol
	PriceChange     string `json:"p"`  // Price change
	PriceChangePct  string `json:"P"`  // Price change percent
	WeightedAvgPrice string `json:"w"` // Weighted average price
	FirstTradePrice string `json:"x"`  // First trade(F)-1 price (first trade before the 24hr rolling window)
	LastPrice       string `json:"c"`  // Last price
	LastQty         string `json:"Q"`  // Last quantity
	BestBidPrice    string `json:"b"`  // Best bid price
	BestBidQty      string `json:"B"`  // Best bid quantity
	BestAskPrice    string `json:"a"`  // Best ask price
	BestAskQty      string `json:"A"`  // Best ask quantity
	OpenPrice       string `json:"o"`  // Open price
	HighPrice       string `json:"h"`  // High price
	LowPrice        string `json:"l"`  // Low price
	Volume          string `json:"v"`  // Total traded base asset volume
	QuoteVolume     string `json:"q"`  // Total traded quote asset volume
	StatOpenTime    int64  `json:"O"`  // Statistics open time
	StatCloseTime   int64  `json:"C"`  // Statistics close time
	FirstTradeId    int64  `json:"F"`  // First trade ID
	LastTradeId     int64  `json:"L"`  // Last trade Id
	TradeCount      int64  `json:"n"`  // Total number of trades
}

// WebSocketDepthResponse represents depth data from WebSocket
type WebSocketDepthResponse struct {
	EventType        string     `json:"e"` // Event type
	EventTime        int64      `json:"E"` // Event time
	Symbol           string     `json:"s"` // Symbol
	FirstUpdateId    int64      `json:"U"` // First update ID in event
	FinalUpdateId    int64      `json:"u"` // Final update ID in event
	Bids             [][]string `json:"b"` // Bids to be updated
	Asks             [][]string `json:"a"` // Asks to be updated
}

// WebSocketTradeResponse represents trade data from WebSocket
type WebSocketTradeResponse struct {
	EventType         string `json:"e"` // Event type
	EventTime         int64  `json:"E"` // Event time
	Symbol            string `json:"s"` // Symbol
	TradeId           int64  `json:"t"` // Trade ID
	Price             string `json:"p"` // Price
	Quantity          string `json:"q"` // Quantity
	BuyerOrderId      int64  `json:"b"` // Buyer order ID
	SellerOrderId     int64  `json:"a"` // Seller order ID
	TradeTime         int64  `json:"T"` // Trade time
	IsBuyerMaker      bool   `json:"m"` // Is the buyer the market maker?
	Ignore            bool   `json:"M"` // Ignore
}

// WebSocketMessage represents a generic WebSocket message
type WebSocketMessage struct {
	Stream string      `json:"stream"`
	Data   interface{} `json:"data"`
}

// ErrorResponse represents error responses from Binance
type ErrorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// Helper methods for data conversion

// ParseKline converts Binance kline response to structured data
func (k KlineResponse) ParseKline() (*KlineData, error) {
	if len(k) < 12 {
		return nil, ErrInvalidKlineResponse
	}

	openTime, err := parseTimestamp(k[0])
	if err != nil {
		return nil, err
	}

	closeTime, err := parseTimestamp(k[6])
	if err != nil {
		return nil, err
	}

	open, err := parseFloat64(k[1])
	if err != nil {
		return nil, err
	}

	high, err := parseFloat64(k[2])
	if err != nil {
		return nil, err
	}

	low, err := parseFloat64(k[3])
	if err != nil {
		return nil, err
	}

	closePrice, err := parseFloat64(k[4])
	if err != nil {
		return nil, err
	}

	volume, err := parseFloat64(k[5])
	if err != nil {
		return nil, err
	}

	quoteVolume, err := parseFloat64(k[7])
	if err != nil {
		return nil, err
	}

	trades, err := parseInt64(k[8])
	if err != nil {
		return nil, err
	}

	return &KlineData{
		OpenTime:                 time.Unix(openTime/1000, 0),
		Open:                     open,
		High:                     high,
		Low:                      low,
		Close:                    closePrice,
		Volume:                   volume,
		CloseTime:                time.Unix(closeTime/1000, 0),
		QuoteAssetVolume:         quoteVolume,
		NumberOfTrades:           trades,
		TakerBuyBaseAssetVolume:  0, // k[9]
		TakerBuyQuoteAssetVolume: 0, // k[10]
	}, nil
}

// KlineData represents parsed kline data
type KlineData struct {
	OpenTime                 time.Time `json:"open_time"`
	Open                     float64   `json:"open"`
	High                     float64   `json:"high"`
	Low                      float64   `json:"low"`
	Close                    float64   `json:"close"`
	Volume                   float64   `json:"volume"`
	CloseTime                time.Time `json:"close_time"`
	QuoteAssetVolume         float64   `json:"quote_asset_volume"`
	NumberOfTrades           int64     `json:"number_of_trades"`
	TakerBuyBaseAssetVolume  float64   `json:"taker_buy_base_asset_volume"`
	TakerBuyQuoteAssetVolume float64   `json:"taker_buy_quote_asset_volume"`
}

// NormalizeSymbol converts standard symbol format to Binance format
// e.g., BTC -> BTCUSDT, ETH -> ETHUSDT
func NormalizeSymbol(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// If already in USDT format, return as is
	if strings.HasSuffix(symbol, "USDT") {
		return symbol
	}

	// Common symbol mappings
	symbolMappings := map[string]string{
		"BTC":  "BTCUSDT",
		"ETH":  "ETHUSDT",
		"BNB":  "BNBUSDT",
		"ADA":  "ADAUSDT",
		"XRP":  "XRPUSDT",
		"DOT":  "DOTUSDT",
		"LINK": "LINKUSDT",
		"LTC":  "LTCUSDT",
		"BCH":  "BCHUSDT",
		"XLM":  "XLMUSDT",
		"DOGE": "DOGEUSDT",
		"UNI":  "UNIUSDT",
		"SOL":  "SOLUSDT",
		"MATIC": "MATICUSDT",
		"AVAX": "AVAXUSDT",
		"ATOM": "ATOMUSDT",
	}

	if normalized, exists := symbolMappings[symbol]; exists {
		return normalized
	}

	// Default to USDT pair
	return symbol + "USDT"
}

// DenormalizeSymbol converts Binance format back to standard format
// e.g., BTCUSDT -> BTC
func DenormalizeSymbol(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// Remove USDT suffix if present
	if strings.HasSuffix(symbol, "USDT") {
		return symbol[:len(symbol)-4]
	}

	// Handle other quote currencies
	quoteCurrencies := []string{"BTC", "ETH", "BNB", "BUSD"}
	for _, quote := range quoteCurrencies {
		if strings.HasSuffix(symbol, quote) {
			return symbol[:len(symbol)-len(quote)]
		}
	}

	return symbol
}

// ValidateSymbol checks if a symbol is valid for Binance
func ValidateSymbol(symbol string) bool {
	if len(symbol) < 3 || len(symbol) > 20 {
		return false
	}

	// Check if symbol contains only alphanumeric characters
	for _, char := range symbol {
		if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
			return false
		}
	}

	return true
}

// NormalizeInterval converts common interval formats to Binance format
func NormalizeInterval(interval string) string {
	switch strings.ToLower(interval) {
	case "1m", "1min", "1minute":
		return "1m"
	case "3m", "3min", "3minutes":
		return "3m"
	case "5m", "5min", "5minutes":
		return "5m"
	case "15m", "15min", "15minutes":
		return "15m"
	case "30m", "30min", "30minutes":
		return "30m"
	case "1h", "1hour", "hourly":
		return "1h"
	case "2h", "2hour", "2hours":
		return "2h"
	case "4h", "4hour", "4hours":
		return "4h"
	case "6h", "6hour", "6hours":
		return "6h"
	case "8h", "8hour", "8hours":
		return "8h"
	case "12h", "12hour", "12hours":
		return "12h"
	case "1d", "1day", "daily":
		return "1d"
	case "3d", "3day", "3days":
		return "3d"
	case "1w", "1week", "weekly":
		return "1w"
	case "1M", "1month", "monthly":
		return "1M"
	default:
		return "1h"
	}
}

// GetSupportedIntervals returns list of supported intervals
func GetSupportedIntervals() []string {
	return []string{
		"1m", "3m", "5m", "15m", "30m",
		"1h", "2h", "4h", "6h", "8h", "12h",
		"1d", "3d", "1w", "1M",
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

// Helper functions for parsing

func parseFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case string:
		return strconv.ParseFloat(v, 64)
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	default:
		return 0, ErrInvalidDataType
	}
}

func parseInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case string:
		return strconv.ParseInt(v, 10, 64)
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	default:
		return 0, ErrInvalidDataType
	}
}

func parseTimestamp(value interface{}) (int64, error) {
	switch v := value.(type) {
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, ErrInvalidDataType
	}
}

// Custom errors
var (
	ErrInvalidKlineResponse = NewBinanceError(-1, "invalid kline response format")
	ErrInvalidDataType     = NewBinanceError(-2, "invalid data type")
)

// NewBinanceError creates a new Binance-specific error
func NewBinanceError(code int, message string) error {
	return &BinanceError{
		Code:    code,
		Message: message,
	}
}

// BinanceError represents a Binance-specific error
type BinanceError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *BinanceError) Error() string {
	return "Binance API Error " + strconv.Itoa(e.Code) + ": " + e.Message
}

// IsRateLimitError checks if the error is a rate limit error
func (e *BinanceError) IsRateLimitError() bool {
	return e.Code == -1003 || e.Code == -1015
}

// IsRetryableError checks if the error is retryable
func (e *BinanceError) IsRetryableError() bool {
	retryableCodes := []int{
		-1003, // Way too many requests
		-1015, // Too many new orders
		-1021, // Timestamp outside recv window
		-2010, // New order rejected
		-2013, // Order does not exist
		-2014, // API-key format invalid
	}

	for _, code := range retryableCodes {
		if e.Code == code {
			return true
		}
	}

	return false
}