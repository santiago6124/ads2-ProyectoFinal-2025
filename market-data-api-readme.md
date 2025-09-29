# 📊 Market Data API - Microservicio de Datos de Mercado

## 📋 Descripción

El microservicio **Market Data API** es el proveedor central de datos de mercado en tiempo real para la plataforma CryptoSim. Se integra con múltiples fuentes externas (CoinGecko, Binance), gestiona WebSockets para actualizaciones en tiempo real, implementa cache con Redis para optimización y proporciona datos históricos y estadísticas de mercado.

## 🎯 Responsabilidades

- **Datos en Tiempo Real**: Precios actuales de criptomonedas con baja latencia
- **Agregación de Fuentes**: Combina datos de múltiples exchanges
- **WebSocket Streaming**: Actualizaciones de precios en tiempo real
- **Datos Históricos**: Almacenamiento y consulta de históricos de precios
- **Estadísticas de Mercado**: Cálculo de métricas (ATH, ATL, volatilidad)
- **Rate Limiting**: Gestión eficiente de límites de API externas
- **Cache Inteligente**: Estrategias de cache adaptativas con Redis
- **Fallback System**: Sistema de respaldo ante fallos de APIs

## 🏗️ Arquitectura

### Estructura del Proyecto
```
market-data-api/
├── cmd/
│   └── main.go                       # Punto de entrada
├── internal/
│   ├── controllers/                  # Controladores HTTP y WebSocket
│   │   ├── price_controller.go
│   │   ├── history_controller.go
│   │   ├── stats_controller.go
│   │   └── websocket_controller.go
│   ├── services/                     # Lógica de negocio
│   │   ├── price_service.go
│   │   ├── aggregator_service.go     # Agregación de múltiples fuentes
│   │   ├── history_service.go
│   │   ├── statistics_service.go
│   │   └── streaming_service.go
│   ├── repositories/                 # Acceso a datos
│   │   ├── redis_repository.go
│   │   └── timeseries_repository.go
│   ├── models/                       # Modelos de dominio
│   │   ├── price.go
│   │   ├── market_data.go
│   │   ├── candle.go
│   │   └── statistics.go
│   ├── dto/                          # Data Transfer Objects
│   │   ├── price_response.go
│   │   ├── history_request.go
│   │   └── ws_message.go
│   ├── providers/                    # Proveedores externos
│   │   ├── provider_interface.go
│   │   ├── coingecko/
│   │   │   ├── client.go
│   │   │   ├── mapper.go
│   │   │   └── rate_limiter.go
│   │   ├── binance/
│   │   │   ├── client.go
│   │   │   ├── websocket.go
│   │   │   └── mapper.go
│   │   └── coinbase/
│   │       ├── client.go
│   │       └── mapper.go
│   ├── websocket/                    # Gestión de WebSockets
│   │   ├── hub.go
│   │   ├── client.go
│   │   ├── pool.go
│   │   └── message_handler.go
│   ├── cache/                        # Estrategias de cache
│   │   ├── redis_cache.go
│   │   ├── cache_warmer.go
│   │   └── ttl_manager.go
│   ├── aggregator/                   # Motor de agregación
│   │   ├── price_aggregator.go
│   │   ├── weighted_average.go
│   │   └── outlier_detector.go
│   ├── scheduler/                    # Tareas programadas
│   │   ├── price_fetcher.go
│   │   ├── history_collector.go
│   │   └── cleanup_job.go
│   ├── middleware/                   # Middlewares
│   │   ├── rate_limit_middleware.go
│   │   ├── cache_middleware.go
│   │   └── logging_middleware.go
│   └── config/                       # Configuración
│       └── config.go
├── pkg/
│   ├── utils/                        # Utilidades
│   │   ├── decimal.go
│   │   ├── time_utils.go
│   │   └── response.go
│   ├── metrics/                      # Métricas Prometheus
│   │   └── metrics.go
│   └── errors/                       # Manejo de errores
│       └── market_errors.go
├── tests/                            # Tests
│   ├── unit/
│   │   └── aggregator_test.go
│   ├── integration/
│   │   └── providers_test.go
│   └── mocks/
│       └── provider_mock.go
├── scripts/                          # Scripts de utilidad
│   ├── warmup_cache.sh
│   └── historical_import.sh
├── docs/                             # Documentación
│   ├── swagger.yaml
│   ├── websocket_protocol.md
│   └── providers_comparison.md
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── .env.example
```

