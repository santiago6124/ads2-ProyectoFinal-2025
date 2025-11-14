# Flujo Técnico: Compra y Venta de Criptomonedas

## Descripción General

Este documento describe el flujo completo de creación, validación y ejecución de órdenes de compra/venta de criptomonedas en CryptoSim.

## Arquitectura Involucrada

```
┌──────────┐    ┌───────────┐    ┌────────────┐    ┌──────────────┐    ┌──────────┐
│  Cliente │───>│Orders API │───>│ Users API  │    │ Market Data  │    │Portfolio │
│          │<───│   :8002   │<───│   :8001    │    │     :8004    │    │   :8005  │
└──────────┘    └─────┬─────┘    └────────────┘    └──────────────┘    └────┬─────┘
                      │                                                        │
                      ▼                                                        ▼
                ┌──────────┐          ┌──────────┐                      ┌──────────┐
                │ MongoDB  │          │  Redis   │                      │ MongoDB  │
                │ (Orders) │          │ (Cache)  │                      │(Portfolio│
                └──────────┘          └──────────┘                      └──────────┘
                      │                                                        │
                      └────────────────>[ RabbitMQ ]<────────────────────────┘
                                       orders.events
```

---

## 1. FLUJO DE COMPRA (BUY)

### Fase 1: Creación de la Orden

#### Endpoint
```
POST /api/v1/orders
Authorization: Bearer {JWT_TOKEN}
Content-Type: application/json
```

#### Request Body
```json
{
  "type": "buy",
  "crypto_symbol": "BTC",
  "quantity": 0.001,
  "order_kind": "market"
}
```

#### Paso 1.1: Cliente envía request
```bash
curl -X POST http://localhost:8002/api/v1/orders \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "type": "buy",
    "crypto_symbol": "BTC",
    "quantity": 0.001,
    "order_kind": "market"
  }'
```

#### Paso 1.2: Middleware JWT extrae user_id
```go
// AuthMiddleware en orders-api/middleware/auth.go
token := extractTokenFromHeader(c.GetHeader("Authorization"))
claims := parseJWT(token)
c.Set("user_id", claims["user_id"])  // Ej: 123
c.Next()
```

#### Paso 1.3: Handler recibe request
```go
// CreateOrder en orders-api/handlers/order_handler.go
func (h *OrderHandler) CreateOrder(c *gin.Context) {
    userID := c.GetInt64("user_id")  // 123 desde JWT

    var req CreateOrderRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }

    // Validaciones básicas
    if req.Quantity <= 0 {
        c.JSON(400, gin.H{"error": "Quantity must be positive"})
        return
    }

    if req.Type != "buy" && req.Type != "sell" {
        c.JSON(400, gin.H{"error": "Type must be 'buy' or 'sell'"})
        return
    }

    // Pasar al service
    order, err := h.orderService.CreateOrder(c.Request.Context(), userID, req)
    ...
}
```

#### Paso 1.4: Service Layer - Validaciones

**1.4.1 Verificar usuario existe (Users API)**
```go
// orders-api/services/order_service.go
userResp, err := http.Get(fmt.Sprintf("http://users-api:8001/api/users/%d/verify", userID))
if err != nil || userResp.StatusCode != 200 {
    return nil, errors.New("User not found or invalid")
}

var userData UserResponse
json.NewDecoder(userResp.Body).Decode(&userData)
```

**Request a Users API:**
```http
GET http://users-api:8001/api/users/123/verify
Authorization: Internal-Service
```

**Response de Users API:**
```json
{
  "status": "success",
  "data": {
    "user": {
      "id": 123,
      "username": "trader123",
      "email": "trader@cryptosim.com",
      "initial_balance": 100000.00,
      "is_active": true
    }
  }
}
```

**1.4.2 Obtener precio actual (Market Data API)**
```go
priceResp, err := http.Get(fmt.Sprintf("http://market-data-api:8004/api/v1/prices/%s", cryptoSymbol))
if err != nil || priceResp.StatusCode != 200 {
    return nil, errors.New("Failed to fetch crypto price")
}

var priceData PriceResponse
json.NewDecoder(priceResp.Body).Decode(&priceData)
currentPrice := priceData.Data.Price  // Ej: 50000.00 USD
```

**Request a Market Data API:**
```http
GET http://market-data-api:8004/api/v1/prices/BTC
```

