# Guía de Diagnóstico: Órdenes no aparecen en búsqueda

## Flujo completo de una orden

1. **Frontend** → `orders-api`: POST `/api/v1/orders` (crear orden)
2. **orders-api**: Guarda en MongoDB
3. **orders-api**: Publica evento `orders.created` a RabbitMQ
4. **search-api consumer**: Recibe evento de RabbitMQ
5. **search-api**: Obtiene orden completa desde `orders-api`
6. **search-api**: Indexa orden en Solr
7. **Frontend** → `search-api`: POST `/api/v1/search` (buscar órdenes)

---

## Logs a revisar (en orden)

### 1. **orders-api** - Verificar que la orden se creó

**Comando:**
```bash
docker logs cryptosim-orders-api --tail 100 -f
```

**Qué buscar:**
- ✅ `POST /api/v1/orders` - Request recibido
- ✅ `Order created successfully` o similar - Orden guardada en MongoDB
- ✅ `Published event: orders.created for order <order_id>` - Evento publicado a RabbitMQ
- ❌ `Warning: failed to publish order created event` - Error al publicar evento

**Si hay error al publicar:**
- Verificar conexión a RabbitMQ
- Verificar que el exchange `orders.events` existe

---

### 2. **orders-api** - Verificar que la orden está en MongoDB

**Comando:**
```bash
docker exec -it cryptosim-orders-mongo mongosh
use orders_db
db.orders.find().sort({created_at: -1}).limit(5).pretty()
```

**Qué buscar:**
- Verificar que la orden existe con el `user_id` correcto
- Verificar que tiene `status`, `type`, `crypto_symbol`, etc.

---

### 3. **RabbitMQ** - Verificar que el evento llegó a la cola

**Comando:**
```bash
docker exec -it cryptosim-rabbitmq rabbitmqctl list_queues name messages messages_ready messages_unacknowledged
```

**Qué buscar:**
- Verificar que hay mensajes en las colas del search-api
- Si hay mensajes `unacknowledged`, el consumer puede estar fallando

**Ver mensajes en la cola:**
```bash
docker exec -it cryptosim-rabbitmq rabbitmqctl list_exchanges
docker exec -it cryptosim-rabbitmq rabbitmqctl list_bindings
```

---

### 4. **search-api** - Verificar que el consumer recibió el evento

**Comando:**
```bash
docker logs cryptosim-search-api --tail 200 -f
```

**Qué buscar:**
- ✅ `RabbitMQ consumer started successfully` - Consumer iniciado
- ✅ `Received legacy order event` - Evento recibido
- ✅ `order_id: <order_id>` - ID de la orden recibida
- ✅ `Order event processed via legacy flow` - Evento procesado
- ❌ `Failed to unmarshal message` - Error al parsear el mensaje
- ❌ `Failed to sync order from legacy event` - Error al sincronizar

**Si no aparece "Received legacy order event":**
- El consumer puede no estar escuchando la cola correcta
- El routing key puede no coincidir
- Verificar bindings en RabbitMQ

---

### 5. **search-api** - Verificar que obtuvo la orden desde orders-api

**Comando:**
```bash
docker logs cryptosim-search-api --tail 200 | grep -i "order\|fetch\|sync"
```

**Qué buscar:**
- ✅ `Fetching order from orders-api` - Intentando obtener orden
- ✅ `Order fetched successfully` - Orden obtenida
- ✅ `Syncing order` - Sincronizando orden
- ❌ `Failed to fetch order from orders-api` - Error al obtener orden
- ❌ `order not found` - Orden no encontrada en orders-api

**Si falla al obtener la orden:**
- Verificar que `orders-api` está corriendo
- Verificar la URL de `orders-api` en la configuración de `search-api`
- Verificar que el `order_id` es correcto

---

### 6. **search-api** - Verificar que indexó en Solr

**Comando:**
```bash
docker logs cryptosim-search-api --tail 200 | grep -i "solr\|index"
```

