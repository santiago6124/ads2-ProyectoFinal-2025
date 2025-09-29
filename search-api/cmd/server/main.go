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
	solrClient, err := solr.NewClient(&solr.Config{
		BaseURL:           cfg.Solr.BaseURL,
		Collection:        cfg.Solr.Collection,
		ConnectionTimeout: time.Duration(cfg.Solr.TimeoutMs) * time.Millisecond,
		RequestTimeout:    time.Duration(cfg.Solr.TimeoutMs) * time.Millisecond,
		MaxRetries:        cfg.Solr.MaxRetries,
		RetryDelay:        time.Second,
	}, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create Solr client")
	}

	// Test Solr connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := solrClient.Ping(ctx); err != nil {
		logger.WithError(err).Warn("Solr connection test failed - service will start but may not function properly")
	} else {
		logger.Info("Solr connection successful")
	}

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
	if err := cacheManager.Ping(ctx); err != nil {
		logger.WithError(err).Warn("Cache connection test failed - service will continue with local cache only")
	} else {
		logger.Info("Cache connection successful")
	}

	// Initialize repositories
	solrRepo := repositories.NewSolrRepository(solrClient, logger)
	cacheRepo := repositories.NewCacheRepository(cacheManager)

	// Initialize trending service
	trendingConfig := services.DefaultTrendingConfig()
	trendingService := services.NewTrendingService(solrRepo, trendingConfig, logger)

	// Start trending service
	go func() {
		if err := trendingService.Start(ctx); err != nil {
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
		rabbitConfig := &messaging.Config{
			URL:             cfg.RabbitMQ.URL,
			ExchangeName:    cfg.RabbitMQ.ExchangeName,
			QueueName:       cfg.RabbitMQ.QueueName,
			RoutingKeys:     cfg.RabbitMQ.RoutingKeys,
			WorkerPoolSize:  cfg.RabbitMQ.WorkerPoolSize,
			RetryAttempts:   cfg.RabbitMQ.RetryAttempts,
			RetryDelay:      time.Duration(cfg.RabbitMQ.RetryDelayMs) * time.Millisecond,
			DeadLetterQueue: cfg.RabbitMQ.DeadLetterQueue,
		}

		trendingEventHandler := services.NewTrendingEventHandler(trendingService, logger)
		consumer = messaging.NewConsumer(rabbitConfig, trendingEventHandler, logger)

		// Start RabbitMQ consumer
		go func() {
			if err := consumer.Start(ctx); err != nil {
				logger.WithError(err).Error("Failed to start RabbitMQ consumer")
			}
		}()
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