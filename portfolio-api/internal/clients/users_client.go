package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"portfolio-api/internal/config"
)

type UsersClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	timeout    time.Duration
	retries    int
}

func NewUsersClient(cfg config.ExternalAPIsConfig) *UsersClient {
	return &UsersClient{
		baseURL: cfg.UsersAPI.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		apiKey:  cfg.UsersAPI.APIKey,
		timeout: cfg.Timeout,
		retries: cfg.RetryCount,
	}
}

// User represents a user in the system
type User struct {
	ID                int64     `json:"id"`
	Email             string    `json:"email"`
	Username          string    `json:"username"`
	FirstName         string    `json:"first_name"`
	LastName          string    `json:"last_name"`
	Status            string    `json:"status"`
	Role              string    `json:"role"`
	EmailVerified     bool      `json:"email_verified"`
	TwoFactorEnabled  bool      `json:"two_factor_enabled"`
	RiskTolerance     string    `json:"risk_tolerance"`
	InvestmentGoals   []string  `json:"investment_goals"`
	PreferredCurrency string    `json:"preferred_currency"`
	Timezone          string    `json:"timezone"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	LastLoginAt       *time.Time `json:"last_login_at,omitempty"`
}

// UserProfile represents detailed user profile information
type UserProfile struct {
	UserID            int64              `json:"user_id"`
	PersonalInfo      PersonalInfo       `json:"personal_info"`
	TradingProfile    TradingProfile     `json:"trading_profile"`
	Preferences       UserPreferences    `json:"preferences"`
	RiskAssessment    RiskAssessment     `json:"risk_assessment"`
	VerificationStatus VerificationStatus `json:"verification_status"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// PersonalInfo represents personal information
type PersonalInfo struct {
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	DateOfBirth  *time.Time `json:"date_of_birth,omitempty"`
	PhoneNumber  string     `json:"phone_number"`
	Address      Address    `json:"address"`
	Occupation   string     `json:"occupation"`
	Income       string     `json:"income"`
}

// Address represents user address
type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// TradingProfile represents trading-related profile information
type TradingProfile struct {
	ExperienceLevel      string   `json:"experience_level"`
	InvestmentHorizon    string   `json:"investment_horizon"`
	RiskTolerance        string   `json:"risk_tolerance"`
	PreferredAssetClasses []string `json:"preferred_asset_classes"`
	TradingGoals         []string `json:"trading_goals"`
	MaxPortfolioSize     string   `json:"max_portfolio_size"`
	LiquidityNeeds       string   `json:"liquidity_needs"`
}

// UserPreferences represents user preferences
type UserPreferences struct {
	Currency           string   `json:"currency"`
	Language           string   `json:"language"`
	Timezone           string   `json:"timezone"`
	NotificationChannels []string `json:"notification_channels"`
	EmailNotifications bool     `json:"email_notifications"`
	PushNotifications  bool     `json:"push_notifications"`
	SMSNotifications   bool     `json:"sms_notifications"`
	NewsletterSubscribed bool   `json:"newsletter_subscribed"`
	MarketingConsent   bool     `json:"marketing_consent"`
}

// RiskAssessment represents user risk assessment
type RiskAssessment struct {
	Score              int       `json:"score"`
	Level              string    `json:"level"`
	Questionnaire      []QAItem  `json:"questionnaire"`
	LastAssessedAt     time.Time `json:"last_assessed_at"`
	NextAssessmentDue  time.Time `json:"next_assessment_due"`
	RecommendedProfile string    `json:"recommended_profile"`
}

// QAItem represents a question and answer item
type QAItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Score    int    `json:"score"`
}

