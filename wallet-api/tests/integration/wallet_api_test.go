package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"wallet-api/internal/config"
	"wallet-api/internal/controller"
	"wallet-api/internal/engine"
	"wallet-api/internal/middleware"
	"wallet-api/internal/models"
	"wallet-api/internal/repository"
	"wallet-api/internal/service"
)

type WalletAPITestSuite struct {
	suite.Suite
	router          *gin.Engine
	walletService   service.WalletService
	walletRepo      repository.WalletRepository
	transactionRepo repository.TransactionRepository
	ctx             context.Context
}

func (suite *WalletAPITestSuite) SetupSuite() {
	// Set gin to test mode
	gin.SetMode(gin.TestMode)

	// Initialize test context
	suite.ctx = context.Background()

	// For integration tests, we would typically use test databases
	// For this example, we'll use mock implementations
	suite.setupMockDependencies()
	suite.setupRouter()
}

func (suite *WalletAPITestSuite) setupMockDependencies() {
	// In a real integration test, you would:
	// 1. Set up test MongoDB database
	// 2. Set up test Redis instance
	// 3. Initialize actual repositories with test database connections

	// For this example, we'll use in-memory mock implementations
	suite.walletRepo = &MockWalletRepository{
		wallets: make(map[string]*models.Wallet),
	}
	suite.transactionRepo = &MockTransactionRepository{
		transactions: make(map[string]*models.Transaction),
	}

	// Create transaction engine
	transactionEngine := engine.NewTransactionEngine(
		suite.walletRepo,
		suite.transactionRepo,
		&MockDistributedLock{},
		&MockIdempotencyService{},
		&engine.TransactionEngineConfig{
			DefaultLockTimeout: 30 * time.Second,
			MaxRetries:         3,
			RetryDelay:         100 * time.Millisecond,
		},
	)

	// Create wallet service
	suite.walletService = service.NewWalletService(
		suite.walletRepo,
		transactionEngine,
		&service.WalletServiceConfig{
			DefaultCurrency:      "USD",
			MaxDailyTransactions: 100,
			MaxTransactionAmount: decimal.NewFromFloat(10000),
		},
	)
}

func (suite *WalletAPITestSuite) setupRouter() {
	suite.router = gin.New()

	// Add middleware
	suite.router.Use(middleware.RequestIDMiddleware())
	suite.router.Use(middleware.LoggingMiddleware())
	suite.router.Use(middleware.CORSMiddleware())

	// Create controllers
	walletController := controller.NewWalletController(suite.walletService)

	// Setup routes
	api := suite.router.Group("/api/v1")
	{
		wallets := api.Group("/wallets")
		{
			wallets.POST("", walletController.CreateWallet)
			wallets.GET("/:walletId", walletController.GetWallet)
			wallets.GET("/:walletId/balance", walletController.GetWalletBalance)
			wallets.POST("/:walletId/deposit", walletController.Deposit)
			wallets.POST("/:walletId/withdraw", walletController.Withdraw)
			wallets.POST("/:walletId/lock-funds", walletController.LockFunds)
			wallets.POST("/:walletId/unlock-funds", walletController.UnlockFunds)
			wallets.GET("/:walletId/transactions", walletController.GetTransactions)
		}

		api.POST("/transfers", walletController.Transfer)
	}

	// Health check endpoint
	suite.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
}

func (suite *WalletAPITestSuite) TearDownSuite() {
	// Cleanup resources if needed
}

func (suite *WalletAPITestSuite) SetupTest() {
	// Reset mock data before each test
	if mockRepo, ok := suite.walletRepo.(*MockWalletRepository); ok {
		mockRepo.wallets = make(map[string]*models.Wallet)
	}
	if mockRepo, ok := suite.transactionRepo.(*MockTransactionRepository); ok {
		mockRepo.transactions = make(map[string]*models.Transaction)
	}
}

func (suite *WalletAPITestSuite) TestHealthCheck() {
	req, _ := http.NewRequest("GET", "/health", nil)
	resp := httptest.NewRecorder()

	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", response["status"])
}

