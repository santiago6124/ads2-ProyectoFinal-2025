package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Logger returns a gin middleware for logging HTTP requests
func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Custom log entry
		logger.WithFields(logrus.Fields{
			"status_code":    param.StatusCode,
			"latency":        param.Latency,
			"client_ip":      param.ClientIP,
			"method":         param.Method,
			"path":           param.Path,
			"request_id":     param.Keys["request_id"],
			"user_agent":     param.Request.UserAgent(),
			"response_size":  param.BodySize,
			"timestamp":      param.TimeStamp.Format(time.RFC3339),
		}).Info("HTTP request processed")

		return ""
	})
}

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := generateRequestID()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// generateRequestID creates a simple request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}