package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"wallet-api/internal/models"
)

// Mock repositories for testing
type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) CreateWallet(ctx context.Context, wallet *models.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockWalletRepository) GetWalletByID(ctx context.Context, walletID primitive.ObjectID) (*models.Wallet, error) {
	args := m.Called(ctx, walletID)
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletRepository) GetWalletsByUserID(ctx context.Context, userID int64) ([]*models.Wallet, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.Wallet), args.Error(1)
}

func (m *MockWalletRepository) UpdateWallet(ctx context.Context, wallet *models.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockWalletRepository) UpdateBalance(ctx context.Context, walletID primitive.ObjectID, balance *models.WalletBalance) error {
	args := m.Called(ctx, walletID, balance)
	return args.Error(0)
}

func (m *MockWalletRepository) LockWallet(ctx context.Context, walletID primitive.ObjectID) error {
	args := m.Called(ctx, walletID)
	return args.Error(0)
}

func (m *MockWalletRepository) UnlockWallet(ctx context.Context, walletID primitive.ObjectID) error {
	args := m.Called(ctx, walletID)
	return args.Error(0)
}

func (m *MockWalletRepository) GetWalletBalance(ctx context.Context, walletID primitive.ObjectID) (*models.WalletBalance, error) {
	args := m.Called(ctx, walletID)
	return args.Get(0).(*models.WalletBalance), args.Error(1)
}

func (m *MockWalletRepository) AddFundsLock(ctx context.Context, walletID primitive.ObjectID, lock *models.FundsLock) error {
	args := m.Called(ctx, walletID, lock)
	return args.Error(0)
}

func (m *MockWalletRepository) RemoveFundsLock(ctx context.Context, walletID primitive.ObjectID, lockID string) error {
	args := m.Called(ctx, walletID, lockID)
	return args.Error(0)
}

func (m *MockWalletRepository) GetExpiredLocks(ctx context.Context, before time.Time) ([]*models.FundsLock, error) {
	args := m.Called(ctx, before)
	return args.Get(0).([]*models.FundsLock), args.Error(1)
}

