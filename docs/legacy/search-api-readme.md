# 🔍 Search API - Microservicio de Búsqueda y Descubrimiento

## 📋 Descripción

El microservicio **Search API** proporciona capacidades avanzadas de búsqueda, filtrado y descubrimiento de criptomonedas en la plataforma CryptoSim. Utiliza Apache SolR como motor de búsqueda, implementa un sistema de cache multinivel (CCache local + Memcached distribuido) y mantiene sincronización en tiempo real mediante RabbitMQ.

## 🎯 Responsabilidades

- **Búsqueda Full-Text**: Búsqueda avanzada de criptomonedas con relevancia
- **Filtrado Dinámico**: Filtros por categoría, precio, capitalización, etc.
- **Ordenamiento Flexible**: Múltiples criterios de ordenamiento
- **Cache Multinivel**: Optimización con cache local y distribuido
- **Sincronización en Tiempo Real**: Consumidor RabbitMQ para actualizaciones
- **Trending Detection**: Identificación de criptomonedas en tendencia
- **Autocompletado**: Sugerencias de búsqueda en tiempo real
- **Faceted Search**: Búsqueda por facetas para mejor UX

## 🏗️ Arquitectura

### Estructura del Proyecto
```
search-api/
├── cmd/
│   └── main.go                      # Punto de entrada
├── internal/
│   ├── controllers/                 # Controladores HTTP
│   │   ├── search_controller.go
│   │   ├── trending_controller.go
│   │   └── admin_controller.go
│   ├── services/                    # Lógica de negocio
│   │   ├── search_service.go
│   │   ├── indexing_service.go
│   │   ├── trending_service.go
│   │   └── suggestion_service.go
│   ├── repositories/                # Acceso a datos
│   │   ├── solr_repository.go
│   │   └── cache_repository.go
│   ├── models/                      # Modelos de dominio
│   │   ├── crypto.go
│   │   ├── search_result.go
│   │   └── filter.go
│   ├── dto/                         # Data Transfer Objects
│   │   ├── search_request.go
│   │   ├── search_response.go
│   │   └── filter_dto.go
│   ├── indexer/                     # Indexación y sincronización
│   │   ├── indexer.go
│   │   ├── mapper.go
│   │   └── batch_processor.go
│   ├── cache/                       # Sistema de cache
│   │   ├── cache_manager.go
│   │   ├── local_cache.go          # CCache
│   │   ├── distributed_cache.go    # Memcached
│   │   └── cache_key_builder.go
│   ├── messaging/                   # RabbitMQ
│   │   ├── consumer.go
│   │   ├── event_handler.go
│   │   └── sync_processor.go
│   ├── solr/                        # Cliente SolR
│   │   ├── client.go
│   │   ├── query_builder.go
│   │   └── facet_builder.go
│   ├── middleware/                  # Middlewares
│   │   ├── cache_middleware.go
│   │   ├── rate_limit_middleware.go
│   │   └── logging_middleware.go
│   └── config/                      # Configuración
│       └── config.go
├── pkg/
│   ├── utils/                       # Utilidades
│   │   ├── pagination.go
│   │   ├── validator.go
│   │   └── response.go
│   └── errors/                      # Manejo de errores
│       └── search_errors.go
├── tests/                           # Tests
│   ├── unit/
│   │   └── search_service_test.go
│   ├── integration/
│   │   └── solr_integration_test.go
│   └── mocks/
│       └── repository_mock.go
├── solr/                            # Configuración SolR
│   ├── schema.xml
│   ├── solrconfig.xml
│   └── managed-schema
├── scripts/                         # Scripts de utilidad
│   ├── setup_solr.sh
│   ├── reindex.sh
│   └── clear_cache.sh
├── docs/                            # Documentación
│   ├── swagger.yaml
│   └── search_guide.md
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── .env.example
```

## 💾 Esquema de Datos

