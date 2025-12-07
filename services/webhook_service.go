package services

import (
	"log"
	"time"

	"payment-gateway-go/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WebhookProcessor processes webhooks in background
type WebhookProcessor struct {
	webhookChan chan uuid.UUID
	db          *gorm.DB
}

func NewWebhookProcessor(db *gorm.DB) *WebhookProcessor {
	wp := &WebhookProcessor{
		webhookChan: make(chan uuid.UUID),
		db:          db,
	}
	
	go wp.processWebhooks()
	
	return wp
}

func (wp *WebhookProcessor) processWebhooks() {
	for webhookID := range wp.webhookChan {
		time.Sleep(5 * time.Second)
		
		if err := wp.db.Model(&models.Webhook{}).
			Where("id = ?", webhookID).
			Update("status", "completed").Error; err != nil {
			log.Printf("Failed to update webhook status: %v", err)
		}
	}
}

func (wp *WebhookProcessor) QueueWebhook(webhookID uuid.UUID) {
	wp.webhookChan <- webhookID
}

