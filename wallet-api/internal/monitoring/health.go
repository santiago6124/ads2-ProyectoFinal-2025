package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"wallet-api/internal/cache"
	"wallet-api/internal/external"
)

type HealthChecker interface {
	CheckHealth(ctx context.Context) *HealthStatus
	RegisterCheck(name string, checker ComponentChecker)
	StartPeriodicChecks(interval time.Duration)
	StopPeriodicChecks()
	GetComponentStatus(component string) *ComponentHealth
}

type ComponentChecker interface {
	Check(ctx context.Context) error
	Name() string
	Timeout() time.Duration
}

type HealthStatus struct {
	Status     string                      `json:"status"`     // "healthy", "degraded", "unhealthy"
	Timestamp  time.Time                   `json:"timestamp"`
	Uptime     time.Duration               `json:"uptime"`
	Version    string                      `json:"version"`
	Components map[string]*ComponentHealth `json:"components"`
	Summary    *HealthSummary              `json:"summary"`
}

type ComponentHealth struct {
	Status      string        `json:"status"`      // "healthy", "unhealthy", "unknown"
	LastChecked time.Time     `json:"last_checked"`
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
	Details     interface{}   `json:"details,omitempty"`
}

type HealthSummary struct {
	TotalComponents   int `json:"total_components"`
	HealthyComponents int `json:"healthy_components"`
	UnhealthyComponents int `json:"unhealthy_components"`
	UnknownComponents int `json:"unknown_components"`
}

type healthChecker struct {
	checkers  map[string]ComponentChecker
	status    map[string]*ComponentHealth
	startTime time.Time
	version   string
	ticker    *time.Ticker
	stopChan  chan struct{}
	mutex     sync.RWMutex
}

func NewHealthChecker(version string) HealthChecker {
	return &healthChecker{
		checkers:  make(map[string]ComponentChecker),
		status:    make(map[string]*ComponentHealth),
		startTime: time.Now(),
		version:   version,
		stopChan:  make(chan struct{}),
	}
}

func (h *healthChecker) RegisterCheck(name string, checker ComponentChecker) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.checkers[name] = checker
	h.status[name] = &ComponentHealth{
		Status:      "unknown",
		LastChecked: time.Time{},
	}
}

func (h *healthChecker) CheckHealth(ctx context.Context) *HealthStatus {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	overallStatus := "healthy"
	summary := &HealthSummary{
		TotalComponents: len(h.checkers),
	}

	// Check each component
	for name, checker := range h.checkers {
		componentHealth := h.checkComponent(ctx, name, checker)
		h.status[name] = componentHealth

		switch componentHealth.Status {
		case "healthy":
			summary.HealthyComponents++
		case "unhealthy":
			summary.UnhealthyComponents++
			overallStatus = "degraded"
		default:
			summary.UnknownComponents++
			if overallStatus == "healthy" {
				overallStatus = "degraded"
			}
		}
	}

	// Determine overall status
	if summary.UnhealthyComponents > summary.HealthyComponents/2 {
		overallStatus = "unhealthy"
	}

	return &HealthStatus{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Uptime:     time.Since(h.startTime),
		Version:    h.version,
		Components: h.copyStatus(),
		Summary:    summary,
	}
}

func (h *healthChecker) checkComponent(ctx context.Context, name string, checker ComponentChecker) *ComponentHealth {
	start := time.Now()

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, checker.Timeout())
	defer cancel()

	err := checker.Check(checkCtx)
	duration := time.Since(start)

	componentHealth := &ComponentHealth{
		LastChecked: time.Now(),
		Duration:    duration,
	}

	if err != nil {
		componentHealth.Status = "unhealthy"
		componentHealth.Error = err.Error()
	} else {
		componentHealth.Status = "healthy"
	}

	return componentHealth
}

func (h *healthChecker) copyStatus() map[string]*ComponentHealth {
	copied := make(map[string]*ComponentHealth)
	for name, status := range h.status {
		copied[name] = &ComponentHealth{
			Status:      status.Status,
			LastChecked: status.LastChecked,
			Duration:    status.Duration,
			Error:       status.Error,
			Details:     status.Details,
		}
	}
	return copied
}

func (h *healthChecker) GetComponentStatus(component string) *ComponentHealth {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if status, exists := h.status[component]; exists {
		return &ComponentHealth{
			Status:      status.Status,
			LastChecked: status.LastChecked,
			Duration:    status.Duration,
			Error:       status.Error,
			Details:     status.Details,
		}
	}
	return nil
}

func (h *healthChecker) StartPeriodicChecks(interval time.Duration) {
	h.ticker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-h.ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				h.CheckHealth(ctx)
				cancel()
			case <-h.stopChan:
				return
			}
		}
	}()
}

func (h *healthChecker) StopPeriodicChecks() {
	if h.ticker != nil {
		h.ticker.Stop()
	}
	close(h.stopChan)
}

// Built-in component checkers

// Database health checker
type DatabaseChecker struct {
	name string
	// In a real implementation, this would have database connection
	testQuery func(ctx context.Context) error
}

