package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type RateLimitMiddleware struct {
	redisClient *redis.Client
	config      *RateLimitConfig
}

type RateLimitConfig struct {
	// Global limits
	GlobalRPM int // Requests per minute globally
	GlobalRPH int // Requests per hour globally

	// Per-IP limits
	IPRequestsPerMinute int
	IPRequestsPerHour   int

	// Per-user limits
	UserRequestsPerMinute int
	UserRequestsPerHour   int

	// Transaction-specific limits
	TransactionRequestsPerMinute int
	TransactionRequestsPerHour   int

	// Admin limits (higher)
	AdminRequestsPerMinute int
	AdminRequestsPerHour   int

	// Burst allowance
	BurstAllowance int

	// Whitelist IPs (no limits)
	WhitelistIPs map[string]bool

	// Enable/disable features
	EnableIPLimiting   bool
	EnableUserLimiting bool
	EnableBurstControl bool
}

func NewRateLimitMiddleware(redisClient *redis.Client, config *RateLimitConfig) *RateLimitMiddleware {
	if config == nil {
		config = &RateLimitConfig{
			GlobalRPM:                    10000,
			GlobalRPH:                    100000,
			IPRequestsPerMinute:         100,
			IPRequestsPerHour:           1000,
			UserRequestsPerMinute:       60,
			UserRequestsPerHour:         500,
			TransactionRequestsPerMinute: 20,
			TransactionRequestsPerHour:   100,
			AdminRequestsPerMinute:      200,
			AdminRequestsPerHour:        2000,
			BurstAllowance:              10,
			EnableIPLimiting:            true,
			EnableUserLimiting:          true,
			EnableBurstControl:          true,
			WhitelistIPs:                make(map[string]bool),
		}
	}

	return &RateLimitMiddleware{
		redisClient: redisClient,
		config:      config,
	}
}

