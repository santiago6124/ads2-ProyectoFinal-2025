package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"wallet-api/internal/models"
	"wallet-api/internal/service"
)

type WalletController struct {
	walletService service.WalletService
}

func NewWalletController(walletService service.WalletService) *WalletController {
	return &WalletController{
		walletService: walletService,
	}
}

// @Summary Create a new wallet
// @Description Create a new wallet for a user
// @Tags wallets
// @Accept json
// @Produce json
// @Param request body CreateWalletRequest true "Create wallet request"
// @Success 201 {object} service.CreateWalletResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/wallet [post]
func (c *WalletController) CreateWallet(ctx *gin.Context) {
	var req CreateWalletRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Validate request
	if err := c.validateCreateWalletRequest(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &service.CreateWalletRequest{
		UserID:         req.UserID,
		InitialBalance: req.InitialBalance,
		Currency:       req.Currency,
		Limits:         req.Limits,
	}

	// Create wallet
	response, err := c.walletService.CreateWallet(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create wallet",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Wallet creation failed",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

// @Summary Get wallet information
// @Description Get wallet information for a specific user
// @Tags wallets
// @Produce json
// @Param userId path int true "User ID"
// @Success 200 {object} service.GetWalletResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/wallet/{userId} [get]
func (c *WalletController) GetWallet(ctx *gin.Context) {
	userID, err := c.getUserIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	response, err := c.walletService.GetWallet(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get wallet",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Wallet not found",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Get wallet balance
// @Description Get current balance for a specific user's wallet
// @Tags wallets
// @Produce json
// @Param userId path int true "User ID"
// @Success 200 {object} service.GetBalanceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/wallet/{userId}/balance [get]
func (c *WalletController) GetBalance(ctx *gin.Context) {
	userID, err := c.getUserIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	response, err := c.walletService.GetBalance(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get balance",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Balance not found",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Deposit funds
// @Description Deposit funds to a user's wallet
// @Tags wallets
// @Accept json
// @Produce json
// @Param userId path int true "User ID"
// @Param request body DepositRequest true "Deposit request"
// @Success 200 {object} service.DepositResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/wallet/{userId}/deposit [post]
func (c *WalletController) Deposit(ctx *gin.Context) {
	userID, err := c.getUserIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	var req DepositRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Validate request
	if err := c.validateDepositRequest(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &service.DepositRequest{
		UserID:         userID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Reference:      req.Reference,
		IdempotencyKey: req.IdempotencyKey,
		Metadata:       req.Metadata,
		AuditInfo:      c.extractAuditInfo(ctx),
	}

	response, err := c.walletService.Deposit(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to process deposit",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Deposit failed",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Withdraw funds
// @Description Withdraw funds from a user's wallet
// @Tags wallets
// @Accept json
// @Produce json
// @Param userId path int true "User ID"
// @Param request body WithdrawRequest true "Withdraw request"
// @Success 200 {object} service.WithdrawResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/wallet/{userId}/withdraw [post]
func (c *WalletController) Withdraw(ctx *gin.Context) {
	userID, err := c.getUserIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	var req WithdrawRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Validate request
	if err := c.validateWithdrawRequest(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &service.WithdrawRequest{
		UserID:         userID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Reference:      req.Reference,
		IdempotencyKey: req.IdempotencyKey,
		Metadata:       req.Metadata,
		AuditInfo:      c.extractAuditInfo(ctx),
	}

	response, err := c.walletService.Withdraw(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to process withdrawal",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Withdrawal failed",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Lock funds
// @Description Lock funds in a user's wallet for pending orders
// @Tags wallets
// @Accept json
// @Produce json
// @Param userId path int true "User ID"
// @Param request body LockFundsRequest true "Lock funds request"
// @Success 200 {object} service.LockFundsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/wallet/{userId}/lock [post]
func (c *WalletController) LockFunds(ctx *gin.Context) {
	userID, err := c.getUserIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	var req LockFundsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Validate request
	if err := c.validateLockFundsRequest(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &service.LockFundsRequest{
		UserID:         userID,
		Amount:         req.Amount,
		OrderID:        req.OrderID,
		Reason:         req.Reason,
		ExpirationTime: req.ExpirationTime,
		IdempotencyKey: req.IdempotencyKey,
		AuditInfo:      c.extractAuditInfo(ctx),
	}

	response, err := c.walletService.LockFunds(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to lock funds",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Fund locking failed",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Release funds
// @Description Release locked funds in a user's wallet
// @Tags wallets
// @Accept json
// @Produce json
// @Param userId path int true "User ID"
// @Param lockId path string true "Lock ID"
// @Success 200 {object} service.ReleaseFundsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/wallet/{userId}/release/{lockId} [post]
func (c *WalletController) ReleaseFunds(ctx *gin.Context) {
	userID, err := c.getUserIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	lockID := ctx.Param("lockId")
	if lockID == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Lock ID is required",
			Message: "Lock ID parameter is missing",
		})
		return
	}

	// Convert to service request
	serviceReq := &service.ReleaseFundsRequest{
		UserID:    userID,
		LockID:    lockID,
		AuditInfo: c.extractAuditInfo(ctx),
	}

	response, err := c.walletService.ReleaseFunds(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to release funds",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Fund release failed",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Execute lock
// @Description Execute a fund lock to complete an order
// @Tags wallets
// @Accept json
// @Produce json
// @Param userId path int true "User ID"
// @Param lockId path string true "Lock ID"
// @Param request body ExecuteLockRequest true "Execute lock request"
// @Success 200 {object} service.ExecuteLockResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/wallet/{userId}/execute/{lockId} [post]
func (c *WalletController) ExecuteLock(ctx *gin.Context) {
	userID, err := c.getUserIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	lockID := ctx.Param("lockId")
	if lockID == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Lock ID is required",
			Message: "Lock ID parameter is missing",
		})
		return
	}

	var req ExecuteLockRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Validate request
	if err := c.validateExecuteLockRequest(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &service.ExecuteLockRequest{
		UserID:          userID,
		LockID:          lockID,
		ActualAmount:    req.ActualAmount,
		TransactionType: req.TransactionType,
		Reference:       req.Reference,
		IdempotencyKey:  req.IdempotencyKey,
		Metadata:        req.Metadata,
		AuditInfo:       c.extractAuditInfo(ctx),
	}

	response, err := c.walletService.ExecuteLock(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to execute lock",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Lock execution failed",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Get transaction history
// @Description Get transaction history for a user's wallet
// @Tags wallets
// @Produce json
// @Param userId path int true "User ID"
// @Param limit query int false "Number of transactions to return" default(50)
// @Param offset query int false "Number of transactions to skip" default(0)
// @Param type query string false "Transaction type filter"
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Success 200 {object} service.GetTransactionHistoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/wallet/{userId}/transactions [get]
func (c *WalletController) GetTransactions(ctx *gin.Context) {
	userID, err := c.getUserIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	// Parse query parameters
	limit := c.getQueryInt(ctx, "limit", 50)
	offset := c.getQueryInt(ctx, "offset", 0)
	transactionType := ctx.Query("type")

	var startDate, endDate time.Time
	if startDateStr := ctx.Query("start_date"); startDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startDate = parsed
		}
	}
	if endDateStr := ctx.Query("end_date"); endDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endDate = parsed
		}
	}

	// Convert to service request
	serviceReq := &service.GetTransactionHistoryRequest{
		UserID:          userID,
		Limit:           limit,
		Offset:          offset,
		TransactionType: transactionType,
		StartDate:       startDate,
		EndDate:         endDate,
	}

	response, err := c.walletService.GetTransactionHistory(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get transaction history",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to retrieve transactions",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Get specific transaction
// @Description Get details of a specific transaction
// @Tags wallets
// @Produce json
// @Param userId path int true "User ID"
// @Param transactionId path string true "Transaction ID"
// @Success 200 {object} service.GetTransactionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/wallet/{userId}/transaction/{transactionId} [get]
func (c *WalletController) GetTransaction(ctx *gin.Context) {
	userID, err := c.getUserIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	transactionID := ctx.Param("transactionId")
	if transactionID == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Transaction ID is required",
			Message: "Transaction ID parameter is missing",
		})
		return
	}

	// Convert to service request
	serviceReq := &service.GetTransactionRequest{
		UserID:        userID,
		TransactionID: transactionID,
	}

	response, err := c.walletService.GetTransaction(ctx.Request.Context(), serviceReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get transaction",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Transaction not found",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Get wallet statistics
// @Description Get statistics and metrics for a user's wallet
// @Tags wallets
// @Produce json
// @Param userId path int true "User ID"
// @Success 200 {object} service.WalletStatsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/wallet/{userId}/stats [get]
func (c *WalletController) GetWalletStats(ctx *gin.Context) {
	userID, err := c.getUserIDFromPath(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID",
			Message: err.Error(),
		})
		return
	}

	response, err := c.walletService.GetWalletStats(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get wallet statistics",
			Message: err.Error(),
		})
		return
	}

	if !response.Success {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to retrieve statistics",
			Message: response.ErrorMessage,
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Request/Response DTOs
type CreateWalletRequest struct {
	UserID         int64           `json:"user_id" binding:"required,min=1"`
	InitialBalance decimal.Decimal `json:"initial_balance" binding:"required"`
	Currency       string          `json:"currency" binding:"required,len=3"`
	Limits         *models.Limits  `json:"limits,omitempty"`
}

type DepositRequest struct {
	Amount         decimal.Decimal        `json:"amount" binding:"required"`
	Currency       string                 `json:"currency" binding:"required,len=3"`
	Reference      models.Reference       `json:"reference" binding:"required"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type WithdrawRequest struct {
	Amount         decimal.Decimal        `json:"amount" binding:"required"`
	Currency       string                 `json:"currency" binding:"required,len=3"`
	Reference      models.Reference       `json:"reference" binding:"required"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type LockFundsRequest struct {
	Amount         decimal.Decimal `json:"amount" binding:"required"`
	OrderID        string          `json:"order_id" binding:"required"`
	Reason         string          `json:"reason" binding:"required"`
	ExpirationTime time.Duration   `json:"expiration_time,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
}

type ExecuteLockRequest struct {
	ActualAmount    decimal.Decimal        `json:"actual_amount" binding:"required"`
	TransactionType string                 `json:"transaction_type" binding:"required"`
	Reference       models.Reference       `json:"reference" binding:"required"`
	IdempotencyKey  string                 `json:"idempotency_key,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// Helper methods
func (c *WalletController) getUserIDFromPath(ctx *gin.Context) (int64, error) {
	userIDStr := ctx.Param("userId")
	return strconv.ParseInt(userIDStr, 10, 64)
}

func (c *WalletController) getQueryInt(ctx *gin.Context, key string, defaultValue int) int {
	if valueStr := ctx.Query(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

func (c *WalletController) extractAuditInfo(ctx *gin.Context) models.AuditInfo {
	return models.AuditInfo{
		IPAddress:  ctx.ClientIP(),
		UserAgent:  ctx.GetHeader("User-Agent"),
		SessionID:  ctx.GetHeader("X-Session-ID"),
		APIVersion: ctx.GetHeader("X-API-Version"),
	}
}

// Validation methods
func (c *WalletController) validateCreateWalletRequest(req *CreateWalletRequest) error {
	if req.UserID <= 0 {
		return fmt.Errorf("user ID must be positive")
	}

	if req.InitialBalance.LessThan(decimal.Zero) {
		return fmt.Errorf("initial balance cannot be negative")
	}

	if req.Currency == "" {
		return fmt.Errorf("currency is required")
	}

	return nil
}

func (c *WalletController) validateDepositRequest(req *DepositRequest) error {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("deposit amount must be positive")
	}

	if req.Currency == "" {
		return fmt.Errorf("currency is required")
	}

	if req.Reference.Type == "" {
		return fmt.Errorf("reference type is required")
	}

	return nil
}

func (c *WalletController) validateWithdrawRequest(req *WithdrawRequest) error {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("withdrawal amount must be positive")
	}

	if req.Currency == "" {
		return fmt.Errorf("currency is required")
	}

	if req.Reference.Type == "" {
		return fmt.Errorf("reference type is required")
	}

	return nil
}

func (c *WalletController) validateLockFundsRequest(req *LockFundsRequest) error {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("lock amount must be positive")
	}

	if req.OrderID == "" {
		return fmt.Errorf("order ID is required")
	}

	if req.Reason == "" {
		return fmt.Errorf("reason is required")
	}

	return nil
}

func (c *WalletController) validateExecuteLockRequest(req *ExecuteLockRequest) error {
	if req.ActualAmount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("actual amount must be positive")
	}

	if req.TransactionType == "" {
		return fmt.Errorf("transaction type is required")
	}

	if req.Reference.Type == "" {
		return fmt.Errorf("reference type is required")
	}

	return nil
}