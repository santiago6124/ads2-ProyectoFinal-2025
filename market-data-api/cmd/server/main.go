package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8004"
	}

	// Set Gin to debug mode for development
	gin.SetMode(gin.DebugMode)

	router := gin.Default()

	// Add CORS middleware - must be before routes
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"service":   "market-data-api",
		})
	})

	// API endpoints
	api := router.Group("/api/v1")
	{
		api.GET("/prices", getPrices)
		api.GET("/prices/:symbol", getPriceBySymbol)
		api.GET("/history/:symbol", getPriceHistory)
	}

	log.Printf("Market Data API starting on 0.0.0.0:%s", port)
	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func getPrices(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Get all prices endpoint",
		"data":    []string{"BTC", "ETH", "ADA"},
	})
}

func getPriceBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")
	c.JSON(http.StatusOK, gin.H{
		"symbol": symbol,
		"price":  50000.00,
		"timestamp": time.Now().Unix(),
	})
}

func getPriceHistory(c *gin.Context) {
	symbol := c.Param("symbol")
	c.JSON(http.StatusOK, gin.H{
		"symbol": symbol,
		"history": []gin.H{
			{"timestamp": time.Now().Add(-1 * time.Hour).Unix(), "price": 49500.00},
			{"timestamp": time.Now().Unix(), "price": 50000.00},
		},
	})
}
