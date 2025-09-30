package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type LoggingMiddleware struct {
	logger      *logrus.Logger
	auditLogger *logrus.Logger
	config      *LoggingConfig
}

type LoggingConfig struct {
	EnableRequestLogging    bool
	EnableResponseLogging   bool
	EnableBodyLogging       bool
	EnableAuditLogging      bool
	EnableMetrics           bool
	LogSensitiveData        bool
	MaxBodySize             int64
	SensitiveFields         []string
	ExcludePaths            []string
	SlowRequestThreshold    time.Duration
	LogLevel                logrus.Level
}

type RequestLog struct {
	RequestID    string            `json:"request_id"`
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	RemoteAddr   string            `json:"remote_addr"`
	UserAgent    string            `json:"user_agent"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         interface{}       `json:"body,omitempty"`
	UserID       int64             `json:"user_id,omitempty"`
	Username     string            `json:"username,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
	ContentType  string            `json:"content_type"`
	ContentLength int64            `json:"content_length"`
}

type ResponseLog struct {
	RequestID     string            `json:"request_id"`
	StatusCode    int               `json:"status_code"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          interface{}       `json:"body,omitempty"`
	Duration      time.Duration     `json:"duration_ms"`
	Size          int               `json:"size_bytes"`
	Timestamp     time.Time         `json:"timestamp"`
	Error         string            `json:"error,omitempty"`
}

type AuditLog struct {
	RequestID     string                 `json:"request_id"`
	UserID        int64                  `json:"user_id,omitempty"`
	AdminID       string                 `json:"admin_id,omitempty"`
	Action        string                 `json:"action"`
	Resource      string                 `json:"resource"`
	Method        string                 `json:"method"`
	URL           string                 `json:"url"`
	RemoteAddr    string                 `json:"remote_addr"`
	UserAgent     string                 `json:"user_agent"`
	Success       bool                   `json:"success"`
	StatusCode    int                    `json:"status_code"`
	Duration      time.Duration          `json:"duration_ms"`
	RequestData   map[string]interface{} `json:"request_data,omitempty"`
	ResponseData  map[string]interface{} `json:"response_data,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	ComplianceFlags []string             `json:"compliance_flags,omitempty"`
	RiskScore     int                    `json:"risk_score,omitempty"`
}

type MetricsData struct {
	RequestCount      int64                    `json:"request_count"`
	ResponseTime      time.Duration            `json:"response_time"`
	StatusCodes       map[int]int64            `json:"status_codes"`
	Endpoints         map[string]int64         `json:"endpoints"`
	UserActivity      map[int64]int64          `json:"user_activity"`
	ErrorRate         float64                  `json:"error_rate"`
	SlowRequests      int64                    `json:"slow_requests"`
	Timestamp         time.Time                `json:"timestamp"`
}

func NewLoggingMiddleware(logger, auditLogger *logrus.Logger, config *LoggingConfig) *LoggingMiddleware {
	if config == nil {
		config = &LoggingConfig{
			EnableRequestLogging:  true,
			EnableResponseLogging: true,
			EnableBodyLogging:     false,
			EnableAuditLogging:    true,
			EnableMetrics:         true,
			LogSensitiveData:      false,
			MaxBodySize:           10 * 1024, // 10KB
			SensitiveFields:       []string{"password", "token", "secret", "key", "authorization"},
			ExcludePaths:          []string{"/health", "/ready", "/metrics"},
			SlowRequestThreshold:  2 * time.Second,
			LogLevel:              logrus.InfoLevel,
		}
	}

	return &LoggingMiddleware{
		logger:      logger,
		auditLogger: auditLogger,
		config:      config,
	}
}

// RequestResponseLogger logs detailed request and response information
func (l *LoggingMiddleware) RequestResponseLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Skip excluded paths
		if l.shouldExcludePath(param.Path) {
			return ""
		}

		// Create structured log entry
		logEntry := logrus.WithFields(logrus.Fields{
			"request_id":     param.Keys["request_id"],
			"method":         param.Method,
			"path":           param.Path,
			"status_code":    param.StatusCode,
			"latency":        param.Latency.Milliseconds(),
			"client_ip":      param.ClientIP,
			"user_agent":     param.Request.UserAgent(),
			"response_size":  param.BodySize,
			"timestamp":      param.TimeStamp.Format(time.RFC3339),
		})

		// Add user context if available
		if userID, exists := param.Keys["user_id"]; exists {
			logEntry = logEntry.WithField("user_id", userID)
		}

		// Log slow requests as warnings
		if param.Latency > l.config.SlowRequestThreshold {
			logEntry = logEntry.WithField("slow_request", true)
			logEntry.Warn("Slow request detected")
		}

		// Log errors
		if param.StatusCode >= 400 {
			if param.ErrorMessage != "" {
				logEntry = logEntry.WithField("error", param.ErrorMessage)
			}

			if param.StatusCode >= 500 {
				logEntry.Error("Server error")
			} else {
				logEntry.Warn("Client error")
			}
		} else {
			logEntry.Info("Request completed")
		}

		return ""
	})
}

