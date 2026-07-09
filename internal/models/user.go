package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" gorm:"type:varchar(255);uniqueIndex" binding:"required,email"`
}
