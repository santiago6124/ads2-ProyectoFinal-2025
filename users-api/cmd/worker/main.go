package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"users-api/internal/config"
	"users-api/internal/messaging"
	"users-api/internal/repositories"
	"users-api/internal/services"
	"users-api/pkg/database"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	logger.Info("🚀 Starting Users API Balance Worker")

	// Load configuration
	cfg := config.LoadConfig()

	// Connect to MySQL database
	db, err := database.NewConnection()
	if err != nil {
		logger.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()
	logger.Info("✅ Connected to MySQL database")

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db.DB)
	balanceRepo := repositories.NewBalanceTransactionRepository(db.DB)
	logger.Info("✅ Repositories initialized")

	// Initialize user service with balance support
	userService := services.NewUserServiceWithBalance(userRepo, balanceRepo)
	logger.Info("✅ User service initialized")

	// Initialize RabbitMQ components
	logger.Info("🔌 Initializing RabbitMQ components...")

	// Create balance response publisher
	responsePublisher, err := messaging.NewBalanceResponsePublisher(
		cfg.RabbitMQ.URL,
		cfg.RabbitMQ.BalanceResponseExchange,
		cfg.RabbitMQ.BalanceResponseRoutingKey,
		logger,
	)
	if err != nil {
		logger.Fatalf("❌ Failed to create response publisher: %v", err)
	}
	defer responsePublisher.Close()

	// Create balance request consumer
	requestConsumer, err := messaging.NewBalanceRequestConsumer(
		cfg.RabbitMQ.URL,
		cfg.RabbitMQ.BalanceRequestQueue,
		userService,
		responsePublisher,
		logger,
	)
	if err != nil {
		logger.Fatalf("❌ Failed to create request consumer: %v", err)
	}
	defer requestConsumer.Close()

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Handle shutdown signal in goroutine
	go func() {
		sig := <-sigChan
		logger.Infof("📡 Received signal %v, initiating graceful shutdown...", sig)
		cancel()
	}()

	// Print startup banner
	fmt.Println("╔════════════════════════════════════════════════════╗")
	fmt.Println("║   Users API Balance Worker                        ║")
	fmt.Println("║   Ready to process balance requests               ║")
	fmt.Println("╚════════════════════════════════════════════════════╝")
	logger.Info("🟢 Worker is ready to process messages")

	// Start consuming (blocking)
	if err := requestConsumer.Start(ctx); err != nil {
		if err != context.Canceled {
			logger.Errorf("❌ Worker stopped with error: %v", err)
			os.Exit(1)
		}
	}

	logger.Info("👋 Worker shutdown complete")
}
