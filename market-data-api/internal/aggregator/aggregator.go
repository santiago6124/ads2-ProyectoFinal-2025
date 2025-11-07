package aggregator

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"market-data-api/internal/models"
	"market-data-api/internal/providers"
	"market-data-api/internal/types"
)

// PriceAggregator handles price aggregation from multiple providers
type PriceAggregator struct {
	providerManager *providers.ProviderManager
	config          *Config
	mu              sync.RWMutex

	// Caching
	priceCache     map[string]*CachedPrice
	cacheTTL       time.Duration
	cleanupTicker  *time.Ticker

	// Statistics
	stats          *AggregatorStats
	outlierDetector *OutlierDetector
}

// Config represents aggregator configuration
type Config struct {
	// Aggregation strategy
	Strategy               string        `json:"strategy"`                // weighted_average, median, best_price
	OutlierDetectionMethod string        `json:"outlier_detection"`       // z_score, iqr, isolation_forest
	OutlierThreshold       float64       `json:"outlier_threshold"`       // threshold for outlier detection

	// Weighting
	MinProviders          int           `json:"min_providers"`            // minimum providers for aggregation
	MaxProviders          int           `json:"max_providers"`            // maximum providers to use
	WeightByLatency       bool          `json:"weight_by_latency"`        // weight by response time
	WeightByReliability   bool          `json:"weight_by_reliability"`    // weight by historical accuracy
	WeightDecayFactor     float64       `json:"weight_decay_factor"`      // decay factor for temporal weighting

	// Quality control
	MaxPriceDeviation     decimal.Decimal `json:"max_price_deviation"`    // max deviation from median (%)
	MinConfidenceScore    float64         `json:"min_confidence_score"`   // minimum confidence for result
	RequireQuorum         bool            `json:"require_quorum"`         // require majority agreement

	// Caching
	CacheTTL              time.Duration   `json:"cache_ttl"`              // cache time-to-live
	EnableCaching         bool            `json:"enable_caching"`         // enable result caching

	// Concurrent processing
	MaxConcurrency        int             `json:"max_concurrency"`        // max concurrent provider requests
	RequestTimeout        time.Duration   `json:"request_timeout"`        // timeout per provider request

	// Fallback behavior
	FallbackStrategy      string          `json:"fallback_strategy"`      // single_provider, cached, error
	FallbackProvider      string          `json:"fallback_provider"`      // preferred fallback provider
}

// CachedPrice represents a cached aggregated price
type CachedPrice struct {
	Price     *models.AggregatedPrice `json:"price"`
	Timestamp time.Time               `json:"timestamp"`
	TTL       time.Duration           `json:"ttl"`
}

// AggregatorStats tracks aggregation statistics
type AggregatorStats struct {
	TotalRequests       int64             `json:"total_requests"`
	SuccessfulRequests  int64             `json:"successful_requests"`
	FailedRequests      int64             `json:"failed_requests"`
	CacheHits           int64             `json:"cache_hits"`
	CacheMisses         int64             `json:"cache_misses"`
	AverageLatency      time.Duration     `json:"average_latency"`
	AverageConfidence   float64           `json:"average_confidence"`
	OutliersDetected    int64             `json:"outliers_detected"`
	ProviderStats       map[string]*ProviderAggregatorStats `json:"provider_stats"`
	LastUpdated         time.Time         `json:"last_updated"`
	mu                  sync.RWMutex
}

// ProviderAggregatorStats tracks per-provider aggregation statistics
type ProviderAggregatorStats struct {
	RequestCount    int64         `json:"request_count"`
	SuccessCount    int64         `json:"success_count"`
	ErrorCount      int64         `json:"error_count"`
	AverageLatency  time.Duration `json:"average_latency"`
	OutlierCount    int64         `json:"outlier_count"`
	ReliabilityScore float64      `json:"reliability_score"`
	LastUsed        time.Time     `json:"last_used"`
}