// DetailedRequestLogger logs comprehensive request details
func (l *LoggingMiddleware) DetailedRequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.config.EnableRequestLogging || l.shouldExcludePath(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		requestID := l.getRequestID(c)

		// Create request log
		requestLog := &RequestLog{
			RequestID:     requestID,
			Method:        c.Request.Method,
			URL:           c.Request.URL.String(),
			RemoteAddr:    c.ClientIP(),
			UserAgent:     c.Request.UserAgent(),
			Timestamp:     start,
			ContentType:   c.GetHeader("Content-Type"),
			ContentLength: c.Request.ContentLength,
		}

		// Add user context
		if userID, exists := c.Get("user_id"); exists {
			requestLog.UserID = userID.(int64)
		}
		if username, exists := c.Get("username"); exists {
			requestLog.Username = username.(string)
		}

		// Log headers (excluding sensitive ones)
		if l.logger.Level >= logrus.DebugLevel {
			requestLog.Headers = l.sanitizeHeaders(c.Request.Header)
		}

		// Log request body if enabled
		if l.config.EnableBodyLogging && l.shouldLogBody(c.Request.Method) {
			if body := l.captureRequestBody(c); body != nil {
				requestLog.Body = body
			}
		}

		// Log request
		l.logger.WithFields(logrus.Fields{
			"type":    "request",
			"details": requestLog,
		}).Info("HTTP Request")

		c.Next()

		// Log response
		if l.config.EnableResponseLogging {
			l.logResponse(c, requestID, time.Since(start))
		}

		// Log audit trail
		if l.config.EnableAuditLogging {
			l.logAuditTrail(c, requestID, start, time.Since(start))
		}
	}
}

// AuditLogger logs security and compliance events
func (l *LoggingMiddleware) AuditLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.config.EnableAuditLogging || l.shouldExcludePath(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		requestID := l.getRequestID(c)

		c.Next()

		// Create audit log entry
		auditLog := &AuditLog{
			RequestID:  requestID,
			Action:     l.determineAction(c.Request.Method, c.Request.URL.Path),
			Resource:   l.determineResource(c.Request.URL.Path),
			Method:     c.Request.Method,
			URL:        c.Request.URL.String(),
			RemoteAddr: c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			Success:    c.Writer.Status() < 400,
			StatusCode: c.Writer.Status(),
			Duration:   time.Since(start),
			Timestamp:  start,
		}

		// Add user context
		if userID, exists := c.Get("user_id"); exists {
			auditLog.UserID = userID.(int64)
		}
		if adminID, exists := c.Get("admin_id"); exists {
			auditLog.AdminID = adminID.(string)
		}

		// Add request data for important operations
		if l.isImportantOperation(c.Request.URL.Path) {
			auditLog.RequestData = l.extractRequestData(c)
		}

		// Add compliance flags
		auditLog.ComplianceFlags = l.generateComplianceFlags(c)

		// Calculate risk score
		auditLog.RiskScore = l.calculateRiskScore(c, auditLog)

		// Add error information
		if !auditLog.Success {
			if errorMsg, exists := c.Get("error_message"); exists {
				auditLog.ErrorMessage = errorMsg.(string)
			}
		}

		// Log to audit logger
		l.auditLogger.WithFields(logrus.Fields{
			"type":    "audit",
			"details": auditLog,
		}).Info("Audit Event")

		// Also log high-risk events to main logger
		if auditLog.RiskScore >= 8 {
			l.logger.WithFields(logrus.Fields{
				"type":       "high_risk_audit",
				"risk_score": auditLog.RiskScore,
				"details":    auditLog,
			}).Warn("High risk audit event")
		}
	}
}

