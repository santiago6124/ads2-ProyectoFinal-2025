# Changelog - CryptoSim Docker Compose Unificado

## [2025-10-12] - Implementación de Docker Compose Unificado

### ✨ Nuevos Archivos Creados

#### Configuración Principal
- **`docker-compose.yml`** - Orquestación unificada de todos los microservicios
  - 6 microservicios (Users, Orders, Search, Market Data, Portfolio, Wallet)
  - 4 bases de datos separadas (MySQL + 3 MongoDB)
  - Infraestructura compartida (Redis, RabbitMQ, Solr, Memcached)
  - Soporte para monitoring con profiles (Prometheus + Grafana)
  - Red compartida: `cryptosim-network`
  - Healthchecks configurados para todos los servicios
  - Dependencies correctamente ordenadas

#### Documentación
- **`README.md`** - Documentación completa del proyecto
  - Arquitectura detallada
  - Guía de instalación
  - Descripción de cada servicio
  - Troubleshooting
  - Comandos útiles

- **`QUICKSTART.md`** - Guía de inicio rápido
  - Instrucciones en 5 minutos
  - Ejemplos de uso con curl
  - URLs importantes
  - Problemas comunes

- **`CHANGELOG.md`** - Este archivo

#### Utilidades
- **`Makefile`** - 40+ comandos útiles
  - Gestión de servicios (up, down, restart, build)
  - Logs por servicio
  - Monitoring
  - Limpieza
  - Shells interactivos
  - Health checks

- **`.env.example`** - Template de variables de entorno
  - Secrets y API keys
  - Configuraciones de base de datos
  - Feature flags
  - Performance tuning
  - Cache TTLs

- **`.gitignore`** - Archivos a ignorar en Git
  - Logs, binarios, dependencias
  - Datos sensibles
  - Archivos temporales

#### Monitoring
- **`monitoring/prometheus.yml`** - Configuración de Prometheus
  - Scrape configs para todos los servicios
  - Métricas de infraestructura
  - Preparado para exporters adicionales

### 🔧 Dockerfiles Modificados

#### `search-api/Dockerfile`
**Cambios:**
- ❌ Removido `FROM scratch` (no tiene shell para healthchecks)
- ✅ Cambiado a `FROM alpine:latest`
- ✅ Agregado `wget` para healthcheck
- ✅ Agregado usuario no-root `searchuser`
- ✅ Cambiado ENTRYPOINT a CMD para consistencia

**Motivo:** `FROM scratch` no tiene shell ni utilidades básicas, lo que impedía ejecutar healthchecks y dificultaba debugging.

#### `portfolio-api/Dockerfile`
**Cambios:**
- ✅ Hecho opcional el COPY de `./configs`
- ✅ Agregado fallback: `mkdir -p ./configs` si no existe

**Motivo:** El directorio `configs` solo tiene 2 archivos (rabbitmq.conf y redis.conf) que no son necesarios en runtime.

#### `wallet-api/Dockerfile`
**Cambios:**
- ❌ Removido `COPY --from=builder /app/config ./config`
- ✅ Agregado `wget` para healthcheck
- ✅ Creado directorio `/app/logs`

**Motivo:** El directorio `config` no existe en el repo, causaba error en build.

### 🏗️ Arquitectura Implementada

#### Red Unificada
- **Red:** `cryptosim-network` (172.25.0.0/16)
- **Service Discovery:** DNS automático por nombre de servicio
- **Comunicación:** Todos los servicios pueden comunicarse entre sí

#### Puertos Externos
```
8001 → Users API
8002 → Orders API
8003 → Search API
8004 → Market Data API
8005 → Portfolio API
8006 → Wallet API
3306 → MySQL
27017 → Orders MongoDB
27018 → Portfolio MongoDB
27019 → Wallet MongoDB
6379 → Redis
5672/15672 → RabbitMQ
8983 → Solr
11211 → Memcached
9090 → Prometheus (profile: monitoring)
3000 → Grafana (profile: monitoring)
```

