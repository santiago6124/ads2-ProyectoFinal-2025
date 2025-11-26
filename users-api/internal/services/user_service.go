package services

import (
	"fmt"
	"strings"
	"time"

	"users-api/internal/models"
	"users-api/internal/repositories"
	"users-api/pkg/utils"
)

type UserService interface {
	CreateUser(req *models.RegisterRequest) (*models.User, error)
	GetUserByID(id int32) (*models.User, error)
	GetByID(id int32) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	UpdateUser(id int32, req *models.UpdateUserRequest) (*models.User, error)
	ChangePassword(id int32, req *models.ChangePasswordRequest) error
	UpdateBalance(id int32, newBalance float64) error
	UpdateBalanceWithTransaction(id int32, amount float64, description string) (float64, error)
	DeactivateUser(id int32) error
	DeleteUser(id int32) error
	ListUsers(page, limit int, search, role string, isActive *bool) ([]models.User, int64, error)
	GetAllUsers(offset, limit int) ([]models.User, int, error)
	UpgradeUserToAdmin(id int32) (*models.User, error)
	VerifyUser(id int32) (*models.UserVerificationResponse, error)
}

type userService struct {
	userRepo        repositories.UserRepository
	balanceRepo     repositories.BalanceTransactionRepository
}

func NewUserService(userRepo repositories.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func NewUserServiceWithBalance(userRepo repositories.UserRepository, balanceRepo repositories.BalanceTransactionRepository) UserService {
	return &userService{
		userRepo:    userRepo,
		balanceRepo: balanceRepo,
	}
}

func (s *userService) CreateUser(req *models.RegisterRequest) (*models.User, error) {
	if err := utils.ValidateUsername(req.Username); err != nil {
		return nil, fmt.Errorf("invalid username: %w", err)
	}

	if err := utils.ValidateEmail(req.Email); err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	if err := utils.ValidatePassword(req.Password); err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Username = strings.TrimSpace(req.Username)

	existingUser, _ := s.userRepo.GetByEmail(req.Email)
	if existingUser != nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	existingUser, _ = s.userRepo.GetByUsername(req.Username)
	if existingUser != nil {
		return nil, fmt.Errorf("user with this username already exists")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Username:       req.Username,
		Email:          req.Email,
		PasswordHash:   hashedPassword,
		Role:           models.RoleNormal,
		InitialBalance: 100000.00,
		IsActive:       true,
	}

	if req.FirstName != "" {
		firstName := strings.TrimSpace(req.FirstName)
		user.FirstName = &firstName
	}

	if req.LastName != "" {
		lastName := strings.TrimSpace(req.LastName)
		user.LastName = &lastName
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *userService) GetUserByID(id int32) (*models.User, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("user account is deactivated")
	}

	// Calculate current balance from latest transaction
	if s.balanceRepo != nil {
		latestTx, err := s.balanceRepo.GetLatestByUserID(id)
		if err == nil && latestTx != nil {
			user.CurrentBalance = latestTx.NewBalance
		} else {
			// No transactions yet, use initial balance
			user.CurrentBalance = user.InitialBalance
		}
	} else {
		// Fallback to initial balance if balanceRepo is nil
		user.CurrentBalance = user.InitialBalance
	}

	return user, nil
}

func (s *userService) GetUserByEmail(email string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (s *userService) UpdateUser(id int32, req *models.UpdateUserRequest) (*models.User, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Allow updating IsActive even if user is deactivated (for reactivation)
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	} else if !user.IsActive {
		return nil, fmt.Errorf("cannot update deactivated user")
	}

	if req.FirstName != nil {
		firstName := strings.TrimSpace(*req.FirstName)
		if firstName == "" {
			user.FirstName = nil
		} else {
			user.FirstName = &firstName
		}
	}

	if req.LastName != nil {
		lastName := strings.TrimSpace(*req.LastName)
		if lastName == "" {
			user.LastName = nil
		} else {
			user.LastName = &lastName
		}
	}

	if req.Preferences != "" {
		user.Preferences = req.Preferences
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

func (s *userService) ChangePassword(id int32, req *models.ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if !user.IsActive {
		return fmt.Errorf("cannot change password for deactivated user")
	}

	if !utils.CheckPasswordHash(req.CurrentPassword, user.PasswordHash) {
		return fmt.Errorf("current password is incorrect")
	}

	if err := utils.ValidatePassword(req.NewPassword); err != nil {
		return fmt.Errorf("invalid new password: %w", err)
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	user.PasswordHash = hashedPassword

	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

func (s *userService) UpdateBalance(id int32, newBalance float64) error {
	if err := s.userRepo.UpdateBalance(id, newBalance); err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}
	return nil
}

func (s *userService) DeactivateUser(id int32) error {
	exists, err := s.userRepo.Exists(id)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("user not found or already deactivated")
	}

	if err := s.userRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	return nil
}

func (s *userService) ListUsers(page, limit int, search, role string, isActive *bool) ([]models.User, int64, error) {
	if page < 1 {
		page = 1
	}

	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	users, total, err := s.userRepo.List(offset, limit, search, role, isActive)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

func (s *userService) UpgradeUserToAdmin(id int32) (*models.User, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("cannot upgrade deactivated user")
	}

	if user.Role == models.RoleAdmin {
		return nil, fmt.Errorf("user is already an admin")
	}

	user.Role = models.RoleAdmin

	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to upgrade user: %w", err)
	}

	return user, nil
}

func (s *userService) VerifyUser(id int32) (*models.UserVerificationResponse, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return &models.UserVerificationResponse{
			Exists:   false,
			UserID:   0,
			Role:     "",
			IsActive: false,
		}, nil
	}

	return &models.UserVerificationResponse{
		Exists:   true,
		UserID:   user.ID,
		Role:     user.Role,
		IsActive: user.IsActive,
	}, nil
}

// GetByID is an alias for GetUserByID for admin controller compatibility
func (s *userService) GetByID(id int32) (*models.User, error) {
	return s.GetUserByID(id)
}

// DeleteUser performs a soft delete on a user
func (s *userService) DeleteUser(id int32) error {
	return s.DeactivateUser(id)
}

// GetAllUsers returns all users with pagination
func (s *userService) GetAllUsers(offset, limit int) ([]models.User, int, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}

	users, total, err := s.userRepo.List(offset, limit, "", "", nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}

	// Add current balance for each user
	for i := range users {
		if s.balanceRepo != nil {
			latestTx, err := s.balanceRepo.GetLatestByUserID(users[i].ID)
			if err == nil && latestTx != nil {
				users[i].CurrentBalance = latestTx.NewBalance
			} else {
				users[i].CurrentBalance = users[i].InitialBalance
			}
		} else {
			users[i].CurrentBalance = users[i].InitialBalance
		}
	}

	return users, int(total), nil
}

