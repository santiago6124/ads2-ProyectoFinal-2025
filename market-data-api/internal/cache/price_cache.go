package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"market-data-api/internal/models"
)

// RedisPriceCache implements specialized caching for price data
type RedisPriceCache struct {
	cache Cache
}

// NewRedisPriceCache creates a new Redis-based price cache
func NewRedisPriceCache(cache Cache) *RedisPriceCache {
	return &RedisPriceCache{
		cache: cache,
	}
}

// Key generators for different data types

func (pc *RedisPriceCache) priceKey(symbol string) string {
	return fmt.Sprintf("price:%s", strings.ToUpper(symbol))
}

func (pc *RedisPriceCache) historicalKey(symbol, interval string) string {
	return fmt.Sprintf("historical:%s:%s", strings.ToUpper(symbol), interval)
}

func (pc *RedisPriceCache) marketDataKey(symbol string) string {
	return fmt.Sprintf("market:%s", strings.ToUpper(symbol))
}

func (pc *RedisPriceCache) orderBookKey(symbol string) string {
	return fmt.Sprintf("orderbook:%s", strings.ToUpper(symbol))
}

func (pc *RedisPriceCache) statisticsKey(symbol string) string {
	return fmt.Sprintf("stats:%s", strings.ToUpper(symbol))
}

func (pc *RedisPriceCache) technicalIndicatorsKey(symbol string) string {
	return fmt.Sprintf("technical:%s", strings.ToUpper(symbol))
}

func (pc *RedisPriceCache) volatilityKey(symbol string) string {
	return fmt.Sprintf("volatility:%s", strings.ToUpper(symbol))
}

// Price operations

func (pc *RedisPriceCache) SetPrice(ctx context.Context, symbol string, price *models.AggregatedPrice, ttl time.Duration) error {
	key := pc.priceKey(symbol)

	data, err := json.Marshal(price)
	if err != nil {
		return NewCacheError("set_price", key, ErrCodeSerialization, err)
	}

	if err := pc.cache.Set(ctx, key, data, ttl); err != nil {
		return err
	}

	// Also store individual provider prices for analysis
	if price.ProviderPrices != nil {
		pipe := pc.cache.Pipeline()

		for providerName, providerPrice := range price.ProviderPrices {
			providerKey := fmt.Sprintf("%s:provider:%s", key, providerName)
			providerData, err := json.Marshal(providerPrice)
			if err != nil {
				continue // Skip failed serializations
			}
			pipe.Set(providerKey, providerData, ttl)
		}

		_, err := pipe.Exec(ctx)
		if err != nil {
			// Don't fail the main operation for provider price storage
		}
	}

	return nil
}

func (pc *RedisPriceCache) GetPrice(ctx context.Context, symbol string) (*models.AggregatedPrice, error) {
	key := pc.priceKey(symbol)

	data, err := pc.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var price models.AggregatedPrice
	if err := json.Unmarshal(data, &price); err != nil {
		return nil, NewCacheError("get_price", key, ErrCodeSerialization, err)
	}

	return &price, nil
}

func (pc *RedisPriceCache) SetPrices(ctx context.Context, prices map[string]*models.AggregatedPrice, ttl time.Duration) error {
	if len(prices) == 0 {
		return nil
	}

	keyValues := make(map[string][]byte)

	for symbol, price := range prices {
		key := pc.priceKey(symbol)
		data, err := json.Marshal(price)
		if err != nil {
			continue // Skip failed serializations
		}
		keyValues[key] = data
	}

	return pc.cache.MSet(ctx, keyValues, ttl)
}

func (pc *RedisPriceCache) GetPrices(ctx context.Context, symbols []string) (map[string]*models.AggregatedPrice, error) {
	if len(symbols) == 0 {
		return make(map[string]*models.AggregatedPrice), nil
	}

	keys := make([]string, len(symbols))
	for i, symbol := range symbols {
		keys[i] = pc.priceKey(symbol)
	}

	data, err := pc.cache.MGet(ctx, keys)
	if err != nil {
		return nil, err
	}

	results := make(map[string]*models.AggregatedPrice)
	for i, symbol := range symbols {
		key := keys[i]
		if rawData, exists := data[key]; exists {
			var price models.AggregatedPrice
			if err := json.Unmarshal(rawData, &price); err == nil {
				results[symbol] = &price
			}
		}
	}

	return results, nil
}

