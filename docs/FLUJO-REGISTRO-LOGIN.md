# Flujo Técnico: Registro y Autenticación

## Descripción General

Este documento describe el flujo completo de registro de usuario, login y autenticación mediante JWT en CryptoSim.

## Arquitectura Involucrada

```
┌──────────┐         ┌──────────┐         ┌─────────┐
│  Cliente │────────>│ Users API│────────>│  MySQL  │
│ (Frontend│         │  :8001   │         │         │
│  /Postman│<────────│          │<────────│         │
└──────────┘         └──────────┘         └─────────┘
                           │
                           ▼
                     ┌──────────┐
                     │  Redis   │
                     │ (Cache)  │
                     └──────────┘
```

---

## 1. FLUJO DE REGISTRO (Sign Up)

### Endpoint
```
POST /api/users/register
Content-Type: application/json
```

### Request Body
```json
{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "securePassword123"
}
```

### Proceso Paso a Paso

#### 1.1 Cliente envía solicitud
```http
POST http://localhost:8001/api/users/register
Content-Type: application/json

{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "securePassword123"
}
```

#### 1.2 Users API - Validación de entrada
El handler `RegisterUser` en `users-api/handlers/user_handler.go` realiza:

1. **Parse del JSON**: Convierte el body a struct `RegisterRequest`
2. **Validaciones básicas**:
   - Username: 3-50 caracteres, alfanumérico
   - Email: formato válido
   - Password: mínimo 8 caracteres
3. Si falla: retorna `400 Bad Request`

#### 1.3 Service Layer - Lógica de negocio
El service `user_service.go` ejecuta:

1. **Verificar email único**:
   ```sql
   SELECT id FROM users WHERE email = 'john@example.com' AND deleted_at IS NULL
   ```
   - Si existe: retorna `409 Conflict` - "Email already registered"

2. **Verificar username único**:
   ```sql
   SELECT id FROM users WHERE username = 'johndoe' AND deleted_at IS NULL
   ```
   - Si existe: retorna `409 Conflict` - "Username already taken"

3. **Hash de password**:
   ```go
   hashedPassword := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
   ```
   - Usa bcrypt con cost factor 10
   - El hash generado es irreversible

4. **Crear usuario en MySQL**:
   ```sql
   INSERT INTO users (
     username,
     email,
     password_hash,
     role,
     initial_balance,
     created_at,
     updated_at
   ) VALUES (
     'johndoe',
     'john@example.com',
     '$2a$10$hashed_password_here',
     'normal',
     100000.00,
     NOW(),
     NOW()
   )
   ```

   **Campos automáticos**:
   - `id`: Auto-increment (ej: 123)
   - `role`: Siempre `normal` por defecto
   - `initial_balance`: $100,000 USD virtuales
   - `is_active`: `true`
   - `preferences`: JSON vacío `{}`

#### 1.4 Response exitoso
```json
{
  "status": "success",
  "message": "User registered successfully",
  "data": {
    "user": {
      "id": 123,
      "username": "johndoe",
      "email": "john@example.com",
      "role": "normal",
      "initial_balance": 100000.00,
      "is_active": true,
      "created_at": "2025-11-14T10:30:00Z"
    }
  }
}
```

**Status Code**: `201 Created`

#### 1.5 Errores posibles

| Código | Error | Causa |
|--------|-------|-------|
| 400 | Invalid input | Validación de campos falló |
| 409 | Email already registered | Email ya existe en BD |
| 409 | Username already taken | Username ya existe en BD |
| 500 | Internal server error | Error de BD o hash |

---

## 2. FLUJO DE LOGIN

### Endpoint
```
POST /api/users/login
Content-Type: application/json
```

### Request Body
```json
{
  "email": "john@example.com",
  "password": "securePassword123"
}
```

### Proceso Paso a Paso

#### 2.1 Cliente envía credenciales
```http
POST http://localhost:8001/api/users/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "securePassword123"
}
```

#### 2.2 Users API - Validación y búsqueda
El handler `Login` ejecuta:

1. **Parse del JSON**: Convierte a struct `LoginRequest`
2. **Validación básica**:
   - Email no vacío y formato válido
   - Password no vacío

#### 2.3 Service Layer - Autenticación

1. **Buscar usuario por email**:
   ```sql
   SELECT id, username, email, password_hash, role, initial_balance, is_active
   FROM users
   WHERE email = 'john@example.com' AND deleted_at IS NULL
   LIMIT 1
   ```

2. **Validar usuario existe**:
   - Si no existe: retorna `401 Unauthorized` - "Invalid credentials"

