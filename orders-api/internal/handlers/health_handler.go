package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"orders-api/internal/clients"
	"orders-api/internal/messaging"
	"orders-api/internal/repositories"
)

type HealthHandler struct {
	orderRepo         repositories.OrderRepository
	userClient        *clients.UserClient
	userBalanceClient *clients.UserBalanceClient
	marketClient      *clients.MarketClient
	publisher         *messaging.Publisher
	// consumer eliminado en sistema simplificado
}

type HealthResponse struct {
	Status      string                    `json:"status"`
	Timestamp   time.Time                 `json:"timestamp"`
	Version     string                    `json:"version"`
	Uptime      time.Duration             `json:"uptime"`
	Environment string                    `json:"environment"`
	Services    map[string]ServiceHealth  `json:"services"`
}

type ServiceHealth struct {
	Status      string        `json:"status"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
	Error       string        `json:"error,omitempty"`
	LastCheck   time.Time     `json:"last_check"`
	Details     interface{}   `json:"details,omitempty"`
}

type ReadinessResponse struct {
	Ready    bool                     `json:"ready"`
	Services map[string]ServiceHealth `json:"services"`
}

type LivenessResponse struct {
	Alive bool `json:"alive"`
}

var startTime = time.Now()

func NewHealthHandler(
	orderRepo repositories.OrderRepository,
	userClient *clients.UserClient,
	userBalanceClient *clients.UserBalanceClient,
	marketClient *clients.MarketClient,
	publisher *messaging.Publisher,
	consumer interface{}, // No se usa en sistema simplificado
) *HealthHandler {
	return &HealthHandler{
		orderRepo:         orderRepo,
		userClient:        userClient,
		userBalanceClient: userBalanceClient,
		marketClient:      marketClient,
		publisher:         publisher,
	}
}

func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	services := make(map[string]ServiceHealth)

	// Check MongoDB
	services["mongodb"] = h.checkMongoDB(ctx)

	// Check User API
	services["user_api"] = h.checkUserAPI(ctx)

	// Check User Balance Client
	services["user_balance_client"] = h.checkUserBalanceClient(ctx)

	// Check Market Data API
	services["market_api"] = h.checkMarketAPI(ctx)

	// Check RabbitMQ Publisher
	services["rabbitmq_publisher"] = h.checkRabbitMQPublisher()

	// Check RabbitMQ Consumer
	services["rabbitmq_consumer"] = h.checkRabbitMQConsumer()

	// Determine overall status
	overallStatus := "healthy"
	for _, service := range services {
		// Ignore not_applicable status (e.g., consumer in simplified system)
		if service.Status != "healthy" && service.Status != "not_applicable" {
			overallStatus = "unhealthy"
			break
		}
	}

	response := &HealthResponse{
		Status:      overallStatus,
		Timestamp:   time.Now(),
		Version:     "1.0.0",
		Uptime:      time.Since(startTime),
		Environment: getEnvironment(),
		Services:    services,
	}

	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	services := make(map[string]ServiceHealth)

	// Check critical services for readiness
	services["mongodb"] = h.checkMongoDB(ctx)
	services["user_api"] = h.checkUserAPI(ctx)
	services["user_balance_client"] = h.checkUserBalanceClient(ctx)
	services["market_api"] = h.checkMarketAPI(ctx)

	ready := true
	for _, service := range services {
		if service.Status != "healthy" {
			ready = false
		}
	}

	response := &ReadinessResponse{
		Ready:    ready,
		Services: services,
	}

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

func (h *HealthHandler) Liveness(c *gin.Context) {
	response := &LivenessResponse{
		Alive: true,
	}

	c.JSON(http.StatusOK, response)
}

func (h *HealthHandler) checkMongoDB(ctx context.Context) ServiceHealth {
	start := time.Now()

	// Repository health check - just verify it's available
	var err error
	if h.orderRepo == nil {
		err = fmt.Errorf("order repository not available")
	}
	responseTime := time.Since(start)

	if err != nil {
		return ServiceHealth{
			Status:       "unhealthy",
			ResponseTime: responseTime,
			Error:        err.Error(),
			LastCheck:    time.Now(),
		}
	}

	return ServiceHealth{
		Status:       "healthy",
		ResponseTime: responseTime,
		LastCheck:    time.Now(),
		Details: map[string]interface{}{
			"database": "orders",
		},
	}
}

func (h *HealthHandler) checkUserAPI(ctx context.Context) ServiceHealth {
	start := time.Now()

	err := h.userClient.HealthCheck(ctx)
	responseTime := time.Since(start)

	if err != nil {
		return ServiceHealth{
			Status:       "unhealthy",
			ResponseTime: responseTime,
			Error:        err.Error(),
			LastCheck:    time.Now(),
		}
	}

	return ServiceHealth{
		Status:       "healthy",
		ResponseTime: responseTime,
		LastCheck:    time.Now(),
		Details: map[string]interface{}{
			"service": "user-api",
		},
	}
}

func (h *HealthHandler) checkUserBalanceClient(ctx context.Context) ServiceHealth {
	start := time.Now()

	err := h.userBalanceClient.HealthCheck(ctx)
	responseTime := time.Since(start)

	if err != nil {
		return ServiceHealth{
			Status:       "unhealthy",
			ResponseTime: responseTime,
			Error:        err.Error(),
			LastCheck:    time.Now(),
		}
	}

	return ServiceHealth{
		Status:       "healthy",
		ResponseTime: responseTime,
		LastCheck:    time.Now(),
		Details: map[string]interface{}{
			"service": "user-balance-client",
		},
	}
}

func (h *HealthHandler) checkMarketAPI(ctx context.Context) ServiceHealth {
	start := time.Now()

	err := h.marketClient.HealthCheck(ctx)
	responseTime := time.Since(start)

	if err != nil {
		return ServiceHealth{
			Status:       "unhealthy",
			ResponseTime: responseTime,
			Error:        err.Error(),
			LastCheck:    time.Now(),
		}
	}

	return ServiceHealth{
		Status:       "healthy",
		ResponseTime: responseTime,
		LastCheck:    time.Now(),
		Details: map[string]interface{}{
			"service": "market-data-api",
		},
	}
}

func (h *HealthHandler) checkRabbitMQPublisher() ServiceHealth {
	start := time.Now()

	err := h.publisher.HealthCheck()
	responseTime := time.Since(start)

	if err != nil {
		return ServiceHealth{
			Status:       "unhealthy",
			ResponseTime: responseTime,
			Error:        err.Error(),
			LastCheck:    time.Now(),
		}
	}

	return ServiceHealth{
		Status:       "healthy",
		ResponseTime: responseTime,
		LastCheck:    time.Now(),
		Details: map[string]interface{}{
			"component": "rabbitmq-publisher",
		},
	}
}

// checkRabbitMQConsumer comentado - no hay consumer en sistema simplificado
func (h *HealthHandler) checkRabbitMQConsumer() ServiceHealth {
	return ServiceHealth{
		Status:       "not_applicable",
		ResponseTime: 0,
		Error:        "",
		LastCheck:    time.Now(),
		Details: map[string]interface{}{
			"component": "rabbitmq-consumer",
			"note":      "Consumer not used in simplified system",
		},
	}
}

func getEnvironment() string {
	// This would typically come from environment variables
	// For now, return a default value
	return "development"
}

func (h *HealthHandler) Metrics(c *gin.Context) {
	// Basic metrics endpoint
	metrics := map[string]interface{}{
		"uptime_seconds":    time.Since(startTime).Seconds(),
		"timestamp":         time.Now().Unix(),
		"goroutines":        "N/A", // Would need runtime.NumGoroutine()
		"memory_usage":      "N/A", // Would need runtime.MemStats
		"requests_total":    "N/A", // Would need request counter middleware
		"requests_duration": "N/A", // Would need request duration middleware
	}

	c.JSON(http.StatusOK, metrics)
}