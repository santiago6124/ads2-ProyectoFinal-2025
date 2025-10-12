package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"users-api/internal/dto"
	"users-api/internal/models"
	"users-api/internal/services"
	"users-api/pkg/utils"
)

type AuthController struct {
	authService services.AuthService
	userService services.UserService
}

func NewAuthController(authService services.AuthService, userService services.UserService) *AuthController {
	return &AuthController{
		authService: authService,
		userService: userService,
	}
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Registration details"
// @Success 201 {object} dto.APIResponse{data=dto.UserResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/users/register [post]
func (ac *AuthController) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, err)
		return
	}

	user, err := ac.userService.CreateUser(&req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			utils.SendConflictError(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid") {
			utils.SendValidationError(c, err)
			return
		}
		utils.SendInternalError(c, err)
		return
	}

	userResponse := dto.ToUserResponse(user)
	utils.SendSuccessResponse(c, http.StatusCreated, "User created successfully", userResponse)
}

// Login godoc
// @Summary Authenticate user
// @Description Authenticate user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} dto.APIResponse{data=dto.LoginResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 429 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/users/login [post]
func (ac *AuthController) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, err)
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	authResponse, err := ac.authService.Authenticate(req.Email, req.Password, ipAddress, userAgent)
	if err != nil {
		if strings.Contains(err.Error(), "too many failed") {
			utils.SendTooManyRequestsError(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "deactivated") {
			utils.SendUnauthorizedError(c, err.Error())
			return
		}
		utils.SendInternalError(c, err)
		return
	}

	loginResponse := dto.ToLoginResponse(authResponse)
	utils.SendSuccessResponse(c, http.StatusOK, "Login successful", loginResponse)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Generate a new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} dto.APIResponse{data=dto.RefreshResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/users/refresh [post]
func (ac *AuthController) RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, err)
		return
	}

	tokenPair, err := ac.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "expired") || strings.Contains(err.Error(), "revoked") {
			utils.SendUnauthorizedError(c, err.Error())
			return
		}
		utils.SendInternalError(c, err)
		return
	}

	refreshResponse := dto.RefreshResponse{
		AccessToken: tokenPair.AccessToken,
		ExpiresIn:   tokenPair.ExpiresIn,
	}

	utils.SendSuccessResponse(c, http.StatusOK, "Token refreshed successfully", refreshResponse)
}

// Logout godoc
// @Summary Logout user
// @Description Revoke refresh token and logout user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/users/logout [post]
func (ac *AuthController) Logout(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, err)
		return
	}

	if err := ac.authService.Logout(req.RefreshToken); err != nil {
		utils.SendInternalError(c, err)
		return
	}

	utils.SendSuccessResponse(c, http.StatusOK, "Logout successful", nil)
}

// LogoutAll godoc
// @Summary Logout from all devices
// @Description Revoke all refresh tokens for the user
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} dto.APIResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/users/logout-all [post]
func (ac *AuthController) LogoutAll(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.SendUnauthorizedError(c, "User not authenticated")
		return
	}

	if err := ac.authService.LogoutAll(userID.(int32)); err != nil {
		utils.SendInternalError(c, err)
		return
	}

	utils.SendSuccessResponse(c, http.StatusOK, "Logged out from all devices successfully", nil)
}