**Response de Market Data API:**
```json
{
  "status": "success",
  "data": {
    "symbol": "BTC",
    "name": "Bitcoin",
    "price": 50000.00,
    "price_change_24h": 2.5,
    "volume_24h": 28500000000,
    "market_cap": 980000000000,
    "last_updated": "2025-11-14T10:30:00Z"
  }
}
```

**1.4.3 Calcular totales**
```go
quantity := 0.001              // Del request
price := 50000.00              // De Market Data API
totalAmount := quantity * price  // 0.001 * 50000 = 50.00 USD

feeRate := 0.001               // 0.1% de comisión
fee := totalAmount * feeRate   // 50.00 * 0.001 = 0.05 USD

totalRequired := totalAmount + fee  // 50.05 USD
```

**1.4.4 Verificar balance suficiente**
```go
userBalance := userData.User.InitialBalance  // 100000.00 USD

if totalRequired > userBalance {
    return nil, errors.New("Insufficient balance")
}
```

**1.4.5 Crear orden en MongoDB**
```go
order := &Order{
    UserID:       123,
    Type:         "buy",
    Status:       "pending",
    CryptoSymbol: "BTC",
    Quantity:     decimal.NewFromFloat(0.001),
    Price:        decimal.NewFromFloat(50000.00),
    TotalAmount:  decimal.NewFromFloat(50.00),
    Fee:          decimal.NewFromFloat(0.05),
    OrderKind:    "market",
    CreatedAt:    time.Now(),
    UpdatedAt:    time.Now(),
}

result, err := ordersCollection.InsertOne(ctx, order)
order.ID = result.InsertedID.(primitive.ObjectID)
```

**Documento MongoDB creado:**
```json
{
  "_id": ObjectId("673b5f8a9e1234567890abcd"),
  "user_id": 123,
  "type": "buy",
  "status": "pending",
  "crypto_symbol": "BTC",
  "quantity": NumberDecimal("0.001"),
  "price": NumberDecimal("50000.00"),
  "total_amount": NumberDecimal("50.00"),
  "fee": NumberDecimal("0.05"),
  "order_kind": "market",
  "created_at": ISODate("2025-11-14T10:30:00Z"),
  "updated_at": ISODate("2025-11-14T10:30:00Z"),
  "executed_at": null,
  "error_message": null
}
```

#### Paso 1.5: Response de creación
```json
{
  "status": "success",
  "message": "Order created successfully",
  "data": {
    "order": {
      "id": "673b5f8a9e1234567890abcd",
      "user_id": 123,
      "type": "buy",
      "status": "pending",
      "crypto_symbol": "BTC",
      "quantity": "0.001",
      "price": "50000.00",
      "total_amount": "50.00",
      "fee": "0.05",
      "order_kind": "market",
      "created_at": "2025-11-14T10:30:00Z"
    }
  }
}
```

**Status Code**: `201 Created`

---

### Fase 2: Ejecución de la Orden

#### Endpoint
```
POST /api/v1/orders/:id/execute
Authorization: Bearer {JWT_TOKEN}
```

#### Paso 2.1: Cliente ejecuta orden
```bash
curl -X POST http://localhost:8002/api/v1/orders/673b5f8a9e1234567890abcd/execute \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

#### Paso 2.2: Validaciones previas

**2.2.1 Buscar orden en MongoDB**
```go
var order Order
filter := bson.M{"_id": objectID, "user_id": userID}
err := ordersCollection.FindOne(ctx, filter).Decode(&order)
if err != nil {
    return nil, errors.New("Order not found or unauthorized")
}
```

**2.2.2 Verificar propietario**
```go
if order.UserID != userID {
    return nil, errors.New("Unauthorized: not order owner")
}
```

**2.2.3 Verificar estado**
```go
if order.Status != "pending" {
    return nil, errors.New("Order cannot be executed (already processed)")
}
```

#### Paso 2.3: Actualizar balance en Users API

**Request:**
```http
PUT http://users-api:8001/api/users/123/balance
Content-Type: application/json
Authorization: Internal-Service

