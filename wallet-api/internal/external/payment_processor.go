package external

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"wallet-api/internal/models"
)

type PaymentProcessor interface {
	ProcessDeposit(ctx context.Context, req *DepositRequest) (*DepositResponse, error)
	ProcessWithdrawal(ctx context.Context, req *WithdrawalRequest) (*WithdrawalResponse, error)
	VerifyWebhook(req *http.Request) (*WebhookEvent, error)
	GetTransactionStatus(ctx context.Context, externalTxID string) (*TransactionStatus, error)
	RefundTransaction(ctx context.Context, externalTxID string, reason string) (*RefundResponse, error)
	GetBalance(ctx context.Context) (*ProcessorBalance, error)
}

type paymentProcessor struct {
	config     *PaymentConfig
	httpClient *http.Client
}

type PaymentConfig struct {
	APIKey        string
	APISecret     string
	BaseURL       string
	WebhookSecret string
	Environment   string // "sandbox" or "production"
	Timeout       time.Duration
	RetryAttempts int
}

func NewPaymentProcessor(config *PaymentConfig) PaymentProcessor {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}

	return &paymentProcessor{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Request/Response types
type DepositRequest struct {
	UserID          int64           `json:"user_id"`
	Amount          decimal.Decimal `json:"amount"`
	Currency        string          `json:"currency"`
	PaymentMethod   string          `json:"payment_method"`
	Description     string          `json:"description"`
	IdempotencyKey  string          `json:"idempotency_key"`
	CustomerInfo    *CustomerInfo   `json:"customer_info,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type DepositResponse struct {
	ExternalTxID    string          `json:"external_tx_id"`
	Status          string          `json:"status"`
	Amount          decimal.Decimal `json:"amount"`
	Currency        string          `json:"currency"`
	Fee             decimal.Decimal `json:"fee"`
	NetAmount       decimal.Decimal `json:"net_amount"`
	PaymentURL      string          `json:"payment_url,omitempty"`
	ExpiresAt       time.Time       `json:"expires_at,omitempty"`
	ProcessorData   map[string]interface{} `json:"processor_data,omitempty"`
}

type WithdrawalRequest struct {
	UserID          int64           `json:"user_id"`
	Amount          decimal.Decimal `json:"amount"`
	Currency        string          `json:"currency"`
	Destination     string          `json:"destination"`
	Description     string          `json:"description"`
	IdempotencyKey  string          `json:"idempotency_key"`
	CustomerInfo    *CustomerInfo   `json:"customer_info,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type WithdrawalResponse struct {
	ExternalTxID    string          `json:"external_tx_id"`
	Status          string          `json:"status"`
	Amount          decimal.Decimal `json:"amount"`
	Currency        string          `json:"currency"`
	Fee             decimal.Decimal `json:"fee"`
	NetAmount       decimal.Decimal `json:"net_amount"`
	EstimatedTime   time.Duration   `json:"estimated_time,omitempty"`
	ProcessorData   map[string]interface{} `json:"processor_data,omitempty"`
}

type WebhookEvent struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"`
	ExternalTxID  string                 `json:"external_tx_id"`
	Status        string                 `json:"status"`
	Amount        decimal.Decimal        `json:"amount"`
	Currency      string                 `json:"currency"`
	Fee           decimal.Decimal        `json:"fee"`
	Timestamp     time.Time              `json:"timestamp"`
	Data          map[string]interface{} `json:"data"`
	Signature     string                 `json:"signature"`
}

