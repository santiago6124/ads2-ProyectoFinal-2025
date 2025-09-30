package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"wallet-api/internal/models"
	"wallet-api/internal/repository"
)

type ComplianceService interface {
	ValidateTransaction(ctx context.Context, req *ValidateTransactionRequest) (*ValidationResult, error)
	MonitorTransactionPatterns(ctx context.Context, userID int64, transaction *models.Transaction) (*PatternAnalysisResult, error)
	CheckVelocityLimits(ctx context.Context, userID int64, amount decimal.Decimal, transactionType string) (*VelocityCheckResult, error)
	DetectSuspiciousActivity(ctx context.Context, userID int64) (*SuspiciousActivityResult, error)
	GenerateRiskScore(ctx context.Context, userID int64, transaction *models.Transaction) (*RiskScoreResult, error)
	ProcessComplianceAlert(ctx context.Context, alert *ComplianceAlert) error
	GetComplianceStatus(ctx context.Context, userID int64) (*ComplianceStatusResult, error)
	UpdateRiskProfile(ctx context.Context, userID int64, updates *RiskProfileUpdate) error
	PerformKYCCheck(ctx context.Context, userID int64) (*KYCResult, error)
	MonitorWalletHealth(ctx context.Context, walletID primitive.ObjectID) (*WalletHealthResult, error)
}

type complianceService struct {
	transactionRepo repository.TransactionRepository
	walletRepo      repository.WalletRepository
	auditService    AuditService
	riskEngine      RiskEngine
}

func NewComplianceService(
	transactionRepo repository.TransactionRepository,
	walletRepo repository.WalletRepository,
	auditService AuditService,
	riskEngine RiskEngine,
) ComplianceService {
	return &complianceService{
		transactionRepo: transactionRepo,
		walletRepo:      walletRepo,
		auditService:    auditService,
		riskEngine:      riskEngine,
	}
}

// Risk Engine interface for complex risk calculations
type RiskEngine interface {
	CalculateTransactionRisk(ctx context.Context, transaction *models.Transaction, userProfile *UserRiskProfile) (*RiskAssessment, error)
	UpdateUserRiskProfile(ctx context.Context, userID int64, transaction *models.Transaction) error
	GetUserRiskProfile(ctx context.Context, userID int64) (*UserRiskProfile, error)
	DetectAnomalies(ctx context.Context, userID int64, transaction *models.Transaction) (*AnomalyDetectionResult, error)
}

// Data structures
type ValidateTransactionRequest struct {
	UserID      int64                 `json:"user_id"`
	Transaction *models.Transaction   `json:"transaction"`
	Context     *TransactionContext   `json:"context"`
}

