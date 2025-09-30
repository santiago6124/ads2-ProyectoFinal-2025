package database

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"wallet-api/internal/config"
	"wallet-api/internal/repository"
)

type Database struct {
	MongoDB    *mongo.Database
	RedisDB    *redis.Client
	Repositories *Repositories
}

type Repositories struct {
	Wallet      repository.WalletRepository
	Transaction repository.TransactionRepository
	Lock        repository.LockRepository
	Idempotency repository.IdempotencyRepository
	LockManager *repository.WalletLockManager
}

func Initialize(ctx context.Context, cfg *config.Config) (*Database, error) {
	// Initialize MongoDB
	mongoDB, err := initializeMongoDB(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MongoDB: %w", err)
	}

	// Initialize Redis
	redisDB, err := initializeRedis(ctx, cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Initialize repositories
	repos := &Repositories{
		Wallet:      repository.NewWalletRepository(mongoDB),
		Transaction: repository.NewTransactionRepository(mongoDB),
		Lock:        repository.NewLockRepository(redisDB),
		Idempotency: repository.NewIdempotencyRepository(redisDB),
	}

	// Initialize lock manager
	repos.LockManager = repository.NewWalletLockManager(repos.Lock)

	// Create database indexes
	if err := createIndexes(ctx, repos); err != nil {
		return nil, fmt.Errorf("failed to create database indexes: %w", err)
	}

	return &Database{
		MongoDB:      mongoDB,
		RedisDB:      redisDB,
		Repositories: repos,
	}, nil
}

func initializeMongoDB(ctx context.Context, cfg config.DatabaseConfig) (*mongo.Database, error) {
	// Build connection string
	var uri string
	if cfg.Username != "" && cfg.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=%s",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.AuthSource)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%d/%s", cfg.Host, cfg.Port, cfg.Name)
	}

	// Set client options
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(uint64(cfg.MaxPoolSize)).
		SetMinPoolSize(uint64(cfg.MinPoolSize)).
		SetMaxConnIdleTime(cfg.MaxConnIdleTime).
		SetConnectTimeout(cfg.ConnectTimeout).
		SetSocketTimeout(cfg.SocketTimeout).
		SetServerSelectionTimeout(cfg.ServerSelectionTimeout)

	// Enable SSL if configured
	if cfg.SSL.Enabled {
		tlsConfig := &options.TLSConfig{
			Insecure: cfg.SSL.InsecureSkipVerify,
		}
		if cfg.SSL.CertFile != "" && cfg.SSL.KeyFile != "" {
			tlsConfig.CertificateFile = cfg.SSL.CertFile
			tlsConfig.PrivateKeyFile = cfg.SSL.KeyFile
		}
		if cfg.SSL.CAFile != "" {
			tlsConfig.CaFile = cfg.SSL.CAFile
		}
		clientOptions.SetTLSConfig(tlsConfig)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return client.Database(cfg.Name), nil
}

func initializeRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	// Configure Redis client
	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Enable SSL if configured
	if cfg.SSL.Enabled {
		opts.TLSConfig = &redis.TLSConfig{
			InsecureSkipVerify: cfg.SSL.InsecureSkipVerify,
			ServerName:         cfg.SSL.ServerName,
		}
		if cfg.SSL.CertFile != "" && cfg.SSL.KeyFile != "" {
			// Load client certificate
			// Note: Implementation would load cert/key files here
		}
	}

	client := redis.NewClient(opts)

	// Test connection
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}

func createIndexes(ctx context.Context, repos *Repositories) error {
	// Note: Index creation would be implemented by adding CreateIndexes methods
	// to the repository interfaces and implementing them in the concrete types
	// For now, we'll skip this as it requires interface changes
	return nil
}

func (db *Database) Close(ctx context.Context) error {
	var errs []error

	// Close MongoDB connection
	if db.MongoDB != nil {
		if err := db.MongoDB.Client().Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to close MongoDB: %w", err))
		}
	}

	// Close Redis connection
	if db.RedisDB != nil {
		if err := db.RedisDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Redis: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing database connections: %v", errs)
	}

	return nil
}

// Health check methods
func (db *Database) HealthCheck(ctx context.Context) error {
	// Check MongoDB
	if err := db.MongoDB.Client().Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("MongoDB health check failed: %w", err)
	}

	// Check Redis
	if _, err := db.RedisDB.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}

	return nil
}

// Transaction support for MongoDB
func (db *Database) WithTransaction(ctx context.Context, fn func(context.Context, mongo.Session) error) error {
	session, err := db.MongoDB.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		return nil, fn(sc, session)
	})

	return err
}

// Cleanup operations
func (db *Database) RunMaintenance(ctx context.Context) error {
	// Cleanup expired locks
	if err := db.Repositories.Lock.CleanupExpiredLocks(ctx); err != nil {
		return fmt.Errorf("failed to cleanup expired locks: %w", err)
	}

	// Cleanup expired wallet locks
	if err := db.Repositories.Wallet.CleanupExpiredLocks(ctx); err != nil {
		return fmt.Errorf("failed to cleanup expired wallet locks: %w", err)
	}

	// Cleanup old failed transactions (older than 30 days)
	oldTime := time.Now().AddDate(0, 0, -30)
	if err := db.Repositories.Transaction.CleanupOldTransactions(ctx, oldTime); err != nil {
		return fmt.Errorf("failed to cleanup old transactions: %w", err)
	}

	return nil
}