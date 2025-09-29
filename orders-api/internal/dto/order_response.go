package dto

import (
	"time"

	"github.com/shopspring/decimal"
	"orders-api/internal/models"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    int    `json:"code,omitempty"`
}

type OrderResponse struct {
	OrderID          string                    `json:"order_id"`
	OrderNumber      string                    `json:"order_number"`
	UserID           int                       `json:"user_id"`
	Type             models.OrderType          `json:"type"`
	Status           models.OrderStatus        `json:"status"`
	CryptoSymbol     string                    `json:"crypto_symbol"`
	CryptoName       string                    `json:"crypto_name"`
	Quantity         decimal.Decimal           `json:"quantity"`
	OrderType        models.OrderKind          `json:"order_type"`
	LimitPrice       *decimal.Decimal          `json:"limit_price,omitempty"`
	OrderPrice       decimal.Decimal           `json:"order_price"`
	ExecutionPrice   *decimal.Decimal          `json:"execution_price,omitempty"`
	TotalAmount      decimal.Decimal           `json:"total_amount"`
	Fee              decimal.Decimal           `json:"fee"`
	CreatedAt        time.Time                 `json:"created_at"`
	ExecutedAt       *time.Time                `json:"executed_at,omitempty"`
	UpdatedAt        time.Time                 `json:"updated_at"`
	ExecutionDetails *models.ExecutionDetails  `json:"execution_details,omitempty"`
}

type OrderSummaryResponse struct {
	OrderID      string                `json:"order_id"`
	OrderNumber  string                `json:"order_number"`
	Type         models.OrderType      `json:"type"`
	Status       models.OrderStatus    `json:"status"`
	CryptoSymbol string                `json:"crypto_symbol"`
	Quantity     decimal.Decimal       `json:"quantity"`
	TotalAmount  decimal.Decimal       `json:"total_amount"`
	CreatedAt    time.Time             `json:"created_at"`
	ExecutedAt   *time.Time            `json:"executed_at,omitempty"`
}

type OrderListResponse struct {
	Orders     []OrderSummaryResponse `json:"orders"`
	Pagination PaginationResponse     `json:"pagination"`
	Summary    *OrdersSummary         `json:"summary,omitempty"`
}

type PaginationResponse struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

type OrdersSummary struct {
	TotalInvested    decimal.Decimal `json:"total_invested"`
	TotalOrders      int64           `json:"total_orders"`
	SuccessfulOrders int64           `json:"successful_orders"`
	FailedOrders     int64           `json:"failed_orders"`
	TotalFees        decimal.Decimal `json:"total_fees"`
	ProfitLoss       decimal.Decimal `json:"profit_loss,omitempty"`
}

type ExecutionResponse struct {
	OrderID          string                   `json:"order_id"`
	Status           models.OrderStatus       `json:"status"`
	ExecutionPrice   decimal.Decimal          `json:"execution_price"`
	ExecutedAt       time.Time                `json:"executed_at"`
	ExecutionDetails *models.ExecutionDetails `json:"execution_details"`
}

type AdminOrderListResponse struct {
	Orders     []AdminOrderResponse `json:"orders"`
	Pagination PaginationResponse   `json:"pagination"`
	Statistics *AdminStatistics     `json:"statistics,omitempty"`
}

type AdminOrderResponse struct {
	OrderResponse
	UserInfo *UserInfo `json:"user_info,omitempty"`
}

type UserInfo struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type AdminStatistics struct {
	TotalOrders         int64           `json:"total_orders"`
	TotalVolume         decimal.Decimal `json:"total_volume"`
	OrdersToday         int64           `json:"orders_today"`
	VolumeToday         decimal.Decimal `json:"volume_today"`
	AverageOrderSize    decimal.Decimal `json:"average_order_size"`
	TotalFeesCollected  decimal.Decimal `json:"total_fees_collected"`
	TopCryptocurrencies []CryptoStats   `json:"top_cryptocurrencies"`
}

type CryptoStats struct {
	Symbol      string          `json:"symbol"`
	TotalOrders int64           `json:"total_orders"`
	TotalVolume decimal.Decimal `json:"total_volume"`
}

type OrderExecutionResult struct {
	Success         bool                     `json:"success"`
	OrderID         string                   `json:"order_id"`
	ExecutionTime   time.Duration            `json:"execution_time"`
	ExecutionSteps  []models.ProcessingStep  `json:"execution_steps"`
	FinalStatus     models.OrderStatus       `json:"final_status"`
	ErrorMessage    string                   `json:"error_message,omitempty"`
}

func ToOrderResponse(order *models.Order) OrderResponse {
	response := OrderResponse{
		OrderID:          order.ID.Hex(),
		OrderNumber:      order.OrderNumber,
		UserID:           order.UserID,
		Type:             order.Type,
		Status:           order.Status,
		CryptoSymbol:     order.CryptoSymbol,
		CryptoName:       order.CryptoName,
		Quantity:         order.Quantity,
		OrderType:        order.OrderKind,
		LimitPrice:       order.LimitPrice,
		OrderPrice:       order.OrderPrice,
		ExecutionPrice:   order.ExecutionPrice,
		TotalAmount:      order.TotalAmount,
		Fee:              order.Fee,
		CreatedAt:        order.CreatedAt,
		ExecutedAt:       order.ExecutedAt,
		UpdatedAt:        order.UpdatedAt,
		ExecutionDetails: order.ExecutionDetails,
	}

	return response
}

func ToOrderSummaryResponse(order *models.Order) OrderSummaryResponse {
	return OrderSummaryResponse{
		OrderID:      order.ID.Hex(),
		OrderNumber:  order.OrderNumber,
		Type:         order.Type,
		Status:       order.Status,
		CryptoSymbol: order.CryptoSymbol,
		Quantity:     order.Quantity,
		TotalAmount:  order.TotalAmount,
		CreatedAt:    order.CreatedAt,
		ExecutedAt:   order.ExecutedAt,
	}
}

func ToOrderListResponse(orders []models.Order, total int64, page, limit int, summary *OrdersSummary) OrderListResponse {
	orderResponses := make([]OrderSummaryResponse, len(orders))
	for i, order := range orders {
		orderResponses[i] = ToOrderSummaryResponse(&order)
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	hasNext := page < totalPages
	hasPrev := page > 1

	return OrderListResponse{
		Orders: orderResponses,
		Pagination: PaginationResponse{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
		},
		Summary: summary,
	}
}

func NewSuccessResponse(message string, data interface{}) APIResponse {
	return APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

func NewErrorResponse(message string) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error:   message,
	}
}

func NewErrorResponseWithCode(message string, code int) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error:   message,
		Code:    code,
	}
}