package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	RabbitMQ   RabbitMQConfig   `mapstructure:"rabbitmq"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Limits     LimitsConfig     `mapstructure:"limits"`
	External   ExternalConfig   `mapstructure:"external"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host               string        `mapstructure:"host"`
	Port               int           `mapstructure:"port"`
	ReadTimeout        time.Duration `mapstructure:"read_timeout"`
	WriteTimeout       time.Duration `mapstructure:"write_timeout"`
	IdleTimeout        time.Duration `mapstructure:"idle_timeout"`
	GracefulTimeout    time.Duration `mapstructure:"graceful_timeout"`
	MaxRequestSize     int64         `mapstructure:"max_request_size"`
	EnableProfiling    bool          `mapstructure:"enable_profiling"`
	EnableSwagger      bool          `mapstructure:"enable_swagger"`
	TrustedProxies     []string      `mapstructure:"trusted_proxies"`
}

// DatabaseConfig contains MongoDB configuration
type DatabaseConfig struct {
	URI                string        `mapstructure:"uri"`
	Database           string        `mapstructure:"database"`
	MaxPoolSize        int           `mapstructure:"max_pool_size"`
	MinPoolSize        int           `mapstructure:"min_pool_size"`
	MaxIdleTime        time.Duration `mapstructure:"max_idle_time"`
	ConnectTimeout     time.Duration `mapstructure:"connect_timeout"`
	SocketTimeout      time.Duration `mapstructure:"socket_timeout"`
	SelectionTimeout   time.Duration `mapstructure:"selection_timeout"`
	HeartbeatInterval  time.Duration `mapstructure:"heartbeat_interval"`
	ReplicaSet         string        `mapstructure:"replica_set"`
	ReadPreference     string        `mapstructure:"read_preference"`
	WriteConcern       string        `mapstructure:"write_concern"`
}

// RedisConfig contains Redis configuration
type RedisConfig struct {
	Host               string        `mapstructure:"host"`
	Port               int           `mapstructure:"port"`
	Password           string        `mapstructure:"password"`
	DB                 int           `mapstructure:"db"`
	MaxRetries         int           `mapstructure:"max_retries"`
	PoolSize           int           `mapstructure:"pool_size"`
	MinIdleConnections int           `mapstructure:"min_idle_connections"`
	DialTimeout        time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout        time.Duration `mapstructure:"read_timeout"`
	WriteTimeout       time.Duration `mapstructure:"write_timeout"`
	PoolTimeout        time.Duration `mapstructure:"pool_timeout"`
	IdleTimeout        time.Duration `mapstructure:"idle_timeout"`
	LockTTL            time.Duration `mapstructure:"lock_ttl"`
	IdempotencyTTL     time.Duration `mapstructure:"idempotency_ttl"`
}

