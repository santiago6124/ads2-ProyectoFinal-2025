package mongodb

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
	"portfolio-api/pkg/database"
)

type portfolioRepository struct {
	db         *database.MongoDB
	collection *mongo.Collection
}

// NewPortfolioRepository creates a new MongoDB portfolio repository
func NewPortfolioRepository(db *database.MongoDB) repositories.PortfolioRepository {
	return &portfolioRepository{
		db:         db,
		collection: db.Collection("portfolios"),
	}
}

func (r *portfolioRepository) Create(ctx context.Context, portfolio *models.Portfolio) error {
	if portfolio.ID.IsZero() {
		portfolio.ID = primitive.NewObjectID()
	}

	portfolio.CreatedAt = time.Now()
	portfolio.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, portfolio)
	if err != nil {
		return fmt.Errorf("failed to create portfolio: %w", err)
	}

	return nil
}

func (r *portfolioRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Portfolio, error) {
	var portfolio models.Portfolio
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&portfolio)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get portfolio by ID: %w", err)
	}

	return &portfolio, nil
}

func (r *portfolioRepository) GetByUserID(ctx context.Context, userID int64) (*models.Portfolio, error) {
	var portfolio models.Portfolio
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&portfolio)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get portfolio by user ID: %w", err)
	}

	return &portfolio, nil
}

func (r *portfolioRepository) Update(ctx context.Context, portfolio *models.Portfolio) error {
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

func (r *portfolioRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete portfolio: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("portfolio not found")
	}

	return nil
}

func (r *portfolioRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"user_id": userID})
	if err != nil {
		return fmt.Errorf("failed to delete portfolio by user ID: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("portfolio not found for user %d", userID)
	}

	return nil
}

func (r *portfolioRepository) List(ctx context.Context, limit, offset int) ([]*models.Portfolio, error) {
	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}
	opts.SetSort(bson.D{{Key: "updated_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list portfolios: %w", err)
	}
	defer cursor.Close(ctx)

	var portfolios []*models.Portfolio
	for cursor.Next(ctx) {
		var portfolio models.Portfolio
		if err := cursor.Decode(&portfolio); err != nil {
			return nil, fmt.Errorf("failed to decode portfolio: %w", err)
		}
		portfolios = append(portfolios, &portfolio)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return portfolios, nil
}

func (r *portfolioRepository) GetNeedingRecalculation(ctx context.Context, limit int) ([]*models.Portfolio, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"metadata.needs_recalculation": true},
			{"metadata.last_calculated": bson.M{"$lt": time.Now().Add(-24 * time.Hour)}},
		},
	}

	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	opts.SetSort(bson.D{{Key: "metadata.last_calculated", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolios needing recalculation: %w", err)
	}
	defer cursor.Close(ctx)

	var portfolios []*models.Portfolio
	for cursor.Next(ctx) {
		var portfolio models.Portfolio
		if err := cursor.Decode(&portfolio); err != nil {
			return nil, fmt.Errorf("failed to decode portfolio: %w", err)
		}
		portfolios = append(portfolios, &portfolio)
	}

	return portfolios, nil
}

func (r *portfolioRepository) GetByUserIDs(ctx context.Context, userIDs []int64) ([]*models.Portfolio, error) {
	filter := bson.M{"user_id": bson.M{"$in": userIDs}}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolios by user IDs: %w", err)
	}
	defer cursor.Close(ctx)

	var portfolios []*models.Portfolio
	for cursor.Next(ctx) {
		var portfolio models.Portfolio
		if err := cursor.Decode(&portfolio); err != nil {
			return nil, fmt.Errorf("failed to decode portfolio: %w", err)
		}
		portfolios = append(portfolios, &portfolio)
	}

	return portfolios, nil
}

