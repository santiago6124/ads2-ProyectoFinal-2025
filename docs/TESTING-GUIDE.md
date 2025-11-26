# 🧪 Guía de Testing - Implementaciones Nuevas

## Requisitos Previos

1. **Servicios Docker corriendo**:
```bash
docker-compose up -d
```

2. **Verificar que todos los servicios estén healthy**:
```bash
docker-compose ps
```

---

## 1. ✅ Fix de Búsqueda en Search API

### Problema Original
Error 400: `"invalid sort option: created_at_desc"`

### Cómo Probar

**Endpoint**: `POST http://localhost:8003/api/v1/search`

**Request Body**:
```json
{
  "query": "BTC",
  "page": 1,
  "limit": 10
}
```

**Con cURL**:
```bash
curl -X POST http://localhost:8003/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query":"BTC","page":1,"limit":10}'
```

**Resultado Esperado**:
- Status: `200 OK`
- Response con resultados de búsqueda
- **NO** debe aparecer error de validación

---

## 2. 👨‍💼 Usuario Administrador (Seed)

### Paso 1: Ejecutar el Seed

**Opción A - SQL Directo**:
```bash
# Dentro del contenedor de MySQL
docker-compose exec users-db mysql -u cryptosim_user -pcryptosim_password cryptosim_users < users-api/seeds/001_admin_user.sql
```

**Opción B - Comando Go**:
```bash
# Desde el directorio users-api
cd users-api
go run cmd/seed/main.go
```

**Opción C - Manualmente**:
```bash
# Conectar a MySQL
docker-compose exec users-db mysql -u cryptosim_user -pcryptosim_password cryptosim_users

# Ejecutar el contenido del archivo 001_admin_user.sql
```

### Paso 2: Verificar que el Usuario fue Creado

**SQL**:
```sql
SELECT id, username, email, role, is_active, initial_balance, created_at
FROM users
WHERE email = 'admin@cryptosim.com';
```

**Resultado Esperado**:
```
+----+----------+----------------------+-------+-----------+-----------------+---------------------+
| id | username | email                | role  | is_active | initial_balance | created_at          |
+----+----------+----------------------+-------+-----------+-----------------+---------------------+
|  X | admin    | admin@cryptosim.com  | admin |         1 |      100000.00  | 2025-11-26 XX:XX:XX |
+----+----------+----------------------+-------+-----------+-----------------+---------------------+
```

### Paso 3: Probar Login con Usuario Admin

**Endpoint**: `POST http://localhost:8001/api/users/login`

**Request**:
```json
{
  "email": "admin@cryptosim.com",
  "password": "admin1234"
}
```

**Con cURL**:
```bash
curl -X POST http://localhost:8001/api/users/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@cryptosim.com","password":"admin1234"}'
```

**Resultado Esperado**:
```json
{
  "user": {
    "id": X,
    "username": "admin",
    "email": "admin@cryptosim.com",
    "role": "admin",
    "is_active": true,
    "initial_balance": 100000.00,
    "current_balance": 100000.00
  },
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 3600
}
```

**⚠️ IMPORTANTE**: Copiar el `access_token` para las siguientes pruebas.

---

## 3. 🔐 Backend CRUD de Usuarios (Admin)

### Preparación: Obtener Token de Admin

Si no tienes el token del paso anterior:
```bash
export ADMIN_TOKEN=$(curl -X POST http://localhost:8001/api/users/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@cryptosim.com","password":"admin1234"}' \
  | jq -r '.access_token')

echo "Token guardado: $ADMIN_TOKEN"
```

### 3.1. GET - Listar Todos los Usuarios

**Endpoint**: `GET http://localhost:8001/api/admin/users?page=1&limit=20`

