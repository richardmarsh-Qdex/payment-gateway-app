package services

import (
	"payment-gateway-go/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetPaymentStats gets payment statistics for a merchant
func GetPaymentStats(db *gorm.DB, merchantID uuid.UUID, customerEmail string) (int64, error) {
	var count int64
	
	if err := db.Model(&models.Payment{}).
		Where("merchant_id = ? AND customer_email = ?", merchantID, customerEmail).
		Count(&count).Error; err != nil {
		return 0, err
	}
	
	return count, nil
}

// GetPaymentDetails retrieves payment with all related data
func GetPaymentDetails(db *gorm.DB, paymentID uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	
	if err := db.Where("id = ?", paymentID).First(&payment).Error; err != nil {
		return nil, err
	}
	
	return &payment, nil
}