## 💾 Modelo de Datos

### Redis Data Structures

#### 1. Precio Actual (String con TTL)
```redis
# Key: market:price:{symbol}
# TTL: 30 seconds
# Example:
market:price:BTC = {
  "symbol": "BTC",
  "price": 45000.50,
  "timestamp": 1699123456,
  "source": "aggregated",
  "providers": {
    "coingecko": 45010.00,
    "binance": 44995.50,
    "coinbase": 45000.00
  },
  "volume_24h": 25000000000,
  "market_cap": 880000000000,
  "confidence": 0.98
}
```

#### 2. Histórico (Sorted Set)
```redis
# Key: market:history:{interval}:{symbol}
# Score: timestamp
# Member: price data
# Example for 1-minute candles:
market:history:1m:BTC
├── 1699123380 -> {"o": 45000, "h": 45100, "l": 44950, "c": 45050, "v": 1234}
├── 1699123440 -> {"o": 45050, "h": 45080, "l": 45000, "c": 45030, "v": 1567}
└── 1699123500 -> {"o": 45030, "h": 45060, "l": 45020, "c": 45055, "v": 1890}
```

#### 3. Estadísticas (Hash)
```redis
# Key: market:stats:{symbol}
# TTL: 5 minutes
market:stats:BTC
├── high_24h -> 46000
├── low_24h -> 44000
├── volume_24h -> 25000000000
├── price_change_24h -> 2.5
├── price_change_24h_percentage -> 5.7
├── ath -> 69000
├── ath_date -> 2021-11-10
├── atl -> 67.81
├── atl_date -> 2013-07-06
├── volatility_24h -> 0.042
├── market_dominance -> 48.5
```

#### 4. Order Book Snapshot (List)
```redis
# Key: market:orderbook:{symbol}:{side}
# TTL: 5 seconds
market:orderbook:BTC:bids = [
  {"price": 44990, "amount": 2.5},
  {"price": 44985, "amount": 5.0},
  {"price": 44980, "amount": 3.2}
]

market:orderbook:BTC:asks = [
  {"price": 45010, "amount": 1.8},
  {"price": 45015, "amount": 3.5},
  {"price": 45020, "amount": 4.1}
]
```

#### 5. Provider Status (Hash)
```redis
# Key: market:provider:status
market:provider:status
├── coingecko -> {"status": "healthy", "latency": 45, "last_update": 1699123456}
├── binance -> {"status": "healthy", "latency": 12, "last_update": 1699123458}
├── coinbase -> {"status": "degraded", "latency": 250, "last_update": 1699123400}
```

## 🔌 API Endpoints

### Precios en Tiempo Real

#### GET `/api/market/price/:symbol`
Obtiene el precio actual de una criptomoneda.

**Parameters:**
- `symbol`: Símbolo de la criptomoneda (BTC, ETH, etc.)

**Query Parameters:**
- `source`: Fuente específica (coingecko, binance, aggregated) - default: aggregated
- `include_metadata`: Incluir metadata adicional (true/false) - default: false