3. **Validar cuenta activa**:
   ```go
   if !user.IsActive {
     return 403 Forbidden - "Account is disabled"
   }
   ```

4. **Verificar password**:
   ```go
   err := bcrypt.CompareHashAndPassword(
     []byte(user.PasswordHash),
     []byte(inputPassword)
   )
   ```
   - Si no coincide: retorna `401 Unauthorized` - "Invalid credentials"

5. **Registrar intento de login**:
   ```sql
   INSERT INTO login_attempts (
     email,
     ip_address,
     success,
     attempted_at
   ) VALUES (
     'john@example.com',
     '192.168.1.100',
     true,
     NOW()
   )
   ```

6. **Actualizar last_login**:
   ```sql
   UPDATE users
   SET last_login = NOW()
   WHERE id = 123
   ```

#### 2.4 Generación de JWT Tokens

**Access Token** (corta duración):
```go
claims := jwt.MapClaims{
  "user_id":  123,
  "username": "johndoe",
  "email":    "john@example.com",
  "role":     "normal",
  "exp":      time.Now().Add(15 * time.Minute).Unix(), // 15 minutos
  "iat":      time.Now().Unix(),
  "nbf":      time.Now().Unix()
}

accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
signedAccessToken, _ := accessToken.SignedString([]byte(JWT_SECRET))
```

**Refresh Token** (larga duración):
```go
refreshClaims := jwt.MapClaims{
  "user_id":  123,
  "type":     "refresh",
  "exp":      time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 días
  "iat":      time.Now().Unix()
}

refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
signedRefreshToken, _ := refreshToken.SignedString([]byte(JWT_SECRET))
```

#### 2.5 Guardar Refresh Token en BD
```sql
INSERT INTO refresh_tokens (
  user_id,
  token,
  expires_at,
  created_at,
  revoked
) VALUES (
  123,
  'eyJhbGciOiJIUzI1NiIs...',
  DATE_ADD(NOW(), INTERVAL 7 DAY),
  NOW(),
  false
)
```

#### 2.6 Cachear sesión en Redis (opcional)
```
SET session:123 '{"user_id":123,"username":"johndoe","role":"normal"}' EX 900
```
- Key: `session:{user_id}`
- TTL: 900 segundos (15 minutos)

#### 2.7 Response exitoso
```json
{
  "status": "success",
  "message": "Login successful",
  "data": {
    "user": {
      "id": 123,
      "username": "johndoe",
      "email": "john@example.com",
      "role": "normal",
      "initial_balance": 100000.00
    },
    "tokens": {
      "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMjMsInVzZXJuYW1lIjoiam9obmRvZSIsImVtYWlsIjoiam9obkBleGFtcGxlLmNvbSIsInJvbGUiOiJub3JtYWwiLCJleHAiOjE2MzE1NDcyMDAsImlhdCI6MTYzMTU0NjMwMCwibmJmIjoxNjMxNTQ2MzAwfQ.signature",
      "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMjMsInR5cGUiOiJyZWZyZXNoIiwiZXhwIjoxNjMyMTUxMTAwLCJpYXQiOjE2MzE1NDYzMDB9.signature",
      "token_type": "Bearer",
      "expires_in": 900
    }
  }
}
```

**Status Code**: `200 OK`

---

## 3. FLUJO DE REFRESH TOKEN

### Endpoint
```
POST /api/users/refresh
Content-Type: application/json
```

### Request Body
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Proceso

#### 3.1 Validar Refresh Token
1. **Parsear JWT**:
   ```go
   token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
     return []byte(JWT_SECRET), nil
   })
   ```

2. **Verificar no expirado**:
   - Si expiró: retorna `401 Unauthorized` - "Token expired"

3. **Verificar tipo**:
   ```go
   if claims["type"] != "refresh" {
     return 401 Unauthorized - "Invalid token type"
   }
   ```

4. **Verificar no revocado en BD**:
   ```sql
   SELECT revoked
   FROM refresh_tokens
   WHERE token = '...' AND user_id = 123
   ```
   - Si revoked=true: retorna `401 Unauthorized` - "Token revoked"

#### 3.2 Generar nuevo Access Token
- Mismo proceso que en login (paso 2.4)
- Nuevo token con expiración de 15 minutos

