# 🔍 Search API - CryptoSim Platform

Microservicio de búsqueda de criptomonedas con Apache Solr, cache distribuido y filtros avanzados.

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

### Es consumido por:
- Frontend (búsqueda de criptomonedas)
- Trading interface (selección de activos)

**Documentación completa**: Ver [README principal](../README.md)

---

## ⚡ Características

- **Búsqueda Full-Text**: Motor Solr con tokenización avanzada
- **Filtros Complejos**: Por precio, volumen, categoría, trending score
- **Cache Multinivel**: CCache local + Memcached distribuido
- **Trending Detection**: Algoritmo de detección de criptos en tendencia
- **Faceted Search**: Búsqueda por facetas (categorías, rangos)
- **Paginación**: Resultados paginados con offset/limit

## 📊 Endpoints Principales

### Buscar Criptomonedas
```http
GET /api/search/cryptos?q=bitcoin&limit=20&offset=0
Content-Type: application/json
```

**Parámetros:**
- `q` (string): Término de búsqueda
- `category` (string): Filtrar por categoría (defi, nft, stablecoin, etc)
- `min_price` (float): Precio mínimo
- `max_price` (float): Precio máximo
- `min_volume` (int): Volumen 24h mínimo
- `sort` (string): Campo de ordenamiento (price, volume, market_cap)
- `limit` (int): Resultados por página (default: 20)
- `offset` (int): Offset para paginación (default: 0)

### Criptomonedas en Tendencia
```http
GET /api/search/cryptos/trending?limit=10
```

### Obtener Filtros Disponibles
```http
GET /api/search/cryptos/filters
```

Respuesta:
```json
{
  "categories": ["defi", "nft", "stablecoin", "exchange", "gaming"],
  "price_ranges": [
    {"min": 0, "max": 1, "count": 245},
    {"min": 1, "max": 100, "count": 520},
    {"min": 100, "max": 1000, "count": 89}
  ],
  "volume_ranges": [...]
}
```

### Reindexar (Admin)
```http
POST /api/search/reindex
Authorization: Bearer {admin_jwt_token}
```

## 🔧 Variables de Entorno

Ver [`.env.example`](../.env.example) en la raíz del proyecto.

Principales variables:
```env
# Solr
SOLR_BASE_URL=http://solr:8983/solr
SOLR_COLLECTION=crypto_search

# Memcached
CACHE_MEMCACHED_HOSTS=memcached:11211

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@shared-rabbitmq:5672/
RABBITMQ_ENABLED=true

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

El schema de Solr define los campos indexados:

```xml
<field name="symbol" type="string" indexed="true" stored="true"/>
<field name="name" type="string" indexed="true" stored="true"/>
<field name="current_price" type="pdouble" indexed="true" stored="true"/>
<field name="market_cap" type="plong" indexed="true" stored="true"/>
<field name="volume_24h" type="plong" indexed="true" stored="true"/>
<field name="price_change_24h" type="pdouble" indexed="true" stored="true"/>
<field name="category" type="string" indexed="true" stored="true" multiValued="true"/>
<field name="trending_score" type="pint" indexed="true" stored="true"/>
<field name="is_active" type="boolean" indexed="true" stored="true"/>
```

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

### Reindexar datos
```bash
# Si la colección está vacía o corrupta
curl -X POST http://localhost:8003/api/search/reindex \
  -H "Authorization: Bearer {admin_token}"
```

## 📚 Documentación Adicional

- [README Principal](../README.md) - Documentación completa del proyecto
- [QUICKSTART](../QUICKSTART.md) - Guía de inicio rápido
- [Solr Admin UI](http://localhost:8983/solr) - Interfaz de administración (cuando está corriendo)

---

**Search API** - Parte del ecosistema de microservicios CryptoSim 🚀
