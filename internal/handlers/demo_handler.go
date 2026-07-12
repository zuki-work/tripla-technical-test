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

type highTrafficDemoRequest struct {
	RequestCount    int  `json:"request_count"`
	TicketStock     uint `json:"ticket_stock"`
	WorkerCount     int  `json:"worker_count"`
	WorkerBatchSize int  `json:"worker_batch_size"`
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

func (h *DemoHandler) RunHighTrafficDemo(c *gin.Context) {
	var input highTrafficDemoRequest
	if err := c.ShouldBindJSON(&input); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.demoService.RunHighTrafficDemo(input.RequestCount, input.TicketStock, input.WorkerCount, input.WorkerBatchSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
