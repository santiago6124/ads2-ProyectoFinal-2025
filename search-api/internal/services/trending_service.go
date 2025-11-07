package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"search-api/internal/models"
	"search-api/internal/repositories"
)

// TrendingService handles trending detection and scoring
type TrendingService struct {
	searchRepo    repositories.SearchRepository
	trendingData  map[string]*TrendingData
	mu            sync.RWMutex
	logger        *logrus.Logger
	updateTicker  *time.Ticker
	stopChan      chan struct{}
}

// TrendingData represents trending information for a cryptocurrency
type TrendingData struct {
	ID              string
	Symbol          string
	Name            string
	BaseScore       float32
	VolumeScore     float32
	SearchScore     float32
	MentionScore    float32
	PriceScore      float32
	FinalScore      float32
	SearchCount     int64
	VolumeIncrease  float64
	PriceChange24h  float64
	MentionsCount   int64
	LastUpdated     time.Time
	Rank            int
}

// TrendingConfig represents trending service configuration
type TrendingConfig struct {
	UpdateInterval      time.Duration
	MaxTrendingItems    int
	ScoreDecayRate      float32
	VolumeWeight        float32
	SearchWeight        float32
	MentionWeight       float32
	PriceWeight         float32
	MinSearchThreshold  int64
}

// NewTrendingService creates a new trending service
func NewTrendingService(searchRepo repositories.SearchRepository, config *TrendingConfig, logger *logrus.Logger) *TrendingService {
	if config == nil {
		config = DefaultTrendingConfig()
	}

	return &TrendingService{
		searchRepo:   searchRepo,
		trendingData: make(map[string]*TrendingData),
		logger:       logger,
		stopChan:     make(chan struct{}),
	}
}

// Start starts the trending service background processing
func (ts *TrendingService) Start(ctx context.Context) error {
	ts.logger.Info("Starting trending service")

	// Initial data load
	if err := ts.refreshTrendingData(ctx); err != nil {
		ts.logger.WithError(err).Error("Failed to load initial trending data")
		return fmt.Errorf("failed to start trending service: %w", err)
	}

	// Start periodic updates
	ts.updateTicker = time.NewTicker(5 * time.Minute) // Update every 5 minutes

	go ts.backgroundUpdate(ctx)

	ts.logger.Info("Trending service started successfully")
	return nil
}

// Stop stops the trending service
func (ts *TrendingService) Stop() error {
	ts.logger.Info("Stopping trending service")

	if ts.updateTicker != nil {
		ts.updateTicker.Stop()
	}

	close(ts.stopChan)

	ts.logger.Info("Trending service stopped")
	return nil
}

// UpdateTrendingScore updates the trending score for a cryptocurrency
func (ts *TrendingService) UpdateTrendingScore(cryptoID string, eventType string, value float64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	data, exists := ts.trendingData[cryptoID]
	if !exists {
		// Create new trending data entry
		data = &TrendingData{
			ID:          cryptoID,
			LastUpdated: time.Now(),
		}
		ts.trendingData[cryptoID] = data
	}

	// Update based on event type
	switch eventType {
	case "search":
		data.SearchCount++
		data.SearchScore = ts.calculateSearchScore(data.SearchCount)
	case "order_executed":
		data.VolumeIncrease += value
		data.VolumeScore = ts.calculateVolumeScore(data.VolumeIncrease)
	case "price_change":
		data.PriceChange24h = value
		data.PriceScore = ts.calculatePriceScore(value)
	case "mention":
		data.MentionsCount++
		data.MentionScore = ts.calculateMentionScore(data.MentionsCount)
	}

	// Recalculate final score
	data.FinalScore = ts.calculateFinalScore(data)
	data.LastUpdated = time.Now()

	// Update trending status in search index asynchronously
	go ts.updateSearchIndex(cryptoID, data.FinalScore)

	ts.logger.WithFields(logrus.Fields{
		"crypto_id":  cryptoID,
		"event_type": eventType,
		"score":      data.FinalScore,
	}).Debug("Trending score updated")
}

// GetTrendingScore returns the current trending score for a cryptocurrency
func (ts *TrendingService) GetTrendingScore(ctx context.Context, cryptoID string) (float32, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if data, exists := ts.trendingData[cryptoID]; exists {
		return data.FinalScore, true
	}
	return 0, false
}

