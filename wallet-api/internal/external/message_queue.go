package external

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"

	"wallet-api/internal/models"
)

type MessageQueue interface {
	PublishTransactionEvent(ctx context.Context, event *TransactionEvent) error
	PublishWalletEvent(ctx context.Context, event *WalletEvent) error
	PublishAuditEvent(ctx context.Context, event *AuditEvent) error
	PublishNotificationEvent(ctx context.Context, event *NotificationEvent) error
	PublishComplianceAlert(ctx context.Context, alert *ComplianceAlert) error
	Close() error
}

type messageQueue struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	config   *MessageQueueConfig
	exchanges map[string]bool
}

type MessageQueueConfig struct {
	URL             string
	ExchangeName    string
	RetryAttempts   int
	RetryDelay      time.Duration
	MessageTTL      time.Duration
	PrefetchCount   int
	EnableDeadLetter bool
}

func NewMessageQueue(config *MessageQueueConfig) (MessageQueue, error) {
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}
	if config.MessageTTL == 0 {
		config.MessageTTL = 24 * time.Hour
	}
	if config.PrefetchCount == 0 {
		config.PrefetchCount = 10
	}

	mq := &messageQueue{
		config:    config,
		exchanges: make(map[string]bool),
	}

	if err := mq.connect(); err != nil {
		return nil, err
	}

	if err := mq.setupExchanges(); err != nil {
		return nil, err
	}

	return mq, nil
}

// Event types
type TransactionEvent struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"` // "created", "completed", "failed", "reversed"
	TransactionID string                 `json:"transaction_id"`
	WalletID      string                 `json:"wallet_id"`
	UserID        int64                  `json:"user_id"`
	Amount        string                 `json:"amount"`
	Currency      string                 `json:"currency"`
	Type          string                 `json:"type"`
	Status        string                 `json:"status"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type WalletEvent struct {
	EventID     string                 `json:"event_id"`
	EventType   string                 `json:"event_type"` // "created", "suspended", "balance_updated", "lock_added"
	WalletID    string                 `json:"wallet_id"`
	UserID      int64                  `json:"user_id"`
	Balance     string                 `json:"balance,omitempty"`
	Currency    string                 `json:"currency,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type AuditEvent struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"` // "transaction_audit", "admin_action", "compliance_check"
	UserID        int64                  `json:"user_id,omitempty"`
	AdminID       string                 `json:"admin_id,omitempty"`
	Action        string                 `json:"action"`
	Resource      string                 `json:"resource"`
	Success       bool                   `json:"success"`
	IPAddress     string                 `json:"ip_address"`
	RiskScore     int                    `json:"risk_score"`
	Timestamp     time.Time              `json:"timestamp"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

