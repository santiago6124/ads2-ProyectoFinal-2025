# 📈 Orders API - Microservicio de Gestión de Órdenes

## 📋 Descripción

El microservicio **Orders API** es el núcleo del sistema de trading de CryptoSim. Gestiona la creación, ejecución y administración de órdenes de compra/venta de criptomonedas, implementando un sofisticado sistema de procesamiento concurrente usando Go Routines, Channels y Wait Groups para simular las condiciones reales del mercado.

## 🎯 Responsabilidades

- **Gestión de Órdenes**: Creación, actualización y cancelación de órdenes de trading
- **Ejecución Concurrente**: Procesamiento paralelo de validaciones y cálculos
- **Validación de Fondos**: Verificación de saldo con Wallet API
- **Precio de Mercado**: Integración con Market Data API para precios actuales
- **Notificaciones**: Publicación de eventos en RabbitMQ
- **Cálculo de Fees**: Computación de comisiones y costos de transacción
- **Simulación de Mercado**: Slippage y latencia realista

## 🏗️ Arquitectura

### Estructura del Proyecto
```
orders-api/
├── cmd/
│   └── main.go                    # Punto de entrada
├── internal/
│   ├── controllers/               # Controladores HTTP
│   │   ├── order_controller.go
│   │   └── admin_controller.go
│   ├── services/                  # Lógica de negocio
│   │   ├── order_service.go
│   │   ├── execution_service.go   # Motor de ejecución concurrente
│   │   ├── validation_service.go
│   │   └── fee_calculator.go
│   ├── repositories/              # Acceso a datos
│   │   ├── order_repository.go
│   │   └── mongodb_repository.go
│   ├── models/                    # Modelos de dominio
│   │   ├── order.go
│   │   ├── execution.go
│   │   └── fee.go
│   ├── dto/                       # Data Transfer Objects
│   │   ├── order_request.go
│   │   ├── order_response.go
│   │   └── execution_result.go
│   ├── clients/                   # Clientes HTTP internos
│   │   ├── users_client.go
│   │   ├── wallet_client.go
│   │   └── market_client.go
│   ├── messaging/                 # RabbitMQ
│   │   ├── publisher.go
│   │   └── events.go
│   ├── concurrent/                # Procesamiento concurrente
│   │   ├── executor.go
│   │   ├── workers.go
│   │   └── orchestrator.go
│   ├── middleware/                # Middlewares
│   │   ├── auth_middleware.go
│   │   ├── logging_middleware.go
│   │   └── rate_limit_middleware.go
│   └── config/                    # Configuración
│       └── config.go
├── pkg/
│   ├── database/                  # Conexión MongoDB
│   │   └── mongodb.go
│   ├── utils/                     # Utilidades
│   │   ├── decimal.go
│   │   ├── validator.go
│   │   └── response.go
│   └── errors/                    # Manejo de errores
│       └── order_errors.go
├── tests/                         # Tests
│   ├── unit/
│   │   └── order_service_test.go
│   ├── integration/
│   │   └── order_flow_test.go
│   └── mocks/
│       ├── repository_mock.go
│       └── client_mock.go
├── docs/                          # Documentación
│   ├── swagger.yaml
│   └── architecture.md
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── .env.example
```

## 💾 Modelo de Datos

