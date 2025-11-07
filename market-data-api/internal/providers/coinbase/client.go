package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"market-data-api/internal/models"
	"market-data-api/internal/types"
)

const (
	BaseURL = "https://api.exchange.coinbase.com"
	Name    = "coinbase"
)

// Client represents the Coinbase Pro API client
type Client struct {
	*types.ProviderClient

	httpClient   *http.Client
	baseURL      string
	apiKey       string
	secret       string
	passphrase   string
	sandbox      bool

	// Rate limiting
	requestCount int64
	lastReset    time.Time
	mu           sync.RWMutex

	// WebSocket
	wsConnected bool
	wsURL       string
}

// Config represents configuration for Coinbase client
type Config struct {
	APIKey     string        `json:"api_key"`
	Secret     string        `json:"secret"`
	Passphrase string        `json:"passphrase"`
	Sandbox    bool          `json:"sandbox"`
	Timeout    time.Duration `json:"timeout"`
	RateLimit  int           `json:"rate_limit"`
	Weight     float64       `json:"weight"`
}

// NewClient creates a new Coinbase client
func NewClient(config *Config) *Client {
	baseURL := BaseURL
	wsURL := "wss://ws-feed.exchange.coinbase.com"

	if config.Sandbox {
		baseURL = "https://api-public.sandbox.exchange.coinbase.com"
		wsURL = "wss://ws-feed-public.sandbox.exchange.coinbase.com"
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	if config.RateLimit == 0 {
		config.RateLimit = 10 // Coinbase Pro allows 10 requests per second
	}

	if config.Weight == 0 {
		config.Weight = 1.0
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	client := &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		apiKey:     config.APIKey,
		secret:     config.Secret,
		passphrase: config.Passphrase,
		sandbox:    config.Sandbox,
		wsURL:      wsURL,
		lastReset:  time.Now(),
	}

	// Initialize base provider client
	client.ProviderClient = &types.ProviderClient{}

	return client
}

// GetName returns the provider name
func (c *Client) GetName() string {
	return Name
}

// GetWeight returns the provider weight
func (c *Client) GetWeight() float64 {
	return 1.0 // Default weight
}

// GetPrice retrieves current price for a symbol
func (c *Client) GetPrice(ctx context.Context, symbol string) (*models.Price, error) {
	start := time.Now()

	if err := c.CheckRateLimit(); err != nil {
		return nil, err
	}

	normalizedSymbol := NormalizeSymbol(symbol)
	endpoint := fmt.Sprintf("/products/%s/ticker", normalizedSymbol)

	var tickerResp TickerResponse
	if err := c.makeRequest(ctx, "GET", endpoint, nil, &tickerResp); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError(Name, types.ErrorCodeServerError,
			fmt.Sprintf("failed to get price for %s: %v", symbol, err), true)
	}

	price, err := decimal.NewFromString(tickerResp.Price)
	if err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError(Name, types.ErrorCodeBadRequest,
			fmt.Sprintf("invalid price format: %s", tickerResp.Price), false)
	}

	volume, _ := decimal.NewFromString(tickerResp.Volume)
	result := &models.Price{
		Symbol:    DenormalizeSymbol(normalizedSymbol),
		Price:     price,
		PriceUSD:  price,
		Volume24h: volume,
		Timestamp: time.Now(),
		Source:    Name,
	}

	c.UpdateMetrics(true, time.Since(start))
	return result, nil
}

// GetPrices retrieves current prices for multiple symbols
func (c *Client) GetPrices(ctx context.Context, symbols []string) (map[string]*models.Price, error) {
	if len(symbols) == 0 {
		return nil, types.NewProviderError(Name, types.ErrorCodeBadRequest, "symbols list is empty", false)
	}

	results := make(map[string]*models.Price)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Coinbase doesn't have a batch endpoint, so we'll fetch individually
	// Use a semaphore to limit concurrent requests
	semaphore := make(chan struct{}, 5)

	for _, symbol := range symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			price, err := c.GetPrice(ctx, sym)
			if err != nil {
				return // Skip failed requests
			}

			mu.Lock()
			results[sym] = price
			mu.Unlock()
		}(symbol)
	}

	wg.Wait()

	if len(results) == 0 {
		return nil, types.NewProviderError(Name, types.ErrorCodeNoData, "no prices retrieved", true)
	}

	return results, nil
}