// GetTopTrending returns the top trending cryptocurrencies
func (ts *TrendingService) GetTopTrending(limit int, period string) []models.TrendingCrypto {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	// Convert map to slice for sorting
	trending := make([]*TrendingData, 0, len(ts.trendingData))
	cutoff := ts.getPeriodCutoff(period)

	for _, data := range ts.trendingData {
		// Filter by period
		if data.LastUpdated.After(cutoff) && data.FinalScore > 10 {
			trending = append(trending, data)
		}
	}

	// Sort by final score (descending)
	sort.Slice(trending, func(i, j int) bool {
		return trending[i].FinalScore > trending[j].FinalScore
	})

	// Limit results
	if limit > 0 && limit < len(trending) {
		trending = trending[:limit]
	}

	// Convert to TrendingCrypto models
	results := make([]models.TrendingCrypto, len(trending))
	for i, data := range trending {
		results[i] = models.TrendingCrypto{
			Rank:                    i + 1,
			ID:                      data.ID,
			Symbol:                  data.Symbol,
			Name:                    data.Name,
			TrendingScore:           data.FinalScore,
			SearchVolumeIncrease:    fmt.Sprintf("%.0f%%", data.VolumeIncrease),
			MentionsCount:           data.MentionsCount,
		}
	}

	return results
}

// GetTrendingMetrics returns trending service metrics
func (ts *TrendingService) GetTrendingMetrics() map[string]interface{} {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	totalTrending := 0
	highScoreTrending := 0
	var avgScore float32

	for _, data := range ts.trendingData {
		if data.FinalScore > 10 {
			totalTrending++
		}
		if data.FinalScore > 50 {
			highScoreTrending++
		}
		avgScore += data.FinalScore
	}

	if len(ts.trendingData) > 0 {
		avgScore /= float32(len(ts.trendingData))
	}

	return map[string]interface{}{
		"total_tracked":       len(ts.trendingData),
		"total_trending":      totalTrending,
		"high_score_trending": highScoreTrending,
		"average_score":       avgScore,
	}
}

// Background processing methods

func (ts *TrendingService) backgroundUpdate(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ts.stopChan:
			return
		case <-ts.updateTicker.C:
			if err := ts.refreshTrendingData(ctx); err != nil {
				ts.logger.WithError(err).Error("Failed to refresh trending data")
			}
			ts.decayScores()
		}
	}
}

func (ts *TrendingService) refreshTrendingData(ctx context.Context) error {
	// This would typically fetch data from external sources
	// For now, we'll focus on internal score management

	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Update rankings
	ts.updateRankings()

	ts.logger.WithField("tracked_items", len(ts.trendingData)).Debug("Trending data refreshed")
	return nil
}

func (ts *TrendingService) decayScores() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	now := time.Now()
	decayRate := float32(0.95) // 5% decay per update cycle

	for _, data := range ts.trendingData {
		// Apply time-based decay to prevent stale trending items
		age := now.Sub(data.LastUpdated)
		if age > time.Hour {
			data.SearchScore *= decayRate
			data.VolumeScore *= decayRate
			data.MentionScore *= decayRate
			data.FinalScore = ts.calculateFinalScore(data)
		}
	}

	ts.logger.Debug("Applied score decay to trending data")
}

func (ts *TrendingService) updateRankings() {
	// Convert to slice for sorting
	trending := make([]*TrendingData, 0, len(ts.trendingData))
	for _, data := range ts.trendingData {
		trending = append(trending, data)
	}

	// Sort by final score
	sort.Slice(trending, func(i, j int) bool {
		return trending[i].FinalScore > trending[j].FinalScore
	})

	// Update ranks
	for i, data := range trending {
		data.Rank = i + 1
	}
}

// Score calculation methods

func (ts *TrendingService) calculateSearchScore(searchCount int64) float32 {
	if searchCount == 0 {
		return 0
	}

	// Logarithmic scaling for search count
	return float32(math.Log(float64(searchCount)+1) * 10)
}

func (ts *TrendingService) calculateVolumeScore(volumeIncrease float64) float32 {
	if volumeIncrease <= 0 {
		return 0
	}

	// Cap the volume score to prevent extreme values
	score := math.Min(volumeIncrease, 1000) / 10
	return float32(score)
}

