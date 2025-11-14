# CryptoSim - Tu Simulador de Trading de Criptomonedas

## ¿Qué es esto?

CryptoSim es una plataforma donde podes aprender a hacer trading de criptomonedas sin arriesgar plata real. Es como un juego donde te dan plata virtual y podes comprar y vender cripto para ver cómo te va. Perfecto para practicar antes de meterte en el mundo cripto de verdad.

## Estado del Proyecto - Primera Entrega

Este proyecto está en su **primera etapa de desarrollo** (del 7/11 al 14/11). El backend está completamente funcional y probado:

### ✅ ¿Qué funciona ahora?
- **Registro y Login**: Crear cuenta y autenticarte con JWT
- **Búsqueda de Órdenes**: Buscar órdenes con filtros avanzados usando Apache Solr
- **Precios en Tiempo Real**: Ver cotizaciones actuales de 50+ criptomonedas
- **Comprar/Vender Cripto**: Crear y ejecutar órdenes de mercado
- **Portfolio Avanzado**: Ver tu portfolio con 30+ métricas de análisis (ROI, Sharpe Ratio, etc.)
- **Historial de Operaciones**: Ver todas tus órdenes ejecutadas/pendientes/canceladas
- **Balance Virtual**: Arrancas con $100,000 USD virtuales


## ¿Cómo está armado? (La arquitectura)

El proyecto está dividido en pedacitos (microservicios) que trabajan juntos. Pensalo como una empresa donde cada empleado tiene su trabajo específico:

```
┌─────────────────────────────┐
│   Frontend (React)          │  <- Lo que ves en el navegador
│   Tu interfaz visual        │
└──────────┬──────────────────┘
           │
┌──────────▼──────────────────┐
│   API Gateway               │  <- El que dirige el tráfico
└──┬────┬────┬────┬──────────┘
   │    │    │    │
   ▼    ▼    ▼    ▼
┌─────┐ ┌──────┐ ┌──────┐ ┌──────┐
│Users│ │Orders│ │Search│ │Market│  <- Los trabajadores
│     │ │      │ │      │ │Data  │
└──┬──┘ └──┬───┘ └──┬───┘ └──┬───┘
   │       │        │        │
   ▼       ▼        ▼        ▼
[MySQL] [MongoDB] [Solr]  [Redis]   <- Donde se guarda todo
```

### Los "empleados" del sistema (Microservicios)

| Servicio | Puerto | Para qué sirve |
|----------|--------|----------------|
| **Users API** | 8001 | Maneja registro, login, autenticación JWT y tu balance virtual |
| **Orders API** | 8002 | Crea, ejecuta y gestiona tus órdenes de compra/venta |
| **Search API** | 8003 | Busca órdenes con filtros avanzados (usa Apache Solr) |
| **Market Data API** | 8004 | Trae los precios reales de 50+ criptomonedas desde FreeCryptoAPI |
| **Portfolio API** | 8005 | Analiza tu portafolio con 30+ métricas (ROI, Sharpe Ratio, etc.) |

### Tecnologías usadas (por si te interesa)

**Backend (lo que no ves):**
- Go - El lenguaje de programación
- Gin - Framework web
- GORM - Para hablar con MySQL

**Bases de datos (donde guardamos las cosas):**
- MySQL - Para usuarios
- MongoDB - Para órdenes y portafolio
- Redis - Para hacer todo más rápido (cache)
- Apache Solr - Para buscar cripto rápido

**Comunicación:**
- RabbitMQ - Para que los servicios se hablen entre sí

**Infraestructura:**
- Docker - Para que todo corra en contenedores
- Docker Compose - Para manejar todos los contenedores juntos

## ¿Qué necesitas para correrlo?

Antes de empezar, necesitas tener instalado:

- **Docker Desktop** (versión 20.10 o más nueva)
  - Descargalo de: https://www.docker.com/products/docker-desktop
- **Al menos 8GB de RAM** en tu compu
- **20GB de espacio libre** en disco

## ¿Cómo lo hago andar?

### Paso 1: Bajar el código

```bash
git clone <url-del-repositorio>
cd ads2-ProyectoFinal-2025
```

### Paso 2: Configurar las variables de entorno

```bash
# Si estás en Windows con PowerShell:
Copy-Item .env.example .env

# Si estás en Linux o Mac:
cp .env.example .env
```

Después podes editar el archivo `.env` si querés cambiar algo (pero no es necesario para empezar).

### Paso 3: Levantar todo

**Windows (PowerShell):**
```powershell
docker-compose up -d
```

**Linux/Mac:**
```bash
docker-compose up -d
```

Este comando va a:
1. Descargar todas las imágenes necesarias (la primera vez tarda un rato)
2. Crear las bases de datos
3. Levantar todos los servicios
4. Configurar las conexiones entre ellos

### Paso 4: Verificar que todo ande

**Ver el estado:**
```bash
docker-compose ps
```

Deberías ver todos los servicios en estado "Up" o "Running".

**Ver los logs (por si algo no anda):**
```bash
# Todos los logs
docker-compose logs

# Solo de un servicio específico
docker-compose logs users-api
docker-compose logs orders-api
```

### ¡Listo! Ahora podes acceder a:

