package services

import (
	"errors"
	"time"

	"tripla-technical-test/internal/models"
	"tripla-technical-test/internal/repositories"

	"gorm.io/gorm"
)

const transactionExpiryDuration = 10 * time.Minute

var (
	ErrInvalidTransactionQuantity = errors.New("transaction quantity must be greater than zero")
	ErrInsufficientTickets        = errors.New("not enough tickets available")
	ErrTransactionNotPending      = errors.New("transaction is not pending")
	ErrTransactionExpired         = errors.New("transaction has expired")
)

type TransactionPaymentResult struct {
	Transaction *models.Transaction `json:"transaction"`
	Payment     *models.Payment     `json:"payment"`
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

	if err := s.ExpirePendingTransactions(); err != nil {
		return nil, err
	}

	var transactionID uint
	err := s.db.Transaction(func(tx *gorm.DB) error {
		ticketRepository := s.ticketRepository.WithTx(tx)
		transactionRepository := s.transactionRepository.WithTx(tx)

		ticket, err := ticketRepository.FindByIDForUpdate(ticketID)
		if err != nil {
			return err
		}

		if ticket.QuantityAvailable < quantity {
			return ErrInsufficientTickets
		}

		ticket.QuantityAvailable -= quantity
		if err := ticketRepository.Save(ticket); err != nil {
			return err
		}

		transaction := models.Transaction{
			UserID:    userID,
			TicketID:  ticketID,
			Quantity:  quantity,
			Status:    models.TransactionStatusPending,
			ExpiresAt: time.Now().Add(transactionExpiryDuration),
		}
		if err := transactionRepository.Create(&transaction); err != nil {
			return err
		}

		transactionID = transaction.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.transactionRepository.FindByID(transactionID)
}

func (s *TransactionService) GetTransactions() ([]models.Transaction, error) {
	if err := s.ExpirePendingTransactions(); err != nil {
		return nil, err
	}

	return s.transactionRepository.FindAll()
}

func (s *TransactionService) GetTransaction(id uint) (*models.Transaction, error) {
	if err := s.ExpirePendingTransactions(); err != nil {
		return nil, err
	}

	return s.transactionRepository.FindByID(id)
}

func (s *TransactionService) PayTransaction(id uint, paymentMethod string) (*TransactionPaymentResult, error) {
	if err := s.ExpirePendingTransactions(); err != nil {
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

		if transaction.Status != models.TransactionStatusPending {
			return ErrTransactionNotPending
		}
		if !transaction.ExpiresAt.After(time.Now()) {
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

		transaction.Status = models.TransactionStatusConfirmed
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

func (s *TransactionService) ExpirePendingTransactions() error {
	now := time.Now()

	return s.db.Transaction(func(tx *gorm.DB) error {
		ticketRepository := s.ticketRepository.WithTx(tx)
		transactionRepository := s.transactionRepository.WithTx(tx)

		transactions, err := transactionRepository.FindExpiredPending(now)
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
