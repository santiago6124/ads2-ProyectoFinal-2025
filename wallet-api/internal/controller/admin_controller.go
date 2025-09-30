package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"wallet-api/internal/service"
)

type AdminController struct {
	adminService      service.AdminService
	auditService      service.AuditService
	complianceService service.ComplianceService
}

func NewAdminController(
	adminService service.AdminService,
	auditService service.AuditService,
	complianceService service.ComplianceService,
) *AdminController {
	return &AdminController{
		adminService:      adminService,
		auditService:      auditService,
		complianceService: complianceService,
	}
}

// @Summary Reconcile wallet
// @Description Perform reconciliation for a specific wallet
// @Tags admin
// @Accept json
// @Produce json
// @Param request body ReconcileWalletRequest true "Reconcile wallet request"
// @Success 200 {object} service.ReconcileWalletResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security InternalAPI
// @Router /api/wallet/admin/reconcile [post]
func (c *AdminController) ReconcileWallet(ctx *gin.Context) {
	var req ReconcileWalletRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Validate request
	if req.UserID <= 0 {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: "User ID must be positive",
		})
		return
	}

	// Convert to service request
	serviceReq := &service.ReconcileWalletRequest{
		UserID: req.UserID,
	}

	response, err := c.adminService.ReconcileWallet(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to reconcile wallet",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Reconciliation failed",
			Message: response.ErrorMessage,
		})
		return
	}

	// Log admin action
	c.logAdminAction(ctx, "wallet_reconciliation", req.UserID, map[string]interface{}{
		"user_id": req.UserID,
		"result":  response.Result,
	})

	ctx.JSON(http.StatusOK, response)
}

// @Summary Reconcile all wallets
// @Description Perform batch reconciliation for multiple wallets
// @Tags admin
// @Accept json
// @Produce json
// @Param request body ReconcileAllWalletsRequest true "Reconcile all wallets request"
// @Success 200 {object} service.ReconcileAllWalletsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security InternalAPI
// @Router /api/wallet/admin/reconcile/batch [post]
func (c *AdminController) ReconcileAllWallets(ctx *gin.Context) {
	var req ReconcileAllWalletsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Set default batch size if not provided
	if req.BatchSize <= 0 {
		req.BatchSize = 100
	}

	// Convert to service request
	serviceReq := &service.ReconcileAllWalletsRequest{
		BatchSize: req.BatchSize,
	}

	response, err := c.adminService.ReconcileAllWallets(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to reconcile wallets",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Batch reconciliation failed",
			Message: response.ErrorMessage,
		})
		return
	}

	// Log admin action
	c.logAdminAction(ctx, "batch_reconciliation", 0, map[string]interface{}{
		"batch_size":         req.BatchSize,
		"reconciled_wallets": response.Result.ReconciledWallets,
		"discrepancies":      response.Result.DiscrepanciesFound,
		"errors":             response.Result.ErrorsEncountered,
	})

	ctx.JSON(http.StatusOK, response)
}

// @Summary Get audit report
// @Description Generate audit report for a specific user and period
// @Tags admin
// @Produce json
// @Param userId path int true "User ID"
// @Param start_date query string true "Start date (RFC3339 format)"
// @Param end_date query string true "End date (RFC3339 format)"
// @Success 200 {object} service.GetAuditReportResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security InternalAPI
// @Router /api/wallet/admin/audit/{userId} [get]
func (c *AdminController) GetAuditReport(ctx *gin.Context) {
	userID, err := c.getUserIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	// Parse date parameters
	startDateStr := ctx.Query("start_date")
	endDateStr := ctx.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Date parameters required",
			Message: "Both start_date and end_date are required",
		})
		return
	}

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid start date format",
			Message: "Start date must be in RFC3339 format",
		})
		return
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid end date format",
			Message: "End date must be in RFC3339 format",
		})
		return
	}

	// Convert to service request
	serviceReq := &service.GetAuditReportRequest{
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	response, err := c.adminService.GetAuditReport(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate audit report",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Audit report generation failed",
			Message: response.ErrorMessage,
		})
		return
	}

	// Log admin action
	c.logAdminAction(ctx, "audit_report_generated", userID, map[string]interface{}{
		"user_id":    userID,
		"start_date": startDate,
		"end_date":   endDate,
	})

	ctx.JSON(http.StatusOK, response)
}

