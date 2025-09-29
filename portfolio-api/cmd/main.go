package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"portfolio-api/internal/config"
	"portfolio-api/internal/controllers"
	"portfolio-api/internal/middleware"
	"portfolio-api/internal/repositories"
	"portfolio-api/internal/services"
	"portfolio-api/internal/clients"
	"portfolio-api/internal/messaging"
	"portfolio-api/internal/scheduler"
	"portfolio-api/pkg/cache"
	"portfolio-api/pkg/database"
	"portfolio-api/pkg/logger"
)

// @title Portfolio API
// @version 1.0
// @description Microservicio de gestión de portafolios para CryptoSim
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.cryptosim.com/support
// @contact.email support@cryptosim.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8083
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger.Init(cfg.Logger)
	log := logrus.WithField("service", "portfolio-api")

	log.Info("Starting Portfolio API service...")

	// Initialize database connection
	db, err := database.NewMongoDB(cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB: ", err)
	}
	defer db.Disconnect()

	// Initialize Redis cache
	cacheClient, err := cache.NewRedisClient(cfg.Cache)
	if err != nil {
		log.Fatal("Failed to connect to Redis: ", err)
	}
	defer cacheClient.Close()

	// Initialize repositories
	portfolioRepo := repositories.NewPortfolioRepository(db)
	snapshotRepo := repositories.NewSnapshotRepository(db)

	// Initialize external clients
	marketClient := clients.NewMarketClient(cfg.ExternalAPIs.MarketDataAPI)
	ordersClient := clients.NewOrdersClient(cfg.ExternalAPIs.OrdersAPI)
	usersClient := clients.NewUsersClient(cfg.ExternalAPIs.UsersAPI)

	// Initialize services
	calculationService := services.NewCalculationService(marketClient)
	performanceService := services.NewPerformanceService(portfolioRepo, snapshotRepo, calculationService)
	portfolioService := services.NewPortfolioService(portfolioRepo, calculationService, performanceService, cacheClient)
	snapshotService := services.NewSnapshotService(snapshotRepo, portfolioService)
	rebalancingService := services.NewRebalancingService(portfolioService, calculationService)
	benchmarkService := services.NewBenchmarkService(marketClient, performanceService)

	// Initialize controllers
	portfolioController := controllers.NewPortfolioController(portfolioService, performanceService)
	performanceController := controllers.NewPerformanceController(performanceService, benchmarkService)
	holdingsController := controllers.NewHoldingsController(portfolioService)
	analyticsController := controllers.NewAnalyticsController(portfolioService, rebalancingService)

	// Initialize RabbitMQ consumer
	var orderConsumer *messaging.OrderConsumer
	if cfg.RabbitMQ.Enabled {
		orderConsumer, err = messaging.NewOrderConsumer(cfg.RabbitMQ, portfolioService)
		if err != nil {
			log.Error("Failed to initialize RabbitMQ consumer: ", err)
		} else {
			go orderConsumer.Start()
		}
	}

	// Initialize schedulers
	snapshotScheduler := scheduler.NewSnapshotScheduler(snapshotService)
	metricsUpdater := scheduler.NewMetricsUpdater(portfolioService)
	cleanupJob := scheduler.NewCleanupJob(snapshotRepo)

	// Start scheduled jobs
	if cfg.Scheduler.Enabled {
		go snapshotScheduler.Start()
		go metricsUpdater.Start()
		go cleanupJob.Start()
	}

	// Setup HTTP server
	router := setupRouter(cfg, portfolioController, performanceController, holdingsController, analyticsController)

	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.Server.WriteTimeout) * time.Second,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// Start server in goroutine
	go func() {
		log.WithField("port", cfg.Server.Port).Info("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server: ", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown: ", err)
	}

	// Close RabbitMQ consumer
	if orderConsumer != nil {
		orderConsumer.Stop()
	}

	// Stop schedulers
	snapshotScheduler.Stop()
	metricsUpdater.Stop()
	cleanupJob.Stop()

	log.Info("Server exited")
}

func setupRouter(cfg *config.Config,
	portfolioController *controllers.PortfolioController,
	performanceController *controllers.PerformanceController,
	holdingsController *controllers.HoldingsController,
	analyticsController *controllers.AnalyticsController) *gin.Engine {

	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logging())

	// Rate limiting
	if cfg.RateLimit.Enabled {
		router.Use(middleware.RateLimit(cfg.RateLimit))
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "portfolio-api",
			"timestamp": time.Now().UTC(),
		})
	})

	// API routes
	api := router.Group("/api")
	{
		// Portfolio routes
		portfolio := api.Group("/portfolio")
		portfolio.Use(middleware.Auth(cfg.Auth))
		{
			portfolio.GET("/:userId", portfolioController.GetPortfolio)
			portfolio.GET("/:userId/holdings", holdingsController.GetHoldings)
			portfolio.GET("/:userId/performance", performanceController.GetPerformance)
			portfolio.GET("/:userId/history", performanceController.GetHistory)
			portfolio.GET("/:userId/analysis", analyticsController.GetAnalysis)
			portfolio.GET("/:userId/rebalancing", analyticsController.GetRebalancing)
			portfolio.POST("/:userId/snapshot", portfolioController.CreateSnapshot)
		}

		// Admin routes (internal use)
		admin := api.Group("/admin")
		admin.Use(middleware.Auth(cfg.Auth))
		admin.Use(middleware.AdminOnly())
		{
			admin.POST("/recalculate/:userId", portfolioController.RecalculatePortfolio)
			admin.GET("/metrics", portfolioController.GetMetrics)
			admin.DELETE("/snapshots/cleanup", portfolioController.CleanupSnapshots)
		}
	}

	return router
}