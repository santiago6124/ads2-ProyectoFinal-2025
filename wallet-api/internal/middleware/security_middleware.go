package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SecurityMiddleware struct {
	webhookSecret string
	trustedProxies []string
	config        *SecurityConfig
}

type SecurityConfig struct {
	EnableCSRF              bool
	EnableCORS              bool
	EnableRequestID         bool
	EnableSecurityHeaders   bool
	EnableWebhookValidation bool
	EnableInputSanitization bool
	MaxRequestSize          int64 // in bytes
	AllowedOrigins          []string
	AllowedMethods          []string
	AllowedHeaders          []string
	CSRFTokenLength         int
	RequestTimeout          time.Duration
}

func NewSecurityMiddleware(webhookSecret string, config *SecurityConfig) *SecurityMiddleware {
	if config == nil {
		config = &SecurityConfig{
			EnableCSRF:              true,
			EnableCORS:              true,
			EnableRequestID:         true,
			EnableSecurityHeaders:   true,
			EnableWebhookValidation: true,
			EnableInputSanitization: true,
			MaxRequestSize:          10 * 1024 * 1024, // 10MB
			AllowedOrigins:          []string{"https://cryptosim.com", "https://app.cryptosim.com"},
			AllowedMethods:          []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:          []string{"Authorization", "Content-Type", "X-Requested-With", "X-API-Key", "X-Session-ID"},
			CSRFTokenLength:         32,
			RequestTimeout:          30 * time.Second,
		}
	}

	return &SecurityMiddleware{
		webhookSecret: webhookSecret,
		config:        config,
	}
}

// SecurityHeaders adds security headers to responses
func (s *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.config.EnableSecurityHeaders {
			c.Next()
			return
		}

		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';")

		// HSTS header for HTTPS
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Cache control for sensitive endpoints
		if s.isSensitiveEndpoint(c.Request.URL.Path) {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	}
}

// CORS handles Cross-Origin Resource Sharing
func (s *SecurityMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.config.EnableCORS {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")

		// Check if origin is allowed
		if s.isAllowedOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", strings.Join(s.config.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(s.config.AllowedHeaders, ", "))
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequestID adds unique request ID to each request
func (s *SecurityMiddleware) RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.config.EnableRequestID {
			c.Next()
			return
		}

		// Check if request ID already exists (from load balancer, etc.)
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// InputSanitization sanitizes and validates input data
func (s *SecurityMiddleware) InputSanitization() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.config.EnableInputSanitization {
			c.Next()
			return
		}

		// Check content length
		if c.Request.ContentLength > s.config.MaxRequestSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "Request too large",
				"message": fmt.Sprintf("Request size exceeds maximum allowed (%d bytes)", s.config.MaxRequestSize),
			})
			c.Abort()
			return
		}

		// Validate content type for POST/PUT requests
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			if !s.isValidContentType(contentType) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid content type",
					"message": "Content-Type must be application/json",
				})
				c.Abort()
				return
			}
		}

		// Validate and sanitize query parameters
		if err := s.sanitizeQueryParams(c); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid query parameters",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Validate headers for security threats
		if err := s.validateHeaders(c); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid headers",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// WebhookSignatureValidation validates webhook signatures
func (s *SecurityMiddleware) WebhookSignatureValidation() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.config.EnableWebhookValidation {
			c.Next()
			return
		}

		// Only validate webhooks (specific paths)
		if !s.isWebhookEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		signature := c.GetHeader("X-Webhook-Signature")
		if signature == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Missing webhook signature",
				"message": "X-Webhook-Signature header is required",
			})
			c.Abort()
			return
		}

		// Read body for signature verification
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Failed to read request body",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Restore body for next handlers
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// Verify signature
		if !s.verifyWebhookSignature(body, signature) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid webhook signature",
				"message": "Webhook signature verification failed",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequestTimeout sets timeout for requests
func (s *SecurityMiddleware) RequestTimeout() gin.HandlerFunc {
	return gin.TimeoutWithHandler(s.config.RequestTimeout, func(c *gin.Context) {
		c.JSON(http.StatusRequestTimeout, gin.H{
			"error":   "Request timeout",
			"message": "Request took too long to process",
		})
	})
}