{
  "amount": -50.05,
  "transaction_type": "buy",
  "order_id": "673b5f8a9e1234567890abcd"
}
```

**Proceso en Users API:**
```go
// users-api/handlers/user_handler.go
func (h *UserHandler) UpdateBalance(c *gin.Context) {
    userID := c.Param("id")
    var req UpdateBalanceRequest
    c.ShouldBindJSON(&req)

    // Iniciar transacción MySQL
    tx := db.Begin()

    // 1. Verificar balance actual
    var user User
    tx.Where("id = ?", userID).First(&user)

    newBalance := user.InitialBalance + req.Amount  // 100000 - 50.05 = 99949.95

    if newBalance < 0 {
        tx.Rollback()
        return errors.New("Insufficient balance")
    }

    // 2. Verificar idempotencia (evitar double-spend)
    var existingTx BalanceTransaction
    result := tx.Where("order_id = ?", req.OrderID).First(&existingTx)
    if result.RowsAffected > 0 {
        tx.Rollback()
        return errors.New("Transaction already processed")
    }

    // 3. Actualizar balance
    tx.Model(&user).Update("initial_balance", newBalance)

    // 4. Registrar transacción
    balanceTx := BalanceTransaction{
        OrderID:         req.OrderID,
        UserID:          user.ID,
        Amount:          req.Amount,
        TransactionType: req.TransactionType,
        PreviousBalance: user.InitialBalance,
        NewBalance:      newBalance,
        ProcessedAt:     time.Now(),
    }
    tx.Create(&balanceTx)

    // 5. Commit
    tx.Commit()
}
```

**Tablas MySQL actualizadas:**

```sql
-- users table
UPDATE users
SET initial_balance = 99949.95, updated_at = NOW()
WHERE id = 123;

-- balance_transactions table
INSERT INTO balance_transactions (
  order_id,
  user_id,
  amount,
  transaction_type,
  previous_balance,
  new_balance,
  processed_at,
  created_at
) VALUES (
  '673b5f8a9e1234567890abcd',
  123,
  -50.05,
  'buy',
  100000.00,
  99949.95,
  NOW(),
  NOW()
);
```

#### Paso 2.4: Actualizar orden en MongoDB

```go
update := bson.M{
    "$set": bson.M{
        "status":      "executed",
        "executed_at": time.Now(),
        "updated_at":  time.Now(),
    },
}

filter := bson.M{"_id": order.ID}
_, err := ordersCollection.UpdateOne(ctx, filter, update)
```

**Documento MongoDB actualizado:**
```json
{
  "_id": ObjectId("673b5f8a9e1234567890abcd"),
  "user_id": 123,
  "type": "buy",
  "status": "executed",  // <-- Cambió de "pending"
  "crypto_symbol": "BTC",
  "quantity": NumberDecimal("0.001"),
  "price": NumberDecimal("50000.00"),
  "total_amount": NumberDecimal("50.00"),
  "fee": NumberDecimal("0.05"),
  "order_kind": "market",
  "created_at": ISODate("2025-11-14T10:30:00Z"),
  "updated_at": ISODate("2025-11-14T10:30:05Z"),
  "executed_at": ISODate("2025-11-14T10:30:05Z"),  // <-- Nuevo
  "error_message": null
}
```

#### Paso 2.5: Publicar evento en RabbitMQ

```go
// orders-api/messaging/publisher.go
event := OrderEvent{
    EventType:    "executed",
    OrderID:      order.ID.Hex(),
    UserID:       order.UserID,
    Type:         order.Type,
    CryptoSymbol: order.CryptoSymbol,
    Quantity:     order.Quantity.String(),
    Price:        order.Price.String(),
    TotalAmount:  order.TotalAmount.String(),
    Fee:          order.Fee.String(),
    Timestamp:    time.Now(),
}

eventJSON, _ := json.Marshal(event)