type TransactionStatus struct {
	ExternalTxID    string          `json:"external_tx_id"`
	Status          string          `json:"status"`
	Amount          decimal.Decimal `json:"amount"`
	Currency        string          `json:"currency"`
	Fee             decimal.Decimal `json:"fee"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	FailureReason   string          `json:"failure_reason,omitempty"`
	ProcessorData   map[string]interface{} `json:"processor_data,omitempty"`
}

type RefundResponse struct {
	RefundID        string          `json:"refund_id"`
	ExternalTxID    string          `json:"external_tx_id"`
	Status          string          `json:"status"`
	Amount          decimal.Decimal `json:"amount"`
	Currency        string          `json:"currency"`
	ProcessedAt     time.Time       `json:"processed_at"`
}

type ProcessorBalance struct {
	Available       decimal.Decimal `json:"available"`
	Pending         decimal.Decimal `json:"pending"`
	Currency        string          `json:"currency"`
	LastUpdated     time.Time       `json:"last_updated"`
}

type CustomerInfo struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Phone       string `json:"phone,omitempty"`
	Country     string `json:"country,omitempty"`
}

// Implementation
func (p *paymentProcessor) ProcessDeposit(ctx context.Context, req *DepositRequest) (*DepositResponse, error) {
	// Prepare request payload
	payload := map[string]interface{}{
		"amount":           req.Amount.String(),
		"currency":         req.Currency,
		"payment_method":   req.PaymentMethod,
		"description":      req.Description,
		"idempotency_key":  req.IdempotencyKey,
		"customer_id":      strconv.FormatInt(req.UserID, 10),
		"return_url":       p.buildReturnURL("deposit", req.UserID),
		"webhook_url":      p.buildWebhookURL(),
		"metadata":         req.Metadata,
	}

	if req.CustomerInfo != nil {
		payload["customer"] = req.CustomerInfo
	}

	// Make API request
	response, err := p.makeRequest(ctx, "POST", "/deposits", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to process deposit: %w", err)
	}

	// Parse response
	var result struct {
		ID              string                 `json:"id"`
		Status          string                 `json:"status"`
		Amount          string                 `json:"amount"`
		Currency        string                 `json:"currency"`
		Fee             string                 `json:"fee"`
		NetAmount       string                 `json:"net_amount"`
		PaymentURL      string                 `json:"payment_url"`
		ExpiresAt       string                 `json:"expires_at"`
		ProcessorData   map[string]interface{} `json:"processor_data"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse deposit response: %w", err)
	}

	// Convert amounts
	amount, _ := decimal.NewFromString(result.Amount)
	fee, _ := decimal.NewFromString(result.Fee)
	netAmount, _ := decimal.NewFromString(result.NetAmount)

	// Parse expiration time
	var expiresAt time.Time
	if result.ExpiresAt != "" {
		expiresAt, _ = time.Parse(time.RFC3339, result.ExpiresAt)
	}

	return &DepositResponse{
		ExternalTxID:  result.ID,
		Status:        result.Status,
		Amount:        amount,
		Currency:      result.Currency,
		Fee:           fee,
		NetAmount:     netAmount,
		PaymentURL:    result.PaymentURL,
		ExpiresAt:     expiresAt,
		ProcessorData: result.ProcessorData,
	}, nil
}

