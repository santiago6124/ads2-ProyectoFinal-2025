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

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"wallet-api/internal/cache"
	"wallet-api/internal/config"
	"wallet-api/internal/controller"
	"wallet-api/internal/database"
	"wallet-api/internal/engine"
	"wallet-api/internal/external"
	"wallet-api/internal/middleware"
	"wallet-api/internal/monitoring"
	"wallet-api/internal/repository"
	"wallet-api/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := setupLogger(cfg.LogLevel)

	// Initialize dependencies
	ctx := context.Background()
	deps, err := initializeDependencies(ctx, cfg, logger)
	if err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}
	defer deps.cleanup()

	// Setup HTTP server
	server := setupHTTPServer(cfg, deps, logger)

	// Setup graceful shutdown
	setupGracefulShutdown(server, deps, logger)

	// Start server
	logger.Printf("Starting Wallet API server on port %s", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}

type dependencies struct {
	database          *database.Database
	cache             cache.CacheService
	messageQueue      external.MessageQueue
	walletService     service.WalletService
	complianceService service.ComplianceService
	auditService      service.AuditService
	metrics           monitoring.MetricsService
	healthChecker     monitoring.HealthChecker
}

func (d *dependencies) cleanup() {
	if d.messageQueue != nil {
		d.messageQueue.Close()
	}
	if d.cache != nil {
		d.cache.Close()
	}
	if d.database != nil {
		d.database.Close()
	}
}

