package aggregator

import (
	"sort"
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"market-data-api/internal/models"
	"market-data-api/internal/providers"
)

// Service provides high-level aggregation services
type Service struct {
	aggregator        *PriceAggregator
	technicalAnalyzer *TechnicalAnalyzer
	providerManager   *providers.ProviderManager
	config            *ServiceConfig

	// Background processing
	backgroundCtx    context.Context
	backgroundCancel context.CancelFunc
	wg               sync.WaitGroup

	// Metrics
	metrics *ServiceMetrics
	mu      sync.RWMutex
}

// ServiceConfig represents aggregation service configuration
type ServiceConfig struct {
	// Background processing
	EnableBackgroundProcessing bool          `json:"enable_background_processing"`
	ProcessingInterval        time.Duration `json:"processing_interval"`
	PopularSymbols           []string      `json:"popular_symbols"`

	// Precomputation
	EnablePrecomputation      bool          `json:"enable_precomputation"`
	PrecomputeSymbols        []string      `json:"precompute_symbols"`
	PrecomputeInterval       time.Duration `json:"precompute_interval"`

	// Quality assurance
	EnableQualityChecks      bool          `json:"enable_quality_checks"`
	QualityCheckInterval     time.Duration `json:"quality_check_interval"`

	// Performance optimization
	MaxConcurrentRequests    int           `json:"max_concurrent_requests"`
	RequestTimeout          time.Duration `json:"request_timeout"`

	// Alerting
	EnableAlerting          bool          `json:"enable_alerting"`
	AlertThresholds         map[string]float64 `json:"alert_thresholds"`
}

// ServiceMetrics tracks service performance metrics
type ServiceMetrics struct {
	TotalRequests           int64         `json:"total_requests"`
	SuccessfulRequests      int64         `json:"successful_requests"`
	FailedRequests          int64         `json:"failed_requests"`
	AverageResponseTime     time.Duration `json:"average_response_time"`
	BackgroundProcessingRuns int64        `json:"background_processing_runs"`
	PrecomputedPrices       int64         `json:"precomputed_prices"`
	QualityCheckRuns        int64         `json:"quality_check_runs"`
	AlertsTriggered         int64         `json:"alerts_triggered"`
	LastUpdated            time.Time     `json:"last_updated"`
}

// NewService creates a new aggregation service
func NewService(providerManager *providers.ProviderManager, config *ServiceConfig) *Service {
	if config == nil {
		config = GetDefaultServiceConfig()
	}

	aggregatorConfig := GetDefaultConfig()
	aggregator := NewPriceAggregator(providerManager, aggregatorConfig)
	technicalAnalyzer := NewTechnicalAnalyzer(providerManager)

	backgroundCtx, backgroundCancel := context.WithCancel(context.Background())

	service := &Service{
		aggregator:        aggregator,
		technicalAnalyzer: technicalAnalyzer,
		providerManager:   providerManager,
		config:            config,
		backgroundCtx:     backgroundCtx,
		backgroundCancel:  backgroundCancel,
		metrics:          &ServiceMetrics{},
	}

	// Start background processes if enabled
	if config.EnableBackgroundProcessing {
		service.startBackgroundProcesses()
	}

	return service
}

// GetAggregatedPrice retrieves aggregated price with enhanced features
func (s *Service) GetAggregatedPrice(ctx context.Context, symbol string, options *PriceOptions) (*EnhancedAggregatedPrice, error) {
	start := time.Now()
	s.incrementMetric("total_requests")

	// Set defaults for options
	if options == nil {
		options = &PriceOptions{}
	}

	// Get basic aggregated price
	price, err := s.aggregator.GetAggregatedPrice(ctx, symbol)
	if err != nil {
		s.incrementMetric("failed_requests")
		return nil, fmt.Errorf("failed to get aggregated price: %w", err)
	}

	// Create enhanced result
	result := &EnhancedAggregatedPrice{
		AggregatedPrice: price,
		TechnicalSignals: nil,
		QualityScore:    s.calculateQualityScore(price),
		Timestamp:       time.Now(),
	}

	// Add technical analysis if requested
	if options.IncludeTechnicalAnalysis {
		indicators, err := s.technicalAnalyzer.AnalyzeTechnicalIndicators(ctx, symbol, options.TechnicalPeriod)
		if err == nil {
			signals := s.technicalAnalyzer.GetTechnicalSignals(indicators)
			result.TechnicalSignals = signals
			result.TechnicalIndicators = indicators
		}
	}

	// Add market sentiment if requested
	if options.IncludeMarketSentiment {
		sentiment := s.calculateMarketSentiment(price)
		result.MarketSentiment = sentiment
	}

	// Add volatility analysis if requested
	if options.IncludeVolatility {
		volatility, err := s.calculateVolatility(ctx, symbol, options.VolatilityPeriod)
		if err == nil {
			result.Volatility = volatility
		}
	}

	// Update metrics
	s.updateResponseTime(time.Since(start))
	s.incrementMetric("successful_requests")

	return result, nil
}

