package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

// Publisher para publicar eventos desde users-api
type Publisher struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

// OrderFailureEvent evento cuando una orden falla por problemas de balance
type OrderFailureEvent struct {
	OrderID      string    `json:"order_id"`
	UserID       int       `json:"user_id"`
	FailureType  string    `json:"failure_type"` // "insufficient_balance"
	ErrorMessage string    `json:"error_message"`
	Timestamp    time.Time `json:"timestamp"`
}

// connectWithRetryPublisher intenta conectarse a RabbitMQ con reintentos
func connectWithRetryPublisher(url string, maxRetries int) (*amqp.Connection, error) {
	for i := 0; i < maxRetries; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			log.Printf("✅ Balance Publisher successfully connected to RabbitMQ")
			return conn, nil
		}

		if i < maxRetries-1 {
			wait := time.Duration(1<<uint(i)) * time.Second
			log.Printf("⚠️ Balance Publisher failed to connect to RabbitMQ (attempt %d/%d), retrying in %v...", i+1, maxRetries, wait)
			time.Sleep(wait)
		}
	}
	return nil, fmt.Errorf("failed to connect to RabbitMQ after %d retries", maxRetries)
}

// NewPublisher crea un nuevo publisher
func NewPublisher(rabbitmqURL string) (*Publisher, error) {
	// Conectar con retry
	conn, err := connectWithRetryPublisher(rabbitmqURL, 7) // 7 intentos: ~127 segundos
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	log.Printf("✅ Balance Publisher initialized")

	return &Publisher{
		connection: conn,
		channel:    ch,
	}, nil
}

// PublishOrderFailed publica un evento cuando una orden falla
func (p *Publisher) PublishOrderFailed(orderID string, userID int, errorMessage string) error {
	// Declarar exchange (idempotente)
	exchange := "orders.events"
	err := p.channel.ExchangeDeclare(
		exchange,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Crear evento
	event := &OrderFailureEvent{
		OrderID:      orderID,
		UserID:       userID,
		FailureType:  "insufficient_balance",
		ErrorMessage: errorMessage,
		Timestamp:    time.Now(),
	}

	// Serializar
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publicar
	err = p.channel.Publish(
		exchange,        // exchange
		"orders.failed", // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish order failed event: %w", err)
	}

	log.Printf("✓ Published order.failed event for order %s", orderID)
	return nil
}

// Close cierra la conexión
func (p *Publisher) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.connection != nil {
		return p.connection.Close()
	}
	return nil
}
