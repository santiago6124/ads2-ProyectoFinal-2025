package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"users-api/internal/models"
	"users-api/internal/repositories"
)

type TokenService interface {
	GenerateTokenPair(user *models.User) (*models.TokenPair, error)
	ValidateAccessToken(tokenString string) (*models.CustomClaims, error)
	RefreshAccessToken(refreshToken string) (*models.TokenPair, error)
	RevokeRefreshToken(token string) error
	RevokeAllUserTokens(userID uint) error
}

type tokenService struct {
	jwtConfig              *models.JWTConfig
	refreshTokenRepository repositories.RefreshTokenRepository
}

func NewTokenService(jwtConfig *models.JWTConfig, refreshTokenRepo repositories.RefreshTokenRepository) TokenService {
	return &tokenService{
		jwtConfig:              jwtConfig,
		refreshTokenRepository: refreshTokenRepo,
	}
}

func (s *tokenService) GenerateTokenPair(user *models.User) (*models.TokenPair, error) {
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtConfig.AccessTokenTTL.Seconds()),
	}, nil
}

func (s *tokenService) generateAccessToken(user *models.User) (string, error) {
	now := time.Now()
	claims := &models.CustomClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtConfig.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.jwtConfig.Issuer,
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtConfig.SecretKey))
}

func (s *tokenService) generateRefreshToken(user *models.User) (string, error) {
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	tokenString := base64.URLEncoding.EncodeToString(tokenBytes)

	refreshToken := &models.RefreshToken{
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: time.Now().Add(s.jwtConfig.RefreshTokenTTL),
	}

	if err := s.refreshTokenRepository.Create(refreshToken); err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return tokenString, nil
}

func (s *tokenService) ValidateAccessToken(tokenString string) (*models.CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtConfig.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*models.CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (s *tokenService) RefreshAccessToken(refreshToken string) (*models.TokenPair, error) {
	storedToken, err := s.refreshTokenRepository.GetByToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if storedToken.IsExpired() {
		return nil, fmt.Errorf("refresh token expired")
	}

	if storedToken.Revoked {
		return nil, fmt.Errorf("refresh token revoked")
	}

	if err := s.refreshTokenRepository.RevokeByToken(refreshToken); err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	return s.GenerateTokenPair(&storedToken.User)
}

func (s *tokenService) RevokeRefreshToken(token string) error {
	return s.refreshTokenRepository.RevokeByToken(token)
}

func (s *tokenService) RevokeAllUserTokens(userID uint) error {
	return s.refreshTokenRepository.RevokeByUserID(userID)
}