// GetBatchAggregatedPrices retrieves multiple aggregated prices efficiently
func (s *Service) GetBatchAggregatedPrices(ctx context.Context, symbols []string, options *PriceOptions) (map[string]*EnhancedAggregatedPrice, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("symbols list cannot be empty")
	}

	results := make(map[string]*EnhancedAggregatedPrice)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use semaphore to limit concurrent requests
	concurrency := s.config.MaxConcurrentRequests
	if concurrency <= 0 {
		concurrency = 10
	}

	semaphore := make(chan struct{}, concurrency)

	for _, symbol := range symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			price, err := s.GetAggregatedPrice(ctx, sym, options)
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

// GetMarketOverview provides a comprehensive market overview
func (s *Service) GetMarketOverview(ctx context.Context) (*MarketOverview, error) {
	popularSymbols := s.config.PopularSymbols
	if len(popularSymbols) == 0 {
		popularSymbols = []string{"BTC", "ETH", "ADA", "DOT", "LINK"} // Default symbols
	}

	options := &PriceOptions{
		IncludeTechnicalAnalysis: true,
		IncludeMarketSentiment:   true,
		IncludeVolatility:        true,
		TechnicalPeriod:         "24h",
		VolatilityPeriod:        "7d",
	}

	prices, err := s.GetBatchAggregatedPrices(ctx, popularSymbols, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get market prices: %w", err)
	}

	overview := &MarketOverview{
		Timestamp:      time.Now(),
		TotalSymbols:   len(prices),
		MarketPrices:   prices,
		MarketSentiment: s.calculateOverallMarketSentiment(prices),
		TopGainers:     s.findTopMovers(prices, true),
		TopLosers:      s.findTopMovers(prices, false),
		Statistics:     s.calculateMarketStatistics(prices),
	}

	return overview, nil
}

// Quality assessment methods

func (s *Service) calculateQualityScore(price *models.AggregatedPrice) float64 {
	score := 0.0
	factors := 0.0

	// Provider count factor (more providers = higher quality)
	if price.Metadata != nil {
		providerFactor := float64(len(price.Metadata.ProvidersUsed)) / 5.0 // Normalize to max 5 providers
		if providerFactor > 1.0 {
			providerFactor = 1.0
		}
		score += providerFactor * 0.3
		factors += 0.3
	}

	// Confidence factor
	score += price.Confidence * 0.4
	factors += 0.4

	// Volume factor (higher volume = more reliable)
	if !price.Volume24h.IsZero() {
		volumeFactor := math.Min(price.Volume24h.InexactFloat64()/1000000, 1.0) // Normalize to 1M volume
		score += volumeFactor * 0.2
		factors += 0.2
	}

	// Recency factor (more recent = higher quality)
	ageMinutes := time.Since(price.Timestamp).Minutes()
	recencyFactor := math.Max(0, 1.0-ageMinutes/60) // Decay over 1 hour
	score += recencyFactor * 0.1
	factors += 0.1

	if factors == 0 {
		return 0.5 // Default score
	}

	return score / factors
}

