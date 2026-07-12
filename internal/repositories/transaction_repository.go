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

func (r *TransactionRepository) FindExpiredPending(now time.Time) ([]models.Transaction, error) {
	var transactions []models.Transaction
	if err := r.db.Where("status = ? AND expires_at <= ?", models.TransactionStatusPending, now).Find(&transactions).Error; err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *TransactionRepository) Save(transaction *models.Transaction) error {
	return r.db.Save(transaction).Error
}
