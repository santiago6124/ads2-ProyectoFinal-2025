# 04 - Arquitectura General de CryptoSim

## 📋 Tabla de Contenidos
- [Visión Arquitectónica](#visión-arquitectónica)
- [Patrones Arquitectónicos](#patrones-arquitectónicos)
- [Diagrama de Componentes](#diagrama-de-componentes)
- [Microservicios](#microservicios)
- [Comunicación entre Servicios](#comunicación-entre-servicios)
- [Gestión de Datos](#gestión-de-datos)
- [Seguridad](#seguridad)
- [Escalabilidad](#escalabilidad)

---

## 🎯 Visión Arquitectónica

CryptoSim implementa una **arquitectura de microservicios** moderna, siguiendo principios de diseño como:

### Principios Fundamentales

1. **Single Responsibility Principle (SRP)**
   - Cada microservicio tiene una única responsabilidad de negocio
   - Separación clara de concerns
   - Facilita mantenimiento y testing

2. **Bounded Contexts (DDD)**
   - Cada servicio representa un contexto delimitado
   - Modelo de dominio específico por servicio
   - Autonomía en decisiones de diseño

3. **API-First Design**
   - Contratos de API claramente definidos
   - Documentación como código
   - Versionado de APIs

4. **Event-Driven Architecture**
   - Comunicación asíncrona mediante eventos
   - Desacoplamiento temporal
   - Resiliencia y escalabilidad

5. **Database per Service**
   - Cada servicio gestiona su propia base de datos
   - Tecnología de BD optimizada para cada caso de uso
   - Independencia en esquema y evolución

---

## 🏛️ Patrones Arquitectónicos

### 1. Microservicios Pattern

**Características**:
- Servicios pequeños e independientes
- Deployable independientemente
- Organizados alrededor de capacidades de negocio
- Comunicación vía APIs (HTTP/REST) y mensajería (AMQP)

**Beneficios**:
- ✅ Escalabilidad independiente
- ✅ Desarrollo paralelo por equipos
- ✅ Tecnologías heterogéneas
- ✅ Resiliencia (fallo aislado)

**Desafíos**:
- ⚠️ Complejidad operacional
- ⚠️ Transacciones distribuidas
- ⚠️ Testing de integración
- ⚠️ Monitoreo distribuido

### 2. Event-Driven Pattern

**Implementación**: RabbitMQ como message broker

**Tipos de Eventos**:
```
orders.created       → Nueva orden creada
orders.executed      → Orden ejecutada con éxito
orders.cancelled     → Orden cancelada
orders.failed        → Orden fallida
balance.request      → Solicitud de balance
balance.response     → Respuesta de balance
portfolio.updated    → Portfolio actualizado
```

**Ventajas**:
- Desacoplamiento entre servicios
- Procesamiento asíncrono
- Escalabilidad horizontal
- Reintento automático de mensajes

### 3. API Gateway Pattern (Planificado)

**Estado Actual**: Acceso directo a cada microservicio
**Próxima Versión**: API Gateway centralizado

**Funciones del Gateway**:
- Routing y composición
- Autenticación y autorización
- Rate limiting
- Load balancing
- Caché de respuestas
- Transformación de requests/responses

### 4. CQRS (Command Query Responsibility Segregation)

**Aplicado en**: Search API y Portfolio API

**Separación**:
- **Commands**: Orders API escribe órdenes
- **Queries**: Search API lee y busca órdenes

**Beneficios**:
- Optimización independiente de lectura/escritura
- Escalado diferenciado
- Modelos de datos especializados

### 5. Circuit Breaker Pattern

**Implementación**: En comunicación HTTP entre servicios

```go
// Ejemplo conceptual
if marketDataServiceDown() {
    return cachedPrice() // Fallback a cache
}
```

**Protección contra**:
- Cascading failures
- Timeouts excesivos
- Degradación gradual del sistema

---

## 📊 Diagrama de Componentes Detallado

```
┌─────────────────────────────────────────────────────────────────┐
│                    CAPA DE PRESENTACIÓN                          │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │           Frontend (Next.js / React)                        │ │
│  │  - Dashboard de usuario                                     │ │
│  │  - Gráficos de precios                                      │ │
│  │  - Gestión de órdenes                                       │ │
│  │  - Análisis de portfolio                                    │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │ HTTP/REST
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                  CAPA DE APLICACIÓN (APIs)                       │
│                                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────┐  ┌──────┐ │
│  │  Users   │  │  Orders  │  │  Search  │  │Market │  │Portfo│ │
│  │   API    │  │   API    │  │   API    │  │ Data  │  │ lio  │ │
│  │  :8001   │  │  :8002   │  │  :8003   │  │ :8004 │  │:8005 │ │
│  │          │  │          │  │          │  │       │  │      │ │
│  │ • Auth   │  │ • Orders │  │ • Query  │  │• Price│  │• PnL │ │
│  │ • Users  │  │ • Trades │  │ • Filter │  │• Cache│  │• Risk│ │
│  │ • Balance│  │ • Exec   │  │ • Index  │  │• Aggr │  │• Perf│ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └───┬───┘  └───┬──┘ │
│       │             │             │             │          │    │
└───────┼─────────────┼─────────────┼─────────────┼──────────┼────┘
        │             │             │             │          │
        │   ┌─────────┴─────────────┴─────────────┴────┐     │
        │   │                                           │     │
        ▼   ▼              CAPA DE MENSAJERÍA           ▼     ▼
┌─────────────────────────────────────────────────────────────────┐
│                     RabbitMQ Message Broker                      │
│                                                                  │
│  Exchanges:                      Queues:                         │
│  • orders.events                 • portfolio.updates            │
│  • balance.request.exchange      • search.sync                  │
│  • balance.response.exchange     • balance.request              │
│                                  • balance.response.portfolio   │
└─────────────────────────────────────────────────────────────────┘
        │             │             │             │          │
        ▼             ▼             ▼             ▼          ▼
┌─────────────────────────────────────────────────────────────────┐
│                    CAPA DE PERSISTENCIA                          │
│                                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────┐  ┌──────┐ │
│  │  MySQL   │  │ MongoDB  │  │  Solr    │  │ Redis │  │Mongo │ │
│  │  :3307   │  │  :27017  │  │  :8983   │  │ :6379 │  │:27018│ │
│  │          │  │          │  │          │  │       │  │      │ │
│  │  Users   │  │  Orders  │  │  Search  │  │ Cache │  │Portfo│ │
│  │  Balance │  │  Trades  │  │  Index   │  │ Price │  │ Stats│ │
│  └──────────┘  └──────────┘  └──────────┘  └───────┘  └──────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    CAPA DE INFRAESTRUCTURA                       │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │ Docker       │  │ Docker       │  │ Health Checks          │ │
│  │ Containers   │  │ Compose      │  │ & Monitoring           │ │
│  └──────────────┘  └──────────────┘  └────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🔧 Microservicios Detallados

### 🔵 Users API - Gestión de Usuarios

**Puerto**: 8001
**Base de Datos**: MySQL
**Lenguaje**: Go + Gin + GORM

#### Responsabilidades
- Registro y autenticación de usuarios
- Generación y validación de tokens JWT
- Gestión de balance virtual
- API interna para validación de usuarios entre servicios

#### Componentes Internos
```
users-api/
├── cmd/
│   ├── main.go              # Entry point API HTTP
│   └── worker/
│       └── main.go          # Worker de balance requests
├── internal/
│   ├── config/              # Configuración
│   ├── controllers/         # Handlers HTTP
│   ├── dto/                 # Data Transfer Objects
│   ├── messaging/           # RabbitMQ consumers/publishers
│   ├── middleware/          # JWT, CORS, logging
│   ├── models/              # Modelos de BD (GORM)
│   ├── repository/          # Capa de acceso a datos
│   └── services/            # Lógica de negocio
└── migrations/              # Migraciones SQL
```

#### Endpoints Principales
```
POST   /api/users/register     # Registrar usuario
POST   /api/users/login        # Autenticar y obtener JWT
GET    /api/users/me           # Obtener perfil (autenticado)
GET    /health                 # Health check

# API Interna (requiere INTERNAL_API_KEY)
GET    /internal/users/:id     # Validar usuario
```

#### Flujo de Balance Request (RabbitMQ)
```
1. Portfolio API publica → balance.request.exchange
2. Users Worker consume → balance.request queue
3. Users Worker consulta → MySQL
4. Users Worker publica → balance.response.exchange
5. Portfolio API consume → balance.response.portfolio queue
```

---

### 🟢 Orders API - Gestión de Órdenes

**Puerto**: 8002
**Base de Datos**: MongoDB
**Lenguaje**: Go + Gin

#### Responsabilidades
- Crear órdenes de compra/venta
- Ejecutar órdenes de mercado
- Validar balance antes de ejecución
- Calcular comisiones (maker/taker fees)
- Publicar eventos de órdenes

#### Componentes Internos
```
orders-api/
├── cmd/
│   └── server/
│       └── main.go          # Entry point
├── internal/
│   ├── clients/             # Clientes HTTP a otros servicios
│   │   ├── market_client.go
│   │   ├── user_client.go
│   │   └── user_balance_client.go
│   ├── config/              # Configuración
│   ├── dto/                 # DTOs
│   ├── handlers/            # HTTP handlers
│   ├── messaging/           # RabbitMQ
│   ├── middleware/          # Auth, logging
│   ├── models/              # Modelos de dominio
│   ├── repository/          # MongoDB repository
│   └── services/            # Lógica de negocio
```

#### Endpoints Principales
```
POST   /api/v1/orders              # Crear orden
GET    /api/v1/orders              # Listar órdenes del usuario
GET    /api/v1/orders/:id          # Obtener orden específica
POST   /api/v1/orders/:id/execute  # Ejecutar orden
POST   /api/v1/orders/:id/cancel   # Cancelar orden
GET    /health                     # Health check
```

#### Tipos de Órdenes
```go
type OrderType string
const (
    Buy  OrderType = "buy"
    Sell OrderType = "sell"
)

type OrderKind string
const (
    Market OrderKind = "market"  // Actual
    Limit  OrderKind = "limit"   // Futuro
    Stop   OrderKind = "stop"    // Futuro
)
```

#### Cálculo de Comisiones
```go
// Configuración actual
FEE_MAKER = 0.0008  // 0.08%
FEE_TAKER = 0.0012  // 0.12%
FEE_BASE  = 0.001   // 0.1% (market orders)

// Ejemplo: Compra de 0.1 BTC a $50,000
amount = 0.1 * 50000 = $5,000
fee = 5000 * 0.001 = $5
total_cost = 5000 + 5 = $5,005
```

---

### 🟡 Search API - Búsqueda Avanzada

**Puerto**: 8003
**Base de Datos**: Apache Solr
**Cache**: Memcached
**Lenguaje**: Go + Gin

#### Responsabilidades
- Indexar órdenes en Solr
- Búsqueda full-text
- Filtros avanzados (estado, tipo, fecha, usuario)
- Sincronización con Orders API
- Cache de resultados frecuentes

#### Componentes Internos
```
search-api/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── cache/              # Memcached integration
│   ├── config/
│   ├── dto/
│   ├── handlers/
│   ├── indexer/            # Solr indexing logic
│   ├── messaging/          # RabbitMQ consumer
│   ├── middleware/
│   └── solr/               # Solr client
└── scripts/
    └── solr-init-orders.sh # Inicialización de schema
```

#### Schema de Solr
```xml
<field name="order_id" type="string" indexed="true" stored="true" required="true"/>
<field name="user_id" type="string" indexed="true" stored="true"/>
<field name="crypto_symbol" type="string" indexed="true" stored="true"/>
<field name="order_type" type="string" indexed="true" stored="true"/>
<field name="status" type="string" indexed="true" stored="true"/>
<field name="created_at" type="pdate" indexed="true" stored="true"/>
<field name="executed_at" type="pdate" indexed="true" stored="true"/>
<field name="quantity" type="pfloat" indexed="true" stored="true"/>
<field name="price" type="pfloat" indexed="true" stored="true"/>
<field name="total_amount" type="pfloat" indexed="true" stored="true"/>
```

#### Endpoints
```
POST   /api/v1/search              # Búsqueda con filtros
POST   /api/v1/index/:orderId      # Indexar orden manualmente
DELETE /api/v1/index/:orderId      # Eliminar de índice
GET    /api/v1/health              # Health check
GET    /api/v1/stats               # Estadísticas del índice
```

#### Sincronización con RabbitMQ
```
Exchange: orders.events
Routing Keys: orders.created, orders.executed, orders.cancelled
Queue: search.sync

Flujo:
1. Order creada/ejecutada → Orders API publica evento
2. Search API consume evento
3. Search API indexa/actualiza en Solr
4. Cache invalidado si existe
```

---

### 🔴 Market Data API - Datos de Mercado

**Puerto**: 8004
**Cache**: Redis
**Lenguaje**: Go + Gin

#### Responsabilidades
- Agregación de precios de múltiples fuentes
- Cache de precios en Redis
- Detección de outliers (precios anormales)
- Indicadores técnicos básicos
- Gestión de 50+ criptomonedas

#### Fuentes de Datos
1. **CoinGecko** (principal)
2. **Binance** (secundaria)
3. **Coinbase** (terciaria)

#### Componentes Internos
```
market-data-api/
├── cmd/
│   └── server/
│       ├── main.go
│       └── freecrypto_client.go
├── internal/
│   ├── aggregator/         # Lógica de agregación
│   │   ├── aggregator.go
│   │   ├── outlier_detector.go
│   │   └── technical_analysis.go
│   ├── cache/              # Redis cache manager
│   │   ├── redis.go
│   │   └── price_cache.go
│   ├── config/
│   ├── dto/
│   ├── models/
│   └── providers/          # Clientes de APIs externas
│       ├── binance/
│       ├── coinbase/
│       └── coingecko/
```

#### Algoritmo de Agregación
```go
// Pseudocódigo
func AggregatePrice(symbol string) Price {
    prices := []Price{}

    // Consultar fuentes
    prices = append(prices, coinGecko.GetPrice(symbol))
    prices = append(prices, binance.GetPrice(symbol))
    prices = append(prices, coinbase.GetPrice(symbol))

    // Detectar outliers
    filtered := removeOutliers(prices)

    // Calcular precio agregado (mediana o promedio ponderado)
    finalPrice := calculateMedian(filtered)

    // Cache en Redis (TTL: 30 segundos)
    redis.Set(symbol, finalPrice, 30*time.Second)

    return finalPrice
}
```

#### Endpoints
```
GET    /api/v1/prices                  # Todas las criptos
GET    /api/v1/prices/:symbol          # Precio específico
GET    /api/v1/prices/:symbol/history  # Historial
GET    /health                         # Health check
```

---

### 🟣 Portfolio API - Gestión de Portafolios

**Puerto**: 8005
**Base de Datos**: MongoDB
**Cache**: Redis
**Lenguaje**: Go + Gin

#### Responsabilidades
- Calcular 30+ métricas de portafolio
- Análisis de riesgo (Sharpe, Sortino, Drawdown)
- Seguimiento de rendimiento (ROI, PnL)
- Optimización de portafolio
- Snapshots históricos

#### Métricas Calculadas

**Rendimiento**:
- ROI (Return on Investment)
- Total PnL (Profit and Loss)
- Realized vs Unrealized PnL
- TWRR (Time-Weighted Rate of Return)

**Riesgo**:
- Sharpe Ratio
- Sortino Ratio
- Maximum Drawdown
- Value at Risk (VaR)
- Volatilidad

**Diversificación**:
- Herfindahl Index
- Número de posiciones
- Concentración por asset

**Correlación**:
- Matriz de correlación entre assets
- Beta del portfolio

#### Componentes Internos
```
portfolio-api/
├── cmd/
│   └── main.go
├── internal/
│   ├── analytics/          # Análisis avanzado
│   │   ├── portfolio_analyzer.go
│   │   ├── correlation_analyzer.go
│   │   └── portfolio_optimizer.go
│   ├── calculator/         # Cálculos de métricas
│   │   ├── pnl_calculator.go
│   │   ├── risk_calculator.go
│   │   └── roi_calculator.go
│   ├── clients/            # HTTP clients
│   ├── config/
│   ├── handlers/
│   ├── messaging/          # RabbitMQ
│   ├── models/
│   ├── repository/         # MongoDB
│   ├── scheduler/          # Cron jobs
│   └── services/
```

#### Scheduler de Recalculo
```go
// Cron: cada 15 minutos
PORTFOLIO_CALC_CRON = "0 */15 * * * *"

Tareas:
1. Recalcular métricas de todos los portfolios activos
2. Actualizar precios actuales desde Market Data API
3. Generar snapshots históricos
4. Invalidar cache
```

#### Endpoints
```
GET    /api/portfolios/:userId           # Portfolio del usuario
GET    /api/portfolios/:userId/holdings  # Holdings detallados
GET    /api/portfolios/:userId/performance # Métricas de rendimiento
GET    /api/portfolios/:userId/risk      # Métricas de riesgo
GET    /api/portfolios/:userId/snapshots # Snapshots históricos
POST   /api/portfolios/:userId/calculate # Recalcular manual
GET    /health                           # Health check
```

---

## 🔄 Comunicación entre Servicios

### 1. Comunicación Síncrona (HTTP/REST)

**Patrón**: Request-Response
**Uso**: Cuando se necesita respuesta inmediata

#### Ejemplos

**Orders API → Market Data API**
```go
// Obtener precio actual para ejecutar orden
price, err := marketClient.GetPrice(ctx, "BTC")
```

**Orders API → Users API (Interno)**
```go
// Validar que el usuario existe
user, err := userClient.GetUser(ctx, userId, internalAPIKey)
```

**Portfolio API → Market Data API**
```go
// Obtener precios actuales para calcular portfolio
prices, err := marketClient.GetMultiplePrices(ctx, symbols)
```

### 2. Comunicación Asíncrona (RabbitMQ)

**Patrón**: Pub/Sub
**Uso**: Eventos que no requieren respuesta inmediata

#### Exchanges y Routing

```
┌─────────────────────────────────────────────────────┐
│                  orders.events                       │
│                  (topic exchange)                    │
└───────┬─────────────────────┬───────────────────────┘
        │                     │
        │ orders.executed     │ orders.created
        │                     │ orders.cancelled
        ▼                     ▼
┌───────────────┐     ┌───────────────────┐
│ portfolio.    │     │   search.sync     │
│ updates       │     │                   │
│ (queue)       │     │   (queue)         │
└───────────────┘     └───────────────────┘
        │                     │
        ▼                     ▼
┌───────────────┐     ┌───────────────────┐
│ Portfolio API │     │   Search API      │
│ (consumer)    │     │   (consumer)      │
└───────────────┘     └───────────────────┘
```

#### Ejemplo de Publicación
```go
// Orders API publica orden ejecutada
event := OrderExecutedEvent{
    OrderID: order.ID,
    UserID: order.UserID,
    Symbol: order.CryptoSymbol,
    Type: order.Type,
    Quantity: order.Quantity,
    Price: order.ExecutionPrice,
    Timestamp: time.Now(),
}

err := rabbitmq.Publish(
    "orders.events",        // exchange
    "orders.executed",      // routing key
    event,                  // payload
)
```

#### Ejemplo de Consumo
```go
// Portfolio API consume evento
func (s *Service) ConsumeOrderExecuted(msg OrderExecutedEvent) {
    // Actualizar holdings
    s.updateHoldings(msg.UserID, msg.Symbol, msg.Quantity)

    // Recalcular métricas
    s.recalculatePortfolio(msg.UserID)

    // Publicar evento de portfolio actualizado (si es necesario)
    s.publishPortfolioUpdated(msg.UserID)
}
```

### 3. Request-Reply Pattern (Balance Query)

**Caso especial**: Portfolio necesita balance actual de usuario

```
┌─────────────┐                           ┌─────────────┐
│ Portfolio   │                           │ Users       │
│ API         │                           │ Worker      │
└──────┬──────┘                           └──────┬──────┘
       │                                         │
       │ 1. Publish balance.request              │
       ├─────────────────────────────────────────>
       │    {userID: "123", correlationID}       │
       │                                         │
       │                           2. Query MySQL│
       │                                   ◄─────┤
       │                                         │
       │ 3. Publish balance.response             │
       <─────────────────────────────────────────┤
       │    {balance: 5000, correlationID}       │
       │                                         │
       │ 4. Match by correlationID               │
       │    and continue processing              │
       ▼                                         │
```

---

## 💾 Gestión de Datos

### Estrategia de Persistencia

| Servicio | BD | Justificación |
|----------|-----|---------------|
| Users | MySQL | ACID, relaciones, transacciones |
| Orders | MongoDB | Documentos flexibles, alto write throughput |
| Search | Solr | Optimizado para búsqueda full-text |
| Market Data | Redis | Cache de alta velocidad, TTL automático |
| Portfolio | MongoDB | Documentos complejos, agregaciones |

### Modelo de Datos por Servicio

#### Users (MySQL)
```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    balance DECIMAL(20, 8) DEFAULT 100000.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_email (email),
    INDEX idx_username (username)
);
```

#### Orders (MongoDB)
```json
{
  "_id": "ObjectId",
  "user_id": "string",
  "crypto_symbol": "BTC",
  "order_type": "buy|sell",
  "order_kind": "market|limit",
  "quantity": 0.1,
  "price_at_creation": 50000.00,
  "execution_price": 50050.00,
  "total_amount": 5005.00,
  "fee": 5.00,
  "fee_percentage": 0.001,
  "status": "pending|executed|cancelled|failed",
  "created_at": "ISODate",
  "executed_at": "ISODate|null",
  "error_message": "string|null"
}
```

#### Portfolio (MongoDB)
```json
{
  "_id": "ObjectId",
  "user_id": "string",
  "holdings": [
    {
      "symbol": "BTC",
      "quantity": 0.5,
      "average_buy_price": 48000.00,
      "current_price": 50000.00,
      "total_value": 25000.00,
      "unrealized_pnl": 1000.00,
      "percentage": 50.0
    }
  ],
  "total_value": 50000.00,
  "cash_balance": 25000.00,
  "metrics": {
    "roi": 0.15,
    "total_pnl": 7500.00,
    "sharpe_ratio": 1.5,
    "sortino_ratio": 2.1,
    "max_drawdown": -0.12,
    "volatility": 0.25
  },
  "last_calculated": "ISODate",
  "created_at": "ISODate",
  "updated_at": "ISODate"
}
```

### Consistencia Eventual

**Concepto**: Los datos pueden estar temporalmente inconsistentes entre servicios

**Ejemplo**:
```
1. Orden ejecutada en Orders API → MongoDB actualizado
2. Evento publicado a RabbitMQ
3. Portfolio API recibe evento (pequeño delay)
4. Portfolio recalculado (eventual consistency)
```

**Ventajas**:
- Mayor disponibilidad
- Mejor rendimiento
- Escalabilidad horizontal

**Desafíos**:
- UI debe manejar estados transitorios
- Necesidad de idempotencia en consumers
- Complejidad en debugging

---

## 🔒 Seguridad

### 1. Autenticación JWT

**Flujo**:
```
1. Usuario hace login → Users API
2. Users API valida credenciales
3. Genera JWT firmado con JWT_SECRET
4. Frontend almacena token (localStorage/cookies)
5. Cada request incluye: Authorization: Bearer <token>
6. Cada servicio valida JWT independientemente
```

**Estructura del Token**:
```json
{
  "sub": "user_id_123",
  "email": "user@example.com",
  "username": "trader123",
  "iat": 1699999999,
  "exp": 1700086399,
  "iss": "users-api",
  "aud": "cryptosim"
}
```

### 2. Comunicación Interna

**API Keys internas**: Para comunicación service-to-service

```go
// Orders API → Users API (interno)
req.Header.Set("X-Internal-API-Key", os.Getenv("INTERNAL_API_KEY"))
```

### 3. Validación de Inputs

**Validación con go-playground/validator**:
```go
type OrderRequest struct {
    CryptoSymbol string  `json:"crypto_symbol" validate:"required,min=2,max=10"`
    Quantity     float64 `json:"quantity" validate:"required,gt=0"`
    OrderType    string  `json:"order_type" validate:"required,oneof=buy sell"`
}
```

### 4. Rate Limiting (Planificado)

**En API Gateway**:
- 1000 requests/hora por usuario
- 10 requests/segundo por IP
- Throttling en endpoints de escritura

### 5. HTTPS/TLS (Producción)

**Configuración**:
- Certificados SSL/TLS
- HTTP Strict Transport Security (HSTS)
- Redirección HTTP → HTTPS

---

## 📈 Escalabilidad

### Escalado Horizontal

**Servicios Stateless**: Pueden escalar horizontalmente

```yaml
# Docker Compose con replicas
services:
  orders-api:
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
```

**Load Balancing**: Nginx/Traefik como reverse proxy

```
           ┌──────────────┐
           │ Load Balancer│
           └──────┬───────┘
                  │
         ┌────────┼────────┐
         ▼        ▼        ▼
    [Orders-1][Orders-2][Orders-3]
```

### Escalado de Bases de Datos

**MySQL**:
- Read replicas para consultas
- Sharding por user_id (futuro)

**MongoDB**:
- Replica Set (3 nodos)
- Sharding por user_id o date range

**Redis**:
- Redis Cluster
- Particionamiento de keys

### Optimizaciones de Performance

**1. Caching en múltiples niveles**:
```
Browser Cache (5min)
  ↓
CDN Cache (1h)
  ↓
Redis Cache (30s - 1h)
  ↓
Database
```

**2. Connection Pooling**:
```go
// MongoDB
MongoDBMaxPoolSize = 100
MongoDBMinPoolSize = 10

// MySQL
DBMaxOpenConns = 100
DBMaxIdleConns = 10
```

**3. Batch Processing**:
- Procesar múltiples mensajes RabbitMQ en batch
- Bulk inserts en MongoDB
- Pipeline de Redis para múltiples operaciones

---

## 🎯 Próximos Pasos Arquitectónicos

### Versión 1.1
- [ ] API Gateway (Kong/Nginx)
- [ ] Service Mesh (Istio/Linkerd)
- [ ] Distributed Tracing (Jaeger)
- [ ] Centralized Logging (ELK Stack)

### Versión 2.0
- [ ] Kubernetes deployment
- [ ] Autoscaling basado en métricas
- [ ] Multi-region deployment
- [ ] GraphQL Federation
- [ ] Event Sourcing completo

---

## 📚 Documentos Relacionados

- **[Arquitectura RabbitMQ](ARQUITECTURA-RABBITMQ.md)**: Mensajería detallada
- **[Stack Tecnológico](05-STACK-TECNOLOGICO.md)**: Tecnologías en profundidad
- **[Infraestructura](06-INFRAESTRUCTURA.md)**: Docker, BD, y despliegue

---

**Siguiente**: [05 - Stack Tecnológico](05-STACK-TECNOLOGICO.md) →
