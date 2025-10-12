# 💰 Wallet API - CryptoSim Platform

Microservicio de gestión de billeteras virtuales con soporte para transacciones ACID.

## 🚀 Quick Start (Recommended)

```bash
# Desde la raíz del proyecto
cd /ads2-ProyectoFinal-2025
make up              # Levantar todos los servicios
# O:
make up-wallet       # Levantar solo Wallet API + dependencias
```

**URLs del servicio:**
- **Wallet API**: http://localhost:8006
- **Health Check**: http://localhost:8006/health

**Ver logs:**
```bash
make logs-wallet
```

---

## 🏗️ Arquitectura & Dependencias

### Dependencias:
- **MongoDB 7.0** (`wallet-mongo`) - Base de datos transaccional
- **Redis** (`shared-redis`) - Cache y locks distribuidos
- **RabbitMQ** (`shared-rabbitmq`) - Eventos de transacciones

### Comunica con:
- **Users API** (http://users-api:8001) - Validación de usuarios
- **Orders API** (http://orders-api:8080) - Lock/release de fondos

### Es consumido por:
- Orders API (verificación de saldo, lock/release)
- Admin Panel (depósitos, retiros manuales)

---

## ⚡ Características

- **Transacciones ACID**: Garantías de atomicidad para operaciones
- **Lock de Fondos**: Sistema de reserva para órdenes pendientes
- **Historial Completo**: Tracking de todas las transacciones
- **Balance Separado**: Available vs Locked balance
- **Auditoría**: Log de todas las operaciones para compliance
- **Concurrencia Segura**: Manejo de race conditions con locks distribuidos

## 📊 Endpoints Principales

### Obtener Wallet
```http
GET /api/wallet/:userId
Authorization: Bearer {jwt_token}
```

Respuesta:
```json
{
  "user_id": 1,
  "available_balance": 98500.00,
  "locked_balance": 1500.00,
  "total_balance": 100000.00,
  "currency": "USD",
  "last_transaction": "2025-10-12T15:30:00Z"
}
```

### Obtener Solo Balance
```http
GET /api/wallet/:userId/balance
Authorization: Bearer {jwt_token}
```

### Historial de Transacciones
```http
GET /api/wallet/:userId/transactions?limit=50&offset=0&type=all
Authorization: Bearer {jwt_token}
```

Tipos de transacciones:
- `deposit` - Depósito de fondos
- `withdrawal` - Retiro de fondos
- `order_lock` - Lock de fondos para orden
- `order_release` - Release de lock (orden cancelada)
- `order_execute` - Ejecución de orden

### Depositar Fondos (Admin)
```http
POST /api/wallet/:userId/deposit
Authorization: Bearer {admin_jwt_token}
Content-Type: application/json

{
  "amount": 10000.00,
  "description": "Initial deposit",
  "reference": "admin_action_001"
}
```

### Retirar Fondos
```http
POST /api/wallet/:userId/withdraw
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "amount": 5000.00,
  "description": "Withdrawal to bank"
}
```

## 🔒 Endpoints Internos (Service-to-Service)

### Lock de Fondos (usado por Orders API)
```http
POST /api/wallet/:userId/lock
X-Internal-Service: orders-api
X-API-Key: {internal_api_key}
Content-Type: application/json

{
  "amount": 1500.00,
  "order_id": "order_12345",
  "description": "Lock for BTC purchase"
}
```

### Release de Lock
```http
POST /api/wallet/:userId/release
X-Internal-Service: orders-api
X-API-Key: {internal_api_key}
Content-Type: application/json

{
  "amount": 1500.00,
  "order_id": "order_12345"
}
```

### Ejecutar Orden (deducir fondos)
```http
POST /api/wallet/:userId/execute-order
X-Internal-Service: orders-api
X-API-Key: {internal_api_key}
Content-Type: application/json

{
  "amount": 1500.00,
  "order_id": "order_12345",
  "type": "buy"
}
```

## 🔧 Variables de Entorno

```env
GO_ENV=production
API_PORT=8080

# MongoDB
MONGODB_URI=mongodb://wallet-mongo:27017/cryptosim_wallet

# Redis
REDIS_HOST=shared-redis
REDIS_PORT=6379

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@shared-rabbitmq:5672/

# JWT
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# External APIs
USERS_API_URL=http://users-api:8001
ORDERS_API_URL=http://orders-api:8080

# Internal Auth
INTERNAL_API_KEY=internal-secret-key
```

## 💾 Modelo de Datos

### Wallet
```javascript
{
  "user_id": 1,
  "available_balance": 98500.00,
  "locked_balance": 1500.00,
  "total_balance": 100000.00,
  "currency": "USD",
  "created_at": ISODate("2025-01-01T00:00:00Z"),
  "updated_at": ISODate("2025-10-12T15:30:00Z")
}
```

### Transaction
```javascript
{
  "wallet_id": ObjectId("..."),
  "user_id": 1,
  "type": "order_lock",
  "amount": 1500.00,
  "balance_before": 100000.00,
  "balance_after": 98500.00,
  "reference_type": "order",
  "reference_id": "order_12345",
  "description": "Lock for BTC purchase",
  "timestamp": ISODate("2025-10-12T15:30:00Z")
}
```

## 🧪 Testing

```bash
cd wallet-api
go test ./...

# Tests de integración con MongoDB
go test ./tests/integration/...
```

## 🐛 Troubleshooting

### Balance inconsistente
```bash
# Ver últimas transacciones
curl http://localhost:8006/api/wallet/1/transactions?limit=10

# Verificar logs
make logs-wallet
```

### Lock no se libera
- Verificar que Orders API llamó al endpoint de release
- Revisar RabbitMQ: `open http://localhost:15672`
- Ver transacciones pendientes en MongoDB

### MongoDB error de transacciones
- Verificar que MongoDB está en modo replica set (requerido para transacciones)
- Ver logs: `make logs-mongo`

## 📚 Documentación

- [README Principal](../README.md)
- [QUICKSTART](../QUICKSTART.md)

---

**Wallet API** - Parte del ecosistema CryptoSim 🚀
