package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"orders-api/internal/clients"
	"orders-api/internal/concurrent"
	"orders-api/internal/messaging"
	"orders-api/internal/middleware"
	"orders-api/internal/repository"
	"orders-api/internal/services"
)

type Config struct {
	Server     *ServerConfig     `json:"server"`
	Database   *DatabaseConfig   `json:"database"`
	Auth       *AuthConfig       `json:"auth"`
	Logging    *LoggingConfig    `json:"logging"`
	Messaging  *MessagingConfig  `json:"messaging"`
	Clients    *ClientsConfig    `json:"clients"`
	Execution  *ExecutionConfig  `json:"execution"`
	Fee        *FeeConfig        `json:"fee"`
	Worker     *WorkerConfig     `json:"worker"`
}

type ServerConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	Debug           bool          `json:"debug"`
	CORSEnabled     bool          `json:"cors_enabled"`
	AllowedOrigins  []string      `json:"allowed_origins"`
}

type DatabaseConfig struct {
	URI                string        `json:"uri"`
	Database           string        `json:"database"`
	Collection         string        `json:"collection"`
	MaxPoolSize        uint64        `json:"max_pool_size"`
	MinPoolSize        uint64        `json:"min_pool_size"`
	MaxConnIdleTime    time.Duration `json:"max_conn_idle_time"`
	ConnectionTimeout  time.Duration `json:"connection_timeout"`
	SocketTimeout      time.Duration `json:"socket_timeout"`
	ServerSelectionTimeout time.Duration `json:"server_selection_timeout"`
}

type AuthConfig struct {
	SecretKey       string   `json:"secret_key"`
	Issuer          string   `json:"issuer"`
	Audience        string   `json:"audience"`
	TokenExpiry     time.Duration `json:"token_expiry"`
	SkipPaths       []string `json:"skip_paths"`
	PublicEndpoints []string `json:"public_endpoints"`
}

type LoggingConfig struct {
	Level           string   `json:"level"`
	Format          string   `json:"format"`
	Output          string   `json:"output"`
	SkipPaths       []string `json:"skip_paths"`
	LogBody         bool     `json:"log_body"`
	LogHeaders      bool     `json:"log_headers"`
	MaxBodySize     int64    `json:"max_body_size"`
	TimestampFormat string   `json:"timestamp_format"`
}

type MessagingConfig struct {
	URL                string        `json:"url"`
	ExchangeName       string        `json:"exchange_name"`
	DeadLetterExchange string        `json:"dead_letter_exchange"`
	QueuePrefix        string        `json:"queue_prefix"`
	ConsumerTag        string        `json:"consumer_tag"`
	MaxRetries         int           `json:"max_retries"`
	RetryDelay         time.Duration `json:"retry_delay"`
	MessageTTL         time.Duration `json:"message_ttl"`
	Persistent         bool          `json:"persistent"`
	PrefetchCount      int           `json:"prefetch_count"`
	WorkerCount        int           `json:"worker_count"`
	AutoAck            bool          `json:"auto_ack"`
}

type ClientsConfig struct {
	UserAPI   *ClientConfig `json:"user_api"`
	WalletAPI *ClientConfig `json:"wallet_api"`
	MarketAPI *ClientConfig `json:"market_api"`
}

type ClientConfig struct {
	BaseURL string        `json:"base_url"`
	APIKey  string        `json:"api_key"`
	Timeout time.Duration `json:"timeout"`
}

type ExecutionConfig struct {
	MaxWorkers       int             `json:"max_workers"`
	QueueSize        int             `json:"queue_size"`
	ExecutionTimeout time.Duration   `json:"execution_timeout"`
	MaxSlippage      decimal.Decimal `json:"max_slippage"`
	SimulateLatency  bool            `json:"simulate_latency"`
	MinExecutionTime time.Duration   `json:"min_execution_time"`
	MaxExecutionTime time.Duration   `json:"max_execution_time"`
}

type FeeConfig struct {
	BaseFeePercentage decimal.Decimal            `json:"base_fee_percentage"`
	MakerFee          decimal.Decimal            `json:"maker_fee"`
	TakerFee          decimal.Decimal            `json:"taker_fee"`
	MinimumFee        decimal.Decimal            `json:"minimum_fee"`
	MaximumFee        decimal.Decimal            `json:"maximum_fee"`
	VIPDiscounts      map[string]decimal.Decimal `json:"vip_discounts"`
}

type WorkerConfig struct {
	PoolSize    int           `json:"pool_size"`
	QueueSize   int           `json:"queue_size"`
	Timeout     time.Duration `json:"timeout"`
	MaxRetries  int           `json:"max_retries"`
	RetryDelay  time.Duration `json:"retry_delay"`
}

