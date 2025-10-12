package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
)

type Database struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewConnection() (*Database, error) {
	mongoURI := getEnv("MONGODB_URI", "mongodb://localhost:27017")
	databaseName := getEnv("MONGODB_DATABASE", "orders_db")
	timeoutStr := getEnv("MONGODB_CONNECTION_TIMEOUT", "10s")

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(mongoURI)
	clientOptions.SetMaxPoolSize(100)
	clientOptions.SetMinPoolSize(10)
	clientOptions.SetMaxConnIdleTime(30 * time.Second)
	clientOptions.SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(databaseName)

	db := &Database{
		Client:   client,
		Database: database,
	}

	if err := db.CreateIndexes(); err != nil {
		log.Printf("Warning: Failed to create indexes: %v", err)
	}

	log.Printf("Connected to MongoDB database: %s", databaseName)
	return db, nil
}

func (d *Database) CreateIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ordersCollection := d.Database.Collection("orders")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{"user_id", 1},
				{"created_at", -1},
			},
			Options: options.Index().SetName("user_created_idx"),
		},
		{
			Keys: bson.D{
				{"status", 1},
				{"created_at", -1},
			},
			Options: options.Index().SetName("status_created_idx"),
		},
		{
			Keys: bson.D{
				{"order_number", 1},
			},
			Options: options.Index().SetUnique(true).SetName("order_number_unique_idx"),
		},
		{
			Keys: bson.D{
				{"crypto_symbol", 1},
				{"created_at", -1},
			},
			Options: options.Index().SetName("crypto_created_idx"),
		},
		{
			Keys: bson.D{
				{"executed_at", -1},
			},
			Options: options.Index().SetName("executed_at_idx").SetSparse(true),
		},
		{
			Keys: bson.D{
				{"user_id", 1},
				{"status", 1},
			},
			Options: options.Index().SetName("user_status_idx"),
		},
		{
			Keys: bson.D{
				{"user_id", 1},
				{"crypto_symbol", 1},
				{"created_at", -1},
			},
			Options: options.Index().SetName("user_crypto_created_idx"),
		},
		{
			Keys: bson.D{
				{"type", 1},
				{"created_at", -1},
			},
			Options: options.Index().SetName("type_created_idx"),
		},
		{
			Keys: bson.D{
				{"total_amount", -1},
			},
			Options: options.Index().SetName("total_amount_idx"),
		},
		{
			Keys: bson.D{
				{"updated_at", -1},
			},
			Options: options.Index().SetName("updated_at_idx"),
		},
	}

	_, err := ordersCollection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Println("MongoDB indexes created successfully")
	return nil
}

func (d *Database) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := d.Client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}

	log.Println("Disconnected from MongoDB")
	return nil
}

func (d *Database) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return d.Client.Ping(ctx, nil)
}

func (d *Database) GetCollection(name string) *mongo.Collection {
	return d.Database.Collection(name)
}

func (d *Database) StartSession() (mongo.Session, error) {
	return d.Client.StartSession()
}

func (d *Database) ExecuteTransaction(ctx context.Context, fn func(mongo.SessionContext) (interface{}, error)) (interface{}, error) {
	session, err := d.StartSession()
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	result, err := session.WithTransaction(ctx, fn)
	if err != nil {
		return nil, fmt.Errorf("transaction failed: %w", err)
	}

	return result, nil
}

func (d *Database) GetStats() (*DatabaseStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var stats bson.M
	if err := d.Database.RunCommand(ctx, bson.D{{"dbStats", 1}}).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to get database stats: %w", err)
	}

	ordersCount, err := d.GetCollection("orders").EstimatedDocumentCount(ctx)
	if err != nil {
		log.Printf("Warning: Failed to get orders count: %v", err)
		ordersCount = 0
	}

	return &DatabaseStats{
		DatabaseName:   d.Database.Name(),
		Collections:    int(stats["collections"].(int32)),
		Objects:        int(stats["objects"].(int32)),
		DataSize:       int64(stats["dataSize"].(int32)),
		StorageSize:    int64(stats["storageSize"].(int32)),
		IndexSize:      int64(stats["indexSize"].(int32)),
		OrdersCount:    ordersCount,
	}, nil
}

type DatabaseStats struct {
	DatabaseName string `json:"database_name"`
	Collections  int    `json:"collections"`
	Objects      int    `json:"objects"`
	DataSize     int64  `json:"data_size"`
	StorageSize  int64  `json:"storage_size"`
	IndexSize    int64  `json:"index_size"`
	OrdersCount  int64  `json:"orders_count"`
}


func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}