type TransactionContext struct {
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	Location      *GeolocationInfo       `json:"location,omitempty"`
	DeviceInfo    *DeviceInfo           `json:"device_info,omitempty"`
	SessionInfo   *SessionInfo          `json:"session_info,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type GeolocationInfo struct {
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type DeviceInfo struct {
	DeviceID        string `json:"device_id"`
	DeviceType      string `json:"device_type"`
	OperatingSystem string `json:"operating_system"`
	Browser         string `json:"browser"`
	IsMobile        bool   `json:"is_mobile"`
}

type SessionInfo struct {
	SessionID       string    `json:"session_id"`
	SessionStart    time.Time `json:"session_start"`
	LastActivity    time.Time `json:"last_activity"`
	TransactionCount int       `json:"transaction_count"`
}

type ValidationResult struct {
	IsValid           bool                   `json:"is_valid"`
	RiskScore         int                    `json:"risk_score"`
	ComplianceFlags   []string               `json:"compliance_flags"`
	Violations        []ComplianceViolation  `json:"violations"`
	Recommendations   []string               `json:"recommendations"`
	RequiresApproval  bool                   `json:"requires_approval"`
	BlockTransaction  bool                   `json:"block_transaction"`
	AdditionalChecks  []string               `json:"additional_checks"`
}

type ComplianceViolation struct {
	ViolationType string                 `json:"violation_type"`
	Severity      string                 `json:"severity"`
	Description   string                 `json:"description"`
	RuleViolated  string                 `json:"rule_violated"`
	Evidence      map[string]interface{} `json:"evidence"`
}

type PatternAnalysisResult struct {
	PatternsDetected    []TransactionPattern `json:"patterns_detected"`
	AnomalyScore        int                  `json:"anomaly_score"`
	SuspiciousPatterns  []string             `json:"suspicious_patterns"`
	RecommendedActions  []string             `json:"recommended_actions"`
}

type TransactionPattern struct {
	PatternType   string                 `json:"pattern_type"`
	Description   string                 `json:"description"`
	Frequency     int                    `json:"frequency"`
	Confidence    float64                `json:"confidence"`
	RiskLevel     string                 `json:"risk_level"`
	Evidence      map[string]interface{} `json:"evidence"`
}

type VelocityCheckResult struct {
	WithinLimits        bool            `json:"within_limits"`
	CurrentVelocity     VelocityMetrics `json:"current_velocity"`
	LimitViolations     []LimitViolation `json:"limit_violations"`
	RecommendedWaitTime time.Duration   `json:"recommended_wait_time"`
}

type VelocityMetrics struct {
	TransactionsLast1Hour  int             `json:"transactions_last_1h"`
	TransactionsLast24Hour int             `json:"transactions_last_24h"`
	TransactionsLast7Days  int             `json:"transactions_last_7d"`
	VolumeLast1Hour        decimal.Decimal `json:"volume_last_1h"`
	VolumeLast24Hour       decimal.Decimal `json:"volume_last_24h"`
	VolumeLast7Days        decimal.Decimal `json:"volume_last_7d"`
}

type LimitViolation struct {
	LimitType   string          `json:"limit_type"`
	CurrentValue decimal.Decimal `json:"current_value"`
	LimitValue   decimal.Decimal `json:"limit_value"`
	Severity     string          `json:"severity"`
}

type SuspiciousActivityResult struct {
	IsSuspicious        bool                      `json:"is_suspicious"`
	SuspiciousActivities []SuspiciousActivity      `json:"suspicious_activities"`
	OverallRiskScore    int                       `json:"overall_risk_score"`
	RecommendedActions  []string                  `json:"recommended_actions"`
	RequiresInvestigation bool                    `json:"requires_investigation"`
}

type SuspiciousActivity struct {
	ActivityType  string                 `json:"activity_type"`
	Description   string                 `json:"description"`
	Severity      string                 `json:"severity"`
	Evidence      map[string]interface{} `json:"evidence"`
	DetectedAt    time.Time              `json:"detected_at"`
	ConfidenceScore float64              `json:"confidence_score"`
}

type RiskScoreResult struct {
	RiskScore       int                    `json:"risk_score"`
	RiskLevel       string                 `json:"risk_level"`
	RiskFactors     []RiskFactor           `json:"risk_factors"`
	Recommendations []string               `json:"recommendations"`
	Assessment      *RiskAssessment        `json:"assessment"`
}

type RiskFactor struct {
	FactorType    string  `json:"factor_type"`
	Description   string  `json:"description"`
	Weight        float64 `json:"weight"`
	Score         int     `json:"score"`
	Contribution  float64 `json:"contribution"`
}

type RiskAssessment struct {
	TransactionRisk  int                    `json:"transaction_risk"`
	UserRisk         int                    `json:"user_risk"`
	ContextualRisk   int                    `json:"contextual_risk"`
	HistoricalRisk   int                    `json:"historical_risk"`
	OverallRisk      int                    `json:"overall_risk"`
	RiskBreakdown    map[string]interface{} `json:"risk_breakdown"`
}

type ComplianceAlert struct {
	AlertID       string                 `json:"alert_id"`
	AlertType     string                 `json:"alert_type"`
	UserID        int64                  `json:"user_id"`
	TransactionID string                 `json:"transaction_id,omitempty"`
	Severity      string                 `json:"severity"`
	Description   string                 `json:"description"`
	Evidence      map[string]interface{} `json:"evidence"`
	CreatedAt     time.Time              `json:"created_at"`
	Status        string                 `json:"status"`
	AssignedTo    string                 `json:"assigned_to,omitempty"`
}

type ComplianceStatusResult struct {
	UserID            int64                `json:"user_id"`
	ComplianceLevel   string               `json:"compliance_level"`
	RiskRating        string               `json:"risk_rating"`
	KYCStatus         string               `json:"kyc_status"`
	ActiveAlerts      []ComplianceAlert    `json:"active_alerts"`
	RecentViolations  []ComplianceViolation `json:"recent_violations"`
	TransactionLimits *TransactionLimits   `json:"transaction_limits"`
	LastReviewed      time.Time            `json:"last_reviewed"`
}

type TransactionLimits struct {
	DailyLimit      decimal.Decimal `json:"daily_limit"`
	MonthlyLimit    decimal.Decimal `json:"monthly_limit"`
	SingleTxLimit   decimal.Decimal `json:"single_tx_limit"`
	VelocityLimit   int             `json:"velocity_limit"`
}

type RiskProfileUpdate struct {
	RiskLevel       string                 `json:"risk_level,omitempty"`
	KYCStatus       string                 `json:"kyc_status,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
	UpdatedBy       string                 `json:"updated_by"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type KYCResult struct {
	UserID          int64                  `json:"user_id"`
	KYCLevel        string                 `json:"kyc_level"`
	VerificationStatus string              `json:"verification_status"`
	DocumentsVerified []string             `json:"documents_verified"`
	RiskFactors     []string               `json:"risk_factors"`
	ComplianceScore int                    `json:"compliance_score"`
	NextReviewDate  time.Time              `json:"next_review_date"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type WalletHealthResult struct {
	WalletID          primitive.ObjectID     `json:"wallet_id"`
	HealthScore       int                    `json:"health_score"`
	HealthStatus      string                 `json:"health_status"`
	Issues            []WalletHealthIssue    `json:"issues"`
	Recommendations   []string               `json:"recommendations"`
	LastChecked       time.Time              `json:"last_checked"`
}

type WalletHealthIssue struct {
	IssueType     string                 `json:"issue_type"`
	Severity      string                 `json:"severity"`
	Description   string                 `json:"description"`
	Evidence      map[string]interface{} `json:"evidence"`
	DetectedAt    time.Time              `json:"detected_at"`
}

type UserRiskProfile struct {
	UserID              int64                  `json:"user_id"`
	RiskLevel           string                 `json:"risk_level"`
	RiskScore           int                    `json:"risk_score"`
	KYCLevel            string                 `json:"kyc_level"`
	AccountAge          int                    `json:"account_age_days"`
	TransactionHistory  *TransactionHistory    `json:"transaction_history"`
	BehaviorProfile     *BehaviorProfile       `json:"behavior_profile"`
	GeographicRisk      *GeographicRisk        `json:"geographic_risk"`
	LastUpdated         time.Time              `json:"last_updated"`
	Metadata            map[string]interface{} `json:"metadata"`
}

type TransactionHistory struct {
	TotalTransactions   int64           `json:"total_transactions"`
	TotalVolume         decimal.Decimal `json:"total_volume"`
	FailedTransactions  int64           `json:"failed_transactions"`
	ReversedTransactions int64          `json:"reversed_transactions"`
	LargestTransaction  decimal.Decimal `json:"largest_transaction"`
	AverageTransaction  decimal.Decimal `json:"average_transaction"`
}

type BehaviorProfile struct {
	PreferredHours       []int                  `json:"preferred_hours"`
	PreferredDays        []string               `json:"preferred_days"`
	AverageSessionLength time.Duration          `json:"average_session_length"`
	DeviceFingerprints   []string               `json:"device_fingerprints"`
	IPAddressHistory     []string               `json:"ip_address_history"`
	AnomalyCount         int                    `json:"anomaly_count"`
	Metadata             map[string]interface{} `json:"metadata"`
}

type GeographicRisk struct {
	HomeCountry       string   `json:"home_country"`
	VisitedCountries  []string `json:"visited_countries"`
	HighRiskCountries []string `json:"high_risk_countries"`
	CurrentLocation   *GeolocationInfo `json:"current_location"`
}

type AnomalyDetectionResult struct {
	AnomaliesDetected []Anomaly `json:"anomalies_detected"`
	AnomalyScore      int       `json:"anomaly_score"`
	IsAnomalous       bool      `json:"is_anomalous"`
}

type Anomaly struct {
	AnomalyType   string                 `json:"anomaly_type"`
	Description   string                 `json:"description"`
	Severity      string                 `json:"severity"`
	Confidence    float64                `json:"confidence"`
	Evidence      map[string]interface{} `json:"evidence"`
}

// Implementation
func (s *complianceService) ValidateTransaction(ctx context.Context, req *ValidateTransactionRequest) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid: true,
		ComplianceFlags: make([]string, 0),
		Violations: make([]ComplianceViolation, 0),
		Recommendations: make([]string, 0),
	}

	// Get user's risk profile
	userProfile, err := s.riskEngine.GetUserRiskProfile(ctx, req.UserID)
	if err != nil {
		userProfile = &UserRiskProfile{
			UserID: req.UserID,
			RiskLevel: "medium",
			RiskScore: 5,
		}
	}

	// Calculate transaction risk
	riskAssessment, err := s.riskEngine.CalculateTransactionRisk(ctx, req.Transaction, userProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate transaction risk: %w", err)
	}

	result.RiskScore = riskAssessment.OverallRisk

	// Check transaction amount limits
	if err := s.checkAmountLimits(req.Transaction, result); err != nil {
		return nil, err
	}

	// Check velocity limits
	velocityResult, err := s.CheckVelocityLimits(ctx, req.UserID, req.Transaction.Amount.Value.Abs(), req.Transaction.Type)
	if err != nil {
		return nil, err
	}

	if !velocityResult.WithinLimits {
		result.ComplianceFlags = append(result.ComplianceFlags, "VELOCITY_VIOLATION")
		for _, violation := range velocityResult.LimitViolations {
			result.Violations = append(result.Violations, ComplianceViolation{
				ViolationType: "velocity_limit",
				Severity:      violation.Severity,
				Description:   fmt.Sprintf("Velocity limit exceeded: %s", violation.LimitType),
				RuleViolated:  violation.LimitType,
				Evidence: map[string]interface{}{
					"current_value": violation.CurrentValue.String(),
					"limit_value":   violation.LimitValue.String(),
				},
			})
		}
	}

	// Check for suspicious patterns
	patternResult, err := s.MonitorTransactionPatterns(ctx, req.UserID, req.Transaction)
	if err != nil {
		return nil, err
	}

	if len(patternResult.SuspiciousPatterns) > 0 {
		result.ComplianceFlags = append(result.ComplianceFlags, "SUSPICIOUS_PATTERN")
		result.Recommendations = append(result.Recommendations, "Additional monitoring required")
	}

	// Determine if approval is required
	result.RequiresApproval = s.requiresApproval(result.RiskScore, result.Violations)
	result.BlockTransaction = s.shouldBlockTransaction(result.RiskScore, result.Violations)

	// If transaction should be blocked, mark as invalid
	if result.BlockTransaction {
		result.IsValid = false
	}

	return result, nil
}

