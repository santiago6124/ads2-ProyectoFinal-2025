# Flujo Técnico: Búsqueda de Órdenes

## Descripción General

Este documento describe el sistema de búsqueda de órdenes utilizando Apache Solr, el sistema de caché multinivel y la sincronización automática mediante RabbitMQ.

## Arquitectura Involucrada

```
┌──────────┐    ┌────────────┐    ┌────────────┐    ┌───────────┐
│  Cliente │───>│ Search API │───>│Orders API  │    │ Memcached │
│          │<───│   :8003    │<───│   :8002    │    │           │
└──────────┘    └──────┬─────┘    └────────────┘    └─────┬─────┘
                       │                                    │
                       ▼                                    ▼
                ┌─────────────┐                      ┌──────────┐
                │ Apache Solr │                      │  CCache  │
                │   :8983     │                      │ (Local)  │
                └─────────────┘                      └──────────┘
                       ▲
                       │
                ┌──────┴──────┐
                │  RabbitMQ   │
                │ orders.events│
                └─────────────┘
```

---

## 1. BÚSQUEDA DE ÓRDENES

### Endpoint
```
POST /api/v1/search
Authorization: Bearer {JWT_TOKEN}
Content-Type: application/json
```

### Request Body
```json
{
  "query": "*",
  "filters": {
    "status": "executed",
    "type": "buy",
    "crypto_symbol": "BTC",
    "min_amount": 10.0,
    "max_amount": 100.0,
    "start_date": "2025-11-01T00:00:00Z",
    "end_date": "2025-11-14T23:59:59Z"
  },
  "sort": {
    "field": "created_at",
    "order": "desc"
  },
  "pagination": {
    "page": 1,
    "page_size": 20
  }
}
```

### Proceso Paso a Paso

#### 1.1 Cliente envía búsqueda
```bash
curl -X POST http://localhost:8003/api/v1/search \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "query": "*",
    "filters": {
      "status": "executed",
      "crypto_symbol": "BTC"
    },
    "pagination": {
      "page": 1,
      "page_size": 10
    }
  }'
```

#### 1.2 Middleware JWT
```go
// search-api/middleware/auth.go
token := extractTokenFromHeader(c.GetHeader("Authorization"))
claims := parseJWT(token)
userID := claims["user_id"]  // 123
c.Set("user_id", userID)
c.Next()
```

#### 1.3 Handler recibe request
```go
// search-api/handlers/search_handler.go
func (h *SearchHandler) Search(c *gin.Context) {
    userID := c.GetInt64("user_id")

    var req SearchRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }

    // Generar cache key basado en request
    cacheKey := generateCacheKey(userID, req)

    // Intentar obtener de caché
    if cachedResult := h.cacheService.Get(cacheKey); cachedResult != nil {
        c.JSON(200, cachedResult)
        return
    }

    // Si no está en caché, buscar en Solr
    results, err := h.searchService.Search(c.Request.Context(), userID, req)
    if err != nil {
        c.JSON(500, gin.H{"error": "Search failed"})
        return
    }

    // Cachear resultado
    h.cacheService.Set(cacheKey, results, 5*time.Minute)

    c.JSON(200, results)
}
```

#### 1.4 Service Layer - Sistema de Caché Multinivel

**Nivel 1: CCache (In-Memory Local)**
```go
// search-api/cache/ccache_service.go
func (cs *CCacheService) Get(key string) interface{} {
    item := cs.cache.Get(key)
    if item != nil && !item.Expired() {
        return item.Value()
    }
    return nil
}
```

**Nivel 2: Memcached (Distribuido)**
```go
// search-api/cache/memcached_service.go
func (ms *MemcachedService) Get(key string) (interface{}, error) {
    item, err := ms.client.Get(key)
    if err != nil {
        return nil, err
    }

    var result SearchResult
    json.Unmarshal(item.Value, &result)
    return result, nil
}
```

