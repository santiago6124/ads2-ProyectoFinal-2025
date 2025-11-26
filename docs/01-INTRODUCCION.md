# 01 - Introducción a CryptoSim

## 🎯 ¿Qué es CryptoSim?

**CryptoSim** es una plataforma educativa de simulación de trading de criptomonedas que permite a los usuarios aprender y practicar estrategias de inversión sin riesgo financiero real. Utiliza datos de mercado reales pero opera con un saldo virtual, creando un entorno seguro para el aprendizaje del trading.

---

## 🌟 Visión del Proyecto

Crear una plataforma completa que democratice el aprendizaje del trading de criptomonedas, proporcionando:

- **Educación Práctica**: Aprender haciendo, sin riesgo de pérdidas reales
- **Datos Reales**: Precios y condiciones de mercado actuales
- **Análisis Profesional**: Métricas y herramientas de análisis de portafolio
- **Escalabilidad**: Arquitectura moderna preparada para crecer

---

## ✨ Características Principales

### 🔐 Sistema de Usuarios
- Registro y autenticación con JWT
- Gestión de perfiles
- Balance virtual inicial de $100,000 USD
- Historial de transacciones

### 💰 Trading Simulado
- **Órdenes de mercado**: Ejecución inmediata al precio actual
- **Compra y venta**: Todas las criptomonedas soportadas
- **Gestión de balance**: Control automático de saldo disponible
- **Comisiones realistas**: Fees similares a exchanges reales (0.1% - 0.12%)

### 📊 Portfolio Inteligente
- **30+ métricas de análisis**:
  - ROI (Return on Investment)
  - Sharpe Ratio
  - Sortino Ratio
  - Maximum Drawdown
  - Volatilidad
  - Diversificación
  - Y muchas más...

### 🔍 Búsqueda Avanzada
- Motor de búsqueda Apache Solr
- Filtros por estado, tipo, símbolo, rango de fechas
- Búsqueda full-text en órdenes
- Paginación y ordenamiento

### 📈 Datos de Mercado
- **50+ criptomonedas** soportadas
- Precios en tiempo real
- Múltiples fuentes de datos (CoinGecko, Binance)
- Agregación y validación de precios
- Indicadores técnicos básicos

---

## 🏗️ Arquitectura de Alto Nivel

CryptoSim está construido como una **arquitectura de microservicios** moderna y escalable.

### Principios Arquitectónicos

1. **Separación de Responsabilidades**: Cada servicio tiene un propósito único y bien definido
2. **Comunicación Asíncrona**: RabbitMQ para eventos y mensajería entre servicios
3. **Independencia**: Cada servicio puede desarrollarse, desplegarse y escalar independientemente
4. **Resiliencia**: Manejo de fallos, timeouts y circuit breakers
5. **Observabilidad**: Logs estructurados, métricas y health checks

### Diagrama Simplificado

```
┌─────────────────────────────────────────────────────┐
│              FRONTEND (React/Next.js)               │
│           Interfaz de Usuario Web                   │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│                  API LAYER                          │
├──────────┬──────────┬──────────┬───────────────────┤
│ Users    │ Orders   │ Search   │ Market │ Portfolio│
│ :8001    │ :8002    │ :8003    │ :8004  │ :8005    │
└──────────┴──────────┴──────────┴────────┴──────────┘
     │          │          │          │         │
     ▼          ▼          ▼          ▼         ▼
┌─────────────────────────────────────────────────────┐
│           CAPA DE MENSAJERÍA (RabbitMQ)             │
│     Eventos: orders.*, balance.*, portfolio.*       │
└─────────────────────────────────────────────────────┘
     │          │          │          │         │
     ▼          ▼          ▼          ▼         ▼
┌──────────┬──────────┬──────────┬────────┬──────────┐
│  MySQL   │ MongoDB  │  Solr    │ Redis  │ MongoDB  │
│ (Users)  │ (Orders) │(Search)  │(Cache) │(Portfolio│
└──────────┴──────────┴──────────┴────────┴──────────┘
```

---

## 🎯 Casos de Uso Principales

### 1️⃣ Usuario Nuevo
```
1. Registrarse en la plataforma
2. Recibir $100,000 USD virtuales
3. Explorar precios de criptomonedas
4. Realizar primera compra
5. Ver portfolio actualizado
```

### 2️⃣ Trading Básico
```
1. Consultar precio actual de BTC
2. Crear orden de compra de 0.1 BTC
3. Ejecutar la orden
4. Ver balance actualizado
5. Ver BTC en el portfolio
```

### 3️⃣ Análisis de Portfolio
```
1. Realizar varias operaciones
2. Consultar métricas de rendimiento
3. Analizar ROI y riesgo
4. Revisar diversificación
5. Optimizar estrategia
```

### 4️⃣ Gestión de Órdenes
```
1. Buscar órdenes históricas
2. Filtrar por estado/tipo/fecha
3. Ver detalles de ejecución
4. Analizar comisiones pagadas
```

---

## 🛠️ Stack Tecnológico Resumen

### Backend
- **Lenguaje**: Go 1.21+
- **Framework Web**: Gin
- **ORM**: GORM
- **Validación**: go-playground/validator

### Bases de Datos
- **MySQL 8.0**: Datos de usuarios
- **MongoDB 7.0**: Órdenes y portfolios
- **Redis 7**: Cache y sesiones
- **Apache Solr 9**: Motor de búsqueda

