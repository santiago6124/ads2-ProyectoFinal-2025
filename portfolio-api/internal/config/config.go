package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Config represents the application configuration
type Config struct {
	Server       ServerConfig       `json:"server"`
	Database     DatabaseConfig     `json:"database"`
	Cache        CacheConfig        `json:"cache"`
	RabbitMQ     RabbitMQConfig     `json:"rabbitmq"`
	Auth         AuthConfig         `json:"auth"`
	ExternalAPIs ExternalAPIsConfig `json:"external_apis"`
	Scheduler    SchedulerConfig    `json:"scheduler"`
	RateLimit    RateLimitConfig    `json:"rate_limit"`
	Logger       LoggerConfig       `json:"logger"`
	Performance  PerformanceConfig  `json:"performance"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Port           int    `json:"port"`
	Host           string `json:"host"`
	Environment    string `json:"environment"`
	ReadTimeout    int    `json:"read_timeout"`
	WriteTimeout   int    `json:"write_timeout"`
	MaxHeaderBytes int    `json:"max_header_bytes"`
	EnableTLS      bool   `json:"enable_tls"`
	TLSCertFile    string `json:"tls_cert_file"`
	TLSKeyFile     string `json:"tls_key_file"`
}

// DatabaseConfig represents MongoDB configuration
type DatabaseConfig struct {
	URI            string `json:"uri"`
	Database       string `json:"database"`
	MaxPoolSize    int    `json:"max_pool_size"`
	MinPoolSize    int    `json:"min_pool_size"`
	MaxIdleTime    int    `json:"max_idle_time"`
	ConnectTimeout int    `json:"connect_timeout"`
	SocketTimeout  int    `json:"socket_timeout"`
	EnableSSL      bool   `json:"enable_ssl"`
	ReplicaSet     string `json:"replica_set"`
}

// CacheConfig represents Redis cache configuration
type CacheConfig struct {
	Host               string        `json:"host"`
	Port               int           `json:"port"`
	Password           string        `json:"password"`
	DB                 int           `json:"db"`
	MaxRetries         int           `json:"max_retries"`
	PoolSize           int           `json:"pool_size"`
	MinIdleConnections int           `json:"min_idle_connections"`
	DialTimeout        time.Duration `json:"dial_timeout"`
	ReadTimeout        time.Duration `json:"read_timeout"`
	WriteTimeout       time.Duration `json:"write_timeout"`
	PoolTimeout        time.Duration `json:"pool_timeout"`
	IdleTimeout        time.Duration `json:"idle_timeout"`

	// TTL settings
	PortfolioTTL    time.Duration `json:"portfolio_ttl"`
	PerformanceTTL  time.Duration `json:"performance_ttl"`
	SnapshotTTL     time.Duration `json:"snapshot_ttl"`
	CalculationTTL  time.Duration `json:"calculation_ttl"`
}

// RabbitMQConfig represents RabbitMQ configuration
type RabbitMQConfig struct {
	Enabled     bool   `json:"enabled"`
	URL         string `json:"url"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	VHost       string `json:"vhost"`

	// Exchange and queues
	OrderExchange     string `json:"order_exchange"`
	OrderQueue        string `json:"order_queue"`
	OrderRoutingKey   string `json:"order_routing_key"`

	// Consumer settings
	ConsumerTag       string `json:"consumer_tag"`
	AutoAck          bool   `json:"auto_ack"`
	Exclusive        bool   `json:"exclusive"`
	NoWait           bool   `json:"no_wait"`
	PrefetchCount    int    `json:"prefetch_count"`
	PrefetchSize     int    `json:"prefetch_size"`

	// Connection settings
	Heartbeat        time.Duration `json:"heartbeat"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	MaxReconnectAttempts int        `json:"max_reconnect_attempts"`
	ReconnectDelay   time.Duration `json:"reconnect_delay"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	JWTSecret           string        `json:"jwt_secret"`
	JWTExpiration       time.Duration `json:"jwt_expiration"`
	RefreshExpiration   time.Duration `json:"refresh_expiration"`
	RequireAuth         bool          `json:"require_auth"`
	AdminSecret         string        `json:"admin_secret"`
	EnableAPIKey        bool          `json:"enable_api_key"`
	APIKeyHeader        string        `json:"api_key_header"`
}