### Colección: orders (MongoDB)
```javascript
{
  "_id": ObjectId("507f1f77bcf86cd799439011"),
  "user_id": 123,
  "order_number": "ORD-2024-000001",
  "type": "buy",  // "buy" | "sell"
  "status": "executed",  // "pending" | "processing" | "executed" | "cancelled" | "failed"
  "crypto_symbol": "BTC",
  "crypto_name": "Bitcoin",
  "quantity": NumberDecimal("0.5"),
  "order_type": "market",  // "market" | "limit"
  "limit_price": NumberDecimal("45000.00"),  // Solo para órdenes limit
  "order_price": NumberDecimal("45000.00"),
  "execution_price": NumberDecimal("45050.00"),
  "total_amount": NumberDecimal("22525.00"),
  "fee": NumberDecimal("22.53"),
  "fee_percentage": NumberDecimal("0.001"),
  "created_at": ISODate("2024-01-15T10:30:00Z"),
  "executed_at": ISODate("2024-01-15T10:30:05Z"),
  "updated_at": ISODate("2024-01-15T10:30:05Z"),
  "cancelled_at": null,
  "execution_details": {
    "market_price_at_execution": NumberDecimal("45050.00"),
    "slippage": NumberDecimal("50.00"),
    "slippage_percentage": NumberDecimal("0.0011"),
    "execution_time_ms": 5000,
    "retries": 0,
    "execution_id": "EXEC-2024-000001"
  },
  "validation": {
    "user_verified": true,
    "balance_checked": true,
    "market_hours": true,
    "risk_assessment": "low"
  },
  "metadata": {
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0...",
    "platform": "web",
    "api_version": "v1",
    "session_id": "sess_abc123"
  },
  "audit": {
    "created_by": 123,
    "modified_by": null,
    "modifications": []
  }
}
```

### Índices MongoDB
```javascript
// Índices para optimización de consultas
db.orders.createIndex({ "user_id": 1, "created_at": -1 })
db.orders.createIndex({ "status": 1, "created_at": -1 })
db.orders.createIndex({ "order_number": 1 }, { unique: true })
db.orders.createIndex({ "crypto_symbol": 1, "created_at": -1 })
db.orders.createIndex({ "executed_at": -1 })
db.orders.createIndex({ "user_id": 1, "status": 1 })
```

## 🔌 API Endpoints

### Gestión de Órdenes

#### POST `/api/orders`
Crea una nueva orden con procesamiento concurrente.

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

**Request Body:**
```json
{
  "type": "buy",
  "crypto_symbol": "BTC",
  "quantity": 0.5,
  "order_type": "market"
}
```

**Response (201):**
```json
{
  "success": true,
  "message": "Orden ejecutada exitosamente",
  "data": {
    "order_id": "507f1f77bcf86cd799439011",
    "order_number": "ORD-2024-000001",
    "type": "buy",
    "status": "executed",
    "crypto_symbol": "BTC",
    "quantity": 0.5,
    "execution_price": 45050.00,
    "total_amount": 22525.00,
    "fee": 22.53,
    "executed_at": "2024-01-15T10:30:05Z",
    "execution_details": {
      "market_price": 45050.00,
      "slippage": 50.00,
      "execution_time_ms": 5000
    }
  }
}
```