// GetHistoricalData retrieves historical candle data
func (c *Client) GetHistoricalData(ctx context.Context, symbol, interval string, from, to time.Time, limit int) ([]*models.Candle, error) {
	start := time.Now()

	if err := c.CheckRateLimit(); err != nil {
		return nil, err
	}

	normalizedSymbol := NormalizeSymbol(symbol)
	normalizedInterval := NormalizeInterval(interval)

	if !IsValidInterval(normalizedInterval) {
		return nil, types.NewProviderError(Name, types.ErrorCodeBadRequest,
			fmt.Sprintf("unsupported interval: %s", interval), false)
	}

	endpoint := fmt.Sprintf("/products/%s/candles", normalizedSymbol)

	params := url.Values{}
	params.Set("granularity", strconv.Itoa(IntervalToSeconds(normalizedInterval)))

	if !from.IsZero() {
		params.Set("start", from.Format(time.RFC3339))
	}
	if !to.IsZero() {
		params.Set("end", to.Format(time.RFC3339))
	}

	var candleData [][]float64
	if err := c.makeRequest(ctx, "GET", endpoint+"?"+params.Encode(), nil, &candleData); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError(Name, types.ErrorCodeServerError,
			fmt.Sprintf("failed to get historical data for %s: %v", symbol, err), true)
	}

	candles := make([]*models.Candle, 0, len(candleData))

	for _, data := range candleData {
		if len(data) < 6 {
			continue
		}

		candle := &models.Candle{
			Timestamp: time.Unix(int64(data[0]), 0),
			Open:      decimal.NewFromFloat(data[3]),
			High:      decimal.NewFromFloat(data[2]),
			Low:       decimal.NewFromFloat(data[1]),
			Close:     decimal.NewFromFloat(data[4]),
			Volume:    decimal.NewFromFloat(data[5]),
		}

		candles = append(candles, candle)
	}

	// Apply limit if specified
	if limit > 0 && len(candles) > limit {
		candles = candles[:limit]
	}

	c.UpdateMetrics(true, time.Since(start))
	return candles, nil
}

// GetMarketData retrieves comprehensive market data
func (c *Client) GetMarketData(ctx context.Context, symbol string) (*models.MarketData, error) {
	start := time.Now()

	if err := c.CheckRateLimit(); err != nil {
		return nil, err
	}

	normalizedSymbol := NormalizeSymbol(symbol)

	// Get current ticker data
	tickerEndpoint := fmt.Sprintf("/products/%s/ticker", normalizedSymbol)
	var tickerResp TickerResponse
	if err := c.makeRequest(ctx, "GET", tickerEndpoint, nil, &tickerResp); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError(Name, types.ErrorCodeServerError,
			fmt.Sprintf("failed to get market data for %s: %v", symbol, err), true)
	}

	// Get 24h stats
	statsEndpoint := fmt.Sprintf("/products/%s/stats", normalizedSymbol)
	var statsResp StatsResponse
	if err := c.makeRequest(ctx, "GET", statsEndpoint, nil, &statsResp); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError(Name, types.ErrorCodeServerError,
			fmt.Sprintf("failed to get stats for %s: %v", symbol, err), true)
	}

	price, _ := decimal.NewFromString(tickerResp.Price)
	volume, _ := decimal.NewFromString(statsResp.Volume)
	high24h, _ := decimal.NewFromString(statsResp.High)
	low24h, _ := decimal.NewFromString(statsResp.Low)
	open24h, _ := decimal.NewFromString(statsResp.Open)

	var change24h, changePercent24h decimal.Decimal
	if !open24h.IsZero() {
		change24h = price.Sub(open24h)
		changePercent24h = change24h.Div(open24h).Mul(decimal.NewFromInt(100))
	}

	marketData := &models.MarketData{
		Symbol:                   DenormalizeSymbol(normalizedSymbol),
		CurrentPrice:             price,
		TotalVolume:              volume,
		High24h:                  high24h,
		Low24h:                   low24h,
		PriceChange24h:           change24h,
		PriceChangePercentage24h: changePercent24h,
		LastUpdated:              time.Now(),
		DataSource:               Name,
	}

	c.UpdateMetrics(true, time.Since(start))
	return marketData, nil
}

