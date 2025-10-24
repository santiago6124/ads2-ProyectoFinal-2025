package models

import (
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderType string
type OrderStatus string

const (
	OrderTypeBuy  OrderType = "buy"
	OrderTypeSell OrderType = "sell"
)

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusExecuted   OrderStatus = "executed"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusFailed     OrderStatus = "failed"
)

type OrderKind string

const (
	OrderKindMarket OrderKind = "market"
	OrderKindLimit  OrderKind = "limit"
)

type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC" // Good Till Cancelled
	TimeInForceIOC TimeInForce = "IOC" // Immediate or Cancel
	TimeInForceFOK TimeInForce = "FOK" // Fill or Kill
)

type OrderFilter struct {
	Status       []OrderStatus `json:"status,omitempty"`
	CryptoSymbol string        `json:"crypto_symbol,omitempty"`
	OrderType    []OrderType   `json:"order_type,omitempty"`
	StartDate    *time.Time    `json:"start_date,omitempty"`
	EndDate      *time.Time    `json:"end_date,omitempty"`
	Limit        int           `json:"limit,omitempty"`
	Offset       int           `json:"offset,omitempty"`
}

type CryptoInfo struct {
	Symbol   string `json:"symbol"`
	IsActive bool   `json:"is_active"`
	Name     string `json:"name"`
}


type Order struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderNumber      string            `bson:"order_number" json:"order_number"`
	UserID           int               `bson:"user_id" json:"user_id"`
	Type             OrderType         `bson:"type" json:"type"`
	Status           OrderStatus       `bson:"status" json:"status"`
	CryptoSymbol     string            `bson:"crypto_symbol" json:"crypto_symbol"`
	CryptoName       string            `bson:"crypto_name" json:"crypto_name"`
	Quantity         decimal.Decimal   `bson:"quantity" json:"quantity"`
	OrderKind        OrderKind         `bson:"order_type" json:"order_type"`
	LimitPrice       *decimal.Decimal  `bson:"limit_price,omitempty" json:"limit_price,omitempty"`
	OrderPrice       decimal.Decimal   `bson:"order_price" json:"order_price"`
	ExecutionPrice   *decimal.Decimal  `bson:"execution_price,omitempty" json:"execution_price,omitempty"`
	TotalAmount      decimal.Decimal   `bson:"total_amount" json:"total_amount"`
	Fee              decimal.Decimal   `bson:"fee" json:"fee"`
	FeePercentage    decimal.Decimal   `bson:"fee_percentage" json:"fee_percentage"`
	CreatedAt        time.Time         `bson:"created_at" json:"created_at"`
	ExecutedAt       *time.Time        `bson:"executed_at,omitempty" json:"executed_at,omitempty"`
	UpdatedAt        time.Time         `bson:"updated_at" json:"updated_at"`
	CancelledAt      *time.Time        `bson:"cancelled_at,omitempty" json:"cancelled_at,omitempty"`
	ExecutionDetails *ExecutionDetails `bson:"execution_details,omitempty" json:"execution_details,omitempty"`
	Metadata         map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Validation       *OrderValidation  `bson:"validation,omitempty" json:"validation,omitempty"`
	Audit            *OrderAudit       `bson:"audit,omitempty" json:"audit,omitempty"`
}

type ExecutionDetails struct {
	MarketPriceAtExecution decimal.Decimal `bson:"market_price_at_execution" json:"market_price_at_execution"`
	Slippage              decimal.Decimal `bson:"slippage" json:"slippage"`
	SlippagePercentage    decimal.Decimal `bson:"slippage_percentage" json:"slippage_percentage"`
	ExecutionTimeMs       int64          `bson:"execution_time_ms" json:"execution_time_ms"`
	ExecutionID           string         `bson:"execution_id" json:"execution_id"`
}

type OrderValidation struct {
	IsValid           bool     `json:"is_valid"`
	ErrorMessage      string   `json:"error_message,omitempty"`
	ValidatedAt       time.Time `json:"validated_at"`
	ValidationErrors  []string `json:"validation_errors,omitempty"`
}

type OrderModification struct {
	Field      string      `json:"field"`
	OldValue   interface{} `json:"old_value"`
	NewValue   interface{} `json:"new_value"`
	ModifiedAt time.Time   `json:"modified_at"`
	ModifiedBy string      `json:"modified_by"`
	Reason     string      `json:"reason"`
}

type OrderAudit struct {
	CreatedBy    string              `json:"created_by"`
	CreatedAt    time.Time           `json:"created_at"`
	ModifiedBy   string              `json:"modified_by"`
	ModifiedAt   time.Time           `json:"modified_at"`
	Modifications []OrderModification `json:"modifications,omitempty"`
}

func (o *Order) IsEditable() bool {
	return o.Status == OrderStatusPending
}

func (o *Order) IsCancellable() bool {
	return o.Status == OrderStatusPending || o.Status == OrderStatusProcessing
}

func (o *Order) IsExecuted() bool {
	return o.Status == OrderStatusExecuted
}

func (o *Order) IsFinal() bool {
	return o.Status == OrderStatusExecuted ||
		   o.Status == OrderStatusCancelled ||
		   o.Status == OrderStatusFailed
}

func (o *Order) CalculateTotalWithFee() decimal.Decimal {
	return o.TotalAmount.Add(o.Fee)
}

func (o *Order) GetEffectivePrice() decimal.Decimal {
	if o.ExecutionPrice != nil {
		return *o.ExecutionPrice
	}
	return o.OrderPrice
}

func NewOrderNumber() string {
	return "ORD-" + time.Now().Format("2006") + "-" + primitive.NewObjectID().Hex()[:6]
}

func NewExecutionID() string {
	return "EXEC-" + time.Now().Format("2006") + "-" + primitive.NewObjectID().Hex()[:6]
}