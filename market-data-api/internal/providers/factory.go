package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"market-data-api/internal/providers/binance"
	"market-data-api/internal/providers/coingecko"
	"market-data-api/internal/providers/coinbase"
)

// Factory implements the ProviderFactory interface
type Factory struct {
	supportedProviders map[string]func(*ProviderConfig) (Provider, error)
}

// NewFactory creates a new provider factory
func NewFactory() *Factory {
	factory := &Factory{
		supportedProviders: make(map[string]func(*ProviderConfig) (Provider, error)),
	}

	// Register supported providers
	factory.registerProviders()
	return factory
}

// registerProviders registers all supported providers
func (f *Factory) registerProviders() {
	f.supportedProviders["coingecko"] = f.createCoinGeckoProvider
	f.supportedProviders["binance"] = f.createBinanceProvider
	f.supportedProviders["coinbase"] = f.createCoinbaseProvider
}

// CreateProvider creates a provider instance based on configuration
func (f *Factory) CreateProvider(config *ProviderConfig) (Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("provider config cannot be nil")
	}

	if config.Name == "" {
		return nil, fmt.Errorf("provider name is required")
	}

	providerName := strings.ToLower(config.Name)
	createFunc, exists := f.supportedProviders[providerName]
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", config.Name)
	}

	// Set defaults if not specified
	f.setDefaults(config)

	// Validate configuration
	if err := f.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration for provider %s: %w", config.Name, err)
	}

	return createFunc(config)
}

// GetSupportedProviders returns a list of supported provider names
func (f *Factory) GetSupportedProviders() []string {
	providers := make([]string, 0, len(f.supportedProviders))
	for name := range f.supportedProviders {
		providers = append(providers, name)
	}
	return providers
}

// setDefaults sets default values for provider configuration
func (f *Factory) setDefaults(config *ProviderConfig) {
	if config.Weight == 0 {
		config.Weight = 1.0
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}

	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}

	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}

	// Set provider-specific defaults
	switch strings.ToLower(config.Name) {
	case "coingecko":
		if config.BaseURL == "" {
			config.BaseURL = "https://api.coingecko.com/api/v3"
		}
		if config.RateLimit == 0 {
			config.RateLimit = 10 // 10-30 calls/minute for free tier
		}
	case "binance":
		if config.BaseURL == "" {
			config.BaseURL = "https://api.binance.com"
		}
		if config.RateLimit == 0 {
			config.RateLimit = 1200 // 1200 requests per minute
		}
	case "coinbase":
		if config.BaseURL == "" {
			config.BaseURL = "https://api.exchange.coinbase.com"
		}
		if config.RateLimit == 0 {
			config.RateLimit = 10 // 10 requests per second
		}
	}
}

// validateConfig validates provider configuration
func (f *Factory) validateConfig(config *ProviderConfig) error {
	if config.Weight < 0 || config.Weight > 10 {
		return fmt.Errorf("weight must be between 0 and 10")
	}

	if config.Timeout < time.Second || config.Timeout > time.Minute {
		return fmt.Errorf("timeout must be between 1s and 1m")
	}

	if config.RetryAttempts < 0 || config.RetryAttempts > 10 {
		return fmt.Errorf("retry attempts must be between 0 and 10")
	}

	if config.RetryDelay < 0 || config.RetryDelay > 30*time.Second {
		return fmt.Errorf("retry delay must be between 0 and 30s")
	}

	if config.RateLimit <= 0 {
		return fmt.Errorf("rate limit must be positive")
	}

	return nil
}

// createCoinGeckoProvider creates a CoinGecko provider instance
func (f *Factory) createCoinGeckoProvider(config *ProviderConfig) (Provider, error) {
	clientConfig := &coingecko.Config{
		APIKey:    config.APIKey,
		BaseURL:   config.BaseURL,
		Timeout:   config.Timeout,
		RateLimit: config.RateLimit,
		Weight:    config.Weight,
	}

	client := coingecko.NewClient(clientConfig)
	return client, nil
}

// createBinanceProvider creates a Binance provider instance
func (f *Factory) createBinanceProvider(config *ProviderConfig) (Provider, error) {
	clientConfig := &binance.Config{
		APIKey:    config.APIKey,
		SecretKey: config.SecretKey,
		BaseURL:   config.BaseURL,
		Timeout:   config.Timeout,
		RateLimit: config.RateLimit,
		Weight:    config.Weight,
	}

	client := binance.NewClient(clientConfig)
	return client, nil
}

// createCoinbaseProvider creates a Coinbase provider instance
func (f *Factory) createCoinbaseProvider(config *ProviderConfig) (Provider, error) {
	clientConfig := &coinbase.Config{
		APIKey:     config.APIKey,
		Secret:     config.SecretKey,
		Passphrase: "", // Would need to be provided in extended config
		Timeout:    config.Timeout,
		RateLimit:  config.RateLimit,
		Weight:     config.Weight,
	}

	client := coinbase.NewClient(clientConfig)
	return client, nil
}

// CreateProviderManager creates a provider manager with multiple providers
func (f *Factory) CreateProviderManager(configs []*ProviderConfig) (*ProviderManager, error) {
	manager := NewProviderManager(f)

	for _, config := range configs {
		if !config.Enabled {
			continue
		}

		provider, err := f.CreateProvider(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %s: %w", config.Name, err)
		}

		manager.AddProvider(config.Name, provider, config.Weight)
	}

	return manager, nil
}

