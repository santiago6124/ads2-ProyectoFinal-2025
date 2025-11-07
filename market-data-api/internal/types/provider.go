package types

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"market-data-api/internal/models"
)

// Provider defines the interface for cryptocurrency data providers
type Provider interface {
	// Basic price operations
	GetPrice(ctx context.Context, symbol string) (*models.Price, error)
	GetPrices(ctx context.Context, symbols []string) (map[string]*models.Price, error)

	// Historical data
	GetHistoricalData(ctx context.Context, symbol, interval string, from, to time.Time, limit int) ([]*models.Candle, error)

	// Market data
	GetMarketData(ctx context.Context, symbol string) (*models.MarketData, error)

	// Order book (if supported)
	GetOrderBook(ctx context.Context, symbol string, depth int) (*models.OrderBook, error)

	// Provider information
	GetName() string
	GetWeight() float64
	GetStatus() *models.ProviderStatus
	IsHealthy() bool

	// Rate limiting and health
	CheckRateLimit() error
	Ping(ctx context.Context) error
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow() bool
	Wait(ctx context.Context) error
	Limit() int
	Remaining() int
	Reset() time.Time
}

// CircuitBreaker defines the interface for circuit breaking
type CircuitBreaker interface {
	Call(func() error) error
	State() string
	IsOpen() bool
	IsClosed() bool
	IsHalfOpen() bool
}

// ProviderClient provides a base implementation for HTTP-based providers
type ProviderClient struct {
	Name           string
	BaseURL        string
	Timeout        time.Duration
	RateLimiter    RateLimiter
	CircuitBreaker CircuitBreaker
	Metrics        *ProviderMetrics
	Status         *models.ProviderStatus
	Weight         float64
}

// ProviderMetrics represents metrics for a provider
type ProviderMetrics struct {
	Name                 string        `json:"name"`
	RequestCount         int64         `json:"request_count"`
	SuccessCount         int64         `json:"success_count"`
	ErrorCount           int64         `json:"error_count"`
	SuccessRate          float64       `json:"success_rate"`
	AverageLatency       time.Duration `json:"average_latency"`
	LastRequest          time.Time     `json:"last_request"`
	RateLimitHits        int64         `json:"rate_limit_hits"`
	CircuitBreakerTrips  int64         `json:"circuit_breaker_trips"`
}

// CalculateSuccessRate calculates the success rate
func (pm *ProviderMetrics) CalculateSuccessRate() {
	if pm.RequestCount > 0 {
		pm.SuccessRate = float64(pm.SuccessCount) / float64(pm.RequestCount)
	}
}

// GetName returns the provider name
func (pc *ProviderClient) GetName() string {
	return pc.Name
}

// GetWeight returns the provider weight
func (pc *ProviderClient) GetWeight() float64 {
	return pc.Weight
}

// GetStatus returns the provider status
func (pc *ProviderClient) GetStatus() *models.ProviderStatus {
	return pc.Status
}

// IsHealthy returns whether the provider is healthy
func (pc *ProviderClient) IsHealthy() bool {
	return pc.Status != nil && pc.Status.Status == "healthy"
}

// CheckRateLimit checks if the rate limit allows the request
func (pc *ProviderClient) CheckRateLimit() error {
	if pc.RateLimiter != nil && !pc.RateLimiter.Allow() {
		return &ProviderError{
			Provider:  pc.Name,
			Code:      "RATE_LIMIT_EXCEEDED",
			Message:   "Rate limit exceeded",
			Retryable: true,
			Timestamp: time.Now(),
		}
	}
	return nil
}

// UpdateMetrics updates provider metrics
func (pc *ProviderClient) UpdateMetrics(success bool, latency time.Duration) {
	if pc.Metrics == nil {
		return
	}

	pc.Metrics.RequestCount++
	pc.Metrics.LastRequest = time.Now()

	if success {
		pc.Metrics.SuccessCount++
	} else {
		pc.Metrics.ErrorCount++
	}

	pc.Metrics.CalculateSuccessRate()

	// Update average latency (simple moving average)
	if pc.Metrics.AverageLatency == 0 {
		pc.Metrics.AverageLatency = latency
	} else {
		pc.Metrics.AverageLatency = (pc.Metrics.AverageLatency + latency) / 2
	}
}

// UpdateStatus updates provider status
func (pc *ProviderClient) UpdateStatus(status string, latency time.Duration, errorCount int) {
	if pc.Status == nil {
		pc.Status = &models.ProviderStatus{
			Name: pc.Name,
		}
	}

	pc.Status.Status = status
	pc.Status.Latency = latency
	pc.Status.LastUpdate = time.Now()
	pc.Status.ErrorCount = errorCount
	pc.Status.Weight = pc.Weight

	if pc.Metrics != nil {
		pc.Status.SuccessRate = pc.Metrics.SuccessRate
		pc.Status.ResponseTime = pc.Metrics.AverageLatency
	}
}

// ProviderError represents an error from a data provider
type ProviderError struct {
	Provider  string    `json:"provider"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Retryable bool      `json:"retryable"`
	Timestamp time.Time `json:"timestamp"`
}

// Error implements the error interface
func (pe *ProviderError) Error() string {
	return pe.Provider + ": " + pe.Message
}

// IsRetryable returns whether the error is retryable
func (pe *ProviderError) IsRetryable() bool {
	return pe.Retryable
}

// PriceUpdate represents a real-time price update
type PriceUpdate struct {
	Symbol    string          `json:"symbol"`
	Price     decimal.Decimal `json:"price"`
	Volume    decimal.Decimal `json:"volume,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Provider  string          `json:"provider"`
	Change24h decimal.Decimal `json:"change_24h,omitempty"`
}

// OrderBookUpdate represents a real-time order book update
type OrderBookUpdate struct {
	Symbol    string               `json:"symbol"`
	Bids      []*models.OrderLevel `json:"bids"`
	Asks      []*models.OrderLevel `json:"asks"`
	Timestamp time.Time            `json:"timestamp"`
	Provider  string               `json:"provider"`
}

// TradeUpdate represents a real-time trade update
type TradeUpdate struct {
	Symbol    string          `json:"symbol"`
	Price     decimal.Decimal `json:"price"`
	Quantity  decimal.Decimal `json:"quantity"`
	Side      string          `json:"side"` // "buy" or "sell"
	Timestamp time.Time       `json:"timestamp"`
	TradeID   string          `json:"trade_id,omitempty"`
	Provider  string          `json:"provider"`
}

// Common error codes
const (
	ErrorCodeRateLimit     = "RATE_LIMIT_EXCEEDED"
	ErrorCodeUnauthorized  = "UNAUTHORIZED"
	ErrorCodeNotFound      = "NOT_FOUND"
	ErrorCodeBadRequest    = "BAD_REQUEST"
	ErrorCodeServerError   = "SERVER_ERROR"
	ErrorCodeTimeout       = "TIMEOUT"
	ErrorCodeNetworkError  = "NETWORK_ERROR"
	ErrorCodeInvalidSymbol = "INVALID_SYMBOL"
	ErrorCodeNoData        = "NO_DATA"
	ErrorCodeMaintenance   = "MAINTENANCE"
)

// Provider status constants
const (
	StatusHealthy     = "healthy"
	StatusDegraded    = "degraded"
	StatusDown        = "down"
	StatusMaintenance = "maintenance"
)

// NewProviderError creates a new provider error
func NewProviderError(provider, code, message string, retryable bool) *ProviderError {
	return &ProviderError{
		Provider:  provider,
		Code:      code,
		Message:   message,
		Retryable: retryable,
		Timestamp: time.Now(),
	}
}