// NewPriceAggregator creates a new price aggregator
func NewPriceAggregator(providerManager *providers.ProviderManager, config *Config) *PriceAggregator {
	if config == nil {
		config = GetDefaultConfig()
	}

	aggregator := &PriceAggregator{
		providerManager: providerManager,
		config:          config,
		priceCache:      make(map[string]*CachedPrice),
		cacheTTL:        config.CacheTTL,
		stats:           &AggregatorStats{
			ProviderStats: make(map[string]*ProviderAggregatorStats),
		},
		outlierDetector: NewOutlierDetector(config.OutlierDetectionMethod, config.OutlierThreshold),
	}

	// Start cleanup goroutine for cache
	if config.EnableCaching {
		aggregator.startCacheCleanup()
	}

	return aggregator
}

// GetAggregatedPrice retrieves and aggregates prices from multiple providers
func (pa *PriceAggregator) GetAggregatedPrice(ctx context.Context, symbol string) (*models.AggregatedPrice, error) {
	start := time.Now()
	pa.stats.TotalRequests++

	// Check cache first
	if pa.config.EnableCaching {
		if cached := pa.getCachedPrice(symbol); cached != nil {
			pa.stats.CacheHits++
			return cached.Price, nil
		}
		pa.stats.CacheMisses++
	}

	// Get healthy providers
	healthyProviders := pa.providerManager.GetHealthyProviders()
	if len(healthyProviders) < pa.config.MinProviders {
		pa.stats.FailedRequests++
		return nil, fmt.Errorf("insufficient healthy providers: %d < %d",
			len(healthyProviders), pa.config.MinProviders)
	}

	// Limit providers if configured
	selectedProviders := pa.selectProviders(healthyProviders)

	// Fetch prices concurrently
	prices, err := pa.fetchPricesFromProviders(ctx, symbol, selectedProviders)
	if err != nil {
		pa.stats.FailedRequests++
		return nil, fmt.Errorf("failed to fetch prices: %w", err)
	}

	if len(prices) < pa.config.MinProviders {
		pa.stats.FailedRequests++
		return nil, fmt.Errorf("insufficient price data: %d < %d",
			len(prices), pa.config.MinProviders)
	}

	// Remove outliers
	filteredPrices := pa.removeOutliers(prices)
	if len(filteredPrices) < pa.config.MinProviders {
		// Use original prices if too many outliers detected
		filteredPrices = prices
	}

	// Aggregate prices
	aggregatedPrice, err := pa.aggregatePrices(symbol, filteredPrices)
	if err != nil {
		pa.stats.FailedRequests++
		return nil, fmt.Errorf("failed to aggregate prices: %w", err)
	}

	// Validate result quality
	if err := pa.validateAggregatedPrice(aggregatedPrice); err != nil {
		pa.stats.FailedRequests++
		return nil, fmt.Errorf("aggregated price validation failed: %w", err)
	}

	// Cache result
	if pa.config.EnableCaching {
		pa.cachePrice(symbol, aggregatedPrice)
	}

	// Update statistics
	pa.updateStats(time.Since(start), aggregatedPrice, len(prices), len(filteredPrices))
	pa.stats.SuccessfulRequests++

	return aggregatedPrice, nil
}

// GetBatchAggregatedPrices retrieves aggregated prices for multiple symbols
func (pa *PriceAggregator) GetBatchAggregatedPrices(ctx context.Context, symbols []string) (map[string]*models.AggregatedPrice, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("symbols list cannot be empty")
	}

	results := make(map[string]*models.AggregatedPrice)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use semaphore to limit concurrent requests
	semaphore := make(chan struct{}, pa.config.MaxConcurrency)

	for _, symbol := range symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			price, err := pa.GetAggregatedPrice(ctx, sym)
			if err != nil {
				return // Skip failed requests
			}

			mu.Lock()
			results[sym] = price
			mu.Unlock()
		}(symbol)
	}

	wg.Wait()

	if len(results) == 0 {
		return nil, fmt.Errorf("no prices retrieved for any symbol")
	}

	return results, nil
}

