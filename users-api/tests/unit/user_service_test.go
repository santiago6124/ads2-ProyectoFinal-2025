package unit

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"users-api/internal/models"
	"users-api/internal/services"
	"users-api/tests/mocks"
)

func TestUserService_CreateUser(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := services.NewUserService(mockRepo)

	t.Run("successful user creation", func(t *testing.T) {
		req := &models.RegisterRequest{
			Username:  "testuser",
			Email:     "test@example.com",
			Password:  "Test123!",
			FirstName: "Test",
			LastName:  "User",
		}

		mockRepo.On("GetByEmail", "test@example.com").Return(nil, fmt.Errorf("user not found")).Once()
		mockRepo.On("GetByUsername", "testuser").Return(nil, fmt.Errorf("user not found")).Once()
		mockRepo.On("Create", mock.AnythingOfType("*models.User")).Return(nil).Once()

		user, err := service.CreateUser(req)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "test@example.com", user.Email)
		assert.NotEmpty(t, user.PasswordHash)
		assert.NotEqual(t, "Test123!", user.PasswordHash)
		assert.Equal(t, models.RoleNormal, user.Role)
		assert.Equal(t, "Test", *user.FirstName)
		assert.Equal(t, "User", *user.LastName)
		mockRepo.AssertExpectations(t)
	})

	t.Run("email already exists", func(t *testing.T) {
		req := &models.RegisterRequest{
			Username: "testuser",
			Email:    "existing@example.com",
			Password: "Test123!",
		}

		existingUser := &models.User{
			ID:    1,
			Email: "existing@example.com",
		}

		mockRepo.On("GetByEmail", "existing@example.com").Return(existingUser, nil).Once()

		user, err := service.CreateUser(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "already exists")
		mockRepo.AssertExpectations(t)
	})

	t.Run("username already exists", func(t *testing.T) {
		req := &models.RegisterRequest{
			Username: "existinguser",
			Email:    "new@example.com",
			Password: "Test123!",
		}

		existingUser := &models.User{
			ID:       1,
			Username: "existinguser",
		}

		mockRepo.On("GetByEmail", "new@example.com").Return(nil, fmt.Errorf("user not found")).Once()
		mockRepo.On("GetByUsername", "existinguser").Return(existingUser, nil).Once()

		user, err := service.CreateUser(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "already exists")
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid password", func(t *testing.T) {
		req := &models.RegisterRequest{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "weak",
		}

		user, err := service.CreateUser(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid password")
	})

	t.Run("invalid email", func(t *testing.T) {
		req := &models.RegisterRequest{
			Username: "testuser",
			Email:    "invalid-email",
			Password: "Test123!",
		}

		user, err := service.CreateUser(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid email")
	})

	t.Run("invalid username", func(t *testing.T) {
		req := &models.RegisterRequest{
			Username: "ab",
			Email:    "test@example.com",
			Password: "Test123!",
		}

		user, err := service.CreateUser(req)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid username")
	})
}

func TestUserService_GetUserByID(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := services.NewUserService(mockRepo)

	t.Run("successful get user", func(t *testing.T) {
		expectedUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsActive: true,
		}

		mockRepo.On("GetByID", uint(1)).Return(expectedUser, nil).Once()

		user, err := service.GetUserByID(1)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Username, user.Username)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo.On("GetByID", uint(999)).Return(nil, fmt.Errorf("user not found")).Once()

		user, err := service.GetUserByID(999)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "not found")
		mockRepo.AssertExpectations(t)
	})

	t.Run("deactivated user", func(t *testing.T) {
		deactivatedUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsActive: false,
		}

		mockRepo.On("GetByID", uint(1)).Return(deactivatedUser, nil).Once()

		user, err := service.GetUserByID(1)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "deactivated")
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := services.NewUserService(mockRepo)

	t.Run("successful user update", func(t *testing.T) {
		existingUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsActive: true,
		}

		firstName := "Updated"
		lastName := "Name"
		req := &models.UpdateUserRequest{
			FirstName: &firstName,
			LastName:  &lastName,
		}

		mockRepo.On("GetByID", uint(1)).Return(existingUser, nil).Once()
		mockRepo.On("Update", mock.AnythingOfType("*models.User")).Return(nil).Once()

		user, err := service.UpdateUser(1, req)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, firstName, *user.FirstName)
		assert.Equal(t, lastName, *user.LastName)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		req := &models.UpdateUserRequest{}

		mockRepo.On("GetByID", uint(999)).Return(nil, fmt.Errorf("user not found")).Once()

		user, err := service.UpdateUser(999, req)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "not found")
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_ChangePassword(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := services.NewUserService(mockRepo)

	t.Run("successful password change", func(t *testing.T) {
		existingUser := &models.User{
			ID:           1,
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdF6Xe5dPaLu3U6", // "Test123!"
			IsActive:     true,
		}

		req := &models.ChangePasswordRequest{
			CurrentPassword: "Test123!",
			NewPassword:     "NewTest456!",
		}

		mockRepo.On("GetByID", uint(1)).Return(existingUser, nil).Once()
		mockRepo.On("Update", mock.AnythingOfType("*models.User")).Return(nil).Once()

		err := service.ChangePassword(1, req)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("incorrect current password", func(t *testing.T) {
		existingUser := &models.User{
			ID:           1,
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdF6Xe5dPaLu3U6", // "Test123!"
			IsActive:     true,
		}

		req := &models.ChangePasswordRequest{
			CurrentPassword: "WrongPassword",
			NewPassword:     "NewTest456!",
		}

		mockRepo.On("GetByID", uint(1)).Return(existingUser, nil).Once()

		err := service.ChangePassword(1, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "incorrect")
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid new password", func(t *testing.T) {
		existingUser := &models.User{
			ID:           1,
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdF6Xe5dPaLu3U6", // "Test123!"
			IsActive:     true,
		}

		req := &models.ChangePasswordRequest{
			CurrentPassword: "Test123!",
			NewPassword:     "weak",
		}

		mockRepo.On("GetByID", uint(1)).Return(existingUser, nil).Once()

		err := service.ChangePassword(1, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
		mockRepo.AssertExpectations(t)
	})
}