**Estrategia de caché:**
```go
func (cs *CacheService) Get(key string) interface{} {
    // 1. Intentar CCache (local, ultra rápido)
    if result := cs.ccache.Get(key); result != nil {
        log.Info("Cache hit: CCache")
        return result
    }

    // 2. Intentar Memcached (distribuido)
    if result, err := cs.memcached.Get(key); err == nil {
        log.Info("Cache hit: Memcached")
        // Guardar en CCache para próxima vez
        cs.ccache.Set(key, result, 5*time.Minute)
        return result
    }

    log.Info("Cache miss")
    return nil
}
```

#### 1.5 Construir query de Solr

```go
// search-api/services/solr_service.go
func (ss *SolrService) buildSolrQuery(userID int64, req SearchRequest) string {
    // Base query (solo órdenes del usuario)
    q := fmt.Sprintf("user_id:%d", userID)

    // Si hay query text
    if req.Query != "" && req.Query != "*" {
        q += fmt.Sprintf(" AND (crypto_symbol:%s OR crypto_name:%s)", req.Query, req.Query)
    }

    // Filtros
    filters := []string{}

    if req.Filters.Status != "" {
        filters = append(filters, fmt.Sprintf("status:%s", req.Filters.Status))
    }

    if req.Filters.Type != "" {
        filters = append(filters, fmt.Sprintf("type:%s", req.Filters.Type))
    }

    if req.Filters.CryptoSymbol != "" {
        filters = append(filters, fmt.Sprintf("crypto_symbol:%s", req.Filters.CryptoSymbol))
    }

    if req.Filters.MinAmount > 0 {
        filters = append(filters, fmt.Sprintf("total_amount:[%f TO *]", req.Filters.MinAmount))
    }

    if req.Filters.MaxAmount > 0 {
        filters = append(filters, fmt.Sprintf("total_amount:[* TO %f]", req.Filters.MaxAmount))
    }

    if req.Filters.StartDate != "" {
        filters = append(filters, fmt.Sprintf("created_at:[%s TO *]", req.Filters.StartDate))
    }

    if req.Filters.EndDate != "" {
        filters = append(filters, fmt.Sprintf("created_at:[* TO %s]", req.Filters.EndDate))
    }

    // Combinar filtros
    if len(filters) > 0 {
        q += " AND " + strings.Join(filters, " AND ")
    }

    return q
}
```

**Query Solr generada:**
```
q=user_id:123 AND status:executed AND crypto_symbol:BTC
&start=0
&rows=10
&sort=created_at desc
&wt=json
```

#### 1.6 Ejecutar búsqueda en Solr

```go
// Hacer request a Solr
solrURL := fmt.Sprintf("http://solr:8983/solr/orders_search/select?q=%s&start=%d&rows=%d&sort=%s&wt=json",
    url.QueryEscape(query),
    start,
    rows,
    sortParam,
)

resp, err := http.Get(solrURL)
if err != nil {
    return nil, err
}

var solrResponse SolrResponse
json.NewDecoder(resp.Body).Decode(&solrResponse)
```

**Response de Solr:**
```json
{
  "responseHeader": {
    "status": 0,
    "QTime": 12
  },
  "response": {
    "numFound": 3,
    "start": 0,
    "docs": [
      {
        "id": "673b5f8a9e1234567890abcd",
        "user_id": 123,
        "type": "buy",
        "status": "executed",
        "crypto_symbol": "BTC",
        "crypto_name": "Bitcoin",
        "quantity": 0.001,
        "price": 50000.0,
        "total_amount": 50.0,
        "fee": 0.05,
        "order_kind": "market",
        "created_at": "2025-11-14T10:30:00Z",
        "executed_at": "2025-11-14T10:30:05Z"
      },
      {
        "id": "673b5f8a9e1234567890abce",
        "user_id": 123,
        "type": "buy",
        "status": "executed",
        "crypto_symbol": "BTC",
        "crypto_name": "Bitcoin",
        "quantity": 0.002,
        "price": 48000.0,
        "total_amount": 96.0,
        "fee": 0.096,
        "order_kind": "market",
        "created_at": "2025-11-12T14:20:00Z",
        "executed_at": "2025-11-12T14:20:03Z"
      }
    ]
  }
}
```

#### 1.7 Response de Search API