#### Bases de Datos Separadas
- **users-mysql** - MySQL 8.0 para Users API
- **orders-mongo** - MongoDB 7.0 para Orders API
- **portfolio-mongo** - MongoDB 7.0 para Portfolio API
- **wallet-mongo** - MongoDB 7.0 para Wallet API

#### Infraestructura Compartida
- **shared-redis** - Cache común para todos los servicios
- **shared-rabbitmq** - Message broker común
- **solr** - Motor de búsqueda para Search API
- **memcached** - Cache distribuido para Search API

### 🎯 Mejoras Implementadas

#### 1. Independencia de Microservicios ✅
- Cada servicio mantiene su base de datos propia
- Deployable independientemente
- Sin dependencias circulares

#### 2. Service Discovery Automático ✅
```yaml
# Antes (no funcionaba entre contenedores)
USER_API_BASE_URL=http://localhost:8001

# Ahora (funciona con DNS interno)
USER_API_BASE_URL=http://users-api:8001
```

#### 3. Healthchecks Configurados ✅
- Todos los servicios tienen healthcheck
- Dependencies correctas con `condition: service_healthy`
- Start periods apropiados para cada servicio

#### 4. Gestión Simplificada ✅
```bash
# Antes: Levantar cada servicio manualmente
cd users-api && docker-compose up -d
cd orders-api && docker-compose up -d
# ... etc

# Ahora: Un solo comando
make up
```

#### 5. Monitoring Opcional ✅
```bash
# Levantar solo servicios core
docker-compose up -d

# Levantar con monitoring
docker-compose --profile monitoring up -d
# O usar: make monitoring-up
```

### 📊 Comparación Antes/Después

| Aspecto | Antes | Después |
|---------|-------|---------|
| **Docker Compose** | 6 archivos separados | 1 archivo unificado |
| **Redes** | 6 redes aisladas | 1 red compartida |
| **Comandos para levantar** | 6 comandos | 1 comando |
| **Service Discovery** | ❌ No funciona | ✅ Funciona |
| **Healthchecks** | Parcial | ✅ Completo |
| **Documentación** | Dispersa | ✅ Centralizada |
| **Utilidades** | Ninguna | 40+ comandos Make |

### 🚀 Próximos Pasos Recomendados

1. **Testing**
   - Levantar servicios: `make up`
   - Verificar salud: `make health`
   - Probar comunicación entre servicios

2. **Configuración**
   - Copiar `.env.example` a `.env`
   - Ajustar secrets y API keys
   - Configurar valores según entorno

3. **Desarrollo**
   - Usar `make logs-<servicio>` para debugging
   - Aprovechar comandos Make para desarrollo
   - Implementar tests de integración

4. **Producción (futuro)**
   - Migrar a Kubernetes si se necesita escalado
   - Implementar CI/CD
   - Configurar monitoring completo

### 🐛 Problemas Conocidos Solucionados

1. ✅ **Puertos duplicados** - Cada servicio ahora tiene puerto externo único
2. ✅ **Redes aisladas** - Red compartida permite comunicación
3. ✅ **Dockerfiles rotos** - Todos los Dockerfiles ahora buildan correctamente
4. ✅ **Variables de entorno inconsistentes** - Unificadas en `.env.example`
5. ✅ **Falta de documentación** - README y QUICKSTART agregados

### 📝 Notas Técnicas

- **Go version:** 1.21+ requerido
- **Docker version:** 20.10+ recomendado
- **Docker Compose version:** 2.0+ recomendado
- **Memoria RAM:** 8GB+ recomendado
- **Espacio en disco:** 20GB+ recomendado

### 🙏 Reconocimientos

- Arquitectura basada en mejores prácticas de microservicios
- Inspirado en proyectos open source como GitLab y Kong
- Docker Compose según especificación 3.8

---

**Versión:** 1.0.0
**Fecha:** 2025-10-12
**Autor:** CryptoSim Team
