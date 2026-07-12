package database

import (
	"fmt"
	"log"
	"os"

	"tripla-technical-test/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Connect establishes connection to MySQL database and performs auto-migration.
func Connect() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	log.Println("Database connection established successfully")

	// Run Auto-migrations
	log.Println("Running database migrations...")
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Ticket{},
		&models.Transaction{},
		&models.Payment{},
	); err != nil {
		return fmt.Errorf("database auto-migration failed: %w", err)
	}
	log.Println("Database migration completed successfully")

	return nil
}
