package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	"search-api/internal/services"
)

// Consumer handles RabbitMQ message consumption
type Consumer struct {
	connection      *amqp.Connection
	channel         *amqp.Channel
	config          *ConsumerConfig
	handlers        map[string]MessageHandler
	trendingHandler *services.TrendingEventHandler
	indexingService *services.IndexingService
	logger          *logrus.Logger
	consuming       bool
	stopChan        chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
}

// ConsumerConfig represents consumer configuration
type ConsumerConfig struct {
	URL           string
	ExchangeName  string
	QueueName     string
	RoutingKeys   []string
	ConsumerTag   string
	PrefetchCount int
	AutoAck       bool
	WorkerCount   int
	RetryDelay    time.Duration
	MaxRetries    int
	DeadLetterTTL time.Duration
}

// MessageHandler defines the interface for message handling
type MessageHandler func(ctx context.Context, message *EventMessage) error

// EventMessage represents a message from the event system
type EventMessage struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Source     string                 `json:"source"`
	Subject    string                 `json:"subject"`
	Data       map[string]interface{} `json:"data"`
	Timestamp  time.Time              `json:"timestamp"`
	Version    string                 `json:"version"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	RetryCount int                    `json:"retry_count"`
}

// OrderEvent represents an order-related event
type OrderEvent struct {
	EventType    string `json:"event_type"`
	OrderID      string `json:"order_id"`
	UserID       int    `json:"user_id"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	CryptoSymbol string `json:"crypto_symbol"`
	Quantity     string `json:"quantity"`
	Price        string `json:"price"`
	TotalAmount  string `json:"total_amount"`
	Fee          string `json:"fee"`
	Timestamp    string `json:"timestamp"`
	ErrorMessage string `json:"error_message"`
}

// PriceEvent represents a price change event
type PriceEvent struct {
	CryptoID      string  `json:"crypto_id"`
	Symbol        string  `json:"symbol"`
	OldPrice      float64 `json:"old_price"`
	NewPrice      float64 `json:"new_price"`
	ChangePercent float64 `json:"change_percent"`
	Volume24h     float64 `json:"volume_24h"`
	Timestamp     string  `json:"timestamp"`
}

// NewConsumer creates a new RabbitMQ consumer
func NewConsumer(config *ConsumerConfig, trendingHandler *services.TrendingEventHandler, indexingService *services.IndexingService, logger *logrus.Logger) (*Consumer, error) {
	conn, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS for prefetch control
	if config.PrefetchCount > 0 {
		err = ch.Qos(config.PrefetchCount, 0, false)
		if err != nil {
			ch.Close()
			conn.Close()
			return nil, fmt.Errorf("failed to set QoS: %w", err)
		}
	}

	consumer := &Consumer{
		connection:      conn,
		channel:         ch,
		config:          config,
		handlers:        make(map[string]MessageHandler),
		trendingHandler: trendingHandler,
		indexingService: indexingService,
		logger:          logger,
		stopChan:        make(chan struct{}),
	}

	// Register default handlers
	consumer.registerDefaultHandlers()

	return consumer, nil
}

// Start starts consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.consuming {
		c.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	c.consuming = true
	c.mu.Unlock()

	// Setup exchange and queue
	if err := c.setupInfrastructure(); err != nil {
		return fmt.Errorf("failed to setup infrastructure: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"exchange":     c.config.ExchangeName,
		"queue":        c.config.QueueName,
		"routing_keys": c.config.RoutingKeys,
		"workers":      c.config.WorkerCount,
	}).Info("Starting RabbitMQ consumer")

	// Start worker goroutines
	for i := 0; i < c.config.WorkerCount; i++ {
		c.wg.Add(1)
		go c.worker(ctx, i)
	}

	c.logger.Info("RabbitMQ consumer started successfully")
	return nil
}

