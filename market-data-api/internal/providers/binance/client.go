package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/time/rate"

	"market-data-api/internal/models"
	"market-data-api/internal/types"
)

// Client represents a Binance API client
type Client struct {
	*types.ProviderClient
	apiKey      string
	secretKey   string
	baseURL     string
	httpClient  *http.Client
	rateLimiter *rate.Limiter
}

// Config represents Binance client configuration
type Config struct {
	APIKey        string
	SecretKey     string
	BaseURL       string
	Timeout       time.Duration
	RateLimit     int
	Weight        float64
	RetryAttempts int
	RetryDelay    time.Duration
}

// NewClient creates a new Binance client
func NewClient(config *Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.binance.com"
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	if config.RateLimit == 0 {
		config.RateLimit = 1200 // requests per minute
	}

	// Create rate limiter
	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(config.RateLimit)), 10)

	client := &Client{
		apiKey:      config.APIKey,
		secretKey:   config.SecretKey,
		baseURL:     config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		rateLimiter: limiter,
		ProviderClient: &types.ProviderClient{
			Name:    "binance",
			Weight:  config.Weight,
			BaseURL: config.BaseURL,
			Timeout: config.Timeout,
			Status: &models.ProviderStatus{
				Name:   "binance",
				Status: "healthy",
				Weight: config.Weight,
			},
			Metrics: &types.ProviderMetrics{
				Name: "binance",
			},
		},
	}

	return client
}

// GetPrice fetches the current price for a single cryptocurrency
func (c *Client) GetPrice(ctx context.Context, symbol string) (*models.Price, error) {
	if err := c.CheckRateLimit(); err != nil {
		return nil, err
	}

	start := time.Now()
	defer func() {
		latency := time.Since(start)
		c.UpdateMetrics(true, latency)
	}()

	// Convert symbol to Binance format (e.g., BTC -> BTCUSDT)
	binanceSymbol := c.formatSymbol(symbol)

	endpoint := "/api/v3/ticker/24hr"
	params := url.Values{}
	params.Set("symbol", binanceSymbol)

	data, err := c.makeRequest(ctx, "GET", endpoint, params, false)
	if err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, err
	}

	// Parse response
	var response TickerResponse
	if err := json.Unmarshal(data, &response); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError("binance", "PARSE_ERROR", "Failed to parse response", false)
	}

	// Convert to Price model
	price, err := decimal.NewFromString(response.LastPrice)
	if err != nil {
		return nil, types.NewProviderError("binance", "PARSE_ERROR", "Invalid price format", false)
	}

	volume24h, _ := decimal.NewFromString(response.Volume)
	priceChange24h, _ := decimal.NewFromString(response.PriceChange)
	priceChangePercent, _ := decimal.NewFromString(response.PriceChangePercent)

	priceModel := &models.Price{
		Symbol:        symbol,
		Price:         price,
		PriceUSD:      price,
		Timestamp:     time.Unix(response.CloseTime/1000, 0),
		Source:        "binance",
		Provider:      "binance",
		Volume24h:     volume24h,
		Change24h:     priceChange24h,
		ChangePercent: priceChangePercent,
		Confidence:    0.98, // Binance is very reliable
		Latency:       time.Since(start).Milliseconds(),
	}

	return priceModel, nil
}

// GetPrices fetches prices for multiple cryptocurrencies
func (c *Client) GetPrices(ctx context.Context, symbols []string) (map[string]*models.Price, error) {
	if err := c.CheckRateLimit(); err != nil {
		return nil, err
	}

	start := time.Now()
	defer func() {
		latency := time.Since(start)
		c.UpdateMetrics(true, latency)
	}()

	// Get all ticker data
	endpoint := "/api/v3/ticker/24hr"
	data, err := c.makeRequest(ctx, "GET", endpoint, nil, false)
	if err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, err
	}

	// Parse response
	var tickers []TickerResponse
	if err := json.Unmarshal(data, &tickers); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError("binance", "PARSE_ERROR", "Failed to parse response", false)
	}

	// Create symbol map for filtering
	symbolMap := make(map[string]bool)
	for _, symbol := range symbols {
		symbolMap[strings.ToUpper(symbol)] = true
	}

	// Convert to prices map
	prices := make(map[string]*models.Price)
	latency := time.Since(start).Milliseconds()

	for _, ticker := range tickers {
		// Extract base symbol from Binance symbol (e.g., BTCUSDT -> BTC)
		symbol := c.extractSymbol(ticker.Symbol)
		if symbol == "" || !symbolMap[symbol] {
			continue
		}

		price, err := decimal.NewFromString(ticker.LastPrice)
		if err != nil {
			continue
		}

		volume24h, _ := decimal.NewFromString(ticker.Volume)
		priceChange24h, _ := decimal.NewFromString(ticker.PriceChange)
		priceChangePercent, _ := decimal.NewFromString(ticker.PriceChangePercent)

		prices[symbol] = &models.Price{
			Symbol:        symbol,
			Price:         price,
			PriceUSD:      price,
			Timestamp:     time.Unix(ticker.CloseTime/1000, 0),
			Source:        "binance",
			Provider:      "binance",
			Volume24h:     volume24h,
			Change24h:     priceChange24h,
			ChangePercent: priceChangePercent,
			Confidence:    0.98,
			Latency:       latency,
		}
	}

	return prices, nil
}