type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) CreateTransaction(ctx context.Context, transaction *models.Transaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetTransactionByID(ctx context.Context, transactionID string) (*models.Transaction, error) {
	args := m.Called(ctx, transactionID)
	return args.Get(0).(*models.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) UpdateTransaction(ctx context.Context, transaction *models.Transaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetTransactionsByWalletID(ctx context.Context, walletID primitive.ObjectID, filter *models.TransactionFilter) ([]*models.Transaction, error) {
	args := m.Called(ctx, walletID, filter)
	return args.Get(0).([]*models.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetTransactionsByUserID(ctx context.Context, userID int64, filter *models.TransactionFilter) ([]*models.Transaction, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).([]*models.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetPendingTransactions(ctx context.Context, before time.Time) ([]*models.Transaction, error) {
	args := m.Called(ctx, before)
	return args.Get(0).([]*models.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetTransactionStats(ctx context.Context, userID int64, period time.Duration) (*models.TransactionStats, error) {
	args := m.Called(ctx, userID, period)
	return args.Get(0).(*models.TransactionStats), args.Error(1)
}

type MockDistributedLock struct {
	mock.Mock
}

func (m *MockDistributedLock) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	args := m.Called(ctx, key, ttl)
	return args.Bool(0), args.Error(1)
}

func (m *MockDistributedLock) ReleaseLock(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockDistributedLock) ExtendLock(ctx context.Context, key string, ttl time.Duration) error {
	args := m.Called(ctx, key, ttl)
	return args.Error(0)
}

type MockIdempotencyService struct {
	mock.Mock
}

func (m *MockIdempotencyService) CheckIdempotency(ctx context.Context, key string) (*IdempotencyResult, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(*IdempotencyResult), args.Error(1)
}

func (m *MockIdempotencyService) StoreResult(ctx context.Context, key string, result interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, result, ttl)
	return args.Error(0)
}

func (m *MockIdempotencyService) InvalidateKey(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func TestTransactionEngine_ProcessDeposit(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.TransactionRequest
		wallet      *models.Wallet
		setupMocks  func(*MockWalletRepository, *MockTransactionRepository, *MockDistributedLock, *MockIdempotencyService)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful deposit",
			request: &models.TransactionRequest{
				UserID:   12345,
				WalletID: primitive.NewObjectID(),
				Type:     "deposit",
				Amount:   decimal.NewFromFloat(100.0),
				Currency: "USD",
				Reference: "test-deposit",
			},
			wallet: &models.Wallet{
				ID:     primitive.NewObjectID(),
				UserID: 12345,
				Balance: &models.WalletBalance{
					Total:     decimal.NewFromFloat(50.0),
					Available: decimal.NewFromFloat(50.0),
					Locked:    decimal.Zero,
					Currency:  "USD",
				},
				Status: "active",
			},
			setupMocks: func(wr *MockWalletRepository, tr *MockTransactionRepository, dl *MockDistributedLock, is *MockIdempotencyService) {
				// Idempotency check
				is.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(&IdempotencyResult{Found: false}, nil)

				// Lock acquisition
				dl.On("AcquireLock", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(true, nil)
				dl.On("ReleaseLock", mock.Anything, mock.AnythingOfType("string")).Return(nil)

				// Wallet operations
				wr.On("GetWalletByID", mock.Anything, mock.AnythingOfType("primitive.ObjectID")).Return(&models.Wallet{
					ID:     primitive.NewObjectID(),
					UserID: 12345,
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(50.0),
						Available: decimal.NewFromFloat(50.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
					Status: "active",
				}, nil)
				wr.On("UpdateBalance", mock.Anything, mock.AnythingOfType("primitive.ObjectID"), mock.AnythingOfType("*models.WalletBalance")).Return(nil)

				// Transaction operations
				tr.On("CreateTransaction", mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil)
				tr.On("UpdateTransaction", mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil)

				// Store idempotency result
				is.On("StoreResult", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)
			},
			expectError: false,
		},
		{
			name: "insufficient balance for withdrawal",
			request: &models.TransactionRequest{
				UserID:   12345,
				WalletID: primitive.NewObjectID(),
				Type:     "withdrawal",
				Amount:   decimal.NewFromFloat(100.0),
				Currency: "USD",
				Reference: "test-withdrawal",
			},
			wallet: &models.Wallet{
				ID:     primitive.NewObjectID(),
				UserID: 12345,
				Balance: &models.WalletBalance{
					Total:     decimal.NewFromFloat(50.0),
					Available: decimal.NewFromFloat(50.0),
					Locked:    decimal.Zero,
					Currency:  "USD",
				},
				Status: "active",
			},
			setupMocks: func(wr *MockWalletRepository, tr *MockTransactionRepository, dl *MockDistributedLock, is *MockIdempotencyService) {
				// Idempotency check
				is.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(&IdempotencyResult{Found: false}, nil)

				// Lock acquisition
				dl.On("AcquireLock", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(true, nil)
				dl.On("ReleaseLock", mock.Anything, mock.AnythingOfType("string")).Return(nil)

				// Wallet operations
				wr.On("GetWalletByID", mock.Anything, mock.AnythingOfType("primitive.ObjectID")).Return(&models.Wallet{
					ID:     primitive.NewObjectID(),
					UserID: 12345,
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(50.0),
						Available: decimal.NewFromFloat(50.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
					Status: "active",
				}, nil)

				// Transaction operations (for failed transaction record)
				tr.On("CreateTransaction", mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil)
				tr.On("UpdateTransaction", mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil)
			},
			expectError: true,
			errorMsg:    "insufficient balance",
		},
		{
			name: "idempotent request returns cached result",
			request: &models.TransactionRequest{
				UserID:   12345,
				WalletID: primitive.NewObjectID(),
				Type:     "deposit",
				Amount:   decimal.NewFromFloat(100.0),
				Currency: "USD",
				Reference: "test-deposit",
			},
			setupMocks: func(wr *MockWalletRepository, tr *MockTransactionRepository, dl *MockDistributedLock, is *MockIdempotencyService) {
				// Idempotency check returns cached result
				cachedTransaction := &models.Transaction{
					TransactionID: "TXN-cached-123",
					Status:        "completed",
					Amount: models.TransactionAmount{
						Value: decimal.NewFromFloat(100.0),
						Fee:   decimal.Zero,
						Net:   decimal.NewFromFloat(100.0),
					},
				}
				is.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(&IdempotencyResult{
					Found:  true,
					Result: cachedTransaction,
				}, nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockWalletRepo := &MockWalletRepository{}
			mockTransactionRepo := &MockTransactionRepository{}
			mockDistributedLock := &MockDistributedLock{}
			mockIdempotencyService := &MockIdempotencyService{}

			// Setup mocks
			tt.setupMocks(mockWalletRepo, mockTransactionRepo, mockDistributedLock, mockIdempotencyService)

			// Create transaction engine
			engine := &transactionEngine{
				walletRepo:         mockWalletRepo,
				transactionRepo:    mockTransactionRepo,
				distributedLock:    mockDistributedLock,
				idempotencyService: mockIdempotencyService,
				config: &TransactionEngineConfig{
					DefaultLockTimeout: 30 * time.Second,
					MaxRetries:         3,
					RetryDelay:         100 * time.Millisecond,
				},
			}

			// Execute test
			ctx := context.Background()
			result, err := engine.ProcessTransaction(ctx, tt.request)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.TransactionID)
			}

			// Verify all expectations were met
			mockWalletRepo.AssertExpectations(t)
			mockTransactionRepo.AssertExpectations(t)
			mockDistributedLock.AssertExpectations(t)
			mockIdempotencyService.AssertExpectations(t)
		})
	}
}

func TestTransactionEngine_ProcessTransfer(t *testing.T) {
	fromWalletID := primitive.NewObjectID()
	toWalletID := primitive.NewObjectID()

	tests := []struct {
		name        string
		request     *models.TransferRequest
		fromWallet  *models.Wallet
		toWallet    *models.Wallet
		setupMocks  func(*MockWalletRepository, *MockTransactionRepository, *MockDistributedLock, *MockIdempotencyService)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful transfer",
			request: &models.TransferRequest{
				FromWalletID: fromWalletID,
				ToWalletID:   toWalletID,
				Amount:       decimal.NewFromFloat(50.0),
				Currency:     "USD",
				Reference:    "test-transfer",
			},
			fromWallet: &models.Wallet{
				ID:     fromWalletID,
				UserID: 12345,
				Balance: &models.WalletBalance{
					Total:     decimal.NewFromFloat(100.0),
					Available: decimal.NewFromFloat(100.0),
					Locked:    decimal.Zero,
					Currency:  "USD",
				},
				Status: "active",
			},
			toWallet: &models.Wallet{
				ID:     toWalletID,
				UserID: 67890,
				Balance: &models.WalletBalance{
					Total:     decimal.NewFromFloat(25.0),
					Available: decimal.NewFromFloat(25.0),
					Locked:    decimal.Zero,
					Currency:  "USD",
				},
				Status: "active",
			},
			setupMocks: func(wr *MockWalletRepository, tr *MockTransactionRepository, dl *MockDistributedLock, is *MockIdempotencyService) {
				// Idempotency check
				is.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(&IdempotencyResult{Found: false}, nil)

				// Lock acquisition for both wallets
				dl.On("AcquireLock", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(true, nil).Times(2)
				dl.On("ReleaseLock", mock.Anything, mock.AnythingOfType("string")).Return(nil).Times(2)

				// From wallet operations
				fromWallet := &models.Wallet{
					ID:     fromWalletID,
					UserID: 12345,
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(100.0),
						Available: decimal.NewFromFloat(100.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
					Status: "active",
				}
				wr.On("GetWalletByID", mock.Anything, fromWalletID).Return(fromWallet, nil)
				wr.On("UpdateBalance", mock.Anything, fromWalletID, mock.AnythingOfType("*models.WalletBalance")).Return(nil)

				// To wallet operations
				toWallet := &models.Wallet{
					ID:     toWalletID,
					UserID: 67890,
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(25.0),
						Available: decimal.NewFromFloat(25.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
					Status: "active",
				}
				wr.On("GetWalletByID", mock.Anything, toWalletID).Return(toWallet, nil)
				wr.On("UpdateBalance", mock.Anything, toWalletID, mock.AnythingOfType("*models.WalletBalance")).Return(nil)

				// Transaction operations (one for each wallet)
				tr.On("CreateTransaction", mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil).Times(2)
				tr.On("UpdateTransaction", mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil).Times(2)

				// Store idempotency result
				is.On("StoreResult", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)
			},
			expectError: false,
		},
		{
			name: "transfer with insufficient balance",
			request: &models.TransferRequest{
				FromWalletID: fromWalletID,
				ToWalletID:   toWalletID,
				Amount:       decimal.NewFromFloat(150.0),
				Currency:     "USD",
				Reference:    "test-transfer",
			},
			setupMocks: func(wr *MockWalletRepository, tr *MockTransactionRepository, dl *MockDistributedLock, is *MockIdempotencyService) {
				// Idempotency check
				is.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(&IdempotencyResult{Found: false}, nil)

				// Lock acquisition
				dl.On("AcquireLock", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(true, nil)
				dl.On("ReleaseLock", mock.Anything, mock.AnythingOfType("string")).Return(nil)

				// From wallet operations
				fromWallet := &models.Wallet{
					ID:     fromWalletID,
					UserID: 12345,
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(100.0),
						Available: decimal.NewFromFloat(100.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
					Status: "active",
				}
				wr.On("GetWalletByID", mock.Anything, fromWalletID).Return(fromWallet, nil)

				// Failed transaction record
				tr.On("CreateTransaction", mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil)
				tr.On("UpdateTransaction", mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil)
			},
			expectError: true,
			errorMsg:    "insufficient balance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockWalletRepo := &MockWalletRepository{}
			mockTransactionRepo := &MockTransactionRepository{}
			mockDistributedLock := &MockDistributedLock{}
			mockIdempotencyService := &MockIdempotencyService{}

			// Setup mocks
			tt.setupMocks(mockWalletRepo, mockTransactionRepo, mockDistributedLock, mockIdempotencyService)

			// Create transaction engine
			engine := &transactionEngine{
				walletRepo:         mockWalletRepo,
				transactionRepo:    mockTransactionRepo,
				distributedLock:    mockDistributedLock,
				idempotencyService: mockIdempotencyService,
				config: &TransactionEngineConfig{
					DefaultLockTimeout: 30 * time.Second,
					MaxRetries:         3,
					RetryDelay:         100 * time.Millisecond,
				},
			}

			// Execute test
			ctx := context.Background()
			result, err := engine.ProcessTransfer(ctx, tt.request)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			// Verify all expectations were met
			mockWalletRepo.AssertExpectations(t)
			mockTransactionRepo.AssertExpectations(t)
			mockDistributedLock.AssertExpectations(t)
			mockIdempotencyService.AssertExpectations(t)
		})
	}
}

func TestTransactionEngine_LockFunds(t *testing.T) {
	walletID := primitive.NewObjectID()

	tests := []struct {
		name        string
		request     *models.FundsLockRequest
		wallet      *models.Wallet
		setupMocks  func(*MockWalletRepository, *MockDistributedLock)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful funds lock",
			request: &models.FundsLockRequest{
				WalletID:  walletID,
				Amount:    decimal.NewFromFloat(50.0),
				Reference: "order-123",
				ExpiresAt: time.Now().Add(30 * time.Minute),
			},
			wallet: &models.Wallet{
				ID:     walletID,
				UserID: 12345,
				Balance: &models.WalletBalance{
					Total:     decimal.NewFromFloat(100.0),
					Available: decimal.NewFromFloat(100.0),
					Locked:    decimal.Zero,
					Currency:  "USD",
				},
				Status: "active",
			},
			setupMocks: func(wr *MockWalletRepository, dl *MockDistributedLock) {
				// Lock acquisition
				dl.On("AcquireLock", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(true, nil)
				dl.On("ReleaseLock", mock.Anything, mock.AnythingOfType("string")).Return(nil)

				// Wallet operations
				wallet := &models.Wallet{
					ID:     walletID,
					UserID: 12345,
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(100.0),
						Available: decimal.NewFromFloat(100.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
					Status: "active",
					Locks:  []models.FundsLock{},
				}
				wr.On("GetWalletByID", mock.Anything, walletID).Return(wallet, nil)
				wr.On("AddFundsLock", mock.Anything, walletID, mock.AnythingOfType("*models.FundsLock")).Return(nil)
				wr.On("UpdateBalance", mock.Anything, walletID, mock.AnythingOfType("*models.WalletBalance")).Return(nil)
			},
			expectError: false,
		},
		{
			name: "insufficient available balance for lock",
			request: &models.FundsLockRequest{
				WalletID:  walletID,
				Amount:    decimal.NewFromFloat(150.0),
				Reference: "order-123",
				ExpiresAt: time.Now().Add(30 * time.Minute),
			},
			setupMocks: func(wr *MockWalletRepository, dl *MockDistributedLock) {
				// Lock acquisition
				dl.On("AcquireLock", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(true, nil)
				dl.On("ReleaseLock", mock.Anything, mock.AnythingOfType("string")).Return(nil)

				// Wallet operations
				wallet := &models.Wallet{
					ID:     walletID,
					UserID: 12345,
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(100.0),
						Available: decimal.NewFromFloat(100.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
					Status: "active",
					Locks:  []models.FundsLock{},
				}
				wr.On("GetWalletByID", mock.Anything, walletID).Return(wallet, nil)
			},
			expectError: true,
			errorMsg:    "insufficient available balance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockWalletRepo := &MockWalletRepository{}
			mockDistributedLock := &MockDistributedLock{}

			// Setup mocks
			tt.setupMocks(mockWalletRepo, mockDistributedLock)

			// Create transaction engine
			engine := &transactionEngine{
				walletRepo:      mockWalletRepo,
				distributedLock: mockDistributedLock,
				config: &TransactionEngineConfig{
					DefaultLockTimeout: 30 * time.Second,
				},
			}

			// Execute test
			ctx := context.Background()
			result, err := engine.LockFunds(ctx, tt.request)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.LockID)
				assert.Equal(t, tt.request.Amount, result.Amount)
			}

			// Verify all expectations were met
			mockWalletRepo.AssertExpectations(t)
			mockDistributedLock.AssertExpectations(t)
		})
	}
}

func TestTransactionEngine_HandleRetries(t *testing.T) {
	walletID := primitive.NewObjectID()

	tests := []struct {
		name        string
		setupMocks  func(*MockWalletRepository, *MockDistributedLock)
		expectError bool
		retryCount  int
	}{
		{
			name: "succeeds after retry",
			setupMocks: func(wr *MockWalletRepository, dl *MockDistributedLock) {
				// First attempt fails to acquire lock
				dl.On("AcquireLock", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(false, errors.New("lock contention")).Once()
				// Second attempt succeeds
				dl.On("AcquireLock", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(true, nil).Once()
				dl.On("ReleaseLock", mock.Anything, mock.AnythingOfType("string")).Return(nil)

				// Wallet operations for successful attempt
				wallet := &models.Wallet{
					ID:     walletID,
					UserID: 12345,
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(100.0),
						Available: decimal.NewFromFloat(100.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
					Status: "active",
				}
				wr.On("GetWalletByID", mock.Anything, walletID).Return(wallet, nil)
				wr.On("UpdateBalance", mock.Anything, walletID, mock.AnythingOfType("*models.WalletBalance")).Return(nil)
			},
			expectError: false,
			retryCount:  1,
		},
		{
			name: "fails after max retries",
			setupMocks: func(wr *MockWalletRepository, dl *MockDistributedLock) {
				// All attempts fail to acquire lock
				dl.On("AcquireLock", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(false, errors.New("lock contention")).Times(3)
			},
			expectError: true,
			retryCount:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockWalletRepo := &MockWalletRepository{}
			mockDistributedLock := &MockDistributedLock{}
			mockTransactionRepo := &MockTransactionRepository{}
			mockIdempotencyService := &MockIdempotencyService{}

			// Setup mocks
			tt.setupMocks(mockWalletRepo, mockDistributedLock)

			// Only setup other mocks if we expect success
			if !tt.expectError {
				mockIdempotencyService.On("CheckIdempotency", mock.Anything, mock.AnythingOfType("string")).Return(&IdempotencyResult{Found: false}, nil)
				mockTransactionRepo.On("CreateTransaction", mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil)
				mockTransactionRepo.On("UpdateTransaction", mock.Anything, mock.AnythingOfType("*models.Transaction")).Return(nil)
				mockIdempotencyService.On("StoreResult", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)
			}

			// Create transaction engine with short retry delay for testing
			engine := &transactionEngine{
				walletRepo:         mockWalletRepo,
				transactionRepo:    mockTransactionRepo,
				distributedLock:    mockDistributedLock,
				idempotencyService: mockIdempotencyService,
				config: &TransactionEngineConfig{
					DefaultLockTimeout: 30 * time.Second,
					MaxRetries:         3,
					RetryDelay:         1 * time.Millisecond, // Very short for testing
				},
			}

			// Create a simple deposit request
			request := &models.TransactionRequest{
				UserID:    12345,
				WalletID:  walletID,
				Type:      "deposit",
				Amount:    decimal.NewFromFloat(100.0),
				Currency:  "USD",
				Reference: "test-retry",
			}

			// Execute test
			ctx := context.Background()
			_, err := engine.ProcessTransaction(ctx, request)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify all expectations were met
			mockWalletRepo.AssertExpectations(t)
			mockDistributedLock.AssertExpectations(t)
			if !tt.expectError {
				mockTransactionRepo.AssertExpectations(t)
				mockIdempotencyService.AssertExpectations(t)
			}
		})
	}
}