```json
{
  "status": "success",
  "data": {
    "results": [
      {
        "id": "673b5f8a9e1234567890abcd",
        "user_id": 123,
        "type": "buy",
        "status": "executed",
        "crypto_symbol": "BTC",
        "crypto_name": "Bitcoin",
        "quantity": "0.001",
        "price": "50000.00",
        "total_amount": "50.00",
        "fee": "0.05",
        "order_kind": "market",
        "created_at": "2025-11-14T10:30:00Z",
        "executed_at": "2025-11-14T10:30:05Z"
      },
      {
        "id": "673b5f8a9e1234567890abce",
        "user_id": 123,
        "type": "buy",
        "status": "executed",
        "crypto_symbol": "BTC",
        "crypto_name": "Bitcoin",
        "quantity": "0.002",
        "price": "48000.00",
        "total_amount": "96.00",
        "fee": "0.096",
        "order_kind": "market",
        "created_at": "2025-11-12T14:20:00Z",
        "executed_at": "2025-11-12T14:20:03Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 10,
      "total_results": 3,
      "total_pages": 1
    },
    "query_info": {
      "query": "*",
      "filters_applied": ["status", "crypto_symbol"],
      "sort": "created_at desc",
      "execution_time_ms": 45
    }
  }
}
```

---

## 2. SINCRONIZACIÓN CON SOLR VÍA RABBITMQ

### Consumer de eventos de órdenes

#### 2.1 Inicializar consumer
```go
// search-api/messaging/consumer.go
func (c *OrdersConsumer) Start() {
    // Declarar exchange
    err := c.channel.ExchangeDeclare(
        "orders.events",  // Name
        "topic",          // Type
        true,             // Durable
        false,            // Auto-delete
        false,            // Internal
        false,            // No-wait
        nil,              // Args
    )

    // Declarar queue
    queue, err := c.channel.QueueDeclare(
        "search.sync",    // Name
        true,             // Durable
        false,            // Auto-delete
        false,            // Exclusive
        false,            // No-wait
        nil,              // Args
    )

    // Bind queue a exchange con múltiples routing keys
    routingKeys := []string{
        "orders.created",
        "orders.executed",
        "orders.cancelled",
        "orders.failed",
    }

    for _, key := range routingKeys {
        c.channel.QueueBind(
            queue.Name,       // Queue name
            key,              // Routing key
            "orders.events",  // Exchange
            false,
            nil,
        )
    }

    // Consumir mensajes
    msgs, err := c.channel.Consume(
        queue.Name,
        "search-consumer",  // Consumer tag
        false,              // Auto-ack
        false,              // Exclusive
        false,              // No-local
        false,              // No-wait
        nil,
    )

    // Procesar mensajes
    for msg := range msgs {
        c.handleMessage(msg)
    }
}
```

#### 2.2 Procesar mensaje

```go
func (c *OrdersConsumer) handleMessage(msg amqp.Delivery) {
    var event OrderEvent
    err := json.Unmarshal(msg.Body, &event)
    if err != nil {
        log.Error("Failed to unmarshal event", err)
        msg.Nack(false, false)  // Dead letter queue
        return
    }

    log.Info("Received order event", "type", event.EventType, "order_id", event.OrderID)

    switch event.EventType {
    case "created", "executed":
        err = c.indexOrder(event)
    case "cancelled", "failed":
        err = c.updateOrderStatus(event)
    }

    if err != nil {
        log.Error("Failed to process event", err)
        msg.Nack(false, true)  // Requeue
        return
    }

    msg.Ack(false)
}
```

#### 2.3 Indexar orden en Solr