// Stop stops the consumer
func (c *Consumer) Stop() error {
	c.mu.Lock()
	if !c.consuming {
		c.mu.Unlock()
		return fmt.Errorf("consumer is not running")
	}
	c.consuming = false
	c.mu.Unlock()

	c.logger.Info("Stopping RabbitMQ consumer...")

	// Signal workers to stop
	close(c.stopChan)

	// Wait for all workers to finish
	c.wg.Wait()

	// Close channel and connection
	if c.channel != nil {
		c.channel.Close()
	}
	if c.connection != nil {
		c.connection.Close()
	}

	c.logger.Info("RabbitMQ consumer stopped")
	return nil
}

// RegisterHandler registers a message handler for a specific message type
func (c *Consumer) RegisterHandler(messageType string, handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[messageType] = handler
	c.logger.WithField("message_type", messageType).Debug("Message handler registered")
}

// setupInfrastructure sets up the RabbitMQ exchange and queue
func (c *Consumer) setupInfrastructure() error {
	// Declare exchange
	err := c.channel.ExchangeDeclare(
		c.config.ExchangeName,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Setup dead letter exchange if configured
	if c.config.DeadLetterTTL > 0 {
		dlxName := c.config.ExchangeName + ".dlx"
		err := c.channel.ExchangeDeclare(
			dlxName,
			"topic",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to declare dead letter exchange: %w", err)
		}
	}

	// Declare queue
	args := amqp.Table{}
	if c.config.DeadLetterTTL > 0 {
		args["x-message-ttl"] = int64(c.config.DeadLetterTTL.Milliseconds())
		args["x-dead-letter-exchange"] = c.config.ExchangeName + ".dlx"
	}

	queue, err := c.channel.QueueDeclare(
		c.config.QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		args,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange with routing keys
	for _, routingKey := range c.config.RoutingKeys {
		err = c.channel.QueueBind(
			queue.Name,
			routingKey,
			c.config.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue with routing key %s: %w", routingKey, err)
		}
	}

	return nil
}

// worker processes messages from the queue
func (c *Consumer) worker(ctx context.Context, workerID int) {
	defer c.wg.Done()

	consumerTag := fmt.Sprintf("%s-worker-%d", c.config.ConsumerTag, workerID)

	// Start consuming
	deliveries, err := c.channel.Consume(
		c.config.QueueName,
		consumerTag,
		c.config.AutoAck,
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		c.logger.WithError(err).Errorf("Worker %d failed to start consuming", workerID)
		return
	}

	c.logger.Infof("Worker %d started consuming", workerID)

	for {
		select {
		case <-ctx.Done():
			c.logger.Infof("Worker %d stopping due to context cancellation", workerID)
			return
		case <-c.stopChan:
			c.logger.Infof("Worker %d stopping due to stop signal", workerID)
			return
		case delivery, ok := <-deliveries:
			if !ok {
				c.logger.Infof("Worker %d stopping due to closed delivery channel", workerID)
				return
			}

			c.processMessage(ctx, &delivery, workerID)
		}
	}
}

// processMessage processes a single message
func (c *Consumer) processMessage(ctx context.Context, delivery *amqp.Delivery, workerID int) {
	startTime := time.Now()

	// Process message with timeout
	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Parse message
	var eventMsg EventMessage
	if err := json.Unmarshal(delivery.Body, &eventMsg); err != nil {
		c.logger.WithFields(logrus.Fields{
			"worker": workerID,
			"error":  err,
		}).Error("Failed to unmarshal message")

		if !c.config.AutoAck {
			delivery.Nack(false, false) // Don't requeue malformed messages
		}
		return
	}

	// Handle legacy/simple order events published directly by orders-api
	var orderEvent OrderEvent
	if err := json.Unmarshal(delivery.Body, &orderEvent); err == nil && orderEvent.OrderID != "" {
		eventType := delivery.RoutingKey
		if eventType == "" {
			eventType = orderEvent.EventType
			if eventType != "" && !strings.HasPrefix(eventType, "orders.") {
				eventType = "orders." + eventType
			}
		}

		if c.indexingService != nil && eventType != "" {
			legacy := &services.LegacyOrderEvent{
				EventType:    orderEvent.EventType,
				OrderID:      orderEvent.OrderID,
				UserID:       orderEvent.UserID,
				Type:         orderEvent.Type,
				Status:       orderEvent.Status,
				CryptoSymbol: orderEvent.CryptoSymbol,
				Quantity:     orderEvent.Quantity,
				Price:        orderEvent.Price,
				TotalAmount:  orderEvent.TotalAmount,
				Fee:          orderEvent.Fee,
				Timestamp:    orderEvent.Timestamp,
				ErrorMessage: orderEvent.ErrorMessage,
			}
			if err := c.indexingService.SyncOrderFromEvent(processCtx, orderEvent.OrderID, eventType, legacy); err != nil {
				c.logger.WithFields(logrus.Fields{
					"order_id":   orderEvent.OrderID,
					"event_type": eventType,
					"error":      err,
				}).Error("Failed to sync order from legacy event")

				if !c.config.AutoAck {
					delivery.Nack(false, true)
				}
				return
			}
		}

		// Update trending metrics if applicable
		if c.trendingHandler != nil && strings.HasSuffix(eventType, "orders.executed") && orderEvent.CryptoSymbol != "" {
			if totalValue, err := parseFloat64(orderEvent.TotalAmount); err == nil {
				c.trendingHandler.HandleOrderEvent(orderEvent.CryptoSymbol, totalValue)
			}
		}

		c.logger.WithFields(logrus.Fields{
			"worker":     workerID,
			"order_id":   orderEvent.OrderID,
			"event_type": eventType,
			"duration":   time.Since(startTime),
		}).Info("Order event processed via legacy flow")

		if !c.config.AutoAck {
			delivery.Ack(false)
		}
		return
	}

	c.logger.WithFields(logrus.Fields{
		"worker":       workerID,
		"message_id":   eventMsg.ID,
		"message_type": eventMsg.Type,
	}).Debug("Processing message")

	err := c.handleMessage(processCtx, &eventMsg)
	processingTime := time.Since(startTime)

	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"worker":     workerID,
			"message_id": eventMsg.ID,
			"error":      err,
			"duration":   processingTime,
		}).Error("Message processing failed")

		if !c.config.AutoAck {
			// Check retry count
			retryCount := c.getRetryCount(delivery)
			if retryCount < c.config.MaxRetries {
				c.logger.WithFields(logrus.Fields{
					"message_id":  eventMsg.ID,
					"retry_count": retryCount + 1,
					"max_retries": c.config.MaxRetries,
				}).Info("Requeuing message for retry")

				delivery.Nack(false, true) // Requeue for retry
			} else {
				c.logger.WithField("message_id", eventMsg.ID).Warn("Message exceeded max retries, sending to DLQ")
				delivery.Nack(false, false) // Send to DLQ
			}
		}
		return
	}

	c.logger.WithFields(logrus.Fields{
		"worker":     workerID,
		"message_id": eventMsg.ID,
		"duration":   processingTime,
	}).Info("Message processed successfully")

	if !c.config.AutoAck {
		delivery.Ack(false)
	}
}