err := rabbitChannel.Publish(
    "orders.events",      // Exchange
    "orders.executed",    // Routing key
    false,                // Mandatory
    false,                // Immediate
    amqp.Publishing{
        ContentType:  "application/json",
        Body:         eventJSON,
        DeliveryMode: amqp.Persistent,
        Timestamp:    time.Now(),
    },
)
```

**Mensaje RabbitMQ publicado:**
```json
{
  "event_type": "executed",
  "order_id": "673b5f8a9e1234567890abcd",
  "user_id": 123,
  "type": "buy",
  "crypto_symbol": "BTC",
  "quantity": "0.001",
  "price": "50000.00",
  "total_amount": "50.00",
  "fee": "0.05",
  "timestamp": "2025-11-14T10:30:05Z"
}
```

#### Paso 2.6: Response de ejecución

```json
{
  "status": "success",
  "message": "Order executed successfully",
  "data": {
    "order": {
      "id": "673b5f8a9e1234567890abcd",
      "user_id": 123,
      "type": "buy",
      "status": "executed",
      "crypto_symbol": "BTC",
      "quantity": "0.001",
      "price": "50000.00",
      "total_amount": "50.00",
      "fee": "0.05",
      "order_kind": "market",
      "created_at": "2025-11-14T10:30:00Z",
      "executed_at": "2025-11-14T10:30:05Z"
    }
  }
}
```

**Status Code**: `200 OK`

---

### Fase 3: Procesamiento Asíncrono (Portfolio API)

#### Consumer de RabbitMQ en Portfolio API

```go
// portfolio-api/messaging/consumer.go
msgs, _ := rabbitChannel.Consume(
    "portfolio.updates",  // Queue
    "",                   // Consumer tag
    false,                // Auto-ack
    false,                // Exclusive
    false,                // No-local
    false,                // No-wait
    nil,                  // Args
)

for msg := range msgs {
    var event OrderEvent
    json.Unmarshal(msg.Body, &event)

    if event.EventType == "executed" {
        handleOrderExecuted(event)
    }

    msg.Ack(false)
}
```

#### Paso 3.1: Buscar o crear portfolio

```go
var portfolio Portfolio
filter := bson.M{"user_id": event.UserID}
err := portfolioCollection.FindOne(ctx, filter).Decode(&portfolio)

if err == mongo.ErrNoDocuments {
    // Crear portfolio nuevo
    portfolio = Portfolio{
        UserID:        event.UserID,
        TotalValue:    decimal.Zero,
        TotalInvested: decimal.Zero,
        ProfitLoss:    decimal.Zero,
        Currency:      "USD",
        Holdings:      []Holding{},
        CreatedAt:     time.Now(),
    }
    portfolioCollection.InsertOne(ctx, &portfolio)
}
```

#### Paso 3.2: Obtener precio actual

```go
priceResp, _ := http.Get(fmt.Sprintf("http://market-data-api:8004/api/v1/prices/%s", event.CryptoSymbol))
var priceData PriceResponse
json.NewDecoder(priceResp.Body).Decode(&priceData)
currentPrice := priceData.Data.Price  // Ej: 51000.00 (subió!)
```

#### Paso 3.3: Actualizar o crear holding

**Si es primera compra de BTC:**
```go
holding := Holding{
    Symbol:            "BTC",
    Quantity:          decimal.NewFromString("0.001"),
    AverageBuyPrice:   decimal.NewFromString("50000.00"),
    CurrentPrice:      decimal.NewFromString("51000.00"),
    CurrentValue:      decimal.NewFromString("51.00"),    // 0.001 * 51000
    ProfitLoss:        decimal.NewFromString("1.00"),     // 51 - 50
    ProfitLossPercent: 2.0,                               // ((51-50)/50)*100
    PercentOfPortfolio: 100.0,                            // Por ahora solo BTC
    FirstPurchaseDate: time.Now(),
    LastPurchaseDate:  time.Now(),
    TransactionsCount: 1,
    CostBasis: []CostBasisEntry{
        {
            Quantity:   decimal.NewFromString("0.001"),
            Price:      decimal.NewFromString("50000.00"),
            Date:       time.Now(),
            TotalCost:  decimal.NewFromString("50.00"),
        },
    },
}

portfolio.Holdings = append(portfolio.Holdings, holding)
```

**Si ya tenía BTC (compra adicional):**
```go
// Actualizar holding existente
for i, h := range portfolio.Holdings {
    if h.Symbol == "BTC" {
        oldQuantity := h.Quantity
        newQuantity := oldQuantity.Add(decimal.NewFromString("0.001"))

        // Recalcular precio promedio ponderado
        oldCost := oldQuantity.Mul(h.AverageBuyPrice)
        newCost := decimal.NewFromString("50.00")
        totalCost := oldCost.Add(newCost)
        newAvgPrice := totalCost.Div(newQuantity)

        portfolio.Holdings[i].Quantity = newQuantity
        portfolio.Holdings[i].AverageBuyPrice = newAvgPrice
        portfolio.Holdings[i].TransactionsCount++
        portfolio.Holdings[i].LastPurchaseDate = time.Now()

        // Agregar cost basis entry
        portfolio.Holdings[i].CostBasis = append(portfolio.Holdings[i].CostBasis, CostBasisEntry{...})
    }
}
```

#### Paso 3.4: Recalcular métricas del portfolio

```go
// Totales
totalValue := decimal.Zero
totalInvested := decimal.Zero

