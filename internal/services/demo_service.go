package services

import (
	"fmt"
	"sync"
	"time"

	"tripla-technical-test/internal/models"
	"tripla-technical-test/internal/repositories"
)

const (
	defaultDemoAttempts           = 10
	maxDemoAttempts               = 50
	defaultHighTrafficRequests    = 1000
	maxHighTrafficRequests        = 10000
	defaultHighTrafficTicketStock = 1000
	defaultHighTrafficWorkerCount = 10
	maxHighTrafficWorkerCount     = 100
	defaultHighTrafficWorkerBatch = 100
	maxHighTrafficWorkerBatch     = 1000
)

type ConcurrencyDemoResult struct {
	Attempt       int    `json:"attempt"`
	TransactionID uint   `json:"transaction_id,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	Status        string `json:"status"`
	FailedReason  string `json:"failed_reason,omitempty"`
	Error         string `json:"error,omitempty"`
}

type ConcurrencyDemoSummary struct {
	User                   *models.User            `json:"user"`
	Ticket                 *models.Ticket          `json:"ticket"`
	Attempts               int                     `json:"attempts"`
	PendingCreatedCount    int                     `json:"pending_created_count"`
	WaitingForPaymentCount int                     `json:"waiting_for_payment_count"`
	FailedCount            int                     `json:"failed_count"`
	Results                []ConcurrencyDemoResult `json:"results"`
}

type HighTrafficDemoSummary struct {
	User                   *models.User   `json:"user"`
	Ticket                 *models.Ticket `json:"ticket"`
	RequestedCount         int            `json:"requested_count"`
	TicketStock            uint           `json:"ticket_stock"`
	WorkerCount            int            `json:"worker_count"`
	WorkerBatchSize        int            `json:"worker_batch_size"`
	StoredPendingCount     int            `json:"stored_pending_count"`
	ProcessedCount         int            `json:"processed_count"`
	WaitingForPaymentCount int            `json:"waiting_for_payment_count"`
	FailedCount            int            `json:"failed_count"`
	CreateDurationMs       int64          `json:"create_duration_ms"`
	ProcessDurationMs      int64          `json:"process_duration_ms"`
	TotalDurationMs        int64          `json:"total_duration_ms"`
}

type DemoService struct {
	userService            *UserService
	ticketService          *TicketService
	transactionService     *TransactionService
	externalDataRepository *repositories.ExternalDataRepository
}

func NewDemoService(userService *UserService, ticketService *TicketService, transactionService *TransactionService, externalDataRepository *repositories.ExternalDataRepository) *DemoService {
	return &DemoService{
		userService:            userService,
		ticketService:          ticketService,
		transactionService:     transactionService,
		externalDataRepository: externalDataRepository,
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
	for i := 0; i < attempts; i++ {
		transaction, err := s.transactionService.CreateTransaction(user.ID, ticket.ID, 1)
		if err != nil {
			results[i] = ConcurrencyDemoResult{
				Attempt: i + 1,
				Status:  "create_failed",
				Error:   err.Error(),
			}
			continue
		}

		results[i] = ConcurrencyDemoResult{
			Attempt:       i + 1,
			TransactionID: transaction.ID,
			RequestID:     transaction.RequestID,
			Status:        transaction.Status,
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.transactionService.ProcessPendingTransactions(1)
		}()
	}
	wg.Wait()

	pendingCreatedCount := 0
	waitingForPaymentCount := 0
	failedCount := 0
	for i := range results {
		if results[i].Status == models.TransactionStatusPending {
			pendingCreatedCount++
		}
		if results[i].TransactionID == 0 {
			continue
		}

		transaction, err := s.transactionService.GetTransaction(results[i].TransactionID)
		if err != nil {
			results[i].Error = err.Error()
			continue
		}

		results[i].Status = transaction.Status
		results[i].FailedReason = transaction.FailedReason
		if transaction.Status == models.TransactionStatusWaitingForPayment {
			waitingForPaymentCount++
		}
		if transaction.Status == models.TransactionStatusFailed {
			failedCount++
		}
	}

	finalTicket, err := s.ticketService.GetTicket(ticket.ID)
	if err != nil {
		return nil, err
	}

	return &ConcurrencyDemoSummary{
		User:                   user,
		Ticket:                 finalTicket,
		Attempts:               attempts,
		PendingCreatedCount:    pendingCreatedCount,
		WaitingForPaymentCount: waitingForPaymentCount,
		FailedCount:            failedCount,
		Results:                results,
	}, nil
}

func (s *DemoService) RunHighTrafficDemo(requestCount int, ticketStock uint, workerCount int, workerBatchSize int) (*HighTrafficDemoSummary, error) {
	if requestCount <= 0 {
		requestCount = defaultHighTrafficRequests
	}
	if requestCount > maxHighTrafficRequests {
		requestCount = maxHighTrafficRequests
	}
	if ticketStock == 0 {
		ticketStock = defaultHighTrafficTicketStock
	}
	if workerCount <= 0 {
		workerCount = defaultHighTrafficWorkerCount
	}
	if workerCount > maxHighTrafficWorkerCount {
		workerCount = maxHighTrafficWorkerCount
	}
	if workerBatchSize <= 0 {
		workerBatchSize = defaultHighTrafficWorkerBatch
	}
	if workerBatchSize > maxHighTrafficWorkerBatch {
		workerBatchSize = maxHighTrafficWorkerBatch
	}

	startedAt := time.Now()
	now := startedAt.UnixNano()
	user, err := s.userService.CreateUser(&models.User{
		Name:  "High Traffic Demo User",
		Email: fmt.Sprintf("high-traffic-demo-%d@example.com", now),
	})
	if err != nil {
		return nil, err
	}

	ticket, err := s.ticketService.CreateTicket(&models.Ticket{
		Name:          fmt.Sprintf("High Traffic Demo Ticket %d", now),
		QuantityTotal: ticketStock,
	})
	if err != nil {
		return nil, err
	}

	createStartedAt := time.Now()
	storedPendingCount := 0
	for i := 0; i < requestCount; i++ {
		transaction, err := s.transactionService.CreateTransaction(user.ID, ticket.ID, 1)
		if err != nil {
			return nil, err
		}
		if transaction.Status == models.TransactionStatusPending {
			storedPendingCount++
		}
	}
	createDuration := time.Since(createStartedAt)

	processStartedAt := time.Now()
	processedCount := 0
	waitingForPaymentCount := 0
	failedCount := 0
	var mu sync.Mutex
	var wg sync.WaitGroup
	var processErr error

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				mu.Lock()
				shouldContinue := processedCount < requestCount && processErr == nil
				mu.Unlock()
				if !shouldContinue {
					return
				}

				result, err := s.transactionService.ProcessPendingTransactionsWithoutExpiry(workerBatchSize)
				if err != nil {
					mu.Lock()
					if processErr == nil {
						processErr = err
					}
					mu.Unlock()
					return
				}
				if result.ProcessedCount == 0 {
					return
				}

				mu.Lock()
				processedCount += result.ProcessedCount
				waitingForPaymentCount += result.WaitingForPaymentCount
				failedCount += result.FailedCount
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if processErr != nil {
		return nil, processErr
	}
	processDuration := time.Since(processStartedAt)

	finalTicket, err := s.ticketService.GetTicket(ticket.ID)
	if err != nil {
		return nil, err
	}

	return &HighTrafficDemoSummary{
		User:                   user,
		Ticket:                 finalTicket,
		RequestedCount:         requestCount,
		TicketStock:            ticketStock,
		WorkerCount:            workerCount,
		WorkerBatchSize:        workerBatchSize,
		StoredPendingCount:     storedPendingCount,
		ProcessedCount:         processedCount,
		WaitingForPaymentCount: waitingForPaymentCount,
		FailedCount:            failedCount,
		CreateDurationMs:       createDuration.Milliseconds(),
		ProcessDurationMs:      processDuration.Milliseconds(),
		TotalDurationMs:        time.Since(startedAt).Milliseconds(),
	}, nil
}

type DuplicatePaymentWebhookDemoResult struct {
	Attempt   int    `json:"attempt"`
	PaymentID uint   `json:"payment_id,omitempty"`
	Duplicate bool   `json:"duplicate"`
	Error     string `json:"error,omitempty"`
}

type DuplicatePaymentWebhookDemoSummary struct {
	User                   *models.User                        `json:"user,omitempty"`
	Ticket                 *models.Ticket                      `json:"ticket,omitempty"`
	Transaction            *models.Transaction                 `json:"transaction"`
	ExternalPaymentID      string                              `json:"external_payment_id"`
	ConcurrentRequests     int                                 `json:"concurrent_requests"`
	CreatedCount           int                                 `json:"created_count"`
	DuplicateCount         int                                 `json:"duplicate_count"`
	PaymentCountInDatabase int64                               `json:"payment_count_in_database"`
	Results                []DuplicatePaymentWebhookDemoResult `json:"results"`
}

func (s *DemoService) RunDuplicatePaymentWebhookDemo(transactionID uint, externalPaymentID string, concurrentRequests int) (*DuplicatePaymentWebhookDemoSummary, error) {
	if concurrentRequests <= 0 {
		concurrentRequests = 2
	}
	if concurrentRequests > 20 {
		concurrentRequests = 20
	}
	if externalPaymentID == "" {
		externalPaymentID = fmt.Sprintf("pay_duplicate_demo_%d", time.Now().UnixNano())
	}

	var user *models.User
	var ticket *models.Ticket
	if transactionID == 0 {
		now := time.Now().UnixNano()
		createdUser, err := s.userService.CreateUser(&models.User{
			Name:  "Duplicate Webhook Demo User",
			Email: fmt.Sprintf("duplicate-webhook-demo-%d@example.com", now),
		})
		if err != nil {
			return nil, err
		}
		user = createdUser

		createdTicket, err := s.ticketService.CreateTicket(&models.Ticket{
			Name:          fmt.Sprintf("Duplicate Webhook Demo Ticket %d", now),
			QuantityTotal: 1,
		})
		if err != nil {
			return nil, err
		}
		ticket = createdTicket

		transaction, err := s.transactionService.CreateTransaction(user.ID, ticket.ID, 1)
		if err != nil {
			return nil, err
		}
		transactionID = transaction.ID

		if _, err := s.transactionService.ProcessPendingTransactions(1); err != nil {
			return nil, err
		}
	}

	results := make([]DuplicatePaymentWebhookDemoResult, concurrentRequests)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start

			result, err := s.transactionService.HandlePaymentWebhook(externalPaymentID, transactionID, models.PaymentStatusSuccess, "third_party", nil)
			results[index] = DuplicatePaymentWebhookDemoResult{Attempt: index + 1}
			if err != nil {
				results[index].Error = err.Error()
				return
			}

			results[index].Duplicate = result.Duplicate
			if result.Payment != nil {
				results[index].PaymentID = result.Payment.ID
			}
		}(i)
	}
	close(start)
	wg.Wait()

	createdCount := 0
	duplicateCount := 0
	for i := range results {
		if results[i].Error != "" {
			continue
		}
		if results[i].Duplicate {
			duplicateCount++
		} else {
			createdCount++
		}
	}

	paymentCount, err := s.transactionService.CountPaymentsByExternalPaymentID(externalPaymentID)
	if err != nil {
		return nil, err
	}

	transaction, err := s.transactionService.GetTransaction(transactionID)
	if err != nil {
		return nil, err
	}
	if ticket == nil && transaction.TicketID != 0 {
		ticket, _ = s.ticketService.GetTicket(transaction.TicketID)
	}

	return &DuplicatePaymentWebhookDemoSummary{
		User:                   user,
		Ticket:                 ticket,
		Transaction:            transaction,
		ExternalPaymentID:      externalPaymentID,
		ConcurrentRequests:     concurrentRequests,
		CreatedCount:           createdCount,
		DuplicateCount:         duplicateCount,
		PaymentCountInDatabase: paymentCount,
		Results:                results,
	}, nil
}

type StockUpdatePayload struct {
	TicketID uint `json:"ticket_id"`
	Stock    uint `json:"stock"`
	Version  uint `json:"version"`
}

type StockUpdateDeliveryResult struct {
	Payload      StockUpdatePayload   `json:"payload"`
	Applied      bool                 `json:"applied"`
	ExternalData *models.ExternalData `json:"external_data"`
}

type OutOfOrderStockDemoSummary struct {
	UpdatesReceivedOrder []StockUpdateDeliveryResult `json:"updates_received_order"`
	ExternalFinalStocks  []*models.ExternalData      `json:"external_final_stocks"`
}

func (s *DemoService) RunOutOfOrderStockDemo(updates []StockUpdatePayload) (*OutOfOrderStockDemoSummary, error) {
	if len(updates) == 0 {
		updates = []StockUpdatePayload{
			{TicketID: 1, Stock: 2, Version: 2},
			{TicketID: 1, Stock: 5, Version: 1},
		}
	}

	deliveryResults := make([]StockUpdateDeliveryResult, 0, len(updates))
	finalStockByTicketID := make(map[uint]*models.ExternalData)
	for _, update := range updates {
		externalData, applied, err := s.externalDataRepository.ApplyTicketStock(update.TicketID, update.Stock, update.Version)
		if err != nil {
			return nil, err
		}

		finalStockByTicketID[update.TicketID] = externalData
		deliveryResults = append(deliveryResults, StockUpdateDeliveryResult{
			Payload:      update,
			Applied:      applied,
			ExternalData: externalData,
		})
	}

	finalStocks := make([]*models.ExternalData, 0, len(finalStockByTicketID))
	for _, externalData := range finalStockByTicketID {
		finalStocks = append(finalStocks, externalData)
	}

	return &OutOfOrderStockDemoSummary{
		UpdatesReceivedOrder: deliveryResults,
		ExternalFinalStocks:  finalStocks,
	}, nil
}
