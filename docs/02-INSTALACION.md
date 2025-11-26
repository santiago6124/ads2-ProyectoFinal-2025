# 02 - Guía de Instalación

## 📋 Tabla de Contenidos
- [Requisitos Previos](#requisitos-previos)
- [Instalación Rápida](#instalación-rápida)
- [Configuración](#configuración)
- [Verificación](#verificación)
- [Solución de Problemas](#solución-de-problemas)

---

## 💻 Requisitos Previos

### Software Necesario

1. **Docker Desktop** (recomendado) o Docker Engine + Docker Compose
   - **Versión**: 20.10+ (Docker) y 2.0+ (Docker Compose)
   - **Windows**: [Docker Desktop for Windows](https://docs.docker.com/desktop/install/windows-install/)
   - **Mac**: [Docker Desktop for Mac](https://docs.docker.com/desktop/install/mac-install/)
   - **Linux**: [Docker Engine](https://docs.docker.com/engine/install/) + [Docker Compose](https://docs.docker.com/compose/install/)

2. **Git** (para clonar el repositorio)
   - [Descargar Git](https://git-scm.com/downloads)

### Recursos de Sistema

| Recurso | Mínimo | Recomendado |
|---------|--------|-------------|
| RAM | 8 GB | 16 GB |
| CPU | 4 cores | 8 cores |
| Disco | 20 GB libres | 50 GB libres |

### Puertos Necesarios

Los siguientes puertos deben estar disponibles en tu sistema:

| Puerto | Servicio |
|--------|----------|
| 3000 | Frontend |
| 8001 | Users API |
| 8002 | Orders API |
| 8003 | Search API |
| 8004 | Market Data API |
| 8005 | Portfolio API |
| 3307 | MySQL |
| 6379 | Redis |
| 5672 | RabbitMQ |
| 15672 | RabbitMQ Management |
| 8983 | Apache Solr |
| 11211 | Memcached |
| 27017 | MongoDB (Orders) |
| 27018 | MongoDB (Portfolio) |

---

## 🚀 Instalación Rápida

### 1. Clonar el Repositorio

```bash
git clone https://github.com/tu-usuario/ads2-ProyectoFinal-2025.git
cd ads2-ProyectoFinal-2025
```

### 2. Configurar Variables de Entorno

**Windows (PowerShell)**:
```powershell
Copy-Item .env.example .env
```

**Linux/Mac**:
```bash
cp .env.example .env
```

**Editar .env** (opcional pero recomendado para producción):
```bash
# Con tu editor favorito
notepad .env      # Windows
nano .env         # Linux
vim .env          # Linux/Mac
code .env         # VSCode
```

**Variables críticas a cambiar**:
```env
# Seguridad (CAMBIAR en producción)
JWT_SECRET=tu-clave-super-secreta-de-64-caracteres-minimo-por-seguridad
INTERNAL_API_KEY=clave-secreta-para-comunicacion-interna

# Bases de datos
MYSQL_ROOT_PASSWORD=tu_password_mysql
MONGO_PASSWORD=tu_password_mongo
```

### 3. Levantar los Servicios

```bash
docker-compose up -d
```

Este comando:
- 📥 Descarga todas las imágenes necesarias (primera vez: ~10-15 minutos)
- 🏗️ Construye las imágenes de los microservicios
- 🚀 Levanta todos los contenedores en segundo plano
- 🔗 Crea la red `cryptosim-network`
- 💾 Crea los volúmenes de persistencia

**Progreso esperado**:
```
[+] Building 120.5s
[+] Running 15/15
 ✔ Network cryptosim-network               Created
 ✔ Volume cryptosim-users-mysql-data       Created
 ✔ Volume cryptosim-orders-mongo-data      Created
 ✔ Container cryptosim-redis               Started
 ✔ Container cryptosim-rabbitmq            Started
 ✔ Container cryptosim-users-mysql         Started
 ✔ Container cryptosim-orders-mongo        Started
 ✔ Container cryptosim-portfolio-mongo     Started
 ✔ Container cryptosim-solr                Started
 ✔ Container cryptosim-memcached           Started
 ✔ Container cryptosim-users-api           Started
 ✔ Container cryptosim-market-data-api     Started
 ✔ Container cryptosim-orders-api          Started
 ✔ Container cryptosim-search-api          Started
 ✔ Container cryptosim-portfolio-api       Started
```

### 4. Esperar a que los Servicios Estén Listos

Los servicios tardan en iniciarse completamente (especialmente las bases de datos).

**Ver estado**:
```bash
docker-compose ps
```

Deberías ver todos los servicios con estado `Up (healthy)`:
```
NAME                        STATUS
cryptosim-frontend          Up
cryptosim-users-api         Up (healthy)
cryptosim-orders-api        Up (healthy)
cryptosim-search-api        Up (healthy)
cryptosim-market-data-api   Up (healthy)
cryptosim-portfolio-api     Up (healthy)
...
```

**Ver logs en tiempo real**:
```bash
# Todos los servicios
docker-compose logs -f

# Solo un servicio específico
docker-compose logs -f users-api
```

---

## ⚙️ Configuración

### Archivo .env

El archivo `.env` contiene todas las configuraciones del sistema.

#### Secciones Principales

**1. Seguridad**:
```env
# JWT para autenticación
JWT_SECRET=your-super-secret-jwt-key-change-in-production-use-64-chars-minimum

# API Key interna para comunicación entre servicios
INTERNAL_API_KEY=internal-secret-key-change-in-production
```

**2. Bases de Datos**:
```env
# MySQL (Users API)
MYSQL_ROOT_PASSWORD=rootpassword
MYSQL_PASSWORD=password

# MongoDB (Orders, Portfolio)
MONGO_PASSWORD=password
```

**3. APIs Externas** (opcional):
```env
# CoinGecko (para datos de mercado)
COINGECKO_API_KEY=

# Binance (opcional)
BINANCE_API_KEY=
BINANCE_API_SECRET=
```

**4. Features**:
```env
# Habilitar/deshabilitar funcionalidades
SCHEDULER_ENABLED=true
RABBITMQ_ENABLED=true
CORS_ENABLED=true
```

**5. Performance**:
```env
# Pool de conexiones MongoDB
MONGODB_MAX_POOL_SIZE=100
MONGODB_MIN_POOL_SIZE=10

# Workers RabbitMQ
RABBITMQ_WORKER_COUNT=5
RABBITMQ_PREFETCH_COUNT=10
```

### Docker Compose

Para cambiar puertos externos, edita `docker-compose.yml`:

```yaml
services:
  users-api:
    ports:
      - "8001:8001"  # <puerto_externo>:<puerto_interno>
```

---

## ✅ Verificación

### 1. Health Checks

Verificar que todos los servicios respondan correctamente:

```bash
# Users API
curl http://localhost:8001/health

# Orders API
curl http://localhost:8002/health

# Search API
curl http://localhost:8003/api/v1/health

# Market Data API
curl http://localhost:8004/health

# Portfolio API
curl http://localhost:8005/health
```

**Respuesta esperada**:
```json
{
  "status": "healthy",
  "service": "users-api",
  "timestamp": "2025-11-14T15:30:00Z"
}
```

### 2. Bases de Datos

**MySQL**:
```bash
docker exec -it cryptosim-users-mysql mysql -uroot -p
# Password: el de MYSQL_ROOT_PASSWORD en .env
mysql> SHOW DATABASES;
mysql> USE users_db;
mysql> SHOW TABLES;
```

**MongoDB (Orders)**:
```bash
docker exec -it cryptosim-orders-mongo mongosh
> show dbs
> use cryptosim_orders
> show collections
```

**Redis**:
```bash
docker exec -it cryptosim-redis redis-cli
127.0.0.1:6379> PING
PONG
```

### 3. RabbitMQ Management

Abrir navegador: http://localhost:15672

- **Usuario**: `guest`
- **Password**: `guest`

Verificar:
- Exchanges: `orders.events`, `balance.request.exchange`, etc.
- Queues: `portfolio.updates`, `search.sync`, etc.

### 4. Apache Solr

Abrir navegador: http://localhost:8983/solr

Verificar:
- Collection `orders_search` existe
- Schema configurado correctamente

### 5. Prueba End-to-End

**Script de verificación** (incluido en el repo):

**Windows**:
```powershell
.\verify-services.ps1
```

**Linux/Mac**:
```bash
chmod +x verify-services.sh
./verify-services.sh
```

**Prueba manual completa**:

```bash
# 1. Registrar usuario
curl -X POST http://localhost:8001/api/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@test.com",
    "password": "test123456",
    "username": "testuser"
  }'

# 2. Login
TOKEN=$(curl -X POST http://localhost:8001/api/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@test.com",
    "password": "test123456"
  }' | jq -r '.access_token')

# 3. Ver balance
curl -X GET http://localhost:8001/api/users/me \
  -H "Authorization: Bearer $TOKEN"

# 4. Ver precios
curl http://localhost:8004/api/v1/prices/bitcoin

# 5. Crear orden de compra
ORDER_ID=$(curl -X POST http://localhost:8002/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "crypto_symbol": "BTC",
    "quantity": 0.001,
    "order_type": "buy",
    "order_kind": "market"
  }' | jq -r '.id')

# 6. Ejecutar orden
curl -X POST http://localhost:8002/api/v1/orders/$ORDER_ID/execute \
  -H "Authorization: Bearer $TOKEN"

# 7. Ver portfolio
curl -X GET http://localhost:8005/api/portfolios/1 \
  -H "Authorization: Bearer $TOKEN"
```

---

## 🔧 Solución de Problemas

### Problema: "Port already allocated"

**Causa**: El puerto ya está en uso por otra aplicación.

**Solución 1** - Cerrar aplicación que usa el puerto:
```bash
# Windows
netstat -ano | findstr :8001
taskkill /PID <PID> /F

# Linux/Mac
lsof -i :8001
kill -9 <PID>
```

**Solución 2** - Cambiar puerto en docker-compose.yml:
```yaml
ports:
  - "8101:8001"  # Usa puerto externo 8101
```

---

### Problema: "Cannot connect to database"

**Causa**: La base de datos aún no está lista.

**Solución**:
```bash
# Ver logs de la base de datos
docker-compose logs mysql
docker-compose logs orders-mongo

# Esperar 30-60 segundos y reintentar
docker-compose restart users-api
```

---

### Problema: "Out of memory"

**Causa**: Docker no tiene suficiente RAM asignada.

**Solución en Docker Desktop**:
1. Abrir Docker Desktop
2. Settings → Resources → Memory
3. Aumentar a mínimo 8GB
4. Apply & Restart

---

### Problema: Contenedores se caen constantemente

**Diagnóstico**:
```bash
# Ver logs del servicio problemático
docker-compose logs <servicio>

# Ver últimas 50 líneas
docker-compose logs --tail=50 <servicio>

# Ver en tiempo real
docker-compose logs -f <servicio>
```

**Soluciones comunes**:
```bash
# Reconstruir imágenes
docker-compose down
docker-compose build --no-cache
docker-compose up -d

# Limpiar volúmenes (BORRA DATOS)
docker-compose down -v
docker-compose up -d
```

---

### Problema: "Service unhealthy"

**Diagnóstico**:
```bash
# Verificar health check
docker inspect cryptosim-users-api | grep -A 10 Health

# Ejecutar health check manualmente
docker exec cryptosim-users-api wget --spider http://localhost:8001/health
```

**Solución**:
```bash
# Reiniciar servicio
docker-compose restart users-api

# Si persiste, verificar logs
docker-compose logs users-api
```

---

### Problema: Solr no indexa órdenes

**Diagnóstico**:
```bash
# Verificar schema de Solr
curl http://localhost:8983/solr/orders_search/schema

# Ver logs de Search API
docker-compose logs search-api

# Verificar RabbitMQ
# Abrir http://localhost:15672
# Verificar que queue search.sync recibe mensajes
```

**Solución**:
```bash
# Re-inicializar Solr
docker-compose restart solr

# Esperar 30 segundos
sleep 30

# Reiniciar Search API
docker-compose restart search-api
```

---

### Problema: RabbitMQ no conecta

**Diagnóstico**:
```bash
# Ver logs de RabbitMQ
docker-compose logs rabbitmq

# Verificar que esté corriendo
docker-compose ps rabbitmq
```

**Solución**:
```bash
# Reiniciar RabbitMQ
docker-compose restart rabbitmq

# Esperar a que esté healthy
docker-compose ps rabbitmq

# Reiniciar servicios que lo usan
docker-compose restart users-api orders-api search-api portfolio-api
```

---

### Limpieza Completa (Último Recurso)

⚠️ **ADVERTENCIA**: Esto borra TODOS los datos.

```bash
# Detener y eliminar todo
docker-compose down -v

# Limpiar imágenes y cache de Docker
docker system prune -a --volumes

# Volver a empezar
docker-compose up -d --build
```

---

## 🎯 Próximos Pasos

Instalación completada exitosamente. Ahora puedes:

1. **[Leer la Guía de Uso](03-GUIA-USO.md)** - Aprender a usar la plataforma
2. **[Explorar la Arquitectura](04-ARQUITECTURA-GENERAL.md)** - Entender cómo funciona
3. **[Ver Endpoints](08-USERS-ENDPOINTS.md)** - Referencia completa de APIs
4. **[Probar con Postman](https://www.postman.com/)** - Importar colección de requests

---

**Siguiente**: [03 - Guía de Uso](03-GUIA-USO.md) →
