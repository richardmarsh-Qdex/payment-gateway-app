package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"payment-gateway-go/middleware"
	"payment-gateway-go/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type MerchantHandler struct {
	db *gorm.DB
}

func NewMerchantHandler(db *gorm.DB) *MerchantHandler {
	return &MerchantHandler{db: db}
}

type CreateMerchantRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type MerchantResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	APIKey    string    `json:"api_key"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateMerchant creates a new merchant account
func (h *MerchantHandler) CreateMerchant(c *gin.Context) {
	var req CreateMerchantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existingMerchant models.Merchant
	err := h.db.Where("email = ?", req.Email).First(&existingMerchant).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}

	secretKey, err := generateSecretKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate secret key"})
		return
	}

	// Hash the secret key before storing
	secretKeyHash, err := bcrypt.GenerateFromPassword([]byte(secretKey), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash secret key"})
		return
	}

	merchant := models.Merchant{
		Name:         req.Name,
		Email:        req.Email,
		APIKey:       apiKey,
		SecretKeyHash: string(secretKeyHash),
		IsActive:     true,
	}

	if err := h.db.Create(&merchant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create merchant"})
		return
	}

	c.JSON(http.StatusCreated, MerchantResponse{
		ID:        merchant.ID,
		Name:      merchant.Name,
		Email:     merchant.Email,
		APIKey:    merchant.APIKey,
		IsActive:  merchant.IsActive,
		CreatedAt: merchant.CreatedAt,
	})
}

// GetMerchantProfile retrieves merchant profile
func (h *MerchantHandler) GetMerchantProfile(c *gin.Context) {
	merchantID, ok := middleware.GetMerchantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var merchant models.Merchant
	if err := h.db.Where("id = ?", merchantID).First(&merchant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Merchant not found"})
		return
	}

	type ProfileResponse struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		IsActive  bool      `json:"is_active"`
		CreatedAt time.Time `json:"created_at"`
	}

	c.JSON(http.StatusOK, ProfileResponse{
		ID:        merchant.ID,
		Name:      merchant.Name,
		Email:     merchant.Email,
		IsActive:  merchant.IsActive,
		CreatedAt: merchant.CreatedAt,
	})
}

// GenerateToken generates JWT token for merchant
func (h *MerchantHandler) GenerateToken(c *gin.Context) {
	var req struct {
		APIKey    string `json:"api_key" binding:"required"`
		SecretKey string `json:"secret_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var merchant models.Merchant
	if err := h.db.Where("api_key = ? AND is_active = ?", req.APIKey, true).First(&merchant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred"})
		}
		return
	}

	// Compare secret key using bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(merchant.SecretKeyHash), []byte(req.SecretKey)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := middleware.GenerateJWT(merchant.ID, merchant.Email, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_in": 900,
	})
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "pk_" + base64.URLEncoding.EncodeToString(bytes), nil
}

func generateSecretKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "sk_" + base64.URLEncoding.EncodeToString(bytes), nil
}