// handleMessage routes messages to appropriate handlers
func (c *Consumer) handleMessage(ctx context.Context, eventMsg *EventMessage) error {
	c.mu.RLock()
	handler, exists := c.handlers[eventMsg.Type]
	c.mu.RUnlock()

	if !exists {
		c.logger.WithField("message_type", eventMsg.Type).Warn("No handler found for message type")
		return nil // Don't treat as error to avoid infinite retries
	}

	return handler(ctx, eventMsg)
}

// registerDefaultHandlers registers default message handlers
func (c *Consumer) registerDefaultHandlers() {
	// Register order event handlers
	c.RegisterHandler("orders.executed", c.handleOrderExecuted)
	c.RegisterHandler("orders.created", c.handleOrderCreated)
	c.RegisterHandler("orders.cancelled", c.handleOrderCancelled)
	c.RegisterHandler("orders.failed", c.handleOrderFailed)

	// Register price event handlers
	c.RegisterHandler("market.price_change", c.handlePriceChange)
	c.RegisterHandler("market.volume_change", c.handleVolumeChange)

	// Register search event handlers
	c.RegisterHandler("search.query", c.handleSearchQuery)
}

// Default message handlers

func (c *Consumer) handleOrderExecuted(ctx context.Context, eventMsg *EventMessage) error {
	// Extract order ID from event data
	orderID, ok := eventMsg.Data["order_id"].(string)
	if !ok {
		// Try alternative field names
		if id, ok := eventMsg.Data["id"].(string); ok {
			orderID = id
		} else {
			return fmt.Errorf("order_id not found in event data")
		}
	}

	// Sync order to search index by fetching complete order from orders-api
	if c.indexingService != nil {
		if err := c.indexingService.SyncOrderFromEvent(ctx, orderID, "orders.executed", nil); err != nil {
			c.logger.WithFields(logrus.Fields{
				"order_id": orderID,
				"error":    err,
			}).Error("Failed to sync order from event")
			return fmt.Errorf("failed to sync order: %w", err)
		}
	}

	// Update trending score based on order execution (if still needed)
	if c.trendingHandler != nil {
		if cryptoID, ok := eventMsg.Data["crypto_symbol"].(string); ok {
			var totalValue float64
			if tv, ok := eventMsg.Data["total_amount"].(string); ok {
				// Parse string to float64 if needed
				if parsed, err := parseFloat64(tv); err == nil {
					totalValue = parsed
				}
			}
			c.trendingHandler.HandleOrderEvent(cryptoID, totalValue)
		}
	}

	c.logger.WithFields(logrus.Fields{
		"order_id": orderID,
	}).Info("Processed order executed event and synced to index")

	return nil
}

