package config

import (
	"os"
	"strconv"
	"strings"
)

// Config represents the application configuration
type Config struct {
	Environment string
	Version     string
	Server      ServerConfig
	Solr        SolrConfig
	Cache       CacheConfig
	RabbitMQ    RabbitMQConfig
	Logging     LoggingConfig
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Port           int
	ReadTimeoutMs  int
	WriteTimeoutMs int
	IdleTimeoutMs  int
	MaxHeaderBytes int
}

// SolrConfig represents Apache Solr configuration
type SolrConfig struct {
	BaseURL    string
	Collection string
	TimeoutMs  int
	MaxRetries int
}

// CacheConfig represents caching configuration
type CacheConfig struct {
	LocalTTLMinutes         int
	DistributedTTLMinutes   int
	MaxLocalSize            int
	LocalItemsToPrune       int
	MemcachedHosts          []string
	MemcachedTimeoutMs      int
	MemcachedMaxIdleConns   int
	KeyPrefix               string
	EnableMetrics           bool
}

// RabbitMQConfig represents RabbitMQ configuration
type RabbitMQConfig struct {
	Enabled         bool
	URL             string
	ExchangeName    string
	QueueName       string
	RoutingKeys     []string
	WorkerPoolSize  int
	RetryAttempts   int
	RetryDelayMs    int
	DeadLetterQueue string
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Version:     getEnv("VERSION", "1.0.0"),
		Server: ServerConfig{
			Port:           getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeoutMs:  getEnvAsInt("SERVER_READ_TIMEOUT_MS", 10000),
			WriteTimeoutMs: getEnvAsInt("SERVER_WRITE_TIMEOUT_MS", 10000),
			IdleTimeoutMs:  getEnvAsInt("SERVER_IDLE_TIMEOUT_MS", 60000),
			MaxHeaderBytes: getEnvAsInt("SERVER_MAX_HEADER_BYTES", 1048576),
		},
		Solr: SolrConfig{
			BaseURL:    getEnv("SOLR_BASE_URL", "http://localhost:8983/solr"),
			Collection: getEnv("SOLR_COLLECTION", "crypto_search"),
			TimeoutMs:  getEnvAsInt("SOLR_TIMEOUT_MS", 5000),
			MaxRetries: getEnvAsInt("SOLR_MAX_RETRIES", 3),
		},
		Cache: CacheConfig{
			LocalTTLMinutes:         getEnvAsInt("CACHE_LOCAL_TTL_MINUTES", 5),
			DistributedTTLMinutes:   getEnvAsInt("CACHE_DISTRIBUTED_TTL_MINUTES", 15),
			MaxLocalSize:            getEnvAsInt("CACHE_MAX_LOCAL_SIZE", 1000000),
			LocalItemsToPrune:       getEnvAsInt("CACHE_LOCAL_ITEMS_TO_PRUNE", 100),
			MemcachedHosts:          getEnvAsSlice("CACHE_MEMCACHED_HOSTS", []string{"localhost:11211"}),
			MemcachedTimeoutMs:      getEnvAsInt("CACHE_MEMCACHED_TIMEOUT_MS", 5000),
			MemcachedMaxIdleConns:   getEnvAsInt("CACHE_MEMCACHED_MAX_IDLE_CONNS", 100),
			KeyPrefix:               getEnv("CACHE_KEY_PREFIX", "search"),
			EnableMetrics:           getEnvAsBool("CACHE_ENABLE_METRICS", true),
		},
		RabbitMQ: RabbitMQConfig{
			Enabled:         getEnvAsBool("RABBITMQ_ENABLED", true),
			URL:             getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
			ExchangeName:    getEnv("RABBITMQ_EXCHANGE_NAME", "crypto_events"),
			QueueName:       getEnv("RABBITMQ_QUEUE_NAME", "search_events"),
			RoutingKeys:     getEnvAsSlice("RABBITMQ_ROUTING_KEYS", []string{"order.executed", "price.changed", "search.performed"}),
			WorkerPoolSize:  getEnvAsInt("RABBITMQ_WORKER_POOL_SIZE", 5),
			RetryAttempts:   getEnvAsInt("RABBITMQ_RETRY_ATTEMPTS", 3),
			RetryDelayMs:    getEnvAsInt("RABBITMQ_RETRY_DELAY_MS", 1000),
			DeadLetterQueue: getEnv("RABBITMQ_DEAD_LETTER_QUEUE", "search_events_dlq"),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
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

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}