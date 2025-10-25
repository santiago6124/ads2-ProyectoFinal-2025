package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"portfolio-api/internal/config"
	"portfolio-api/internal/controllers"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	logger.WithField("service", "portfolio-api").Info("Starting Portfolio API service...")

	// Setup HTTP server
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "portfolio-api",
			"timestamp": time.Now().UTC(),
		})
	})

	// Initialize portfolio controller
	// Note: This is a simplified implementation - in production, you would initialize
	// all dependencies (database, services, etc.)
	controller := controllers.NewPortfolioController(logger, nil) // nil for now
	
	// API routes
	api := router.Group("/api")
	{
		portfolio := api.Group("/portfolio")
		controller.RegisterRoutes(portfolio)
	}

	port := cfg.Server.Port
	if port == 0 {
		port = 8080 // Use 8080 to match docker-compose.yml mapping
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	logger.WithField("port", port).Info("HTTP server started")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Failed to start server: ", err)
		os.Exit(1)
	}
}