**Response (200):**
```json
{
  "success": true,
  "data": {
    "symbol": "BTC",
    "price": 45000.50,
    "price_usd": 45000.50,
    "timestamp": 1699123456,
    "source": "aggregated",
    "confidence_score": 0.98,
    "metadata": {
      "providers": {
        "coingecko": {
          "price": 45010.00,
          "latency_ms": 45,
          "weight": 0.33
        },
        "binance": {
          "price": 44995.50,
          "latency_ms": 12,
          "weight": 0.34
        },
        "coinbase": {
          "price": 45000.00,
          "latency_ms": 38,
          "weight": 0.33
        }
      },
      "aggregation_method": "weighted_average",
      "outliers_removed": 0,
      "last_update": "2024-01-15T10:30:56Z"
    }
  },
  "cache": {
    "hit": true,
    "ttl_seconds": 28,
    "key": "market:price:BTC"
  }
}
```

#### POST `/api/market/prices`
Obtiene precios de múltiples criptomonedas (batch).

**Request Body:**
```json
{
  "symbols": ["BTC", "ETH", "BNB", "SOL", "MATIC"],
  "include_24h_change": true,
  "include_volume": true
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "prices": {
      "BTC": {
        "price": 45000.50,
        "change_24h": 2.5,
        "change_24h_percentage": 5.7,
        "volume_24h": 25000000000
      },
      "ETH": {
        "price": 3000.25,
        "change_24h": 50.25,
        "change_24h_percentage": 1.7,
        "volume_24h": 15000000000
      }
    },
    "timestamp": 1699123456,
    "currency": "USD"
  }
}
```

### Datos Históricos

#### GET `/api/market/history/:symbol`
Obtiene datos históricos de precios.

**Parameters:**
- `symbol`: Símbolo de la criptomoneda

**Query Parameters:**
- `interval`: Intervalo de velas (1m, 5m, 15m, 1h, 4h, 1d) - default: 1h
- `from`: Timestamp inicial (Unix timestamp)
- `to`: Timestamp final (Unix timestamp)
- `limit`: Número máximo de velas (max: 1000) - default: 100

**Response (200):**
```json
{
  "success": true,
  "data": {
    "symbol": "BTC",
    "interval": "1h",
    "candles": [
      {
        "timestamp": 1699120000,
        "open": 44800.00,
        "high": 45200.00,
        "low": 44750.00,
        "close": 45000.00,
        "volume": 1234567890
      },
      {
        "timestamp": 1699123600,
        "open": 45000.00,
        "high": 45300.00,
        "low": 44950.00,
        "close": 45150.00,
        "volume": 987654321
      }
    ],
    "metadata": {
      "total_candles": 24,
      "time_range": {
        "from": "2024-01-14T10:00:00Z",
        "to": "2024-01-15T10:00:00Z"
      }
    }
  }
}
```

### Estadísticas de Mercado

