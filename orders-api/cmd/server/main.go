package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"

	"orders-api/internal/clients"
	"orders-api/internal/concurrent"
	"orders-api/internal/config"
	"orders-api/internal/handlers"
	"orders-api/internal/messaging"
	"orders-api/internal/middleware"
	"orders-api/internal/models"
	"orders-api/internal/repositories"
	"orders-api/internal/routes"
	"orders-api/internal/services"
	"orders-api/pkg/database"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Setup logger
	logger := setupLogger(cfg.Logging)
	logger.Info("Starting Orders API service...")

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database repository
	logger.Info("Connecting to MongoDB...")
	db, err := database.NewConnection()
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	orderRepo := repositories.NewOrderRepository(db)

	// Test database connection
	if err := db.Client.Ping(ctx, nil); err != nil {
		logger.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	logger.Info("Successfully connected to MongoDB")

	// Initialize external service clients
	logger.Info("Initializing external service clients...")
	userClient := clients.NewUserClient(cfg.ToUserClientConfig())
	userBalanceClient := clients.NewUserBalanceClient(cfg.ToUserBalanceClientConfig())
	marketClient := clients.NewMarketClient(cfg.ToMarketClientConfig())

	// Test client connections
	if err := testClientConnections(ctx, userClient, userBalanceClient, marketClient, logger); err != nil {
		logger.Warnf("Some client connections failed: %v", err)
	}

	// Initialize messaging
	logger.Info("Setting up RabbitMQ messaging...")
	publisher, err := messaging.NewPublisher(cfg.ToMessagingConfig())
	if err != nil {
		logger.Fatalf("Failed to create message publisher: %v", err)
	}
	defer publisher.Close()

	consumer, err := messaging.NewConsumer(cfg.ToConsumerConfig())
	if err != nil {
		logger.Fatalf("Failed to create message consumer: %v", err)
	}
	defer consumer.Stop()

	// Setup message handlers
	setupMessageHandlers(consumer, logger)

	// Start message consumer
	if err := consumer.StartConsuming(ctx, cfg.Messaging.ExchangeName); err != nil {
		logger.Fatalf("Failed to start message consumer: %v", err)
	}
	logger.Info("Message consumer started successfully")

	// Initialize concurrent execution services
	logger.Info("Setting up concurrent execution services...")
	feeCalculator := services.NewFeeCalculator(cfg.ToFeeConfig())

	executionService := concurrent.NewExecutionService(
		userClient,
		userBalanceClient,
		marketClient,
		feeCalculator,
		cfg.ToExecutionConfig(),
	)

	workerPool := concurrent.NewWorkerPool(
		cfg.Worker.PoolSize,
		cfg.Worker.QueueSize,
		executionService,
	)

	if err := workerPool.Start(ctx); err != nil {
		logger.Fatalf("Failed to start worker pool: %v", err)
	}
	defer workerPool.Stop()
	logger.Info("Worker pool started successfully")

	orchestrator := concurrent.NewOrderOrchestrator(
		cfg.Worker.PoolSize,
		cfg.Worker.QueueSize,
		executionService,
	)

	if err := orchestrator.Start(ctx); err != nil {
		logger.Fatalf("Failed to start order orchestrator: %v", err)
	}
	defer orchestrator.Stop()
	logger.Info("Order orchestrator started successfully")

	// Initialize business services
	logger.Info("Initializing business services...")
	
	// Create market service (mock implementation)
	marketService := &mockMarketService{}
	
	// Create event publisher wrapper
	eventPublisher := &eventPublisherWrapper{publisher: publisher}
	
	orderService := services.NewOrderService(
		orderRepo,
		orchestrator,
		executionService,
		feeCalculator,
		marketService,
		eventPublisher,
	)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.ToAuthConfig())
	loggingMiddleware := middleware.NewLoggingMiddleware(logger, cfg.ToLoggingConfig())

	// Initialize handlers
	orderHandler := handlers.NewOrderHandler(orderService)
	healthHandler := handlers.NewHealthHandler(
		orderRepo,
		userClient,
		userBalanceClient,
		marketClient,
		publisher,
		consumer,
	)

	// Setup routes
	logger.Info("Setting up HTTP routes...")
	router := routes.NewRouter(
		orderHandler,
		healthHandler,
		authMiddleware,
		loggingMiddleware,
		&routes.RouterConfig{
			Debug:       cfg.Server.Debug,
			CORSEnabled: cfg.Server.CORSEnabled,
			AllowedOrigins: cfg.Server.AllowedOrigins,
		},
	)

	router.SetupRoutes(&routes.RouterConfig{
		Debug:       cfg.Server.Debug,
		CORSEnabled: cfg.Server.CORSEnabled,
		AllowedOrigins: cfg.Server.AllowedOrigins,
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router.GetEngine(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start HTTP server in a goroutine
	go func() {
		logger.Infof("Starting HTTP server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

func setupLogger(config *LoggingConfig) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set log format
	switch config.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: config.TimestampFormat,
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: config.TimestampFormat,
		})
	default:
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: config.TimestampFormat,
		})
	}

	// Set output
	switch config.Output {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	default:
		logger.SetOutput(os.Stdout)
	}

	return logger
}

