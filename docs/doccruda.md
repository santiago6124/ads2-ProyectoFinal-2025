Búsqueda de ordenes

Cliente envia busqueda, middleware jwt, handler recibe request (se genera cache key, y se verifica si esta en cache, sino busca en Solr y luego se cachea el resultado)

Luego Service Layer - sistema de cache multinivel, nivel 1 ccache (in-memory local), nivel2 memcached (distribuido), primero intenta cache local, ultra rapido, luego intenta memcached, y guarda en ccache para proxima vez.

Construccion de query de Solr con filtros y demas, y luego combinarlos

Ejecutar busqueda en Solr

sincronización con solr via rabbitmq consumer de eventos de ordenes, se declara exchange, la cola, y se consumen los mensajes, luego obtiene orden completa y luego se indexa el orden en Solr y se invalida cache


Compra-Venta

Cliente POST /orders, middleware JWT extrae user_id, handler recibe request con type, crypto_symbol, quantity, order_kind.

Service Layer valida: GET users-api/verify para verificar usuario existe, GET market-data-api/prices/:symbol para obtener precio actual, calcula total + fee (0.1%), verifica balance suficiente.

Crear orden en MongoDB con status pending, user_id, type, crypto_symbol, quantity, price, total_amount, fee, order_kind.

Response 201 Created con order_id.

EJECUCION: POST /orders/:id/execute, buscar orden en MongoDB, validar propietario y status=pending.

PUT users-api/balance con amount negativo (compra) o positivo (venta), transaction_type, order_id. Users API usa transaction MySQL, verifica idempotencia con order_id (evitar double-spend), actualiza balance, registra en balance_transactions.

Update orden en MongoDB: status=executed, executed_at=now.

Publicar evento RabbitMQ: exchange orders.events, routing key orders.executed, payload OrderEvent (event_type, order_id, user_id, type, crypto_symbol, quantity, price, total_amount, fee, timestamp). Mensaje persistente.

VENTA: validaciones adicionales, GET portfolio-api/holdings para verificar cantidad disponible, update balance suma en lugar de restar, portfolio actualiza holdings con FIFO (First In First Out) en cost basis.

Estados: pending → executed/cancelled/failed.

1. CLIENTE (Frontend/Postman)
   │
   │  HTTP POST /api/v1/orders
   │  (type: "buy", crypto_symbol: "BTC", quantity: 0.001)
   │
   ▼
2. ORDERS API (Puerto 8002)
   │  - Middleware JWT
   │  - Validaciones
   │  - Crear orden en MongoDB (status: pending)
   │  - Response 201 con order_id
   │
   │  HTTP POST /api/v1/orders/:id/execute
   │  (ejecutar la orden)
   │
   ▼
3. ORDERS API ejecuta la orden
   │  - Actualizar balance en Users API (HTTP)
   │  - Cambiar status a "executed"
   │  - PUBLICAR evento en RabbitMQ ✅
   │
   ▼
4. RABBITMQ orders.events
   │  - Evento: orders.executed
   │
   ├──> Portfolio API (consumer)
   │    - Actualiza holdings
   │    - Calcula métricas
   │
   └──> Search API (consumer)
        - Indexa en Solr

Registro y Login

POST /register con username, email, password. Validaciones: username 3-50 chars, email formato valido, password min 8 chars.

Verificar email unico en MySQL, verificar username unico, hash password con bcrypt cost 10.

INSERT en users table: username, email, password_hash, role=normal, initial_balance=100000.00, created_at, updated_at.

Response 201 con user data (sin password).

LOGIN: POST /login con email, password. Buscar usuario en MySQL por email, validar is_active=true, bcrypt.CompareHashAndPassword para verificar password.

Registrar login_attempts (email, ip, success, attempted_at), UPDATE users.last_login.

Generar JWT access token: claims (user_id, username, email, role, exp 15min, iat, nbf), signing method HS256, JWT_SECRET.

Generar refresh token: claims (user_id, type=refresh, exp 7 dias), INSERT en refresh_tokens table.

Cachear session en Redis: key session:{user_id}, TTL 900 segundos.

Response 200 con user + tokens (access_token, refresh_token, token_type=Bearer, expires_in=900).

MIDDLEWARE: extraer token de header Authorization: Bearer {token}, parsear JWT con JWT_SECRET, verificar expiracion, inyectar en context (user_id, username, email, role), handler accede con c.GetInt64("user_id").

REFRESH: POST /refresh con refresh_token, parsear JWT, verificar type=refresh, verificar no revoked en BD, generar nuevo access token, response con nuevo access_token.


Portfolio

Consumer RabbitMQ escucha orders.events con routing key orders.executed, queue portfolio.updates, exchange topic durable.

Cuando recibe evento: deserializar OrderEvent, buscar o crear portfolio en MongoDB (user_id, total_value, total_invested, holdings array).

