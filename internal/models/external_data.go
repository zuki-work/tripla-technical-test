package models

import (
	"time"

	"gorm.io/gorm"
)

type ExternalData struct {
	gorm.Model
	TicketID           uint       `json:"ticket_id" gorm:"uniqueIndex"`
	Stock              uint       `json:"stock"`
	StockVersion       uint       `json:"stock_version"`
	LastAppliedAt      *time.Time `json:"last_applied_at"`
	LastIgnoredStock   uint       `json:"last_ignored_stock"`
	LastIgnoredVersion uint       `json:"last_ignored_version"`
}

func (ExternalData) TableName() string {
	return "external_data"
}
