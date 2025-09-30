package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricsService interface {
	// HTTP metrics
	RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration)
	IncrementHTTPErrors(method, endpoint string, errorType string)

	// Transaction metrics
	RecordTransaction(transactionType, status string, amount float64, duration time.Duration)
	IncrementTransactionErrors(transactionType, errorType string)
	RecordTransactionVolume(currency string, amount float64)

	// Wallet metrics
	RecordWalletOperation(operation, status string, duration time.Duration)
	IncrementActiveWallets(currency string)
	DecrementActiveWallets(currency string)
	RecordWalletBalance(currency string, balance float64)

	// Cache metrics
	RecordCacheOperation(operation string, hit bool, duration time.Duration)
	IncrementCacheSize(cacheType string, size int64)

	// Database metrics
	RecordDatabaseOperation(operation, collection string, duration time.Duration)
	IncrementDatabaseErrors(operation, collection, errorType string)
	RecordDatabaseConnections(active, idle int)

	// External service metrics
	RecordExternalServiceCall(service, operation string, success bool, duration time.Duration)
	IncrementExternalServiceErrors(service, operation, errorType string)

	// Business metrics
	RecordComplianceCheck(checkType string, result string, riskScore int, duration time.Duration)
	RecordFraudDetection(alertType string, riskLevel string, action string)

	// System metrics
	RecordSystemMetrics()
	GetMetrics() map[string]interface{}
}

type prometheusMetrics struct {
	// HTTP metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpErrorsTotal      *prometheus.CounterVec

	// Transaction metrics
	transactionsTotal        *prometheus.CounterVec
	transactionDuration      *prometheus.HistogramVec
	transactionErrorsTotal   *prometheus.CounterVec
	transactionVolumeTotal   *prometheus.CounterVec
	transactionAmountGauge   *prometheus.GaugeVec

	// Wallet metrics
	walletOperationsTotal     *prometheus.CounterVec
	walletOperationDuration   *prometheus.HistogramVec
	activeWalletsGauge        *prometheus.GaugeVec
	walletBalanceGauge        *prometheus.GaugeVec

	// Cache metrics
	cacheOperationsTotal     *prometheus.CounterVec
	cacheOperationDuration   *prometheus.HistogramVec
	cacheHitRatio           *prometheus.GaugeVec
	cacheSizeGauge          *prometheus.GaugeVec

	// Database metrics
	databaseOperationsTotal    *prometheus.CounterVec
	databaseOperationDuration  *prometheus.HistogramVec
	databaseErrorsTotal        *prometheus.CounterVec
	databaseConnectionsGauge   *prometheus.GaugeVec

	// External service metrics
	externalServiceCallsTotal    *prometheus.CounterVec
	externalServiceDuration      *prometheus.HistogramVec
	externalServiceErrorsTotal   *prometheus.CounterVec

	// Business metrics
	complianceChecksTotal     *prometheus.CounterVec
	complianceCheckDuration   *prometheus.HistogramVec
	fraudDetectionAlertsTotal *prometheus.CounterVec
	riskScoreGauge           *prometheus.GaugeVec

	// System metrics
	memoryUsageGauge     prometheus.Gauge
	goroutineCountGauge  prometheus.Gauge
	cpuUsageGauge        prometheus.Gauge
	uptimeGauge          prometheus.Gauge

	startTime time.Time
	mutex     sync.RWMutex
}

func NewPrometheusMetrics() MetricsService {
	m := &prometheusMetrics{
		startTime: time.Now(),
	}

	m.initMetrics()
	return m
}

