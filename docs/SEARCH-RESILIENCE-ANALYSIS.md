# Análisis de Resiliencia del Sistema de Búsqueda

## 📊 Estado Actual

### Arquitectura de Búsqueda

```
┌─────────────────────────────────────────────────────────────┐
│ Frontend (Next.js)                                          │
├─────────────────────────────────────────────────────────────┤
│ • Recent Activity (getRecentOrders)                         │
│ • Orders Page (searchOrders)                                │
│ • Transaction History (getUserOrders)                       │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ↓ HTTP
┌─────────────────────────────────────────────────────────────┐
│ Search API (Go)                                             │
├─────────────────────────────────────────────────────────────┤
│ SearchService → [Redis Cache] → [Solr Repository]          │
│                     ↓ Cache Miss     ↓                      │
│                   [Solr]          [Orders API Client]       │
└─────────────────────────────────────────────────────────────┘
```

### Capas de Resiliencia Existentes

#### ✅ Backend (Search API)
1. **Redis Cache** (cache_repository.go):
   - Cache de resultados de búsqueda (TTL: 5 min)
   - Cache de órdenes trending (TTL: 1 min)
   - Cache de sugerencias (TTL: 10 min)
   - Cache de filtros (TTL: 15 min)

2. **Orders API Client** (orders_client.go):
   - Cliente HTTP para comunicación directa con Orders API
   - Timeout configurable (default: 10s)
   - Actualmente NO se usa como fallback

#### ❌ Frontend
1. **NO hay cache local**
2. **NO hay fallback a Orders API directo**
3. **Retorna arrays vacíos en error 429**

## 🔍 Análisis de Puntos de Fallo

### Escenario 1: Solr Caído

**¿Qué pasa?**
```
Request → Search API → solrRepo.Search() → ERROR
                     ↓
         Retorna error al frontend
                     ↓
         Frontend muestra [] (array vacío)
```

**Impacto**:
- ❌ Recent Activity vacío
- ❌ Orders Page sin resultados
- ❌ Transaction History desaparece
- ✅ **Redis cache puede servir requests anteriores**

**Duración del caché**: 5 minutos para búsquedas

### Escenario 2: Redis Caído

**¿Qué pasa?**
```
Request → Search API → cacheRepo.Get() → SKIP (error ignorado)
                     ↓
         solrRepo.Search() → Solr responde
                     ↓
         Funciona pero más lento
```

**Impacto**:
- ✅ Sistema sigue funcionando
- ⚠️ Mayor latencia (sin cache)
- ⚠️ Mayor carga en Solr

### Escenario 3: Solr Saturado (429 Rate Limit)

**¿Qué pasa ACTUALMENTE?**
```
Request → Search API → solrRepo.Search() → Solr 429
                     ↓
         Retorna error al frontend
                     ↓
         Frontend detecta 429 → return { results: [] }
```

**Impacto**:
- ❌ Datos desaparecen aunque existan en cache
- ❌ No se intenta fallback a Orders API
- ❌ Usuario ve dashboard vacío

## 🛡️ Solución Propuesta: Sistema de Fallback Multinivel

### Nivel 1: Redis Cache (YA EXISTE)
```go
// search_service.go línea 48-57
if result, found := s.cacheRepo.GetSearchResults(ctx, req); found {
    return s.buildSearchResponse(result, req, true, time.Since(startTime)), nil
}
```

### Nivel 2: Orders API Fallback (IMPLEMENTAR)
```go
// Cuando Solr falla, intentar Orders API directo
if err != nil {
    // Try fallback to Orders API
    if s.ordersClient != nil {
        result, err := s.fallbackToOrdersAPI(ctx, req)
        if err == nil {
            return result, nil
        }
    }
    return nil, err
}
```

### Nivel 3: Cache Stale (IMPLEMENTAR)
```go
// Si todo falla, retornar cache viejo (stale) si existe
if err != nil {
    if stale, found := s.cacheRepo.GetStaleResults(ctx, req); found {
        s.logger.Warn("Returning stale cache due to service failure")
        return s.buildSearchResponse(stale, req, true, time.Since(startTime)), nil
    }
    return nil, err
}
```

