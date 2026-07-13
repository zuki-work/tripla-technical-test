package handlers

import (
	"errors"
	"io"
	"net/http"

	"tripla-technical-test/internal/repositories"
	"tripla-technical-test/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TransactionHandler struct {
	transactionService *services.TransactionService
}

func NewTransactionHandler(transactionService *services.TransactionService) *TransactionHandler {
	return &TransactionHandler{transactionService: transactionService}
}

type createTransactionRequest struct {
	UserID   uint `json:"user_id" binding:"required"`
	TicketID uint `json:"ticket_id" binding:"required"`
	Quantity uint `json:"quantity" binding:"required"`
}

type processPendingTransactionsRequest struct {
	Limit int `json:"limit"`
}

type payTransactionRequest struct {
	PaymentMethod string `json:"payment_method"`
}

func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	var input createTransactionRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transaction, err := h.transactionService.CreateTransaction(input.UserID, input.TicketID, input.Quantity)
	if err != nil {
		writeTransactionError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"data": transaction})
}

func (h *TransactionHandler) GetTransactions(c *gin.Context) {
	transactions, err := h.transactionService.GetTransactions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transactions})
}

func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transaction, err := h.transactionService.GetTransaction(id)
	if err != nil {
		writeTransactionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

func (h *TransactionHandler) ProcessPendingTransactions(c *gin.Context) {
	var input processPendingTransactionsRequest
	if err := c.ShouldBindJSON(&input); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.transactionService.ProcessPendingTransactions(input.Limit)
	if err != nil {
		writeTransactionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *TransactionHandler) PayTransaction(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var input payTransactionRequest
	if err := c.ShouldBindJSON(&input); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.transactionService.PayTransaction(id, input.PaymentMethod)
	if err != nil {
		writeTransactionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *TransactionHandler) SyncTransactionAccounting(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transaction, err := h.transactionService.SyncTransactionAccounting(id)
	if err != nil {
		if isAccountingError(err) {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "data": transaction})
			return
		}

		writeTransactionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

func writeTransactionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
	case errors.Is(err, services.ErrInvalidTransactionQuantity),
		errors.Is(err, services.ErrInsufficientTickets),
		errors.Is(err, services.ErrTransactionNotWaitingForPayment),
		errors.Is(err, services.ErrTransactionExpired),
		errors.Is(err, services.ErrTransactionNotSuccess):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case isAccountingError(err):
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func isAccountingError(err error) bool {
	return errors.Is(err, repositories.ErrAccountingKnownError) || errors.Is(err, repositories.ErrAccountingInternalServerError)
}