// selectProviders selects which providers to use based on configuration
func (pa *PriceAggregator) selectProviders(providers map[string]types.Provider) map[string]types.Provider {
	if len(providers) <= pa.config.MaxProviders {
		return providers
	}

	// Convert to slice for sorting
	type providerWeight struct {
		name     string
		provider types.Provider
		weight   float64
		score    float64
	}

	var providerList []providerWeight
	weights := pa.providerManager.GetProviderWeights()

	for name, provider := range providers {
		weight := weights[name]

		// Adjust weight based on reliability and latency if configured
		score := weight
		if pa.config.WeightByReliability || pa.config.WeightByLatency {
			score = pa.calculateProviderScore(name, weight)
		}

		providerList = append(providerList, providerWeight{
			name:     name,
			provider: provider,
			weight:   weight,
			score:    score,
		})
	}

	// Sort by score (descending)
	sort.Slice(providerList, func(i, j int) bool {
		return providerList[i].score > providerList[j].score
	})

	// Select top providers
	selected := make(map[string]types.Provider)
	for i := 0; i < pa.config.MaxProviders && i < len(providerList); i++ {
		p := providerList[i]
		selected[p.name] = p.provider
	}

	return selected
}

// calculateProviderScore calculates a score for provider selection
func (pa *PriceAggregator) calculateProviderScore(providerName string, baseWeight float64) float64 {
	pa.stats.mu.RLock()
	providerStats, exists := pa.stats.ProviderStats[providerName]
	pa.stats.mu.RUnlock()

	if !exists {
		return baseWeight
	}

	score := baseWeight

	// Adjust for reliability
	if pa.config.WeightByReliability {
		score *= providerStats.ReliabilityScore
	}

	// Adjust for latency (inverse relationship)
	if pa.config.WeightByLatency && providerStats.AverageLatency > 0 {
		// Lower latency = higher score
		latencyFactor := 1.0 / (1.0 + providerStats.AverageLatency.Seconds())
		score *= latencyFactor
	}

	// Apply time decay for inactive providers
	if pa.config.WeightDecayFactor > 0 {
		timeSinceLastUse := time.Since(providerStats.LastUsed)
		decayFactor := math.Exp(-pa.config.WeightDecayFactor * timeSinceLastUse.Hours())
		score *= decayFactor
	}

	return score
}

// fetchPricesFromProviders fetches prices from selected providers concurrently
func (pa *PriceAggregator) fetchPricesFromProviders(ctx context.Context, symbol string, providers map[string]types.Provider) (map[string]*models.ProviderPrice, error) {
	prices := make(map[string]*models.ProviderPrice)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, pa.config.RequestTimeout)
	defer cancel()

	for name, provider := range providers {
		wg.Add(1)
		go func(providerName string, p types.Provider) {
			defer wg.Done()

			start := time.Now()
			price, err := p.GetPrice(ctx, symbol)
			latency := time.Since(start)

			// Update provider statistics
			pa.updateProviderStats(providerName, err == nil, latency)

			if err != nil {
				return // Skip failed requests
			}

			// Convert to ProviderPrice
			providerPrice := &models.ProviderPrice{
				Price:     price.Price,
				Timestamp: price.Timestamp,
				Latency:   latency,
				Weight:    1.0,
				IsOutlier: false,
			}

			mu.Lock()
			prices[providerName] = providerPrice
			mu.Unlock()
		}(name, provider)
	}

	wg.Wait()

	if len(prices) == 0 {
		return nil, fmt.Errorf("no valid prices retrieved from any provider")
	}

	return prices, nil
}

// removeOutliers removes outlier prices using the configured method
func (pa *PriceAggregator) removeOutliers(prices map[string]*models.ProviderPrice) map[string]*models.ProviderPrice {
	if len(prices) < 3 {
		return prices // Not enough data for outlier detection
	}

	// Extract prices for analysis
	priceValues := make([]float64, 0, len(prices))
	for _, p := range prices {
		priceValues = append(priceValues, p.Price.InexactFloat64())
	}

	// Detect outliers
	outlierIndices := pa.outlierDetector.DetectOutliers(priceValues)
	if len(outlierIndices) == 0 {
		return prices
	}

	// Create mapping of indices to provider names
	var providerNames []string
	for name := range prices {
		providerNames = append(providerNames, name)
	}

	// Remove outliers
	filtered := make(map[string]*models.ProviderPrice)
	for i, name := range providerNames {
		isOutlier := false
		for _, outlierIdx := range outlierIndices {
			if i == outlierIdx {
				isOutlier = true
				break
			}
		}

		if !isOutlier {
			filtered[name] = prices[name]
		} else {
			pa.stats.OutliersDetected++
			pa.updateProviderOutlierStats(name)
		}
	}

	return filtered
}

