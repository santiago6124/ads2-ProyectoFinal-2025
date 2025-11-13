package routes

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"orders-api/internal/handlers"
	"orders-api/internal/middleware"
)

type Router struct {
	engine         *gin.Engine
	orderHandler   *handlers.OrderHandler
	healthHandler  *handlers.HealthHandler
	authMiddleware *middleware.AuthMiddleware
	logMiddleware  *middleware.LoggingMiddleware
}

type RouterConfig struct {
	Debug          bool
	CORSEnabled    bool
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

func NewRouter(
	orderHandler *handlers.OrderHandler,
	healthHandler *handlers.HealthHandler,
	authMiddleware *middleware.AuthMiddleware,
	logMiddleware *middleware.LoggingMiddleware,
	config *RouterConfig,
) *Router {
	if !config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	return &Router{
		engine:         engine,
		orderHandler:   orderHandler,
		healthHandler:  healthHandler,
		authMiddleware: authMiddleware,
		logMiddleware:  logMiddleware,
	}
}

func (r *Router) SetupRoutes(config *RouterConfig) {
	// Global middleware
	r.setupGlobalMiddleware(config)

	// Health endpoints (no auth required)
	r.setupHealthRoutes()

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	r.setupAPIRoutes(v1)

	// Public endpoints
	public := r.engine.Group("/public")
	r.setupPublicRoutes(public)
}

func (r *Router) setupGlobalMiddleware(config *RouterConfig) {
	// Panic recovery
	r.engine.Use(r.logMiddleware.LogPanic())

	// Structured logging
	r.engine.Use(r.logMiddleware.StructuredLogging())

	// Request logging
	r.engine.Use(r.logMiddleware.LogRequests())

	// Error logging
	r.engine.Use(r.logMiddleware.LogErrors())

	// Slow request logging (requests taking more than 5 seconds)
	r.engine.Use(r.logMiddleware.LogSlowRequests(5 * time.Second))

	// CORS
	if config.CORSEnabled {
		corsConfig := cors.Config{
			AllowOrigins: config.AllowedOrigins,
			AllowMethods: config.AllowedMethods,
			AllowHeaders: config.AllowedHeaders,
			AllowCredentials: true,
			MaxAge: 12 * time.Hour,
		}

		if len(corsConfig.AllowOrigins) == 0 {
			corsConfig.AllowAllOrigins = true
		}

		if len(corsConfig.AllowMethods) == 0 {
			corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
		}

		if len(corsConfig.AllowHeaders) == 0 {
			corsConfig.AllowHeaders = []string{
				"Origin",
				"Content-Length",
				"Content-Type",
				"Authorization",
				"X-Requested-With",
				"X-Request-ID",
			}
		}

		r.engine.Use(cors.New(corsConfig))
	}

	// Security headers
	r.engine.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})
}

func (r *Router) setupHealthRoutes() {
	health := r.engine.Group("/health")
	{
		health.GET("", r.healthHandler.Health)
		health.GET("/live", r.healthHandler.Liveness)
		health.GET("/ready", r.healthHandler.Readiness)
	}

	// Metrics endpoint
	r.engine.GET("/metrics", r.healthHandler.Metrics)
}

func (r *Router) setupAPIRoutes(v1 *gin.RouterGroup) {
	// Apply authentication middleware to all API routes
	v1.Use(r.authMiddleware.ValidateToken())

	// Orders endpoints
	orders := v1.Group("/orders")
	{
		orders.POST("", r.orderHandler.CreateOrder)
		orders.GET("", r.orderHandler.ListUserOrders)
		orders.GET("/:id", r.orderHandler.GetOrder)
		orders.PUT("/:id", r.orderHandler.UpdateOrder)
		orders.DELETE("/:id", r.orderHandler.DeleteOrder)
		orders.POST("/:id/cancel", r.orderHandler.CancelOrder)
		orders.POST("/:id/execute", r.orderHandler.ExecuteOrder) // Endpoint de acción
	}

	// User-specific order endpoints
	users := v1.Group("/users/:user_id")
	users.Use(r.authMiddleware.ValidateOwnership())
	{
		userOrders := users.Group("/orders")
		{
			userOrders.GET("", r.orderHandler.ListUserOrders)
			userOrders.POST("", r.orderHandler.CreateOrder)
		}
	}

	// Admin endpoints (require admin role)
	admin := v1.Group("/admin")
	admin.Use(r.authMiddleware.RequireRole("admin"))
	{
		adminOrders := admin.Group("/orders")
		{
			adminOrders.GET("", r.orderHandler.ListUserOrders)
			adminOrders.GET("/:id", r.orderHandler.GetOrder)
			adminOrders.PUT("/:id", r.orderHandler.UpdateOrder)
			adminOrders.DELETE("/:id", r.orderHandler.DeleteOrder)
			adminOrders.POST("/:id/cancel", r.orderHandler.CancelOrder)
			adminOrders.POST("/:id/execute", r.orderHandler.ExecuteOrder)
		}
	}
}

func (r *Router) setupPublicRoutes(public *gin.RouterGroup) {
	// Optional authentication for public routes
	public.Use(r.authMiddleware.OptionalAuth())

	// Public health endpoint
	public.GET("/health", r.healthHandler.Health)

	// Add any other public endpoints here
}

func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

func (r *Router) AddCustomRoute(method, path string, handlers ...gin.HandlerFunc) {
	switch method {
	case "GET":
		r.engine.GET(path, handlers...)
	case "POST":
		r.engine.POST(path, handlers...)
	case "PUT":
		r.engine.PUT(path, handlers...)
	case "PATCH":
		r.engine.PATCH(path, handlers...)
	case "DELETE":
		r.engine.DELETE(path, handlers...)
	default:
		logrus.Warnf("Unsupported HTTP method: %s", method)
	}
}

func (r *Router) AddProtectedRoute(method, path string, requiredRole string, handlers ...gin.HandlerFunc) {
	allHandlers := []gin.HandlerFunc{
		r.authMiddleware.ValidateToken(),
	}

	if requiredRole != "" {
		allHandlers = append(allHandlers, r.authMiddleware.RequireRole(requiredRole))
	}

	allHandlers = append(allHandlers, handlers...)

	r.AddCustomRoute(method, path, allHandlers...)
}

func (r *Router) AddPublicRoute(method, path string, handlers ...gin.HandlerFunc) {
	allHandlers := []gin.HandlerFunc{
		r.authMiddleware.OptionalAuth(),
	}

	allHandlers = append(allHandlers, handlers...)

	r.AddCustomRoute(method, path, allHandlers...)
}

func DefaultRouterConfig() *RouterConfig {
	return &RouterConfig{
		Debug:       false,
		CORSEnabled: true,
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"https://cryptosim.example.com",
		},
		AllowedMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"HEAD",
			"OPTIONS",
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"X-Requested-With",
			"X-Request-ID",
			"Accept",
			"Accept-Encoding",
			"Accept-Language",
		},
	}
}