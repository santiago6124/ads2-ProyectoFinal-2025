package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"users-api/internal/services"
)

// BalanceRequestConsumer consumes balance requests and publishes responses
type BalanceRequestConsumer struct {
	conn              *amqp.Connection
	channel           *amqp.Channel
	queueName         string
	userService       services.UserService
	responsePublisher *BalanceResponsePublisher
	logger            *logrus.Logger
}

// NewBalanceRequestConsumer creates a new balance request consumer
func NewBalanceRequestConsumer(
	rabbitURL string,
	queueName string,
	userService services.UserService,
	responsePublisher *BalanceResponsePublisher,
	logger *logrus.Logger,
) (*BalanceRequestConsumer, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange for requests
	err = channel.ExchangeDeclare(
		"balance.request.exchange", // name
		"direct",                    // type
		true,                        // durable
		false,                       // auto-deleted
		false,                       // internal
		false,                       // no-wait
		nil,                         // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare request exchange: %w", err)
	}

	// Declare queue for balance requests
	queue, err := channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		amqp.Table{
			"x-dead-letter-exchange": "balance.request.dlx",
		},
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = channel.QueueBind(
		queue.Name,                 // queue name
		"balance.request",          // routing key
		"balance.request.exchange", // exchange
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	// Set QoS - process up to 10 messages concurrently
	err = channel.Qos(
		10,    // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	logger.Infof("✅ Balance request consumer initialized (queue: %s)", queueName)

	return &BalanceRequestConsumer{
		conn:              conn,
		channel:           channel,
		queueName:         queueName,
		userService:       userService,
		responsePublisher: responsePublisher,
		logger:            logger,
	}, nil
}

// Start starts consuming balance requests
func (c *BalanceRequestConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		c.queueName, // queue
		"",          // consumer tag
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	c.logger.Info("🔄 Balance request worker started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("🛑 Balance request worker shutting down...")
			return ctx.Err()

		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("message channel closed")
			}

			// Process message
			if err := c.processRequest(ctx, msg); err != nil {
				c.logger.Errorf("Failed to process balance request: %v", err)
				// Requeue message on error (up to dead letter queue limits)
				msg.Nack(false, true)
			} else {
				// Acknowledge successful processing
				msg.Ack(false)
			}
		}
	}
}

// processRequest handles a single balance request message
func (c *BalanceRequestConsumer) processRequest(ctx context.Context, msg amqp.Delivery) error {
	var request BalanceRequestMessage
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		return fmt.Errorf("failed to unmarshal request: %w", err)
	}

	c.logger.Infof("📨 Received balance request for user %d (correlation: %s)",
		request.UserID, request.CorrelationID)

	// Create response message
	response := BalanceResponseMessage{
		CorrelationID: request.CorrelationID,
		UserID:        request.UserID,
		Timestamp:     time.Now(),
		Currency:      "USD",
	}

	// Query user balance
	user, err := c.userService.GetUserByID(int32(request.UserID))
	if err != nil {
		// User not found or database error
		c.logger.Warnf("Failed to get user %d: %v", request.UserID, err)
		response.Success = false
		response.Error = fmt.Sprintf("user not found: %v", err)
	} else {
		// Success - return balance
		response.Success = true
		response.Balance = fmt.Sprintf("%.2f", user.CurrentBalance)

		c.logger.Infof("✅ Found user %d with balance: %s", request.UserID, response.Balance)
	}

	// Publish response
	if err := c.responsePublisher.PublishResponse(ctx, response); err != nil {
		return fmt.Errorf("failed to publish response: %w", err)
	}

	c.logger.Infof("✅ Sent balance response for user %d (success: %v)",
		request.UserID, response.Success)

	return nil
}

// Close closes the consumer channel and connection
func (c *BalanceRequestConsumer) Close() error {
	if err := c.channel.Close(); err != nil {
		c.logger.Warnf("Error closing channel: %v", err)
	}
	if err := c.conn.Close(); err != nil {
		c.logger.Warnf("Error closing connection: %v", err)
		return err
	}
	c.logger.Info("Balance request consumer closed")
	return nil
}
