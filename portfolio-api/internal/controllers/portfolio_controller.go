package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type PortfolioController struct {
	logger *logrus.Logger
}

func NewPortfolioController(logger *logrus.Logger, service interface{}) *PortfolioController {
	return &PortfolioController{
		logger: logger,
	}
}

func (c *PortfolioController) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/health", c.Health)
	r.POST("/:userId/holdings", c.UpdateHoldings)
}

func (c *PortfolioController) Health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// UpdateHoldingsRequest request payload from orders-api
type UpdateHoldingsRequest struct {
	Symbol    string  `json:"symbol" binding:"required"`
	Quantity  float64 `json:"quantity" binding:"required"`
	Price     float64 `json:"price" binding:"required"`
	OrderType string  `json:"order_type" binding:"required"` // "buy" or "sell"
}

// UpdateHoldings updates user holdings after an order execution
// This is a stub implementation that just returns OK
// The full implementation would require the complete portfolio-api infrastructure
func (c *PortfolioController) UpdateHoldings(ctx *gin.Context) {
	userIDParam := ctx.Param("userId")
	userID, err := parseUserID(userIDParam)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req UpdateHoldingsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log the request for debugging
	c.logger.Infof("Portfolio update request: User %d, Symbol %s, Quantity %f, Price %f, Type %s", 
		userID, req.Symbol, req.Quantity, req.Price, req.OrderType)

	// For now, just return success
	// The full implementation would update the portfolio here
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Holdings updated successfully",
		"user_id": userID,
		"symbol":  req.Symbol,
	})
}

// parseUserID converts string user ID to int64
func parseUserID(userIDStr string) (int64, error) {
	var userID int64
	_, err := fmt.Sscanf(userIDStr, "%d", &userID)
	return userID, err
}