**Obtener detalles completos de Orders API:**
```go
func (c *OrdersConsumer) indexOrder(event OrderEvent) error {
    // 1. Obtener orden completa desde Orders API
    orderResp, err := http.Get(fmt.Sprintf("http://orders-api:8002/api/v1/orders/%s", event.OrderID))
    if err != nil {
        return err
    }

    var orderData OrderResponse
    json.NewDecoder(orderResp.Body).Decode(&orderData)
    order := orderData.Data.Order

    // 2. Construir documento Solr
    solrDoc := SolrDocument{
        ID:           order.ID,
        UserID:       order.UserID,
        Type:         order.Type,
        Status:       order.Status,
        CryptoSymbol: order.CryptoSymbol,
        CryptoName:   getCryptoName(order.CryptoSymbol),  // BTC -> Bitcoin
        Quantity:     order.Quantity,
        Price:        order.Price,
        TotalAmount:  order.TotalAmount,
        Fee:          order.Fee,
        OrderKind:    order.OrderKind,
        CreatedAt:    order.CreatedAt,
        ExecutedAt:   order.ExecutedAt,
    }

    // 3. Indexar en Solr
    solrURL := "http://solr:8983/solr/orders_search/update?commit=true"
    solrPayload := []SolrDocument{solrDoc}
    solrJSON, _ := json.Marshal(solrPayload)

    resp, err := http.Post(solrURL, "application/json", bytes.NewBuffer(solrJSON))
    if err != nil || resp.StatusCode != 200 {
        return errors.New("Failed to index in Solr")
    }

    log.Info("Order indexed in Solr", "order_id", order.ID)

    // 4. Invalidar caché relacionado
    c.cacheService.DeletePattern(fmt.Sprintf("search:user:%d:*", order.UserID))

    return nil
}
```

**Request Solr:**
```http
POST http://solr:8983/solr/orders_search/update?commit=true
Content-Type: application/json

[
  {
    "id": "673b5f8a9e1234567890abcd",
    "user_id": 123,
    "type": "buy",
    "status": "executed",
    "crypto_symbol": "BTC",
    "crypto_name": "Bitcoin",
    "quantity": 0.001,
    "price": 50000.0,
    "total_amount": 50.0,
    "fee": 0.05,
    "order_kind": "market",
    "created_at": "2025-11-14T10:30:00Z",
    "executed_at": "2025-11-14T10:30:05Z"
  }
]
```

#### 2.4 Actualizar estado de orden

```go
func (c *OrdersConsumer) updateOrderStatus(event OrderEvent) error {
    // Actualizar solo el campo status en Solr
    solrURL := "http://solr:8983/solr/orders_search/update?commit=true"

    updateDoc := map[string]interface{}{
        "id":     event.OrderID,
        "status": map[string]string{"set": event.EventType},  // "cancelled" o "failed"
    }

    updateJSON, _ := json.Marshal([]map[string]interface{}{updateDoc})
    http.Post(solrURL, "application/json", bytes.NewBuffer(updateJSON))

    // Invalidar caché
    c.cacheService.DeletePattern(fmt.Sprintf("search:user:%d:*", event.UserID))

    return nil
}
```

---

## 3. TRENDING CRIPTOMONEDAS

### Endpoint
```
GET /api/v1/trending
```

### Proceso

#### 3.1 Request
```bash
curl http://localhost:8003/api/v1/trending
```

#### 3.2 Consultar Solr con facets
```go
// Obtener top cryptos por cantidad de órdenes ejecutadas
solrQuery := `
q=status:executed AND created_at:[NOW-7DAYS TO NOW]
&facet=true
&facet.field=crypto_symbol
&facet.limit=10
&facet.sort=count
&rows=0
&wt=json
`

solrURL := fmt.Sprintf("http://solr:8983/solr/orders_search/select?%s", solrQuery)
resp, _ := http.Get(solrURL)

var solrResp SolrFacetResponse
json.NewDecoder(resp.Body).Decode(&solrResp)
```

**Response Solr:**
```json
{
  "response": {
    "numFound": 150,
    "docs": []
  },
  "facet_counts": {
    "facet_fields": {
      "crypto_symbol": [
        "BTC", 45,
        "ETH", 32,
        "BNB", 18,
        "SOL", 15,
        "ADA", 12
      ]
    }
  }
}
```

#### 3.3 Response de Trending
```json
{
  "status": "success",
  "data": {
    "trending": [
      {
        "symbol": "BTC",
        "name": "Bitcoin",
        "orders_count": 45,
        "trend": "up"
      },
      {
        "symbol": "ETH",
        "name": "Ethereum",
        "orders_count": 32,
        "trend": "up"
      },
      {
        "symbol": "BNB",
        "name": "Binance Coin",
        "orders_count": 18,
        "trend": "stable"
      }
    ],
    "period": "7d",
    "total_orders": 150
  }
}
```

