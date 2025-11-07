package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"portfolio-api/internal/config"
)

type OrdersClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	timeout    time.Duration
	retries    int
}

func NewOrdersClient(cfg config.ExternalAPIsConfig) *OrdersClient {
	return &OrdersClient{
		baseURL: cfg.OrdersAPI.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.OrdersAPI.Timeout,
		},
		apiKey:  cfg.OrdersAPI.APIKey,
		timeout: cfg.OrdersAPI.Timeout,
		retries: cfg.OrdersAPI.MaxRetries,
	}
}

// Order represents an order in the system
type Order struct {
	ID              string          `json:"id"`
	UserID          int64           `json:"user_id"`
	Symbol          string          `json:"symbol"`
	Type            string          `json:"type"` // "market", "limit", "stop"
	Side            string          `json:"side"` // "buy", "sell"
	Quantity        decimal.Decimal `json:"quantity"`
	Price           decimal.Decimal `json:"price,omitempty"`
	StopPrice       decimal.Decimal `json:"stop_price,omitempty"`
	Status          string          `json:"status"`
	FilledQuantity  decimal.Decimal `json:"filled_quantity"`
	AveragePrice    decimal.Decimal `json:"average_price"`
	Fee             decimal.Decimal `json:"fee"`
	FeeCurrency     string          `json:"fee_currency"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	FilledAt        *time.Time      `json:"filled_at,omitempty"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
}

// CreateOrderRequest represents a request to create an order
type CreateOrderRequest struct {
	UserID    int64           `json:"user_id"`
	Symbol    string          `json:"symbol"`
	Type      string          `json:"type"`
	Side      string          `json:"side"`
	Quantity  decimal.Decimal `json:"quantity"`
	Price     decimal.Decimal `json:"price,omitempty"`
	StopPrice decimal.Decimal `json:"stop_price,omitempty"`
	TimeInForce string        `json:"time_in_force,omitempty"` // "GTC", "IOC", "FOK"
}

// OrderHistory represents order history with pagination
type OrderHistory struct {
	Orders     []Order `json:"orders"`
	TotalCount int64   `json:"total_count"`
	HasMore    bool    `json:"has_more"`
	NextCursor string  `json:"next_cursor"`
}

// Trade represents a trade execution
type Trade struct {
	ID        string          `json:"id"`
	OrderID   string          `json:"order_id"`
	UserID    int64           `json:"user_id"`
	Symbol    string          `json:"symbol"`
	Side      string          `json:"side"`
	Quantity  decimal.Decimal `json:"quantity"`
	Price     decimal.Decimal `json:"price"`
	Fee       decimal.Decimal `json:"fee"`
	Total     decimal.Decimal `json:"total"`
	Timestamp time.Time       `json:"timestamp"`
}

// OrderStats represents order statistics
type OrderStats struct {
	TotalOrders    int64           `json:"total_orders"`
	FilledOrders   int64           `json:"filled_orders"`
	CancelledOrders int64          `json:"cancelled_orders"`
	TotalVolume    decimal.Decimal `json:"total_volume"`
	TotalFees      decimal.Decimal `json:"total_fees"`
	AverageOrderSize decimal.Decimal `json:"average_order_size"`
	SuccessRate    decimal.Decimal `json:"success_rate"`
}

// CreateOrder creates a new order
func (oc *OrdersClient) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
	url := fmt.Sprintf("%s/orders", oc.baseURL)

	var response struct {
		Data Order `json:"data"`
	}

	err := oc.makeRequest(ctx, "POST", url, req, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return &response.Data, nil
}

// GetOrder retrieves an order by ID
func (oc *OrdersClient) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	url := fmt.Sprintf("%s/orders/%s", oc.baseURL, orderID)

	var response struct {
		Data Order `json:"data"`
	}

	err := oc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get order %s: %w", orderID, err)
	}

	return &response.Data, nil
}

// GetUserOrders retrieves orders for a specific user
func (oc *OrdersClient) GetUserOrders(ctx context.Context, userID int64, status string, limit int, cursor string) (*OrderHistory, error) {
	url := fmt.Sprintf("%s/users/%d/orders?limit=%d", oc.baseURL, userID, limit)

	if status != "" {
		url += fmt.Sprintf("&status=%s", status)
	}
	if cursor != "" {
		url += fmt.Sprintf("&cursor=%s", cursor)
	}

	var response struct {
		Data OrderHistory `json:"data"`
	}

	err := oc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders for user %d: %w", userID, err)
	}

	return &response.Data, nil
}

