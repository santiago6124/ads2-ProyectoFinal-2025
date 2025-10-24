package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CoinGecko API structures
type CoinGeckoPrice struct {
	ID                string  `json:"id"`
	Symbol            string  `json:"symbol"`
	Name              string  `json:"name"`
	CurrentPrice      float64 `json:"current_price"`
	MarketCap         float64 `json:"market_cap"`
	TotalVolume       float64 `json:"total_volume"`
	PriceChange24h    float64 `json:"price_change_24h"`
	PriceChange24hPct float64 `json:"price_change_percentage_24h"`
	LastUpdated       string  `json:"last_updated"`
}

type CoinGeckoResponse []CoinGeckoPrice

// Symbol mapping for CoinGecko
var symbolToID = map[string]string{
	"BTC":   "bitcoin",
	"ETH":   "ethereum",
	"ADA":   "cardano",
	"SOL":   "solana",
	"MATIC": "matic-network",
	"AVAX":  "avalanche-2",
}

// Simple cache structure
type CacheEntry struct {
	Data      CoinGeckoResponse
	Timestamp time.Time
}

var priceCache = make(map[string]CacheEntry)

const cacheExpiry = 5 * time.Minute // Cache for 5 minutes

func main() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8004"
	}

	// Set Gin to debug mode for development
	gin.SetMode(gin.DebugMode)

	router := gin.Default()

	// Add CORS middleware - must be before routes
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"service":   "market-data-api",
		})
	})

	// API endpoints
	api := router.Group("/api/v1")
	{
		api.GET("/prices", getPrices)
		api.GET("/prices/:symbol", getPriceBySymbol)
		api.GET("/history/:symbol", getPriceHistory)
	}

	log.Printf("Market Data API starting on 0.0.0.0:%s", port)
	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func getPrices(c *gin.Context) {
	// Get symbols from query parameter or use default
	symbolsParam := c.Query("symbols")
	var symbols []string

	if symbolsParam != "" {
		symbols = strings.Split(symbolsParam, ",")
	} else {
		// Default symbols
		symbols = []string{"BTC", "ETH", "ADA"}
	}

	// Fetch real data from CoinGecko
	data, err := fetchCoinGeckoData(symbols)
	if err != nil {
		log.Printf("Error fetching CoinGecko data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch market data",
		})
		return
	}

	// Convert to our response format
	var response []gin.H
	for _, coin := range data {
		response = append(response, gin.H{
			"symbol":     strings.ToUpper(coin.Symbol),
			"name":       coin.Name,
			"price":      coin.CurrentPrice,
			"change_24h": coin.PriceChange24hPct,
			"market_cap": coin.MarketCap,
			"volume":     coin.TotalVolume,
			"timestamp":  time.Now().Unix(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

func getPriceBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")
	log.Printf("Fetching price for symbol: %s", symbol)

	// Fetch real data from CoinGecko
	data, err := fetchCoinGeckoData([]string{symbol})
	if err != nil {
		log.Printf("Error fetching CoinGecko data for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch market data",
		})
		return
	}

	log.Printf("CoinGecko data received: %+v", data)

	// Find the specific coin
	coin, err := findPriceBySymbol(data, symbol)
	if err != nil {
		log.Printf("Symbol %s not found: %v", symbol, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Symbol not found",
		})
		return
	}

	log.Printf("Found coin: %+v", coin)

	c.JSON(http.StatusOK, gin.H{
		"symbol":     strings.ToUpper(coin.Symbol),
		"name":       coin.Name,
		"price":      coin.CurrentPrice,
		"change_24h": coin.PriceChange24hPct,
		"market_cap": coin.MarketCap,
		"volume":     coin.TotalVolume,
		"timestamp":  time.Now().Unix(),
	})
}

func getPriceHistory(c *gin.Context) {
	symbol := c.Param("symbol")
	c.JSON(http.StatusOK, gin.H{
		"symbol": symbol,
		"history": []gin.H{
			{"timestamp": time.Now().Add(-1 * time.Hour).Unix(), "price": 49500.00},
			{"timestamp": time.Now().Unix(), "price": 50000.00},
		},
	})
}

// Helper function to fetch data from CoinGecko
func fetchCoinGeckoData(symbols []string) (CoinGeckoResponse, error) {
	// Create cache key
	cacheKey := strings.Join(symbols, ",")

	// Check cache first
	if cachedData, found := getCachedData(cacheKey); found {
		log.Printf("Using cached data for symbols: %s", cacheKey)
		return cachedData, nil
	}

	// Build the URL with the required coin IDs
	var ids []string
	for _, symbol := range symbols {
		if id, exists := symbolToID[symbol]; exists {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no valid symbols provided")
	}

	// Create the URL
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=%s&order=market_cap_desc&per_page=100&page=1&sparkline=false",
		fmt.Sprintf("%s", ids[0]))
	for i := 1; i < len(ids); i++ {
		url += fmt.Sprintf(",%s", ids[i])
	}

	log.Printf("Fetching from CoinGecko: %s", url)

	// Make the HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CoinGecko API returned status: %d", resp.StatusCode)
	}

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data CoinGeckoResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	// Cache the data
	setCachedData(cacheKey, data)
	log.Printf("Cached data for symbols: %s", cacheKey)

	return data, nil
}

// Helper function to find price by symbol
func findPriceBySymbol(data CoinGeckoResponse, symbol string) (*CoinGeckoPrice, error) {
	for _, coin := range data {
		if coin.Symbol == strings.ToLower(symbol) {
			return &coin, nil
		}
	}
	return nil, fmt.Errorf("symbol %s not found", symbol)
}

// Cache helper functions
func getCachedData(cacheKey string) (CoinGeckoResponse, bool) {
	entry, exists := priceCache[cacheKey]
	if !exists {
		return nil, false
	}

	if time.Since(entry.Timestamp) > cacheExpiry {
		delete(priceCache, cacheKey)
		return nil, false
	}

	return entry.Data, true
}

func setCachedData(cacheKey string, data CoinGeckoResponse) {
	priceCache[cacheKey] = CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
	}
}