#### 3.3 Response
```json
{
  "status": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

---

## 4. FLUJO DE AUTENTICACIÓN EN REQUESTS

### Middleware JWT

Todas las rutas protegidas usan el middleware `AuthMiddleware` que:

#### 4.1 Extraer token del header
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

#### 4.2 Validar token
```go
token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
  if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
    return nil, fmt.Errorf("unexpected signing method")
  }
  return []byte(JWT_SECRET), nil
})
```

#### 4.3 Verificar expiración
```go
if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
  if exp, ok := claims["exp"].(float64); ok {
    if time.Now().Unix() > int64(exp) {
      return 401 Unauthorized - "Token expired"
    }
  }
}
```

#### 4.4 Inyectar datos en contexto
```go
c.Set("user_id", claims["user_id"])
c.Set("username", claims["username"])
c.Set("email", claims["email"])
c.Set("role", claims["role"])
c.Next()
```

#### 4.5 Handler accede a datos
```go
userID := c.GetInt64("user_id")  // 123
role := c.GetString("role")       // "normal"
```

---

## 5. BASES DE DATOS

### Tabla: users
```sql
CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(50) UNIQUE NOT NULL,
  email VARCHAR(100) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  role ENUM('normal', 'admin') DEFAULT 'normal',
  initial_balance DECIMAL(15,2) DEFAULT 100000.00,
  is_active BOOLEAN DEFAULT true,
  preferences JSON,
  last_login TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at TIMESTAMP NULL,
  INDEX idx_email (email),
  INDEX idx_username (username),
  INDEX idx_deleted (deleted_at)
);
```

### Tabla: refresh_tokens
```sql
CREATE TABLE refresh_tokens (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  token TEXT NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  revoked BOOLEAN DEFAULT false,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  INDEX idx_user_id (user_id),
  INDEX idx_expires (expires_at)
);
```

### Tabla: login_attempts
```sql
CREATE TABLE login_attempts (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(100) NOT NULL,
  ip_address VARCHAR(45),
  success BOOLEAN NOT NULL,
  attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_email (email),
  INDEX idx_attempted (attempted_at)
);
```

---

## 6. SEGURIDAD

### 6.1 Protecciones implementadas
- ✅ **Password Hashing**: bcrypt con cost 10
- ✅ **JWT Signature**: HMAC-SHA256
- ✅ **Token Expiration**: Access 15min, Refresh 7 días
- ✅ **Token Revocation**: Tabla refresh_tokens
- ✅ **Soft Delete**: deleted_at para no perder historial
- ✅ **Login Audit**: Registro de intentos

### 6.2 Variables de entorno críticas
```env
JWT_SECRET=your-super-secret-key-change-in-production
JWT_EXPIRATION=15m
REFRESH_TOKEN_EXPIRATION=168h
```

### 6.3 Validaciones
- Username: 3-50 chars, alfanumérico
- Email: formato RFC 5322
- Password: mínimo 8 caracteres (recomendado: mayúscula + número + símbolo)

---

## 7. EJEMPLOS DE USO

### Registro completo
```bash
curl -X POST http://localhost:8001/api/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "trader123",
    "email": "trader@cryptosim.com",
    "password": "SecurePass123!"
  }'
```

### Login
```bash
curl -X POST http://localhost:8001/api/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "trader@cryptosim.com",
    "password": "SecurePass123!"
  }'
```

### Usar token en request protegido
```bash
curl http://localhost:8002/api/v1/orders \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

### Renovar token
```bash
curl -X POST http://localhost:8001/api/users/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
  }'
```

---

## 8. DIAGRAMA DE SECUENCIA

```
Cliente          Users API        MySQL         Redis
  │                 │               │             │
  │─Register────────>│               │             │
  │                 │─Validate────> │             │
  │                 │<─User Exists─ │             │
  │                 │─Hash Password │             │
  │                 │─Insert User──>│             │
  │<─201 Created────│<─User ID─────│             │
  │                 │               │             │
  │─Login───────────>│               │             │
  │                 │─Query User───>│             │
  │                 │<─User Data────│             │
  │                 │─Verify Pass   │             │
  │                 │─Generate JWT  │             │
  │                 │─Save Refresh─>│             │
  │                 │───Cache Session────────────>│
  │<─200 OK + JWT───│               │             │
  │                 │               │             │
  │─API Request─────>│               │             │
  │ (with JWT)      │───Verify JWT  │             │
  │                 │<──Token Valid │             │
  │<─Response───────│               │             │
```

---

## Resumen

- **Registro**: Validación → Hash → Inserción en MySQL → Balance inicial $100k
- **Login**: Validación → Verificación → JWT (access 15min + refresh 7 días)
- **Autenticación**: Middleware valida JWT en cada request protegido
- **Refresh**: Renueva access token sin pedir password
- **Seguridad**: bcrypt + JWT + auditoría + soft delete
