package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig
	Redis      RedisConfig
	Providers  ProvidersConfig
	WebSocket  WebSocketConfig
	Aggregator AggregatorConfig
	Cache      CacheConfig
	Performance PerformanceConfig
	Environment string
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Port            int
	Environment     string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	URL      string
	DB       int
	Password string
	PoolSize int
	Timeout  time.Duration
}

// ProvidersConfig represents external providers configuration
type ProvidersConfig struct {
	CoinGecko CoinGeckoConfig
	Binance   BinanceConfig
	Coinbase  CoinbaseConfig
}

// CoinGeckoConfig represents CoinGecko API configuration
type CoinGeckoConfig struct {
	APIKey    string
	BaseURL   string
	RateLimit int
	Weight    float64
	Timeout   time.Duration
}

// BinanceConfig represents Binance API configuration
type BinanceConfig struct {
	APIKey    string
	SecretKey string
	BaseURL   string
	WSUrl     string
	Weight    float64
	Timeout   time.Duration
}

// CoinbaseConfig represents Coinbase API configuration
type CoinbaseConfig struct {
	APIKey    string
	Secret    string
	BaseURL   string
	Weight    float64
	Timeout   time.Duration
}

// WebSocketConfig represents WebSocket configuration
type WebSocketConfig struct {
	MaxConnections   int
	PingInterval     time.Duration
	PongTimeout      time.Duration
	MaxMessageSize   int64
	ReadBufferSize   int
	WriteBufferSize  int
	HandshakeTimeout time.Duration
}

// AggregatorConfig represents price aggregation configuration
type AggregatorConfig struct {
	OutlierThreshold     float64
	ConfidenceMinProviders int
	AggregationTimeout   time.Duration
	MinProvidersRequired int
	MaxRetryAttempts     int
	RetryDelay          time.Duration
}

// CacheConfig represents cache TTL configuration
type CacheConfig struct {
	PriceTTL     time.Duration
	StatsTTL     time.Duration
	HistoryTTL   time.Duration
	OrderBookTTL time.Duration
	ProviderTTL  time.Duration
}

