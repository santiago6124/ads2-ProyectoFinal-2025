# 🚀 CryptoSim - Plataforma de Simulación de Trading de Criptomonedas

## 📋 Descripción del Proyecto

CryptoSim es una plataforma educativa de simulación de trading de criptomonedas que permite a los usuarios aprender y practicar estrategias de inversión sin riesgo financiero real. Los usuarios reciben un saldo virtual inicial y pueden operar con precios de mercado reales, gestionar su portafolio, analizar rendimientos y competir con otros traders en un ambiente seguro y controlado.

## 🎯 Objetivos del Sistema

- Proporcionar un entorno seguro para aprender trading de criptomonedas
- Simular condiciones reales del mercado con datos actualizados
- Ofrecer herramientas de análisis y seguimiento de rendimiento
- Fomentar el aprendizaje mediante rankings y estadísticas comparativas
- Gestionar portafolios virtuales con múltiples criptomonedas

## 🏗️ Arquitectura de Microservicios

### Diagrama de Arquitectura

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                             │
│  [Login] [Register] [Dashboard] [Trading] [Portfolio] [Admin Panel]  │
└────────────┬────────────────────────────────────────┬───────────────┘
             │              HTTP/JSON                  │
┌────────────▼────────────────────────────────────────▼───────────────┐
│                          API Gateway                                 │
└──┬──────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┬────────┘
   │      │      │      │      │      │      │      │      │
┌──▼──┐┌──▼──┐┌──▼──┐┌──▼──┐┌──▼──┐┌──▼──┐┌──▼──┐┌──▼──┐┌──▼──┐
│Users││Orders││Search││Market││Port-││Wallet││Rank-││Noti-││Audit│
│ API ││  API  ││ API  ││Data  ││folio││ API  ││ing  ││fica-││ API │
│     ││       ││      ││ API  ││ API ││      ││ API ││tions││     │
└──┬──┘└──┬───┘└──┬──┘└──┬──┘└──┬──┘└──┬──┘└──┬──┘└──┬──┘└──┬──┘
   │      │       │      │      │      │      │      │      │
┌──▼──┐┌──▼───┐┌──▼──┐┌──▼──┐┌──▼──┐┌──▼──┐┌──▼──┐   │      │
│MySQL││MongoDB││SolR ││Redis││Mongo││Mongo││Post-│   │      │
│     ││       ││     ││     ││ DB  ││ DB  ││greSQL   │      │
└─────┘└───────┘└─────┘└─────┘└─────┘└─────┘└─────┘   │      │
                                                        │      │
                    ┌─────────────────────────┐        │      │
                    │     RabbitMQ Broker     │◄───────┴──────┘
                    │   [Orders] [Portfolio]  │
                    │   [Notifications]       │
                    └─────────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │    Memcached      │
                    │  (Distributed)    │
                    └───────────────────┘
