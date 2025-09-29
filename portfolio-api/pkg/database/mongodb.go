package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"portfolio-api/internal/config"
)

// MongoDB represents MongoDB database connection
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(cfg config.DatabaseConfig) (*MongoDB, error) {
	// Create client options
	clientOpts := options.Client().ApplyURI(cfg.URI)

	// Set connection pool options
	if cfg.MaxPoolSize > 0 {
		clientOpts.SetMaxPoolSize(uint64(cfg.MaxPoolSize))
	}
	if cfg.MinPoolSize > 0 {
		clientOpts.SetMinPoolSize(uint64(cfg.MinPoolSize))
	}
	if cfg.MaxIdleTime > 0 {
		clientOpts.SetMaxConnIdleTime(time.Duration(cfg.MaxIdleTime) * time.Second)
	}

	// Set timeouts
	if cfg.ConnectTimeout > 0 {
		clientOpts.SetConnectTimeout(time.Duration(cfg.ConnectTimeout) * time.Second)
	}
	if cfg.SocketTimeout > 0 {
		clientOpts.SetSocketTimeout(time.Duration(cfg.SocketTimeout) * time.Second)
	}

	// Set replica set
	if cfg.ReplicaSet != "" {
		clientOpts.SetReplicaSet(cfg.ReplicaSet)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(cfg.Database)

	// Create indexes
	if err := createIndexes(ctx, database); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return &MongoDB{
		client:   client,
		database: database,
	}, nil
}

// GetDatabase returns the database instance
func (m *MongoDB) GetDatabase() *mongo.Database {
	return m.database
}

// GetClient returns the client instance
func (m *MongoDB) GetClient() *mongo.Client {
	return m.client
}

// Collection returns a collection
func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.database.Collection(name)
}

// Disconnect closes the database connection
func (m *MongoDB) Disconnect() error {
	if m.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return m.client.Disconnect(ctx)
}

// Ping checks the database connection
func (m *MongoDB) Ping(ctx context.Context) error {
	return m.client.Ping(ctx, readpref.Primary())
}

// createIndexes creates necessary indexes for collections
func createIndexes(ctx context.Context, db *mongo.Database) error {
	// Portfolio collection indexes
	portfolioCollection := db.Collection("portfolios")
	portfolioIndexes := []mongo.IndexModel{
		{
			Keys:    map[string]interface{}{"user_id": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: map[string]interface{}{"updated_at": -1},
		},
		{
			Keys: map[string]interface{}{"metadata.needs_recalculation": 1},
		},
		{
			Keys: map[string]interface{}{"metadata.last_calculated": -1},
		},
		{
			Keys: map[string]interface{}{"total_value": -1},
		},
	}

	if _, err := portfolioCollection.Indexes().CreateMany(ctx, portfolioIndexes); err != nil {
		return fmt.Errorf("failed to create portfolio indexes: %w", err)
	}

	// Portfolio snapshots collection indexes
	snapshotCollection := db.Collection("portfolio_snapshots")
	snapshotIndexes := []mongo.IndexModel{
		{
			Keys: map[string]interface{}{"user_id": 1, "timestamp": -1},
		},
		{
			Keys: map[string]interface{}{"portfolio_id": 1, "interval": 1, "timestamp": -1},
		},
		{
			Keys:    map[string]interface{}{"timestamp": -1},
			Options: options.Index().SetExpireAfterSeconds(7776000), // 90 days
		},
		{
			Keys: map[string]interface{}{"interval": 1, "timestamp": -1},
		},
		{
			Keys: map[string]interface{}{"tags": 1},
		},
	}

	if _, err := snapshotCollection.Indexes().CreateMany(ctx, snapshotIndexes); err != nil {
		return fmt.Errorf("failed to create snapshot indexes: %w", err)
	}

	return nil
}

// Transaction executes a function within a MongoDB transaction
func (m *MongoDB) Transaction(ctx context.Context, fn func(ctx mongo.SessionContext) error) error {
	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Execute transaction
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		return nil, fn(sc)
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

// DropDatabase drops the entire database (for testing)
func (m *MongoDB) DropDatabase(ctx context.Context) error {
	return m.database.Drop(ctx)
}

// GetDatabaseStats returns database statistics
func (m *MongoDB) GetDatabaseStats(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := m.database.RunCommand(ctx, map[string]interface{}{"dbStats": 1}).Decode(&result)
	return result, err
}

// GetCollectionStats returns collection statistics
func (m *MongoDB) GetCollectionStats(ctx context.Context, collectionName string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := m.database.RunCommand(ctx, map[string]interface{}{
		"collStats": collectionName,
	}).Decode(&result)
	return result, err
}

// Health check for database
func (m *MongoDB) IsHealthy(ctx context.Context) bool {
	return m.Ping(ctx) == nil
}

// GetConnectionInfo returns connection information
func (m *MongoDB) GetConnectionInfo() map[string]interface{} {
	return map[string]interface{}{
		"database": m.database.Name(),
	}
}