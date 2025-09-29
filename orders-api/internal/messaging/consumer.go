package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

type Consumer struct {
	connection    *amqp.Connection
	channel       *amqp.Channel
	config        *ConsumerConfig
	handlers      map[string]MessageHandler
	queues        map[string]bool
	consuming     bool
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
}

type ConsumerConfig struct {
	URL            string
	QueuePrefix    string
	ConsumerTag    string
	PrefetchCount  int
	AutoAck        bool
	Exclusive      bool
	NoLocal        bool
	NoWait         bool
	WorkerCount    int
	RetryDelay     time.Duration
	MaxRetries     int
	DeadLetterTTL  time.Duration
}

type MessageHandler func(ctx context.Context, message *EventMessage) error

type ConsumerStats struct {
	MessagesReceived int64
	MessagesProcessed int64
	MessagesFailed   int64
	AverageProcessTime time.Duration
	LastMessageTime  time.Time
	WorkerStatus     map[int]string
	mu              sync.RWMutex
}

func NewConsumer(config *ConsumerConfig) (*Consumer, error) {
	conn, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if config.PrefetchCount > 0 {
		err = ch.Qos(config.PrefetchCount, 0, false)
		if err != nil {
			ch.Close()
			conn.Close()
			return nil, fmt.Errorf("failed to set QoS: %w", err)
		}
	}

	consumer := &Consumer{
		connection: conn,
		channel:    ch,
		config:     config,
		handlers:   make(map[string]MessageHandler),
		queues:     make(map[string]bool),
		stopChan:   make(chan struct{}),
	}

	return consumer, nil
}

func (c *Consumer) RegisterHandler(routingKey string, handler MessageHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[routingKey] = handler
	log.Printf("Handler registered for routing key: %s", routingKey)
	return nil
}

func (c *Consumer) setupQueue(routingKey, exchangeName string) (string, error) {
	queueName := fmt.Sprintf("%s.%s", c.config.QueuePrefix, routingKey)

	args := amqp.Table{}
	if c.config.DeadLetterTTL > 0 {
		args["x-message-ttl"] = int64(c.config.DeadLetterTTL.Milliseconds())
		args["x-dead-letter-exchange"] = fmt.Sprintf("%s.dlx", exchangeName)
		args["x-dead-letter-routing-key"] = fmt.Sprintf("dlx.%s", routingKey)
	}

	queue, err := c.channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		c.config.Exclusive,
		c.config.NoWait,
		args,
	)
	if err != nil {
		return "", fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	err = c.channel.QueueBind(
		queue.Name,
		routingKey,
		exchangeName,
		c.config.NoWait,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to bind queue %s to exchange %s: %w", queue.Name, exchangeName, err)
	}

	c.queues[queueName] = true
	log.Printf("Queue %s setup successfully for routing key %s", queueName, routingKey)

	return queueName, nil
}

