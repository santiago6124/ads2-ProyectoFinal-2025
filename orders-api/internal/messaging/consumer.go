package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"
	"github.com/streadway/amqp"

	"orders-api/internal/models"
	"orders-api/internal/repositories"
)

// OrderConsumer consumer para procesar órdenes creadas
type OrderConsumer struct {
	connection    *amqp.Connection
	channel       *amqp.Channel
	queueName     string
	orderRepo     repositories.OrderRepository
	publisher     *Publisher
	userClient    UserClient
	marketClient  MarketClient
}

// UserClient interface para validar usuarios
type UserClient interface {
	VerifyUser(ctx context.Context, userID int) (*models.ValidationResult, error)
}

// MarketClient interface para obtener precios
type MarketClient interface {
	GetCurrentPrice(ctx context.Context, symbol string) (*models.PriceResult, error)
}

// connectWithRetryConsumer intenta conectarse a RabbitMQ con reintentos y backoff exponencial
func connectWithRetryConsumer(url string, maxRetries int) (*amqp.Connection, error) {
	for i := 0; i < maxRetries; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			log.Printf("✅ Order Consumer successfully connected to RabbitMQ")
			return conn, nil
		}

		if i < maxRetries-1 {
			wait := time.Duration(1<<uint(i)) * time.Second // Backoff: 1s, 2s, 4s, 8s, 16s
			log.Printf("⚠️ Order Consumer failed to connect to RabbitMQ (attempt %d/%d), retrying in %v...", i+1, maxRetries, wait)
			time.Sleep(wait)
		}
	}
	return nil, fmt.Errorf("failed to connect to RabbitMQ after %d retries", maxRetries)
}

// NewOrderConsumer crea un nuevo consumer de órdenes
func NewOrderConsumer(
	rabbitmqURL string,
	orderRepo repositories.OrderRepository,
	publisher *Publisher,
	userClient UserClient,
	marketClient MarketClient,
) (*OrderConsumer, error) {
	// Usar connectWithRetry para conexión robusta
	conn, err := connectWithRetryConsumer(rabbitmqURL, 7) // 7 intentos: ~127 segundos total
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declarar exchange
	exchangeName := "orders.events"
	err = ch.ExchangeDeclare(
		exchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declarar queue
	queueName := "orders.pending"
	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange": "orders.dlx",
		},
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = ch.QueueBind(
		q.Name,
		"orders.created",
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	log.Printf("✅ Order consumer initialized, listening on queue: %s", queueName)

	return &OrderConsumer{
		connection:   conn,
		channel:      ch,
		queueName:    queueName,
		orderRepo:    orderRepo,
		publisher:    publisher,
		userClient:   userClient,
		marketClient: marketClient,
	}, nil
}

// Start inicia el consumo de mensajes
func (c *OrderConsumer) Start(ctx context.Context) error {
	// Set QoS - procesar un mensaje a la vez
	err := c.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := c.channel.Consume(
		c.queueName,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("🔄 Order worker started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 Order worker shutting down...")
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("message channel closed")
			}

			// Procesar mensaje
			if err := c.processMessage(ctx, msg); err != nil {
				log.Printf("❌ Error processing message: %v", err)
				// Nack con requeue si es un error recuperable
				msg.Nack(false, true)
			} else {
				// Ack si todo salió bien
				msg.Ack(false)
			}
		}
	}
}

// processMessage procesa un mensaje de orden creada
func (c *OrderConsumer) processMessage(ctx context.Context, msg amqp.Delivery) error {
	start := time.Now()

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📦 ORDER WORKER - Starting processing")

	var event OrderEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("❌ Failed to unmarshal event: %v", err)
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("📋 [Order ID: %s]", event.OrderID)
	log.Printf("👤 [User ID: %d] Type: %s, Symbol: %s", event.UserID, event.Type, event.CryptoSymbol)

	// Obtener orden de la BD
	log.Printf("🔍 [Order %s] Fetching from database...", event.OrderID)
	order, err := c.orderRepo.GetByID(ctx, event.OrderID)
	if err != nil {
		log.Printf("❌ [Order %s] Failed to get order: %v", event.OrderID, err)
		return fmt.Errorf("failed to get order: %w", err)
	}
	log.Printf("✓ [Order %s] Found in database, status: %s", event.OrderID, order.Status)

	// Verificar que la orden está en estado pending
	if order.Status != models.OrderStatusPending {
		log.Printf("⚠️ [Order %s] Not pending (status: %s), skipping", event.OrderID, order.Status)
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		return nil // No es error, solo que ya fue procesada
	}

	// Procesar orden
	if err := c.executeOrder(ctx, order); err != nil {
		log.Printf("❌ ORDER WORKER - Failed in %v", time.Since(start))
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		return c.handleOrderFailure(ctx, order, err)
	}

	log.Printf("✅ ORDER WORKER - Completed in %v", time.Since(start))
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return nil
}

