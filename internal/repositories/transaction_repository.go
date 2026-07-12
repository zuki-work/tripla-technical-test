package repositories

import (
	"time"

	"tripla-technical-test/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) WithTx(tx *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: tx}
}

func (r *TransactionRepository) Create(transaction *models.Transaction) error {
	return r.db.Create(transaction).Error
}

func (r *TransactionRepository) FindAll() ([]models.Transaction, error) {
	var transactions []models.Transaction
	if err := r.db.Preload("User").Preload("Ticket").Find(&transactions).Error; err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *TransactionRepository) FindByID(id uint) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := r.db.Preload("User").Preload("Ticket").First(&transaction, id).Error; err != nil {
		return nil, err
	}

	return &transaction, nil
}

func (r *TransactionRepository) FindByIDForUpdate(id uint) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&transaction, id).Error; err != nil {
		return nil, err
	}

	return &transaction, nil
}

func (r *TransactionRepository) FindNextPending() (*models.Transaction, error) {
	var transaction models.Transaction
	if err := r.db.Where("status = ?", models.TransactionStatusPending).Order("id ASC").First(&transaction).Error; err != nil {
		return nil, err
	}

	return &transaction, nil
}

func (r *TransactionRepository) MarkPendingAsProcessing(id uint) (bool, error) {
	result := r.db.Model(&models.Transaction{}).
		Where("id = ? AND status = ?", id, models.TransactionStatusPending).
		Update("status", models.TransactionStatusProcessing)
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}

func (r *TransactionRepository) FindExpiredWaitingForPayment(now time.Time) ([]models.Transaction, error) {
	var transactions []models.Transaction
	if err := r.db.Where("status = ? AND expires_at <= ?", models.TransactionStatusWaitingForPayment, now).Find(&transactions).Error; err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *TransactionRepository) Save(transaction *models.Transaction) error {
	return r.db.Save(transaction).Error
}
