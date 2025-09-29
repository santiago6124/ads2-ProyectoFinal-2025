package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	"orders-api/internal/models"
)

type WalletClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

type WalletClientConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

type BalanceResponse struct {
	Balance   *BalanceData `json:"balance"`
	Status    string       `json:"status"`
	Error     string       `json:"error,omitempty"`
	Timestamp string       `json:"timestamp"`
}

type BalanceData struct {
	UserID        int             `json:"user_id"`
	Currency      string          `json:"currency"`
	Available     decimal.Decimal `json:"available"`
	Locked        decimal.Decimal `json:"locked"`
	Total         decimal.Decimal `json:"total"`
	LastUpdated   string          `json:"last_updated"`
	HasSufficient bool            `json:"has_sufficient"`
	Required      decimal.Decimal `json:"required,omitempty"`
}

type LockFundsRequest struct {
	UserID       int             `json:"user_id"`
	Amount       decimal.Decimal `json:"amount"`
	Currency     string          `json:"currency"`
	OrderID      string          `json:"order_id"`
	LockType     string          `json:"lock_type"`
	ExpiresAt    *time.Time      `json:"expires_at,omitempty"`
	Description  string          `json:"description,omitempty"`
}

type LockFundsResponse struct {
	LockID    string `json:"lock_id"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
}

type ReleaseFundsRequest struct {
	UserID   int             `json:"user_id"`
	Amount   decimal.Decimal `json:"amount"`
	Currency string          `json:"currency"`
	LockID   string          `json:"lock_id,omitempty"`
	OrderID  string          `json:"order_id,omitempty"`
	Reason   string          `json:"reason"`
}

type ReleaseFundsResponse struct {
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
}

type TransactionRequest struct {
	UserID      int             `json:"user_id"`
	Amount      decimal.Decimal `json:"amount"`
	Currency    string          `json:"currency"`
	Type        string          `json:"type"`
	OrderID     string          `json:"order_id"`
	Description string          `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type TransactionResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
	Timestamp     string `json:"timestamp"`
}

func NewWalletClient(config *WalletClientConfig) *WalletClient {
	if config.Timeout == 0 {
		config.Timeout = 15 * time.Second
	}

	return &WalletClient{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

func (c *WalletClient) CheckBalance(ctx context.Context, userID int, amount decimal.Decimal) (*models.BalanceResult, error) {
	url := fmt.Sprintf("%s/api/wallets/%d/balance?currency=USD&required_amount=%s", c.baseURL, userID, amount.String())

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

	var balanceResp BalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&balanceResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if balanceResp.Error != "" {
		return &models.BalanceResult{
			HasSufficient: false,
			Available:     decimal.Zero,
			Required:      amount,
			Currency:      "USD",
			Message:       balanceResp.Error,
		}, nil
	}

	if balanceResp.Balance == nil {
		return &models.BalanceResult{
			HasSufficient: false,
			Available:     decimal.Zero,
			Required:      amount,
			Currency:      "USD",
			Message:       "balance information not available",
		}, nil
	}

	result := &models.BalanceResult{
		HasSufficient: balanceResp.Balance.HasSufficient,
		Available:     balanceResp.Balance.Available,
		Required:      amount,
		Currency:      balanceResp.Balance.Currency,
		Locked:        balanceResp.Balance.Locked,
		Total:         balanceResp.Balance.Total,
	}

	if result.HasSufficient {
		result.Message = "sufficient balance available"
	} else {
		result.Message = fmt.Sprintf("insufficient balance: need %s, have %s",
			amount.String(), balanceResp.Balance.Available.String())
	}

	return result, nil
}

func (c *WalletClient) LockFunds(ctx context.Context, userID int, amount decimal.Decimal) error {
	return c.LockFundsWithDetails(ctx, userID, amount, "", "order_execution", nil, "Order execution lock")
}

func (c *WalletClient) LockFundsWithDetails(ctx context.Context, userID int, amount decimal.Decimal, orderID, lockType string, expiresAt *time.Time, description string) error {
	url := fmt.Sprintf("%s/api/wallets/%d/lock", c.baseURL, userID)

	request := LockFundsRequest{
		UserID:      userID,
		Amount:      amount,
		Currency:    "USD",
		OrderID:     orderID,
		LockType:    lockType,
		ExpiresAt:   expiresAt,
		Description: description,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var lockResp LockFundsResponse
	if err := json.NewDecoder(resp.Body).Decode(&lockResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if lockResp.Error != "" {
		return fmt.Errorf("wallet service error: %s", lockResp.Error)
	}

	if lockResp.Status != "success" && lockResp.Status != "locked" {
		return fmt.Errorf("failed to lock funds: status %s", lockResp.Status)
	}

	return nil
}

func (c *WalletClient) ReleaseFunds(ctx context.Context, userID int, amount decimal.Decimal) error {
	return c.ReleaseFundsWithDetails(ctx, userID, amount, "", "", "Order execution completed")
}

func (c *WalletClient) ReleaseFundsWithDetails(ctx context.Context, userID int, amount decimal.Decimal, lockID, orderID, reason string) error {
	url := fmt.Sprintf("%s/api/wallets/%d/release", c.baseURL, userID)

	request := ReleaseFundsRequest{
		UserID:   userID,
		Amount:   amount,
		Currency: "USD",
		LockID:   lockID,
		OrderID:  orderID,
		Reason:   reason,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var releaseResp ReleaseFundsResponse
	if err := json.NewDecoder(resp.Body).Decode(&releaseResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if releaseResp.Error != "" {
		return fmt.Errorf("wallet service error: %s", releaseResp.Error)
	}

	if releaseResp.Status != "success" && releaseResp.Status != "released" {
		return fmt.Errorf("failed to release funds: status %s", releaseResp.Status)
	}

	return nil
}

func (c *WalletClient) ProcessTransaction(ctx context.Context, userID int, amount decimal.Decimal, transactionType, orderID, description string) (string, error) {
	url := fmt.Sprintf("%s/api/wallets/%d/transactions", c.baseURL, userID)

	request := TransactionRequest{
		UserID:      userID,
		Amount:      amount,
		Currency:    "USD",
		Type:        transactionType,
		OrderID:     orderID,
		Description: description,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var txResp TransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&txResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if txResp.Error != "" {
		return "", fmt.Errorf("wallet service error: %s", txResp.Error)
	}

	if txResp.Status != "success" && txResp.Status != "completed" {
		return "", fmt.Errorf("transaction failed: status %s", txResp.Status)
	}

	return txResp.TransactionID, nil
}

func (c *WalletClient) GetBalance(ctx context.Context, userID int, currency string) (*BalanceData, error) {
	url := fmt.Sprintf("%s/api/wallets/%d/balance?currency=%s", c.baseURL, userID, currency)

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

	var balanceResp BalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&balanceResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if balanceResp.Error != "" {
		return nil, fmt.Errorf("wallet service error: %s", balanceResp.Error)
	}

	return balanceResp.Balance, nil
}

func (c *WalletClient) HealthCheck(ctx context.Context) error {
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
		return fmt.Errorf("wallet service health check failed with status %d", resp.StatusCode)
	}

	return nil
}