// aggregatePrices combines prices from multiple providers using the configured strategy
func (pa *PriceAggregator) aggregatePrices(symbol string, prices map[string]*models.ProviderPrice) (*models.AggregatedPrice, error) {
	if len(prices) == 0 {
		return nil, fmt.Errorf("no prices to aggregate")
	}

	var aggregatedPrice decimal.Decimal
	var err error

	switch pa.config.Strategy {
	case "weighted_average":
		aggregatedPrice, err = pa.calculateWeightedAverage(prices)
	case "median":
		aggregatedPrice, err = pa.calculateMedian(prices)
	case "best_price":
		aggregatedPrice, err = pa.selectBestPrice(prices)
	default:
		aggregatedPrice, err = pa.calculateWeightedAverage(prices)
	}

	if err != nil {
		return nil, err
	}

	// Calculate confidence score
	confidence := pa.calculateConfidenceScore(prices, aggregatedPrice)

	// Calculate aggregated volume and other metrics
	totalVolume := decimal.Zero
	var latencies []time.Duration
	var timestamps []time.Time

	for _, price := range prices {
		totalVolume = totalVolume.Add(decimal.Zero)
		latencies = append(latencies, price.Latency)
		timestamps = append(timestamps, price.Timestamp)
	}

	// Calculate average latency
	var totalLatency time.Duration
	for _, latency := range latencies {
		totalLatency += latency
	}
	avgLatency := totalLatency / time.Duration(len(latencies))

	// Find most recent timestamp
	var latestTimestamp time.Time
	for _, ts := range timestamps {
		if ts.After(latestTimestamp) {
			latestTimestamp = ts
		}
	}

	// Create aggregation metadata
	metadata := &models.AggregationMetadata{
		ProvidersUsed:   getMapKeys(prices),
		OutliersRemoved:   0, // This would be tracked separately
		Method: pa.config.Strategy,
		LastUpdate:      time.Now(),
		ProcessingTime:    avgLatency,
	}

	result := &models.AggregatedPrice{
		Symbol:         symbol,
		Price:          aggregatedPrice,
		Volume24h:         totalVolume,
		Timestamp:      latestTimestamp,
		Source:         "aggregated",
		Confidence:     confidence,
		ProviderPrices: prices,
		Metadata:       metadata,
	}

	return result, nil
}

// calculateWeightedAverage calculates weighted average price
func (pa *PriceAggregator) calculateWeightedAverage(prices map[string]*models.ProviderPrice) (decimal.Decimal, error) {
	if len(prices) == 0 {
		return decimal.Zero, fmt.Errorf("no prices provided")
	}

	weights := pa.providerManager.GetProviderWeights()

	var weightedSum decimal.Decimal
	var totalWeight decimal.Decimal

	for providerName, price := range prices {
		weight := decimal.NewFromFloat(weights[providerName])

		// Adjust weight based on confidence and other factors
		adjustedWeight := weight.Mul(decimal.NewFromFloat(1.0))

		// Weight by volume if significant
		if !decimal.Zero.IsZero() {
			volumeFactor := decimal.Zero.Div(decimal.Zero.Add(decimal.NewFromInt(1000000))) // Normalize volume impact
			adjustedWeight = adjustedWeight.Mul(decimal.NewFromInt(1).Add(volumeFactor))
		}

		weightedSum = weightedSum.Add(price.Price.Mul(adjustedWeight))
		totalWeight = totalWeight.Add(adjustedWeight)
	}

	if totalWeight.IsZero() {
		return decimal.Zero, fmt.Errorf("total weight is zero")
	}

	return weightedSum.Div(totalWeight), nil
}

// calculateMedian calculates median price
func (pa *PriceAggregator) calculateMedian(prices map[string]*models.ProviderPrice) (decimal.Decimal, error) {
	if len(prices) == 0 {
		return decimal.Zero, fmt.Errorf("no prices provided")
	}

	priceList := make([]decimal.Decimal, 0, len(prices))
	for _, price := range prices {
		priceList = append(priceList, price.Price)
	}

	// Sort prices
	sort.Slice(priceList, func(i, j int) bool {
		return priceList[i].LessThan(priceList[j])
	})

	n := len(priceList)
	if n%2 == 0 {
		// Even number of prices - return average of middle two
		mid1 := priceList[n/2-1]
		mid2 := priceList[n/2]
		return mid1.Add(mid2).Div(decimal.NewFromInt(2)), nil
	} else {
		// Odd number of prices - return middle value
		return priceList[n/2], nil
	}
}