type RateLimitInfo struct {
	Limit     int           `json:"limit"`
	Remaining int           `json:"remaining"`
	ResetTime time.Time     `json:"reset_time"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

// GlobalRateLimit applies global rate limiting
func (r *RateLimitMiddleware) GlobalRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Check global rate limits
		allowed, info, err := r.checkGlobalLimit(ctx)
		if err != nil {
			// Log error but don't block request
			c.Header("X-RateLimit-Error", err.Error())
			c.Next()
			return
		}

		// Set rate limit headers
		r.setRateLimitHeaders(c, info)

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     "Global rate limit exceeded. Please try again later.",
				"retry_after": int(info.RetryAfter.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPRateLimit applies per-IP rate limiting
func (r *RateLimitMiddleware) IPRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.config.EnableIPLimiting {
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		// Check if IP is whitelisted
		if r.config.WhitelistIPs[clientIP] {
			c.Next()
			return
		}

		ctx := c.Request.Context()

		// Check IP rate limits
		allowed, info, err := r.checkIPLimit(ctx, clientIP)
		if err != nil {
			c.Header("X-RateLimit-Error", err.Error())
			c.Next()
			return
		}

		// Set rate limit headers
		r.setRateLimitHeaders(c, info)

		if !allowed {
			// Log suspicious activity for very high request rates
			if info.Remaining < -50 {
				go r.logSuspiciousActivity(clientIP, "excessive_requests")
			}

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "IP rate limit exceeded",
				"message":     "Too many requests from this IP. Please try again later.",
				"retry_after": int(info.RetryAfter.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// UserRateLimit applies per-user rate limiting
func (r *RateLimitMiddleware) UserRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.config.EnableUserLimiting {
			c.Next()
			return
		}

		// Skip if no user context (unauthenticated requests)
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		// Check if admin (higher limits)
		isAdmin, _ := c.Get("is_admin")
		isAdminUser := isAdmin != nil && isAdmin.(bool)

		ctx := c.Request.Context()
		userIDStr := fmt.Sprintf("%d", userID.(int64))

		// Check user rate limits
		allowed, info, err := r.checkUserLimit(ctx, userIDStr, isAdminUser)
		if err != nil {
			c.Header("X-RateLimit-Error", err.Error())
			c.Next()
			return
		}

		// Set rate limit headers
		r.setRateLimitHeaders(c, info)

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "User rate limit exceeded",
				"message":     "Too many requests for this user. Please try again later.",
				"retry_after": int(info.RetryAfter.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// TransactionRateLimit applies special limits for transaction endpoints
func (r *RateLimitMiddleware) TransactionRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to transaction endpoints
		if !r.isTransactionEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Skip for internal services
		if isInternal, exists := c.Get("is_internal"); exists && isInternal.(bool) {
			c.Next()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		userIDStr := fmt.Sprintf("%d", userID.(int64))

		// Check transaction-specific rate limits
		allowed, info, err := r.checkTransactionLimit(ctx, userIDStr)
		if err != nil {
			c.Header("X-RateLimit-Error", err.Error())
			c.Next()
			return
		}

		// Set rate limit headers
		r.setRateLimitHeaders(c, info)

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Transaction rate limit exceeded",
				"message":     "Too many transaction requests. Please try again later.",
				"retry_after": int(info.RetryAfter.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// BurstProtection provides burst protection using sliding window
func (r *RateLimitMiddleware) BurstProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.config.EnableBurstControl {
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		// Skip whitelisted IPs
		if r.config.WhitelistIPs[clientIP] {
			c.Next()
			return
		}

		ctx := c.Request.Context()

		// Check for burst patterns
		allowed, err := r.checkBurstLimit(ctx, clientIP)
		if err != nil {
			c.Header("X-RateLimit-Error", err.Error())
			c.Next()
			return
		}

		if !allowed {
			// Log potential attack
			go r.logSuspiciousActivity(clientIP, "burst_attack")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Burst limit exceeded",
				"message": "Request rate too high. Please slow down.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Implementation methods
func (r *RateLimitMiddleware) checkGlobalLimit(ctx context.Context) (bool, *RateLimitInfo, error) {
	now := time.Now()
	minuteKey := fmt.Sprintf("global:rpm:%d", now.Unix()/60)
	hourKey := fmt.Sprintf("global:rph:%d", now.Unix()/3600)

	// Check minute limit
	minuteCount, err := r.incrementAndGet(ctx, minuteKey, time.Minute)
	if err != nil {
		return true, nil, err
	}

	if minuteCount > r.config.GlobalRPM {
		return false, &RateLimitInfo{
			Limit:      r.config.GlobalRPM,
			Remaining:  r.config.GlobalRPM - minuteCount,
			ResetTime:  now.Truncate(time.Minute).Add(time.Minute),
			RetryAfter: time.Until(now.Truncate(time.Minute).Add(time.Minute)),
		}, nil
	}

	// Check hour limit
	hourCount, err := r.incrementAndGet(ctx, hourKey, time.Hour)
	if err != nil {
		return true, nil, err
	}

	if hourCount > r.config.GlobalRPH {
		return false, &RateLimitInfo{
			Limit:      r.config.GlobalRPH,
			Remaining:  r.config.GlobalRPH - hourCount,
			ResetTime:  now.Truncate(time.Hour).Add(time.Hour),
			RetryAfter: time.Until(now.Truncate(time.Hour).Add(time.Hour)),
		}, nil
	}

	return true, &RateLimitInfo{
		Limit:     r.config.GlobalRPM,
		Remaining: r.config.GlobalRPM - minuteCount,
		ResetTime: now.Truncate(time.Minute).Add(time.Minute),
	}, nil
}

func (r *RateLimitMiddleware) checkIPLimit(ctx context.Context, ip string) (bool, *RateLimitInfo, error) {
	now := time.Now()
	minuteKey := fmt.Sprintf("ip:%s:rpm:%d", ip, now.Unix()/60)
	hourKey := fmt.Sprintf("ip:%s:rph:%d", ip, now.Unix()/3600)

	// Check minute limit
	minuteCount, err := r.incrementAndGet(ctx, minuteKey, time.Minute)
	if err != nil {
		return true, nil, err
	}

	if minuteCount > r.config.IPRequestsPerMinute {
		return false, &RateLimitInfo{
			Limit:      r.config.IPRequestsPerMinute,
			Remaining:  r.config.IPRequestsPerMinute - minuteCount,
			ResetTime:  now.Truncate(time.Minute).Add(time.Minute),
			RetryAfter: time.Until(now.Truncate(time.Minute).Add(time.Minute)),
		}, nil
	}

	// Check hour limit
	hourCount, err := r.incrementAndGet(ctx, hourKey, time.Hour)
	if err != nil {
		return true, nil, err
	}

	if hourCount > r.config.IPRequestsPerHour {
		return false, &RateLimitInfo{
			Limit:      r.config.IPRequestsPerHour,
			Remaining:  r.config.IPRequestsPerHour - hourCount,
			ResetTime:  now.Truncate(time.Hour).Add(time.Hour),
			RetryAfter: time.Until(now.Truncate(time.Hour).Add(time.Hour)),
		}, nil
	}

	return true, &RateLimitInfo{
		Limit:     r.config.IPRequestsPerMinute,
		Remaining: r.config.IPRequestsPerMinute - minuteCount,
		ResetTime: now.Truncate(time.Minute).Add(time.Minute),
	}, nil
}

func (r *RateLimitMiddleware) checkUserLimit(ctx context.Context, userID string, isAdmin bool) (bool, *RateLimitInfo, error) {
	now := time.Now()

	// Use different limits for admin users
	minuteLimit := r.config.UserRequestsPerMinute
	hourLimit := r.config.UserRequestsPerHour

	if isAdmin {
		minuteLimit = r.config.AdminRequestsPerMinute
		hourLimit = r.config.AdminRequestsPerHour
	}

	minuteKey := fmt.Sprintf("user:%s:rpm:%d", userID, now.Unix()/60)
	hourKey := fmt.Sprintf("user:%s:rph:%d", userID, now.Unix()/3600)

	// Check minute limit
	minuteCount, err := r.incrementAndGet(ctx, minuteKey, time.Minute)
	if err != nil {
		return true, nil, err
	}

	if minuteCount > minuteLimit {
		return false, &RateLimitInfo{
			Limit:      minuteLimit,
			Remaining:  minuteLimit - minuteCount,
			ResetTime:  now.Truncate(time.Minute).Add(time.Minute),
			RetryAfter: time.Until(now.Truncate(time.Minute).Add(time.Minute)),
		}, nil
	}

	// Check hour limit
	hourCount, err := r.incrementAndGet(ctx, hourKey, time.Hour)
	if err != nil {
		return true, nil, err
	}

	if hourCount > hourLimit {
		return false, &RateLimitInfo{
			Limit:      hourLimit,
			Remaining:  hourLimit - hourCount,
			ResetTime:  now.Truncate(time.Hour).Add(time.Hour),
			RetryAfter: time.Until(now.Truncate(time.Hour).Add(time.Hour)),
		}, nil
	}

	return true, &RateLimitInfo{
		Limit:     minuteLimit,
		Remaining: minuteLimit - minuteCount,
		ResetTime: now.Truncate(time.Minute).Add(time.Minute),
	}, nil
}

func (r *RateLimitMiddleware) checkTransactionLimit(ctx context.Context, userID string) (bool, *RateLimitInfo, error) {
	now := time.Now()
	minuteKey := fmt.Sprintf("user:%s:tx:rpm:%d", userID, now.Unix()/60)
	hourKey := fmt.Sprintf("user:%s:tx:rph:%d", userID, now.Unix()/3600)

	// Check minute limit
	minuteCount, err := r.incrementAndGet(ctx, minuteKey, time.Minute)
	if err != nil {
		return true, nil, err
	}

	if minuteCount > r.config.TransactionRequestsPerMinute {
		return false, &RateLimitInfo{
			Limit:      r.config.TransactionRequestsPerMinute,
			Remaining:  r.config.TransactionRequestsPerMinute - minuteCount,
			ResetTime:  now.Truncate(time.Minute).Add(time.Minute),
			RetryAfter: time.Until(now.Truncate(time.Minute).Add(time.Minute)),
		}, nil
	}

	// Check hour limit
	hourCount, err := r.incrementAndGet(ctx, hourKey, time.Hour)
	if err != nil {
		return true, nil, err
	}

	if hourCount > r.config.TransactionRequestsPerHour {
		return false, &RateLimitInfo{
			Limit:      r.config.TransactionRequestsPerHour,
			Remaining:  r.config.TransactionRequestsPerHour - hourCount,
			ResetTime:  now.Truncate(time.Hour).Add(time.Hour),
			RetryAfter: time.Until(now.Truncate(time.Hour).Add(time.Hour)),
		}, nil
	}

	return true, &RateLimitInfo{
		Limit:     r.config.TransactionRequestsPerMinute,
		Remaining: r.config.TransactionRequestsPerMinute - minuteCount,
		ResetTime: now.Truncate(time.Minute).Add(time.Minute),
	}, nil
}

func (r *RateLimitMiddleware) checkBurstLimit(ctx context.Context, ip string) (bool, error) {
	now := time.Now()
	burstKey := fmt.Sprintf("burst:%s", ip)

	// Use sliding window of last 10 seconds
	windowStart := now.Add(-10 * time.Second).Unix()
	_ = windowStart // Used for cleanup but not in current implementation

	// Add current request timestamp
	pipe := r.redisClient.Pipeline()
	pipe.ZAdd(ctx, burstKey, &redis.Z{
		Score:  float64(now.Unix()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// Remove old entries
	pipe.ZRemRangeByScore(ctx, burstKey, "0", fmt.Sprintf("%d", windowStart))

	// Count requests in window
	pipe.ZCard(ctx, burstKey)

	// Set expiration
	pipe.Expire(ctx, burstKey, 10*time.Second)

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return true, err
	}

	// Get count from ZCard result
	countCmd := cmds[2].(*redis.IntCmd)
	count, err := countCmd.Result()
	if err != nil {
		return true, err
	}

	// Allow burst up to configured limit
	return count <= int64(r.config.BurstAllowance), nil
}

func (r *RateLimitMiddleware) incrementAndGet(ctx context.Context, key string, expiration time.Duration) (int, error) {
	pipe := r.redisClient.Pipeline()

	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, expiration)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return int(incr.Val()), nil
}

func (r *RateLimitMiddleware) setRateLimitHeaders(c *gin.Context, info *RateLimitInfo) {
	if info == nil {
		return
	}

	c.Header("X-RateLimit-Limit", strconv.Itoa(info.Limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(info.ResetTime.Unix(), 10))

	if info.RetryAfter > 0 {
		c.Header("Retry-After", strconv.Itoa(int(info.RetryAfter.Seconds())))
	}
}

func (r *RateLimitMiddleware) isTransactionEndpoint(path string) bool {
	transactionPaths := []string{
		"/api/wallet/",
	}

	transactionOperations := []string{
		"/deposit",
		"/withdraw",
		"/lock",
		"/release",
		"/execute",
	}

	// Check if path contains transaction operations
	for _, txPath := range transactionPaths {
		if len(path) > len(txPath) && path[:len(txPath)] == txPath {
			for _, op := range transactionOperations {
				if len(path) >= len(op) && path[len(path)-len(op):] == op {
					return true
				}
			}
		}
	}

	return false
}

func (r *RateLimitMiddleware) logSuspiciousActivity(ip, activityType string) {
	// In a real implementation, this would log to security monitoring system
	// For now, we'll just log to standard output
	fmt.Printf("SECURITY ALERT: Suspicious activity detected - IP: %s, Type: %s, Time: %s\n",
		ip, activityType, time.Now().Format(time.RFC3339))
}

// WhitelistIP adds an IP to the whitelist
func (r *RateLimitMiddleware) WhitelistIP(ip string) {
	r.config.WhitelistIPs[ip] = true
}

// RemoveWhitelistIP removes an IP from the whitelist
func (r *RateLimitMiddleware) RemoveWhitelistIP(ip string) {
	delete(r.config.WhitelistIPs, ip)
}

// GetRateLimitStatus returns current rate limit status for debugging
func (r *RateLimitMiddleware) GetRateLimitStatus(ctx context.Context, identifier, limitType string) (*RateLimitInfo, error) {
	now := time.Now()
	var key string
	var limit int

	switch limitType {
	case "ip_minute":
		key = fmt.Sprintf("ip:%s:rpm:%d", identifier, now.Unix()/60)
		limit = r.config.IPRequestsPerMinute
	case "ip_hour":
		key = fmt.Sprintf("ip:%s:rph:%d", identifier, now.Unix()/3600)
		limit = r.config.IPRequestsPerHour
	case "user_minute":
		key = fmt.Sprintf("user:%s:rpm:%d", identifier, now.Unix()/60)
		limit = r.config.UserRequestsPerMinute
	case "user_hour":
		key = fmt.Sprintf("user:%s:rph:%d", identifier, now.Unix()/3600)
		limit = r.config.UserRequestsPerHour
	default:
		return nil, fmt.Errorf("unknown limit type: %s", limitType)
	}

	count, err := r.redisClient.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			count = 0
		} else {
			return nil, err
		}
	}

	return &RateLimitInfo{
		Limit:     limit,
		Remaining: limit - count,
		ResetTime: now.Truncate(time.Minute).Add(time.Minute),
	}, nil
}