func (m *prometheusMetrics) initMetrics() {
	// HTTP metrics
	m.httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	m.httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wallet_api_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	m.httpErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_http_errors_total",
			Help: "Total number of HTTP errors",
		},
		[]string{"method", "endpoint", "error_type"},
	)

	// Transaction metrics
	m.transactionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_transactions_total",
			Help: "Total number of transactions",
		},
		[]string{"type", "status"},
	)

	m.transactionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wallet_api_transaction_duration_seconds",
			Help:    "Transaction processing duration in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"type"},
	)

	m.transactionErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_transaction_errors_total",
			Help: "Total number of transaction errors",
		},
		[]string{"type", "error_type"},
	)

	m.transactionVolumeTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_transaction_volume_total",
			Help: "Total transaction volume",
		},
		[]string{"currency"},
	)

	m.transactionAmountGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wallet_api_transaction_amount",
			Help: "Current transaction amount",
		},
		[]string{"currency", "type"},
	)

	// Wallet metrics
	m.walletOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_wallet_operations_total",
			Help: "Total number of wallet operations",
		},
		[]string{"operation", "status"},
	)

	m.walletOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wallet_api_wallet_operation_duration_seconds",
			Help:    "Wallet operation duration in seconds",
			Buckets: []float64{0.01, 0.1, 0.5, 1.0, 2.0},
		},
		[]string{"operation"},
	)

	m.activeWalletsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wallet_api_active_wallets",
			Help: "Number of active wallets",
		},
		[]string{"currency"},
	)

	m.walletBalanceGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wallet_api_wallet_balance",
			Help: "Current wallet balance",
		},
		[]string{"currency"},
	)

	// Cache metrics
	m.cacheOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "result"},
	)

	m.cacheOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wallet_api_cache_operation_duration_seconds",
			Help:    "Cache operation duration in seconds",
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1.0},
		},
		[]string{"operation"},
	)

	m.cacheHitRatio = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wallet_api_cache_hit_ratio",
			Help: "Cache hit ratio",
		},
		[]string{"cache_type"},
	)

	m.cacheSizeGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wallet_api_cache_size",
			Help: "Current cache size",
		},
		[]string{"cache_type"},
	)

	// Database metrics
	m.databaseOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "collection"},
	)

	m.databaseOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wallet_api_database_operation_duration_seconds",
			Help:    "Database operation duration in seconds",
			Buckets: []float64{0.01, 0.1, 0.5, 1.0, 5.0},
		},
		[]string{"operation", "collection"},
	)

	m.databaseErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_database_errors_total",
			Help: "Total number of database errors",
		},
		[]string{"operation", "collection", "error_type"},
	)

	m.databaseConnectionsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wallet_api_database_connections",
			Help: "Current database connections",
		},
		[]string{"state"},
	)

	// External service metrics
	m.externalServiceCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_external_service_calls_total",
			Help: "Total number of external service calls",
		},
		[]string{"service", "operation", "success"},
	)

	m.externalServiceDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wallet_api_external_service_duration_seconds",
			Help:    "External service call duration in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 5.0, 10.0, 30.0},
		},
		[]string{"service", "operation"},
	)

	m.externalServiceErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_external_service_errors_total",
			Help: "Total number of external service errors",
		},
		[]string{"service", "operation", "error_type"},
	)

	// Business metrics
	m.complianceChecksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_compliance_checks_total",
			Help: "Total number of compliance checks",
		},
		[]string{"check_type", "result"},
	)

	m.complianceCheckDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wallet_api_compliance_check_duration_seconds",
			Help:    "Compliance check duration in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0},
		},
		[]string{"check_type"},
	)

	m.fraudDetectionAlertsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_api_fraud_detection_alerts_total",
			Help: "Total number of fraud detection alerts",
		},
		[]string{"alert_type", "risk_level", "action"},
	)

	m.riskScoreGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wallet_api_risk_score",
			Help: "Current risk score",
		},
		[]string{"user_id", "score_type"},
	)

	// System metrics
	m.memoryUsageGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wallet_api_memory_usage_bytes",
			Help: "Current memory usage in bytes",
		},
	)

	m.goroutineCountGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wallet_api_goroutines_count",
			Help: "Current number of goroutines",
		},
	)

	m.cpuUsageGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wallet_api_cpu_usage_percent",
			Help: "Current CPU usage percentage",
		},
	)

	m.uptimeGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wallet_api_uptime_seconds",
			Help: "Application uptime in seconds",
		},
	)
}