// IPWhitelist restricts access to specific IPs for sensitive endpoints
func (s *SecurityMiddleware) IPWhitelist(allowedIPs []string) gin.HandlerFunc {
	allowedIPMap := make(map[string]bool)
	for _, ip := range allowedIPs {
		allowedIPMap[ip] = true
	}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !allowedIPMap[clientIP] {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "IP not allowed",
				"message": "Your IP address is not authorized to access this endpoint",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SQLInjectionProtection detects potential SQL injection attempts
func (s *SecurityMiddleware) SQLInjectionProtection() gin.HandlerFunc {
	sqlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union\s+select|select\s+.*\s+from|insert\s+into|update\s+.*\s+set|delete\s+from)`),
		regexp.MustCompile(`(?i)(exec\s*\(|execute\s*\(|sp_|xp_)`),
		regexp.MustCompile(`(?i)(\'\s*or\s+\'\s*\=\s*\'|\'\s*or\s+1\s*\=\s*1|--|\#|\/*|\*/)`),
		regexp.MustCompile(`(?i)(concat\s*\(|char\s*\(|ascii\s*\(|substring\s*\()`),
	}

	return func(c *gin.Context) {
		// Check query parameters
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if s.containsSQLInjection(value, sqlPatterns) {
					s.logSecurityThreat(c, "sql_injection", fmt.Sprintf("Query param %s: %s", key, value))
					c.JSON(http.StatusBadRequest, gin.H{
						"error":   "Invalid input detected",
						"message": "Request contains potentially malicious content",
					})
					c.Abort()
					return
				}
			}
		}

		// Check path parameters
		for _, param := range c.Params {
			if s.containsSQLInjection(param.Value, sqlPatterns) {
				s.logSecurityThreat(c, "sql_injection", fmt.Sprintf("Path param %s: %s", param.Key, param.Value))
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid input detected",
					"message": "Request contains potentially malicious content",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// XSSProtection detects potential XSS attempts
func (s *SecurityMiddleware) XSSProtection() gin.HandlerFunc {
	xssPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
		regexp.MustCompile(`(?i)javascript\s*:`),
		regexp.MustCompile(`(?i)on\w+\s*=`),
		regexp.MustCompile(`(?i)<img[^>]+src[^>]*=.*?>`),
	}

	return func(c *gin.Context) {
		// Check query parameters
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if s.containsXSS(value, xssPatterns) {
					s.logSecurityThreat(c, "xss_attempt", fmt.Sprintf("Query param %s: %s", key, value))
					c.JSON(http.StatusBadRequest, gin.H{
						"error":   "Invalid input detected",
						"message": "Request contains potentially malicious content",
					})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}

// Helper methods
func (s *SecurityMiddleware) isSensitiveEndpoint(path string) bool {
	sensitivePatterns := []string{
		"/api/wallet/",
		"/api/admin/",
		"/auth/",
		"/login",
		"/token",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

func (s *SecurityMiddleware) isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	for _, allowed := range s.config.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}

	return false
}

func (s *SecurityMiddleware) isValidContentType(contentType string) bool {
	validTypes := []string{
		"application/json",
		"application/json; charset=utf-8",
		"multipart/form-data",
	}

	for _, validType := range validTypes {
		if strings.HasPrefix(contentType, validType) {
			return true
		}
	}

	return false
}

func (s *SecurityMiddleware) sanitizeQueryParams(c *gin.Context) error {
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			// Check for excessively long parameters
			if len(value) > 1000 {
				return fmt.Errorf("query parameter %s exceeds maximum length", key)
			}

			// Check for null bytes
			if strings.Contains(value, "\x00") {
				return fmt.Errorf("query parameter %s contains null bytes", key)
			}

			// Validate specific parameter types
			if err := s.validateSpecificParam(key, value); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SecurityMiddleware) validateSpecificParam(key, value string) error {
	switch key {
	case "userId", "user_id":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return fmt.Errorf("invalid user ID format")
		}
	case "limit":
		if limit, err := strconv.Atoi(value); err != nil || limit < 0 || limit > 1000 {
			return fmt.Errorf("invalid limit value (must be 0-1000)")
		}
	case "offset":
		if offset, err := strconv.Atoi(value); err != nil || offset < 0 {
			return fmt.Errorf("invalid offset value (must be >= 0)")
		}
	}

	return nil
}

func (s *SecurityMiddleware) validateHeaders(c *gin.Context) error {
	// Check for suspicious user agents
	userAgent := c.GetHeader("User-Agent")
	if userAgent == "" {
		return fmt.Errorf("missing User-Agent header")
	}

	// Check for excessively long headers
	for name, values := range c.Request.Header {
		for _, value := range values {
			if len(value) > 4096 {
				return fmt.Errorf("header %s exceeds maximum length", name)
			}
		}
	}

	return nil
}

func (s *SecurityMiddleware) isWebhookEndpoint(path string) bool {
	webhookPaths := []string{
		"/webhook/",
		"/callbacks/",
		"/notify/",
	}

	for _, webhookPath := range webhookPaths {
		if strings.Contains(path, webhookPath) {
			return true
		}
	}

	return false
}

func (s *SecurityMiddleware) verifyWebhookSignature(body []byte, signature string) bool {
	// Remove 'sha256=' prefix if present
	signature = strings.TrimPrefix(signature, "sha256=")

	// Calculate HMAC
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (s *SecurityMiddleware) containsSQLInjection(input string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

func (s *SecurityMiddleware) containsXSS(input string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

func (s *SecurityMiddleware) logSecurityThreat(c *gin.Context, threatType, details string) {
	// In a real implementation, this would send to security monitoring system
	fmt.Printf("SECURITY THREAT DETECTED: Type: %s, IP: %s, Path: %s, Details: %s, Time: %s\n",
		threatType,
		c.ClientIP(),
		c.Request.URL.Path,
		details,
		time.Now().Format(time.RFC3339),
	)
}

// HTTPSRedirect redirects HTTP requests to HTTPS
func (s *SecurityMiddleware) HTTPSRedirect() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Header.Get("X-Forwarded-Proto") == "http" {
			httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
			c.Redirect(http.StatusMovedPermanently, httpsURL)
			c.Abort()
			return
		}
		c.Next()
	}
}

// CSRFProtection protects against Cross-Site Request Forgery
func (s *SecurityMiddleware) CSRFProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.config.EnableCSRF {
			c.Next()
			return
		}

		// Skip CSRF for safe methods and API endpoints with proper auth
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Skip for API endpoints with proper authentication
		if c.GetHeader("Authorization") != "" || c.GetHeader("X-API-Key") != "" {
			c.Next()
			return
		}

		// Check CSRF token
		csrfToken := c.GetHeader("X-CSRF-Token")
		if csrfToken == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "CSRF token required",
				"message": "X-CSRF-Token header is required for this request",
			})
			c.Abort()
			return
		}

		// Validate CSRF token (simplified - in production use proper CSRF library)
		if !s.validateCSRFToken(csrfToken) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Invalid CSRF token",
				"message": "CSRF token validation failed",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (s *SecurityMiddleware) validateCSRFToken(token string) bool {
	// Simplified CSRF validation - in production use proper implementation
	return len(token) >= s.config.CSRFTokenLength
}