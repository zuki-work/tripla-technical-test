package repositories

import (
	"errors"
	"math/rand"
	"os"
	"strings"

	"tripla-technical-test/internal/models"
)

const (
	AccountingMockModeForceSuccess = "force_success"
	AccountingMockModeForceFail    = "force_fail"
	AccountingMockModeRandom       = "random"
)

var (
	ErrAccountingKnownError          = errors.New("accounting service returned a known error")
	ErrAccountingInternalServerError = errors.New("accounting service returned internal server error")
)

type AccountingRepository struct{}

func NewAccountingRepository() *AccountingRepository {
	return &AccountingRepository{}
}

func (r *AccountingRepository) SendTransaction(transaction *models.Transaction) error {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("ACCOUNTING_MOCK_MODE")))

	switch mode {
	case AccountingMockModeForceSuccess:
		return nil
	case AccountingMockModeForceFail:
		return ErrAccountingInternalServerError
	case "", AccountingMockModeRandom:
		return randomAccountingResult()
	default:
		return randomAccountingResult()
	}
}

func randomAccountingResult() error {
	roll := rand.Intn(100)
	switch {
	case roll < 90:
		return nil
	case roll < 95:
		return ErrAccountingKnownError
	default:
		return ErrAccountingInternalServerError
	}
}