// @Summary Create balance adjustment
// @Description Create a manual balance adjustment for a user's wallet
// @Tags admin
// @Accept json
// @Produce json
// @Param request body CreateBalanceAdjustmentRequest true "Balance adjustment request"
// @Success 200 {object} service.CreateBalanceAdjustmentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security InternalAPI
// @Router /api/wallet/admin/adjust [post]
func (c *AdminController) CreateBalanceAdjustment(ctx *gin.Context) {
	var req CreateBalanceAdjustmentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Validate request
	if err := c.validateBalanceAdjustmentRequest(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &service.CreateBalanceAdjustmentRequest{
		UserID:     req.UserID,
		Amount:     req.Amount,
		Reason:     req.Reason,
		AdjustedBy: c.getAdminID(ctx),
		AuditInfo:  c.extractAuditInfo(ctx),
	}

	response, err := c.adminService.CreateBalanceAdjustment(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create balance adjustment",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Balance adjustment failed",
			Message: response.ErrorMessage,
		})
		return
	}

	// Log admin action
	c.logAdminAction(ctx, "balance_adjustment", req.UserID, map[string]interface{}{
		"user_id":        req.UserID,
		"amount":         req.Amount.String(),
		"reason":         req.Reason,
		"transaction_id": response.Transaction.TransactionID,
	})

	ctx.JSON(http.StatusOK, response)
}

// @Summary Force unlock funds
// @Description Force unlock funds for a specific lock
// @Tags admin
// @Accept json
// @Produce json
// @Param request body ForceUnlockFundsRequest true "Force unlock request"
// @Success 200 {object} service.ForceUnlockFundsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security InternalAPI
// @Router /api/wallet/admin/unlock [post]
func (c *AdminController) ForceUnlockFunds(ctx *gin.Context) {
	var req ForceUnlockFundsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Validate request
	if err := c.validateForceUnlockRequest(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &service.ForceUnlockFundsRequest{
		UserID:     req.UserID,
		LockID:     req.LockID,
		Reason:     req.Reason,
		UnlockedBy: c.getAdminID(ctx),
	}

	response, err := c.adminService.ForceUnlockFunds(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to force unlock funds",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Force unlock failed",
			Message: response.ErrorMessage,
		})
		return
	}

	// Log admin action
	c.logAdminAction(ctx, "force_unlock_funds", req.UserID, map[string]interface{}{
		"user_id": req.UserID,
		"lock_id": req.LockID,
		"reason":  req.Reason,
	})

	ctx.JSON(http.StatusOK, response)
}

// @Summary Get system health
// @Description Get overall system health status
// @Tags admin
// @Produce json
// @Success 200 {object} service.SystemHealthResponse
// @Failure 500 {object} ErrorResponse
// @Security InternalAPI
// @Router /api/wallet/admin/health [get]
func (c *AdminController) GetSystemHealth(ctx *gin.Context) {
	response, err := c.adminService.GetSystemHealth(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get system health",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "System health check failed",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Get wallet metrics
// @Description Get wallet metrics for a specific period
// @Tags admin
// @Produce json
// @Param start_date query string true "Start date (RFC3339 format)"
// @Param end_date query string true "End date (RFC3339 format)"
// @Success 200 {object} service.GetWalletMetricsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security InternalAPI
// @Router /api/wallet/admin/metrics [get]
func (c *AdminController) GetWalletMetrics(ctx *gin.Context) {
	// Parse date parameters
	startDateStr := ctx.Query("start_date")
	endDateStr := ctx.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Date parameters required",
			Message: "Both start_date and end_date are required",
		})
		return
	}

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid start date format",
			Message: "Start date must be in RFC3339 format",
		})
		return
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid end date format",
			Message: "End date must be in RFC3339 format",
		})
		return
	}

	// Convert to service request
	serviceReq := &service.GetWalletMetricsRequest{
		StartDate: startDate,
		EndDate:   endDate,
	}

	response, err := c.adminService.GetWalletMetrics(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get wallet metrics",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Metrics retrieval failed",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Cleanup expired locks
// @Description Clean up expired fund locks across all wallets
// @Tags admin
// @Produce json
// @Success 200 {object} service.CleanupResponse
// @Failure 500 {object} ErrorResponse
// @Security InternalAPI
// @Router /api/wallet/admin/cleanup [post]
func (c *AdminController) CleanupExpiredLocks(ctx *gin.Context) {
	response, err := c.adminService.CleanupExpiredLocks(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to cleanup expired locks",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Cleanup operation failed",
			Message: response.ErrorMessage,
		})
		return
	}

	// Log admin action
	c.logAdminAction(ctx, "cleanup_expired_locks", 0, map[string]interface{}{
		"locks_removed": response.ExpiredLocksRemoved,
	})

	ctx.JSON(http.StatusOK, response)
}

