package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"portfolio-api/internal/clients"
	"portfolio-api/internal/messaging"
	"portfolio-api/internal/repositories"
)

type PortfolioController struct {
	logger          *logrus.Logger
	userClient      *clients.UserClient
	marketClient    *clients.MarketDataClient
	portfolioRepo   repositories.PortfolioRepository
	balancePublisher *messaging.BalancePublisher
	balanceConsumer  *messaging.BalanceResponseConsumer
}

func NewPortfolioController(logger *logrus.Logger, userClient interface{}) *PortfolioController {
	var client *clients.UserClient
	if userClient != nil {
		client = userClient.(*clients.UserClient)
	}
	return &PortfolioController{
		logger:     logger,
		userClient: client,
	}
}

func NewPortfolioControllerWithClients(
	logger *logrus.Logger,
	userClient *clients.UserClient,
	marketClient *clients.MarketDataClient,
	portfolioRepo repositories.PortfolioRepository,
) *PortfolioController {
	return &PortfolioController{
		logger:        logger,
		userClient:    userClient,
		marketClient:  marketClient,
		portfolioRepo: portfolioRepo,
	}
}

func NewPortfolioControllerWithClientsAndMessaging(
	logger *logrus.Logger,
	userClient *clients.UserClient,
	marketClient *clients.MarketDataClient,
	portfolioRepo repositories.PortfolioRepository,
	balancePublisher *messaging.BalancePublisher,
	balanceConsumer *messaging.BalanceResponseConsumer,
) *PortfolioController {
	return &PortfolioController{
		logger:           logger,
		userClient:       userClient,
		marketClient:     marketClient,
		portfolioRepo:    portfolioRepo,
		balancePublisher: balancePublisher,
		balanceConsumer:  balanceConsumer,
	}
}

func (c *PortfolioController) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/health", c.Health)
	r.GET("/:userId", c.GetPortfolio)
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

	c.logger.Infof("📨 Holdings update: User %d, %s %f %s @ $%f",
		userID, req.OrderType, req.Quantity, req.Symbol, req.Price)

	// Validate portfolio repository is available
	if c.portfolioRepo == nil {
		c.logger.Error("Portfolio repository not initialized")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "portfolio service unavailable"})
		return
	}

	requestCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Call the repository method to update holdings
	err = c.portfolioRepo.UpdateHoldingsFromOrder(requestCtx, userID, req.Symbol, req.Quantity, req.Price, req.OrderType)
	if err != nil {
		c.logger.Errorf("Failed to update holdings for user %d: %v", userID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update holdings: %v", err)})
		return
	}

	c.logger.Infof("✅ Holdings updated successfully for user %d: %s %s", userID, req.OrderType, req.Symbol)
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Holdings updated successfully",
		"user_id": userID,
		"symbol":  req.Symbol,
		"order_type": req.OrderType,
	})
}

