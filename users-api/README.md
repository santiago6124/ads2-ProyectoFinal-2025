# 👤 Users API - CryptoSim Platform

A comprehensive Go-based microservice for user management, authentication, and authorization in the CryptoSim cryptocurrency trading simulation platform.

## 🚀 Quick Start (Recommended)

**This service is part of the CryptoSim microservices ecosystem.** The easiest way to run it is using the **unified Docker Compose** at the project root:

```bash
# From the project root
cd /ads2-ProyectoFinal-2025
make up              # Levantar todos los servicios
# Or:
make up-users        # Levantar solo Users API + dependencias
```

**Service URLs:**
- **Users API**: http://localhost:8001
- **Health Check**: http://localhost:8001/health
- **Swagger Docs**: http://localhost:8001/swagger

**View logs:**
```bash
make logs-users
```

**Access shell:**
```bash
make shell-users
```

---

## 🏗️ Architecture & Dependencies

**This service requires:**
- **MySQL 8.0** (`users-mysql` container)
- **Redis** (`shared-redis` container)

**Communicates with:**
- Called by: Orders API, Wallet API, Portfolio API
- Internal endpoints for service-to-service communication

**Full documentation**: See [main README](../README.md)

---

## 🚀 Features

- **User Registration & Authentication**: Secure user registration with email validation and JWT-based authentication
- **Role-Based Access Control**: Support for normal users and administrators
- **Password Security**: bcrypt hashing with configurable cost factor
- **Rate Limiting**: Protection against brute force attacks
- **Token Management**: Access and refresh token pairs with automatic rotation
- **Audit Logging**: Comprehensive logging of all authentication attempts
- **Health Monitoring**: Built-in health checks and Prometheus metrics
- **API Documentation**: Auto-generated Swagger documentation

## 🏗️ Architecture

Built following clean architecture principles with:
- **MVC Pattern**: Controllers, Services, and Repositories
- **Dependency Injection**: Loosely coupled components
- **Interface-Based Design**: Easy testing and mocking
- **SOLID Principles**: Maintainable and extensible code

## 🛠️ Technology Stack

- **Language**: Go 1.21+
- **Framework**: Gin HTTP framework
- **Database**: MySQL 8.0 with GORM ORM
- **Authentication**: JWT tokens
- **Password Hashing**: bcrypt
- **Testing**: Testify framework
- **Containerization**: Docker & Docker Compose
- **Documentation**: Swagger/OpenAPI

## 📦 Installation

### Prerequisites

- Go 1.21 or higher
- MySQL 8.0+
- Docker (optional)
- Make (optional)

### Local Development

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd users-api
   ```

2. **Set up environment**
   ```bash
   make setup
   # Or manually:
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Install dependencies**
   ```bash
   make install-deps
   # Or:
   go mod download
   ```

4. **Run database migrations**
   ```bash
   make migrate-up
   ```

5. **Start the service**
   ```bash
   make run
   # Or with hot reload:
   make dev
   ```

### Docker Deployment

**⚠️ IMPORTANT:** Use the unified Docker Compose at the project root instead:

```bash
# From project root
cd /ads2-ProyectoFinal-2025
make up
```

For standalone deployment (advanced):
```bash
# Build image only
docker build -t users-api:latest .

# Run with external database
docker run -d \
  -p 8001:8001 \
  -e DB_HOST=host.docker.internal \
  -e DB_PORT=3306 \
  users-api:latest
```

## 🔧 Configuration

### Environment Variables

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

# Internal Services
INTERNAL_API_KEY=internal-secret-key
```

## 📚 API Documentation

### Authentication Endpoints

#### Register User
```http
POST /api/users/register
Content-Type: application/json

{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "SecurePass123!",
  "first_name": "John",
  "last_name": "Doe"
}
```

#### Login
```http
POST /api/users/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "SecurePass123!"
}
```

#### Refresh Token
```http
POST /api/users/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### User Management Endpoints

#### Get User Profile
```http
GET /api/users/{id}
Authorization: Bearer {access_token}
```

#### Update User Profile
```http
PUT /api/users/{id}
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "first_name": "John",
  "last_name": "Smith",
  "preferences": {
    "theme": "dark",
    "notifications": true,
    "language": "en"
  }
}
```

#### Change Password
```http
PUT /api/users/{id}/password
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "current_password": "OldPass123!",
  "new_password": "NewPass456!"
}
```

### Admin Endpoints

#### List Users (Admin Only)
```http
GET /api/users?page=1&limit=20&search=john&role=normal&is_active=true
Authorization: Bearer {admin_access_token}
```

#### Upgrade User to Admin
```http
POST /api/users/{id}/upgrade
Authorization: Bearer {admin_access_token}
```

### Internal Service Endpoints

#### Verify User (Internal)
```http
GET /api/users/{id}/verify
X-Internal-Service: orders-api
X-API-Key: internal-secret-key
```

## 🧪 Testing

### Run Tests

```bash
# Unit tests
make test

# Integration tests
make test-integration

# All tests
make test-all

# Tests with coverage
make test-coverage
```

### Test Structure

```
tests/
├── unit/
│   ├── user_service_test.go
│   ├── auth_service_test.go
│   └── token_service_test.go
├── integration/
│   ├── auth_integration_test.go
│   └── user_integration_test.go
└── mocks/
    ├── mock_repositories.go
    └── mock_services.go
```

## 🔒 Security Features

### Password Requirements
- Minimum 8 characters
- Must contain uppercase and lowercase letters
- Must contain at least one number
- Must contain at least one special character

### Rate Limiting
- Maximum 5 failed login attempts per email
- 15-minute lockout period
- Configurable limits and time windows

### JWT Security
- Secure token generation
- Configurable expiration times
- Automatic token rotation
- Refresh token revocation

## 📊 Monitoring & Health Checks

### Health Endpoints

```http
GET /health        # Overall health status
GET /ready         # Readiness check
GET /live          # Liveness check
GET /metrics       # Prometheus metrics
```

### Metrics

- `user_login_attempts_total`: Total login attempts by status
- `registered_users_total`: Total registered users
- `auth_tokens_generated_total`: Total JWT tokens generated

## 🚀 Development

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Run security scan
make security

# Full development cycle
make dev-cycle
```

### Database Migrations

```bash
# Run migrations
make migrate-up

# Rollback migrations
make migrate-down

# Force migration version
make migrate-force VERSION=1
```

## 📋 API Rate Limits

| Endpoint | Limit | Window |
|----------|-------|--------|
| `POST /api/users/login` | 5 requests | 15 minutes |
| `POST /api/users/register` | 3 requests | 1 hour |
| `PUT /api/users/*/password` | 3 requests | 1 hour |

## 🐛 Troubleshooting

### Common Issues

1. **Database Connection Failed**
   ```bash
   # Check MySQL is running
   sudo systemctl status mysql

   # Verify credentials
   mysql -u root -p
   ```

2. **JWT Token Invalid**
   - Verify `JWT_SECRET` matches across environments
   - Check token expiration
   - Ensure proper Authorization header format

3. **Rate Limited**
   ```bash
   # Clear rate limit cache
   redis-cli
   > DEL rate_limit:login:user@example.com
   ```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Run linting and formatting
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Issues**: Create an issue on GitHub
- **Documentation**: Check the `/docs` folder for detailed API documentation
- **Health Check**: `GET /health` for service status

---

**Users API** - Part of the CryptoSim Microservices Ecosystem 🚀