// CancelOrder cancels an order
func (oc *OrdersClient) CancelOrder(ctx context.Context, orderID string) (*Order, error) {
	url := fmt.Sprintf("%s/orders/%s/cancel", oc.baseURL, orderID)

	var response struct {
		Data Order `json:"data"`
	}

	err := oc.makeRequest(ctx, "POST", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order %s: %w", orderID, err)
	}

	return &response.Data, nil
}

// CancelAllUserOrders cancels all open orders for a user
func (oc *OrdersClient) CancelAllUserOrders(ctx context.Context, userID int64, symbol string) ([]Order, error) {
	url := fmt.Sprintf("%s/users/%d/orders/cancel-all", oc.baseURL, userID)

	if symbol != "" {
		url += fmt.Sprintf("?symbol=%s", symbol)
	}

	var response struct {
		Data []Order `json:"data"`
	}

	err := oc.makeRequest(ctx, "POST", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel all orders for user %d: %w", userID, err)
	}

	return response.Data, nil
}

// GetUserTrades retrieves trade history for a user
func (oc *OrdersClient) GetUserTrades(ctx context.Context, userID int64, symbol string, from, to time.Time, limit int) ([]Trade, error) {
	url := fmt.Sprintf("%s/users/%d/trades?limit=%d", oc.baseURL, userID, limit)

	if symbol != "" {
		url += fmt.Sprintf("&symbol=%s", symbol)
	}
	if !from.IsZero() {
		url += fmt.Sprintf("&from=%s", from.Format("2006-01-02T15:04:05Z"))
	}
	if !to.IsZero() {
		url += fmt.Sprintf("&to=%s", to.Format("2006-01-02T15:04:05Z"))
	}

	var response struct {
		Data []Trade `json:"data"`
	}

	err := oc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades for user %d: %w", userID, err)
	}

	return response.Data, nil
}

// GetOrderTrades retrieves trades for a specific order
func (oc *OrdersClient) GetOrderTrades(ctx context.Context, orderID string) ([]Trade, error) {
	url := fmt.Sprintf("%s/orders/%s/trades", oc.baseURL, orderID)

	var response struct {
		Data []Trade `json:"data"`
	}

	err := oc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades for order %s: %w", orderID, err)
	}

	return response.Data, nil
}

// GetUserOrderStats retrieves order statistics for a user
func (oc *OrdersClient) GetUserOrderStats(ctx context.Context, userID int64, from, to time.Time) (*OrderStats, error) {
	url := fmt.Sprintf("%s/users/%d/stats", oc.baseURL, userID)

	if !from.IsZero() {
		url += fmt.Sprintf("?from=%s", from.Format("2006-01-02T15:04:05Z"))
	}
	if !to.IsZero() {
		if !from.IsZero() {
			url += fmt.Sprintf("&to=%s", to.Format("2006-01-02T15:04:05Z"))
		} else {
			url += fmt.Sprintf("?to=%s", to.Format("2006-01-02T15:04:05Z"))
		}
	}

	var response struct {
		Data OrderStats `json:"data"`
	}

	err := oc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get order stats for user %d: %w", userID, err)
	}

	return &response.Data, nil
}

// GetOpenOrders retrieves all open orders for a user
func (oc *OrdersClient) GetOpenOrders(ctx context.Context, userID int64, symbol string) ([]Order, error) {
	url := fmt.Sprintf("%s/users/%d/orders/open", oc.baseURL, userID)

	if symbol != "" {
		url += fmt.Sprintf("?symbol=%s", symbol)
	}

	var response struct {
		Data []Order `json:"data"`
	}

	err := oc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders for user %d: %w", userID, err)
	}

	return response.Data, nil
}

// ModifyOrder modifies an existing order (price, quantity, etc.)
func (oc *OrdersClient) ModifyOrder(ctx context.Context, orderID string, updates map[string]interface{}) (*Order, error) {
	url := fmt.Sprintf("%s/orders/%s", oc.baseURL, orderID)

	var response struct {
		Data Order `json:"data"`
	}

	err := oc.makeRequest(ctx, "PUT", url, updates, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to modify order %s: %w", orderID, err)
	}

	return &response.Data, nil
}