---

## 4. AUTOCOMPLETE / SUGERENCIAS

### Endpoint
```
GET /api/v1/suggestions?q=bit
```

### Proceso

#### 4.1 Request
```bash
curl http://localhost:8003/api/v1/suggestions?q=bit
```

#### 4.2 Buscar con partial match en Solr
```go
solrQuery := fmt.Sprintf(`
q=crypto_symbol:%s* OR crypto_name:%s*
&rows=5
&fl=crypto_symbol,crypto_name
&group=true
&group.field=crypto_symbol
&wt=json
`, query, query)

solrURL := fmt.Sprintf("http://solr:8983/solr/orders_search/select?%s", url.QueryEscape(solrQuery))
```

#### 4.3 Response
```json
{
  "status": "success",
  "data": {
    "suggestions": [
      {
        "symbol": "BTC",
        "name": "Bitcoin"
      }
    ]
  }
}
```

---

## 5. FILTROS DISPONIBLES

### Endpoint
```
GET /api/v1/filters
```

### Response
```json
{
  "status": "success",
  "data": {
    "filters": {
      "status": ["pending", "executed", "cancelled", "failed"],
      "type": ["buy", "sell"],
      "order_kind": ["market", "limit"],
      "crypto_symbols": ["BTC", "ETH", "BNB", "SOL", "ADA", "XRP", "DOT", "DOGE", ...]
    },
    "sort_fields": ["created_at", "executed_at", "total_amount", "quantity"],
    "sort_orders": ["asc", "desc"]
  }
}
```

---

## 6. REINDEXACIÓN MANUAL

### Endpoint (Admin)
```
POST /api/v1/admin/reindex
Authorization: Bearer {ADMIN_JWT_TOKEN}
```

### Proceso

#### 6.1 Request
```bash
curl -X POST http://localhost:8003/api/v1/admin/reindex \
  -H "Authorization: Bearer admin_token_here"
```

#### 6.2 Obtener todas las órdenes desde MongoDB
```go
func (s *SearchService) ReindexAll() error {
    // 1. Borrar colección actual en Solr
    solrURL := "http://solr:8983/solr/orders_search/update?commit=true"
    deletePayload := `{"delete": {"query": "*:*"}}`
    http.Post(solrURL, "application/json", strings.NewReader(deletePayload))

    // 2. Obtener todas las órdenes desde Orders API
    ordersResp, err := http.Get("http://orders-api:8002/api/v1/admin/orders/all")
    var ordersData AllOrdersResponse
    json.NewDecoder(ordersResp.Body).Decode(&ordersData)

    // 3. Indexar en lotes de 1000
    batchSize := 1000
    for i := 0; i < len(ordersData.Data.Orders); i += batchSize {
        end := i + batchSize
        if end > len(ordersData.Data.Orders) {
            end = len(ordersData.Data.Orders)
        }

        batch := ordersData.Data.Orders[i:end]
        solrDocs := convertToSolrDocs(batch)

        solrJSON, _ := json.Marshal(solrDocs)
        http.Post(solrURL, "application/json", bytes.NewBuffer(solrJSON))
    }

    // 4. Limpiar todo el caché
    s.cacheService.FlushAll()

    log.Info("Reindexed orders", "total", len(ordersData.Data.Orders))
    return nil
}
```

---

## 7. CONFIGURACIÓN DE SOLR

### Schema de Solr (orders_search collection)

