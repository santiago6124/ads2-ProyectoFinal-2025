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

// HoldingsCheckResult result from checking holdings
// Exported to be used by services package
type HoldingsCheckResult struct {
	HasSufficient bool    `json:"has_sufficient"`
	Available     float64 `json:"available"`
	Required      float64 `json:"required"`
}

// PortfolioResponse response from portfolio API
type PortfolioResponse struct {
	Holdings []HoldingInfo `json:"holdings"`
}

// HoldingInfo holding information
type HoldingInfo struct {
	Symbol   string `json:"symbol"`
	Quantity string `json:"quantity"`
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

// CheckHoldings checks if user has sufficient holdings for a sell order
func (c *PortfolioClient) CheckHoldings(ctx context.Context, userID int64, symbol string, required decimal.Decimal) (*HoldingsCheckResult, error) {
	url := fmt.Sprintf("%s/api/portfolios/%d", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Internal-Service", "orders-api")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("portfolio API returned status %d", resp.StatusCode)
	}

	var portfolio PortfolioResponse
	if err := json.NewDecoder(resp.Body).Decode(&portfolio); err != nil {
		return nil, fmt.Errorf("failed to decode portfolio: %w", err)
	}

	// Find holding for symbol
	var available decimal.Decimal = decimal.Zero
	for _, holding := range portfolio.Holdings {
		if holding.Symbol == symbol {
			qty, err := decimal.NewFromString(holding.Quantity)
			if err != nil {
				return nil, fmt.Errorf("invalid quantity format: %w", err)
			}
			available = qty
			break
		}
	}

	result := &HoldingsCheckResult{
		HasSufficient: available.GreaterThanOrEqual(required),
		Available:     available.InexactFloat64(),
		Required:      required.InexactFloat64(),
	}

	return result, nil
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