### Esquema SolR (managed-schema)
```xml
<?xml version="1.0" encoding="UTF-8"?>
<schema name="cryptos" version="1.6">
  <!-- Campos únicos -->
  <field name="_version_" type="plong" indexed="false" stored="false"/>
  <field name="_root_" type="string" indexed="true" stored="false" docValues="false"/>
  
  <!-- Campos principales -->
  <field name="id" type="string" indexed="true" stored="true" required="true" multiValued="false"/>
  <field name="symbol" type="string" indexed="true" stored="true" required="true"/>
  <field name="name" type="text_general" indexed="true" stored="true" required="true"/>
  <field name="description" type="text_general" indexed="true" stored="true"/>
  
  <!-- Campos numéricos -->
  <field name="current_price" type="pdouble" indexed="true" stored="true"/>
  <field name="market_cap" type="plong" indexed="true" stored="true"/>
  <field name="volume_24h" type="plong" indexed="true" stored="true"/>
  <field name="price_change_24h" type="pdouble" indexed="true" stored="true"/>
  <field name="price_change_7d" type="pdouble" indexed="true" stored="true"/>
  <field name="price_change_30d" type="pdouble" indexed="true" stored="true"/>
  <field name="total_supply" type="plong" indexed="true" stored="true"/>
  <field name="circulating_supply" type="plong" indexed="true" stored="true"/>
  <field name="max_supply" type="plong" indexed="true" stored="true"/>
  
  <!-- Campos de ranking -->
  <field name="market_cap_rank" type="pint" indexed="true" stored="true"/>
  <field name="trending_score" type="pfloat" indexed="true" stored="true"/>
  <field name="popularity_score" type="pfloat" indexed="true" stored="true"/>
  
  <!-- Categorías y tags -->
  <field name="category" type="string" indexed="true" stored="true" multiValued="true"/>
  <field name="tags" type="string" indexed="true" stored="true" multiValued="true"/>
  <field name="platform" type="string" indexed="true" stored="true"/>
  
  <!-- Métricas adicionales -->
  <field name="ath" type="pdouble" indexed="true" stored="true"/> <!-- All Time High -->
  <field name="ath_date" type="pdate" indexed="true" stored="true"/>
  <field name="atl" type="pdouble" indexed="true" stored="true"/> <!-- All Time Low -->
  <field name="atl_date" type="pdate" indexed="true" stored="true"/>
  
  <!-- Campos de estado -->
  <field name="is_active" type="boolean" indexed="true" stored="true" default="true"/>
  <field name="is_trending" type="boolean" indexed="true" stored="true" default="false"/>
  <field name="last_updated" type="pdate" indexed="true" stored="true"/>
  <field name="indexed_at" type="pdate" indexed="true" stored="true" default="NOW"/>
  
  <!-- Campos de búsqueda optimizados -->
  <field name="search_text" type="text_general" indexed="true" stored="false" multiValued="true"/>
  <field name="symbol_exact" type="string" indexed="true" stored="false"/>
  <field name="name_exact" type="string" indexed="true" stored="false"/>
  
  <!-- Copy Fields para búsqueda -->
  <copyField source="symbol" dest="search_text"/>
  <copyField source="name" dest="search_text"/>
  <copyField source="description" dest="search_text"/>
  <copyField source="category" dest="search_text"/>
  <copyField source="tags" dest="search_text"/>
  <copyField source="symbol" dest="symbol_exact"/>
  <copyField source="name" dest="name_exact"/>
  
  <!-- Campo único -->
  <uniqueKey>id</uniqueKey>
  
  <!-- Tipos de campo -->
  <fieldType name="string" class="solr.StrField" sortMissingLast="true" docValues="true"/>
  <fieldType name="plong" class="solr.LongPointField" docValues="true"/>
  <fieldType name="pdouble" class="solr.DoublePointField" docValues="true"/>
  <fieldType name="pfloat" class="solr.FloatPointField" docValues="true"/>
  <fieldType name="pint" class="solr.IntPointField" docValues="true"/>
  <fieldType name="pdate" class="solr.DatePointField" docValues="true"/>
  <fieldType name="boolean" class="solr.BoolField" sortMissingLast="true"/>
  
  <!-- Tipo de texto con análisis -->
  <fieldType name="text_general" class="solr.TextField" positionIncrementGap="100">
    <analyzer type="index">
      <tokenizer class="solr.StandardTokenizerFactory"/>
      <filter class="solr.StopFilterFactory" words="stopwords.txt" ignoreCase="true"/>
      <filter class="solr.LowerCaseFilterFactory"/>
      <filter class="solr.EnglishPossessiveFilterFactory"/>
      <filter class="solr.PorterStemFilterFactory"/>
    </analyzer>
    <analyzer type="query">
      <tokenizer class="solr.StandardTokenizerFactory"/>
      <filter class="solr.StopFilterFactory" words="stopwords.txt" ignoreCase="true"/>
      <filter class="solr.SynonymGraphFilterFactory" expand="true" ignoreCase="true" synonyms="synonyms.txt"/>
      <filter class="solr.LowerCaseFilterFactory"/>
      <filter class="solr.EnglishPossessiveFilterFactory"/>
      <filter class="solr.PorterStemFilterFactory"/>
    </analyzer>
  </fieldType>
</schema>
```

