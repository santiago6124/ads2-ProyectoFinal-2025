# RabbitMQ Balance Request-Response Design

## 📋 Overview

Implement bidirectional RabbitMQ messaging pattern where Portfolio API requests user balance from Users API asynchronously.

**Current State:**
- ✅ Portfolio API: Consumes order events, uses HTTP to fetch user balance
- ⚠️ Users API: Has RabbitMQ consumer code but NOT integrated/running

**Target State:**
- ✅ Portfolio API: Publishes balance requests + consumes balance responses
- ✅ Users API: Worker consumes balance requests + publishes balance responses

---

## 🏗️ Architecture Design

### Message Flow

```
┌─────────────────┐                           ┌──────────────────┐
│  Portfolio API  │                           │   Users API      │
│                 │                           │   (Worker)       │
└────────┬────────┘                           └────────┬─────────┘
         │                                              │
         │ 1. Publish BalanceRequest                   │
         │    correlation_id: "uuid-123"               │
         │    user_id: 42                              │
         ├──────────────────────────────────────────>  │
         │         (balance.request queue)             │
         │                                              │
         │                                    2. Consume request
         │                                    3. Query DB for balance
         │                                              │
         │ 4. Consume BalanceResponse                  │
         │    correlation_id: "uuid-123"               │
         │    balance: 75000.50                        │
         │  <──────────────────────────────────────────┤
         │       (balance.response.portfolio queue)    │
         │                                              │
    5. Match correlation_id                            │
    6. Update portfolio.TotalCash                      │
```

### RabbitMQ Topology

**Exchanges:**
```yaml
balance.request.exchange:
  type: direct
  durable: true
  auto_delete: false

balance.response.exchange:
  type: direct
  durable: true
  auto_delete: false
```

**Queues:**
```yaml
balance.request:
  durable: true
  routing_key: balance.request
  binds_to: balance.request.exchange
  consumers: users-worker

balance.response.portfolio:
  durable: true
  routing_key: balance.response.portfolio
  binds_to: balance.response.exchange
  consumers: portfolio-api
  ttl: 60000  # 60 seconds message expiry
```

**Dead Letter Queues:**
```yaml
balance.request.dlq:
  durable: true
  purpose: Failed balance requests

balance.response.dlq:
  durable: true
  purpose: Unmatched or failed responses
```

---

## 📦 Message Structures

### Balance Request Message

**Publisher:** Portfolio API
**Queue:** `balance.request`
**Consumer:** Users API Worker

```go
type BalanceRequestMessage struct {
    CorrelationID string    `json:"correlation_id"` // UUID for matching response
    UserID        int64     `json:"user_id"`        // User to query
    RequestedBy   string    `json:"requested_by"`   // "portfolio-api"
    Timestamp     time.Time `json:"timestamp"`
}
```

**Example:**
```json
{
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 42,
  "requested_by": "portfolio-api",
  "timestamp": "2025-11-13T10:30:00Z"
}
```

### Balance Response Message

**Publisher:** Users API Worker
**Queue:** `balance.response.portfolio`
**Consumer:** Portfolio API

```go
type BalanceResponseMessage struct {
    CorrelationID string    `json:"correlation_id"` // Matches request
    UserID        int64     `json:"user_id"`
    Balance       string    `json:"balance"`        // Decimal as string: "75000.50"
    Currency      string    `json:"currency"`       // "USD"
    Success       bool      `json:"success"`
    Error         string    `json:"error,omitempty"` // If success=false
    Timestamp     time.Time `json:"timestamp"`
}
```

**Success Example:**
```json
{
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 42,
  "balance": "75000.50",
  "currency": "USD",
  "success": true,
  "timestamp": "2025-11-13T10:30:01Z"
}
```

**Error Example:**
```json
{
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 999,
  "success": false,
  "error": "user not found",
  "timestamp": "2025-11-13T10:30:01Z"
}
```

---

## 🔧 Implementation Components