// GetHistoricalData fetches historical price data
func (c *Client) GetHistoricalData(ctx context.Context, symbol, interval string, from, to time.Time, limit int) ([]*models.Candle, error) {
	if err := c.CheckRateLimit(); err != nil {
		return nil, err
	}

	start := time.Now()
	defer func() {
		latency := time.Since(start)
		c.UpdateMetrics(true, latency)
	}()

	binanceSymbol := c.formatSymbol(symbol)
	binanceInterval := c.convertInterval(interval)

	endpoint := "/api/v3/klines"
	params := url.Values{}
	params.Set("symbol", binanceSymbol)
	params.Set("interval", binanceInterval)
	params.Set("startTime", strconv.FormatInt(from.Unix()*1000, 10))
	params.Set("endTime", strconv.FormatInt(to.Unix()*1000, 10))

	if limit > 0 {
		params.Set("limit", strconv.Itoa(min(limit, 1000))) // Binance max is 1000
	}

	data, err := c.makeRequest(ctx, "GET", endpoint, params, false)
	if err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, err
	}

	// Parse response
	var response [][]interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError("binance", "PARSE_ERROR", "Failed to parse klines data", false)
	}

	// Convert to candles
	candles := make([]*models.Candle, 0, len(response))

	for _, kline := range response {
		if len(kline) < 11 {
			continue
		}

		timestamp := time.Unix(int64(kline[0].(float64))/1000, 0)

		open, _ := decimal.NewFromString(kline[1].(string))
		high, _ := decimal.NewFromString(kline[2].(string))
		low, _ := decimal.NewFromString(kline[3].(string))
		close, _ := decimal.NewFromString(kline[4].(string))
		volume, _ := decimal.NewFromString(kline[5].(string))

		candle := &models.Candle{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			Trades:    int64(kline[8].(float64)),
		}

		// Calculate VWAP if volume > 0
		if volume.GreaterThan(decimal.Zero) {
			typical := high.Add(low).Add(close).Div(decimal.NewFromInt(3))
			candle.VWAP = typical
		}

		candles = append(candles, candle)
	}

	return candles, nil
}

// GetMarketData fetches comprehensive market data for a cryptocurrency
func (c *Client) GetMarketData(ctx context.Context, symbol string) (*models.MarketData, error) {
	if err := c.CheckRateLimit(); err != nil {
		return nil, err
	}

	start := time.Now()
	defer func() {
		latency := time.Since(start)
		c.UpdateMetrics(true, latency)
	}()

	binanceSymbol := c.formatSymbol(symbol)

	// Get 24hr ticker data
	endpoint := "/api/v3/ticker/24hr"
	params := url.Values{}
	params.Set("symbol", binanceSymbol)

	data, err := c.makeRequest(ctx, "GET", endpoint, params, false)
	if err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, err
	}

	var ticker TickerResponse
	if err := json.Unmarshal(data, &ticker); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError("binance", "PARSE_ERROR", "Failed to parse ticker data", false)
	}

	// Convert to MarketData
	currentPrice, _ := decimal.NewFromString(ticker.LastPrice)
	high24h, _ := decimal.NewFromString(ticker.HighPrice)
	low24h, _ := decimal.NewFromString(ticker.LowPrice)
	volume24h, _ := decimal.NewFromString(ticker.Volume)
	priceChange24h, _ := decimal.NewFromString(ticker.PriceChange)
	priceChangePercent24h, _ := decimal.NewFromString(ticker.PriceChangePercent)

	marketData := &models.MarketData{
		Symbol:                       symbol,
		CurrentPrice:                 currentPrice,
		TotalVolume:                  volume24h,
		High24h:                      high24h,
		Low24h:                       low24h,
		PriceChange24h:               priceChange24h,
		PriceChangePercentage24h:     priceChangePercent24h,
		LastUpdated:                  time.Unix(ticker.CloseTime/1000, 0),
		DataSource:                   "binance",
		Confidence:                   0.98,
	}

	return marketData, nil
}

