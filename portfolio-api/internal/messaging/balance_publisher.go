package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	amqp "github.com/streadway/amqp"
	"github.com/sirupsen/logrus"
)

// BalancePublisher publishes balance request messages to Users API
type BalancePublisher struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	exchange   string
	routingKey string
	logger     *logrus.Logger
}

// NewBalancePublisher creates a new balance request publisher
func NewBalancePublisher(rabbitURL, exchange, routingKey string, logger *logrus.Logger) (*BalancePublisher, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange (idempotent)
	err = channel.ExchangeDeclare(
		exchange, // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	logger.Infof("✅ Balance request publisher initialized (exchange: %s, routing_key: %s)", exchange, routingKey)

	return &BalancePublisher{
		conn:       conn,
		channel:    channel,
		exchange:   exchange,
		routingKey: routingKey,
		logger:     logger,
	}, nil
}

// RequestBalance publishes a balance request and returns the correlation ID
func (p *BalancePublisher) RequestBalance(ctx context.Context, userID int64) (string, error) {
	// Generate unique correlation ID
	correlationID := uuid.New().String()

	// Create request message
	request := BalanceRequestMessage{
		CorrelationID: correlationID,
		UserID:        userID,
		RequestedBy:   "portfolio-api",
		Timestamp:     time.Now(),
	}

	// Marshal to JSON
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Publish message
	err = p.channel.Publish(
		p.exchange,   // exchange
		p.routingKey, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			CorrelationId: correlationID,
			ContentType:   "application/json",
			Body:          body,
			Timestamp:     time.Now(),
			DeliveryMode:  amqp.Persistent, // Durable message
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to publish request: %w", err)
	}

	p.logger.Debugf("📤 Published balance request (correlation_id: %s, user_id: %d)", correlationID, userID)

	return correlationID, nil
}

// Close closes the publisher channel and connection
func (p *BalancePublisher) Close() error {
	if err := p.channel.Close(); err != nil {
		p.logger.Warnf("Error closing channel: %v", err)
	}
	if err := p.conn.Close(); err != nil {
		p.logger.Warnf("Error closing connection: %v", err)
		return err
	}
	p.logger.Info("Balance request publisher closed")
	return nil
}