func (s *complianceService) MonitorTransactionPatterns(ctx context.Context, userID int64, transaction *models.Transaction) (*PatternAnalysisResult, error) {
	result := &PatternAnalysisResult{
		PatternsDetected: make([]TransactionPattern, 0),
		SuspiciousPatterns: make([]string, 0),
		RecommendedActions: make([]string, 0),
	}

	// Get recent transactions for pattern analysis
	transactions, err := s.transactionRepo.GetByUserID(ctx, userID, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get user transactions: %w", err)
	}

	// Analyze patterns
	s.analyzeAmountPatterns(transactions, transaction, result)
	s.analyzeTimePatterns(transactions, transaction, result)
	s.analyzeFrequencyPatterns(transactions, transaction, result)

	// Calculate anomaly score
	result.AnomalyScore = s.calculateAnomalyScore(result.PatternsDetected)

	return result, nil
}

func (s *complianceService) CheckVelocityLimits(ctx context.Context, userID int64, amount decimal.Decimal, transactionType string) (*VelocityCheckResult, error) {
	result := &VelocityCheckResult{
		WithinLimits: true,
		LimitViolations: make([]LimitViolation, 0),
	}

	// Get recent transactions for velocity calculation
	now := time.Now()
	last1Hour := now.Add(-1 * time.Hour)
	last24Hours := now.Add(-24 * time.Hour)
	last7Days := now.Add(-7 * 24 * time.Hour)

	// Get wallet first
	wallet, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	// Get transactions for different time periods
	transactions1h, _ := s.transactionRepo.GetTransactionsByDateRange(ctx, wallet.ID, last1Hour, now)
	transactions24h, _ := s.transactionRepo.GetTransactionsByDateRange(ctx, wallet.ID, last24Hours, now)
	transactions7d, _ := s.transactionRepo.GetTransactionsByDateRange(ctx, wallet.ID, last7Days, now)

	// Calculate current velocity
	result.CurrentVelocity = s.calculateVelocityMetrics(transactions1h, transactions24h, transactions7d)

	// Check against limits
	s.checkHourlyLimits(result, amount)
	s.checkDailyLimits(result, amount)
	s.checkWeeklyLimits(result, amount)

	if len(result.LimitViolations) > 0 {
		result.WithinLimits = false
	}

	return result, nil
}

