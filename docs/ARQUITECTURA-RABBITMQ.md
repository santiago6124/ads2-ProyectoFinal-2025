# Arquitectura de Mensajería con RabbitMQ

## Descripción General

Este documento describe la arquitectura completa de mensajería asíncrona implementada con RabbitMQ en CryptoSim, incluyendo todos los exchanges, queues, routing keys, publishers y consumers.

## Diagrama General

```
                                    ┌─────────────────┐
                                    │   RabbitMQ      │
                                    │   Server        │
                                    └────────┬────────┘
                                             │
                    ┌────────────────────────┼────────────────────────┐
                    │                        │                        │
         ┌──────────▼─────────┐   ┌─────────▼────────┐   ┌──────────▼─────────┐
         │  orders.events     │   │ balance.request  │   │ balance.response   │
         │  (Topic Exchange)  │   │ (Topic Exchange) │   │ (Topic Exchange)   │
         └──────────┬─────────┘   └─────────┬────────┘   └──────────┬─────────┘
                    │                       │                        │
        ┌───────────┼────────────┐          │                        │
        │           │            │          │                        │
        ▼           ▼            ▼          ▼                        ▼
  ┌──────────┐ ┌─────────┐ ┌─────────┐ ┌──────────┐        ┌──────────────────┐
  │portfolio.│ │ search. │ │ audit.  │ │ balance. │        │balance.response. │
  │ updates  │ │  sync   │ │  log    │ │ request  │        │    portfolio     │
  │ (Queue)  │ │ (Queue) │ │ (Queue) │ │ (Queue)  │        │     (Queue)      │
  └────┬─────┘ └────┬────┘ └────┬────┘ └────┬─────┘        └────────┬─────────┘
       │            │           │           │                        │
       ▼            ▼           ▼           ▼                        ▼
┌────────────┐ ┌────────────┐ ┌────────┐ ┌───────────┐      ┌──────────────┐
│Portfolio   │ │Search      │ │Audit   │ │Users      │      │Portfolio     │
│Consumer    │ │Consumer    │ │Consumer│ │Worker     │      │Balance Client│
│(Go)        │ │(Go)        │ │(Go)    │ │(Go)       │      │(Go)          │
└────────────┘ └────────────┘ └────────┘ └───────────┘      └──────────────┘
```

---

## 1. EXCHANGES

### 1.1 orders.events (Topic Exchange)

**Propósito**: Publicar eventos del ciclo de vida de órdenes

**Configuración**:
```go
err := channel.ExchangeDeclare(
    "orders.events",  // Name
    "topic",          // Type
    true,             // Durable (persiste reinicio)
    false,            // Auto-delete
    false,            // Internal
    false,            // No-wait
    nil,              // Arguments
)
```

**Routing Keys**:
- `orders.created` - Orden creada (status: pending)
- `orders.executed` - Orden ejecutada exitosamente
- `orders.cancelled` - Orden cancelada por usuario
- `orders.failed` - Orden falló al ejecutar

**Publisher**: Orders API

**Consumers**:
- Portfolio API (solo `orders.executed`)
- Search API (todas las keys)
- Audit API (todas las keys) [futuro]

**Características**:
- ✅ Durable: Sobrevive a reinicios de RabbitMQ
- ✅ Persistent messages: Mensajes guardados en disco
- ✅ Fanout pattern: Múltiples consumers independientes
- ✅ Dead Letter Queue: Mensajes fallidos van a DLQ

---

### 1.2 balance.request.exchange (Topic Exchange)

**Propósito**: Solicitudes asíncronas de balance de usuario

**Configuración**:
```go
err := channel.ExchangeDeclare(
    "balance.request.exchange",  // Name
    "topic",                      // Type
    true,                         // Durable
    false, false, false, nil,
)
```

**Routing Key**:
- `balance.request`

**Publisher**: Portfolio API

**Consumer**: Users Worker (balance worker)

**Patrón**: Request-Reply asíncrono

---

### 1.3 balance.response.exchange (Topic Exchange)

**Propósito**: Respuestas a solicitudes de balance

**Configuración**:
```go
err := channel.ExchangeDeclare(
    "balance.response.exchange",  // Name
    "topic",                       // Type
    true,                          // Durable
    false, false, false, nil,
)
```

