package messaging

import "time"

// BalanceRequestMessage represents a request for user balance sent to Users API
type BalanceRequestMessage struct {
	CorrelationID string    `json:"correlation_id"` // UUID for matching response
	UserID        int64     `json:"user_id"`        // User to query
	RequestedBy   string    `json:"requested_by"`   // "portfolio-api"
	Timestamp     time.Time `json:"timestamp"`
}

// BalanceResponseMessage represents the response containing user balance from Users API
type BalanceResponseMessage struct {
	CorrelationID string    `json:"correlation_id"` // Matches request
	UserID        int64     `json:"user_id"`
	Balance       string    `json:"balance"`        // Decimal as string for precision
	Currency      string    `json:"currency"`       // e.g., "USD"
	Success       bool      `json:"success"`
	Error         string    `json:"error,omitempty"` // Error message if success=false
	Timestamp     time.Time `json:"timestamp"`
}