// selectBestPrice selects the best price based on criteria (lowest spread, highest confidence, etc.)
func (pa *PriceAggregator) selectBestPrice(prices map[string]*models.ProviderPrice) (decimal.Decimal, error) {
	if len(prices) == 0 {
		return decimal.Zero, fmt.Errorf("no prices provided")
	}

	var bestPrice *models.ProviderPrice
	var bestScore float64

	for _, price := range prices {
		// Calculate composite score based on confidence, spread, and latency
		score := 1.0

		// Lower spread is better

		// Lower latency is better
		latencyFactor := 1.0 / (1.0 + price.Latency.Seconds())
		score *= latencyFactor

		if bestPrice == nil || score > bestScore {
			bestPrice = price
			bestScore = score
		}
	}

	return bestPrice.Price, nil
}

// calculateConfidenceScore calculates confidence score for aggregated price
func (pa *PriceAggregator) calculateConfidenceScore(prices map[string]*models.ProviderPrice, aggregatedPrice decimal.Decimal) float64 {
	if len(prices) == 0 {
		return 0.0
	}

	// Base confidence on number of providers
	baseConfidence := float64(len(prices)) / float64(pa.config.MaxProviders)
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	// Calculate price variance
	var variance decimal.Decimal
	for _, price := range prices {
		diff := price.Price.Sub(aggregatedPrice).Abs()
		variance = variance.Add(diff.Mul(diff))
	}

	if len(prices) > 1 {
		variance = variance.Div(decimal.NewFromInt(int64(len(prices) - 1)))
	}

	// Lower variance = higher confidence
	varianceFactor := 1.0
	if !variance.IsZero() {
		coefficientOfVariation := variance.Div(aggregatedPrice.Abs()).InexactFloat64()
		varianceFactor = 1.0 / (1.0 + coefficientOfVariation)
	}

	// Combine factors
	confidence := baseConfidence * varianceFactor

	// Cap between 0 and 1
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}

// validateAggregatedPrice validates the quality of aggregated price
func (pa *PriceAggregator) validateAggregatedPrice(price *models.AggregatedPrice) error {
	// Check confidence score
	if 1.0 < pa.config.MinConfidenceScore {
		return fmt.Errorf("confidence score too low: %.2f < %.2f",
			1.0, pa.config.MinConfidenceScore)
	}

	// Check price deviation if configured
	if !pa.config.MaxPriceDeviation.IsZero() {
		median, _ := pa.calculateMedian(price.ProviderPrices)
		if !median.IsZero() {
			deviation := price.Price.Sub(median).Abs().Div(median).Mul(decimal.NewFromInt(100))
			if deviation.GreaterThan(pa.config.MaxPriceDeviation) {
				return fmt.Errorf("price deviation too high: %s%% > %s%%",
					deviation.String(), pa.config.MaxPriceDeviation.String())
			}
		}
	}

	// Check quorum if required
	if pa.config.RequireQuorum {
		requiredQuorum := len(price.ProviderPrices)/2 + 1
		if len(price.ProviderPrices) < requiredQuorum {
			return fmt.Errorf("insufficient quorum: %d < %d",
				len(price.ProviderPrices), requiredQuorum)
		}
	}

	return nil
}

// Cache management methods

func (pa *PriceAggregator) getCachedPrice(symbol string) *CachedPrice {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	cached, exists := pa.priceCache[symbol]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Since(cached.Timestamp) > cached.TTL {
		delete(pa.priceCache, symbol)
		return nil
	}

	return cached
}

func (pa *PriceAggregator) cachePrice(symbol string, price *models.AggregatedPrice) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pa.priceCache[symbol] = &CachedPrice{
		Price:     price,
		Timestamp: time.Now(),
		TTL:       pa.cacheTTL,
	}
}

