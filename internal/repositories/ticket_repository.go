package repositories

import (
	"tripla-technical-test/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TicketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

func (r *TicketRepository) WithTx(tx *gorm.DB) *TicketRepository {
	return &TicketRepository{db: tx}
}

func (r *TicketRepository) Create(ticket *models.Ticket) error {
	return r.db.Create(ticket).Error
}

func (r *TicketRepository) FindAll() ([]models.Ticket, error) {
	var tickets []models.Ticket
	if err := r.db.Find(&tickets).Error; err != nil {
		return nil, err
	}

	return tickets, nil
}

func (r *TicketRepository) FindByID(id uint) (*models.Ticket, error) {
	var ticket models.Ticket
	if err := r.db.First(&ticket, id).Error; err != nil {
		return nil, err
	}

	return &ticket, nil
}

func (r *TicketRepository) FindByIDForUpdate(id uint) (*models.Ticket, error) {
	var ticket models.Ticket
	if err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&ticket, id).Error; err != nil {
		return nil, err
	}

	return &ticket, nil
}

func (r *TicketRepository) Save(ticket *models.Ticket) error {
	return r.db.Save(ticket).Error
}
