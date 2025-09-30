package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"wallet-api/internal/models"
)

// Mock transaction engine for testing
type MockTransactionEngine struct {
	mock.Mock
}

func (m *MockTransactionEngine) ProcessTransaction(ctx context.Context, req *models.TransactionRequest) (*models.Transaction, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.Transaction), args.Error(1)
}

func (m *MockTransactionEngine) ProcessTransfer(ctx context.Context, req *models.TransferRequest) (*models.TransferResult, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.TransferResult), args.Error(1)
}

func (m *MockTransactionEngine) LockFunds(ctx context.Context, req *models.FundsLockRequest) (*models.FundsLock, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.FundsLock), args.Error(1)
}

func (m *MockTransactionEngine) UnlockFunds(ctx context.Context, req *models.FundsUnlockRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockTransactionEngine) ReverseTransaction(ctx context.Context, req *models.TransactionReversalRequest) (*models.Transaction, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.Transaction), args.Error(1)
}

func (m *MockTransactionEngine) GetTransactionStatus(ctx context.Context, transactionID string) (*models.TransactionStatus, error) {
	args := m.Called(ctx, transactionID)
	return args.Get(0).(*models.TransactionStatus), args.Error(1)
}

func (m *MockTransactionEngine) CleanupExpiredLocks(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTransactionEngine) ReconcileWallet(ctx context.Context, walletID primitive.ObjectID) (*models.ReconciliationResult, error) {
	args := m.Called(ctx, walletID)
	return args.Get(0).(*models.ReconciliationResult), args.Error(1)
}

func TestWalletService_CreateWallet(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.CreateWalletRequest
		setupMocks  func(*MockWalletRepository, *MockTransactionEngine)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful wallet creation",
			request: &models.CreateWalletRequest{
				UserID:   12345,
				Currency: "USD",
				Type:     "personal",
			},
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// Check for existing wallet
				wr.On("GetWalletsByUserID", mock.Anything, int64(12345)).Return([]*models.Wallet{}, nil)

				// Create new wallet
				wr.On("CreateWallet", mock.Anything, mock.AnythingOfType("*models.Wallet")).Return(nil)
			},
			expectError: false,
		},
		{
			name: "wallet already exists for user and currency",
			request: &models.CreateWalletRequest{
				UserID:   12345,
				Currency: "USD",
				Type:     "personal",
			},
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// Return existing wallet
				existingWallet := &models.Wallet{
					ID:       primitive.NewObjectID(),
					UserID:   12345,
					Currency: "USD",
					Type:     "personal",
					Status:   "active",
				}
				wr.On("GetWalletsByUserID", mock.Anything, int64(12345)).Return([]*models.Wallet{existingWallet}, nil)
			},
			expectError: true,
			errorMsg:    "wallet already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockWalletRepo := &MockWalletRepository{}
			mockTransactionEngine := &MockTransactionEngine{}

			// Setup mocks
			tt.setupMocks(mockWalletRepo, mockTransactionEngine)

			// Create wallet service
			service := &walletService{
				walletRepo:        mockWalletRepo,
				transactionEngine: mockTransactionEngine,
			}

			// Execute test
			ctx := context.Background()
			result, err := service.CreateWallet(ctx, tt.request)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.request.UserID, result.UserID)
				assert.Equal(t, tt.request.Currency, result.Currency)
				assert.Equal(t, tt.request.Type, result.Type)
				assert.Equal(t, "active", result.Status)
			}

			// Verify all expectations were met
			mockWalletRepo.AssertExpectations(t)
			mockTransactionEngine.AssertExpectations(t)
		})
	}
}