func (pa *PriceAggregator) startCacheCleanup() {
	pa.cleanupTicker = time.NewTicker(pa.cacheTTL / 2) // Cleanup at half TTL interval

	go func() {
		for range pa.cleanupTicker.C {
			pa.cleanupExpiredCache()
		}
	}()
}

func (pa *PriceAggregator) cleanupExpiredCache() {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	now := time.Now()
	for symbol, cached := range pa.priceCache {
		if now.Sub(cached.Timestamp) > cached.TTL {
			delete(pa.priceCache, symbol)
		}
	}
}

// Statistics management methods

func (pa *PriceAggregator) updateStats(latency time.Duration, price *models.AggregatedPrice, totalProviders, filteredProviders int) {
	pa.stats.mu.Lock()
	defer pa.stats.mu.Unlock()

	// Update average latency
	if pa.stats.AverageLatency == 0 {
		pa.stats.AverageLatency = latency
	} else {
		pa.stats.AverageLatency = (pa.stats.AverageLatency + latency) / 2
	}

	// Update average confidence
	if pa.stats.AverageConfidence == 0 {
		pa.stats.AverageConfidence = 1.0
	} else {
		pa.stats.AverageConfidence = (pa.stats.AverageConfidence + 1.0) / 2
	}

	pa.stats.LastUpdated = time.Now()
}

func (pa *PriceAggregator) updateProviderStats(providerName string, success bool, latency time.Duration) {
	pa.stats.mu.Lock()
	defer pa.stats.mu.Unlock()

	if pa.stats.ProviderStats[providerName] == nil {
		pa.stats.ProviderStats[providerName] = &ProviderAggregatorStats{}
	}

	stats := pa.stats.ProviderStats[providerName]
	stats.RequestCount++
	stats.LastUsed = time.Now()

	if success {
		stats.SuccessCount++
	} else {
		stats.ErrorCount++
	}

	// Update average latency
	if stats.AverageLatency == 0 {
		stats.AverageLatency = latency
	} else {
		stats.AverageLatency = (stats.AverageLatency + latency) / 2
	}

	// Calculate reliability score
	if stats.RequestCount > 0 {
		stats.ReliabilityScore = float64(stats.SuccessCount) / float64(stats.RequestCount)
	}
}

func (pa *PriceAggregator) updateProviderOutlierStats(providerName string) {
	pa.stats.mu.Lock()
	defer pa.stats.mu.Unlock()

	if pa.stats.ProviderStats[providerName] == nil {
		pa.stats.ProviderStats[providerName] = &ProviderAggregatorStats{}
	}

	pa.stats.ProviderStats[providerName].OutlierCount++
}

// GetStats returns aggregator statistics
func (pa *PriceAggregator) GetStats() *AggregatorStats {
	pa.stats.mu.RLock()
	defer pa.stats.mu.RUnlock()

	// Create a copy to avoid race conditions
	statsCopy := *pa.stats
	providerStats := make(map[string]*ProviderAggregatorStats)

	for name, stats := range pa.stats.ProviderStats {
		statsCopy := *stats
		providerStats[name] = &statsCopy
	}

	statsCopy.ProviderStats = providerStats
	return &statsCopy
}

// GetDefaultConfig returns default aggregator configuration
func GetDefaultConfig() *Config {
	return &Config{
		Strategy:               "weighted_average",
		OutlierDetectionMethod: "z_score",
		OutlierThreshold:       2.0,
		MinProviders:          2,
		MaxProviders:          5,
		WeightByLatency:       true,
		WeightByReliability:   true,
		WeightDecayFactor:     0.1,
		MaxPriceDeviation:     decimal.NewFromFloat(10.0), // 10%
		MinConfidenceScore:    0.5,
		RequireQuorum:         false,
		CacheTTL:             30 * time.Second,
		EnableCaching:        true,
		MaxConcurrency:       10,
		RequestTimeout:       10 * time.Second,
		FallbackStrategy:     "cached",
		FallbackProvider:     "coingecko",
	}
}

// Stop stops the aggregator and cleanup processes
func (pa *PriceAggregator) Stop() {
	if pa.cleanupTicker != nil {
		pa.cleanupTicker.Stop()
	}
}

// getMapKeys extracts keys from map
func getMapKeys(m map[string]*models.ProviderPrice) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}