package main

import (
	"log"
	"os"

	"tripla-technical-test/internal/database"
	"tripla-technical-test/internal/handlers"
	"tripla-technical-test/internal/repositories"
	"tripla-technical-test/internal/routes"
	"tripla-technical-test/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("Info: No .env file found or error loading it")
	}

	// Initialize MySQL connection and run migrations
	if err := database.Connect(); err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	userRepository := repositories.NewUserRepository(database.DB)
	userService := services.NewUserService(userRepository)
	userHandler := handlers.NewUserHandler(userService)

	ticketRepository := repositories.NewTicketRepository(database.DB)
	ticketService := services.NewTicketService(ticketRepository)
	ticketHandler := handlers.NewTicketHandler(ticketService)

	transactionRepository := repositories.NewTransactionRepository(database.DB)
	paymentRepository := repositories.NewPaymentRepository(database.DB)
	accountingRepository := repositories.NewAccountingRepository()
	transactionService := services.NewTransactionService(database.DB, ticketRepository, transactionRepository, paymentRepository, accountingRepository)
	transactionHandler := handlers.NewTransactionHandler(transactionService)
	webhookHandler := handlers.NewWebhookHandler(transactionService)

	demoService := services.NewDemoService(userService, ticketService, transactionService)
	demoHandler := handlers.NewDemoHandler(demoService)

	router := gin.Default()
	routes.Register(router, userHandler, ticketHandler, transactionHandler, webhookHandler, demoHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
