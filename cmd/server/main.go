package main

import (
	"log"
	"os"

	"tripla-technical-test/internal/database"
	"tripla-technical-test/internal/handlers"

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

	router := gin.Default()

	router.GET("/", handlers.Home)
	router.GET("/health", handlers.Health)

	router.POST("/users", handlers.CreateUser)
	router.GET("/users", handlers.GetUsers)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
