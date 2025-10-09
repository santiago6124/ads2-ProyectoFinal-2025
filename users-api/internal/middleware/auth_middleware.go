package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"users-api/internal/models"
	"users-api/internal/services"
	"users-api/pkg/utils"
)

func AuthMiddleware(tokenService services.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.SendUnauthorizedError(c, "Authorization header required")
			c.Abort()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			utils.SendUnauthorizedError(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := tokenParts[1]
		claims, err := tokenService.ValidateAccessToken(tokenString)
		if err != nil {
			utils.SendUnauthorizedError(c, "Invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			utils.SendUnauthorizedError(c, "User role not found")
			c.Abort()
			return
		}

		if userRole.(models.UserRole) != models.RoleAdmin {
			utils.SendForbiddenError(c, "Admin access required")
			c.Abort()
			return
		}

		c.Next()
	}
}

func InternalServiceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.GetHeader("X-Internal-Service")
		apiKey := c.GetHeader("X-API-Key")

		expectedAPIKey := os.Getenv("INTERNAL_API_KEY")
		if expectedAPIKey == "" {
			expectedAPIKey = "internal-secret-key"
		}

		if serviceName == "" || apiKey == "" {
			utils.SendUnauthorizedError(c, "Internal service headers required")
			c.Abort()
			return
		}

		if apiKey != expectedAPIKey {
			utils.SendUnauthorizedError(c, "Invalid internal API key")
			c.Abort()
			return
		}

		allowedServices := []string{
			"orders-api",
			"portfolio-api",
			"wallet-api",
			"ranking-api",
			"notifications-api",
			"audit-api",
		}

		serviceAllowed := false
		for _, allowedService := range allowedServices {
			if serviceName == allowedService {
				serviceAllowed = true
				break
			}
		}

		if !serviceAllowed {
			utils.SendForbiddenError(c, "Service not authorized")
			c.Abort()
			return
		}

		c.Set("service_name", serviceName)
		c.Next()
	}
}

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}