```xml
<!-- solr/configsets/orders_search/conf/schema.xml -->
<schema name="orders_search" version="1.6">
  <field name="id" type="string" indexed="true" stored="true" required="true" />
  <field name="user_id" type="plong" indexed="true" stored="true" />
  <field name="type" type="string" indexed="true" stored="true" />
  <field name="status" type="string" indexed="true" stored="true" />
  <field name="crypto_symbol" type="string" indexed="true" stored="true" />
  <field name="crypto_name" type="text_general" indexed="true" stored="true" />
  <field name="quantity" type="pdouble" indexed="true" stored="true" />
  <field name="price" type="pdouble" indexed="true" stored="true" />
  <field name="total_amount" type="pdouble" indexed="true" stored="true" />
  <field name="fee" type="pdouble" indexed="true" stored="true" />
  <field name="order_kind" type="string" indexed="true" stored="true" />
  <field name="created_at" type="pdate" indexed="true" stored="true" />
  <field name="executed_at" type="pdate" indexed="true" stored="true" />

  <uniqueKey>id</uniqueKey>

  <!-- Copy fields for full-text search -->
  <copyField source="crypto_symbol" dest="text" />
  <copyField source="crypto_name" dest="text" />

  <fieldType name="string" class="solr.StrField" />
  <fieldType name="plong" class="solr.LongPointField" />
  <fieldType name="pdouble" class="solr.DoublePointField" />
  <fieldType name="pdate" class="solr.DatePointField" />
  <fieldType name="text_general" class="solr.TextField" positionIncrementGap="100">
    <analyzer type="index">
      <tokenizer class="solr.StandardTokenizerFactory"/>
      <filter class="solr.LowerCaseFilterFactory"/>
    </analyzer>
    <analyzer type="query">
      <tokenizer class="solr.StandardTokenizerFactory"/>
      <filter class="solr.LowerCaseFilterFactory"/>
    </analyzer>
  </fieldType>
</schema>
```

---

## 8. DIAGRAMA DE SECUENCIA - BÚSQUEDA

```
Cliente    Search API   CCache   Memcached   Solr   Orders API   RabbitMQ
  │            │          │          │         │         │           │
  │─POST search>│          │          │         │         │           │
  │            │─Get cache>│          │         │         │           │
  │            │<─MISS─────│          │         │         │           │
  │            │─Get cache────────────>│         │         │           │
  │            │<─MISS─────────────────│         │         │           │
  │            │─Query Solr─────────────────────>│         │           │
  │            │<─Results───────────────────────│         │           │
  │            │─Set cache>│          │         │         │           │
  │            │─Set cache────────────>│         │         │           │
  │<─200 OK────│          │          │         │         │           │
  │            │          │          │         │         │           │
  │            │          │          │         │         │<─Event─────│
  │            │<─────────────────────────────────────────(order.executed)
  │            │─GET order──────────────────────────────>│           │
  │            │<─Order data─────────────────────────────│           │
  │            │─Index Solr───────────────────>│         │           │
  │            │─Invalidate cache>│          │         │           │
  │            │─Invalidate cache────────────>│         │           │
```

---

## 9. EJEMPLOS COMPLETOS

### Búsqueda simple
```bash
curl -X POST http://localhost:8003/api/v1/search \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "*",
    "pagination": {"page": 1, "page_size": 20}
  }'
```

### Búsqueda con filtros
```bash
curl -X POST http://localhost:8003/api/v1/search \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "*",
    "filters": {
      "status": "executed",
      "type": "buy",
      "crypto_symbol": "BTC",
      "start_date": "2025-11-01T00:00:00Z"
    },
    "sort": {
      "field": "total_amount",
      "order": "desc"
    },
    "pagination": {"page": 1, "page_size": 10}
  }'
```

### Ver trending
```bash
curl http://localhost:8003/api/v1/trending
```

### Autocomplete
```bash
curl "http://localhost:8003/api/v1/suggestions?q=eth"
```

### Reindexar (admin)
```bash
curl -X POST http://localhost:8003/api/v1/admin/reindex \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## Resumen

1. **Búsqueda**: Caché multinivel (CCache → Memcached) → Solr query → Response
2. **Sincronización**: RabbitMQ consumer escucha events → Obtiene orden completa → Indexa en Solr → Invalida caché
3. **Trending**: Facets de Solr sobre órdenes ejecutadas últimos 7 días
4. **Autocomplete**: Partial match en Solr con grouping
5. **Reindexación**: Admin endpoint para recargar todo desde Orders API
6. **Performance**: Cache hit rate alto (~80%), queries Solr < 50ms
7. **Escalabilidad**: Memcached distribuido, Solr replicable, consumer asíncrono