func (c *Consumer) handleOrderCreated(ctx context.Context, eventMsg *EventMessage) error {
	// Extract order ID from event data
	orderID, ok := eventMsg.Data["order_id"].(string)
	if !ok {
		if id, ok := eventMsg.Data["id"].(string); ok {
			orderID = id
		} else {
			return fmt.Errorf("order_id not found in event data")
		}
	}

	// Sync order to search index by fetching complete order from orders-api
	if c.indexingService != nil {
		if err := c.indexingService.SyncOrderFromEvent(ctx, orderID, "orders.created", nil); err != nil {
			c.logger.WithFields(logrus.Fields{
				"order_id": orderID,
				"error":    err,
			}).Error("Failed to sync order from event")
			return fmt.Errorf("failed to sync order: %w", err)
		}
	}

	c.logger.WithField("order_id", orderID).Info("Processed order created event and synced to index")
	return nil
}

func (c *Consumer) handleOrderCancelled(ctx context.Context, eventMsg *EventMessage) error {
	// Extract order ID from event data
	orderID, ok := eventMsg.Data["order_id"].(string)
	if !ok {
		if id, ok := eventMsg.Data["id"].(string); ok {
			orderID = id
		} else {
			return fmt.Errorf("order_id not found in event data")
		}
	}

	// Sync order to search index (will delete from index)
	if c.indexingService != nil {
		if err := c.indexingService.SyncOrderFromEvent(ctx, orderID, "orders.cancelled", nil); err != nil {
			c.logger.WithFields(logrus.Fields{
				"order_id": orderID,
				"error":    err,
			}).Error("Failed to sync order cancellation")
			return fmt.Errorf("failed to sync order cancellation: %w", err)
		}
	}

	c.logger.WithField("order_id", orderID).Info("Processed order cancelled event and synced to index")
	return nil
}

