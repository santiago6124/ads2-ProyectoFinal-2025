package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	"orders-api/internal/models"
)

type MarketClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

type MarketClientConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

type PriceResponse struct {
	Price     *PriceData `json:"price"`
	Status    string     `json:"status"`
	Error     string     `json:"error,omitempty"`
	Timestamp string     `json:"timestamp"`
}

type PriceData struct {
	Symbol         string          `json:"symbol"`
	Price          decimal.Decimal `json:"price"`
	BidPrice       decimal.Decimal `json:"bid_price"`
	AskPrice       decimal.Decimal `json:"ask_price"`
	Volume24h      decimal.Decimal `json:"volume_24h"`
	Change24h      decimal.Decimal `json:"change_24h"`
	ChangePercent  decimal.Decimal `json:"change_percent"`
	High24h        decimal.Decimal `json:"high_24h"`
	Low24h         decimal.Decimal `json:"low_24h"`
	LastUpdated    string          `json:"last_updated"`
	Source         string          `json:"source"`
	Confidence     float64         `json:"confidence"`
}

type MarketConditionsResponse struct {
	Conditions *MarketConditionsData `json:"conditions"`
	Status     string                `json:"status"`
	Error      string                `json:"error,omitempty"`
	Timestamp  string                `json:"timestamp"`
}

type MarketConditionsData struct {
	Symbol           string          `json:"symbol"`
	Volatility       decimal.Decimal `json:"volatility"`
	Liquidity        decimal.Decimal `json:"liquidity"`
	Spread           decimal.Decimal `json:"spread"`
	SpreadPercent    decimal.Decimal `json:"spread_percent"`
	TradingVolume    decimal.Decimal `json:"trading_volume"`
	OrderBookDepth   decimal.Decimal `json:"order_book_depth"`
	MarketCap        decimal.Decimal `json:"market_cap"`
	CirculatingSupply decimal.Decimal `json:"circulating_supply"`
	MarketSentiment  string          `json:"market_sentiment"`
	TradingStatus    string          `json:"trading_status"`
	LastUpdated      string          `json:"last_updated"`
}

type CandlestickResponse struct {
	Candles   []*CandlestickData `json:"candles"`
	Status    string             `json:"status"`
	Error     string             `json:"error,omitempty"`
	Timestamp string             `json:"timestamp"`
}