func initializeDependencies(ctx context.Context, cfg *config.Config, logger *log.Logger) (*dependencies, error) {
	deps := &dependencies{}

	// Initialize metrics
	deps.metrics = monitoring.NewPrometheusMetrics()

	// Initialize health checker
	deps.healthChecker = monitoring.NewHealthChecker("1.0.0")

	// Initialize database
	db, err := database.NewDatabase(&database.DatabaseConfig{
		MongoURI:    cfg.Database.MongoURI,
		DatabaseName: cfg.Database.DatabaseName,
		MaxPoolSize: cfg.Database.MaxPoolSize,
		Timeout:     cfg.Database.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	deps.database = db

	// Register database health check
	deps.healthChecker.RegisterCheck("database", monitoring.NewDatabaseChecker("mongodb", func(ctx context.Context) error {
		return db.Ping(ctx)
	}))

	// Initialize cache
	cacheService, err := cache.NewRedisCache(&cache.CacheConfig{
		Host:         cfg.Redis.Host,
		Port:         cfg.Redis.Port,
		Password:     cfg.Redis.Password,
		Database:     cfg.Redis.Database,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		Timeout:      cfg.Redis.Timeout,
		KeyPrefix:    "wallet:",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}
	deps.cache = cacheService

	// Register cache health check
	deps.healthChecker.RegisterCheck("cache", monitoring.NewCacheChecker("redis", cacheService))

	// Initialize message queue
	messageQueue, err := external.NewMessageQueue(&external.MessageQueueConfig{
		URL:             cfg.MessageQueue.URL,
		ExchangeName:    cfg.MessageQueue.ExchangeName,
		RetryAttempts:   cfg.MessageQueue.RetryAttempts,
		RetryDelay:      cfg.MessageQueue.RetryDelay,
		MessageTTL:      cfg.MessageQueue.MessageTTL,
		PrefetchCount:   cfg.MessageQueue.PrefetchCount,
		EnableDeadLetter: cfg.MessageQueue.EnableDeadLetter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize message queue: %w", err)
	}
	deps.messageQueue = messageQueue

	// Register message queue health check
	deps.healthChecker.RegisterCheck("message_queue", monitoring.NewMessageQueueChecker("rabbitmq", messageQueue))

	// Initialize repositories
	walletRepo := repository.NewWalletRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	complianceRepo := repository.NewComplianceRepository(db)

	// Initialize distributed lock and idempotency services
	distributedLock := engine.NewRedisDistributedLock(cacheService)
	idempotencyService := engine.NewRedisIdempotencyService(cacheService)

	// Initialize transaction engine
	transactionEngine := engine.NewTransactionEngine(
		walletRepo,
		transactionRepo,
		distributedLock,
		idempotencyService,
		&engine.TransactionEngineConfig{
			DefaultLockTimeout: 30 * time.Second,
			MaxRetries:         cfg.Engine.MaxRetries,
			RetryDelay:         cfg.Engine.RetryDelay,
		},
	)

	// Initialize external services
	fraudDetectionService := external.NewMockFraudDetectionService() // Use mock for demo
	blockchainService := external.NewBlockchainService(&external.BlockchainConfig{
		Providers: map[string]external.ProviderConfig{
			"BTC": {Type: "mock", Network: "testnet"},
			"ETH": {Type: "mock", Network: "testnet"},
		},
		DefaultProvider: "mock",
		Timeout:         30 * time.Second,
		MaxRetries:      3,
	})
	notificationService := external.NewNotificationService(&external.NotificationConfig{
		Providers: map[string]external.NotificationProviderConfig{
			"email": {Type: "mock"},
			"sms":   {Type: "mock"},
		},
		DefaultProvider: "mock",
		Timeout:         10 * time.Second,
		MaxRetries:      3,
	})

	// Initialize event publisher
	eventPublisher := external.NewEventPublisher(messageQueue)

	// Initialize business services
	deps.walletService = service.NewWalletService(
		walletRepo,
		transactionEngine,
		&service.WalletServiceConfig{
			DefaultCurrency:      "USD",
			MaxDailyTransactions: 1000,
			MaxTransactionAmount: cfg.Limits.MaxTransactionAmount,
		},
	)

	deps.complianceService = service.NewComplianceService(
		complianceRepo,
		transactionRepo,
		fraudDetectionService,
		eventPublisher,
		&service.ComplianceServiceConfig{
			VelocityLimits: cfg.Limits.VelocityLimits,
			RiskThresholds: cfg.Limits.RiskThresholds,
			EnableRealTimeMonitoring: true,
		},
	)

	deps.auditService = service.NewAuditService(
		auditRepo,
		eventPublisher,
		&service.AuditServiceConfig{
			RetentionPeriod: 7 * 24 * time.Hour,
			EnableRealTimeAudit: true,
		},
	)

	// Start periodic tasks
	monitoring.StartSystemMetricsRecording(deps.metrics, 30*time.Second)
	deps.healthChecker.StartPeriodicChecks(60 * time.Second)

	return deps, nil
}

func setupHTTPServer(cfg *config.Config, deps *dependencies, logger *log.Logger) *http.Server {
	// Set gin mode
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.New()

	// Add middleware
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.SecurityMiddleware())

	// Add rate limiting middleware
	rateLimiter := middleware.NewRateLimiter(deps.cache, &middleware.RateLimitConfig{
		GlobalLimit:  cfg.RateLimit.GlobalLimit,
		UserLimit:    cfg.RateLimit.UserLimit,
		IPLimit:      cfg.RateLimit.IPLimit,
		WindowSize:   cfg.RateLimit.WindowSize,
		EnableBurst:  cfg.RateLimit.EnableBurst,
	})
	router.Use(rateLimiter.RateLimitMiddleware())

	// Add metrics middleware
	metricsMiddleware := monitoring.NewMetricsMiddleware(deps.metrics)
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		deps.metrics.RecordHTTPRequest(c.Request.Method, c.FullPath(), c.Writer.Status(), duration)
	})

	// Health check endpoints
	router.GET("/health", func(c *gin.Context) {
		health := deps.healthChecker.CheckHealth(c.Request.Context())
		status := http.StatusOK
		if health.Status != "healthy" {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, health)
	})

	router.GET("/health/:component", func(c *gin.Context) {
		component := c.Param("component")
		health := deps.healthChecker.GetComponentStatus(component)
		if health == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Component not found"})
			return
		}
		status := http.StatusOK
		if health.Status != "healthy" {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, health)
	})

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes
	api := router.Group("/api/v1")

	// Initialize controllers
	walletController := controller.NewWalletController(deps.walletService)
	transactionController := controller.NewTransactionController(deps.walletService)
	complianceController := controller.NewComplianceController(deps.complianceService)
	auditController := controller.NewAuditController(deps.auditService)

	// Setup routes
	setupAPIRoutes(api, walletController, transactionController, complianceController, auditController)

	// Create HTTP server
	return &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

func setupAPIRoutes(
	api *gin.RouterGroup,
	walletController *controller.WalletController,
	transactionController *controller.TransactionController,
	complianceController *controller.ComplianceController,
	auditController *controller.AuditController,
) {
	// Wallet routes
	wallets := api.Group("/wallets")
	{
		wallets.POST("", walletController.CreateWallet)
		wallets.GET("/:walletId", walletController.GetWallet)
		wallets.PUT("/:walletId", walletController.UpdateWallet)
		wallets.GET("/:walletId/balance", walletController.GetWalletBalance)
		wallets.POST("/:walletId/deposit", walletController.Deposit)
		wallets.POST("/:walletId/withdraw", walletController.Withdraw)
		wallets.POST("/:walletId/lock-funds", walletController.LockFunds)
		wallets.POST("/:walletId/unlock-funds", walletController.UnlockFunds)
		wallets.GET("/:walletId/transactions", walletController.GetTransactions)
		wallets.POST("/:walletId/reconcile", walletController.ReconcileWallet)
	}

	// User wallet routes
	users := api.Group("/users")
	{
		users.GET("/:userId/wallets", walletController.GetUserWallets)
		users.GET("/:userId/transactions", walletController.GetUserTransactions)
		users.GET("/:userId/summary", walletController.GetUserSummary)
	}

	// Transaction routes
	transactions := api.Group("/transactions")
	{
		transactions.GET("/:transactionId", transactionController.GetTransaction)
		transactions.POST("/:transactionId/reverse", transactionController.ReverseTransaction)
		transactions.GET("/:transactionId/status", transactionController.GetTransactionStatus)
	}

	// Transfer routes
	api.POST("/transfers", walletController.Transfer)

	// Compliance routes
	compliance := api.Group("/compliance")
	{
		compliance.POST("/check", complianceController.RunComplianceCheck)
		compliance.GET("/alerts", complianceController.GetAlerts)
		compliance.PUT("/alerts/:alertId", complianceController.UpdateAlert)
		compliance.GET("/reports", complianceController.GetComplianceReports)
		compliance.POST("/velocity-check", complianceController.VelocityCheck)
	}

	// Audit routes
	audit := api.Group("/audit")
	{
		audit.GET("/events", auditController.GetAuditEvents)
		audit.GET("/trail/:resourceId", auditController.GetAuditTrail)
		audit.GET("/reports", auditController.GetAuditReports)
		audit.POST("/events", auditController.CreateAuditEvent)
	}

	// Admin routes (would typically require admin authentication)
	admin := api.Group("/admin")
	{
		admin.GET("/stats", walletController.GetSystemStats)
		admin.POST("/maintenance/cleanup", walletController.CleanupExpiredLocks)
		admin.POST("/maintenance/reconcile-all", walletController.ReconcileAllWallets)
	}
}

func setupLogger(level string) *log.Logger {
	logger := log.New(os.Stdout, "[WALLET-API] ", log.LstdFlags|log.Lshortfile)
	return logger
}

func setupGracefulShutdown(server *http.Server, deps *dependencies, logger *log.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		logger.Println("Shutting down server...")

		// Stop health checks
		deps.healthChecker.StopPeriodicChecks()

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Shutdown HTTP server
		if err := server.Shutdown(ctx); err != nil {
			logger.Printf("Server forced to shutdown: %v", err)
		}

		// Cleanup dependencies
		deps.cleanup()

		logger.Println("Server shutdown complete")
		os.Exit(0)
	}()
}