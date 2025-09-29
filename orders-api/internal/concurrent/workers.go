package concurrent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"orders-api/internal/models"
)

type WorkerPool struct {
	workers      []*Worker
	taskQueue    chan *WorkerTask
	resultQueue  chan *WorkerResult
	errorQueue   chan *WorkerError
	workerCount  int
	queueSize    int
	running      bool
	stopChan     chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	executor     *ExecutionService
	metrics      *WorkerPoolMetrics
}

type Worker struct {
	ID           int
	pool         *WorkerPool
	taskCount    int64
	lastTaskTime time.Time
	status       WorkerStatus
	mu           sync.RWMutex
}

type WorkerStatus string

const (
	WorkerStatusIdle       WorkerStatus = "idle"
	WorkerStatusProcessing WorkerStatus = "processing"
	WorkerStatusStopped    WorkerStatus = "stopped"
	WorkerStatusError      WorkerStatus = "error"
)

type WorkerTask struct {
	ID        string
	Order     *models.Order
	Context   context.Context
	Priority  int
	CreatedAt time.Time
	StartedAt *time.Time
	Retries   int
	MaxRetries int
}

type WorkerResult struct {
	TaskID          string
	OrderID         string
	WorkerID        int
	ExecutionResult *models.ExecutionResult
	ProcessingTime  time.Duration
	CompletedAt     time.Time
}

type WorkerError struct {
	TaskID    string
	OrderID   string
	WorkerID  int
	Error     error
	Retries   int
	Timestamp time.Time
	Retryable bool
}

type WorkerPoolMetrics struct {
	TotalTasks       int64
	CompletedTasks   int64
	FailedTasks      int64
	AverageTime      time.Duration
	ActiveWorkers    int
	IdleWorkers      int
	QueueLength      int
	Throughput       float64
	LastUpdateTime   time.Time
	WorkerStats      map[int]*WorkerStats
	mu               sync.RWMutex
}

type WorkerStats struct {
	WorkerID        int
	TasksProcessed  int64
	TasksFailed     int64
	AverageTime     time.Duration
	Status          WorkerStatus
	LastTaskTime    time.Time
	TotalUptime     time.Duration
}

func NewWorkerPool(workerCount, queueSize int, executor *ExecutionService) *WorkerPool {
	return &WorkerPool{
		workers:     make([]*Worker, workerCount),
		taskQueue:   make(chan *WorkerTask, queueSize),
		resultQueue: make(chan *WorkerResult, queueSize),
		errorQueue:  make(chan *WorkerError, queueSize),
		workerCount: workerCount,
		queueSize:   queueSize,
		stopChan:    make(chan struct{}),
		executor:    executor,
		metrics: &WorkerPoolMetrics{
			WorkerStats: make(map[int]*WorkerStats),
		},
	}
}

func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.mu.Lock()
	if wp.running {
		wp.mu.Unlock()
		return fmt.Errorf("worker pool is already running")
	}
	wp.running = true
	wp.mu.Unlock()

	log.Printf("Starting worker pool with %d workers", wp.workerCount)

	for i := 0; i < wp.workerCount; i++ {
		worker := &Worker{
			ID:     i,
			pool:   wp,
			status: WorkerStatusIdle,
		}
		wp.workers[i] = worker

		wp.metrics.WorkerStats[i] = &WorkerStats{
			WorkerID: i,
			Status:   WorkerStatusIdle,
		}

		wp.wg.Add(1)
		go worker.run(ctx)
	}

	wp.wg.Add(1)
	go wp.metricsCollector(ctx)

	wp.wg.Add(1)
	go wp.resultProcessor(ctx)

	return nil
}

func (wp *WorkerPool) Stop() error {
	wp.mu.Lock()
	if !wp.running {
		wp.mu.Unlock()
		return fmt.Errorf("worker pool is not running")
	}
	wp.running = false
	wp.mu.Unlock()

	log.Println("Stopping worker pool...")

	close(wp.stopChan)
	close(wp.taskQueue)

	wp.wg.Wait()

	close(wp.resultQueue)
	close(wp.errorQueue)

	for _, worker := range wp.workers {
		worker.mu.Lock()
		worker.status = WorkerStatusStopped
		worker.mu.Unlock()
	}

	log.Println("Worker pool stopped")
	return nil
}

