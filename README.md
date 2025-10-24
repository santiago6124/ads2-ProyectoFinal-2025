# 🚀 CryptoSim - Plataforma de Simulación de Trading

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-20.10+-2496ED?style=flat&logo=docker)](https://www.docker.com)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Plataforma educativa de simulación de trading de criptomonedas con arquitectura de microservicios.

## 📋 Tabla de Contenidos

- [Descripción](#-descripción)
- [Arquitectura](#-arquitectura)
- [Tecnologías](#-tecnologías)
- [Inicio Rápido](#-inicio-rápido)
- [Servicios](#-servicios)
- [Comandos Útiles](#-comandos-útiles)
- [Desarrollo](#-desarrollo)
- [Testing](#-testing)
- [Troubleshooting](#-troubleshooting)

## 📖 Descripción

CryptoSim permite a los usuarios aprender y practicar estrategias de trading de criptomonedas en un entorno simulado, sin riesgo financiero real. Los usuarios reciben un saldo virtual y pueden operar con precios de mercado reales.

### ✨ Características principales

- 🔐 **Autenticación JWT** - Sistema seguro de usuarios
- 💰 **Balance Integrado** - Gestión de saldo directamente en Users API
- 📊 **Trading en Tiempo Real** - Órdenes de compra/venta con datos reales
- 📈 **Gestión de Portafolio** - Seguimiento de inversiones y rendimiento
- 🔍 **Búsqueda Avanzada** - Motor de búsqueda con Apache Solr
- 📉 **Datos de Mercado** - Precios actualizados de criptomonedas
- 🏆 **Sistema de Rankings** - Leaderboards y estadísticas
- 📧 **Notificaciones** - Alertas en tiempo real
- 📝 **Auditoría** - Registro completo de operaciones

> **Nota**: Este proyecto utiliza una arquitectura simplificada para fines educativos. El balance USD se gestiona directamente en Users API en lugar de usar un microservicio separado de Wallet.

## 🏗️ Arquitectura

```
┌─────────────────────────────────────────────────────────┐
│                    Frontend (React)                      │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│                   API Gateway                            │
└──┬────┬────┬────┬────┬─────────────────────────────────┘
   │    │    │    │    │
   ▼    ▼    ▼    ▼    ▼
┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
│Users │ │Orders│ │Search│ │Market│ │Port- │
│ API  │ │ API  │ │ API  │ │Data  │ │folio │
│      │ │      │ │      │ │ API  │ │ API  │
└──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘
   │        │        │        │        │
   ▼        ▼        ▼        ▼        ▼
┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
│MySQL │ │MongoDB│ │Solr  │ │Redis │ │MongoDB│
└──────┘ └──────┘ └──────┘ └──────┘ └──────┘

        ┌─────────────────────────────┐
        │    Shared Infrastructure    │
        │  Redis | RabbitMQ | Solr    │
        └─────────────────────────────┘
```

### Microservicios

| Servicio | Puerto | Base de Datos | Descripción |
|----------|--------|---------------|-------------|
| **Users API** | 8001 | MySQL | Autenticación y gestión de usuarios |
| **Orders API** | 8002 | MongoDB | Órdenes de compra/venta |
| **Search API** | 8003 | Solr | Búsqueda de criptomonedas |
| **Market Data API** | 8004 | Redis | Precios en tiempo real |
| **Portfolio API** | 8005 | MongoDB | Gestión de portafolios |

## 🛠️ Tecnologías

### Backend
- **Go 1.21+** - Lenguaje principal
- **Gin** - Framework HTTP
- **GORM** - ORM para MySQL
- **MongoDB Driver** - Cliente oficial de MongoDB

### Bases de Datos
- **MySQL 8.0** - Base de datos relacional
- **MongoDB 7.0** - Base de datos NoSQL
- **Redis 7** - Cache distribuido
- **Apache Solr 9** - Motor de búsqueda

### Mensajería y Cache
- **RabbitMQ 3.12** - Message broker
- **Memcached 1.6** - Cache distribuido

### Monitoring (Opcional)
- **Prometheus** - Métricas
- **Grafana** - Dashboards

### DevOps
- **Docker** - Containerización
- **Docker Compose** - Orquestación

## 🚀 Inicio Rápido

### Prerrequisitos

- [Docker](https://docs.docker.com/get-docker/) 20.10+
- [Docker Compose](https://docs.docker.com/compose/install/) 2.0+
- 8GB RAM mínimo
- 20GB espacio en disco

### Instalación

1. **Clonar el repositorio**
   ```bash
   git clone <repository-url>
   cd ads2-ProyectoFinal-2025
   ```

2. **Configurar variables de entorno**
   ```bash
   cp .env.example .env
   # Editar .env con tus valores
   ```

3. **Levantar todos los servicios**
   ```bash
   make up
   # O usar docker-compose directamente:
   # docker-compose up -d
   ```

4. **Verificar que todo está funcionando**
   ```bash
   make status
   # O verificar salud de servicios:
   make health
   ```

5. **Ver logs**
   ```bash
   make logs
   # O logs de un servicio específico:
   make logs-users
   ```

### 🎉 ¡Listo!

Los servicios estarán disponibles en:

- **Users API**: http://localhost:8001
- **Orders API**: http://localhost:8002
- **Search API**: http://localhost:8003
- **Market Data API**: http://localhost:8004
- **Portfolio API**: http://localhost:8005
- **RabbitMQ Management**: http://localhost:15672 (guest/guest)

## 📚 Servicios

### Users API (Puerto 8001)

Gestión de usuarios, autenticación y autorización.

**Endpoints principales:**
```
POST   /api/users/register      - Registrar usuario
POST   /api/users/login         - Login
GET    /api/users/:id           - Obtener usuario
PUT    /api/users/:id           - Actualizar usuario
POST   /api/users/:id/upgrade   - Convertir a admin
```

**Tecnologías:**
- Go + Gin
- MySQL (GORM)
- Redis (cache)
- JWT

### Orders API (Puerto 8002)

Gestión de órdenes de trading con ejecución concurrente.

**Endpoints principales:**
```
POST   /api/orders              - Crear orden
GET    /api/orders/:id          - Obtener orden
GET    /api/orders/user/:userId - Listar órdenes
POST   /api/orders/:id/execute  - Ejecutar orden
DELETE /api/orders/:id          - Cancelar orden
```

**Características:**
- Ejecución concurrente con goroutines
- Cálculo de fees y slippage
- Integración con RabbitMQ
- Comunicación con Users y Market Data APIs
- Gestión directa de balance desde Users API

### Search API (Puerto 8003)

Motor de búsqueda de criptomonedas con Apache Solr.

**Endpoints principales:**
```
GET    /api/search/cryptos           - Buscar criptomonedas
GET    /api/search/cryptos/trending  - Trending
GET    /api/search/cryptos/filters   - Filtros disponibles
POST   /api/search/reindex           - Reindexar (admin)
```

**Características:**
- Búsqueda full-text
- Filtros por categoría, precio, volumen
- Cache con Memcached y CCache
- Consumidor de RabbitMQ

### Market Data API (Puerto 8004)

Datos de mercado en tiempo real.

**Endpoints principales:**
```
GET    /api/market/price/:symbol  - Precio actual
GET    /api/market/prices         - Múltiples precios
GET    /api/market/history/:symbol - Histórico
GET    /api/market/stats/:symbol  - Estadísticas
WS     /api/market/stream         - Stream en tiempo real
```

**Características:**
- Integración con CoinGecko y Binance
- Cache en Redis con TTL corto
- WebSockets para updates en tiempo real

### Portfolio API (Puerto 8005)

Gestión y cálculo de portafolios de inversión.

**Endpoints principales:**
```
GET    /api/portfolio/:userId              - Portafolio completo
GET    /api/portfolio/:userId/performance  - Métricas
GET    /api/portfolio/:userId/history      - Histórico
GET    /api/portfolio/:userId/holdings     - Holdings
POST   /api/portfolio/:userId/snapshot     - Crear snapshot
```

**Características:**
- Cálculo automático de P&L
- Métricas de rendimiento
- Scheduler para actualizaciones periódicas
- Consumer de RabbitMQ para eventos de órdenes
- Integración con Users API para balance USD

## 🎮 Comandos Útiles

### Gestión de servicios

```bash
make up              # Levantar todos los servicios
make down            # Detener todos los servicios
make restart         # Reiniciar servicios
make build           # Construir imágenes
make rebuild         # Reconstruir sin cache
```

### Logs

```bash
make logs            # Ver todos los logs
make logs-users      # Logs del Users API
make logs-orders     # Logs del Orders API
make logs-search     # Logs del Search API
make logs-market     # Logs del Market Data API
make logs-portfolio  # Logs del Portfolio API
```

### Monitoreo

```bash
make status          # Estado de servicios
make ps              # Contenedores activos
make health          # Health check de APIs
```

### Servicios individuales

```bash
make up-users        # Solo Users API + dependencias
make up-orders       # Solo Orders API + dependencias
make up-infra        # Solo infraestructura (DBs, Redis, etc)
```

### Monitoring

```bash
make monitoring-up   # Levantar Prometheus + Grafana
make monitoring-down # Detener monitoring
```

### Limpieza

```bash
make clean           # Limpiar contenedores y volúmenes
make clean-all       # Limpieza completa (incluye imágenes)
make prune           # Limpiar recursos no usados de Docker
```

### Utilidades

```bash
make env             # Crear .env desde .env.example
make shell-users     # Shell en Users API container
make shell-mysql     # MySQL CLI
make shell-mongo     # MongoDB CLI
make shell-redis     # Redis CLI
```

## 💻 Desarrollo

### Estructura del proyecto

```
.
├── users-api/          # Microservicio de usuarios
├── orders-api/         # Microservicio de órdenes
├── search-api/         # Microservicio de búsqueda
├── market-data-api/    # Microservicio de datos de mercado
├── portfolio-api/      # Microservicio de portafolios
├── docker-compose.yml  # Orquestación unificada
├── .env.example        # Variables de entorno
├── Makefile            # Comandos útiles
└── README.md           # Este archivo
```

### Agregar un nuevo servicio

1. Crear directorio del servicio
2. Agregar Dockerfile
3. Agregar configuración en `docker-compose.yml`
4. Configurar variables de entorno en `.env`
5. Agregar comandos en `Makefile` (opcional)

### Variables de entorno

Las principales variables están en [`.env.example`](.env.example):

```bash
# Security
JWT_SECRET=your-super-secret-key

# Databases
MYSQL_ROOT_PASSWORD=rootpassword
MONGO_PASSWORD=password

# External APIs
COINGECKO_API_KEY=your-api-key

# Monitoring
GRAFANA_PASSWORD=admin
```

## 🧪 Testing

```bash
# Todos los tests
make test

# Test individual por servicio
cd users-api && go test ./...
cd orders-api && go test ./...
```

## 🐛 Troubleshooting

### Los contenedores no inician

```bash
# Ver logs detallados
make logs

# Verificar estado
make ps

# Reconstruir desde cero
make clean
make rebuild
```

### Puertos ocupados

Edita `docker-compose.yml` o `.env` para cambiar los puertos externos:

```yaml
ports:
  - "8001:8001"  # Cambiar primer número (externo)
```

### Problemas de memoria

Los servicios requieren ~6-8GB RAM. Aumenta memoria de Docker:

- **Docker Desktop**: Settings → Resources → Memory → 8GB+

### Base de datos no conecta

```bash
# Verificar salud de bases de datos
docker-compose ps users-mysql orders-mongo

# Ver logs de la base de datos
make logs-mysql
make logs-mongo

# Recrear volúmenes
make clean
make up
```

### RabbitMQ no funciona

```bash
# Ver logs
make logs-rabbitmq

# Acceder a management UI
open http://localhost:15672

# Recrear contenedor
docker-compose restart shared-rabbitmq
```

### Solr no indexa

```bash
# Verificar Solr
curl http://localhost:8983/solr/admin/ping

# Ver logs
docker-compose logs solr

# Crear colección manualmente
docker-compose exec solr solr create -c crypto_search
```

## 📊 Monitoring

Para habilitar Prometheus y Grafana:

```bash
make monitoring-up
```

Acceder a:
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

## 🔒 Seguridad

### Recomendaciones para producción

1. **Cambiar todos los secrets** en `.env`
2. **Usar HTTPS** con certificados SSL
3. **Habilitar autenticación** en Redis y RabbitMQ
4. **Configurar firewall** para puertos
5. **Implementar rate limiting**
6. **Usar usuarios no-root** en containers (ya configurado)
7. **Escanear imágenes** con `docker scan`

## 📝 Licencia

MIT License - Ver [LICENSE](LICENSE) para más detalles

## 👥 Contribuir

1. Fork el proyecto
2. Crear feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit cambios (`git commit -m 'Add AmazingFeature'`)
4. Push a branch (`git push origin feature/AmazingFeature`)
5. Abrir Pull Request

## 📞 Soporte

- Documentación: [Docs](./docs/)
- Issues: [GitHub Issues](https://github.com/tu-repo/issues)

---

⭐ Si este proyecto te fue útil, dale una estrella en GitHub!