### Portfolio API Changes

#### 1. Balance Request Publisher

**File:** `portfolio-api/internal/messaging/balance_publisher.go` (NEW)

**Responsibilities:**
- Publish balance request messages
- Generate correlation IDs
- Configure retry logic

**Key Methods:**
```go
type BalancePublisher struct {
    channel  *amqp.Channel
    exchange string
}

func (p *BalancePublisher) RequestBalance(ctx context.Context, userID int64) (string, error) {
    correlationID := uuid.New().String()

    msg := BalanceRequestMessage{
        CorrelationID: correlationID,
        UserID:        userID,
        RequestedBy:   "portfolio-api",
        Timestamp:     time.Now(),
    }

    // Publish with correlation_id in AMQP properties
    err := p.channel.Publish(
        p.exchange,           // exchange
        "balance.request",    // routing key
        false,                // mandatory
        false,                // immediate
        amqp.Publishing{
            CorrelationId: correlationID,
            ContentType:   "application/json",
            Body:          json.Marshal(msg),
            Timestamp:     time.Now(),
        },
    )

    return correlationID, err
}
```

#### 2. Balance Response Consumer

**File:** `portfolio-api/internal/messaging/balance_consumer.go` (NEW)

**Responsibilities:**
- Consume balance response messages
- Match correlation IDs to pending requests
- Update portfolio with balance data
- Handle timeouts and errors

**Key Methods:**
```go
type BalanceResponseConsumer struct {
    channel        *amqp.Channel
    queueName      string
    pendingRequests map[string]chan BalanceResponseMessage // correlation_id -> response channel
    mu             sync.RWMutex
}

func (c *BalanceResponseConsumer) Start(ctx context.Context) error {
    msgs, err := c.channel.Consume(c.queueName, "", false, false, false, false, nil)

    for {
        select {
        case msg := <-msgs:
            var response BalanceResponseMessage
            json.Unmarshal(msg.Body, &response)

            c.mu.RLock()
            responseChan, exists := c.pendingRequests[response.CorrelationID]
            c.mu.RUnlock()

            if exists {
                responseChan <- response  // Send to waiting goroutine
                msg.Ack(false)
            } else {
                // Orphaned response - send to DLQ
                msg.Nack(false, false)
            }
        case <-ctx.Done():
            return nil
        }
    }
}

func (c *BalanceResponseConsumer) WaitForResponse(correlationID string, timeout time.Duration) (*BalanceResponseMessage, error) {
    responseChan := make(chan BalanceResponseMessage, 1)

    c.mu.Lock()
    c.pendingRequests[correlationID] = responseChan
    c.mu.Unlock()

    defer func() {
        c.mu.Lock()
        delete(c.pendingRequests, correlationID)
        c.mu.Unlock()
    }()

    select {
    case response := <-responseChan:
        return &response, nil
    case <-time.After(timeout):
        return nil, fmt.Errorf("timeout waiting for balance response")
    }
}
```

#### 3. Service Layer Integration

**File:** `portfolio-api/internal/services/portfolio_service.go` (MODIFY)

**Change:**
```go
// OLD: HTTP call
func (s *portfolioService) GetPortfolio(ctx context.Context, userID int64) (*models.Portfolio, error) {
    balance, err := s.userClient.GetUserBalance(ctx, userID)  // HTTP call
    // ...
}

// NEW: RabbitMQ request-response
func (s *portfolioService) GetPortfolio(ctx context.Context, userID int64) (*models.Portfolio, error) {
    // Publish balance request
    correlationID, err := s.balancePublisher.RequestBalance(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to request balance: %w", err)
    }

    // Wait for response (with 5-second timeout)
    response, err := s.balanceConsumer.WaitForResponse(correlationID, 5*time.Second)
    if err != nil {
        // Fallback to HTTP if messaging fails
        log.Warnf("Balance request timeout, falling back to HTTP: %v", err)
        balance, err := s.userClient.GetUserBalance(ctx, userID)
        if err != nil {
            return nil, err
        }
        portfolio.TotalCash = balance
    } else {
        if !response.Success {
            return nil, fmt.Errorf("balance request failed: %s", response.Error)
        }
        balance, _ := decimal.NewFromString(response.Balance)
        portfolio.TotalCash = balance
    }

    // ... rest of portfolio logic
}
```

