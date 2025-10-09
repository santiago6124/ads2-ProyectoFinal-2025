# 👤 Users API - Microservicio de Gestión de Usuarios

## 📋 Descripción

El microservicio **Users API** es responsable de la gestión completa del ciclo de vida de los usuarios en la plataforma CryptoSim. Maneja la autenticación, autorización, registro de usuarios y gestión de perfiles, implementando seguridad mediante JWT y bcrypt para el hashing de contraseñas.

## 🎯 Responsabilidades

- **Autenticación**: Login seguro con generación de tokens JWT
- **Registro**: Creación de nuevos usuarios con validación de datos
- **Gestión de perfiles**: Actualización de información personal
- **Autorización**: Control de acceso basado en roles (normal/admin)
- **Verificación**: Validación de existencia de usuarios para otros microservicios
- **Seguridad**: Hashing de contraseñas y gestión de tokens

## 🏗️ Arquitectura

### Patrón MVC
```
users-api/
├── cmd/
│   └── main.go                 # Punto de entrada de la aplicación
├── internal/
│   ├── controllers/            # Controladores HTTP
│   │   ├── user_controller.go
│   │   └── auth_controller.go
│   ├── services/               # Lógica de negocio
│   │   ├── user_service.go
│   │   ├── auth_service.go
│   │   └── token_service.go
│   ├── repositories/           # Acceso a datos
│   │   └── user_repository.go
│   ├── models/                 # Modelos de dominio
│   │   ├── user.go
│   │   └── auth.go
│   ├── dto/                    # Data Transfer Objects
│   │   ├── user_dto.go
│   │   └── auth_dto.go
│   ├── middleware/             # Middlewares
│   │   ├── auth_middleware.go
│   │   ├── cors_middleware.go
│   │   └── logging_middleware.go
│   └── config/                 # Configuración
│       └── config.go
├── pkg/
│   ├── database/               # Conexión a BD
│   │   └── mysql.go
│   ├── utils/                  # Utilidades
│   │   ├── hash.go
│   │   ├── validator.go
│   │   └── response.go
│   └── errors/                 # Manejo de errores
│       └── errors.go
├── tests/                      # Tests
│   ├── unit/
│   ├── integration/
│   └── mocks/
├── migrations/                 # Migraciones de BD
│   ├── 001_create_users.up.sql
│   └── 001_create_users.down.sql
├── docs/                       # Documentación
│   └── swagger.yaml
├── Dockerfile
├── go.mod
├── go.sum
└── .env.example
```

## 💾 Modelo de Datos

### Tabla: users
```sql
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    role ENUM('normal', 'admin') DEFAULT 'normal',
    initial_balance DECIMAL(15,2) DEFAULT 100000.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_login TIMESTAMP NULL,
    is_active BOOLEAN DEFAULT TRUE,
    preferences JSON,
    INDEX idx_email (email),
    INDEX idx_username (username)
);

-- Tabla de tokens de refresh (opcional)
CREATE TABLE refresh_tokens (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    token VARCHAR(500) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    revoked BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_token (token),
    INDEX idx_user_id (user_id)
);

-- Tabla de auditoría de login
CREATE TABLE login_attempts (
    id INT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(100),
    ip_address VARCHAR(45),
    user_agent TEXT,
    success BOOLEAN,
    attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_email (email),
    INDEX idx_ip (ip_address)
);
```

## 🔌 API Endpoints

### Autenticación

#### POST `/api/users/register`
Registra un nuevo usuario en el sistema.

**Request Body:**
```json
{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "SecurePass123!",
  "first_name": "John",
  "last_name": "Doe"
}
```

