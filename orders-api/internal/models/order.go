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
	Validation       *OrderValidation  `bson:"validation,omitempty" json:"validation,omitempty"`
	Metadata         *OrderMetadata    `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Audit            *OrderAudit       `bson:"audit,omitempty" json:"audit,omitempty"`
}

type ExecutionDetails struct {
	MarketPriceAtExecution decimal.Decimal `bson:"market_price_at_execution" json:"market_price_at_execution"`
	Slippage              decimal.Decimal `bson:"slippage" json:"slippage"`
	SlippagePercentage    decimal.Decimal `bson:"slippage_percentage" json:"slippage_percentage"`
	ExecutionTimeMs       int64          `bson:"execution_time_ms" json:"execution_time_ms"`
	Retries               int            `bson:"retries" json:"retries"`
	ExecutionID           string         `bson:"execution_id" json:"execution_id"`
}

type OrderValidation struct {
	UserVerified     bool   `bson:"user_verified" json:"user_verified"`
	BalanceChecked   bool   `bson:"balance_checked" json:"balance_checked"`
	MarketHours      bool   `bson:"market_hours" json:"market_hours"`
	RiskAssessment   string `bson:"risk_assessment" json:"risk_assessment"`
	ValidationErrors []string `bson:"validation_errors,omitempty" json:"validation_errors,omitempty"`
}

type OrderMetadata struct {
	IPAddress   string `bson:"ip_address" json:"ip_address"`
	UserAgent   string `bson:"user_agent" json:"user_agent"`
	Platform    string `bson:"platform" json:"platform"`
	APIVersion  string `bson:"api_version" json:"api_version"`
	SessionID   string `bson:"session_id" json:"session_id"`
}

type OrderAudit struct {
	CreatedBy     int                  `bson:"created_by" json:"created_by"`
	ModifiedBy    *int                 `bson:"modified_by,omitempty" json:"modified_by,omitempty"`
	Modifications []OrderModification  `bson:"modifications,omitempty" json:"modifications,omitempty"`
}

type OrderModification struct {
	Field        string      `bson:"field" json:"field"`
	OldValue     interface{} `bson:"old_value" json:"old_value"`
	NewValue     interface{} `bson:"new_value" json:"new_value"`
	ModifiedBy   int         `bson:"modified_by" json:"modified_by"`
	ModifiedAt   time.Time   `bson:"modified_at" json:"modified_at"`
	Reason       string      `bson:"reason,omitempty" json:"reason,omitempty"`
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