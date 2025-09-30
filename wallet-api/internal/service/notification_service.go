package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"wallet-api/internal/models"
)

type NotificationService interface {
	NotifyTransactionCompleted(ctx context.Context, transaction *models.Transaction, wallet *models.Wallet) error
	NotifyTransactionFailed(ctx context.Context, transaction *models.Transaction, reason string) error
	NotifyFundsLocked(ctx context.Context, userID int64, lockID string, amount decimal.Decimal) error
	NotifyFundsReleased(ctx context.Context, userID int64, lockID string, amount decimal.Decimal) error
	NotifyBalanceAdjustment(ctx context.Context, userID int64, adjustment decimal.Decimal, reason string) error
	NotifyWalletSuspended(ctx context.Context, userID int64, reason string) error
	NotifyReconciliationDiscrepancy(ctx context.Context, userID int64, discrepancy decimal.Decimal) error
	NotifySystemAlert(ctx context.Context, alert *SystemAlert) error
}

type notificationService struct {
	// In a real implementation, this would include message queue clients,
	// email service clients, webhook clients, etc.
}

func NewNotificationService() NotificationService {
	return &notificationService{}
}

type NotificationEvent struct {
	EventID     string                 `json:"event_id"`
	EventType   string                 `json:"event_type"`
	UserID      int64                  `json:"user_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
	Priority    string                 `json:"priority"`   // "low", "medium", "high", "critical"
	Channel     []string               `json:"channel"`    // "email", "sms", "push", "webhook"
	Metadata    map[string]interface{} `json:"metadata"`
}

type SystemAlert struct {
	AlertID     string                 `json:"alert_id"`
	AlertType   string                 `json:"alert_type"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
}

func (s *notificationService) NotifyTransactionCompleted(ctx context.Context, transaction *models.Transaction, wallet *models.Wallet) error {
	event := &NotificationEvent{
		EventID:   fmt.Sprintf("tx_completed_%s", transaction.TransactionID),
		EventType: "transaction_completed",
		UserID:    transaction.UserID,
		Timestamp: time.Now(),
		Priority:  "medium",
		Channel:   []string{"push", "email"},
		Data: map[string]interface{}{
			"transaction_id":   transaction.TransactionID,
			"transaction_type": transaction.Type,
			"amount":          transaction.Amount.Value.String(),
			"currency":        transaction.Amount.Currency,
			"new_balance":     wallet.Balance.Total.String(),
			"description":     transaction.GetDescription(),
		},
		Metadata: map[string]interface{}{
			"wallet_id": wallet.ID.Hex(),
		},
	}

	return s.sendNotification(ctx, event)
}

func (s *notificationService) NotifyTransactionFailed(ctx context.Context, transaction *models.Transaction, reason string) error {
	event := &NotificationEvent{
		EventID:   fmt.Sprintf("tx_failed_%s", transaction.TransactionID),
		EventType: "transaction_failed",
		UserID:    transaction.UserID,
		Timestamp: time.Now(),
		Priority:  "high",
		Channel:   []string{"push", "email"},
		Data: map[string]interface{}{
			"transaction_id":   transaction.TransactionID,
			"transaction_type": transaction.Type,
			"amount":          transaction.Amount.Value.String(),
			"currency":        transaction.Amount.Currency,
			"failure_reason":  reason,
			"description":     transaction.GetDescription(),
		},
		Metadata: map[string]interface{}{
			"wallet_id": transaction.WalletID.Hex(),
		},
	}

	return s.sendNotification(ctx, event)
}

func (s *notificationService) NotifyFundsLocked(ctx context.Context, userID int64, lockID string, amount decimal.Decimal) error {
	event := &NotificationEvent{
		EventID:   fmt.Sprintf("funds_locked_%s", lockID),
		EventType: "funds_locked",
		UserID:    userID,
		Timestamp: time.Now(),
		Priority:  "medium",
		Channel:   []string{"push"},
		Data: map[string]interface{}{
			"lock_id": lockID,
			"amount":  amount.String(),
			"message": fmt.Sprintf("Funds locked: %s USD", amount.String()),
		},
	}

	return s.sendNotification(ctx, event)
}

func (s *notificationService) NotifyFundsReleased(ctx context.Context, userID int64, lockID string, amount decimal.Decimal) error {
	event := &NotificationEvent{
		EventID:   fmt.Sprintf("funds_released_%s", lockID),
		EventType: "funds_released",
		UserID:    userID,
		Timestamp: time.Now(),
		Priority:  "medium",
		Channel:   []string{"push"},
		Data: map[string]interface{}{
			"lock_id": lockID,
			"amount":  amount.String(),
			"message": fmt.Sprintf("Funds released: %s USD", amount.String()),
		},
	}

	return s.sendNotification(ctx, event)
}

func (s *notificationService) NotifyBalanceAdjustment(ctx context.Context, userID int64, adjustment decimal.Decimal, reason string) error {
	event := &NotificationEvent{
		EventID:   fmt.Sprintf("balance_adjustment_%d_%d", userID, time.Now().Unix()),
		EventType: "balance_adjustment",
		UserID:    userID,
		Timestamp: time.Now(),
		Priority:  "high",
		Channel:   []string{"push", "email"},
		Data: map[string]interface{}{
			"adjustment_amount": adjustment.String(),
			"reason":           reason,
			"message":          fmt.Sprintf("Balance adjusted by %s USD. Reason: %s", adjustment.String(), reason),
		},
	}

	return s.sendNotification(ctx, event)
}