// VerificationStatus represents user verification status
type VerificationStatus struct {
	IdentityVerified bool      `json:"identity_verified"`
	EmailVerified    bool      `json:"email_verified"`
	PhoneVerified    bool      `json:"phone_verified"`
	AddressVerified  bool      `json:"address_verified"`
	BankVerified     bool      `json:"bank_verified"`
	KYCLevel         string    `json:"kyc_level"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
}

// UserSettings represents user settings
type UserSettings struct {
	UserID              int64              `json:"user_id"`
	NotificationSettings NotificationSettings `json:"notification_settings"`
	TradingSettings     TradingSettings    `json:"trading_settings"`
	DisplaySettings     DisplaySettings    `json:"display_settings"`
	SecuritySettings    SecuritySettings   `json:"security_settings"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

// NotificationSettings represents notification preferences
type NotificationSettings struct {
	PriceAlerts        bool     `json:"price_alerts"`
	OrderExecutions    bool     `json:"order_executions"`
	PortfolioUpdates   bool     `json:"portfolio_updates"`
	MarketNews         bool     `json:"market_news"`
	SystemMaintenance  bool     `json:"system_maintenance"`
	SecurityAlerts     bool     `json:"security_alerts"`
	Channels           []string `json:"channels"`
	QuietHours         QuietHours `json:"quiet_hours"`
}

// QuietHours represents notification quiet hours
type QuietHours struct {
	Enabled   bool   `json:"enabled"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Timezone  string `json:"timezone"`
}

// TradingSettings represents trading preferences
type TradingSettings struct {
	DefaultOrderType      string  `json:"default_order_type"`
	ConfirmOrders         bool    `json:"confirm_orders"`
	AutoRebalancing       bool    `json:"auto_rebalancing"`
	StopLossDefault       bool    `json:"stop_loss_default"`
	TakeProfitDefault     bool    `json:"take_profit_default"`
	MaxOrderSize          float64 `json:"max_order_size"`
	DailyTradingLimit     float64 `json:"daily_trading_limit"`
	AllowMarginTrading    bool    `json:"allow_margin_trading"`
	AllowOptionsTrading   bool    `json:"allow_options_trading"`
}

// DisplaySettings represents display preferences
type DisplaySettings struct {
	Theme             string   `json:"theme"`
	ChartType         string   `json:"chart_type"`
	DefaultTimeframe  string   `json:"default_timeframe"`
	ShowPortfolioValue bool    `json:"show_portfolio_value"`
	HiddenColumns     []string `json:"hidden_columns"`
	CustomLayouts     map[string]interface{} `json:"custom_layouts"`
}

// SecuritySettings represents security preferences
type SecuritySettings struct {
	TwoFactorEnabled    bool      `json:"two_factor_enabled"`
	BiometricEnabled    bool      `json:"biometric_enabled"`
	SessionTimeout      int       `json:"session_timeout"`
	IPWhitelisting      bool      `json:"ip_whitelisting"`
	WhitelistedIPs      []string  `json:"whitelisted_ips"`
	LastPasswordChange  time.Time `json:"last_password_change"`
	LoginNotifications  bool      `json:"login_notifications"`
	DeviceManagement    bool      `json:"device_management"`
}

// GetUser retrieves user information by ID
func (uc *UsersClient) GetUser(ctx context.Context, userID int64) (*User, error) {
	url := fmt.Sprintf("%s/users/%d", uc.baseURL, userID)

	var response struct {
		Data User `json:"data"`
	}

	err := uc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get user %d: %w", userID, err)
	}

	return &response.Data, nil
}

// GetUserProfile retrieves detailed user profile
func (uc *UsersClient) GetUserProfile(ctx context.Context, userID int64) (*UserProfile, error) {
	url := fmt.Sprintf("%s/users/%d/profile", uc.baseURL, userID)

	var response struct {
		Data UserProfile `json:"data"`
	}

	err := uc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile %d: %w", userID, err)
	}

	return &response.Data, nil
}

// GetUserSettings retrieves user settings
func (uc *UsersClient) GetUserSettings(ctx context.Context, userID int64) (*UserSettings, error) {
	url := fmt.Sprintf("%s/users/%d/settings", uc.baseURL, userID)

	var response struct {
		Data UserSettings `json:"data"`
	}

	err := uc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings %d: %w", userID, err)
	}

	return &response.Data, nil
}

// UpdateUserProfile updates user profile information
func (uc *UsersClient) UpdateUserProfile(ctx context.Context, userID int64, profile *UserProfile) (*UserProfile, error) {
	url := fmt.Sprintf("%s/users/%d/profile", uc.baseURL, userID)

	var response struct {
		Data UserProfile `json:"data"`
	}

	err := uc.makeRequest(ctx, "PUT", url, profile, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to update user profile %d: %w", userID, err)
	}

	return &response.Data, nil
}

// UpdateUserSettings updates user settings
func (uc *UsersClient) UpdateUserSettings(ctx context.Context, userID int64, settings *UserSettings) (*UserSettings, error) {
	url := fmt.Sprintf("%s/users/%d/settings", uc.baseURL, userID)

	var response struct {
		Data UserSettings `json:"data"`
	}

	err := uc.makeRequest(ctx, "PUT", url, settings, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to update user settings %d: %w", userID, err)
	}

	return &response.Data, nil
}

// GetUsersByIDs retrieves multiple users by their IDs
func (uc *UsersClient) GetUsersByIDs(ctx context.Context, userIDs []int64) (map[int64]*User, error) {
	url := fmt.Sprintf("%s/users/batch", uc.baseURL)

	requestBody := map[string][]int64{
		"user_ids": userIDs,
	}

	var response struct {
		Data map[string]User `json:"data"`
	}

	err := uc.makeRequest(ctx, "POST", url, requestBody, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by IDs: %w", err)
	}

	// Convert string keys to int64
	result := make(map[int64]*User)
	for idStr, user := range response.Data {
		// Parse ID from string
		var id int64
		fmt.Sscanf(idStr, "%d", &id)
		userCopy := user
		result[id] = &userCopy
	}

	return result, nil
}

// ValidateUser validates if a user exists and is active
func (uc *UsersClient) ValidateUser(ctx context.Context, userID int64) (*UserValidation, error) {
	url := fmt.Sprintf("%s/users/%d/validate", uc.baseURL, userID)

	var response struct {
		Data UserValidation `json:"data"`
	}

	err := uc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to validate user %d: %w", userID, err)
	}

	return &response.Data, nil
}

// UserValidation represents user validation result
type UserValidation struct {
	UserID    int64  `json:"user_id"`
	IsValid   bool   `json:"is_valid"`
	IsActive  bool   `json:"is_active"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
	KYCLevel  string `json:"kyc_level"`
	TradingEnabled bool `json:"trading_enabled"`
}

// GetUserRiskProfile retrieves user risk profile
func (uc *UsersClient) GetUserRiskProfile(ctx context.Context, userID int64) (*RiskAssessment, error) {
	url := fmt.Sprintf("%s/users/%d/risk-profile", uc.baseURL, userID)

	var response struct {
		Data RiskAssessment `json:"data"`
	}

	err := uc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get user risk profile %d: %w", userID, err)
	}

	return &response.Data, nil
}

