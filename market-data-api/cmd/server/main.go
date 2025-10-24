package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Server holds all dependencies
type Server struct {
	router *gin.Engine
	port   int
}

// Popular cryptocurrencies with realistic data
var cryptoData = map[string]CryptoInfo{
	"BTC":   {Name: "Bitcoin", BasePrice: 110764.70, Volatility: 0.02},
	"ETH":   {Name: "Ethereum", BasePrice: 3930.00, Volatility: 0.03},
	"BNB":   {Name: "Binance Coin", BasePrice: 710.50, Volatility: 0.025},
	"SOL":   {Name: "Solana", BasePrice: 193.41, Volatility: 0.04},
	"ADA":   {Name: "Cardano", BasePrice: 1.12, Volatility: 0.035},
	"XRP":   {Name: "Ripple", BasePrice: 2.45, Volatility: 0.03},
	"DOT":   {Name: "Polkadot", BasePrice: 28.50, Volatility: 0.035},
	"DOGE":  {Name: "Dogecoin", BasePrice: 0.35, Volatility: 0.05},
	"AVAX":  {Name: "Avalanche", BasePrice: 125.30, Volatility: 0.04},
	"MATIC": {Name: "Polygon", BasePrice: 2.15, Volatility: 0.04},
	"LINK":  {Name: "Chainlink", BasePrice: 24.80, Volatility: 0.035},
	"UNI":   {Name: "Uniswap", BasePrice: 18.50, Volatility: 0.04},
	"ATOM":  {Name: "Cosmos", BasePrice: 32.10, Volatility: 0.035},
	"LTC":   {Name: "Litecoin", BasePrice: 215.00, Volatility: 0.025},
	"ETC":   {Name: "Ethereum Classic", BasePrice: 45.20, Volatility: 0.03},
	"XLM":   {Name: "Stellar", BasePrice: 0.38, Volatility: 0.04},
	"ALGO":  {Name: "Algorand", BasePrice: 1.25, Volatility: 0.04},
	"VET":   {Name: "VeChain", BasePrice: 0.085, Volatility: 0.045},
	"ICP":   {Name: "Internet Computer", BasePrice: 35.80, Volatility: 0.05},
	"FIL":   {Name: "Filecoin", BasePrice: 18.90, Volatility: 0.04},
	"AAVE":  {Name: "Aave", BasePrice: 285.00, Volatility: 0.04},
	"GRT":   {Name: "The Graph", BasePrice: 0.65, Volatility: 0.045},
	"THETA": {Name: "Theta Network", BasePrice: 3.20, Volatility: 0.04},
	"SAND":  {Name: "The Sandbox", BasePrice: 2.85, Volatility: 0.05},
	"MANA":  {Name: "Decentraland", BasePrice: 2.10, Volatility: 0.05},
	"AXS":   {Name: "Axie Infinity", BasePrice: 45.50, Volatility: 0.06},
	"CHZ":   {Name: "Chiliz", BasePrice: 0.28, Volatility: 0.045},
	"ENJ":   {Name: "Enjin Coin", BasePrice: 1.85, Volatility: 0.04},
	"ZIL":   {Name: "Zilliqa", BasePrice: 0.095, Volatility: 0.045},
	"BAT":   {Name: "Basic Attention Token", BasePrice: 0.68, Volatility: 0.04},
	"COMP":  {Name: "Compound", BasePrice: 175.00, Volatility: 0.045},
	"YFI":   {Name: "yearn.finance", BasePrice: 28500.00, Volatility: 0.05},
	"SNX":   {Name: "Synthetix", BasePrice: 12.50, Volatility: 0.045},
	"MKR":   {Name: "Maker", BasePrice: 3200.00, Volatility: 0.04},
	"SUSHI": {Name: "SushiSwap", BasePrice: 8.50, Volatility: 0.045},
	"CRV":   {Name: "Curve DAO Token", BasePrice: 3.80, Volatility: 0.04},
	"1INCH": {Name: "1inch", BasePrice: 1.45, Volatility: 0.045},
	"CAKE":  {Name: "PancakeSwap", BasePrice: 8.20, Volatility: 0.045},
	"RUNE":  {Name: "THORChain", BasePrice: 15.80, Volatility: 0.05},
	"KSM":   {Name: "Kusama", BasePrice: 95.00, Volatility: 0.04},
	"ZEC":   {Name: "Zcash", BasePrice: 125.00, Volatility: 0.03},
	"DASH":  {Name: "Dash", BasePrice: 85.00, Volatility: 0.035},
	"WAVES": {Name: "Waves", BasePrice: 12.50, Volatility: 0.04},
	"QTUM":  {Name: "Qtum", BasePrice: 9.80, Volatility: 0.04},
	"ONT":   {Name: "Ontology", BasePrice: 1.95, Volatility: 0.04},
	"ZRX":   {Name: "0x", BasePrice: 1.25, Volatility: 0.045},
	"CELO":  {Name: "Celo", BasePrice: 3.50, Volatility: 0.04},
	"HBAR":  {Name: "Hedera", BasePrice: 0.28, Volatility: 0.045},
	"KLAY":  {Name: "Klaytn", BasePrice: 1.15, Volatility: 0.04},
	"NEAR":  {Name: "NEAR Protocol", BasePrice: 18.50, Volatility: 0.045},
}