### Infraestructura
- **Docker & Docker Compose**: Contenedorización
- **RabbitMQ 3.12**: Message broker
- **Memcached**: Cache distribuido

### Frontend (En desarrollo)
- **Next.js 14**: Framework React
- **TypeScript**: Tipado estático
- **TailwindCSS**: Estilos

---

## 📦 Microservicios

### 🔵 Users API (Puerto 8001)
**Responsabilidad**: Gestión de usuarios y autenticación

**Funcionalidades**:
- Registro de usuarios
- Login con JWT
- Gestión de balance virtual
- Perfiles de usuario
- API interna para validación

**Base de Datos**: MySQL

---

### 🟢 Orders API (Puerto 8002)
**Responsabilidad**: Creación y ejecución de órdenes

**Funcionalidades**:
- Crear órdenes de compra/venta
- Ejecutar órdenes de mercado
- Validar balance antes de ejecutar
- Calcular comisiones
- Publicar eventos de órdenes

**Base de Datos**: MongoDB

---

### 🟡 Search API (Puerto 8003)
**Responsabilidad**: Búsqueda avanzada de órdenes

**Funcionalidades**:
- Indexación en Apache Solr
- Búsqueda full-text
- Filtros avanzados
- Sincronización con Orders API
- Cache distribuido

**Base de Datos**: Apache Solr

---

### 🔴 Market Data API (Puerto 8004)
**Responsabilidad**: Provisión de datos de mercado

**Funcionalidades**:
- Agregación de precios de múltiples fuentes
- Cache de precios en Redis
- Detección de outliers
- Indicadores técnicos básicos
- API de precios en tiempo real

**Base de Datos**: Redis

---

### 🟣 Portfolio API (Puerto 8005)
**Responsabilidad**: Análisis y gestión de portafolios

**Funcionalidades**:
- Cálculo de métricas avanzadas
- Análisis de riesgo
- Seguimiento de rendimiento
- Optimización de portafolio
- Snapshots históricos

**Base de Datos**: MongoDB

---

## 🔄 Flujo de Datos

### Ejemplo: Compra de Bitcoin

```
1. Usuario solicita compra → Orders API
2. Orders API valida JWT → Users API (interno)
3. Orders API consulta precio → Market Data API
4. Orders API valida balance → Users API (RabbitMQ)
5. Orders API crea orden → MongoDB
6. Orders API ejecuta orden → MongoDB
7. Orders API publica evento → RabbitMQ
8. Portfolio API escucha evento → actualiza portfolio
9. Search API escucha evento → indexa en Solr
10. Users API actualiza balance → MySQL
```

---

## 📊 Métricas y KPIs

### Métricas de Rendimiento
- **Tiempo de respuesta**: < 200ms (p95)
- **Throughput**: 1000 req/s por servicio
- **Disponibilidad**: 99.9%

### Métricas de Negocio
- Usuarios registrados
- Órdenes ejecutadas
- Volumen de trading virtual
- Criptomonedas más populares
- ROI promedio de usuarios

---

## 🎓 Valor Educativo

### Para Estudiantes
- Aprender arquitectura de microservicios
- Práctica con Go y frameworks modernos
- Experiencia con Docker y contenedores
- Uso de message brokers (RabbitMQ)
- Integración con APIs externas

### Para Traders Principiantes
- Entender mecánicas del trading
- Practicar sin riesgo financiero
- Aprender análisis de portfolio
- Familiarizarse con términos y conceptos
- Desarrollar estrategias

---

## 🚀 Estado Actual y Roadmap

### ✅ Versión 1.0 (Actual)
- Sistema completo de backend funcional
- 5 microservicios operativos
- Autenticación y autorización
- Trading básico (market orders)
- Portfolio con 30+ métricas
- Búsqueda avanzada

### 🔄 Versión 1.1 (Próxima)
- Frontend completo con Next.js
- Gráficos interactivos
- Órdenes limit y stop-loss
- Notificaciones en tiempo real

### 🔮 Versión 2.0 (Futuro)
- Machine Learning para predicciones
- Social trading (copiar estrategias)
- Rankings y competencias
- API pública para desarrolladores
- Mobile app (React Native)

---

## ⚠️ Limitaciones y Disclaimer

### Limitaciones Técnicas
- **No es un exchange real**: Solo simulación
- **Datos con delay**: Puede haber latencia en precios
- **No hay orderbook**: Solo market orders
- **Liquidez infinita**: No hay problemas de slippage

### Disclaimer Legal
```
⚠️ IMPORTANTE:
- Este proyecto es EXCLUSIVAMENTE educativo
- NO usar con dinero real
- Los precios son reales pero las operaciones son simuladas
- No somos asesores financieros
- No nos hacemos responsables por decisiones de inversión
```

---

## 📚 Documentos Relacionados

- **[Arquitectura General](04-ARQUITECTURA-GENERAL.md)**: Detalles técnicos de arquitectura
- **[Stack Tecnológico](05-STACK-TECNOLOGICO.md)**: Tecnologías en profundidad
- **[Guía de Instalación](02-INSTALACION.md)**: Cómo empezar a usar el proyecto
- **[Flujos de Negocio](FLUJO-COMPRA-VENTA.md)**: Diagramas de secuencia detallados

---

**Siguiente**: [02 - Guía de Instalación](02-INSTALACION.md) →
