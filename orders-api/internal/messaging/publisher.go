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

type Publisher struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	config     *MessagingConfig
	exchanges  map[string]bool
}

type MessagingConfig struct {
	URL             string
	ExchangeName    string
	DeadLetterExchange string
	MaxRetries      int
	RetryDelay      time.Duration
	MessageTTL      time.Duration
	Persistent      bool
}

type EventMessage struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Source      string                 `json:"source"`
	Subject     string                 `json:"subject"`
	Data        interface{}            `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RoutingKey  string                 `json:"routing_key"`
	Exchange    string                 `json:"exchange"`
	RetryCount  int                    `json:"retry_count"`
	Priority    uint8                  `json:"priority"`
}

type OrderEvent struct {
	OrderID       string                 `json:"order_id"`
	UserID        int                    `json:"user_id"`
	Type          string                 `json:"type"`
	Status        string                 `json:"status"`
	Amount        string                 `json:"amount"`
	Price         string                 `json:"price"`
	Symbol        string                 `json:"symbol"`
	Timestamp     time.Time              `json:"timestamp"`
	ExecutionTime time.Duration          `json:"execution_time,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

func NewPublisher(config *MessagingConfig) (*Publisher, error) {
	conn, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	publisher := &Publisher{
		connection: conn,
		channel:    ch,
		config:     config,
		exchanges:  make(map[string]bool),
	}

	if err := publisher.setupExchanges(); err != nil {
		publisher.Close()
		return nil, fmt.Errorf("failed to setup exchanges: %w", err)
	}

	return publisher, nil
}

func (p *Publisher) setupExchanges() error {
	exchanges := []struct {
		name       string
		kind       string
		durable    bool
		autoDelete bool
	}{
		{p.config.ExchangeName, "topic", true, false},
		{p.config.DeadLetterExchange, "topic", true, false},
		{"orders.events", "topic", true, false},
		{"orders.audit", "topic", true, false},
		{"orders.monitoring", "topic", true, false},
	}

	for _, exchange := range exchanges {
		err := p.channel.ExchangeDeclare(
			exchange.name,
			exchange.kind,
			exchange.durable,
			exchange.autoDelete,
			false,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to declare exchange %s: %w", exchange.name, err)
		}
		p.exchanges[exchange.name] = true
		log.Printf("Exchange %s declared successfully", exchange.name)
	}

	return nil
}

func (p *Publisher) PublishOrderCreated(ctx context.Context, order *models.Order) error {
	event := &OrderEvent{
		OrderID:   order.ID.Hex(),
		UserID:    order.UserID,
		Type:      string(order.Type),
		Status:    string(order.Status),
		Amount:    order.Quantity.String(),
		Price:     order.OrderPrice.String(),
		Symbol:    order.CryptoSymbol,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"order_kind": string(order.OrderKind),
			"created_at": order.CreatedAt,
			"user_agent": order.Metadata.UserAgent,
			"ip_address": order.Metadata.IPAddress,
		},
	}

	message := &EventMessage{
		ID:         fmt.Sprintf("order_created_%s_%d", order.ID.Hex(), time.Now().UnixNano()),
		Type:       "orders.created",
		Source:     "orders-api",
		Subject:    fmt.Sprintf("order.%s", order.ID.Hex()),
		Data:       event,
		Timestamp:  time.Now(),
		Version:    "1.0",
		RoutingKey: "orders.created",
		Exchange:   "orders.events",
		Priority:   5,
	}

	return p.publishMessage(ctx, message)
}

func (p *Publisher) PublishOrderUpdated(ctx context.Context, order *models.Order, oldStatus models.OrderStatus) error {
	event := &OrderEvent{
		OrderID:   order.ID.Hex(),
		UserID:    order.UserID,
		Type:      string(order.Type),
		Status:    string(order.Status),
		Amount:    order.Quantity.String(),
		Price:     order.OrderPrice.String(),
		Symbol:    order.CryptoSymbol,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"old_status":   string(oldStatus),
			"updated_at":   order.UpdatedAt,
			"order_kind":   string(order.OrderKind),
			"updated_by":   "system",
		},
	}

	message := &EventMessage{
		ID:         fmt.Sprintf("order_updated_%s_%d", order.ID.Hex(), time.Now().UnixNano()),
		Type:       "orders.updated",
		Source:     "orders-api",
		Subject:    fmt.Sprintf("order.%s", order.ID.Hex()),
		Data:       event,
		Timestamp:  time.Now(),
		Version:    "1.0",
		RoutingKey: "orders.updated",
		Exchange:   "orders.events",
		Priority:   6,
	}

	return p.publishMessage(ctx, message)
}

