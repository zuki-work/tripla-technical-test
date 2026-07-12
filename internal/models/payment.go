package models

import (
	"time"

	"gorm.io/gorm"
)

const PaymentStatusSuccess = "success"

type Payment struct {
	gorm.Model
	TransactionID uint        `json:"transaction_id" binding:"required"`
	Transaction   Transaction `json:"transaction"`
	Status        string      `json:"status" gorm:"type:varchar(20);index"`
	PaymentMethod string      `json:"payment_method"`
	PaidAt        *time.Time  `json:"paid_at"`
}
