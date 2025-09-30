package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthMiddleware struct {
	jwtSecret    string
	internalKey  string
	skipPaths    map[string]bool
}

func NewAuthMiddleware(jwtSecret, internalKey string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret:   jwtSecret,
		internalKey: internalKey,
		skipPaths: map[string]bool{
			"/health":     true,
			"/ready":      true,
			"/version":    true,
			"/metrics":    true,
			"/swagger":    true,
			"/docs":       true,
		},
	}
}

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Scope    string `json:"scope"`
	jwt.RegisteredClaims
}

type AdminClaims struct {
	AdminID     string   `json:"admin_id"`
	Username    string   `json:"username"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// JWTAuth validates JWT tokens for user authentication
func (a *AuthMiddleware) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for certain paths
		if a.shouldSkipAuth(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Authorization header required",
				"message": "Missing Authorization header",
			})
			c.Abort()
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid authorization format",
				"message": "Authorization header must be 'Bearer <token>'",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(a.jwtSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid token",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			// Check token expiration
			if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Token expired",
					"message": "JWT token has expired",
				})
				c.Abort()
				return
			}

			// Set user context
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("role", claims.Role)
			c.Set("scope", claims.Scope)
			c.Set("jwt_claims", claims)

			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid token claims",
				"message": "Token contains invalid claims",
			})
			c.Abort()
			return
		}
	}
}

// InternalAPIAuth validates internal service API keys
func (a *AuthMiddleware) InternalAPIAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "API key required",
				"message": "Missing X-API-Key header",
			})
			c.Abort()
			return
		}

		if apiKey != a.internalKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid API key",
				"message": "Invalid or expired API key",
			})
			c.Abort()
			return
		}

		// Set internal service context
		c.Set("is_internal", true)
		c.Set("service_name", c.GetHeader("X-Service-Name"))
		c.Next()
	}
}

// AdminAuth validates admin JWT tokens with enhanced permissions
func (a *AuthMiddleware) AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check for internal API key (services can act as admin)
		if apiKey := c.GetHeader("X-API-Key"); apiKey == a.internalKey {
			c.Set("is_internal", true)
			c.Set("is_admin", true)
			c.Next()
			return
		}

		// Check for admin JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Authorization required",
				"message": "Admin access requires authentication",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid authorization format",
				"message": "Authorization header must be 'Bearer <token>'",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse admin token
		token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(a.jwtSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid admin token",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*AdminClaims); ok && token.Valid {
			// Check if user has admin role
			if claims.Role != "admin" && claims.Role != "super_admin" {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "Insufficient privileges",
					"message": "Admin access required",
				})
				c.Abort()
				return
			}

			// Check token expiration
			if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Token expired",
					"message": "Admin token has expired",
				})
				c.Abort()
				return
			}

			// Set admin context
			c.Set("admin_id", claims.AdminID)
			c.Set("admin_username", claims.Username)
			c.Set("admin_role", claims.Role)
			c.Set("admin_permissions", claims.Permissions)
			c.Set("is_admin", true)
			c.Set("admin_claims", claims)

			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid admin token claims",
				"message": "Token contains invalid admin claims",
			})
			c.Abort()
			return
		}
	}
}

// RequirePermission checks if admin has specific permission
func (a *AuthMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if internal service
		if isInternal, exists := c.Get("is_internal"); exists && isInternal.(bool) {
			c.Next()
			return
		}

		// Check admin permissions
		if permissions, exists := c.Get("admin_permissions"); exists {
			perms := permissions.([]string)
			hasPermission := false

			for _, perm := range perms {
				if perm == permission || perm == "*" {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "Permission denied",
					"message": fmt.Sprintf("Required permission: %s", permission),
				})
				c.Abort()
				return
			}
		} else {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "No permissions found",
				"message": "User permissions not available",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ValidateUserAccess ensures users can only access their own resources
func (a *AuthMiddleware) ValidateUserAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for internal services and admins
		if isInternal, exists := c.Get("is_internal"); exists && isInternal.(bool) {
			c.Next()
			return
		}

		if isAdmin, exists := c.Get("is_admin"); exists && isAdmin.(bool) {
			c.Next()
			return
		}

		// Get user ID from token
		tokenUserID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "User ID not found",
				"message": "User ID not available in token",
			})
			c.Abort()
			return
		}

		// Get user ID from URL parameter
		requestedUserID := c.Param("userId")
		if requestedUserID == "" {
			// If no userId in path, allow (might be creating wallet)
			c.Next()
			return
		}

		// Convert and compare user IDs
		if fmt.Sprintf("%d", tokenUserID.(int64)) != requestedUserID {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Access denied",
				"message": "Cannot access other user's resources",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// TokenRefresh handles JWT token refresh
func (a *AuthMiddleware) TokenRefresh() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"message": err.Error(),
			})
			return
		}

		// Parse refresh token
		token, err := jwt.ParseWithClaims(req.RefreshToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(a.jwtSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid refresh token",
				"message": err.Error(),
			})
			return
		}

		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			// Generate new access token
			newClaims := &Claims{
				UserID:   claims.UserID,
				Username: claims.Username,
				Role:     claims.Role,
				Scope:    claims.Scope,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					Issuer:    "wallet-api",
				},
			}

			newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
			tokenString, err := newToken.SignedString([]byte(a.jwtSecret))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Token generation failed",
					"message": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"access_token": tokenString,
				"token_type":   "Bearer",
				"expires_in":   900, // 15 minutes
			})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid token claims",
				"message": "Refresh token contains invalid claims",
			})
		}
	}
}

// Helper methods
func (a *AuthMiddleware) shouldSkipAuth(path string) bool {
	// Check exact matches
	if a.skipPaths[path] {
		return true
	}

	// Check path prefixes
	skipPrefixes := []string{
		"/swagger/",
		"/docs/",
		"/static/",
		"/.well-known/",
	}

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// GenerateJWT creates a new JWT token for a user
func (a *AuthMiddleware) GenerateJWT(userID int64, username, role, scope string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		Scope:    scope,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "wallet-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.jwtSecret))
}

// GenerateRefreshJWT creates a refresh token
func (a *AuthMiddleware) GenerateRefreshJWT(userID int64, username, role, scope string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		Scope:    scope,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "wallet-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.jwtSecret))
}

// GenerateAdminJWT creates a new JWT token for an admin
func (a *AuthMiddleware) GenerateAdminJWT(adminID, username, role string, permissions []string) (string, error) {
	claims := &AdminClaims{
		AdminID:     adminID,
		Username:    username,
		Role:        role,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)), // 30 minutes for admin
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "wallet-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.jwtSecret))
}