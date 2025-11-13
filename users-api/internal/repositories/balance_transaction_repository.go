package repositories

import (
	"gorm.io/gorm"
	"users-api/internal/models"
)

type BalanceTransactionRepository interface {
	FindByOrderID(orderID string) (*models.BalanceTransaction, error)
	Create(transaction *models.BalanceTransaction) error
	GetLatestByUserID(userID int32) (*models.BalanceTransaction, error)
}

type balanceTransactionRepository struct {
	db *gorm.DB
}

func NewBalanceTransactionRepository(db *gorm.DB) BalanceTransactionRepository {
	return &balanceTransactionRepository{db: db}
}

func (r *balanceTransactionRepository) FindByOrderID(orderID string) (*models.BalanceTransaction, error) {
	var transaction models.BalanceTransaction
	result := r.db.Where("order_id = ?", orderID).First(&transaction)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // No encontrado, no es error
		}
		return nil, result.Error
	}

	return &transaction, nil
}

func (r *balanceTransactionRepository) Create(transaction *models.BalanceTransaction) error {
	return r.db.Create(transaction).Error
}

func (r *balanceTransactionRepository) GetLatestByUserID(userID int32) (*models.BalanceTransaction, error) {
	var transaction models.BalanceTransaction
	result := r.db.Where("user_id = ?", userID).Order("processed_at DESC").First(&transaction)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // No transactions yet
		}
		return nil, result.Error
	}

	return &transaction, nil
}
