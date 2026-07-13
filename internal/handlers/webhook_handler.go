package handlers

import (
	"errors"
	"io"
	"net/http"
	"time"

	"tripla-technical-test/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WebhookHandler struct {
	transactionService *services.TransactionService
}

func NewWebhookHandler(transactionService *services.TransactionService) *WebhookHandler {
	return &WebhookHandler{transactionService: transactionService}
}

type paymentWebhookRequest struct {
	ExternalPaymentID string `json:"external_payment_id" binding:"required"`
	TransactionID     uint   `json:"transaction_id" binding:"required"`
	Status            string `json:"status"`
	PaymentMethod     string `json:"payment_method"`
	PaidAt            string `json:"paid_at"`
}

func (h *WebhookHandler) HandlePaymentWebhook(c *gin.Context) {
	var input paymentWebhookRequest
	if err := c.ShouldBindJSON(&input); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	paidAt, err := parseOptionalWebhookTime(input.PaidAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.transactionService.HandlePaymentWebhook(input.ExternalPaymentID, input.TransactionID, input.Status, input.PaymentMethod, paidAt)
	if err != nil {
		writeWebhookError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func parseOptionalWebhookTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func writeWebhookError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
	case errors.Is(err, services.ErrInvalidPaymentWebhook):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