#### GET `/api/orders/:id`
Obtiene detalles de una orden específica.

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "order_id": "507f1f77bcf86cd799439011",
    "order_number": "ORD-2024-000001",
    "user_id": 123,
    "type": "buy",
    "status": "executed",
    "crypto_symbol": "BTC",
    "crypto_name": "Bitcoin",
    "quantity": 0.5,
    "order_price": 45000.00,
    "execution_price": 45050.00,
    "total_amount": 22525.00,
    "fee": 22.53,
    "created_at": "2024-01-15T10:30:00Z",
    "executed_at": "2024-01-15T10:30:05Z"
  }
}
```

#### PUT `/api/orders/:id`
Actualiza una orden (solo órdenes pendientes).

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Request Body:**
```json
{
  "quantity": 0.75,
  "limit_price": 44500.00
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Orden actualizada exitosamente",
  "data": {
    "order_id": "507f1f77bcf86cd799439011",
    "status": "pending",
    "quantity": 0.75,
    "limit_price": 44500.00,
    "updated_at": "2024-01-15T11:00:00Z"
  }
}
```

#### DELETE `/api/orders/:id`
Cancela una orden pendiente.

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (200):**
```json
{
  "success": true,
  "message": "Orden cancelada exitosamente",
  "data": {
    "order_id": "507f1f77bcf86cd799439011",
    "status": "cancelled",
    "cancelled_at": "2024-01-15T11:30:00Z"
  }
}
```

#### GET `/api/orders/user/:userId`
Lista todas las órdenes de un usuario.

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Query Parameters:**
- `status`: Filtrar por estado (pending/executed/cancelled/failed)
- `crypto`: Filtrar por símbolo de cripto
- `type`: Filtrar por tipo (buy/sell)
- `from`: Fecha desde (YYYY-MM-DD)
- `to`: Fecha hasta (YYYY-MM-DD)
- `page`: Página (default: 1)
- `limit`: Límite por página (default: 20, max: 100)
- `sort`: Campo de ordenamiento (created_at/-created_at)

**Response (200):**
```json
{
  "success": true,
  "data": {
    "orders": [
      {
        "order_id": "507f1f77bcf86cd799439011",
        "order_number": "ORD-2024-000001",
        "type": "buy",
        "status": "executed",
        "crypto_symbol": "BTC",
        "quantity": 0.5,
        "total_amount": 22525.00,
        "created_at": "2024-01-15T10:30:00Z"
      }
    ],
    "pagination": {
      "total": 150,
      "page": 1,
      "limit": 20,
      "total_pages": 8,
      "has_next": true,
      "has_prev": false
    },
    "summary": {
      "total_invested": 100000.00,
      "total_orders": 150,
      "successful_orders": 145,
      "failed_orders": 5
    }
  }
}
```

#### POST `/api/orders/:id/execute`
Ejecuta manualmente una orden pendiente (acción especial).

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (200):**
```json
{
  "success": true,
  "message": "Orden ejecutada manualmente",
  "data": {
    "order_id": "507f1f77bcf86cd799439011",
    "status": "executed",
    "execution_price": 45100.00,
    "executed_at": "2024-01-15T12:00:00Z"
  }
}
```

### Endpoints de Administración

#### GET `/api/orders/admin/all`
Lista todas las órdenes del sistema (solo admin).

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "orders": [...],
    "statistics": {
      "total_orders": 10000,
      "total_volume": 5000000.00,
      "orders_today": 150,
      "volume_today": 75000.00
    }
  }
}
```

#### POST `/api/orders/admin/reprocess/:id`
Reprocesa una orden fallida (solo admin).

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## ⚡ Procesamiento Concurrente

