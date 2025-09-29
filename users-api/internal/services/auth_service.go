package services

import (
	"fmt"
	"strings"
	"time"

	"users-api/internal/models"
	"users-api/internal/repositories"
	"users-api/pkg/utils"
)

type AuthService interface {
	Authenticate(email, password string, ipAddress, userAgent string) (*models.AuthResponse, error)
	RefreshToken(refreshToken string) (*models.TokenPair, error)
	Logout(refreshToken string) error
	LogoutAll(userID uint) error
	IsRateLimited(email string) (bool, error)
}

type authService struct {
	userRepo             repositories.UserRepository
	loginAttemptRepo     repositories.LoginAttemptRepository
	tokenService         TokenService
	maxFailedAttempts    int
	rateLimitWindow      time.Duration
}

func NewAuthService(
	userRepo repositories.UserRepository,
	loginAttemptRepo repositories.LoginAttemptRepository,
	tokenService TokenService,
) AuthService {
	return &authService{
		userRepo:             userRepo,
		loginAttemptRepo:     loginAttemptRepo,
		tokenService:         tokenService,
		maxFailedAttempts:    5,
		rateLimitWindow:      15 * time.Minute,
	}
}

func (s *authService) Authenticate(email, password, ipAddress, userAgent string) (*models.AuthResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	isLimited, err := s.IsRateLimited(email)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	if isLimited {
		s.recordLoginAttempt(email, ipAddress, userAgent, false)
		return nil, fmt.Errorf("too many failed login attempts. Please try again later")
	}

	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		s.recordLoginAttempt(email, ipAddress, userAgent, false)
		return nil, fmt.Errorf("invalid email or password")
	}

	if !user.IsActive {
		s.recordLoginAttempt(email, ipAddress, userAgent, false)
		return nil, fmt.Errorf("account is deactivated")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		s.recordLoginAttempt(email, ipAddress, userAgent, false)
		return nil, fmt.Errorf("invalid email or password")
	}

	tokenPair, err := s.tokenService.GenerateTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	if err := s.userRepo.UpdateLastLogin(user.ID); err != nil {
		return nil, fmt.Errorf("failed to update last login: %w", err)
	}

	s.recordLoginAttempt(email, ipAddress, userAgent, true)

	return &models.AuthResponse{
		User:         user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

func (s *authService) RefreshToken(refreshToken string) (*models.TokenPair, error) {
	tokenPair, err := s.tokenService.RefreshAccessToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return tokenPair, nil
}

func (s *authService) Logout(refreshToken string) error {
	err := s.tokenService.RevokeRefreshToken(refreshToken)
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	return nil
}

func (s *authService) LogoutAll(userID uint) error {
	err := s.tokenService.RevokeAllUserTokens(userID)
	if err != nil {
		return fmt.Errorf("failed to logout from all devices: %w", err)
	}

	return nil
}

func (s *authService) IsRateLimited(email string) (bool, error) {
	since := time.Now().Add(-s.rateLimitWindow)

	failedCount, err := s.loginAttemptRepo.CountFailedAttempts(email, since)
	if err != nil {
		return false, fmt.Errorf("failed to count failed attempts: %w", err)
	}

	return failedCount >= int64(s.maxFailedAttempts), nil
}

func (s *authService) recordLoginAttempt(email, ipAddress, userAgent string, success bool) {
	attempt := &models.LoginAttempt{
		Email:       email,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Success:     success,
		AttemptedAt: time.Now(),
	}

	s.loginAttemptRepo.Create(attempt)
}