// GetPortfolio retrieves a user's portfolio
func (c *PortfolioController) GetPortfolio(ctx *gin.Context) {
	userIDParam := ctx.Param("userId")
	userID, err := parseUserID(userIDParam)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	requestCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get user balance - Try RabbitMQ first, fallback to HTTP
	var totalCash string = "100000.00" // default
	balanceFetched := false

	// Try RabbitMQ messaging if available
	if c.balancePublisher != nil && c.balanceConsumer != nil {
		c.logger.Debugf("📤 Requesting balance via RabbitMQ for user %d", userID)
		correlationID, err := c.balancePublisher.RequestBalance(requestCtx, userID)
		if err != nil {
			c.logger.Warnf("Failed to publish balance request: %v - falling back to HTTP", err)
		} else {
			// Wait for response with 5 second timeout
			response, err := c.balanceConsumer.WaitForResponse(correlationID, 5*time.Second)
			if err != nil {
				c.logger.Warnf("Timeout waiting for balance response: %v - falling back to HTTP", err)
			} else if response.Success {
				totalCash = response.Balance
				balanceFetched = true
				c.logger.Debugf("✅ Received balance via RabbitMQ for user %d: %s", userID, totalCash)
			} else {
				c.logger.Warnf("Balance request failed: %s - falling back to HTTP", response.Error)
			}
		}
	}

	// Fallback to HTTP if RabbitMQ failed or not available
	if !balanceFetched && c.userClient != nil {
		c.logger.Debugf("⚠️ Using HTTP fallback to fetch balance for user %d", userID)
		balance, err := c.userClient.GetUserBalance(requestCtx, userID)
		if err == nil {
			totalCash = balance.String()
			c.logger.Debugf("✅ Received balance via HTTP for user %d: %s", userID, totalCash)
		} else {
			c.logger.Warnf("Failed to get user balance for user %d: %v", userID, err)
		}
	}

	// Try to get portfolio from database
	var holdings []gin.H
	var totalInvested float64
	var totalHoldingsValue float64

	if c.portfolioRepo != nil {
		portfolio, err := c.portfolioRepo.GetByUserID(requestCtx, userID)
		if err == nil && portfolio != nil {
			// Fetch current prices for all holdings
			for _, holding := range portfolio.Holdings {
				currentPrice := holding.CurrentPrice.InexactFloat64()

				// Try to get latest price from market data API
				if c.marketClient != nil {
					prices, err := c.marketClient.GetPrices(requestCtx, []string{holding.Symbol})
					if err == nil && len(prices) > 0 {
						if price, ok := prices[holding.Symbol]; ok {
							currentPrice = price.Price.InexactFloat64()
						}
					}
				}

				quantity := holding.Quantity.InexactFloat64()
				avgPrice := holding.AverageBuyPrice.InexactFloat64()
				totalValue := currentPrice * quantity
				invested := avgPrice * quantity
				profitLoss := totalValue - invested
				profitLossPct := 0.0
				if invested > 0 {
					profitLossPct = (profitLoss / invested) * 100
				}

				totalHoldingsValue += totalValue
				totalInvested += invested

				holdings = append(holdings, gin.H{
					"symbol":                   holding.Symbol,
					"name":                     holding.Name,
					"quantity":                 fmt.Sprintf("%.8f", quantity),
					"average_buy_price":        fmt.Sprintf("%.2f", avgPrice),
					"current_price":            fmt.Sprintf("%.2f", currentPrice),
					"current_value":            fmt.Sprintf("%.2f", totalValue),
					"total_value":              fmt.Sprintf("%.2f", totalValue),
					"profit_loss":              fmt.Sprintf("%.2f", profitLoss),
					"profit_loss_percentage":   fmt.Sprintf("%.2f", profitLossPct),
					"allocation_percentage":    "0", // Will calculate after we know total
				})
			}

			c.logger.Infof("Retrieved portfolio for user %d with %d holdings", userID, len(holdings))
		} else {
			c.logger.Infof("No portfolio found for user %d in database", userID)
		}
	}

	// Calculate totals
	cashFloat := 100000.00 // default
	fmt.Sscanf(totalCash, "%f", &cashFloat)

	totalValue := totalHoldingsValue + cashFloat
	profitLoss := totalValue - (totalInvested + cashFloat)
	profitLossPct := 0.0
	if totalInvested + cashFloat > 0 {
		profitLossPct = (profitLoss / (totalInvested + cashFloat)) * 100
	}

	// Update allocation percentages
	for i := range holdings {
		holdingValue := 0.0
		fmt.Sscanf(holdings[i]["total_value"].(string), "%f", &holdingValue)
		if totalValue > 0 {
			allocationPct := (holdingValue / totalValue) * 100
			holdings[i]["allocation_percentage"] = fmt.Sprintf("%.2f", allocationPct)
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"id":                      fmt.Sprintf("portfolio-%d", userID),
		"user_id":                 userID,
		"total_value":             fmt.Sprintf("%.2f", totalValue),
		"total_invested":          fmt.Sprintf("%.2f", totalInvested),
		"total_cash":              totalCash,
		"profit_loss":             fmt.Sprintf("%.2f", profitLoss),
		"profit_loss_percentage":  fmt.Sprintf("%.2f", profitLossPct),
		"currency":                "USD",
		"holdings":                holdings,
		"performance": gin.H{
			"daily_change":            "0",
			"daily_change_percentage": "0",
		},
	})
}

// parseUserID converts string user ID to int64
func parseUserID(userIDStr string) (int64, error) {
	var userID int64
	_, err := fmt.Sscanf(userIDStr, "%d", &userID)
	return userID, err
}
