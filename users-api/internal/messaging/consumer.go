package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/streadway/amqp"
	"users-api/internal/models"
	"users-api/internal/repositories"
	"users-api/internal/services"
)

// InsufficientBalanceError error cuando el usuario no tiene fondos suficientes
type InsufficientBalanceError struct {
	UserID      int
	Required    float64
	Available   float64
	ResultingIn float64
}

func (e *InsufficientBalanceError) Error() string {
	return fmt.Sprintf("insufficient balance for user %d: required %.2f, available %.2f, would result in %.2f",
		e.UserID, e.Required, e.Available, e.ResultingIn)
}

// BalanceUpdateEvent evento para actualizar saldo de usuario
type BalanceUpdateEvent struct {
	OrderID         string    `json:"order_id"`
	UserID          int       `json:"user_id"`
	Amount          string    `json:"amount"` // decimal como string
	TransactionType string    `json:"transaction_type"` // "buy" o "sell"
	CryptoSymbol    string    `json:"crypto_symbol"`
	Quantity        string    `json:"quantity"`
	Price           string    `json:"price"`
	Description     string    `json:"description"`
	Timestamp       time.Time `json:"timestamp"`
}

// connectWithRetry intenta conectarse a RabbitMQ con reintentos y backoff exponencial
func connectWithRetry(url string, maxRetries int) (*amqp.Connection, error) {
	for i := 0; i < maxRetries; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			log.Printf("✅ Balance Consumer successfully connected to RabbitMQ")
			return conn, nil
		}

		if i < maxRetries-1 {
			wait := time.Duration(1<<uint(i)) * time.Second // Backoff: 1s, 2s, 4s, 8s, 16s
			log.Printf("⚠️ Balance Consumer failed to connect to RabbitMQ (attempt %d/%d), retrying in %v...", i+1, maxRetries, wait)
			time.Sleep(wait)
		}
	}
	return nil, fmt.Errorf("failed to connect to RabbitMQ after %d retries", maxRetries)
}

// BalanceConsumer consumer para procesar actualizaciones de saldo
type BalanceConsumer struct {
	connection  *amqp.Connection
	channel     *amqp.Channel
	queueName   string
	userService services.UserService
	txRepo      repositories.BalanceTransactionRepository
	publisher   *Publisher
}

// NewBalanceConsumer crea un nuevo consumer de balance
func NewBalanceConsumer(
	rabbitmqURL string,
	userService services.UserService,
	txRepo repositories.BalanceTransactionRepository,
	publisher *Publisher,
) (*BalanceConsumer, error) {
	// Usar connectWithRetry para conexión robusta
	conn, err := connectWithRetry(rabbitmqURL, 7) // 7 intentos: ~127 segundos total
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declarar exchange
	exchangeName := "balance.events"
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
	queueName := "balance.updates"
	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange": "balance.dlx",
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
		"balance.update",
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	// Declarar Dead Letter Exchange (DLX)
	dlxName := "balance.dlx"
	err = ch.ExchangeDeclare(
		dlxName,
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
		return nil, fmt.Errorf("failed to declare DLX: %w", err)
	}

	// Declarar Dead Letter Queue
	_, err = ch.QueueDeclare(
		"balance.failed",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Bind DLQ to DLX
	err = ch.QueueBind(
		"balance.failed",
		"#",
		dlxName,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind DLQ: %w", err)
	}

	log.Printf("✅ Balance consumer initialized, listening on queue: %s", queueName)

	return &BalanceConsumer{
		connection:  conn,
		channel:     ch,
		queueName:   queueName,
		userService: userService,
		txRepo:      txRepo,
		publisher:   publisher,
	}, nil
}

// Start inicia el consumo de mensajes
func (c *BalanceConsumer) Start(ctx context.Context) error {
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

	log.Printf("🔄 Balance worker started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 Balance worker shutting down...")
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("message channel closed")
			}

			// Procesar mensaje
			if err := c.processMessage(ctx, msg); err != nil {
				log.Printf("❌ Error processing balance update: %v", err)

				// Verificar si es un error no recuperable (balance insuficiente)
				var insufficientBalanceErr *InsufficientBalanceError
				if errors.As(err, &insufficientBalanceErr) {
					// Error NO recuperable: usuario sin fondos suficientes
					log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
					log.Printf("🚫 NON-RECOVERABLE ERROR: Insufficient balance")
					log.Printf("   User %d: Has %.2f, Needs %.2f, Would result in %.2f",
						insufficientBalanceErr.UserID,
						insufficientBalanceErr.Available,
						insufficientBalanceErr.Required,
						insufficientBalanceErr.ResultingIn)

					// Extraer OrderID del mensaje
					var event BalanceUpdateEvent
					if unmarshalErr := json.Unmarshal(msg.Body, &event); unmarshalErr == nil {
						// Publicar evento de fallo a orders-api
						log.Printf("   📤 Publishing order failure event to orders-api...")
						if c.publisher != nil {
							pubErr := c.publisher.PublishOrderFailed(
								event.OrderID,
								event.UserID,
								fmt.Sprintf("Insufficient balance: user has %.2f but needs %.2f",
									insufficientBalanceErr.Available,
									insufficientBalanceErr.Required),
							)
							if pubErr != nil {
								log.Printf("   ❌ Failed to publish order failure event: %v", pubErr)
							} else {
								log.Printf("   ✓ Order failure event published successfully")
							}
						} else {
							log.Printf("   ⚠️ Publisher not available, order will NOT be marked as failed")
						}
					}

					log.Printf("   ⚠️ Message DISCARDED (will NOT retry)")
					log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
					msg.Ack(false)
				} else {
					// Error potencialmente recuperable (BD, timeout, etc.)
					// Hacer Nack con requeue para reintentar
					log.Printf("⚠️ Recoverable error, requeuing message for retry")
					msg.Nack(false, true)
				}
			} else {
				// Ack si todo salió bien
				msg.Ack(false)
			}
		}
	}
}

