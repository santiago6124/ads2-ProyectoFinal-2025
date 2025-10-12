package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"users-api/internal/dto"
	"users-api/internal/models"
	"users-api/internal/services"
	"users-api/pkg/utils"
)

type UserController struct {
	userService services.UserService
}

func NewUserController(userService services.UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

// GetUser godoc
// @Summary Get user by ID
// @Description Get user information by ID
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} dto.APIResponse{data=dto.UserResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/users/{id} [get]
func (uc *UserController) GetUser(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		utils.SendValidationError(c, err)
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		utils.SendUnauthorizedError(c, "User not authenticated")
		return
	}

	currentUserRole, exists := c.Get("user_role")
	if !exists {
		utils.SendUnauthorizedError(c, "User role not found")
		return
	}

	if currentUserID.(int32) != int32(id) && currentUserRole.(models.UserRole) != models.RoleAdmin {
		utils.SendForbiddenError(c, "Access denied")
		return
	}

	user, err := uc.userService.GetUserByID(int32(id))
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "deactivated") {
			utils.SendNotFoundError(c, "User")
			return
		}
		utils.SendInternalError(c, err)
		return
	}

	userResponse := dto.ToUserResponse(user)
	utils.SendSuccessResponse(c, http.StatusOK, "", userResponse)
}

// UpdateUser godoc
// @Summary Update user information
// @Description Update user profile information
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body models.UpdateUserRequest true "User update data"
// @Success 200 {object} dto.APIResponse{data=dto.UserResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/users/{id} [put]
func (uc *UserController) UpdateUser(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		utils.SendValidationError(c, err)
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		utils.SendUnauthorizedError(c, "User not authenticated")
		return
	}

	currentUserRole, exists := c.Get("user_role")
	if !exists {
		utils.SendUnauthorizedError(c, "User role not found")
		return
	}

	if currentUserID.(int32) != int32(id) && currentUserRole.(models.UserRole) != models.RoleAdmin {
		utils.SendForbiddenError(c, "Access denied")
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, err)
		return
	}

	user, err := uc.userService.UpdateUser(int32(id), &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "deactivated") {
			utils.SendNotFoundError(c, "User")
			return
		}
		utils.SendInternalError(c, err)
		return
	}

	userResponse := dto.ToUserResponse(user)
	utils.SendSuccessResponse(c, http.StatusOK, "User updated successfully", userResponse)
}

// ChangePassword godoc
// @Summary Change user password
// @Description Change user password
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body models.ChangePasswordRequest true "Password change data"
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/users/{id}/password [put]
func (uc *UserController) ChangePassword(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		utils.SendValidationError(c, err)
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		utils.SendUnauthorizedError(c, "User not authenticated")
		return
	}

	if currentUserID.(int32) != int32(id) {
		utils.SendForbiddenError(c, "Access denied")
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, err)
		return
	}

	if err := uc.userService.ChangePassword(int32(id), &req); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "deactivated") {
			utils.SendNotFoundError(c, "User")
			return
		}
		if strings.Contains(err.Error(), "incorrect") || strings.Contains(err.Error(), "invalid") {
			utils.SendValidationError(c, err)
			return
		}
		utils.SendInternalError(c, err)
		return
	}

	utils.SendSuccessResponse(c, http.StatusOK, "Password updated successfully", nil)
}

// DeleteUser godoc
// @Summary Deactivate user account
// @Description Deactivate user account (soft delete)
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/users/{id} [delete]
func (uc *UserController) DeleteUser(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		utils.SendValidationError(c, err)
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		utils.SendUnauthorizedError(c, "User not authenticated")
		return
	}

	currentUserRole, exists := c.Get("user_role")
	if !exists {
		utils.SendUnauthorizedError(c, "User role not found")
		return
	}

	if currentUserID.(int32) != int32(id) && currentUserRole.(models.UserRole) != models.RoleAdmin {
		utils.SendForbiddenError(c, "Access denied")
		return
	}

	if err := uc.userService.DeactivateUser(int32(id)); err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.SendNotFoundError(c, "User")
			return
		}
		utils.SendInternalError(c, err)
		return
	}

	utils.SendSuccessResponse(c, http.StatusOK, "User deactivated successfully", nil)
}

// ListUsers godoc
// @Summary List all users (Admin only)
// @Description Get paginated list of users with filters
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param search query string false "Search by username or email"
// @Param role query string false "Filter by role (normal/admin)"
// @Param is_active query bool false "Filter by active status"
// @Success 200 {object} dto.APIResponse{data=dto.UserListResponse}
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/users [get]
func (uc *UserController) ListUsers(c *gin.Context) {
	currentUserRole, exists := c.Get("user_role")
	if !exists {
		utils.SendUnauthorizedError(c, "User role not found")
		return
	}

	if currentUserRole.(models.UserRole) != models.RoleAdmin {
		utils.SendForbiddenError(c, "Admin access required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	role := c.Query("role")
	isActiveStr := c.Query("is_active")

	var isActive *bool
	if isActiveStr != "" {
		if isActiveBool, err := strconv.ParseBool(isActiveStr); err == nil {
			isActive = &isActiveBool
		}
	}

	users, total, err := uc.userService.ListUsers(page, limit, search, role, isActive)
	if err != nil {
		utils.SendInternalError(c, err)
		return
	}

	userListResponse := dto.ToUserListResponse(users, total, page, limit)
	utils.SendSuccessResponse(c, http.StatusOK, "", userListResponse)
}

// UpgradeUser godoc
// @Summary Upgrade user to admin (Admin only)
// @Description Upgrade user role to admin
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} dto.APIResponse{data=dto.UserResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/users/{id}/upgrade [post]
func (uc *UserController) UpgradeUser(c *gin.Context) {
	currentUserRole, exists := c.Get("user_role")
	if !exists {
		utils.SendUnauthorizedError(c, "User role not found")
		return
	}

	if currentUserRole.(models.UserRole) != models.RoleAdmin {
		utils.SendForbiddenError(c, "Admin access required")
		return
	}

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		utils.SendValidationError(c, err)
		return
	}

	user, err := uc.userService.UpgradeUserToAdmin(int32(id))
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "deactivated") {
			utils.SendNotFoundError(c, "User")
			return
		}
		if strings.Contains(err.Error(), "already") {
			utils.SendValidationError(c, err)
			return
		}
		utils.SendInternalError(c, err)
		return
	}

	userResponse := dto.ToUserResponse(user)
	utils.SendSuccessResponse(c, http.StatusOK, "User promoted to admin successfully", userResponse)
}

// VerifyUser godoc
// @Summary Verify user existence (Internal use)
// @Description Verify if user exists and is active (for other microservices)
// @Tags internal
// @Produce json
// @Param id path int true "User ID"
// @Param X-Internal-Service header string true "Service name"
// @Param X-API-Key header string true "Internal API key"
// @Success 200 {object} models.UserVerificationResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/users/{id}/verify [get]
func (uc *UserController) VerifyUser(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		utils.SendValidationError(c, err)
		return
	}

	verification, err := uc.userService.VerifyUser(int32(id))
	if err != nil {
		utils.SendInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, verification)
}