**Routing Keys**:
- `balance.response.portfolio` - Respuestas para Portfolio API
- `balance.response.orders` - Respuestas para Orders API [futuro]

**Publisher**: Users Worker

**Consumers**:
- Portfolio API (balance client)
- Orders API [futuro]

**TTL**: Mensajes expiran en 60 segundos

---

## 2. QUEUES

### 2.1 portfolio.updates

**Propósito**: Recibir eventos de órdenes ejecutadas para actualizar portfolios

**Declaración**:
```go
queue, err := channel.QueueDeclare(
    "portfolio.updates",  // Name
    true,                 // Durable
    false,                // Delete when unused
    false,                // Exclusive
    false,                // No-wait
    amqp.Table{
        "x-message-ttl":           3600000,  // 1 hora TTL
        "x-dead-letter-exchange":  "dlx",    // DLX exchange
        "x-dead-letter-routing-key": "portfolio.failed",
    },
)
```

**Binding**:
```go
err := channel.QueueBind(
    "portfolio.updates",  // Queue name
    "orders.executed",    // Routing key (SOLO ejecutadas)
    "orders.events",      // Exchange
    false,
    nil,
)
```

**Consumer**: Portfolio API

**Procesamiento**:
1. Deserializar evento OrderEvent
2. Obtener precio actual (Market Data API)
3. Actualizar holdings (add/remove según buy/sell)
4. Recalcular 30+ métricas
5. Guardar en MongoDB
6. ACK mensaje

**Configuración Consumer**:
```go
msgs, err := channel.Consume(
    "portfolio.updates",      // Queue
    "portfolio-consumer",     // Consumer tag
    false,                    // Auto-ack (manual ACK)
    false,                    // Exclusive
    false,                    // No-local
    false,                    // No-wait
    nil,
)
```

**Manejo de errores**:
- Error recuperable → NACK con requeue=true
- Error permanente → NACK con requeue=false (va a DLQ)
- Éxito → ACK

---

### 2.2 search.sync

**Propósito**: Sincronizar órdenes en Apache Solr para búsqueda

**Declaración**:
```go
queue, err := channel.QueueDeclare(
    "search.sync",
    true,
    false,
    false,
    false,
    amqp.Table{
        "x-message-ttl":          600000,  // 10 minutos TTL
        "x-dead-letter-exchange": "dlx",
    },
)
```

**Bindings** (múltiples routing keys):
```go
routingKeys := []string{
    "orders.created",
    "orders.executed",
    "orders.cancelled",
    "orders.failed",
}

for _, key := range routingKeys {
    channel.QueueBind(
        "search.sync",
        key,
        "orders.events",
        false,
        nil,
    )
}
```

**Consumer**: Search API

**Procesamiento**:
1. Deserializar evento OrderEvent
2. Obtener orden completa (Orders API)
3. Indexar/actualizar en Solr
4. Invalidar caché relacionado
5. ACK mensaje

---

### 2.3 balance.request

**Propósito**: Queue para solicitudes de balance

**Declaración**:
```go
queue, err := channel.QueueDeclare(
    "balance.request",
    true,
    false,
    false,
    false,
    amqp.Table{
        "x-message-ttl": 60000,  // 60 segundos
    },
)
```

**Binding**:
```go
channel.QueueBind(
    "balance.request",
    "balance.request",
    "balance.request.exchange",
    false,
    nil,
)
```

**Consumer**: Users Worker (balance-worker)

**Procesamiento**:
1. Deserializar BalanceRequest
2. Buscar usuario en MySQL
3. Extraer balance actual
4. Crear BalanceResponse con correlation_id
5. Publicar en balance.response.exchange
6. ACK mensaje

---

### 2.4 balance.response.portfolio

**Propósito**: Queue exclusiva para respuestas de balance a Portfolio API

**Declaración**:
```go
queue, err := channel.QueueDeclare(
    "balance.response.portfolio",
    false,                // Non-durable (temporal)
    false,
    true,                 // Exclusive (solo esta conexión)
    false,
    amqp.Table{
        "x-message-ttl": 60000,  // 60 segundos
        "x-expires":     120000, // Queue expira si no se usa 2 min
    },
)
```

