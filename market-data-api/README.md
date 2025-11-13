# 📈 Market Data API - CryptoSim Platform

Microservicio de datos de mercado en tiempo real con integración a CoinGecko API.

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
- **CoinGecko API** - Precios y datos de mercado en tiempo real (19,000+ criptomonedas)

### Es consumido por:
- Orders API (verificación de precios)
- Portfolio API (valoración de holdings)
- Frontend (gráficos y datos de mercado)

---

## ⚡ Características

- **Precios en Tiempo Real**: Integración directa con CoinGecko API
- **Cache en Memoria**: Cache local con TTL configurable (30 segundos por defecto)
- **Históricos**: Datos históricos de precios con múltiples intervalos (1m, 5m, 15m, 1h, 4h, 1d, 1w)
- **Rate Limiting**: Control de límites de la API (50 requests/minuto en free tier)
- **Soporte Amplio**: Más de 50 criptomonedas populares mapeadas

## 📊 Endpoints Principales

### Obtener Precio de una Cripto
```http
GET /api/v1/prices/:symbol
```

Ejemplo:
```bash
curl http://localhost:8004/api/v1/prices/BTC
```

Respuesta:
```json
{
  "symbol": "BTC",
  "name": "BTC",
  "price": 110356.09,
  "change_24h": 1.41,
  "market_cap": 2096765764086.71,
  "volume": 288465858972.93,
  "timestamp": 1763047843,
  "source": "coingecko"
}
```

### Obtener Múltiples Precios
```http
GET /api/v1/prices?symbols=BTC,ETH,SOL
```

### Obtener Todos los Precios Populares
```http
GET /api/v1/prices
```
Retorna las 10 criptomonedas más populares por defecto.

### Histórico de Precios
```http
GET /api/v1/history/:symbol?interval=1h&limit=24
```

**Intervalos soportados:** `1m`, `5m`, `15m`, `1h`, `4h`, `1d`, `1w`

### Estadísticas de Mercado
```http
GET /api/v1/market/stats
```

Respuesta:
```json
{
  "totalMarketCap": 35642757098220.65,
  "totalVolume24h": 2487980016320.79,
  "btcDominance": 5.90,
  "ethDominance": 1.32,
  "activeCryptos": 10,
  "timestamp": 1763047937
}
```

## 🔧 Variables de Entorno

```env
SERVER_PORT=8004
REDIS_URL=redis://shared-redis:6379
ENVIRONMENT=development

# CoinGecko API (opcional - funciona sin API key en free tier)
COINGECKO_API_KEY=your-api-key
COINGECKO_BASE_URL=https://api.coingecko.com/api/v3
COINGECKO_RATE_LIMIT=50
COINGECKO_TIMEOUT=10s
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
- Verificar API key de CoinGecko en `.env` (opcional, funciona sin API key en free tier)
- Ver logs: `make logs-market`
- Revisar rate limits de CoinGecko (50 requests/minuto en free tier)
- Verificar conexión a internet

## 📚 Documentación

- [README Principal](../README.md)
- [QUICKSTART](../QUICKSTART.md)

---

**Market Data API** - Parte del ecosistema CryptoSim 🚀