func (wp *WorkerPool) SubmitTask(task *WorkerTask) error {
	wp.mu.RLock()
	if !wp.running {
		wp.mu.RUnlock()
		return fmt.Errorf("worker pool is not running")
	}
	wp.mu.RUnlock()

	select {
	case wp.taskQueue <- task:
		wp.updateQueueMetrics()
		return nil
	case <-task.Context.Done():
		return task.Context.Err()
	default:
		return fmt.Errorf("task queue is full")
	}
}

func (w *Worker) run(ctx context.Context) {
	defer w.pool.wg.Done()

	log.Printf("Worker %d started", w.ID)
	w.setStatus(WorkerStatusIdle)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping due to context cancellation", w.ID)
			w.setStatus(WorkerStatusStopped)
			return
		case <-w.pool.stopChan:
			log.Printf("Worker %d stopping due to stop signal", w.ID)
			w.setStatus(WorkerStatusStopped)
			return
		case task := <-w.pool.taskQueue:
			if task == nil {
				log.Printf("Worker %d stopping due to closed channel", w.ID)
				w.setStatus(WorkerStatusStopped)
				return
			}

			w.processTask(task)
		}
	}
}

func (w *Worker) processTask(task *WorkerTask) {
	w.setStatus(WorkerStatusProcessing)
	start := time.Now()
	now := time.Now()
	task.StartedAt = &now

	defer func() {
		w.setStatus(WorkerStatusIdle)
		w.mu.Lock()
		w.taskCount++
		w.lastTaskTime = time.Now()
		w.mu.Unlock()
	}()

	log.Printf("Worker %d processing task %s for order %s", w.ID, task.ID, task.Order.ID.Hex())

	executionResult, err := w.pool.executor.ExecuteOrderConcurrent(task.Context, task.Order)
	processingTime := time.Since(start)

	if err != nil {
		workerError := &WorkerError{
			TaskID:    task.ID,
			OrderID:   task.Order.ID.Hex(),
			WorkerID:  w.ID,
			Error:     err,
			Retries:   task.Retries,
			Timestamp: time.Now(),
			Retryable: w.isRetryableError(err),
		}

		select {
		case w.pool.errorQueue <- workerError:
		default:
			log.Printf("Error queue full, dropping error for task %s", task.ID)
		}

		w.pool.updateFailureMetrics(w.ID)

		if workerError.Retryable && task.Retries < task.MaxRetries {
			task.Retries++
			log.Printf("Retrying task %s (attempt %d/%d)", task.ID, task.Retries, task.MaxRetries)

			go func() {
				time.Sleep(time.Duration(task.Retries) * time.Second)
				w.pool.SubmitTask(task)
			}()
		}

		return
	}

	result := &WorkerResult{
		TaskID:          task.ID,
		OrderID:         task.Order.ID.Hex(),
		WorkerID:        w.ID,
		ExecutionResult: executionResult,
		ProcessingTime:  processingTime,
		CompletedAt:     time.Now(),
	}

	select {
	case w.pool.resultQueue <- result:
	default:
		log.Printf("Result queue full, dropping result for task %s", task.ID)
	}

	w.pool.updateSuccessMetrics(w.ID, processingTime)
}

func (w *Worker) setStatus(status WorkerStatus) {
	w.mu.Lock()
	w.status = status
	w.mu.Unlock()

	w.pool.metrics.mu.Lock()
	if stats, exists := w.pool.metrics.WorkerStats[w.ID]; exists {
		stats.Status = status
	}
	w.pool.metrics.mu.Unlock()
}

func (w *Worker) isRetryableError(err error) bool {
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"network error",
		"service unavailable",
		"rate limit",
	}

	errStr := err.Error()
	for _, retryable := range retryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}

	return false
}

