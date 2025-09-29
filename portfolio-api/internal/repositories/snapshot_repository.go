package repositories

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"portfolio-api/internal/models"
)

// SnapshotRepository defines the interface for snapshot data operations
type SnapshotRepository interface {
	// Create creates a new snapshot
	Create(ctx context.Context, snapshot *models.Snapshot) error

	// GetByID retrieves a snapshot by its ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Snapshot, error)

	// GetByUserID retrieves snapshots for a user with pagination
	GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.Snapshot, error)

	// GetByInterval retrieves snapshots by interval type
	GetByInterval(ctx context.Context, userID int64, interval string, limit, offset int) ([]models.Snapshot, error)

	// GetByDateRange retrieves snapshots within a date range
	GetByDateRange(ctx context.Context, userID int64, startDate, endDate time.Time) ([]models.Snapshot, error)

	// GetByTags retrieves snapshots with specific tags
	GetByTags(ctx context.Context, userID int64, tags []string) ([]models.Snapshot, error)

	// GetLatest retrieves the latest snapshot for a user
	GetLatest(ctx context.Context, userID int64) (*models.Snapshot, error)

	// GetLatestByInterval retrieves the latest snapshot for each interval type
	GetLatestByInterval(ctx context.Context, userID int64) (map[string]*models.Snapshot, error)

	// Update updates an existing snapshot
	Update(ctx context.Context, snapshot *models.Snapshot) error

	// Delete deletes a snapshot by ID
	Delete(ctx context.Context, id primitive.ObjectID) error

	// DeleteByUserID deletes all snapshots for a user
	DeleteByUserID(ctx context.Context, userID int64) error

	// DeleteOldSnapshots deletes snapshots older than specified duration
	DeleteOldSnapshots(ctx context.Context, olderThan time.Duration) (int64, error)

	// GetPortfolioHistory retrieves portfolio history with filters
	GetPortfolioHistory(ctx context.Context, filter *models.SnapshotFilter) ([]models.Snapshot, error)

	// BulkCreate creates multiple snapshots
	BulkCreate(ctx context.Context, snapshots []models.Snapshot) error

	// GetAggregatedData retrieves aggregated snapshot data
	GetAggregatedData(ctx context.Context, userID int64, groupBy string, startDate, endDate time.Time) ([]AggregatedSnapshot, error)

	// GetPerformanceMetrics calculates performance metrics from snapshots
	GetPerformanceMetrics(ctx context.Context, userID int64, period string) (*PerformanceMetrics, error)

	// GetVolatilityData retrieves volatility data for analysis
	GetVolatilityData(ctx context.Context, userID int64, days int) ([]VolatilityPoint, error)

	// GetCorrelationData retrieves data for correlation analysis
	GetCorrelationData(ctx context.Context, userIDs []int64, startDate, endDate time.Time) (map[int64][]models.Snapshot, error)
}

// AggregatedSnapshot represents aggregated snapshot data
type AggregatedSnapshot struct {
	Date         time.Time `json:"date"`
	TotalValue   float64   `json:"total_value"`
	ProfitLoss   float64   `json:"profit_loss"`
	DailyChange  float64   `json:"daily_change"`
	Count        int       `json:"count"`
}

// PerformanceMetrics represents calculated performance metrics
type PerformanceMetrics struct {
	TotalReturn      float64 `json:"total_return"`
	AnnualizedReturn float64 `json:"annualized_return"`
	Volatility       float64 `json:"volatility"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	WinRate          float64 `json:"win_rate"`
	BestDay          float64 `json:"best_day"`
	WorstDay         float64 `json:"worst_day"`
}

// VolatilityPoint represents a point in volatility analysis
type VolatilityPoint struct {
	Date       time.Time `json:"date"`
	Value      float64   `json:"value"`
	Return     float64   `json:"return"`
	Volatility float64   `json:"volatility"`
}