// @Summary Get compliance status
// @Description Get compliance status for a specific user
// @Tags admin
// @Produce json
// @Param userId path int true "User ID"
// @Success 200 {object} service.ComplianceStatusResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security InternalAPI
// @Router /api/wallet/admin/compliance/{userId} [get]
func (c *AdminController) GetComplianceStatus(ctx *gin.Context) {
	userID, err := c.getUserIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	response, err := c.complianceService.GetComplianceStatus(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get compliance status",
			Message: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Get suspicious transactions
// @Description Get suspicious transactions for review
// @Tags admin
// @Produce json
// @Param start_date query string true "Start date (RFC3339 format)"
// @Param end_date query string true "End date (RFC3339 format)"
// @Param limit query int false "Number of transactions to return" default(100)
// @Success 200 {object} service.GetSuspiciousTransactionsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security InternalAPI
// @Router /api/wallet/admin/suspicious [get]
func (c *AdminController) GetSuspiciousTransactions(ctx *gin.Context) {
	// Parse date parameters
	startDateStr := ctx.Query("start_date")
	endDateStr := ctx.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Date parameters required",
			Message: "Both start_date and end_date are required",
		})
		return
	}

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid start date format",
			Message: "Start date must be in RFC3339 format",
		})
		return
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid end date format",
			Message: "End date must be in RFC3339 format",
		})
		return
	}

	limit := c.getQueryInt(ctx, "limit", 100)

	// Convert to service request
	serviceReq := &service.GetSuspiciousTransactionsRequest{
		StartDate: startDate,
		EndDate:   endDate,
		Limit:     limit,
	}

	response, err := c.adminService.GetSuspiciousTransactions(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get suspicious transactions",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to retrieve suspicious transactions",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Request DTOs
type ReconcileWalletRequest struct {
	UserID int64 `json:"user_id" binding:"required,min=1"`
}

type ReconcileAllWalletsRequest struct {
	BatchSize int `json:"batch_size" binding:"min=1,max=1000"`
}

type CreateBalanceAdjustmentRequest struct {
	UserID int64           `json:"user_id" binding:"required,min=1"`
	Amount decimal.Decimal `json:"amount" binding:"required"`
	Reason string          `json:"reason" binding:"required,min=1,max=500"`
}

type ForceUnlockFundsRequest struct {
	UserID int64  `json:"user_id" binding:"required,min=1"`
	LockID string `json:"lock_id" binding:"required"`
	Reason string `json:"reason" binding:"required,min=1,max=500"`
}

// Helper methods
func (c *AdminController) getUserIDFromPath(ctx *gin.Context) (int64, error) {
	userIDStr := ctx.Param("userId")
	return strconv.ParseInt(userIDStr, 10, 64)
}

func (c *AdminController) getQueryInt(ctx *gin.Context, key string, defaultValue int) int {
	if valueStr := ctx.Query(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

func (c *AdminController) getAdminID(ctx *gin.Context) string {
	// In a real implementation, this would extract the admin ID from JWT token or session
	return ctx.GetHeader("X-Admin-ID")
}

func (c *AdminController) extractAuditInfo(ctx *gin.Context) service.AuditInfo {
	return service.AuditInfo{
		IPAddress:  ctx.ClientIP(),
		UserAgent:  ctx.GetHeader("User-Agent"),
		SessionID:  ctx.GetHeader("X-Session-ID"),
		APIVersion: ctx.GetHeader("X-API-Version"),
	}
}

func (c *AdminController) logAdminAction(ctx *gin.Context, action string, targetUserID int64, details map[string]interface{}) {
	adminID := c.getAdminID(ctx)
	if adminID == "" {
		adminID = "unknown"
	}

	// Log admin action asynchronously
	go func() {
		c.auditService.LogAdminAction(
			ctx.Request.Context(),
			adminID,
			action,
			targetUserID,
			details,
		)
	}()
}

// Validation methods
func (c *AdminController) validateBalanceAdjustmentRequest(req *CreateBalanceAdjustmentRequest) error {
	if req.UserID <= 0 {
		return fmt.Errorf("user ID must be positive")
	}

	if req.Amount.IsZero() {
		return fmt.Errorf("adjustment amount cannot be zero")
	}

	// Validate amount is within reasonable bounds
	maxAdjustment, _ := decimal.NewFromString("1000000") // 1M limit
	if req.Amount.Abs().GreaterThan(maxAdjustment) {
		return fmt.Errorf("adjustment amount exceeds maximum allowed")
	}

	if req.Reason == "" {
		return fmt.Errorf("reason is required")
	}

	return nil
}

func (c *AdminController) validateForceUnlockRequest(req *ForceUnlockFundsRequest) error {
	if req.UserID <= 0 {
		return fmt.Errorf("user ID must be positive")
	}

	if req.LockID == "" {
		return fmt.Errorf("lock ID is required")
	}

	if req.Reason == "" {
		return fmt.Errorf("reason is required")
	}

	return nil
}