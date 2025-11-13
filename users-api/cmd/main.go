package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"users-api/internal/config"
	"users-api/internal/controllers"
	"users-api/internal/middleware"
	"users-api/internal/repositories"
	"users-api/internal/services"
	"users-api/pkg/database"
)

// @title Users API
// @version 1.0
// @description CryptoSim Users Management API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://cryptosim.com/support
// @contact.email support@cryptosim.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8001
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	cfg := config.LoadConfig()

	db, err := database.NewConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.AutoMigrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	userRepo := repositories.NewUserRepository(db.DB)
	refreshTokenRepo := repositories.NewRefreshTokenRepository(db.DB)
	loginAttemptRepo := repositories.NewLoginAttemptRepository(db.DB)
	balanceTransactionRepo := repositories.NewBalanceTransactionRepository(db.DB)

	tokenService := services.NewTokenService(&cfg.JWT, refreshTokenRepo)
	userService := services.NewUserServiceWithBalance(userRepo, balanceTransactionRepo)
	authService := services.NewAuthService(userRepo, loginAttemptRepo, tokenService)

	authController := controllers.NewAuthController(authService, userService)
	userController := controllers.NewUserController(userService)
	healthController := controllers.NewHealthController(db)

	router := setupRouter(cfg, authController, userController, healthController, tokenService)

	log.Printf("Starting Users API server on port %s", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, router))
}

func setupRouter(
	cfg *config.Config,
	authController *controllers.AuthController,
	userController *controllers.UserController,
	healthController *controllers.HealthController,
	tokenService services.TokenService,
) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.LoggingMiddleware())
	router.Use(gin.Recovery())

	router.GET("/health", healthController.Health)
	router.GET("/ready", healthController.Readiness)
	router.GET("/live", healthController.Liveness)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := router.Group("/api")
	{
		users := api.Group("/users")
		{
			users.POST("/register", authController.Register)
			users.POST("/login", authController.Login)
			users.POST("/refresh", authController.RefreshToken)
			users.POST("/logout", authController.Logout)

			authenticated := users.Group("")
			authenticated.Use(middleware.AuthMiddleware(tokenService))
			{
				authenticated.POST("/logout-all", authController.LogoutAll)
				authenticated.GET("/:id", userController.GetUser)
				authenticated.PUT("/:id", userController.UpdateUser)
				authenticated.PUT("/:id/password", userController.ChangePassword)
				authenticated.DELETE("/:id", userController.DeleteUser)

				admin := authenticated.Group("")
				admin.Use(middleware.AdminOnlyMiddleware())
				{
					admin.GET("", userController.ListUsers)
					admin.POST("/:id/upgrade", userController.UpgradeUser)
				}
			}

			internal := users.Group("")
			internal.Use(middleware.InternalServiceMiddleware())
			{
				internal.GET("/:id/verify", userController.VerifyUser)
				internal.PUT("/:id/balance", userController.UpdateBalance)
			}
		}
	}

	return router
}