func (suite *WalletAPITestSuite) TestCreateWallet() {
	request := map[string]interface{}{
		"user_id":  12345,
		"currency": "USD",
		"type":     "personal",
	}

	requestBody, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/wallets", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusCreated, resp.Code)

	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "wallet_id")
	assert.Equal(suite.T(), float64(12345), response["user_id"])
	assert.Equal(suite.T(), "USD", response["currency"])
	assert.Equal(suite.T(), "personal", response["type"])
	assert.Equal(suite.T(), "active", response["status"])
}

func (suite *WalletAPITestSuite) TestCreateWalletInvalidRequest() {
	request := map[string]interface{}{
		"user_id": "invalid", // Should be integer
		"currency": "USD",
	}

	requestBody, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/wallets", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}

func (suite *WalletAPITestSuite) TestDepositWorkflow() {
	// First, create a wallet
	walletID := suite.createTestWallet(12345, "USD")

	// Now test deposit
	depositRequest := map[string]interface{}{
		"amount":    100.50,
		"currency":  "USD",
		"reference": "test-deposit",
	}

	requestBody, _ := json.Marshal(depositRequest)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/deposit", walletID), bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "transaction_id")
	assert.Equal(suite.T(), "deposit", response["type"])
	assert.Equal(suite.T(), 100.5, response["amount"].(map[string]interface{})["value"])
	assert.Equal(suite.T(), "completed", response["status"])
}

func (suite *WalletAPITestSuite) TestWithdrawWorkflow() {
	// Create a wallet and add some balance
	walletID := suite.createTestWallet(12345, "USD")
	suite.depositToWallet(walletID, 200.0)

	// Test withdrawal
	withdrawRequest := map[string]interface{}{
		"amount":    50.0,
		"currency":  "USD",
		"reference": "test-withdrawal",
	}

	requestBody, _ := json.Marshal(withdrawRequest)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/withdraw", walletID), bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "transaction_id")
	assert.Equal(suite.T(), "withdrawal", response["type"])
	assert.Equal(suite.T(), "completed", response["status"])
}

func (suite *WalletAPITestSuite) TestTransferWorkflow() {
	// Create two wallets
	fromWalletID := suite.createTestWallet(12345, "USD")
	toWalletID := suite.createTestWallet(67890, "USD")

	// Add balance to from wallet
	suite.depositToWallet(fromWalletID, 200.0)

	// Test transfer
	transferRequest := map[string]interface{}{
		"from_wallet_id": fromWalletID,
		"to_wallet_id":   toWalletID,
		"amount":         75.0,
		"currency":       "USD",
		"reference":      "test-transfer",
	}

	requestBody, _ := json.Marshal(transferRequest)
	req, _ := http.NewRequest("POST", "/api/v1/transfers", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "from_transaction")
	assert.Contains(suite.T(), response, "to_transaction")

	fromTx := response["from_transaction"].(map[string]interface{})
	toTx := response["to_transaction"].(map[string]interface{})

	assert.Equal(suite.T(), "transfer_out", fromTx["type"])
	assert.Equal(suite.T(), "transfer_in", toTx["type"])
	assert.Equal(suite.T(), "completed", fromTx["status"])
	assert.Equal(suite.T(), "completed", toTx["status"])
}

func (suite *WalletAPITestSuite) TestFundsLockWorkflow() {
	// Create a wallet and add balance
	walletID := suite.createTestWallet(12345, "USD")
	suite.depositToWallet(walletID, 200.0)

	// Test funds lock
	lockRequest := map[string]interface{}{
		"amount":    50.0,
		"reference": "order-123",
		"expires_at": time.Now().Add(30 * time.Minute).Format(time.RFC3339),
	}

	requestBody, _ := json.Marshal(lockRequest)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/lock-funds", walletID), bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "lock_id")
	assert.Equal(suite.T(), 50.0, response["amount"])
	assert.Equal(suite.T(), "order-123", response["reference"])
	assert.Equal(suite.T(), "active", response["status"])

	lockID := response["lock_id"].(string)

	// Test unlock funds
	unlockRequest := map[string]interface{}{
		"lock_id": lockID,
	}

	requestBody, _ = json.Marshal(unlockRequest)
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/unlock-funds", walletID), bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp = httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)
}

func (suite *WalletAPITestSuite) TestGetWalletBalance() {
	// Create a wallet and add balance
	walletID := suite.createTestWallet(12345, "USD")
	suite.depositToWallet(walletID, 150.0)

	// Get wallet balance
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/wallets/%s/balance", walletID), nil)
	resp := httptest.NewRecorder()

	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), 150.0, response["total"])
	assert.Equal(suite.T(), 150.0, response["available"])
	assert.Equal(suite.T(), 0.0, response["locked"])
	assert.Equal(suite.T(), "USD", response["currency"])
}