**Binding**:
```go
channel.QueueBind(
    "balance.response.portfolio",
    "balance.response.portfolio",
    "balance.response.exchange",
    false,
    nil,
)
```

**Consumer**: Portfolio API (balance client)

**Procesamiento**:
1. Recibir BalanceResponse
2. Matchear correlation_id con request
3. Retornar balance al caller
4. ACK mensaje

**Timeout**: 5 segundos esperando respuesta

---

## 3. MENSAJES

### 3.1 OrderEvent

**Publisher**: Orders API

**Exchange**: orders.events

**Routing Keys**: orders.created, orders.executed, orders.cancelled, orders.failed

**Estructura**:
```go
type OrderEvent struct {
    EventType    string    `json:"event_type"`     // "created", "executed", "cancelled", "failed"
    OrderID      string    `json:"order_id"`       // MongoDB ObjectID
    UserID       int64     `json:"user_id"`
    Type         string    `json:"type"`           // "buy", "sell"
    CryptoSymbol string    `json:"crypto_symbol"`  // "BTC", "ETH", etc
    Quantity     string    `json:"quantity"`       // Decimal as string
    Price        string    `json:"price"`          // Decimal as string
    TotalAmount  string    `json:"total_amount"`   // Decimal as string
    Fee          string    `json:"fee"`            // Decimal as string
    Timestamp    time.Time `json:"timestamp"`
}
```

**Ejemplo JSON**:
```json
{
  "event_type": "executed",
  "order_id": "673b5f8a9e1234567890abcd",
  "user_id": 123,
  "type": "buy",
  "crypto_symbol": "BTC",
  "quantity": "0.001",
  "price": "50000.00",
  "total_amount": "50.00",
  "fee": "0.05",
  "timestamp": "2025-11-14T10:30:05Z"
}
```

**Código de publicación**:
```go
event := OrderEvent{
    EventType:    "executed",
    OrderID:      order.ID.Hex(),
    UserID:       order.UserID,
    Type:         order.Type,
    CryptoSymbol: order.CryptoSymbol,
    Quantity:     order.Quantity.String(),
    Price:        order.Price.String(),
    TotalAmount:  order.TotalAmount.String(),
    Fee:          order.Fee.String(),
    Timestamp:    time.Now(),
}

eventJSON, _ := json.Marshal(event)

err := channel.Publish(
    "orders.events",      // Exchange
    "orders.executed",    // Routing key
    false,                // Mandatory
    false,                // Immediate
    amqp.Publishing{
        ContentType:  "application/json",
        Body:         eventJSON,
        DeliveryMode: amqp.Persistent,  // Mensaje persistente
        Timestamp:    time.Now(),
        MessageId:    uuid.New().String(),
    },
)
```

---

### 3.2 BalanceRequest

**Publisher**: Portfolio API

**Exchange**: balance.request.exchange

**Routing Key**: balance.request

**Estructura**:
```go
type BalanceRequest struct {
    CorrelationID string    `json:"correlation_id"`  // UUID para matchear respuesta
    UserID        int64     `json:"user_id"`
    RequestedBy   string    `json:"requested_by"`    // "portfolio-api"
    Timestamp     time.Time `json:"timestamp"`
}
```

**Ejemplo JSON**:
```json
{
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 123,
  "requested_by": "portfolio-api",
  "timestamp": "2025-11-14T11:30:00Z"
}
```

**Código de publicación**:
```go
correlationID := uuid.New().String()

request := BalanceRequest{
    CorrelationID: correlationID,
    UserID:        userID,
    RequestedBy:   "portfolio-api",
    Timestamp:     time.Now(),
}

requestJSON, _ := json.Marshal(request)

err := channel.Publish(
    "balance.request.exchange",
    "balance.request",
    false, false,
    amqp.Publishing{
        ContentType:   "application/json",
        Body:          requestJSON,
        CorrelationId: correlationID,
        ReplyTo:       "balance.response.portfolio",  // Para respuesta
        Expiration:    "60000",  // 60 segundos TTL
        Timestamp:     time.Now(),
    },
)
```

---

### 3.3 BalanceResponse

**Publisher**: Users Worker