// MetricsCollector collects performance and usage metrics
func (l *LoggingMiddleware) MetricsCollector() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.config.EnableMetrics || l.shouldExcludePath(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()

		c.Next()

		duration := time.Since(start)

		// Collect metrics
		metrics := &MetricsData{
			RequestCount: 1,
			ResponseTime: duration,
			StatusCodes:  map[int]int64{c.Writer.Status(): 1},
			Endpoints:    map[string]int64{c.Request.URL.Path: 1},
			Timestamp:    start,
		}

		// Add user activity
		if userID, exists := c.Get("user_id"); exists {
			metrics.UserActivity = map[int64]int64{userID.(int64): 1}
		}

		// Calculate error rate
		if c.Writer.Status() >= 400 {
			metrics.ErrorRate = 1.0
		}

		// Track slow requests
		if duration > l.config.SlowRequestThreshold {
			metrics.SlowRequests = 1
		}

		// Log metrics (in production, this would go to metrics system like Prometheus)
		l.logger.WithFields(logrus.Fields{
			"type":    "metrics",
			"details": metrics,
		}).Debug("Request Metrics")
	}
}

// HealthCheckLogger logs health check requests separately
func (l *LoggingMiddleware) HealthCheckLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if l.isHealthCheckPath(c.Request.URL.Path) {
			start := time.Now()
			c.Next()

			// Only log failed health checks
			if c.Writer.Status() >= 400 {
				l.logger.WithFields(logrus.Fields{
					"type":        "health_check",
					"path":        c.Request.URL.Path,
					"status_code": c.Writer.Status(),
					"duration":    time.Since(start).Milliseconds(),
					"timestamp":   start.Format(time.RFC3339),
				}).Warn("Health check failed")
			}
		} else {
			c.Next()
		}
	}
}

// Helper methods
func (l *LoggingMiddleware) getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		return requestID.(string)
	}
	return c.GetHeader("X-Request-ID")
}

func (l *LoggingMiddleware) shouldExcludePath(path string) bool {
	for _, excludePath := range l.config.ExcludePaths {
		if strings.HasPrefix(path, excludePath) {
			return true
		}
	}
	return false
}

func (l *LoggingMiddleware) shouldLogBody(method string) bool {
	return method == "POST" || method == "PUT" || method == "PATCH"
}

func (l *LoggingMiddleware) sanitizeHeaders(headers http.Header) map[string]string {
	sanitized := make(map[string]string)

	for name, values := range headers {
		lowerName := strings.ToLower(name)

		// Skip sensitive headers
		isSensitive := false
		for _, sensitive := range l.config.SensitiveFields {
			if strings.Contains(lowerName, sensitive) {
				isSensitive = true
				break
			}
		}

		if !isSensitive && len(values) > 0 {
			sanitized[name] = values[0]
		} else if isSensitive {
			sanitized[name] = "[REDACTED]"
		}
	}

	return sanitized
}

func (l *LoggingMiddleware) captureRequestBody(c *gin.Context) interface{} {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return nil
	}

	if c.Request.ContentLength > l.config.MaxBodySize {
		return map[string]string{"message": "Body too large to log"}
	}

	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return map[string]string{"error": "Failed to read body"}
	}

	// Restore body for next handlers
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// Try to parse as JSON
	var jsonBody interface{}
	if err := json.Unmarshal(body, &jsonBody); err == nil {
		return l.sanitizeJSONBody(jsonBody)
	}

	// Return as string if not JSON
	return string(body)
}

func (l *LoggingMiddleware) sanitizeJSONBody(body interface{}) interface{} {
	if !l.config.LogSensitiveData {
		return l.redactSensitiveFields(body)
	}
	return body
}

func (l *LoggingMiddleware) redactSensitiveFields(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			lowerKey := strings.ToLower(key)
			isSensitive := false

			for _, sensitive := range l.config.SensitiveFields {
				if strings.Contains(lowerKey, sensitive) {
					isSensitive = true
					break
				}
			}

			if isSensitive {
				result[key] = "[REDACTED]"
			} else {
				result[key] = l.redactSensitiveFields(value)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = l.redactSensitiveFields(item)
		}
		return result
	default:
		return v
	}
}

func (l *LoggingMiddleware) logResponse(c *gin.Context, requestID string, duration time.Duration) {
	responseLog := &ResponseLog{
		RequestID:  requestID,
		StatusCode: c.Writer.Status(),
		Duration:   duration,
		Size:       c.Writer.Size(),
		Timestamp:  time.Now(),
	}

	// Add error information
	if c.Writer.Status() >= 400 {
		if errorMsg, exists := c.Get("error_message"); exists {
			responseLog.Error = errorMsg.(string)
		}
	}

	// Log response headers in debug mode
	if l.logger.Level >= logrus.DebugLevel {
		responseLog.Headers = make(map[string]string)
		for name, values := range c.Writer.Header() {
			if len(values) > 0 {
				responseLog.Headers[name] = values[0]
			}
		}
	}

	l.logger.WithFields(logrus.Fields{
		"type":    "response",
		"details": responseLog,
	}).Info("HTTP Response")
}

