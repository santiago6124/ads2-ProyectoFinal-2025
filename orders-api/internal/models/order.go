package models

import (
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Tipos básicos
type OrderType string
type OrderStatus string
type OrderKind string

// OrderType define si es compra o venta
const (
	OrderTypeBuy  OrderType = "buy"
	OrderTypeSell OrderType = "sell"
)

// OrderStatus define el estado de la orden
const (
	OrderStatusPending   OrderStatus = "pending"   // Orden creada, esperando ejecución
	OrderStatusExecuted  OrderStatus = "executed"  // Orden ejecutada exitosamente
	OrderStatusCancelled OrderStatus = "cancelled" // Orden cancelada por el usuario
	OrderStatusFailed    OrderStatus = "failed"    // Orden falló durante ejecución
)

// OrderKind define el tipo de orden
const (
	OrderKindMarket OrderKind = "market" // Se ejecuta al precio actual de mercado
	OrderKindLimit  OrderKind = "limit"  // Se ejecuta solo si se alcanza el precio límite
)

// Order representa una orden de compra/venta simplificada
type Order struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"` // ID único generado por MongoDB
	UserID       int                `bson:"user_id" json:"user_id"`
	Type         OrderType          `bson:"type" json:"type"`                 // buy o sell
	Status       OrderStatus        `bson:"status" json:"status"`             // pending, executed, cancelled, failed
	CryptoSymbol string             `bson:"crypto_symbol" json:"crypto_symbol"` // BTC, ETH, etc
	CryptoName   string             `bson:"crypto_name" json:"crypto_name"`     // Bitcoin, Ethereum, etc
	Quantity     decimal.Decimal    `bson:"quantity" json:"quantity"`           // Cantidad a comprar/vender
	OrderKind    OrderKind          `bson:"order_kind" json:"order_kind"`       // market o limit
	Price        decimal.Decimal    `bson:"price" json:"price"`                 // Precio de ejecución
	TotalAmount  decimal.Decimal    `bson:"total_amount" json:"total_amount"`   // Quantity * Price
	Fee          decimal.Decimal    `bson:"fee" json:"fee"`                     // Comisión (0.1%)
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	ExecutedAt   *time.Time         `bson:"executed_at,omitempty" json:"executed_at,omitempty"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
	ErrorMessage string             `bson:"error_message,omitempty" json:"error_message,omitempty"` // Si falla
}

// IsCancellable verifica si la orden puede ser cancelada
func (o *Order) IsCancellable() bool {
	return o.Status == OrderStatusPending
}

// IsExecuted verifica si la orden fue ejecutada
func (o *Order) IsExecuted() bool {
	return o.Status == OrderStatusExecuted
}

// IsFinal verifica si la orden está en un estado final
func (o *Order) IsFinal() bool {
	return o.Status == OrderStatusExecuted ||
		o.Status == OrderStatusCancelled ||
		o.Status == OrderStatusFailed
}

// CalculateTotalWithFee calcula el total incluyendo la comisión
func (o *Order) CalculateTotalWithFee() decimal.Decimal {
	return o.TotalAmount.Add(o.Fee)
}

// CryptoInfo información básica de una criptomoneda
type CryptoInfo struct {
	Symbol   string `json:"symbol"`
	IsActive bool   `json:"is_active"`
	Name     string `json:"name"`
}