```

## 📦 Microservicios Detallados

### 1. Users API (`users-api`)
**Responsabilidad:** Gestión completa de usuarios, autenticación y autorización.

**Tecnologías:**
- Lenguaje: Go
- Base de datos: MySQL (GORM)
- Autenticación: JWT
- Hashing: bcrypt

**Endpoints principales:**
- `POST /api/users/register` - Registro de nuevo usuario
- `POST /api/users/login` - Autenticación y generación de JWT
- `GET /api/users/:id` - Obtener usuario por ID
- `PUT /api/users/:id` - Actualizar perfil de usuario
- `GET /api/users/:id/verify` - Verificar existencia de usuario (interno)
- `POST /api/users/:id/upgrade` - Cambiar usuario a admin

**Modelo de datos (MySQL):**
```sql
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    role ENUM('normal', 'admin') DEFAULT 'normal',
    initial_balance DECIMAL(15,2) DEFAULT 100000.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_login TIMESTAMP NULL,
    is_active BOOLEAN DEFAULT TRUE,
    preferences JSON,
    INDEX idx_email (email),
    INDEX idx_username (username)
);
```

### 2. Orders API (`orders-api`)
**Responsabilidad:** Gestión de órdenes de compra/venta, ejecución de trades y sincronización con billetera.

**Tecnologías:**
- Lenguaje: Go
- Base de datos: MongoDB
- Message Broker: RabbitMQ
- Concurrencia: Go Routines + Channels + Wait Groups

**Endpoints principales:**
- `POST /api/orders` - Crear nueva orden (con cálculo concurrente)
- `GET /api/orders/:id` - Obtener orden por ID
- `PUT /api/orders/:id` - Actualizar orden (solo admin o owner)
- `DELETE /api/orders/:id` - Cancelar orden pendiente
- `GET /api/orders/user/:userId` - Listar órdenes de un usuario
- `POST /api/orders/:id/execute` - Ejecutar orden manualmente (acción especial)

**Modelo de datos (MongoDB):**
```javascript
{
  "_id": ObjectId,
  "user_id": Number,
  "type": "buy" | "sell",
  "status": "pending" | "executed" | "cancelled" | "failed",
  "crypto_symbol": String,
  "crypto_name": String,
  "quantity": Decimal128,
  "order_price": Decimal128,
  "execution_price": Decimal128,
  "total_amount": Decimal128,
  "fee": Decimal128,
  "created_at": ISODate,
  "executed_at": ISODate,
  "updated_at": ISODate,
  "execution_details": {
    "market_price_at_execution": Decimal128,
    "slippage": Decimal128,
    "execution_time_ms": Number
  },
  "metadata": {
    "ip_address": String,
    "user_agent": String,
    "platform": String
  }
}
```

**Proceso Concurrente de Ejecución:**
```go
// Ejemplo de estructura del cálculo concurrente
func ExecuteOrder(order *Order) (*ExecutionResult, error) {
    var wg sync.WaitGroup
    resultChan := make(chan interface{}, 4)
    errorChan := make(chan error, 4)
    
    // 1. Validar saldo del usuario
    wg.Add(1)
    go validateUserBalance(order, resultChan, errorChan, &wg)
    
    // 2. Obtener precio actual del mercado
    wg.Add(1)
    go fetchCurrentMarketPrice(order, resultChan, errorChan, &wg)
    
    // 3. Calcular fees y comisiones
    wg.Add(1)
    go calculateFeesAndCommissions(order, resultChan, errorChan, &wg)
    
    // 4. Simular latencia de mercado y slippage
    wg.Add(1)
    go simulateMarketConditions(order, resultChan, errorChan, &wg)
    
    wg.Wait()
    close(resultChan)
    close(errorChan)
    
    // Procesar resultados y actualizar orden
    return processExecutionResults(resultChan, errorChan)
}
```

### 3. Search API (`search-api`)
**Responsabilidad:** Búsqueda y filtrado de criptomonedas disponibles para trading.

**Tecnologías:**
- Lenguaje: Go
- Motor de búsqueda: Apache SolR
- Cache: CCache (local) + Memcached (distribuido)
- Message Consumer: RabbitMQ

**Endpoints principales:**
- `GET /api/search/cryptos` - Búsqueda paginada de criptomonedas
- `GET /api/search/cryptos/trending` - Criptomonedas en tendencia
- `GET /api/search/cryptos/filters` - Obtener filtros disponibles
- `POST /api/search/reindex` - Reindexar datos (admin only)

**Esquema SolR:**
```xml
<field name="id" type="string" indexed="true" stored="true" required="true"/>
<field name="symbol" type="string" indexed="true" stored="true"/>
<field name="name" type="string" indexed="true" stored="true"/>
<field name="current_price" type="pdouble" indexed="true" stored="true"/>
<field name="market_cap" type="plong" indexed="true" stored="true"/>
<field name="volume_24h" type="plong" indexed="true" stored="true"/>
<field name="price_change_24h" type="pdouble" indexed="true" stored="true"/>
<field name="price_change_7d" type="pdouble" indexed="true" stored="true"/>
<field name="total_supply" type="plong" indexed="true" stored="true"/>
<field name="circulating_supply" type="plong" indexed="true" stored="true"/>
<field name="category" type="string" indexed="true" stored="true" multiValued="true"/>
<field name="description" type="text_general" indexed="true" stored="true"/>
<field name="trending_score" type="pint" indexed="true" stored="true"/>
<field name="last_updated" type="pdate" indexed="true" stored="true"/>
<field name="is_active" type="boolean" indexed="true" stored="true"/>
```

### 4. Market Data API (`market-data-api`)
**Responsabilidad:** Obtención y gestión de datos de mercado en tiempo real.

**Tecnologías:**
- Lenguaje: Go
- Cache: Redis
- APIs externas: CoinGecko, Binance
- WebSockets para actualizaciones en tiempo real

**Endpoints principales:**
- `GET /api/market/price/:symbol` - Precio actual de una criptomoneda
- `GET /api/market/prices` - Precios de múltiples criptomonedas
- `GET /api/market/history/:symbol` - Histórico de precios
- `GET /api/market/stats/:symbol` - Estadísticas de mercado
- `WS /api/market/stream` - Stream de precios en tiempo real

**Modelo de datos (Redis):**
```javascript
// Precio actual (TTL: 30 segundos)
market:price:{symbol} = {
  "symbol": "BTC",
  "price": 45000.50,
  "timestamp": 1699123456,
  "source": "binance"
}

