# 08 - Users API - Referencia de Endpoints

## 📋 Tabla de Contenidos
- [Autenticación](#autenticación)
- [Gestión de Usuarios](#gestión-de-usuarios)
- [API Interna](#api-interna)
- [Health Check](#health-check)

---

## 🔐 Autenticación

### POST /api/users/register

Registra un nuevo usuario en la plataforma.

**Request**:
```http
POST /api/users/register HTTP/1.1
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securePassword123",
  "username": "trader_001"
}
```

**Response** (201 Created):
```json
{
  "id": 1,
  "email": "user@example.com",
  "username": "trader_001",
  "balance": 100000.00,
  "created_at": "2025-11-14T10:00:00Z"
}
```

**Errores**:
- `400 Bad Request`: Email/username ya existe
- `400 Bad Request`: Validación fallida (password corto, email inválido)

---

### POST /api/users/login

Autentica un usuario y devuelve tokens JWT.

**Request**:
```http
POST /api/users/login HTTP/1.1
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securePassword123"
}
```

**Response** (200 OK):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

**Errores**:
- `401 Unauthorized`: Credenciales inválidas
- `400 Bad Request`: Datos faltantes

---

## 👤 Gestión de Usuarios

### GET /api/users/me

Obtiene el perfil del usuario autenticado.

**Request**:
```http
GET /api/users/me HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response** (200 OK):
```json
{
  "id": 1,
  "email": "user@example.com",
  "username": "trader_001",
  "balance": 95000.50,
  "created_at": "2025-11-14T10:00:00Z",
  "updated_at": "2025-11-14T15:30:00Z"
}
```

**Errores**:
- `401 Unauthorized`: Token inválido o expirado
- `404 Not Found`: Usuario no encontrado

---

### GET /api/users/balance

Obtiene el balance actual del usuario autenticado.

**Request**:
```http
GET /api/users/balance HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response** (200 OK):
```json
{
  "user_id": 1,
  "balance": 95000.50,
  "updated_at": "2025-11-14T15:30:00Z"
}
```

---

## 🔒 API Interna

### GET /internal/users/:id

Obtiene información de un usuario (solo para comunicación entre servicios).

**Headers**:
- `X-Internal-API-Key`: Clave secreta para autenticación interna

**Request**:
```http
GET /internal/users/123 HTTP/1.1
X-Internal-API-Key: internal-secret-key
```

**Response** (200 OK):
```json
{
  "id": 123,
  "email": "user@example.com",
  "username": "trader_001",
  "balance": 95000.50,
  "created_at": "2025-11-14T10:00:00Z"
}
```

**Errores**:
- `401 Unauthorized`: API Key inválida
- `404 Not Found`: Usuario no existe

---

### POST /internal/users/:id/update-balance

Actualiza el balance de un usuario (solo interno).

**Request**:
```http
POST /internal/users/123/update-balance HTTP/1.1
X-Internal-API-Key: internal-secret-key
Content-Type: application/json

{
  "amount": -5000.00,
  "operation": "deduct",
  "reason": "order_execution"
}
```

**Response** (200 OK):
```json
{
  "user_id": 123,
  "previous_balance": 100000.00,
  "new_balance": 95000.00,
  "updated_at": "2025-11-14T15:30:00Z"
}
```

---

## ❤️ Health Check

### GET /health

Verifica el estado del servicio.

**Request**:
```http
GET /health HTTP/1.1
```

**Response** (200 OK):
```json
{
  "status": "healthy",
  "service": "users-api",
  "timestamp": "2025-11-14T15:30:00Z",
  "database": "connected",
  "redis": "connected",
  "rabbitmq": "connected"
}
```

**Response** (503 Service Unavailable):
```json
{
  "status": "unhealthy",
  "service": "users-api",
  "timestamp": "2025-11-14T15:30:00Z",
  "database": "disconnected",
  "error": "Connection timeout to MySQL"
}
```

---

## 📊 Códigos de Estado HTTP

| Código | Significado | Cuándo se usa |
|--------|-------------|---------------|
| 200 | OK | Operación exitosa |
| 201 | Created | Usuario registrado |
| 400 | Bad Request | Validación fallida |
| 401 | Unauthorized | Token inválido/expirado |
| 404 | Not Found | Recurso no encontrado |
| 409 | Conflict | Email/username duplicado |
| 500 | Internal Server Error | Error del servidor |
| 503 | Service Unavailable | Servicio no disponible |

---

## 🔑 Autenticación JWT

### Formato del Token

```
Authorization: Bearer <access_token>
```

### Payload del JWT

```json
{
  "sub": "1",
  "email": "user@example.com",
  "username": "trader_001",
  "iat": 1699999999,
  "exp": 1700003599,
  "iss": "users-api",
  "aud": "cryptosim"
}
```

### Expiración

- **Access Token**: 1 hora (3600 segundos)
- **Refresh Token**: 7 días (604800 segundos)

---

## 🧪 Ejemplos con cURL

### Registrar Usuario

```bash
curl -X POST http://localhost:8001/api/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "test123456",
    "username": "testuser"
  }'
```

### Login

```bash
curl -X POST http://localhost:8001/api/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "test123456"
  }'
```

### Obtener Perfil

```bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

curl -X GET http://localhost:8001/api/users/me \
  -H "Authorization: Bearer $TOKEN"
```

---

## 📝 Notas Importantes

1. **JWT Secret**: Debe cambiarse en producción
2. **HTTPS**: Usar siempre en producción
3. **Rate Limiting**: Implementar para prevenir abuso
4. **Password Policy**: Mínimo 8 caracteres recomendado
5. **Balance Inicial**: $100,000 USD virtual por defecto