#### GET `/api/market/stats/:symbol`
Obtiene estadísticas completas del mercado.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "symbol": "BTC",
    "current_price": 45000.50,
    "market_cap": 880000000000,
    "fully_diluted_valuation": 945000000000,
    "total_volume": 25000000000,
    "high_24h": 46000.00,
    "low_24h": 44000.00,
    "price_change_24h": 1000.50,
    "price_change_percentage_24h": 2.27,
    "price_change_percentage_7d": -1.5,
    "price_change_percentage_30d": 15.3,
    "price_change_percentage_1y": 145.7,
    "ath": 69000.00,
    "ath_change_percentage": -34.78,
    "ath_date": "2021-11-10T14:24:11.849Z",
    "atl": 67.81,
    "atl_change_percentage": 66253.74,
    "atl_date": "2013-07-06T00:00:00.000Z",
    "circulating_supply": 19500000,
    "total_supply": 21000000,
    "max_supply": 21000000,
    "market_metrics": {
      "volatility_24h": 0.042,
      "volatility_7d": 0.068,
      "sharpe_ratio": 1.85,
      "beta": 1.2,
      "correlation_with_market": 0.95
    }
  },
  "last_updated": "2024-01-15T10:30:00Z"
}
```

#### GET `/api/market/volatility/:symbol`
Calcula la volatilidad histórica.

**Query Parameters:**
- `period`: Período de cálculo (24h, 7d, 30d) - default: 24h
- `interval`: Intervalo de muestreo (5m, 1h, 1d) - default: 1h

**Response (200):**
```json
{
  "success": true,
  "data": {
    "symbol": "BTC",
    "period": "24h",
    "volatility": 0.042,
    "volatility_percentage": 4.2,
    "standard_deviation": 890.5,
    "variance": 792990.25,
    "samples": 24,
    "calculation_method": "close-to-close",
    "annualized_volatility": 0.803
  }
}
```

### WebSocket Streaming

#### WS `/api/market/stream`
WebSocket para recibir actualizaciones de precios en tiempo real.

**Connection:**
```javascript
ws://localhost:8004/api/market/stream
```

**Subscribe Message:**
```json
{
  "action": "subscribe",
  "channels": [
    {
      "name": "price",
      "symbols": ["BTC", "ETH"]
    },
    {
      "name": "orderbook",
      "symbols": ["BTC"],
      "depth": 10
    }
  ]
}
```

**Price Update Message:**
```json
{
  "type": "price_update",
  "data": {
    "symbol": "BTC",
    "price": 45050.75,
    "timestamp": 1699123456,
    "change_24h": 0.5,
    "volume": 25000000000
  }
}
```

**OrderBook Update Message:**
```json
{
  "type": "orderbook_update",
  "data": {
    "symbol": "BTC",
    "bids": [
      [44990, 2.5],
      [44985, 5.0]
    ],
    "asks": [
      [45010, 1.8],
      [45015, 3.5]
    ],
    "timestamp": 1699123456
  }
}
```

### Order Book

#### GET `/api/market/orderbook/:symbol`
Obtiene el libro de órdenes actual.

**Query Parameters:**
- `depth`: Profundidad del libro (default: 20, max: 100)

**Response (200):**
```json
{
  "success": true,
  "data": {
    "symbol": "BTC",
    "bids": [
      {"price": 44990.00, "amount": 2.5, "total": 112475.00},
      {"price": 44985.00, "amount": 5.0, "total": 224925.00}
    ],
    "asks": [
      {"price": 45010.00, "amount": 1.8, "total": 81018.00},
      {"price": 45015.00, "amount": 3.5, "total": 157552.50}
    ],
    "spread": 20.00,
    "spread_percentage": 0.044,
    "timestamp": 1699123456
  }
}
```

## ⚡ Sistema de Agregación

### Price Aggregator Implementation
```go
// price_aggregator.go
package aggregator

import (
    "math"
    "sort"
    "sync"
    "time"
)

type PriceAggregator struct {
    providers      map[string]Provider
    weights        map[string]float64
    outlierDetector *OutlierDetector
    cache          *cache.RedisCache
}

type AggregatedPrice struct {
    Symbol         string
    Price          float64
    Timestamp      time.Time
    Source         string
    Confidence     float64
    ProviderPrices map[string]ProviderPrice
}

type ProviderPrice struct {
    Price    float64
    Latency  time.Duration
    Weight   float64
    IsOutlier bool
}

func (pa *PriceAggregator) GetAggregatedPrice(symbol string) (*AggregatedPrice, error) {
    // Fetch prices from all providers concurrently
    prices := pa.fetchPricesFromProviders(symbol)
    
    // Detect and remove outliers
    validPrices := pa.outlierDetector.FilterOutliers(prices)
    
    // Calculate weighted average
    aggregatedPrice := pa.calculateWeightedAverage(validPrices)
    
    // Calculate confidence score
    confidence := pa.calculateConfidence(prices, validPrices)
    
    result := &AggregatedPrice{
        Symbol:         symbol,
        Price:          aggregatedPrice,
        Timestamp:      time.Now(),
        Source:         "aggregated",
        Confidence:     confidence,
        ProviderPrices: prices,
    }
    
    // Cache the result
    pa.cache.Set(fmt.Sprintf("market:price:%s", symbol), result, 30*time.Second)
    
    return result, nil
}

