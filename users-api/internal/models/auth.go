package models

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

type AuthResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type CustomClaims struct {
	UserID   int32    `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Role     UserRole `json:"role"`
	jwt.RegisteredClaims
}

type JWTConfig struct {
	SecretKey       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Issuer          string
}

func NewJWTConfig() *JWTConfig {
	return &JWTConfig{
		SecretKey:       "your-super-secret-key",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: time.Hour * 24 * 7,
		Issuer:          "users-api",
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type RegisterRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"max=50"`
	LastName  string `json:"last_name" binding:"max=50"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

type UpdateUserRequest struct {
	FirstName   *string    `json:"first_name" binding:"omitempty,max=50"`
	LastName    *string    `json:"last_name" binding:"omitempty,max=50"`
	Preferences *UserPrefs `json:"preferences"`
}

type UserVerificationResponse struct {
	Exists   bool     `json:"exists"`
	UserID   int32    `json:"user_id"`
	Role     UserRole `json:"role"`
	IsActive bool     `json:"is_active"`
}