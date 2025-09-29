package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthChecker provides comprehensive health checking functionality
type HealthChecker struct {
	checks   map[string]HealthCheck
	mu       sync.RWMutex
	logger   *logrus.Logger
	lastRun  time.Time
	interval time.Duration
}

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string
	CheckFunc   func(ctx context.Context) error
	Timeout     time.Duration
	Critical    bool
	Description string
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status     string                    `json:"status"`
	Timestamp  string                    `json:"timestamp"`
	Uptime     string                    `json:"uptime"`
	Version    string                    `json:"version"`
	Checks     map[string]CheckResult    `json:"checks"`
	System     SystemHealth              `json:"system"`
	startTime  time.Time
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Status      string        `json:"status"`
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
	LastChecked string        `json:"last_checked"`
	Critical    bool          `json:"critical"`
	Description string        `json:"description"`
}

// SystemHealth represents system-level health information
type SystemHealth struct {
	MemoryStats    MemoryStats    `json:"memory"`
	GoroutineCount int            `json:"goroutine_count"`
	CPUCount       int            `json:"cpu_count"`
	GoVersion      string         `json:"go_version"`
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	Alloc        uint64 `json:"alloc_bytes"`
	TotalAlloc   uint64 `json:"total_alloc_bytes"`
	Sys          uint64 `json:"sys_bytes"`
	NumGC        uint32 `json:"num_gc"`
	PauseTotalNs uint64 `json:"pause_total_ns"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(logger *logrus.Logger) *HealthChecker {
	return &HealthChecker{
		checks:   make(map[string]HealthCheck),
		logger:   logger,
		interval: 30 * time.Second,
	}
}

// RegisterCheck registers a new health check
func (hc *HealthChecker) RegisterCheck(name string, checkFunc func(ctx context.Context) error, timeout time.Duration, critical bool, description string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[name] = HealthCheck{
		Name:        name,
		CheckFunc:   checkFunc,
		Timeout:     timeout,
		Critical:    critical,
		Description: description,
	}

	hc.logger.WithFields(logrus.Fields{
		"check":       name,
		"critical":    critical,
		"timeout":     timeout,
		"description": description,
	}).Info("Health check registered")
}

// RemoveCheck removes a health check
func (hc *HealthChecker) RemoveCheck(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	delete(hc.checks, name)
	hc.logger.WithField("check", name).Info("Health check removed")
}

// CheckHealth performs all registered health checks
func (hc *HealthChecker) CheckHealth(ctx context.Context, version string, startTime time.Time) *HealthStatus {
	hc.mu.RLock()
	checks := make(map[string]HealthCheck)
	for k, v := range hc.checks {
		checks[k] = v
	}
	hc.mu.RUnlock()

	status := &HealthStatus{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    time.Since(startTime).String(),
		Version:   version,
		Checks:    make(map[string]CheckResult),
		System:    hc.getSystemHealth(),
		startTime: startTime,
	}

	overallHealthy := true
	checkResults := make(chan struct {
		name   string
		result CheckResult
	}, len(checks))

	// Run all checks concurrently
	var wg sync.WaitGroup
	for name, check := range checks {
		wg.Add(1)
		go func(name string, check HealthCheck) {
			defer wg.Done()

			checkCtx, cancel := context.WithTimeout(ctx, check.Timeout)
			defer cancel()

			startTime := time.Now()
			result := CheckResult{
				Status:      "healthy",
				Critical:    check.Critical,
				Description: check.Description,
				LastChecked: startTime.UTC().Format(time.RFC3339),
			}

			err := check.CheckFunc(checkCtx)
			result.Duration = time.Since(startTime)

			if err != nil {
				result.Status = "unhealthy"
				result.Error = err.Error()

				hc.logger.WithFields(logrus.Fields{
					"check":    name,
					"error":    err,
					"duration": result.Duration,
					"critical": check.Critical,
				}).Error("Health check failed")

				if check.Critical {
					overallHealthy = false
				}
			} else {
				hc.logger.WithFields(logrus.Fields{
					"check":    name,
					"duration": result.Duration,
				}).Debug("Health check passed")
			}

			checkResults <- struct {
				name   string
				result CheckResult
			}{name, result}
		}(name, check)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(checkResults)
	}()

	// Collect results
	for result := range checkResults {
		status.Checks[result.name] = result.result
	}

	if overallHealthy {
		status.Status = "healthy"
	} else {
		status.Status = "unhealthy"
	}

	hc.lastRun = time.Now()

	hc.logger.WithFields(logrus.Fields{
		"status":      status.Status,
		"checks_run":  len(status.Checks),
		"duration":    time.Since(time.Now().Add(-time.Since(hc.lastRun))),
	}).Info("Health check completed")

	return status
}

// getSystemHealth collects system health information
func (hc *HealthChecker) getSystemHealth() SystemHealth {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemHealth{
		MemoryStats: MemoryStats{
			Alloc:        m.Alloc,
			TotalAlloc:   m.TotalAlloc,
			Sys:          m.Sys,
			NumGC:        m.NumGC,
			PauseTotalNs: m.PauseTotalNs,
		},
		GoroutineCount: runtime.NumGoroutine(),
		CPUCount:       runtime.NumCPU(),
		GoVersion:      runtime.Version(),
	}
}

// StartPeriodicChecks starts periodic health checks
func (hc *HealthChecker) StartPeriodicChecks(ctx context.Context, version string, startTime time.Time) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	hc.logger.WithField("interval", hc.interval).Info("Starting periodic health checks")

	for {
		select {
		case <-ctx.Done():
			hc.logger.Info("Stopping periodic health checks")
			return
		case <-ticker.C:
			status := hc.CheckHealth(ctx, version, startTime)
			if status.Status != "healthy" {
				hc.logger.WithField("status", status.Status).Warn("System health check failed")
			}
		}
	}
}

// GetLastCheckTime returns the time of the last health check
func (hc *HealthChecker) GetLastCheckTime() time.Time {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.lastRun
}

// SetCheckInterval sets the interval for periodic health checks
func (hc *HealthChecker) SetCheckInterval(interval time.Duration) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.interval = interval
}

// CreateSolrHealthCheck creates a health check for Solr
func CreateSolrHealthCheck(pingFunc func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if err := pingFunc(ctx); err != nil {
			return fmt.Errorf("solr ping failed: %w", err)
		}
		return nil
	}
}

// CreateCacheHealthCheck creates a health check for cache
func CreateCacheHealthCheck(pingFunc func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if err := pingFunc(ctx); err != nil {
			return fmt.Errorf("cache ping failed: %w", err)
		}
		return nil
	}
}

// CreateDatabaseHealthCheck creates a health check for database connectivity
func CreateDatabaseHealthCheck(pingFunc func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if err := pingFunc(ctx); err != nil {
			return fmt.Errorf("database ping failed: %w", err)
		}
		return nil
	}
}

// CreateMemoryHealthCheck creates a health check for memory usage
func CreateMemoryHealthCheck(maxMemoryMB uint64) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		allocMB := m.Alloc / 1024 / 1024
		if allocMB > maxMemoryMB {
			return fmt.Errorf("memory usage too high: %d MB (limit: %d MB)", allocMB, maxMemoryMB)
		}
		return nil
	}
}

// CreateGoroutineHealthCheck creates a health check for goroutine count
func CreateGoroutineHealthCheck(maxGoroutines int) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		count := runtime.NumGoroutine()
		if count > maxGoroutines {
			return fmt.Errorf("too many goroutines: %d (limit: %d)", count, maxGoroutines)
		}
		return nil
	}
}