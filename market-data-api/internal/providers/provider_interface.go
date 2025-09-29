package providers

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

// WebSocketProvider defines the interface for real-time data providers
type WebSocketProvider interface {
	Provider

	// WebSocket operations
	Connect(ctx context.Context) error
	Disconnect() error
	Subscribe(symbols []string, channels []string) error
	Unsubscribe(symbols []string, channels []string) error

	// Data streaming
	GetPriceStream() <-chan *PriceUpdate
	GetOrderBookStream() <-chan *OrderBookUpdate
	GetTradeStream() <-chan *TradeUpdate

	// Connection management
	IsConnected() bool
	Reconnect(ctx context.Context) error
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
	Symbol    string                 `json:"symbol"`
	Bids      []*models.OrderLevel   `json:"bids"`
	Asks      []*models.OrderLevel   `json:"asks"`
	Timestamp time.Time              `json:"timestamp"`
	Provider  string                 `json:"provider"`
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

// ProviderConfig represents configuration for a provider
type ProviderConfig struct {
	Name              string        `json:"name"`
	BaseURL           string        `json:"base_url"`
	APIKey            string        `json:"api_key,omitempty"`
	SecretKey         string        `json:"secret_key,omitempty"`
	Weight            float64       `json:"weight"`
	RateLimit         int           `json:"rate_limit"`
	Timeout           time.Duration `json:"timeout"`
	RetryAttempts     int           `json:"retry_attempts"`
	RetryDelay        time.Duration `json:"retry_delay"`
	Enabled           bool          `json:"enabled"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

// ProviderError represents an error from a data provider
type ProviderError struct {
	Provider string    `json:"provider"`
	Code     string    `json:"code"`
	Message  string    `json:"message"`
	Details  string    `json:"details,omitempty"`
	Retryable bool     `json:"retryable"`
	Timestamp time.Time `json:"timestamp"`
}

// Error implements the error interface
func (pe *ProviderError) Error() string {
	return pe.Provider + ": " + pe.Message
}

// IsRetryable returns whether the error is retryable
func (pe *ProviderError) IsRetryable() bool {
	return pe.retryable
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
	StatusHealthy   = "healthy"
	StatusDegraded  = "degraded"
	StatusDown      = "down"
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

// ProviderFactory defines the interface for creating providers
type ProviderFactory interface {
	CreateProvider(config *ProviderConfig) (Provider, error)
	GetSupportedProviders() []string
}

// ProviderManager manages multiple data providers
type ProviderManager struct {
	providers map[string]Provider
	weights   map[string]float64
	factory   ProviderFactory
}

// NewProviderManager creates a new provider manager
func NewProviderManager(factory ProviderFactory) *ProviderManager {
	return &ProviderManager{
		providers: make(map[string]Provider),
		weights:   make(map[string]float64),
		factory:   factory,
	}
}

// AddProvider adds a provider to the manager
func (pm *ProviderManager) AddProvider(name string, provider Provider, weight float64) {
	pm.providers[name] = provider
	pm.weights[name] = weight
}

// RemoveProvider removes a provider from the manager
func (pm *ProviderManager) RemoveProvider(name string) {
	delete(pm.providers, name)
	delete(pm.weights, name)
}

// GetProvider returns a provider by name
func (pm *ProviderManager) GetProvider(name string) (Provider, bool) {
	provider, exists := pm.providers[name]
	return provider, exists
}

// GetAllProviders returns all providers
func (pm *ProviderManager) GetAllProviders() map[string]Provider {
	return pm.providers
}

// GetHealthyProviders returns only healthy providers
func (pm *ProviderManager) GetHealthyProviders() map[string]Provider {
	healthy := make(map[string]Provider)
	for name, provider := range pm.providers {
		if provider.IsHealthy() {
			healthy[name] = provider
		}
	}
	return healthy
}

// GetProviderWeights returns provider weights
func (pm *ProviderManager) GetProviderWeights() map[string]float64 {
	return pm.weights
}

// UpdateProviderWeight updates the weight of a provider
func (pm *ProviderManager) UpdateProviderWeight(name string, weight float64) {
	if _, exists := pm.providers[name]; exists {
		pm.weights[name] = weight
	}
}

// GetProviderStatuses returns the status of all providers
func (pm *ProviderManager) GetProviderStatuses() map[string]*models.ProviderStatus {
	statuses := make(map[string]*models.ProviderStatus)
	for name, provider := range pm.providers {
		statuses[name] = provider.GetStatus()
	}
	return statuses
}

// HealthCheck performs health checks on all providers
func (pm *ProviderManager) HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error)
	for name, provider := range pm.providers {
		results[name] = provider.Ping(ctx)
	}
	return results
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

// ProviderMetrics represents metrics for a provider
type ProviderMetrics struct {
	Name             string        `json:"name"`
	RequestCount     int64         `json:"request_count"`
	SuccessCount     int64         `json:"success_count"`
	ErrorCount       int64         `json:"error_count"`
	SuccessRate      float64       `json:"success_rate"`
	AverageLatency   time.Duration `json:"average_latency"`
	LastRequest      time.Time     `json:"last_request"`
	RateLimitHits    int64         `json:"rate_limit_hits"`
	CircuitBreakerTrips int64      `json:"circuit_breaker_trips"`
}

// CalculateSuccessRate calculates the success rate
func (pm *ProviderMetrics) CalculateSuccessRate() {
	if pm.RequestCount > 0 {
		pm.SuccessRate = float64(pm.SuccessCount) / float64(pm.RequestCount)
	}
}

// ProviderClient provides a base implementation for HTTP-based providers
type ProviderClient struct {
	name         string
	baseURL      string
	timeout      time.Duration
	rateLimiter  RateLimiter
	circuitBreaker CircuitBreaker
	metrics      *ProviderMetrics
	status       *models.ProviderStatus
	weight       float64
}

// GetName returns the provider name
func (pc *ProviderClient) GetName() string {
	return pc.name
}

// GetWeight returns the provider weight
func (pc *ProviderClient) GetWeight() float64 {
	return pc.weight
}

// GetStatus returns the provider status
func (pc *ProviderClient) GetStatus() *models.ProviderStatus {
	return pc.status
}

// IsHealthy returns whether the provider is healthy
func (pc *ProviderClient) IsHealthy() bool {
	return pc.status != nil && pc.status.Status == StatusHealthy
}

// CheckRateLimit checks if the rate limit allows the request
func (pc *ProviderClient) CheckRateLimit() error {
	if pc.rateLimiter != nil && !pc.rateLimiter.Allow() {
		return NewProviderError(pc.name, ErrorCodeRateLimit, "Rate limit exceeded", true)
	}
	return nil
}

// UpdateMetrics updates provider metrics
func (pc *ProviderClient) UpdateMetrics(success bool, latency time.Duration) {
	if pc.metrics == nil {
		return
	}

	pc.metrics.RequestCount++
	pc.metrics.LastRequest = time.Now()

	if success {
		pc.metrics.SuccessCount++
	} else {
		pc.metrics.ErrorCount++
	}

	pc.metrics.CalculateSuccessRate()

	// Update average latency (simple moving average)
	if pc.metrics.AverageLatency == 0 {
		pc.metrics.AverageLatency = latency
	} else {
		pc.metrics.AverageLatency = (pc.metrics.AverageLatency + latency) / 2
	}
}

// UpdateStatus updates provider status
func (pc *ProviderClient) UpdateStatus(status string, latency time.Duration, errorCount int) {
	if pc.status == nil {
		pc.status = &models.ProviderStatus{
			Name: pc.name,
		}
	}

	pc.status.Status = status
	pc.status.Latency = latency
	pc.status.LastUpdate = time.Now()
	pc.status.ErrorCount = errorCount
	pc.status.Weight = pc.weight

	if pc.metrics != nil {
		pc.status.SuccessRate = pc.metrics.SuccessRate
		pc.status.ResponseTime = pc.metrics.AverageLatency
	}
}