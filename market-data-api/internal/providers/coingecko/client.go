package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/time/rate"

	"market-data-api/internal/models"
	"market-data-api/internal/types"
)

// Client represents a CoinGecko API client
type Client struct {
	*types.ProviderClient
	apiKey      string
	httpClient  *http.Client
	rateLimiter *rate.Limiter
}

// Config represents CoinGecko client configuration
type Config struct {
	APIKey            string
	BaseURL           string
	Timeout           time.Duration
	RateLimit         int
	Weight            float64
	RetryAttempts     int
	RetryDelay        time.Duration
}

// NewClient creates a new CoinGecko client
func NewClient(config *Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.coingecko.com/api/v3"
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	if config.RateLimit == 0 {
		config.RateLimit = 50 // requests per minute
	}

	// Create rate limiter (50 requests per minute)
	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(config.RateLimit)), 1)

	client := &Client{
		apiKey: config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		rateLimiter: limiter,
		ProviderClient: &types.ProviderClient{},
	}

	// Initialize provider client
	client.ProviderClient = &types.ProviderClient{}

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

	// Convert symbol to CoinGecko ID
	coinID, err := c.symbolToCoinID(symbol)
	if err != nil {
		return nil, types.NewProviderError("coingecko", types.ErrorCodeInvalidSymbol, fmt.Sprintf("Invalid symbol: %s", symbol), false)
	}

	// Build URL
	endpoint := fmt.Sprintf("/simple/price?ids=%s&vs_currencies=usd&include_24hr_change=true&include_24hr_vol=true&include_market_cap=true", coinID)
	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, err
	}

	// Parse response
	var response map[string]SimplePriceResponse
	if err := json.Unmarshal(data, &response); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError("coingecko", "PARSE_ERROR", "Failed to parse response", false)
	}

	priceData, exists := response[coinID]
	if !exists {
		return nil, types.NewProviderError("coingecko", types.ErrorCodeNoData, fmt.Sprintf("No data for symbol: %s", symbol), false)
	}

	price := &models.Price{
		Symbol:        symbol,
		Price:         decimal.NewFromFloat(priceData.USD),
		PriceUSD:      decimal.NewFromFloat(priceData.USD),
		Timestamp:     time.Now(),
		Source:        "coingecko",
		Provider:      "coingecko",
		Volume24h:     decimal.NewFromFloat(priceData.USD24hVol),
		MarketCap:     decimal.NewFromFloat(priceData.USDMarketCap),
		Change24h:     decimal.NewFromFloat(priceData.USD24hChange),
		ChangePercent: decimal.NewFromFloat(priceData.USD24hChange),
		Confidence:    0.95, // CoinGecko is generally reliable
		Latency:       time.Since(start).Milliseconds(),
	}

	return price, nil
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

	// Convert symbols to CoinGecko IDs
	coinIDs := make([]string, 0, len(symbols))
	symbolToID := make(map[string]string)

	for _, symbol := range symbols {
		coinID, err := c.symbolToCoinID(symbol)
		if err == nil {
			coinIDs = append(coinIDs, coinID)
			symbolToID[coinID] = symbol
		}
	}

	if len(coinIDs) == 0 {
		return nil, types.NewProviderError("coingecko", types.ErrorCodeInvalidSymbol, "No valid symbols provided", false)
	}

	// Build URL
	idsParam := strings.Join(coinIDs, ",")
	endpoint := fmt.Sprintf("/simple/price?ids=%s&vs_currencies=usd&include_24hr_change=true&include_24hr_vol=true&include_market_cap=true", idsParam)

	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, err
	}

	// Parse response
	var response map[string]SimplePriceResponse
	if err := json.Unmarshal(data, &response); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError("coingecko", "PARSE_ERROR", "Failed to parse response", false)
	}

	// Convert to prices map
	prices := make(map[string]*models.Price)
	latency := time.Since(start).Milliseconds()

	for coinID, priceData := range response {
		symbol := symbolToID[coinID]
		prices[symbol] = &models.Price{
			Symbol:        symbol,
			Price:         decimal.NewFromFloat(priceData.USD),
			PriceUSD:      decimal.NewFromFloat(priceData.USD),
			Timestamp:     time.Now(),
			Source:        "coingecko",
			Provider:      "coingecko",
			Volume24h:     decimal.NewFromFloat(priceData.USD24hVol),
			MarketCap:     decimal.NewFromFloat(priceData.USDMarketCap),
			Change24h:     decimal.NewFromFloat(priceData.USD24hChange),
			ChangePercent: decimal.NewFromFloat(priceData.USD24hChange),
			Confidence:    0.95,
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

	coinID, err := c.symbolToCoinID(symbol)
	if err != nil {
		return nil, types.NewProviderError("coingecko", types.ErrorCodeInvalidSymbol, fmt.Sprintf("Invalid symbol: %s", symbol), false)
	}

	// CoinGecko uses different endpoints for different time ranges
	var endpoint string
	var days int

	// Calculate days difference
	duration := to.Sub(from)
	days = int(duration.Hours() / 24)

	if days <= 1 {
		endpoint = fmt.Sprintf("/coins/%s/market_chart?vs_currency=usd&days=1", coinID)
	} else if days <= 90 {
		endpoint = fmt.Sprintf("/coins/%s/market_chart?vs_currency=usd&days=%d", coinID, days)
	} else {
		endpoint = fmt.Sprintf("/coins/%s/market_chart?vs_currency=usd&days=%d&interval=daily", coinID, days)
	}

	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, err
	}

	// Parse response
	var response MarketChartResponse
	if err := json.Unmarshal(data, &response); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError("coingecko", "PARSE_ERROR", "Failed to parse historical data", false)
	}

	// Convert to candles
	candles := make([]*models.Candle, 0, len(response.Prices))

	for i, pricePoint := range response.Prices {
		if len(pricePoint) < 2 {
			continue
		}

		timestamp := time.Unix(int64(pricePoint[0])/1000, 0)
		price := decimal.NewFromFloat(pricePoint[1])

		// For CoinGecko, we only have price data, so we'll use it for OHLC
		candle := &models.Candle{
			Timestamp: timestamp,
			Open:      price,
			High:      price,
			Low:       price,
			Close:     price,
			Volume:    decimal.Zero,
		}

		// Add volume if available
		if i < len(response.TotalVolumes) && len(response.TotalVolumes[i]) >= 2 {
			candle.Volume = decimal.NewFromFloat(response.TotalVolumes[i][1])
		}

		// Filter by time range
		if timestamp.After(from) && timestamp.Before(to) {
			candles = append(candles, candle)
		}

		// Apply limit
		if limit > 0 && len(candles) >= limit {
			break
		}
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

	coinID, err := c.symbolToCoinID(symbol)
	if err != nil {
		return nil, types.NewProviderError("coingecko", types.ErrorCodeInvalidSymbol, fmt.Sprintf("Invalid symbol: %s", symbol), false)
	}

	endpoint := fmt.Sprintf("/coins/%s?localization=false&tickers=false&market_data=true&community_data=false&developer_data=false", coinID)

	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, err
	}

	// Parse response
	var response CoinResponse
	if err := json.Unmarshal(data, &response); err != nil {
		c.UpdateMetrics(false, time.Since(start))
		return nil, types.NewProviderError("coingecko", "PARSE_ERROR", "Failed to parse market data", false)
	}

	// Convert to MarketData
	marketData := &models.MarketData{
		Symbol:                       symbol,
		Name:                         response.Name,
		CurrentPrice:                 decimal.NewFromFloat(response.MarketData.CurrentPrice.USD),
		MarketCap:                    decimal.NewFromFloat(response.MarketData.MarketCap.USD),
		FullyDilutedValuation:        decimal.NewFromFloat(response.MarketData.FullyDilutedValuation.USD),
		TotalVolume:                  decimal.NewFromFloat(response.MarketData.TotalVolume.USD),
		High24h:                      decimal.NewFromFloat(response.MarketData.High24h.USD),
		Low24h:                       decimal.NewFromFloat(response.MarketData.Low24h.USD),
		PriceChange24h:               decimal.NewFromFloat(response.MarketData.PriceChange24h),
		PriceChangePercentage24h:     decimal.NewFromFloat(response.MarketData.PriceChangePercentage24h),
		PriceChangePercentage7d:      decimal.NewFromFloat(response.MarketData.PriceChangePercentage7d),
		PriceChangePercentage30d:     decimal.NewFromFloat(response.MarketData.PriceChangePercentage30d),
		PriceChangePercentage1y:      decimal.NewFromFloat(response.MarketData.PriceChangePercentage1y),
		ATH:                          decimal.NewFromFloat(response.MarketData.ATH.USD),
		ATHChangePercentage:          decimal.NewFromFloat(response.MarketData.ATHChangePercentage.USD),
		ATL:                          decimal.NewFromFloat(response.MarketData.ATL.USD),
		ATLChangePercentage:          decimal.NewFromFloat(response.MarketData.ATLChangePercentage.USD),
		CirculatingSupply:            decimal.NewFromFloat(response.MarketData.CirculatingSupply),
		TotalSupply:                  decimal.NewFromFloat(response.MarketData.TotalSupply),
		MaxSupply:                    decimal.NewFromFloat(response.MarketData.MaxSupply),
		LastUpdated:                  time.Now(),
		DataSource:                   "coingecko",
		Confidence:                   0.95,
	}

	// Parse dates
	if response.MarketData.ATHDate.USD != "" {
		if athDate, err := time.Parse(time.RFC3339, response.MarketData.ATHDate.USD); err == nil {
			marketData.ATHDate = &athDate
		}
	}

	if response.MarketData.ATLDate.USD != "" {
		if atlDate, err := time.Parse(time.RFC3339, response.MarketData.ATLDate.USD); err == nil {
			marketData.ATLDate = &atlDate
		}
	}

	return marketData, nil
}