func (p *Publisher) PublishOrderExecuted(ctx context.Context, order *models.Order, executionResult *models.ExecutionResult) error {
	event := &OrderEvent{
		OrderID:       order.ID.Hex(),
		UserID:        order.UserID,
		Type:          string(order.Type),
		Status:        string(order.Status),
		Amount:        order.Quantity.String(),
		Price:         order.OrderPrice.String(),
		Symbol:        order.CryptoSymbol,
		Timestamp:     time.Now(),
		ExecutionTime: executionResult.ExecutionTime,
		Metadata: map[string]interface{}{
			"execution_id":     executionResult.OrderID,
			"market_price":     executionResult.MarketPrice.MarketPrice.String(),
			"execution_price":  executionResult.MarketPrice.ExecutionPrice.String(),
			"slippage":         executionResult.MarketPrice.Slippage.String(),
			"total_fee":        executionResult.FeeCalculation.TotalFee.String(),
			"fee_type":         executionResult.FeeCalculation.FeeType,
			"execution_steps":  len(executionResult.Steps),
			"success":          executionResult.Success,
		},
	}

	message := &EventMessage{
		ID:         fmt.Sprintf("order_executed_%s_%d", order.ID.Hex(), time.Now().UnixNano()),
		Type:       "orders.executed",
		Source:     "orders-api",
		Subject:    fmt.Sprintf("order.%s", order.ID.Hex()),
		Data:       event,
		Timestamp:  time.Now(),
		Version:    "1.0",
		RoutingKey: "orders.executed",
		Exchange:   "orders.events",
		Priority:   8,
	}

	return p.publishMessage(ctx, message)
}

func (p *Publisher) PublishOrderFailed(ctx context.Context, order *models.Order, errorMessage string) error {
	event := &OrderEvent{
		OrderID:      order.ID.Hex(),
		UserID:       order.UserID,
		Type:         string(order.Type),
		Status:       string(order.Status),
		Amount:       order.Quantity.String(),
		Price:        order.OrderPrice.String(),
		Symbol:       order.CryptoSymbol,
		Timestamp:    time.Now(),
		ErrorMessage: errorMessage,
		Metadata: map[string]interface{}{
			"failure_reason": errorMessage,
			"failed_at":      time.Now(),
			"order_kind":     string(order.OrderKind),
		},
	}

	message := &EventMessage{
		ID:         fmt.Sprintf("order_failed_%s_%d", order.ID.Hex(), time.Now().UnixNano()),
		Type:       "orders.failed",
		Source:     "orders-api",
		Subject:    fmt.Sprintf("order.%s", order.ID.Hex()),
		Data:       event,
		Timestamp:  time.Now(),
		Version:    "1.0",
		RoutingKey: "orders.failed",
		Exchange:   "orders.events",
		Priority:   9,
	}

	return p.publishMessage(ctx, message)
}

func (p *Publisher) PublishOrderCancelled(ctx context.Context, order *models.Order, reason string) error {
	event := &OrderEvent{
		OrderID:   order.ID.Hex(),
		UserID:    order.UserID,
		Type:      string(order.Type),
		Status:    string(order.Status),
		Amount:    order.Quantity.String(),
		Price:     order.OrderPrice.String(),
		Symbol:    order.CryptoSymbol,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"cancellation_reason": reason,
			"cancelled_at":        time.Now(),
			"order_kind":          string(order.OrderKind),
		},
	}

	message := &EventMessage{
		ID:         fmt.Sprintf("order_cancelled_%s_%d", order.ID.Hex(), time.Now().UnixNano()),
		Type:       "orders.cancelled",
		Source:     "orders-api",
		Subject:    fmt.Sprintf("order.%s", order.ID.Hex()),
		Data:       event,
		Timestamp:  time.Now(),
		Version:    "1.0",
		RoutingKey: "orders.cancelled",
		Exchange:   "orders.events",
		Priority:   7,
	}

	return p.publishMessage(ctx, message)
}