- **Frontend**: http://localhost:3000 (cuando esté implementado)
- **Users API**: http://localhost:8001
- **Orders API**: http://localhost:8002
- **Search API**: http://localhost:8003
- **Market Data API**: http://localhost:8004
- **Portfolio API**: http://localhost:8005
- **RabbitMQ** (para ver las colas): http://localhost:15672
  - Usuario: `guest`
  - Password: `guest`

## Comandos útiles

### Para el día a día:

```bash
# Levantar todo
docker-compose up -d

# Apagar todo
docker-compose down

# Ver qué está corriendo
docker-compose ps

# Ver logs en tiempo real
docker-compose logs -f

# Ver logs de un servicio específico
docker-compose logs -f users-api

# Reiniciar un servicio
docker-compose restart users-api

# Reconstruir un servicio (si cambiaste código)
docker-compose up -d --build users-api
```

### Si algo se rompe:

```bash
# Apagar todo y borrar volúmenes (esto borra TODA la data)
docker-compose down -v

# Limpiar Docker completo
docker system prune -a

# Volver a empezar desde cero
docker-compose down -v
docker-compose up -d --build
```

## ¿Cómo probar que funciona?

### 1. Probar el login

```bash
# Crear un usuario de prueba (con curl o Postman)
curl -X POST http://localhost:8001/api/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@test.com",
    "password": "test123",
    "username": "testuser"
  }'

# Hacer login
curl -X POST http://localhost:8001/api/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@test.com",
    "password": "test123"
  }'
```

### 2. Ver precios de criptomonedas

```bash
# Ver precio de Bitcoin
curl http://localhost:8004/api/v1/prices/bitcoin

# Ver todas las criptos disponibles
curl http://localhost:8004/api/v1/prices
```

### 3. Buscar órdenes (necesitas tener órdenes primero)

```bash
curl -X POST http://localhost:8003/api/v1/search \
  -H "Authorization: Bearer TU_TOKEN_AQUI" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "*",
    "filters": {
      "status": "executed"
    }
  }'
```

### 4. Hacer una orden de compra

```bash
# Crear la orden (con el token del login)
curl -X POST http://localhost:8002/api/v1/orders \
  -H "Authorization: Bearer TU_TOKEN_AQUI" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "buy",
    "crypto_symbol": "BTC",
    "quantity": 0.001,
    "order_kind": "market"
  }'

# Esto te devuelve un order_id. Luego ejecutar la orden:
curl -X POST http://localhost:8002/api/v1/orders/ORDER_ID_AQUI/execute \
  -H "Authorization: Bearer TU_TOKEN_AQUI"
```

### 5. Ver tu portfolio

```bash
curl http://localhost:8005/api/portfolios/TU_USER_ID \
  -H "Authorization: Bearer TU_TOKEN_AQUI"
```

### 6. Ver tu historial de órdenes

```bash
curl http://localhost:8002/api/v1/orders \
  -H "Authorization: Bearer TU_TOKEN_AQUI"
```

## Problemas comunes y soluciones

### "Error: port is already allocated"
Significa que el puerto ya está en uso. Soluciones:
- Cerrá la aplicación que está usando ese puerto
- O cambia el puerto en `docker-compose.yml`

### "Cannot connect to database"
Esperá un minuto. Las bases de datos tardan en arrancar. Podes ver el progreso con:
```bash
docker-compose logs mysql
docker-compose logs mongodb
```

### "Out of memory"
Docker Desktop se quedó sin RAM. Andá a Settings → Resources → Memory y subilo a 8GB mínimo.

### Los contenedores se caen solos
```bash
# Ver qué pasó
docker-compose logs

# Intentar rebuild
docker-compose down
docker-compose up -d --build
```

## Estructura del proyecto

```
.
├── users-api/          # Todo lo de usuarios y autenticación
├── orders-api/         # Compra/venta de cripto
├── search-api/         # Búsqueda de criptomonedas
├── market-data-api/    # Precios en tiempo real
├── portfolio-api/      # Tu portafolio de inversiones
├── frontend/           # La interfaz web (React)
├── docker-compose.yml  # Configuración de todos los servicios
├── .env.example        # Variables de entorno de ejemplo
└── README.md           # Este archivo
```

## ¿Qué sigue después de esta entrega?

En las próximas versiones vamos a agregar:
- Panel de administración
- Registro de usuarios desde el frontend
- Historial completo de tus operaciones
- Cálculos más avanzados con procesamiento concurrente
- Sistema de notificaciones
- Rankings de mejores traders
- Y más...

## ¿Necesitas ayuda?

Si algo no te funciona o tenés dudas:

1. Revisá los logs: `docker-compose logs`
2. Verificá que Docker Desktop esté corriendo
3. Asegurate de tener los puertos libres (8001-8005, 3000, 15672)
4. Revisá que tengas suficiente RAM y espacio en disco

## Notas importantes

- Este proyecto es **educativo**, no está pensado para usar en producción con plata real
- Los precios de las criptos son reales (vienen de CoinGecko API)
- Tu "plata" es virtual, no podes sacarla ni es real
- Cada usuario arranca con un balance virtual inicial

---

Hecho con ❤️ para aprender sobre arquitectura de microservicios y trading