// GetOrderBook returns an error as CoinGecko doesn't provide order book data
func (c *Client) GetOrderBook(ctx context.Context, symbol string, depth int) (*models.OrderBook, error) {
	return nil, types.NewProviderError("coingecko", "NOT_SUPPORTED", "Order book data not available from CoinGecko", false)
}

// Ping checks if the CoinGecko API is accessible
func (c *Client) Ping(ctx context.Context) error {
	start := time.Now()
	_, err := c.makeRequest(ctx, "/ping")
	latency := time.Since(start)

	if err != nil {
		c.UpdateStatus(types.StatusDown, latency, 1)
		return err
	}

	c.UpdateStatus(types.StatusHealthy, latency, 0)
	return nil
}

// makeRequest makes an HTTP request to the CoinGecko API
func (c *Client) makeRequest(ctx context.Context, endpoint string) ([]byte, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, types.NewProviderError("coingecko", types.ErrorCodeRateLimit, "Rate limit wait cancelled", true)
	}

	// Build full URL
	fullURL := c.GetBaseURL() + endpoint

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, types.NewProviderError("coingecko", "REQUEST_ERROR", "Failed to create request", false)
	}

	// Add headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "market-data-api/1.0")

	if c.apiKey != "" {
		req.Header.Set("x-cg-pro-api-key", c.apiKey)
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, types.NewProviderError("coingecko", types.ErrorCodeNetworkError, "Network error: "+err.Error(), true)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewProviderError("coingecko", "READ_ERROR", "Failed to read response", false)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	return body, nil
}