func (p *paymentProcessor) ProcessWithdrawal(ctx context.Context, req *WithdrawalRequest) (*WithdrawalResponse, error) {
	// Prepare request payload
	payload := map[string]interface{}{
		"amount":          req.Amount.String(),
		"currency":        req.Currency,
		"destination":     req.Destination,
		"description":     req.Description,
		"idempotency_key": req.IdempotencyKey,
		"customer_id":     strconv.FormatInt(req.UserID, 10),
		"webhook_url":     p.buildWebhookURL(),
		"metadata":        req.Metadata,
	}

	if req.CustomerInfo != nil {
		payload["customer"] = req.CustomerInfo
	}

	// Make API request
	response, err := p.makeRequest(ctx, "POST", "/withdrawals", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to process withdrawal: %w", err)
	}

	// Parse response
	var result struct {
		ID              string                 `json:"id"`
		Status          string                 `json:"status"`
		Amount          string                 `json:"amount"`
		Currency        string                 `json:"currency"`
		Fee             string                 `json:"fee"`
		NetAmount       string                 `json:"net_amount"`
		EstimatedTime   int                    `json:"estimated_time_minutes"`
		ProcessorData   map[string]interface{} `json:"processor_data"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse withdrawal response: %w", err)
	}

	// Convert amounts
	amount, _ := decimal.NewFromString(result.Amount)
	fee, _ := decimal.NewFromString(result.Fee)
	netAmount, _ := decimal.NewFromString(result.NetAmount)

	return &WithdrawalResponse{
		ExternalTxID:    result.ID,
		Status:          result.Status,
		Amount:          amount,
		Currency:        result.Currency,
		Fee:             fee,
		NetAmount:       netAmount,
		EstimatedTime:   time.Duration(result.EstimatedTime) * time.Minute,
		ProcessorData:   result.ProcessorData,
	}, nil
}

func (p *paymentProcessor) VerifyWebhook(req *http.Request) (*WebhookEvent, error) {
	// Read the body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read webhook body: %w", err)
	}
	req.Body = io.NopCloser(bytes.NewBuffer(body))

	// Verify signature
	signature := req.Header.Get("X-Webhook-Signature")
	if !p.verifyWebhookSignature(body, signature) {
		return nil, fmt.Errorf("webhook signature verification failed")
	}

	// Parse webhook event
	var event struct {
		ID            string                 `json:"id"`
		Type          string                 `json:"type"`
		TransactionID string                 `json:"transaction_id"`
		Status        string                 `json:"status"`
		Amount        string                 `json:"amount"`
		Currency      string                 `json:"currency"`
		Fee           string                 `json:"fee"`
		Timestamp     string                 `json:"timestamp"`
		Data          map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook event: %w", err)
	}

	// Convert amounts
	amount, _ := decimal.NewFromString(event.Amount)
	fee, _ := decimal.NewFromString(event.Fee)

	// Parse timestamp
	timestamp, _ := time.Parse(time.RFC3339, event.Timestamp)

	return &WebhookEvent{
		EventID:      event.ID,
		EventType:    event.Type,
		ExternalTxID: event.TransactionID,
		Status:       event.Status,
		Amount:       amount,
		Currency:     event.Currency,
		Fee:          fee,
		Timestamp:    timestamp,
		Data:         event.Data,
		Signature:    signature,
	}, nil
}

func (p *paymentProcessor) GetTransactionStatus(ctx context.Context, externalTxID string) (*TransactionStatus, error) {
	// Make API request
	response, err := p.makeRequest(ctx, "GET", fmt.Sprintf("/transactions/%s", externalTxID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction status: %w", err)
	}

	// Parse response
	var result struct {
		ID              string                 `json:"id"`
		Status          string                 `json:"status"`
		Amount          string                 `json:"amount"`
		Currency        string                 `json:"currency"`
		Fee             string                 `json:"fee"`
		CreatedAt       string                 `json:"created_at"`
		UpdatedAt       string                 `json:"updated_at"`
		FailureReason   string                 `json:"failure_reason"`
		ProcessorData   map[string]interface{} `json:"processor_data"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse transaction status response: %w", err)
	}

	// Convert amounts
	amount, _ := decimal.NewFromString(result.Amount)
	fee, _ := decimal.NewFromString(result.Fee)

	// Parse timestamps
	createdAt, _ := time.Parse(time.RFC3339, result.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, result.UpdatedAt)

	return &TransactionStatus{
		ExternalTxID:    result.ID,
		Status:          result.Status,
		Amount:          amount,
		Currency:        result.Currency,
		Fee:             fee,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		FailureReason:   result.FailureReason,
		ProcessorData:   result.ProcessorData,
	}, nil
}

