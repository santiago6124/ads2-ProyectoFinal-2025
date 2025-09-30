package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"wallet-api/internal/models"
)

type WalletRepository interface {
	Create(ctx context.Context, wallet *models.Wallet) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Wallet, error)
	GetByUserID(ctx context.Context, userID int64) (*models.Wallet, error)
	GetByWalletNumber(ctx context.Context, walletNumber string) (*models.Wallet, error)
	Update(ctx context.Context, wallet *models.Wallet) error
	UpdateBalance(ctx context.Context, walletID primitive.ObjectID, availableBalance, lockedBalance decimal.Decimal) error
	AddLock(ctx context.Context, walletID primitive.ObjectID, lock models.FundsLock) error
	UpdateLock(ctx context.Context, walletID primitive.ObjectID, lockID string, status string) error
	CleanupExpiredLocks(ctx context.Context) error
	GetWalletsForReconciliation(ctx context.Context, limit int) ([]*models.Wallet, error)
	UpdateVerificationInfo(ctx context.Context, walletID primitive.ObjectID, verification models.Verification) error
	GetActiveWallets(ctx context.Context, limit int, offset int) ([]*models.Wallet, error)
	SetWalletStatus(ctx context.Context, walletID primitive.ObjectID, status string) error
}

type walletRepository struct {
	collection *mongo.Collection
	db         *mongo.Database
}

func NewWalletRepository(db *mongo.Database) WalletRepository {
	return &walletRepository{
		collection: db.Collection("wallets"),
		db:         db,
	}
}

func (r *walletRepository) Create(ctx context.Context, wallet *models.Wallet) error {
	wallet.CreatedAt = time.Now()
	wallet.UpdatedAt = time.Now()
	wallet.LastActivity = time.Now()

	result, err := r.collection.InsertOne(ctx, wallet)
	if err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	wallet.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *walletRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&wallet)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("wallet not found")
		}
		return nil, fmt.Errorf("failed to get wallet by ID: %w", err)
	}
	return &wallet, nil
}

func (r *walletRepository) GetByUserID(ctx context.Context, userID int64) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&wallet)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("wallet not found for user %d", userID)
		}
		return nil, fmt.Errorf("failed to get wallet by user ID: %w", err)
	}
	return &wallet, nil
}

func (r *walletRepository) GetByWalletNumber(ctx context.Context, walletNumber string) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.collection.FindOne(ctx, bson.M{"wallet_number": walletNumber}).Decode(&wallet)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("wallet not found with number %s", walletNumber)
		}
		return nil, fmt.Errorf("failed to get wallet by wallet number: %w", err)
	}
	return &wallet, nil
}

func (r *walletRepository) Update(ctx context.Context, wallet *models.Wallet) error {
	wallet.UpdatedAt = time.Now()

	filter := bson.M{"_id": wallet.ID}
	update := bson.M{"$set": wallet}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update wallet: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("wallet not found for update")
	}

	return nil
}

