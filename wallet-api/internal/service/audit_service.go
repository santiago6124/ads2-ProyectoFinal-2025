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

type AuditService interface {
	LogTransaction(ctx context.Context, transaction *models.Transaction, action string) error
	LogWalletAction(ctx context.Context, walletID primitive.ObjectID, userID int64, action string, details map[string]interface{}) error
	LogAdminAction(ctx context.Context, adminID string, action string, targetUserID int64, details map[string]interface{}) error
	LogSystemEvent(ctx context.Context, eventType string, details map[string]interface{}) error
	GetAuditTrail(ctx context.Context, req *GetAuditTrailRequest) (*GetAuditTrailResponse, error)
	GetComplianceReport(ctx context.Context, req *GetComplianceReportRequest) (*GetComplianceReportResponse, error)
	GenerateRegulatoryReport(ctx context.Context, req *GenerateRegulatoryReportRequest) (*GenerateRegulatoryReportResponse, error)
	TrackSuspiciousActivity(ctx context.Context, req *TrackSuspiciousActivityRequest) error
	ExportAuditData(ctx context.Context, req *ExportAuditDataRequest) (*ExportAuditDataResponse, error)
}

type auditService struct {
	auditRepo       AuditRepository
	transactionRepo repository.TransactionRepository
	walletRepo      repository.WalletRepository
}

func NewAuditService(auditRepo AuditRepository, transactionRepo repository.TransactionRepository, walletRepo repository.WalletRepository) AuditService {
	return &auditService{
		auditRepo:       auditRepo,
		transactionRepo: transactionRepo,
		walletRepo:      walletRepo,
	}
}

// AuditRepository interface for audit data persistence
type AuditRepository interface {
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	GetAuditLogs(ctx context.Context, filter *AuditFilter, limit, offset int) ([]*AuditLog, error)
	GetAuditLogsByUserID(ctx context.Context, userID int64, startDate, endDate time.Time) ([]*AuditLog, error)
	GetAuditLogsByType(ctx context.Context, logType string, startDate, endDate time.Time) ([]*AuditLog, error)
	CreateComplianceReport(ctx context.Context, report *ComplianceReport) error
	GetComplianceReport(ctx context.Context, reportID string) (*ComplianceReport, error)
}

type AuditLog struct {
	ID              primitive.ObjectID     `bson:"_id,omitempty" json:"id,omitempty"`
	LogType         string                 `bson:"log_type" json:"log_type"`
	Action          string                 `bson:"action" json:"action"`
	UserID          int64                  `bson:"user_id,omitempty" json:"user_id,omitempty"`
	AdminID         string                 `bson:"admin_id,omitempty" json:"admin_id,omitempty"`
	WalletID        primitive.ObjectID     `bson:"wallet_id,omitempty" json:"wallet_id,omitempty"`
	TransactionID   string                 `bson:"transaction_id,omitempty" json:"transaction_id,omitempty"`
	IPAddress       string                 `bson:"ip_address" json:"ip_address"`
	UserAgent       string                 `bson:"user_agent" json:"user_agent"`
	SessionID       string                 `bson:"session_id" json:"session_id"`
	Details         map[string]interface{} `bson:"details" json:"details"`
	Metadata        map[string]interface{} `bson:"metadata" json:"metadata"`
	Timestamp       time.Time              `bson:"timestamp" json:"timestamp"`
	Severity        string                 `bson:"severity" json:"severity"`
	ComplianceFlags []string               `bson:"compliance_flags" json:"compliance_flags"`
	RiskScore       int                    `bson:"risk_score" json:"risk_score"`
}

type AuditFilter struct {
	UserID        int64     `json:"user_id,omitempty"`
	LogType       string    `json:"log_type,omitempty"`
	Action        string    `json:"action,omitempty"`
	StartDate     time.Time `json:"start_date,omitempty"`
	EndDate       time.Time `json:"end_date,omitempty"`
	Severity      string    `json:"severity,omitempty"`
	IPAddress     string    `json:"ip_address,omitempty"`
	MinRiskScore  int       `json:"min_risk_score,omitempty"`
	MaxRiskScore  int       `json:"max_risk_score,omitempty"`
}

