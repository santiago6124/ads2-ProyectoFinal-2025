package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type LoggingMiddleware struct {
	logger *logrus.Logger
	config *LoggingConfig
}

type LoggingConfig struct {
	SkipPaths    []string
	LogBody      bool
	LogHeaders   bool
	MaxBodySize  int64
	TimestampFormat string
}

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if w.body != nil {
		w.body.Write(data)
	}
	return w.ResponseWriter.Write(data)
}

func NewLoggingMiddleware(logger *logrus.Logger, config *LoggingConfig) *LoggingMiddleware {
	if config == nil {
		config = DefaultLoggingConfig()
	}

	return &LoggingMiddleware{
		logger: logger,
		config: config,
	}
}

func (l *LoggingMiddleware) LogRequests() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for certain paths
		for _, skipPath := range l.config.SkipPaths {
			if c.Request.URL.Path == skipPath {
				c.Next()
				return
			}
		}

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Read request body if logging is enabled
		var requestBody []byte
		if l.config.LogBody && c.Request.Body != nil {
			var err error
			requestBody, err = io.ReadAll(io.LimitReader(c.Request.Body, l.config.MaxBodySize))
			if err == nil {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}
		}

		// Wrap response writer to capture response body
		var responseBody *bytes.Buffer
		if l.config.LogBody {
			responseBody = &bytes.Buffer{}
			c.Writer = &responseWriter{
				ResponseWriter: c.Writer,
				body:           responseBody,
			}
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		bodySize := c.Writer.Size()
		userAgent := c.Request.UserAgent()

		if raw != "" {
			path = path + "?" + raw
		}

		// Create log entry
		entry := l.logger.WithFields(logrus.Fields{
			"status_code":   statusCode,
			"latency":       latency,
			"client_ip":     clientIP,
			"method":        method,
			"path":          path,
			"body_size":     bodySize,
			"user_agent":    userAgent,
			"timestamp":     start.Format(l.config.TimestampFormat),
		})

		// Add user information if available
		if userID, exists := c.Get("user_id"); exists {
			entry = entry.WithField("user_id", userID)
		}
		if userEmail, exists := c.Get("user_email"); exists {
			entry = entry.WithField("user_email", userEmail)
		}

		// Add request/response bodies if enabled
		if l.config.LogBody {
			if len(requestBody) > 0 {
				entry = entry.WithField("request_body", string(requestBody))
			}
			if responseBody != nil && responseBody.Len() > 0 {
				entry = entry.WithField("response_body", responseBody.String())
			}
		}

		// Add headers if enabled
		if l.config.LogHeaders {
			entry = entry.WithField("request_headers", c.Request.Header)
			entry = entry.WithField("response_headers", c.Writer.Header())
		}

		// Add error information if present
		if len(c.Errors) > 0 {
			entry = entry.WithField("errors", c.Errors.Errors())
		}

		// Log with appropriate level based on status code
		switch {
		case statusCode >= 500:
			entry.Error("Server error")
		case statusCode >= 400:
			entry.Warn("Client error")
		case statusCode >= 300:
			entry.Info("Redirect")
		default:
			entry.Info("Request completed")
		}
	}
}

func (l *LoggingMiddleware) LogErrors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Log any errors that occurred during request processing
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				entry := l.logger.WithFields(logrus.Fields{
					"method":     c.Request.Method,
					"path":       c.Request.URL.Path,
					"client_ip":  c.ClientIP(),
					"user_agent": c.Request.UserAgent(),
					"error_type": err.Type,
					"timestamp":  time.Now().Format(l.config.TimestampFormat),
				})

				// Add user information if available
				if userID, exists := c.Get("user_id"); exists {
					entry = entry.WithField("user_id", userID)
				}

				switch err.Type {
				case gin.ErrorTypeBind:
					entry.Warn("Request binding error: " + err.Error())
				case gin.ErrorTypePublic:
					entry.Info("Public error: " + err.Error())
				case gin.ErrorTypePrivate:
					entry.Error("Private error: " + err.Error())
				default:
					entry.Error("Request error: " + err.Error())
				}
			}
		}
	}
}

func (l *LoggingMiddleware) LogPanic() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		entry := l.logger.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"panic":      recovered,
			"timestamp":  time.Now().Format(l.config.TimestampFormat),
		})

		// Add user information if available
		if userID, exists := c.Get("user_id"); exists {
			entry = entry.WithField("user_id", userID)
		}

		entry.Error("Panic recovered")

		c.AbortWithStatus(500)
	})
}

func (l *LoggingMiddleware) StructuredLogging() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add request ID for tracing
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Add structured logging context
		logger := l.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"client_ip":  c.ClientIP(),
		})

		// Store logger in context for use in handlers
		c.Set("logger", logger)

		c.Next()
	}
}

func (l *LoggingMiddleware) LogSlowRequests(threshold time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		if latency > threshold {
			entry := l.logger.WithFields(logrus.Fields{
				"method":       c.Request.Method,
				"path":         c.Request.URL.Path,
				"latency":      latency,
				"threshold":    threshold,
				"status_code":  c.Writer.Status(),
				"client_ip":    c.ClientIP(),
				"timestamp":    start.Format(l.config.TimestampFormat),
			})

			if userID, exists := c.Get("user_id"); exists {
				entry = entry.WithField("user_id", userID)
			}

			if requestID, exists := c.Get("request_id"); exists {
				entry = entry.WithField("request_id", requestID)
			}

			entry.Warn("Slow request detected")
		}
	}
}

func generateRequestID() string {
	// Simple implementation - in production, use a more sophisticated approach
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		SkipPaths: []string{
			"/health",
			"/health/live",
			"/health/ready",
			"/metrics",
		},
		LogBody:         false, // Enable only for debugging
		LogHeaders:      false, // Enable only for debugging
		MaxBodySize:     1024,  // 1KB
		TimestampFormat: time.RFC3339,
	}
}