// UpdateBalanceWithTransaction updates user balance and creates a transaction record
func (s *userService) UpdateBalanceWithTransaction(id int32, amount float64, description string) (float64, error) {
	if s.balanceRepo == nil {
		return 0, fmt.Errorf("balance transaction repository not available")
	}

	// Get current balance
	user, err := s.GetUserByID(id)
	if err != nil {
		return 0, fmt.Errorf("failed to get user: %w", err)
	}

	// Calculate new balance
	newBalance := user.CurrentBalance + amount

	if newBalance < 0 {
		return 0, fmt.Errorf("insufficient balance")
	}

	// Create transaction record
	txType := "deposit"
	if amount < 0 {
		txType = "withdrawal"
	}

	// Generate unique order ID for admin balance updates
	orderID := fmt.Sprintf("ADMIN_%d_%d", id, time.Now().Unix())

	transaction := &models.BalanceTransaction{
		OrderID:         orderID,
		UserID:          id,
		Amount:          amount,
		TransactionType: txType,
		CryptoSymbol:    "USD", // Admin balance adjustments are in fiat currency
		PreviousBalance: user.CurrentBalance,
		NewBalance:      newBalance,
	}

	if err := s.balanceRepo.Create(transaction); err != nil {
		return 0, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Update user's current balance
	if err := s.UpdateBalance(id, newBalance); err != nil {
		return 0, fmt.Errorf("failed to update user balance: %w", err)
	}

	return newBalance, nil
}
