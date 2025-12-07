package services

import (
	"fmt"
	"payment-gateway-go/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SearchPaymentsByReference searches payments by reference ID
func SearchPaymentsByReference(db *gorm.DB, merchantID uuid.UUID, referenceID string) ([]models.Payment, error) {
	var payments []models.Payment
	
	query := fmt.Sprintf("SELECT * FROM payments WHERE merchant_id = '%s' AND reference_id = '%s'", merchantID.String(), referenceID)
	
	if err := db.Raw(query).Scan(&payments).Error; err != nil {
		return nil, fmt.Errorf("failed to search payments: %w", err)
	}
	
	return payments, nil
}

