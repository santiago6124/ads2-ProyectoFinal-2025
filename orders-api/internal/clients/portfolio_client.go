package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// PortfolioClient handles communication with Portfolio API
type PortfolioClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// PortfolioClientConfig configuration for portfolio client
type PortfolioClientConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// UpdateHoldingRequest request payload to update holdings
type UpdateHoldingRequest struct {
	Symbol     string  `json:"symbol"`
	Quantity   float64 `json:"quantity"`
	Price      float64 `json:"price"`
	OrderType  string  `json:"order_type"` // "buy" or "sell"
}

// NewPortfolioClient creates a new portfolio client
func NewPortfolioClient(config *PortfolioClientConfig) *PortfolioClient {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &PortfolioClient{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// UpdateHoldings updates a user's holdings after an order execution
// This signature matches the PortfolioClient interface in execution_service.go
func (c *PortfolioClient) UpdateHoldings(ctx context.Context, userID int64, symbol string, quantity, price decimal.Decimal, orderType string) error {
	url := fmt.Sprintf("%s/api/portfolio/%d/holdings", c.baseURL, userID)

	req := UpdateHoldingRequest{
		Symbol:    symbol,
		Quantity:  quantity.InexactFloat64(),
		Price:     price.InexactFloat64(),
		OrderType: orderType,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Service", "orders-api")
	httpReq.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		fmt.Printf("⚠️ Portfolio API connection failed (non-critical): %v\n", err)
		return nil // Don't fail the order if portfolio update fails
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		fmt.Printf("⚠️ Portfolio API returned status %d (non-critical)\n", resp.StatusCode)
		return nil // Don't fail the order if portfolio update fails
	}

	fmt.Printf("✅ Portfolio holdings updated: User %d, %s %s @ %f\n", 
		userID, orderType, symbol, price.InexactFloat64())
	return nil
}

// HealthCheck verifies connectivity with Portfolio API
func (c *PortfolioClient) HealthCheck(ctx context.Context) error {
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
		return fmt.Errorf("portfolio API health check failed with status %d", resp.StatusCode)
	}

	return nil
}