### Estructura de Cache
```go
// Cache Keys Structure
const (
    CacheKeyPrefix        = "search:"
    CacheTTLLocal        = 5 * time.Minute
    CacheTTLDistributed  = 15 * time.Minute
)

type CacheEntry struct {
    Key       string
    Value     interface{}
    TTL       time.Duration
    CreatedAt time.Time
    HitCount  int64
}

// Example cache keys:
// search:query:bitcoin:page:1:limit:20
// search:trending:24h
// search:filters:all
// search:suggestions:bit
```

## 🔌 API Endpoints

### Búsqueda Principal

#### GET `/api/search/cryptos`
Búsqueda paginada de criptomonedas con filtros avanzados.

**Query Parameters:**
- `q`: Query de búsqueda (opcional, empty query permitido)
- `page`: Página (default: 1)
- `limit`: Resultados por página (default: 20, max: 100)
- `sort`: Campo de ordenamiento (price_asc, price_desc, market_cap_desc, trending_desc, name_asc)
- `category`: Filtro por categoría (DeFi, NFT, Gaming, Layer1, Layer2)
- `min_price`: Precio mínimo
- `max_price`: Precio máximo
- `min_market_cap`: Capitalización mínima
- `max_market_cap`: Capitalización máxima
- `price_change_24h`: Filtro por cambio de precio (positive, negative)
- `is_trending`: Solo mostrar trending (true/false)

**Response (200):**
```json
{
  "success": true,
  "data": {
    "results": [
      {
        "id": "bitcoin",
        "symbol": "BTC",
        "name": "Bitcoin",
        "current_price": 45000.00,
        "market_cap": 880000000000,
        "market_cap_rank": 1,
        "volume_24h": 25000000000,
        "price_change_24h": 2.5,
        "price_change_7d": -1.2,
        "circulating_supply": 19500000,
        "total_supply": 21000000,
        "category": ["Cryptocurrency", "Layer1"],
        "trending_score": 95.5,
        "is_trending": true,
        "last_updated": "2024-01-15T10:30:00Z"
      }
    ],
    "pagination": {
      "total": 5000,
      "page": 1,
      "limit": 20,
      "total_pages": 250,
      "has_next": true,
      "has_prev": false
    },
    "facets": {
      "categories": {
        "DeFi": 1250,
        "NFT": 800,
        "Gaming": 450,
        "Layer1": 50,
        "Layer2": 120
      },
      "price_ranges": {
        "0-1": 2500,
        "1-10": 1200,
        "10-100": 800,
        "100-1000": 400,
        "1000+": 100
      }
    },
    "query_info": {
      "query": "bitcoin",
      "execution_time_ms": 15,
      "cache_hit": false,
      "total_found": 5
    }
  }
}
```