func (s *notificationService) NotifyWalletSuspended(ctx context.Context, userID int64, reason string) error {
	event := &NotificationEvent{
		EventID:   fmt.Sprintf("wallet_suspended_%d", userID),
		EventType: "wallet_suspended",
		UserID:    userID,
		Timestamp: time.Now(),
		Priority:  "critical",
		Channel:   []string{"push", "email", "sms"},
		Data: map[string]interface{}{
			"reason":  reason,
			"message": fmt.Sprintf("Your wallet has been suspended. Reason: %s", reason),
			"action_required": "Contact support to reactivate your wallet",
		},
	}

	return s.sendNotification(ctx, event)
}

func (s *notificationService) NotifyReconciliationDiscrepancy(ctx context.Context, userID int64, discrepancy decimal.Decimal) error {
	event := &NotificationEvent{
		EventID:   fmt.Sprintf("reconciliation_discrepancy_%d", userID),
		EventType: "reconciliation_discrepancy",
		UserID:    userID,
		Timestamp: time.Now(),
		Priority:  "high",
		Channel:   []string{"email"},
		Data: map[string]interface{}{
			"discrepancy_amount": discrepancy.String(),
			"message":           fmt.Sprintf("Reconciliation discrepancy detected: %s USD", discrepancy.String()),
			"action_taken":      "Balance has been automatically adjusted",
		},
	}

	return s.sendNotification(ctx, event)
}

func (s *notificationService) NotifySystemAlert(ctx context.Context, alert *SystemAlert) error {
	// System alerts are sent to administrators, not regular users
	event := &NotificationEvent{
		EventID:   alert.AlertID,
		EventType: "system_alert",
		UserID:    0, // System alerts don't have a specific user
		Timestamp: alert.Timestamp,
		Priority:  s.mapSeverityToPriority(alert.Severity),
		Channel:   []string{"email", "webhook"},
		Data: map[string]interface{}{
			"alert_type":   alert.AlertType,
			"severity":     alert.Severity,
			"title":        alert.Title,
			"description":  alert.Description,
			"alert_data":   alert.Data,
		},
	}

	return s.sendSystemNotification(ctx, event)
}

func (s *notificationService) sendNotification(ctx context.Context, event *NotificationEvent) error {
	// In a real implementation, this would:
	// 1. Serialize the event
	// 2. Send to message queue (RabbitMQ, Kafka, etc.)
	// 3. Log the notification for audit purposes
	// 4. Handle retry logic for failed notifications

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to serialize notification event: %w", err)
	}

	// Simulate sending notification
	fmt.Printf("Sending notification: %s\n", string(eventJSON))

	// Log notification for audit
	return s.logNotification(ctx, event)
}

func (s *notificationService) sendSystemNotification(ctx context.Context, event *NotificationEvent) error {
	// System notifications might have different routing/handling
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to serialize system notification: %w", err)
	}

	// Simulate sending system notification
	fmt.Printf("Sending system notification: %s\n", string(eventJSON))

	return s.logNotification(ctx, event)
}

func (s *notificationService) logNotification(ctx context.Context, event *NotificationEvent) error {
	// In a real implementation, this would log to audit trail
	// For now, we'll just return nil
	return nil
}

func (s *notificationService) mapSeverityToPriority(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

// EventDispatcher handles event routing and processing
type EventDispatcher struct {
	notificationService NotificationService
}

func NewEventDispatcher(notificationService NotificationService) *EventDispatcher {
	return &EventDispatcher{
		notificationService: notificationService,
	}
}

func (d *EventDispatcher) DispatchTransactionEvent(ctx context.Context, event string, transaction *models.Transaction, wallet *models.Wallet, metadata map[string]interface{}) error {
	switch event {
	case "transaction.completed":
		return d.notificationService.NotifyTransactionCompleted(ctx, transaction, wallet)
	case "transaction.failed":
		reason := "Unknown error"
		if reasonVal, ok := metadata["reason"].(string); ok {
			reason = reasonVal
		}
		return d.notificationService.NotifyTransactionFailed(ctx, transaction, reason)
	default:
		return fmt.Errorf("unknown transaction event: %s", event)
	}
}

func (d *EventDispatcher) DispatchWalletEvent(ctx context.Context, event string, userID int64, metadata map[string]interface{}) error {
	switch event {
	case "wallet.suspended":
		reason := "Administrative action"
		if reasonVal, ok := metadata["reason"].(string); ok {
			reason = reasonVal
		}
		return d.notificationService.NotifyWalletSuspended(ctx, userID, reason)
	case "funds.locked":
		lockID := metadata["lock_id"].(string)
		amount := metadata["amount"].(decimal.Decimal)
		return d.notificationService.NotifyFundsLocked(ctx, userID, lockID, amount)
	case "funds.released":
		lockID := metadata["lock_id"].(string)
		amount := metadata["amount"].(decimal.Decimal)
		return d.notificationService.NotifyFundsReleased(ctx, userID, lockID, amount)
	default:
		return fmt.Errorf("unknown wallet event: %s", event)
	}
}

func (d *EventDispatcher) DispatchSystemEvent(ctx context.Context, alert *SystemAlert) error {
	return d.notificationService.NotifySystemAlert(ctx, alert)
}