func (wp *WorkerPool) resultProcessor(ctx context.Context) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-wp.stopChan:
			return
		case result := <-wp.resultQueue:
			if result == nil {
				return
			}
			wp.handleResult(result)
		case err := <-wp.errorQueue:
			if err == nil {
				return
			}
			wp.handleError(err)
		}
	}
}

func (wp *WorkerPool) handleResult(result *WorkerResult) {
	log.Printf("Task %s completed successfully by worker %d in %v",
		result.TaskID, result.WorkerID, result.ProcessingTime)
}

func (wp *WorkerPool) handleError(workerError *WorkerError) {
	log.Printf("Task %s failed on worker %d (retry %d): %v",
		workerError.TaskID, workerError.WorkerID, workerError.Retries, workerError.Error)
}

func (wp *WorkerPool) metricsCollector(ctx context.Context) {
	defer wp.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-wp.stopChan:
			return
		case <-ticker.C:
			wp.collectMetrics()
		}
	}
}

func (wp *WorkerPool) collectMetrics() {
	wp.metrics.mu.Lock()
	defer wp.metrics.mu.Unlock()

	wp.metrics.QueueLength = len(wp.taskQueue)
	wp.metrics.ActiveWorkers = 0
	wp.metrics.IdleWorkers = 0

	for _, worker := range wp.workers {
		worker.mu.RLock()
		status := worker.status
		worker.mu.RUnlock()

		switch status {
		case WorkerStatusProcessing:
			wp.metrics.ActiveWorkers++
		case WorkerStatusIdle:
			wp.metrics.IdleWorkers++
		}
	}

	if wp.metrics.CompletedTasks > 0 {
		elapsed := time.Since(wp.metrics.LastUpdateTime)
		if elapsed > 0 {
			wp.metrics.Throughput = float64(wp.metrics.CompletedTasks) / elapsed.Seconds()
		}
	}

	wp.metrics.LastUpdateTime = time.Now()
}

func (wp *WorkerPool) updateSuccessMetrics(workerID int, processingTime time.Duration) {
	wp.metrics.mu.Lock()
	defer wp.metrics.mu.Unlock()

	wp.metrics.CompletedTasks++
	wp.metrics.TotalTasks++

	if wp.metrics.AverageTime == 0 {
		wp.metrics.AverageTime = processingTime
	} else {
		wp.metrics.AverageTime = (wp.metrics.AverageTime + processingTime) / 2
	}

	if stats, exists := wp.metrics.WorkerStats[workerID]; exists {
		stats.TasksProcessed++
		if stats.AverageTime == 0 {
			stats.AverageTime = processingTime
		} else {
			stats.AverageTime = (stats.AverageTime + processingTime) / 2
		}
		stats.LastTaskTime = time.Now()
	}
}

func (wp *WorkerPool) updateFailureMetrics(workerID int) {
	wp.metrics.mu.Lock()
	defer wp.metrics.mu.Unlock()

	wp.metrics.FailedTasks++
	wp.metrics.TotalTasks++

	if stats, exists := wp.metrics.WorkerStats[workerID]; exists {
		stats.TasksFailed++
	}
}

func (wp *WorkerPool) updateQueueMetrics() {
	wp.metrics.mu.Lock()
	defer wp.metrics.mu.Unlock()

	wp.metrics.QueueLength = len(wp.taskQueue)
}

func (wp *WorkerPool) GetMetrics() *WorkerPoolMetrics {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()

	metricsCopy := *wp.metrics
	metricsCopy.WorkerStats = make(map[int]*WorkerStats)
	for k, v := range wp.metrics.WorkerStats {
		statsCopy := *v
		metricsCopy.WorkerStats[k] = &statsCopy
	}

	return &metricsCopy
}

func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.running
}

func (wp *WorkerPool) GetQueueLength() int {
	return len(wp.taskQueue)
}

func (wp *WorkerPool) GetWorkerCount() int {
	return wp.workerCount
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr ||
		   (len(s) > len(substr) && contains(s[1:], substr))
}