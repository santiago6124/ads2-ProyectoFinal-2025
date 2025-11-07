package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"users-api/internal/models"
	"users-api/tests/mocks"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestAuthIntegration_Register(t *testing.T) {
	mockUserService := new(mocks.MockUserService)
	_ = mockUserService // TODO: Wire this into the router
	// mockAuthService := new(mocks.MockAuthService)

	router := setupTestRouter()

	t.Run("successful registration", func(t *testing.T) {
		payload := models.RegisterRequest{
			Username:  "newuser",
			Email:     "new@example.com",
			Password:  "SecurePass123!",
			FirstName: "New",
			LastName:  "User",
		}

		expectedUser := &models.User{
			ID:       1,
			Username: "newuser",
			Email:    "new@example.com",
			Role:     models.RoleNormal,
		}

		mockUserService.On("CreateUser", &payload).Return(expectedUser, nil).Once()

		body, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/users/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		assert.Equal(t, 201, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.True(t, response["success"].(bool))
		assert.NotNil(t, response["data"])
		mockUserService.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		payload := map[string]string{
			"username": "ab",
			"email":    "invalid-email",
			"password": "weak",
		}

		body, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/users/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.False(t, response["success"].(bool))
		assert.NotNil(t, response["error"])
	})
}

func TestAuthIntegration_Login(t *testing.T) {
	// mockUserService := new(mocks.MockUserService)
	mockAuthService := new(mocks.MockAuthService)
	_ = mockAuthService // TODO: Wire this into the router

	router := setupTestRouter()

	t.Run("successful login", func(t *testing.T) {
		payload := models.LoginRequest{
			Email:    "test@example.com",
			Password: "SecurePass123!",
		}

		expectedAuth := &models.AuthResponse{
			User: &models.User{
				ID:       1,
				Username: "testuser",
				Email:    "test@example.com",
				Role:     models.RoleNormal,
			},
			AccessToken:  "access_token",
			RefreshToken: "refresh_token",
			ExpiresIn:    3600,
		}

		mockAuthService.On("Authenticate", "test@example.com", "SecurePass123!", "192.0.2.1", "").Return(expectedAuth, nil).Once()

		body, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/users/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.True(t, response["success"].(bool))
		assert.NotNil(t, response["data"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "access_token", data["access_token"])
		assert.Equal(t, "refresh_token", data["refresh_token"])
		mockAuthService.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		payload := models.LoginRequest{
			Email:    "test@example.com",
			Password: "WrongPassword",
		}

		mockAuthService.On("Authenticate", "test@example.com", "WrongPassword", "192.0.2.1", "").Return(nil, assert.AnError).Once()

		body, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/users/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		assert.Equal(t, 500, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.False(t, response["success"].(bool))
		assert.NotNil(t, response["error"])
		mockAuthService.AssertExpectations(t)
	})
}

func TestAuthIntegration_RefreshToken(t *testing.T) {
	// mockUserService := new(mocks.MockUserService)
	mockAuthService := new(mocks.MockAuthService)
	_ = mockAuthService // TODO: Wire this into the router

	router := setupTestRouter()

	t.Run("successful token refresh", func(t *testing.T) {
		payload := models.RefreshTokenRequest{
			RefreshToken: "valid_refresh_token",
		}

		expectedTokenPair := &models.TokenPair{
			AccessToken:  "new_access_token",
			RefreshToken: "new_refresh_token",
			ExpiresIn:    3600,
		}

		mockAuthService.On("RefreshToken", "valid_refresh_token").Return(expectedTokenPair, nil).Once()

		body, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/users/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.True(t, response["success"].(bool))
		assert.NotNil(t, response["data"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "new_access_token", data["access_token"])
		assert.Equal(t, float64(3600), data["expires_in"])
		mockAuthService.AssertExpectations(t)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		payload := models.RefreshTokenRequest{
			RefreshToken: "invalid_refresh_token",
		}

		mockAuthService.On("RefreshToken", "invalid_refresh_token").Return(nil, assert.AnError).Once()

		body, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/users/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		assert.Equal(t, 500, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.False(t, response["success"].(bool))
		assert.NotNil(t, response["error"])
		mockAuthService.AssertExpectations(t)
	})
}