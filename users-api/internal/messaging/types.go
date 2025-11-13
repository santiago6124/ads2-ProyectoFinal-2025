package messaging

import "time"

// BalanceRequestMessage represents a request for user balance
// Published by: Portfolio API
// Consumed by: Users API Worker
type BalanceRequestMessage struct {
	CorrelationID string    `json:"correlation_id"` // UUID for matching response
	UserID        int64     `json:"user_id"`        // User to query
	RequestedBy   string    `json:"requested_by"`   // Service name (e.g., "portfolio-api")
	Timestamp     time.Time `json:"timestamp"`
}

// BalanceResponseMessage represents the response containing user balance
// Published by: Users API Worker
// Consumed by: Portfolio API
type BalanceResponseMessage struct {
	CorrelationID string    `json:"correlation_id"` // Matches request
	UserID        int64     `json:"user_id"`
	Balance       string    `json:"balance"`        // Decimal as string for precision
	Currency      string    `json:"currency"`       // e.g., "USD"
	Success       bool      `json:"success"`
	Error         string    `json:"error,omitempty"` // Error message if success=false
	Timestamp     time.Time `json:"timestamp"`
}
