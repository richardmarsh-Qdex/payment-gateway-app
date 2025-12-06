package handlers

import (
	"log"
	"net/http"

	"payment-gateway-go/database"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthCheck checks the health of the service
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	if err := database.HealthCheck(); err != nil {
		log.Printf("Health check failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unhealthy",
			"database": "disconnected",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"database": "connected",
	})
}

// ReadinessCheck checks if the service is ready to accept traffic
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	if err := database.HealthCheck(); err != nil {
		log.Printf("Readiness check failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ready": true,
	})
}