func (r *portfolioRepository) GetTopPerformers(ctx context.Context, limit int, period string) ([]*models.Portfolio, error) {
	var sortField string
	switch period {
	case "daily":
		sortField = "performance.daily_change_percentage"
	case "weekly":
		sortField = "performance.weekly_change_percentage"
	case "monthly":
		sortField = "performance.monthly_change_percentage"
	case "yearly":
		sortField = "performance.yearly_change_percentage"
	default:
		sortField = "profit_loss_percentage"
	}

	opts := options.Find()
	opts.SetSort(bson.D{{Key: sortField, Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get top performers: %w", err)
	}
	defer cursor.Close(ctx)

	var portfolios []*models.Portfolio
	for cursor.Next(ctx) {
		var portfolio models.Portfolio
		if err := cursor.Decode(&portfolio); err != nil {
			return nil, fmt.Errorf("failed to decode portfolio: %w", err)
		}
		portfolios = append(portfolios, &portfolio)
	}

	return portfolios, nil
}

func (r *portfolioRepository) GetPortfolioStats(ctx context.Context) (*repositories.PortfolioStats, error) {
	// Aggregate pipeline for portfolio statistics
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":             nil,
				"total_count":     bson.M{"$sum": 1},
				"total_value":     bson.M{"$sum": "$total_value"},
				"total_holdings":  bson.M{"$sum": bson.M{"$size": "$holdings"}},
				"positive_pnl":    bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$gt": []interface{}{"$profit_loss", 0}}, 1, 0}}},
				"negative_pnl":    bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$lt": []interface{}{"$profit_loss", 0}}, 1, 0}}},
				"neutral_pnl":     bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$profit_loss", 0}}, 1, 0}}},
				"high_growth":     bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$gt": []interface{}{"$profit_loss_percentage", 0.2}}, 1, 0}}},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate portfolio stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalCount    int64   `bson:"total_count"`
		TotalValue    float64 `bson:"total_value"`
		TotalHoldings int64   `bson:"total_holdings"`
		PositivePnL   int64   `bson:"positive_pnl"`
		NegativePnL   int64   `bson:"negative_pnl"`
		NeutralPnL    int64   `bson:"neutral_pnl"`
		HighGrowth    int64   `bson:"high_growth"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode stats result: %w", err)
		}
	}

	stats := &repositories.PortfolioStats{
		TotalPortfolios: result.TotalCount,
		TotalValue:      result.TotalValue,
		TotalHoldings:   result.TotalHoldings,
		PerformanceRanges: repositories.PerformanceRanges{
			Positive:   result.PositivePnL,
			Negative:   result.NegativePnL,
			Neutral:    result.NeutralPnL,
			HighGrowth: result.HighGrowth,
		},
	}

	if result.TotalCount > 0 {
		stats.AverageValue = result.TotalValue / float64(result.TotalCount)
		stats.AverageHoldings = float64(result.TotalHoldings) / float64(result.TotalCount)
	}

	// Get top sectors
	topSectors, err := r.getTopSectors(ctx)
	if err == nil {
		stats.TopSectors = topSectors
	}

	return stats, nil
}

func (r *portfolioRepository) getTopSectors(ctx context.Context) ([]repositories.SectorStat, error) {
	pipeline := []bson.M{
		{"$unwind": "$holdings"},
		{
			"$group": bson.M{
				"_id":   "$holdings.category",
				"count": bson.M{"$sum": 1},
			},
		},
		{"$sort": bson.M{"count": -1}},
		{"$limit": 10},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get top sectors: %w", err)
	}
	defer cursor.Close(ctx)

	var sectors []repositories.SectorStat
	totalCount := int64(0)

	for cursor.Next(ctx) {
		var result struct {
			Sector string `bson:"_id"`
			Count  int64  `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			continue
		}

		sectors = append(sectors, repositories.SectorStat{
			Sector: result.Sector,
			Count:  result.Count,
		})
		totalCount += result.Count
	}

	// Calculate percentages
	for i := range sectors {
		if totalCount > 0 {
			sectors[i].Percentage = float64(sectors[i].Count) / float64(totalCount) * 100
		}
	}

	return sectors, nil
}

func (r *portfolioRepository) UpdateMetadata(ctx context.Context, userID int64, metadata map[string]interface{}) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"metadata":   metadata,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update portfolio metadata: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("portfolio not found for user %d", userID)
	}

	return nil
}

func (r *portfolioRepository) BulkUpdate(ctx context.Context, portfolios []*models.Portfolio) error {
	if len(portfolios) == 0 {
		return nil
	}

	var operations []mongo.WriteModel

	for _, portfolio := range portfolios {
		portfolio.UpdatedAt = time.Now()

		filter := bson.M{"_id": portfolio.ID}
		update := bson.M{"$set": portfolio}

		operation := mongo.NewUpdateOneModel()
		operation.SetFilter(filter)
		operation.SetUpdate(update)
		operations = append(operations, operation)
	}

	opts := options.BulkWrite().SetOrdered(false)
	result, err := r.collection.BulkWrite(ctx, operations, opts)
	if err != nil {
		return fmt.Errorf("failed to bulk update portfolios: %w", err)
	}

	if result.ModifiedCount != int64(len(portfolios)) {
		return fmt.Errorf("not all portfolios were updated: expected %d, modified %d", len(portfolios), result.ModifiedCount)
	}

	return nil
}