func (s *complianceService) DetectSuspiciousActivity(ctx context.Context, userID int64) (*SuspiciousActivityResult, error) {
	result := &SuspiciousActivityResult{
		SuspiciousActivities: make([]SuspiciousActivity, 0),
		RecommendedActions: make([]string, 0),
	}

	// Get recent transactions
	transactions, err := s.transactionRepo.GetByUserID(ctx, userID, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Check for various suspicious activities
	s.checkForStructuring(transactions, result)
	s.checkForUnusualAmounts(transactions, result)
	s.checkForRapidTransactions(transactions, result)
	s.checkForFailedAttempts(transactions, result)

	// Calculate overall risk score
	result.OverallRiskScore = s.calculateOverallRiskScore(result.SuspiciousActivities)
	result.IsSuspicious = result.OverallRiskScore >= 7

	if result.IsSuspicious {
		result.RequiresInvestigation = true
		result.RecommendedActions = append(result.RecommendedActions, "Manual review required", "Consider transaction monitoring")
	}

	return result, nil
}

func (s *complianceService) GenerateRiskScore(ctx context.Context, userID int64, transaction *models.Transaction) (*RiskScoreResult, error) {
	result := &RiskScoreResult{
		RiskFactors: make([]RiskFactor, 0),
		Recommendations: make([]string, 0),
	}

	// Get user risk profile
	userProfile, err := s.riskEngine.GetUserRiskProfile(ctx, userID)
	if err != nil {
		userProfile = &UserRiskProfile{UserID: userID, RiskLevel: "medium", RiskScore: 5}
	}

	// Calculate risk assessment
	assessment, err := s.riskEngine.CalculateTransactionRisk(ctx, transaction, userProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate risk: %w", err)
	}

	result.Assessment = assessment
	result.RiskScore = assessment.OverallRisk
	result.RiskLevel = s.mapRiskScoreToLevel(result.RiskScore)

	// Generate risk factors
	result.RiskFactors = s.generateRiskFactors(transaction, userProfile, assessment)

	// Generate recommendations
	result.Recommendations = s.generateRiskRecommendations(result.RiskScore, result.RiskFactors)

	return result, nil
}

func (s *complianceService) ProcessComplianceAlert(ctx context.Context, alert *ComplianceAlert) error {
	// Log the alert
	err := s.auditService.LogSystemEvent(ctx, "compliance_alert", map[string]interface{}{
		"alert_id":       alert.AlertID,
		"alert_type":     alert.AlertType,
		"user_id":        alert.UserID,
		"transaction_id": alert.TransactionID,
		"severity":       alert.Severity,
		"description":    alert.Description,
		"evidence":       alert.Evidence,
	})

	if err != nil {
		return fmt.Errorf("failed to log compliance alert: %w", err)
	}

	// Handle alert based on severity
	switch alert.Severity {
	case "critical":
		return s.handleCriticalAlert(ctx, alert)
	case "high":
		return s.handleHighAlert(ctx, alert)
	default:
		return s.handleStandardAlert(ctx, alert)
	}
}

func (s *complianceService) GetComplianceStatus(ctx context.Context, userID int64) (*ComplianceStatusResult, error) {
	result := &ComplianceStatusResult{
		UserID: userID,
		ActiveAlerts: make([]ComplianceAlert, 0),
		RecentViolations: make([]ComplianceViolation, 0),
	}

	// Get user risk profile
	userProfile, err := s.riskEngine.GetUserRiskProfile(ctx, userID)
	if err == nil {
		result.ComplianceLevel = s.mapRiskLevelToCompliance(userProfile.RiskLevel)
		result.RiskRating = userProfile.RiskLevel
		result.KYCStatus = userProfile.KYCLevel
	}

	// Get wallet for transaction limits
	wallet, err := s.walletRepo.GetByUserID(ctx, userID)
	if err == nil {
		result.TransactionLimits = &TransactionLimits{
			DailyLimit:    wallet.Limits.DailyWithdrawal,
			MonthlyLimit:  wallet.Limits.MonthlyVolume,
			SingleTxLimit: wallet.Limits.SingleTransaction,
			VelocityLimit: 10, // Default velocity limit
		}
	}

	result.LastReviewed = time.Now()

	return result, nil
}

func (s *complianceService) UpdateRiskProfile(ctx context.Context, userID int64, updates *RiskProfileUpdate) error {
	// This would update the user's risk profile in the risk engine
	return s.riskEngine.UpdateUserRiskProfile(ctx, userID, nil)
}

func (s *complianceService) PerformKYCCheck(ctx context.Context, userID int64) (*KYCResult, error) {
	result := &KYCResult{
		UserID: userID,
		DocumentsVerified: make([]string, 0),
		RiskFactors: make([]string, 0),
		Metadata: make(map[string]interface{}),
	}

	// Get user profile
	userProfile, err := s.riskEngine.GetUserRiskProfile(ctx, userID)
	if err == nil {
		result.KYCLevel = userProfile.KYCLevel
		result.ComplianceScore = 10 - userProfile.RiskScore // Inverse relationship
	} else {
		result.KYCLevel = "basic"
		result.ComplianceScore = 5
	}

	result.VerificationStatus = "verified"
	result.NextReviewDate = time.Now().AddDate(1, 0, 0) // Annual review

	return result, nil
}

func (s *complianceService) MonitorWalletHealth(ctx context.Context, walletID primitive.ObjectID) (*WalletHealthResult, error) {
	result := &WalletHealthResult{
		WalletID: walletID,
		Issues: make([]WalletHealthIssue, 0),
		Recommendations: make([]string, 0),
		LastChecked: time.Now(),
	}

	// Get wallet
	wallet, err := s.walletRepo.GetByID(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	// Check wallet health indicators
	s.checkBalanceConsistency(wallet, result)
	s.checkLockHealth(wallet, result)
	s.checkTransactionHistory(ctx, wallet, result)

	// Calculate health score
	result.HealthScore = s.calculateWalletHealthScore(result.Issues)
	result.HealthStatus = s.mapHealthScoreToStatus(result.HealthScore)

	return result, nil
}

// Helper methods
func (s *complianceService) checkAmountLimits(transaction *models.Transaction, result *ValidationResult) error {
	// CTR threshold check
	ctrThreshold, _ := decimal.NewFromString("10000")
	if transaction.Amount.Value.Abs().GreaterThan(ctrThreshold) {
		result.ComplianceFlags = append(result.ComplianceFlags, "CTR_THRESHOLD")
		result.Recommendations = append(result.Recommendations, "CTR reporting required")
	}

	// Large transaction check
	largeThreshold, _ := decimal.NewFromString("5000")
	if transaction.Amount.Value.Abs().GreaterThan(largeThreshold) {
		result.ComplianceFlags = append(result.ComplianceFlags, "LARGE_TRANSACTION")
	}

	return nil
}

func (s *complianceService) requiresApproval(riskScore int, violations []ComplianceViolation) bool {
	if riskScore >= 8 {
		return true
	}

	for _, violation := range violations {
		if violation.Severity == "high" || violation.Severity == "critical" {
			return true
		}
	}

	return false
}

func (s *complianceService) shouldBlockTransaction(riskScore int, violations []ComplianceViolation) bool {
	if riskScore >= 9 {
		return true
	}

	for _, violation := range violations {
		if violation.Severity == "critical" {
			return true
		}
	}

	return false
}

func (s *complianceService) analyzeAmountPatterns(transactions []*models.Transaction, current *models.Transaction, result *PatternAnalysisResult) {
	// Analyze for round number patterns
	amount := current.Amount.Value.Abs()
	if s.isRoundNumber(amount) {
		pattern := TransactionPattern{
			PatternType: "round_numbers",
			Description: "Transaction uses round numbers",
			Frequency:   1,
			Confidence:  0.7,
			RiskLevel:   "medium",
		}
		result.PatternsDetected = append(result.PatternsDetected, pattern)
	}

	// Check for structuring patterns (amounts just under reporting thresholds)
	threshold, _ := decimal.NewFromString("9999")
	if amount.GreaterThan(threshold) && amount.LessThan(decimal.NewFromString("10000")) {
		result.SuspiciousPatterns = append(result.SuspiciousPatterns, "potential_structuring")
	}
}

func (s *complianceService) analyzeTimePatterns(transactions []*models.Transaction, current *models.Transaction, result *PatternAnalysisResult) {
	// Analyze transaction timing patterns
	hour := current.CreatedAt.Hour()
	if hour < 6 || hour > 22 {
		pattern := TransactionPattern{
			PatternType: "unusual_hours",
			Description: "Transaction outside normal business hours",
			Frequency:   1,
			Confidence:  0.6,
			RiskLevel:   "low",
		}
		result.PatternsDetected = append(result.PatternsDetected, pattern)
	}
}

func (s *complianceService) analyzeFrequencyPatterns(transactions []*models.Transaction, current *models.Transaction, result *PatternAnalysisResult) {
	// Analyze transaction frequency
	recentCount := 0
	cutoff := time.Now().Add(-1 * time.Hour)

	for _, tx := range transactions {
		if tx.CreatedAt.After(cutoff) {
			recentCount++
		}
	}

	if recentCount > 10 {
		result.SuspiciousPatterns = append(result.SuspiciousPatterns, "high_frequency")
	}
}

func (s *complianceService) calculateAnomalyScore(patterns []TransactionPattern) int {
	score := 0
	for _, pattern := range patterns {
		switch pattern.RiskLevel {
		case "high":
			score += 3
		case "medium":
			score += 2
		case "low":
			score += 1
		}
	}
	return score
}

func (s *complianceService) calculateVelocityMetrics(tx1h, tx24h, tx7d []*models.Transaction) VelocityMetrics {
	metrics := VelocityMetrics{}

	metrics.TransactionsLast1Hour = len(tx1h)
	metrics.TransactionsLast24Hour = len(tx24h)
	metrics.TransactionsLast7Days = len(tx7d)

	// Calculate volumes
	for _, tx := range tx1h {
		if tx.Status == "completed" {
			metrics.VolumeLast1Hour = metrics.VolumeLast1Hour.Add(tx.Amount.Value.Abs())
		}
	}
	for _, tx := range tx24h {
		if tx.Status == "completed" {
			metrics.VolumeLast24Hour = metrics.VolumeLast24Hour.Add(tx.Amount.Value.Abs())
		}
	}
	for _, tx := range tx7d {
		if tx.Status == "completed" {
			metrics.VolumeLast7Days = metrics.VolumeLast7Days.Add(tx.Amount.Value.Abs())
		}
	}

	return metrics
}

func (s *complianceService) checkHourlyLimits(result *VelocityCheckResult, amount decimal.Decimal) {
	hourlyLimit, _ := decimal.NewFromString("1000")
	if result.CurrentVelocity.VolumeLast1Hour.Add(amount).GreaterThan(hourlyLimit) {
		violation := LimitViolation{
			LimitType:    "hourly_volume",
			CurrentValue: result.CurrentVelocity.VolumeLast1Hour.Add(amount),
			LimitValue:   hourlyLimit,
			Severity:     "medium",
		}
		result.LimitViolations = append(result.LimitViolations, violation)
	}
}

func (s *complianceService) checkDailyLimits(result *VelocityCheckResult, amount decimal.Decimal) {
	dailyLimit, _ := decimal.NewFromString("10000")
	if result.CurrentVelocity.VolumeLast24Hour.Add(amount).GreaterThan(dailyLimit) {
		violation := LimitViolation{
			LimitType:    "daily_volume",
			CurrentValue: result.CurrentVelocity.VolumeLast24Hour.Add(amount),
			LimitValue:   dailyLimit,
			Severity:     "high",
		}
		result.LimitViolations = append(result.LimitViolations, violation)
	}
}

func (s *complianceService) checkWeeklyLimits(result *VelocityCheckResult, amount decimal.Decimal) {
	weeklyLimit, _ := decimal.NewFromString("50000")
	if result.CurrentVelocity.VolumeLast7Days.Add(amount).GreaterThan(weeklyLimit) {
		violation := LimitViolation{
			LimitType:    "weekly_volume",
			CurrentValue: result.CurrentVelocity.VolumeLast7Days.Add(amount),
			LimitValue:   weeklyLimit,
			Severity:     "critical",
		}
		result.LimitViolations = append(result.LimitViolations, violation)
	}
}

func (s *complianceService) checkForStructuring(transactions []*models.Transaction, result *SuspiciousActivityResult) {
	// Look for patterns of transactions just under reporting thresholds
	threshold, _ := decimal.NewFromString("10000")
	structuringCount := 0

	for _, tx := range transactions {
		amount := tx.Amount.Value.Abs()
		if amount.GreaterThan(decimal.NewFromString("9000")) && amount.LessThan(threshold) {
			structuringCount++
		}
	}

	if structuringCount >= 3 {
		activity := SuspiciousActivity{
			ActivityType:    "structuring",
			Description:     "Multiple transactions just below reporting threshold",
			Severity:        "high",
			DetectedAt:      time.Now(),
			ConfidenceScore: 0.8,
			Evidence: map[string]interface{}{
				"transaction_count": structuringCount,
				"threshold":         threshold.String(),
			},
		}
		result.SuspiciousActivities = append(result.SuspiciousActivities, activity)
	}
}

func (s *complianceService) checkForUnusualAmounts(transactions []*models.Transaction, result *SuspiciousActivityResult) {
	// Check for unusually large amounts compared to user's history
	if len(transactions) < 5 {
		return
	}

	var totalAmount decimal.Decimal
	for _, tx := range transactions[1:] {
		totalAmount = totalAmount.Add(tx.Amount.Value.Abs())
	}

	avgAmount := totalAmount.Div(decimal.NewFromInt(int64(len(transactions) - 1)))
	latestAmount := transactions[0].Amount.Value.Abs()

	// If latest transaction is 10x the average, flag as suspicious
	if latestAmount.GreaterThan(avgAmount.Mul(decimal.NewFromInt(10))) {
		activity := SuspiciousActivity{
			ActivityType:    "unusual_amount",
			Description:     "Transaction amount significantly higher than historical average",
			Severity:        "medium",
			DetectedAt:      time.Now(),
			ConfidenceScore: 0.7,
			Evidence: map[string]interface{}{
				"transaction_amount": latestAmount.String(),
				"average_amount":     avgAmount.String(),
				"multiplier":         latestAmount.Div(avgAmount).String(),
			},
		}
		result.SuspiciousActivities = append(result.SuspiciousActivities, activity)
	}
}

func (s *complianceService) checkForRapidTransactions(transactions []*models.Transaction, result *SuspiciousActivityResult) {
	// Check for multiple transactions in short time period
	if len(transactions) < 5 {
		return
	}

	rapidCount := 0
	cutoff := time.Now().Add(-10 * time.Minute)

	for _, tx := range transactions {
		if tx.CreatedAt.After(cutoff) {
			rapidCount++
		}
	}

	if rapidCount >= 5 {
		activity := SuspiciousActivity{
			ActivityType:    "rapid_transactions",
			Description:     "Multiple transactions in short time period",
			Severity:        "medium",
			DetectedAt:      time.Now(),
			ConfidenceScore: 0.6,
			Evidence: map[string]interface{}{
				"transaction_count": rapidCount,
				"time_window":       "10 minutes",
			},
		}
		result.SuspiciousActivities = append(result.SuspiciousActivities, activity)
	}
}

func (s *complianceService) checkForFailedAttempts(transactions []*models.Transaction, result *SuspiciousActivityResult) {
	// Check for multiple failed transaction attempts
	failedCount := 0
	for _, tx := range transactions {
		if tx.Status == "failed" {
			failedCount++
		}
	}

	if failedCount >= 3 {
		activity := SuspiciousActivity{
			ActivityType:    "multiple_failures",
			Description:     "Multiple failed transaction attempts",
			Severity:        "low",
			DetectedAt:      time.Now(),
			ConfidenceScore: 0.5,
			Evidence: map[string]interface{}{
				"failed_count": failedCount,
			},
		}
		result.SuspiciousActivities = append(result.SuspiciousActivities, activity)
	}
}

func (s *complianceService) calculateOverallRiskScore(activities []SuspiciousActivity) int {
	score := 0
	for _, activity := range activities {
		switch activity.Severity {
		case "critical":
			score += 4
		case "high":
			score += 3
		case "medium":
			score += 2
		case "low":
			score += 1
		}
	}
	return score
}

func (s *complianceService) handleCriticalAlert(ctx context.Context, alert *ComplianceAlert) error {
	// Immediate action for critical alerts
	return nil
}

func (s *complianceService) handleHighAlert(ctx context.Context, alert *ComplianceAlert) error {
	// Escalated handling for high severity alerts
	return nil
}

func (s *complianceService) handleStandardAlert(ctx context.Context, alert *ComplianceAlert) error {
	// Standard processing for normal alerts
	return nil
}

func (s *complianceService) mapRiskScoreToLevel(score int) string {
	if score >= 8 {
		return "high"
	} else if score >= 5 {
		return "medium"
	}
	return "low"
}

func (s *complianceService) mapRiskLevelToCompliance(riskLevel string) string {
	switch riskLevel {
	case "low":
		return "excellent"
	case "medium":
		return "good"
	case "high":
		return "poor"
	default:
		return "fair"
	}
}

func (s *complianceService) generateRiskFactors(transaction *models.Transaction, userProfile *UserRiskProfile, assessment *RiskAssessment) []RiskFactor {
	factors := make([]RiskFactor, 0)

	// Amount-based risk factor
	amount := transaction.Amount.Value.Abs()
	if amount.GreaterThan(decimal.NewFromString("1000")) {
		factor := RiskFactor{
			FactorType:   "transaction_amount",
			Description:  "Large transaction amount",
			Weight:       0.3,
			Score:        8,
			Contribution: 2.4,
		}
		factors = append(factors, factor)
	}

	// User risk factor
	if userProfile.RiskLevel == "high" {
		factor := RiskFactor{
			FactorType:   "user_risk",
			Description:  "High-risk user profile",
			Weight:       0.4,
			Score:        userProfile.RiskScore,
			Contribution: 0.4 * float64(userProfile.RiskScore),
		}
		factors = append(factors, factor)
	}

	return factors
}

func (s *complianceService) generateRiskRecommendations(riskScore int, factors []RiskFactor) []string {
	recommendations := make([]string, 0)

	if riskScore >= 8 {
		recommendations = append(recommendations, "Manual review required")
		recommendations = append(recommendations, "Enhanced monitoring")
	}

	if riskScore >= 6 {
		recommendations = append(recommendations, "Additional verification")
	}

	return recommendations
}

func (s *complianceService) checkBalanceConsistency(wallet *models.Wallet, result *WalletHealthResult) {
	// Check if balance calculations are consistent
	calculatedTotal := wallet.Balance.Available.Add(wallet.Balance.Locked)
	if !calculatedTotal.Equal(wallet.Balance.Total) {
		issue := WalletHealthIssue{
			IssueType:   "balance_inconsistency",
			Severity:    "high",
			Description: "Wallet balance totals do not match",
			DetectedAt:  time.Now(),
			Evidence: map[string]interface{}{
				"stored_total":     wallet.Balance.Total.String(),
				"calculated_total": calculatedTotal.String(),
				"discrepancy":      calculatedTotal.Sub(wallet.Balance.Total).String(),
			},
		}
		result.Issues = append(result.Issues, issue)
	}
}

func (s *complianceService) checkLockHealth(wallet *models.Wallet, result *WalletHealthResult) {
	// Check for expired locks that should have been cleaned up
	now := time.Now()
	expiredLocks := 0

	for _, lock := range wallet.Locks {
		if lock.Status == "active" && now.After(lock.ExpiresAt) {
			expiredLocks++
		}
	}

	if expiredLocks > 0 {
		issue := WalletHealthIssue{
			IssueType:   "expired_locks",
			Severity:    "medium",
			Description: "Wallet has expired locks that need cleanup",
			DetectedAt:  time.Now(),
			Evidence: map[string]interface{}{
				"expired_lock_count": expiredLocks,
			},
		}
		result.Issues = append(result.Issues, issue)
	}
}

func (s *complianceService) checkTransactionHistory(ctx context.Context, wallet *models.Wallet, result *WalletHealthResult) {
	// This would check recent transaction patterns for health indicators
	// Implementation would analyze transaction success rates, patterns, etc.
}

func (s *complianceService) calculateWalletHealthScore(issues []WalletHealthIssue) int {
	score := 10 // Start with perfect score

	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			score -= 4
		case "high":
			score -= 3
		case "medium":
			score -= 2
		case "low":
			score -= 1
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (s *complianceService) mapHealthScoreToStatus(score int) string {
	if score >= 9 {
		return "excellent"
	} else if score >= 7 {
		return "good"
	} else if score >= 5 {
		return "fair"
	} else if score >= 3 {
		return "poor"
	}
	return "critical"
}

func (s *complianceService) isRoundNumber(amount decimal.Decimal) bool {
	// Check if amount is a round number (divisible by 100, 500, 1000, etc.)
	mod100 := amount.Mod(decimal.NewFromInt(100))
	mod500 := amount.Mod(decimal.NewFromInt(500))
	mod1000 := amount.Mod(decimal.NewFromInt(1000))

	return mod1000.IsZero() || mod500.IsZero() || mod100.IsZero()
}