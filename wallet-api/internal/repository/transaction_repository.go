package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"wallet-api/internal/models"
)

type TransactionRepository interface {
	Create(ctx context.Context, transaction *models.Transaction) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Transaction, error)
	GetByTransactionID(ctx context.Context, transactionID string) (*models.Transaction, error)
	GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*models.Transaction, error)
	Update(ctx context.Context, transaction *models.Transaction) error
	UpdateStatus(ctx context.Context, transactionID string, status string) error
	GetByWalletID(ctx context.Context, walletID primitive.ObjectID, limit int, offset int) ([]*models.Transaction, error)
	GetByUserID(ctx context.Context, userID int64, limit int, offset int) ([]*models.Transaction, error)
	GetPendingTransactions(ctx context.Context, limit int) ([]*models.Transaction, error)
	GetTransactionsByDateRange(ctx context.Context, walletID primitive.ObjectID, startDate, endDate time.Time) ([]*models.Transaction, error)
	GetTransactionsByType(ctx context.Context, walletID primitive.ObjectID, transactionType string, limit int, offset int) ([]*models.Transaction, error)
	GetFailedTransactions(ctx context.Context, limit int, olderThan time.Time) ([]*models.Transaction, error)
	GetReversibleTransactions(ctx context.Context, walletID primitive.ObjectID) ([]*models.Transaction, error)
	MarkAsReversed(ctx context.Context, transactionID string, reversalInfo models.ReversalInfo) error
	GetTransactionStats(ctx context.Context, walletID primitive.ObjectID, startDate, endDate time.Time) (*TransactionStats, error)
	CleanupOldTransactions(ctx context.Context, olderThan time.Time) error
}

type TransactionStats struct {
	TotalTransactions int64                    `json:"total_transactions"`
	TotalVolume       map[string]interface{}   `json:"total_volume"`
	ByType            map[string]int64         `json:"by_type"`
	ByStatus          map[string]int64         `json:"by_status"`
	AverageAmount     map[string]interface{}   `json:"average_amount"`
}

type transactionRepository struct {
	collection *mongo.Collection
	db         *mongo.Database
}

func NewTransactionRepository(db *mongo.Database) TransactionRepository {
	return &transactionRepository{
		collection: db.Collection("transactions"),
		db:         db,
	}
}

func (r *transactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	transaction.CreatedAt = time.Now()
	transaction.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, transaction)
	if err != nil {
		// Check for duplicate key error (idempotency key)
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("transaction with idempotency key already exists")
		}
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	transaction.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *transactionRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&transaction)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction by ID: %w", err)
	}
	return &transaction, nil
}

func (r *transactionRepository) GetByTransactionID(ctx context.Context, transactionID string) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.collection.FindOne(ctx, bson.M{"transaction_id": transactionID}).Decode(&transaction)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("transaction not found with ID %s", transactionID)
		}
		return nil, fmt.Errorf("failed to get transaction by transaction ID: %w", err)
	}
	return &transaction, nil
}

func (r *transactionRepository) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.collection.FindOne(ctx, bson.M{"idempotency_key": idempotencyKey}).Decode(&transaction)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found is not an error for idempotency checks
		}
		return nil, fmt.Errorf("failed to get transaction by idempotency key: %w", err)
	}
	return &transaction, nil
}

func (r *transactionRepository) Update(ctx context.Context, transaction *models.Transaction) error {
	transaction.UpdatedAt = time.Now()

	filter := bson.M{"_id": transaction.ID}
	update := bson.M{"$set": transaction}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("transaction not found for update")
	}

	return nil
}

func (r *transactionRepository) UpdateStatus(ctx context.Context, transactionID string, status string) error {
	filter := bson.M{"transaction_id": transactionID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("transaction not found for status update")
	}

	return nil
}

