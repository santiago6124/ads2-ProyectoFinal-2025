package concurrent

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"orders-api/internal/models"
)

type ExecutionService struct {
	userClient        UserClient
	userBalanceClient UserBalanceClient
	marketClient      MarketClient
	feeCalculator     FeeCalculator
	config            *ExecutionConfig
}

type UserClient interface {
	VerifyUser(ctx context.Context, userID int) (*models.ValidationResult, error)
}

type UserBalanceClient interface {
	CheckBalance(ctx context.Context, userID int, amount decimal.Decimal) (*models.BalanceResult, error)
	LockFunds(ctx context.Context, userID int, amount decimal.Decimal) error
	ReleaseFunds(ctx context.Context, userID int, amount decimal.Decimal) error
}

type MarketClient interface {
	GetCurrentPrice(ctx context.Context, symbol string) (*models.PriceResult, error)
	GetMarketConditions(ctx context.Context, symbol string) (*models.MarketConditions, error)
}

type FeeCalculator interface {
	Calculate(ctx context.Context, order *models.Order) (*models.FeeResult, error)
}

type ExecutionConfig struct {
	MaxWorkers       int           `json:"max_workers"`
	QueueSize        int           `json:"queue_size"`
	ExecutionTimeout time.Duration `json:"execution_timeout"`
	MaxSlippage      decimal.Decimal `json:"max_slippage"`
	SimulateLatency  bool          `json:"simulate_latency"`
	MinExecutionTime time.Duration `json:"min_execution_time"`
	MaxExecutionTime time.Duration `json:"max_execution_time"`
}

func NewExecutionService(userClient UserClient, userBalanceClient UserBalanceClient, marketClient MarketClient, feeCalculator FeeCalculator, config *ExecutionConfig) *ExecutionService {
	return &ExecutionService{
		userClient:        userClient,
		userBalanceClient: userBalanceClient,
		marketClient:      marketClient,
		feeCalculator:     feeCalculator,
		config:            config,
	}
}

func (s *ExecutionService) ExecuteOrderConcurrent(ctx context.Context, order *models.Order) (*models.ExecutionResult, error) {
	start := time.Now()
	result := models.NewExecutionResult(order.ID.Hex())

	if s.config.SimulateLatency {
		s.simulateProcessingDelay()
	}

	var wg sync.WaitGroup
	resultChan := make(chan *ConcurrentTaskResult, 4)
	errorChan := make(chan error, 4)

	executionCtx, cancel := context.WithTimeout(ctx, s.config.ExecutionTimeout)
	defer cancel()

	tasks := []ConcurrentTask{
		{
			Name: "user_validation",
			Function: func() (interface{}, error) {
				step := result.AddStep("Validating user")
				defer func() {
					if step.Error != "" {
						step.Fail(fmt.Errorf(step.Error))
					} else {
						step.Complete(result.UserValidation)
					}
				}()

				validation, err := s.userClient.VerifyUser(executionCtx, order.UserID)
				if err != nil {
					step.Error = err.Error()
					return nil, err
				}
				result.UserValidation = validation
				return validation, nil
			},
		},
		{
			Name: "balance_check",
			Function: func() (interface{}, error) {
				step := result.AddStep("Checking balance")
				defer func() {
					if step.Error != "" {
						step.Fail(fmt.Errorf(step.Error))
					} else {
						step.Complete(result.BalanceCheck)
					}
				}()

				estimatedAmount := order.Quantity.Mul(order.OrderPrice)
				balance, err := s.userBalanceClient.CheckBalance(executionCtx, order.UserID, estimatedAmount)
				if err != nil {
					step.Error = err.Error()
					return nil, err
				}

				if !balance.HasSufficient {
					err := fmt.Errorf("insufficient balance: required %s, available %s",
						balance.Required.String(), balance.Available.String())
					step.Error = err.Error()
					return nil, err
				}

				result.BalanceCheck = balance
				return balance, nil
			},
		},
		{
			Name: "market_price",
			Function: func() (interface{}, error) {
				step := result.AddStep("Fetching market price")
				defer func() {
					if step.Error != "" {
						step.Fail(fmt.Errorf(step.Error))
					} else {
						step.Complete(result.MarketPrice)
					}
				}()

				price, err := s.marketClient.GetCurrentPrice(executionCtx, order.CryptoSymbol)
				if err != nil {
					step.Error = err.Error()
					return nil, err
				}

				slippage := s.calculateSlippage(order.Type, order.Quantity)
				price.Slippage = slippage

				if order.Type == models.OrderTypeBuy {
					price.ExecutionPrice = price.MarketPrice.Mul(decimal.NewFromFloat(1).Add(slippage))
				} else {
					price.ExecutionPrice = price.MarketPrice.Mul(decimal.NewFromFloat(1).Sub(slippage))
				}

				price.SlippagePerc = slippage.Mul(decimal.NewFromFloat(100))
				result.MarketPrice = price
				return price, nil
			},
		},
		{
			Name: "fee_calculation",
			Function: func() (interface{}, error) {
				step := result.AddStep("Calculating fees")
				defer func() {
					if step.Error != "" {
						step.Fail(fmt.Errorf(step.Error))
					} else {
						step.Complete(result.FeeCalculation)
					}
				}()

				if s.config.SimulateLatency {
					time.Sleep(time.Duration(rand.Intn(200)+50) * time.Millisecond)
				}

				fee, err := s.feeCalculator.Calculate(executionCtx, order)
				if err != nil {
					step.Error = err.Error()
					return nil, err
				}

				result.FeeCalculation = fee
				return fee, nil
			},
		},
	}

	for i, task := range tasks {
		wg.Add(1)
		go func(index int, t ConcurrentTask) {
			defer wg.Done()

			taskResult := &ConcurrentTaskResult{
				TaskIndex: index,
				TaskName:  t.Name,
				StartTime: time.Now(),
			}

			result, err := t.Function()
			taskResult.EndTime = time.Now()
			taskResult.Duration = taskResult.EndTime.Sub(taskResult.StartTime)
			taskResult.Result = result
			taskResult.Error = err

			if err != nil {
				errorChan <- fmt.Errorf("task %s failed: %w", t.Name, err)
			} else {
				resultChan <- taskResult
			}
		}(i, task)
	}

	wg.Wait()
	close(resultChan)
	close(errorChan)

	result.ExecutionTime = time.Since(start)

	select {
	case err := <-errorChan:
		result.Success = false
		result.Error = err.Error()
		return result, err
	default:
		result.Success = true
	}

	var taskResults []*ConcurrentTaskResult
	for taskResult := range resultChan {
		taskResults = append(taskResults, taskResult)
	}

	if !result.IsValid() {
		result.Success = false
		result.Error = "execution validation failed"
		return result, fmt.Errorf("execution validation failed")
	}

	return result, nil
}