// executeOrder ejecuta la orden y publica eventos
func (c *OrderConsumer) executeOrder(ctx context.Context, order *models.Order) error {
	orderID := order.ID.Hex()
	log.Printf("⚙️ [Order %s] Starting execution", orderID)
	log.Printf("📊 [Order %s] User: %d, Symbol: %s, Quantity: %s",
		orderID, order.UserID, order.CryptoSymbol, order.Quantity.String())

	// 1. Usar precio de la orden (ya validado y obtenido desde el frontend)
	log.Printf("1️⃣ [Order %s] Using price from order: %s for %s",
		orderID, order.Price.String(), order.CryptoSymbol)
	executedPrice := order.Price

	// 2. Calcular monto total y comisión
	totalAmount := order.Quantity.Mul(executedPrice)
	fee := totalAmount.Mul(decimal.NewFromFloat(0.001)) // 0.1%
	minFee := decimal.NewFromFloat(0.01)
	if fee.LessThan(minFee) {
		fee = minFee
	}
	log.Printf("✓ [Order %s] Calculated: Total=%s, Fee=%s", orderID, totalAmount.String(), fee.String())

	// 3. Actualizar orden a ejecutada
	log.Printf("2️⃣ [Order %s] Updating order status to executed...", orderID)
	order.Status = models.OrderStatusExecuted
	order.Price = executedPrice
	order.TotalAmount = totalAmount
	order.Fee = fee
	now := time.Now()
	order.ExecutedAt = &now
	order.UpdatedAt = now

	if err := c.orderRepo.Update(ctx, order); err != nil {
		log.Printf("❌ [Order %s] Failed to update order: %v", orderID, err)
		return fmt.Errorf("failed to update order: %w", err)
	}
	log.Printf("✓ [Order %s] Order updated to executed status", orderID)

	// 4. Publicar evento de orden ejecutada
	log.Printf("3️⃣ [Order %s] Publishing order executed event...", orderID)
	if err := c.publisher.PublishOrderExecuted(ctx, order); err != nil {
		log.Printf("⚠️ [Order %s] Failed to publish order executed event: %v", orderID, err)
	} else {
		log.Printf("✓ [Order %s] Order executed event published", orderID)
	}

	// 5. Publicar evento de actualización de balance
	log.Printf("4️⃣ [Order %s] Publishing balance update event...", orderID)
	if err := c.publisher.PublishBalanceUpdate(ctx, order); err != nil {
		log.Printf("⚠️ [Order %s] Failed to publish balance update event: %v", orderID, err)
	} else {
		log.Printf("✓ [Order %s] Balance update event published to balance.events", orderID)
	}

	// 6. Publicar evento de actualización de portfolio
	log.Printf("5️⃣ [Order %s] Publishing portfolio update event...", orderID)
	if err := c.publisher.PublishPortfolioUpdate(ctx, order); err != nil {
		log.Printf("⚠️ [Order %s] Failed to publish portfolio update event: %v", orderID, err)
	} else {
		log.Printf("✓ [Order %s] Portfolio update event published to portfolio.events", orderID)
	}

	log.Printf("✅ [Order %s] Order executed successfully (Price: %s, Total: %s, Fee: %s)",
		orderID, executedPrice.String(), totalAmount.String(), fee.String())
	return nil
}

// handleOrderFailure maneja el fallo de una orden
func (c *OrderConsumer) handleOrderFailure(ctx context.Context, order *models.Order, err error) error {
	orderID := order.ID.Hex()
	log.Printf("❌ Order %s failed: %v", orderID, err)

	// Actualizar orden a fallida
	order.Status = models.OrderStatusFailed
	order.ErrorMessage = err.Error()
	order.UpdatedAt = time.Now()

	if updateErr := c.orderRepo.Update(ctx, order); updateErr != nil {
		log.Printf("⚠️ Failed to update order status: %v", updateErr)
	}

	// Publicar evento de orden fallida
	if pubErr := c.publisher.PublishOrderFailed(ctx, order, err.Error()); pubErr != nil {
		log.Printf("⚠️ Failed to publish order failed event: %v", pubErr)
	}

	return err
}

// Close cierra la conexión
func (c *OrderConsumer) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.connection != nil {
		return c.connection.Close()
	}
	return nil
}
