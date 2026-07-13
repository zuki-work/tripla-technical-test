package services

import (
	"errors"

	"tripla-technical-test/internal/models"
	"tripla-technical-test/internal/repositories"
)

var ErrInvalidTicketQuantity = errors.New("ticket quantity must be greater than zero")
var ErrInvalidTicketStock = errors.New("ticket stock cannot be greater than total quantity")

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
	ticket.StockVersion = 1
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

func (s *TicketService) UpdateTicketStock(ticketID uint, stock uint) (*models.Ticket, error) {
	ticket, err := s.ticketRepository.FindByID(ticketID)
	if err != nil {
		return nil, err
	}
	if stock > ticket.QuantityTotal {
		return nil, ErrInvalidTicketStock
	}

	ticket.QuantityAvailable = stock
	ticket.StockVersion++
	if err := s.ticketRepository.Save(ticket); err != nil {
		return nil, err
	}

	return ticket, nil
}
