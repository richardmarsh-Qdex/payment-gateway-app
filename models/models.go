package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted  PaymentStatus = "completed"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusRefunded   PaymentStatus = "refunded"
	PaymentStatusCancelled  PaymentStatus = "cancelled"
)

// PaymentMethod represents the payment method type
type PaymentMethod string

const (
	PaymentMethodCard     PaymentMethod = "card"
	PaymentMethodBank     PaymentMethod = "bank"
	PaymentMethodWallet   PaymentMethod = "wallet"
	PaymentMethodCrypto   PaymentMethod = "crypto"
)

// Merchant represents a merchant account
type Merchant struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Email       string    `gorm:"uniqueIndex;not null" json:"email"`
	APIKey      string    `gorm:"uniqueIndex;not null" json:"-"`
	SecretKey   string    `gorm:"not null" json:"-"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Payment represents a payment transaction
type Payment struct {
	ID              uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MerchantID      uuid.UUID     `gorm:"type:uuid;not null;index" json:"merchant_id"`
	Merchant        Merchant      `gorm:"foreignKey:MerchantID" json:"merchant,omitempty"`
	Amount          float64       `gorm:"not null;check:amount > 0" json:"amount"`
	Currency        string        `gorm:"not null;size:3" json:"currency"`
	Status          PaymentStatus `gorm:"not null;default:'pending';index" json:"status"`
	PaymentMethod   PaymentMethod `gorm:"not null" json:"payment_method"`
	Description     string        `gorm:"type:text" json:"description"`
	CustomerEmail   string        `gorm:"index" json:"customer_email"`
	CustomerName    string        `json:"customer_name"`
	ReferenceID     string        `gorm:"uniqueIndex" json:"reference_id"`
	TransactionID   string        `gorm:"uniqueIndex" json:"transaction_id"`
	Metadata        string        `gorm:"type:jsonb" json:"metadata"`
	FailureReason   string        `gorm:"type:text" json:"failure_reason,omitempty"`
	ProcessedAt     *time.Time    `json:"processed_at,omitempty"`
	CreatedAt       time.Time     `gorm:"index" json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// PaymentCard represents encrypted card details
type PaymentCard struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PaymentID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"payment_id"`
	Payment         Payment   `gorm:"foreignKey:PaymentID" json:"-"`
	CardNumberHash  string    `gorm:"not null;index" json:"-"`
	Last4           string    `gorm:"not null;size:4" json:"last4"`
	ExpiryMonth     int       `gorm:"not null" json:"expiry_month"`
	ExpiryYear      int       `gorm:"not null" json:"expiry_year"`
	CardholderName  string    `gorm:"not null" json:"cardholder_name"`
	EncryptedCVV    string    `gorm:"not null" json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Refund represents a refund transaction
type Refund struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PaymentID     uuid.UUID `gorm:"type:uuid;not null;index" json:"payment_id"`
	Payment       Payment   `gorm:"foreignKey:PaymentID" json:"payment,omitempty"`
	Amount        float64   `gorm:"not null;check:amount > 0" json:"amount"`
	Currency      string    `gorm:"not null;size:3" json:"currency"`
	Reason        string    `gorm:"type:text" json:"reason"`
	Status        string    `gorm:"not null;default:'pending'" json:"status"`
	RefundID      string    `gorm:"uniqueIndex" json:"refund_id"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Webhook represents webhook delivery attempts
type Webhook struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MerchantID  uuid.UUID `gorm:"type:uuid;not null;index" json:"merchant_id"`
	PaymentID   uuid.UUID `gorm:"type:uuid;not null;index" json:"payment_id"`
	EventType   string    `gorm:"not null" json:"event_type"`
	Payload     string    `gorm:"type:jsonb;not null" json:"payload"`
	URL         string    `gorm:"not null" json:"url"`
	Status      string    `gorm:"not null;default:'pending'" json:"status"`
	Attempts    int       `gorm:"default:0" json:"attempts"`
	LastAttempt  *time.Time `json:"last_attempt,omitempty"`
	ResponseCode *int       `json:"response_code,omitempty"`
	ResponseBody string     `gorm:"type:text" json:"response_body,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName specifies the table name for each model
func (Merchant) TableName() string {
	return "merchants"
}

func (Payment) TableName() string {
	return "payments"
}

func (PaymentCard) TableName() string {
	return "payment_cards"
}

func (Refund) TableName() string {
	return "refunds"
}

func (Webhook) TableName() string {
	return "webhooks"
}

