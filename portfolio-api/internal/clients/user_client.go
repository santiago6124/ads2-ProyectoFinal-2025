package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// UserClient maneja la comunicación con Users API
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
	ID             int32   `json:"id"`
	Username       string  `json:"username"`
	Email          string  `json:"email"`
	FirstName      *string `json:"first_name"`
	LastName       *string `json:"last_name"`
	Role           string  `json:"role"`
	InitialBalance float64 `json:"initial_balance"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	LastLogin      *string `json:"last_login"`
	IsActive       bool    `json:"is_active"`
}

func NewUserClient(config *UserClientConfig) *UserClient {
	if config.Timeout == 0 {
		config.Timeout = 15 * time.Second
	}

	return &UserClient{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// GetUser obtiene la información del usuario desde Users API
func (c *UserClient) GetUser(ctx context.Context, userID int64) (*UserResponse, error) {
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
		return nil, fmt.Errorf("user API returned status %d", resp.StatusCode)
	}

	var user UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &user, nil
}

// GetUserBalance obtiene el balance USD del usuario
func (c *UserClient) GetUserBalance(ctx context.Context, userID int64) (decimal.Decimal, error) {
	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get user: %w", err)
	}

	return decimal.NewFromFloat(user.InitialBalance), nil
}

// HealthCheck verifica la conectividad con Users API
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
		return fmt.Errorf("users API health check failed with status %d", resp.StatusCode)
	}

	return nil
}
