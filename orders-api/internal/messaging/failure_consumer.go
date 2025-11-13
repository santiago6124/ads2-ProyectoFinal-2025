package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
	"orders-api/internal/models"
	"orders-api/internal/repositories"
)

// OrderFailureEvent evento cuando una orden falla por problemas externos
type OrderFailureEvent struct {
	OrderID      string    `json:"order_id"`
	UserID       int       `json:"user_id"`
	FailureType  string    `json:"failure_type"`  // "insufficient_balance"
	ErrorMessage string    `json:"error_message"`
	Timestamp    time.Time `json:"timestamp"`
}

// FailureConsumer consumer para procesar fallos de órdenes
type FailureConsumer struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queueName  string
	orderRepo  repositories.OrderRepository
}

// connectWithRetryFailure intenta conectarse a RabbitMQ con reintentos
func connectWithRetryFailure(url string, maxRetries int) (*amqp.Connection, error) {
	for i := 0; i < maxRetries; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			log.Printf("✅ Failure Consumer successfully connected to RabbitMQ")
			return conn, nil
		}

		if i < maxRetries-1 {
			wait := time.Duration(1<<uint(i)) * time.Second
			log.Printf("⚠️ Failure Consumer failed to connect to RabbitMQ (attempt %d/%d), retrying in %v...", i+1, maxRetries, wait)
			time.Sleep(wait)
		}
	}
	return nil, fmt.Errorf("failed to connect to RabbitMQ after %d retries", maxRetries)
}

// NewFailureConsumer crea un nuevo consumer de fallos
func NewFailureConsumer(
	rabbitmqURL string,
	orderRepo repositories.OrderRepository,
) (*FailureConsumer, error) {
	// Conectar con retry
	conn, err := connectWithRetryFailure(rabbitmqURL, 7) // 7 intentos: ~127 segundos
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
	queueName := "orders.failure_handler"
	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = ch.QueueBind(
		q.Name,
		"orders.failed",
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	log.Printf("✅ Failure consumer initialized, listening on queue: %s", queueName)

	return &FailureConsumer{
		connection: conn,
		channel:    ch,
		queueName:  queueName,
		orderRepo:  orderRepo,
	}, nil
}

// Start inicia el consumo de mensajes
func (c *FailureConsumer) Start(ctx context.Context) error {
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

	log.Printf("🔄 Failure handler worker started, waiting for order failure messages...")

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 Failure handler worker shutting down...")
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("message channel closed")
			}

			// Procesar mensaje
			if err := c.processMessage(ctx, msg); err != nil {
				log.Printf("❌ Error processing order failure: %v", err)
				// Nack con requeue - puede ser error temporal de BD
				msg.Nack(false, true)
			} else {
				// Ack si todo salió bien
				msg.Ack(false)
			}
		}
	}
}

// processMessage procesa un mensaje de fallo de orden
func (c *FailureConsumer) processMessage(ctx context.Context, msg amqp.Delivery) error {
	start := time.Now()

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("🚨 ORDER FAILURE HANDLER - Starting processing")

	var event OrderFailureEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("❌ Failed to unmarshal event: %v", err)
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("📋 [Order ID: %s] User: %d", event.OrderID, event.UserID)
	log.Printf("⚠️  Failure Type: %s", event.FailureType)
	log.Printf("💬 Error: %s", event.ErrorMessage)

	// 1. Obtener orden de la BD
	log.Printf("1️⃣ [Order %s] Fetching order from database...", event.OrderID)
	order, err := c.orderRepo.GetByID(ctx, event.OrderID)
	if err != nil {
		log.Printf("❌ [Order %s] Failed to get order: %v", event.OrderID, err)
		return fmt.Errorf("failed to get order: %w", err)
	}
	log.Printf("✓ [Order %s] Found order: %s, current status: %s", event.OrderID, order.OrderNumber, order.Status)

	// 2. Verificar si ya está en estado final
	if order.Status == models.OrderStatusFailed {
		log.Printf("⚠️ [Order %s] Already marked as failed, skipping", event.OrderID)
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		return nil
	}

	// 3. Actualizar orden a failed
	log.Printf("2️⃣ [Order %s] Updating order to failed status...", event.OrderID)
	order.Status = models.OrderStatusFailed
	order.ErrorMessage = event.ErrorMessage
	order.UpdatedAt = time.Now()

	if err := c.orderRepo.Update(ctx, order); err != nil {
		log.Printf("❌ [Order %s] Failed to update order: %v", event.OrderID, err)
		return fmt.Errorf("failed to update order: %w", err)
	}
	log.Printf("✓ [Order %s] Order marked as FAILED", event.OrderID)
	log.Printf("   Reason: %s", event.ErrorMessage)

	log.Printf("✅ ORDER FAILURE HANDLER - Completed in %v", time.Since(start))
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return nil
}

// Close cierra la conexión
func (c *FailureConsumer) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.connection != nil {
		return c.connection.Close()
	}
	return nil
}
