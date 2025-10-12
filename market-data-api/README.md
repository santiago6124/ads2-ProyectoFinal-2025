# 📈 Market Data API - CryptoSim Platform

Microservicio de datos de mercado en tiempo real con integración a APIs externas (CoinGecko, Binance).

## 🚀 Quick Start (Recommended)

```bash
# Desde la raíz del proyecto
cd /ads2-ProyectoFinal-2025
make up              # Levantar todos los servicios
# O:
make up-market       # Levantar solo Market Data API + dependencias
```

**URLs del servicio:**
- **Market Data API**: http://localhost:8004
- **Health Check**: http://localhost:8004/health

**Ver logs:**
```bash
make logs-market
```

---

## 🏗️ Arquitectura & Dependencias

### Dependencias:
- **Redis 7** (`shared-redis`) - Cache de precios con TTL corto (30s)

### APIs Externas:
- **CoinGecko API** - Precios y datos de mercado
- **Binance API** - Precios en tiempo real (opcional)

### Es consumido por:
- Orders API (verificación de precios)
- Portfolio API (valoración de holdings)
- Frontend (gráficos y datos de mercado)

---

## ⚡ Características

- **Precios en Tiempo Real**: Actualización cada 30 segundos
- **Cache Inteligente**: Redis con TTL automático
- **WebSockets** (planeado): Stream de precios en tiempo real
- **Históricos**: Datos históricos de precios
- **Múltiples Fuentes**: Failover entre CoinGecko y Binance

## 📊 Endpoints Principales

### Obtener Precio de una Cripto
```http
GET /api/market/price/:symbol
```

Ejemplo:
```bash
curl http://localhost:8004/api/market/price/BTC
```

Respuesta:
```json
{
  "symbol": "BTC",
  "price": 45000.50,
  "timestamp": 1699123456,
  "source": "coingecko"
}
```

### Obtener Múltiples Precios
```http
GET /api/market/prices?symbols=BTC,ETH,USDT
```

### Histórico de Precios
```http
GET /api/market/history/:symbol?interval=1h&from=1699000000&to=1699123456
```

### Estadísticas de Mercado
```http
GET /api/market/stats/:symbol
```

Respuesta:
```json
{
  "symbol": "BTC",
  "high_24h": 46000.00,
  "low_24h": 44000.00,
  "volume_24h": 987654321,
  "market_cap": 876543210000,
  "price_change_24h": 2.5
}
```

## 🔧 Variables de Entorno

```env
SERVER_PORT=8004
REDIS_URL=redis://shared-redis:6379
ENVIRONMENT=development

# API Keys (opcional)
COINGECKO_API_KEY=your-api-key
BINANCE_API_KEY=your-api-key
```

## 🧪 Testing

```bash
cd market-data-api
go test ./...
```

## 🐛 Troubleshooting

### Redis no conecta
```bash
make logs-redis
docker-compose restart shared-redis
```

### API externa no responde
- Verificar API keys en `.env`
- Ver logs: `make logs-market`
- Revisar rate limits de CoinGecko/Binance

## 📚 Documentación

- [README Principal](../README.md)
- [QUICKSTART](../QUICKSTART.md)

---

**Market Data API** - Parte del ecosistema CryptoSim 🚀
