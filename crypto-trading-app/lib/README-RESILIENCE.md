# Sistema de Resiliencia para Búsqueda de Órdenes

## 🛡️ Descripción

El sistema de búsqueda de órdenes implementa un **fallback multinivel en el backend** para garantizar la disponibilidad de datos incluso cuando Solr está caído o saturado.

## 🏗️ Arquitectura

```
┌──────────────────────────────────────────────────────────┐
│ Frontend solicita órdenes                               │
└────────────────────┬─────────────────────────────────────┘
                     │
                     ↓
┌──────────────────────────────────────────────────────────┐
│ Search API - Backend                                     │
├──────────────────────────────────────────────────────────┤
│ 1. Cache Fresh (Redis/Memcached - 5 min)                │
│    ✅ Hit → Retornar inmediatamente                     │
│    ❌ Miss → Continuar                                  │
├──────────────────────────────────────────────────────────┤
│ 2. Solr Search                                           │
│    ✅ Success → Retornar + guardar en cache            │
│    ❌ Error → Continuar al fallback                     │
├──────────────────────────────────────────────────────────┤
│ 3. Cache Stale (30 min)                                 │
│    ✅ Disponible → Retornar datos viejos                │
│    ❌ No disponible → Continuar                        │
├──────────────────────────────────────────────────────────┤
│ 4. Orders API (Base de Datos directa)                   │
│    ✅ Success → Retornar + guardar en cache            │
│    ❌ Error → Lanzar error                              │
└──────────────────────────────────────────────────────────┘
```

## 📦 Componentes

### Backend (Search API)

El backend maneja todo el cache y fallback:

1. **Cache Fresh**: Redis/Memcached con TTL de 5 minutos
2. **Cache Stale**: Búsqueda de cache hasta 30 minutos de antigüedad
3. **Fallback a Orders API**: Consulta directa a la base de datos cuando Solr falla

### Frontend

El frontend hace llamadas directas al Search API sin cache local:

- `searchOrders()`: Llamada directa al API
- `getRecentOrders()`: Llamada directa al API
- Sin cache en LocalStorage
- Sin revalidación en background

## 📊 Escenarios de Fallo y Respuestas

| Escenario | Cache Fresh | Solr | Cache Stale | Orders API | Resultado |
|-----------|-------------|------|-------------|------------|-----------|
| **Normal** | ❌ Miss | ✅ OK | N/A | ⏭️ Skip | **Datos frescos desde Solr** |
| **Cache hit** | ✅ Hit | ⏭️ Skip | N/A | ⏭️ Skip | **Datos desde cache** |
| **Solr saturado** | ❌ Miss | ❌ 429 | ✅ Disponible | ⏭️ Skip | **Datos stale (hasta 30 min)** |
| **Solr caído** | ❌ Miss | ❌ Error | ✅ Disponible | ⏭️ Skip | **Datos stale (hasta 30 min)** |
| **Sin cache** | ❌ Miss | ❌ Error | ❌ No disponible | ✅ OK | **Datos desde DB (Orders API)** |
| **Todo caído** | ❌ Miss | ❌ Error | ❌ No disponible | ❌ Error | **❌ Error** |

## 🚀 Beneficios

### Disponibilidad
- **99.9%+ uptime** para datos de órdenes
- Funciona incluso con Solr completamente caído
- Fallback automático a base de datos

### Simplicidad
- **Sin cache en frontend**: Evita datos desactualizados
- **Lógica centralizada**: Todo el cache en el backend
- **Menos complejidad**: Frontend solo consume el API

## ⚠️ Notas

- El frontend **no cachea nada** para evitar mostrar datos incorrectos
- Todo el cache y fallback está en el **backend (Search API)**
- Si el Search API está caído, el frontend mostrará error (esperado)
