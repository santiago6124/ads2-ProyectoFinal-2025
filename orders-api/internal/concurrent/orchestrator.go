package concurrent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"orders-api/internal/models"
)

type OrderOrchestrator struct {
	workers       int
	orderQueue    chan *OrderTask
	resultQueue   chan *OrderResult
	errorQueue    chan *OrderError
	executor      *ExecutionService
	running       bool
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
	metrics       *OrchestratorMetrics
}

type OrderTask struct {
	Order     *models.Order
	Context   context.Context
	Priority  int
	CreatedAt time.Time
	Callback  func(*OrderResult, error)
}

type OrderResult struct {
	OrderID         string
	ExecutionResult *models.ExecutionResult
	ProcessingTime  time.Duration
	WorkerID        int
}

type OrderError struct {
	OrderID   string
	Error     error
	WorkerID  int
	Timestamp time.Time
}

type OrchestratorMetrics struct {
	TotalProcessed   int64
	TotalErrors      int64
	AverageTime      time.Duration
	ActiveWorkers    int
	QueueSize        int
	ProcessingRate   float64
	LastProcessTime  time.Time
	mu               sync.RWMutex
}

func NewOrderOrchestrator(workers int, queueSize int, executor *ExecutionService) *OrderOrchestrator {
	return &OrderOrchestrator{
		workers:     workers,
		orderQueue:  make(chan *OrderTask, queueSize),
		resultQueue: make(chan *OrderResult, queueSize),
		errorQueue:  make(chan *OrderError, queueSize),
		executor:    executor,
		stopChan:    make(chan struct{}),
		metrics:     &OrchestratorMetrics{},
	}
}

func (o *OrderOrchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	if o.running {
		o.mu.Unlock()
		return fmt.Errorf("orchestrator is already running")
	}
	o.running = true
	o.mu.Unlock()

	log.Printf("Starting order orchestrator with %d workers", o.workers)

	for i := 0; i < o.workers; i++ {
		o.wg.Add(1)
		go o.worker(ctx, i)
	}

	o.wg.Add(1)
	go o.metricsCollector(ctx)

	o.wg.Add(1)
	go o.resultHandler(ctx)

	return nil
}

func (o *OrderOrchestrator) Stop() error {
	o.mu.Lock()
	if !o.running {
		o.mu.Unlock()
		return fmt.Errorf("orchestrator is not running")
	}
	o.running = false
	o.mu.Unlock()

	log.Println("Stopping order orchestrator...")

	close(o.stopChan)
	close(o.orderQueue)

	o.wg.Wait()

	close(o.resultQueue)
	close(o.errorQueue)

	log.Println("Order orchestrator stopped")
	return nil
}

func (o *OrderOrchestrator) SubmitOrder(order *models.Order, ctx context.Context, callback func(*OrderResult, error)) error {
	o.mu.RLock()
	if !o.running {
		o.mu.RUnlock()
		return fmt.Errorf("orchestrator is not running")
	}
	o.mu.RUnlock()

	task := &OrderTask{
		Order:     order,
		Context:   ctx,
		Priority:  o.calculatePriority(order),
		CreatedAt: time.Now(),
		Callback:  callback,
	}

	select {
	case o.orderQueue <- task:
		o.updateQueueMetrics()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("order queue is full")
	}
}

func (o *OrderOrchestrator) worker(ctx context.Context, workerID int) {
	defer o.wg.Done()

	log.Printf("Worker %d started", workerID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping due to context cancellation", workerID)
			return
		case <-o.stopChan:
			log.Printf("Worker %d stopping due to stop signal", workerID)
			return
		case task := <-o.orderQueue:
			if task == nil {
				log.Printf("Worker %d stopping due to closed channel", workerID)
				return
			}

			o.processOrderTask(task, workerID)
		}
	}
}

