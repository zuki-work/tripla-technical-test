package models

import "gorm.io/gorm"

type Ticket struct {
	gorm.Model
	Name              string `json:"name" binding:"required"`
	QuantityTotal     uint   `json:"quantity_total" binding:"required"`
	QuantityAvailable uint   `json:"quantity_available"`
	StockVersion      uint   `json:"stock_version"`
}