**Exchange**: balance.response.exchange

**Routing Key**: balance.response.portfolio

**Estructura**:
```go
type BalanceResponse struct {
    CorrelationID string    `json:"correlation_id"`  // MISMO que request
    UserID        int64     `json:"user_id"`
    Balance       string    `json:"balance"`         // Decimal as string
    Currency      string    `json:"currency"`        // "USD"
    Success       bool      `json:"success"`
    Error         string    `json:"error,omitempty"`
    Timestamp     time.Time `json:"timestamp"`
}
```

**Ejemplo JSON (éxito)**:
```json
{
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 123,
  "balance": "99949.95",
  "currency": "USD",
  "success": true,
  "error": null,
  "timestamp": "2025-11-14T11:30:01Z"
}
```

**Ejemplo JSON (error)**:
```json
{
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 999,
  "balance": "0",
  "currency": "USD",
  "success": false,
  "error": "User not found",
  "timestamp": "2025-11-14T11:30:01Z"
}
```

**Código de publicación**:
```go
response := BalanceResponse{
    CorrelationID: request.CorrelationID,  // Copiar de request
    UserID:        user.ID,
    Balance:       user.InitialBalance.String(),
    Currency:      "USD",
    Success:       true,
    Timestamp:     time.Now(),
}

responseJSON, _ := json.Marshal(response)

err := channel.Publish(
    "balance.response.exchange",
    replyTo,  // De msg.ReplyTo (ej: "balance.response.portfolio")
    false, false,
    amqp.Publishing{
        ContentType:   "application/json",
        Body:          responseJSON,
        CorrelationId: request.CorrelationID,
        Timestamp:     time.Now(),
    },
)
```

---

## 4. PATRONES DE MENSAJERÍA

### 4.1 Publish-Subscribe (Fan-out)

**Caso**: Evento orders.executed

```
Orders API (Publisher)
    │
    └──> orders.events (Exchange)
              │
              ├──> portfolio.updates (Queue) ──> Portfolio Consumer
              │
              └──> search.sync (Queue) ──> Search Consumer
```

**Características**:
- Un mensaje → Múltiples consumers independientes
- Cada consumer recibe copia del mensaje
- Consumers no se bloquean entre sí
- Ideal para eventos de dominio

**Ventajas**:
- ✅ Desacoplamiento total
- ✅ Escalabilidad horizontal (múltiples workers por queue)
- ✅ Tolerancia a fallos (si Portfolio cae, Search sigue funcionando)

---

### 4.2 Request-Reply Asíncrono

**Caso**: Portfolio solicita balance a Users

```
Portfolio API (Requester)
    │
    └──[1] Publish request ──> balance.request (Queue)
                                      │
                                      └──> Users Worker (Replier)
                                              │
                      [2] Consume response <──┘
                                              │
    Portfolio API <──────────────────────────┘
         │
         └─[3] Match correlation_id
```

**Flujo**:
1. Portfolio genera `correlation_id` único (UUID)
2. Publica request con `ReplyTo="balance.response.portfolio"`
3. Users Worker consume request
4. Users Worker publica response con mismo `correlation_id`
5. Portfolio consume de su queue exclusiva
6. Portfolio matchea `correlation_id` y retorna resultado

**Ventajas**:
- ✅ No bloquea hilo principal
- ✅ Timeout configurable (5 segundos)
- ✅ Escalable (múltiples requesters simultáneos)

**Desventajas**:
- ⚠️ Complejidad mayor que HTTP sync
- ⚠️ Necesita manejo de timeouts
- ⚠️ Correlation ID tracking

---

### 4.3 Competing Consumers

**Caso**: Múltiples instancias de Portfolio API

```
orders.events (Exchange)
    │
    └──> portfolio.updates (Queue)
              │
              ├──> Portfolio Instance 1 (Consumer)
              │
              ├──> Portfolio Instance 2 (Consumer)
              │
              └──> Portfolio Instance 3 (Consumer)
```

**Round-robin**: RabbitMQ distribuye mensajes equitativamente

**Prefetch Count**:
```go
channel.Qos(
    1,     // Prefetch count (1 mensaje a la vez)
    0,     // Prefetch size
    false, // Global
)
```

