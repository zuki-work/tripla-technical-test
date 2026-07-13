package repositories

import (
	"time"

	"tripla-technical-test/internal/models"

	"gorm.io/gorm"
)

type ExternalDataRepository struct {
	db *gorm.DB
}

func NewExternalDataRepository(db *gorm.DB) *ExternalDataRepository {
	return &ExternalDataRepository{db: db}
}

func (r *ExternalDataRepository) ApplyTicketStock(ticketID uint, stock uint, stockVersion uint) (*models.ExternalData, bool, error) {
	var externalData models.ExternalData
	err := r.db.Where("ticket_id = ?", ticketID).First(&externalData).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, false, err
	}

	now := time.Now()
	if err == gorm.ErrRecordNotFound {
		externalData = models.ExternalData{
			TicketID:      ticketID,
			Stock:         stock,
			StockVersion:  stockVersion,
			LastAppliedAt: &now,
		}
		if err := r.db.Create(&externalData).Error; err != nil {
			return nil, false, err
		}

		return &externalData, true, nil
	}

	if stockVersion <= externalData.StockVersion {
		externalData.LastIgnoredStock = stock
		externalData.LastIgnoredVersion = stockVersion
		if err := r.db.Save(&externalData).Error; err != nil {
			return nil, false, err
		}

		return &externalData, false, nil
	}

	externalData.Stock = stock
	externalData.StockVersion = stockVersion
	externalData.LastAppliedAt = &now
	if err := r.db.Save(&externalData).Error; err != nil {
		return nil, false, err
	}

	return &externalData, true, nil
}