**Con cURL**:
```bash
curl -X GET "http://localhost:8001/api/admin/users?page=1&limit=20" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Resultado Esperado**:
```json
{
  "users": [
    {
      "id": 1,
      "username": "admin",
      "email": "admin@cryptosim.com",
      "role": "admin",
      "is_active": true,
      "initial_balance": 100000.00,
      "current_balance": 100000.00,
      "created_at": "2025-11-26T..."
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1,
    "pages": 1
  }
}
```

### 3.2. GET - Obtener Usuario por ID

**Endpoint**: `GET http://localhost:8001/api/admin/users/:id`

**Con cURL**:
```bash
# Reemplazar :id con el ID del usuario (ej: 1)
curl -X GET "http://localhost:8001/api/admin/users/1" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Resultado Esperado**:
```json
{
  "id": 1,
  "username": "admin",
  "email": "admin@cryptosim.com",
  "role": "admin",
  "is_active": true,
  "initial_balance": 100000.00,
  "current_balance": 100000.00,
  "created_at": "2025-11-26T...",
  "updated_at": "2025-11-26T..."
}
```

### 3.3. PUT - Actualizar Usuario

**Endpoint**: `PUT http://localhost:8001/api/admin/users/:id`

**Request Body**:
```json
{
  "first_name": "Admin",
  "last_name": "CryptoSim",
  "email": "admin@cryptosim.com",
  "role": "admin",
  "is_active": true
}
```

**Con cURL**:
```bash
curl -X PUT "http://localhost:8001/api/admin/users/1" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Admin",
    "last_name": "CryptoSim",
    "is_active": true
  }'
```

**Resultado Esperado**:
```json
{
  "id": 1,
  "username": "admin",
  "email": "admin@cryptosim.com",
  "first_name": "Admin",
  "last_name": "CryptoSim",
  "role": "admin",
  "is_active": true,
  "initial_balance": 100000.00,
  "current_balance": 100000.00
}
```

### 3.4. PATCH - Actualizar Balance de Usuario

**Endpoint**: `PATCH http://localhost:8001/api/admin/users/:id/balance`

**Request Body**:
```json
{
  "amount": 5000.00,
  "description": "Bonus administrativo"
}
```

**Con cURL - Agregar Balance**:
```bash
curl -X PATCH "http://localhost:8001/api/admin/users/1/balance" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 5000.00,
    "description": "Bonus administrativo"
  }'
```

**Con cURL - Retirar Balance**:
```bash
curl -X PATCH "http://localhost:8001/api/admin/users/1/balance" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": -2000.00,
    "description": "Retiro administrativo"
  }'
```

**Resultado Esperado**:
```json
{
  "message": "Balance updated successfully",
  "new_balance": 105000.00
}
```

**Verificar Transacciones**:
```sql
SELECT * FROM balance_transactions
WHERE user_id = 1
ORDER BY created_at DESC;
```

### 3.5. DELETE - Eliminar Usuario (Soft Delete)

**Endpoint**: `DELETE http://localhost:8001/api/admin/users/:id`

**⚠️ PRECAUCIÓN**: Primero crear un usuario de prueba para eliminar

**Crear Usuario de Prueba**:
```bash
curl -X POST http://localhost:8001/api/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "Test1234!",
    "first_name": "Test",
    "last_name": "User"
  }'
```

**Eliminar Usuario de Prueba** (obtener ID del response anterior):
```bash
# Reemplazar :id con el ID del usuario de prueba
curl -X DELETE "http://localhost:8001/api/admin/users/2" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Resultado Esperado**:
```json
{
  "message": "User deleted successfully"
}
```

**Verificar Soft Delete**:
```sql
SELECT id, username, email, is_active, deleted_at
FROM users
WHERE id = 2;
```

El campo `deleted_at` debe tener un timestamp, e `is_active` debe ser `0`.

---

## 4. 🧪 Tests de Seguridad

### 4.1. Intentar Acceder sin Token

```bash
curl -X GET "http://localhost:8001/api/admin/users" \
  -H "Content-Type: application/json"
```

**Resultado Esperado**: `401 Unauthorized`

### 4.2. Intentar Acceder con Usuario Normal (No Admin)

**Paso 1**: Crear usuario normal y hacer login
```bash
# Registrar usuario normal
curl -X POST http://localhost:8001/api/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "normaluser",
    "email": "normal@example.com",
    "password": "Normal1234!"
  }'

# Login
NORMAL_TOKEN=$(curl -X POST http://localhost:8001/api/users/login \
  -H "Content-Type: application/json" \
  -d '{"email":"normal@example.com","password":"Normal1234!"}' \
  | jq -r '.access_token')
```

**Paso 2**: Intentar acceder a endpoint admin
```bash
curl -X GET "http://localhost:8001/api/admin/users" \
  -H "Authorization: Bearer $NORMAL_TOKEN"
```

**Resultado Esperado**: `403 Forbidden`

---

## 5. 📋 Checklist de Testing Completo

### Search API Fix
- [ ] Búsqueda sin error de validación
- [ ] Sort por defecto funciona correctamente
- [ ] Diferentes opciones de sort funcionan

### Admin User Seed
- [ ] Usuario admin creado en base de datos
- [ ] Login exitoso con credenciales admin
- [ ] Token recibido correctamente
- [ ] Role es "admin"

### Admin CRUD
- [ ] GET /api/admin/users - Lista usuarios
- [ ] GET /api/admin/users/:id - Obtiene usuario específico
- [ ] PUT /api/admin/users/:id - Actualiza usuario
- [ ] PATCH /api/admin/users/:id/balance - Actualiza balance
- [ ] DELETE /api/admin/users/:id - Soft delete de usuario

### Seguridad
- [ ] Sin token = 401 Unauthorized
- [ ] Usuario normal intentando admin = 403 Forbidden
- [ ] Solo admin puede acceder a endpoints admin
- [ ] Transacciones de balance se registran correctamente

---

## 6. 🛠️ Troubleshooting

### Error: "Failed to connect to database"
```bash
# Verificar que MySQL esté corriendo
docker-compose ps users-db