// HTTP metrics implementation
func (m *prometheusMetrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	m.httpRequestsTotal.WithLabelValues(method, endpoint, fmt.Sprintf("%d", statusCode)).Inc()
	m.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func (m *prometheusMetrics) IncrementHTTPErrors(method, endpoint string, errorType string) {
	m.httpErrorsTotal.WithLabelValues(method, endpoint, errorType).Inc()
}

// Transaction metrics implementation
func (m *prometheusMetrics) RecordTransaction(transactionType, status string, amount float64, duration time.Duration) {
	m.transactionsTotal.WithLabelValues(transactionType, status).Inc()
	m.transactionDuration.WithLabelValues(transactionType).Observe(duration.Seconds())
}

func (m *prometheusMetrics) IncrementTransactionErrors(transactionType, errorType string) {
	m.transactionErrorsTotal.WithLabelValues(transactionType, errorType).Inc()
}

func (m *prometheusMetrics) RecordTransactionVolume(currency string, amount float64) {
	m.transactionVolumeTotal.WithLabelValues(currency).Add(amount)
}

// Wallet metrics implementation
func (m *prometheusMetrics) RecordWalletOperation(operation, status string, duration time.Duration) {
	m.walletOperationsTotal.WithLabelValues(operation, status).Inc()
	m.walletOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

func (m *prometheusMetrics) IncrementActiveWallets(currency string) {
	m.activeWalletsGauge.WithLabelValues(currency).Inc()
}

func (m *prometheusMetrics) DecrementActiveWallets(currency string) {
	m.activeWalletsGauge.WithLabelValues(currency).Dec()
}

func (m *prometheusMetrics) RecordWalletBalance(currency string, balance float64) {
	m.walletBalanceGauge.WithLabelValues(currency).Set(balance)
}

// Cache metrics implementation
func (m *prometheusMetrics) RecordCacheOperation(operation string, hit bool, duration time.Duration) {
	result := "miss"
	if hit {
		result = "hit"
	}
	m.cacheOperationsTotal.WithLabelValues(operation, result).Inc()
	m.cacheOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

func (m *prometheusMetrics) IncrementCacheSize(cacheType string, size int64) {
	m.cacheSizeGauge.WithLabelValues(cacheType).Add(float64(size))
}

// Database metrics implementation
func (m *prometheusMetrics) RecordDatabaseOperation(operation, collection string, duration time.Duration) {
	m.databaseOperationsTotal.WithLabelValues(operation, collection).Inc()
	m.databaseOperationDuration.WithLabelValues(operation, collection).Observe(duration.Seconds())
}

func (m *prometheusMetrics) IncrementDatabaseErrors(operation, collection, errorType string) {
	m.databaseErrorsTotal.WithLabelValues(operation, collection, errorType).Inc()
}

func (m *prometheusMetrics) RecordDatabaseConnections(active, idle int) {
	m.databaseConnectionsGauge.WithLabelValues("active").Set(float64(active))
	m.databaseConnectionsGauge.WithLabelValues("idle").Set(float64(idle))
}

// External service metrics implementation
func (m *prometheusMetrics) RecordExternalServiceCall(service, operation string, success bool, duration time.Duration) {
	successStr := "false"
	if success {
		successStr = "true"
	}
	m.externalServiceCallsTotal.WithLabelValues(service, operation, successStr).Inc()
	m.externalServiceDuration.WithLabelValues(service, operation).Observe(duration.Seconds())
}

func (m *prometheusMetrics) IncrementExternalServiceErrors(service, operation, errorType string) {
	m.externalServiceErrorsTotal.WithLabelValues(service, operation, errorType).Inc()
}

// Business metrics implementation
func (m *prometheusMetrics) RecordComplianceCheck(checkType string, result string, riskScore int, duration time.Duration) {
	m.complianceChecksTotal.WithLabelValues(checkType, result).Inc()
	m.complianceCheckDuration.WithLabelValues(checkType).Observe(duration.Seconds())
}

func (m *prometheusMetrics) RecordFraudDetection(alertType string, riskLevel string, action string) {
	m.fraudDetectionAlertsTotal.WithLabelValues(alertType, riskLevel, action).Inc()
}

// System metrics implementation
func (m *prometheusMetrics) RecordSystemMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.memoryUsageGauge.Set(float64(memStats.Alloc))
	m.goroutineCountGauge.Set(float64(runtime.NumGoroutine()))
	m.uptimeGauge.Set(time.Since(m.startTime).Seconds())
}

func (m *prometheusMetrics) GetMetrics() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"memory_usage":     memStats.Alloc,
		"goroutine_count":  runtime.NumGoroutine(),
		"uptime_seconds":   time.Since(m.startTime).Seconds(),
		"start_time":       m.startTime,
	}
}

// Metrics middleware for automatic HTTP request tracking
type MetricsMiddleware struct {
	metrics MetricsService
}

func NewMetricsMiddleware(metrics MetricsService) *MetricsMiddleware {
	return &MetricsMiddleware{metrics: metrics}
}

// Helper function to start system metrics recording
func StartSystemMetricsRecording(metrics MetricsService, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			metrics.RecordSystemMetrics()
		}
	}()
}