### Arquitectura de Ejecución
```go
// execution_service.go
package services

import (
    "sync"
    "time"
)

type ExecutionService struct {
    walletClient  *clients.WalletClient
    marketClient  *clients.MarketClient
    userClient    *clients.UserClient
    feeCalculator *FeeCalculator
}

type ExecutionResult struct {
    UserValidation   *ValidationResult
    BalanceCheck     *BalanceResult
    MarketPrice      *PriceResult
    FeeCalculation   *FeeResult
    ExecutionTime    time.Duration
    Success          bool
    Error            error
}

func (s *ExecutionService) ExecuteOrderConcurrent(order *models.Order) (*ExecutionResult, error) {
    start := time.Now()
    
    var wg sync.WaitGroup
    resultChan := make(chan interface{}, 4)
    errorChan := make(chan error, 4)
    
    // 1. Validar usuario (Goroutine 1)
    wg.Add(1)
    go func() {
        defer wg.Done()
        user, err := s.userClient.VerifyUser(order.UserID)
        if err != nil {
            errorChan <- err
            return
        }
        resultChan <- &ValidationResult{
            UserID:    user.ID,
            IsActive:  user.IsActive,
            Role:      user.Role,
            Validated: true,
        }
    }()
    
    // 2. Verificar balance (Goroutine 2)
    wg.Add(1)
    go func() {
        defer wg.Done()
        // Calcular monto necesario
        estimatedAmount := order.Quantity * order.EstimatedPrice
        
        balance, err := s.walletClient.CheckBalance(order.UserID, estimatedAmount)
        if err != nil {
            errorChan <- err
            return
        }
        
        if balance.Available < estimatedAmount {
            errorChan <- ErrInsufficientFunds
            return
        }
        
        resultChan <- &BalanceResult{
            Available:    balance.Available,
            Required:     estimatedAmount,
            HasSufficient: true,
        }
    }()
    
    // 3. Obtener precio actual del mercado (Goroutine 3)
    wg.Add(1)
    go func() {
        defer wg.Done()
        price, err := s.marketClient.GetCurrentPrice(order.CryptoSymbol)
        if err != nil {
            errorChan <- err
            return
        }
        
        // Simular slippage
        slippage := s.calculateSlippage(order.Type, order.Quantity)
        finalPrice := price.Current * (1 + slippage)
        
        resultChan <- &PriceResult{
            MarketPrice:   price.Current,
            ExecutionPrice: finalPrice,
            Slippage:      slippage,
            Timestamp:     time.Now(),
        }
    }()
    
    // 4. Calcular fees y comisiones (Goroutine 4)
    wg.Add(1)
    go func() {
        defer wg.Done()
        
        // Simular latencia de cálculo
        time.Sleep(100 * time.Millisecond)
        
        fee := s.feeCalculator.Calculate(order)
        resultChan <- &FeeResult{
            BaseFee:       fee.Base,
            PercentageFee: fee.Percentage,
            TotalFee:      fee.Total,
        }
    }()
    
    // Esperar a que todas las goroutines terminen
    wg.Wait()
    close(resultChan)
    close(errorChan)
    
    // Procesar errores
    select {
    case err := <-errorChan:
        if err != nil {
            return nil, err
        }
    default:
    }
    
    // Consolidar resultados
    result := &ExecutionResult{
        ExecutionTime: time.Since(start),
        Success:       true,
    }
    
    for res := range resultChan {
        switch v := res.(type) {
        case *ValidationResult:
            result.UserValidation = v
        case *BalanceResult:
            result.BalanceCheck = v
        case *PriceResult:
            result.MarketPrice = v
        case *FeeResult:
            result.FeeCalculation = v
        }
    }
    
    return result, nil
}

// Función auxiliar para calcular slippage
func (s *ExecutionService) calculateSlippage(orderType string, quantity float64) float64 {
    baseSlippage := 0.001 // 0.1% base
    
    // Mayor slippage para órdenes grandes
    if quantity > 1.0 {
        baseSlippage *= (1 + quantity*0.1)
    }
    
    // Slippage adicional para órdenes de venta
    if orderType == "sell" {
        baseSlippage *= 1.2
    }
    
    // Añadir factor aleatorio para simular condiciones de mercado
    randomFactor := (rand.Float64() - 0.5) * 0.001
    
    return baseSlippage + randomFactor
}
```

### Canal de Comunicación
```go
// orchestrator.go
package concurrent

type OrderOrchestrator struct {
    workers      int
    orderQueue   chan *models.Order
    resultQueue  chan *ExecutionResult
    errorQueue   chan error
    executor     *ExecutionService
}

func NewOrderOrchestrator(workers int, executor *ExecutionService) *OrderOrchestrator {
    return &OrderOrchestrator{
        workers:     workers,
        orderQueue:  make(chan *models.Order, 100),
        resultQueue: make(chan *ExecutionResult, 100),
        errorQueue:  make(chan error, 100),
        executor:    executor,
    }
}

func (o *OrderOrchestrator) Start() {
    for i := 0; i < o.workers; i++ {
        go o.worker(i)
    }
}

func (o *OrderOrchestrator) worker(id int) {
    for order := range o.orderQueue {
        result, err := o.executor.ExecuteOrderConcurrent(order)
        if err != nil {
            o.errorQueue <- err
            continue
        }
        o.resultQueue <- result
    }
}
```

## 📨 Mensajería con RabbitMQ

