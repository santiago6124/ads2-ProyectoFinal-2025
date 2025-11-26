# 03 - Guía de Uso de CryptoSim

## 📋 Tabla de Contenidos
- [Primeros Pasos](#primeros-pasos)
- [Registro y Login](#registro-y-login)
- [Consultar Precios](#consultar-precios)
- [Realizar Trading](#realizar-trading)
- [Gestionar Portfolio](#gestionar-portfolio)
- [Buscar Órdenes](#buscar-órdenes)
- [Casos de Uso Completos](#casos-de-uso-completos)

---

## 🚀 Primeros Pasos

### Herramientas Recomendadas

Para interactuar con la API, puedes usar:

1. **cURL** (línea de comandos)
2. **Postman** (GUI amigable) - [Descargar](https://www.postman.com/)
3. **Insomnia** (alternativa a Postman) - [Descargar](https://insomnia.rest/)
4. **HTTPie** (CLI mejorado) - [Descargar](https://httpie.io/)

### Variables de Entorno (para cURL)

**Linux/Mac**:
```bash
export API_BASE=http://localhost:8001
export ORDERS_BASE=http://localhost:8002
export SEARCH_BASE=http://localhost:8003
export MARKET_BASE=http://localhost:8004
export PORTFOLIO_BASE=http://localhost:8005
```

**Windows (PowerShell)**:
```powershell
$env:API_BASE = "http://localhost:8001"
$env:ORDERS_BASE = "http://localhost:8002"
$env:SEARCH_BASE = "http://localhost:8003"
$env:MARKET_BASE = "http://localhost:8004"
$env:PORTFOLIO_BASE = "http://localhost:8005"
```

---

## 👤 Registro y Login

### 1. Registrar un Nuevo Usuario

```bash
curl -X POST $API_BASE/api/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "trader@cryptosim.com",
    "password": "SecurePass123!",
    "username": "crypto_trader_01"
  }'
```

**Respuesta**:
```json
{
  "id": 1,
  "email": "trader@cryptosim.com",
  "username": "crypto_trader_01",
  "balance": 100000.00,
  "created_at": "2025-11-14T10:00:00Z"
}
```

**Nota**: Recibes $100,000 USD virtuales automáticamente.

---

### 2. Hacer Login

```bash
curl -X POST $API_BASE/api/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "trader@cryptosim.com",
    "password": "SecurePass123!"
  }'
```

**Respuesta**:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwiZW1haWwiOiJ0cmFkZXJAY3J5cHRvc2ltLmNvbSIsInVzZXJuYW1lIjoiY3J5cHRvX3RyYWRlcl8wMSIsImlhdCI6MTY5OTk5OTk5OSwiZXhwIjoxNzAwMDAzNTk5LCJpc3MiOiJ1c2Vycy1hcGkiLCJhdWQiOiJjcnlwdG9zaW0ifQ.XXXX",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.XXXX",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

**Guardar el token**:

**Linux/Mac**:
```bash
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**Windows**:
```powershell
$env:TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

### 3. Ver tu Perfil

```bash
curl -X GET $API_BASE/api/users/me \
  -H "Authorization: Bearer $TOKEN"
```

**Respuesta**:
```json
{
  "id": 1,
  "email": "trader@cryptosim.com",
  "username": "crypto_trader_01",
  "balance": 100000.00,
  "created_at": "2025-11-14T10:00:00Z",
  "updated_at": "2025-11-14T10:00:00Z"
}
```

---

## 💰 Consultar Precios

### Ver Todas las Criptomonedas Disponibles

```bash
curl -X GET $MARKET_BASE/api/v1/prices
```

**Respuesta** (abreviada):
```json
{
  "prices": [
    {
      "symbol": "bitcoin",
      "name": "Bitcoin",
      "current_price": 50000.00,
      "market_cap": 980000000000,
      "volume_24h": 35000000000,
      "price_change_24h": 2.5,
      "last_updated": "2025-11-14T15:30:00Z"
    },
    {
      "symbol": "ethereum",
      "name": "Ethereum",
      "current_price": 3000.00,
      "price_change_24h": 1.8
    }
  ],
  "total": 50,
  "timestamp": "2025-11-14T15:30:00Z"
}
```

---

### Ver Precio de una Criptomoneda Específica

```bash
# Bitcoin
curl -X GET $MARKET_BASE/api/v1/prices/bitcoin

# Ethereum
curl -X GET $MARKET_BASE/api/v1/prices/ethereum

# Solana
curl -X GET $MARKET_BASE/api/v1/prices/solana
```

**Respuesta**:
```json
{
  "symbol": "bitcoin",
  "name": "Bitcoin",
  "current_price": 50000.00,
  "market_cap": 980000000000,
  "volume_24h": 35000000000,
  "price_change_24h": 2.5,
  "price_change_percentage_24h": 0.05,
  "high_24h": 51000.00,
  "low_24h": 49000.00,
  "circulating_supply": 19600000,
  "max_supply": 21000000,
  "last_updated": "2025-11-14T15:30:00Z"
}
```

---

### Ver Historial de Precios

```bash
curl -X GET "$MARKET_BASE/api/v1/prices/bitcoin/history?days=7"
```

**Parámetros**:
- `days`: Número de días (1, 7, 30, 90, 365)

---

## 📈 Realizar Trading

### Flujo Completo: Comprar Bitcoin

#### Paso 1: Ver Precio Actual

```bash
curl -X GET $MARKET_BASE/api/v1/prices/bitcoin
```

**Nota el precio**: Ej. $50,000

#### Paso 2: Crear Orden de Compra

```bash
curl -X POST $ORDERS_BASE/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "crypto_symbol": "BTC",
    "quantity": 0.1,
    "order_type": "buy",
    "order_kind": "market"
  }'
```

**Respuesta**:
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
  "created_at": "2025-11-14T15:30:00Z"
}
```

**Guardar el ORDER_ID**:
```bash
export ORDER_ID="6554a2b8c3e4f1234567890a"
```

#### Paso 3: Ejecutar la Orden

```bash
curl -X POST $ORDERS_BASE/api/v1/orders/$ORDER_ID/execute \
  -H "Authorization: Bearer $TOKEN"
```

**Respuesta**:
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
  "executed_at": "2025-11-14T15:30:05Z"
}
```

**Cálculo**:
```
Precio: $50,050
Cantidad: 0.1 BTC
Subtotal: 0.1 × $50,050 = $5,005
Fee (0.1%): $5,005 × 0.001 = $5.00
Total pagado: $5,005 + $5 = $5,010
Balance restante: $100,000 - $5,010 = $94,990
```

---

### Vender Criptomonedas

```bash
# 1. Crear orden de venta
curl -X POST $ORDERS_BASE/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "crypto_symbol": "BTC",
    "quantity": 0.05,
    "order_type": "sell",
    "order_kind": "market"
  }'

# 2. Ejecutar (usar el ORDER_ID recibido)
curl -X POST $ORDERS_BASE/api/v1/orders/$ORDER_ID/execute \
  -H "Authorization: Bearer $TOKEN"
```

---

### Cancelar una Orden Pendiente

```bash
curl -X POST $ORDERS_BASE/api/v1/orders/$ORDER_ID/cancel \
  -H "Authorization: Bearer $TOKEN"
```

---

### Ver tus Órdenes

**Todas las órdenes**:
```bash
curl -X GET $ORDERS_BASE/api/v1/orders \
  -H "Authorization: Bearer $TOKEN"
```

**Solo órdenes ejecutadas**:
```bash
curl -X GET "$ORDERS_BASE/api/v1/orders?status=executed" \
  -H "Authorization: Bearer $TOKEN"
```

**Solo órdenes de compra**:
```bash
curl -X GET "$ORDERS_BASE/api/v1/orders?order_type=buy" \
  -H "Authorization: Bearer $TOKEN"
```

**Filtros combinados**:
```bash
curl -X GET "$ORDERS_BASE/api/v1/orders?status=executed&order_type=buy&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 💼 Gestionar Portfolio

### Ver tu Portfolio Completo

```bash
curl -X GET $PORTFOLIO_BASE/api/portfolios/1 \
  -H "Authorization: Bearer $TOKEN"
```

**Respuesta**:
```json
{
  "user_id": "1",
  "total_value": 105000.00,
  "cash_balance": 94990.00,
  "crypto_value": 10010.00,
  "holdings": [
    {
      "symbol": "BTC",
      "quantity": 0.1,
      "average_buy_price": 50050.00,
      "current_price": 51000.00,
      "total_value": 5100.00,
      "unrealized_pnl": 95.00,
      "unrealized_pnl_percentage": 1.89,
      "percentage_of_portfolio": 4.86
    }
  ],
  "performance_metrics": {
    "total_pnl": 5000.00,
    "roi": 5.0,
    "roi_percentage": 5.0
  },
  "risk_metrics": {
    "sharpe_ratio": 1.5,
    "sortino_ratio": 2.1,
    "max_drawdown": -0.05,
    "volatility": 0.15,
    "var_95": -1500.00
  },
  "diversification": {
    "holdings_count": 1,
    "herfindahl_index": 1.0,
    "concentration_top_3": 100.0
  },
  "last_calculated": "2025-11-14T15:30:00Z"
}
```

---

### Ver Solo Holdings

```bash
curl -X GET $PORTFOLIO_BASE/api/portfolios/1/holdings \
  -H "Authorization: Bearer $TOKEN"
```

---

### Ver Métricas de Rendimiento

```bash
curl -X GET $PORTFOLIO_BASE/api/portfolios/1/performance \
  -H "Authorization: Bearer $TOKEN"
```

**Respuesta**:
```json
{
  "total_pnl": 5000.00,
  "realized_pnl": 0.00,
  "unrealized_pnl": 5000.00,
  "roi": 5.0,
  "roi_percentage": 5.0,
  "twrr": 5.2,
  "total_invested": 100000.00,
  "current_value": 105000.00,
  "total_fees_paid": 10.00
}
```

---

### Ver Métricas de Riesgo

```bash
curl -X GET $PORTFOLIO_BASE/api/portfolios/1/risk \
  -H "Authorization: Bearer $TOKEN"
```

**Respuesta**:
```json
{
  "sharpe_ratio": 1.5,
  "sortino_ratio": 2.1,
  "max_drawdown": -0.05,
  "max_drawdown_percentage": -5.0,
  "volatility": 0.15,
  "volatility_percentage": 15.0,
  "var_95": -1500.00,
  "cvar_95": -2000.00,
  "beta": 0.8
}
```

**Interpretación de Métricas**:

- **Sharpe Ratio > 1**: Buen rendimiento ajustado por riesgo
- **Sortino Ratio > 1**: Buen rendimiento considerando solo volatilidad negativa
- **Max Drawdown**: Mayor pérdida desde un pico
- **Volatility**: Variabilidad de retornos (menor = más estable)
- **VaR 95%**: Máxima pérdida esperada en el 95% de los casos

---

### Ver Snapshots Históricos

```bash
curl -X GET $PORTFOLIO_BASE/api/portfolios/1/snapshots \
  -H "Authorization: Bearer $TOKEN"
```

**Response**:
```json
{
  "snapshots": [
    {
      "timestamp": "2025-11-14T00:00:00Z",
      "total_value": 100000.00,
      "crypto_value": 0.00,
      "cash_balance": 100000.00
    },
    {
      "timestamp": "2025-11-14T12:00:00Z",
      "total_value": 103000.00,
      "crypto_value": 8000.00,
      "cash_balance": 95000.00
    }
  ]
}
```

---

## 🔍 Buscar Órdenes

### Búsqueda Simple

```bash
curl -X POST $SEARCH_BASE/api/v1/search \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "*"
  }'
```

---

### Búsqueda con Filtros

```bash
curl -X POST $SEARCH_BASE/api/v1/search \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "BTC",
    "filters": {
      "status": "executed",
      "order_type": "buy"
    },
    "sort_by": "created_at",
    "sort_order": "desc",
    "page": 1,
    "page_size": 10
  }'
```

**Filtros disponibles**:
- `status`: pending, executed, cancelled, failed
- `order_type`: buy, sell
- `crypto_symbol`: BTC, ETH, etc.
- `date_from`: 2025-11-01
- `date_to`: 2025-11-30
- `min_amount`: 1000
- `max_amount`: 10000

---

### Búsqueda por Rango de Fechas

```bash
curl -X POST $SEARCH_BASE/api/v1/search \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "*",
    "filters": {
      "date_from": "2025-11-01",
      "date_to": "2025-11-30"
    }
  }'
```

---

## 🎯 Casos de Uso Completos

### Caso 1: Day Trading - Comprar y Vender en el Mismo Día

```bash
# 1. Ver precio de Ethereum
curl $MARKET_BASE/api/v1/prices/ethereum

# 2. Comprar 1 ETH
curl -X POST $ORDERS_BASE/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"crypto_symbol": "ETH", "quantity": 1, "order_type": "buy", "order_kind": "market"}'

# Guardar ORDER_ID de la respuesta
ORDER_BUY="..."

# 3. Ejecutar compra
curl -X POST $ORDERS_BASE/api/v1/orders/$ORDER_BUY/execute \
  -H "Authorization: Bearer $TOKEN"

# 4. Esperar cambio de precio...
# 5. Verificar precio actual
curl $MARKET_BASE/api/v1/prices/ethereum

# 6. Vender 0.5 ETH
curl -X POST $ORDERS_BASE/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"crypto_symbol": "ETH", "quantity": 0.5, "order_type": "sell", "order_kind": "market"}'

ORDER_SELL="..."

# 7. Ejecutar venta
curl -X POST $ORDERS_BASE/api/v1/orders/$ORDER_SELL/execute \
  -H "Authorization: Bearer $TOKEN"

# 8. Ver portfolio actualizado
curl $PORTFOLIO_BASE/api/portfolios/1 -H "Authorization: Bearer $TOKEN"
```

---

### Caso 2: Portfolio Diversificado

```bash
# Comprar múltiples criptomonedas

# Bitcoin (40% del capital)
curl -X POST $ORDERS_BASE/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"crypto_symbol": "BTC", "quantity": 0.8, "order_type": "buy", "order_kind": "market"}'
# Ejecutar...

# Ethereum (30% del capital)
curl -X POST $ORDERS_BASE/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"crypto_symbol": "ETH", "quantity": 10, "order_type": "buy", "order_kind": "market"}'
# Ejecutar...

# Solana (20% del capital)
curl -X POST $ORDERS_BASE/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"crypto_symbol": "SOL", "quantity": 200, "order_type": "buy", "order_kind": "market"}'
# Ejecutar...

# Cardano (10% del capital)
curl -X POST $ORDERS_BASE/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"crypto_symbol": "ADA", "quantity": 10000, "order_type": "buy", "order_kind": "market"}'
# Ejecutar...