func (pa *PriceAggregator) fetchPricesFromProviders(symbol string) map[string]ProviderPrice {
    var wg sync.WaitGroup
    results := make(map[string]ProviderPrice)
    mu := &sync.Mutex{}
    
    for name, provider := range pa.providers {
        wg.Add(1)
        go func(providerName string, p Provider) {
            defer wg.Done()
            
            start := time.Now()
            price, err := p.GetPrice(symbol)
            latency := time.Since(start)
            
            if err == nil {
                mu.Lock()
                results[providerName] = ProviderPrice{
                    Price:   price,
                    Latency: latency,
                    Weight:  pa.weights[providerName],
                }
                mu.Unlock()
            }
        }(name, provider)
    }
    
    wg.Wait()
    return results
}

func (pa *PriceAggregator) calculateWeightedAverage(prices map[string]ProviderPrice) float64 {
    var weightedSum, totalWeight float64
    
    for _, price := range prices {
        if !price.IsOutlier {
            // Adjust weight based on latency (lower latency = higher weight)
            adjustedWeight := price.Weight * (1.0 / (1.0 + float64(price.Latency.Milliseconds())/1000.0))
            weightedSum += price.Price * adjustedWeight
            totalWeight += adjustedWeight
        }
    }
    
    if totalWeight == 0 {
        return 0
    }
    
    return weightedSum / totalWeight
}

func (pa *PriceAggregator) calculateConfidence(all, valid map[string]ProviderPrice) float64 {
    if len(all) == 0 {
        return 0
    }
    
    // Base confidence on provider availability
    availabilityScore := float64(len(valid)) / float64(len(pa.providers))
    
    // Calculate price variance
    prices := make([]float64, 0, len(valid))
    for _, p := range valid {
        prices = append(prices, p.Price)
    }
    
    variance := calculateVariance(prices)
    varianceScore := 1.0 / (1.0 + variance/1000.0) // Normalize variance
    
    // Combined confidence score
    return (availabilityScore * 0.6) + (varianceScore * 0.4)
}
```

### Outlier Detection
```go
// outlier_detector.go
package aggregator

type OutlierDetector struct {
    threshold float64 // Standard deviations from mean
}

func (od *OutlierDetector) FilterOutliers(prices map[string]ProviderPrice) map[string]ProviderPrice {
    if len(prices) < 3 {
        return prices // Need at least 3 prices for outlier detection
    }
    
    // Extract price values
    values := make([]float64, 0, len(prices))
    for _, p := range prices {
        values = append(values, p.Price)
    }
    
    // Calculate mean and standard deviation
    mean := calculateMean(values)
    stdDev := calculateStdDev(values, mean)
    
    // Mark outliers
    filtered := make(map[string]ProviderPrice)
    for name, price := range prices {
        deviation := math.Abs(price.Price - mean)
        if deviation <= od.threshold*stdDev {
            filtered[name] = price
        } else {
            price.IsOutlier = true
            prices[name] = price
        }
    }
    
    return filtered
}
```

## 🔄 WebSocket Hub

### WebSocket Management
```go
// hub.go
package websocket

import (
    "encoding/json"
    "log"
    "sync"
)

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    
    // Price subscriptions
    priceSubscriptions map[string]map[*Client]bool
    mu                 sync.RWMutex
}

func NewHub() *Hub {
    return &Hub{
        clients:            make(map[*Client]bool),
        broadcast:          make(chan []byte),
        register:           make(chan *Client),
        unregister:         make(chan *Client),
        priceSubscriptions: make(map[string]map[*Client]bool),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
            log.Printf("Client connected. Total: %d", len(h.clients))
            
        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
                h.removeSubscriptions(client)
                log.Printf("Client disconnected. Total: %d", len(h.clients))
            }
            
        case message := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}

