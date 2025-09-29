package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"portfolio-api/internal/config"
)

type MarketDataClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	timeout    time.Duration
	retries    int
}

func NewMarketDataClient(cfg config.ExternalAPIsConfig) *MarketDataClient {
	return &MarketDataClient{
		baseURL: cfg.MarketDataAPI.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		apiKey:  cfg.MarketDataAPI.APIKey,
		timeout: cfg.Timeout,
		retries: cfg.RetryCount,
	}
}

// MarketPrice represents current market price data
type MarketPrice struct {
	Symbol    string          `json:"symbol"`
	Price     decimal.Decimal `json:"price"`
	Change24h decimal.Decimal `json:"change_24h"`
	Volume24h decimal.Decimal `json:"volume_24h"`
	Timestamp time.Time       `json:"timestamp"`
}

// HistoricalPrice represents historical price data
type HistoricalPrice struct {
	Symbol    string          `json:"symbol"`
	Price     decimal.Decimal `json:"price"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Volume    decimal.Decimal `json:"volume"`
	Timestamp time.Time       `json:"timestamp"`
}

// MarketData represents comprehensive market data
type MarketData struct {
	Symbol      string          `json:"symbol"`
	Name        string          `json:"name"`
	Price       decimal.Decimal `json:"price"`
	MarketCap   decimal.Decimal `json:"market_cap"`
	Volume24h   decimal.Decimal `json:"volume_24h"`
	Change1h    decimal.Decimal `json:"change_1h"`
	Change24h   decimal.Decimal `json:"change_24h"`
	Change7d    decimal.Decimal `json:"change_7d"`
	Supply      decimal.Decimal `json:"supply"`
	MaxSupply   decimal.Decimal `json:"max_supply"`
	Rank        int             `json:"rank"`
	LastUpdated time.Time       `json:"last_updated"`
}

// GetPrice retrieves current price for a symbol
func (mdc *MarketDataClient) GetPrice(ctx context.Context, symbol string) (*MarketPrice, error) {
	url := fmt.Sprintf("%s/price/%s", mdc.baseURL, strings.ToUpper(symbol))

	var response struct {
		Data MarketPrice `json:"data"`
	}

	err := mdc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get price for %s: %w", symbol, err)
	}

	return &response.Data, nil
}

// GetPrices retrieves current prices for multiple symbols
func (mdc *MarketDataClient) GetPrices(ctx context.Context, symbols []string) (map[string]*MarketPrice, error) {
	if len(symbols) == 0 {
		return make(map[string]*MarketPrice), nil
	}

	symbolsParam := strings.Join(symbols, ",")
	url := fmt.Sprintf("%s/prices?symbols=%s", mdc.baseURL, strings.ToUpper(symbolsParam))

	var response struct {
		Data map[string]MarketPrice `json:"data"`
	}

	err := mdc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get prices for symbols %v: %w", symbols, err)
	}

	result := make(map[string]*MarketPrice)
	for symbol, price := range response.Data {
		priceCopy := price
		result[symbol] = &priceCopy
	}

	return result, nil
}

// GetHistoricalPrices retrieves historical price data
func (mdc *MarketDataClient) GetHistoricalPrices(ctx context.Context, symbol string, from, to time.Time, interval string) ([]HistoricalPrice, error) {
	url := fmt.Sprintf("%s/historical/%s?from=%s&to=%s&interval=%s",
		mdc.baseURL,
		strings.ToUpper(symbol),
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
		interval)

	var response struct {
		Data []HistoricalPrice `json:"data"`
	}

	err := mdc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical prices for %s: %w", symbol, err)
	}

	return response.Data, nil
}

// GetMarketData retrieves comprehensive market data for a symbol
func (mdc *MarketDataClient) GetMarketData(ctx context.Context, symbol string) (*MarketData, error) {
	url := fmt.Sprintf("%s/market-data/%s", mdc.baseURL, strings.ToUpper(symbol))

	var response struct {
		Data MarketData `json:"data"`
	}

	err := mdc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get market data for %s: %w", symbol, err)
	}

	return &response.Data, nil
}

// GetTopCoins retrieves top coins by market cap
func (mdc *MarketDataClient) GetTopCoins(ctx context.Context, limit int) ([]MarketData, error) {
	url := fmt.Sprintf("%s/top-coins?limit=%d", mdc.baseURL, limit)

	var response struct {
		Data []MarketData `json:"data"`
	}

	err := mdc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get top coins: %w", err)
	}

	return response.Data, nil
}

// SearchSymbols searches for symbols matching a query
func (mdc *MarketDataClient) SearchSymbols(ctx context.Context, query string, limit int) ([]MarketData, error) {
	url := fmt.Sprintf("%s/search?q=%s&limit=%d", mdc.baseURL, query, limit)

	var response struct {
		Data []MarketData `json:"data"`
	}

	err := mdc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to search symbols for query %s: %w", query, err)
	}

	return response.Data, nil
}

// GetCandlestickData retrieves candlestick/OHLCV data
func (mdc *MarketDataClient) GetCandlestickData(ctx context.Context, symbol string, interval string, limit int) ([]HistoricalPrice, error) {
	url := fmt.Sprintf("%s/candlestick/%s?interval=%s&limit=%d",
		mdc.baseURL,
		strings.ToUpper(symbol),
		interval,
		limit)

	var response struct {
		Data []HistoricalPrice `json:"data"`
	}

	err := mdc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get candlestick data for %s: %w", symbol, err)
	}

	return response.Data, nil
}

// GetExchangeRates retrieves exchange rates for currency conversion
func (mdc *MarketDataClient) GetExchangeRates(ctx context.Context, baseCurrency string) (map[string]decimal.Decimal, error) {
	url := fmt.Sprintf("%s/exchange-rates?base=%s", mdc.baseURL, strings.ToUpper(baseCurrency))

	var response struct {
		Data map[string]decimal.Decimal `json:"data"`
	}

	err := mdc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rates for %s: %w", baseCurrency, err)
	}

	return response.Data, nil
}

// GetMarketStats retrieves overall market statistics
func (mdc *MarketDataClient) GetMarketStats(ctx context.Context) (*MarketStats, error) {
	url := fmt.Sprintf("%s/market-stats", mdc.baseURL)

	var response struct {
		Data MarketStats `json:"data"`
	}

	err := mdc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get market stats: %w", err)
	}

	return &response.Data, nil
}

// MarketStats represents overall market statistics
type MarketStats struct {
	TotalMarketCap       decimal.Decimal `json:"total_market_cap"`
	Total24hVolume       decimal.Decimal `json:"total_24h_volume"`
	BitcoinDominance     decimal.Decimal `json:"bitcoin_dominance"`
	EthereumDominance    decimal.Decimal `json:"ethereum_dominance"`
	ActiveCryptocurrencies int           `json:"active_cryptocurrencies"`
	TotalExchanges       int             `json:"total_exchanges"`
	MarketCapChange24h   decimal.Decimal `json:"market_cap_change_24h"`
	LastUpdated          time.Time       `json:"last_updated"`
}

// makeRequest performs HTTP request with retry logic
func (mdc *MarketDataClient) makeRequest(ctx context.Context, method, url string, body interface{}, response interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= mdc.retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		// Add headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Portfolio-API/1.0")
		if mdc.apiKey != "" {
			req.Header.Set("X-API-Key", mdc.apiKey)
		}

		resp, err := mdc.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}
		defer resp.Body.Close()

		// Check for rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("HTTP %d: request failed", resp.StatusCode)
			continue
		}

		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			lastErr = fmt.Errorf("failed to decode response: %w", err)
			continue
		}

		return nil
	}

	return fmt.Errorf("request failed after %d attempts: %w", mdc.retries+1, lastErr)
}

// IsHealthy checks if the market data service is healthy
func (mdc *MarketDataClient) IsHealthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/health", mdc.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := mdc.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetSupportedSymbols retrieves list of supported symbols
func (mdc *MarketDataClient) GetSupportedSymbols(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/symbols", mdc.baseURL)

	var response struct {
		Data []string `json:"data"`
	}

	err := mdc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get supported symbols: %w", err)
	}

	return response.Data, nil
}

// GetPriceChange retrieves price change data for a symbol
func (mdc *MarketDataClient) GetPriceChange(ctx context.Context, symbol string, timeframe string) (*PriceChange, error) {
	url := fmt.Sprintf("%s/price-change/%s?timeframe=%s", mdc.baseURL, strings.ToUpper(symbol), timeframe)

	var response struct {
		Data PriceChange `json:"data"`
	}

	err := mdc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get price change for %s: %w", symbol, err)
	}

	return &response.Data, nil
}

// PriceChange represents price change data
type PriceChange struct {
	Symbol           string          `json:"symbol"`
	CurrentPrice     decimal.Decimal `json:"current_price"`
	PreviousPrice    decimal.Decimal `json:"previous_price"`
	AbsoluteChange   decimal.Decimal `json:"absolute_change"`
	PercentageChange decimal.Decimal `json:"percentage_change"`
	Timeframe        string          `json:"timeframe"`
	Timestamp        time.Time       `json:"timestamp"`
}

// GetVolatilityData retrieves volatility data for a symbol
func (mdc *MarketDataClient) GetVolatilityData(ctx context.Context, symbol string, days int) (*VolatilityData, error) {
	url := fmt.Sprintf("%s/volatility/%s?days=%d", mdc.baseURL, strings.ToUpper(symbol), days)

	var response struct {
		Data VolatilityData `json:"data"`
	}

	err := mdc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get volatility data for %s: %w", symbol, err)
	}

	return &response.Data, nil
}

// VolatilityData represents volatility metrics
type VolatilityData struct {
	Symbol           string          `json:"symbol"`
	Volatility       decimal.Decimal `json:"volatility"`
	AnnualizedVol    decimal.Decimal `json:"annualized_volatility"`
	StandardDev      decimal.Decimal `json:"standard_deviation"`
	AverageReturn    decimal.Decimal `json:"average_return"`
	MaxDrawdown      decimal.Decimal `json:"max_drawdown"`
	SharpeRatio      decimal.Decimal `json:"sharpe_ratio"`
	Days             int             `json:"days"`
	LastUpdated      time.Time       `json:"last_updated"`
}