// ExternalAPIsConfig represents external API configurations
type ExternalAPIsConfig struct {
	MarketDataAPI MarketDataAPIConfig `json:"market_data_api"`
	OrdersAPI     OrdersAPIConfig     `json:"orders_api"`
	UsersAPI      UsersAPIConfig      `json:"users_api"`
}

// MarketDataAPIConfig represents market data API configuration
type MarketDataAPIConfig struct {
	BaseURL        string        `json:"base_url"`
	APIKey         string        `json:"api_key"`
	Timeout        time.Duration `json:"timeout"`
	MaxRetries     int           `json:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay"`
	RateLimit      int           `json:"rate_limit"`
	EnableCache    bool          `json:"enable_cache"`
	CacheTTL       time.Duration `json:"cache_ttl"`
}

// OrdersAPIConfig represents orders API configuration
type OrdersAPIConfig struct {
	BaseURL        string        `json:"base_url"`
	APIKey         string        `json:"api_key"`
	Timeout        time.Duration `json:"timeout"`
	MaxRetries     int           `json:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay"`
}

// UsersAPIConfig represents users API configuration
type UsersAPIConfig struct {
	BaseURL        string        `json:"base_url"`
	APIKey         string        `json:"api_key"`
	Timeout        time.Duration `json:"timeout"`
	MaxRetries     int           `json:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay"`
}

// SchedulerConfig represents background job scheduling configuration
type SchedulerConfig struct {
	Enabled              bool          `json:"enabled"`
	SnapshotInterval     string        `json:"snapshot_interval"`      // Cron expression
	MetricsUpdateInterval string       `json:"metrics_update_interval"` // Cron expression
	CleanupInterval      string        `json:"cleanup_interval"`       // Cron expression
	TimeZone             string        `json:"timezone"`
	MaxConcurrentJobs    int           `json:"max_concurrent_jobs"`
	JobTimeout           time.Duration `json:"job_timeout"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled        bool          `json:"enabled"`
	RequestsPerMin int           `json:"requests_per_minute"`
	BurstSize      int           `json:"burst_size"`
	WindowSize     time.Duration `json:"window_size"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// LoggerConfig represents logging configuration
type LoggerConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	Output     string `json:"output"`
	Filename   string `json:"filename"`
	MaxSize    int    `json:"max_size"`
	MaxAge     int    `json:"max_age"`
	MaxBackups int    `json:"max_backups"`
	Compress   bool   `json:"compress"`
}

// PerformanceConfig represents performance-related configurations
type PerformanceConfig struct {
	// Calculation settings
	DefaultPeriod        string        `json:"default_period"`
	MaxHistoryPeriod     string        `json:"max_history_period"`
	CalculationBatchSize int           `json:"calculation_batch_size"`
	CalculationTimeout   time.Duration `json:"calculation_timeout"`

	// Risk calculation settings
	VaRConfidenceLevel   float64 `json:"var_confidence_level"`
	RiskFreeRate         float64 `json:"risk_free_rate"`
	BenchmarkSymbol      string  `json:"benchmark_symbol"`

	// Rebalancing settings
	RebalanceThreshold   float64 `json:"rebalance_threshold"`
	MinPositionSize      float64 `json:"min_position_size"`
	MaxPositionSize      float64 `json:"max_position_size"`

	// Performance optimization
	EnableAsyncCalculation bool          `json:"enable_async_calculation"`
	CalculationWorkers     int           `json:"calculation_workers"`
	CacheCalculations      bool          `json:"cache_calculations"`
	PrecomputeMetrics      bool          `json:"precompute_metrics"`
}