func (ts *TrendingService) calculatePriceScore(priceChange float64) float32 {
	// High price changes (positive or negative) increase trending score
	absChange := math.Abs(priceChange)

	// Use square root to dampen extreme values
	return float32(math.Sqrt(absChange) * 5)
}

func (ts *TrendingService) calculateMentionScore(mentionCount int64) float32 {
	if mentionCount == 0 {
		return 0
	}

	// Logarithmic scaling for mention count
	return float32(math.Log(float64(mentionCount)+1) * 5)
}

func (ts *TrendingService) calculateFinalScore(data *TrendingData) float32 {
	config := DefaultTrendingConfig()

	score := data.SearchScore*config.SearchWeight +
			data.VolumeScore*config.VolumeWeight +
			data.PriceScore*config.PriceWeight +
			data.MentionScore*config.MentionWeight

	// Apply time decay
	age := time.Since(data.LastUpdated)
	if age > time.Hour {
		decayFactor := math.Exp(-float64(age.Hours()) / 24) // Decay over 24 hours
		score = float32(float64(score) * decayFactor)
	}

	return score
}

func (ts *TrendingService) updateSearchIndex(cryptoID string, score float32) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ts.searchRepo.UpdateTrendingScore(ctx, cryptoID, score); err != nil {
		ts.logger.WithFields(logrus.Fields{
			"crypto_id": cryptoID,
			"score":     score,
			"error":     err,
		}).Error("Failed to update trending score in search index")
	}
}

func (ts *TrendingService) getPeriodCutoff(period string) time.Time {
	now := time.Now()

	switch period {
	case "1h":
		return now.Add(-1 * time.Hour)
	case "24h":
		return now.Add(-24 * time.Hour)
	case "7d":
		return now.Add(-7 * 24 * time.Hour)
	case "30d":
		return now.Add(-30 * 24 * time.Hour)
	default:
		return now.Add(-24 * time.Hour)
	}
}

// DefaultTrendingConfig returns default trending service configuration
func DefaultTrendingConfig() *TrendingConfig {
	return &TrendingConfig{
		UpdateInterval:      5 * time.Minute,
		MaxTrendingItems:    100,
		ScoreDecayRate:      0.95,
		VolumeWeight:        0.3,
		SearchWeight:        0.4,
		MentionWeight:       0.2,
		PriceWeight:         0.1,
		MinSearchThreshold:  5,
	}
}

// TrendingEventHandler handles trending events from external sources
type TrendingEventHandler struct {
	trendingService *TrendingService
	logger          *logrus.Logger
}

// NewTrendingEventHandler creates a new trending event handler
func NewTrendingEventHandler(trendingService *TrendingService, logger *logrus.Logger) *TrendingEventHandler {
	return &TrendingEventHandler{
		trendingService: trendingService,
		logger:          logger,
	}
}

// HandleOrderEvent handles order execution events
func (teh *TrendingEventHandler) HandleOrderEvent(cryptoID string, orderValue float64) {
	// Calculate volume increase factor
	volumeIncrease := orderValue / 1000000 // Normalize by 1M

	teh.trendingService.UpdateTrendingScore(cryptoID, "order_executed", volumeIncrease)

	teh.logger.WithFields(logrus.Fields{
		"crypto_id":       cryptoID,
		"order_value":     orderValue,
		"volume_increase": volumeIncrease,
	}).Debug("Processed order event for trending")
}

// HandleSearchEvent handles search query events
func (teh *TrendingEventHandler) HandleSearchEvent(cryptoID string) {
	teh.trendingService.UpdateTrendingScore(cryptoID, "search", 1)

	teh.logger.WithField("crypto_id", cryptoID).Debug("Processed search event for trending")
}

// HandlePriceChangeEvent handles price change events
func (teh *TrendingEventHandler) HandlePriceChangeEvent(cryptoID string, priceChange float64) {
	teh.trendingService.UpdateTrendingScore(cryptoID, "price_change", priceChange)

	teh.logger.WithFields(logrus.Fields{
		"crypto_id":    cryptoID,
		"price_change": priceChange,
	}).Debug("Processed price change event for trending")
}