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
	logger.Info("🚀 Starting Orders API service (SIMPLIFIED)...")

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	logger.Info("📦 Connecting to MongoDB...")
	db, err := database.NewConnection()
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	orderRepo := repositories.NewOrderRepository(db)

	// Test database connection
	if err := db.Client.Ping(ctx, nil); err != nil {
		logger.Fatalf("Failed to ping MongoDB: %v", err)
	}
	logger.Info("✅ Successfully connected to MongoDB")

	// Initialize external service clients
	logger.Info("🔗 Initializing external service clients...")
	userClient := clients.NewUserClient(cfg.ToUserClientConfig())
	userBalanceClient := clients.NewUserBalanceClient(cfg.ToUserBalanceClientConfig())
	marketClient := clients.NewMarketClient(cfg.ToMarketClientConfig())

	// Test client connections (non-blocking)
	go testClientConnections(ctx, userClient, userBalanceClient, marketClient, logger)

	// Initialize RabbitMQ publisher (simplified)
	logger.Info("📨 Setting up RabbitMQ messaging...")
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@localhost:5672/"
	}

	publisher, err := messaging.NewPublisher(rabbitmqURL)
	if err != nil {
		logger.Warnf("Failed to create RabbitMQ publisher (continuing without events): %v", err)
		publisher = nil // Sistema puede funcionar sin eventos
	} else {
		defer publisher.Close()
		logger.Info("✅ RabbitMQ publisher initialized")
	}

	// Initialize simplified services
	logger.Info("⚙️ Initializing business services (simplified)...")

	// Create execution service (simplified - no concurrency)
	executionService := services.NewExecutionService(
		userClient,
		userBalanceClient,
		marketClient,
		nil, // No necesitamos fee calculator separado
	)

	// Create market service adapter
	marketService := &marketServiceAdapter{marketClient: marketClient}

	// Create event publisher adapter (puede ser nil)
	var eventPublisher services.EventPublisher
	if publisher != nil {
		eventPublisher = &eventPublisherAdapter{publisher: publisher}
	} else {
		eventPublisher = &noopPublisher{} // No-op si no hay RabbitMQ
	}

	// Initialize simplified order service (no orchestrator, no workers)
	orderService := services.NewOrderServiceSimple(
		orderRepo,
		executionService,
		marketService,
		eventPublisher,
	)

	logger.Info("✅ Business services initialized (simplified, no concurrency)")

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
		nil, // No consumer
	)

	// Setup routes
	logger.Info("🛣️ Setting up HTTP routes...")
	router := routes.NewRouter(
		orderHandler,
		healthHandler,
		authMiddleware,
		loggingMiddleware,
		&routes.RouterConfig{
			Debug:          cfg.Server.Debug,
			CORSEnabled:    cfg.Server.CORSEnabled,
			AllowedOrigins: cfg.Server.AllowedOrigins,
		},
	)

	router.SetupRoutes(&routes.RouterConfig{
		Debug:          cfg.Server.Debug,
		CORSEnabled:    cfg.Server.CORSEnabled,
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

	// Start HTTP server
	go func() {
		logger.Infof("🌐 HTTP server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	logger.Info("✨ Orders API is ready to accept requests!")
	logger.Info("📝 System simplified: No workers, no orchestrator, synchronous execution")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("👋 Server exited gracefully")
}

// setupLogger configures the application logger
func setupLogger(config *config.LoggingConfig) *logrus.Logger {
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
			ForceColors:     true,
		})
	default:
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: config.TimestampFormat,
			ForceColors:     true,
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

// testClientConnections tests external service connections (non-blocking)
func testClientConnections(
	ctx context.Context,
	userClient *clients.UserClient,
	userBalanceClient *clients.UserBalanceClient,
	marketClient *clients.MarketClient,
	logger *logrus.Logger,
) {
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Test User API
	if err := userClient.HealthCheck(testCtx); err != nil {
		logger.Warnf("⚠️ User API health check failed: %v", err)
	} else {
		logger.Info("✅ User API connection successful")
	}

	// Test User Balance Client
	if err := userBalanceClient.HealthCheck(testCtx); err != nil {
		logger.Warnf("⚠️ User Balance Client health check failed: %v", err)
	} else {
		logger.Info("✅ User Balance Client connection successful")
	}

	// Test Market API
	if err := marketClient.HealthCheck(testCtx); err != nil {
		logger.Warnf("⚠️ Market API health check failed: %v", err)
	} else {
		logger.Info("✅ Market API connection successful")
	}
}

// marketServiceAdapter adapts MarketClient to MarketService interface
type marketServiceAdapter struct {
	marketClient *clients.MarketClient
}

func (m *marketServiceAdapter) GetCurrentPrice(ctx context.Context, symbol string) (decimal.Decimal, error) {
	price, err := m.marketClient.GetCurrentPrice(ctx, symbol)
	if err != nil {
		// Fallback: usar precios simulados si Market API no responde
		log.Printf("Market API error, using fallback price for %s: %v", symbol, err)
		return m.getFallbackPrice(symbol), nil
	}
	return price.MarketPrice, nil
}

func (m *marketServiceAdapter) ValidateSymbol(ctx context.Context, symbol string) (*services.CryptoInfo, error) {
	// Intentar obtener precio de Market API
	price, err := m.marketClient.GetCurrentPrice(ctx, symbol)

	var currentPrice decimal.Decimal
	if err != nil {
		// Fallback: validar contra lista conocida y usar precio simulado
		log.Printf("Market API error for %s, using fallback: %v", symbol, err)
		if !m.isKnownSymbol(symbol) {
			return nil, fmt.Errorf("symbol %s not found or invalid", symbol)
		}
		currentPrice = m.getFallbackPrice(symbol)
	} else {
		currentPrice = price.MarketPrice
	}

	return &services.CryptoInfo{
		Symbol:       symbol,
		Name:         m.getCryptoName(symbol),
		CurrentPrice: currentPrice,
		IsActive:     true,
	}, nil
}

// isKnownSymbol verifica si el símbolo es conocido
func (m *marketServiceAdapter) isKnownSymbol(symbol string) bool {
	knownSymbols := map[string]bool{
		"BTC": true, "ETH": true, "BNB": true, "SOL": true,
		"XRP": true, "ADA": true, "DOGE": true, "AVAX": true,
		"DOT": true, "MATIC": true, "LTC": true, "LINK": true,
	}
	return knownSymbols[symbol]
}

// getFallbackPrice retorna un precio simulado para testing
func (m *marketServiceAdapter) getFallbackPrice(symbol string) decimal.Decimal {
	// Precios simulados para desarrollo/testing
	prices := map[string]float64{
		"BTC":   50000.00,
		"ETH":   3000.00,
		"BNB":   400.00,
		"SOL":   100.00,
		"XRP":   0.60,
		"ADA":   0.50,
		"DOGE":  0.10,
		"AVAX":  35.00,
		"DOT":   7.00,
		"MATIC": 0.80,
		"LTC":   70.00,
		"LINK":  15.00,
	}

	if price, ok := prices[symbol]; ok {
		return decimal.NewFromFloat(price)
	}
	return decimal.NewFromFloat(1000.00) // Precio por defecto
}

// getCryptoName retorna el nombre completo de la crypto
func (m *marketServiceAdapter) getCryptoName(symbol string) string {
	names := map[string]string{
		"BTC":   "Bitcoin",
		"ETH":   "Ethereum",
		"BNB":   "Binance Coin",
		"SOL":   "Solana",
		"XRP":   "Ripple",
		"ADA":   "Cardano",
		"DOGE":  "Dogecoin",
		"AVAX":  "Avalanche",
		"DOT":   "Polkadot",
		"MATIC": "Polygon",
		"LTC":   "Litecoin",
		"LINK":  "Chainlink",
	}

	if name, ok := names[symbol]; ok {
		return name
	}
	return symbol
}

// eventPublisherAdapter adapts messaging.Publisher to EventPublisher interface
type eventPublisherAdapter struct {
	publisher *messaging.Publisher
}

func (e *eventPublisherAdapter) PublishOrderCreated(ctx context.Context, order *Order) error {
	return e.publisher.PublishOrderCreated(ctx, order)
}

func (e *eventPublisherAdapter) PublishOrderExecuted(ctx context.Context, order *Order) error {
	return e.publisher.PublishOrderExecuted(ctx, order)
}

func (e *eventPublisherAdapter) PublishOrderCancelled(ctx context.Context, order *Order, reason string) error {
	return e.publisher.PublishOrderCancelled(ctx, order, reason)
}

func (e *eventPublisherAdapter) PublishOrderFailed(ctx context.Context, order *Order, reason string) error {
	return e.publisher.PublishOrderFailed(ctx, order, reason)
}

// noopPublisher is a no-op publisher when RabbitMQ is not available
type noopPublisher struct{}

func (n *noopPublisher) PublishOrderCreated(ctx context.Context, order *Order) error {
	log.Println("No-op: Order created event (RabbitMQ not available)")
	return nil
}

func (n *noopPublisher) PublishOrderExecuted(ctx context.Context, order *Order) error {
	log.Println("No-op: Order executed event (RabbitMQ not available)")
	return nil
}

func (n *noopPublisher) PublishOrderCancelled(ctx context.Context, order *Order, reason string) error {
	log.Println("No-op: Order cancelled event (RabbitMQ not available)")
	return nil
}

func (n *noopPublisher) PublishOrderFailed(ctx context.Context, order *Order, reason string) error {
	log.Println("No-op: Order failed event (RabbitMQ not available)")
	return nil
}

// Type alias to avoid import issues
type Order = models.Order
