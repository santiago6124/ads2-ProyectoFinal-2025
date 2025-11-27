package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"market-data-api/internal/config"
	"market-data-api/internal/models"
)

// Simple cache entry
type CachedPrice struct {
	Data      []byte
	Timestamp time.Time
}

// Server holds all dependencies
type Server struct {
	router      *gin.Engine
	port        int
	coingecko   *FreeCryptoClient
	config      *config.Config
	redisClient *redis.Client
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Get port from environment or use default
	port := cfg.Server.Port
	if port == 0 {
		port = 8004
	}

	// Set Gin mode
	env := cfg.Environment
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize FreeCrypto provider
	apiKey := "ir4h8w22gcaa9nfgijoc" // FreeCryptoAPI key
	coingeckoClient := NewFreeCryptoClient(apiKey)

	// Initialize Redis client
	log.Println("Initializing Redis cache...")
	redisURL := cfg.Redis.URL
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	// Parse Redis URL
	redisHost := "localhost"
	redisPort := 6379
	if strings.HasPrefix(redisURL, "redis://") {
		urlParts := strings.TrimPrefix(redisURL, "redis://")
		hostPort := strings.Split(urlParts, ":")
		if len(hostPort) >= 1 {
			redisHost = hostPort[0]
		}
		if len(hostPort) >= 2 {
			fmt.Sscanf(hostPort[1], "%d", &redisPort)
		}
	}

	// Create simple Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisHost, redisPort),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		log.Println("Continuing without cache...")
		redisClient = nil
	} else {
		log.Println("Redis cache initialized successfully")
	}

	// Initialize server
	srv := &Server{
		router:      gin.Default(),
		port:        port,
		coingecko:   coingeckoClient,
		config:      cfg,
		redisClient: redisClient,
	}

	// Setup routes
	srv.setupRoutes()

	// Start HTTP server
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("Market Data API starting on %s (environment: %s)", addr, env)
	log.Printf("Using FreeCryptoAPI")

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      srv.router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func (s *Server) setupRoutes() {
	// Add CORS middleware
	s.router.Use(corsMiddleware())

	// Health check endpoint
	s.router.GET("/health", s.handleHealth)

	// API v1 routes
	api := s.router.Group("/api/v1")
	{
		// Price endpoints
		api.GET("/prices", s.handleGetPrices)
		api.GET("/prices/:symbol", s.handleGetPriceBySymbol)

		// History endpoint
		api.GET("/history/:symbol", s.handleGetPriceHistory)

		// Market endpoints
		api.GET("/market/stats", s.handleGetMarketStats)
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	// Check provider health
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	providerStatus := "healthy"
	if err := s.coingecko.Ping(ctx); err != nil {
		providerStatus = "degraded"
	}

	cacheEnabled := s.redisClient != nil
	cacheStatus := "disabled"
	if cacheEnabled {
		if err := s.redisClient.Ping(ctx).Err(); err != nil {
			cacheStatus = "error"
		} else {
			cacheStatus = "healthy"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          "healthy",
		"timestamp":       time.Now().Unix(),
		"service":         "market-data-api",
		"provider_status": providerStatus,
		"cache_enabled":   cacheEnabled,
		"cache_status":    cacheStatus,
	})
}

func (s *Server) handleGetPrices(c *gin.Context) {
	// Get symbols from query param or use popular ones
	symbolsParam := c.Query("symbols")
	var symbols []string

	if symbolsParam != "" {
		symbols = strings.Split(symbolsParam, ",")
		for i := range symbols {
			symbols[i] = strings.ToUpper(strings.TrimSpace(symbols[i]))
		}
	} else {
		// Default: return popular cryptocurrencies
		symbols = []string{"BTC", "ETH", "BNB", "SOL", "ADA", "XRP", "DOT", "DOGE", "AVAX", "MATIC"}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cacheTTL := 5 * time.Second   // Fresh cache - 5 seconds
	backupTTL := 24 * time.Hour   // Backup cache - 24 hours for fallback

	response := make([]gin.H, 0, len(symbols))
	cachedCount := 0
	fetchedSymbols := make([]string, 0)

	// Try to get prices from fresh cache first
	if s.redisClient != nil {
		for _, symbol := range symbols {
			cacheKey := fmt.Sprintf("price:%s", symbol)
			cachedData, err := s.redisClient.Get(ctx, cacheKey).Result()

			if err == nil && cachedData != "" {
				var priceData Price
				if json.Unmarshal([]byte(cachedData), &priceData) == nil {
					response = append(response, gin.H{
						"symbol":     priceData.Symbol,
						"name":       priceData.Name,
						"price":      priceData.Price,
						"change_24h": priceData.Change24h,
						"market_cap": priceData.MarketCap,
						"volume":     priceData.Volume,
						"timestamp":  priceData.Timestamp,
						"cached":     true,
					})
					cachedCount++
				} else {
					fetchedSymbols = append(fetchedSymbols, symbol)
				}
			} else {
				fetchedSymbols = append(fetchedSymbols, symbol)
			}
		}
	} else {
		fetchedSymbols = symbols
	}

	// Fetch missing prices from FreeCryptoAPI
	if len(fetchedSymbols) > 0 {
		prices, err := s.coingecko.GetPrices(ctx, fetchedSymbols)
		if err != nil {
			// API failed - try backup cache for missing symbols
			if s.redisClient != nil {
				for _, symbol := range fetchedSymbols {
					backupKey := fmt.Sprintf("price:%s:backup", symbol)
					backupData, backupErr := s.redisClient.Get(ctx, backupKey).Result()
					if backupErr == nil && backupData != "" {
						var priceData Price
						if json.Unmarshal([]byte(backupData), &priceData) == nil {
							response = append(response, gin.H{
								"symbol":     priceData.Symbol,
								"name":       priceData.Name,
								"price":      priceData.Price,
								"change_24h": priceData.Change24h,
								"market_cap": priceData.MarketCap,
								"volume":     priceData.Volume,
								"timestamp":  priceData.Timestamp,
								"cached":     true,
								"stale":      true,
							})
							cachedCount++
						}
					}
				}
			}

			// If we have any data (fresh or backup), return it
			if len(response) > 0 {
				log.Printf("API failed, returning %d cached prices (some may be stale)", len(response))
				c.JSON(http.StatusOK, gin.H{
					"data":         response,
					"source":       "cache",
					"count":        len(response),
					"cached_count": cachedCount,
					"warning":      "API unavailable - returning cached data",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch prices",
				"message": err.Error(),
			})
			return
		}

		// Convert and cache fetched prices
		for symbol, price := range prices {
			response = append(response, gin.H{
				"symbol":     price.Symbol,
				"name":       price.Name,
				"price":      price.Price,
				"change_24h": price.Change24h,
				"market_cap": price.MarketCap,
				"volume":     price.Volume,
				"timestamp":  price.Timestamp,
				"cached":     false,
			})

			// Store in both caches
			if s.redisClient != nil {
				cacheKey := fmt.Sprintf("price:%s", symbol)
				backupKey := fmt.Sprintf("price:%s:backup", symbol)
				priceJSON, err := json.Marshal(price)
				if err == nil {
					// Fresh cache (5 seconds)
					s.redisClient.Set(ctx, cacheKey, priceJSON, cacheTTL)
					// Backup cache (24 hours)
					s.redisClient.Set(ctx, backupKey, priceJSON, backupTTL)
				}
			}
		}
	}

	sourceType := "freecryptoapi"
	if cachedCount > 0 && len(fetchedSymbols) > 0 {
		sourceType = "mixed"
	} else if cachedCount == len(symbols) {
		sourceType = "cache"
	}

	c.JSON(http.StatusOK, gin.H{
		"data":         response,
		"source":       sourceType,
		"count":        len(response),
		"cached_count": cachedCount,
	})
}

func (s *Server) handleGetPriceBySymbol(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cacheTTL := 5 * time.Second       // Fresh cache - 5 seconds
	backupTTL := 24 * time.Hour       // Backup cache - 24 hours for fallback

	cacheKey := fmt.Sprintf("price:%s", symbol)
	backupKey := fmt.Sprintf("price:%s:backup", symbol)

	// Try to get price from fresh cache first
	if s.redisClient != nil {
		cachedData, err := s.redisClient.Get(ctx, cacheKey).Result()
		if err == nil && cachedData != "" {
			var priceData Price
			if json.Unmarshal([]byte(cachedData), &priceData) == nil {
				c.JSON(http.StatusOK, gin.H{
					"symbol":     priceData.Symbol,
					"name":       priceData.Name,
					"price":      priceData.Price,
					"change_24h": priceData.Change24h,
					"market_cap": priceData.MarketCap,
					"volume":     priceData.Volume,
					"timestamp":  priceData.Timestamp,
					"cached":     true,
				})
				return
			}
		}
	}

	// Fetch from FreeCryptoAPI
	price, err := s.coingecko.GetPrice(ctx, symbol)
	if err != nil {
		// API failed - try backup cache
		if s.redisClient != nil {
			backupData, backupErr := s.redisClient.Get(ctx, backupKey).Result()
			if backupErr == nil && backupData != "" {
				var priceData Price
				if json.Unmarshal([]byte(backupData), &priceData) == nil {
					log.Printf("API failed for %s, returning backup cache", symbol)
					c.JSON(http.StatusOK, gin.H{
						"symbol":     priceData.Symbol,
						"name":       priceData.Name,
						"price":      priceData.Price,
						"change_24h": priceData.Change24h,
						"market_cap": priceData.MarketCap,
						"volume":     priceData.Volume,
						"timestamp":  priceData.Timestamp,
						"cached":     true,
						"stale":      true,
					})
					return
				}
			}
		}
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Price not found",
			"message": err.Error(),
		})
		return
	}

	// Store in both caches
	if s.redisClient != nil {
		priceJSON, err := json.Marshal(price)
		if err == nil {
			// Fresh cache (5 seconds)
			s.redisClient.Set(ctx, cacheKey, priceJSON, cacheTTL)
			// Backup cache (24 hours)
			s.redisClient.Set(ctx, backupKey, priceJSON, backupTTL)
			log.Printf("Cached price for %s (fresh: %v, backup: %v)", symbol, cacheTTL, backupTTL)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":     price.Symbol,
		"name":       price.Name,
		"price":      price.Price,
		"change_24h": price.Change24h,
		"market_cap": price.MarketCap,
		"volume":     price.Volume,
		"timestamp":  price.Timestamp,
		"cached":     false,
	})
}

func (s *Server) handleGetPriceHistory(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	// Get interval from query params (default to 1h)
	interval := c.DefaultQuery("interval", "1h")

	// Get limit from query params
	limitStr := c.DefaultQuery("limit", "")
	var limit int
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	// Calculate time range based on interval
	now := time.Now()
	var from time.Time
	var to time.Time = now

	switch interval {
	case "1m":
		from = now.Add(-60 * time.Minute)
		if limit == 0 {
			limit = 60
		}
	case "5m":
		from = now.Add(-5 * time.Hour)
		if limit == 0 {
			limit = 60
		}
	case "15m":
		from = now.Add(-24 * time.Hour)
		if limit == 0 {
			limit = 96
		}
	case "1h":
		from = now.Add(-24 * time.Hour)
		if limit == 0 {
			limit = 24
		}
	case "4h":
		from = now.Add(-7 * 24 * time.Hour)
		if limit == 0 {
			limit = 42
		}
	case "1d":
		from = now.Add(-30 * 24 * time.Hour)
		if limit == 0 {
			limit = 30
		}
	case "1w":
		from = now.Add(-52 * 7 * 24 * time.Hour)
		if limit == 0 {
			limit = 52
		}
	default:
		from = now.Add(-24 * time.Hour)
		if limit == 0 {
			limit = 24
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	// Fetch historical data from FreeCryptoAPI
	candles, err := s.coingecko.GetHistoricalData(ctx, symbol, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch historical data",
			"message": err.Error(),
		})
		return
	}

	// Convert candles to response format
	history := make([]gin.H, 0, len(candles))
	for _, candle := range candles {
		history = append(history, gin.H{
			"timestamp": candle.Timestamp,
			"price":     candle.Close,
			"open":      candle.Open,
			"high":      candle.High,
			"low":       candle.Low,
			"volume":    candle.Volume,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":  symbol,
		"history": history,
		"source":  "coingecko",
	})
}

func (s *Server) handleGetMarketStats(c *gin.Context) {
	// Get popular cryptocurrencies for market stats
	popularSymbols := []string{"BTC", "ETH", "BNB", "SOL", "ADA", "XRP", "DOT", "DOGE", "AVAX", "MATIC"}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	// Fetch prices for popular cryptos
	prices, err := s.coingecko.GetPrices(ctx, popularSymbols)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch market stats",
			"message": err.Error(),
		})
		return
	}

	// Calculate market statistics
	var totalMarketCap float64
	var totalVolume float64
	var btcMarketCap float64
	var ethMarketCap float64

	for symbol, price := range prices {
		if price.MarketCap > 0 {
			totalMarketCap += price.MarketCap
		}
		if price.Volume > 0 {
			totalVolume += price.Volume
		}
		if symbol == "BTC" {
			btcMarketCap = price.MarketCap
		}
		if symbol == "ETH" {
			ethMarketCap = price.MarketCap
		}
	}

	var btcDominance float64
	var ethDominance float64
	if totalMarketCap > 0 {
		btcDominance = (btcMarketCap / totalMarketCap) * 100
		ethDominance = (ethMarketCap / totalMarketCap) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"totalMarketCap": totalMarketCap,
		"totalVolume24h": totalVolume,
		"btcDominance":   btcDominance,
		"ethDominance":   ethDominance,
		"activeCryptos":  len(prices),
		"timestamp":      time.Now().Unix(),
		"source":         "freecryptoapi",
	})
}

