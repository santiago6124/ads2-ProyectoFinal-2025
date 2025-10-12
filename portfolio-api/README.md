# 💼 Portfolio API - CryptoSim Platform

Microservicio de gestión y cálculo de portafolios de inversión con métricas de rendimiento.

## 🚀 Quick Start (Recommended)

```bash
# Desde la raíz del proyecto
cd /ads2-ProyectoFinal-2025
make up              # Levantar todos los servicios
# O:
make up-portfolio    # Levantar solo Portfolio API + dependencias
```

**URLs del servicio:**
- **Portfolio API**: http://localhost:8005
- **Health Check**: http://localhost:8005/health

**Ver logs:**
```bash
make logs-portfolio
```

---

## 🏗️ Arquitectura & Dependencias

### Dependencias:
- **MongoDB 7.0** (`portfolio-mongo`) - Base de datos principal
- **Redis** (`shared-redis`) - Cache de cálculos (TTL: 15min)
- **RabbitMQ** (`shared-rabbitmq`) - Consumer de eventos de órdenes

### Comunica con:
- **Market Data API** (http://market-data-api:8004) - Precios actuales
- **Orders API** (http://orders-api:8080) - Historial de trades
- **Users API** (http://users-api:8001) - Validación de usuarios

### Eventos que consume:
- `order.executed` - Actualiza holdings cuando se ejecuta una orden
- `order.cancelled` - Limpia locks de holdings

---

## ⚡ Características

- **Cálculo Automático**: Scheduler que actualiza portfolios cada 15 minutos
- **P&L Tracking**: Profit/Loss absoluto y porcentual
- **Performance Metrics**: Daily, weekly, monthly, yearly returns
- **Holdings Management**: Gestión automática de holdings por usuario
- **Snapshots Históricos**: Fotos del portfolio en puntos específicos del tiempo
- **Risk Metrics**: Métricas de riesgo (Sharpe ratio, volatilidad, etc.)

## 📊 Endpoints Principales

### Obtener Portfolio Completo
```http
GET /api/portfolio/:userId
Authorization: Bearer {jwt_token}
```

Respuesta:
```json
{
  "user_id": 1,
  "total_value": 52450.75,
  "total_invested": 50000.00,
  "profit_loss": 2450.75,
  "profit_loss_percentage": 4.90,
  "holdings": [
    {
      "symbol": "BTC",
      "name": "Bitcoin",
      "quantity": 0.5,
      "average_buy_price": 42000.00,
      "current_price": 45000.00,
      "current_value": 22500.00,
      "profit_loss": 1500.00,
      "profit_loss_percentage": 7.14
    }
  ],
  "last_calculated": "2025-10-12T15:30:00Z"
}
```

### Métricas de Rendimiento
```http
GET /api/portfolio/:userId/performance
Authorization: Bearer {jwt_token}
```

### Holdings Actuales
```http
GET /api/portfolio/:userId/holdings
Authorization: Bearer {jwt_token}
```

### Histórico de Portfolio
```http
GET /api/portfolio/:userId/history?from=2025-01-01&to=2025-10-12
Authorization: Bearer {jwt_token}
```

### Crear Snapshot Manual
```http
POST /api/portfolio/:userId/snapshot
Authorization: Bearer {jwt_token}
```

## 🔧 Variables de Entorno

```env
PORT=8080
HOST=0.0.0.0

# MongoDB
DB_URI=mongodb://portfolio-mongo:27017/portfolio_db
DB_NAME=portfolio_db

# Redis
REDIS_HOST=shared-redis
REDIS_PORT=6379
CACHE_PORTFOLIO_TTL=1h
CACHE_PERFORMANCE_TTL=30m

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@shared-rabbitmq:5672/
RABBITMQ_EXCHANGE=portfolio_events
RABBITMQ_QUEUE=portfolio_calculations

# External APIs
MARKET_DATA_API_URL=http://market-data-api:8004
ORDERS_API_URL=http://orders-api:8080
USERS_API_URL=http://users-api:8001

# Scheduler
SCHEDULER_ENABLED=true
PORTFOLIO_CALC_CRON=0 */15 * * * *  # Cada 15 minutos
```

## 🕒 Scheduler

El scheduler ejecuta tareas automáticas:

| Tarea | Frecuencia | Descripción |
|-------|-----------|-------------|
| Portfolio Calculation | Cada 15 min | Recalcula portfolios activos |
| Daily Snapshot | Cada 24h (00:00) | Crea snapshot diario |
| Cleanup Old Data | Cada 24h (02:00) | Limpia snapshots antiguos (>90 días) |

## 🧪 Testing

```bash
cd portfolio-api
go test ./...
```

## 🐛 Troubleshooting

### Portfolio no se actualiza
- Verificar scheduler: `SCHEDULER_ENABLED=true`
- Ver logs: `make logs-portfolio`
- Verificar Market Data API está corriendo

### RabbitMQ no consume eventos
```bash
# Ver estado del consumer
make logs-rabbitmq

# Verificar cola
open http://localhost:15672  # guest/guest
```

## 📚 Documentación

- [README Principal](../README.md)
- [QUICKSTART](../QUICKSTART.md)

---

**Portfolio API** - Parte del ecosistema CryptoSim 🚀
