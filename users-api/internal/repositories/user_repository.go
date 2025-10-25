package repositories

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"users-api/internal/models"
)

type UserRepository interface {
	Create(user *models.User) error
	GetByID(id int32) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	Update(user *models.User) error
	UpdateBalance(id int32, newBalance float64) error
	Delete(id int32) error
	List(offset, limit int, search string, role string, isActive *bool) ([]models.User, int64, error)
	UpdateLastLogin(id int32) error
	Exists(id int32) (bool, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) Create(user *models.User) error {
	if err := r.db.Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("user already exists")
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *userRepository) GetByID(id int32) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

func (r *userRepository) Update(user *models.User) error {
	if err := r.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateBalance(id int32, newBalance float64) error {
	result := r.db.Model(&models.User{}).Where("id = ?", id).Update("initial_balance", newBalance)
	if result.Error != nil {
		return fmt.Errorf("failed to update balance: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *userRepository) Delete(id int32) error {
	result := r.db.Model(&models.User{}).Where("id = ?", id).Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *userRepository) List(offset, limit int, search string, role string, isActive *bool) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := r.db.Model(&models.User{})

	if search != "" {
		query = query.Where("username LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if role != "" {
		query = query.Where("role = ?", role)
	}

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

func (r *userRepository) UpdateLastLogin(id int32) error {
	now := time.Now()
	result := r.db.Model(&models.User{}).Where("id = ?", id).Update("last_login", &now)
	if result.Error != nil {
		return fmt.Errorf("failed to update last login: %w", result.Error)
	}
	return nil
}

func (r *userRepository) Exists(id int32) (bool, error) {
	var count int64
	if err := r.db.Model(&models.User{}).Where("id = ? AND is_active = ?", id, true).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return count > 0, nil
}

type RefreshTokenRepository interface {
	Create(token *models.RefreshToken) error
	GetByToken(token string) (*models.RefreshToken, error)
	RevokeByUserID(userID int32) error
	RevokeByToken(token string) error
	DeleteExpired() error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{
		db: db,
	}
}

func (r *refreshTokenRepository) Create(token *models.RefreshToken) error {
	if err := r.db.Create(token).Error; err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) GetByToken(token string) (*models.RefreshToken, error) {
	var refreshToken models.RefreshToken
	if err := r.db.Preload("User").Where("token = ? AND revoked = ?", token, false).First(&refreshToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("refresh token not found")
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	return &refreshToken, nil
}

func (r *refreshTokenRepository) RevokeByUserID(userID int32) error {
	if err := r.db.Model(&models.RefreshToken{}).Where("user_id = ?", userID).Update("revoked", true).Error; err != nil {
		return fmt.Errorf("failed to revoke refresh tokens: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) RevokeByToken(token string) error {
	result := r.db.Model(&models.RefreshToken{}).Where("token = ?", token).Update("revoked", true)
	if result.Error != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("refresh token not found")
	}
	return nil
}

func (r *refreshTokenRepository) DeleteExpired() error {
	if err := r.db.Where("expires_at < ? OR revoked = ?", time.Now(), true).Delete(&models.RefreshToken{}).Error; err != nil {
		return fmt.Errorf("failed to delete expired refresh tokens: %w", err)
	}
	return nil
}

type LoginAttemptRepository interface {
	Create(attempt *models.LoginAttempt) error
	GetRecentAttempts(email string, since time.Time) ([]models.LoginAttempt, error)
	CountFailedAttempts(email string, since time.Time) (int64, error)
}

type loginAttemptRepository struct {
	db *gorm.DB
}

func NewLoginAttemptRepository(db *gorm.DB) LoginAttemptRepository {
	return &loginAttemptRepository{
		db: db,
	}
}

func (r *loginAttemptRepository) Create(attempt *models.LoginAttempt) error {
	if err := r.db.Create(attempt).Error; err != nil {
		return fmt.Errorf("failed to create login attempt: %w", err)
	}
	return nil
}

func (r *loginAttemptRepository) GetRecentAttempts(email string, since time.Time) ([]models.LoginAttempt, error) {
	var attempts []models.LoginAttempt
	if err := r.db.Where("email = ? AND attempted_at >= ?", email, since).Order("attempted_at DESC").Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("failed to get recent login attempts: %w", err)
	}
	return attempts, nil
}

func (r *loginAttemptRepository) CountFailedAttempts(email string, since time.Time) (int64, error) {
	var count int64
	if err := r.db.Model(&models.LoginAttempt{}).Where("email = ? AND success = ? AND attempted_at >= ?", email, false, since).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count failed login attempts: %w", err)
	}
	return count, nil
}