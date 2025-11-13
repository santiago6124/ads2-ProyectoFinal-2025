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

	"search-api/internal/cache"
	"search-api/internal/clients"
	"search-api/internal/config"
	"search-api/internal/messaging"
	"search-api/internal/repositories"
	"search-api/internal/routes"
	"search-api/internal/services"
	"search-api/internal/solr"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Load configuration
	cfg := config.Load()

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
		logger.SetLevel(logrus.WarnLevel)
	} else {
		gin.SetMode(gin.DebugMode)
		logger.SetLevel(logrus.DebugLevel)
	}

	logger.WithFields(logrus.Fields{
		"environment": cfg.Environment,
		"port":        cfg.Server.Port,
		"version":     cfg.Version,
	}).Info("Starting Search API server")

	// Initialize Solr client
	solrClient := solr.NewClient(&solr.Config{
		BaseURL:    cfg.Solr.BaseURL,
		Core:       cfg.Solr.Collection,
		Timeout:    time.Duration(cfg.Solr.TimeoutMs) * time.Millisecond,
		MaxRetries: cfg.Solr.MaxRetries,
		RetryDelay: time.Second,
	})

	// Test Solr connection with a short-lived context
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer pingCancel()
	if err := solrClient.Ping(pingCtx); err != nil {
		logger.WithError(err).Warn("Solr connection test failed - service will start but may not function properly")
	} else {
		logger.Info("Solr connection successful")
	}

	// Create application-wide context for background workers
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	// Initialize cache manager
	cacheConfig := &cache.Config{
		LocalTTL:              time.Duration(cfg.Cache.LocalTTLMinutes) * time.Minute,
		DistributedTTL:        time.Duration(cfg.Cache.DistributedTTLMinutes) * time.Minute,
		MaxLocalSize:          int64(cfg.Cache.MaxLocalSize),
		LocalItemsToPrune:     uint32(cfg.Cache.LocalItemsToPrune),
		MemcachedHosts:        cfg.Cache.MemcachedHosts,
		MemcachedTimeout:      time.Duration(cfg.Cache.MemcachedTimeoutMs) * time.Millisecond,
		MemcachedMaxIdleConns: cfg.Cache.MemcachedMaxIdleConns,
		KeyPrefix:             cfg.Cache.KeyPrefix,
		EnableMetrics:         cfg.Cache.EnableMetrics,
	}

	cacheManager := cache.NewCacheManager(cacheConfig, logger)

	// Test cache connection
	if err := cacheManager.Ping(pingCtx); err != nil {
		logger.WithError(err).Warn("Cache connection test failed - service will continue with local cache only")
	} else {
		logger.Info("Cache connection successful")
	}

	// Initialize repositories
	solrRepo := repositories.NewSolrRepository(solrClient)
	cacheRepo := repositories.NewCacheRepository(cacheManager)

	// Initialize orders-api client
	ordersClientConfig := &clients.OrdersClientConfig{
		BaseURL: cfg.OrdersAPI.BaseURL,
		APIKey:  cfg.OrdersAPI.APIKey,
		Timeout: time.Duration(cfg.OrdersAPI.Timeout) * time.Millisecond,
	}
	ordersClient := clients.NewOrdersClient(ordersClientConfig)

	// Initialize indexing service
	indexingService := services.NewIndexingService(ordersClient, solrRepo, logger)

	// Initialize trending service
	trendingConfig := services.DefaultTrendingConfig()
	trendingService := services.NewTrendingService(solrRepo, trendingConfig, logger)

	// Start trending service
	go func() {
		if err := trendingService.Start(appCtx); err != nil {
			logger.WithError(err).Error("Failed to start trending service")
		}
	}()

	// Initialize search service
	searchService := services.NewSearchService(solrRepo, cacheRepo, trendingService, logger)

	// Warm cache
	go func() {
		warmCtx, warmCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer warmCancel()

		if err := searchService.WarmCache(warmCtx); err != nil {
			logger.WithError(err).Warn("Cache warming failed")
		} else {
			logger.Info("Cache warmed successfully")
		}
	}()

	// Initialize RabbitMQ consumer
	var consumer *messaging.Consumer
	if cfg.RabbitMQ.Enabled {
		rabbitConfig := &messaging.ConsumerConfig{
			URL:           cfg.RabbitMQ.URL,
			ExchangeName:  cfg.RabbitMQ.ExchangeName,
			QueueName:     cfg.RabbitMQ.QueueName,
			RoutingKeys:   cfg.RabbitMQ.RoutingKeys,
			ConsumerTag:   "search-api-consumer",
			PrefetchCount: 10,
			AutoAck:       false,
			WorkerCount:   cfg.RabbitMQ.WorkerPoolSize,
			RetryDelay:    time.Duration(cfg.RabbitMQ.RetryDelayMs) * time.Millisecond,
			MaxRetries:    cfg.RabbitMQ.RetryAttempts,
		}

		trendingEventHandler := services.NewTrendingEventHandler(trendingService, logger)
		var err error
		consumer, err = messaging.NewConsumer(rabbitConfig, trendingEventHandler, indexingService, logger)
		if err != nil {
			logger.WithError(err).Error("Failed to create RabbitMQ consumer")
			consumer = nil
		}

		// Start RabbitMQ consumer
		if consumer != nil {
			go func() {
				if err := consumer.Start(appCtx); err != nil {
					logger.WithError(err).Error("Failed to start RabbitMQ consumer")
				}
			}()
		}
	} else {
		logger.Info("RabbitMQ consumer disabled in configuration")
	}

	// Initialize HTTP server
	router := gin.New()

	// Setup routes
	routes.SetupRoutes(router, searchService, logger)

	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    time.Duration(cfg.Server.ReadTimeoutMs) * time.Millisecond,
		WriteTimeout:   time.Duration(cfg.Server.WriteTimeoutMs) * time.Millisecond,
		IdleTimeout:    time.Duration(cfg.Server.IdleTimeoutMs) * time.Millisecond,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("addr", server.Addr).Info("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Give outstanding requests a deadline for completion
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	}

	// Stop trending service
	if err := trendingService.Stop(); err != nil {
		logger.WithError(err).Error("Failed to stop trending service")
	}

	// Stop RabbitMQ consumer
	if consumer != nil {
		if err := consumer.Stop(); err != nil {
			logger.WithError(err).Error("Failed to stop RabbitMQ consumer")
		}
	}

	// Close cache manager
	if err := cacheManager.Close(); err != nil {
		logger.WithError(err).Error("Failed to close cache manager")
	}

	logger.Info("Server shutdown complete")
}
