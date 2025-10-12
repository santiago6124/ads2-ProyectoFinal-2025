package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "wallet-api",
			"timestamp": time.Now().UTC(),
		})
	})

	// API placeholder
	api := router.Group("/api")
	{
		api.GET("/wallet/:userId", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Wallet API - Implementation in progress",
				"userId":  c.Param("userId"),
			})
		})
	}

	fmt.Printf("Wallet API listening on port %s\n", port)
	if err := router.Run(":" + port); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