for _, h := range portfolio.Holdings {
    h.CurrentValue = h.Quantity.Mul(h.CurrentPrice)
    totalValue = totalValue.Add(h.CurrentValue)
    totalInvested = totalInvested.Add(h.Quantity.Mul(h.AverageBuyPrice))
}

portfolio.TotalValue = totalValue          // 51.00
portfolio.TotalInvested = totalInvested    // 50.00
portfolio.ProfitLoss = totalValue.Sub(totalInvested)  // 1.00
portfolio.ProfitLossPercentage = (portfolio.ProfitLoss.Div(totalInvested)).Mul(decimal.NewFromInt(100))  // 2%

// Performance metrics
portfolio.Performance = calculatePerformanceMetrics(portfolio)

// Risk metrics (Sharpe, Sortino, etc.)
portfolio.RiskMetrics = calculateRiskMetrics(portfolio)

// Diversification
portfolio.Diversification = calculateDiversification(portfolio)

portfolio.UpdatedAt = time.Now()
```

#### Paso 3.5: Guardar portfolio actualizado

```go
filter := bson.M{"user_id": portfolio.UserID}
update := bson.M{"$set": portfolio}
_, err := portfolioCollection.UpdateOne(ctx, filter, update)
```

**Documento MongoDB Portfolio:**
```json
{
  "_id": ObjectId("673b5f8a9e1234567890abce"),
  "user_id": 123,
  "total_value": NumberDecimal("51.00"),
  "total_invested": NumberDecimal("50.00"),
  "profit_loss": NumberDecimal("1.00"),
  "profit_loss_percentage": 2.0,
  "currency": "USD",
  "holdings": [
    {
      "symbol": "BTC",
      "quantity": NumberDecimal("0.001"),
      "average_buy_price": NumberDecimal("50000.00"),
      "current_price": NumberDecimal("51000.00"),
      "current_value": NumberDecimal("51.00"),
      "profit_loss": NumberDecimal("1.00"),
      "profit_loss_percentage": 2.0,
      "percentage_of_portfolio": 100.0,
      "first_purchase_date": ISODate("2025-11-14T10:30:05Z"),
      "last_purchase_date": ISODate("2025-11-14T10:30:05Z"),
      "transactions_count": 1,
      "cost_basis": [
        {
          "quantity": NumberDecimal("0.001"),
          "price": NumberDecimal("50000.00"),
          "date": ISODate("2025-11-14T10:30:05Z"),
          "total_cost": NumberDecimal("50.00")
        }
      ]
    }
  ],
  "performance": {
    "daily_change": NumberDecimal("1.00"),
    "daily_change_percentage": 2.0,
    "roi": 2.0,
    "all_time_high": NumberDecimal("51.00"),
    "all_time_low": NumberDecimal("50.00")
  },
  "risk_metrics": {
    "volatility_24h": 0.0,
    "sharpe_ratio": 0.0,
    "max_drawdown": 0.0
  },
  "diversification": {
    "herfindahl_index": 1.0,
    "effective_holdings": 1,
    "largest_position_percentage": 100.0
  },
  "metadata": {
    "last_calculated": ISODate("2025-11-14T10:30:05Z"),
    "last_order_processed": "673b5f8a9e1234567890abcd",
    "needs_recalculation": false
  },
  "created_at": ISODate("2025-11-14T10:30:05Z"),
  "updated_at": ISODate("2025-11-14T10:30:05Z")
}
```

---

## 2. FLUJO DE VENTA (SELL)

### Diferencias clave con el flujo de compra

#### Request de venta
```json
{
  "type": "sell",
  "crypto_symbol": "BTC",
  "quantity": 0.0005,
  "order_kind": "market"
}
```

#### Validaciones adicionales en venta

**Verificar cantidad disponible en portfolio:**
```go
// orders-api/services/order_service.go
portfolioResp, err := http.Get(fmt.Sprintf("http://portfolio-api:8005/api/portfolio/%d/holdings", userID))

var holdings []Holding
json.NewDecoder(portfolioResp.Body).Decode(&holdings)