func (h *Hub) BroadcastPriceUpdate(symbol string, price float64) {
    h.mu.RLock()
    subscribers := h.priceSubscriptions[symbol]
    h.mu.RUnlock()
    
    if len(subscribers) == 0 {
        return
    }
    
    update := map[string]interface{}{
        "type": "price_update",
        "data": map[string]interface{}{
            "symbol":    symbol,
            "price":     price,
            "timestamp": time.Now().Unix(),
        },
    }
    
    message, _ := json.Marshal(update)
    
    for client := range subscribers {
        select {
        case client.send <- message:
        default:
            // Client's send channel is full, skip
        }
    }
}
```

## 🧪 Testing

### Unit Tests
```go
// aggregator_test.go
package aggregator

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
)

func TestPriceAggregator_CalculateWeightedAverage(t *testing.T) {
    aggregator := NewPriceAggregator()
    
    prices := map[string]ProviderPrice{
        "coingecko": {Price: 45000, Weight: 0.33, Latency: 50 * time.Millisecond},
        "binance":   {Price: 45100, Weight: 0.34, Latency: 10 * time.Millisecond},
        "coinbase":  {Price: 45050, Weight: 0.33, Latency: 30 * time.Millisecond},
    }
    
    result := aggregator.calculateWeightedAverage(prices)
    
    // Should be close to 45050 with latency-adjusted weights
    assert.InDelta(t, 45050, result, 100)
}

func TestOutlierDetector_FilterOutliers(t *testing.T) {
    detector := NewOutlierDetector(2.0) // 2 standard deviations
    
    prices := map[string]ProviderPrice{
        "provider1": {Price: 45000},
        "provider2": {Price: 45100},
        "provider3": {Price: 45050},
        "provider4": {Price: 50000}, // Outlier
    }
    
    filtered := detector.FilterOutliers(prices)
    
    assert.Len(t, filtered, 3)
    assert.NotContains(t, filtered, "provider4")
}
```

## 🚀 Instalación y Configuración

### Variables de Entorno
```env
# Server
SERVER_PORT=8004
SERVER_ENV=development

# Redis
REDIS_URL=redis://localhost:6379
REDIS_DB=0
REDIS_PASSWORD=
REDIS_POOL_SIZE=10

# CoinGecko API
COINGECKO_API_KEY=your-api-key
COINGECKO_BASE_URL=https://api.coingecko.com/api/v3
COINGECKO_RATE_LIMIT=50
COINGECKO_WEIGHT=0.33

# Binance API
BINANCE_API_KEY=your-api-key
BINANCE_SECRET_KEY=your-secret
BINANCE_BASE_URL=https://api.binance.com
BINANCE_WS_URL=wss://stream.binance.com:9443
BINANCE_WEIGHT=0.34

# Coinbase API
COINBASE_API_KEY=your-api-key
COINBASE_SECRET=your-secret
COINBASE_BASE_URL=https://api.coinbase.com
COINBASE_WEIGHT=0.33

# WebSocket
WS_MAX_CONNECTIONS=1000
WS_PING_INTERVAL=30s
WS_PONG_TIMEOUT=60s
WS_MAX_MESSAGE_SIZE=512000

# Aggregation
OUTLIER_THRESHOLD=2.0
CONFIDENCE_MIN_PROVIDERS=2
AGGREGATION_TIMEOUT=5s

# Cache TTL
PRICE_CACHE_TTL=30s
STATS_CACHE_TTL=5m
HISTORY_CACHE_TTL=1h

# Performance
WORKER_POOL_SIZE=20
BATCH_SIZE=100
UPDATE_INTERVAL=5s
```

### Docker Compose
```yaml
version: '3.8'

services:
  market-data-api:
    build: .
    ports:
      - "8004:8004"
    environment:
      - REDIS_URL=redis://redis:6379
      - COINGECKO_API_KEY=${COINGECKO_API_KEY}
      - BINANCE_API_KEY=${BINANCE_API_KEY}
    depends_on:
      - redis
    volumes:
      - ./config:/app/config

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes

volumes:
  redis_data:
```

---

**Market Data API** - Motor de datos en tiempo real de CryptoSim 📊