package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	// "market-data-api/internal/cache" // Cache disabled for now
	"market-data-api/internal/config"
	"market-data-api/internal/models"
)

// Server holds all dependencies
type Server struct {
	router    *gin.Engine
	port      int
	coingecko *FreeCryptoClient
	config    *config.Config
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

	// Cache disabled for now - will be implemented later if needed
	// var cacheManager *cache.Manager

	// Initialize server
	srv := &Server{
		router:    gin.Default(),
		port:      port,
		coingecko: coingeckoClient,
		config:    cfg,
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

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
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

	c.JSON(http.StatusOK, gin.H{
		"status":          "healthy",
		"timestamp":       time.Now().Unix(),
		"service":         "market-data-api",
		"provider_status": providerStatus,
		"cache_enabled":   false,
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

	// Cache will be implemented later if needed
	// For now, we fetch directly from CoinGecko

	// Fetch prices from CoinGecko
	prices, err := s.coingecko.GetPrices(ctx, symbols)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch prices",
			"message": err.Error(),
		})
		return
	}

	// Cache prices (will be implemented later if needed)

	// Convert to response format
	response := make([]gin.H, 0, len(prices))
	for _, price := range prices {
		response = append(response, gin.H{
			"symbol":     price.Symbol,
			"name":       price.Name,
			"price":      price.Price,
			"change_24h": price.Change24h,
			"market_cap": price.MarketCap,
			"volume":     price.Volume,
			"timestamp":  price.Timestamp,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   response,
		"source": "freecryptoapi",
		"count":  len(response),
	})
}

func (s *Server) handleGetPriceBySymbol(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Cache will be implemented later if needed

	// Fetch from CoinGecko
	price, err := s.coingecko.GetPrice(ctx, symbol)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Price not found",
			"message": err.Error(),
		})
		return
	}

	// Cache the price (will be implemented later if needed)

	c.JSON(http.StatusOK, gin.H{
		"symbol":     price.Symbol,
		"name":       price.Name,
		"price":      price.Price,
		"change_24h": price.Change24h,
		"market_cap": price.MarketCap,
		"volume":     price.Volume,
		"timestamp":  price.Timestamp,
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