// Helper functions

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func priceToResponse(price *models.Price) gin.H {
	response := gin.H{
		"symbol":    price.Symbol,
		"name":      getCryptoName(price.Symbol),
		"price":     price.Price.String(),
		"timestamp": price.Timestamp.Unix(),
		"source":    price.Source,
	}

	if price.Change24h.GreaterThan(decimal.Zero) || price.Change24h.LessThan(decimal.Zero) {
		response["change_24h"] = price.Change24h.String()
	}

	if price.MarketCap.GreaterThan(decimal.Zero) {
		response["market_cap"] = price.MarketCap.String()
	}

	if price.Volume24h.GreaterThan(decimal.Zero) {
		response["volume"] = price.Volume24h.String()
	}

	return response
}

func getCryptoName(symbol string) string {
	names := map[string]string{
		"BTC":   "Bitcoin",
		"ETH":   "Ethereum",
		"BNB":   "Binance Coin",
		"SOL":   "Solana",
		"ADA":   "Cardano",
		"XRP":   "Ripple",
		"DOT":   "Polkadot",
		"DOGE":  "Dogecoin",
		"AVAX":  "Avalanche",
		"MATIC": "Polygon",
		"LINK":  "Chainlink",
		"UNI":   "Uniswap",
		"ATOM":  "Cosmos",
		"LTC":   "Litecoin",
		"ETC":   "Ethereum Classic",
		"XLM":   "Stellar",
		"ALGO":  "Algorand",
		"VET":   "VeChain",
		"ICP":   "Internet Computer",
		"FIL":   "Filecoin",
		"AAVE":  "Aave",
		"GRT":   "The Graph",
		"THETA": "Theta Network",
		"SAND":  "The Sandbox",
		"MANA":  "Decentraland",
		"AXS":   "Axie Infinity",
		"CHZ":   "Chiliz",
		"ENJ":   "Enjin Coin",
		"ZIL":   "Zilliqa",
		"BAT":   "Basic Attention Token",
		"COMP":  "Compound",
		"YFI":   "yearn.finance",
		"SNX":   "Synthetix",
		"MKR":   "Maker",
		"SUSHI": "SushiSwap",
		"CRV":   "Curve DAO Token",
		"1INCH": "1inch",
		"CAKE":  "PancakeSwap",
		"RUNE":  "THORChain",
		"KSM":   "Kusama",
		"ZEC":   "Zcash",
		"DASH":  "Dash",
		"WAVES": "Waves",
		"QTUM":  "Qtum",
		"ONT":   "Ontology",
		"ZRX":   "0x",
		"CELO":  "Celo",
		"HBAR":  "Hedera",
		"KLAY":  "Klaytn",
		"NEAR":  "NEAR Protocol",
	}

	if name, ok := names[symbol]; ok {
		return name
	}
	return symbol
}