**Qué buscar:**
- ✅ `Indexing order` - Indexando orden
- ✅ `Order indexed successfully` - Orden indexada
- ❌ `Failed to index order` - Error al indexar
- ❌ `Solr error` - Error de Solr

**Si falla al indexar:**
- Verificar que Solr está corriendo
- Verificar la configuración de Solr en `search-api`
- Verificar que el schema de Solr está correcto

---

### 7. **Solr** - Verificar que la orden está indexada

**Comando:**
```bash
# Verificar que Solr está corriendo
docker exec -it cryptosim-solr curl "http://localhost:8983/solr/orders/select?q=*:*&rows=5&wt=json"

# Buscar por user_id específico
docker exec -it cryptosim-solr curl "http://localhost:8983/solr/orders/select?q=user_id:<TU_USER_ID>&wt=json"
```

**Qué buscar:**
- Verificar que hay documentos en el índice
- Verificar que la orden aparece con el `user_id` correcto
- Verificar que los campos están correctamente indexados

---

### 8. **search-api** - Verificar la búsqueda

**Comando:**
```bash
docker logs cryptosim-search-api --tail 100 | grep -i "search\|query"
```

**Qué buscar:**
- ✅ `Search completed` - Búsqueda completada
- ✅ `results: <número>` - Número de resultados
- ✅ `total: <número>` - Total de resultados
- ❌ `Search execution failed` - Error en la búsqueda
- ❌ `search query failed` - Error en la query de Solr

**Verificar la query que se envía:**
- Revisar los logs para ver la query de Solr generada
- Verificar que los filtros (especialmente `user_id`) están correctos

---

### 9. **Frontend** - Verificar la request

**En el navegador (DevTools → Network):**
- Verificar que la request a `/api/v1/search` se está haciendo
- Verificar el payload de la request (especialmente `user_id`)
- Verificar la respuesta (debe incluir `total` y `results`)

**Qué buscar en la respuesta:**
- `total: 0` - No hay resultados (pero la búsqueda funcionó)
- `results: []` - Array vacío
- Error 500 - Error en el servidor

---

## Checklist rápido

- [ ] La orden se creó en `orders-api` (status 201)
- [ ] La orden existe en MongoDB
- [ ] El evento se publicó a RabbitMQ
- [ ] El consumer de `search-api` recibió el evento
- [ ] `search-api` obtuvo la orden desde `orders-api`
- [ ] La orden se indexó en Solr
- [ ] La orden aparece en Solr con el `user_id` correcto
- [ ] La búsqueda desde el frontend incluye el `user_id` correcto
- [ ] La respuesta de `search-api` tiene los resultados correctos

---

## Problemas comunes y soluciones

### Problema: El consumer no recibe eventos
**Solución:**
- Verificar que el consumer está corriendo: `docker ps | grep search-api`
- Verificar los bindings de RabbitMQ
- Verificar que el routing key coincide (`orders.created`)

### Problema: Error al obtener orden desde orders-api
**Solución:**
- Verificar que `orders-api` está corriendo
- Verificar la URL en la configuración de `search-api`
- Verificar que el `order_id` es correcto (debe ser ObjectID de MongoDB)

### Problema: Error al indexar en Solr
**Solución:**
- Verificar que Solr está corriendo
- Verificar el schema de Solr
- Verificar que los campos requeridos están presentes

### Problema: La búsqueda no encuentra órdenes
**Solución:**
- Verificar que el `user_id` en la búsqueda coincide con el de las órdenes
- Verificar que los filtros no están excluyendo las órdenes
- Verificar que Solr tiene las órdenes indexadas

---

## Comandos útiles

```bash
# Ver todos los logs de search-api
docker logs cryptosim-search-api --tail 500

# Ver logs en tiempo real
docker logs cryptosim-search-api -f

# Ver logs de orders-api
docker logs cryptosim-orders-api --tail 500 -f

# Verificar estado de los servicios
docker ps | grep -E "orders|search"

# Reiniciar search-api
docker restart cryptosim-search-api

# Ver configuración de search-api
docker exec -it cryptosim-search-api env | grep -E "ORDERS|SOLR|RABBITMQ"
```

