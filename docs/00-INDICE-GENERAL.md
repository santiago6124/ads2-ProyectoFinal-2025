# 📚 CryptoSim - Índice General de Documentación

Bienvenido a la documentación completa del proyecto **CryptoSim**, una plataforma de simulación de trading de criptomonedas construida con arquitectura de microservicios.

---

## 🎯 Documentación General

### Para Empezar
- **[01 - Introducción al Proyecto](01-INTRODUCCION.md)** - Visión general, propósito y características principales
- **[02 - Guía de Instalación](02-INSTALACION.md)** - Configuración del entorno y primeros pasos
- **[03 - Guía de Uso](03-GUIA-USO.md)** - Cómo usar la plataforma paso a paso

### Arquitectura
- **[04 - Arquitectura General](04-ARQUITECTURA-GENERAL.md)** - Diseño de microservicios y patrones arquitectónicos
- **[05 - Stack Tecnológico](05-STACK-TECNOLOGICO.md)** - Tecnologías, frameworks y herramientas utilizadas
- **[06 - Infraestructura](06-INFRAESTRUCTURA.md)** - Docker, bases de datos y servicios compartidos

### Comunicación entre Servicios
- **[ARQUITECTURA-RABBITMQ.md](ARQUITECTURA-RABBITMQ.md)** - Sistema de mensajería y eventos asíncronos (YA EXISTENTE)

---

## 🔧 Documentación por Microservicio

### Users API - Gestión de Usuarios
- **[07 - Users API Overview](07-USERS-API.md)** - Arquitectura, autenticación JWT y gestión de usuarios
- **[08 - Users API Endpoints](08-USERS-ENDPOINTS.md)** - Referencia completa de endpoints

### Orders API - Órdenes de Trading
- **[09 - Orders API Overview](09-ORDERS-API.md)** - Sistema de órdenes y ejecución de trades
- **[10 - Orders API Endpoints](10-ORDERS-ENDPOINTS.md)** - Referencia completa de endpoints

### Search API - Búsqueda Avanzada
- **[11 - Search API Overview](11-SEARCH-API.md)** - Apache Solr y búsqueda de órdenes
- **[12 - Search API Endpoints](12-SEARCH-ENDPOINTS.md)** - Referencia completa de endpoints

### Market Data API - Datos de Mercado
- **[13 - Market Data API Overview](13-MARKET-DATA-API.md)** - Agregación de precios y análisis técnico
- **[14 - Market Data API Endpoints](14-MARKET-DATA-ENDPOINTS.md)** - Referencia completa de endpoints

### Portfolio API - Gestión de Portafolios
- **[15 - Portfolio API Overview](15-PORTFOLIO-API.md)** - Análisis de portafolio y métricas avanzadas
- **[16 - Portfolio API Endpoints](16-PORTFOLIO-ENDPOINTS.md)** - Referencia completa de endpoints

---

## 🔄 Flujos de Negocio

Documentación detallada de los flujos principales del sistema:

- **[FLUJO-REGISTRO-LOGIN.md](FLUJO-REGISTRO-LOGIN.md)** - Registro de usuarios y autenticación (YA EXISTENTE)
- **[FLUJO-COMPRA-VENTA.md](FLUJO-COMPRA-VENTA.md)** - Proceso completo de órdenes de trading (YA EXISTENTE)
- **[FLUJO-BUSQUEDA.md](FLUJO-BUSQUEDA.md)** - Sistema de búsqueda con Apache Solr (YA EXISTENTE)
- **[FLUJO-PORTFOLIO.md](FLUJO-PORTFOLIO.md)** - Cálculo y actualización de portafolios (YA EXISTENTE)

---

## 📖 Guías Técnicas

### Desarrollo
- **[17 - Guía de Desarrollo](17-GUIA-DESARROLLO.md)** - Estándares de código, estructura de proyectos Go y mejores prácticas
- **[18 - Testing](18-TESTING.md)** - Estrategias de testing y ejecución de pruebas
- **[19 - Debugging](19-DEBUGGING.md)** - Herramientas y técnicas de debugging

### DevOps
- **[20 - Docker y Contenedores](20-DOCKER.md)** - Gestión de contenedores y docker-compose
- **[21 - Bases de Datos](21-BASES-DATOS.md)** - MySQL, MongoDB, Redis y Solr
- **[22 - Monitoring](22-MONITORING.md)** - Logs, métricas y observabilidad

### Seguridad
- **[23 - Seguridad](23-SEGURIDAD.md)** - JWT, validación, rate limiting y mejores prácticas

---

## 📊 Referencia Rápida

### Comandos Esenciales
```bash
# Levantar todos los servicios
docker-compose up -d

# Ver logs en tiempo real
docker-compose logs -f

# Detener servicios
docker-compose down

# Reconstruir un servicio
docker-compose up -d --build <servicio>
```

### Puertos de Servicios
| Servicio | Puerto | URL |
|----------|--------|-----|
| Frontend | 3000 | http://localhost:3000 |
| Users API | 8001 | http://localhost:8001 |
| Orders API | 8002 | http://localhost:8002 |
| Search API | 8003 | http://localhost:8003 |
| Market Data API | 8004 | http://localhost:8004 |
| Portfolio API | 8005 | http://localhost:8005 |
| RabbitMQ Management | 15672 | http://localhost:15672 |
| Solr Admin | 8983 | http://localhost:8983 |

### Variables de Entorno Críticas
```bash
JWT_SECRET=your-super-secret-jwt-key
INTERNAL_API_KEY=internal-secret-key
MYSQL_ROOT_PASSWORD=rootpassword
MONGO_PASSWORD=password
```

---

## 🚀 Roadmap

### Versión Actual (v1.0)
- ✅ Sistema de usuarios y autenticación
- ✅ Órdenes de compra/venta
- ✅ Búsqueda avanzada con Solr
- ✅ Datos de mercado en tiempo real
- ✅ Portfolio con métricas avanzadas

### Próximas Versiones
- 🔄 Panel de administración
- 🔄 Notificaciones en tiempo real
- 🔄 Sistema de rankings
- 🔄 Gráficos interactivos
- 🔄 API Gateway unificado
- 🔄 Análisis predictivo con ML

---

## 🤝 Contribución

Para contribuir al proyecto:
1. Lee la **[Guía de Desarrollo](17-GUIA-DESARROLLO.md)**
2. Revisa los estándares de código
3. Crea una rama feature
4. Ejecuta los tests
5. Crea un Pull Request

---

## 📞 Soporte

- **Issues**: Reporta problemas en el repositorio
- **Documentación**: Este directorio `/docs`
- **Logs**: `docker-compose logs <servicio>`

---

## 📝 Notas Importantes

⚠️ **Este proyecto es educativo**. No usar con dinero real.

💡 **Precios reales**: Los precios vienen de APIs públicas (CoinGecko, Binance)

🎮 **Simulación**: El balance es virtual y solo para práctica

---

**Última actualización**: Noviembre 2025
**Versión**: 1.0.0