func TestWalletService_Deposit(t *testing.T) {
	walletID := primitive.NewObjectID()

	tests := []struct {
		name        string
		request     *models.DepositRequest
		wallet      *models.Wallet
		setupMocks  func(*MockWalletRepository, *MockTransactionEngine)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful deposit",
			request: &models.DepositRequest{
				WalletID:  walletID,
				Amount:    decimal.NewFromFloat(100.0),
				Currency:  "USD",
				Reference: "test-deposit",
			},
			wallet: &models.Wallet{
				ID:       walletID,
				UserID:   12345,
				Currency: "USD",
				Status:   "active",
				Balance: &models.WalletBalance{
					Total:     decimal.NewFromFloat(50.0),
					Available: decimal.NewFromFloat(50.0),
					Locked:    decimal.Zero,
					Currency:  "USD",
				},
			},
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// Get wallet
				wallet := &models.Wallet{
					ID:       walletID,
					UserID:   12345,
					Currency: "USD",
					Status:   "active",
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(50.0),
						Available: decimal.NewFromFloat(50.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
				}
				wr.On("GetWalletByID", mock.Anything, walletID).Return(wallet, nil)

				// Process transaction
				transaction := &models.Transaction{
					TransactionID: "TXN-deposit-123",
					UserID:        12345,
					WalletID:      walletID,
					Type:          "deposit",
					Status:        "completed",
					Amount: models.TransactionAmount{
						Value: decimal.NewFromFloat(100.0),
						Fee:   decimal.Zero,
						Net:   decimal.NewFromFloat(100.0),
					},
					Currency:  "USD",
					Reference: "test-deposit",
					CreatedAt: time.Now(),
				}
				te.On("ProcessTransaction", mock.Anything, mock.AnythingOfType("*models.TransactionRequest")).Return(transaction, nil)
			},
			expectError: false,
		},
		{
			name: "wallet not found",
			request: &models.DepositRequest{
				WalletID:  walletID,
				Amount:    decimal.NewFromFloat(100.0),
				Currency:  "USD",
				Reference: "test-deposit",
			},
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// Wallet not found
				wr.On("GetWalletByID", mock.Anything, walletID).Return((*models.Wallet)(nil), assert.AnError)
			},
			expectError: true,
			errorMsg:    "wallet not found",
		},
		{
			name: "inactive wallet",
			request: &models.DepositRequest{
				WalletID:  walletID,
				Amount:    decimal.NewFromFloat(100.0),
				Currency:  "USD",
				Reference: "test-deposit",
			},
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// Return inactive wallet
				wallet := &models.Wallet{
					ID:       walletID,
					UserID:   12345,
					Currency: "USD",
					Status:   "suspended",
				}
				wr.On("GetWalletByID", mock.Anything, walletID).Return(wallet, nil)
			},
			expectError: true,
			errorMsg:    "wallet is not active",
		},
		{
			name: "currency mismatch",
			request: &models.DepositRequest{
				WalletID:  walletID,
				Amount:    decimal.NewFromFloat(100.0),
				Currency:  "EUR",
				Reference: "test-deposit",
			},
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// Return wallet with different currency
				wallet := &models.Wallet{
					ID:       walletID,
					UserID:   12345,
					Currency: "USD",
					Status:   "active",
				}
				wr.On("GetWalletByID", mock.Anything, walletID).Return(wallet, nil)
			},
			expectError: true,
			errorMsg:    "currency mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockWalletRepo := &MockWalletRepository{}
			mockTransactionEngine := &MockTransactionEngine{}

			// Setup mocks
			tt.setupMocks(mockWalletRepo, mockTransactionEngine)

			// Create wallet service
			service := &walletService{
				walletRepo:        mockWalletRepo,
				transactionEngine: mockTransactionEngine,
			}

			// Execute test
			ctx := context.Background()
			result, err := service.Deposit(ctx, tt.request)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "deposit", result.Type)
				assert.Equal(t, tt.request.Amount, result.Amount.Value)
			}

			// Verify all expectations were met
			mockWalletRepo.AssertExpectations(t)
			mockTransactionEngine.AssertExpectations(t)
		})
	}
}

