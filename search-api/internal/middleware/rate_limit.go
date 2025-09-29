package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	clients map[string]*clientBucket
	mu      sync.RWMutex
	rate    int           // requests per window
	window  time.Duration // time window
	cleanup time.Duration // cleanup interval
}

type clientBucket struct {
	tokens    int
	lastRefill time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*clientBucket),
		rate:    rate,
		window:  window,
		cleanup: 10 * time.Minute,
	}

	// Start cleanup goroutine
	go rl.cleanupRoutine()

	return rl
}

// RateLimit returns a gin middleware for rate limiting
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !rl.allowRequest(clientIP) {
			c.Header("X-RateLimit-Limit", strconv.Itoa(rl.rate))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(rl.window).Unix(), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Rate limit exceeded. Please try again later.",
					"details": gin.H{
						"limit":  rl.rate,
						"window": rl.window.String(),
					},
				},
				"success": false,
			})
			c.Abort()
			return
		}

		// Add rate limit headers
		bucket := rl.getClientBucket(clientIP)
		bucket.mu.Lock()
		remaining := bucket.tokens
		bucket.mu.Unlock()

		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.rate))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(rl.window).Unix(), 10))

		c.Next()
	}
}

// allowRequest checks if a request should be allowed
func (rl *RateLimiter) allowRequest(clientIP string) bool {
	bucket := rl.getClientBucket(clientIP)

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()

	// Refill tokens based on time elapsed
	if now.Sub(bucket.lastRefill) >= rl.window {
		bucket.tokens = rl.rate
		bucket.lastRefill = now
	}

	// Check if request can be allowed
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// getClientBucket gets or creates a bucket for a client
func (rl *RateLimiter) getClientBucket(clientIP string) *clientBucket {
	rl.mu.RLock()
	bucket, exists := rl.clients[clientIP]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	// Create new bucket
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if bucket, exists := rl.clients[clientIP]; exists {
		return bucket
	}

	bucket = &clientBucket{
		tokens:    rl.rate,
		lastRefill: time.Now(),
	}
	rl.clients[clientIP] = bucket

	return bucket
}

// cleanupRoutine removes old client buckets
func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()

		for clientIP, bucket := range rl.clients {
			bucket.mu.Lock()
			if now.Sub(bucket.lastRefill) > 2*rl.window {
				delete(rl.clients, clientIP)
			}
			bucket.mu.Unlock()
		}

		rl.mu.Unlock()
	}
}

// SearchRateLimit returns a rate limiter configured for search endpoints
func SearchRateLimit() gin.HandlerFunc {
	limiter := NewRateLimiter(100, time.Minute) // 100 requests per minute
	return limiter.RateLimit()
}

// AdminRateLimit returns a rate limiter configured for admin endpoints
func AdminRateLimit() gin.HandlerFunc {
	limiter := NewRateLimiter(20, time.Minute) // 20 requests per minute
	return limiter.RateLimit()
}