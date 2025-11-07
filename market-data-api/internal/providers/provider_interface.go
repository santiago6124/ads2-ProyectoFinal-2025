package providers

import (
	"context"
	"time"

	"market-data-api/internal/models"
	"market-data-api/internal/types"
)

// Re-export types from the types package to maintain backward compatibility
type Provider = types.Provider
type ProviderClient = types.ProviderClient
type ProviderMetrics = types.ProviderMetrics
type ProviderError = types.ProviderError
type PriceUpdate = types.PriceUpdate
type OrderBookUpdate = types.OrderBookUpdate
type TradeUpdate = types.TradeUpdate
type RateLimiter = types.RateLimiter
type CircuitBreaker = types.CircuitBreaker

// NewProviderError creates a new provider error (wrapper around types)
func NewProviderError(provider, code, message string, retryable bool) *types.ProviderError {
	return &types.ProviderError{
		Provider:  provider,
		Code:      code,
		Message:   message,
		Retryable: retryable,
		Timestamp: time.Now(),
	}
}

// Provider interface is now aliased from types package
// Original definition kept below for reference only
/*
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
*/

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

// PriceUpdate, OrderBookUpdate, and TradeUpdate are now re-exported from types package (see above)

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

// ProviderError is now re-exported from types package (see above)

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

// NewProviderError is defined at the top of this file

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

// RateLimiter, CircuitBreaker, ProviderMetrics, and ProviderClient are now re-exported from types package (see above)
// All their methods are implemented in the types package