// Histórico 24h (TTL: 1 hora)
market:history:24h:{symbol} = [
  {"time": 1699123456, "price": 45000.50, "volume": 1234567},
  ...
]

// Estadísticas (TTL: 5 minutos)
market:stats:{symbol} = {
  "high_24h": 46000,
  "low_24h": 44000,
  "volume_24h": 987654321,
  "market_cap": 876543210000
}
```

### 5. Portfolio API (`portfolio-api`)
**Responsabilidad:** Cálculo y gestión del portafolio de inversiones de cada usuario.

**Tecnologías:**
- Lenguaje: Go
- Base de datos: MongoDB
- Message Broker: RabbitMQ (consumidor)

**Endpoints principales:**
- `GET /api/portfolio/:userId` - Obtener portafolio completo
- `GET /api/portfolio/:userId/performance` - Métricas de rendimiento
- `GET /api/portfolio/:userId/history` - Histórico de valor del portafolio
- `GET /api/portfolio/:userId/holdings` - Holdings actuales
- `POST /api/portfolio/:userId/snapshot` - Crear snapshot del portafolio

**Modelo de datos (MongoDB):**
```javascript
// Colección: portfolios
{
  "_id": ObjectId,
  "user_id": Number,
  "total_value": Decimal128,
  "total_invested": Decimal128,
  "profit_loss": Decimal128,
  "profit_loss_percentage": Decimal128,
  "holdings": [
    {
      "symbol": String,
      "name": String,
      "quantity": Decimal128,
      "average_buy_price": Decimal128,
      "current_price": Decimal128,
      "current_value": Decimal128,
      "profit_loss": Decimal128,
      "profit_loss_percentage": Decimal128,
      "last_updated": ISODate
    }
  ],
  "performance": {
    "daily_change": Decimal128,
    "weekly_change": Decimal128,
    "monthly_change": Decimal128,
    "yearly_change": Decimal128,
    "all_time_high": Decimal128,
    "all_time_low": Decimal128
  },
  "last_calculated": ISODate,
  "created_at": ISODate,
  "updated_at": ISODate
}

// Colección: portfolio_snapshots (histórico)
{
  "_id": ObjectId,
  "user_id": Number,
  "timestamp": ISODate,
  "total_value": Decimal128,
  "holdings_snapshot": Array,
  "metadata": Object
}
```

### 6. Wallet API (`wallet-api`)
**Responsabilidad:** Gestión de la billetera virtual y saldo de los usuarios.

**Tecnologías:**
- Lenguaje: Go
- Base de datos: MongoDB
- Transacciones ACID para operaciones críticas

**Endpoints principales:**
- `GET /api/wallet/:userId` - Obtener billetera del usuario
- `GET /api/wallet/:userId/balance` - Obtener saldo disponible
- `POST /api/wallet/:userId/deposit` - Depositar fondos virtuales (admin)
- `POST /api/wallet/:userId/withdraw` - Retirar fondos virtuales
- `GET /api/wallet/:userId/transactions` - Historial de transacciones

**Modelo de datos (MongoDB):**
```javascript
// Colección: wallets
{
  "_id": ObjectId,
  "user_id": Number,
  "available_balance": Decimal128,
  "locked_balance": Decimal128,
  "total_balance": Decimal128,
  "currency": "USD",
  "created_at": ISODate,
  "updated_at": ISODate,
  "last_transaction": ISODate
}

