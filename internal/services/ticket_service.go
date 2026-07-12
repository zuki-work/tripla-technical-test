package services

import (
	"errors"

	"tripla-technical-test/internal/models"
	"tripla-technical-test/internal/repositories"
)

var ErrInvalidTicketQuantity = errors.New("ticket quantity must be greater than zero")

type TicketService struct {
	ticketRepository *repositories.TicketRepository
}

func NewTicketService(ticketRepository *repositories.TicketRepository) *TicketService {
	return &TicketService{ticketRepository: ticketRepository}
}

func (s *TicketService) CreateTicket(ticket *models.Ticket) (*models.Ticket, error) {
	if ticket.QuantityTotal == 0 {
		return nil, ErrInvalidTicketQuantity
	}

	ticket.QuantityAvailable = ticket.QuantityTotal
	if err := s.ticketRepository.Create(ticket); err != nil {
		return nil, err
	}

	return ticket, nil
}

func (s *TicketService) GetTickets() ([]models.Ticket, error) {
	return s.ticketRepository.FindAll()
}

func (s *TicketService) GetTicket(id uint) (*models.Ticket, error) {
	return s.ticketRepository.FindByID(id)
}
