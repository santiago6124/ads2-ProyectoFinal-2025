package dto

import (
	"time"

	"github.com/shopspring/decimal"
	"market-data-api/internal/models"
)

// PriceResponse represents the response for a single price request
type PriceResponse struct {
	Success bool       `json:"success"`
	Data    *PriceData `json:"data"`
	Cache   *CacheInfo `json:"cache,omitempty"`
	Error   *ErrorInfo `json:"error,omitempty"`
}

// PriceData represents the price data in the response
type PriceData struct {
	Symbol          string                     `json:"symbol"`
	Price           decimal.Decimal            `json:"price"`
	PriceUSD        decimal.Decimal            `json:"price_usd"`
	Timestamp       int64                      `json:"timestamp"`
	Source          string                     `json:"source"`
	ConfidenceScore float64                    `json:"confidence_score,omitempty"`
	Metadata        *models.AggregationMetadata `json:"metadata,omitempty"`
}

// BatchPricesResponse represents the response for multiple price requests
type BatchPricesResponse struct {
	Success bool              `json:"success"`
	Data    *BatchPricesData  `json:"data"`
	Error   *ErrorInfo        `json:"error,omitempty"`
}

// BatchPricesData represents the batch price data
type BatchPricesData struct {
	Prices    map[string]*BatchPriceItem `json:"prices"`
	Timestamp int64                      `json:"timestamp"`
	Currency  string                     `json:"currency"`
	Source    string                     `json:"source,omitempty"`
}

// BatchPriceItem represents a single price item in batch response
type BatchPriceItem struct {
	Price                 decimal.Decimal `json:"price"`
	Change24h             decimal.Decimal `json:"change_24h,omitempty"`
	Change24hPercentage   decimal.Decimal `json:"change_24h_percentage,omitempty"`
	Volume24h             decimal.Decimal `json:"volume_24h,omitempty"`
	MarketCap             decimal.Decimal `json:"market_cap,omitempty"`
	LastUpdated           int64           `json:"last_updated,omitempty"`
}

// HistoryResponse represents the response for historical data
type HistoryResponse struct {
	Success bool         `json:"success"`
	Data    *HistoryData `json:"data"`
	Error   *ErrorInfo   `json:"error,omitempty"`
}

// HistoryData represents historical price data
type HistoryData struct {
	Symbol   string           `json:"symbol"`
	Interval string           `json:"interval"`
	Candles  []*CandleData    `json:"candles"`
	Metadata *HistoryMetadata `json:"metadata,omitempty"`
}