// handleErrorResponse handles error responses from the API
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	var errorMsg string
	var retryable bool

	switch statusCode {
	case http.StatusTooManyRequests:
		errorMsg = "Rate limit exceeded"
		retryable = true
	case http.StatusUnauthorized:
		errorMsg = "Unauthorized - check API key"
		retryable = false
	case http.StatusNotFound:
		errorMsg = "Resource not found"
		retryable = false
	case http.StatusBadRequest:
		errorMsg = "Bad request"
		retryable = false
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		errorMsg = "Server error"
		retryable = true
	default:
		errorMsg = fmt.Sprintf("HTTP %d", statusCode)
		retryable = false
	}

	// Try to parse error details from response body
	if len(body) > 0 {
		var errorResp ErrorResponse
		if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != "" {
			errorMsg = errorResp.Error
		}
	}

	return types.NewProviderError("coingecko", strconv.Itoa(statusCode), errorMsg, retryable)
}

// GetBaseURL returns the base URL for the API
func (c *Client) GetBaseURL() string {
	return "https://api.coingecko.com/api/v3" // Default base URL
}

// symbolToCoinID converts a cryptocurrency symbol to CoinGecko coin ID
func (c *Client) symbolToCoinID(symbol string) (string, error) {
	// This is a simplified mapping. In a real implementation, you might want to
	// fetch this mapping from CoinGecko's /coins/list endpoint and cache it
	symbolMap := map[string]string{
		"BTC":  "bitcoin",
		"ETH":  "ethereum",
		"BNB":  "binancecoin",
		"SOL":  "solana",
		"ADA":  "cardano",
		"DOT":  "polkadot",
		"MATIC": "matic-network",
		"AVAX": "avalanche-2",
		"LINK": "chainlink",
		"UNI":  "uniswap",
		"LTC":  "litecoin",
		"BCH":  "bitcoin-cash",
		"XRP":  "ripple",
		"DOGE": "dogecoin",
		"SHIB": "shiba-inu",
	}

	coinID, exists := symbolMap[strings.ToUpper(symbol)]
	if !exists {
		return "", fmt.Errorf("unknown symbol: %s", symbol)
	}

	return coinID, nil
}