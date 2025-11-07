package unit

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"users-api/internal/models"
	"users-api/internal/services"
	"users-api/pkg/utils"
	"users-api/tests/mocks"
)

func TestAuthService_Authenticate(t *testing.T) {
	t.Run("successful authentication", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockLoginAttemptRepo := new(mocks.MockLoginAttemptRepository)
		mockTokenService := new(mocks.MockTokenService)
		service := services.NewAuthService(mockUserRepo, mockLoginAttemptRepo, mockTokenService)

		hashedPassword, _ := utils.HashPassword("Test123!")
		user := &models.User{
			ID:           1,
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: hashedPassword,
			IsActive:     true,
		}

		tokenPair := &models.TokenPair{
			AccessToken:  "access_token",
			RefreshToken: "refresh_token",
			ExpiresIn:    3600,
		}

		mockLoginAttemptRepo.On("CountFailedAttempts", "test@example.com", mock.AnythingOfType("time.Time")).Return(int64(0), nil).Once()
		mockUserRepo.On("GetByEmail", "test@example.com").Return(user, nil).Once()
		mockTokenService.On("GenerateTokenPair", user).Return(tokenPair, nil).Once()
		mockUserRepo.On("UpdateLastLogin", int32(1)).Return(nil).Once()
		mockLoginAttemptRepo.On("Create", mock.AnythingOfType("*models.LoginAttempt")).Return(nil).Once()

		authResponse, err := service.Authenticate("test@example.com", "Test123!", "192.168.1.1", "Mozilla/5.0")

		assert.NoError(t, err)
		assert.NotNil(t, authResponse)
		assert.Equal(t, user, authResponse.User)
		assert.Equal(t, "access_token", authResponse.AccessToken)
		assert.Equal(t, "refresh_token", authResponse.RefreshToken)
		mockUserRepo.AssertExpectations(t)
		mockTokenService.AssertExpectations(t)
		mockLoginAttemptRepo.AssertExpectations(t)
	})

	t.Run("invalid email", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockLoginAttemptRepo := new(mocks.MockLoginAttemptRepository)
		mockTokenService := new(mocks.MockTokenService)
		service := services.NewAuthService(mockUserRepo, mockLoginAttemptRepo, mockTokenService)

		mockLoginAttemptRepo.On("CountFailedAttempts", "notfound@example.com", mock.AnythingOfType("time.Time")).Return(int64(0), nil).Once()
		mockUserRepo.On("GetByEmail", "notfound@example.com").Return(nil, fmt.Errorf("user not found")).Once()
		mockLoginAttemptRepo.On("Create", mock.AnythingOfType("*models.LoginAttempt")).Return(nil).Once()

		authResponse, err := service.Authenticate("notfound@example.com", "Test123!", "192.168.1.1", "Mozilla/5.0")

		assert.Error(t, err)
		assert.Nil(t, authResponse)
		assert.Contains(t, err.Error(), "invalid email or password")
		mockUserRepo.AssertExpectations(t)
		mockLoginAttemptRepo.AssertExpectations(t)
	})

	t.Run("invalid password", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockLoginAttemptRepo := new(mocks.MockLoginAttemptRepository)
		mockTokenService := new(mocks.MockTokenService)
		service := services.NewAuthService(mockUserRepo, mockLoginAttemptRepo, mockTokenService)

		hashedPassword, _ := utils.HashPassword("Test123!")
		user := &models.User{
			ID:           1,
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: hashedPassword,
			IsActive:     true,
		}

		mockLoginAttemptRepo.On("CountFailedAttempts", "test@example.com", mock.AnythingOfType("time.Time")).Return(int64(0), nil).Once()
		mockUserRepo.On("GetByEmail", "test@example.com").Return(user, nil).Once()
		mockLoginAttemptRepo.On("Create", mock.AnythingOfType("*models.LoginAttempt")).Return(nil).Once()

		authResponse, err := service.Authenticate("test@example.com", "WrongPassword", "192.168.1.1", "Mozilla/5.0")

		assert.Error(t, err)
		assert.Nil(t, authResponse)
		assert.Contains(t, err.Error(), "invalid email or password")
		mockUserRepo.AssertExpectations(t)
		mockLoginAttemptRepo.AssertExpectations(t)
	})

	t.Run("deactivated user", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockLoginAttemptRepo := new(mocks.MockLoginAttemptRepository)
		mockTokenService := new(mocks.MockTokenService)
		service := services.NewAuthService(mockUserRepo, mockLoginAttemptRepo, mockTokenService)

		hashedPassword, _ := utils.HashPassword("Test123!")
		user := &models.User{
			ID:           1,
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: hashedPassword,
			IsActive:     false,
		}

		mockLoginAttemptRepo.On("CountFailedAttempts", "test@example.com", mock.AnythingOfType("time.Time")).Return(int64(0), nil).Once()
		mockUserRepo.On("GetByEmail", "test@example.com").Return(user, nil).Once()
		mockLoginAttemptRepo.On("Create", mock.AnythingOfType("*models.LoginAttempt")).Return(nil).Once()

		authResponse, err := service.Authenticate("test@example.com", "Test123!", "192.168.1.1", "Mozilla/5.0")

		assert.Error(t, err)
		assert.Nil(t, authResponse)
		assert.Contains(t, err.Error(), "deactivated")
		mockUserRepo.AssertExpectations(t)
		mockLoginAttemptRepo.AssertExpectations(t)
	})

	t.Run("rate limited", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockLoginAttemptRepo := new(mocks.MockLoginAttemptRepository)
		mockTokenService := new(mocks.MockTokenService)
		service := services.NewAuthService(mockUserRepo, mockLoginAttemptRepo, mockTokenService)

		mockLoginAttemptRepo.On("CountFailedAttempts", "test@example.com", mock.AnythingOfType("time.Time")).Return(int64(5), nil).Once()
		mockLoginAttemptRepo.On("Create", mock.AnythingOfType("*models.LoginAttempt")).Return(nil).Once()

		authResponse, err := service.Authenticate("test@example.com", "Test123!", "192.168.1.1", "Mozilla/5.0")

		assert.Error(t, err)
		assert.Nil(t, authResponse)
		assert.Contains(t, err.Error(), "too many failed")
		mockLoginAttemptRepo.AssertExpectations(t)
	})
}

