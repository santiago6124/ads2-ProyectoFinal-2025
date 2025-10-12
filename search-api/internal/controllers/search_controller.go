package controllers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"search-api/internal/dto"
	"search-api/internal/services"
)

// SearchController handles search-related HTTP endpoints
type SearchController struct {
	searchService *services.SearchService
	logger        *logrus.Logger
}

// NewSearchController creates a new search controller
func NewSearchController(searchService *services.SearchService, logger *logrus.Logger) *SearchController {
	return &SearchController{
		searchService: searchService,
		logger:        logger,
	}
}

// Search handles POST /api/v1/search
func (sc *SearchController) Search(c *gin.Context) {
	startTime := time.Now()

	var req dto.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sc.logger.WithFields(logrus.Fields{
			"error": err,
			"path":  c.Request.URL.Path,
		}).Error("Invalid search request")

		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid search parameters: "+err.Error(),
			nil,
		))
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		sc.logger.WithFields(logrus.Fields{
			"error":   err,
			"request": req,
		}).Error("Search request validation failed")

		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"VALIDATION_ERROR",
			"Request validation failed: "+err.Error(),
			map[string]interface{}{
				"query": req.Query,
				"page":  req.Page,
				"limit": req.Limit,
			},
		))
		return
	}

	// Execute search with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := sc.searchService.Search(ctx, &req)
	if err != nil {
		sc.logger.WithFields(logrus.Fields{
			"error":   err,
			"request": req,
			"time":    time.Since(startTime),
		}).Error("Search execution failed")

		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"SEARCH_ERROR",
			"Search execution failed",
			map[string]interface{}{
				"query": req.Query,
			},
		))
		return
	}

	// Log successful search
	sc.logger.WithFields(logrus.Fields{
		"query":        req.Query,
		"results":      len(response.Results),
		"total":        response.Pagination.Total,
		"time":         time.Since(startTime),
		"cache_hit":    response.QueryInfo.CacheHit,
		"user_agent":   c.GetHeader("User-Agent"),
		"ip":           c.ClientIP(),
	}).Info("Search completed")

	c.JSON(http.StatusOK, response)
}

// GetTrending handles GET /api/v1/trending
func (sc *SearchController) GetTrending(c *gin.Context) {
	startTime := time.Now()

	// Parse query parameters
	period := c.DefaultQuery("period", "24h")
	limitStr := c.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		sc.logger.WithFields(logrus.Fields{
			"limit": limitStr,
			"error": err,
		}).Error("Invalid limit parameter")

		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_PARAMETER",
			"Invalid limit parameter, must be between 1 and 100",
			map[string]interface{}{
				"limit": limitStr,
			},
		))
		return
	}

	req := &dto.TrendingRequest{
		Period: period,
		Limit:  limit,
	}

	if err := req.Validate(); err != nil {
		sc.logger.WithFields(logrus.Fields{
			"error": err,
			"req":   req,
		}).Error("Trending request validation failed")

		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"VALIDATION_ERROR",
			"Trending request validation failed: "+err.Error(),
			map[string]interface{}{
				"period": period,
				"limit":  limit,
			},
		))
		return
	}

	// Execute trending search
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	response, err := sc.searchService.GetTrending(ctx, req)
	if err != nil {
		sc.logger.WithFields(logrus.Fields{
			"error":  err,
			"period": period,
			"limit":  limit,
			"time":   time.Since(startTime),
		}).Error("Trending search failed")

		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"TRENDING_ERROR",
			"Failed to fetch trending cryptocurrencies",
			map[string]interface{}{
				"period": period,
			},
		))
		return
	}

	sc.logger.WithFields(logrus.Fields{
		"period":  period,
		"limit":   limit,
		"results": len(response.Trending),
		"time":    time.Since(startTime),
	}).Info("Trending search completed")

	c.JSON(http.StatusOK, response)
}

// GetSuggestions handles GET /api/v1/suggestions
func (sc *SearchController) GetSuggestions(c *gin.Context) {
	startTime := time.Now()

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"MISSING_PARAMETER",
			"Query parameter 'q' is required",
			nil,
		))
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50 {
		sc.logger.WithFields(logrus.Fields{
			"limit": limitStr,
			"error": err,
		}).Error("Invalid limit parameter for suggestions")

		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"INVALID_PARAMETER",
			"Invalid limit parameter, must be between 1 and 50",
			map[string]interface{}{
				"limit": limitStr,
			},
		))
		return
	}

	req := &dto.SuggestionRequest{
		Query: query,
		Limit: limit,
	}

	// Execute suggestions search
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := sc.searchService.GetSuggestions(ctx, req)
	if err != nil {
		sc.logger.WithFields(logrus.Fields{
			"error": err,
			"query": query,
			"limit": limit,
			"time":  time.Since(startTime),
		}).Error("Suggestions search failed")

		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"SUGGESTIONS_ERROR",
			"Failed to fetch suggestions",
			map[string]interface{}{
				"query": query,
			},
		))
		return
	}

	sc.logger.WithFields(logrus.Fields{
		"query":       query,
		"limit":       limit,
		"suggestions": len(response.Suggestions),
		"time":        time.Since(startTime),
	}).Debug("Suggestions search completed")

	c.JSON(http.StatusOK, response)
}

// GetCryptoByID handles GET /api/v1/crypto/:id
func (sc *SearchController) GetCryptoByID(c *gin.Context) {
	startTime := time.Now()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			"MISSING_PARAMETER",
			"Crypto ID is required",
			nil,
		))
		return
	}

	// Execute crypto lookup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	crypto, err := sc.searchService.GetCryptoByID(ctx, id)
	if err != nil {
		sc.logger.WithFields(logrus.Fields{
			"error": err,
			"id":    id,
			"time":  time.Since(startTime),
		}).Error("Crypto lookup failed")

		c.JSON(http.StatusNotFound, dto.NewErrorResponse(
			"CRYPTO_NOT_FOUND",
			"Cryptocurrency not found",
			map[string]interface{}{
				"id": id,
			},
		))
		return
	}

	sc.logger.WithFields(logrus.Fields{
		"id":     id,
		"symbol": crypto.Symbol,
		"name":   crypto.Name,
		"time":   time.Since(startTime),
	}).Debug("Crypto lookup completed")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    crypto,
		"meta": gin.H{
			"execution_time_ms": time.Since(startTime).Milliseconds(),
		},
	})
}

// GetFilters handles GET /api/v1/filters
func (sc *SearchController) GetFilters(c *gin.Context) {
	startTime := time.Now()

	// Execute filters lookup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filters, err := sc.searchService.GetFilters(ctx)
	if err != nil {
		sc.logger.WithFields(logrus.Fields{
			"error": err,
			"time":  time.Since(startTime),
		}).Error("Filters lookup failed")

		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			"FILTERS_ERROR",
			"Failed to fetch available filters",
			nil,
		))
		return
	}

	sc.logger.WithFields(logrus.Fields{
		"categories": len(filters.Categories),
		"time":       time.Since(startTime),
	}).Debug("Filters lookup completed")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    filters,
		"meta": gin.H{
			"execution_time_ms": time.Since(startTime).Milliseconds(),
		},
	})
}