// RabbitMQConfig contains RabbitMQ configuration
type RabbitMQConfig struct {
	URL                 string        `mapstructure:"url"`
	Exchange            string        `mapstructure:"exchange"`
	TransactionQueue    string        `mapstructure:"transaction_queue"`
	NotificationQueue   string        `mapstructure:"notification_queue"`
	DeadLetterExchange  string        `mapstructure:"dead_letter_exchange"`
	RetryAttempts       int           `mapstructure:"retry_attempts"`
	RetryDelay          time.Duration `mapstructure:"retry_delay"`
	ConnectionTimeout   time.Duration `mapstructure:"connection_timeout"`
	HeartbeatInterval   time.Duration `mapstructure:"heartbeat_interval"`
	PrefetchCount       int           `mapstructure:"prefetch_count"`
	AutoAck             bool          `mapstructure:"auto_ack"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	JWTSecret           string        `mapstructure:"jwt_secret"`
	JWTExpiry           time.Duration `mapstructure:"jwt_expiry"`
	JWTIssuer           string        `mapstructure:"jwt_issuer"`
	InternalAPIKey      string        `mapstructure:"internal_api_key"`
	AdminAPIKey         string        `mapstructure:"admin_api_key"`
	SessionTimeout      time.Duration `mapstructure:"session_timeout"`
	MaxLoginAttempts    int           `mapstructure:"max_login_attempts"`
	LockoutDuration     time.Duration `mapstructure:"lockout_duration"`
}

// LimitsConfig contains wallet and transaction limits
type LimitsConfig struct {
	DefaultDailyWithdrawal    float64 `mapstructure:"default_daily_withdrawal"`
	DefaultDailyDeposit       float64 `mapstructure:"default_daily_deposit"`
	DefaultSingleTransaction  float64 `mapstructure:"default_single_transaction"`
	DefaultMonthlyVolume      float64 `mapstructure:"default_monthly_volume"`
	MaxTransactionAmount      float64 `mapstructure:"max_transaction_amount"`
	MinTransactionAmount      float64 `mapstructure:"min_transaction_amount"`
	LockDuration             time.Duration `mapstructure:"lock_duration"`
	MaxConcurrentLocks       int           `mapstructure:"max_concurrent_locks"`
	TransactionTimeout       time.Duration `mapstructure:"transaction_timeout"`
	ReconciliationThreshold  float64       `mapstructure:"reconciliation_threshold"`
}

// ExternalConfig contains external service configurations
type ExternalConfig struct {
	UsersAPI   ExternalServiceConfig `mapstructure:"users_api"`
	OrdersAPI  ExternalServiceConfig `mapstructure:"orders_api"`
	Timeout    time.Duration         `mapstructure:"timeout"`
	RetryCount int                   `mapstructure:"retry_count"`
}

// ExternalServiceConfig contains configuration for external services
type ExternalServiceConfig struct {
	URL    string `mapstructure:"url"`
	APIKey string `mapstructure:"api_key"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level       string `mapstructure:"level"`
	Format      string `mapstructure:"format"`
	Output      string `mapstructure:"output"`
	Filename    string `mapstructure:"filename"`
	MaxSize     int    `mapstructure:"max_size"`
	MaxAge      int    `mapstructure:"max_age"`
	MaxBackups  int    `mapstructure:"max_backups"`
	Compress    bool   `mapstructure:"compress"`
	EnableAudit bool   `mapstructure:"enable_audit"`
	AuditFile   string `mapstructure:"audit_file"`
}