**Ventajas**:
- ✅ Load balancing automático
- ✅ Escalabilidad horizontal
- ✅ Alta disponibilidad (si un worker cae, otros continúan)

---

## 5. CONFIGURACIÓN DE CONEXIÓN

### 5.1 RabbitMQ Connection Manager

```go
// shared/rabbitmq/connection.go
type RabbitMQConnection struct {
    conn    *amqp.Connection
    channel *amqp.Channel
    url     string
}

func NewRabbitMQConnection(url string) (*RabbitMQConnection, error) {
    conn, err := amqp.Dial(url)
    if err != nil {
        return nil, err
    }

    channel, err := conn.Channel()
    if err != nil {
        return nil, err
    }

    // Configurar QoS
    err = channel.Qos(
        1,     // Prefetch count
        0,     // Prefetch size
        false, // Global
    )

    return &RabbitMQConnection{
        conn:    conn,
        channel: channel,
        url:     url,
    }, nil
}

func (r *RabbitMQConnection) Reconnect() error {
    r.Close()

    conn, err := amqp.Dial(r.url)
    if err != nil {
        return err
    }

    channel, err := conn.Channel()
    if err != nil {
        return err
    }

    r.conn = conn
    r.channel = channel

    return nil
}

func (r *RabbitMQConnection) Close() {
    if r.channel != nil {
        r.channel.Close()
    }
    if r.conn != nil {
        r.conn.Close()
    }
}
```

### 5.2 Auto-Reconnect en Consumer

```go
func (c *Consumer) StartWithReconnect() {
    for {
        err := c.Start()
        if err != nil {
            log.Error("Consumer error, reconnecting in 5s", err)
            time.Sleep(5 * time.Second)

            err = c.rabbitConn.Reconnect()
            if err != nil {
                log.Error("Reconnect failed", err)
                continue
            }

            log.Info("Reconnected successfully")
        }
    }
}
```

---

## 6. DEAD LETTER QUEUE (DLQ)

### Configuración de DLX

```go
// Declarar DLX exchange
channel.ExchangeDeclare(
    "dlx",      // Dead Letter Exchange
    "topic",
    true,
    false, false, false, nil,
)

// Declarar DLQ queue
channel.QueueDeclare(
    "dlq.portfolio",
    true,
    false, false, false,
    nil,
)

// Bind DLQ
channel.QueueBind(
    "dlq.portfolio",
    "portfolio.failed",
    "dlx",
    false, nil,
)
```

### Mensajes van a DLQ cuando:
1. **NACK con requeue=false**
2. **TTL expirado**
3. **Max retries alcanzado**
4. **Queue rechaza mensaje** (queue llena)

### Monitoreo de DLQ

```bash
# Ver mensajes en DLQ
rabbitmqadmin get queue=dlq.portfolio count=10

# Purgar DLQ
rabbitmqadmin purge queue=dlq.portfolio

# Requeue mensajes (manual)
# Desde Management UI: Dead Letter Queue → Get Messages → Requeue
```

---

## 7. MONITOREO Y MÉTRICAS

### Management UI

**URL**: http://localhost:15672
**Usuario**: guest
**Password**: guest

**Información visible**:
- ✅ Exchanges y sus bindings
- ✅ Queues (mensajes ready, unacked, total)
- ✅ Connections activas
- ✅ Channels abiertos
- ✅ Consumers por queue
- ✅ Message rates (publish/deliver/ack)
- ✅ Memory usage

### Métricas Clave

**Queues**:
```
Ready: Mensajes esperando ser consumidos
Unacked: Mensajes entregados pero no ACKed
Total: Ready + Unacked
Rate: Mensajes/segundo (in, out, ack)
```

**Consumers**:
```
Consumer count: Número de consumers activos
Prefetch count: Mensajes prefetched por consumer
```

### Alertas Recomendadas

1. **Queue depth > 1000**: Consumers lentos o caídos
2. **Unacked > 100**: Consumer no está haciendo ACK
3. **Connection drops**: Problemas de red o crashes
4. **Message age > 5min**: Mensajes estancados
5. **DLQ > 10 messages**: Errores persistentes

---

## 8. BEST PRACTICES IMPLEMENTADAS

