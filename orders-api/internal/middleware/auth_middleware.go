package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

type AuthMiddleware struct {
	secretKey       []byte
	issuer          string
	audience        string
	skipPaths       map[string]bool
	publicEndpoints map[string]bool
}

type Claims struct {
	UserID   int    `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type AuthConfig struct {
	SecretKey       string
	Issuer          string
	Audience        string
	SkipPaths       []string
	PublicEndpoints []string
}

func NewAuthMiddleware(config *AuthConfig) *AuthMiddleware {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	publicEndpoints := make(map[string]bool)
	for _, endpoint := range config.PublicEndpoints {
		publicEndpoints[endpoint] = true
	}

	return &AuthMiddleware{
		secretKey:       []byte(config.SecretKey),
		issuer:          config.Issuer,
		audience:        config.Audience,
		skipPaths:       skipPaths,
		publicEndpoints: publicEndpoints,
	}
}

func (a *AuthMiddleware) ValidateToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for certain paths
		if a.skipPaths[c.Request.URL.Path] || a.publicEndpoints[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Check for Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "missing authorization header",
				"code":    "AUTH_MISSING",
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check Bearer token format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid authorization format",
				"code":    "AUTH_INVALID_FORMAT",
				"message": "Authorization header must be in 'Bearer <token>' format",
			})
			c.Abort()
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "empty token",
				"code":    "AUTH_EMPTY_TOKEN",
				"message": "Token cannot be empty",
			})
			c.Abort()
			return
		}

		// Parse and validate token
		claims, err := a.parseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid token",
				"code":    "AUTH_INVALID_TOKEN",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_username", claims.Username)
		c.Set("user_role", claims.Role)
		c.Set("token_claims", claims)

		c.Next()
	}
}

func (a *AuthMiddleware) parseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate issuer
	if a.issuer != "" && claims.Issuer != a.issuer {
		return nil, fmt.Errorf("invalid issuer")
	}

	// Validate audience
	if a.audience != "" && !claims.VerifyAudience(a.audience, true) {
		return nil, fmt.Errorf("invalid audience")
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token has expired")
	}

	// Check not before
	if claims.NotBefore != nil && claims.NotBefore.Time.After(time.Now()) {
		return nil, fmt.Errorf("token is not yet valid")
	}

	return claims, nil
}

func (a *AuthMiddleware) RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "user role not found",
				"code":    "AUTH_ROLE_MISSING",
				"message": "User role information is missing",
			})
			c.Abort()
			return
		}

		if userRole.(string) != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient permissions",
				"code":    "AUTH_INSUFFICIENT_PERMISSIONS",
				"message": fmt.Sprintf("Required role: %s", requiredRole),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (a *AuthMiddleware) RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "user role not found",
				"code":    "AUTH_ROLE_MISSING",
				"message": "User role information is missing",
			})
			c.Abort()
			return
		}

		userRoleStr := userRole.(string)
		hasPermission := false

		for _, role := range roles {
			if userRoleStr == role {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient permissions",
				"code":    "AUTH_INSUFFICIENT_PERMISSIONS",
				"message": fmt.Sprintf("Required roles: %v", roles),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (a *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No token provided, continue without authentication
			c.Next()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			// Invalid format, continue without authentication
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			// Empty token, continue without authentication
			c.Next()
			return
		}

		// Try to parse token, but don't fail if invalid
		claims, err := a.parseToken(tokenString)
		if err == nil {
			// Valid token, set user information
			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("user_username", claims.Username)
			c.Set("user_role", claims.Role)
			c.Set("token_claims", claims)
			c.Set("authenticated", true)
		} else {
			c.Set("authenticated", false)
		}

		c.Next()
	}
}

func (a *AuthMiddleware) ExtractUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get user ID from token claims first
		if userID, exists := c.Get("user_id"); exists {
			c.Next()
			return
		}

		// Try to get user ID from URL parameter
		userIDParam := c.Param("user_id")
		if userIDParam != "" {
			userID, err := strconv.Atoi(userIDParam)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid user ID",
					"code":    "INVALID_USER_ID",
					"message": "User ID must be a valid integer",
				})
				c.Abort()
				return
			}
			c.Set("user_id", userID)
		}

		// Try to get user ID from query parameter
		userIDQuery := c.Query("user_id")
		if userIDQuery != "" {
			userID, err := strconv.Atoi(userIDQuery)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid user ID",
					"code":    "INVALID_USER_ID",
					"message": "User ID must be a valid integer",
				})
				c.Abort()
				return
			}
			c.Set("user_id", userID)
		}

		c.Next()
	}
}

func (a *AuthMiddleware) ValidateOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenUserID, tokenExists := c.Get("user_id")
		if !tokenExists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "user not authenticated",
				"code":    "AUTH_NOT_AUTHENTICATED",
				"message": "User authentication is required",
			})
			c.Abort()
			return
		}

		// Check if user is trying to access their own resources
		resourceUserIDParam := c.Param("user_id")
		if resourceUserIDParam != "" {
			resourceUserID, err := strconv.Atoi(resourceUserIDParam)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid user ID",
					"code":    "INVALID_USER_ID",
					"message": "User ID must be a valid integer",
				})
				c.Abort()
				return
			}

			if tokenUserID.(int) != resourceUserID {
				// Check if user has admin role
				if userRole, exists := c.Get("user_role"); !exists || userRole.(string) != "admin" {
					c.JSON(http.StatusForbidden, gin.H{
						"error":   "access denied",
						"code":    "AUTH_ACCESS_DENIED",
						"message": "You can only access your own resources",
					})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}

func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		SecretKey: "your-secret-key-here", // Should come from environment
		Issuer:    "orders-api",
		Audience:  "cryptosim",
		SkipPaths: []string{
			"/health",
			"/health/live",
			"/health/ready",
			"/metrics",
		},
		PublicEndpoints: []string{
			"/api/v1/public/health",
		},
	}
}