#### GET `/api/search/cryptos/:id`
Obtiene detalles completos de una criptomoneda.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "ethereum",
    "symbol": "ETH",
    "name": "Ethereum",
    "description": "Ethereum is a decentralized platform that runs smart contracts...",
    "current_price": 3000.00,
    "market_cap": 360000000000,
    "market_cap_rank": 2,
    "volume_24h": 15000000000,
    "price_change_24h": 3.2,
    "price_change_7d": 5.8,
    "price_change_30d": -2.1,
    "ath": 4878.26,
    "ath_date": "2021-11-10",
    "atl": 0.43,
    "atl_date": "2015-10-20",
    "circulating_supply": 120000000,
    "total_supply": null,
    "max_supply": null,
    "category": ["Smart Contracts", "Layer1", "DeFi"],
    "tags": ["ethereum-ecosystem", "smart-contracts", "dapps"],
    "platform": "Ethereum",
    "trending_score": 88.3,
    "is_trending": true,
    "is_active": true,
    "last_updated": "2024-01-15T10:30:00Z"
  }
}
```

### Trending y Descubrimiento

#### GET `/api/search/cryptos/trending`
Obtiene las criptomonedas en tendencia.

**Query Parameters:**
- `period`: Período de tendencia (1h, 24h, 7d, 30d) - default: 24h
- `limit`: Número de resultados (default: 10, max: 50)

**Response (200):**
```json
{
  "success": true,
  "data": {
    "trending": [
      {
        "rank": 1,
        "id": "pepe",
        "symbol": "PEPE",
        "name": "Pepe",
        "current_price": 0.000001234,
        "price_change_24h": 45.6,
        "volume_24h": 500000000,
        "trending_score": 98.5,
        "search_volume_increase": "2500%",
        "mentions_count": 15000
      }
    ],
    "period": "24h",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

#### GET `/api/search/cryptos/suggestions`
Autocompletado y sugerencias de búsqueda.

**Query Parameters:**
- `q`: Término de búsqueda parcial
- `limit`: Número de sugerencias (default: 5, max: 10)

**Response (200):**
```json
{
  "success": true,
  "data": {
    "suggestions": [
      {
        "id": "bitcoin",
        "symbol": "BTC",
        "name": "Bitcoin",
        "match_type": "symbol",
        "score": 100
      },
      {
        "id": "bitcoin-cash",
        "symbol": "BCH",
        "name": "Bitcoin Cash",
        "match_type": "name",
        "score": 85
      }
    ],
    "query": "bit",
    "execution_time_ms": 5
  }
}
```

### Filtros y Facetas

#### GET `/api/search/cryptos/filters`
Obtiene los filtros disponibles con conteos.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "filters": {
      "categories": [
        {"value": "DeFi", "count": 1250, "label": "Decentralized Finance"},
        {"value": "NFT", "count": 800, "label": "Non-Fungible Tokens"},
        {"value": "Gaming", "count": 450, "label": "Gaming & Metaverse"}
      ],
      "price_ranges": [
        {"min": 0, "max": 1, "count": 2500, "label": "Under $1"},
        {"min": 1, "max": 10, "count": 1200, "label": "$1 - $10"},
        {"min": 10, "max": 100, "count": 800, "label": "$10 - $100"}
      ],
      "market_cap_ranges": [
        {"min": 0, "max": 1000000, "count": 3000, "label": "Micro Cap"},
        {"min": 1000000, "max": 10000000, "count": 1000, "label": "Small Cap"},
        {"min": 10000000, "max": 100000000, "count": 500, "label": "Mid Cap"}
      ],
      "sort_options": [
        {"value": "market_cap_desc", "label": "Market Cap ↓"},
        {"value": "price_desc", "label": "Price ↓"},
        {"value": "trending_desc", "label": "Trending ↓"},
        {"value": "volume_desc", "label": "Volume 24h ↓"}
      ]
    }
  }
}
```

### Administración

#### POST `/api/search/reindex`
Reindexa todos los datos en SolR (admin only).

**Headers:**
```
Authorization: Bearer [admin-token]
```

**Request Body:**
```json
{
  "full_reindex": true,
  "clear_existing": false,
  "batch_size": 100
}
```

**Response (202):**
```json
{
  "success": true,
  "message": "Reindexación iniciada",
  "data": {
    "job_id": "reindex_20240115_103000",
    "estimated_time": "5 minutes",
    "total_documents": 5000
  }
}
```

#### DELETE `/api/search/cache/clear`
Limpia todo el cache (admin only).

**Headers:**
```
Authorization: Bearer [admin-token]
```

**Response (200):**
```json
{
  "success": true,
  "message": "Cache limpiado exitosamente",
  "data": {
    "local_cache_cleared": true,
    "distributed_cache_cleared": true,
    "entries_removed": 1250
  }
}
```

## 🚀 Sistema de Cache Multinivel

### Implementación del Cache
```go
// cache_manager.go
package cache