func (r *transactionRepository) GetByWalletID(ctx context.Context, walletID primitive.ObjectID, limit int, offset int) ([]*models.Transaction, error) {
	filter := bson.M{"wallet_id": walletID}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by wallet ID: %w", err)
	}
	defer cursor.Close(ctx)

	var transactions []*models.Transaction
	for cursor.Next(ctx) {
		var transaction models.Transaction
		if err := cursor.Decode(&transaction); err != nil {
			continue
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, cursor.Err()
}

func (r *transactionRepository) GetByUserID(ctx context.Context, userID int64, limit int, offset int) ([]*models.Transaction, error) {
	filter := bson.M{"user_id": userID}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by user ID: %w", err)
	}
	defer cursor.Close(ctx)

	var transactions []*models.Transaction
	for cursor.Next(ctx) {
		var transaction models.Transaction
		if err := cursor.Decode(&transaction); err != nil {
			continue
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, cursor.Err()
}

func (r *transactionRepository) GetPendingTransactions(ctx context.Context, limit int) ([]*models.Transaction, error) {
	filter := bson.M{
		"status": bson.M{"$in": []string{"pending", "processing"}},
	}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.M{"created_at": 1}) // Oldest first

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending transactions: %w", err)
	}
	defer cursor.Close(ctx)

	var transactions []*models.Transaction
	for cursor.Next(ctx) {
		var transaction models.Transaction
		if err := cursor.Decode(&transaction); err != nil {
			continue
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, cursor.Err()
}

func (r *transactionRepository) GetTransactionsByDateRange(ctx context.Context, walletID primitive.ObjectID, startDate, endDate time.Time) ([]*models.Transaction, error) {
	filter := bson.M{
		"wallet_id": walletID,
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}
	opts := options.Find().SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by date range: %w", err)
	}
	defer cursor.Close(ctx)

	var transactions []*models.Transaction
	for cursor.Next(ctx) {
		var transaction models.Transaction
		if err := cursor.Decode(&transaction); err != nil {
			continue
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, cursor.Err()
}

func (r *transactionRepository) GetTransactionsByType(ctx context.Context, walletID primitive.ObjectID, transactionType string, limit int, offset int) ([]*models.Transaction, error) {
	filter := bson.M{
		"wallet_id": walletID,
		"type":      transactionType,
	}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by type: %w", err)
	}
	defer cursor.Close(ctx)

	var transactions []*models.Transaction
	for cursor.Next(ctx) {
		var transaction models.Transaction
		if err := cursor.Decode(&transaction); err != nil {
			continue
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, cursor.Err()
}

func (r *transactionRepository) GetFailedTransactions(ctx context.Context, limit int, olderThan time.Time) ([]*models.Transaction, error) {
	filter := bson.M{
		"status": "failed",
		"created_at": bson.M{"$lt": olderThan},
	}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.M{"created_at": 1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed transactions: %w", err)
	}
	defer cursor.Close(ctx)

	var transactions []*models.Transaction
	for cursor.Next(ctx) {
		var transaction models.Transaction
		if err := cursor.Decode(&transaction); err != nil {
			continue
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, cursor.Err()
}

func (r *transactionRepository) GetReversibleTransactions(ctx context.Context, walletID primitive.ObjectID) ([]*models.Transaction, error) {
	filter := bson.M{
		"wallet_id": walletID,
		"status":    "completed",
		"reversal.is_reversed": false,
		"type": bson.M{"$in": []string{"deposit", "withdrawal", "refund", "adjustment"}},
	}
	opts := options.Find().SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get reversible transactions: %w", err)
	}
	defer cursor.Close(ctx)

	var transactions []*models.Transaction
	for cursor.Next(ctx) {
		var transaction models.Transaction
		if err := cursor.Decode(&transaction); err != nil {
			continue
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, cursor.Err()
}

func (r *transactionRepository) MarkAsReversed(ctx context.Context, transactionID string, reversalInfo models.ReversalInfo) error {
	filter := bson.M{"transaction_id": transactionID}
	update := bson.M{
		"$set": bson.M{
			"reversal":   reversalInfo,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to mark transaction as reversed: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("transaction not found for reversal marking")
	}

	return nil
}

func (r *transactionRepository) GetTransactionStats(ctx context.Context, walletID primitive.ObjectID, startDate, endDate time.Time) (*TransactionStats, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"wallet_id": walletID,
				"created_at": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
				"status": "completed",
			},
		},
		{
			"$group": bson.M{
				"_id": nil,
				"total_transactions": bson.M{"$sum": 1},
				"total_volume": bson.M{
					"$sum": bson.M{
						"$abs": "$amount.value",
					},
				},
				"by_type": bson.M{
					"$push": "$type",
				},
				"by_status": bson.M{
					"$push": "$status",
				},
				"average_amount": bson.M{
					"$avg": bson.M{
						"$abs": "$amount.value",
					},
				},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalTransactions int64     `bson:"total_transactions"`
		TotalVolume       float64   `bson:"total_volume"`
		ByType            []string  `bson:"by_type"`
		ByStatus          []string  `bson:"by_status"`
		AverageAmount     float64   `bson:"average_amount"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode transaction stats: %w", err)
		}
	}

	// Count occurrences by type and status
	typeCount := make(map[string]int64)
	statusCount := make(map[string]int64)

	for _, t := range result.ByType {
		typeCount[t]++
	}

	for _, s := range result.ByStatus {
		statusCount[s]++
	}

	stats := &TransactionStats{
		TotalTransactions: result.TotalTransactions,
		TotalVolume:       map[string]interface{}{"USD": result.TotalVolume},
		ByType:            typeCount,
		ByStatus:          statusCount,
		AverageAmount:     map[string]interface{}{"USD": result.AverageAmount},
	}

	return stats, nil
}

func (r *transactionRepository) CleanupOldTransactions(ctx context.Context, olderThan time.Time) error {
	// Only cleanup failed transactions that are very old
	filter := bson.M{
		"status":     "failed",
		"created_at": bson.M{"$lt": olderThan},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to cleanup old transactions: %w", err)
	}

	if result.DeletedCount > 0 {
		// Log cleanup operation
	}

	return nil
}

// CreateIndexes creates necessary indexes for the transaction collection
func (r *transactionRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "transaction_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "idempotency_key", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"idempotency_key": bson.M{"$exists": true}}),
		},
		{
			Keys: bson.D{{Key: "wallet_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "wallet_id", Value: 1},
				{Key: "type", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "reversal.is_reversed", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "processing.initiated_at", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create transaction indexes: %w", err)
	}

	return nil
}