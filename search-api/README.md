# 🔍 Search API - CryptoSim Platform

Microservicio de búsqueda de órdenes con Apache Solr, cache distribuido (CCache + Memcached) y sincronización automática vía RabbitMQ.

## 🚀 Quick Start (Recommended)

**Este servicio es parte del ecosistema CryptoSim.** La forma recomendada de ejecutarlo es usando el **Docker Compose unificado** en la raíz:

```bash
# Desde la raíz del proyecto
cd /ads2-ProyectoFinal-2025
make up              # Levantar todos los servicios
# O:
make up-search       # Levantar solo Search API + dependencias
```

**URLs del servicio:**
- **Search API**: http://localhost:8003
- **Health Check**: http://localhost:8003/api/v1/health
- **Solr Admin**: http://localhost:8983/solr

**Ver logs:**
```bash
make logs-search
```

**Acceder al contenedor:**
```bash
make shell-search
```

---

## 🏗️ Arquitectura & Dependencias

### Dependencias requeridas:
- **Apache Solr 9** (`solr` container) - Motor de búsqueda
- **Memcached** (`memcached` container) - Cache distribuido
- **RabbitMQ** (`shared-rabbitmq` container) - Message broker

### Cache en dos niveles:
1. **CCache** (local) - Cache en memoria del proceso
2. **Memcached** (distribuido) - Cache compartido entre instancias