func LoadConfig() (*Config, error) {
	config := &Config{
		Server:    loadServerConfig(),
		Database:  loadDatabaseConfig(),
		Auth:      loadAuthConfig(),
		Logging:   loadLoggingConfig(),
		Messaging: loadMessagingConfig(),
		Clients:   loadClientsConfig(),
		Execution: loadExecutionConfig(),
		Fee:       loadFeeConfig(),
		Worker:    loadWorkerConfig(),
	}

	return config, nil
}

func loadServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:            getEnv("SERVER_HOST", "0.0.0.0"),
		Port:            getEnvAsInt("SERVER_PORT", 8080),
		ReadTimeout:     getEnvAsDuration("SERVER_READ_TIMEOUT", 30*time.Second),
		WriteTimeout:    getEnvAsDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:     getEnvAsDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
		ShutdownTimeout: getEnvAsDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		Debug:           getEnvAsBool("SERVER_DEBUG", false),
		CORSEnabled:     getEnvAsBool("SERVER_CORS_ENABLED", true),
		AllowedOrigins: getEnvAsSlice("SERVER_ALLOWED_ORIGINS", []string{
			"http://localhost:3000",
			"http://localhost:8080",
		}),
	}
}

func loadDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		URI:                    getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		Database:               getEnv("MONGODB_DATABASE", "cryptosim_orders"),
		Collection:             getEnv("MONGODB_COLLECTION", "orders"),
		MaxPoolSize:            getEnvAsUint64("MONGODB_MAX_POOL_SIZE", 100),
		MinPoolSize:            getEnvAsUint64("MONGODB_MIN_POOL_SIZE", 5),
		MaxConnIdleTime:        getEnvAsDuration("MONGODB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		ConnectionTimeout:      getEnvAsDuration("MONGODB_CONNECTION_TIMEOUT", 10*time.Second),
		SocketTimeout:          getEnvAsDuration("MONGODB_SOCKET_TIMEOUT", 30*time.Second),
		ServerSelectionTimeout: getEnvAsDuration("MONGODB_SERVER_SELECTION_TIMEOUT", 10*time.Second),
	}
}

func loadAuthConfig() *AuthConfig {
	return &AuthConfig{
		SecretKey:   getEnv("JWT_SECRET_KEY", "your-super-secret-key-change-this-in-production"),
		Issuer:      getEnv("JWT_ISSUER", "orders-api"),
		Audience:    getEnv("JWT_AUDIENCE", "cryptosim"),
		TokenExpiry: getEnvAsDuration("JWT_TOKEN_EXPIRY", 24*time.Hour),
		SkipPaths: getEnvAsSlice("AUTH_SKIP_PATHS", []string{
			"/health",
			"/health/live",
			"/health/ready",
			"/metrics",
		}),
		PublicEndpoints: getEnvAsSlice("AUTH_PUBLIC_ENDPOINTS", []string{
			"/api/v1/public/health",
		}),
	}
}

func loadLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:           getEnv("LOG_LEVEL", "info"),
		Format:          getEnv("LOG_FORMAT", "json"),
		Output:          getEnv("LOG_OUTPUT", "stdout"),
		LogBody:         getEnvAsBool("LOG_BODY", false),
		LogHeaders:      getEnvAsBool("LOG_HEADERS", false),
		MaxBodySize:     getEnvAsInt64("LOG_MAX_BODY_SIZE", 1024),
		TimestampFormat: getEnv("LOG_TIMESTAMP_FORMAT", time.RFC3339),
		SkipPaths: getEnvAsSlice("LOG_SKIP_PATHS", []string{
			"/health",
			"/health/live",
			"/health/ready",
			"/metrics",
		}),
	}
}

func loadMessagingConfig() *MessagingConfig {
	return &MessagingConfig{
		URL:                getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		ExchangeName:       getEnv("RABBITMQ_EXCHANGE", "orders"),
		DeadLetterExchange: getEnv("RABBITMQ_DLX", "orders.dlx"),
		QueuePrefix:        getEnv("RABBITMQ_QUEUE_PREFIX", "orders"),
		ConsumerTag:        getEnv("RABBITMQ_CONSUMER_TAG", "orders-consumer"),
		MaxRetries:         getEnvAsInt("RABBITMQ_MAX_RETRIES", 3),
		RetryDelay:         getEnvAsDuration("RABBITMQ_RETRY_DELAY", 5*time.Second),
		MessageTTL:         getEnvAsDuration("RABBITMQ_MESSAGE_TTL", 24*time.Hour),
		Persistent:         getEnvAsBool("RABBITMQ_PERSISTENT", true),
		PrefetchCount:      getEnvAsInt("RABBITMQ_PREFETCH_COUNT", 10),
		WorkerCount:        getEnvAsInt("RABBITMQ_WORKER_COUNT", 5),
		AutoAck:            getEnvAsBool("RABBITMQ_AUTO_ACK", false),
	}
}

