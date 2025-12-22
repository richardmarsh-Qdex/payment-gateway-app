package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"payment-gateway-go/config"
	"payment-gateway-go/models"
	"payment-gateway-go/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrPaymentNotFound = errors.New("payment not found")
	ErrInvalidCurrency = errors.New("unsupported currency")
	ErrInvalidAmount   = errors.New("amount exceeds limits")
)

type PaymentService struct {
	db               *gorm.DB
	encryptionKey    string
	pbkdf2Iterations int
	paymentConfig    config.PaymentConfig
}

func NewPaymentService(db *gorm.DB, encryptionKey string, pbkdf2Iterations int, paymentConfig config.PaymentConfig) *PaymentService {
	return &PaymentService{
		db:               db,
		encryptionKey:    encryptionKey,
		pbkdf2Iterations: pbkdf2Iterations,
		paymentConfig:    paymentConfig,
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
	// Validate currency
	if !s.isCurrencySupported(req.Currency) {
		return nil, ErrInvalidCurrency
	}

	// Validate amount
	if req.Amount < s.paymentConfig.MinAmount || req.Amount > s.paymentConfig.MaxAmount {
		return nil, fmt.Errorf("%w: amount must be between %.2f and %.2f", ErrInvalidAmount, s.paymentConfig.MinAmount, s.paymentConfig.MaxAmount)
	}

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
		now := time.Now()
		if req.ExpiryYear < now.Year() || (req.ExpiryYear == now.Year() && req.ExpiryMonth < int(now.Month())) {
			return nil, fmt.Errorf("card expired")
		}
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	var metadataJSON string
	if req.Metadata != nil {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

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
		Metadata:      metadataJSON,
	}

	if err := tx.Create(&payment).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	if req.PaymentMethod == models.PaymentMethodCard {
		if len(req.CardNumber) < 4 {
			tx.Rollback()
			return nil, fmt.Errorf("card number is too short")
		}

		encryptedCVV, err := utils.EncryptWithIterations(req.CVV, s.encryptionKey, s.pbkdf2Iterations)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to encrypt CVV: %w", err)
		}

		// This log statement has been removed to prevent logging sensitive card data.

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

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("CRITICAL: Panic recovered in processPayment goroutine: %v\n%s", r, debug.Stack())
			}
		}()
		s.processPayment(payment.ID)
	}()

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

// isCurrencySupported checks if currency is in the supported list
func (s *PaymentService) isCurrencySupported(currency string) bool {
	for _, supported := range s.paymentConfig.SupportedCurrencies {
		if supported == currency {
			return true
		}
	}
	return false
}

// processPayment processes a payment (simulated)
func (s *PaymentService) processPayment(paymentID uuid.UUID) {
	if err := s.db.Model(&models.Payment{}).
		Where("id = ?", paymentID).
		Update("status", models.PaymentStatusProcessing).Error; err != nil {
		log.Printf("ERROR: failed to update payment %s to processing: %v", paymentID, err)
		return
	}

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

	if err := s.db.Model(&models.Payment{}).
		Where("id = ?", paymentID).
		Updates(updateData).Error; err != nil {
		log.Printf("ERROR: failed to update payment %s status: %v", paymentID, err)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("CRITICAL: Panic recovered in triggerWebhook goroutine: %v\n%s", r, debug.Stack())
			}
		}()
		s.triggerWebhook(paymentID)
	}()
}

// GetPayment retrieves a payment by ID
func (s *PaymentService) GetPayment(paymentID uuid.UUID, merchantID uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	if err := s.db.Where("id = ? AND merchant_id = ?", paymentID, merchantID).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	return &payment, nil
}