func (p *paymentProcessor) RefundTransaction(ctx context.Context, externalTxID string, reason string) (*RefundResponse, error) {
	// Prepare request payload
	payload := map[string]interface{}{
		"transaction_id": externalTxID,
		"reason":         reason,
	}

	// Make API request
	response, err := p.makeRequest(ctx, "POST", "/refunds", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to process refund: %w", err)
	}

	// Parse response
	var result struct {
		ID              string `json:"id"`
		TransactionID   string `json:"transaction_id"`
		Status          string `json:"status"`
		Amount          string `json:"amount"`
		Currency        string `json:"currency"`
		ProcessedAt     string `json:"processed_at"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse refund response: %w", err)
	}

	// Convert amount
	amount, _ := decimal.NewFromString(result.Amount)

	// Parse timestamp
	processedAt, _ := time.Parse(time.RFC3339, result.ProcessedAt)

	return &RefundResponse{
		RefundID:     result.ID,
		ExternalTxID: result.TransactionID,
		Status:       result.Status,
		Amount:       amount,
		Currency:     result.Currency,
		ProcessedAt:  processedAt,
	}, nil
}

func (p *paymentProcessor) GetBalance(ctx context.Context) (*ProcessorBalance, error) {
	// Make API request
	response, err := p.makeRequest(ctx, "GET", "/balance", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	// Parse response
	var result struct {
		Available   string `json:"available"`
		Pending     string `json:"pending"`
		Currency    string `json:"currency"`
		LastUpdated string `json:"last_updated"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse balance response: %w", err)
	}

	// Convert amounts
	available, _ := decimal.NewFromString(result.Available)
	pending, _ := decimal.NewFromString(result.Pending)

	// Parse timestamp
	lastUpdated, _ := time.Parse(time.RFC3339, result.LastUpdated)

	return &ProcessorBalance{
		Available:   available,
		Pending:     pending,
		Currency:    result.Currency,
		LastUpdated: lastUpdated,
	}, nil
}

// Helper methods
func (p *paymentProcessor) makeRequest(ctx context.Context, method, endpoint string, payload interface{}) ([]byte, error) {
	var body io.Reader
	var contentType string

	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
		contentType = "application/json"
	}

	url := p.config.BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("User-Agent", "CryptoSim-Wallet-API/1.0")

	// Add signature for authentication
	if body != nil {
		signature := p.generateRequestSignature(method, endpoint, payload)
		req.Header.Set("X-Signature", signature)
	}

	// Make request with retry logic
	var resp *http.Response
	var respErr error

	for attempt := 0; attempt < p.config.RetryAttempts; attempt++ {
		resp, respErr = p.httpClient.Do(req)
		if respErr == nil && resp.StatusCode < 500 {
			break
		}

		if attempt < p.config.RetryAttempts-1 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	if respErr != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", p.config.RetryAttempts, respErr)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errorResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}

		if json.Unmarshal(responseBody, &errorResp) == nil {
			return nil, fmt.Errorf("API error %d: %s - %s", resp.StatusCode, errorResp.Error, errorResp.Message)
		}

		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

func (p *paymentProcessor) generateRequestSignature(method, endpoint string, payload interface{}) string {
	// Create signature data
	data := method + endpoint
	if payload != nil {
		if jsonData, err := json.Marshal(payload); err == nil {
			data += string(jsonData)
		}
	}
	data += strconv.FormatInt(time.Now().Unix(), 10)

	// Generate HMAC signature
	mac := hmac.New(sha256.New, []byte(p.config.APISecret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func (p *paymentProcessor) verifyWebhookSignature(body []byte, signature string) bool {
	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(p.config.WebhookSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (p *paymentProcessor) buildReturnURL(operation string, userID int64) string {
	return fmt.Sprintf("https://app.cryptosim.com/wallet/%s/return?user_id=%d", operation, userID)
}

func (p *paymentProcessor) buildWebhookURL() string {
	return "https://wallet-api.cryptosim.com/webhook/payment"
}

// MapStatusToWalletStatus maps external processor status to internal wallet status
func MapStatusToWalletStatus(externalStatus string) string {
	switch externalStatus {
	case "pending", "processing", "submitted":
		return "processing"
	case "completed", "success", "confirmed":
		return "completed"
	case "failed", "declined", "rejected":
		return "failed"
	case "cancelled", "canceled":
		return "cancelled"
	case "refunded":
		return "reversed"
	default:
		return "pending"
	}
}

// CreateReferenceFromProcessor creates a wallet reference from processor data
func CreateReferenceFromProcessor(processorResponse interface{}, txType string) models.Reference {
	return models.Reference{
		Type:        "external",
		ID:          fmt.Sprintf("processor_%s", txType),
		Description: fmt.Sprintf("External %s transaction", txType),
		Metadata: map[string]interface{}{
			"processor": "payment_processor",
			"type":      txType,
			"data":      processorResponse,
		},
	}
}