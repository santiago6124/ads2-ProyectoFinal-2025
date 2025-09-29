package dto

import (
	"github.com/shopspring/decimal"
	"orders-api/internal/models"
)

type CreateOrderRequest struct {
	Type         models.OrderType `json:"type" binding:"required,oneof=buy sell"`
	CryptoSymbol string           `json:"crypto_symbol" binding:"required,min=2,max=10"`
	Quantity     decimal.Decimal  `json:"quantity" binding:"required"`
	OrderType    models.OrderKind `json:"order_type" binding:"required,oneof=market limit"`
	LimitPrice   *decimal.Decimal `json:"limit_price,omitempty"`
}

type UpdateOrderRequest struct {
	Quantity   *decimal.Decimal `json:"quantity,omitempty"`
	LimitPrice *decimal.Decimal `json:"limit_price,omitempty"`
}

type OrderFilterRequest struct {
	Status     *models.OrderStatus `json:"status,omitempty"`
	CryptoSymbol *string           `json:"crypto,omitempty"`
	Type       *models.OrderType   `json:"type,omitempty"`
	From       *string             `json:"from,omitempty"`     // YYYY-MM-DD
	To         *string             `json:"to,omitempty"`       // YYYY-MM-DD
	Page       int                 `json:"page,omitempty"`
	Limit      int                 `json:"limit,omitempty"`
	Sort       *string             `json:"sort,omitempty"`     // created_at, -created_at
}

type AdminOrderFilterRequest struct {
	OrderFilterRequest
	UserID *int `json:"user_id,omitempty"`
}

func (r *CreateOrderRequest) Validate() error {
	if r.Type == "" {
		return ErrInvalidOrderType
	}

	if r.CryptoSymbol == "" {
		return ErrInvalidCryptoSymbol
	}

	if r.Quantity.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidQuantity
	}

	if r.OrderType == models.OrderKindLimit && (r.LimitPrice == nil || r.LimitPrice.LessThanOrEqual(decimal.Zero)) {
		return ErrLimitPriceRequired
	}

	// Maximum quantity validation (prevent abuse)
	maxQuantity := decimal.NewFromFloat(1000000)
	if r.Quantity.GreaterThan(maxQuantity) {
		return ErrQuantityTooLarge
	}

	return nil
}

func (r *UpdateOrderRequest) Validate() error {
	if r.Quantity != nil && r.Quantity.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidQuantity
	}

	if r.LimitPrice != nil && r.LimitPrice.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidLimitPrice
	}

	// At least one field must be provided
	if r.Quantity == nil && r.LimitPrice == nil {
		return ErrNoFieldsToUpdate
	}

	return nil
}

func (r *OrderFilterRequest) SetDefaults() {
	if r.Page <= 0 {
		r.Page = 1
	}

	if r.Limit <= 0 || r.Limit > 100 {
		r.Limit = 20
	}

	if r.Sort == nil {
		defaultSort := "-created_at"
		r.Sort = &defaultSort
	}
}

func (r *OrderFilterRequest) GetOffset() int {
	return (r.Page - 1) * r.Limit
}

func (r *OrderFilterRequest) IsValidSort() bool {
	if r.Sort == nil {
		return true
	}

	validSorts := map[string]bool{
		"created_at":     true,
		"-created_at":    true,
		"executed_at":    true,
		"-executed_at":   true,
		"total_amount":   true,
		"-total_amount":  true,
		"crypto_symbol":  true,
		"-crypto_symbol": true,
	}

	return validSorts[*r.Sort]
}

type ExecuteOrderRequest struct {
	ForceExecution bool `json:"force_execution,omitempty"`
	OverridePrice  *decimal.Decimal `json:"override_price,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

type BulkCancelRequest struct {
	OrderIDs []string `json:"order_ids" binding:"required,min=1,max=50"`
	Reason   string   `json:"reason,omitempty"`
}

type ReprocessOrderRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Reason  string `json:"reason" binding:"required"`
}

var (
	ErrInvalidOrderType    = NewValidationError("invalid order type")
	ErrInvalidCryptoSymbol = NewValidationError("invalid crypto symbol")
	ErrInvalidQuantity     = NewValidationError("quantity must be greater than zero")
	ErrInvalidLimitPrice   = NewValidationError("limit price must be greater than zero")
	ErrLimitPriceRequired  = NewValidationError("limit price is required for limit orders")
	ErrQuantityTooLarge    = NewValidationError("quantity is too large")
	ErrNoFieldsToUpdate    = NewValidationError("at least one field must be provided for update")
)

type ValidationError struct {
	Message string `json:"message"`
}

func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}

func (e *ValidationError) Error() string {
	return e.Message
}