// Colección: wallet_transactions
{
  "_id": ObjectId,
  "wallet_id": ObjectId,
  "user_id": Number,
  "type": "deposit" | "withdrawal" | "order_lock" | "order_release" | "order_execute",
  "amount": Decimal128,
  "balance_before": Decimal128,
  "balance_after": Decimal128,
  "reference_type": "order" | "admin_action" | "system",
  "reference_id": String,
  "description": String,
  "timestamp": ISODate,
  "metadata": Object
}
```

### 7. Ranking API (`ranking-api`)
**Responsabilidad:** Cálculo y gestión de rankings y leaderboards.

**Tecnologías:**
- Lenguaje: Go
- Base de datos: PostgreSQL (para consultas complejas y agregaciones)
- Cache: Redis (para rankings en tiempo real)

**Endpoints principales:**
- `GET /api/ranking/global` - Ranking global de traders
- `GET /api/ranking/weekly` - Ranking semanal
- `GET /api/ranking/monthly` - Ranking mensual
- `GET /api/ranking/user/:userId` - Posición de un usuario específico
- `GET /api/ranking/stats` - Estadísticas generales

**Modelo de datos (PostgreSQL):**
```sql
CREATE TABLE rankings (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    period_type VARCHAR(20) NOT NULL, -- 'daily', 'weekly', 'monthly', 'all_time'
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    rank_position INTEGER NOT NULL,
    total_profit_loss DECIMAL(15,2),
    profit_loss_percentage DECIMAL(10,4),
    total_trades INTEGER,
    successful_trades INTEGER,
    win_rate DECIMAL(5,2),
    best_trade DECIMAL(15,2),
    worst_trade DECIMAL(15,2),
    score DECIMAL(15,2), -- Puntaje calculado con fórmula personalizada
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, period_type, period_start)
);

