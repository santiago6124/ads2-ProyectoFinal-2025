package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type ExecutionResult struct {
	ExecutionID      string              `json:"execution_id"`
	OrderID          string              `json:"order_id"`
	UserValidation   *ValidationResult   `json:"user_validation"`
	BalanceCheck     *BalanceResult      `json:"balance_check"`
	MarketPrice      *PriceResult        `json:"market_price"`
	FeeCalculation   *FeeResult          `json:"fee_calculation"`
	ExecutionTime    time.Duration       `json:"execution_time"`
	Success          bool                `json:"success"`
	Error            string              `json:"error,omitempty"`
	ProcessingSteps  []ProcessingStep    `json:"processing_steps"`
}

type ValidationResult struct {
	UserID       int    `json:"user_id"`
	IsActive     bool   `json:"is_active"`
	Role         string `json:"role"`
	Validated    bool   `json:"validated"`
	IsValid      bool   `json:"is_valid"`
	Message      string `json:"message"`
	UserData     interface{} `json:"user_data"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type BalanceResult struct {
	Available     decimal.Decimal `json:"available"`
	Locked        decimal.Decimal `json:"locked"`
	Required      decimal.Decimal `json:"required"`
	Total         decimal.Decimal `json:"total"`
	Currency      string          `json:"currency"`
	Message       string          `json:"message"`
	HasSufficient bool            `json:"has_sufficient"`
	ErrorMessage  string          `json:"error_message,omitempty"`
}

type PriceResult struct {
	Symbol         string          `json:"symbol"`
	MarketPrice    decimal.Decimal `json:"market_price"`
	BidPrice       decimal.Decimal `json:"bid_price"`
	AskPrice       decimal.Decimal `json:"ask_price"`
	ExecutionPrice decimal.Decimal `json:"execution_price"`
	Volume24h      string          `json:"volume_24h"`
	Change24h      string          `json:"change_24h"`
	ChangePercent  string          `json:"change_percent"`
	High24h        decimal.Decimal `json:"high_24h"`
	Low24h         decimal.Decimal `json:"low_24h"`
	Source         string          `json:"source"`
	Confidence     string          `json:"confidence"`
	LastUpdated    string          `json:"last_updated"`
	Slippage       decimal.Decimal `json:"slippage"`
	SlippagePerc   decimal.Decimal `json:"slippage_percentage"`
	Timestamp      time.Time       `json:"timestamp"`
}

type FeeResult struct {
	BaseFee        decimal.Decimal `json:"base_fee"`
	PercentageFee  decimal.Decimal `json:"percentage_fee"`
	TotalFee       decimal.Decimal `json:"total_fee"`
	FeePercentage  decimal.Decimal `json:"fee_percentage"`
	FeeType        string          `json:"fee_type"` // "maker", "taker"
}

type ProcessingStep struct {
	Step        string        `json:"step"`
	Status      string        `json:"status"` // "started", "completed", "failed"
	Duration    time.Duration `json:"duration"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     *time.Time    `json:"end_time,omitempty"`
	Error       string        `json:"error,omitempty"`
	Data        interface{}   `json:"data,omitempty"`
}

type MarketConditions struct {
	Symbol              string          `json:"symbol"`
	CurrentPrice        decimal.Decimal `json:"current_price"`
	Volume24h           decimal.Decimal `json:"volume_24h"`
	PriceChange24h      decimal.Decimal `json:"price_change_24h"`
	MarketCap           decimal.Decimal `json:"market_cap"`
	Liquidity           string          `json:"liquidity"` // "high", "medium", "low"
	Volatility          string          `json:"volatility"` // "high", "medium", "low"
	Spread              decimal.Decimal `json:"spread"`
	SpreadPercent       decimal.Decimal `json:"spread_percent"`
	TradingVolume       decimal.Decimal `json:"trading_volume"`
	OrderBookDepth      decimal.Decimal `json:"order_book_depth"`
	CirculatingSupply   decimal.Decimal `json:"circulating_supply"`
	LastUpdated         time.Time       `json:"last_updated"`
}

type ExecutionContext struct {
	RequestID       string               `json:"request_id"`
	UserID          int                  `json:"user_id"`
	Order           *Order               `json:"order"`
	MarketData      *MarketConditions    `json:"market_data"`
	Configuration   *ExecutionConfig     `json:"configuration"`
	Timeout         time.Duration        `json:"timeout"`
	MaxRetries      int                  `json:"max_retries"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type ExecutionConfig struct {
	MaxSlippage      decimal.Decimal `json:"max_slippage"`
	TimeoutSeconds   int             `json:"timeout_seconds"`
	RetryAttempts    int             `json:"retry_attempts"`
	SimulateLatency  bool            `json:"simulate_latency"`
	MinExecutionTime int             `json:"min_execution_time_ms"`
	MaxExecutionTime int             `json:"max_execution_time_ms"`
}

type ConcurrentTask struct {
	Name     string      `json:"name"`
	Function func() (interface{}, error) `json:"-"`
	Result   interface{} `json:"result"`
	Error    error       `json:"error"`
	Duration time.Duration `json:"duration"`
}

func NewExecutionResult(orderID string) *ExecutionResult {
	return &ExecutionResult{
		ExecutionID:     NewExecutionID(),
		OrderID:         orderID,
		ProcessingSteps: make([]ProcessingStep, 0),
		Success:         false,
	}
}

func (er *ExecutionResult) AddStep(step string) *ProcessingStep {
	newStep := ProcessingStep{
		Step:      step,
		Status:    "started",
		StartTime: time.Now(),
	}
	er.ProcessingSteps = append(er.ProcessingSteps, newStep)
	return &er.ProcessingSteps[len(er.ProcessingSteps)-1]
}

func (ps *ProcessingStep) Complete(data interface{}) {
	now := time.Now()
	ps.Status = "completed"
	ps.EndTime = &now
	ps.Duration = now.Sub(ps.StartTime)
	ps.Data = data
}

func (ps *ProcessingStep) Fail(err error) {
	now := time.Now()
	ps.Status = "failed"
	ps.EndTime = &now
	ps.Duration = now.Sub(ps.StartTime)
	ps.Error = err.Error()
}

func (er *ExecutionResult) IsValid() bool {
	return er.UserValidation != nil && er.UserValidation.Validated &&
		   er.BalanceCheck != nil && er.BalanceCheck.HasSufficient &&
		   er.MarketPrice != nil &&
		   er.FeeCalculation != nil
}

func (er *ExecutionResult) GetTotalExecutionTime() time.Duration {
	return er.ExecutionTime
}

func (er *ExecutionResult) HasErrors() bool {
	return er.Error != "" ||
		   (er.UserValidation != nil && er.UserValidation.ErrorMessage != "") ||
		   (er.BalanceCheck != nil && er.BalanceCheck.ErrorMessage != "")
}