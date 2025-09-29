package mocks

import (
	"time"

	"github.com/stretchr/testify/mock"
	"users-api/internal/models"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id uint) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) List(offset, limit int, search string, role string, isActive *bool) ([]models.User, int64, error) {
	args := m.Called(offset, limit, search, role, isActive)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) UpdateLastLogin(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) Exists(id uint) (bool, error) {
	args := m.Called(id)
	return args.Bool(0), args.Error(1)
}

type MockRefreshTokenRepository struct {
	mock.Mock
}

func (m *MockRefreshTokenRepository) Create(token *models.RefreshToken) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) GetByToken(token string) (*models.RefreshToken, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepository) RevokeByUserID(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) RevokeByToken(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) DeleteExpired() error {
	args := m.Called()
	return args.Error(0)
}

type MockLoginAttemptRepository struct {
	mock.Mock
}

func (m *MockLoginAttemptRepository) Create(attempt *models.LoginAttempt) error {
	args := m.Called(attempt)
	return args.Error(0)
}

func (m *MockLoginAttemptRepository) GetRecentAttempts(email string, since time.Time) ([]models.LoginAttempt, error) {
	args := m.Called(email, since)
	return args.Get(0).([]models.LoginAttempt), args.Error(1)
}

func (m *MockLoginAttemptRepository) CountFailedAttempts(email string, since time.Time) (int64, error) {
	args := m.Called(email, since)
	return args.Get(0).(int64), args.Error(1)
}