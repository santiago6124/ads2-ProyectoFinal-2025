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
	"strings"
	"time"

	"wallet-api/internal/models"
)

type FraudDetectionService interface {
	AnalyzeTransaction(ctx context.Context, transaction *models.Transaction, wallet *models.Wallet) (*FraudAnalysisResult, error)
	ReportFraud(ctx context.Context, report *FraudReport) error
	GetRiskScore(ctx context.Context, userID int64) (*RiskScore, error)
	UpdateUserBehavior(ctx context.Context, userID int64, behavior *UserBehavior) error
}

type fraudDetectionService struct {
	httpClient *http.Client
	config     *FraudDetectionConfig
}

type FraudDetectionConfig struct {
	BaseURL    string
	APIKey     string
	SecretKey  string
	Timeout    time.Duration
	MaxRetries int
}

type FraudAnalysisResult struct {
	TransactionID string            `json:"transaction_id"`
	RiskScore     int               `json:"risk_score"`     // 0-100
	RiskLevel     string            `json:"risk_level"`     // low, medium, high, critical
	Flags         []string          `json:"flags"`
	Reasons       []string          `json:"reasons"`
	Action        string            `json:"action"`         // allow, review, block
	Confidence    float64           `json:"confidence"`     // 0.0-1.0
	Metadata      map[string]interface{} `json:"metadata"`
	Timestamp     time.Time         `json:"timestamp"`
}

type FraudReport struct {
	ReportID      string            `json:"report_id"`
	TransactionID string            `json:"transaction_id"`
	UserID        int64             `json:"user_id"`
	ReportType    string            `json:"report_type"`    // suspicious_activity, confirmed_fraud, false_positive
	Description   string            `json:"description"`
	Evidence      map[string]interface{} `json:"evidence"`
	ReportedBy    string            `json:"reported_by"`
	Timestamp     time.Time         `json:"timestamp"`
}

type RiskScore struct {
	UserID        int64     `json:"user_id"`
	CurrentScore  int       `json:"current_score"`
	BaselineScore int       `json:"baseline_score"`
	TrendScore    int       `json:"trend_score"`
	LastUpdated   time.Time `json:"last_updated"`
	Factors       []RiskFactor `json:"factors"`
}

