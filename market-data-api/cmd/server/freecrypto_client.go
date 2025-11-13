package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FreeCryptoClient is a client for FreeCryptoAPI
type FreeCryptoClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewFreeCryptoClient creates a new FreeCrypto client
func NewFreeCryptoClient(apiKey string) *FreeCryptoClient {
	return &FreeCryptoClient{
		baseURL: "https://api.freecryptoapi.com/v1",
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Price represents a cryptocurrency price
type Price struct {
	Symbol    string
	Name      string
	Price     float64
	Change24h float64
	MarketCap float64
	Volume    float64
	Timestamp int64
}

// Candle represents historical price data
type Candle struct {
	Timestamp int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// FreeCryptoAPIResponse represents the API response structure
type FreeCryptoAPIResponse struct {
	Data map[string]FreeCryptoCoinData `json:"data"`
}

// FreeCryptoCoinData represents coin data from API
type FreeCryptoCoinData struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Change24h float64 `json:"change_24h"`
	MarketCap float64 `json:"market_cap"`
	Volume24h float64 `json:"volume_24h"`
}

// GetPrices fetches prices for multiple cryptocurrencies
func (c *FreeCryptoClient) GetPrices(ctx context.Context, symbols []string) (map[string]*Price, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("no symbols provided")
	}

	// Fetch each symbol individually
	prices := make(map[string]*Price)

	for _, symbol := range symbols {
		price, err := c.GetPrice(ctx, symbol)
		if err != nil {
			// Log error but continue with other symbols
			continue
		}
		prices[strings.ToUpper(symbol)] = price
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("no prices found for requested symbols")
	}

	return prices, nil
}

// GetPrice fetches the price for a single cryptocurrency
func (c *FreeCryptoClient) GetPrice(ctx context.Context, symbol string) (*Price, error) {
	// Build URL with symbol query parameter
	url := fmt.Sprintf("%s/getData?symbol=%s", c.baseURL, strings.ToUpper(symbol))

	// Make request
	data, err := c.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	// Parse response
	var response struct {
		Status  string `json:"status"`
		Symbols []struct {
			Symbol                 string `json:"symbol"`
			Last                   string `json:"last"`
			LastBTC                string `json:"last_btc"`
			Lowest                 string `json:"lowest"`
			Highest                string `json:"highest"`
			Date                   string `json:"date"`
			DailyChangePercentage  string `json:"daily_change_percentage"`
			SourceExchange         string `json:"source_exchange"`
		} `json:"symbols"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if response.Status != "success" || len(response.Symbols) == 0 {
		return nil, fmt.Errorf("no data for symbol: %s", symbol)
	}

	coin := response.Symbols[0]

	// Parse string values to float64
	var price, change24h float64
	fmt.Sscanf(coin.Last, "%f", &price)
	fmt.Sscanf(coin.DailyChangePercentage, "%f", &change24h)

	return &Price{
		Symbol:    strings.ToUpper(coin.Symbol),
		Name:      strings.ToUpper(coin.Symbol), // API doesn't provide full name
		Price:     price,
		Change24h: change24h,
		MarketCap: 0, // API doesn't provide market cap
		Volume:    0, // API doesn't provide volume
		Timestamp: time.Now().Unix(),
	}, nil
}

// GetHistoricalData fetches historical price data
func (c *FreeCryptoClient) GetHistoricalData(ctx context.Context, symbol string, from, to time.Time) ([]*Candle, error) {
	// Calculate days difference
	duration := to.Sub(from)
	days := int(duration.Hours() / 24)
	if days < 1 {
		days = 1
	}

	// Build URL
	url := fmt.Sprintf("%s/history/%s?days=%d", c.baseURL, strings.ToLower(symbol), days)

	// Make request
	data, err := c.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	// Parse response
	var response struct {
		Data []struct {
			Timestamp int64   `json:"timestamp"`
			Price     float64 `json:"price"`
			Volume    float64 `json:"volume"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse historical data: %w", err)
	}

	// Convert to candles
	candles := make([]*Candle, 0, len(response.Data))

	for _, point := range response.Data {
		// For simple price data, we'll use it for OHLC
		candle := &Candle{
			Timestamp: point.Timestamp,
			Open:      point.Price,
			High:      point.Price,
			Low:       point.Price,
			Close:     point.Price,
			Volume:    point.Volume,
		}
		candles = append(candles, candle)
	}

	return candles, nil
}

// Ping checks if the API is accessible
func (c *FreeCryptoClient) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/ping", c.baseURL)
	_, err := c.makeRequest(ctx, url)
	return err
}

// makeRequest makes an HTTP request to the API
func (c *FreeCryptoClient) makeRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "market-data-api/1.0")

	// Add API key as Bearer token
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: HTTP %d - %s", resp.StatusCode, string(body))
	}

	return body, nil
}