func loadClientsConfig() *ClientsConfig {
	return &ClientsConfig{
		UserAPI: &ClientConfig{
			BaseURL: getEnv("USER_API_BASE_URL", "http://localhost:8081"),
			APIKey:  getEnv("USER_API_KEY", "user-api-key"),
			Timeout: getEnvAsDuration("USER_API_TIMEOUT", 10*time.Second),
		},
		WalletAPI: &ClientConfig{
			BaseURL: getEnv("WALLET_API_BASE_URL", "http://localhost:8082"),
			APIKey:  getEnv("WALLET_API_KEY", "wallet-api-key"),
			Timeout: getEnvAsDuration("WALLET_API_TIMEOUT", 15*time.Second),
		},
		MarketAPI: &ClientConfig{
			BaseURL: getEnv("MARKET_API_BASE_URL", "http://localhost:8083"),
			APIKey:  getEnv("MARKET_API_KEY", "market-api-key"),
			Timeout: getEnvAsDuration("MARKET_API_TIMEOUT", 10*time.Second),
		},
	}
}

func loadExecutionConfig() *ExecutionConfig {
	return &ExecutionConfig{
		MaxWorkers:       getEnvAsInt("EXECUTION_MAX_WORKERS", 10),
		QueueSize:        getEnvAsInt("EXECUTION_QUEUE_SIZE", 100),
		ExecutionTimeout: getEnvAsDuration("EXECUTION_TIMEOUT", 30*time.Second),
		MaxSlippage:      getEnvAsDecimal("EXECUTION_MAX_SLIPPAGE", decimal.NewFromFloat(0.05)),
		SimulateLatency:  getEnvAsBool("EXECUTION_SIMULATE_LATENCY", true),
		MinExecutionTime: getEnvAsDuration("EXECUTION_MIN_TIME", 100*time.Millisecond),
		MaxExecutionTime: getEnvAsDuration("EXECUTION_MAX_TIME", 2*time.Second),
	}
}

func loadFeeConfig() *FeeConfig {
	vipDiscounts := make(map[string]decimal.Decimal)
	vipDiscounts["bronze"] = getEnvAsDecimal("FEE_VIP_BRONZE", decimal.NewFromFloat(0.05))
	vipDiscounts["silver"] = getEnvAsDecimal("FEE_VIP_SILVER", decimal.NewFromFloat(0.10))
	vipDiscounts["gold"] = getEnvAsDecimal("FEE_VIP_GOLD", decimal.NewFromFloat(0.15))
	vipDiscounts["platinum"] = getEnvAsDecimal("FEE_VIP_PLATINUM", decimal.NewFromFloat(0.25))

	return &FeeConfig{
		BaseFeePercentage: getEnvAsDecimal("FEE_BASE_PERCENTAGE", decimal.NewFromFloat(0.001)),
		MakerFee:          getEnvAsDecimal("FEE_MAKER", decimal.NewFromFloat(0.0008)),
		TakerFee:          getEnvAsDecimal("FEE_TAKER", decimal.NewFromFloat(0.0012)),
		MinimumFee:        getEnvAsDecimal("FEE_MINIMUM", decimal.NewFromFloat(0.01)),
		MaximumFee:        getEnvAsDecimal("FEE_MAXIMUM", decimal.NewFromFloat(1000.0)),
		VIPDiscounts:      vipDiscounts,
	}
}

func loadWorkerConfig() *WorkerConfig {
	return &WorkerConfig{
		PoolSize:   getEnvAsInt("WORKER_POOL_SIZE", 10),
		QueueSize:  getEnvAsInt("WORKER_QUEUE_SIZE", 100),
		Timeout:    getEnvAsDuration("WORKER_TIMEOUT", 30*time.Second),
		MaxRetries: getEnvAsInt("WORKER_MAX_RETRIES", 3),
		RetryDelay: getEnvAsDuration("WORKER_RETRY_DELAY", 5*time.Second),
	}
}

// Convert config to service-specific configs
func (c *Config) ToRepositoryConfig() *repository.Config {
	return &repository.Config{
		URI:                    c.Database.URI,
		Database:               c.Database.Database,
		Collection:             c.Database.Collection,
		MaxPoolSize:            c.Database.MaxPoolSize,
		MinPoolSize:            c.Database.MinPoolSize,
		MaxConnIdleTime:        c.Database.MaxConnIdleTime,
		ConnectionTimeout:      c.Database.ConnectionTimeout,
		SocketTimeout:          c.Database.SocketTimeout,
		ServerSelectionTimeout: c.Database.ServerSelectionTimeout,
	}
}

func (c *Config) ToUserClientConfig() *clients.UserClientConfig {
	return &clients.UserClientConfig{
		BaseURL: c.Clients.UserAPI.BaseURL,
		APIKey:  c.Clients.UserAPI.APIKey,
		Timeout: c.Clients.UserAPI.Timeout,
	}
}