// MonitoringConfig contains monitoring and metrics configuration
type MonitoringConfig struct {
	EnableMetrics     bool          `mapstructure:"enable_metrics"`
	MetricsPath       string        `mapstructure:"metrics_path"`
	EnableHealthCheck bool          `mapstructure:"enable_health_check"`
	HealthCheckPath   string        `mapstructure:"health_check_path"`
	EnablePprof       bool          `mapstructure:"enable_pprof"`
	PProfPath         string        `mapstructure:"pprof_path"`
	MetricsInterval   time.Duration `mapstructure:"metrics_interval"`
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Host:               getEnv("SERVER_HOST", "0.0.0.0"),
			Port:               getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:        getEnvAsDuration("SERVER_READ_TIMEOUT", "30s"),
			WriteTimeout:       getEnvAsDuration("SERVER_WRITE_TIMEOUT", "30s"),
			IdleTimeout:        getEnvAsDuration("SERVER_IDLE_TIMEOUT", "120s"),
			GracefulTimeout:    getEnvAsDuration("SERVER_GRACEFUL_TIMEOUT", "30s"),
			MaxRequestSize:     getEnvAsInt64("SERVER_MAX_REQUEST_SIZE", 10*1024*1024), // 10MB
			EnableProfiling:    getEnvAsBool("SERVER_ENABLE_PROFILING", false),
			EnableSwagger:      getEnvAsBool("SERVER_ENABLE_SWAGGER", true),
			TrustedProxies:     []string{"127.0.0.1", "::1"},
		},
		Database: DatabaseConfig{
			URI:                getEnv("DB_URI", "mongodb://localhost:27017/wallet_db"),
			Database:           getEnv("DB_NAME", "wallet_db"),
			MaxPoolSize:        getEnvAsInt("DB_MAX_POOL_SIZE", 100),
			MinPoolSize:        getEnvAsInt("DB_MIN_POOL_SIZE", 10),
			MaxIdleTime:        getEnvAsDuration("DB_MAX_IDLE_TIME", "300s"),
			ConnectTimeout:     getEnvAsDuration("DB_CONNECT_TIMEOUT", "30s"),
			SocketTimeout:      getEnvAsDuration("DB_SOCKET_TIMEOUT", "60s"),
			SelectionTimeout:   getEnvAsDuration("DB_SELECTION_TIMEOUT", "30s"),
			HeartbeatInterval:  getEnvAsDuration("DB_HEARTBEAT_INTERVAL", "10s"),
			ReplicaSet:         getEnv("DB_REPLICA_SET", ""),
			ReadPreference:     getEnv("DB_READ_PREFERENCE", "primary"),
			WriteConcern:       getEnv("DB_WRITE_CONCERN", "majority"),
		},
		Redis: RedisConfig{
			Host:               getEnv("REDIS_HOST", "localhost"),
			Port:               getEnvAsInt("REDIS_PORT", 6379),
			Password:           getEnv("REDIS_PASSWORD", ""),
			DB:                 getEnvAsInt("REDIS_DB", 0),
			MaxRetries:         getEnvAsInt("REDIS_MAX_RETRIES", 3),
			PoolSize:           getEnvAsInt("REDIS_POOL_SIZE", 10),
			MinIdleConnections: getEnvAsInt("REDIS_MIN_IDLE_CONNECTIONS", 5),
			DialTimeout:        getEnvAsDuration("REDIS_DIAL_TIMEOUT", "5s"),
			ReadTimeout:        getEnvAsDuration("REDIS_READ_TIMEOUT", "3s"),
			WriteTimeout:       getEnvAsDuration("REDIS_WRITE_TIMEOUT", "3s"),
			PoolTimeout:        getEnvAsDuration("REDIS_POOL_TIMEOUT", "4s"),
			IdleTimeout:        getEnvAsDuration("REDIS_IDLE_TIMEOUT", "300s"),
			LockTTL:            getEnvAsDuration("REDIS_LOCK_TTL", "30m"),
			IdempotencyTTL:     getEnvAsDuration("REDIS_IDEMPOTENCY_TTL", "24h"),
		},
		RabbitMQ: RabbitMQConfig{
			URL:                 getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
			Exchange:            getEnv("RABBITMQ_EXCHANGE", "wallet_events"),
			TransactionQueue:    getEnv("RABBITMQ_TRANSACTION_QUEUE", "wallet_transactions"),
			NotificationQueue:   getEnv("RABBITMQ_NOTIFICATION_QUEUE", "wallet_notifications"),
			DeadLetterExchange:  getEnv("RABBITMQ_DLX", "wallet_dlx"),
			RetryAttempts:       getEnvAsInt("RABBITMQ_RETRY_ATTEMPTS", 3),
			RetryDelay:          getEnvAsDuration("RABBITMQ_RETRY_DELAY", "5s"),
			ConnectionTimeout:   getEnvAsDuration("RABBITMQ_CONNECTION_TIMEOUT", "30s"),
			HeartbeatInterval:   getEnvAsDuration("RABBITMQ_HEARTBEAT_INTERVAL", "60s"),
			PrefetchCount:       getEnvAsInt("RABBITMQ_PREFETCH_COUNT", 10),
			AutoAck:             getEnvAsBool("RABBITMQ_AUTO_ACK", false),
		},
		Auth: AuthConfig{
			JWTSecret:           getEnv("JWT_SECRET", "wallet-api-secret-key-change-in-production"),
			JWTExpiry:           getEnvAsDuration("JWT_EXPIRY", "24h"),
			JWTIssuer:           getEnv("JWT_ISSUER", "wallet-api"),
			InternalAPIKey:      getEnv("INTERNAL_API_KEY", "internal-secret-key"),
			AdminAPIKey:         getEnv("ADMIN_API_KEY", "admin-secret-key"),
			SessionTimeout:      getEnvAsDuration("AUTH_SESSION_TIMEOUT", "30m"),
			MaxLoginAttempts:    getEnvAsInt("AUTH_MAX_LOGIN_ATTEMPTS", 5),
			LockoutDuration:     getEnvAsDuration("AUTH_LOCKOUT_DURATION", "15m"),
		},
		Limits: LimitsConfig{
			DefaultDailyWithdrawal:   getEnvAsFloat64("LIMITS_DEFAULT_DAILY_WITHDRAWAL", 10000.00),
			DefaultDailyDeposit:      getEnvAsFloat64("LIMITS_DEFAULT_DAILY_DEPOSIT", 50000.00),
			DefaultSingleTransaction: getEnvAsFloat64("LIMITS_DEFAULT_SINGLE_TRANSACTION", 25000.00),
			DefaultMonthlyVolume:     getEnvAsFloat64("LIMITS_DEFAULT_MONTHLY_VOLUME", 500000.00),
			MaxTransactionAmount:     getEnvAsFloat64("LIMITS_MAX_TRANSACTION_AMOUNT", 100000.00),
			MinTransactionAmount:     getEnvAsFloat64("LIMITS_MIN_TRANSACTION_AMOUNT", 0.01),
			LockDuration:            getEnvAsDuration("LIMITS_LOCK_DURATION", "30m"),
			MaxConcurrentLocks:      getEnvAsInt("LIMITS_MAX_CONCURRENT_LOCKS", 10),
			TransactionTimeout:      getEnvAsDuration("LIMITS_TRANSACTION_TIMEOUT", "30s"),
			ReconciliationThreshold: getEnvAsFloat64("LIMITS_RECONCILIATION_THRESHOLD", 0.01),
		},
		External: ExternalConfig{
			UsersAPI: ExternalServiceConfig{
				URL:    getEnv("USERS_API_URL", "http://users-api:8080"),
				APIKey: getEnv("USERS_API_KEY", "users-api-key"),
			},
			OrdersAPI: ExternalServiceConfig{
				URL:    getEnv("ORDERS_API_URL", "http://orders-api:8080"),
				APIKey: getEnv("ORDERS_API_KEY", "orders-api-key"),
			},
			Timeout:    getEnvAsDuration("EXTERNAL_TIMEOUT", "30s"),
			RetryCount: getEnvAsInt("EXTERNAL_RETRY_COUNT", 3),
		},
		Logging: LoggingConfig{
			Level:       getEnv("LOG_LEVEL", "info"),
			Format:      getEnv("LOG_FORMAT", "json"),
			Output:      getEnv("LOG_OUTPUT", "stdout"),
			Filename:    getEnv("LOG_FILENAME", "/app/logs/wallet-api.log"),
			MaxSize:     getEnvAsInt("LOG_MAX_SIZE", 100),
			MaxAge:      getEnvAsInt("LOG_MAX_AGE", 30),
			MaxBackups:  getEnvAsInt("LOG_MAX_BACKUPS", 5),
			Compress:    getEnvAsBool("LOG_COMPRESS", true),
			EnableAudit: getEnvAsBool("LOG_ENABLE_AUDIT", true),
			AuditFile:   getEnv("LOG_AUDIT_FILE", "/app/logs/wallet-audit.log"),
		},
		Monitoring: MonitoringConfig{
			EnableMetrics:     getEnvAsBool("MONITORING_ENABLE_METRICS", true),
			MetricsPath:       getEnv("MONITORING_METRICS_PATH", "/metrics"),
			EnableHealthCheck: getEnvAsBool("MONITORING_ENABLE_HEALTH_CHECK", true),
			HealthCheckPath:   getEnv("MONITORING_HEALTH_CHECK_PATH", "/health"),
			EnablePprof:       getEnvAsBool("MONITORING_ENABLE_PPROF", false),
			PProfPath:         getEnv("MONITORING_PPROF_PATH", "/debug/pprof"),
			MetricsInterval:   getEnvAsDuration("MONITORING_METRICS_INTERVAL", "15s"),
		},
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.URI == "" {
		return fmt.Errorf("database URI is required")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	if c.Auth.InternalAPIKey == "" {
		return fmt.Errorf("internal API key is required")
	}

	if c.Limits.MaxTransactionAmount <= 0 {
		return fmt.Errorf("max transaction amount must be positive")
	}

	if c.Limits.MinTransactionAmount < 0 {
		return fmt.Errorf("min transaction amount cannot be negative")
	}

	return nil
}

// Helper functions to parse environment variables

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
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsFloat64(key string, defaultValue float64) float64 {
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
	return 0
}