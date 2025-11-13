package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// BalanceResponsePublisher publishes balance responses to Portfolio API
type BalanceResponsePublisher struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	exchange     string
	routingKey   string
	logger       *logrus.Logger
}

// NewBalanceResponsePublisher creates a new balance response publisher
func NewBalanceResponsePublisher(rabbitURL, exchange, routingKey string, logger *logrus.Logger) (*BalanceResponsePublisher, error) {
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

	logger.Infof("✅ Balance response publisher initialized (exchange: %s, routing_key: %s)", exchange, routingKey)

	return &BalanceResponsePublisher{
		conn:       conn,
		channel:    channel,
		exchange:   exchange,
		routingKey: routingKey,
		logger:     logger,
	}, nil
}

// PublishResponse publishes a balance response message
func (p *BalanceResponsePublisher) PublishResponse(ctx context.Context, response BalanceResponseMessage) error {
	body, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	err = p.channel.PublishWithContext(
		ctx,
		p.exchange,   // exchange
		p.routingKey, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			CorrelationId: response.CorrelationID,
			ContentType:   "application/json",
			Body:          body,
			Timestamp:     time.Now(),
			DeliveryMode:  amqp.Persistent, // Durable message
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish response: %w", err)
	}

	p.logger.Debugf("📤 Published balance response (correlation_id: %s, user_id: %d, success: %v)",
		response.CorrelationID, response.UserID, response.Success)

	return nil
}

// Close closes the publisher channel and connection
func (p *BalanceResponsePublisher) Close() error {
	if err := p.channel.Close(); err != nil {
		p.logger.Warnf("Error closing channel: %v", err)
	}
	if err := p.conn.Close(); err != nil {
		p.logger.Warnf("Error closing connection: %v", err)
		return err
	}
	p.logger.Info("Balance response publisher closed")
	return nil
}
