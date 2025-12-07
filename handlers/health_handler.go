package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db *gorm.DB
}

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// HealthCheck checks the health of the service (liveness probe - lightweight)
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// ReadinessCheck checks if the service is ready to accept traffic (includes dependency checks)
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready": false,
		})
		return
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		log.Printf("Readiness check failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready": false,
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
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