func (s *ExecutionService) calculateSlippage(orderType models.OrderType, quantity decimal.Decimal) decimal.Decimal {
	baseSlippage := decimal.NewFromFloat(0.001) // 0.1% base

	if quantity.GreaterThan(decimal.NewFromFloat(1.0)) {
		multiplier := quantity.Mul(decimal.NewFromFloat(0.1))
		baseSlippage = baseSlippage.Mul(decimal.NewFromFloat(1).Add(multiplier))
	}

	if orderType == models.OrderTypeSell {
		baseSlippage = baseSlippage.Mul(decimal.NewFromFloat(1.2))
	}

	randomFactor := decimal.NewFromFloat((rand.Float64() - 0.5) * 0.001)
	finalSlippage := baseSlippage.Add(randomFactor)

	if finalSlippage.LessThan(decimal.Zero) {
		finalSlippage = decimal.NewFromFloat(0.0001)
	}

	maxSlippage := s.config.MaxSlippage
	if finalSlippage.GreaterThan(maxSlippage) {
		finalSlippage = maxSlippage
	}

	return finalSlippage
}

func (s *ExecutionService) simulateProcessingDelay() {
	minDelay := s.config.MinExecutionTime
	maxDelay := s.config.MaxExecutionTime

	if minDelay == 0 {
		minDelay = 100 * time.Millisecond
	}
	if maxDelay == 0 {
		maxDelay = 2 * time.Second
	}

	delayRange := maxDelay - minDelay
	randomDelay := time.Duration(rand.Int63n(int64(delayRange)))
	finalDelay := minDelay + randomDelay

	time.Sleep(finalDelay)
}

type ConcurrentTask struct {
	Name     string
	Function func() (interface{}, error)
}

type ConcurrentTaskResult struct {
	TaskIndex int
	TaskName  string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Result    interface{}
	Error     error
}

func (r *ConcurrentTaskResult) IsSuccessful() bool {
	return r.Error == nil
}

func (r *ConcurrentTaskResult) GetErrorMessage() string {
	if r.Error != nil {
		return r.Error.Error()
	}
	return ""
}

func DefaultExecutionConfig() *ExecutionConfig {
	return &ExecutionConfig{
		MaxWorkers:       10,
		QueueSize:        100,
		ExecutionTimeout: 30 * time.Second,
		MaxSlippage:      decimal.NewFromFloat(0.05), // 5%
		SimulateLatency:  true,
		MinExecutionTime: 100 * time.Millisecond,
		MaxExecutionTime: 2 * time.Second,
	}
}