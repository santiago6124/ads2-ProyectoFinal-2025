# 📦 Orders API - CryptoSim Platform

Microservicio de gestión de órdenes de compra/venta con ejecución concurrente y procesamiento de fees.

## 🚀 Quick Start (Recommended)

**Este servicio es parte del ecosistema CryptoSim.** La forma recomendada de ejecutarlo es usando el **Docker Compose unificado** en la raíz:

```bash
# Desde la raíz del proyecto
cd /ads2-ProyectoFinal-2025
make up              # Levantar todos los servicios
# O:
make up-orders       # Levantar solo Orders API + dependencias
```

**URLs del servicio:**
- **Orders API**: http://localhost:8002
- **Health Check**: http://localhost:8002/health
- **Metrics**: http://localhost:8002/metrics

**Ver logs:**
```bash
make logs-orders
```

**Acceder al contenedor:**
```bash
make shell-orders
```

---

## 🏗️ Arquitectura & Dependencias

### Dependencias requeridas:
- **MongoDB 7.0** (`orders-mongo` container)
- **RabbitMQ 3.12** (`shared-rabbitmq` container)
- **Redis** (`shared-redis` container - opcional para cache)

### Comunica con:
- **Users API** (http://users-api:8001) - Verificación de usuarios
- **Wallet API** (http://wallet-api:8080) - Lock/release de fondos
- **Market Data API** (http://market-data-api:8004) - Precios actuales

### Es consumido por:
- Portfolio API (para actualización de holdings)
- Frontend (creación de órdenes)

**Documentación completa**: Ver [README principal](../README.md)

---

## ⚡ Características

- **Ejecución Concurrente**: Uso de goroutines para procesamiento paralelo
- **Cálculo de Fees**: Maker/Taker fees con configuración personalizada
- **Slippage Simulation**: Simulación realista de condiciones de mercado
- **Message Queue**: Integración con RabbitMQ para eventos async
- **Validación de Saldo**: Verificación automática con Wallet API
- **Historial Completo**: Tracking de todas las órdenes por usuario

## 📊 Endpoints Principales

### Crear Orden
```http
POST /api/orders
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "user_id": 1,
  "type": "buy",
  "crypto_symbol": "BTC",
  "crypto_name": "Bitcoin",
  "quantity": 0.1,
  "order_price": 45000.00
}
```

### Obtener Orden
```http
GET /api/orders/:id
Authorization: Bearer {jwt_token}
```

### Listar Órdenes de Usuario
```http
GET /api/orders/user/:userId?status=executed&limit=20&offset=0
Authorization: Bearer {jwt_token}
```

### Ejecutar Orden
```http
POST /api/orders/:id/execute
Authorization: Bearer {jwt_token}
```

### Cancelar Orden
```http
DELETE /api/orders/:id
Authorization: Bearer {jwt_token}
```

## 🔧 Variables de Entorno

Ver [`.env.example`](../.env.example) en la raíz del proyecto.

Principales variables:
```env
# MongoDB
MONGODB_URI=mongodb://orders-mongo:27017
MONGODB_DATABASE=cryptosim_orders

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@shared-rabbitmq:5672/
RABBITMQ_EXCHANGE=orders
RABBITMQ_WORKER_COUNT=5

# External APIs
USER_API_BASE_URL=http://users-api:8001
WALLET_API_BASE_URL=http://wallet-api:8080
MARKET_API_BASE_URL=http://market-data-api:8004

# Fees
FEE_BASE_PERCENTAGE=0.001
FEE_MAKER=0.0008
FEE_TAKER=0.0012
```

## 🧪 Testing

```bash
# Desde la raíz del proyecto
cd orders-api

# Unit tests
go test ./internal/...

# Integration tests
go test ./tests/integration/...

# Con coverage
go test -cover ./...
```

## 🛠️ Desarrollo Local

Para desarrollo sin Docker:

```bash
cd orders-api

# Instalar dependencias
go mod download

# Ejecutar (requiere MongoDB y RabbitMQ externos)
go run cmd/server/main.go
```

## 🐛 Troubleshooting

### Orden no se ejecuta
- Verificar saldo suficiente en Wallet API
- Revisar logs: `make logs-orders`
- Verificar conexión con Market Data API

### Error de conexión MongoDB
```bash
# Verificar que MongoDB está corriendo
docker-compose ps orders-mongo

# Ver logs de MongoDB
make logs-mongo
```

### RabbitMQ no conecta
```bash
# Ver estado de RabbitMQ
docker-compose ps shared-rabbitmq

# Acceder al management UI
open http://localhost:15672  # guest/guest
```

## 📚 Documentación Adicional

- [README Principal](../README.md) - Documentación completa del proyecto
- [QUICKSTART](../QUICKSTART.md) - Guía de inicio rápido
- [API Docs](http://localhost:8002/swagger) - Swagger/OpenAPI (cuando está corriendo)

---

**Orders API** - Parte del ecosistema de microservicios CryptoSim 🚀