type RiskFactor struct {
	Type        string  `json:"type"`
	Impact      int     `json:"impact"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}

type UserBehavior struct {
	UserID           int64     `json:"user_id"`
	DeviceFingerprint string   `json:"device_fingerprint"`
	IPAddress        string    `json:"ip_address"`
	Location         *Location `json:"location"`
	SessionDuration  int64     `json:"session_duration"`
	TransactionCount int       `json:"transaction_count"`
	AvgTransactionAmount string `json:"avg_transaction_amount"`
	Timestamp        time.Time `json:"timestamp"`
}

type Location struct {
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func NewFraudDetectionService(config *FraudDetectionConfig) FraudDetectionService {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &fraudDetectionService{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		config: config,
	}
}

func (f *fraudDetectionService) AnalyzeTransaction(ctx context.Context, transaction *models.Transaction, wallet *models.Wallet) (*FraudAnalysisResult, error) {
	payload := map[string]interface{}{
		"transaction_id": transaction.TransactionID,
		"user_id":        transaction.UserID,
		"wallet_id":      transaction.WalletID.Hex(),
		"amount":         transaction.Amount.Value.String(),
		"currency":       transaction.Amount.Currency,
		"type":           transaction.Type,
		"reference":      transaction.Reference,
		"timestamp":      transaction.CreatedAt,
		"wallet_balance": wallet.Balance.Total.String(),
		"wallet_age":     time.Since(wallet.CreatedAt).Hours(),
	}

	var result FraudAnalysisResult
	err := f.makeRequest(ctx, "POST", "/analyze/transaction", payload, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze transaction: %w", err)
	}

	return &result, nil
}

func (f *fraudDetectionService) ReportFraud(ctx context.Context, report *FraudReport) error {
	return f.makeRequest(ctx, "POST", "/reports/fraud", report, nil)
}

func (f *fraudDetectionService) GetRiskScore(ctx context.Context, userID int64) (*RiskScore, error) {
	var result RiskScore
	endpoint := fmt.Sprintf("/users/%d/risk-score", userID)
	err := f.makeRequest(ctx, "GET", endpoint, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk score: %w", err)
	}

	return &result, nil
}

func (f *fraudDetectionService) UpdateUserBehavior(ctx context.Context, userID int64, behavior *UserBehavior) error {
	endpoint := fmt.Sprintf("/users/%d/behavior", userID)
	return f.makeRequest(ctx, "POST", endpoint, behavior, nil)
}

func (f *fraudDetectionService) makeRequest(ctx context.Context, method, endpoint string, payload interface{}, result interface{}) error {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	url := f.config.BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", f.config.APIKey)

	// Add signature for request authentication
	if payload != nil {
		signature := f.generateSignature(payload)
		req.Header.Set("X-Signature", signature)
	}

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < f.config.MaxRetries; attempt++ {
		resp, lastErr = f.httpClient.Do(req)
		if lastErr == nil && resp.StatusCode < 500 {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < f.config.MaxRetries-1 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	if lastErr != nil {
		return fmt.Errorf("request failed after %d attempts: %w", f.config.MaxRetries, lastErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (f *fraudDetectionService) generateSignature(payload interface{}) string {
	jsonData, _ := json.Marshal(payload)
	h := hmac.New(sha256.New, []byte(f.config.SecretKey))
	h.Write(jsonData)
	return hex.EncodeToString(h.Sum(nil))
}

// Mock fraud detection service for testing/development
type MockFraudDetectionService struct{}

func NewMockFraudDetectionService() FraudDetectionService {
	return &MockFraudDetectionService{}
}

func (m *MockFraudDetectionService) AnalyzeTransaction(ctx context.Context, transaction *models.Transaction, wallet *models.Wallet) (*FraudAnalysisResult, error) {
	// Simple mock logic
	riskScore := 10
	riskLevel := "low"
	action := "allow"

	// Increase risk for large transactions
	if transaction.Amount.Value.GreaterThan(models.NewDecimal("10000")) {
		riskScore = 60
		riskLevel = "medium"
		action = "review"
	}

	// High risk for very large transactions
	if transaction.Amount.Value.GreaterThan(models.NewDecimal("50000")) {
		riskScore = 85
		riskLevel = "high"
		action = "block"
	}

	// Check for suspicious patterns
	flags := []string{}
	if strings.Contains(transaction.Reference, "suspicious") {
		flags = append(flags, "suspicious_reference")
		riskScore += 30
	}

	return &FraudAnalysisResult{
		TransactionID: transaction.TransactionID,
		RiskScore:     riskScore,
		RiskLevel:     riskLevel,
		Flags:         flags,
		Reasons:       []string{"automated_analysis"},
		Action:        action,
		Confidence:    0.85,
		Metadata: map[string]interface{}{
			"provider": "mock",
			"version":  "1.0",
		},
		Timestamp: time.Now(),
	}, nil
}

func (m *MockFraudDetectionService) ReportFraud(ctx context.Context, report *FraudReport) error {
	return nil
}

func (m *MockFraudDetectionService) GetRiskScore(ctx context.Context, userID int64) (*RiskScore, error) {
	return &RiskScore{
		UserID:        userID,
		CurrentScore:  25,
		BaselineScore: 20,
		TrendScore:    5,
		LastUpdated:   time.Now(),
		Factors: []RiskFactor{
			{
				Type:        "transaction_volume",
				Impact:      10,
				Weight:      0.3,
				Description: "Recent transaction volume within normal range",
			},
			{
				Type:        "device_consistency",
				Impact:      5,
				Weight:      0.2,
				Description: "Consistent device usage pattern",
			},
		},
	}, nil
}

func (m *MockFraudDetectionService) UpdateUserBehavior(ctx context.Context, userID int64, behavior *UserBehavior) error {
	return nil
}