### Publisher de Eventos
```go
// publisher.go
package messaging

import (
    "encoding/json"
    "github.com/streadway/amqp"
)

type OrderEventPublisher struct {
    conn    *amqp.Connection
    channel *amqp.Channel
}

type OrderEvent struct {
    EventType   string      `json:"event_type"`
    OrderID     string      `json:"order_id"`
    UserID      int         `json:"user_id"`
    Data        interface{} `json:"data"`
    Timestamp   time.Time   `json:"timestamp"`
}

func (p *OrderEventPublisher) PublishOrderCreated(order *models.Order) error {
    event := OrderEvent{
        EventType: "order.created",
        OrderID:   order.ID,
        UserID:    order.UserID,
        Data:      order,
        Timestamp: time.Now(),
    }
    
    return p.publish("orders.events", event)
}

func (p *OrderEventPublisher) PublishOrderExecuted(order *models.Order) error {
    event := OrderEvent{
        EventType: "order.executed",
        OrderID:   order.ID,
        UserID:    order.UserID,
        Data: map[string]interface{}{
            "crypto_symbol":    order.CryptoSymbol,
            "quantity":        order.Quantity,
            "execution_price": order.ExecutionPrice,
            "total_amount":    order.TotalAmount,
        },
        Timestamp: time.Now(),
    }
    
    return p.publish("orders.events", event)
}

func (p *OrderEventPublisher) publish(routingKey string, event OrderEvent) error {
    body, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    return p.channel.Publish(
        "cryptosim", // exchange
        routingKey,  // routing key
        false,       // mandatory
        false,       // immediate
        amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
            Timestamp:   time.Now(),
        },
    )
}
```

## 🧪 Testing

### Test del Servicio de Órdenes
```go
// order_service_test.go
package services

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestOrderService_CreateOrder_Success(t *testing.T) {
    // Arrange
    mockRepo := new(mocks.MockOrderRepository)
    mockWalletClient := new(mocks.MockWalletClient)
    mockMarketClient := new(mocks.MockMarketClient)
    mockUserClient := new(mocks.MockUserClient)
    mockPublisher := new(mocks.MockPublisher)
    
    service := NewOrderService(
        mockRepo,
        mockWalletClient,
        mockMarketClient,
        mockUserClient,
        mockPublisher,
    )
    
    order := &models.Order{
        UserID:       123,
        Type:         "buy",
        CryptoSymbol: "BTC",
        Quantity:     0.5,
        OrderType:    "market",
    }
    
    // Mock responses
    mockUserClient.On("VerifyUser", 123).Return(&models.User{
        ID:       123,
        IsActive: true,
        Role:     "normal",
    }, nil)
    
    mockWalletClient.On("CheckBalance", 123, mock.Anything).Return(&models.Balance{
        Available: 50000.00,
    }, nil)
    
    mockMarketClient.On("GetCurrentPrice", "BTC").Return(&models.Price{
        Symbol:  "BTC",
        Current: 45000.00,
    }, nil)
    
    mockWalletClient.On("LockFunds", 123, mock.Anything).Return(nil)
    mockRepo.On("Create", mock.AnythingOfType("*models.Order")).Return(nil)
    mockPublisher.On("PublishOrderCreated", mock.Anything).Return(nil)
    mockPublisher.On("PublishOrderExecuted", mock.Anything).Return(nil)
    
    // Act
    result, err := service.CreateOrder(order)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "executed", result.Status)
    assert.True(t, result.ExecutionPrice > 0)
    assert.True(t, result.TotalAmount > 0)
    assert.True(t, result.Fee > 0)
    
    // Verify all mocks were called
    mockRepo.AssertExpectations(t)
    mockWalletClient.AssertExpectations(t)
    mockMarketClient.AssertExpectations(t)
    mockUserClient.AssertExpectations(t)
    mockPublisher.AssertExpectations(t)
}

func TestOrderService_CreateOrder_InsufficientBalance(t *testing.T) {
    // Similar setup but with insufficient balance
    mockWalletClient.On("CheckBalance", 123, mock.Anything).Return(&models.Balance{
        Available: 100.00, // Insufficient
    }, nil)
    
    // Act
    result, err := service.CreateOrder(order)
    
    // Assert
    assert.Error(t, err)
    assert.Nil(t, result)
    assert.Equal(t, ErrInsufficientFunds, err)
}

func TestExecutionService_ConcurrentExecution(t *testing.T) {
    // Test concurrent execution
    service := NewExecutionService(/* dependencies */)
    
    order := &models.Order{
        UserID:         123,
        Type:          "buy",
        CryptoSymbol:  "ETH",
        Quantity:      2.0,
        EstimatedPrice: 3000.00,
    }
    
    // Act
    start := time.Now()
    result, err := service.ExecuteOrderConcurrent(order)
    duration := time.Since(start)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.True(t, result.Success)
    assert.NotNil(t, result.UserValidation)
    assert.NotNil(t, result.BalanceCheck)
    assert.NotNil(t, result.MarketPrice)
    assert.NotNil(t, result.FeeCalculation)
    
    // Verify concurrent execution (should be faster than sequential)
    assert.Less(t, duration, 500*time.Millisecond)
}

func TestOrderService_CancelOrder(t *testing.T) {
    // Test order cancellation
    mockRepo := new(mocks.MockOrderRepository)
    service := NewOrderService(mockRepo, /* other deps */)
    
    existingOrder := &models.Order{
        ID:     "507f1f77bcf86cd799439011",
        UserID: 123,
        Status: "pending",
    }
    
    mockRepo.On("GetByID", "507f1f77bcf86cd799439011").Return(existingOrder, nil)
    mockRepo.On("Update", mock.AnythingOfType("*models.Order")).Return(nil)
    
    // Act
    err := service.CancelOrder("507f1f77bcf86cd799439011", 123)
    
    // Assert
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

### Benchmark Tests
```go
// benchmark_test.go
package services

