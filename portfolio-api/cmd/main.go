package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"

	"portfolio-api/internal/clients"
	"portfolio-api/internal/config"
	"portfolio-api/internal/controllers"
	"portfolio-api/internal/messaging"
	"portfolio-api/internal/repositories"
	repomongo "portfolio-api/internal/repositories/mongo"
	"portfolio-api/pkg/database"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	logger.WithField("service", "portfolio-api").Info("Starting Portfolio API service...")

	// Connect to MongoDB
	logger.Info("Connecting to MongoDB...")
	mongodb, err := database.NewMongoDB(cfg.Database)
	var db *mongo.Database
	if err != nil {
		logger.Warnf("Failed to connect to MongoDB: %v - running without database", err)
		mongodb = nil
		db = nil
	} else {
		logger.Info("✅ Connected to MongoDB")
		db = mongodb.GetDatabase()
	}

	// Initialize API clients
	logger.Info("Initializing API clients...")
	userClient := clients.NewUserClient(&clients.UserClientConfig{
		BaseURL: cfg.ExternalAPIs.UsersAPI.BaseURL,
		APIKey:  cfg.Auth.AdminSecret, // Use admin secret as internal API key
		Timeout: 10 * time.Second,
	})
	logger.Info("✅ User API client initialized")

	marketClient := clients.NewMarketDataClient(&clients.MarketDataClientConfig{
		BaseURL: cfg.ExternalAPIs.MarketDataAPI.BaseURL,
		Timeout: 10 * time.Second,
	})
	logger.Info("✅ Market Data API client initialized")

	// Initialize repositories if database is available
	var portfolioRepo repositories.PortfolioRepository
	if db != nil {
		logger.Info("Initializing repositories...")
		portfolioRepo = repomongo.NewPortfolioRepository(db)
		logger.Info("✅ Repositories initialized")
	}

	// Setup HTTP server
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "portfolio-api",
			"timestamp": time.Now().UTC(),
		})
	})

	// Initialize balance messaging components
	var balancePublisher *messaging.BalancePublisher
	var balanceConsumer *messaging.BalanceResponseConsumer

	if cfg.RabbitMQ.Enabled {
		logger.Info("🔌 Initializing balance messaging components...")

		// Initialize balance request publisher
		balancePublisher, err = messaging.NewBalancePublisher(
			cfg.RabbitMQ.URL,
			cfg.RabbitMQ.BalanceRequestExchange,
			cfg.RabbitMQ.BalanceRequestRoutingKey,
			logger,
		)
		if err != nil {
			logger.Warnf("Failed to initialize balance publisher: %v - will use HTTP fallback", err)
			balancePublisher = nil
		}

		// Initialize balance response consumer
		balanceConsumer, err = messaging.NewBalanceResponseConsumer(
			cfg.RabbitMQ.URL,
			cfg.RabbitMQ.BalanceResponseQueue,
			logger,
		)
		if err != nil {
			logger.Warnf("Failed to initialize balance consumer: %v - will use HTTP fallback", err)
			balanceConsumer = nil
		} else {
			// Start balance response consumer in background
			ctx := context.Background()
			go func() {
				if err := balanceConsumer.Start(ctx); err != nil {
					logger.Errorf("Balance consumer error: %v", err)
				}
			}()

			// Handle graceful shutdown
			defer func() {
				if balanceConsumer != nil {
					if err := balanceConsumer.Close(); err != nil {
						logger.Errorf("Error closing balance consumer: %v", err)
					}
				}
				if balancePublisher != nil {
					if err := balancePublisher.Close(); err != nil {
						logger.Errorf("Error closing balance publisher: %v", err)
					}
				}
			}()
		}

		if balancePublisher != nil && balanceConsumer != nil {
			logger.Info("✅ Balance messaging initialized successfully")
		}
	}

	// Initialize portfolio controller with balance messaging support
	controller := controllers.NewPortfolioControllerWithClientsAndMessaging(
		logger,
		userClient,
		marketClient,
		portfolioRepo,
		balancePublisher,
		balanceConsumer,
	)

	// API routes
	api := router.Group("/api")
	{
		// Plural route: /api/portfolios
		portfolios := api.Group("/portfolios")
		controller.RegisterRoutes(portfolios)

		// Singular route for orders-api compatibility: /api/portfolio
		portfolio := api.Group("/portfolio")
		{
			portfolio.POST("/:userId/holdings", controller.UpdateHoldings)
		}
	}

	// Initialize and start RabbitMQ consumer for portfolio updates
	if portfolioRepo != nil && cfg.RabbitMQ.Enabled {
		logger.Info("Initializing RabbitMQ consumer...")
		consumer, err := messaging.NewConsumer(cfg.RabbitMQ.URL, portfolioRepo, logger)
		if err != nil {
			logger.Warnf("Failed to initialize RabbitMQ consumer: %v - running without portfolio updates", err)
		} else {
			logger.Info("✅ RabbitMQ consumer initialized")

			// Start consumer in a goroutine
			ctx := context.Background()
			go func() {
				if err := consumer.Start(ctx); err != nil {
					logger.Errorf("Portfolio consumer error: %v", err)
				}
			}()

			// Handle graceful shutdown
			defer func() {
				if err := consumer.Stop(); err != nil {
					logger.Errorf("Error stopping consumer: %v", err)
				}
			}()
		}
	}

	port := cfg.Server.Port
	if port == 0 {
		port = 8080 // Use 8080 to match docker-compose.yml mapping
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	logger.WithField("port", port).Info("🚀 HTTP server started")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Failed to start server: ", err)
		os.Exit(1)
	}
}