func (p *Publisher) PublishAuditEvent(ctx context.Context, eventType, orderID string, userID int, details map[string]interface{}) error {
	auditEvent := map[string]interface{}{
		"event_type": eventType,
		"order_id":   orderID,
		"user_id":    userID,
		"timestamp":  time.Now(),
		"details":    details,
	}

	message := &EventMessage{
		ID:         fmt.Sprintf("audit_%s_%s_%d", eventType, orderID, time.Now().UnixNano()),
		Type:       fmt.Sprintf("orders.audit.%s", eventType),
		Source:     "orders-api",
		Subject:    fmt.Sprintf("audit.%s", orderID),
		Data:       auditEvent,
		Timestamp:  time.Now(),
		Version:    "1.0",
		RoutingKey: fmt.Sprintf("orders.audit.%s", eventType),
		Exchange:   "orders.audit",
		Priority:   3,
	}

	return p.publishMessage(ctx, message)
}

func (p *Publisher) PublishMetricsEvent(ctx context.Context, metricType string, data map[string]interface{}) error {
	metricsEvent := map[string]interface{}{
		"metric_type": metricType,
		"timestamp":   time.Now(),
		"data":        data,
		"source":      "orders-api",
	}

	message := &EventMessage{
		ID:         fmt.Sprintf("metrics_%s_%d", metricType, time.Now().UnixNano()),
		Type:       fmt.Sprintf("orders.metrics.%s", metricType),
		Source:     "orders-api",
		Subject:    fmt.Sprintf("metrics.%s", metricType),
		Data:       metricsEvent,
		Timestamp:  time.Now(),
		Version:    "1.0",
		RoutingKey: fmt.Sprintf("orders.metrics.%s", metricType),
		Exchange:   "orders.monitoring",
		Priority:   2,
	}

	return p.publishMessage(ctx, message)
}

func (p *Publisher) publishMessage(ctx context.Context, message *EventMessage) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	headers := amqp.Table{
		"message_id":   message.ID,
		"message_type": message.Type,
		"source":       message.Source,
		"version":      message.Version,
		"timestamp":    message.Timestamp.Unix(),
		"retry_count":  message.RetryCount,
	}

	if message.Metadata != nil {
		for k, v := range message.Metadata {
			headers[fmt.Sprintf("metadata_%s", k)] = v
		}
	}

	publishing := amqp.Publishing{
		Headers:      headers,
		ContentType:  "application/json",
		DeliveryMode: amqp.Transient,
		Priority:     message.Priority,
		MessageId:    message.ID,
		Timestamp:    message.Timestamp,
		Type:         message.Type,
		Body:         body,
	}

	if p.config.Persistent {
		publishing.DeliveryMode = amqp.Persistent
	}

	if p.config.MessageTTL > 0 {
		publishing.Expiration = fmt.Sprintf("%d", p.config.MessageTTL.Milliseconds())
	}

	return p.channel.Publish(
		message.Exchange,
		message.RoutingKey,
		false, // mandatory
		false, // immediate
		publishing,
	)
}

func (p *Publisher) PublishWithRetry(ctx context.Context, message *EventMessage, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := p.publishMessage(ctx, message)
		if err == nil {
			return nil
		}

		lastErr = err
		log.Printf("Failed to publish message (attempt %d/%d): %v", attempt+1, maxRetries+1, err)

		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(p.config.RetryDelay * time.Duration(attempt+1)):
				continue
			}
		}
	}

	return fmt.Errorf("failed to publish message after %d attempts: %w", maxRetries+1, lastErr)
}

func (p *Publisher) Close() error {
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			log.Printf("Error closing channel: %v", err)
		}
	}

	if p.connection != nil {
		if err := p.connection.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
			return err
		}
	}

	return nil
}

func (p *Publisher) HealthCheck() error {
	if p.connection == nil || p.connection.IsClosed() {
		return fmt.Errorf("RabbitMQ connection is closed")
	}

	if p.channel == nil {
		return fmt.Errorf("RabbitMQ channel is not available")
	}

	return nil
}

func DefaultMessagingConfig() *MessagingConfig {
	return &MessagingConfig{
		URL:                "amqp://guest:guest@localhost:5672/",
		ExchangeName:       "orders",
		DeadLetterExchange: "orders.dlx",
		MaxRetries:         3,
		RetryDelay:         5 * time.Second,
		MessageTTL:         24 * time.Hour,
		Persistent:         true,
	}
}