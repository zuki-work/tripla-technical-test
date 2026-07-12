package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"tripla-technical-test/internal/models"
	"tripla-technical-test/internal/repositories"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const (
	transactionExpiryDuration      = 10 * time.Minute
	defaultPendingTransactionLimit = 100
	maxPendingTransactionLimit     = 1000
	maxDeadlockRetries             = 3
)

var (
	ErrInvalidTransactionQuantity      = errors.New("transaction quantity must be greater than zero")
	ErrInsufficientTickets             = errors.New("not enough tickets available")
	ErrTransactionNotWaitingForPayment = errors.New("transaction is not waiting for payment")
	ErrTransactionExpired              = errors.New("transaction has expired")
	errTransactionClaimConflict        = errors.New("transaction was claimed by another worker")
)

type TransactionPaymentResult struct {
	Transaction *models.Transaction `json:"transaction"`
	Payment     *models.Payment     `json:"payment"`
}

type ProcessPendingTransactionResult struct {
	TransactionID uint   `json:"transaction_id"`
	RequestID     string `json:"request_id"`
	Status        string `json:"status"`
	FailedReason  string `json:"failed_reason,omitempty"`
}

type ProcessPendingTransactionsResult struct {
	Limit                  int                               `json:"limit"`
	ProcessedCount         int                               `json:"processed_count"`
	WaitingForPaymentCount int                               `json:"waiting_for_payment_count"`
	FailedCount            int                               `json:"failed_count"`
	Results                []ProcessPendingTransactionResult `json:"results"`
}

type TransactionService struct {
	db                    *gorm.DB
	ticketRepository      *repositories.TicketRepository
	transactionRepository *repositories.TransactionRepository
	paymentRepository     *repositories.PaymentRepository
}

func NewTransactionService(db *gorm.DB, ticketRepository *repositories.TicketRepository, transactionRepository *repositories.TransactionRepository, paymentRepository *repositories.PaymentRepository) *TransactionService {
	return &TransactionService{
		db:                    db,
		ticketRepository:      ticketRepository,
		transactionRepository: transactionRepository,
		paymentRepository:     paymentRepository,
	}
}

func (s *TransactionService) CreateTransaction(userID, ticketID, quantity uint) (*models.Transaction, error) {
	if quantity == 0 {
		return nil, ErrInvalidTransactionQuantity
	}

	transaction := models.Transaction{
		RequestID: generateRequestID(),
		UserID:    userID,
		TicketID:  ticketID,
		Quantity:  quantity,
		Status:    models.TransactionStatusPending,
	}
	if err := s.transactionRepository.Create(&transaction); err != nil {
		return nil, err
	}

	return s.transactionRepository.FindByID(transaction.ID)
}

func (s *TransactionService) GetTransactions() ([]models.Transaction, error) {
	if err := s.ExpireWaitingForPaymentTransactions(); err != nil {
		return nil, err
	}

	return s.transactionRepository.FindAll()
}

func (s *TransactionService) GetTransaction(id uint) (*models.Transaction, error) {
	if err := s.ExpireWaitingForPaymentTransactions(); err != nil {
		return nil, err
	}

	return s.transactionRepository.FindByID(id)
}

func (s *TransactionService) ProcessPendingTransactions(limit int) (*ProcessPendingTransactionsResult, error) {
	return s.processPendingTransactions(limit, true)
}

func (s *TransactionService) ProcessPendingTransactionsWithoutExpiry(limit int) (*ProcessPendingTransactionsResult, error) {
	return s.processPendingTransactions(limit, false)
}

func (s *TransactionService) processPendingTransactions(limit int, expireFirst bool) (*ProcessPendingTransactionsResult, error) {
	if limit <= 0 {
		limit = defaultPendingTransactionLimit
	}
	if limit > maxPendingTransactionLimit {
		limit = maxPendingTransactionLimit
	}

	if expireFirst {
		if err := s.ExpireWaitingForPaymentTransactions(); err != nil {
			return nil, err
		}
	}

	result := &ProcessPendingTransactionsResult{
		Limit:   limit,
		Results: make([]ProcessPendingTransactionResult, 0, limit),
	}

	for i := 0; i < limit; i++ {
		processed, err := s.processNextPendingTransactionWithRetry()
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return nil, err
		}

		result.ProcessedCount++
		if processed.Status == models.TransactionStatusWaitingForPayment {
			result.WaitingForPaymentCount++
		}
		if processed.Status == models.TransactionStatusFailed {
			result.FailedCount++
		}

		result.Results = append(result.Results, ProcessPendingTransactionResult{
			TransactionID: processed.ID,
			RequestID:     processed.RequestID,
			Status:        processed.Status,
			FailedReason:  processed.FailedReason,
		})
	}

	return result, nil
}