---

### Users API Changes

#### 1. Balance Request Consumer (Worker)

**File:** `users-api/internal/messaging/balance_request_consumer.go` (NEW)

**Responsibilities:**
- Consume balance request messages
- Query user balance from database
- Publish balance response messages

**Implementation:**
```go
type BalanceRequestConsumer struct {
    channel         *amqp.Channel
    queueName       string
    userService     services.UserService
    responsePublisher *BalanceResponsePublisher
    logger          *logrus.Logger
}

func (c *BalanceRequestConsumer) Start(ctx context.Context) error {
    // Set QoS
    c.channel.Qos(10, 0, false)  // Process up to 10 concurrent requests

    msgs, err := c.channel.Consume(c.queueName, "", false, false, false, false, nil)
    if err != nil {
        return fmt.Errorf("failed to start consuming: %w", err)
    }

    c.logger.Info("🔄 Balance request worker started")

    for {
        select {
        case <-ctx.Done():
            c.logger.Info("🛑 Balance request worker shutting down")
            return ctx.Err()
        case msg, ok := <-msgs:
            if !ok {
                return fmt.Errorf("message channel closed")
            }

            if err := c.processRequest(ctx, msg); err != nil {
                c.logger.Errorf("Failed to process balance request: %v", err)
                msg.Nack(false, true)  // Requeue on error
            } else {
                msg.Ack(false)
            }
        }
    }
}

func (c *BalanceRequestConsumer) processRequest(ctx context.Context, msg amqp.Delivery) error {
    var request BalanceRequestMessage
    if err := json.Unmarshal(msg.Body, &request); err != nil {
        return fmt.Errorf("failed to unmarshal request: %w", err)
    }

    c.logger.Infof("📨 Received balance request for user %d (correlation: %s)",
        request.UserID, request.CorrelationID)

    // Query user balance
    user, err := c.userService.GetUserByID(int32(request.UserID))

    var response BalanceResponseMessage
    response.CorrelationID = request.CorrelationID
    response.UserID = request.UserID
    response.Timestamp = time.Now()

    if err != nil {
        response.Success = false
        response.Error = fmt.Sprintf("user not found: %v", err)
    } else {
        response.Success = true
        response.Balance = fmt.Sprintf("%.2f", user.CurrentBalance)
        response.Currency = "USD"
    }

    // Publish response
    if err := c.responsePublisher.PublishResponse(ctx, response); err != nil {
        return fmt.Errorf("failed to publish response: %w", err)
    }

    c.logger.Infof("✅ Sent balance response for user %d: success=%v",
        request.UserID, response.Success)

    return nil
}
```

#### 2. Balance Response Publisher

**File:** `users-api/internal/messaging/balance_response_publisher.go` (NEW)

**Implementation:**
```go
type BalanceResponsePublisher struct {
    channel  *amqp.Channel
    exchange string
}

func (p *BalanceResponsePublisher) PublishResponse(ctx context.Context, response BalanceResponseMessage) error {
    body, err := json.Marshal(response)
    if err != nil {
        return fmt.Errorf("failed to marshal response: %w", err)
    }

    return p.channel.Publish(
        p.exchange,                    // exchange
        "balance.response.portfolio",  // routing key (specific to portfolio-api)
        false,                         // mandatory
        false,                         // immediate
        amqp.Publishing{
            CorrelationId: response.CorrelationID,
            ContentType:   "application/json",
            Body:          body,
            Timestamp:     time.Now(),
        },
    )
}
```

#### 3. Worker Entry Point