import (
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/karlseguin/ccache/v2"
    "github.com/bradfitz/gomemcache/memcache"
)

type CacheManager struct {
    localCache      *ccache.Cache
    distributedCache *memcache.Client
    config          *CacheConfig
}

type CacheConfig struct {
    LocalTTL        time.Duration
    DistributedTTL  time.Duration
    MaxLocalSize    int64
    MemcachedHosts  []string
}

func NewCacheManager(config *CacheConfig) *CacheManager {
    // Configurar cache local (CCache)
    localCache := ccache.New(ccache.Configure().
        MaxSize(config.MaxLocalSize).
        ItemsToPrune(100))
    
    // Configurar cache distribuido (Memcached)
    distributedCache := memcache.New(config.MemcachedHosts...)
    
    return &CacheManager{
        localCache:       localCache,
        distributedCache: distributedCache,
        config:          config,
    }
}

// Get intenta obtener del cache en orden: local -> distribuido
func (cm *CacheManager) Get(key string) (interface{}, bool) {
    // 1. Intentar cache local
    if item := cm.localCache.Get(key); item != nil && !item.Expired() {
        cm.incrementHitCount(key, "local")
        return item.Value(), true
    }
    
    // 2. Intentar cache distribuido
    if item, err := cm.distributedCache.Get(key); err == nil {
        var value interface{}
        if err := json.Unmarshal(item.Value, &value); err == nil {
            // Actualizar cache local
            cm.localCache.Set(key, value, cm.config.LocalTTL)
            cm.incrementHitCount(key, "distributed")
            return value, true
        }
    }
    
    cm.incrementMissCount(key)
    return nil, false
}

// Set actualiza ambos niveles de cache
func (cm *CacheManager) Set(key string, value interface{}, ttl time.Duration) error {
    // 1. Actualizar cache local
    cm.localCache.Set(key, value, ttl)
    
    // 2. Actualizar cache distribuido
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    
    return cm.distributedCache.Set(&memcache.Item{
        Key:        key,
        Value:      data,
        Expiration: int32(ttl.Seconds()),
    })
}

// InvalidatePattern invalida todas las claves que coinciden con un patrón
func (cm *CacheManager) InvalidatePattern(pattern string) error {
    // Limpiar cache local
    cm.localCache.DeletePrefix(pattern)
    
    // Para memcached, necesitamos mantener un índice de claves
    // o usar una estrategia de versionado
    return cm.invalidateDistributedPattern(pattern)
}

// Warming precalienta el cache con búsquedas populares
func (cm *CacheManager) WarmCache(searchService *SearchService) error {
    popularQueries := []string{
        "",           // Empty query (homepage)
        "bitcoin",
        "ethereum",
        "trending",
    }
    
    for _, query := range popularQueries {
        request := &SearchRequest{
            Query: query,
            Page:  1,
            Limit: 20,
        }
        
        result, err := searchService.Search(request)
        if err == nil {
            key := cm.buildKey(request)
            cm.Set(key, result, cm.config.DistributedTTL)
        }
    }
    
    return nil
}
```

## 📨 Sincronización con RabbitMQ

### Consumer Implementation
```go
// consumer.go
package messaging

import (
    "encoding/json"
    "log"
    
    "github.com/streadway/amqp"
)

type SearchConsumer struct {
    conn            *amqp.Connection
    channel         *amqp.Channel
    indexingService *services.IndexingService
}

func (sc *SearchConsumer) Start() error {
    // Declarar exchange y cola
    err := sc.channel.ExchangeDeclare(
        "cryptosim",
        "topic",
        true,
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        return err
    }
    
    q, err := sc.channel.QueueDeclare(
        "search.sync",
        true,
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        return err
    }
    
    // Bind para eventos de órdenes
    err = sc.channel.QueueBind(
        q.Name,
        "orders.#",
        "cryptosim",
        false,
        nil,
    )
    if err != nil {
        return err
    }
    
    // Consumir mensajes
    msgs, err := sc.channel.Consume(
        q.Name,
        "",
        false,
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        return err
    }
    
    go sc.processMessages(msgs)
    
    return nil
}

