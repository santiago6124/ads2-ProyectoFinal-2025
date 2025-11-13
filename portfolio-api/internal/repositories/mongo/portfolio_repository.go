package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"portfolio-api/internal/models"
	"portfolio-api/internal/repositories"
)

// MongoPortfolioRepository implements PortfolioRepository using MongoDB
type MongoPortfolioRepository struct {
	collection *mongo.Collection
}

// NewPortfolioRepository creates a new MongoDB portfolio repository
func NewPortfolioRepository(db *mongo.Database) repositories.PortfolioRepository {
	return &MongoPortfolioRepository{
		collection: db.Collection("portfolios"),
	}
}

// Create creates a new portfolio
func (r *MongoPortfolioRepository) Create(ctx context.Context, portfolio *models.Portfolio) error {
	if portfolio.ID.IsZero() {
		portfolio.ID = primitive.NewObjectID()
	}
	portfolio.CreatedAt = time.Now()
	portfolio.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, portfolio)
	if err != nil {
		// Handle duplicate key error gracefully (race condition in concurrent requests)
		if mongo.IsDuplicateKeyError(err) {
			return nil // Portfolio already exists, treat as success
		}
		return fmt.Errorf("failed to create portfolio: %w", err)
	}

	return nil
}

// GetByID retrieves a portfolio by its ID
func (r *MongoPortfolioRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Portfolio, error) {
	var portfolio models.Portfolio
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&portfolio)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("portfolio not found")
		}
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	return &portfolio, nil
}

// GetByUserID retrieves a portfolio by user ID
func (r *MongoPortfolioRepository) GetByUserID(ctx context.Context, userID int64) (*models.Portfolio, error) {
	var portfolio models.Portfolio
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&portfolio)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("portfolio not found for user %d", userID)
		}
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	return &portfolio, nil
}

// Update updates an existing portfolio
func (r *MongoPortfolioRepository) Update(ctx context.Context, portfolio *models.Portfolio) error {
	portfolio.UpdatedAt = time.Now()

	filter := bson.M{"_id": portfolio.ID}
	update := bson.M{"$set": portfolio}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update portfolio: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("portfolio not found")
	}

	return nil
}

// Delete deletes a portfolio by ID
func (r *MongoPortfolioRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete portfolio: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("portfolio not found")
	}

	return nil
}

// DeleteByUserID deletes a portfolio by user ID
func (r *MongoPortfolioRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"user_id": userID})
	if err != nil {
		return fmt.Errorf("failed to delete portfolio: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("portfolio not found for user %d", userID)
	}

	return nil
}

// List retrieves portfolios with pagination
func (r *MongoPortfolioRepository) List(ctx context.Context, limit, offset int) ([]*models.Portfolio, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "updated_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list portfolios: %w", err)
	}
	defer cursor.Close(ctx)

	var portfolios []*models.Portfolio
	if err := cursor.All(ctx, &portfolios); err != nil {
		return nil, fmt.Errorf("failed to decode portfolios: %w", err)
	}

	return portfolios, nil
}

// GetNeedingRecalculation retrieves portfolios that need recalculation
func (r *MongoPortfolioRepository) GetNeedingRecalculation(ctx context.Context, limit int) ([]*models.Portfolio, error) {
	filter := bson.M{"metadata.needs_recalculation": true}
	opts := options.Find().SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolios needing recalculation: %w", err)
	}
	defer cursor.Close(ctx)

	var portfolios []*models.Portfolio
	if err := cursor.All(ctx, &portfolios); err != nil {
		return nil, fmt.Errorf("failed to decode portfolios: %w", err)
	}

	return portfolios, nil
}

// GetByUserIDs retrieves portfolios for multiple users
func (r *MongoPortfolioRepository) GetByUserIDs(ctx context.Context, userIDs []int64) ([]*models.Portfolio, error) {
	filter := bson.M{"user_id": bson.M{"$in": userIDs}}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolios: %w", err)
	}
	defer cursor.Close(ctx)

	var portfolios []*models.Portfolio
	if err := cursor.All(ctx, &portfolios); err != nil {
		return nil, fmt.Errorf("failed to decode portfolios: %w", err)
	}

	return portfolios, nil
}

// GetTopPerformers retrieves top performing portfolios
func (r *MongoPortfolioRepository) GetTopPerformers(ctx context.Context, limit int, period string) ([]*models.Portfolio, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "profit_loss_percentage", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get top performers: %w", err)
	}
	defer cursor.Close(ctx)

	var portfolios []*models.Portfolio
	if err := cursor.All(ctx, &portfolios); err != nil {
		return nil, fmt.Errorf("failed to decode portfolios: %w", err)
	}

	return portfolios, nil
}

// GetPortfolioStats retrieves portfolio statistics
func (r *MongoPortfolioRepository) GetPortfolioStats(ctx context.Context) (*repositories.PortfolioStats, error) {
	// Simplified implementation - would use aggregation pipeline in production
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count portfolios: %w", err)
	}

	return &repositories.PortfolioStats{
		TotalPortfolios: count,
	}, nil
}

// UpdateMetadata updates only the metadata field
func (r *MongoPortfolioRepository) UpdateMetadata(ctx context.Context, userID int64, metadata map[string]interface{}) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"metadata":   metadata,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("portfolio not found for user %d", userID)
	}

	return nil
}

// BulkUpdate updates multiple portfolios
func (r *MongoPortfolioRepository) BulkUpdate(ctx context.Context, portfolios []*models.Portfolio) error {
	if len(portfolios) == 0 {
		return nil
	}

	var writes []mongo.WriteModel
	for _, portfolio := range portfolios {
		portfolio.UpdatedAt = time.Now()
		filter := bson.M{"_id": portfolio.ID}
		update := bson.M{"$set": portfolio}
		writes = append(writes, mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update))
	}

	_, err := r.collection.BulkWrite(ctx, writes)
	if err != nil {
		return fmt.Errorf("failed to bulk update portfolios: %w", err)
	}

	return nil
}
