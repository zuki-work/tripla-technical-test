package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	TransactionStatusPending   = "pending"
	TransactionStatusConfirmed = "confirmed"
	TransactionStatusCancelled = "cancelled"
	TransactionStatusExpired   = "expired"
)

type Transaction struct {
	gorm.Model
	UserID    uint      `json:"user_id" binding:"required"`
	User      User      `json:"user"`
	TicketID  uint      `json:"ticket_id" binding:"required"`
	Ticket    Ticket    `json:"ticket"`
	Quantity  uint      `json:"quantity" binding:"required"`
	Status    string    `json:"status" gorm:"type:varchar(20);index"`
	ExpiresAt time.Time `json:"expires_at"`
}