func (s *Service) calculateMarketSentiment(price *models.AggregatedPrice) *MarketSentiment {
	sentiment := &MarketSentiment{
		Symbol:    price.Symbol,
		Timestamp: time.Now(),
	}

	// Calculate sentiment based on price movement and volume
	// This is a simplified implementation
	if len(price.ProviderPrices) > 1 {
		prices := make([]decimal.Decimal, 0, len(price.ProviderPrices))
		for _, providerPrice := range price.ProviderPrices {
			prices = append(prices, providerPrice.Price)
		}

		// Calculate price variance as a sentiment indicator
		var sum, variance decimal.Decimal
		for _, p := range prices {
			sum = sum.Add(p)
		}
		mean := sum.Div(decimal.NewFromInt(int64(len(prices))))

		for _, p := range prices {
			diff := p.Sub(mean)
			variance = variance.Add(diff.Mul(diff))
		}
		variance = variance.Div(decimal.NewFromInt(int64(len(prices))))

		// Low variance = consensus = positive sentiment
		coeffVar := variance.Div(mean).InexactFloat64()
		if coeffVar < 0.01 {
			sentiment.Sentiment = "BULLISH"
			sentiment.Score = 0.8
		} else if coeffVar < 0.05 {
			sentiment.Sentiment = "NEUTRAL"
			sentiment.Score = 0.5
		} else {
			sentiment.Sentiment = "BEARISH"
			sentiment.Score = 0.2
		}
	} else {
		sentiment.Sentiment = "NEUTRAL"
		sentiment.Score = 0.5
	}

	return sentiment
}

func (s *Service) calculateVolatility(ctx context.Context, symbol string, period string) (*models.VolatilityData, error) {
	// Get historical data for volatility calculation
	candles, err := s.technicalAnalyzer.getHistoricalCandles(ctx, symbol, "1h", period, 200)
	if err != nil {
		return nil, err
	}

	if len(candles) < 2 {
		return nil, fmt.Errorf("insufficient data for volatility calculation")
	}

	// Calculate returns
	returns := make([]float64, 0, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		if !candles[i-1].Close.IsZero() {
			returnValue := candles[i].Close.Sub(candles[i-1].Close).Div(candles[i-1].Close).InexactFloat64()
			returns = append(returns, returnValue)
		}
	}

	if len(returns) < 2 {
		return nil, fmt.Errorf("insufficient returns for volatility calculation")
	}

	// Calculate various volatility measures
	volatility := &models.VolatilityData{
		Symbol:           symbol,
		Period:           period,
		CalculationMethod: "close-to-close",
	}

	// Standard deviation of returns
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns) - 1)

	volatility.Volatility = decimal.NewFromFloat(math.Sqrt(variance))
	volatility.AnnualizedVolatility = volatility.Volatility.Mul(decimal.NewFromFloat(math.Sqrt(365 * 24))) // Assuming hourly data

	return volatility, nil
}

// Market analysis methods

func (s *Service) calculateOverallMarketSentiment(prices map[string]*EnhancedAggregatedPrice) string {
	bullishCount := 0
	bearishCount := 0
	neutralCount := 0

	for _, price := range prices {
		if price.MarketSentiment != nil {
			switch price.MarketSentiment.Sentiment {
			case "BULLISH":
				bullishCount++
			case "BEARISH":
				bearishCount++
			default:
				neutralCount++
			}
		} else {
			neutralCount++
		}
	}

	total := bullishCount + bearishCount + neutralCount
	if total == 0 {
		return "NEUTRAL"
	}

	bullishRatio := float64(bullishCount) / float64(total)
	bearishRatio := float64(bearishCount) / float64(total)

	if bullishRatio > 0.6 {
		return "BULLISH"
	} else if bearishRatio > 0.6 {
		return "BEARISH"
	} else {
		return "NEUTRAL"
	}
}