# Si no está corriendo
docker-compose up -d users-db
```

### Error: "User already exists"
```bash
# El seed ya se ejecutó. Verificar:
docker-compose exec users-db mysql -u cryptosim_user -pcryptosim_password \
  -e "SELECT * FROM cryptosim_users.users WHERE email='admin@cryptosim.com';"
```

### Error 401 en endpoints admin
```bash
# Verificar que el token no haya expirado (duración: 1 hora)
# Hacer login nuevamente

# Verificar formato del header
# Correcto: "Authorization: Bearer eyJhbGc..."
# Incorrecto: "Authorization: eyJhbGc..." (falta "Bearer")
```

### Error 403 en endpoints admin
```bash
# Verificar que el usuario sea admin
docker-compose exec users-db mysql -u cryptosim_user -pcryptosim_password \
  -e "SELECT id, username, email, role FROM cryptosim_users.users WHERE email='admin@cryptosim.com';"

# El campo 'role' debe ser 'admin'
```

---

## 7. 📊 Scripts de Testing Automatizado

### Script Completo de Testing (Bash)

Guardar como `test-admin-api.sh`:

```bash
#!/bin/bash

BASE_URL="http://localhost:8001"
ADMIN_EMAIL="admin@cryptosim.com"
ADMIN_PASSWORD="admin1234"

echo "🧪 Testing Admin API Implementation"
echo "===================================="

# 1. Login como admin
echo ""
echo "1️⃣ Login as admin..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/users/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")

ADMIN_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')

if [ "$ADMIN_TOKEN" == "null" ] || [ -z "$ADMIN_TOKEN" ]; then
  echo "❌ Login failed!"
  echo $LOGIN_RESPONSE | jq .
  exit 1
fi

echo "✅ Login successful! Token obtained."