type ComplianceReport struct {
	ID                primitive.ObjectID     `bson:"_id,omitempty" json:"id,omitempty"`
	ReportType        string                 `bson:"report_type" json:"report_type"`
	Period            ReportPeriod           `bson:"period" json:"period"`
	GeneratedAt       time.Time              `bson:"generated_at" json:"generated_at"`
	GeneratedBy       string                 `bson:"generated_by" json:"generated_by"`
	Summary           ComplianceSummary      `bson:"summary" json:"summary"`
	Findings          []ComplianceFinding    `bson:"findings" json:"findings"`
	Recommendations   []string               `bson:"recommendations" json:"recommendations"`
	RegulatoryMetrics RegulatoryMetrics      `bson:"regulatory_metrics" json:"regulatory_metrics"`
	Status            string                 `bson:"status" json:"status"`
	Metadata          map[string]interface{} `bson:"metadata" json:"metadata"`
}

type ComplianceSummary struct {
	TotalTransactions        int64           `json:"total_transactions"`
	TotalTransactionVolume   decimal.Decimal `json:"total_transaction_volume"`
	SuspiciousTransactions   int64           `json:"suspicious_transactions"`
	ComplianceViolations     int64           `json:"compliance_violations"`
	RiskScore                int             `json:"risk_score"`
	ComplianceScore          int             `json:"compliance_score"`
}

type ComplianceFinding struct {
	FindingID   string                 `json:"finding_id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	UserID      int64                  `json:"user_id,omitempty"`
	Evidence    map[string]interface{} `json:"evidence"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
}

type RegulatoryMetrics struct {
	LargeCashTransactions     int64           `json:"large_cash_transactions"`
	InternationalTransfers    int64           `json:"international_transfers"`
	HighRiskCountryTransfers  int64           `json:"high_risk_country_transfers"`
	CTRThresholdExceeded      int64           `json:"ctr_threshold_exceeded"`
	SARFilingRequired         int64           `json:"sar_filing_required"`
	AverageTransactionAmount  decimal.Decimal `json:"average_transaction_amount"`
	VelocityAlerts            int64           `json:"velocity_alerts"`
	PatternAlerts             int64           `json:"pattern_alerts"`
}

