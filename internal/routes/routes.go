package routes

import (
	"tripla-technical-test/internal/handlers"

	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine, userHandler *handlers.UserHandler, ticketHandler *handlers.TicketHandler, transactionHandler *handlers.TransactionHandler, demoHandler *handlers.DemoHandler) {
	router.GET("/", handlers.Home)
	router.GET("/health", handlers.Health)

	router.POST("/users", userHandler.CreateUser)
	router.GET("/users", userHandler.GetUsers)

	router.POST("/tickets", ticketHandler.CreateTicket)
	router.GET("/tickets", ticketHandler.GetTickets)
	router.GET("/tickets/:id", ticketHandler.GetTicket)

	router.POST("/transactions", transactionHandler.CreateTransaction)
	router.GET("/transactions", transactionHandler.GetTransactions)
	router.POST("/workers/transactions/process-pending", transactionHandler.ProcessPendingTransactions)
	router.GET("/transactions/:id", transactionHandler.GetTransaction)
	router.POST("/transactions/:id/pay", transactionHandler.PayTransaction)

	router.POST("/demo/concurrency", demoHandler.RunConcurrencyDemo)
	router.POST("/demo/high-traffic", demoHandler.RunHighTrafficDemo)
}
