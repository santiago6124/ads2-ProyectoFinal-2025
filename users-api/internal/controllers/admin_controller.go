package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"users-api/internal/models"
	"users-api/internal/services"
)

type AdminController struct {
	userService services.UserService
}

func NewAdminController(userService services.UserService) *AdminController {
	return &AdminController{
		userService: userService,
	}
}

// GetAllUsers returns all users (admin only)
// @Summary Get all users
// @Description Get a list of all users in the system
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/admin/users [get]
func (ac *AdminController) GetAllUsers(c *gin.Context) {
	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Get users
	users, total, err := ac.userService.GetAllUsers(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + limit - 1) / limit,
		},
	})
}

// GetUserByID returns a specific user by ID
// @Summary Get user by ID
// @Description Get detailed information about a specific user
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/admin/users/{id} [get]
func (ac *AdminController) GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	user, err := ac.userService.GetByID(int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser updates user information
// @Summary Update user
// @Description Update user information (admin only)
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body UpdateUserRequest true "User update data"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/admin/users/{id} [put]
func (ac *AdminController) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Build update request
	updateReq := &models.UpdateUserRequest{}

	if req.FirstName != nil {
		updateReq.FirstName = req.FirstName
	}
	if req.LastName != nil {
		updateReq.LastName = req.LastName
	}

	// Use UpdateUser for basic fields
	updatedUser, err := ac.userService.UpdateUser(int32(id), updateReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update user",
		})
		return
	}

	// Update admin-only fields directly
	needsDirectUpdate := false

	if req.Email != nil && *req.Email != updatedUser.Email {
		updatedUser.Email = *req.Email
		needsDirectUpdate = true
	}
	if req.Role != nil && *req.Role != updatedUser.Role {
		updatedUser.Role = *req.Role
		needsDirectUpdate = true
	}
	if req.IsActive != nil && *req.IsActive != updatedUser.IsActive {
		updatedUser.IsActive = *req.IsActive
		needsDirectUpdate = true
	}
	if req.InitialBalance != nil && *req.InitialBalance != updatedUser.InitialBalance {
		updatedUser.InitialBalance = *req.InitialBalance
		needsDirectUpdate = true
	}

	if needsDirectUpdate {
		// Update through repository directly for admin fields
		if err := ac.userService.UpdateBalance(int32(id), updatedUser.InitialBalance); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update balance",
			})
			return
		}
	}

	c.JSON(http.StatusOK, updatedUser)
}

// DeleteUser soft deletes a user
// @Summary Delete user
// @Description Soft delete a user (admin only)
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/admin/users/{id} [delete]
func (ac *AdminController) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	if err := ac.userService.DeleteUser(int32(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found or already deleted",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}

// UpdateUserBalance updates a user's balance
// @Summary Update user balance
// @Description Update user's initial balance (admin only)
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param balance body UpdateBalanceRequest true "Balance update data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/admin/users/{id}/balance [patch]
func (ac *AdminController) UpdateUserBalance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var req UpdateBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if req.Amount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Amount must be non-zero",
		})
		return
	}

	// Update balance
	newBalance, err := ac.userService.UpdateBalanceWithTransaction(int32(id), req.Amount, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update balance",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Balance updated successfully",
		"new_balance": newBalance,
	})
}

// CreateUser creates a new user (admin only)
// @Summary Create user
// @Description Create a new user in the system (admin only)
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param user body CreateUserRequest true "User creation data"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/admin/users [post]
func (ac *AdminController) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Create user using existing CreateUser service
	firstName := ""
	if req.FirstName != nil {
		firstName = *req.FirstName
	}
	lastName := ""
	if req.LastName != nil {
		lastName = *req.LastName
	}

	registerReq := &models.RegisterRequest{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: firstName,
		LastName:  lastName,
	}

	user, err := ac.userService.CreateUser(registerReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Set role if specified (default is normal)
	if req.Role != nil {
		user.Role = *req.Role
	}

	// Set initial balance if specified
	if req.InitialBalance != nil {
		user.InitialBalance = *req.InitialBalance
		user.CurrentBalance = *req.InitialBalance
	}

	c.JSON(http.StatusCreated, user)
}

// ReactivateUser reactivates a deleted user
// @Summary Reactivate user
// @Description Reactivate a soft-deleted user (admin only)
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/admin/users/{id}/reactivate [post]
func (ac *AdminController) ReactivateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Update is_active to true using UpdateUser
	isActive := true
	updateReq := &models.UpdateUserRequest{
		IsActive: &isActive,
	}

	updatedUser, err := ac.userService.UpdateUser(int32(id), updateReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to reactivate user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User reactivated successfully",
		"user":    updatedUser,
	})
}

// Request DTOs
type CreateUserRequest struct {
	Username       string           `json:"username" binding:"required"`
	Email          string           `json:"email" binding:"required,email"`
	Password       string           `json:"password" binding:"required,min=8"`
	FirstName      *string          `json:"first_name"`
	LastName       *string          `json:"last_name"`
	Role           *models.UserRole `json:"role"`
	InitialBalance *float64         `json:"initial_balance"`
}

type UpdateUserRequest struct {
	Email          *string          `json:"email"`
	FirstName      *string          `json:"first_name"`
	LastName       *string          `json:"last_name"`
	Role           *models.UserRole `json:"role"`
	IsActive       *bool            `json:"is_active"`
	InitialBalance *float64         `json:"initial_balance"`
}

type UpdateBalanceRequest struct {
	Amount      float64 `json:"amount" binding:"required"`
	Description string  `json:"description"`
}