func (s *TransactionService) processNextPendingTransactionWithRetry() (*models.Transaction, error) {
	var lastErr error
	for attempt := 0; attempt <= maxDeadlockRetries; attempt++ {
		processed, err := s.processNextPendingTransaction()
		if err == nil {
			return processed, nil
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if errors.Is(err, errTransactionClaimConflict) {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
			continue
		}
		if !isRetryableMySQLLockError(err) {
			return nil, err
		}

		lastErr = err
		time.Sleep(time.Duration(attempt+1) * 25 * time.Millisecond)
	}

	if errors.Is(lastErr, errTransactionClaimConflict) {
		return nil, gorm.ErrRecordNotFound
	}

	return nil, lastErr
}

func (s *TransactionService) processNextPendingTransaction() (*models.Transaction, error) {
	var processed models.Transaction

	err := s.db.Transaction(func(tx *gorm.DB) error {
		ticketRepository := s.ticketRepository.WithTx(tx)
		transactionRepository := s.transactionRepository.WithTx(tx)

		transaction, err := transactionRepository.FindNextPending()
		if err != nil {
			return err
		}

		claimed, err := transactionRepository.MarkPendingAsProcessing(transaction.ID)
		if err != nil {
			return err
		}
		if !claimed {
			return errTransactionClaimConflict
		}

		now := time.Now()
		transaction.Status = models.TransactionStatusProcessing

		ticket, err := ticketRepository.FindByIDForUpdate(transaction.TicketID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				transaction.Status = models.TransactionStatusFailed
				transaction.FailedReason = "ticket not found"
				transaction.ProcessedAt = &now
				if err := transactionRepository.Save(transaction); err != nil {
					return err
				}
				processed = *transaction
				return nil
			}
			return err
		}

		if ticket.QuantityAvailable < transaction.Quantity {
			transaction.Status = models.TransactionStatusFailed
			transaction.FailedReason = ErrInsufficientTickets.Error()
			transaction.ProcessedAt = &now
			if err := transactionRepository.Save(transaction); err != nil {
				return err
			}
			processed = *transaction
			return nil
		}

		ticket.QuantityAvailable -= transaction.Quantity
		if err := ticketRepository.Save(ticket); err != nil {
			return err
		}

		expiresAt := now.Add(transactionExpiryDuration)
		transaction.Status = models.TransactionStatusWaitingForPayment
		transaction.FailedReason = ""
		transaction.ProcessedAt = &now
		transaction.ExpiresAt = &expiresAt
		if err := transactionRepository.Save(transaction); err != nil {
			return err
		}

		processed = *transaction
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &processed, nil
}

func (s *TransactionService) PayTransaction(id uint, paymentMethod string) (*TransactionPaymentResult, error) {
	if err := s.ExpireWaitingForPaymentTransactions(); err != nil {
		return nil, err
	}

	if paymentMethod == "" {
		paymentMethod = "auto"
	}

	var transactionID uint
	var paymentID uint
	err := s.db.Transaction(func(tx *gorm.DB) error {
		ticketRepository := s.ticketRepository.WithTx(tx)
		transactionRepository := s.transactionRepository.WithTx(tx)
		paymentRepository := s.paymentRepository.WithTx(tx)

		transaction, err := transactionRepository.FindByIDForUpdate(id)
		if err != nil {
			return err
		}

		if transaction.Status != models.TransactionStatusWaitingForPayment {
			return ErrTransactionNotWaitingForPayment
		}
		if transaction.ExpiresAt == nil || !transaction.ExpiresAt.After(time.Now()) {
			ticket, err := ticketRepository.FindByIDForUpdate(transaction.TicketID)
			if err != nil {
				return err
			}

			ticket.QuantityAvailable += transaction.Quantity
			if err := ticketRepository.Save(ticket); err != nil {
				return err
			}

			transaction.Status = models.TransactionStatusExpired
			if err := transactionRepository.Save(transaction); err != nil {
				return err
			}
			return ErrTransactionExpired
		}

		now := time.Now()
		payment := models.Payment{
			TransactionID: transaction.ID,
			Status:        models.PaymentStatusSuccess,
			PaymentMethod: paymentMethod,
			PaidAt:        &now,
		}
		if err := paymentRepository.Create(&payment); err != nil {
			return err
		}

		transaction.Status = models.TransactionStatusSuccess
		if err := transactionRepository.Save(transaction); err != nil {
			return err
		}

		transactionID = transaction.ID
		paymentID = payment.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	transaction, err := s.transactionRepository.FindByID(transactionID)
	if err != nil {
		return nil, err
	}

	payment, err := s.paymentRepository.FindByID(paymentID)
	if err != nil {
		return nil, err
	}

	return &TransactionPaymentResult{Transaction: transaction, Payment: payment}, nil
}

func (s *TransactionService) ExpireWaitingForPaymentTransactions() error {
	now := time.Now()

	return s.db.Transaction(func(tx *gorm.DB) error {
		ticketRepository := s.ticketRepository.WithTx(tx)
		transactionRepository := s.transactionRepository.WithTx(tx)

		transactions, err := transactionRepository.FindExpiredWaitingForPayment(now)
		if err != nil {
			return err
		}

		for i := range transactions {
			transaction := &transactions[i]
			ticket, err := ticketRepository.FindByIDForUpdate(transaction.TicketID)
			if err != nil {
				return err
			}

			ticket.QuantityAvailable += transaction.Quantity
			if err := ticketRepository.Save(ticket); err != nil {
				return err
			}

			transaction.Status = models.TransactionStatusExpired
			if err := transactionRepository.Save(transaction); err != nil {
				return err
			}
		}

		return nil
	})
}

func isRetryableMySQLLockError(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}

	return mysqlErr.Number == 1205 || mysqlErr.Number == 1213
}

func generateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("tr_%d", time.Now().UnixNano())
	}

	return "tr_" + hex.EncodeToString(bytes)
}