# Ver diversificación
curl $PORTFOLIO_BASE/api/portfolios/1 -H "Authorization: Bearer $TOKEN" | jq '.diversification'
```

---

### Caso 3: Análisis de Rendimiento

```bash
# 1. Ver métricas generales
curl $PORTFOLIO_BASE/api/portfolios/1/performance \
  -H "Authorization: Bearer $TOKEN"

# 2. Ver métricas de riesgo
curl $PORTFOLIO_BASE/api/portfolios/1/risk \
  -H "Authorization: Bearer $TOKEN"

# 3. Ver historial de operaciones
curl "$ORDERS_BASE/api/v1/orders?status=executed" \
  -H "Authorization: Bearer $TOKEN"

# 4. Calcular fees totales pagados
curl "$ORDERS_BASE/api/v1/orders?status=executed" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '[.orders[].fee] | add'
```

---

## 📱 Integración con Postman

### Importar Colección

1. Descargar [colección de Postman](../postman/CryptoSim.postman_collection.json)
2. Abrir Postman
3. Import → Upload Files → Seleccionar archivo JSON
4. Configurar variables de entorno:
   - `base_url`: http://localhost:8001
   - `token`: (se auto-completa al hacer login)

---

## 📝 Buenas Prácticas

1. **Siempre verificar precios antes de operar**
2. **Revisar balance antes de crear órdenes grandes**
3. **Guardar el token JWT de manera segura**
4. **No compartir credenciales**
5. **Usar cantidades pequeñas para probar primero**
6. **Monitorear portfolio regularmente**
7. **Analizar métricas de riesgo antes de decisiones importantes**

---

**Siguiente**: [04 - Arquitectura General](04-ARQUITECTURA-GENERAL.md) →
