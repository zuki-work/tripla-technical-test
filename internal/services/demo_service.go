package services

import (
	"fmt"
	"sync"
	"time"

	"tripla-technical-test/internal/models"
)

const (
	defaultDemoAttempts = 10
	maxDemoAttempts     = 50
)

type ConcurrencyDemoResult struct {
	Attempt       int    `json:"attempt"`
	Status        string `json:"status"`
	TransactionID uint   `json:"transaction_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

type ConcurrencyDemoSummary struct {
	User         *models.User            `json:"user"`
	Ticket       *models.Ticket          `json:"ticket"`
	Attempts     int                     `json:"attempts"`
	SuccessCount int                     `json:"success_count"`
	FailedCount  int                     `json:"failed_count"`
	Results      []ConcurrencyDemoResult `json:"results"`
}

type DemoService struct {
	userService        *UserService
	ticketService      *TicketService
	transactionService *TransactionService
}

func NewDemoService(userService *UserService, ticketService *TicketService, transactionService *TransactionService) *DemoService {
	return &DemoService{
		userService:        userService,
		ticketService:      ticketService,
		transactionService: transactionService,
	}
}

func (s *DemoService) RunConcurrencyDemo(attempts int) (*ConcurrencyDemoSummary, error) {
	if attempts <= 0 {
		attempts = defaultDemoAttempts
	}
	if attempts > maxDemoAttempts {
		attempts = maxDemoAttempts
	}

	now := time.Now().UnixNano()
	user, err := s.userService.CreateUser(&models.User{
		Name:  "Concurrency Demo User",
		Email: fmt.Sprintf("concurrency-demo-%d@example.com", now),
	})
	if err != nil {
		return nil, err
	}

	ticket, err := s.ticketService.CreateTicket(&models.Ticket{
		Name:          fmt.Sprintf("VIP Concurrency Demo %d", now),
		QuantityTotal: 1,
	})
	if err != nil {
		return nil, err
	}

	results := make([]ConcurrencyDemoResult, attempts)
	var wg sync.WaitGroup

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			transaction, err := s.transactionService.CreateTransaction(user.ID, ticket.ID, 1)
			if err != nil {
				results[index] = ConcurrencyDemoResult{
					Attempt: index + 1,
					Status:  "failed",
					Error:   err.Error(),
				}
				return
			}

			results[index] = ConcurrencyDemoResult{
				Attempt:       index + 1,
				Status:        "success",
				TransactionID: transaction.ID,
			}
		}(i)
	}

	wg.Wait()

	successCount := 0
	for _, result := range results {
		if result.Status == "success" {
			successCount++
		}
	}

	finalTicket, err := s.ticketService.GetTicket(ticket.ID)
	if err != nil {
		return nil, err
	}

	return &ConcurrencyDemoSummary{
		User:         user,
		Ticket:       finalTicket,
		Attempts:     attempts,
		SuccessCount: successCount,
		FailedCount:  attempts - successCount,
		Results:      results,
	}, nil
}