// PerformanceConfig represents performance tuning configuration
type PerformanceConfig struct {
	WorkerPoolSize  int
	BatchSize       int
	UpdateInterval  time.Duration
	MaxConcurrency  int
	ChannelBuffer   int
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Server: ServerConfig{
			Port:            getEnvAsInt("SERVER_PORT", 8004),
			Environment:     getEnv("SERVER_ENV", "development"),
			ReadTimeout:     getEnvAsDuration("SERVER_READ_TIMEOUT", "30s"),
			WriteTimeout:    getEnvAsDuration("SERVER_WRITE_TIMEOUT", "30s"),
			IdleTimeout:     getEnvAsDuration("SERVER_IDLE_TIMEOUT", "60s"),
			ShutdownTimeout: getEnvAsDuration("SERVER_SHUTDOWN_TIMEOUT", "30s"),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379"),
			DB:       getEnvAsInt("REDIS_DB", 0),
			Password: getEnv("REDIS_PASSWORD", ""),
			PoolSize: getEnvAsInt("REDIS_POOL_SIZE", 10),
			Timeout:  getEnvAsDuration("REDIS_TIMEOUT", "5s"),
		},
		Providers: ProvidersConfig{
			CoinGecko: CoinGeckoConfig{
				APIKey:    getEnv("COINGECKO_API_KEY", ""),
				BaseURL:   getEnv("COINGECKO_BASE_URL", "https://api.coingecko.com/api/v3"),
				RateLimit: getEnvAsInt("COINGECKO_RATE_LIMIT", 50),
				Weight:    getEnvAsFloat("COINGECKO_WEIGHT", 0.33),
				Timeout:   getEnvAsDuration("COINGECKO_TIMEOUT", "10s"),
			},
			Binance: BinanceConfig{
				APIKey:    getEnv("BINANCE_API_KEY", ""),
				SecretKey: getEnv("BINANCE_SECRET_KEY", ""),
				BaseURL:   getEnv("BINANCE_BASE_URL", "https://api.binance.com"),
				WSUrl:     getEnv("BINANCE_WS_URL", "wss://stream.binance.com:9443"),
				Weight:    getEnvAsFloat("BINANCE_WEIGHT", 0.34),
				Timeout:   getEnvAsDuration("BINANCE_TIMEOUT", "10s"),
			},
			Coinbase: CoinbaseConfig{
				APIKey:  getEnv("COINBASE_API_KEY", ""),
				Secret:  getEnv("COINBASE_SECRET", ""),
				BaseURL: getEnv("COINBASE_BASE_URL", "https://api.coinbase.com"),
				Weight:  getEnvAsFloat("COINBASE_WEIGHT", 0.33),
				Timeout: getEnvAsDuration("COINBASE_TIMEOUT", "10s"),
			},
		},
		WebSocket: WebSocketConfig{
			MaxConnections:   getEnvAsInt("WS_MAX_CONNECTIONS", 1000),
			PingInterval:     getEnvAsDuration("WS_PING_INTERVAL", "30s"),
			PongTimeout:      getEnvAsDuration("WS_PONG_TIMEOUT", "60s"),
			MaxMessageSize:   getEnvAsInt64("WS_MAX_MESSAGE_SIZE", 512000),
			ReadBufferSize:   getEnvAsInt("WS_READ_BUFFER_SIZE", 1024),
			WriteBufferSize:  getEnvAsInt("WS_WRITE_BUFFER_SIZE", 1024),
			HandshakeTimeout: getEnvAsDuration("WS_HANDSHAKE_TIMEOUT", "10s"),
		},
		Aggregator: AggregatorConfig{
			OutlierThreshold:     getEnvAsFloat("OUTLIER_THRESHOLD", 2.0),
			ConfidenceMinProviders: getEnvAsInt("CONFIDENCE_MIN_PROVIDERS", 2),
			AggregationTimeout:   getEnvAsDuration("AGGREGATION_TIMEOUT", "5s"),
			MinProvidersRequired: getEnvAsInt("MIN_PROVIDERS_REQUIRED", 2),
			MaxRetryAttempts:     getEnvAsInt("MAX_RETRY_ATTEMPTS", 3),
			RetryDelay:          getEnvAsDuration("RETRY_DELAY", "1s"),
		},
		Cache: CacheConfig{
			PriceTTL:     getEnvAsDuration("PRICE_CACHE_TTL", "30s"),
			StatsTTL:     getEnvAsDuration("STATS_CACHE_TTL", "5m"),
			HistoryTTL:   getEnvAsDuration("HISTORY_CACHE_TTL", "1h"),
			OrderBookTTL: getEnvAsDuration("ORDERBOOK_CACHE_TTL", "5s"),
			ProviderTTL:  getEnvAsDuration("PROVIDER_CACHE_TTL", "1m"),
		},
		Performance: PerformanceConfig{
			WorkerPoolSize: getEnvAsInt("WORKER_POOL_SIZE", 20),
			BatchSize:      getEnvAsInt("BATCH_SIZE", 100),
			UpdateInterval: getEnvAsDuration("UPDATE_INTERVAL", "5s"),
			MaxConcurrency: getEnvAsInt("MAX_CONCURRENCY", 50),
			ChannelBuffer:  getEnvAsInt("CHANNEL_BUFFER", 1000),
		},
	}
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if int64Value, err := strconv.ParseInt(value, 10, 64); err == nil {
			return int64Value
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	if duration, err := time.ParseDuration(defaultValue); err == nil {
		return duration
	}
	return time.Second * 30 // Fallback
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsTest returns true if running in test environment
func (c *Config) IsTest() bool {
	return c.Environment == "test"
}