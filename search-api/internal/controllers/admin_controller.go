package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"search-api/internal/dto"
	"search-api/internal/services"
)

// AdminController handles administrative HTTP endpoints
type AdminController struct {
	searchService *services.SearchService
	logger        *logrus.Logger
}

// NewAdminController creates a new admin controller
func NewAdminController(searchService *services.SearchService, logger *logrus.Logger) *AdminController {
	return &AdminController{
		searchService: searchService,
		logger:        logger,
	}
}

// GetHealth handles GET /api/v1/health
func (ac *AdminController) GetHealth(c *gin.Context) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health := ac.searchService.GetHealthStatus(ctx)

	var httpStatus int
	if health["overall_healthy"].(bool) {
		httpStatus = http.StatusOK
	} else {
		httpStatus = http.StatusServiceUnavailable
	}

	ac.logger.WithFields(logrus.Fields{
		"status":           httpStatus,
		"solr_healthy":     health["solr_healthy"],
		"cache_healthy":    health["cache_healthy"],
		"document_count":   health["document_count"],
		"execution_time":   time.Since(startTime),
	}).Info("Health check completed")

	c.JSON(httpStatus, gin.H{
		"success": health["overall_healthy"],
		"data":    health,
		"meta": gin.H{
			"execution_time_ms": time.Since(startTime).Milliseconds(),
			"timestamp":         time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// GetMetrics handles GET /api/v1/metrics
func (ac *AdminController) GetMetrics(c *gin.Context) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	metrics, err := ac.searchService.GetMetrics(ctx)
	if err != nil {
		ac.logger.WithFields(logrus.Fields{
			"error": err,
			"time":  time.Since(startTime),
		}).Error("Failed to get metrics")

		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"METRICS_ERROR",
			"Failed to retrieve metrics",
			nil,
		))
		return
	}

	ac.logger.WithFields(logrus.Fields{
		"total_searches":       metrics.TotalSearches,
		"cache_hit_rate":       metrics.CacheHitRate,
		"average_response_time": metrics.AverageResponseTime,
		"execution_time":       time.Since(startTime),
	}).Debug("Metrics retrieved")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
		"meta": gin.H{
			"execution_time_ms": time.Since(startTime).Milliseconds(),
			"timestamp":         time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// ClearCache handles POST /api/v1/admin/cache/clear
func (ac *AdminController) ClearCache(c *gin.Context) {
	startTime := time.Now()

	var req struct {
		Pattern string `json:"pattern,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ac.logger.WithFields(logrus.Fields{
			"error": err,
			"path":  c.Request.URL.Path,
		}).Error("Invalid cache clear request")

		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid cache clear parameters: "+err.Error(),
			nil,
		))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	if req.Pattern != "" {
		err = ac.searchService.InvalidateCache(ctx, req.Pattern)
	} else {
		// Clear all cache if no pattern specified
		err = ac.searchService.InvalidateCache(ctx, "*")
	}

	if err != nil {
		ac.logger.WithFields(logrus.Fields{
			"error":   err,
			"pattern": req.Pattern,
			"time":    time.Since(startTime),
		}).Error("Cache clear failed")

		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"CACHE_ERROR",
			"Failed to clear cache",
			map[string]interface{}{
				"pattern": req.Pattern,
			},
		))
		return
	}

	ac.logger.WithFields(logrus.Fields{
		"pattern": req.Pattern,
		"time":    time.Since(startTime),
	}).Info("Cache cleared successfully")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache cleared successfully",
		"data": gin.H{
			"pattern": req.Pattern,
		},
		"meta": gin.H{
			"execution_time_ms": time.Since(startTime).Milliseconds(),
			"timestamp":         time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// WarmCache handles POST /api/v1/admin/cache/warm
func (ac *AdminController) WarmCache(c *gin.Context) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := ac.searchService.WarmCache(ctx)
	if err != nil {
		ac.logger.WithFields(logrus.Fields{
			"error": err,
			"time":  time.Since(startTime),
		}).Error("Cache warming failed")

		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"CACHE_WARM_ERROR",
			"Failed to warm cache",
			nil,
		))
		return
	}

	ac.logger.WithFields(logrus.Fields{
		"time": time.Since(startTime),
	}).Info("Cache warming completed")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache warming completed successfully",
		"meta": gin.H{
			"execution_time_ms": time.Since(startTime).Milliseconds(),
			"timestamp":         time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// GetCacheStats handles GET /api/v1/admin/cache/stats
func (ac *AdminController) GetCacheStats(c *gin.Context) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health := ac.searchService.GetHealthStatus(ctx)
	cacheStats := health["cache_stats"]

	ac.logger.WithFields(logrus.Fields{
		"time": time.Since(startTime),
	}).Debug("Cache stats retrieved")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cacheStats,
		"meta": gin.H{
			"execution_time_ms": time.Since(startTime).Milliseconds(),
			"timestamp":         time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// ReindexData handles POST /api/v1/admin/reindex
func (ac *AdminController) ReindexData(c *gin.Context) {
	startTime := time.Now()

	var req struct {
		Force bool `json:"force,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Ignore binding errors for this endpoint since force is optional
		req.Force = false
	}

	ac.logger.WithFields(logrus.Fields{
		"force":      req.Force,
		"started_at": startTime,
	}).Info("Reindexing started")

	// This would typically trigger a background job to reindex data
	// For now, we'll simulate the process
	go func() {
		time.Sleep(2 * time.Second) // Simulate reindexing work
		ac.logger.Info("Reindexing process completed")
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"message": "Reindexing process started",
		"data": gin.H{
			"force":      req.Force,
			"started_at": startTime.UTC().Format(time.RFC3339),
		},
		"meta": gin.H{
			"execution_time_ms": time.Since(startTime).Milliseconds(),
			"timestamp":         time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// GetSystemInfo handles GET /api/v1/admin/system
func (ac *AdminController) GetSystemInfo(c *gin.Context) {
	startTime := time.Now()

	systemInfo := gin.H{
		"service": gin.H{
			"name":        "search-api",
			"version":     "1.0.0",
			"environment": "development", // This would come from config
			"build_time":  "2025-01-15T10:00:00Z", // This would come from build info
		},
		"runtime": gin.H{
			"go_version": "1.21",
			"uptime":     time.Since(startTime).String(), // This would track actual uptime
		},
		"dependencies": gin.H{
			"solr":      "9.x",
			"memcached": "1.6.x",
			"rabbitmq":  "3.12.x",
		},
	}

	ac.logger.WithFields(logrus.Fields{
		"time": time.Since(startTime),
	}).Debug("System info retrieved")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    systemInfo,
		"meta": gin.H{
			"execution_time_ms": time.Since(startTime).Milliseconds(),
			"timestamp":         time.Now().UTC().Format(time.RFC3339),
		},
	})
}