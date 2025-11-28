# Guía de Pruebas del Sistema de Resiliencia

## ✅ Sistema Rebuild Completado

Todos los servicios están corriendo:
- ✅ Frontend (Next.js) - http://localhost:3000
- ✅ Users API - http://localhost:8001
- ✅ Orders API - http://localhost:8002
- ✅ Search API - http://localhost:8003
- ✅ Market Data API - http://localhost:8004
- ✅ Portfolio API - http://localhost:8005
- ✅ Solr - http://localhost:8983
- ✅ Redis - localhost:6379

## 🧪 Escenarios de Prueba

### Prueba 1: Funcionamiento Normal (Baseline)

**Objetivo**: Verificar que el sistema funciona correctamente sin fallos.

**Pasos**:
1. Abrir http://localhost:3000
2. Hacer login con tus credenciales
3. Ir al Dashboard
4. Verificar que aparecen "Recent Activity" y datos del portfolio

**Resultado Esperado**:
- ✅ Recent Activity muestra órdenes recientes
- ✅ Portfolio muestra balance y estadísticas
- ✅ Dashboard carga rápido
- ✅ Console muestra: `🔍 Search API request params:`

**Console Logs Esperados**:
```
✅ Cache hit for recent orders (si ya visitaste antes)
🔍 Search API request params: {...}
```

---

### Prueba 2: Solr Caído (Fallback a Cache)

**Objetivo**: Verificar que el sistema sigue funcionando con Solr caído.

**Pasos**:
1. **Visitar Dashboard primero** para cargar datos en cache
2. Abrir consola de logs: `docker-compose logs -f search-api`
3. Detener Solr:
   ```bash
   docker-compose stop solr
   ```
4. **Esperar 5 segundos**
5. Refrescar el navegador (F5)
6. Observar comportamiento

**Resultado Esperado**:
- ✅ Recent Activity muestra datos (desde cache)
- ✅ Portfolio muestra datos (desde cache)
- ⚠️ Puede haber warning en consola del navegador
- ✅ Dashboard NO está vacío

**Console Logs Esperados (Frontend)**:
```
✅ Cache hit for recent orders
✅ Returning cache after error
```

**Console Logs Esperados (Backend)**:
```
WARN Search execution failed
ERROR Search failed: Solr connection refused
```

**Para Restaurar**:
```bash
docker-compose start solr
```

---

### Prueba 3: Solr Saturado (Error 429)

**Objetivo**: Verificar manejo de rate limiting.

**Pasos**:
1. Visitar Dashboard para cargar cache
2. Simular saturación de Solr (esto requiere muchas peticiones simultáneas)
3. Alternativamente, podemos forzar el error 429 modificando temporalmente el código

**Resultado Esperado**:
- ✅ Sistema retorna datos desde cache stale
- ✅ Warning en consola sobre rate limit
- ✅ No crashea la aplicación

**Console Logs Esperados**:
```
⚠️ Search API rate limit reached, trying cache
✅ Returning stale cache for rate limit
```

---

### Prueba 4: Search API Caído (Fallback Completo)

**Objetivo**: Verificar comportamiento cuando Search API está completamente caído.

**Pasos**:
1. Visitar Dashboard para cargar cache
2. Detener Search API:
   ```bash
   docker-compose stop search-api
   ```
3. Esperar 5 segundos
4. Refrescar navegador (F5)

**Resultado Esperado**:
- ✅ Recent Activity muestra datos desde cache
- ✅ Portfolio muestra datos desde cache
- ❌ Error network en console del navegador (esperado)
- ✅ Dashboard muestra datos aunque sean stales

**Console Logs Esperados**:
```
❌ Search API error: Failed to fetch
✅ Returning cache after error
```

**Para Restaurar**:
```bash
docker-compose start search-api
```

---

### Prueba 5: Redis Caído (Backend sin Cache)

**Objetivo**: Verificar que el sistema funciona sin Redis (más lento).

**Pasos**:
1. Detener Redis:
   ```bash
   docker-compose stop redis
   ```
2. Esperar 5 segundos
3. Visitar Dashboard

**Resultado Esperado**:
- ✅ Dashboard carga (más lento que antes)
- ✅ Datos vienen directamente de Solr
- ⚠️ Mayor latencia
- ✅ Frontend cache sigue funcionando

**Console Logs Esperados (Backend)**:
```
WARN Cache connection failed
INFO Falling back to Solr direct
```

**Para Restaurar**:
```bash
docker-compose start redis
```

---

### Prueba 6: Cache Stale (Datos Viejos)

**Objetivo**: Verificar que el sistema retorna datos viejos cuando todo falla.

**Pasos**:
1. Visitar Dashboard y crear alguna orden
2. Esperar que los datos se cachechen
3. Detener Solr y Search API:
   ```bash
   docker-compose stop solr search-api
   ```
4. Esperar **6 minutos** (para que cache expire pero stale aún válido)
5. Refrescar navegador