type CandlestickData struct {
	Timestamp string          `json:"timestamp"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
	Volume    decimal.Decimal `json:"volume"`
	Trades    int64           `json:"trades"`
}

type OrderBookResponse struct {
	OrderBook *OrderBookData `json:"order_book"`
	Status    string         `json:"status"`
	Error     string         `json:"error,omitempty"`
	Timestamp string         `json:"timestamp"`
}

type OrderBookData struct {
	Symbol      string                 `json:"symbol"`
	Bids        []*OrderBookLevel      `json:"bids"`
	Asks        []*OrderBookLevel      `json:"asks"`
	LastUpdated string                 `json:"last_updated"`
	Sequence    int64                  `json:"sequence"`
}

type OrderBookLevel struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
	Count    int             `json:"count"`
}

func NewMarketClient(config *MarketClientConfig) *MarketClient {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &MarketClient{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

func (c *MarketClient) GetCurrentPrice(ctx context.Context, symbol string) (*models.PriceResult, error) {
	url := fmt.Sprintf("%s/api/market/price/%s", c.baseURL, symbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var priceResp PriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if priceResp.Error != "" {
		return nil, fmt.Errorf("market service error: %s", priceResp.Error)
	}

	if priceResp.Price == nil {
		return nil, fmt.Errorf("price data not available for symbol %s", symbol)
	}

	result := &models.PriceResult{
		Symbol:         priceResp.Price.Symbol,
		MarketPrice:    priceResp.Price.Price,
		BidPrice:       priceResp.Price.BidPrice,
		AskPrice:       priceResp.Price.AskPrice,
		ExecutionPrice: priceResp.Price.Price, // Will be adjusted with slippage
		Volume24h:      priceResp.Price.Volume24h,
		Change24h:      priceResp.Price.Change24h,
		ChangePercent:  priceResp.Price.ChangePercent,
		High24h:        priceResp.Price.High24h,
		Low24h:         priceResp.Price.Low24h,
		Source:         priceResp.Price.Source,
		Confidence:     priceResp.Price.Confidence,
		LastUpdated:    priceResp.Price.LastUpdated,
		Slippage:       decimal.Zero, // Will be calculated separately
		SlippagePerc:   decimal.Zero, // Will be calculated separately
	}

	return result, nil
}

func (c *MarketClient) GetMarketConditions(ctx context.Context, symbol string) (*models.MarketConditions, error) {
	url := fmt.Sprintf("%s/api/market/conditions/%s", c.baseURL, symbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var conditionsResp MarketConditionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&conditionsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if conditionsResp.Error != "" {
		return nil, fmt.Errorf("market service error: %s", conditionsResp.Error)
	}

	if conditionsResp.Conditions == nil {
		return nil, fmt.Errorf("market conditions not available for symbol %s", symbol)
	}

	conditions := conditionsResp.Conditions
	result := &models.MarketConditions{
		Symbol:            conditions.Symbol,
		Volatility:        conditions.Volatility,
		Liquidity:         conditions.Liquidity,
		Spread:            conditions.Spread,
		SpreadPercent:     conditions.SpreadPercent,
		TradingVolume:     conditions.TradingVolume,
		OrderBookDepth:    conditions.OrderBookDepth,
		MarketCap:         conditions.MarketCap,
		CirculatingSupply: conditions.CirculatingSupply,
		MarketSentiment:   conditions.MarketSentiment,
		TradingStatus:     conditions.TradingStatus,
		LastUpdated:       conditions.LastUpdated,
	}

	return result, nil
}

func (c *MarketClient) GetCandlestickData(ctx context.Context, symbol string, interval string, limit int) ([]*CandlestickData, error) {
	url := fmt.Sprintf("%s/api/market/candlesticks/%s?interval=%s&limit=%d", c.baseURL, symbol, interval, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var candleResp CandlestickResponse
	if err := json.NewDecoder(resp.Body).Decode(&candleResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if candleResp.Error != "" {
		return nil, fmt.Errorf("market service error: %s", candleResp.Error)
	}

	return candleResp.Candles, nil
}

func (c *MarketClient) GetOrderBook(ctx context.Context, symbol string, depth int) (*OrderBookData, error) {
	url := fmt.Sprintf("%s/api/market/orderbook/%s?depth=%d", c.baseURL, symbol, depth)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var orderBookResp OrderBookResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderBookResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if orderBookResp.Error != "" {
		return nil, fmt.Errorf("market service error: %s", orderBookResp.Error)
	}

	return orderBookResp.OrderBook, nil
}

func (c *MarketClient) GetMultiplePrices(ctx context.Context, symbols []string) (map[string]*models.PriceResult, error) {
	results := make(map[string]*models.PriceResult)

	for _, symbol := range symbols {
		price, err := c.GetCurrentPrice(ctx, symbol)
		if err != nil {
			return nil, fmt.Errorf("failed to get price for %s: %w", symbol, err)
		}
		results[symbol] = price
	}

	return results, nil
}

func (c *MarketClient) GetTradingPairs(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/api/market/pairs", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Pairs  []string `json:"pairs"`
		Status string   `json:"status"`
		Error  string   `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("market service error: %s", result.Error)
	}

	return result.Pairs, nil
}

func (c *MarketClient) SubscribeToPrice(ctx context.Context, symbol string, callback func(*models.PriceResult)) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			price, err := c.GetCurrentPrice(ctx, symbol)
			if err != nil {
				continue
			}
			callback(price)
		}
	}
}

func (c *MarketClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("market service health check failed with status %d", resp.StatusCode)
	}

	return nil
}