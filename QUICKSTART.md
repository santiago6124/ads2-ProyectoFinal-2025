# 🚀 CryptoSim - Inicio Rápido

Guía de 5 minutos para levantar todo el proyecto.

## ⚡ Comandos Rápidos

```bash
# 1. Configurar variables de entorno
make env

# 2. Levantar todos los servicios
make up

# 3. Ver estado
make status

# 4. Ver logs (opcional)
make logs
```

## 📝 Paso a Paso Detallado

### 1️⃣ Preparar el entorno

```bash
# Copiar archivo de configuración
cp .env.example .env

# (Opcional) Editar valores en .env si es necesario
nano .env  # o usa tu editor favorito
```

### 2️⃣ Levantar servicios

```bash
# Opción A: Usar Makefile (recomendado)
make up

# Opción B: Usar docker-compose directamente
docker-compose up -d

# Opción C: Levantar con monitoring (Prometheus + Grafana)
make dev-up
```

### 3️⃣ Verificar que todo funciona

```bash
# Ver estado de contenedores
make status

# Health check de APIs
make health

# Ver logs en tiempo real
make logs
```

## 🎯 Probar los Servicios

### Users API (Puerto 8001)

```bash
# Registrar un usuario
curl -X POST http://localhost:8001/api/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123",
    "first_name": "Test",
    "last_name": "User"
  }'

# Login
curl -X POST http://localhost:8001/api/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

### Market Data API (Puerto 8004)

```bash
# Obtener precio de Bitcoin
curl http://localhost:8004/api/market/price/BTC

# Obtener múltiples precios
curl http://localhost:8004/api/market/prices?symbols=BTC,ETH,USDT
```

### Orders API (Puerto 8002)

```bash
# Crear orden de compra (necesitas JWT token del login)
curl -X POST http://localhost:8002/api/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "user_id": 1,
    "type": "buy",
    "crypto_symbol": "BTC",
    "quantity": 0.1,
    "order_price": 45000
  }'
```

## 🔍 URLs Importantes

| Servicio | URL | Credenciales |
|----------|-----|--------------|
| Users API | http://localhost:8001 | - |
| Orders API | http://localhost:8002 | - |
| Search API | http://localhost:8003 | - |
| Market Data API | http://localhost:8004 | - |
| Portfolio API | http://localhost:8005 | - |
| Wallet API | http://localhost:8006 | - |
| RabbitMQ Management | http://localhost:15672 | guest / guest |
| Prometheus | http://localhost:9090 | - |
| Grafana | http://localhost:3000 | admin / admin |

## 🛑 Detener los Servicios

```bash
# Detener todos los servicios (preserva datos)
make down

# Detener y eliminar volúmenes (datos borrados)
make clean
```

## 🐛 Problemas Comunes

### Error: "port already in use"

```bash
# Ver qué proceso usa el puerto
sudo lsof -i :8001  # Cambiar 8001 por el puerto problemático

# Cambiar el puerto en docker-compose.yml:
ports:
  - "8101:8001"  # Cambia primer número (externo)
```

### Error: "Cannot connect to Docker daemon"

```bash
# Iniciar Docker Desktop
# O en Linux:
sudo systemctl start docker
```

### Servicios no se comunican

```bash
# Verificar que están en la misma red
docker network inspect cryptosim-network

# Recrear red
docker-compose down
docker-compose up -d
```

### Bases de datos no inician

```bash
# Ver logs de base de datos específica
make logs-mysql    # Para MySQL
make logs-mongo    # Para MongoDB

# Limpiar volúmenes y reiniciar
make clean
make up
```

## 📚 Próximos Pasos

1. ✅ Servicios funcionando
2. 📖 Leer [README.md](README.md) completo
3. 🧪 Ejecutar tests: `make test`
4. 💻 Explorar código en `/users-api`, `/orders-api`, etc.
5. 📊 Habilitar monitoring: `make monitoring-up`

## 💡 Tips

- **Ver logs de un servicio específico**: `make logs-users`
- **Entrar a un contenedor**: `make shell-users`
- **Reiniciar solo un servicio**: `docker-compose restart users-api`
- **Ver todos los comandos**: `make help`

## 🆘 ¿Necesitas ayuda?

- Ver logs: `make logs`
- Ver estado: `make status`
- Limpiar todo: `make clean && make up`
- Documentación completa: [README.md](README.md)

---

¡Listo! Ahora puedes empezar a desarrollar 🚀