func (s *Service) findTopMovers(prices map[string]*EnhancedAggregatedPrice, gainers bool) []TopMover {
	type mover struct {
		symbol string
		change decimal.Decimal
		price  *EnhancedAggregatedPrice
	}

	var movers []mover
	for symbol, price := range prices {
		// Calculate change based on technical indicators or provider data
		change := decimal.Zero
		if price.TechnicalIndicators != nil && !price.TechnicalIndicators.MovingAverages.MA20.IsZero() {
			change = price.Price.Sub(price.TechnicalIndicators.MovingAverages.MA20).Div(price.TechnicalIndicators.MovingAverages.MA20).Mul(decimal.NewFromInt(100))
		}

		movers = append(movers, mover{
			symbol: symbol,
			change: change,
			price:  price,
		})
	}

	// Sort by change
	sort.Slice(movers, func(i, j int) bool {
		if gainers {
			return movers[i].change.GreaterThan(movers[j].change)
		} else {
			return movers[i].change.LessThan(movers[j].change)
		}
	})

	// Return top 5
	limit := 5
	if len(movers) < limit {
		limit = len(movers)
	}

	result := make([]TopMover, limit)
	for i := 0; i < limit; i++ {
		result[i] = TopMover{
			Symbol:     movers[i].symbol,
			Price:      movers[i].price.Price,
			Change:     movers[i].change,
			ChangePercent: movers[i].change,
		}
	}

	return result
}

func (s *Service) calculateMarketStatistics(prices map[string]*EnhancedAggregatedPrice) *MarketStatistics {
	if len(prices) == 0 {
		return &MarketStatistics{}
	}

	totalVolume := decimal.Zero
	totalValue := decimal.Zero
	qualitySum := 0.0
	confidenceSum := 0.0

	for _, price := range prices {
		totalVolume = totalVolume.Add(price.Volume24h)
		totalValue = totalValue.Add(price.Price.Mul(price.Volume24h))
		qualitySum += price.QualityScore
		confidenceSum += price.Confidence
	}

	count := float64(len(prices))
	avgPrice := decimal.Zero
	if !totalVolume.IsZero() {
		avgPrice = totalValue.Div(totalVolume)
	}

	return &MarketStatistics{
		TotalSymbols:      len(prices),
		TotalVolume:       totalVolume,
		AveragePrice:      avgPrice,
		AverageQuality:    qualitySum / count,
		AverageConfidence: confidenceSum / count,
		Timestamp:         time.Now(),
	}
}

// Background processing methods

func (s *Service) startBackgroundProcesses() {
	s.wg.Add(1)
	go s.backgroundProcessingLoop()

	if s.config.EnablePrecomputation {
		s.wg.Add(1)
		go s.precomputationLoop()
	}

	if s.config.EnableQualityChecks {
		s.wg.Add(1)
		go s.qualityCheckLoop()
	}
}

func (s *Service) backgroundProcessingLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.ProcessingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.backgroundCtx.Done():
			return
		case <-ticker.C:
			s.runBackgroundProcessing()
		}
	}
}

func (s *Service) precomputationLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.PrecomputeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.backgroundCtx.Done():
			return
		case <-ticker.C:
			s.runPrecomputation()
		}
	}
}

func (s *Service) qualityCheckLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.QualityCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.backgroundCtx.Done():
			return
		case <-ticker.C:
			s.runQualityChecks()
		}
	}
}

func (s *Service) runBackgroundProcessing() {
	s.incrementMetric("background_processing_runs")

	// Perform maintenance tasks
	// - Clean up expired cache entries
	// - Update provider health status
	// - Rotate logs

	// This is a placeholder for actual background processing logic
}

func (s *Service) runPrecomputation() {
	if len(s.config.PrecomputeSymbols) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(s.backgroundCtx, s.config.RequestTimeout)
	defer cancel()

	options := &PriceOptions{
		IncludeTechnicalAnalysis: true,
		IncludeMarketSentiment:  true,
		TechnicalPeriod:         "24h",
	}

	_, err := s.GetBatchAggregatedPrices(ctx, s.config.PrecomputeSymbols, options)
	if err == nil {
		s.incrementMetric("precomputed_prices")
	}
}

func (s *Service) runQualityChecks() {
	s.incrementMetric("quality_check_runs")

	// Perform quality checks
	// - Validate data consistency
	// - Check for anomalies
	// - Monitor provider performance

	// This is a placeholder for actual quality check logic
}

// Metrics methods

