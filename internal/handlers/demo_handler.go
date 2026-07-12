package handlers

import (
	"errors"
	"io"
	"net/http"

	"tripla-technical-test/internal/services"

	"github.com/gin-gonic/gin"
)

type DemoHandler struct {
	demoService *services.DemoService
}

func NewDemoHandler(demoService *services.DemoService) *DemoHandler {
	return &DemoHandler{demoService: demoService}
}

type concurrencyDemoRequest struct {
	Attempts int `json:"attempts"`
}

func (h *DemoHandler) RunConcurrencyDemo(c *gin.Context) {
	var input concurrencyDemoRequest
	if err := c.ShouldBindJSON(&input); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.demoService.RunConcurrencyDemo(input.Attempts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