// CandleData represents a single candlestick data point
type CandleData struct {
	Timestamp int64           `json:"timestamp"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
	Volume    decimal.Decimal `json:"volume"`
	VWAP      decimal.Decimal `json:"vwap,omitempty"`
	Trades    int64           `json:"trades,omitempty"`
}

// HistoryMetadata represents metadata for historical data
type HistoryMetadata struct {
	TotalCandles int         `json:"total_candles"`
	TimeRange    *TimeRange  `json:"time_range"`
	DataQuality  float64     `json:"data_quality,omitempty"`
	Source       string      `json:"source,omitempty"`
}

// TimeRange represents a time range
type TimeRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// StatsResponse represents the response for market statistics
type StatsResponse struct {
	Success bool       `json:"success"`
	Data    *StatsData `json:"data"`
	Error   *ErrorInfo `json:"error,omitempty"`
}

// StatsData represents comprehensive market statistics
type StatsData struct {
	Symbol                      string                     `json:"symbol"`
	CurrentPrice                decimal.Decimal            `json:"current_price"`
	MarketCap                   decimal.Decimal            `json:"market_cap"`
	FullyDilutedValuation       decimal.Decimal            `json:"fully_diluted_valuation,omitempty"`
	TotalVolume                 decimal.Decimal            `json:"total_volume"`
	High24h                     decimal.Decimal            `json:"high_24h"`
	Low24h                      decimal.Decimal            `json:"low_24h"`
	PriceChange24h              decimal.Decimal            `json:"price_change_24h"`
	PriceChangePercentage24h    decimal.Decimal            `json:"price_change_percentage_24h"`
	PriceChangePercentage7d     decimal.Decimal            `json:"price_change_percentage_7d,omitempty"`
	PriceChangePercentage30d    decimal.Decimal            `json:"price_change_percentage_30d,omitempty"`
	PriceChangePercentage1y     decimal.Decimal            `json:"price_change_percentage_1y,omitempty"`
	ATH                         decimal.Decimal            `json:"ath"`
	ATHChangePercentage         decimal.Decimal            `json:"ath_change_percentage"`
	ATHDate                     string                     `json:"ath_date,omitempty"`
	ATL                         decimal.Decimal            `json:"atl"`
	ATLChangePercentage         decimal.Decimal            `json:"atl_change_percentage"`
	ATLDate                     string                     `json:"atl_date,omitempty"`
	CirculatingSupply           decimal.Decimal            `json:"circulating_supply,omitempty"`
	TotalSupply                 decimal.Decimal            `json:"total_supply,omitempty"`
	MaxSupply                   decimal.Decimal            `json:"max_supply,omitempty"`
	MarketMetrics               *MarketMetricsData         `json:"market_metrics,omitempty"`
	LastUpdated                 string                     `json:"last_updated"`
}

// MarketMetricsData represents advanced market metrics
type MarketMetricsData struct {
	Volatility24h           decimal.Decimal `json:"volatility_24h"`
	Volatility7d            decimal.Decimal `json:"volatility_7d"`
	SharpeRatio             decimal.Decimal `json:"sharpe_ratio,omitempty"`
	Beta                    decimal.Decimal `json:"beta,omitempty"`
	CorrelationWithMarket   decimal.Decimal `json:"correlation_with_market,omitempty"`
	MarketDominance         decimal.Decimal `json:"market_dominance,omitempty"`
	LiquidityScore          decimal.Decimal `json:"liquidity_score,omitempty"`
	TurnoverRate            decimal.Decimal `json:"turnover_rate,omitempty"`
}

// VolatilityResponse represents the response for volatility calculations
type VolatilityResponse struct {
	Success bool             `json:"success"`
	Data    *VolatilityData  `json:"data"`
	Error   *ErrorInfo       `json:"error,omitempty"`
}

// VolatilityData represents volatility calculation results
type VolatilityData struct {
	Symbol                string          `json:"symbol"`
	Period                string          `json:"period"`
	Volatility            decimal.Decimal `json:"volatility"`
	VolatilityPercentage  decimal.Decimal `json:"volatility_percentage"`
	StandardDeviation     decimal.Decimal `json:"standard_deviation"`
	Variance              decimal.Decimal `json:"variance"`
	Samples               int             `json:"samples"`
	CalculationMethod     string          `json:"calculation_method"`
	AnnualizedVolatility  decimal.Decimal `json:"annualized_volatility"`
}

// OrderBookResponse represents the response for order book data
type OrderBookResponse struct {
	Success bool           `json:"success"`
	Data    *OrderBookData `json:"data"`
	Error   *ErrorInfo     `json:"error,omitempty"`
}

// OrderBookData represents order book information
type OrderBookData struct {
	Symbol            string              `json:"symbol"`
	Bids              []*OrderLevelData   `json:"bids"`
	Asks              []*OrderLevelData   `json:"asks"`
	Spread            decimal.Decimal     `json:"spread"`
	SpreadPercentage  decimal.Decimal     `json:"spread_percentage"`
	Timestamp         int64               `json:"timestamp"`
	LastUpdate        int64               `json:"last_update,omitempty"`
}

// OrderLevelData represents a single order book level
type OrderLevelData struct {
	Price  decimal.Decimal `json:"price"`
	Amount decimal.Decimal `json:"amount"`
	Total  decimal.Decimal `json:"total,omitempty"`
	Count  int             `json:"count,omitempty"`
}

// CacheInfo represents cache-related information
type CacheInfo struct {
	Hit        bool   `json:"hit"`
	TTLSeconds int64  `json:"ttl_seconds,omitempty"`
	Key        string `json:"key,omitempty"`
	Source     string `json:"source,omitempty"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Success bool        `json:"success"`
	Data    *HealthData `json:"data"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// HealthData represents health status information
type HealthData struct {
	Status     string                 `json:"status"`
	Timestamp  string                 `json:"timestamp"`
	Uptime     string                 `json:"uptime"`
	Version    string                 `json:"version"`
	Services   map[string]*ServiceHealth `json:"services"`
	Metrics    *HealthMetrics         `json:"metrics,omitempty"`
}

// ServiceHealth represents the health of a specific service
type ServiceHealth struct {
	Status      string  `json:"status"`
	Latency     int64   `json:"latency_ms,omitempty"`
	LastCheck   string  `json:"last_check"`
	ErrorCount  int     `json:"error_count,omitempty"`
	SuccessRate float64 `json:"success_rate,omitempty"`
}

// HealthMetrics represents system health metrics
type HealthMetrics struct {
	RequestsPerSecond     float64 `json:"requests_per_second"`
	AverageResponseTime   int64   `json:"avg_response_time_ms"`
	ErrorRate             float64 `json:"error_rate"`
	ActiveConnections     int     `json:"active_connections"`
	CacheHitRate          float64 `json:"cache_hit_rate"`
	MemoryUsage           int64   `json:"memory_usage_bytes"`
	GoRoutines            int     `json:"goroutines"`
}

// ProvidersStatusResponse represents provider status response
type ProvidersStatusResponse struct {
	Success bool                           `json:"success"`
	Data    map[string]*ProviderStatusData `json:"data"`
	Error   *ErrorInfo                     `json:"error,omitempty"`
}

// ProviderStatusData represents the status of a data provider
type ProviderStatusData struct {
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	Latency         int64   `json:"latency_ms"`
	LastUpdate      string  `json:"last_update"`
	ErrorCount      int     `json:"error_count"`
	SuccessRate     float64 `json:"success_rate"`
	RateLimit       int     `json:"rate_limit_remaining,omitempty"`
	Weight          float64 `json:"weight"`
	RequestsToday   int     `json:"requests_today,omitempty"`
}

// BuildPriceResponse builds a price response from aggregated price data
func BuildPriceResponse(price *models.AggregatedPrice, includeMetadata bool, cacheHit bool, cacheKey string, ttl int64) *PriceResponse {
	response := &PriceResponse{
		Success: true,
		Data: &PriceData{
			Symbol:          price.Symbol,
			Price:           price.Price,
			PriceUSD:        price.PriceUSD,
			Timestamp:       price.Timestamp.Unix(),
			Source:          price.Source,
			ConfidenceScore: price.Confidence,
		},
	}

	if includeMetadata && price.Metadata != nil {
		response.Data.Metadata = price.Metadata
	}

	if cacheHit || cacheKey != "" {
		response.Cache = &CacheInfo{
			Hit:        cacheHit,
			Key:        cacheKey,
			TTLSeconds: ttl,
		}
	}

	return response
}

// BuildBatchPricesResponse builds a batch prices response
func BuildBatchPricesResponse(prices map[string]*models.AggregatedPrice, include24hChange, includeVolume bool) *BatchPricesResponse {
	batchPrices := make(map[string]*BatchPriceItem)

	for symbol, price := range prices {
		item := &BatchPriceItem{
			Price:       price.Price,
			LastUpdated: price.Timestamp.Unix(),
		}

		if include24hChange {
			item.Change24h = price.Change24h
			item.Change24hPercentage = price.ChangePercent
		}

		if includeVolume {
			item.Volume24h = price.Volume24h
			item.MarketCap = price.MarketCap
		}

		batchPrices[symbol] = item
	}

	return &BatchPricesResponse{
		Success: true,
		Data: &BatchPricesData{
			Prices:    batchPrices,
			Timestamp: time.Now().Unix(),
			Currency:  "USD",
			Source:    "aggregated",
		},
	}
}

// BuildHistoryResponse builds a history response from candle data
func BuildHistoryResponse(symbol, interval string, candles []*models.Candle, metadata *models.Metadata) *HistoryResponse {
	candleData := make([]*CandleData, len(candles))

	for i, candle := range candles {
		candleData[i] = &CandleData{
			Timestamp: candle.Timestamp.Unix(),
			Open:      candle.Open,
			High:      candle.High,
			Low:       candle.Low,
			Close:     candle.Close,
			Volume:    candle.Volume,
			VWAP:      candle.VWAP,
			Trades:    candle.Trades,
		}
	}

	var historyMetadata *HistoryMetadata
	if metadata != nil {
		historyMetadata = &HistoryMetadata{
			TotalCandles: metadata.TotalCandles,
			DataQuality:  metadata.Quality,
			Source:       metadata.Source,
		}

		if metadata.TimeRange != nil {
			historyMetadata.TimeRange = &TimeRange{
				From: metadata.TimeRange.From.Format(time.RFC3339),
				To:   metadata.TimeRange.To.Format(time.RFC3339),
			}
		}
	}

	return &HistoryResponse{
		Success: true,
		Data: &HistoryData{
			Symbol:   symbol,
			Interval: interval,
			Candles:  candleData,
			Metadata: historyMetadata,
		},
	}
}

// BuildStatsResponse builds a market statistics response
func BuildStatsResponse(marketData *models.MarketData) *StatsResponse {
	response := &StatsResponse{
		Success: true,
		Data: &StatsData{
			Symbol:                   marketData.Symbol,
			CurrentPrice:             marketData.CurrentPrice,
			MarketCap:                marketData.MarketCap,
			FullyDilutedValuation:    marketData.FullyDilutedValuation,
			TotalVolume:              marketData.TotalVolume,
			High24h:                  marketData.High24h,
			Low24h:                   marketData.Low24h,
			PriceChange24h:           marketData.PriceChange24h,
			PriceChangePercentage24h: marketData.PriceChangePercentage24h,
			PriceChangePercentage7d:  marketData.PriceChangePercentage7d,
			PriceChangePercentage30d: marketData.PriceChangePercentage30d,
			PriceChangePercentage1y:  marketData.PriceChangePercentage1y,
			ATH:                      marketData.ATH,
			ATHChangePercentage:      marketData.ATHChangePercentage,
			ATL:                      marketData.ATL,
			ATLChangePercentage:      marketData.ATLChangePercentage,
			CirculatingSupply:        marketData.CirculatingSupply,
			TotalSupply:              marketData.TotalSupply,
			MaxSupply:                marketData.MaxSupply,
			LastUpdated:              marketData.LastUpdated.Format(time.RFC3339),
		},
	}

	if marketData.ATHDate != nil {
		response.Data.ATHDate = marketData.ATHDate.Format(time.RFC3339)
	}

	if marketData.ATLDate != nil {
		response.Data.ATLDate = marketData.ATLDate.Format(time.RFC3339)
	}

	if marketData.MarketMetrics != nil {
		response.Data.MarketMetrics = &MarketMetricsData{
			Volatility24h:         marketData.MarketMetrics.Volatility24h,
			Volatility7d:          marketData.MarketMetrics.Volatility7d,
			SharpeRatio:           marketData.MarketMetrics.SharpeRatio,
			Beta:                  marketData.MarketMetrics.Beta,
			CorrelationWithMarket: marketData.MarketMetrics.CorrelationWithMarket,
			MarketDominance:       marketData.MarketMetrics.MarketDominance,
			LiquidityScore:        marketData.MarketMetrics.LiquidityScore,
			TurnoverRate:          marketData.MarketMetrics.TurnoverRate,
		}
	}

	return response
}

// BuildVolatilityResponse builds a volatility response
func BuildVolatilityResponse(volatilityData *models.VolatilityData) *VolatilityResponse {
	return &VolatilityResponse{
		Success: true,
		Data: &VolatilityData{
			Symbol:                volatilityData.Symbol,
			Period:                volatilityData.Period,
			Volatility:            volatilityData.Volatility,
			VolatilityPercentage:  volatilityData.VolatilityPercentage,
			StandardDeviation:     volatilityData.StandardDeviation,
			Variance:              volatilityData.Variance,
			Samples:               volatilityData.Samples,
			CalculationMethod:     volatilityData.CalculationMethod,
			AnnualizedVolatility:  volatilityData.AnnualizedVolatility,
		},
	}
}

// BuildOrderBookResponse builds an order book response
func BuildOrderBookResponse(orderBook *models.OrderBook) *OrderBookResponse {
	bids := make([]*OrderLevelData, len(orderBook.Bids))
	asks := make([]*OrderLevelData, len(orderBook.Asks))

	for i, bid := range orderBook.Bids {
		bids[i] = &OrderLevelData{
			Price:  bid.Price,
			Amount: bid.Amount,
			Total:  bid.Total,
			Count:  bid.Count,
		}
	}

	for i, ask := range orderBook.Asks {
		asks[i] = &OrderLevelData{
			Price:  ask.Price,
			Amount: ask.Amount,
			Total:  ask.Total,
			Count:  ask.Count,
		}
	}

	return &OrderBookResponse{
		Success: true,
		Data: &OrderBookData{
			Symbol:           orderBook.Symbol,
			Bids:             bids,
			Asks:             asks,
			Spread:           orderBook.Spread,
			SpreadPercentage: orderBook.SpreadPct,
			Timestamp:        orderBook.Timestamp.Unix(),
			LastUpdate:       orderBook.LastUpdate.Unix(),
		},
	}
}

// BuildErrorResponse builds an error response
func BuildErrorResponse(code, message string, details interface{}) interface{} {
	return map[string]interface{}{
		"success": false,
		"error": &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}