func (suite *WalletAPITestSuite) TestInsufficientBalanceError() {
	// Create a wallet without sufficient balance
	walletID := suite.createTestWallet(12345, "USD")
	suite.depositToWallet(walletID, 10.0)

	// Try to withdraw more than available
	withdrawRequest := map[string]interface{}{
		"amount":    50.0,
		"currency":  "USD",
		"reference": "test-withdrawal",
	}

	requestBody, _ := json.Marshal(withdrawRequest)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/withdraw", walletID), bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "error")
}

func (suite *WalletAPITestSuite) TestConcurrentTransactions() {
	// Create a wallet with balance
	walletID := suite.createTestWallet(12345, "USD")
	suite.depositToWallet(walletID, 1000.0)

	// Simulate concurrent withdrawals
	const numConcurrentRequests = 10
	const withdrawAmount = 50.0

	results := make(chan int, numConcurrentRequests)

	for i := 0; i < numConcurrentRequests; i++ {
		go func() {
			withdrawRequest := map[string]interface{}{
				"amount":    withdrawAmount,
				"currency":  "USD",
				"reference": fmt.Sprintf("concurrent-withdrawal-%d", i),
			}

			requestBody, _ := json.Marshal(withdrawRequest)
			req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/withdraw", walletID), bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			resp := httptest.NewRecorder()
			suite.router.ServeHTTP(resp, req)

			results <- resp.Code
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numConcurrentRequests; i++ {
		statusCode := <-results
		if statusCode == http.StatusOK {
			successCount++
		}
	}

	// All requests should succeed since we have sufficient balance
	assert.Equal(suite.T(), numConcurrentRequests, successCount)
}

// Helper methods

func (suite *WalletAPITestSuite) createTestWallet(userID int64, currency string) string {
	request := map[string]interface{}{
		"user_id":  userID,
		"currency": currency,
		"type":     "personal",
	}

	requestBody, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/wallets", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	var response map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &response)

	return response["wallet_id"].(string)
}

func (suite *WalletAPITestSuite) depositToWallet(walletID string, amount float64) {
	depositRequest := map[string]interface{}{
		"amount":    amount,
		"currency":  "USD",
		"reference": "test-setup-deposit",
	}

	requestBody, _ := json.Marshal(depositRequest)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/deposit", walletID), bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
}

// In-memory mock implementations for integration testing

type MockWalletRepository struct {
	wallets map[string]*models.Wallet
}

func (m *MockWalletRepository) CreateWallet(ctx context.Context, wallet *models.Wallet) error {
	if wallet.ID.IsZero() {
		wallet.ID = primitive.NewObjectID()
	}
	m.wallets[wallet.ID.Hex()] = wallet
	return nil
}

func (m *MockWalletRepository) GetWalletByID(ctx context.Context, walletID primitive.ObjectID) (*models.Wallet, error) {
	if wallet, exists := m.wallets[walletID.Hex()]; exists {
		return wallet, nil
	}
	return nil, fmt.Errorf("wallet not found")
}

func (m *MockWalletRepository) GetWalletsByUserID(ctx context.Context, userID int64) ([]*models.Wallet, error) {
	var wallets []*models.Wallet
	for _, wallet := range m.wallets {
		if wallet.UserID == userID {
			wallets = append(wallets, wallet)
		}
	}
	return wallets, nil
}

func (m *MockWalletRepository) UpdateWallet(ctx context.Context, wallet *models.Wallet) error {
	m.wallets[wallet.ID.Hex()] = wallet
	return nil
}

func (m *MockWalletRepository) UpdateBalance(ctx context.Context, walletID primitive.ObjectID, balance *models.WalletBalance) error {
	if wallet, exists := m.wallets[walletID.Hex()]; exists {
		wallet.Balance = balance
		return nil
	}
	return fmt.Errorf("wallet not found")
}

func (m *MockWalletRepository) LockWallet(ctx context.Context, walletID primitive.ObjectID) error {
	return nil
}

func (m *MockWalletRepository) UnlockWallet(ctx context.Context, walletID primitive.ObjectID) error {
	return nil
}

func (m *MockWalletRepository) GetWalletBalance(ctx context.Context, walletID primitive.ObjectID) (*models.WalletBalance, error) {
	if wallet, exists := m.wallets[walletID.Hex()]; exists {
		return wallet.Balance, nil
	}
	return nil, fmt.Errorf("wallet not found")
}

func (m *MockWalletRepository) AddFundsLock(ctx context.Context, walletID primitive.ObjectID, lock *models.FundsLock) error {
	if wallet, exists := m.wallets[walletID.Hex()]; exists {
		wallet.Locks = append(wallet.Locks, *lock)
		return nil
	}
	return fmt.Errorf("wallet not found")
}

func (m *MockWalletRepository) RemoveFundsLock(ctx context.Context, walletID primitive.ObjectID, lockID string) error {
	if wallet, exists := m.wallets[walletID.Hex()]; exists {
		for i, lock := range wallet.Locks {
			if lock.LockID == lockID {
				wallet.Locks = append(wallet.Locks[:i], wallet.Locks[i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("lock not found")
}

func (m *MockWalletRepository) GetExpiredLocks(ctx context.Context, before time.Time) ([]*models.FundsLock, error) {
	return []*models.FundsLock{}, nil
}

type MockTransactionRepository struct {
	transactions map[string]*models.Transaction
}

func (m *MockTransactionRepository) CreateTransaction(ctx context.Context, transaction *models.Transaction) error {
	m.transactions[transaction.TransactionID] = transaction
	return nil
}

func (m *MockTransactionRepository) GetTransactionByID(ctx context.Context, transactionID string) (*models.Transaction, error) {
	if tx, exists := m.transactions[transactionID]; exists {
		return tx, nil
	}
	return nil, fmt.Errorf("transaction not found")
}

func (m *MockTransactionRepository) UpdateTransaction(ctx context.Context, transaction *models.Transaction) error {
	m.transactions[transaction.TransactionID] = transaction
	return nil
}

func (m *MockTransactionRepository) GetTransactionsByWalletID(ctx context.Context, walletID primitive.ObjectID, filter *models.TransactionFilter) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	for _, tx := range m.transactions {
		if tx.WalletID == walletID {
			transactions = append(transactions, tx)
		}
	}
	return transactions, nil
}

func (m *MockTransactionRepository) GetTransactionsByUserID(ctx context.Context, userID int64, filter *models.TransactionFilter) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	for _, tx := range m.transactions {
		if tx.UserID == userID {
			transactions = append(transactions, tx)
		}
	}
	return transactions, nil
}

func (m *MockTransactionRepository) GetPendingTransactions(ctx context.Context, before time.Time) ([]*models.Transaction, error) {
	return []*models.Transaction{}, nil
}

func (m *MockTransactionRepository) GetTransactionStats(ctx context.Context, userID int64, period time.Duration) (*models.TransactionStats, error) {
	return &models.TransactionStats{}, nil
}

func TestWalletAPITestSuite(t *testing.T) {
	suite.Run(t, new(WalletAPITestSuite))
}