GET market-data-api/prices/:symbol para precio actual.

BUY: buscar holding existente por symbol, si no existe crear nuevo con quantity, average_buy_price, current_price, cost_basis array. Si existe calcular precio promedio ponderado: (old_quantity * old_avg_price + new_quantity * new_price) / total_quantity, agregar entry a cost_basis, incrementar transactions_count.

SELL: buscar holding, aplicar FIFO en cost_basis (First In First Out), iterar cost_basis eliminando o reduciendo entries hasta cubrir quantity vendida, actualizar quantity total, recalcular average_buy_price, si quantity=0 eliminar holding.

Calcular valores: current_value = quantity * current_price, profit_loss = current_value - invested_amount, profit_loss_percentage = (P&L / invested) * 100, percentage_of_portfolio = (current_value / total_value) * 100.

METRICAS (30+):
Performance: daily/weekly/monthly/yearly change (absoluto + %), all_time_high/low, ROI, annualized_return, time_weighted_return, money_weighted_return, best/worst_performing_asset.

Risk: volatility (24h/7d/30d) = desviacion estandar de returns, sharpe_ratio = (return - risk_free_rate) / volatility, sortino_ratio = (return - risk_free_rate) / downside_deviation, calmar_ratio = annualized_return / |max_drawdown|, max_drawdown = peor caida desde peak, beta (vs BTC), alpha = excess return, value_at_risk_95 (VaR), conditional_var_95 (CVaR), downside_deviation.

Diversification: herfindahl_index = sum(wi^2), concentration_index, effective_holdings = 1/HHI, largest_position_percentage, top_3_concentration, categories breakdown.

Metadata: last_calculated, last_order_processed, needs_recalculation, version.

Guardar portfolio en MongoDB con $set.

SCHEDULER CRON: cada 15 minutos ("0 */15 * * * *"), obtener todos portfolios, procesar en paralelo con goroutines (max 10 concurrentes con semaphore), GET prices actuales, recalcular metricas, crear snapshot (user_id, total_value, total_invested, profit_loss, timestamp), UPDATE portfolio.

BALANCE REQUEST/RESPONSE: Portfolio publica BalanceRequest (correlation_id UUID, user_id, requested_by, timestamp) en exchange balance.request.exchange, routing key balance.request, con ReplyTo=balance.response.portfolio, Expiration=60000ms.

Users Worker consume de queue balance.request, busca user en MySQL, crea BalanceResponse (correlation_id MISMO, user_id, balance string, currency USD, success bool, error, timestamp), publica en balance.response.exchange con routing key del ReplyTo.

Portfolio consume de queue balance.response.portfolio (exclusive, non-durable, TTL 60s), matchea correlation_id, timeout 5 segundos, retorna balance.


RabbitMQ Arquitectura

3 EXCHANGES:
1. orders.events (topic, durable): routing keys orders.created/executed/cancelled/failed, publisher Orders API, consumers Portfolio API (solo executed), Search API (todos), Audit API (todos).

2. balance.request.exchange (topic, durable): routing key balance.request, publisher Portfolio API, consumer Users Worker.

3. balance.response.exchange (topic, durable): routing keys balance.response.portfolio/orders, publisher Users Worker, consumers Portfolio API, Orders API.

4 QUEUES:
1. portfolio.updates (durable): binding orders.executed, consumer Portfolio API, TTL 1h, DLX dlx, procesamiento: actualizar holdings + calcular metricas + guardar MongoDB + ACK.

2. search.sync (durable): bindings orders.created/executed/cancelled/failed, consumer Search API, TTL 10min, DLX dlx, procesamiento: GET orden completa + indexar Solr + invalidar cache + ACK.

3. balance.request (durable): binding balance.request, consumer Users Worker, TTL 60s, procesamiento: buscar user MySQL + crear response + publicar + ACK.

4. balance.response.portfolio (non-durable, exclusive): binding balance.response.portfolio, consumer Portfolio Balance Client, TTL 60s, expires 2min, procesamiento: matchear correlation_id + retornar balance + ACK.

PATRONES:
Publish-Subscribe (Fan-out): orders.events → multiple queues independientes.
Request-Reply Async: Portfolio request → Users Worker reply con correlation_id matching.
Competing Consumers: multiple instances consumen de misma queue con round-robin.

CONFIGURACION:
Prefetch count = 1 (un mensaje a la vez), manual ACK (no auto-ack), persistent messages (DeliveryMode=2), durable exchanges/queues, DLQ (Dead Letter Queue) con x-dead-letter-exchange=dlx, TTL en mensajes y queues.

MANEJO ERRORES:
Error recuperable → NACK con requeue=true.
Error permanente → NACK con requeue=false (va a DLQ).
Exito → ACK.
Idempotencia con message_id tracking en BD.

CONNECTION:
Auto-reconnect con retry loop cada 5s, heartbeat 10s, close graceful en shutdown.