**File:** `users-api/cmd/worker/main.go` (NEW)

**Complete worker process:**
```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "users-api/internal/config"
    "users-api/internal/messaging"
    "users-api/internal/repositories"
    "users-api/internal/services"
    "users-api/pkg/database"

    "github.com/sirupsen/logrus"
)

func main() {
    logger := logrus.New()
    logger.SetFormatter(&logrus.JSONFormatter{})
    logger.Info("🚀 Starting Users API Balance Worker")

    // Load configuration
    cfg := config.LoadConfig()

    // Connect to database
    db, err := database.NewConnection(&database.Config{
        Host:     cfg.Database.Host,
        Port:     cfg.Database.Port,
        User:     cfg.Database.User,
        Password: cfg.Database.Password,
        DBName:   cfg.Database.DBName,
    })
    if err != nil {
        logger.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()
    logger.Info("✅ Connected to MySQL database")

    // Initialize repositories
    userRepo := repositories.NewUserRepository(db.DB)
    balanceRepo := repositories.NewBalanceTransactionRepository(db.DB)

    // Initialize services
    userService := services.NewUserServiceWithBalance(userRepo, balanceRepo)

    // Initialize RabbitMQ
    responsePublisher, err := messaging.NewBalanceResponsePublisher(cfg.RabbitMQ.URL)
    if err != nil {
        logger.Fatalf("Failed to create response publisher: %v", err)
    }
    defer responsePublisher.Close()
    logger.Info("✅ RabbitMQ response publisher initialized")

    requestConsumer, err := messaging.NewBalanceRequestConsumer(
        cfg.RabbitMQ.URL,
        userService,
        responsePublisher,
        logger,
    )
    if err != nil {
        logger.Fatalf("Failed to create request consumer: %v", err)
    }
    defer requestConsumer.Close()
    logger.Info("✅ RabbitMQ request consumer initialized")

    // Graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        sig := <-sigChan
        logger.Infof("Received signal %v, initiating shutdown", sig)
        cancel()
    }()

    // Start worker (blocking)
    logger.Info("🔄 Balance request worker is ready to process messages")
    if err := requestConsumer.Start(ctx); err != nil {
        if err != context.Canceled {
            logger.Errorf("Worker stopped with error: %v", err)
        }
    }

    logger.Info("👋 Worker shutdown complete")
}
```

---

## 🐳 Docker Integration

### users-api Dockerfile.worker

**File:** `users-api/Dockerfile.worker` (NEW)

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build worker binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags '-extldflags "-static"' \
    -o worker \
    cmd/worker/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/worker .
COPY --from=builder /app/migrations ./migrations

RUN chown -R appuser:appgroup /app

USER appuser

CMD ["./worker"]
```

### Docker Compose Update

**File:** `docker-compose.yml` (ADD SERVICE)

```yaml
services:
  # ... existing services ...

  users-worker:
    build:
      context: ./users-api
      dockerfile: Dockerfile.worker
    container_name: users-balance-worker
    environment:
      - DB_HOST=users-mysql
      - DB_PORT=3306
      - DB_USER=root
      - DB_PASSWORD=usersdbpassword
      - DB_NAME=users_db
      - RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
      - LOG_LEVEL=info
    depends_on:
      users-mysql:
        condition: service_healthy
      rabbitmq:
        condition: service_healthy
    restart: unless-stopped
    networks:
      - cryptosim-network
    healthcheck:
      test: ["CMD", "pgrep", "-f", "worker"]
      interval: 30s
      timeout: 10s
      retries: 3