btcHolding := findHolding(holdings, "BTC")
if btcHolding == nil {
    return nil, errors.New("You don't own this cryptocurrency")
}

if btcHolding.Quantity.LessThan(requestedQuantity) {
    return nil, errors.New(fmt.Sprintf("Insufficient holdings. You have %s BTC", btcHolding.Quantity))
}
```

#### Actualización de balance (suma en lugar de resta)

**Request a Users API:**
```http
PUT http://users-api:8001/api/users/123/balance
Content-Type: application/json

{
  "amount": 25.475,  // POSITIVO (precio venta - fee)
  "transaction_type": "sell",
  "order_id": "673b5f8a9e1234567890abcf"
}
```

**Cálculo:**
```go
sellPrice := 51000.00          // Precio actual
quantity := 0.0005             // Cantidad a vender
totalAmount := 25.50           // 0.0005 * 51000
fee := 0.025                   // 0.1% de 25.50
netAmount := 25.475            // 25.50 - 0.025

// Balance actualizado
newBalance := 99949.95 + 25.475 = 99975.425 USD
```

#### Actualización de portfolio (resta holdings)

```go
// portfolio-api/services/portfolio_service.go
for i, h := range portfolio.Holdings {
    if h.Symbol == "BTC" {
        newQuantity := h.Quantity.Sub(decimal.NewFromString("0.0005"))

        if newQuantity.IsZero() {
            // Eliminar holding si vendió todo
            portfolio.Holdings = append(portfolio.Holdings[:i], portfolio.Holdings[i+1:]...)
        } else {
            // Actualizar cantidad
            portfolio.Holdings[i].Quantity = newQuantity
            portfolio.Holdings[i].TransactionsCount++

            // Actualizar cost basis (FIFO - First In First Out)
            remainingToSell := decimal.NewFromString("0.0005")
            for j := 0; j < len(portfolio.Holdings[i].CostBasis) && remainingToSell.IsPositive(); j++ {
                if portfolio.Holdings[i].CostBasis[j].Quantity.GreaterThan(remainingToSell) {
                    portfolio.Holdings[i].CostBasis[j].Quantity = portfolio.Holdings[i].CostBasis[j].Quantity.Sub(remainingToSell)
                    remainingToSell = decimal.Zero
                } else {
                    remainingToSell = remainingToSell.Sub(portfolio.Holdings[i].CostBasis[j].Quantity)
                    // Eliminar entrada
                    portfolio.Holdings[i].CostBasis = append(
                        portfolio.Holdings[i].CostBasis[:j],
                        portfolio.Holdings[i].CostBasis[j+1:]...,
                    )
                    j--
                }
            }
        }
    }
}
```

---

## 3. DIAGRAMA DE SECUENCIA COMPLETO

```
Cliente    Orders API   Users API   Market Data   MongoDB   RabbitMQ   Portfolio API
  │            │            │             │          │          │            │
  │─POST buy───>│            │             │          │          │            │
  │            │─JWT verify │             │          │          │            │
  │            │─GET user──>│             │          │          │            │
  │            │<─user data─│             │          │          │            │
  │            │─GET price─────────────>│          │          │            │
  │            │<─BTC: 50k──────────────│          │          │            │
  │            │─Calculate: 50.05 USD    │          │          │            │
  │            │─INSERT order──────────────────────>│          │            │
  │<─201 order─│<─order ID───────────────────────────│          │            │
  │            │            │             │          │          │            │
  │─POST exec──>│            │             │          │          │            │
  │            │─GET order──────────────────────────>│          │            │
  │            │─Validate owner/status   │          │          │            │
  │            │─PUT balance>│             │          │          │            │
  │            │ (TX begin)  │             │          │          │            │
  │            │<─TX commit──│             │          │          │            │
  │            │─UPDATE status─────────────────────>│          │            │
  │            │─PUBLISH event─────────────────────────────────>│            │
  │<─200 OK────│            │             │          │          │            │
  │            │            │             │          │          │            │
  │            │            │             │          │          │─CONSUME────│
  │            │            │             │          │          │            │
  │            │            │             │          │          │<─GET price─┤
  │            │            │             │          │          │            │
  │            │            │             │          │          │─UPDATE─────>│
  │            │            │             │          │          │   portfolio│
```

---

## 4. ESTADOS DE UNA ORDEN

```
pending ────> executed
   │
   └────────> cancelled
   │
   └────────> failed
