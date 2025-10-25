package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"orders-api/internal/models"

	"github.com/shopspring/decimal"
)

// UserBalanceClient maneja el balance del usuario directamente desde Users API
type UserBalanceClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

type UserBalanceConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

type UserBalanceResponse struct {
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

func NewUserBalanceClient(config *UserBalanceConfig) *UserBalanceClient {
	if config.Timeout == 0 {
		config.Timeout = 15 * time.Second
	}

	return &UserBalanceClient{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// CheckBalance verifica si el usuario tiene suficiente balance para la orden
func (c *UserBalanceClient) CheckBalance(ctx context.Context, userID int, amount decimal.Decimal) (*models.BalanceResult, error) {
	// Obtener información del usuario desde Users API
	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Convertir balance a decimal
	availableBalance := decimal.NewFromFloat(user.InitialBalance)

	// Verificar si tiene suficiente balance
	hasSufficient := availableBalance.GreaterThanOrEqual(amount)

	result := &models.BalanceResult{
		HasSufficient: hasSufficient,
		Available:     availableBalance,
		Required:      amount,
		Currency:      "USD",
	}

	if hasSufficient {
		result.Message = "sufficient balance available"
	} else {
		result.Message = fmt.Sprintf("insufficient balance: need %s, have %s",
			amount.String(), availableBalance.String())
	}

	return result, nil
}

// LockFunds simula el lock de fondos (no hace nada real en el sistema simplificado)
func (c *UserBalanceClient) LockFunds(ctx context.Context, userID int, amount decimal.Decimal) error {
	// En el sistema simplificado, solo verificamos que tenga suficiente balance
	balance, err := c.CheckBalance(ctx, userID, amount)
	if err != nil {
		return fmt.Errorf("failed to check balance: %w", err)
	}

	if !balance.HasSufficient {
		return fmt.Errorf("insufficient balance: required %s, available %s",
			balance.Required.String(), balance.Available.String())
	}

	// En un sistema real, aquí se actualizaría el balance del usuario
	// Por ahora, solo logueamos la operación
	fmt.Printf("LOCK FUNDS: User %d, Amount %s USD\n", userID, amount.String())

	return nil
}

// ReleaseFunds simula el release de fondos (no hace nada real en el sistema simplificado)
func (c *UserBalanceClient) ReleaseFunds(ctx context.Context, userID int, amount decimal.Decimal) error {
	// En el sistema simplificado, solo logueamos la operación
	fmt.Printf("RELEASE FUNDS: User %d, Amount %s USD\n", userID, amount.String())
	return nil
}

// ProcessTransaction procesa una transacción actualizando el balance del usuario
func (c *UserBalanceClient) ProcessTransaction(ctx context.Context, userID int, amount decimal.Decimal, transactionType, orderID, description string) (string, error) {
	// Obtener usuario actual
	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	// Calcular nuevo balance
	currentBalance := decimal.NewFromFloat(user.InitialBalance)
	var newBalance decimal.Decimal

	switch transactionType {
	case "buy", "purchase":
		// Para compras, restamos el monto del balance
		newBalance = currentBalance.Sub(amount)
	case "sell", "sale":
		// Para ventas, sumamos el monto al balance
		newBalance = currentBalance.Add(amount)
	default:
		return "", fmt.Errorf("unknown transaction type: %s", transactionType)
	}

	// Verificar que el balance no sea negativo
	if newBalance.LessThan(decimal.Zero) {
		return "", fmt.Errorf("insufficient balance: would result in negative balance")
	}

	// En un sistema real, aquí se actualizaría el balance en la base de datos
	// Por ahora, solo logueamos la operación
	fmt.Printf("TRANSACTION: User %d, Type %s, Amount %s USD, New Balance %s USD\n",
		userID, transactionType, amount.String(), newBalance.String())

	// Generar ID de transacción simulado
	transactionID := fmt.Sprintf("tx_%d_%d", userID, time.Now().Unix())

	return transactionID, nil
}

// GetUser obtiene la información del usuario desde Users API
func (c *UserBalanceClient) GetUser(ctx context.Context, userID int) (*UserBalanceResponse, error) {
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

	var user UserBalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &user, nil
}

// HealthCheck verifica la conectividad con Users API
func (c *UserBalanceClient) HealthCheck(ctx context.Context) error {
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
