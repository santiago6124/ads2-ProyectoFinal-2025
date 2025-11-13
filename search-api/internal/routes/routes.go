package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"search-api/internal/controllers"
	"search-api/internal/middleware"
	"search-api/internal/services"
)

// SetupRoutes configures all API routes
func SetupRoutes(
	router *gin.Engine,
	searchService *services.SearchService,
	logger *logrus.Logger,
) {
	// Initialize controllers
	searchController := controllers.NewSearchController(searchService, logger)
	adminController := controllers.NewAdminController(searchService, logger)

	// Global middleware
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(logger))
	router.Use(gin.Recovery())

	// API v1 routes
	v1 := router.Group("/api/v1")

	// Public search endpoints with rate limiting
	searchGroup := v1.Group("")
	searchGroup.Use(middleware.SearchRateLimit())
	{
		// Core search endpoints
		searchGroup.POST("/search", searchController.Search)
		searchGroup.GET("/trending", searchController.GetTrending)
		searchGroup.GET("/suggestions", searchController.GetSuggestions)
		searchGroup.GET("/crypto/:id", searchController.GetCryptoByID)
		searchGroup.GET("/orders/:id", searchController.GetOrderByID)
		searchGroup.GET("/filters", searchController.GetFilters)

		// Health check (public)
		searchGroup.GET("/health", adminController.GetHealth)
	}

	// Admin endpoints with stricter rate limiting
	adminGroup := v1.Group("/admin")
	adminGroup.Use(middleware.AdminRateLimit())
	{
		// Cache management
		adminGroup.POST("/cache/clear", adminController.ClearCache)
		adminGroup.POST("/cache/warm", adminController.WarmCache)
		adminGroup.GET("/cache/stats", adminController.GetCacheStats)

		// System management
		adminGroup.POST("/reindex", adminController.ReindexData)
		adminGroup.GET("/system", adminController.GetSystemInfo)
		adminGroup.GET("/metrics", adminController.GetMetrics)
	}

	// Add a catch-all route for 404s
	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Endpoint not found",
				"path":    c.Request.URL.Path,
			},
			"success": false,
		})
	})

	// Add a method not allowed handler
	router.NoMethod(func(c *gin.Context) {
		c.JSON(405, gin.H{
			"error": gin.H{
				"code":    "METHOD_NOT_ALLOWED",
				"message": "Method not allowed for this endpoint",
				"method":  c.Request.Method,
				"path":    c.Request.URL.Path,
			},
			"success": false,
		})
	})
}