func TestWalletService_Withdraw(t *testing.T) {
	walletID := primitive.NewObjectID()

	tests := []struct {
		name        string
		request     *models.WithdrawRequest
		setupMocks  func(*MockWalletRepository, *MockTransactionEngine)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful withdrawal",
			request: &models.WithdrawRequest{
				WalletID:  walletID,
				Amount:    decimal.NewFromFloat(50.0),
				Currency:  "USD",
				Reference: "test-withdrawal",
			},
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// Get wallet
				wallet := &models.Wallet{
					ID:       walletID,
					UserID:   12345,
					Currency: "USD",
					Status:   "active",
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(100.0),
						Available: decimal.NewFromFloat(100.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
				}
				wr.On("GetWalletByID", mock.Anything, walletID).Return(wallet, nil)

				// Process transaction
				transaction := &models.Transaction{
					TransactionID: "TXN-withdrawal-123",
					UserID:        12345,
					WalletID:      walletID,
					Type:          "withdrawal",
					Status:        "completed",
					Amount: models.TransactionAmount{
						Value: decimal.NewFromFloat(-50.0),
						Fee:   decimal.NewFromFloat(1.0),
						Net:   decimal.NewFromFloat(-51.0),
					},
					Currency:  "USD",
					Reference: "test-withdrawal",
					CreatedAt: time.Now(),
				}
				te.On("ProcessTransaction", mock.Anything, mock.AnythingOfType("*models.TransactionRequest")).Return(transaction, nil)
			},
			expectError: false,
		},
		{
			name: "insufficient balance",
			request: &models.WithdrawRequest{
				WalletID:  walletID,
				Amount:    decimal.NewFromFloat(150.0),
				Currency:  "USD",
				Reference: "test-withdrawal",
			},
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// Get wallet
				wallet := &models.Wallet{
					ID:       walletID,
					UserID:   12345,
					Currency: "USD",
					Status:   "active",
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(100.0),
						Available: decimal.NewFromFloat(100.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
				}
				wr.On("GetWalletByID", mock.Anything, walletID).Return(wallet, nil)

				// Transaction engine returns insufficient balance error
				te.On("ProcessTransaction", mock.Anything, mock.AnythingOfType("*models.TransactionRequest")).Return((*models.Transaction)(nil), assert.AnError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockWalletRepo := &MockWalletRepository{}
			mockTransactionEngine := &MockTransactionEngine{}

			// Setup mocks
			tt.setupMocks(mockWalletRepo, mockTransactionEngine)

			// Create wallet service
			service := &walletService{
				walletRepo:        mockWalletRepo,
				transactionEngine: mockTransactionEngine,
			}

			// Execute test
			ctx := context.Background()
			result, err := service.Withdraw(ctx, tt.request)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "withdrawal", result.Type)
			}

			// Verify all expectations were met
			mockWalletRepo.AssertExpectations(t)
			mockTransactionEngine.AssertExpectations(t)
		})
	}
}

