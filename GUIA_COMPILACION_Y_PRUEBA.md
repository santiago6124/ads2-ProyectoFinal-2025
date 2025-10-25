# Guía de Compilación y Prueba - Sistema Simplificado

**Fecha:** 2025-10-25

---

## 🚀 Compilación Completa

### Opción 1: Docker Compose (RECOMENDADO)

El sistema completo incluye varios servicios:
- MongoDB
- RabbitMQ
- Users API
- Market API
- **Orders API (simplificado)**
- Frontend (Next.js)

#### Pasos:

1. **Asegúrate de tener Docker Desktop corriendo**

2. **Compila y levanta todos los servicios:**
```bash
cd "c:\Users\lolog\Documents\Lorenzo\UCC\3ro\arqui soft 2\ads2-ProyectoFinal-2025"
docker-compose up --build
```

3. **Espera a que todos los servicios estén listos:**
```
✅ MongoDB corriendo en puerto 27017
✅ RabbitMQ corriendo en puerto 5672 (UI: 15672)
✅ Users API corriendo en puerto 8001
✅ Market API corriendo en puerto 8003
✅ Orders API corriendo en puerto 8002
✅ Frontend corriendo en puerto 3000
```

4. **Verifica que los logs digan:**
```
orders-api    | 🚀 Starting Orders API service (SIMPLIFIED)...
orders-api    | 📦 Connecting to MongoDB...
orders-api    | ✅ Successfully connected to MongoDB
orders-api    | 🔗 Initializing external service clients...
orders-api    | ✅ User API connection successful
orders-api    | ✅ Market API connection successful
orders-api    | 📨 Setting up RabbitMQ messaging...
orders-api    | ✅ RabbitMQ publisher initialized
orders-api    | ⚙️ Initializing business services (simplified)...
orders-api    | ✅ Business services initialized (simplified, no concurrency)
orders-api    | 🌐 HTTP server listening on 0.0.0.0:8002
orders-api    | ✨ Orders API is ready to accept requests!
orders-api    | 📝 System simplified: No workers, no orchestrator, synchronous execution
```

---

### Opción 2: Compilación Manual (Solo Orders API)

Si quieres compilar solo el Orders API:

```bash
# 1. Navegar al directorio
cd "c:\Users\lolog\Documents\Lorenzo\UCC\3ro\arqui soft 2\ads2-ProyectoFinal-2025\orders-api"

# 2. Descargar dependencias
go mod download

# 3. Compilar
go build -o bin/orders-api.exe cmd/server/main.go

# 4. Ejecutar (asegúrate de tener MongoDB y RabbitMQ corriendo)
.\bin\orders-api.exe
```

**Variables de entorno necesarias:**
```bash
DB_HOST=localhost
DB_PORT=27017
USERS_API_URL=http://localhost:8001
MARKET_API_URL=http://localhost:8003
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
JWT_SECRET=your-secret-key
PORT=8002
```

---

## 🧪 Pruebas del Sistema

### 1. Health Check

Verifica que el servicio esté corriendo:

```bash
curl http://localhost:8002/health
```

**Respuesta esperada:**
```json
{
  "status": "healthy",
  "timestamp": "2025-10-25T...",
  "version": "1.0.0",
  "services": {
    "database": {
      "status": "healthy",
      "response_time": 5000000
    },
    "user_api": {
      "status": "healthy",
      "response_time": 15000000
    },
    "market_api": {
      "status": "healthy",
      "response_time": 12000000
    },
    "rabbitmq_publisher": {
      "status": "healthy",
      "response_time": 3000000
    }
  }
}
```

---

### 2. Frontend (Más Fácil)

La forma más fácil de probar es usar el frontend:

1. **Abre el navegador:**
```
http://localhost:3000
```

2. **Regístrate o inicia sesión**

3. **Ve a la página de Trading:**
```
http://localhost:3000/trade
```

4. **Busca una criptomoneda (ej: BTC)**

5. **Ingresa cantidad y haz click en "Buy"**

6. **Verifica en los logs del Orders API:**
```
orders-api    | Published event: orders.created for order 673c...
orders-api    | Published event: orders.executed for order 673c...
```

---

### 3. API Manual (cURL)

#### A. Obtener Token JWT (Login)

Primero necesitas autenticarte:

```bash
curl -X POST http://localhost:8001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "tu-email@example.com",
    "password": "tu-password"
  }'
```

**Respuesta:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "email": "tu-email@example.com"
  }
}
```

**Guarda el `access_token` para usarlo en las siguientes requests.**

---

#### B. Crear una Orden de Compra (Market Order)

```bash
curl -X POST http://localhost:8002/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TU_TOKEN_AQUI" \
  -d '{
    "type": "buy",
    "crypto_symbol": "BTC",
    "quantity": "0.001",
    "order_kind": "market"
  }'
```

**Respuesta esperada:**
```json
{
  "id": "673c4a5b2e...",
  "order_number": "ORD-20251025-a1b2c3d4",
  "user_id": 1,
  "type": "buy",
  "order_kind": "market",
  "status": "executed",
  "crypto_symbol": "BTC",
  "crypto_name": "Bitcoin",
  "quantity": "0.001",
  "order_price": "50000.00",
  "total_amount": "50.00",
  "fee": "0.05",
  "fee_percentage": "0.1",
  "created_at": "2025-10-25T10:30:00Z",
  "executed_at": "2025-10-25T10:30:01Z",
  "updated_at": "2025-10-25T10:30:01Z"
}
```

**Observa los logs del Orders API - deberías ver:**
```
Published event: orders.created for order 673c...
Published event: orders.executed for order 673c...
```

---

#### C. Crear una Orden de Venta

```bash
curl -X POST http://localhost:8002/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TU_TOKEN_AQUI" \
  -d '{
    "type": "sell",
    "crypto_symbol": "BTC",
    "quantity": "0.0005",
    "order_kind": "market"
  }'
```

---

#### D. Listar Mis Órdenes

```bash
curl -X GET "http://localhost:8002/api/v1/orders?page=1&limit=10" \
  -H "Authorization: Bearer TU_TOKEN_AQUI"