### 8.1 Idempotencia
```go
// Verificar si mensaje ya fue procesado
var processed ProcessedMessage
result := db.Where("message_id = ?", msg.MessageId).First(&processed)
if result.RowsAffected > 0 {
    msg.Ack(false)  // Ya procesado, ACK sin re-procesar
    return nil
}

// Procesar mensaje...

// Guardar como procesado
db.Create(&ProcessedMessage{
    MessageID:   msg.MessageId,
    ProcessedAt: time.Now(),
})
```

### 8.2 Manual ACK
```go
// NUNCA usar auto-ack en producción
msgs, _ := channel.Consume(
    queueName,
    consumerTag,
    false,  // Auto-ack = FALSE
    false, false, false, nil,
)

for msg := range msgs {
    err := processMessage(msg)
    if err != nil {
        msg.Nack(false, true)  // Requeue
    } else {
        msg.Ack(false)  // Success
    }
}
```

### 8.3 Prefetch Limit
```go
// Limitar mensajes en tránsito
channel.Qos(1, 0, false)  // 1 mensaje a la vez
```

### 8.4 Message TTL
```go
// Evitar acumulación infinita
amqp.Table{
    "x-message-ttl": 3600000,  // 1 hora
}
```

### 8.5 Persistent Messages
```go
amqp.Publishing{
    DeliveryMode: amqp.Persistent,  // Sobrevive a reinicio
    ...
}
```

---

## 9. DOCKER COMPOSE CONFIGURACIÓN

```yaml
rabbitmq:
  image: rabbitmq:3.12-management-alpine
  container_name: shared-rabbitmq
  hostname: rabbitmq
  ports:
    - "5672:5672"    # AMQP
    - "15672:15672"  # Management UI
  environment:
    RABBITMQ_DEFAULT_USER: guest
    RABBITMQ_DEFAULT_PASS: guest
    RABBITMQ_DEFAULT_VHOST: /
  volumes:
    - rabbitmq_data:/var/lib/rabbitmq
  healthcheck:
    test: rabbitmq-diagnostics -q ping
    interval: 30s
    timeout: 10s
    retries: 5
  networks:
    - cryptosim_network
```

---

## 10. TROUBLESHOOTING

### Consumer no recibe mensajes

**Verificar**:
```bash
# ¿Queue tiene mensajes?
rabbitmqadmin list queues name messages

# ¿Consumer está conectado?
rabbitmqadmin list consumers queue_name consumer_tag

# ¿Binding correcto?
rabbitmqadmin list bindings source destination routing_key
```

### Mensajes acumulándose

**Causas**:
- Consumer muy lento (procesamiento pesado)
- Consumer caído (no hay workers)
- Prefetch count muy alto
- No hace ACK (mensajes quedan unacked)

**Soluciones**:
- Escalar horizontalmente (más consumers)
- Optimizar procesamiento
- Reducir prefetch count
- Verificar ACKs

### Connection drops frecuentes

**Causas**:
- Heartbeat timeout (default 60s)
- Network issues
- Memory pressure en RabbitMQ

**Soluciones**:
```go
// Configurar heartbeat
config := amqp.Config{
    Heartbeat: 10 * time.Second,
}
conn, _ := amqp.DialConfig(url, config)
```

### DLQ llenándose

**Investigar**:
```bash
# Ver mensajes en DLQ
rabbitmqadmin get queue=dlq.portfolio count=10

# Ver logs de consumer
docker logs portfolio-api | grep ERROR
```

**Causas comunes**:
- Mensajes malformados (JSON inválido)
- Errores de BD (constraint violations)
- Timeouts de APIs externas
- Bugs en consumer logic

---

## Resumen

- **3 Exchanges**: orders.events, balance.request, balance.response
- **4 Queues principales**: portfolio.updates, search.sync, balance.request, balance.response.portfolio
- **3 Patrones**: Publish-Subscribe, Request-Reply, Competing Consumers
- **Idempotencia**: Message ID tracking en BD
- **Durabilidad**: Exchanges, queues y mensajes persistentes
- **Resilencia**: Auto-reconnect, DLQ, prefetch limits
- **Monitoreo**: Management UI, métricas, alertas
- **Escalabilidad**: Horizontal scaling de consumers