// Load loads configuration from environment variables
func Load() *Config {
	// Load .env file if exists
	godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			Port:           getEnvInt("SERVER_PORT", 8083),
			Host:           getEnv("SERVER_HOST", "0.0.0.0"),
			Environment:    getEnv("ENVIRONMENT", "development"),
			ReadTimeout:    getEnvInt("SERVER_READ_TIMEOUT", 30),
			WriteTimeout:   getEnvInt("SERVER_WRITE_TIMEOUT", 30),
			MaxHeaderBytes: getEnvInt("SERVER_MAX_HEADER_BYTES", 1048576),
			EnableTLS:      getEnvBool("SERVER_ENABLE_TLS", false),
			TLSCertFile:    getEnv("SERVER_TLS_CERT_FILE", ""),
			TLSKeyFile:     getEnv("SERVER_TLS_KEY_FILE", ""),
		},

		Database: DatabaseConfig{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "cryptosim_portfolios"),
			MaxPoolSize:    getEnvInt("MONGODB_MAX_POOL_SIZE", 100),
			MinPoolSize:    getEnvInt("MONGODB_MIN_POOL_SIZE", 5),
			MaxIdleTime:    getEnvInt("MONGODB_MAX_IDLE_TIME", 300),
			ConnectTimeout: getEnvInt("MONGODB_CONNECT_TIMEOUT", 10),
			SocketTimeout:  getEnvInt("MONGODB_SOCKET_TIMEOUT", 30),
			EnableSSL:      getEnvBool("MONGODB_ENABLE_SSL", false),
			ReplicaSet:     getEnv("MONGODB_REPLICA_SET", ""),
		},

		Cache: CacheConfig{
			Host:               getEnv("REDIS_HOST", "localhost"),
			Port:               getEnvInt("REDIS_PORT", 6379),
			Password:           getEnv("REDIS_PASSWORD", ""),
			DB:                 getEnvInt("REDIS_DB", 0),
			MaxRetries:         getEnvInt("REDIS_MAX_RETRIES", 3),
			PoolSize:           getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConnections: getEnvInt("REDIS_MIN_IDLE_CONNECTIONS", 5),
			DialTimeout:        getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:        getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout:       getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			PoolTimeout:        getEnvDuration("REDIS_POOL_TIMEOUT", 4*time.Second),
			IdleTimeout:        getEnvDuration("REDIS_IDLE_TIMEOUT", 5*time.Minute),
			PortfolioTTL:       getEnvDuration("CACHE_PORTFOLIO_TTL", 10*time.Minute),
			PerformanceTTL:     getEnvDuration("CACHE_PERFORMANCE_TTL", 15*time.Minute),
			SnapshotTTL:        getEnvDuration("CACHE_SNAPSHOT_TTL", time.Hour),
			CalculationTTL:     getEnvDuration("CACHE_CALCULATION_TTL", 5*time.Minute),
		},

		RabbitMQ: RabbitMQConfig{
			Enabled:              getEnvBool("RABBITMQ_ENABLED", true),
			URL:                  getEnv("RABBITMQ_URL", ""),
			Host:                 getEnv("RABBITMQ_HOST", "localhost"),
			Port:                 getEnvInt("RABBITMQ_PORT", 5672),
			Username:             getEnv("RABBITMQ_USERNAME", "guest"),
			Password:             getEnv("RABBITMQ_PASSWORD", "guest"),
			VHost:                getEnv("RABBITMQ_VHOST", "/"),
			OrderExchange:        getEnv("RABBITMQ_ORDER_EXCHANGE", "orders"),
			OrderQueue:           getEnv("RABBITMQ_ORDER_QUEUE", "portfolio.orders"),
			OrderRoutingKey:      getEnv("RABBITMQ_ORDER_ROUTING_KEY", "order.executed"),
			ConsumerTag:          getEnv("RABBITMQ_CONSUMER_TAG", "portfolio-service"),
			AutoAck:              getEnvBool("RABBITMQ_AUTO_ACK", false),
			Exclusive:            getEnvBool("RABBITMQ_EXCLUSIVE", false),
			NoWait:               getEnvBool("RABBITMQ_NO_WAIT", false),
			PrefetchCount:        getEnvInt("RABBITMQ_PREFETCH_COUNT", 10),
			PrefetchSize:         getEnvInt("RABBITMQ_PREFETCH_SIZE", 0),
			Heartbeat:            getEnvDuration("RABBITMQ_HEARTBEAT", 30*time.Second),
			ConnectionTimeout:    getEnvDuration("RABBITMQ_CONNECTION_TIMEOUT", 30*time.Second),
			MaxReconnectAttempts: getEnvInt("RABBITMQ_MAX_RECONNECT_ATTEMPTS", 5),
			ReconnectDelay:       getEnvDuration("RABBITMQ_RECONNECT_DELAY", 5*time.Second),
		},

		Auth: AuthConfig{
			JWTSecret:         getEnv("JWT_SECRET", "default-secret-key"),
			JWTExpiration:     getEnvDuration("JWT_EXPIRATION", 24*time.Hour),
			RefreshExpiration: getEnvDuration("JWT_REFRESH_EXPIRATION", 7*24*time.Hour),
			RequireAuth:       getEnvBool("REQUIRE_AUTH", true),
			AdminSecret:       getEnv("INTERNAL_API_KEY", getEnv("ADMIN_SECRET", "admin-secret-key")),
			EnableAPIKey:      getEnvBool("ENABLE_API_KEY", false),
			APIKeyHeader:      getEnv("API_KEY_HEADER", "X-API-Key"),
		},

		ExternalAPIs: ExternalAPIsConfig{
			MarketDataAPI: MarketDataAPIConfig{
				BaseURL:     getEnv("MARKET_DATA_API_URL", "http://localhost:8082"),
				APIKey:      getEnv("MARKET_DATA_API_KEY", ""),
				Timeout:     getEnvDuration("MARKET_DATA_API_TIMEOUT", 30*time.Second),
				MaxRetries:  getEnvInt("MARKET_DATA_API_MAX_RETRIES", 3),
				RetryDelay:  getEnvDuration("MARKET_DATA_API_RETRY_DELAY", time.Second),
				RateLimit:   getEnvInt("MARKET_DATA_API_RATE_LIMIT", 100),
				EnableCache: getEnvBool("MARKET_DATA_API_ENABLE_CACHE", true),
				CacheTTL:    getEnvDuration("MARKET_DATA_API_CACHE_TTL", 5*time.Minute),
			},
			OrdersAPI: OrdersAPIConfig{
				BaseURL:    getEnv("ORDERS_API_URL", "http://localhost:8081"),
				APIKey:     getEnv("ORDERS_API_KEY", ""),
				Timeout:    getEnvDuration("ORDERS_API_TIMEOUT", 30*time.Second),
				MaxRetries: getEnvInt("ORDERS_API_MAX_RETRIES", 3),
				RetryDelay: getEnvDuration("ORDERS_API_RETRY_DELAY", time.Second),
			},
			UsersAPI: UsersAPIConfig{
				BaseURL:    getEnv("USERS_API_URL", "http://localhost:8080"),
				APIKey:     getEnv("USERS_API_KEY", ""),
				Timeout:    getEnvDuration("USERS_API_TIMEOUT", 30*time.Second),
				MaxRetries: getEnvInt("USERS_API_MAX_RETRIES", 3),
				RetryDelay: getEnvDuration("USERS_API_RETRY_DELAY", time.Second),
			},
		},

		Scheduler: SchedulerConfig{
			Enabled:               getEnvBool("SCHEDULER_ENABLED", true),
			SnapshotInterval:      getEnv("SCHEDULER_SNAPSHOT_INTERVAL", "0 0 * * *"),      // Daily at midnight
			MetricsUpdateInterval: getEnv("SCHEDULER_METRICS_UPDATE_INTERVAL", "*/5 * * * *"), // Every 5 minutes
			CleanupInterval:       getEnv("SCHEDULER_CLEANUP_INTERVAL", "0 2 * * *"),       // Daily at 2 AM
			TimeZone:              getEnv("SCHEDULER_TIMEZONE", "UTC"),
			MaxConcurrentJobs:     getEnvInt("SCHEDULER_MAX_CONCURRENT_JOBS", 5),
			JobTimeout:            getEnvDuration("SCHEDULER_JOB_TIMEOUT", 30*time.Minute),
		},

		RateLimit: RateLimitConfig{
			Enabled:         getEnvBool("RATE_LIMIT_ENABLED", true),
			RequestsPerMin:  getEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 100),
			BurstSize:       getEnvInt("RATE_LIMIT_BURST_SIZE", 10),
			WindowSize:      getEnvDuration("RATE_LIMIT_WINDOW_SIZE", time.Minute),
			CleanupInterval: getEnvDuration("RATE_LIMIT_CLEANUP_INTERVAL", 10*time.Minute),
		},

		Logger: LoggerConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			Output:     getEnv("LOG_OUTPUT", "stdout"),
			Filename:   getEnv("LOG_FILENAME", ""),
			MaxSize:    getEnvInt("LOG_MAX_SIZE", 100),
			MaxAge:     getEnvInt("LOG_MAX_AGE", 28),
			MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 3),
			Compress:   getEnvBool("LOG_COMPRESS", true),
		},

		Performance: PerformanceConfig{
			DefaultPeriod:          getEnv("PERFORMANCE_DEFAULT_PERIOD", "30d"),
			MaxHistoryPeriod:       getEnv("PERFORMANCE_MAX_HISTORY_PERIOD", "1y"),
			CalculationBatchSize:   getEnvInt("PERFORMANCE_CALCULATION_BATCH_SIZE", 100),
			CalculationTimeout:     getEnvDuration("PERFORMANCE_CALCULATION_TIMEOUT", 30*time.Second),
			VaRConfidenceLevel:     getEnvFloat("PERFORMANCE_VAR_CONFIDENCE_LEVEL", 95.0),
			RiskFreeRate:           getEnvFloat("PERFORMANCE_RISK_FREE_RATE", 0.02),
			BenchmarkSymbol:        getEnv("PERFORMANCE_BENCHMARK_SYMBOL", "BTC"),
			RebalanceThreshold:     getEnvFloat("PERFORMANCE_REBALANCE_THRESHOLD", 5.0),
			MinPositionSize:        getEnvFloat("PERFORMANCE_MIN_POSITION_SIZE", 0.01),
			MaxPositionSize:        getEnvFloat("PERFORMANCE_MAX_POSITION_SIZE", 50.0),
			EnableAsyncCalculation: getEnvBool("PERFORMANCE_ENABLE_ASYNC_CALCULATION", true),
			CalculationWorkers:     getEnvInt("PERFORMANCE_CALCULATION_WORKERS", 5),
			CacheCalculations:      getEnvBool("PERFORMANCE_CACHE_CALCULATIONS", true),
			PrecomputeMetrics:      getEnvBool("PERFORMANCE_PRECOMPUTE_METRICS", true),
		},
	}

	return config
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.URI == "" {
		return fmt.Errorf("database URI is required")
	}

	if c.Auth.JWTSecret == "" || c.Auth.JWTSecret == "default-secret-key" {
		logrus.Warn("Using default JWT secret key, this is not recommended for production")
	}

	if c.ExternalAPIs.MarketDataAPI.BaseURL == "" {
		return fmt.Errorf("market data API URL is required")
	}

	return nil
}