func (l *LoggingMiddleware) logAuditTrail(c *gin.Context, requestID string, start time.Time, duration time.Duration) {
	// This method was moved to AuditLogger middleware
}

func (l *LoggingMiddleware) determineAction(method, path string) string {
	if strings.Contains(path, "/deposit") {
		return "deposit"
	}
	if strings.Contains(path, "/withdraw") {
		return "withdraw"
	}
	if strings.Contains(path, "/lock") {
		return "lock_funds"
	}
	if strings.Contains(path, "/release") {
		return "release_funds"
	}
	if strings.Contains(path, "/reconcile") {
		return "reconcile"
	}

	switch method {
	case "GET":
		return "view"
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return "unknown"
	}
}

func (l *LoggingMiddleware) determineResource(path string) string {
	if strings.Contains(path, "/wallet") {
		return "wallet"
	}
	if strings.Contains(path, "/transaction") {
		return "transaction"
	}
	if strings.Contains(path, "/admin") {
		return "admin"
	}
	if strings.Contains(path, "/audit") {
		return "audit"
	}
	return "unknown"
}

func (l *LoggingMiddleware) isImportantOperation(path string) bool {
	importantOps := []string{
		"/deposit",
		"/withdraw",
		"/lock",
		"/release",
		"/execute",
		"/reconcile",
		"/adjust",
	}

	for _, op := range importantOps {
		if strings.Contains(path, op) {
			return true
		}
	}
	return false
}

func (l *LoggingMiddleware) extractRequestData(c *gin.Context) map[string]interface{} {
	data := make(map[string]interface{})

	// Add path parameters
	for _, param := range c.Params {
		if param.Key != "userId" { // Don't log sensitive user IDs directly
			data[param.Key] = param.Value
		}
	}

	// Add relevant query parameters
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 && !l.isSensitiveParam(key) {
			data[key] = values[0]
		}
	}

	return data
}

func (l *LoggingMiddleware) isSensitiveParam(param string) bool {
	sensitiveParams := []string{"token", "key", "secret", "password"}
	lowerParam := strings.ToLower(param)

	for _, sensitive := range sensitiveParams {
		if strings.Contains(lowerParam, sensitive) {
			return true
		}
	}
	return false
}

func (l *LoggingMiddleware) generateComplianceFlags(c *gin.Context) []string {
	var flags []string

	// Large transaction flag
	if strings.Contains(c.Request.URL.Path, "/deposit") || strings.Contains(c.Request.URL.Path, "/withdraw") {
		flags = append(flags, "FINANCIAL_TRANSACTION")
	}

	// Admin action flag
	if strings.Contains(c.Request.URL.Path, "/admin") {
		flags = append(flags, "ADMIN_ACTION")
	}

	// High-risk time flag (outside business hours)
	hour := time.Now().Hour()
	if hour < 6 || hour > 22 {
		flags = append(flags, "OFF_HOURS_ACCESS")
	}

	return flags
}

func (l *LoggingMiddleware) calculateRiskScore(c *gin.Context, auditLog *AuditLog) int {
	score := 1

	// Admin actions have higher risk
	if auditLog.AdminID != "" {
		score += 3
	}

	// Financial transactions have higher risk
	if strings.Contains(auditLog.Action, "deposit") || strings.Contains(auditLog.Action, "withdraw") {
		score += 2
	}

	// Failed requests have higher risk
	if !auditLog.Success {
		score += 2
	}

	// Off-hours access
	hour := time.Now().Hour()
	if hour < 6 || hour > 22 {
		score += 1
	}

	// Slow requests might indicate problems
	if auditLog.Duration > l.config.SlowRequestThreshold {
		score += 1
	}

	// Cap at 10
	if score > 10 {
		score = 10
	}

	return score
}

func (l *LoggingMiddleware) isHealthCheckPath(path string) bool {
	healthPaths := []string{"/health", "/ready", "/ping", "/status"}
	for _, healthPath := range healthPaths {
		if path == healthPath {
			return true
		}
	}
	return false
}