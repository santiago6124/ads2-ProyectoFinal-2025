package repositories

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"portfolio-api/internal/models"
)

// PortfolioRepository defines the interface for portfolio data operations
type PortfolioRepository interface {
	// Create creates a new portfolio
	Create(ctx context.Context, portfolio *models.Portfolio) error

	// GetByID retrieves a portfolio by its ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Portfolio, error)

	// GetByUserID retrieves a portfolio by user ID
	GetByUserID(ctx context.Context, userID int64) (*models.Portfolio, error)

	// Update updates an existing portfolio
	Update(ctx context.Context, portfolio *models.Portfolio) error

	// Delete deletes a portfolio by ID
	Delete(ctx context.Context, id primitive.ObjectID) error

	// DeleteByUserID deletes a portfolio by user ID
	DeleteByUserID(ctx context.Context, userID int64) error

	// List retrieves portfolios with pagination
	List(ctx context.Context, limit, offset int) ([]*models.Portfolio, error)

	// GetNeedingRecalculation retrieves portfolios that need recalculation
	GetNeedingRecalculation(ctx context.Context, limit int) ([]*models.Portfolio, error)

	// GetByUserIDs retrieves portfolios for multiple users
	GetByUserIDs(ctx context.Context, userIDs []int64) ([]*models.Portfolio, error)

	// GetTopPerformers retrieves top performing portfolios
	GetTopPerformers(ctx context.Context, limit int, period string) ([]*models.Portfolio, error)

	// GetPortfolioStats retrieves portfolio statistics
	GetPortfolioStats(ctx context.Context) (*PortfolioStats, error)

	// UpdateMetadata updates only the metadata field
	UpdateMetadata(ctx context.Context, userID int64, metadata map[string]interface{}) error

	// BulkUpdate updates multiple portfolios
	BulkUpdate(ctx context.Context, portfolios []*models.Portfolio) error
}

// PortfolioStats represents portfolio statistics
type PortfolioStats struct {
	TotalPortfolios    int64   `json:"total_portfolios"`
	TotalValue         float64 `json:"total_value"`
	AverageValue       float64 `json:"average_value"`
	TotalHoldings      int64   `json:"total_holdings"`
	AverageHoldings    float64 `json:"average_holdings"`
	TopSectors         []SectorStat `json:"top_sectors"`
	PerformanceRanges  PerformanceRanges `json:"performance_ranges"`
}

// SectorStat represents sector statistics
type SectorStat struct {
	Sector     string  `json:"sector"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// PerformanceRanges represents performance distribution
type PerformanceRanges struct {
	Positive   int64 `json:"positive"`
	Negative   int64 `json:"negative"`
	Neutral    int64 `json:"neutral"`
	HighGrowth int64 `json:"high_growth"`
}