### Comunica con:
- **Orders API** (http://orders-api:8080) - Obtiene detalles completos de órdenes para indexación
- **RabbitMQ** - Consume eventos de órdenes (created, executed, cancelled, failed) para sincronización automática

### Es consumido por:
- Frontend (búsqueda de órdenes)
- Trading interface (historial y filtrado de órdenes)

**Documentación completa**: Ver [README principal](../README.md)

---

## ⚡ Características

- **Búsqueda Full-Text**: Motor Solr con tokenización avanzada sobre órdenes
- **Filtros Complejos**: Por status, tipo (buy/sell), order_kind (market/limit), crypto_symbol, monto total, fechas
- **Cache Multinivel**: CCache local + Memcached distribuido para consultas frecuentes
- **Sincronización Automática**: Consumer de RabbitMQ que indexa órdenes automáticamente cuando se crean/actualizan
- **Consistencia de Datos**: Invoca Orders API para obtener detalles completos antes de indexar
- **Faceted Search**: Búsqueda por facetas (status, type, order_kind, crypto_symbol)
- **Paginación**: Resultados paginados con page/limit

## 📊 Endpoints Principales

### Buscar Órdenes
```http
POST /api/v1/search
Content-Type: application/json

{
  "q": "BTC",
  "page": 1,
  "limit": 20,
  "sort": "created_at_desc",
  "status": ["executed", "pending"],
  "type": ["buy"],
  "order_kind": ["market"],
  "crypto_symbol": ["BTC", "ETH"],
  "min_total_amount": 100.0,
  "max_total_amount": 10000.0,
  "date_from": "2025-01-01T00:00:00Z",
  "date_to": "2025-01-31T23:59:59Z"
}
```

**Parámetros:**
- `q` (string): Término de búsqueda (busca en crypto_symbol, crypto_name, order_id)
- `page` (int): Número de página (default: 1)
- `limit` (int): Resultados por página (default: 20, max: 100)
- `sort` (string): Ordenamiento (created_at_desc, total_amount_desc, price_asc, etc.)
- `status` (array): Filtrar por status (pending, executed, cancelled, failed)
- `type` (array): Filtrar por tipo (buy, sell)
- `order_kind` (array): Filtrar por tipo de orden (market, limit)
- `crypto_symbol` (array): Filtrar por símbolo (BTC, ETH, etc.)
- `user_id` (int): Filtrar por ID de usuario
- `min_total_amount` (float): Monto total mínimo
- `max_total_amount` (float): Monto total máximo
- `date_from` (string): Fecha desde (ISO 8601)
- `date_to` (string): Fecha hasta (ISO 8601)

### Obtener Orden por ID
```http
GET /api/v1/orders/:id
```

### Obtener Filtros Disponibles
```http
GET /api/v1/filters
```

Respuesta:
```json
{
  "statuses": [
    {"value": "pending", "label": "Pending", "count": 45},
    {"value": "executed", "label": "Executed", "count": 120}
  ],
  "types": [
    {"value": "buy", "label": "Buy", "count": 85},
    {"value": "sell", "label": "Sell", "count": 80}
  ],
  "order_kinds": [
    {"value": "market", "label": "Market Orders", "count": 100},
    {"value": "limit", "label": "Limit Orders", "count": 65}
  ],
  "crypto_symbols": [
    {"value": "BTC", "label": "BTC", "count": 50},
    {"value": "ETH", "label": "ETH", "count": 40}
  ],
  "sort_options": [...]
}
```

## 🔧 Variables de Entorno

Ver [`.env.example`](../.env.example) en la raíz del proyecto.

Principales variables:
```env
# Solr
SOLR_BASE_URL=http://solr:8983/solr
SOLR_COLLECTION=orders_search

# Memcached
CACHE_MEMCACHED_HOSTS=memcached:11211

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@shared-rabbitmq:5672/
RABBITMQ_ENABLED=true
RABBITMQ_EXCHANGE_NAME=orders.events
RABBITMQ_QUEUE_NAME=search.sync
RABBITMQ_ROUTING_KEYS=orders.created,orders.executed,orders.cancelled,orders.failed

# Orders API (para obtener detalles completos de órdenes)
ORDERS_API_BASE_URL=http://orders-api:8080
ORDERS_API_KEY=internal-secret-key
ORDERS_API_TIMEOUT_MS=10000

# Server
SERVER_PORT=8080
ENVIRONMENT=development
LOG_LEVEL=info
```

## 🧪 Testing

```bash
cd search-api

# Unit tests
go test ./internal/...

# Integration tests (requiere Solr)
go test ./tests/integration/...

# Con coverage
go test -cover ./...
```

## 🛠️ Desarrollo Local

Para desarrollo sin Docker:

```bash
cd search-api

# Instalar dependencias
go mod download

# Ejecutar (requiere Solr y Memcached externos)
go run cmd/server/main.go
```

## 🗂️ Schema de Solr

El schema de Solr define los campos indexados para órdenes:

**Campos principales:**
- `id` (string): ID único de la orden (MongoDB ObjectID)
- `user_id` (int): ID del usuario propietario
- `type` (string): Tipo de orden (buy, sell)
- `status` (string): Estado (pending, executed, cancelled, failed)
- `order_kind` (string): Tipo de orden (market, limit)
- `crypto_symbol` (string): Símbolo de la criptomoneda (BTC, ETH, etc.)
- `crypto_name` (string): Nombre completo de la criptomoneda
- `quantity_s` / `quantity_d` (string/double): Cantidad
- `price_s` / `price_d` (string/double): Precio
- `total_amount_display_s` / `total_amount_value_d` (string/double): Monto total
- `fee_s` / `fee_d` (string/double): Comisión
- `created_at`, `updated_at`, `executed_at`, `cancelled_at` (date): Fechas
- `search_text` (text): Campo de búsqueda full-text

Los campos con sufijo `_s` son strings (para display), los `_d` son doubles (para ordenamiento y filtrado numérico).

## 🐛 Troubleshooting

### Solr no responde
```bash
# Verificar que Solr está corriendo
docker-compose ps solr

# Ver logs
docker-compose logs solr

# Probar conexión
curl http://localhost:8983/solr/admin/ping
```

### Cache no funciona
```bash
# Verificar Memcached
docker-compose ps memcached

# Probar conexión
telnet localhost 11211
```

### Sincronización automática
El servicio se sincroniza automáticamente con Orders API mediante RabbitMQ:
- Cuando se crea una orden → se indexa en Solr
- Cuando se ejecuta una orden → se actualiza el índice
- Cuando se cancela una orden → se elimina del índice
- Cuando falla una orden → se actualiza el estado en el índice

Para verificar la sincronización:
```bash
# Ver logs del consumer
make logs-search | grep "RabbitMQ"

# Verificar eventos en RabbitMQ
open http://localhost:15672  # guest/guest
```

## 📚 Documentación Adicional

- [README Principal](../README.md) - Documentación completa del proyecto
- [QUICKSTART](../QUICKSTART.md) - Guía de inicio rápido
- [Solr Admin UI](http://localhost:8983/solr) - Interfaz de administración (cuando está corriendo)

---

**Search API** - Parte del ecosistema de microservicios CryptoSim 🚀