func TestWalletService_Transfer(t *testing.T) {
	fromWalletID := primitive.NewObjectID()
	toWalletID := primitive.NewObjectID()

	tests := []struct {
		name        string
		request     *models.TransferRequest
		setupMocks  func(*MockWalletRepository, *MockTransactionEngine)
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
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// Get from wallet
				fromWallet := &models.Wallet{
					ID:       fromWalletID,
					UserID:   12345,
					Currency: "USD",
					Status:   "active",
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(100.0),
						Available: decimal.NewFromFloat(100.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
				}
				wr.On("GetWalletByID", mock.Anything, fromWalletID).Return(fromWallet, nil)

				// Get to wallet
				toWallet := &models.Wallet{
					ID:       toWalletID,
					UserID:   67890,
					Currency: "USD",
					Status:   "active",
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(25.0),
						Available: decimal.NewFromFloat(25.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
				}
				wr.On("GetWalletByID", mock.Anything, toWalletID).Return(toWallet, nil)

				// Process transfer
				transferResult := &models.TransferResult{
					FromTransaction: &models.Transaction{
						TransactionID: "TXN-transfer-from-123",
						UserID:        12345,
						WalletID:      fromWalletID,
						Type:          "transfer_out",
						Status:        "completed",
						Amount: models.TransactionAmount{
							Value: decimal.NewFromFloat(-50.0),
							Fee:   decimal.Zero,
							Net:   decimal.NewFromFloat(-50.0),
						},
						Currency: "USD",
					},
					ToTransaction: &models.Transaction{
						TransactionID: "TXN-transfer-to-123",
						UserID:        67890,
						WalletID:      toWalletID,
						Type:          "transfer_in",
						Status:        "completed",
						Amount: models.TransactionAmount{
							Value: decimal.NewFromFloat(50.0),
							Fee:   decimal.Zero,
							Net:   decimal.NewFromFloat(50.0),
						},
						Currency: "USD",
					},
				}
				te.On("ProcessTransfer", mock.Anything, mock.AnythingOfType("*models.TransferRequest")).Return(transferResult, nil)
			},
			expectError: false,
		},
		{
			name: "transfer to same wallet",
			request: &models.TransferRequest{
				FromWalletID: fromWalletID,
				ToWalletID:   fromWalletID,
				Amount:       decimal.NewFromFloat(50.0),
				Currency:     "USD",
				Reference:    "test-transfer",
			},
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// No mocks needed as validation should fail early
			},
			expectError: true,
			errorMsg:    "cannot transfer to same wallet",
		},
		{
			name: "currency mismatch",
			request: &models.TransferRequest{
				FromWalletID: fromWalletID,
				ToWalletID:   toWalletID,
				Amount:       decimal.NewFromFloat(50.0),
				Currency:     "USD",
				Reference:    "test-transfer",
			},
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// Get from wallet (USD)
				fromWallet := &models.Wallet{
					ID:       fromWalletID,
					UserID:   12345,
					Currency: "USD",
					Status:   "active",
				}
				wr.On("GetWalletByID", mock.Anything, fromWalletID).Return(fromWallet, nil)

				// Get to wallet (EUR)
				toWallet := &models.Wallet{
					ID:       toWalletID,
					UserID:   67890,
					Currency: "EUR",
					Status:   "active",
				}
				wr.On("GetWalletByID", mock.Anything, toWalletID).Return(toWallet, nil)
			},
			expectError: true,
			errorMsg:    "currency mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockWalletRepo := &MockWalletRepository{}
			mockTransactionEngine := &MockTransactionEngine{}

			// Setup mocks
			tt.setupMocks(mockWalletRepo, mockTransactionEngine)

			// Create wallet service
			service := &walletService{
				walletRepo:        mockWalletRepo,
				transactionEngine: mockTransactionEngine,
			}

			// Execute test
			ctx := context.Background()
			result, err := service.Transfer(ctx, tt.request)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotNil(t, result.FromTransaction)
				assert.NotNil(t, result.ToTransaction)
			}

			// Verify all expectations were met
			mockWalletRepo.AssertExpectations(t)
			mockTransactionEngine.AssertExpectations(t)
		})
	}
}