**Response (201):**
```json
{
  "success": true,
  "message": "Usuario creado exitosamente",
  "data": {
    "id": 1,
    "username": "johndoe",
    "email": "john@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "normal",
    "initial_balance": 100000.00,
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

#### POST `/api/users/login`
Autentica un usuario y retorna un token JWT.

**Request Body:**
```json
{
  "email": "john@example.com",
  "password": "SecurePass123!"
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Login exitoso",
  "data": {
    "user": {
      "id": 1,
      "username": "johndoe",
      "email": "john@example.com",
      "role": "normal"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600
  }
}
```

#### POST `/api/users/refresh`
Renueva el token de acceso usando un refresh token.

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600
  }
}
```

#### POST `/api/users/logout`
Cierra la sesión del usuario y revoca el token.

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (200):**
```json
{
  "success": true,
  "message": "Logout exitoso"
}
```

### Gestión de Usuarios

#### GET `/api/users/:id`
Obtiene información de un usuario por ID.

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "username": "johndoe",
    "email": "john@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "normal",
    "initial_balance": 100000.00,
    "created_at": "2024-01-15T10:30:00Z",
    "is_active": true
  }
}
```

#### PUT `/api/users/:id`
Actualiza la información de un usuario.

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Request Body:**
```json
{
  "first_name": "John",
  "last_name": "Smith",
  "preferences": {
    "theme": "dark",
    "notifications": true,
    "language": "es"
  }
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Usuario actualizado exitosamente",
  "data": {
    "id": 1,
    "username": "johndoe",
    "email": "john@example.com",
    "first_name": "John",
    "last_name": "Smith",
    "preferences": {
      "theme": "dark",
      "notifications": true,
      "language": "es"
    }
  }
}
```

#### PUT `/api/users/:id/password`
Cambia la contraseña de un usuario.

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Request Body:**
```json
{
  "current_password": "SecurePass123!",
  "new_password": "NewSecurePass456!"
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Contraseña actualizada exitosamente"
}
```

#### DELETE `/api/users/:id`
Desactiva la cuenta de un usuario (soft delete).

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (200):**
```json
{
  "success": true,
  "message": "Usuario desactivado exitosamente"
}
```

### Endpoints Internos

#### GET `/api/users/:id/verify`
Verifica la existencia de un usuario (usado internamente por otros microservicios).

**Headers:**
```
X-Internal-Service: orders-api
X-API-Key: internal-secret-key
```

**Response (200):**
```json
{
  "exists": true,
  "user_id": 1,
  "role": "normal",
  "is_active": true
}
```

### Administración

#### POST `/api/users/:id/upgrade`
Actualiza el rol de un usuario a administrador (solo admin).

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (200):**
```json
{
  "success": true,
  "message": "Usuario promovido a administrador",
  "data": {
    "id": 1,
    "username": "johndoe",
    "role": "admin"
  }
}
```

#### GET `/api/users`
Lista todos los usuarios (solo admin).

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Query Parameters:**
- `page`: Número de página (default: 1)
- `limit`: Elementos por página (default: 20)
- `search`: Búsqueda por username o email
- `role`: Filtrar por rol (normal/admin)
- `is_active`: Filtrar por estado (true/false)

**Response (200):**
```json
{
  "success": true,
  "data": {
    "users": [
      {
        "id": 1,
        "username": "johndoe",
        "email": "john@example.com",
        "role": "normal",
        "created_at": "2024-01-15T10:30:00Z"
      }
    ],
    "pagination": {
      "total": 100,
      "page": 1,
      "limit": 20,
      "total_pages": 5
    }
  }
}
```

## 🔐 Seguridad

### JWT Configuration
```go
type JWTConfig struct {
    SecretKey       string
    AccessTokenTTL  time.Duration // 1 hora
    RefreshTokenTTL time.Duration // 7 días
    Issuer          string
}
```

### JWT Claims
```go
type CustomClaims struct {
    UserID   int    `json:"user_id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    Role     string `json:"role"`
    jwt.StandardClaims
}
```

### Password Hashing
```go
// Utiliza bcrypt con cost factor de 12
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

### Rate Limiting
```go
// Configuración de rate limiting por endpoint
var rateLimits = map[string]RateLimit{
    "POST:/api/users/login":    {Requests: 5, Window: 15 * time.Minute},
    "POST:/api/users/register": {Requests: 3, Window: 1 * time.Hour},
    "PUT:/api/users/password":  {Requests: 3, Window: 1 * time.Hour},
}
```

## 🧪 Testing

### Unit Tests
```go
// user_service_test.go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestUserService_CreateUser(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    service := NewUserService(mockRepo)
    
    user := &models.User{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "Test123!",
    }
    
    mockRepo.On("GetByEmail", user.Email).Return(nil, nil)
    mockRepo.On("Create", mock.AnythingOfType("*models.User")).Return(nil)
    
    // Act
    createdUser, err := service.CreateUser(user)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, createdUser)
    assert.NotEmpty(t, createdUser.ID)
    assert.NotEqual(t, "Test123!", createdUser.PasswordHash)
    mockRepo.AssertExpectations(t)
}

func TestUserService_Authenticate(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    mockTokenService := new(MockTokenService)
    service := NewAuthService(mockRepo, mockTokenService)
    
    hashedPassword, _ := HashPassword("Test123!")
    user := &models.User{
        ID:           1,
        Email:        "test@example.com",
        PasswordHash: hashedPassword,
        Role:         "normal",
    }
    
    mockRepo.On("GetByEmail", "test@example.com").Return(user, nil)
    mockTokenService.On("GenerateTokenPair", user).Return("access_token", "refresh_token", nil)
    
    // Act
    authResponse, err := service.Authenticate("test@example.com", "Test123!")
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, authResponse)
    assert.Equal(t, "access_token", authResponse.AccessToken)
    assert.Equal(t, "refresh_token", authResponse.RefreshToken)
    mockRepo.AssertExpectations(t)
    mockTokenService.AssertExpectations(t)
}

