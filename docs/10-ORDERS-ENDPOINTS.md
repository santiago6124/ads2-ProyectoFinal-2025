# 10 - Orders API - Referencia de Endpoints

## 📋 Tabla de Contenidos
- [Crear Orden](#crear-orden)
- [Listar Órdenes](#listar-órdenes)
- [Obtener Orden](#obtener-orden)
- [Ejecutar Orden](#ejecutar-orden)
- [Cancelar Orden](#cancelar-orden)
- [Health Check](#health-check)

---

## 📝 Crear Orden

### POST /api/v1/orders

Crea una nueva orden de compra o venta.

**Request**:
```http
POST /api/v1/orders HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "crypto_symbol": "BTC",
  "quantity": 0.1,
  "order_type": "buy",
  "order_kind": "market"
}
```

**Parámetros**:
| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| crypto_symbol | string | Sí | Símbolo de la criptomoneda (BTC, ETH, etc.) |
| quantity | float64 | Sí | Cantidad a comprar/vender (> 0) |
| order_type | string | Sí | Tipo: "buy" o "sell" |
| order_kind | string | Sí | Clase: "market" (actualmente solo market) |

**Response** (201 Created):
```json
{
  "id": "6554a2b8c3e4f1234567890a",
  "user_id": "1",
  "crypto_symbol": "BTC",
  "quantity": 0.1,
  "order_type": "buy",
  "order_kind": "market",
  "status": "pending",
  "price_at_creation": 50000.00,
  "total_amount": 0,
  "fee": 0,
  "fee_percentage": 0.001,
  "created_at": "2025-11-14T15:30:00Z",
  "executed_at": null,
  "error_message": null
}
```

**Errores**:
- `400 Bad Request`: Validación fallida
- `401 Unauthorized`: Token inválido
- `422 Unprocessable Entity`: Símbolo no soportado
- `500 Internal Server Error`: Error al crear orden

---

## 📋 Listar Órdenes

### GET /api/v1/orders

Obtiene todas las órdenes del usuario autenticado.

**Query Parameters**:
| Parámetro | Tipo | Descripción |
|-----------|------|-------------|
| status | string | Filtrar por estado: pending, executed, cancelled, failed |
| order_type | string | Filtrar por tipo: buy, sell |
| crypto_symbol | string | Filtrar por símbolo: BTC, ETH, etc. |
| limit | int | Número de resultados (default: 50, max: 100) |
| offset | int | Paginación offset (default: 0) |

**Request**:
```http
GET /api/v1/orders?status=executed&limit=10 HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response** (200 OK):
```json
{
  "orders": [
    {
      "id": "6554a2b8c3e4f1234567890a",
      "user_id": "1",
      "crypto_symbol": "BTC",
      "quantity": 0.1,
      "order_type": "buy",
      "order_kind": "market",
      "status": "executed",
      "price_at_creation": 50000.00,
      "execution_price": 50050.00,
      "total_amount": 5005.00,
      "fee": 5.00,
      "fee_percentage": 0.001,
      "created_at": "2025-11-14T15:30:00Z",
      "executed_at": "2025-11-14T15:30:05Z",
      "error_message": null
    }
  ],
  "total": 1,
  "limit": 10,
  "offset": 0
}
```

---

## 🔍 Obtener Orden

### GET /api/v1/orders/:id

Obtiene los detalles de una orden específica.

**Request**:
```http
GET /api/v1/orders/6554a2b8c3e4f1234567890a HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response** (200 OK):
```json
{
  "id": "6554a2b8c3e4f1234567890a",
  "user_id": "1",
  "crypto_symbol": "BTC",
  "quantity": 0.1,
  "order_type": "buy",
  "order_kind": "market",
  "status": "executed",
  "price_at_creation": 50000.00,
  "execution_price": 50050.00,
  "total_amount": 5005.00,
  "fee": 5.00,
  "fee_percentage": 0.001,
  "created_at": "2025-11-14T15:30:00Z",
  "executed_at": "2025-11-14T15:30:05Z",
  "error_message": null
}
```

**Errores**:
- `404 Not Found`: Orden no existe o no pertenece al usuario
- `401 Unauthorized`: Token inválido

---

## ⚡ Ejecutar Orden

### POST /api/v1/orders/:id/execute

Ejecuta una orden pendiente.

**Request**:
```http
POST /api/v1/orders/6554a2b8c3e4f1234567890a/execute HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Flujo de Ejecución**:
1. Valida que la orden esté en estado "pending"
2. Consulta precio actual de Market Data API
3. Solicita balance actual al Users API (via RabbitMQ)
4. Valida balance suficiente (para compras)
5. Calcula fee (0.1% para market orders)
6. Actualiza estado a "executed"
7. Publica evento `orders.executed` a RabbitMQ
8. Portfolio API y Search API reaccionan al evento

**Response** (200 OK):
```json
{
  "id": "6554a2b8c3e4f1234567890a",
  "user_id": "1",
  "crypto_symbol": "BTC",
  "quantity": 0.1,
  "order_type": "buy",
  "order_kind": "market",
  "status": "executed",
  "price_at_creation": 50000.00,
  "execution_price": 50050.00,
  "total_amount": 5005.00,
  "fee": 5.00,
  "fee_percentage": 0.001,
  "created_at": "2025-11-14T15:30:00Z",
  "executed_at": "2025-11-14T15:30:05Z",
  "error_message": null
}
```

**Errores**:
- `400 Bad Request`: Orden ya ejecutada/cancelada
- `402 Payment Required`: Balance insuficiente
- `404 Not Found`: Orden no existe
- `422 Unprocessable Entity`: Precio no disponible
- `500 Internal Server Error`: Error en ejecución

---

## ❌ Cancelar Orden

### POST /api/v1/orders/:id/cancel

Cancela una orden pendiente.

**Request**:
```http
POST /api/v1/orders/6554a2b8c3e4f1234567890a/cancel HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response** (200 OK):
```json
{
  "id": "6554a2b8c3e4f1234567890a",
  "status": "cancelled",
  "cancelled_at": "2025-11-14T15:35:00Z"
}
```

**Errores**:
- `400 Bad Request`: Solo órdenes "pending" pueden cancelarse
- `404 Not Found`: Orden no existe

---

## ❤️ Health Check

### GET /health

Verifica el estado del servicio.

**Response** (200 OK):
```json
{
  "status": "healthy",
  "service": "orders-api",
  "timestamp": "2025-11-14T15:30:00Z",
  "dependencies": {
    "mongodb": "connected",
    "rabbitmq": "connected",
    "users-api": "reachable",
    "market-data-api": "reachable"
  }
}
```

---

## 💰 Cálculo de Fees

### Fórmula

```
Para COMPRAS (BUY):
  amount = quantity * execution_price
  fee = amount * fee_percentage
  total_cost = amount + fee
  balance_required = total_cost

Para VENTAS (SELL):
  amount = quantity * execution_price
  fee = amount * fee_percentage
  total_received = amount - fee
  balance_added = total_received
```

### Configuración Actual

```
FEE_BASE_PERCENTAGE = 0.001  (0.1%)
FEE_MAKER = 0.0008           (0.08% - futuro)
FEE_TAKER = 0.0012           (0.12% - futuro)
```

### Ejemplos

**Compra de 0.1 BTC a $50,000**:
```
amount = 0.1 * 50000 = $5,000
fee = 5000 * 0.001 = $5
total_cost = 5000 + 5 = $5,005
balance_required = $5,005
```

**Venta de 0.5 ETH a $3,000**:
```
amount = 0.5 * 3000 = $1,500
fee = 1500 * 0.001 = $1.50
total_received = 1500 - 1.50 = $1,498.50
balance_added = $1,498.50
```

---

## 📊 Estados de Orden

| Estado | Descripción | Transiciones Permitidas |
|--------|-------------|-------------------------|
| pending | Orden creada pero no ejecutada | → executed, cancelled |
| executed | Orden ejecutada exitosamente | (final) |
| cancelled | Orden cancelada por el usuario | (final) |
| failed | Orden falló en ejecución | (final) |

### Diagrama de Estados

```
   ┌─────────┐
   │ pending │
   └────┬────┘
        │
        ├───────────┬──────────┐
        │           │          │
        ▼           ▼          ▼
   ┌─────────┐ ┌─────────┐ ┌────────┐
   │executed │ │cancelled│ │ failed │
   └─────────┘ └─────────┘ └────────┘
```

---

## 🔄 Eventos RabbitMQ

### Publicados por Orders API

**Exchange**: `orders.events` (topic)

**Routing Keys**:
- `orders.created`: Orden creada
- `orders.executed`: Orden ejecutada
- `orders.cancelled`: Orden cancelada
- `orders.failed`: Orden falló

**Payload Ejemplo**:
```json
{
  "event_type": "orders.executed",
  "order_id": "6554a2b8c3e4f1234567890a",
  "user_id": "1",
  "crypto_symbol": "BTC",
  "quantity": 0.1,
  "order_type": "buy",
  "execution_price": 50050.00,
  "total_amount": 5005.00,
  "fee": 5.00,
  "timestamp": "2025-11-14T15:30:05Z"
}
```

---

## 🧪 Ejemplos con cURL

### Crear Orden de Compra

```bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

curl -X POST http://localhost:8002/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "crypto_symbol": "BTC",
    "quantity": 0.01,
    "order_type": "buy",
    "order_kind": "market"
  }'
```

### Ejecutar Orden

```bash
ORDER_ID="6554a2b8c3e4f1234567890a"

curl -X POST http://localhost:8002/api/v1/orders/$ORDER_ID/execute \
  -H "Authorization: Bearer $TOKEN"
```

### Listar Órdenes Ejecutadas

```bash
curl -X GET "http://localhost:8002/api/v1/orders?status=executed&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 📝 Notas Importantes

1. **Solo Market Orders**: Actualmente solo soporta órdenes de mercado (ejecución inmediata)
2. **Validación de Balance**: Se valida antes de ejecutar (compras)
3. **Precios Reales**: Se obtienen de Market Data API en tiempo de ejecución
4. **Eventos Asíncronos**: La actualización de portfolio/search es eventual
5. **Idempotencia**: Ejecutar una orden ya ejecutada devuelve error 400
6. **Timeout**: Las operaciones con APIs externas tienen timeout de 30s