**Resultado Esperado**:
- ✅ Dashboard muestra datos (stale hasta 30 min)
- ⚠️ Datos pueden estar desactualizados
- ✅ Mejor que pantalla vacía

**Console Logs Esperados**:
```
[OrdersCache] Returning stale cache for orders_cache_recent_1
✅ Returning cache after error
```

---

### Prueba 7: Fallo Total (Sin Cache)

**Objetivo**: Verificar comportamiento cuando TODO falla y no hay cache.

**Pasos**:
1. Limpiar cache del navegador:
   - DevTools → Application → Local Storage → Clear All
2. Detener Solr y Search API:
   ```bash
   docker-compose stop solr search-api
   ```
3. Visitar Dashboard en modo incógnito (sin cache)

**Resultado Esperado**:
- ❌ Recent Activity vacío (sin datos)
- ❌ Errors en console
- ✅ Aplicación NO crashea
- ⚠️ Muestra mensaje "No recent activity"

**Console Logs Esperados**:
```
❌ Search API error: Failed to fetch
❌ No cache available
```

---

## 📊 Verificación de Estadísticas de Cache

Puedes inspeccionar el cache desde la consola del navegador:

```javascript
// Ver estadísticas de cache
console.log(ordersCache.getStats())
// Output: { entries: 5, totalSize: 12345 }

// Invalidar cache de un usuario
ordersCache.invalidateUser(1)

// Limpiar todo el cache
ordersCache.clearAll()
```

---

## 🔍 Monitoreo de Logs

### Frontend Logs (Navegador)
Abrir DevTools → Console

Buscar por:
- `✅ Cache hit` - Cache funcionando
- `⚠️ Search API rate limit` - Rate limiting
- `❌ Search API error` - Errores de conexión
- `✅ Returning cache after error` - Fallback funcionando

### Backend Logs (Docker)
```bash
# Search API logs
docker-compose logs -f search-api

# Ver todos los servicios
docker-compose logs -f

# Filtrar errores
docker-compose logs search-api | findstr ERROR
```

---

## 🎯 Checklist de Validación

Después de ejecutar las pruebas, verificar:

| Escenario | Dashboard Funciona | Muestra Datos | Cache Activo | Performance |
|-----------|-------------------|---------------|--------------|-------------|
| Normal | ✅ | ✅ | ✅ | Rápido |
| Solr caído | ✅ | ✅ (cache) | ✅ | Rápido |
| Search API caído | ✅ | ✅ (cache) | ✅ | Rápido |
| Redis caído | ✅ | ✅ (Solr) | ⚠️ (solo frontend) | Lento |
| Todo caído + cache | ✅ | ✅ (stale) | ✅ | Rápido |
| Todo caído sin cache | ⚠️ | ❌ | ❌ | N/A |

---

## 🚀 Comandos Útiles

### Reiniciar Todo
```bash
docker-compose restart
```

### Reiniciar Solo un Servicio
```bash
docker-compose restart search-api
docker-compose restart solr
```

### Ver Estado de Servicios
```bash
docker-compose ps
```

### Limpiar y Reiniciar
```bash
docker-compose down
docker-compose up -d
```

### Ver Logs en Tiempo Real
```bash
# Todos los servicios
docker-compose logs -f

# Solo Search API
docker-compose logs -f search-api

# Últimas 100 líneas
docker-compose logs --tail=100 search-api
```

---

## 📝 Notas Importantes

### TTL de Cache
- **Fresh**: 5 minutos
- **Stale**: 30 minutos
- **LocalStorage**: Hasta que se llene (5-10MB)

### Invalidación de Cache
El cache se invalida automáticamente cuando:
- TTL expira (30 min)
- Usuario crea una nueva orden (TODO: implementar)
- Usuario hace logout (TODO: implementar)

### Limitaciones
- LocalStorage tiene límite de 5-10MB
- Cache no se comparte entre tabs/ventanas
- Cache persiste entre sesiones (hasta expiry)

---

## 🐛 Troubleshooting

### "No data available" a pesar del cache
**Solución**: Verificar que visitaste el dashboard al menos una vez antes de detener servicios.

### Cache no se invalida
**Solución**:
```javascript
// Desde consola del navegador
localStorage.clear()
location.reload()
```

### Datos siempre frescos (cache no funciona)
**Solución**: Verificar que el código de `search-api.ts` tiene la integración con `orders-cache`.

### LocalStorage lleno
**Solución**: El sistema limpia automáticamente las 5 entradas más antiguas.

---

## ✅ Resultado Esperado Final

Después de todas las pruebas:
- ✅ Sistema **resiliente** a fallos de Solr
- ✅ Sistema **resiliente** a fallos de Search API
- ✅ Cache funciona correctamente
- ✅ Fallback a datos stale funciona
- ✅ Performance mejorada con cache
- ✅ UX degradada > fallo total

**¡El sistema ahora es mucho más robusto! 🎉**
