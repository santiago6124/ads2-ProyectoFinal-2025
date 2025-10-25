package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"orders-api/internal/dto"
	"orders-api/internal/models"
	"orders-api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
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
	OrderKind    string `json:"order_kind" binding:"required,oneof=market limit"`
	CryptoSymbol string `json:"crypto_symbol" binding:"required"`
	Quantity     string `json:"quantity" binding:"required"`
	OrderPrice   string `json:"order_price,omitempty"`
}

type UpdateOrderRequest struct {
	OrderPrice string `json:"order_price,omitempty"`
	Quantity   string `json:"quantity,omitempty"`
}

type OrderResponse struct {
	ID             string     `json:"id"`
	OrderNumber    string     `json:"order_number"`
	UserID         int        `json:"user_id"`
	Type           string     `json:"type"`
	OrderKind      string     `json:"order_kind"`
	Status         string     `json:"status"`
	CryptoSymbol   string     `json:"crypto_symbol"`
	CryptoName     string     `json:"crypto_name"`
	Quantity       string     `json:"quantity"`
	OrderPrice     string     `json:"order_price"`
	ExecutionPrice string     `json:"execution_price,omitempty"`
	TotalAmount    string     `json:"total_amount"`
	Fee            string     `json:"fee"`
	FeePercentage  string     `json:"fee_percentage"`
	CreatedAt      time.Time  `json:"created_at"`
	ExecutedAt     *time.Time `json:"executed_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CancelledAt    *time.Time `json:"cancelled_at,omitempty"`
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
	OrderID        string                   `json:"order_id"`
	Success        bool                     `json:"success"`
	Error          string                   `json:"error,omitempty"`
	ExecutionTime  time.Duration            `json:"execution_time"`
	UserValidation map[string]interface{}   `json:"user_validation,omitempty"`
	BalanceCheck   map[string]interface{}   `json:"balance_check,omitempty"`
	MarketPrice    map[string]interface{}   `json:"market_price,omitempty"`
	FeeCalculation map[string]interface{}   `json:"fee_calculation,omitempty"`
	Steps          []map[string]interface{} `json:"steps,omitempty"`
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

	// La validación de orderPrice se hace en el DTO
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	dtoReq := &dto.CreateOrderRequest{
		Type:         models.OrderType(req.Type),
		CryptoSymbol: req.CryptoSymbol,
		Quantity:     req.Quantity,
		OrderKind:    models.OrderKind(req.OrderKind),
		LimitPrice:   req.OrderPrice,
	}

	createdOrder, err := h.orderService.CreateOrder(ctx, dtoReq, userID.(int))
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

// UpdateOrder comentado - no está en OrderServiceSimple (sistema simplificado)
/*
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
*/

// ExecuteOrder comentado - no está en OrderServiceSimple (sistema simplificado)
/*
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
*/

func (h *OrderHandler) convertToOrderResponse(order *models.Order) *OrderResponse {
	response := &OrderResponse{
		ID:           order.ID.Hex(),
		OrderNumber:  order.OrderNumber,
		UserID:       order.UserID,
		Type:         string(order.Type),
		OrderKind:    string(order.OrderKind),
		Status:       string(order.Status),
		CryptoSymbol: order.CryptoSymbol,
		CryptoName:   order.CryptoName,
		Quantity:     order.Quantity.String(),
		OrderPrice:   order.Price.String(), // Simplificado: solo Price
		TotalAmount:  order.TotalAmount.String(),
		Fee:          order.Fee.String(),
		FeePercentage: "0.1", // Siempre 0.1% en sistema simplificado
		CreatedAt:    order.CreatedAt,
		UpdatedAt:    order.UpdatedAt,
		ExecutedAt:   order.ExecutedAt,
		// CancelledAt eliminado en modelo simplificado
	}

	return response
}

// convertToExecutionResponse simplificado para modelo básico
func (h *OrderHandler) convertToExecutionResponse(result *models.ExecutionResult) *ExecutionResponse {
	response := &ExecutionResponse{
		OrderID:       result.OrderID,
		Success:       result.Success,
		Error:         result.Error,
		ExecutionTime: result.ExecutionTime,
	}

	// Modelo simplificado no tiene estos detalles
	// Solo retorna información básica
	return response
}
