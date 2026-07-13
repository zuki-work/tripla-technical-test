package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	TransactionStatusPending           = "pending"
	TransactionStatusProcessing        = "processing"
	TransactionStatusWaitingForPayment = "waiting_for_payment"
	TransactionStatusSuccess           = "success"
	TransactionStatusFailed            = "failed"
	TransactionStatusCancelled         = "cancelled"
	TransactionStatusExpired           = "expired"
)

const (
	AccountingStatusPending = "pending"
	AccountingStatusSynced  = "synced"
	AccountingStatusFailed  = "failed"
)

type Transaction struct {
	gorm.Model
	RequestID          string     `json:"request_id" gorm:"type:varchar(64);index"`
	UserID             uint       `json:"user_id" binding:"required"`
	User               User       `json:"user"`
	TicketID           uint       `json:"ticket_id" binding:"required"`
	Ticket             Ticket     `json:"ticket"`
	Quantity           uint       `json:"quantity" binding:"required"`
	Status             string     `json:"status" gorm:"type:varchar(30);index"`
	FailedReason       string     `json:"failed_reason"`
	AccountingStatus   string     `json:"accounting_status" gorm:"type:varchar(30);index"`
	AccountingSyncedAt *time.Time `json:"accounting_synced_at"`
	ProcessedAt        *time.Time `json:"processed_at"`
	ExpiresAt          *time.Time `json:"expires_at"`
}