type NotificationEvent struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"` // "email", "sms", "push", "webhook"
	UserID        int64                  `json:"user_id"`
	Channel       []string               `json:"channel"`
	Priority      string                 `json:"priority"`
	Subject       string                 `json:"subject"`
	Message       string                 `json:"message"`
	TemplateID    string                 `json:"template_id,omitempty"`
	TemplateData  map[string]interface{} `json:"template_data,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	ScheduledFor  *time.Time             `json:"scheduled_for,omitempty"`
}

type ComplianceAlert struct {
	AlertID       string                 `json:"alert_id"`
	AlertType     string                 `json:"alert_type"` // "suspicious_activity", "velocity_limit", "large_transaction"
	Severity      string                 `json:"severity"`
	UserID        int64                  `json:"user_id,omitempty"`
	TransactionID string                 `json:"transaction_id,omitempty"`
	Description   string                 `json:"description"`
	RiskScore     int                    `json:"risk_score"`
	Evidence      map[string]interface{} `json:"evidence"`
	RequiresAction bool                  `json:"requires_action"`
	Timestamp     time.Time              `json:"timestamp"`
	ExpiresAt     *time.Time             `json:"expires_at,omitempty"`
}

// Connection management
func (mq *messageQueue) connect() error {
	var err error
	mq.conn, err = amqp.Dial(mq.config.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	mq.channel, err = mq.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS
	if err := mq.channel.Qos(mq.config.PrefetchCount, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	return nil
}

func (mq *messageQueue) setupExchanges() error {
	exchanges := []struct {
		name    string
		kind    string
		durable bool
	}{
		{"wallet.transactions", "topic", true},
		{"wallet.events", "topic", true},
		{"wallet.audit", "topic", true},
		{"wallet.notifications", "topic", true},
		{"wallet.compliance", "topic", true},
	}

	for _, exchange := range exchanges {
		err := mq.channel.ExchangeDeclare(
			exchange.name,    // name
			exchange.kind,    // type
			exchange.durable, // durable
			false,           // auto-deleted
			false,           // internal
			false,           // no-wait
			nil,             // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare exchange %s: %w", exchange.name, err)
		}
		mq.exchanges[exchange.name] = true
	}

	// Setup dead letter exchange if enabled
	if mq.config.EnableDeadLetter {
		err := mq.channel.ExchangeDeclare(
			"wallet.deadletter", // name
			"direct",           // type
			true,               // durable
			false,              // auto-deleted
			false,              // internal
			false,              // no-wait
			nil,                // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare dead letter exchange: %w", err)
		}
	}

	return nil
}

// Publishing methods
func (mq *messageQueue) PublishTransactionEvent(ctx context.Context, event *TransactionEvent) error {
	routingKey := fmt.Sprintf("transaction.%s.%s", event.Type, event.EventType)
	return mq.publishMessage(ctx, "wallet.transactions", routingKey, event)
}

func (mq *messageQueue) PublishWalletEvent(ctx context.Context, event *WalletEvent) error {
	routingKey := fmt.Sprintf("wallet.%s", event.EventType)
	return mq.publishMessage(ctx, "wallet.events", routingKey, event)
}

func (mq *messageQueue) PublishAuditEvent(ctx context.Context, event *AuditEvent) error {
	routingKey := fmt.Sprintf("audit.%s.%s", event.Resource, event.Action)
	return mq.publishMessage(ctx, "wallet.audit", routingKey, event)
}

func (mq *messageQueue) PublishNotificationEvent(ctx context.Context, event *NotificationEvent) error {
	routingKey := fmt.Sprintf("notification.%s.%s", event.EventType, event.Priority)
	return mq.publishMessage(ctx, "wallet.notifications", routingKey, event)
}

func (mq *messageQueue) PublishComplianceAlert(ctx context.Context, alert *ComplianceAlert) error {
	routingKey := fmt.Sprintf("compliance.%s.%s", alert.AlertType, alert.Severity)
	return mq.publishMessage(ctx, "wallet.compliance", routingKey, alert)
}

func (mq *messageQueue) publishMessage(ctx context.Context, exchange, routingKey string, message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Prepare publishing options
	publishing := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		Timestamp:    time.Now(),
		MessageId:    mq.generateMessageID(),
		DeliveryMode: amqp.Persistent, // Make message persistent
	}

	// Set TTL if configured
	if mq.config.MessageTTL > 0 {
		publishing.Expiration = fmt.Sprintf("%d", mq.config.MessageTTL.Milliseconds())
	}

	// Add correlation ID from context if available
	if correlationID := ctx.Value("correlation_id"); correlationID != nil {
		publishing.CorrelationId = correlationID.(string)
	}

	// Publish with retry logic
	var publishErr error
	for attempt := 0; attempt < mq.config.RetryAttempts; attempt++ {
		publishErr = mq.channel.Publish(
			exchange,   // exchange
			routingKey, // routing key
			false,      // mandatory
			false,      // immediate
			publishing, // message
		)

		if publishErr == nil {
			return nil
		}

		// If connection is closed, try to reconnect
		if mq.conn.IsClosed() {
			if reconnectErr := mq.reconnect(); reconnectErr != nil {
				log.Printf("Failed to reconnect to RabbitMQ: %v", reconnectErr)
			}
		}

		if attempt < mq.config.RetryAttempts-1 {
			time.Sleep(mq.config.RetryDelay * time.Duration(attempt+1))
		}
	}

	return fmt.Errorf("failed to publish message after %d attempts: %w", mq.config.RetryAttempts, publishErr)
}

func (mq *messageQueue) reconnect() error {
	// Close existing connections
	if mq.channel != nil {
		mq.channel.Close()
	}
	if mq.conn != nil {
		mq.conn.Close()
	}

	// Reconnect
	if err := mq.connect(); err != nil {
		return err
	}

	return mq.setupExchanges()
}

func (mq *messageQueue) generateMessageID() string {
	return fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

func (mq *messageQueue) Close() error {
	var errs []error

	if mq.channel != nil {
		if err := mq.channel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close channel: %w", err))
		}
	}

	if mq.conn != nil {
		if err := mq.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing message queue: %v", errs)
	}

	return nil
}

// Helper functions to create events from domain models
func CreateTransactionEvent(transaction *models.Transaction, eventType string) *TransactionEvent {
	return &TransactionEvent{
		EventID:       fmt.Sprintf("tx_event_%s_%d", transaction.TransactionID, time.Now().UnixNano()),
		EventType:     eventType,
		TransactionID: transaction.TransactionID,
		WalletID:      transaction.WalletID.Hex(),
		UserID:        transaction.UserID,
		Amount:        transaction.Amount.Value.String(),
		Currency:      transaction.Amount.Currency,
		Type:          transaction.Type,
		Status:        transaction.Status,
		Timestamp:     time.Now(),
		Metadata: map[string]interface{}{
			"fee":        transaction.Amount.Fee.String(),
			"net_amount": transaction.Amount.Net.String(),
			"reference":  transaction.Reference,
		},
	}
}

func CreateWalletEvent(wallet *models.Wallet, eventType string) *WalletEvent {
	return &WalletEvent{
		EventID:   fmt.Sprintf("wallet_event_%s_%d", wallet.ID.Hex(), time.Now().UnixNano()),
		EventType: eventType,
		WalletID:  wallet.ID.Hex(),
		UserID:    wallet.UserID,
		Balance:   wallet.Balance.Total.String(),
		Currency:  wallet.Balance.Currency,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"available_balance": wallet.Balance.Available.String(),
			"locked_balance":    wallet.Balance.Locked.String(),
			"status":            wallet.Status,
		},
	}
}

func CreateAuditEvent(userID int64, adminID, action, resource string, success bool, ipAddress string, riskScore int, details map[string]interface{}) *AuditEvent {
	return &AuditEvent{
		EventID:   fmt.Sprintf("audit_event_%d_%d", userID, time.Now().UnixNano()),
		EventType: "audit",
		UserID:    userID,
		AdminID:   adminID,
		Action:    action,
		Resource:  resource,
		Success:   success,
		IPAddress: ipAddress,
		RiskScore: riskScore,
		Timestamp: time.Now(),
		Details:   details,
	}
}

func CreateNotificationEvent(userID int64, channels []string, priority, subject, message string, templateData map[string]interface{}) *NotificationEvent {
	return &NotificationEvent{
		EventID:      fmt.Sprintf("notification_%d_%d", userID, time.Now().UnixNano()),
		EventType:    "notification",
		UserID:       userID,
		Channel:      channels,
		Priority:     priority,
		Subject:      subject,
		Message:      message,
		TemplateData: templateData,
		Timestamp:    time.Now(),
	}
}

func CreateComplianceAlert(alertType, severity string, userID int64, transactionID, description string, riskScore int, evidence map[string]interface{}, requiresAction bool) *ComplianceAlert {
	alert := &ComplianceAlert{
		AlertID:        fmt.Sprintf("alert_%s_%d_%d", alertType, userID, time.Now().UnixNano()),
		AlertType:      alertType,
		Severity:       severity,
		UserID:         userID,
		TransactionID:  transactionID,
		Description:    description,
		RiskScore:      riskScore,
		Evidence:       evidence,
		RequiresAction: requiresAction,
		Timestamp:      time.Now(),
	}

	// Set expiration for non-critical alerts
	if severity != "critical" {
		expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days
		alert.ExpiresAt = &expiresAt
	}

	return alert
}

// Publisher wrapper that implements the service interface
type EventPublisher struct {
	messageQueue MessageQueue
}

func NewEventPublisher(messageQueue MessageQueue) *EventPublisher {
	return &EventPublisher{
		messageQueue: messageQueue,
	}
}

func (p *EventPublisher) PublishTransactionCreated(transaction *models.Transaction) error {
	event := CreateTransactionEvent(transaction, "created")
	return p.messageQueue.PublishTransactionEvent(context.Background(), event)
}

func (p *EventPublisher) PublishTransactionCompleted(transaction *models.Transaction) error {
	event := CreateTransactionEvent(transaction, "completed")
	return p.messageQueue.PublishTransactionEvent(context.Background(), event)
}

func (p *EventPublisher) PublishTransactionFailed(transaction *models.Transaction) error {
	event := CreateTransactionEvent(transaction, "failed")
	return p.messageQueue.PublishTransactionEvent(context.Background(), event)
}

func (p *EventPublisher) PublishWalletCreated(wallet *models.Wallet) error {
	event := CreateWalletEvent(wallet, "created")
	return p.messageQueue.PublishWalletEvent(context.Background(), event)
}

func (p *EventPublisher) PublishWalletBalanceUpdated(wallet *models.Wallet) error {
	event := CreateWalletEvent(wallet, "balance_updated")
	return p.messageQueue.PublishWalletEvent(context.Background(), event)
}

func (p *EventPublisher) PublishComplianceAlert(alertType, severity string, userID int64, transactionID, description string, riskScore int, evidence map[string]interface{}) error {
	alert := CreateComplianceAlert(alertType, severity, userID, transactionID, description, riskScore, evidence, riskScore >= 8)
	return p.messageQueue.PublishComplianceAlert(context.Background(), alert)
}

func (p *EventPublisher) Close() error {
	return p.messageQueue.Close()
}