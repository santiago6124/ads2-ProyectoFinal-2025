package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"orders-api/internal/models"
)

type UserClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

type UserClientConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

type UserResponse struct {
	User   *UserData `json:"user"`
	Status string    `json:"status"`
	Error  string    `json:"error,omitempty"`
}

type UserData struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	Status       string `json:"status"`
	TierLevel    int    `json:"tier_level"`
	IsVerified   bool   `json:"is_verified"`
	IsActive     bool   `json:"is_active"`
	IsSuspended  bool   `json:"is_suspended"`
	CreatedAt    string `json:"created_at"`
	LastLoginAt  string `json:"last_login_at"`
	KYCStatus    string `json:"kyc_status"`
	TradingLimit string `json:"trading_limit"`
}

func NewUserClient(config *UserClientConfig) *UserClient {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &UserClient{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

func (c *UserClient) VerifyUser(ctx context.Context, userID int) (*models.ValidationResult, error) {
	url := fmt.Sprintf("%s/api/users/%d/verify", c.baseURL, userID)

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

	if resp.StatusCode != http.StatusOK {
		return &models.ValidationResult{
			IsValid: false,
			Message: fmt.Sprintf("user service returned status %d", resp.StatusCode),
		}, nil
	}

	var userResp UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if userResp.Error != "" {
		return &models.ValidationResult{
			IsValid: false,
			Message: userResp.Error,
		}, nil
	}

	if userResp.User == nil {
		return &models.ValidationResult{
			IsValid: false,
			Message: "user not found",
		}, nil
	}

	validationResult := &models.ValidationResult{
		IsValid: c.validateUserStatus(userResp.User),
		UserID:  userResp.User.ID,
		Message: c.getValidationMessage(userResp.User),
	}

	return validationResult, nil
}

func (c *UserClient) GetUserProfile(ctx context.Context, userID int) (*UserData, error) {
	url := fmt.Sprintf("%s/api/users/%d", c.baseURL, userID)

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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user service returned status %d", resp.StatusCode)
	}

	var userResp UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if userResp.Error != "" {
		return nil, fmt.Errorf("user service error: %s", userResp.Error)
	}

	return userResp.User, nil
}

func (c *UserClient) CheckUserPermissions(ctx context.Context, userID int, action string) (bool, error) {
	url := fmt.Sprintf("%s/api/users/%d/permissions?action=%s", c.baseURL, userID, action)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("user service returned status %d", resp.StatusCode)
	}

	var result struct {
		Allowed bool   `json:"allowed"`
		Error   string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != "" {
		return false, fmt.Errorf("user service error: %s", result.Error)
	}

	return result.Allowed, nil
}

func (c *UserClient) validateUserStatus(user *UserData) bool {
	if !user.IsActive || user.IsSuspended {
		return false
	}

	if user.Status != "active" {
		return false
	}

	if user.KYCStatus != "verified" && user.KYCStatus != "approved" {
		return false
	}

	return true
}

func (c *UserClient) getValidationMessage(user *UserData) string {
	if !user.IsActive {
		return "user account is inactive"
	}

	if user.IsSuspended {
		return "user account is suspended"
	}

	if user.Status != "active" {
		return fmt.Sprintf("user status is %s", user.Status)
	}

	if user.KYCStatus != "verified" && user.KYCStatus != "approved" {
		return fmt.Sprintf("user KYC status is %s", user.KYCStatus)
	}

	return "user is valid for trading"
}

func (c *UserClient) HealthCheck(ctx context.Context) error {
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
		return fmt.Errorf("user service health check failed with status %d", resp.StatusCode)
	}

	return nil
}