// GetOrderBook retrieves order book data
func (c *Client) GetOrderBook(ctx context.Context, symbol string, depth int) (*models.OrderBook, error) {
	start := time.Now()

	if err := c.CheckRateLimit(); err != nil {
		return nil, err
	}

	if depth <= 0 {
		depth = 20
	}
	if depth > 50 {
		depth = 50 // Coinbase Pro max depth
	}

	normalizedSymbol := NormalizeSymbol(symbol)
	endpoint := fmt.Sprintf("/products/%s/book", normalizedSymbol)

	params := url.Values{}
	if depth <= 50 {
		params.Set("level", "2")
	} else {
		params.Set("level", "3")
	}

	var orderBookResp OrderBookResponse
	if err := c.makeRequest(ctx, "GET", endpoint+"?"+params.Encode(), nil, &orderBookResp); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError(Name, types.ErrorCodeServerError,
			fmt.Sprintf("failed to get order book for %s: %v", symbol, err), true)
	}

	// Convert to our format
	bids := make([]*models.OrderLevel, 0)
	asks := make([]*models.OrderLevel, 0)

	for i, bid := range orderBookResp.Bids {
		if i >= depth {
			break
		}
		price, _ := decimal.NewFromString(bid[0])
		size, _ := decimal.NewFromString(bid[1])

		bids = append(bids, &models.OrderLevel{
			Price:    price,
			Amount: size,
		})
	}

	for i, ask := range orderBookResp.Asks {
		if i >= depth {
			break
		}
		price, _ := decimal.NewFromString(ask[0])
		size, _ := decimal.NewFromString(ask[1])

		asks = append(asks, &models.OrderLevel{
			Price:    price,
			Amount: size,
		})
	}

	orderBook := &models.OrderBook{
		Symbol:    DenormalizeSymbol(normalizedSymbol),
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
		// No Source field in OrderBook,
	}

	c.UpdateMetrics(true, time.Since(start))
	return orderBook, nil
}

// CheckRateLimit checks if we can make a request
func (c *Client) CheckRateLimit() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if now.Sub(c.lastReset) >= time.Second {
		c.requestCount = 0
		c.lastReset = now
	}

	if c.requestCount >= 10 { // Coinbase Pro limit: 10 requests per second
		return types.NewProviderError(Name, types.ErrorCodeRateLimit, "rate limit exceeded", true)
	}

	c.requestCount++
	return nil
}

// Ping checks if the service is accessible
func (c *Client) Ping(ctx context.Context) error {
	start := time.Now()

	var timeResp TimeResponse
	err := c.makeRequest(ctx, "GET", "/time", nil, &timeResp)

	c.UpdateMetrics(err == nil, time.Since(start))

	if err != nil {
		c.UpdateStatus(types.StatusDown, time.Since(start), 1)
		return types.NewProviderError(Name, types.ErrorCodeServerError,
			fmt.Sprintf("ping failed: %v", err), true)
	}

	c.UpdateStatus(types.StatusHealthy, time.Since(start), 0)
	return nil
}

// IsHealthy returns whether the provider is healthy
func (c *Client) IsHealthy() bool {
	// Simple health check based on recent errors
	if c.ProviderClient != nil {
		status := c.GetStatus()
		return status != nil && status.Status == types.StatusHealthy
	}
	return true
}

// makeRequest makes an HTTP request to the Coinbase Pro API
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	url := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "MarketDataAPI/1.0")

	// Add authentication if credentials are provided
	if c.apiKey != "" && c.secret != "" && c.passphrase != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)

		// For Coinbase Pro, we need CB-ACCESS-* headers
		req.Header.Set("CB-ACCESS-KEY", c.apiKey)
		req.Header.Set("CB-ACCESS-PASSPHRASE", c.passphrase)
		req.Header.Set("CB-ACCESS-TIMESTAMP", timestamp)

		// Note: For production use, implement proper HMAC-SHA256 signature
		// req.Header.Set("CB-ACCESS-SIGN", signature)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			return fmt.Errorf("API error: %s", errorResp.Message)
		}
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// UpdateMetrics updates provider metrics
func (c *Client) UpdateMetrics(success bool, latency time.Duration) {
	if c.ProviderClient != nil {
		c.ProviderClient.UpdateMetrics(success, latency)
	}
}

// UpdateStatus updates provider status
func (c *Client) UpdateStatus(status string, latency time.Duration, errorCount int) {
	if c.ProviderClient != nil {
		c.ProviderClient.UpdateStatus(status, latency, errorCount)
	}
}