package dto

import (
	"fmt"
	"orders-api/internal/models"

	"github.com/shopspring/decimal"
)

// CreateOrderRequest request simplificado para crear una orden
type CreateOrderRequest struct {
	Type         models.OrderType `json:"type" binding:"required,oneof=buy sell"`
	CryptoSymbol string           `json:"crypto_symbol" binding:"required,min=2,max=10"`
	Quantity     string           `json:"quantity" binding:"required"` // String para evitar problemas de parseo JSON
	OrderKind    models.OrderKind `json:"order_kind" binding:"required,oneof=market limit"`
	LimitPrice   string           `json:"limit_price,omitempty"`  // Solo requerido para limit orders
	MarketPrice  string           `json:"market_price,omitempty"` // Precio de mercado desde el frontend
}

// Validate valida la request y retorna los valores parseados
func (r *CreateOrderRequest) Validate() (quantity decimal.Decimal, limitPrice *decimal.Decimal, marketPrice *decimal.Decimal, err error) {
	// Parsear y validar quantity
	quantity, err = decimal.NewFromString(r.Quantity)
	if err != nil {
		return decimal.Zero, nil, nil, fmt.Errorf("invalid quantity format: must be a valid number")
	}

	if quantity.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, nil, nil, fmt.Errorf("quantity must be greater than zero")
	}

	// Validación de cantidad máxima
	maxQuantity := decimal.NewFromInt(1000000)
	if quantity.GreaterThan(maxQuantity) {
		return decimal.Zero, nil, nil, fmt.Errorf("quantity exceeds maximum allowed (1,000,000)")
	}

	// Validar limit price para órdenes limit
	if r.OrderKind == models.OrderKindLimit {
		if r.LimitPrice == "" {
			return decimal.Zero, nil, nil, fmt.Errorf("limit_price is required for limit orders")
		}

		price, err := decimal.NewFromString(r.LimitPrice)
		if err != nil {
			return decimal.Zero, nil, nil, fmt.Errorf("invalid limit_price format: must be a valid number")
		}

		if price.LessThanOrEqual(decimal.Zero) {
			return decimal.Zero, nil, nil, fmt.Errorf("limit_price must be greater than zero")
		}

		limitPrice = &price
	}

	// Validar market price si viene desde el frontend
	if r.MarketPrice != "" {
		price, err := decimal.NewFromString(r.MarketPrice)
		if err != nil {
			return decimal.Zero, nil, nil, fmt.Errorf("invalid market_price format: must be a valid number")
		}

		if price.LessThanOrEqual(decimal.Zero) {
			return decimal.Zero, nil, nil, fmt.Errorf("market_price must be greater than zero")
		}

		marketPrice = &price
	}

	return quantity, limitPrice, marketPrice, nil
}

// OrderFilterRequest para filtrar y paginar órdenes
type OrderFilterRequest struct {
	Status       *models.OrderStatus `json:"status,omitempty"`
	CryptoSymbol *string             `json:"crypto_symbol,omitempty"`
	Type         *models.OrderType   `json:"type,omitempty"`
	Page         int                 `json:"page,omitempty"`
	Limit        int                 `json:"limit,omitempty"`
}

// SetDefaults establece valores por defecto para paginación
func (r *OrderFilterRequest) SetDefaults() {
	if r.Page <= 0 {
		r.Page = 1
	}

	if r.Limit <= 0 || r.Limit > 100 {
		r.Limit = 20
	}
}

// GetOffset calcula el offset para la query de base de datos
func (r *OrderFilterRequest) GetOffset() int {
	return (r.Page - 1) * r.Limit
}

// OrdersSummary resumen de las órdenes del usuario
type OrdersSummary struct {
	TotalOrders     int64           `json:"total_orders"`
	ExecutedOrders  int64           `json:"executed_orders"`
	PendingOrders   int64           `json:"pending_orders"`
	CancelledOrders int64           `json:"cancelled_orders"`
	FailedOrders    int64           `json:"failed_orders"`
	TotalVolume     decimal.Decimal `json:"total_volume"`      // Volumen total en USD
}
