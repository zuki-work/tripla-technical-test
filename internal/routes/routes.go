package routes

import (
	"tripla-technical-test/internal/handlers"

	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine, userHandler *handlers.UserHandler) {
	router.GET("/", handlers.Home)
	router.GET("/health", handlers.Health)

	router.POST("/users", userHandler.CreateUser)
	router.GET("/users", userHandler.GetUsers)
}
