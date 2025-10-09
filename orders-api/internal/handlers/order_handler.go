package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"orders-api/internal/dto"
	"orders-api/internal/models"
	"orders-api/internal/services"
)

type OrderHandler struct {
	orderService services.OrderService
}

func NewOrderHandler(orderService services.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

type CreateOrderRequest struct {
	Type         string `json:"type" binding:"required,oneof=buy sell"`
	OrderKind    string `json:"order_kind" binding:"required,oneof=market limit stop"`
	CryptoSymbol string `json:"crypto_symbol" binding:"required"`
	Quantity     string `json:"quantity" binding:"required"`
	OrderPrice   string `json:"order_price,omitempty"`
	StopPrice    string `json:"stop_price,omitempty"`
	TimeInForce  string `json:"time_in_force,omitempty"`
	ExpiresAt    string `json:"expires_at,omitempty"`
}

type UpdateOrderRequest struct {
	OrderPrice  string `json:"order_price,omitempty"`
	StopPrice   string `json:"stop_price,omitempty"`
	Quantity    string `json:"quantity,omitempty"`
	TimeInForce string `json:"time_in_force,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

type OrderResponse struct {
	ID             string                 `json:"id"`
	UserID         int                    `json:"user_id"`
	Type           string                 `json:"type"`
	OrderKind      string                 `json:"order_kind"`
	Status         string                 `json:"status"`
	CryptoSymbol   string                 `json:"crypto_symbol"`
	Quantity       string                 `json:"quantity"`
	OrderPrice     string                 `json:"order_price"`
	StopPrice      string                 `json:"stop_price,omitempty"`
	FilledQuantity string                 `json:"filled_quantity"`
	AveragePrice   string                 `json:"average_price"`
	TimeInForce    string                 `json:"time_in_force"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type OrderListResponse struct {
	Orders     []*OrderResponse   `json:"orders"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int64              `json:"total_pages"`
	Summary    *dto.OrdersSummary `json:"summary,omitempty"`
}

type ExecutionResponse struct {
	OrderID         string                 `json:"order_id"`
	Success         bool                   `json:"success"`
	Error           string                 `json:"error,omitempty"`
	ExecutionTime   time.Duration          `json:"execution_time"`
	UserValidation  map[string]interface{} `json:"user_validation,omitempty"`
	BalanceCheck    map[string]interface{} `json:"balance_check,omitempty"`
	MarketPrice     map[string]interface{} `json:"market_price,omitempty"`
	FeeCalculation  map[string]interface{} `json:"fee_calculation,omitempty"`
	Steps           []map[string]interface{} `json:"steps,omitempty"`
}



func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quantity format"})
		return
	}

	if quantity.LessThanOrEqual(decimal.Zero) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity must be greater than zero"})
		return
	}

	var orderPrice decimal.Decimal
	if req.OrderPrice != "" {
		orderPrice, err = decimal.NewFromString(req.OrderPrice)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order price format"})
			return
		}
	}



	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	dtoReq := &dto.CreateOrderRequest{
		Type:         models.OrderType(req.Type),
		CryptoSymbol: req.CryptoSymbol,
		Quantity:     quantity,
		OrderType:    models.OrderKind(req.OrderKind),
		LimitPrice:   &orderPrice,
	}

	createdOrder, err := h.orderService.CreateOrder(ctx, dtoReq, userID.(int), &models.OrderMetadata{
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := h.convertToOrderResponse(createdOrder)
	c.JSON(http.StatusCreated, response)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order ID is required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	order, err := h.orderService.GetOrder(ctx, orderID, userID.(int))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	if order.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	response := h.convertToOrderResponse(order)
	c.JSON(http.StatusOK, response)
}

func (h *OrderHandler) ListUserOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	status := c.Query("status")
	orderType := c.Query("type")
	symbol := c.Query("symbol")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	var statusPtr *models.OrderStatus
	if status != "" {
		statusPtr = (*models.OrderStatus)(&status)
	}
	
	var symbolPtr *string
	if symbol != "" {
		symbolPtr = &symbol
	}
	
	var typePtr *models.OrderType
	if orderType != "" {
		typePtr = (*models.OrderType)(&orderType)
	}

	filter := &dto.OrderFilterRequest{
		Status:       statusPtr,
		CryptoSymbol: symbolPtr,
		Type:         typePtr,
		Limit:        pageSize,
		Page:         page,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	orders, total, summary, err := h.orderService.ListUserOrders(ctx, userID.(int), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]*OrderResponse, len(orders))
	for i, order := range orders {
		responses[i] = h.convertToOrderResponse(&order)
	}

	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)

	response := &OrderListResponse{
		Orders:     responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Summary:    summary,
	}

	c.JSON(http.StatusOK, response)
}

func (h *OrderHandler) UpdateOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order ID is required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	existingOrder, err := h.orderService.GetOrder(ctx, orderID, userID.(int))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	if existingOrder.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	updates := make(map[string]interface{})

	if req.OrderPrice != "" {
		orderPrice, err := decimal.NewFromString(req.OrderPrice)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order price format"})
			return
		}
		updates["order_price"] = orderPrice
	}

	if req.StopPrice != "" {
		stopPrice, err := decimal.NewFromString(req.StopPrice)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stop price format"})
			return
		}
		updates["stop_price"] = stopPrice
	}

	if req.Quantity != "" {
		quantity, err := decimal.NewFromString(req.Quantity)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quantity format"})
			return
		}
		if quantity.LessThanOrEqual(decimal.Zero) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "quantity must be greater than zero"})
			return
		}
		updates["quantity"] = quantity
	}

	if req.TimeInForce != "" {
		updates["time_in_force"] = models.TimeInForce(req.TimeInForce)
	}

	if req.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expires_at format, use RFC3339"})
			return
		}
		updates["expires_at"] = &expiresAt
	}

	dtoReq := &dto.UpdateOrderRequest{}
	// Convert updates map to DTO fields as needed
	
	updatedOrder, err := h.orderService.UpdateOrder(ctx, orderID, dtoReq, userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := h.convertToOrderResponse(updatedOrder)
	c.JSON(http.StatusOK, response)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order ID is required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	reason := c.DefaultQuery("reason", "cancelled by user")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	existingOrder, err := h.orderService.GetOrder(ctx, orderID, userID.(int))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	if existingOrder.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	err = h.orderService.CancelOrder(ctx, orderID, userID.(int), reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order cancelled successfully"})
}

func (h *OrderHandler) ExecuteOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order ID is required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	existingOrder, err := h.orderService.GetOrder(ctx, orderID, userID.(int))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	if existingOrder.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	executionResult, err := h.orderService.ExecuteOrder(ctx, orderID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := h.convertToExecutionResponse(executionResult)
	c.JSON(http.StatusOK, response)
}

func (h *OrderHandler) convertToOrderResponse(order *models.Order) *OrderResponse {
	response := &OrderResponse{
		ID:             order.ID.Hex(),
		UserID:         order.UserID,
		Type:           string(order.Type),
		OrderKind:      string(order.OrderKind),
		Status:         string(order.Status),
		CryptoSymbol:   order.CryptoSymbol,
		Quantity:       order.Quantity.String(),
		OrderPrice:     order.OrderPrice.String(),
		FilledQuantity: order.FilledQuantity.String(),
		AveragePrice:   order.AveragePrice.String(),
		TimeInForce:    string(order.TimeInForce),
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
		ExpiresAt:      order.ExpiresAt,
	}

	if !order.StopPrice.IsZero() {
		response.StopPrice = order.StopPrice.String()
	}

	if order.Metadata.UserAgent != "" || order.Metadata.IPAddress != "" {
		response.Metadata = map[string]interface{}{
			"user_agent": order.Metadata.UserAgent,
			"ip_address": order.Metadata.IPAddress,
		}
	}

	return response
}

func (h *OrderHandler) convertToExecutionResponse(result *models.ExecutionResult) *ExecutionResponse {
	response := &ExecutionResponse{
		OrderID:       result.OrderID,
		Success:       result.Success,
		Error:         result.Error,
		ExecutionTime: result.ExecutionTime,
	}

	if result.UserValidation != nil {
		response.UserValidation = map[string]interface{}{
			"is_valid": result.UserValidation.IsValid,
			"user_id":  result.UserValidation.UserID,
			"message":  result.UserValidation.Message,
		}
	}

	if result.BalanceCheck != nil {
		response.BalanceCheck = map[string]interface{}{
			"has_sufficient": result.BalanceCheck.HasSufficient,
			"available":      result.BalanceCheck.Available.String(),
			"required":       result.BalanceCheck.Required.String(),
			"currency":       result.BalanceCheck.Currency,
			"message":        result.BalanceCheck.Message,
		}
	}

	if result.MarketPrice != nil {
		response.MarketPrice = map[string]interface{}{
			"symbol":          result.MarketPrice.Symbol,
			"market_price":    result.MarketPrice.MarketPrice.String(),
			"execution_price": result.MarketPrice.ExecutionPrice.String(),
			"slippage":        result.MarketPrice.Slippage.String(),
			"slippage_perc":   result.MarketPrice.SlippagePerc.String(),
			"volume_24h":      result.MarketPrice.Volume24h,
		}
	}

	if result.FeeCalculation != nil {
		response.FeeCalculation = map[string]interface{}{
			"base_fee":       result.FeeCalculation.BaseFee.String(),
			"percentage_fee": result.FeeCalculation.PercentageFee.String(),
			"total_fee":      result.FeeCalculation.TotalFee.String(),
			"fee_percentage": result.FeeCalculation.FeePercentage.String(),
			"fee_type":       result.FeeCalculation.FeeType,
		}
	}

	if len(result.ProcessingSteps) > 0 {
		steps := make([]map[string]interface{}, len(result.ProcessingSteps))
		for i, step := range result.ProcessingSteps {
			steps[i] = map[string]interface{}{
				"step":        step.Step,
				"status":      step.Status,
				"start_time":  step.StartTime,
				"end_time":    step.EndTime,
				"duration":    step.Duration,
				"error":       step.Error,
			}
		}
		response.Steps = steps
	}

	return response
}