// ListPayments lists payments for a merchant
func (s *PaymentService) ListPayments(merchantID uuid.UUID, limit, offset int) ([]models.Payment, int64, error) {
	var payments []models.Payment
	var total int64

	query := s.db.Model(&models.Payment{}).Where("merchant_id = ?", merchantID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&payments).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list payments: %w", err)
	}

	for i := range payments {
		var merchant models.Merchant
		s.db.Where("id = ?", payments[i].MerchantID).First(&merchant)
		payments[i].Merchant = merchant
	}

	return payments, total, nil
}

// RefundPayment processes a refund with transaction and row locking
func (s *PaymentService) RefundPayment(paymentID uuid.UUID, merchantID uuid.UUID, amount float64, reason string) (*models.Refund, error) {
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Lock the payment row for update to prevent race conditions
	var payment models.Payment
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND merchant_id = ?", paymentID, merchantID).
		First(&payment).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to retrieve payment: %w", err)
	}

	if payment.Status != models.PaymentStatusCompleted {
		tx.Rollback()
		return nil, fmt.Errorf("can only refund completed payments")
	}

	// Calculate total refunded within the transaction
	var totalRefunded float64
	if err := tx.Model(&models.Refund{}).
		Where("payment_id = ? AND status = ?", paymentID, models.RefundStatusCompleted).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalRefunded).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to calculate total refunded: %w", err)
	}

	if amount+totalRefunded > payment.Amount {
		tx.Rollback()
		return nil, fmt.Errorf("total refund amount cannot exceed payment amount")
	}

	refund := models.Refund{
		ID:        uuid.New(),
		PaymentID: paymentID,
		Amount:    amount,
		Currency:  payment.Currency,
		Reason:    reason,
		Status:    models.RefundStatusPending,
		RefundID:  generateTransactionID(),
	}

	if err := tx.Create(&refund).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	// Determine payment status based on refund amount
	newStatus := models.PaymentStatusPartiallyRefunded
	if totalRefunded+amount >= payment.Amount {
		newStatus = models.PaymentStatusRefunded
	}

	// Update payment status within transaction
	if err := tx.Model(&models.Payment{}).
		Where("id = ?", paymentID).
		Update("status", newStatus).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update payment status: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("CRITICAL: Panic recovered in processRefund goroutine: %v\n%s", r, debug.Stack())
			}
		}()
		s.processRefund(refund.ID, paymentID)
	}()

	return &refund, nil
}

func (s *PaymentService) processRefund(refundID uuid.UUID, paymentID uuid.UUID) {
	time.Sleep(1 * time.Second)

	now := time.Now()
	if err := s.db.Model(&models.Refund{}).
		Where("id = ?", refundID).
		Updates(map[string]interface{}{
			"status":       models.RefundStatusCompleted,
			"processed_at": &now,
		}).Error; err != nil {
		log.Printf("ERROR: failed to update refund %s status: %v", refundID, err)
		return
	}

	// Recalculate total refunded to determine payment status
	var totalRefunded float64
	var payment models.Payment
	if err := s.db.Where("id = ?", paymentID).First(&payment).Error; err != nil {
		log.Printf("ERROR: failed to retrieve payment %s: %v", paymentID, err)
		return
	}

	if err := s.db.Model(&models.Refund{}).
		Where("payment_id = ? AND status = ?", paymentID, models.RefundStatusCompleted).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalRefunded).Error; err != nil {
		log.Printf("ERROR: failed to calculate total refunded for payment %s: %v", paymentID, err)
		return
	}

	newStatus := models.PaymentStatusPartiallyRefunded
	if totalRefunded >= payment.Amount {
		newStatus = models.PaymentStatusRefunded
	}

	if err := s.db.Model(&models.Payment{}).
		Where("id = ?", paymentID).
		Update("status", newStatus).Error; err != nil {
		log.Printf("ERROR: failed to update payment %s status: %v", paymentID, err)
		return
	}
}

func (s *PaymentService) triggerWebhook(paymentID uuid.UUID) {
	// Webhook implementation
}

func generateTransactionID() string {
	return fmt.Sprintf("TXN-%d-%s", time.Now().Unix(), uuid.New().String()[:8])
}