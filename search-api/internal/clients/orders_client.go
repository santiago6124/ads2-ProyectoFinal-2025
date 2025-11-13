package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OrdersClient handles HTTP communication with orders-api
type OrdersClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// OrdersClientConfig represents client configuration
type OrdersClientConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// OrderResponse represents the order response from orders-api
type OrderResponse struct {
	ID             string     `json:"id"`
	UserID         int        `json:"user_id"`
	Type           string     `json:"type"`
	OrderKind      string     `json:"order_kind"`
	Status         string     `json:"status"`
	CryptoSymbol   string     `json:"crypto_symbol"`
	CryptoName     string     `json:"crypto_name"`
	Quantity       string     `json:"quantity"`
	OrderPrice     string     `json:"order_price"`
	ExecutionPrice string     `json:"execution_price,omitempty"`
	TotalAmount    string     `json:"total_amount"`
	Fee            string     `json:"fee"`
	FeePercentage  string     `json:"fee_percentage"`
	CreatedAt      time.Time  `json:"created_at"`
	ExecutedAt     *time.Time `json:"executed_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CancelledAt    *time.Time `json:"cancelled_at,omitempty"`
}

// NewOrdersClient creates a new orders API client
func NewOrdersClient(config *OrdersClientConfig) *OrdersClient {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &OrdersClient{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// GetOrderByID retrieves an order by ID from orders-api
func (c *OrdersClient) GetOrderByID(ctx context.Context, orderID string) (*OrderResponse, error) {
	url := fmt.Sprintf("%s/api/v1/orders/%s", c.baseURL, orderID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Use internal API key for service-to-service communication
	req.Header.Set("X-Internal-Service", "search-api")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("orders-api returned status %d", resp.StatusCode)
	}

	var orderResp OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &orderResp, nil
}