func testClientConnections(ctx context.Context, userClient *clients.UserClient, userBalanceClient *clients.UserBalanceClient, marketClient *clients.MarketClient, logger *logrus.Logger) error {
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Test User API
	if err := userClient.HealthCheck(testCtx); err != nil {
		logger.Warnf("User API health check failed: %v", err)
		return fmt.Errorf("user API connection failed: %w", err)
	}
	logger.Info("User API connection successful")

	// Test User Balance Client (uses User API)
	if err := userBalanceClient.HealthCheck(testCtx); err != nil {
		logger.Warnf("User Balance Client health check failed: %v", err)
		return fmt.Errorf("user balance client connection failed: %w", err)
	}
	logger.Info("User Balance Client connection successful")

	// Test Market API
	if err := marketClient.HealthCheck(testCtx); err != nil {
		logger.Warnf("Market API health check failed: %v", err)
		return fmt.Errorf("market API connection failed: %w", err)
	}
	logger.Info("Market API connection successful")

	return nil
}

func setupMessageHandlers(consumer *messaging.Consumer, logger *logrus.Logger) {
	// Example message handlers - in production, these would be more comprehensive

	// Handle order status updates
	consumer.RegisterHandler("orders.created", func(ctx context.Context, message *messaging.EventMessage) error {
		logger.WithField("message_id", message.ID).Info("Processing order created event")
		// Handle order created event
		return nil
	})

	consumer.RegisterHandler("orders.updated", func(ctx context.Context, message *messaging.EventMessage) error {
		logger.WithField("message_id", message.ID).Info("Processing order updated event")
		// Handle order updated event
		return nil
	})

	consumer.RegisterHandler("orders.executed", func(ctx context.Context, message *messaging.EventMessage) error {
		logger.WithField("message_id", message.ID).Info("Processing order executed event")
		// Handle order executed event
		return nil
	})

	consumer.RegisterHandler("orders.failed", func(ctx context.Context, message *messaging.EventMessage) error {
		logger.WithField("message_id", message.ID).Error("Processing order failed event")
		// Handle order failed event
		return nil
	})

	consumer.RegisterHandler("orders.cancelled", func(ctx context.Context, message *messaging.EventMessage) error {
		logger.WithField("message_id", message.ID).Info("Processing order cancelled event")
		// Handle order cancelled event
		return nil
	})

	logger.Info("Message handlers registered successfully")
}

// Mock implementations for services
type mockMarketService struct{}

func (m *mockMarketService) GetMarketPrice(ctx context.Context, symbol string) (*models.PriceResult, error) {
	return &models.PriceResult{
		Symbol:         symbol,
		MarketPrice:    decimal.NewFromFloat(50000.0),
		BidPrice:       decimal.NewFromFloat(49995.0),
		AskPrice:       decimal.NewFromFloat(50005.0),
		ExecutionPrice: decimal.NewFromFloat(50000.0),
		Volume24h:      "1000000",
		Change24h:      "1000",
		ChangePercent:  "2.0",
		High24h:        decimal.NewFromFloat(51000.0),
		Low24h:         decimal.NewFromFloat(49000.0),
		Source:         "mock",
		Confidence:     "high",
		LastUpdated:    time.Now().Format(time.RFC3339),
		Slippage:       decimal.Zero,
		SlippagePerc:   decimal.Zero,
		Timestamp:      time.Now(),
	}, nil
}

func (m *mockMarketService) GetMarketConditions(ctx context.Context, symbol string) (*models.MarketConditions, error) {
	return &models.MarketConditions{
		Symbol:              symbol,
		CurrentPrice:        decimal.NewFromFloat(50000.0),
		Volume24h:           decimal.NewFromFloat(1000000.0),
		PriceChange24h:      decimal.NewFromFloat(1000.0),
		MarketCap:           decimal.NewFromFloat(1000000000.0),
		Liquidity:           "high",
		Volatility:          "medium",
		Spread:              decimal.NewFromFloat(10.0),
		SpreadPercent:       decimal.NewFromFloat(0.02),
		TradingVolume:       decimal.NewFromFloat(1000000.0),
		OrderBookDepth:      decimal.NewFromFloat(10000.0),
		CirculatingSupply:   decimal.NewFromFloat(21000000.0),
		LastUpdated:         time.Now(),
	}, nil
}

func (m *mockMarketService) ValidateSymbol(ctx context.Context, symbol string) (*services.CryptoInfo, error) {
	return &services.CryptoInfo{
		Symbol:      symbol,
		IsActive:    true,
		Name:        "Mock " + symbol,
		CurrentPrice: decimal.NewFromFloat(50000.0),
		MinQuantity: decimal.NewFromFloat(0.001),
		MaxQuantity: decimal.NewFromFloat(100.0),
	}, nil
}

func (m *mockMarketService) GetCurrentPrice(ctx context.Context, symbol string) (decimal.Decimal, error) {
	return decimal.NewFromFloat(50000.0), nil
}

func (m *mockMarketService) IsMarketOpen(ctx context.Context) bool {
	return true
}

// Event publisher wrapper to match the interface
type eventPublisherWrapper struct {
	publisher *messaging.Publisher
}

func (e *eventPublisherWrapper) PublishOrderExecuted(ctx context.Context, order *models.Order) error {
	return e.publisher.PublishOrderExecuted(ctx, order, nil)
}

func (e *eventPublisherWrapper) PublishOrderCancelled(ctx context.Context, order *models.Order, reason string) error {
	return e.publisher.PublishOrderCancelled(ctx, order, reason)
}

func (e *eventPublisherWrapper) PublishOrderFailed(ctx context.Context, order *models.Order, reason string) error {
	return e.publisher.PublishOrderFailed(ctx, order, reason)
}

func (e *eventPublisherWrapper) PublishOrderCreated(ctx context.Context, order *models.Order) error {
	return e.publisher.PublishOrderCreated(ctx, order)
}

// Type alias to avoid import cycle
type LoggingConfig = config.LoggingConfig