// GetOrderBook fetches order book data
func (c *Client) GetOrderBook(ctx context.Context, symbol string, depth int) (*models.OrderBook, error) {
	if err := c.CheckRateLimit(); err != nil {
		return nil, err
	}

	start := time.Now()
	defer func() {
		latency := time.Since(start)
		c.UpdateMetrics(true, latency)
	}()

	binanceSymbol := c.formatSymbol(symbol)

	endpoint := "/api/v3/depth"
	params := url.Values{}
	params.Set("symbol", binanceSymbol)
	params.Set("limit", strconv.Itoa(min(depth, 5000)))

	data, err := c.makeRequest(ctx, "GET", endpoint, params, false)
	if err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, err
	}

	var response OrderBookResponse
	if err := json.Unmarshal(data, &response); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError("binance", "PARSE_ERROR", "Failed to parse order book", false)
	}

	// Convert to OrderBook
	orderBook := models.NewOrderBook(symbol)
	orderBook.LastUpdate = time.Unix(response.LastUpdateId/1000, 0)

	// Process bids
	for _, bid := range response.Bids {
		if len(bid) >= 2 {
			price, _ := decimal.NewFromString(bid[0])
			amount, _ := decimal.NewFromString(bid[1])
			orderBook.Bids = append(orderBook.Bids, &models.OrderLevel{
				Price:  price,
				Amount: amount,
				Total:  price.Mul(amount),
			})
		}
	}

	// Process asks
	for _, ask := range response.Asks {
		if len(ask) >= 2 {
			price, _ := decimal.NewFromString(ask[0])
			amount, _ := decimal.NewFromString(ask[1])
			orderBook.Asks = append(orderBook.Asks, &models.OrderLevel{
				Price:  price,
				Amount: amount,
				Total:  price.Mul(amount),
			})
		}
	}

	// Calculate spread
	orderBook.CalculateSpread()

	return orderBook, nil
}

// Ping checks if the Binance API is accessible
func (c *Client) Ping(ctx context.Context) error {
	start := time.Now()
	_, err := c.makeRequest(ctx, "GET", "/api/v3/ping", nil, false)
	latency := time.Since(start)

	if err != nil {
		c.UpdateStatus(types.StatusDown, latency, 1)
		return err
	}

	c.UpdateStatus(types.StatusHealthy, latency, 0)
	return nil
}

// makeRequest makes an HTTP request to the Binance API
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, params url.Values, signed bool) ([]byte, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, types.NewProviderError("binance", types.ErrorCodeRateLimit, "Rate limit wait cancelled", true)
	}

	// Build URL
	fullURL := c.baseURL + endpoint

	if signed {
		if params == nil {
			params = url.Values{}
		}
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixNano()/1000000, 10))

		// Create signature
		signature := c.sign(params.Encode())
		params.Set("signature", signature)
	}

	if params != nil && len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, types.NewProviderError("binance", "REQUEST_ERROR", "Failed to create request", false)
	}

	// Add headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "market-data-api/1.0")

	if c.apiKey != "" {
		req.Header.Set("X-MBX-APIKEY", c.apiKey)
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, types.NewProviderError("binance", types.ErrorCodeNetworkError, "Network error: "+err.Error(), true)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewProviderError("binance", "READ_ERROR", "Failed to read response", false)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	return body, nil
}

// sign creates HMAC SHA256 signature
func (c *Client) sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// handleErrorResponse handles error responses from the API
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	var errorMsg string
	var retryable bool

	// Try to parse Binance error response
	var binanceError ErrorResponse
	if json.Unmarshal(body, &binanceError) == nil && binanceError.Msg != "" {
		errorMsg = binanceError.Msg
	} else {
		errorMsg = fmt.Sprintf("HTTP %d", statusCode)
	}

	switch statusCode {
	case http.StatusTooManyRequests:
		retryable = true
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		retryable = true
	default:
		retryable = false
	}

	return types.NewProviderError("binance", strconv.Itoa(statusCode), errorMsg, retryable)
}

// formatSymbol converts a symbol to Binance format
func (c *Client) formatSymbol(symbol string) string {
	// Most cryptocurrencies are paired with USDT on Binance
	return strings.ToUpper(symbol) + "USDT"
}

// extractSymbol extracts the base symbol from Binance format
func (c *Client) extractSymbol(binanceSymbol string) string {
	// Remove common quote currencies
	quoteCurrencies := []string{"USDT", "BUSD", "BTC", "ETH", "BNB"}

	for _, quote := range quoteCurrencies {
		if strings.HasSuffix(binanceSymbol, quote) {
			return strings.TrimSuffix(binanceSymbol, quote)
		}
	}

	return ""
}

// convertInterval converts common interval format to Binance format
func (c *Client) convertInterval(interval string) string {
	switch interval {
	case "1m", "1min":
		return "1m"
	case "5m", "5min":
		return "5m"
	case "15m", "15min":
		return "15m"
	case "30m", "30min":
		return "30m"
	case "1h", "1hour":
		return "1h"
	case "4h", "4hour":
		return "4h"
	case "1d", "1day":
		return "1d"
	case "1w", "1week":
		return "1w"
	case "1M", "1month":
		return "1M"
	default:
		return "1h"
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}