func (c *Consumer) handleOrderFailed(ctx context.Context, eventMsg *EventMessage) error {
	// Extract order ID from event data
	orderID, ok := eventMsg.Data["order_id"].(string)
	if !ok {
		if id, ok := eventMsg.Data["id"].(string); ok {
			orderID = id
		} else {
			return fmt.Errorf("order_id not found in event data")
		}
	}

	// Sync order to search index (update status but keep indexed)
	if c.indexingService != nil {
		if err := c.indexingService.SyncOrderFromEvent(ctx, orderID, "orders.failed", nil); err != nil {
			c.logger.WithFields(logrus.Fields{
				"order_id": orderID,
				"error":    err,
			}).Error("Failed to sync order failure")
			return fmt.Errorf("failed to sync order failure: %w", err)
		}
	}

	c.logger.WithField("order_id", orderID).Info("Processed order failed event and synced to index")
	return nil
}

func (c *Consumer) handlePriceChange(ctx context.Context, eventMsg *EventMessage) error {
	var priceEvent PriceEvent
	if err := c.parseEventData(eventMsg.Data, &priceEvent); err != nil {
		return fmt.Errorf("failed to parse price event: %w", err)
	}

	// Update trending score based on price change
	if c.trendingHandler != nil {
		c.trendingHandler.HandlePriceChangeEvent(priceEvent.CryptoID, priceEvent.ChangePercent)
	}

	c.logger.WithFields(logrus.Fields{
		"crypto_id":      priceEvent.CryptoID,
		"change_percent": priceEvent.ChangePercent,
	}).Debug("Processed price change event")

	return nil
}

func (c *Consumer) handleVolumeChange(ctx context.Context, eventMsg *EventMessage) error {
	var priceEvent PriceEvent
	if err := c.parseEventData(eventMsg.Data, &priceEvent); err != nil {
		return fmt.Errorf("failed to parse volume event: %w", err)
	}

	// Volume changes can indicate trending activity
	if c.trendingHandler != nil && priceEvent.Volume24h > 0 {
		volumeScore := priceEvent.Volume24h / 1000000 // Normalize by 1M
		c.trendingHandler.HandleOrderEvent(priceEvent.CryptoID, volumeScore)
	}

	c.logger.WithFields(logrus.Fields{
		"crypto_id": priceEvent.CryptoID,
		"volume":    priceEvent.Volume24h,
	}).Debug("Processed volume change event")

	return nil
}

func (c *Consumer) handleSearchQuery(ctx context.Context, eventMsg *EventMessage) error {
	cryptoID, ok := eventMsg.Data["crypto_id"].(string)
	if !ok || cryptoID == "" {
		return nil // Skip if no crypto ID
	}

	// Update trending score based on search activity
	if c.trendingHandler != nil {
		c.trendingHandler.HandleSearchEvent(cryptoID)
	}

	c.logger.WithField("crypto_id", cryptoID).Debug("Processed search query event")
	return nil
}

// Helper methods

func (c *Consumer) parseEventData(data map[string]interface{}, target interface{}) error {
	// Convert map to JSON and back to struct
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, target)
}

func (c *Consumer) getRetryCount(delivery *amqp.Delivery) int {
	if delivery.Headers == nil {
		return 0
	}

	if retryCount, ok := delivery.Headers["x-retry-count"]; ok {
		if count, ok := retryCount.(int32); ok {
			return int(count)
		}
	}

	return 0
}

// parseFloat64 parses a string to float64
func parseFloat64(s string) (float64, error) {
	// Remove any whitespace
	s = strings.TrimSpace(s)
	// Try to parse
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse float: %w", err)
	}
	return val, nil
}

// IsConsuming returns whether the consumer is currently consuming
func (c *Consumer) IsConsuming() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.consuming
}

// HealthCheck checks the consumer health
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

// DefaultConsumerConfig returns default consumer configuration
func DefaultConsumerConfig() *ConsumerConfig {
	return &ConsumerConfig{
		URL:           "amqp://guest:guest@localhost:5672/",
		ExchangeName:  "cryptosim",
		QueueName:     "search.sync",
		RoutingKeys:   []string{"orders.#", "market.#", "search.#"},
		ConsumerTag:   "search-consumer",
		PrefetchCount: 10,
		AutoAck:       false,
		WorkerCount:   5,
		RetryDelay:    5 * time.Second,
		MaxRetries:    3,
		DeadLetterTTL: 24 * time.Hour,
	}
}
