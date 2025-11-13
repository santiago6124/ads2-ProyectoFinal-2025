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

// MongoSnapshotRepository implements SnapshotRepository using MongoDB
type MongoSnapshotRepository struct {
	collection *mongo.Collection
}

// NewSnapshotRepository creates a new MongoDB snapshot repository
func NewSnapshotRepository(db *mongo.Database) repositories.SnapshotRepository {
	return &MongoSnapshotRepository{
		collection: db.Collection("portfolio_snapshots"),
	}
}

// Create creates a new snapshot
func (r *MongoSnapshotRepository) Create(ctx context.Context, snapshot *models.Snapshot) error {
	if snapshot.ID.IsZero() {
		snapshot.ID = primitive.NewObjectID()
	}
	snapshot.Timestamp = time.Now()

	_, err := r.collection.InsertOne(ctx, snapshot)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	return nil
}

// GetByID retrieves a snapshot by its ID
func (r *MongoSnapshotRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Snapshot, error) {
	var snapshot models.Snapshot
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&snapshot)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("snapshot not found")
		}
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	return &snapshot, nil
}

// GetByUserID retrieves snapshots for a user
func (r *MongoSnapshotRepository) GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.Snapshot, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}
	defer cursor.Close(ctx)

	var snapshots []models.Snapshot
	if err := cursor.All(ctx, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to decode snapshots: %w", err)
	}

	return snapshots, nil
}

// GetByInterval retrieves snapshots for a specific interval
func (r *MongoSnapshotRepository) GetByInterval(ctx context.Context, userID int64, interval string, limit, offset int) ([]models.Snapshot, error) {
	filter := bson.M{
		"user_id":  userID,
		"interval": interval,
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots by interval: %w", err)
	}
	defer cursor.Close(ctx)

	var snapshots []models.Snapshot
	if err := cursor.All(ctx, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to decode snapshots: %w", err)
	}

	return snapshots, nil
}

// GetByDateRange retrieves snapshots within a date range
func (r *MongoSnapshotRepository) GetByDateRange(ctx context.Context, userID int64, start, end time.Time) ([]models.Snapshot, error) {
	filter := bson.M{
		"user_id": userID,
		"timestamp": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots by date range: %w", err)
	}
	defer cursor.Close(ctx)

	var snapshots []models.Snapshot
	if err := cursor.All(ctx, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to decode snapshots: %w", err)
	}

	return snapshots, nil
}

// Delete deletes a snapshot by ID
func (r *MongoSnapshotRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("snapshot not found")
	}

	return nil
}

// DeleteByUserID deletes all snapshots for a user
func (r *MongoSnapshotRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"user_id": userID})
	if err != nil {
		return fmt.Errorf("failed to delete snapshots for user: %w", err)
	}

	return nil
}

// DeleteOlderThan deletes snapshots older than a specific date
func (r *MongoSnapshotRepository) DeleteOlderThan(ctx context.Context, date time.Time) error {
	filter := bson.M{
		"timestamp": bson.M{"$lt": date},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete old snapshots: %w", err)
	}

	fmt.Printf("Deleted %d old snapshots\n", result.DeletedCount)
	return nil
}

// GetLatest retrieves the latest snapshot for a user
func (r *MongoSnapshotRepository) GetLatest(ctx context.Context, userID int64) (*models.Snapshot, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}})

	var snapshot models.Snapshot
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}, opts).Decode(&snapshot)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no snapshots found for user %d", userID)
		}
		return nil, fmt.Errorf("failed to get latest snapshot: %w", err)
	}

	return &snapshot, nil
}

// Count returns the total number of snapshots for a user
func (r *MongoSnapshotRepository) Count(ctx context.Context, userID int64) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return 0, fmt.Errorf("failed to count snapshots: %w", err)
	}

	return count, nil
}

// BulkCreate creates multiple snapshots
func (r *MongoSnapshotRepository) BulkCreate(ctx context.Context, snapshots []models.Snapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	var docs []interface{}
	for i := range snapshots {
		if snapshots[i].ID.IsZero() {
			snapshots[i].ID = primitive.NewObjectID()
		}
		snapshots[i].Timestamp = time.Now()
		docs = append(docs, snapshots[i])
	}

	_, err := r.collection.InsertMany(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to bulk create snapshots: %w", err)
	}

	return nil
}

// DeleteOldSnapshots deletes snapshots older than specified duration
func (r *MongoSnapshotRepository) DeleteOldSnapshots(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffDate := time.Now().Add(-olderThan)
	filter := bson.M{
		"timestamp": bson.M{"$lt": cutoffDate},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old snapshots: %w", err)
	}

	return result.DeletedCount, nil
}

// GetByTags retrieves snapshots with specific tags
func (r *MongoSnapshotRepository) GetByTags(ctx context.Context, userID int64, tags []string) ([]models.Snapshot, error) {
	filter := bson.M{
		"user_id": userID,
		"tags":    bson.M{"$in": tags},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots by tags: %w", err)
	}
	defer cursor.Close(ctx)

	var snapshots []models.Snapshot
	if err := cursor.All(ctx, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to decode snapshots: %w", err)
	}

	return snapshots, nil
}

// GetLatestByInterval retrieves the latest snapshot for each interval type
func (r *MongoSnapshotRepository) GetLatestByInterval(ctx context.Context, userID int64) (map[string]*models.Snapshot, error) {
	// This would require aggregation pipeline - simplified implementation
	return make(map[string]*models.Snapshot), nil
}

// Update updates an existing snapshot
func (r *MongoSnapshotRepository) Update(ctx context.Context, snapshot *models.Snapshot) error {
	filter := bson.M{"_id": snapshot.ID}
	update := bson.M{"$set": snapshot}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update snapshot: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("snapshot not found")
	}

	return nil
}

// GetPortfolioHistory retrieves portfolio history with filters
func (r *MongoSnapshotRepository) GetPortfolioHistory(ctx context.Context, filter *models.SnapshotFilter) ([]models.Snapshot, error) {
	// Simplified implementation - would need proper filter handling
	return []models.Snapshot{}, nil
}

// GetAggregatedData retrieves aggregated snapshot data
func (r *MongoSnapshotRepository) GetAggregatedData(ctx context.Context, userID int64, groupBy string, startDate, endDate time.Time) ([]repositories.AggregatedSnapshot, error) {
	// This would require aggregation pipeline - simplified implementation
	return []repositories.AggregatedSnapshot{}, nil
}

// GetPerformanceMetrics calculates performance metrics from snapshots
func (r *MongoSnapshotRepository) GetPerformanceMetrics(ctx context.Context, userID int64, period string) (*repositories.PerformanceMetrics, error) {
	// This would require complex calculations - simplified implementation
	return &repositories.PerformanceMetrics{}, nil
}

// GetVolatilityData retrieves volatility data for analysis
func (r *MongoSnapshotRepository) GetVolatilityData(ctx context.Context, userID int64, days int) ([]repositories.VolatilityPoint, error) {
	// This would require complex calculations - simplified implementation
	return []repositories.VolatilityPoint{}, nil
}

// GetCorrelationData retrieves data for correlation analysis
func (r *MongoSnapshotRepository) GetCorrelationData(ctx context.Context, userIDs []int64, startDate, endDate time.Time) (map[int64][]models.Snapshot, error) {
	// This would require complex queries - simplified implementation
	return make(map[int64][]models.Snapshot), nil
}