func TestWalletService_LockFunds(t *testing.T) {
	walletID := primitive.NewObjectID()

	tests := []struct {
		name        string
		request     *models.FundsLockRequest
		setupMocks  func(*MockWalletRepository, *MockTransactionEngine)
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
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// Get wallet
				wallet := &models.Wallet{
					ID:       walletID,
					UserID:   12345,
					Currency: "USD",
					Status:   "active",
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(100.0),
						Available: decimal.NewFromFloat(100.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
				}
				wr.On("GetWalletByID", mock.Anything, walletID).Return(wallet, nil)

				// Lock funds
				fundsLock := &models.FundsLock{
					LockID:    "LOCK-123",
					Amount:    decimal.NewFromFloat(50.0),
					Reference: "order-123",
					Status:    "active",
					ExpiresAt: time.Now().Add(30 * time.Minute),
					CreatedAt: time.Now(),
				}
				te.On("LockFunds", mock.Anything, mock.AnythingOfType("*models.FundsLockRequest")).Return(fundsLock, nil)
			},
			expectError: false,
		},
		{
			name: "insufficient available balance",
			request: &models.FundsLockRequest{
				WalletID:  walletID,
				Amount:    decimal.NewFromFloat(150.0),
				Reference: "order-123",
				ExpiresAt: time.Now().Add(30 * time.Minute),
			},
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				// Get wallet
				wallet := &models.Wallet{
					ID:       walletID,
					UserID:   12345,
					Currency: "USD",
					Status:   "active",
					Balance: &models.WalletBalance{
						Total:     decimal.NewFromFloat(100.0),
						Available: decimal.NewFromFloat(100.0),
						Locked:    decimal.Zero,
						Currency:  "USD",
					},
				}
				wr.On("GetWalletByID", mock.Anything, walletID).Return(wallet, nil)

				// Transaction engine returns insufficient balance error
				te.On("LockFunds", mock.Anything, mock.AnythingOfType("*models.FundsLockRequest")).Return((*models.FundsLock)(nil), assert.AnError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockWalletRepo := &MockWalletRepository{}
			mockTransactionEngine := &MockTransactionEngine{}

			// Setup mocks
			tt.setupMocks(mockWalletRepo, mockTransactionEngine)

			// Create wallet service
			service := &walletService{
				walletRepo:        mockWalletRepo,
				transactionEngine: mockTransactionEngine,
			}

			// Execute test
			ctx := context.Background()
			result, err := service.LockFunds(ctx, tt.request)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.request.Amount, result.Amount)
				assert.Equal(t, tt.request.Reference, result.Reference)
			}

			// Verify all expectations were met
			mockWalletRepo.AssertExpectations(t)
			mockTransactionEngine.AssertExpectations(t)
		})
	}
}

func TestWalletService_GetWalletBalance(t *testing.T) {
	walletID := primitive.NewObjectID()

	tests := []struct {
		name        string
		walletID    primitive.ObjectID
		setupMocks  func(*MockWalletRepository, *MockTransactionEngine)
		expectError bool
		errorMsg    string
		expectedBalance *models.WalletBalance
	}{
		{
			name:     "successful balance retrieval",
			walletID: walletID,
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				balance := &models.WalletBalance{
					Total:     decimal.NewFromFloat(100.0),
					Available: decimal.NewFromFloat(75.0),
					Locked:    decimal.NewFromFloat(25.0),
					Currency:  "USD",
					UpdatedAt: time.Now(),
				}
				wr.On("GetWalletBalance", mock.Anything, walletID).Return(balance, nil)
			},
			expectError: false,
			expectedBalance: &models.WalletBalance{
				Total:     decimal.NewFromFloat(100.0),
				Available: decimal.NewFromFloat(75.0),
				Locked:    decimal.NewFromFloat(25.0),
				Currency:  "USD",
			},
		},
		{
			name:     "wallet not found",
			walletID: walletID,
			setupMocks: func(wr *MockWalletRepository, te *MockTransactionEngine) {
				wr.On("GetWalletBalance", mock.Anything, walletID).Return((*models.WalletBalance)(nil), assert.AnError)
			},
			expectError: true,
			errorMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockWalletRepo := &MockWalletRepository{}
			mockTransactionEngine := &MockTransactionEngine{}

			// Setup mocks
			tt.setupMocks(mockWalletRepo, mockTransactionEngine)

			// Create wallet service
			service := &walletService{
				walletRepo:        mockWalletRepo,
				transactionEngine: mockTransactionEngine,
			}

			// Execute test
			ctx := context.Background()
			result, err := service.GetWalletBalance(ctx, tt.walletID)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedBalance.Total, result.Total)
				assert.Equal(t, tt.expectedBalance.Available, result.Available)
				assert.Equal(t, tt.expectedBalance.Locked, result.Locked)
				assert.Equal(t, tt.expectedBalance.Currency, result.Currency)
			}

			// Verify all expectations were met
			mockWalletRepo.AssertExpectations(t)
			mockTransactionEngine.AssertExpectations(t)
		})
	}
}