```

**Respuesta:**
```json
{
  "orders": [
    {
      "id": "673c4a5b2e...",
      "order_number": "ORD-20251025-a1b2c3d4",
      "type": "buy",
      "status": "executed",
      "crypto_symbol": "BTC",
      "quantity": "0.001",
      "total_amount": "50.00",
      "fee": "0.05",
      "created_at": "2025-10-25T10:30:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10,
  "total_pages": 1,
  "summary": {
    "total_orders": 1,
    "executed_orders": 1,
    "pending_orders": 0,
    "cancelled_orders": 0,
    "failed_orders": 0,
    "total_volume": "50.00"
  }
}
```

---

#### E. Obtener Detalle de una Orden

```bash
curl -X GET http://localhost:8002/api/v1/orders/673c4a5b2e... \
  -H "Authorization: Bearer TU_TOKEN_AQUI"
```

---

### 4. Verificar RabbitMQ

Los eventos se publican a RabbitMQ. Puedes verlos en la UI:

1. **Abre el navegador:**
```
http://localhost:15672
```

2. **Login:**
- Usuario: `guest`
- Password: `guest`

3. **Ve a "Exchanges"**
- Deberías ver el exchange: `orders.events`

4. **Ve a "Queues"**
- Si creaste consumers, verías las colas aquí

5. **Observa los mensajes publicados**
- Click en el exchange `orders.events`
- Ve la sección "Publish rate"

---

## 🔍 Verificar MongoDB

Puedes ver las órdenes guardadas en MongoDB:

### Opción 1: MongoDB Compass (GUI)

1. **Descarga MongoDB Compass:** https://www.mongodb.com/products/compass

2. **Conecta a:**
```
mongodb://localhost:27017
```

3. **Navega a:**
```
Database: orders_db
Collection: orders
```

4. **Verás las órdenes guardadas:**
```json
{
  "_id": ObjectId("673c4a5b2e..."),
  "order_number": "ORD-20251025-a1b2c3d4",
  "user_id": 1,
  "type": "buy",
  "status": "executed",
  "crypto_symbol": "BTC",
  "crypto_name": "Bitcoin",
  "quantity": Decimal128("0.001"),
  "order_kind": "market",
  "price": Decimal128("50000.00"),
  "total_amount": Decimal128("50.00"),
  "fee": Decimal128("0.05"),
  "created_at": ISODate("2025-10-25T10:30:00Z"),
  "executed_at": ISODate("2025-10-25T10:30:01Z"),
  "updated_at": ISODate("2025-10-25T10:30:01Z")
}
```

### Opción 2: Mongo Shell (CLI)

```bash
# Conectar a MongoDB
docker exec -it mongodb mongosh

# Usar la base de datos
use orders_db

# Ver todas las órdenes
db.orders.find().pretty()

# Contar órdenes
db.orders.countDocuments()

# Ver solo órdenes ejecutadas
db.orders.find({status: "executed"}).pretty()

# Ver órdenes de un usuario específico
db.orders.find({user_id: 1}).pretty()

# Salir
exit
```

---

## 🐛 Debugging

### Ver logs en tiempo real:

```bash
# Todos los servicios
docker-compose logs -f

# Solo Orders API
docker-compose logs -f orders-api

# Solo MongoDB
docker-compose logs -f mongodb

# Solo RabbitMQ
docker-compose logs -f rabbitmq
```

---

### Errores Comunes:

#### 1. "Connection refused" al crear orden

**Problema:** Users API o Market API no están corriendo

**Solución:**
```bash
docker-compose ps
# Verifica que todos los servicios estén "Up"

# Si alguno está "Exit", reinicia:
docker-compose restart users-api
docker-compose restart market-api
```

---

#### 2. "Insufficient balance"

**Problema:** El usuario no tiene suficiente saldo

**Solución:** Agrega balance al usuario:
```bash
curl -X POST http://localhost:8001/api/users/balance \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TU_TOKEN" \
  -d '{
    "amount": 10000
  }'
```

---

#### 3. "Symbol not found"

**Problema:** El símbolo de crypto no existe en Market API

**Solución:** Usa símbolos válidos:
- BTC (Bitcoin)
- ETH (Ethereum)
- BNB (Binance Coin)
- SOL (Solana)
- XRP (Ripple)
- ADA (Cardano)

---

#### 4. Orders API no compila

**Problema:** Falta alguna dependencia

**Solución:**
```bash
cd orders-api
go mod tidy
go mod download
go build -o bin/orders-api.exe cmd/server/main.go
```

---

## 📊 Pruebas de Carga (Opcional)

Si quieres probar con muchas órdenes:

```bash
# Instalar Apache Bench (viene con Apache)
# O usar bombardier: https://github.com/codesenberg/bombardier

# Ejemplo con cURL en loop:
for i in {1..10}; do
  curl -X POST http://localhost:8002/api/v1/orders \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer TU_TOKEN" \
    -d '{
      "type": "buy",
      "crypto_symbol": "BTC",
      "quantity": "0.001",
      "order_kind": "market"
    }'
  echo "Order $i created"
  sleep 1
done
```

**Observa los logs** - deberías ver las órdenes procesándose de forma síncrona (una tras otra).

---

## 🎯 Checklist de Pruebas

- [ ] Docker Compose levanta todos los servicios
- [ ] Health check responde OK
- [ ] Frontend carga correctamente
- [ ] Puedo hacer login
- [ ] Puedo crear una orden de compra (market)
- [ ] La orden se ejecuta inmediatamente
- [ ] Veo la orden en MongoDB
- [ ] Veo el evento en RabbitMQ
- [ ] Puedo listar mis órdenes
- [ ] Puedo ver el detalle de una orden
- [ ] Los logs muestran el flujo simplificado

---

## 🆘 Soporte

Si algo no funciona:

1. **Revisa los logs:**
```bash
docker-compose logs -f orders-api
```

2. **Verifica la conexión a servicios externos:**
```bash
curl http://localhost:8001/health  # Users API
curl http://localhost:8003/health  # Market API
```

3. **Reinicia el servicio:**
```bash
docker-compose restart orders-api
```

4. **Reconstruye desde cero:**
```bash
docker-compose down -v
docker-compose up --build
```

---

## ✅ Sistema Funcionando Correctamente

Cuando todo funcione, deberías poder:

1. ✅ Crear órdenes desde el frontend
2. ✅ Ver las órdenes ejecutadas inmediatamente
3. ✅ Ver los eventos en RabbitMQ
4. ✅ Ver las órdenes en MongoDB
5. ✅ Ver los logs simplificados sin complejidad
6. ✅ Entender fácilmente el flujo del código

---

**¡Sistema Simplificado Listo para Uso Educativo! 🎓**

Generado: 2025-10-25