func TestUserService_ValidatePassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {"Valid password", "Test123!", false},
        {"Too short", "Test1!", true},
        {"No uppercase", "test123!", true},
        {"No lowercase", "TEST123!", true},
        {"No number", "TestTest!", true},
        {"No special char", "Test1234", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePassword(tt.password)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Integration Tests
```go
// user_integration_test.go
package tests

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestRegisterEndpoint(t *testing.T) {
    // Setup
    router := setupTestRouter()
    
    // Test data
    payload := map[string]string{
        "username":   "newuser",
        "email":      "new@example.com",
        "password":   "SecurePass123!",
        "first_name": "New",
        "last_name":  "User",
    }
    
    body, _ := json.Marshal(payload)
    
    // Request
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/users/register", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    
    router.ServeHTTP(w, req)
    
    // Assert
    assert.Equal(t, 201, w.Code)
    
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    
    assert.True(t, response["success"].(bool))
    assert.NotNil(t, response["data"])
}
```

## 🚀 Instalación y Configuración

### Requisitos
- Go 1.21+
- MySQL 8.0+
- Docker (opcional)

### Variables de Entorno
```env
# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=users_db

# JWT
JWT_SECRET=your-super-secret-key-change-in-production
JWT_ACCESS_TTL=3600
JWT_REFRESH_TTL=604800

# Server
SERVER_PORT=8001
SERVER_ENV=development

# Redis (for rate limiting)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# Internal Services
INTERNAL_API_KEY=internal-secret-key
```

### Desarrollo Local
```bash
# Clonar el repositorio
git clone https://github.com/cryptosim/users-api.git
cd users-api

# Instalar dependencias
go mod download

# Ejecutar migraciones
migrate -path migrations -database "mysql://user:password@tcp(localhost:3306)/users_db" up

# Ejecutar el servicio
go run cmd/main.go

# Ejecutar tests
go test ./... -v

# Ejecutar con hot reload (usando air)
air
```

### Docker
```bash
# Construir imagen
docker build -t users-api:latest .

# Ejecutar contenedor
docker run -p 8001:8001 \
  -e DB_HOST=mysql \
  -e JWT_SECRET=secret \
  --network cryptosim_network \
  users-api:latest
```

## 📊 Métricas y Monitoreo

### Prometheus Metrics
```go
// Métricas expuestas en /metrics
var (
    loginAttempts = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "user_login_attempts_total",
            Help: "Total number of login attempts",
        },
        []string{"status"},
    )
    
    registeredUsers = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "registered_users_total",
            Help: "Total number of registered users",
        },
    )
    
    authTokensGenerated = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "auth_tokens_generated_total",
            Help: "Total number of JWT tokens generated",
        },
    )
)
```

### Health Check
```
GET /health

Response:
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "database": "connected",
  "uptime": "2h30m15s"
}
```

## 📝 Documentación API

### Swagger/OpenAPI
La documentación completa de la API está disponible en:
- Desarrollo: http://localhost:8001/swagger
- Producción: https://api.cryptosim.com/users/swagger

## 🔄 CI/CD

### GitHub Actions Workflow
```yaml
name: Users API CI/CD

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - uses: actions/setup-go@v2
      with:
        go-version: 1.21
    - run: go test ./... -v
    - run: go build -v ./...

  docker:
    needs: test
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - name: Build and push Docker image
      run: |
        docker build -t users-api:${{ github.sha }} .
        docker push users-api:${{ github.sha }}
```

## 🐛 Troubleshooting

### Problemas Comunes

#### Error: "Connection refused" al conectar con MySQL
```bash
# Verificar que MySQL está corriendo
sudo systemctl status mysql

# Verificar credenciales
mysql -u root -p
```

#### Error: "Invalid JWT token"
```bash
# Verificar que el JWT_SECRET coincide en todos los ambientes
# Verificar que el token no ha expirado
# Verificar el formato del header Authorization: Bearer <token>
```

#### Error: "Too many login attempts"
```bash
# El rate limiting está activo
# Esperar 15 minutos o limpiar el cache de Redis
redis-cli
> DEL rate_limit:login:user@example.com
```

## 📞 Soporte

- **Issues**: https://github.com/cryptosim/users-api/issues
- **Email**: support@cryptosim.com
- **Slack**: #users-api-support

## 📄 Licencia

Este microservicio es parte del proyecto CryptoSim y está licenciado bajo MIT License.

---

**Users API** - Parte del ecosistema de microservicios de CryptoSim 🚀