func (c *Config) ToWalletClientConfig() *clients.WalletClientConfig {
	return &clients.WalletClientConfig{
		BaseURL: c.Clients.WalletAPI.BaseURL,
		APIKey:  c.Clients.WalletAPI.APIKey,
		Timeout: c.Clients.WalletAPI.Timeout,
	}
}

func (c *Config) ToMarketClientConfig() *clients.MarketClientConfig {
	return &clients.MarketClientConfig{
		BaseURL: c.Clients.MarketAPI.BaseURL,
		APIKey:  c.Clients.MarketAPI.APIKey,
		Timeout: c.Clients.MarketAPI.Timeout,
	}
}

func (c *Config) ToMessagingConfig() *messaging.MessagingConfig {
	return &messaging.MessagingConfig{
		URL:             c.Messaging.URL,
		ExchangeName:    c.Messaging.ExchangeName,
		DeadLetterExchange: c.Messaging.DeadLetterExchange,
		MaxRetries:      c.Messaging.MaxRetries,
		RetryDelay:      c.Messaging.RetryDelay,
		MessageTTL:      c.Messaging.MessageTTL,
		Persistent:      c.Messaging.Persistent,
	}
}

func (c *Config) ToConsumerConfig() *messaging.ConsumerConfig {
	return &messaging.ConsumerConfig{
		URL:           c.Messaging.URL,
		QueuePrefix:   c.Messaging.QueuePrefix,
		ConsumerTag:   c.Messaging.ConsumerTag,
		PrefetchCount: c.Messaging.PrefetchCount,
		AutoAck:       c.Messaging.AutoAck,
		WorkerCount:   c.Messaging.WorkerCount,
		RetryDelay:    c.Messaging.RetryDelay,
		MaxRetries:    c.Messaging.MaxRetries,
		DeadLetterTTL: c.Messaging.MessageTTL,
	}
}

func (c *Config) ToExecutionConfig() *concurrent.ExecutionConfig {
	return &concurrent.ExecutionConfig{
		MaxWorkers:       c.Execution.MaxWorkers,
		QueueSize:        c.Execution.QueueSize,
		ExecutionTimeout: c.Execution.ExecutionTimeout,
		MaxSlippage:      c.Execution.MaxSlippage,
		SimulateLatency:  c.Execution.SimulateLatency,
		MinExecutionTime: c.Execution.MinExecutionTime,
		MaxExecutionTime: c.Execution.MaxExecutionTime,
	}
}

func (c *Config) ToFeeConfig() *services.FeeConfig {
	return &services.FeeConfig{
		BaseFeePercentage: c.Fee.BaseFeePercentage,
		MakerFee:          c.Fee.MakerFee,
		TakerFee:          c.Fee.TakerFee,
		MinimumFee:        c.Fee.MinimumFee,
		MaximumFee:        c.Fee.MaximumFee,
		VIPDiscounts:      c.Fee.VIPDiscounts,
	}
}

func (c *Config) ToAuthConfig() *middleware.AuthConfig {
	return &middleware.AuthConfig{
		SecretKey:       c.Auth.SecretKey,
		Issuer:          c.Auth.Issuer,
		Audience:        c.Auth.Audience,
		SkipPaths:       c.Auth.SkipPaths,
		PublicEndpoints: c.Auth.PublicEndpoints,
	}
}

func (c *Config) ToLoggingConfig() *middleware.LoggingConfig {
	return &middleware.LoggingConfig{
		SkipPaths:       c.Logging.SkipPaths,
		LogBody:         c.Logging.LogBody,
		LogHeaders:      c.Logging.LogHeaders,
		MaxBodySize:     c.Logging.MaxBodySize,
		TimestampFormat: c.Logging.TimestampFormat,
	}
}

// Utility functions for environment variables
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsUint64(key string, defaultValue uint64) uint64 {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.ParseUint(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsDecimal(key string, defaultValue decimal.Decimal) decimal.Decimal {
	if value, exists := os.LookupEnv(key); exists {
		if decimalValue, err := decimal.NewFromString(value); err == nil {
			return decimalValue
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value, exists := os.LookupEnv(key); exists {
		// Simple comma-separated parsing
		// In production, consider using a more robust parsing method
		return []string{value} // Simplified for this example
	}
	return defaultValue
}

func (c *Config) Validate() error {
	if c.Database.URI == "" {
		return fmt.Errorf("database URI is required")
	}

	if c.Auth.SecretKey == "" || c.Auth.SecretKey == "your-super-secret-key-change-this-in-production" {
		return fmt.Errorf("JWT secret key must be set and changed from default")
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}

	if c.Messaging.URL == "" {
		return fmt.Errorf("messaging URL is required")
	}

	return nil
}