type CryptoInfo struct {
	Name       string
	BasePrice  float64
	Volatility float64
}

func main() {
	// Seed random for price variations
	rand.Seed(time.Now().UnixNano())

	// Get port from environment or use default
	port := 8004
	if portEnv := os.Getenv("SERVER_PORT"); portEnv != "" {
		fmt.Sscanf(portEnv, "%d", &port)
	}

	// Set Gin mode
	env := os.Getenv("ENVIRONMENT")
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize server
	srv := &Server{
		router: gin.Default(),
		port:   port,
	}

	// Setup routes
	srv.setupRoutes()

	// Start HTTP server
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("Market Data API starting on %s (environment: %s)", addr, env)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      srv.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	s.router.GET("/health", handleHealth)

	// API v1 routes
	api := s.router.Group("/api/v1")
	{
		// Price endpoints
		api.GET("/prices", handleGetPrices)
		api.GET("/prices/:symbol", handleGetPriceBySymbol)

		// History endpoint
		api.GET("/history/:symbol", handleGetPriceHistory)

		// Market endpoints
		api.GET("/market/stats", handleGetMarketStats)
	}
}

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "market-data-api",
	})
}

func handleGetPrices(c *gin.Context) {
	// Get symbols from query param or use all
	symbolsParam := c.Query("symbols")
	var symbols []string

	if symbolsParam != "" {
		symbols = strings.Split(symbolsParam, ",")
		for i := range symbols {
			symbols[i] = strings.ToUpper(strings.TrimSpace(symbols[i]))
		}
	} else {
		// Return all cryptocurrencies
		for symbol := range cryptoData {
			symbols = append(symbols, symbol)
		}
	}

	// Generate prices
	prices := generatePrices(symbols)

	c.JSON(http.StatusOK, gin.H{
		"data":   prices,
		"source": "live",
		"count":  len(prices),
	})
}

func handleGetPriceBySymbol(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	// Generate price
	price := generatePrice(symbol)

	c.JSON(http.StatusOK, price)
}

func handleGetPriceHistory(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	// Generate history
	history := generateHistory(symbol)

	c.JSON(http.StatusOK, history)
}