# 2. Get all users
echo ""
echo "2️⃣ Getting all users..."
USERS_RESPONSE=$(curl -s -X GET "$BASE_URL/api/admin/users?page=1&limit=20" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

USER_COUNT=$(echo $USERS_RESPONSE | jq '.users | length')
echo "✅ Found $USER_COUNT users"

# 3. Get admin user by ID
echo ""
echo "3️⃣ Getting admin user by ID..."
ADMIN_ID=$(echo $LOGIN_RESPONSE | jq -r '.user.id')
USER_RESPONSE=$(curl -s -X GET "$BASE_URL/api/admin/users/$ADMIN_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

USERNAME=$(echo $USER_RESPONSE | jq -r '.username')
echo "✅ Retrieved user: $USERNAME"

# 4. Update balance
echo ""
echo "4️⃣ Testing balance update..."
BALANCE_RESPONSE=$(curl -s -X PATCH "$BASE_URL/api/admin/users/$ADMIN_ID/balance" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount":1000.00,"description":"Test bonus"}')

NEW_BALANCE=$(echo $BALANCE_RESPONSE | jq -r '.new_balance')
echo "✅ Balance updated to: $NEW_BALANCE"

# 5. Security test - without token
echo ""
echo "5️⃣ Testing security (no token)..."
SECURITY_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$BASE_URL/api/admin/users")
HTTP_CODE=$(echo "$SECURITY_RESPONSE" | grep "HTTP_CODE" | cut -d: -f2)

if [ "$HTTP_CODE" == "401" ]; then
  echo "✅ Security test passed! Got 401 Unauthorized"
else
  echo "❌ Security test failed! Expected 401, got $HTTP_CODE"
fi

echo ""
echo "===================================="
echo "✅ All tests completed!"
```

**Ejecutar**:
```bash
chmod +x test-admin-api.sh
./test-admin-api.sh
```

---

## 8. ⚡ Caché de Precios (5 segundos)

### 8.1. Verificar que el caché está habilitado

**Endpoint**: `GET http://localhost:8004/health`

```bash
curl http://localhost:8004/health
```

**Resultado Esperado**:
```json
{
  "status": "healthy",
  "timestamp": 1732588800,
  "service": "market-data-api",
  "provider_status": "healthy",
  "cache_enabled": true,
  "cache_status": "healthy"
}
```

### 8.2. Probar Caché con Precio Individual

**Primera Petición** (debe consultar FreeCryptoAPI):
```bash
curl http://localhost:8004/api/v1/prices/BTC
```

**Resultado Esperado**:
```json
{
  "symbol": "BTC",
  "name": "Bitcoin",
  "price": 98765.43,
  "change_24h": 2.15,
  "market_cap": 1950000000000,
  "volume": 45000000000,
  "timestamp": 1732588800,
  "cached": false
}
```

**Segunda Petición** (dentro de 5 segundos - debe venir del caché):
```bash
curl http://localhost:8004/api/v1/prices/BTC
```

**Resultado Esperado**:
```json
{
  "symbol": "BTC",
  "name": "Bitcoin",
  "price": 98765.43,
  "change_24h": 2.15,
  "market_cap": 1950000000000,
  "volume": 45000000000,
  "timestamp": 1732588800,
  "cached": true
}
```

### 8.3. Probar Caché con Múltiples Precios

**Primera Petición**:
```bash
curl "http://localhost:8004/api/v1/prices?symbols=BTC,ETH,BNB"
```

**Resultado Esperado** (primera vez):
```json
{
  "data": [
    {
      "symbol": "BTC",
      "name": "Bitcoin",
      "price": 98765.43,
      "cached": false
    },
    {
      "symbol": "ETH",
      "name": "Ethereum",
      "price": 3456.78,
      "cached": false
    },
    {
      "symbol": "BNB",
      "name": "Binance Coin",
      "price": 623.45,
      "cached": false
    }
  ],
  "source": "freecryptoapi",
  "count": 3,
  "cached_count": 0
}
```

**Segunda Petición** (dentro de 5 segundos):
```bash
curl "http://localhost:8004/api/v1/prices?symbols=BTC,ETH,BNB"
```

**Resultado Esperado** (desde caché):
```json
{
  "data": [
    {
      "symbol": "BTC",
      "name": "Bitcoin",
      "price": 98765.43,
      "cached": true
    },
    {
      "symbol": "ETH",
      "name": "Ethereum",
      "price": 3456.78,
      "cached": true
    },
    {
      "symbol": "BNB",
      "name": "Binance Coin",
      "price": 623.45,
      "cached": true
    }
  ],
  "source": "cache",
  "count": 3,
  "cached_count": 3
}
```

### 8.4. Verificar Expiración del Caché

1. **Hacer una petición inicial**:
```bash
curl http://localhost:8004/api/v1/prices/ETH
```

2. **Esperar 6 segundos** (el caché expira a los 5 segundos)

3. **Hacer otra petición**:
```bash
curl http://localhost:8004/api/v1/prices/ETH
```

**Resultado Esperado**: La segunda petición debe tener `"cached": false`, indicando que el caché expiró y se consultó nuevamente a FreeCryptoAPI.

### 8.5. Verificar Caché Mixto

**Escenario**: Algunas cryptos en caché, otras no

1. **Primera petición** (solo BTC):
```bash
curl http://localhost:8004/api/v1/prices/BTC
```

2. **Segunda petición** (BTC está en caché, ETH no):
```bash
curl "http://localhost:8004/api/v1/prices?symbols=BTC,ETH"
```

**Resultado Esperado**:
```json
{
  "data": [
    {
      "symbol": "BTC",
      "cached": true
    },
    {
      "symbol": "ETH",
      "cached": false
    }
  ],
  "source": "mixed",
  "count": 2,
  "cached_count": 1
}
```

### 8.6. Monitorear Caché con Redis CLI

**Conectar a Redis**:
```bash
docker-compose exec redis redis-cli
```

**Ver keys de precios**:
```bash
KEYS price:*
```

**Resultado Esperado**:
```
1) "price:BTC"
2) "price:ETH"
3) "price:BNB"
```

**Ver TTL de una key**:
```bash
TTL price:BTC
```

**Resultado Esperado**: Un número entre 0 y 5 (segundos restantes)

**Ver contenido de una key**:
```bash
GET price:BTC
```

**Resultado Esperado**: JSON serializado del objeto AggregatedPrice

---

## 9. 📱 Testing desde Frontend (Próximo Paso)

Una vez implementado el panel de admin en el frontend, podrás probar:

1. Login con usuario admin
2. Visualizar lista de usuarios
3. Editar información de usuarios
4. Modificar balance de usuarios
5. Desactivar/activar usuarios

**Endpoints disponibles para el frontend**:
- `POST /api/users/login` - Login
- `GET /api/admin/users` - Listar usuarios
- `GET /api/admin/users/:id` - Ver usuario
- `PUT /api/admin/users/:id` - Actualizar usuario
- `PATCH /api/admin/users/:id/balance` - Actualizar balance
- `DELETE /api/admin/users/:id` - Eliminar usuario
