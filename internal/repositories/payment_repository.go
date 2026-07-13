package repositories

import (
	"tripla-technical-test/internal/models"

	"gorm.io/gorm"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) WithTx(tx *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: tx}
}

func (r *PaymentRepository) Create(payment *models.Payment) error {
	return r.db.Create(payment).Error
}

func (r *PaymentRepository) FindByID(id uint) (*models.Payment, error) {
	var payment models.Payment
	if err := r.db.First(&payment, id).Error; err != nil {
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) FindByExternalPaymentID(externalPaymentID string) (*models.Payment, error) {
	var payment models.Payment
	if err := r.db.Where("external_payment_id = ?", externalPaymentID).First(&payment).Error; err != nil {
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) CountByExternalPaymentID(externalPaymentID string) (int64, error) {
	var count int64
	if err := r.db.Model(&models.Payment{}).Where("external_payment_id = ?", externalPaymentID).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