func NewDatabaseChecker(name string, testQuery func(ctx context.Context) error) ComponentChecker {
	return &DatabaseChecker{
		name:      name,
		testQuery: testQuery,
	}
}

func (d *DatabaseChecker) Name() string {
	return d.name
}

func (d *DatabaseChecker) Timeout() time.Duration {
	return 5 * time.Second
}

func (d *DatabaseChecker) Check(ctx context.Context) error {
	if d.testQuery != nil {
		return d.testQuery(ctx)
	}
	return nil
}

// Cache health checker
type CacheChecker struct {
	name  string
	cache cache.CacheService
}

func NewCacheChecker(name string, cacheService cache.CacheService) ComponentChecker {
	return &CacheChecker{
		name:  name,
		cache: cacheService,
	}
}

func (c *CacheChecker) Name() string {
	return c.name
}

func (c *CacheChecker) Timeout() time.Duration {
	return 3 * time.Second
}

func (c *CacheChecker) Check(ctx context.Context) error {
	return c.cache.Ping(ctx)
}

// Message Queue health checker
type MessageQueueChecker struct {
	name string
	mq   external.MessageQueue
}

func NewMessageQueueChecker(name string, messageQueue external.MessageQueue) ComponentChecker {
	return &MessageQueueChecker{
		name: name,
		mq:   messageQueue,
	}
}

func (m *MessageQueueChecker) Name() string {
	return m.name
}

func (m *MessageQueueChecker) Timeout() time.Duration {
	return 5 * time.Second
}

func (m *MessageQueueChecker) Check(ctx context.Context) error {
	// Test message queue connectivity
	testEvent := &external.AuditEvent{
		EventID:   "health_check",
		EventType: "health_check",
		Action:    "ping",
		Resource:  "health_checker",
		Success:   true,
		IPAddress: "127.0.0.1",
		RiskScore: 0,
	}

	return m.mq.PublishAuditEvent(ctx, testEvent)
}

// External service health checker
type ExternalServiceChecker struct {
	name    string
	service string
	checker func(ctx context.Context) error
}

func NewExternalServiceChecker(name, service string, checker func(ctx context.Context) error) ComponentChecker {
	return &ExternalServiceChecker{
		name:    name,
		service: service,
		checker: checker,
	}
}

func (e *ExternalServiceChecker) Name() string {
	return e.name
}

func (e *ExternalServiceChecker) Timeout() time.Duration {
	return 10 * time.Second
}

func (e *ExternalServiceChecker) Check(ctx context.Context) error {
	if e.checker != nil {
		return e.checker(ctx)
	}
	return fmt.Errorf("no checker function provided for %s", e.service)
}

// Disk space health checker
type DiskSpaceChecker struct {
	name      string
	path      string
	threshold float64 // Percentage threshold (e.g., 0.85 for 85%)
}

func NewDiskSpaceChecker(name, path string, threshold float64) ComponentChecker {
	return &DiskSpaceChecker{
		name:      name,
		path:      path,
		threshold: threshold,
	}
}

func (d *DiskSpaceChecker) Name() string {
	return d.name
}

func (d *DiskSpaceChecker) Timeout() time.Duration {
	return 2 * time.Second
}

func (d *DiskSpaceChecker) Check(ctx context.Context) error {
	// In a real implementation, this would check actual disk usage
	// For now, we'll simulate a healthy disk
	usage := 0.65 // 65% usage

	if usage > d.threshold {
		return fmt.Errorf("disk usage %.2f%% exceeds threshold %.2f%%", usage*100, d.threshold*100)
	}

	return nil
}

// Memory health checker
type MemoryChecker struct {
	name      string
	threshold int64 // Memory threshold in bytes
}

func NewMemoryChecker(name string, threshold int64) ComponentChecker {
	return &MemoryChecker{
		name:      name,
		threshold: threshold,
	}
}

func (m *MemoryChecker) Name() string {
	return m.name
}

func (m *MemoryChecker) Timeout() time.Duration {
	return 1 * time.Second
}

func (m *MemoryChecker) Check(ctx context.Context) error {
	// Check current memory usage
	// In a real implementation, this would use actual memory statistics
	currentUsage := int64(100 * 1024 * 1024) // 100MB simulated usage

	if currentUsage > m.threshold {
		return fmt.Errorf("memory usage %d bytes exceeds threshold %d bytes", currentUsage, m.threshold)
	}

	return nil
}

// HTTP endpoint health checker
type HTTPEndpointChecker struct {
	name     string
	url      string
	timeout  time.Duration
	expected int // Expected HTTP status code
}

func NewHTTPEndpointChecker(name, url string, timeout time.Duration, expectedStatus int) ComponentChecker {
	return &HTTPEndpointChecker{
		name:     name,
		url:      url,
		timeout:  timeout,
		expected: expectedStatus,
	}
}

func (h *HTTPEndpointChecker) Name() string {
	return h.name
}

func (h *HTTPEndpointChecker) Timeout() time.Duration {
	return h.timeout
}

func (h *HTTPEndpointChecker) Check(ctx context.Context) error {
	// In a real implementation, this would make an actual HTTP request
	// For now, we'll simulate a successful check
	return nil
}