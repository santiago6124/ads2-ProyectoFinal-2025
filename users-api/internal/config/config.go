package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"users-api/internal/models"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      models.JWTConfig
	Redis    RedisConfig
	Internal InternalConfig
	RabbitMQ RabbitMQConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

type InternalConfig struct {
	APIKey string
}

type RabbitMQConfig struct {
	URL                  string
	BalanceRequestQueue  string
	BalanceResponseExchange string
	BalanceResponseRoutingKey string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	accessTTL, err := strconv.Atoi(getEnv("JWT_ACCESS_TTL", "3600"))
	if err != nil {
		accessTTL = 3600
	}

	refreshTTL, err := strconv.Atoi(getEnv("JWT_REFRESH_TTL", "604800"))
	if err != nil {
		refreshTTL = 604800
	}

	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8001"),
			Env:  getEnv("SERVER_ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", "password"),
			Name:     getEnv("DB_NAME", "users_db"),
		},
		JWT: models.JWTConfig{
			SecretKey:       getEnv("JWT_SECRET", "your-super-secret-key-change-in-production"),
			AccessTokenTTL:  time.Duration(accessTTL) * time.Second,
			RefreshTokenTTL: time.Duration(refreshTTL) * time.Second,
			Issuer:          "users-api",
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
		},
		Internal: InternalConfig{
			APIKey: getEnv("INTERNAL_API_KEY", "internal-secret-key"),
		},
		RabbitMQ: RabbitMQConfig{
			URL:                     getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
			BalanceRequestQueue:     getEnv("RABBITMQ_BALANCE_REQUEST_QUEUE", "balance.request"),
			BalanceResponseExchange: getEnv("RABBITMQ_BALANCE_RESPONSE_EXCHANGE", "balance.response.exchange"),
			BalanceResponseRoutingKey: getEnv("RABBITMQ_BALANCE_RESPONSE_ROUTING_KEY", "balance.response.portfolio"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}