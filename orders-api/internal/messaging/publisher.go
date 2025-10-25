package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
	"orders-api/internal/models"
)

// Publisher simplificado para eventos de órdenes
type Publisher struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	exchange   string
}

// OrderEvent evento simplificado de orden
type OrderEvent struct {
	EventType     string    `json:"event_type"` // created, executed, cancelled, failed
	OrderID       string    `json:"order_id"`
	OrderNumber   string    `json:"order_number"`
	UserID        int       `json:"user_id"`
	Type          string    `json:"type"`   // buy, sell
	Status        string    `json:"status"` // pending, executed, cancelled, failed
	CryptoSymbol  string    `json:"crypto_symbol"`
	Quantity      string    `json:"quantity"`
	Price         string    `json:"price"`
	TotalAmount   string    `json:"total_amount"`
	Fee           string    `json:"fee"`
	Timestamp     time.Time `json:"timestamp"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// NewPublisher crea un nuevo publisher simplificado
func NewPublisher(rabbitmqURL string) (*Publisher, error) {
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	exchangeName := "orders.events"

	// Declarar un solo exchange de tipo topic
	err = ch.ExchangeDeclare(
		exchangeName,
		"topic", // tipo topic para routing flexible
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	log.Printf("RabbitMQ publisher initialized with exchange: %s", exchangeName)

	return &Publisher{
		connection: conn,
		channel:    ch,
		exchange:   exchangeName,
	}, nil
}

// PublishOrderCreated publica evento de orden creada
func (p *Publisher) PublishOrderCreated(ctx context.Context, order *models.Order) error {
	event := &OrderEvent{
		EventType:    "created",
		OrderID:      order.ID.Hex(),
		OrderNumber:  order.OrderNumber,
		UserID:       order.UserID,
		Type:         string(order.Type),
		Status:       string(order.Status),
		CryptoSymbol: order.CryptoSymbol,
		Quantity:     order.Quantity.String(),
		Price:        order.Price.String(),
		TotalAmount:  order.TotalAmount.String(),
		Fee:          order.Fee.String(),
		Timestamp:    time.Now(),
	}

	return p.publish("orders.created", event)
}

// PublishOrderExecuted publica evento de orden ejecutada
func (p *Publisher) PublishOrderExecuted(ctx context.Context, order *models.Order) error {
	event := &OrderEvent{
		EventType:    "executed",
		OrderID:      order.ID.Hex(),
		OrderNumber:  order.OrderNumber,
		UserID:       order.UserID,
		Type:         string(order.Type),
		Status:       string(order.Status),
		CryptoSymbol: order.CryptoSymbol,
		Quantity:     order.Quantity.String(),
		Price:        order.Price.String(),
		TotalAmount:  order.TotalAmount.String(),
		Fee:          order.Fee.String(),
		Timestamp:    time.Now(),
	}

	return p.publish("orders.executed", event)
}

// PublishOrderCancelled publica evento de orden cancelada
func (p *Publisher) PublishOrderCancelled(ctx context.Context, order *models.Order, reason string) error {
	event := &OrderEvent{
		EventType:    "cancelled",
		OrderID:      order.ID.Hex(),
		OrderNumber:  order.OrderNumber,
		UserID:       order.UserID,
		Type:         string(order.Type),
		Status:       string(order.Status),
		CryptoSymbol: order.CryptoSymbol,
		Quantity:     order.Quantity.String(),
		Price:        order.Price.String(),
		TotalAmount:  order.TotalAmount.String(),
		Fee:          order.Fee.String(),
		Timestamp:    time.Now(),
		ErrorMessage: reason,
	}

	return p.publish("orders.cancelled", event)
}

// PublishOrderFailed publica evento de orden fallida
func (p *Publisher) PublishOrderFailed(ctx context.Context, order *models.Order, reason string) error {
	event := &OrderEvent{
		EventType:    "failed",
		OrderID:      order.ID.Hex(),
		OrderNumber:  order.OrderNumber,
		UserID:       order.UserID,
		Type:         string(order.Type),
		Status:       string(order.Status),
		CryptoSymbol: order.CryptoSymbol,
		Quantity:     order.Quantity.String(),
		Price:        order.Price.String(),
		TotalAmount:  order.TotalAmount.String(),
		Fee:          order.Fee.String(),
		Timestamp:    time.Now(),
		ErrorMessage: reason,
	}

	return p.publish("orders.failed", event)
}

// publish publica un evento al exchange
func (p *Publisher) publish(routingKey string, event *OrderEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = p.channel.Publish(
		p.exchange,  // exchange
		routingKey,  // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // mensajes persistentes
			Timestamp:    time.Now(),
			Body:         body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published event: %s for order %s", routingKey, event.OrderID)
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

// HealthCheck verifica la conexión
func (p *Publisher) HealthCheck() error {
	if p.connection == nil || p.connection.IsClosed() {
		return fmt.Errorf("RabbitMQ connection is closed")
	}
	return nil
}