// Request/Response types
type GetAuditTrailRequest struct {
	Filter AuditFilter `json:"filter"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

type GetAuditTrailResponse struct {
	AuditLogs    []*AuditLog `json:"audit_logs"`
	Total        int64       `json:"total"`
	Success      bool        `json:"success"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

type GetComplianceReportRequest struct {
	ReportType string       `json:"report_type"`
	Period     ReportPeriod `json:"period"`
	UserID     int64        `json:"user_id,omitempty"`
}

type GetComplianceReportResponse struct {
	Report       *ComplianceReport `json:"report"`
	Success      bool              `json:"success"`
	ErrorMessage string            `json:"error_message,omitempty"`
}

type GenerateRegulatoryReportRequest struct {
	ReportType  string       `json:"report_type"`
	Period      ReportPeriod `json:"period"`
	GeneratedBy string       `json:"generated_by"`
}

type GenerateRegulatoryReportResponse struct {
	Report       *ComplianceReport `json:"report"`
	Success      bool              `json:"success"`
	ErrorMessage string            `json:"error_message,omitempty"`
}

type TrackSuspiciousActivityRequest struct {
	UserID      int64                  `json:"user_id"`
	ActivityType string                `json:"activity_type"`
	Description string                 `json:"description"`
	Evidence    map[string]interface{} `json:"evidence"`
	RiskScore   int                    `json:"risk_score"`
	ReportedBy  string                 `json:"reported_by"`
}

type ExportAuditDataRequest struct {
	Filter AuditFilter `json:"filter"`
	Format string      `json:"format"` // "csv", "json", "xml"
}

type ExportAuditDataResponse struct {
	Data         []byte `json:"data"`
	FileName     string `json:"file_name"`
	ContentType  string `json:"content_type"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func (s *auditService) LogTransaction(ctx context.Context, transaction *models.Transaction, action string) error {
	auditLog := &AuditLog{
		LogType:       "transaction",
		Action:        action,
		UserID:        transaction.UserID,
		WalletID:      transaction.WalletID,
		TransactionID: transaction.TransactionID,
		IPAddress:     transaction.Audit.IPAddress,
		UserAgent:     transaction.Audit.UserAgent,
		SessionID:     transaction.Audit.SessionID,
		Details: map[string]interface{}{
			"transaction_type": transaction.Type,
			"amount":          transaction.Amount.Value.String(),
			"currency":        transaction.Amount.Currency,
			"status":          transaction.Status,
			"fee":            transaction.Amount.Fee.String(),
			"net_amount":     transaction.Amount.Net.String(),
		},
		Metadata: map[string]interface{}{
			"reference_type": transaction.Reference.Type,
			"reference_id":   transaction.Reference.ID,
			"api_version":    transaction.Audit.APIVersion,
		},
		Timestamp:       time.Now(),
		Severity:        s.calculateTransactionSeverity(transaction),
		ComplianceFlags: s.generateComplianceFlags(transaction),
		RiskScore:       s.calculateTransactionRiskScore(transaction),
	}

	return s.auditRepo.CreateAuditLog(ctx, auditLog)
}

func (s *auditService) LogWalletAction(ctx context.Context, walletID primitive.ObjectID, userID int64, action string, details map[string]interface{}) error {
	auditLog := &AuditLog{
		LogType:   "wallet",
		Action:    action,
		UserID:    userID,
		WalletID:  walletID,
		Details:   details,
		Timestamp: time.Now(),
		Severity:  s.calculateWalletActionSeverity(action),
		RiskScore: s.calculateWalletActionRiskScore(action, details),
	}

	return s.auditRepo.CreateAuditLog(ctx, auditLog)
}

func (s *auditService) LogAdminAction(ctx context.Context, adminID string, action string, targetUserID int64, details map[string]interface{}) error {
	auditLog := &AuditLog{
		LogType:   "admin",
		Action:    action,
		AdminID:   adminID,
		UserID:    targetUserID,
		Details:   details,
		Timestamp: time.Now(),
		Severity:  "high", // All admin actions are high severity
		Metadata: map[string]interface{}{
			"admin_action": true,
		},
		RiskScore: 8, // Admin actions have high risk score for audit purposes
	}

	return s.auditRepo.CreateAuditLog(ctx, auditLog)
}

func (s *auditService) LogSystemEvent(ctx context.Context, eventType string, details map[string]interface{}) error {
	auditLog := &AuditLog{
		LogType:   "system",
		Action:    eventType,
		Details:   details,
		Timestamp: time.Now(),
		Severity:  s.calculateSystemEventSeverity(eventType),
		Metadata: map[string]interface{}{
			"system_event": true,
		},
		RiskScore: s.calculateSystemEventRiskScore(eventType),
	}

	return s.auditRepo.CreateAuditLog(ctx, auditLog)
}

func (s *auditService) GetAuditTrail(ctx context.Context, req *GetAuditTrailRequest) (*GetAuditTrailResponse, error) {
	limit := req.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	auditLogs, err := s.auditRepo.GetAuditLogs(ctx, &req.Filter, limit, offset)
	if err != nil {
		return &GetAuditTrailResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get audit trail: %v", err),
		}, nil
	}

	return &GetAuditTrailResponse{
		AuditLogs: auditLogs,
		Total:     int64(len(auditLogs)),
		Success:   true,
	}, nil
}

func (s *auditService) GetComplianceReport(ctx context.Context, req *GetComplianceReportRequest) (*GetComplianceReportResponse, error) {
	// Generate compliance report based on audit logs and transaction data
	report, err := s.generateComplianceReport(ctx, req.ReportType, req.Period, req.UserID)
	if err != nil {
		return &GetComplianceReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to generate compliance report: %v", err),
		}, nil
	}

	return &GetComplianceReportResponse{
		Report:  report,
		Success: true,
	}, nil
}

func (s *auditService) GenerateRegulatoryReport(ctx context.Context, req *GenerateRegulatoryReportRequest) (*GenerateRegulatoryReportResponse, error) {
	report, err := s.generateRegulatoryReport(ctx, req.ReportType, req.Period, req.GeneratedBy)
	if err != nil {
		return &GenerateRegulatoryReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to generate regulatory report: %v", err),
		}, nil
	}

	// Save the report
	if err := s.auditRepo.CreateComplianceReport(ctx, report); err != nil {
		return &GenerateRegulatoryReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to save regulatory report: %v", err),
		}, nil
	}

	return &GenerateRegulatoryReportResponse{
		Report:  report,
		Success: true,
	}, nil
}

func (s *auditService) TrackSuspiciousActivity(ctx context.Context, req *TrackSuspiciousActivityRequest) error {
	auditLog := &AuditLog{
		LogType: "suspicious_activity",
		Action:  req.ActivityType,
		UserID:  req.UserID,
		Details: map[string]interface{}{
			"description": req.Description,
			"evidence":    req.Evidence,
			"reported_by": req.ReportedBy,
		},
		Timestamp:       time.Now(),
		Severity:        "critical",
		ComplianceFlags: []string{"SUSPICIOUS_ACTIVITY", "MANUAL_REVIEW_REQUIRED"},
		RiskScore:       req.RiskScore,
	}

	return s.auditRepo.CreateAuditLog(ctx, auditLog)
}

func (s *auditService) ExportAuditData(ctx context.Context, req *ExportAuditDataRequest) (*ExportAuditDataResponse, error) {
	// Get audit logs based on filter
	auditLogs, err := s.auditRepo.GetAuditLogs(ctx, &req.Filter, 10000, 0) // Large limit for export
	if err != nil {
		return &ExportAuditDataResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to get audit data: %v", err),
		}, nil
	}

	// Export data in requested format
	data, fileName, contentType, err := s.exportAuditLogs(auditLogs, req.Format)
	if err != nil {
		return &ExportAuditDataResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to export audit data: %v", err),
		}, nil
	}

	return &ExportAuditDataResponse{
		Data:        data,
		FileName:    fileName,
		ContentType: contentType,
		Success:     true,
	}, nil
}

// Helper methods
func (s *auditService) calculateTransactionSeverity(transaction *models.Transaction) string {
	// Large amounts are high severity
	largeAmount, _ := decimal.NewFromString("10000")
	if transaction.Amount.Value.Abs().GreaterThan(largeAmount) {
		return "high"
	}

	// Failed transactions are medium severity
	if transaction.Status == "failed" {
		return "medium"
	}

	return "low"
}

func (s *auditService) generateComplianceFlags(transaction *models.Transaction) []string {
	var flags []string

	// Large transaction flag
	largeAmount, _ := decimal.NewFromString("10000")
	if transaction.Amount.Value.Abs().GreaterThan(largeAmount) {
		flags = append(flags, "LARGE_TRANSACTION")
	}

	// Cash reporting threshold
	ctrThreshold, _ := decimal.NewFromString("10000")
	if transaction.Amount.Value.Abs().GreaterThan(ctrThreshold) {
		flags = append(flags, "CTR_THRESHOLD")
	}

	// Failed transaction flag
	if transaction.Status == "failed" {
		flags = append(flags, "FAILED_TRANSACTION")
	}

	// Reversal flag
	if transaction.Type == "reversal" {
		flags = append(flags, "REVERSAL_TRANSACTION")
	}

	return flags
}

func (s *auditService) calculateTransactionRiskScore(transaction *models.Transaction) int {
	score := 1

	// Amount-based risk
	amount := transaction.Amount.Value.Abs()
	if amount.GreaterThan(decimal.NewFromFloat(1000)) {
		score += 2
	}
	if amount.GreaterThan(decimal.NewFromFloat(10000)) {
		score += 3
	}

	// Type-based risk
	switch transaction.Type {
	case "withdrawal":
		score += 2
	case "reversal":
		score += 4
	case "adjustment":
		score += 3
	}

	// Status-based risk
	if transaction.Status == "failed" {
		score += 3
	}

	// Cap at 10
	if score > 10 {
		score = 10
	}

	return score
}

func (s *auditService) calculateWalletActionSeverity(action string) string {
	switch action {
	case "wallet_suspended", "wallet_closed":
		return "critical"
	case "balance_adjustment", "manual_reconciliation":
		return "high"
	case "funds_locked", "funds_released":
		return "medium"
	default:
		return "low"
	}
}

func (s *auditService) calculateWalletActionRiskScore(action string, details map[string]interface{}) int {
	switch action {
	case "wallet_suspended", "wallet_closed":
		return 9
	case "balance_adjustment", "manual_reconciliation":
		return 7
	case "funds_locked", "funds_released":
		return 4
	default:
		return 2
	}
}

func (s *auditService) calculateSystemEventSeverity(eventType string) string {
	switch eventType {
	case "system_error", "database_error":
		return "critical"
	case "reconciliation_discrepancy":
		return "high"
	case "cleanup_completed":
		return "low"
	default:
		return "medium"
	}
}

func (s *auditService) calculateSystemEventRiskScore(eventType string) int {
	switch eventType {
	case "system_error", "database_error":
		return 8
	case "reconciliation_discrepancy":
		return 6
	case "cleanup_completed":
		return 1
	default:
		return 3
	}
}

func (s *auditService) generateComplianceReport(ctx context.Context, reportType string, period ReportPeriod, userID int64) (*ComplianceReport, error) {
	// This is a simplified implementation
	// In a real system, this would involve complex compliance calculations

	report := &ComplianceReport{
		ReportType:  reportType,
		Period:      period,
		GeneratedAt: time.Now(),
		Status:      "completed",
	}

	// Get audit logs for the period
	filter := &AuditFilter{
		StartDate: period.StartDate,
		EndDate:   period.EndDate,
		UserID:    userID,
	}

	auditLogs, err := s.auditRepo.GetAuditLogs(ctx, filter, 10000, 0)
	if err != nil {
		return nil, err
	}

	// Generate summary and findings
	report.Summary = s.generateComplianceSummary(auditLogs)
	report.Findings = s.generateComplianceFindings(auditLogs)

	return report, nil
}

func (s *auditService) generateRegulatoryReport(ctx context.Context, reportType string, period ReportPeriod, generatedBy string) (*ComplianceReport, error) {
	// Generate comprehensive regulatory report
	report := &ComplianceReport{
		ReportType:  reportType,
		Period:      period,
		GeneratedAt: time.Now(),
		GeneratedBy: generatedBy,
		Status:      "completed",
	}

	// This would involve complex regulatory calculations
	// For now, return a basic structure
	return report, nil
}

func (s *auditService) generateComplianceSummary(auditLogs []*AuditLog) ComplianceSummary {
	summary := ComplianceSummary{}

	suspiciousCount := int64(0)
	totalRiskScore := 0

	for _, log := range auditLogs {
		if log.LogType == "transaction" {
			summary.TotalTransactions++
		}

		if log.Severity == "critical" || log.RiskScore >= 8 {
			suspiciousCount++
		}

		totalRiskScore += log.RiskScore
	}

	summary.SuspiciousTransactions = suspiciousCount

	if len(auditLogs) > 0 {
		avgRiskScore := totalRiskScore / len(auditLogs)
		summary.RiskScore = avgRiskScore
		summary.ComplianceScore = 10 - avgRiskScore // Inverse relationship
	}

	return summary
}

func (s *auditService) generateComplianceFindings(auditLogs []*AuditLog) []ComplianceFinding {
	var findings []ComplianceFinding

	for _, log := range auditLogs {
		if log.Severity == "critical" || log.RiskScore >= 8 {
			finding := ComplianceFinding{
				FindingID:   fmt.Sprintf("finding_%s", log.ID.Hex()),
				Type:        log.LogType,
				Severity:    log.Severity,
				Description: fmt.Sprintf("High-risk %s detected", log.Action),
				UserID:      log.UserID,
				Evidence:    log.Details,
				Status:      "open",
				CreatedAt:   log.Timestamp,
			}
			findings = append(findings, finding)
		}
	}

	return findings
}

func (s *auditService) exportAuditLogs(auditLogs []*AuditLog, format string) ([]byte, string, string, error) {
	switch format {
	case "json":
		return s.exportAsJSON(auditLogs)
	case "csv":
		return s.exportAsCSV(auditLogs)
	default:
		return s.exportAsJSON(auditLogs)
	}
}

func (s *auditService) exportAsJSON(auditLogs []*AuditLog) ([]byte, string, string, error) {
	// Implementation would serialize audit logs to JSON
	return []byte("{}"), "audit_export.json", "application/json", nil
}

func (s *auditService) exportAsCSV(auditLogs []*AuditLog) ([]byte, string, string, error) {
	// Implementation would convert audit logs to CSV format
	return []byte(""), "audit_export.csv", "text/csv", nil
}