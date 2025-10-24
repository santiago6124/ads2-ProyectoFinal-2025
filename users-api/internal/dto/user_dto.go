package dto

import (
	"encoding/json"
	"time"
	"users-api/internal/models"
)

type UserResponse struct {
	ID             int32           `json:"id"`
	Username       string          `json:"username"`
	Email          string          `json:"email"`
	FirstName      *string         `json:"first_name"`
	LastName       *string         `json:"last_name"`
	Role           models.UserRole `json:"role"`
	InitialBalance float64         `json:"initial_balance"`
	CreatedAt      time.Time       `json:"created_at"`
	LastLogin      *time.Time      `json:"last_login,omitempty"`
	IsActive       bool            `json:"is_active"`
	Preferences    string          `json:"preferences,omitempty"`
}

type UserSummaryResponse struct {
	ID        int32           `json:"id"`
	Username  string          `json:"username"`
	Email     string          `json:"email"`
	Role      models.UserRole `json:"role"`
	CreatedAt time.Time       `json:"created_at"`
	IsActive  bool            `json:"is_active"`
}

type UserListResponse struct {
	Users      []UserSummaryResponse `json:"users"`
	Pagination PaginationResponse    `json:"pagination"`
}

type PaginationResponse struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

type LoginResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

func ToUserResponse(user *models.User) UserResponse {
	prefs, _ := user.GetPreferences()
	prefsJSON, _ := json.Marshal(prefs)

	return UserResponse{
		ID:             user.ID,
		Username:       user.Username,
		Email:          user.Email,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		Role:           user.Role,
		InitialBalance: user.InitialBalance,
		CreatedAt:      user.CreatedAt,
		LastLogin:      user.LastLogin,
		IsActive:       user.IsActive,
		Preferences:    string(prefsJSON),
	}
}

func ToUserSummaryResponse(user *models.User) UserSummaryResponse {
	return UserSummaryResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		IsActive:  user.IsActive,
	}
}

func ToUserListResponse(users []models.User, total int64, page, limit int) UserListResponse {
	userResponses := make([]UserSummaryResponse, len(users))
	for i, user := range users {
		userResponses[i] = ToUserSummaryResponse(&user)
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return UserListResponse{
		Users: userResponses,
		Pagination: PaginationResponse{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	}
}