CREATE INDEX idx_rankings_period ON rankings(period_type, period_start);
CREATE INDEX idx_rankings_user ON rankings(user_id);
CREATE INDEX idx_rankings_score ON rankings(score DESC);
```

### 8. Notifications API (`notifications-api`)
**Responsabilidad:** Gestión y envío de notificaciones a usuarios.

**Tecnologías:**
- Lenguaje: Go
- Base de datos: MongoDB
- Message Consumer: RabbitMQ
- WebSockets para notificaciones en tiempo real

**Endpoints principales:**
- `GET /api/notifications/:userId` - Obtener notificaciones del usuario
- `PUT /api/notifications/:id/read` - Marcar como leída
- `DELETE /api/notifications/:id` - Eliminar notificación
- `POST /api/notifications/preferences/:userId` - Configurar preferencias
- `WS /api/notifications/stream/:userId` - Stream de notificaciones

**Modelo de datos (MongoDB):**
```javascript
{
  "_id": ObjectId,
  "user_id": Number,
  "type": "order_executed" | "price_alert" | "portfolio_milestone" | "system",
  "title": String,
  "message": String,
  "priority": "low" | "medium" | "high",
  "is_read": Boolean,
  "data": {
    "reference_type": String,
    "reference_id": String,
    "additional_info": Object
  },
  "created_at": ISODate,
  "read_at": ISODate,
  "expires_at": ISODate
}
```

### 9. Audit API (`audit-api`)
**Responsabilidad:** Registro y auditoría de todas las operaciones críticas del sistema.

**Tecnologías:**
- Lenguaje: Go
- Base de datos: MongoDB
- Message Consumer: RabbitMQ

**Endpoints principales:**
- `GET /api/audit/logs` - Obtener logs de auditoría (admin only)
- `GET /api/audit/user/:userId` - Logs de un usuario específico
- `GET /api/audit/stats` - Estadísticas de auditoría
- `POST /api/audit/export` - Exportar logs

**Modelo de datos (MongoDB):**
```javascript
{
  "_id": ObjectId,
  "timestamp": ISODate,
  "user_id": Number,
  "action": String,
  "resource_type": String,
  "resource_id": String,
  "ip_address": String,
  "user_agent": String,
  "request_method": String,
  "request_path": String,
  "response_status": Number,
  "execution_time_ms": Number,
  "changes": {
    "before": Object,
    "after": Object
  },
  "metadata": Object
}
```

## 🖥️ Frontend (React)

### Componentes Principales

#### 1. Páginas
- **Login/Register**: Autenticación y registro con validación
- **Dashboard**: Vista general del portafolio y mercado
- **Trading**: Interfaz de trading con gráficos en tiempo real
- **Portfolio**: Detalle del portafolio y rendimiento
- **Market**: Lista de criptomonedas con búsqueda y filtros
- **Rankings**: Leaderboard y estadísticas
- **Admin Panel**: Gestión del sistema (solo admins)
- **Profile**: Configuración de usuario y preferencias

#### 2. Componentes Reutilizables
```javascript
// Estructura de componentes
src/
├── components/
│   ├── common/
│   │   ├── Header.jsx
│   │   ├── Footer.jsx
│   │   ├── LoadingSpinner.jsx
│   │   └── ErrorBoundary.jsx
│   ├── auth/
│   │   ├── LoginForm.jsx
│   │   ├── RegisterForm.jsx
│   │   └── ProtectedRoute.jsx
│   ├── trading/
│   │   ├── OrderForm.jsx
│   │   ├── OrderBook.jsx
│   │   ├── PriceChart.jsx
│   │   └── TradingView.jsx
│   ├── portfolio/
│   │   ├── PortfolioSummary.jsx
│   │   ├── HoldingsList.jsx
│   │   ├── PerformanceChart.jsx
│   │   └── TransactionHistory.jsx
│   └── market/
│       ├── CryptoList.jsx
│       ├── CryptoCard.jsx
│       ├── SearchBar.jsx
│       └── FilterPanel.jsx
```

#### 3. Estado Global (Redux/Context API)
```javascript
// Estado de la aplicación
{
  auth: {
    user: Object,
    token: String,
    isAuthenticated: Boolean
  },
  portfolio: {
    holdings: Array,
    totalValue: Number,
    performance: Object
  },
  market: {
    cryptos: Array,
    prices: Object,
    loading: Boolean
  },
  orders: {
    active: Array,
    history: Array,
    pending: Array
  },
  notifications: {
    unread: Number,
    items: Array
  }
}
```

## 🔄 Flujos de Trabajo Principales

### 1. Flujo de Trading
```
Usuario → Frontend → Orders API → Market API → Wallet API → Portfolio API → RabbitMQ
    ↑                                                                            ↓
    ←────────────────── Confirmación ←──────────────────────────────────────────
```

### 2. Flujo de Búsqueda con Cache
```
Usuario → Frontend → Search API → CCache → Memcached → SolR
    ↑                     ↓
    ←── Resultados Paginados
```

## 🐳 Docker Compose

```yaml
version: '3.8'