// GetUserPreferences retrieves user preferences
func (uc *UsersClient) GetUserPreferences(ctx context.Context, userID int64) (*UserPreferences, error) {
	url := fmt.Sprintf("%s/users/%d/preferences", uc.baseURL, userID)

	var response struct {
		Data UserPreferences `json:"data"`
	}

	err := uc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get user preferences %d: %w", userID, err)
	}

	return &response.Data, nil
}

// UpdateUserPreferences updates user preferences
func (uc *UsersClient) UpdateUserPreferences(ctx context.Context, userID int64, preferences *UserPreferences) (*UserPreferences, error) {
	url := fmt.Sprintf("%s/users/%d/preferences", uc.baseURL, userID)

	var response struct {
		Data UserPreferences `json:"data"`
	}

	err := uc.makeRequest(ctx, "PUT", url, preferences, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to update user preferences %d: %w", userID, err)
	}

	return &response.Data, nil
}

// GetUserNotifications retrieves user notifications
func (uc *UsersClient) GetUserNotifications(ctx context.Context, userID int64, unreadOnly bool, limit int) ([]Notification, error) {
	url := fmt.Sprintf("%s/users/%d/notifications?limit=%d", uc.baseURL, userID, limit)

	if unreadOnly {
		url += "&unread_only=true"
	}

	var response struct {
		Data []Notification `json:"data"`
	}

	err := uc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get user notifications %d: %w", userID, err)
	}

	return response.Data, nil
}

// Notification represents a user notification
type Notification struct {
	ID          string                 `json:"id"`
	UserID      int64                  `json:"user_id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data,omitempty"`
	IsRead      bool                   `json:"is_read"`
	Priority    string                 `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	ReadAt      *time.Time             `json:"read_at,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

// SendNotification sends a notification to a user
func (uc *UsersClient) SendNotification(ctx context.Context, notification *Notification) error {
	url := fmt.Sprintf("%s/users/%d/notifications", uc.baseURL, notification.UserID)

	err := uc.makeRequest(ctx, "POST", url, notification, nil)
	if err != nil {
		return fmt.Errorf("failed to send notification to user %d: %w", notification.UserID, err)
	}

	return nil
}

// MarkNotificationRead marks a notification as read
func (uc *UsersClient) MarkNotificationRead(ctx context.Context, userID int64, notificationID string) error {
	url := fmt.Sprintf("%s/users/%d/notifications/%s/read", uc.baseURL, userID, notificationID)

	err := uc.makeRequest(ctx, "POST", url, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to mark notification read: %w", err)
	}

	return nil
}

// makeRequest performs HTTP request with retry logic
func (uc *UsersClient) makeRequest(ctx context.Context, method, url string, body interface{}, response interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= uc.retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		var reqBody []byte
		if body != nil {
			var err error
			reqBody, err = json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to marshal request body: %w", err)
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		// Add headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Portfolio-API/1.0")
		if uc.apiKey != "" {
			req.Header.Set("X-API-Key", uc.apiKey)
		}

		resp, err := uc.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}
		defer resp.Body.Close()

		// Check for rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("HTTP %d: request failed", resp.StatusCode)
			continue
		}

		if response != nil {
			if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
				lastErr = fmt.Errorf("failed to decode response: %w", err)
				continue
			}
		}

		return nil
	}

	return fmt.Errorf("request failed after %d attempts: %w", uc.retries+1, lastErr)
}

// IsHealthy checks if the users service is healthy
func (uc *UsersClient) IsHealthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/health", uc.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetUserActivity retrieves user activity logs
func (uc *UsersClient) GetUserActivity(ctx context.Context, userID int64, activityType string, limit int) ([]ActivityLog, error) {
	url := fmt.Sprintf("%s/users/%d/activity?limit=%d", uc.baseURL, userID, limit)

	if activityType != "" {
		url += fmt.Sprintf("&type=%s", activityType)
	}

	var response struct {
		Data []ActivityLog `json:"data"`
	}

	err := uc.makeRequest(ctx, "GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get user activity %d: %w", userID, err)
	}

	return response.Data, nil
}

// ActivityLog represents user activity log entry
type ActivityLog struct {
	ID          string                 `json:"id"`
	UserID      int64                  `json:"user_id"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource"`
	Details     map[string]interface{} `json:"details"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	Timestamp   time.Time              `json:"timestamp"`
}