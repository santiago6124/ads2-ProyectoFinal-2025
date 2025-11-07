package mocks

import (
	"github.com/stretchr/testify/mock"
	"users-api/internal/models"
)

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateTokenPair(user *models.User) (*models.TokenPair, error) {
	args := m.Called(user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TokenPair), args.Error(1)
}

func (m *MockTokenService) ValidateAccessToken(tokenString string) (*models.CustomClaims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CustomClaims), args.Error(1)
}

func (m *MockTokenService) RefreshAccessToken(refreshToken string) (*models.TokenPair, error) {
	args := m.Called(refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TokenPair), args.Error(1)
}

func (m *MockTokenService) RevokeRefreshToken(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockTokenService) RevokeAllUserTokens(userID int32) error {
	args := m.Called(userID)
	return args.Error(0)
}

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(req *models.RegisterRequest) (*models.User, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(id int32) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(id int32, req *models.UpdateUserRequest) (*models.User, error) {
	args := m.Called(id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) ChangePassword(id int32, req *models.ChangePasswordRequest) error {
	args := m.Called(id, req)
	return args.Error(0)
}

func (m *MockUserService) DeactivateUser(id int32) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserService) ListUsers(page, limit int, search, role string, isActive *bool) ([]models.User, int64, error) {
	args := m.Called(page, limit, search, role, isActive)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserService) UpgradeUserToAdmin(id int32) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) VerifyUser(id int32) (*models.UserVerificationResponse, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserVerificationResponse), args.Error(1)
}

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Authenticate(email, password, ipAddress, userAgent string) (*models.AuthResponse, error) {
	args := m.Called(email, password, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthResponse), args.Error(1)
}

func (m *MockAuthService) RefreshToken(refreshToken string) (*models.TokenPair, error) {
	args := m.Called(refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TokenPair), args.Error(1)
}

func (m *MockAuthService) Logout(refreshToken string) error {
	args := m.Called(refreshToken)
	return args.Error(0)
}

func (m *MockAuthService) LogoutAll(userID int32) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockAuthService) IsRateLimited(email string) (bool, error) {
	args := m.Called(email)
	return args.Bool(0), args.Error(1)
}