services:
  # Frontend
  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    environment:
      - REACT_APP_API_URL=http://localhost:8080
    depends_on:
      - users-api
      - orders-api
      - search-api

  # Backend Services
  users-api:
    build: ./backend/users-api
    ports:
      - "8001:8001"
    environment:
      - DB_HOST=mysql
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - mysql

  orders-api:
    build: ./backend/orders-api
    ports:
      - "8002:8002"
    environment:
      - MONGO_URI=mongodb://mongodb:27017/orders
      - RABBITMQ_URL=amqp://rabbitmq:5672
    depends_on:
      - mongodb
      - rabbitmq

  search-api:
    build: ./backend/search-api
    ports:
      - "8003:8003"
    environment:
      - SOLR_URL=http://solr:8983/solr
      - MEMCACHED_HOST=memcached:11211
      - RABBITMQ_URL=amqp://rabbitmq:5672
    depends_on:
      - solr
      - memcached
      - rabbitmq

  market-data-api:
    build: ./backend/market-data-api
    ports:
      - "8004:8004"
    environment:
      - REDIS_URL=redis://redis:6379
      - COINGECKO_API_KEY=${COINGECKO_API_KEY}
    depends_on:
      - redis

  portfolio-api:
    build: ./backend/portfolio-api
    ports:
      - "8005:8005"
    environment:
      - MONGO_URI=mongodb://mongodb:27017/portfolio
      - RABBITMQ_URL=amqp://rabbitmq:5672
    depends_on:
      - mongodb
      - rabbitmq

  wallet-api:
    build: ./backend/wallet-api
    ports:
      - "8006:8006"
    environment:
      - MONGO_URI=mongodb://mongodb:27017/wallet
    depends_on:
      - mongodb

  ranking-api:
    build: ./backend/ranking-api
    ports:
      - "8007:8007"
    environment:
      - POSTGRES_URL=postgres://postgres:password@postgresql:5432/rankings
      - REDIS_URL=redis://redis:6379
    depends_on:
      - postgresql
      - redis

  notifications-api:
    build: ./backend/notifications-api
    ports:
      - "8008:8008"
    environment:
      - MONGO_URI=mongodb://mongodb:27017/notifications
      - RABBITMQ_URL=amqp://rabbitmq:5672
    depends_on:
      - mongodb
      - rabbitmq

  audit-api:
    build: ./backend/audit-api
    ports:
      - "8009:8009"
    environment:
      - MONGO_URI=mongodb://mongodb:27017/audit
      - RABBITMQ_URL=amqp://rabbitmq:5672
    depends_on:
      - mongodb
      - rabbitmq

  # Databases and Infrastructure
  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=rootpassword
      - MYSQL_DATABASE=users_db
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql

  mongodb:
    image: mongo:6.0
    ports:
      - "27017:27017"
    volumes:
      - mongo_data:/data/db

  postgresql:
    image: postgres:15
    environment:
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=rankings
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  rabbitmq:
    image: rabbitmq:3-management
    ports:
      - "5672:5672"
      - "15672:15672"
    environment:
      - RABBITMQ_DEFAULT_USER=admin
      - RABBITMQ_DEFAULT_PASS=admin
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq

  solr:
    image: solr:9
    ports:
      - "8983:8983"
    volumes:
      - solr_data:/var/solr
    command:
      - solr-precreate
      - cryptos

  memcached:
    image: memcached:1.6-alpine
    ports:
      - "11211:11211"

volumes:
  mysql_data:
  mongo_data:
  postgres_data:
  redis_data:
  rabbitmq_data:
  solr_data:
```

## 🧪 Testing

### Ejemplo de Test para Orders Service
```go
// orders_service_test.go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestCreateOrder_Success(t *testing.T) {
    // Arrange
    mockRepo := new(MockOrderRepository)
    mockWalletClient := new(MockWalletClient)
    mockMarketClient := new(MockMarketClient)
    
    service := NewOrderService(mockRepo, mockWalletClient, mockMarketClient)
    
    order := &Order{
        UserID: 1,
        Type: "buy",
        Symbol: "BTC",
        Quantity: 0.1,
    }
    
    mockMarketClient.On("GetPrice", "BTC").Return(45000.0, nil)
    mockWalletClient.On("GetBalance", 1).Return(10000.0, nil)
    mockWalletClient.On("LockFunds", 1, 4500.0).Return(nil)
    mockRepo.On("Create", mock.Anything).Return(nil)
    
    // Act
    result, err := service.CreateOrder(order)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "executed", result.Status)
    mockRepo.AssertExpectations(t)
    mockWalletClient.AssertExpectations(t)
    mockMarketClient.AssertExpectations(t)
}

func TestCreateOrder_InsufficientBalance(t *testing.T) {
    // Arrange
    mockRepo := new(MockOrderRepository)
    mockWalletClient := new(MockWalletClient)
    mockMarketClient := new(MockMarketClient)
    
    service := NewOrderService(mockRepo, mockWalletClient, mockMarketClient)
    
    order := &Order{
        UserID: 1,
        Type: "buy",
        Symbol: "BTC",
        Quantity: 1.0,
    }
    
    mockMarketClient.On("GetPrice", "BTC").Return(45000.0, nil)
    mockWalletClient.On("GetBalance", 1).Return(1000.0, nil) // Saldo insuficiente
    
    // Act
    result, err := service.CreateOrder(order)
    
    // Assert
    assert.Error(t, err)
    assert.Nil(t