// processMessage procesa un mensaje de actualización de saldo
func (c *BalanceConsumer) processMessage(ctx context.Context, msg amqp.Delivery) error {
	start := time.Now()

	var event BalanceUpdateEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("💰 BALANCE WORKER - Starting processing")
	log.Printf("📦 Order ID: %s", event.OrderID)
	log.Printf("👤 User: %d | Amount: %s | Type: %s", event.UserID, event.Amount, event.TransactionType)
	log.Printf("🪙  Symbol: %s | Qty: %s @ %s", event.CryptoSymbol, event.Quantity, event.Price)

	// PASO 1: Verificar idempotencia ANTES de procesar
	log.Printf("🔍 [Order %s] Checking if already processed...", event.OrderID)
	existing, err := c.txRepo.FindByOrderID(event.OrderID)
	if err != nil {
		return fmt.Errorf("failed to check transaction: %w", err)
	}
	if existing != nil {
		log.Printf("✓ [Order %s] Already processed at %v, skipping (idempotent)",
			event.OrderID, existing.ProcessedAt)
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		return nil
	}
	log.Printf("✓ [Order %s] Not processed yet, continuing...", event.OrderID)

	// Parsear el monto
	log.Printf("1️⃣ [Order %s] Parsing amount and getting user...", event.OrderID)
	amount, err := strconv.ParseFloat(event.Amount, 64)
	if err != nil {
		return fmt.Errorf("invalid amount format: %w", err)
	}

	// Obtener el saldo actual del usuario
	user, err := c.userService.GetUserByID(int32(event.UserID))
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	log.Printf("✓ [Order %s] Current balance: %.2f", event.OrderID, user.InitialBalance)

	// Calcular nuevo saldo según el tipo de transacción
	log.Printf("2️⃣ [Order %s] Calculating new balance...", event.OrderID)
	var newBalance float64
	if event.TransactionType == "buy" {
		newBalance = user.InitialBalance - amount
		log.Printf("   Type: BUY - Deducting %.2f", amount)
	} else if event.TransactionType == "sell" {
		newBalance = user.InitialBalance + amount
		log.Printf("   Type: SELL - Adding %.2f", amount)
	} else {
		return fmt.Errorf("invalid transaction type: %s", event.TransactionType)
	}
	log.Printf("   Previous: %.2f → New: %.2f (Δ %.2f)",
		user.InitialBalance, newBalance, newBalance-user.InitialBalance)

	// Validar que el nuevo saldo no sea negativo
	if newBalance < 0 {
		log.Printf("❌ [Order %s] INSUFFICIENT BALANCE - User %d has %.2f but needs %.2f",
			event.OrderID, event.UserID, user.InitialBalance, amount)
		return &InsufficientBalanceError{
			UserID:      event.UserID,
			Required:    amount,
			Available:   user.InitialBalance,
			ResultingIn: newBalance,
		}
	}

	// Actualizar saldo en la base de datos
	log.Printf("3️⃣ [Order %s] Updating balance in database...", event.OrderID)
	if err := c.userService.UpdateBalance(int32(event.UserID), newBalance); err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}
	log.Printf("✓ [Order %s] Balance updated successfully", event.OrderID)

	// PASO 7: Guardar transacción para idempotencia
	log.Printf("4️⃣ [Order %s] Saving transaction for idempotency...", event.OrderID)
	tx := &models.BalanceTransaction{
		OrderID:         event.OrderID,
		UserID:          int32(event.UserID),
		Amount:          amount,
		TransactionType: event.TransactionType,
		CryptoSymbol:    event.CryptoSymbol,
		PreviousBalance: user.InitialBalance,
		NewBalance:      newBalance,
	}
	if err := c.txRepo.Create(tx); err != nil {
		log.Printf("⚠️ [Order %s] Failed to save transaction record (non-critical): %v", event.OrderID, err)
	} else {
		log.Printf("✓ [Order %s] Transaction saved successfully", event.OrderID)
	}

	elapsed := time.Since(start)
	log.Printf("✅ BALANCE WORKER - Completed in %v", elapsed)
	log.Printf("   Order: %s | User: %d | Balance: %.2f → %.2f",
		event.OrderID, event.UserID, user.InitialBalance, newBalance)
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return nil
}

// Close cierra la conexión
func (c *BalanceConsumer) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.connection != nil {
		return c.connection.Close()
	}
	return nil
}