// GetOrderBook retrieves order book for a symbol (if available)
func (oc *OrdersClient) GetOrderBook(ctx context.Context, symbol string, depth int) (*OrderBook, error) {
	url := fmt.Sprintf("%s/orderbook/%s?depth=%d", oc.baseURL, symbol, depth)

	var response struct {
		Data OrderBook `json:"data"`
	}

	err := oc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book for %s: %w", symbol, err)
	}

	return &response.Data, nil
}

// OrderBook represents order book data
type OrderBook struct {
	Symbol    string      `json:"symbol"`
	Bids      []OrderLevel `json:"bids"`
	Asks      []OrderLevel `json:"asks"`
	Timestamp time.Time   `json:"timestamp"`
}

// OrderLevel represents a price level in the order book
type OrderLevel struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
	Count    int             `json:"count,omitempty"`
}

// makeRequest performs HTTP request with retry logic
func (oc *OrdersClient) makeRequest(ctx context.Context, method, url string, body interface{}, response interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= oc.retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		var reqBody []byte
		if body != nil {
			var err error
			reqBody, err = json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to marshal request body: %w", err)
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		// Add headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Portfolio-API/1.0")
		if oc.apiKey != "" {
			req.Header.Set("X-API-Key", oc.apiKey)
		}

		resp, err := oc.httpClient.Do(req)
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

		if response != nil {
			if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
				lastErr = fmt.Errorf("failed to decode response: %w", err)
				continue
			}
		}

		return nil
	}

	return fmt.Errorf("request failed after %d attempts: %w", oc.retries+1, lastErr)
}

// IsHealthy checks if the orders service is healthy
func (oc *OrdersClient) IsHealthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/health", oc.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetTradingPairs retrieves available trading pairs
func (oc *OrdersClient) GetTradingPairs(ctx context.Context) ([]TradingPair, error) {
	url := fmt.Sprintf("%s/trading-pairs", oc.baseURL)

	var response struct {
		Data []TradingPair `json:"data"`
	}

	err := oc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get trading pairs: %w", err)
	}

	return response.Data, nil
}

// TradingPair represents a trading pair configuration
type TradingPair struct {
	Symbol           string          `json:"symbol"`
	BaseAsset        string          `json:"base_asset"`
	QuoteAsset       string          `json:"quote_asset"`
	MinTradeSize     decimal.Decimal `json:"min_trade_size"`
	MaxTradeSize     decimal.Decimal `json:"max_trade_size"`
	PriceIncrement   decimal.Decimal `json:"price_increment"`
	SizeIncrement    decimal.Decimal `json:"size_increment"`
	TakerFee         decimal.Decimal `json:"taker_fee"`
	MakerFee         decimal.Decimal `json:"maker_fee"`
	IsActive         bool            `json:"is_active"`
}

// GetUserBalance retrieves user balance information (if available through orders service)
func (oc *OrdersClient) GetUserBalance(ctx context.Context, userID int64) (*UserBalance, error) {
	url := fmt.Sprintf("%s/users/%d/balance", oc.baseURL, userID)

	var response struct {
		Data UserBalance `json:"data"`
	}

	err := oc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance for user %d: %w", userID, err)
	}

	return &response.Data, nil
}

// UserBalance represents user balance information
type UserBalance struct {
	UserID    int64                      `json:"user_id"`
	Balances  map[string]AssetBalance    `json:"balances"`
	UpdatedAt time.Time                  `json:"updated_at"`
}

// AssetBalance represents balance for a specific asset
type AssetBalance struct {
	Asset     string          `json:"asset"`
	Free      decimal.Decimal `json:"free"`
	Locked    decimal.Decimal `json:"locked"`
	Total     decimal.Decimal `json:"total"`
}

// SimulateOrder simulates order execution without actually placing it
func (oc *OrdersClient) SimulateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderSimulation, error) {
	url := fmt.Sprintf("%s/orders/simulate", oc.baseURL)

	var response struct {
		Data OrderSimulation `json:"data"`
	}

	err := oc.makeRequest(ctx, "POST", url, req, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to simulate order: %w", err)
	}

	return &response.Data, nil
}

// OrderSimulation represents simulated order execution
type OrderSimulation struct {
	EstimatedPrice    decimal.Decimal `json:"estimated_price"`
	EstimatedQuantity decimal.Decimal `json:"estimated_quantity"`
	EstimatedFee      decimal.Decimal `json:"estimated_fee"`
	EstimatedTotal    decimal.Decimal `json:"estimated_total"`
	Slippage          decimal.Decimal `json:"slippage"`
	WouldFill         bool            `json:"would_fill"`
	Warnings          []string        `json:"warnings"`
}