func (r *walletRepository) UpdateBalance(ctx context.Context, walletID primitive.ObjectID, availableBalance, lockedBalance decimal.Decimal) error {
	totalBalance := availableBalance.Add(lockedBalance)

	filter := bson.M{"_id": walletID}
	update := bson.M{
		"$set": bson.M{
			"balance.available": availableBalance,
			"balance.locked":    lockedBalance,
			"balance.total":     totalBalance,
			"updated_at":        time.Now(),
			"last_activity":     time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("wallet not found for balance update")
	}

	return nil
}

func (r *walletRepository) AddLock(ctx context.Context, walletID primitive.ObjectID, lock models.FundsLock) error {
	filter := bson.M{"_id": walletID}
	update := bson.M{
		"$push": bson.M{"locks": lock},
		"$set": bson.M{
			"updated_at":    time.Now(),
			"last_activity": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to add lock: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("wallet not found for adding lock")
	}

	return nil
}

func (r *walletRepository) UpdateLock(ctx context.Context, walletID primitive.ObjectID, lockID string, status string) error {
	filter := bson.M{
		"_id":         walletID,
		"locks.lock_id": lockID,
	}
	update := bson.M{
		"$set": bson.M{
			"locks.$.status":    status,
			"updated_at":        time.Now(),
			"last_activity":     time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update lock status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("wallet or lock not found for status update")
	}

	return nil
}

func (r *walletRepository) CleanupExpiredLocks(ctx context.Context) error {
	now := time.Now()

	// Find wallets with expired active locks
	filter := bson.M{
		"locks": bson.M{
			"$elemMatch": bson.M{
				"status":     "active",
				"expires_at": bson.M{"$lt": now},
			},
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find wallets with expired locks: %w", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var wallet models.Wallet
		if err := cursor.Decode(&wallet); err != nil {
			continue
		}

		// Process expired locks
		var expiredAmount decimal.Decimal
		for i := range wallet.Locks {
			lock := &wallet.Locks[i]
			if lock.Status == "active" && now.After(lock.ExpiresAt) {
				lock.Status = "expired"
				expiredAmount = expiredAmount.Add(lock.Amount)
			}
		}

		if expiredAmount.GreaterThan(decimal.Zero) {
			// Update balance and locks atomically
			wallet.Balance.Locked = wallet.Balance.Locked.Sub(expiredAmount)
			wallet.Balance.Available = wallet.Balance.Available.Add(expiredAmount)
			wallet.Balance.Total = wallet.Balance.Available.Add(wallet.Balance.Locked)
			wallet.UpdatedAt = now

			if err := r.Update(ctx, &wallet); err != nil {
				// Log error but continue with other wallets
				continue
			}
		}
	}

	return cursor.Err()
}

func (r *walletRepository) GetWalletsForReconciliation(ctx context.Context, limit int) ([]*models.Wallet, error) {
	// Get wallets that haven't been reconciled recently
	filter := bson.M{
		"$or": []bson.M{
			{"verification.last_reconciled": bson.M{"$lt": time.Now().Add(-24 * time.Hour)}},
			{"verification.last_reconciled": bson.M{"$exists": false}},
		},
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.M{"verification.last_reconciled": 1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallets for reconciliation: %w", err)
	}
	defer cursor.Close(ctx)

	var wallets []*models.Wallet
	for cursor.Next(ctx) {
		var wallet models.Wallet
		if err := cursor.Decode(&wallet); err != nil {
			continue
		}
		wallets = append(wallets, &wallet)
	}

	return wallets, cursor.Err()
}

func (r *walletRepository) UpdateVerificationInfo(ctx context.Context, walletID primitive.ObjectID, verification models.Verification) error {
	filter := bson.M{"_id": walletID}
	update := bson.M{
		"$set": bson.M{
			"verification": verification,
			"updated_at":   time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update verification info: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("wallet not found for verification update")
	}

	return nil
}

func (r *walletRepository) GetActiveWallets(ctx context.Context, limit int, offset int) ([]*models.Wallet, error) {
	filter := bson.M{"status": "active"}
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.M{"last_activity": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get active wallets: %w", err)
	}
	defer cursor.Close(ctx)

	var wallets []*models.Wallet
	for cursor.Next(ctx) {
		var wallet models.Wallet
		if err := cursor.Decode(&wallet); err != nil {
			continue
		}
		wallets = append(wallets, &wallet)
	}

	return wallets, cursor.Err()
}

func (r *walletRepository) SetWalletStatus(ctx context.Context, walletID primitive.ObjectID, status string) error {
	filter := bson.M{"_id": walletID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update wallet status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("wallet not found for status update")
	}

	return nil
}

// CreateIndexes creates necessary indexes for the wallet collection
func (r *walletRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "wallet_number", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "last_activity", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "verification.last_reconciled", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "locks.lock_id", Value: 1},
				{Key: "locks.status", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "locks.expires_at", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create wallet indexes: %w", err)
	}

	return nil
}