### Nivel 4: Frontend Cache (IMPLEMENTAR)
```typescript
// Local Storage cache con TTL
const CACHE_DURATION = 5 * 60 * 1000 // 5 minutos

class OrdersCache {
  getRecentOrders(userId: number): OrderSearchResult[] | null
  setRecentOrders(userId: number, orders: OrderSearchResult[]): void
  getUserOrders(userId: number, page: number): SearchResponse | null
  setUserOrders(userId: number, page: number, response: SearchResponse): void
}
```

## 🔧 Plan de Implementación

### Fase 1: Backend Fallback (Prioridad ALTA)
**Archivo**: `search-api/internal/services/search_service.go`

**Cambios**:
1. Agregar `ordersClient` al SearchService
2. Implementar método `fallbackToOrdersAPI()`
3. Modificar método `Search()` para usar fallback
4. Agregar método `GetStaleCache()` en cache_repository

**Beneficios**:
- ✅ Sistema sigue funcionando con Solr caído
- ✅ Datos frescos desde Orders API
- ✅ Sin cambios en frontend

### Fase 2: Frontend Cache (Prioridad MEDIA)
**Archivo**: `crypto-trading-app/lib/orders-cache.ts`

**Cambios**:
1. Crear `OrdersCache` class
2. Guardar en localStorage con timestamps
3. Implementar invalidación por TTL
4. Usar cache antes de llamar API

**Beneficios**:
- ✅ Instant loading de órdenes recientes
- ✅ Funciona offline (datos viejos)
- ✅ Reduce llamadas al backend

### Fase 3: Fallback Directo a Orders API (Prioridad BAJA)
**Archivo**: `crypto-trading-app/lib/api.ts`

**Cambios**:
1. Agregar métodos directos a Orders API
2. Modificar `search-api.ts` para usar fallback
3. Implementar circuit breaker pattern

**Beneficios**:
- ✅ Redundancia completa
- ✅ No depende de Search API
- ✅ Resistente a múltiples fallas

## 📈 Matriz de Resiliencia Propuesta

| Servicio Caído | Cache Redis | Orders API | Frontend Cache | Resultado |
|----------------|-------------|------------|----------------|-----------|
| Solr | ✅ Hit | ✅ Fallback | ✅ Disponible | **Funciona** |
| Solr | ❌ Miss | ✅ Fallback | ✅ Disponible | **Funciona** |
| Solr | ❌ Miss | ❌ Error | ✅ Disponible | **Degradado** (cache viejo) |
| Search API | N/A | ✅ Directo | ✅ Disponible | **Funciona** |
| Search API | N/A | ❌ Error | ✅ Disponible | **Degradado** (cache viejo) |
| Todo | ❌ | ❌ | ❌ | **Fallo total** |

## 🎯 Métricas de Éxito

### Antes (Estado Actual)
- **Disponibilidad**: ~95% (depende de Solr)
- **MTTR**: Variable (depende de recuperación de Solr)
- **Experiencia degradada**: ❌ No (todo o nada)

### Después (Con Fallbacks)
- **Disponibilidad**: >99.9% (múltiples fallbacks)
- **MTTR**: <1s (fallback automático)
- **Experiencia degradada**: ✅ Sí (datos stale vs vacío)

## 🚀 Próximos Pasos

1. **Inmediato**: Implementar fallback a Orders API en backend
2. **Corto plazo**: Agregar cache stale support
3. **Mediano plazo**: Implementar frontend cache
4. **Largo plazo**: Circuit breaker y health checks

## 📝 Notas de Implementación

### Consideraciones de Cache Stale
- **TTL Normal**: 5 minutos
- **TTL Stale**: 30 minutos
- **Marker de "stale"**: Incluir timestamp en respuesta

### Consideraciones de Orders API Fallback
- **Limitaciones**: No tiene búsqueda fulltext
- **Solución**: Filtrar en memoria por user_id
- **Performance**: Aceptable para sets pequeños (<1000 órdenes)

### Consideraciones de Frontend Cache
- **Storage**: localStorage (limit 5-10MB)
- **Estrategia**: Cache solo órdenes recientes (últimas 50)
- **Invalidación**: Automática por TTL + manual en nuevas órdenes