```

---

## ⚡ Performance Considerations

### Timeouts & Retries

**Portfolio API (Requester):**
- **Request Timeout:** 5 seconds
- **Fallback Strategy:** HTTP call if messaging fails
- **Max Concurrent Requests:** 100 (channel buffer)

**Users API Worker (Responder):**
- **QoS Prefetch:** 10 messages
- **Processing Timeout:** 2 seconds per message
- **Retry Policy:** Exponential backoff (3 retries)

### Message TTL

**Balance Response Messages:**
- **TTL:** 60 seconds
- **Reason:** Prevents stale balance data
- **After Expiry:** Moved to DLQ

### Connection Pooling

**Portfolio API:**
- **Channels:** 2 (1 publisher, 1 consumer)
- **Connection Reuse:** Single connection for both

**Users Worker:**
- **Channels:** 2 (1 consumer, 1 publisher)
- **Concurrent Consumers:** 10 goroutines

---

## 🧪 Testing Strategy

### Unit Tests

**Portfolio API:**
```go
// Test publisher generates valid correlation IDs
func TestBalancePublisher_RequestBalance(t *testing.T)

// Test consumer matches correlation IDs correctly
func TestBalanceConsumer_WaitForResponse(t *testing.T)

// Test timeout handling
func TestBalanceConsumer_Timeout(t *testing.T)
```

**Users API:**
```go
// Test request processing with valid user
func TestBalanceRequestConsumer_ValidUser(t *testing.T)

// Test request processing with invalid user
func TestBalanceRequestConsumer_UserNotFound(t *testing.T)

// Test response publishing
func TestBalanceResponsePublisher_Publish(t *testing.T)
```

### Integration Tests

**End-to-End Flow:**
```bash
# 1. Start RabbitMQ
docker-compose up -d rabbitmq

# 2. Start users-worker
docker-compose up -d users-worker

# 3. Publish balance request from portfolio-api
curl -X POST http://localhost:8003/api/portfolios/42

# 4. Verify logs show:
#    - Portfolio: "Published balance request for user 42"
#    - Worker: "Received balance request for user 42"
#    - Worker: "Sent balance response for user 42"
#    - Portfolio: "Received balance response: 75000.50"
```

---

## 🚀 Implementation Checklist

### Phase 1: Users API (Worker Setup)
- [ ] Add `github.com/streadway/amqp` to `go.mod`
- [ ] Add RabbitMQ config to `internal/config/config.go`
- [ ] Create `internal/messaging/balance_request_consumer.go`
- [ ] Create `internal/messaging/balance_response_publisher.go`
- [ ] Create `cmd/worker/main.go`
- [ ] Create `Dockerfile.worker`
- [ ] Update `docker-compose.yml` with `users-worker`

### Phase 2: Portfolio API (Request/Response)
- [ ] Create `internal/messaging/balance_publisher.go`
- [ ] Create `internal/messaging/balance_consumer.go`
- [ ] Update `internal/services/portfolio_service.go`
- [ ] Add balance messaging initialization in `cmd/main.go`
- [ ] Update RabbitMQ config with new queues/exchanges

### Phase 3: Testing
- [ ] Unit tests for publishers and consumers
- [ ] Integration test for request-response flow
- [ ] Load test with 1000 concurrent requests
- [ ] Failure scenario testing (timeout, user not found)

### Phase 4: Deployment
- [ ] Deploy to staging environment
- [ ] Monitor RabbitMQ management UI for message flow
- [ ] Verify logs in both services
- [ ] Performance benchmarking vs HTTP approach

---

## 📊 Success Metrics

- **Latency:** < 100ms for balance request-response (vs ~50ms HTTP)
- **Throughput:** > 1000 requests/second
- **Reliability:** 99.9% successful responses
- **Fallback Success:** 100% HTTP fallback on messaging failure
- **Message Loss:** 0% (durable queues + acks)

---

## 🔗 Related Documentation

- RabbitMQ Request-Response Pattern: https://www.rabbitmq.com/tutorials/tutorial-six-go.html
- AMQP Correlation ID: https://www.rabbitmq.com/tutorials/amqp-concepts.html
- Portfolio API Architecture: `claudedocs/portfolio-api-analysis.md`
- Users API Architecture: `claudedocs/users-api-analysis.md`