func (c *Consumer) StartConsuming(ctx context.Context, exchangeName string) error {
	c.mu.Lock()
	if c.consuming {
		c.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	c.consuming = true
	c.mu.Unlock()

	log.Printf("Starting consumer for exchange %s with %d workers", exchangeName, c.config.WorkerCount)

	for routingKey := range c.handlers {
		queueName, err := c.setupQueue(routingKey, exchangeName)
		if err != nil {
			return fmt.Errorf("failed to setup queue for routing key %s: %w", routingKey, err)
		}

		for i := 0; i < c.config.WorkerCount; i++ {
			c.wg.Add(1)
			go c.consumeWorker(ctx, queueName, routingKey, i)
		}
	}

	c.wg.Add(1)
	go c.monitorConsumer(ctx)

	return nil
}

func (c *Consumer) consumeWorker(ctx context.Context, queueName, routingKey string, workerID int) {
	defer c.wg.Done()

	consumerTag := fmt.Sprintf("%s-worker-%d", c.config.ConsumerTag, workerID)
	log.Printf("Worker %d started consuming from queue %s", workerID, queueName)

	deliveries, err := c.channel.Consume(
		queueName,
		consumerTag,
		c.config.AutoAck,
		c.config.Exclusive,
		c.config.NoLocal,
		c.config.NoWait,
		nil,
	)
	if err != nil {
		log.Printf("Failed to start consuming from queue %s: %v", queueName, err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping due to context cancellation", workerID)
			return
		case <-c.stopChan:
			log.Printf("Worker %d stopping due to stop signal", workerID)
			return
		case delivery, ok := <-deliveries:
			if !ok {
				log.Printf("Worker %d stopping due to closed delivery channel", workerID)
				return
			}

			c.processMessage(ctx, &delivery, routingKey, workerID)
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, delivery *amqp.Delivery, routingKey string, workerID int) {
	start := time.Now()

	c.mu.RLock()
	handler, exists := c.handlers[routingKey]
	c.mu.RUnlock()

	if !exists {
		log.Printf("No handler found for routing key %s", routingKey)
		if !c.config.AutoAck {
			delivery.Nack(false, false) // Don't requeue
		}
		return
	}

	var message EventMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		log.Printf("Worker %d failed to unmarshal message: %v", workerID, err)
		if !c.config.AutoAck {
			delivery.Nack(false, false) // Don't requeue malformed messages
		}
		return
	}

	log.Printf("Worker %d processing message %s of type %s", workerID, message.ID, message.Type)

	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := handler(processCtx, &message)
	processingTime := time.Since(start)

	if err != nil {
		log.Printf("Worker %d failed to process message %s: %v (took %v)", workerID, message.ID, err, processingTime)

		retryCount := c.getRetryCount(delivery)
		if retryCount < c.config.MaxRetries {
			log.Printf("Retrying message %s (attempt %d/%d)", message.ID, retryCount+1, c.config.MaxRetries)

			if !c.config.AutoAck {
				delivery.Nack(false, true) // Requeue for retry
			}
		} else {
			log.Printf("Message %s exceeded max retries, sending to DLQ", message.ID)

			if !c.config.AutoAck {
				delivery.Nack(false, false) // Send to DLQ
			}
		}
		return
	}

	log.Printf("Worker %d successfully processed message %s in %v", workerID, message.ID, processingTime)

	if !c.config.AutoAck {
		delivery.Ack(false)
	}
}

func (c *Consumer) getRetryCount(delivery *amqp.Delivery) int {
	if delivery.Headers == nil {
		return 0
	}

	if retryCount, ok := delivery.Headers["retry_count"]; ok {
		if count, ok := retryCount.(int); ok {
			return count
		}
	}

	return 0
}

func (c *Consumer) monitorConsumer(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.logConsumerStats()
		}
	}
}

func (c *Consumer) logConsumerStats() {
	c.mu.RLock()
	handlerCount := len(c.handlers)
	queueCount := len(c.queues)
	c.mu.RUnlock()

	log.Printf("Consumer stats: %d handlers, %d queues, %d workers",
		handlerCount, queueCount, c.config.WorkerCount)

	if c.connection != nil && !c.connection.IsClosed() {
		log.Printf("RabbitMQ connection is healthy")
	} else {
		log.Printf("RabbitMQ connection is not healthy")
	}
}

func (c *Consumer) Stop() error {
	c.mu.Lock()
	if !c.consuming {
		c.mu.Unlock()
		return fmt.Errorf("consumer is not running")
	}
	c.consuming = false
	c.mu.Unlock()

	log.Println("Stopping consumer...")

	close(c.stopChan)
	c.wg.Wait()

	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			log.Printf("Error closing channel: %v", err)
		}
	}

	if c.connection != nil {
		if err := c.connection.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
			return err
		}
	}

	log.Println("Consumer stopped successfully")
	return nil
}

func (c *Consumer) HealthCheck() error {
	if c.connection == nil || c.connection.IsClosed() {
		return fmt.Errorf("RabbitMQ connection is closed")
	}

	if c.channel == nil {
		return fmt.Errorf("RabbitMQ channel is not available")
	}

	c.mu.RLock()
	consuming := c.consuming
	c.mu.RUnlock()

	if !consuming {
		return fmt.Errorf("consumer is not running")
	}

	return nil
}

func (c *Consumer) GetHandlerCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.handlers)
}

func (c *Consumer) GetQueueCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.queues)
}

func (c *Consumer) IsConsuming() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.consuming
}

func DefaultConsumerConfig() *ConsumerConfig {
	return &ConsumerConfig{
		URL:           "amqp://guest:guest@localhost:5672/",
		QueuePrefix:   "orders",
		ConsumerTag:   "orders-consumer",
		PrefetchCount: 10,
		AutoAck:       false,
		Exclusive:     false,
		NoLocal:       false,
		NoWait:        false,
		WorkerCount:   5,
		RetryDelay:    5 * time.Second,
		MaxRetries:    3,
		DeadLetterTTL: 24 * time.Hour,
	}
}