import (
    "testing"
)

func BenchmarkExecuteOrderConcurrent(b *testing.B) {
    service := setupTestService()
    order := createTestOrder()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = service.ExecuteOrderConcurrent(order)
    }
}

func BenchmarkExecuteOrderSequential(b *testing.B) {
    service := setupTestService()
    order := createTestOrder()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = service.ExecuteOrderSequential(order)
    }
}
```

## 🚀 Instalación y Configuración

### Variables de Entorno
```env
# Server
SERVER_PORT=8002
SERVER_ENV=development

# MongoDB
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=orders_db
MONGO_TIMEOUT=10s

# RabbitMQ
RABBITMQ_URL=amqp://admin:admin@localhost:5672/
RABBITMQ_EXCHANGE=cryptosim
RABBITMQ_QUEUE_ORDERS=orders.events

# Internal Services
USERS_API_URL=http://localhost:8001
WALLET_API_URL=http://localhost:8006
MARKET_API_URL=http://localhost:8004
INTERNAL_API_KEY=internal-secret-key

# JWT
JWT_SECRET=your-super-secret-key

# Performance
MAX_WORKERS=10
ORDER_QUEUE_SIZE=100
EXECUTION_TIMEOUT=30s

# Fee Configuration
BASE_FEE_PERCENTAGE=0.001
MAKER_FEE=0.0008
TAKER_FEE=0.0012
```

### Desarrollo Local
```bash
# Instalar dependencias
go mod download

# Ejecutar MongoDB local
docker run -d -p 27017:27017 --name mongodb mongo:6.0

# Ejecutar RabbitMQ local
docker run -d -p 5672:5672 -p 15672:15672 --name rabbitmq rabbitmq:3-management

# Ejecutar el servicio
go run cmd/main.go

# Ejecutar tests
go test ./... -v -cover

# Ejecutar benchmarks
go test -bench=. -benchmem ./...
```

### Docker
```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.