```

- **pending**: Creada pero no ejecutada
- **executed**: Ejecutada exitosamente (balance actualizado)
- **cancelled**: Cancelada por usuario (endpoint `/cancel`)
- **failed**: Falló por error (ej: balance insuficiente al ejecutar)

---

## 5. VALIDACIONES COMPLETAS

### Validaciones en creación (POST /orders)
- ✅ Usuario autenticado (JWT válido)
- ✅ Usuario existe y está activo
- ✅ Tipo válido (buy/sell)
- ✅ Cantidad > 0
- ✅ Símbolo válido (existe en Market Data)
- ✅ Balance suficiente (solo buy)
- ✅ Holdings suficientes (solo sell)

### Validaciones en ejecución (POST /orders/:id/execute)
- ✅ Orden existe
- ✅ Usuario es propietario
- ✅ Estado = pending
- ✅ Balance suficiente (re-verificación)
- ✅ Holdings suficientes (re-verificación en sell)
- ✅ Transacción idempotente (no procesar 2 veces)

---

## 6. MANEJO DE ERRORES

### Errores comunes

| Error | Código | Causa | Solución |
|-------|--------|-------|----------|
| Invalid credentials | 401 | JWT inválido | Re-login |
| User not found | 404 | Usuario no existe | Verificar user_id |
| Insufficient balance | 400 | Balance < total + fee | Esperar o reducir cantidad |
| Insufficient holdings | 400 | No posee suficiente cripto | Verificar portfolio |
| Invalid crypto symbol | 400 | Símbolo no existe | Usar BTC, ETH, etc. |
| Order already executed | 409 | Estado != pending | Ver historial |
| Order not found | 404 | ID inválido | Verificar order_id |
| Transaction already processed | 409 | Idempotencia | Evita double-spend |
| Market Data unavailable | 503 | API externa caída | Reintentar más tarde |

---

## 7. EJEMPLOS COMPLETOS

### Compra completa
```bash
# 1. Login
TOKEN=$(curl -s -X POST http://localhost:8001/api/users/login \
  -H "Content-Type: application/json" \
  -d '{"email":"trader@cryptosim.com","password":"SecurePass123!"}' \
  | jq -r '.data.tokens.access_token')

# 2. Ver balance inicial
curl http://localhost:8001/api/users/123 \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.data.user.initial_balance'
# Output: 100000.00

# 3. Crear orden de compra
ORDER_ID=$(curl -s -X POST http://localhost:8002/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"buy","crypto_symbol":"BTC","quantity":0.001,"order_kind":"market"}' \
  | jq -r '.data.order.id')

echo "Order created: $ORDER_ID"

# 4. Ejecutar orden
curl -X POST http://localhost:8002/api/v1/orders/$ORDER_ID/execute \
  -H "Authorization: Bearer $TOKEN"

# 5. Ver nuevo balance
curl http://localhost:8001/api/users/123 \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.data.user.initial_balance'
# Output: 99949.95

# 6. Ver portfolio
curl http://localhost:8005/api/portfolios/123 \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.data.portfolio.holdings[0]'
```

### Venta completa
```bash
# 1. Crear orden de venta
SELL_ORDER_ID=$(curl -s -X POST http://localhost:8002/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"sell","crypto_symbol":"BTC","quantity":0.0005,"order_kind":"market"}' \
  | jq -r '.data.order.id')

# 2. Ejecutar venta
curl -X POST http://localhost:8002/api/v1/orders/$SELL_ORDER_ID/execute \
  -H "Authorization: Bearer $TOKEN"

# 3. Ver nuevo balance (aumentó)
curl http://localhost:8001/api/users/123 \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.data.user.initial_balance'
# Output: 99975.42
```

---

## Resumen

1. **Crear orden**: Validaciones + cálculos + guardar en MongoDB (status: pending)
2. **Ejecutar orden**: Actualizar balance en MySQL + cambiar status a executed
3. **Evento RabbitMQ**: Publicar orders.executed
4. **Portfolio update**: Consumer actualiza holdings + recalcula métricas
5. **Compra**: Decrementa balance, incrementa holdings
6. **Venta**: Incrementa balance, decrementa holdings (FIFO)
7. **Validaciones**: Múltiples capas (input, balance, holdings, idempotencia)
8. **Transacciones**: MySQL usa transactions, idempotencia con order_id único