func TestAuthService_IsRateLimited(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockLoginAttemptRepo := new(mocks.MockLoginAttemptRepository)
	mockTokenService := new(mocks.MockTokenService)

	service := services.NewAuthService(mockUserRepo, mockLoginAttemptRepo, mockTokenService)

	t.Run("not rate limited", func(t *testing.T) {
		mockLoginAttemptRepo.On("CountFailedAttempts", "test@example.com", mock.AnythingOfType("time.Time")).Return(int64(2), nil).Once()

		limited, err := service.IsRateLimited("test@example.com")

		assert.NoError(t, err)
		assert.False(t, limited)
		mockLoginAttemptRepo.AssertExpectations(t)
	})

	t.Run("rate limited", func(t *testing.T) {
		mockLoginAttemptRepo.On("CountFailedAttempts", "test@example.com", mock.AnythingOfType("time.Time")).Return(int64(5), nil).Once()

		limited, err := service.IsRateLimited("test@example.com")

		assert.NoError(t, err)
		assert.True(t, limited)
		mockLoginAttemptRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockLoginAttemptRepo.On("CountFailedAttempts", "test@example.com", mock.AnythingOfType("time.Time")).Return(int64(0), fmt.Errorf("database error")).Once()

		limited, err := service.IsRateLimited("test@example.com")

		assert.Error(t, err)
		assert.False(t, limited)
		mockLoginAttemptRepo.AssertExpectations(t)
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockLoginAttemptRepo := new(mocks.MockLoginAttemptRepository)
	mockTokenService := new(mocks.MockTokenService)

	service := services.NewAuthService(mockUserRepo, mockLoginAttemptRepo, mockTokenService)

	t.Run("successful token refresh", func(t *testing.T) {
		tokenPair := &models.TokenPair{
			AccessToken:  "new_access_token",
			RefreshToken: "new_refresh_token",
			ExpiresIn:    3600,
		}

		mockTokenService.On("RefreshAccessToken", "refresh_token").Return(tokenPair, nil).Once()

		result, err := service.RefreshToken("refresh_token")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "new_access_token", result.AccessToken)
		assert.Equal(t, "new_refresh_token", result.RefreshToken)
		mockTokenService.AssertExpectations(t)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		mockTokenService.On("RefreshAccessToken", "invalid_token").Return(nil, fmt.Errorf("invalid refresh token")).Once()

		result, err := service.RefreshToken("invalid_token")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to refresh token")
		mockTokenService.AssertExpectations(t)
	})
}

func TestAuthService_Logout(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	mockLoginAttemptRepo := new(mocks.MockLoginAttemptRepository)
	mockTokenService := new(mocks.MockTokenService)

	service := services.NewAuthService(mockUserRepo, mockLoginAttemptRepo, mockTokenService)

	t.Run("successful logout", func(t *testing.T) {
		mockTokenService.On("RevokeRefreshToken", "refresh_token").Return(nil).Once()

		err := service.Logout("refresh_token")

		assert.NoError(t, err)
		mockTokenService.AssertExpectations(t)
	})

	t.Run("token service error", func(t *testing.T) {
		mockTokenService.On("RevokeRefreshToken", "refresh_token").Return(fmt.Errorf("database error")).Once()

		err := service.Logout("refresh_token")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to logout")
		mockTokenService.AssertExpectations(t)
	})
}