// GetDefaultConfigs returns default configurations for all supported providers
func (f *Factory) GetDefaultConfigs() []*ProviderConfig {
	return []*ProviderConfig{
		{
			Name:                "coingecko",
			BaseURL:             "https://api.coingecko.com/api/v3",
			Weight:              1.0,
			RateLimit:           10,
			Timeout:             10 * time.Second,
			RetryAttempts:       3,
			RetryDelay:          time.Second,
			Enabled:             true,
			HealthCheckInterval: 30 * time.Second,
		},
		{
			Name:                "binance",
			BaseURL:             "https://api.binance.com",
			Weight:              1.5,
			RateLimit:           1200,
			Timeout:             10 * time.Second,
			RetryAttempts:       3,
			RetryDelay:          time.Second,
			Enabled:             true,
			HealthCheckInterval: 30 * time.Second,
		},
		{
			Name:                "coinbase",
			BaseURL:             "https://api.exchange.coinbase.com",
			Weight:              1.2,
			RateLimit:           10,
			Timeout:             10 * time.Second,
			RetryAttempts:       3,
			RetryDelay:          time.Second,
			Enabled:             true,
			HealthCheckInterval: 30 * time.Second,
		},
	}
}

// ValidateProviderName checks if a provider name is supported
func (f *Factory) ValidateProviderName(name string) error {
	providerName := strings.ToLower(name)
	if _, exists := f.supportedProviders[providerName]; !exists {
		return fmt.Errorf("unsupported provider: %s. Supported providers: %v",
			name, f.GetSupportedProviders())
	}
	return nil
}

// GetProviderInfo returns information about a specific provider
func (f *Factory) GetProviderInfo(name string) (*ProviderInfo, error) {
	providerName := strings.ToLower(name)

	switch providerName {
	case "coingecko":
		return &ProviderInfo{
			Name:        "CoinGecko",
			Description: "Comprehensive cryptocurrency data provider with extensive market information",
			Features: []string{
				"Current prices", "Historical data", "Market statistics",
				"Coin information", "Exchange data", "Global market data",
			},
			RateLimits:      "10-30 calls/minute (free tier), 500-10000 calls/minute (paid)",
			RequiredCredentials: []string{"api_key (optional for higher limits)"},
			SupportedSymbols:    "8000+ cryptocurrencies",
			WebSocketSupport:    false,
		}, nil

	case "binance":
		return &ProviderInfo{
			Name:        "Binance",
			Description: "World's largest cryptocurrency exchange with high-frequency data",
			Features: []string{
				"Current prices", "Historical klines", "Order book data",
				"Market statistics", "Trading data", "Real-time WebSocket streams",
			},
			RateLimits:          "1200 requests/minute, 100 orders/10s",
			RequiredCredentials: []string{"api_key", "secret_key"},
			SupportedSymbols:    "1000+ trading pairs",
			WebSocketSupport:    true,
		}, nil

	case "coinbase":
		return &ProviderInfo{
			Name:        "Coinbase Pro",
			Description: "Professional trading platform with institutional-grade data",
			Features: []string{
				"Current prices", "Historical candles", "Order book data",
				"Market statistics", "Trade data", "Real-time WebSocket feeds",
			},
			RateLimits:          "10 requests/second",
			RequiredCredentials: []string{"api_key", "secret", "passphrase"},
			SupportedSymbols:    "200+ trading pairs",
			WebSocketSupport:    true,
		}, nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

// ProviderInfo contains information about a data provider
type ProviderInfo struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Features            []string `json:"features"`
	RateLimits          string   `json:"rate_limits"`
	RequiredCredentials []string `json:"required_credentials"`
	SupportedSymbols    string   `json:"supported_symbols"`
	WebSocketSupport    bool     `json:"websocket_support"`
}

// GetAllProviderInfo returns information about all supported providers
func (f *Factory) GetAllProviderInfo() ([]*ProviderInfo, error) {
	var infos []*ProviderInfo

	for _, providerName := range f.GetSupportedProviders() {
		info, err := f.GetProviderInfo(providerName)
		if err != nil {
			continue // Skip providers that fail to get info
		}
		infos = append(infos, info)
	}

	return infos, nil
}

// TestProvider tests if a provider can be created and initialized properly
func (f *Factory) TestProvider(config *ProviderConfig) error {
	provider, err := f.CreateProvider(config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Test basic connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := provider.Ping(ctx); err != nil {
		return fmt.Errorf("provider ping failed: %w", err)
	}

	return nil
}

// Default factory instance
var defaultFactory = NewFactory()

// GetDefaultFactory returns the default factory instance
func GetDefaultFactory() *Factory {
	return defaultFactory
}

// CreateProviderFromConfig is a convenience function to create a provider from config
func CreateProviderFromConfig(config *ProviderConfig) (Provider, error) {
	return defaultFactory.CreateProvider(config)
}

// GetSupportedProviderNames returns supported provider names using default factory
func GetSupportedProviderNames() []string {
	return defaultFactory.GetSupportedProviders()
}