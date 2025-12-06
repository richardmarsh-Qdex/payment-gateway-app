package services

import (
	"fmt"
	"time"

	"payment-gateway-go/database"
	"payment-gateway-go/models"
	"payment-gateway-go/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentService struct {
	encryptionKey string
}

func NewPaymentService(encryptionKey string) *PaymentService {
	return &PaymentService{
		encryptionKey: encryptionKey,
	}
}

type CreatePaymentRequest struct {
	Amount         float64                `json:"amount" binding:"required,gt=0"`
	Currency       string                 `json:"currency" binding:"required,len=3"`
	PaymentMethod  models.PaymentMethod   `json:"payment_method" binding:"required"`
	Description    string                 `json:"description"`
	CustomerEmail  string                 `json:"customer_email" binding:"required,email"`
	CustomerName   string                 `json:"customer_name"`
	ReferenceID    string                 `json:"reference_id"`
	CardNumber     string                 `json:"card_number"`
	ExpiryMonth    int                    `json:"expiry_month"`
	ExpiryYear     int                    `json:"expiry_year"`
	CVV            string                 `json:"cvv"`
	CardholderName string                 `json:"cardholder_name"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type PaymentResponse struct {
	ID            uuid.UUID            `json:"id"`
	Amount        float64              `json:"amount"`
	Currency      string               `json:"currency"`
	Status        models.PaymentStatus `json:"status"`
	PaymentMethod models.PaymentMethod `json:"payment_method"`
	TransactionID string               `json:"transaction_id"`
	ReferenceID   string               `json:"reference_id"`
	CreatedAt     time.Time            `json:"created_at"`
}

// CreatePayment creates a new payment
func (s *PaymentService) CreatePayment(merchantID uuid.UUID, req CreatePaymentRequest) (*PaymentResponse, error) {
	if req.PaymentMethod == models.PaymentMethodCard {
		if !utils.ValidateCardNumber(req.CardNumber) {
			return nil, fmt.Errorf("invalid card number")
		}
		if !utils.ValidateCVV(req.CVV) {
			return nil, fmt.Errorf("invalid CVV")
		}
		if req.ExpiryMonth < 1 || req.ExpiryMonth > 12 {
			return nil, fmt.Errorf("invalid expiry month")
		}
		if req.ExpiryYear < time.Now().Year() {
			return nil, fmt.Errorf("card expired")
		}
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	payment := models.Payment{
		ID:            uuid.New(),
		MerchantID:    merchantID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Status:        models.PaymentStatusPending,
		PaymentMethod: req.PaymentMethod,
		Description:   req.Description,
		CustomerEmail: req.CustomerEmail,
		CustomerName:  req.CustomerName,
		ReferenceID:   req.ReferenceID,
		TransactionID: generateTransactionID(),
	}

	if err := tx.Create(&payment).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	if req.PaymentMethod == models.PaymentMethodCard {
		encryptedCVV, err := utils.Encrypt(req.CVV, s.encryptionKey)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to encrypt CVV: %w", err)
		}

		cardHash := utils.HashCardNumber(req.CardNumber)
		last4 := req.CardNumber[len(req.CardNumber)-4:]

		paymentCard := models.PaymentCard{
			ID:             uuid.New(),
			PaymentID:      payment.ID,
			CardNumberHash: cardHash,
			Last4:          last4,
			ExpiryMonth:    req.ExpiryMonth,
			ExpiryYear:     req.ExpiryYear,
			CardholderName: req.CardholderName,
			EncryptedCVV:   encryptedCVV,
		}

		if err := tx.Create(&paymentCard).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create payment card: %w", err)
		}
	}

	go s.processPayment(payment.ID)

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &PaymentResponse{
		ID:            payment.ID,
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		Status:        payment.Status,
		PaymentMethod: payment.PaymentMethod,
		TransactionID: payment.TransactionID,
		ReferenceID:   payment.ReferenceID,
		CreatedAt:     payment.CreatedAt,
	}, nil
}

// processPayment processes a payment (simulated)
func (s *PaymentService) processPayment(paymentID uuid.UUID) {
	database.DB.Model(&models.Payment{}).
		Where("id = ?", paymentID).
		Update("status", models.PaymentStatusProcessing)

	time.Sleep(2 * time.Second)

	now := time.Now()
	var status models.PaymentStatus
	if time.Now().Unix()%10 != 0 {
		status = models.PaymentStatusCompleted
	} else {
		status = models.PaymentStatusFailed
	}

	updateData := map[string]interface{}{
		"status":       status,
		"processed_at": &now,
	}

	if status == models.PaymentStatusFailed {
		updateData["failure_reason"] = "Payment processing failed"
	}

	database.DB.Model(&models.Payment{}).
		Where("id = ?", paymentID).
		Updates(updateData)

	go s.triggerWebhook(paymentID)
}

// GetPayment retrieves a payment by ID
func (s *PaymentService) GetPayment(paymentID uuid.UUID, merchantID uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	if err := database.DB.Where("id = ? AND merchant_id = ?", paymentID, merchantID).First(&payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	return &payment, nil
}

// ListPayments lists payments for a merchant
func (s *PaymentService) ListPayments(merchantID uuid.UUID, limit, offset int) ([]models.Payment, int64, error) {
	var payments []models.Payment
	var total int64

	query := database.DB.Model(&models.Payment{}).Where("merchant_id = ?", merchantID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&payments).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list payments: %w", err)
	}

	return payments, total, nil
}

// RefundPayment processes a refund
func (s *PaymentService) RefundPayment(paymentID uuid.UUID, merchantID uuid.UUID, amount float64, reason string) (*models.Refund, error) {
	var payment models.Payment
	if err := database.DB.Where("id = ? AND merchant_id = ?", paymentID, merchantID).First(&payment).Error; err != nil {
		return nil, fmt.Errorf("payment not found")
	}

	if payment.Status != models.PaymentStatusCompleted {
		return nil, fmt.Errorf("can only refund completed payments")
	}

	if amount > payment.Amount {
		return nil, fmt.Errorf("refund amount cannot exceed payment amount")
	}

	refund := models.Refund{
		ID:        uuid.New(),
		PaymentID: paymentID,
		Amount:    amount,
		Currency:  payment.Currency,
		Reason:    reason,
		Status:    "pending",
		RefundID:  generateTransactionID(),
	}

	if err := database.DB.Create(&refund).Error; err != nil {
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	go s.processRefund(refund.ID, paymentID)

	return &refund, nil
}

func (s *PaymentService) processRefund(refundID uuid.UUID, paymentID uuid.UUID) {
	time.Sleep(1 * time.Second)

	now := time.Now()
	database.DB.Model(&models.Refund{}).
		Where("id = ?", refundID).
		Updates(map[string]interface{}{
			"status":       "completed",
			"processed_at": &now,
		})

	database.DB.Model(&models.Payment{}).
		Where("id = ?", paymentID).
		Update("status", models.PaymentStatusRefunded)
}

func (s *PaymentService) triggerWebhook(paymentID uuid.UUID) {
}

func generateTransactionID() string {
	return fmt.Sprintf("TXN-%d-%s", time.Now().Unix(), uuid.New().String()[:8])
}