func (o *OrderOrchestrator) processOrderTask(task *OrderTask, workerID int) {
	start := time.Now()

	executionResult, err := o.executor.ExecuteOrderConcurrent(task.Context, task.Order)
	processingTime := time.Since(start)

	if err != nil {
		orderError := &OrderError{
			OrderID:   task.Order.ID.Hex(),
			Error:     err,
			WorkerID:  workerID,
			Timestamp: time.Now(),
		}

		select {
		case o.errorQueue <- orderError:
		default:
			log.Printf("Error queue full, dropping error for order %s", task.Order.ID.Hex())
		}

		if task.Callback != nil {
			task.Callback(nil, err)
		}

		o.updateErrorMetrics()
		return
	}

	result := &OrderResult{
		OrderID:         task.Order.ID.Hex(),
		ExecutionResult: executionResult,
		ProcessingTime:  processingTime,
		WorkerID:        workerID,
	}

	select {
	case o.resultQueue <- result:
	default:
		log.Printf("Result queue full, dropping result for order %s", task.Order.ID.Hex())
	}

	if task.Callback != nil {
		task.Callback(result, nil)
	}

	o.updateSuccessMetrics(processingTime)
}

func (o *OrderOrchestrator) resultHandler(ctx context.Context) {
	defer o.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopChan:
			return
		case result := <-o.resultQueue:
			if result == nil {
				return
			}
			o.handleOrderResult(result)
		case err := <-o.errorQueue:
			if err == nil {
				return
			}
			o.handleOrderError(err)
		}
	}
}

func (o *OrderOrchestrator) handleOrderResult(result *OrderResult) {
	log.Printf("Order %s processed successfully by worker %d in %v",
		result.OrderID, result.WorkerID, result.ProcessingTime)
}

func (o *OrderOrchestrator) handleOrderError(orderError *OrderError) {
	log.Printf("Order %s failed on worker %d: %v",
		orderError.OrderID, orderError.WorkerID, orderError.Error)
}

func (o *OrderOrchestrator) metricsCollector(ctx context.Context) {
	defer o.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopChan:
			return
		case <-ticker.C:
			o.collectMetrics()
		}
	}
}

func (o *OrderOrchestrator) collectMetrics() {
	o.metrics.mu.Lock()
	defer o.metrics.mu.Unlock()

	o.metrics.QueueSize = len(o.orderQueue)
	o.metrics.ActiveWorkers = o.workers

	if o.metrics.TotalProcessed > 0 {
		o.metrics.ProcessingRate = float64(o.metrics.TotalProcessed) / time.Since(o.metrics.LastProcessTime).Seconds()
	}
}

func (o *OrderOrchestrator) updateSuccessMetrics(processingTime time.Duration) {
	o.metrics.mu.Lock()
	defer o.metrics.mu.Unlock()

	o.metrics.TotalProcessed++
	o.metrics.LastProcessTime = time.Now()

	if o.metrics.AverageTime == 0 {
		o.metrics.AverageTime = processingTime
	} else {
		o.metrics.AverageTime = (o.metrics.AverageTime + processingTime) / 2
	}
}

func (o *OrderOrchestrator) updateErrorMetrics() {
	o.metrics.mu.Lock()
	defer o.metrics.mu.Unlock()

	o.metrics.TotalErrors++
}

func (o *OrderOrchestrator) updateQueueMetrics() {
	o.metrics.mu.Lock()
	defer o.metrics.mu.Unlock()

	o.metrics.QueueSize = len(o.orderQueue)
}

func (o *OrderOrchestrator) calculatePriority(order *models.Order) int {
	priority := 100

	if order.Type == models.OrderTypeSell {
		priority += 10
	}

	if order.OrderKind == models.OrderKindMarket {
		priority += 20
	}

	age := time.Since(order.CreatedAt)
	if age > 5*time.Minute {
		priority += int(age.Minutes())
	}

	return priority
}

func (o *OrderOrchestrator) GetMetrics() *OrchestratorMetrics {
	o.metrics.mu.RLock()
	defer o.metrics.mu.RUnlock()

	return &OrchestratorMetrics{
		TotalProcessed:  o.metrics.TotalProcessed,
		TotalErrors:     o.metrics.TotalErrors,
		AverageTime:     o.metrics.AverageTime,
		ActiveWorkers:   o.metrics.ActiveWorkers,
		QueueSize:       o.metrics.QueueSize,
		ProcessingRate:  o.metrics.ProcessingRate,
		LastProcessTime: o.metrics.LastProcessTime,
	}
}

func (o *OrderOrchestrator) IsRunning() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.running
}

func (o *OrderOrchestrator) GetQueueSize() int {
	return len(o.orderQueue)
}

func (o *OrderOrchestrator) GetWorkerCount() int {
	return o.workers
}