func (sc *SearchConsumer) processMessages(msgs <-chan amqp.Delivery) {
    for msg := range msgs {
        var event OrderEvent
        if err := json.Unmarshal(msg.Body, &event); err != nil {
            log.Printf("Error parsing message: %v", err)
            msg.Nack(false, false)
            continue
        }
        
        switch event.EventType {
        case "order.executed":
            sc.handleOrderExecuted(event)
        case "order.created":
            sc.handleOrderCreated(event)
        }
        
        msg.Ack(false)
    }
}

func (sc *SearchConsumer) handleOrderExecuted(event OrderEvent) {
    // Actualizar trending score
    crypto := event.Data["crypto_symbol"].(string)
    sc.indexingService.UpdateTrendingScore(crypto, 10.0)
    
    // Invalidar cache relacionado
    sc.cacheManager.InvalidatePattern(fmt.Sprintf("search:*%s*", crypto))
}
```

## 🧪 Testing

### Unit Tests
```go
// search_service_test.go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestSearchService_Search(t *testing.T) {
    // Arrange
    mockRepo := new(mocks.MockSolrRepository)
    mockCache := new(mocks.MockCacheManager)
    service := NewSearchService(mockRepo, mockCache)
    
    request := &SearchRequest{
        Query: "bitcoin",
        Page:  1,
        Limit: 20,
    }
    
    expectedResults := []Crypto{
        {ID: "bitcoin", Symbol: "BTC", Name: "Bitcoin"},
    }
    
    // Cache miss scenario
    mockCache.On("Get", mock.Anything).Return(nil, false)
    mockRepo.On("Search", request).Return(expectedResults, nil)
    mockCache.On("Set", mock.Anything, expectedResults, mock.Anything).Return(nil)
    
    // Act
    results, err := service.Search(request)
    
    // Assert
    assert.NoError(t, err)
    assert.Len(t, results, 1)
    assert.Equal(t, "bitcoin", results[0].ID)
    mockCache.AssertExpectations(t)
    mockRepo.AssertExpectations(t)
}

func TestSearchService_CacheHit(t *testing.T) {
    // Test cache hit scenario
    mockCache := new(mocks.MockCacheManager)
    service := NewSearchService(nil, mockCache)
    
    cachedResults := []Crypto{
        {ID: "ethereum", Symbol: "ETH", Name: "Ethereum"},
    }
    
    mockCache.On("Get", mock.Anything).Return(cachedResults, true)
    
    // Act
    results, err := service.Search(&SearchRequest{Query: "ethereum"})
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, cachedResults, results)
    // Verify repository was not called
    mockCache.AssertExpectations(t)
}
```

## 🚀 Instalación y Configuración

### Variables de Entorno
```env
# Server
SERVER_PORT=8003
SERVER_ENV=development

# SolR
SOLR_URL=http://localhost:8983/solr
SOLR_CORE=cryptos
SOLR_TIMEOUT=10s
SOLR_MAX_RETRIES=3

# Cache - Local (CCache)
LOCAL_CACHE_SIZE=1000000
LOCAL_CACHE_TTL=5m

# Cache - Distributed (Memcached)
MEMCACHED_HOSTS=localhost:11211
DISTRIBUTED_CACHE_TTL=15m

# RabbitMQ
RABBITMQ_URL=amqp://admin:admin@localhost:5672/
RABBITMQ_EXCHANGE=cryptosim
RABBITMQ_QUEUE=search.sync

# Internal Services
ORDERS_API_URL=http://localhost:8002
MARKET_API_URL=http://localhost:8004

# Performance
BATCH_SIZE=100
WORKER_POOL_SIZE=10
REINDEX_SCHEDULE=0 */6 * * *
```

### Docker Compose
```yaml
version: '3.8'

services:
  search-api:
    build: .
    ports:
      - "8003:8003"
    environment:
      - SOLR_URL=http://solr:8983/solr
      - MEMCACHED_HOSTS=memcached:11211
      - RABBITMQ_URL=amqp://rabbitmq:5672
    depends_on:
      - solr
      - memcached
      - rabbitmq