func (s *Service) incrementMetric(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch name {
	case "total_requests":
		s.metrics.TotalRequests++
	case "successful_requests":
		s.metrics.SuccessfulRequests++
	case "failed_requests":
		s.metrics.FailedRequests++
	case "background_processing_runs":
		s.metrics.BackgroundProcessingRuns++
	case "precomputed_prices":
		s.metrics.PrecomputedPrices++
	case "quality_check_runs":
		s.metrics.QualityCheckRuns++
	case "alerts_triggered":
		s.metrics.AlertsTriggered++
	}

	s.metrics.LastUpdated = time.Now()
}

func (s *Service) updateResponseTime(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.metrics.AverageResponseTime == 0 {
		s.metrics.AverageResponseTime = duration
	} else {
		s.metrics.AverageResponseTime = (s.metrics.AverageResponseTime + duration) / 2
	}
}

// GetMetrics returns service metrics
func (s *Service) GetMetrics() *ServiceMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metricsCopy := *s.metrics
	return &metricsCopy
}

// Stop stops the service and all background processes
func (s *Service) Stop() {
	s.backgroundCancel()
	s.wg.Wait()

	if s.aggregator != nil {
		s.aggregator.Stop()
	}
}

// GetDefaultServiceConfig returns default service configuration
func GetDefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		EnableBackgroundProcessing: true,
		ProcessingInterval:        5 * time.Minute,
		PopularSymbols:           []string{"BTC", "ETH", "ADA", "DOT", "LINK", "UNI", "AAVE"},
		EnablePrecomputation:     true,
		PrecomputeSymbols:       []string{"BTC", "ETH", "ADA"},
		PrecomputeInterval:      time.Minute,
		EnableQualityChecks:     true,
		QualityCheckInterval:    10 * time.Minute,
		MaxConcurrentRequests:   10,
		RequestTimeout:          30 * time.Second,
		EnableAlerting:          false,
		AlertThresholds:         map[string]float64{},
	}
}

// Data structures for enhanced responses

type PriceOptions struct {
	IncludeTechnicalAnalysis bool   `json:"include_technical_analysis"`
	IncludeMarketSentiment   bool   `json:"include_market_sentiment"`
	IncludeVolatility        bool   `json:"include_volatility"`
	TechnicalPeriod         string `json:"technical_period"`
	VolatilityPeriod        string `json:"volatility_period"`
}

type EnhancedAggregatedPrice struct {
	*models.AggregatedPrice
	TechnicalSignals    *TechnicalSignals        `json:"technical_signals,omitempty"`
	TechnicalIndicators *models.TechnicalIndicators `json:"technical_indicators,omitempty"`
	MarketSentiment     *MarketSentiment         `json:"market_sentiment,omitempty"`
	Volatility          *models.VolatilityData   `json:"volatility,omitempty"`
	QualityScore        float64                  `json:"quality_score"`
	Timestamp           time.Time                `json:"timestamp"`
}

type MarketSentiment struct {
	Symbol    string    `json:"symbol"`
	Sentiment string    `json:"sentiment"` // BULLISH, BEARISH, NEUTRAL
	Score     float64   `json:"score"`     // 0.0 to 1.0
	Timestamp time.Time `json:"timestamp"`
}

type MarketOverview struct {
	Timestamp       time.Time                           `json:"timestamp"`
	TotalSymbols    int                                 `json:"total_symbols"`
	MarketPrices    map[string]*EnhancedAggregatedPrice `json:"market_prices"`
	MarketSentiment string                              `json:"market_sentiment"`
	TopGainers      []TopMover                          `json:"top_gainers"`
	TopLosers       []TopMover                          `json:"top_losers"`
	Statistics      *MarketStatistics                   `json:"statistics"`
}

type TopMover struct {
	Symbol        string          `json:"symbol"`
	Price         decimal.Decimal `json:"price"`
	Change        decimal.Decimal `json:"change"`
	ChangePercent decimal.Decimal `json:"change_percent"`
}

type MarketStatistics struct {
	TotalSymbols      int             `json:"total_symbols"`
	TotalVolume       decimal.Decimal `json:"total_volume"`
	AveragePrice      decimal.Decimal `json:"average_price"`
	AverageQuality    float64         `json:"average_quality"`
	AverageConfidence float64         `json:"average_confidence"`
	Timestamp         time.Time       `json:"timestamp"`
}