func handleGetMarketStats(c *gin.Context) {
	// Generate stats
	stats := generateMarketStats()

	c.JSON(http.StatusOK, stats)
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

func generatePrices(symbols []string) []gin.H {
	prices := make([]gin.H, 0, len(symbols))

	for _, symbol := range symbols {
		_, ok := cryptoData[symbol]
		if !ok {
			continue
		}

		price := generatePrice(symbol)
		prices = append(prices, price)
	}

	return prices
}

func generatePrice(symbol string) gin.H {
	info, ok := cryptoData[symbol]
	if !ok {
		// Unknown symbol, return generic data
		return gin.H{
			"symbol":     symbol,
			"name":       symbol,
			"price":      1000.0,
			"change_24h": randomChange(),
			"market_cap": 1000000000.0,
			"volume":     50000000.0,
			"timestamp":  time.Now().Unix(),
		}
	}

	// Add random variation to base price
	variation := (rand.Float64()*2 - 1) * info.Volatility
	currentPrice := info.BasePrice * (1 + variation)

	// Random 24h change
	change24h := randomChange()

	// Calculate market cap and volume based on price
	marketCap := currentPrice * getCirculatingSupply(symbol)
	volume := marketCap * (0.05 + rand.Float64()*0.15) // 5-20% of market cap

	return gin.H{
		"symbol":     symbol,
		"name":       info.Name,
		"price":      currentPrice,
		"change_24h": change24h,
		"market_cap": marketCap,
		"volume":     volume,
		"timestamp":  time.Now().Unix(),
	}
}

func generateHistory(symbol string) gin.H {
	info, ok := cryptoData[symbol]
	if !ok {
		info = CryptoInfo{Name: symbol, BasePrice: 1000.0, Volatility: 0.03}
	}

	now := time.Now()
	history := make([]gin.H, 24)

	for i := 0; i < 24; i++ {
		timestamp := now.Add(time.Duration(-24+i) * time.Hour)
		// Add hourly variation
		variation := (rand.Float64()*2 - 1) * info.Volatility
		price := info.BasePrice * (1 + variation)

		history[i] = gin.H{
			"timestamp": timestamp.Unix(),
			"price":     price,
		}
	}

	return gin.H{
		"symbol":  symbol,
		"history": history,
	}
}

func generateMarketStats() gin.H {
	totalMarketCap := 0.0
	totalVolume := 0.0

	// Calculate totals from all cryptos
	for symbol, info := range cryptoData {
		variation := (rand.Float64()*2 - 1) * info.Volatility
		currentPrice := info.BasePrice * (1 + variation)
		marketCap := currentPrice * getCirculatingSupply(symbol)
		volume := marketCap * (0.05 + rand.Float64()*0.15)

		totalMarketCap += marketCap
		totalVolume += volume
	}

	btcInfo := cryptoData["BTC"]
	btcPrice := btcInfo.BasePrice
	btcMarketCap := btcPrice * getCirculatingSupply("BTC")
	btcDominance := (btcMarketCap / totalMarketCap) * 100

	ethInfo := cryptoData["ETH"]
	ethPrice := ethInfo.BasePrice
	ethMarketCap := ethPrice * getCirculatingSupply("ETH")
	ethDominance := (ethMarketCap / totalMarketCap) * 100

	return gin.H{
		"totalMarketCap":  totalMarketCap,
		"totalVolume24h":  totalVolume,
		"btcDominance":    btcDominance,
		"ethDominance":    ethDominance,
		"activeCryptos":   len(cryptoData),
		"timestamp":       time.Now().Unix(),
	}
}

func randomChange() float64 {
	// Random change between -10% and +10%
	return (rand.Float64()*20 - 10)
}

func getCirculatingSupply(symbol string) float64 {
	supplies := map[string]float64{
		"BTC":   19000000,
		"ETH":   120000000,
		"BNB":   150000000,
		"SOL":   400000000,
		"ADA":   35000000000,
		"XRP":   50000000000,
		"DOT":   1200000000,
		"DOGE":  140000000000,
		"AVAX":  350000000,
		"MATIC": 9000000000,
		"LINK":  500000000,
		"UNI":   750000000,
		"ATOM":  290000000,
		"LTC":   73000000,
		"ETC":   140000000,
		"XLM":   25000000000,
		"ALGO":  7000000000,
		"VET":   65000000000,
		"ICP":   450000000,
		"FIL":   400000000,
	}

	if supply, ok := supplies[symbol]; ok {
		return supply
	}
	return 1000000000 // Default 1B supply
}