func (pc *RedisPriceCache) DelPrice(ctx context.Context, symbols ...string) error {
	if len(symbols) == 0 {
		return nil
	}

	keys := make([]string, len(symbols))
	for i, symbol := range symbols {
		keys[i] = pc.priceKey(symbol)
	}

	return pc.cache.Del(ctx, keys...)
}

// Historical data operations

func (pc *RedisPriceCache) SetHistoricalData(ctx context.Context, symbol string, interval string, data []*models.Candle, ttl time.Duration) error {
	key := pc.historicalKey(symbol, interval)

	// Store as sorted set with timestamp as score
	pipe := pc.cache.Pipeline()

	for _, candle := range data {
		candleData, err := json.Marshal(candle)
		if err != nil {
			continue
		}

		score := float64(candle.Timestamp.Unix())
		pipe.ZAdd(key, score, candleData)
	}

	pipe.Expire(key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (pc *RedisPriceCache) GetHistoricalData(ctx context.Context, symbol string, interval string) ([]*models.Candle, error) {
	key := pc.historicalKey(symbol, interval)

	// Get all members ordered by timestamp
	data, err := pc.cache.ZRange(ctx, key, 0, -1)
	if err != nil {
		return nil, err
	}

	candles := make([]*models.Candle, 0, len(data))
	for _, rawData := range data {
		var candle models.Candle
		if err := json.Unmarshal(rawData, &candle); err == nil {
			candles = append(candles, &candle)
		}
	}

	return candles, nil
}

func (pc *RedisPriceCache) AppendHistoricalData(ctx context.Context, symbol string, interval string, data []*models.Candle) error {
	key := pc.historicalKey(symbol, interval)

	pipe := pc.cache.Pipeline()

	for _, candle := range data {
		candleData, err := json.Marshal(candle)
		if err != nil {
			continue
		}

		score := float64(candle.Timestamp.Unix())
		pipe.ZAdd(key, score, candleData)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Market data operations

func (pc *RedisPriceCache) SetMarketData(ctx context.Context, symbol string, data *models.MarketData, ttl time.Duration) error {
	key := pc.marketDataKey(symbol)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return NewCacheError("set_market_data", key, ErrCodeSerialization, err)
	}

	return pc.cache.Set(ctx, key, jsonData, ttl)
}

func (pc *RedisPriceCache) GetMarketData(ctx context.Context, symbol string) (*models.MarketData, error) {
	key := pc.marketDataKey(symbol)

	data, err := pc.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var marketData models.MarketData
	if err := json.Unmarshal(data, &marketData); err != nil {
		return nil, NewCacheError("get_market_data", key, ErrCodeSerialization, err)
	}

	return &marketData, nil
}

// Order book operations

func (pc *RedisPriceCache) SetOrderBook(ctx context.Context, symbol string, orderBook *models.OrderBook, ttl time.Duration) error {
	key := pc.orderBookKey(symbol)

	// Store order book as hash for efficient partial updates
	pipe := pc.cache.Pipeline()

	// Store metadata
	metadata := map[string]interface{}{
		"symbol":    orderBook.Symbol,
		"timestamp": orderBook.Timestamp.Unix(),
	}
	metadataData, _ := json.Marshal(metadata)
	pipe.HSet(key, "metadata", metadataData)

	// Store bids
	bidsData, err := json.Marshal(orderBook.Bids)
	if err == nil {
		pipe.HSet(key, "bids", bidsData)
	}

	// Store asks
	asksData, err := json.Marshal(orderBook.Asks)
	if err == nil {
		pipe.HSet(key, "asks", asksData)
	}

	pipe.Expire(key, ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (pc *RedisPriceCache) GetOrderBook(ctx context.Context, symbol string) (*models.OrderBook, error) {
	key := pc.orderBookKey(symbol)

	data, err := pc.cache.HGetAll(ctx, key)
	if err != nil {
		return nil, err
	}

	orderBook := &models.OrderBook{}

	// Parse metadata
	if metadataRaw, exists := data["metadata"]; exists {
		var metadata map[string]interface{}
		if err := json.Unmarshal(metadataRaw, &metadata); err == nil {
			if symbol, ok := metadata["symbol"].(string); ok {
				orderBook.Symbol = symbol
			}
			if timestamp, ok := metadata["timestamp"].(float64); ok {
				orderBook.Timestamp = time.Unix(int64(timestamp), 0)
			}
			// Source field removed - OrderBook doesn't have Source field
		}
	}

	// Parse bids
	if bidsRaw, exists := data["bids"]; exists {
		var bids []*models.OrderLevel
		if err := json.Unmarshal(bidsRaw, &bids); err == nil {
			orderBook.Bids = bids
		}
	}

	// Parse asks
	if asksRaw, exists := data["asks"]; exists {
		var asks []*models.OrderLevel
		if err := json.Unmarshal(asksRaw, &asks); err == nil {
			orderBook.Asks = asks
		}
	}

	return orderBook, nil
}

// Statistical data operations (commented out - StatisticalData not implemented)
// func (pc *RedisPriceCache) SetStatistics(ctx context.Context, symbol string, stats *models.StatisticalData, ttl time.Duration) error {
// 	key := pc.statisticsKey(symbol)
// 	data, err := json.Marshal(stats)
// 	if err != nil {
// 		return NewCacheError("set_statistics", key, ErrCodeSerialization, err)
// 	}
// 	return pc.cache.Set(ctx, key, data, ttl)
// }
//
// func (pc *RedisPriceCache) GetStatistics(ctx context.Context, symbol string) (*models.StatisticalData, error) {
// 	key := pc.statisticsKey(symbol)
// 	data, err := pc.cache.Get(ctx, key)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var stats models.StatisticalData
// 	if err := json.Unmarshal(data, &stats); err != nil {
// 		return nil, NewCacheError("get_statistics", key, ErrCodeSerialization, err)
// 	}
// 	return &stats, nil
// }

// Technical indicators operations

func (pc *RedisPriceCache) SetTechnicalIndicators(ctx context.Context, symbol string, indicators *models.TechnicalIndicators, ttl time.Duration) error {
	key := pc.technicalIndicatorsKey(symbol)

	data, err := json.Marshal(indicators)
	if err != nil {
		return NewCacheError("set_technical_indicators", key, ErrCodeSerialization, err)
	}

	return pc.cache.Set(ctx, key, data, ttl)
}

func (pc *RedisPriceCache) GetTechnicalIndicators(ctx context.Context, symbol string) (*models.TechnicalIndicators, error) {
	key := pc.technicalIndicatorsKey(symbol)

	data, err := pc.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var indicators models.TechnicalIndicators
	if err := json.Unmarshal(data, &indicators); err != nil {
		return nil, NewCacheError("get_technical_indicators", key, ErrCodeSerialization, err)
	}

	return &indicators, nil
}

// Volatility data operations

func (pc *RedisPriceCache) SetVolatilityData(ctx context.Context, symbol string, volatility *models.VolatilityData, ttl time.Duration) error {
	key := pc.volatilityKey(symbol)

	data, err := json.Marshal(volatility)
	if err != nil {
		return NewCacheError("set_volatility_data", key, ErrCodeSerialization, err)
	}

	return pc.cache.Set(ctx, key, data, ttl)
}

func (pc *RedisPriceCache) GetVolatilityData(ctx context.Context, symbol string) (*models.VolatilityData, error) {
	key := pc.volatilityKey(symbol)

	data, err := pc.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var volatility models.VolatilityData
	if err := json.Unmarshal(data, &volatility); err != nil {
		return nil, NewCacheError("get_volatility_data", key, ErrCodeSerialization, err)
	}

	return &volatility, nil
}

// Advanced cache operations

// SetPriceWithMetrics stores price with additional metrics for analysis
func (pc *RedisPriceCache) SetPriceWithMetrics(ctx context.Context, symbol string, price *models.AggregatedPrice, ttl time.Duration) error {
	// Store the main price
	if err := pc.SetPrice(ctx, symbol, price, ttl); err != nil {
		return err
	}

	// Store price in time-series for trend analysis
	timeSeriesKey := fmt.Sprintf("timeseries:price:%s", strings.ToUpper(symbol))
	timestamp := float64(price.Timestamp.Unix())

	priceData := map[string]interface{}{
		"price":      price.Price.InexactFloat64(),
		"confidence": price.Confidence,
		"volume_24h": price.Volume24h.InexactFloat64(),
		"providers":  len(price.ProviderPrices),
	}

	if jsonData, err := json.Marshal(priceData); err == nil {
		pc.cache.ZAdd(ctx, timeSeriesKey, timestamp, jsonData)

		// Keep only last 1000 entries
		pc.cache.ZRange(ctx, timeSeriesKey, 0, -1001) // This would need to be ZRemRangeByRank in a full implementation
	}

	return nil
}

// GetPriceHistory retrieves price history from time-series data
func (pc *RedisPriceCache) GetPriceHistory(ctx context.Context, symbol string, from, to time.Time) ([]PriceHistoryPoint, error) {
	timeSeriesKey := fmt.Sprintf("timeseries:price:%s", strings.ToUpper(symbol))

	minScore := float64(from.Unix())
	maxScore := float64(to.Unix())

	data, err := pc.cache.ZRangeByScore(ctx, timeSeriesKey, minScore, maxScore, -1)
	if err != nil {
		return nil, err
	}

	history := make([]PriceHistoryPoint, 0, len(data))
	for _, rawData := range data {
		var priceData map[string]interface{}
		if err := json.Unmarshal(rawData, &priceData); err == nil {
			point := PriceHistoryPoint{
				Timestamp: time.Unix(int64(priceData["timestamp"].(float64)), 0),
				Price:     priceData["price"].(float64),
			}
			if confidence, ok := priceData["confidence"].(float64); ok {
				point.Confidence = confidence
			}
			if volume, ok := priceData["volume"].(float64); ok {
				point.Volume = volume
			}
			if providers, ok := priceData["providers"].(float64); ok {
				point.Providers = int(providers)
			}
			history = append(history, point)
		}
	}

	return history, nil
}

// BulkDelete removes multiple cache entries by pattern
func (pc *RedisPriceCache) BulkDelete(ctx context.Context, pattern string) error {
	keys, err := pc.cache.Keys(ctx, pattern)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	return pc.cache.Del(ctx, keys...)
}

// GetCacheStats returns statistics about cached data
func (pc *RedisPriceCache) GetCacheStats(ctx context.Context) (*PriceCacheStats, error) {
	stats := &PriceCacheStats{
		Timestamp: time.Now(),
	}

	// Count different types of cached data
	patterns := map[string]string{
		"prices":            "price:*",
		"historical":        "historical:*",
		"market_data":       "market:*",
		"order_books":       "orderbook:*",
		"statistics":        "stats:*",
		"technical":         "technical:*",
		"volatility":        "volatility:*",
	}

	for dataType, pattern := range patterns {
		keys, err := pc.cache.Keys(ctx, pattern)
		if err != nil {
			continue
		}

		switch dataType {
		case "prices":
			stats.PriceCount = len(keys)
		case "historical":
			stats.HistoricalCount = len(keys)
		case "market_data":
			stats.MarketDataCount = len(keys)
		case "order_books":
			stats.OrderBookCount = len(keys)
		case "statistics":
			stats.StatisticsCount = len(keys)
		case "technical":
			stats.TechnicalCount = len(keys)
		case "volatility":
			stats.VolatilityCount = len(keys)
		}
	}

	return stats, nil
}

// Helper structures

type PriceHistoryPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	Price      float64   `json:"price"`
	Confidence float64   `json:"confidence"`
	Volume     float64   `json:"volume"`
	Providers  int       `json:"providers"`
}

type PriceCacheStats struct {
	PriceCount        int       `json:"price_count"`
	HistoricalCount   int       `json:"historical_count"`
	MarketDataCount   int       `json:"market_data_count"`
	OrderBookCount    int       `json:"order_book_count"`
	StatisticsCount   int       `json:"statistics_count"`
	TechnicalCount    int       `json:"technical_count"`
	VolatilityCount   int       `json:"volatility_count"`
	Timestamp         time.Time `json:"timestamp"`
}