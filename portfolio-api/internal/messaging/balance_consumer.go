package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/streadway/amqp"
	"github.com/sirupsen/logrus"
)

// BalanceResponseConsumer consumes balance responses from Users API
type BalanceResponseConsumer struct {
	conn            *amqp.Connection
	channel         *amqp.Channel
	queueName       string
	pendingRequests map[string]chan *BalanceResponseMessage // correlation_id -> response channel
	mu              sync.RWMutex
	logger          *logrus.Logger
}

// NewBalanceResponseConsumer creates a new balance response consumer
func NewBalanceResponseConsumer(rabbitURL, queueName string, logger *logrus.Logger) (*BalanceResponseConsumer, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange for responses
	err = channel.ExchangeDeclare(
		"balance.response.exchange", // name
		"direct",                     // type
		true,                         // durable
		false,                        // auto-deleted
		false,                        // internal
		false,                        // no-wait
		nil,                          // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare response exchange: %w", err)
	}

	// Declare queue for balance responses
	queue, err := channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		amqp.Table{
			"x-message-ttl":          60000, // 60 seconds TTL
			"x-dead-letter-exchange": "balance.response.dlq",
		},
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = channel.QueueBind(
		queue.Name,                  // queue name
		queueName,                   // routing key (same as queue name)
		"balance.response.exchange", // exchange
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	logger.Infof("✅ Balance response consumer initialized (queue: %s)", queueName)

	return &BalanceResponseConsumer{
		conn:            conn,
		channel:         channel,
		queueName:       queueName,
		pendingRequests: make(map[string]chan *BalanceResponseMessage),
		logger:          logger,
	}, nil
}

// Start starts consuming balance responses in the background
func (c *BalanceResponseConsumer) Start(ctx context.Context) error {
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

	c.logger.Info("🔄 Balance response consumer started")

	go func() {
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("🛑 Balance response consumer shutting down")
				return

			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn("Message channel closed")
					return
				}

				// Process response
				var response BalanceResponseMessage
				if err := json.Unmarshal(msg.Body, &response); err != nil {
					c.logger.Errorf("Failed to unmarshal response: %v", err)
					msg.Nack(false, false) // Send to DLQ
					continue
				}

				c.logger.Debugf("📨 Received balance response (correlation_id: %s, user_id: %d, success: %v)",
					response.CorrelationID, response.UserID, response.Success)

				// Find waiting goroutine
				c.mu.RLock()
				responseChan, exists := c.pendingRequests[response.CorrelationID]
				c.mu.RUnlock()

				if exists {
					// Send response to waiting goroutine
					select {
					case responseChan <- &response:
						msg.Ack(false)
						c.logger.Debugf("✅ Matched response to pending request (correlation_id: %s)", response.CorrelationID)
					case <-time.After(1 * time.Second):
						c.logger.Warnf("Timeout sending response to channel (correlation_id: %s)", response.CorrelationID)
						msg.Nack(false, true) // Requeue
					}
				} else {
					// Orphaned response - no one waiting
					c.logger.Warnf("Orphaned response received (correlation_id: %s) - sending to DLQ", response.CorrelationID)
					msg.Nack(false, false) // Send to DLQ
				}
			}
		}
	}()

	return nil
}

// WaitForResponse waits for a balance response with the given correlation ID
// Returns the response or an error if timeout occurs
func (c *BalanceResponseConsumer) WaitForResponse(correlationID string, timeout time.Duration) (*BalanceResponseMessage, error) {
	// Create response channel
	responseChan := make(chan *BalanceResponseMessage, 1)

	// Register pending request
	c.mu.Lock()
	c.pendingRequests[correlationID] = responseChan
	c.mu.Unlock()

	// Cleanup function
	defer func() {
		c.mu.Lock()
		delete(c.pendingRequests, correlationID)
		close(responseChan)
		c.mu.Unlock()
	}()

	// Wait for response or timeout
	select {
	case response := <-responseChan:
		if response == nil {
			return nil, fmt.Errorf("received nil response")
		}
		return response, nil

	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for balance response (correlation_id: %s)", correlationID)
	}
}

// Close closes the consumer channel and connection
func (c *BalanceResponseConsumer) Close() error {
	// Close all pending request channels
	c.mu.Lock()
	for _, ch := range c.pendingRequests {
		close(ch)
	}
	c.pendingRequests = make(map[string]chan *BalanceResponseMessage)
	c.mu.Unlock()

	if err := c.channel.Close(); err != nil {
		c.logger.Warnf("Error closing channel: %v", err)
	}
	if err := c.conn.Close(); err != nil {
		c.logger.Warnf("Error closing connection: %v", err)
		return err
	}
	c.logger.Info("Balance response consumer closed")
	return nil
}
