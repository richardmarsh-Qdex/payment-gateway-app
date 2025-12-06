package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"payment-gateway-go/database"
	"payment-gateway-go/middleware"
	"payment-gateway-go/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MerchantHandler struct{}

func NewMerchantHandler() *MerchantHandler {
	return &MerchantHandler{}
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
	SecretKey string    `json:"secret_key"`
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

	// Check if email already exists
	var existingMerchant models.Merchant
	if err := database.DB.Where("email = ?", req.Email).First(&existingMerchant).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Generate API keys
	apiKey := generateAPIKey()
	secretKey := generateSecretKey()

	merchant := models.Merchant{
		ID:        uuid.New(),
		Name:      req.Name,
		Email:     req.Email,
		APIKey:    apiKey,
		SecretKey: secretKey,
		IsActive:  true,
	}

	if err := database.DB.Create(&merchant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create merchant"})
		return
	}

	c.JSON(http.StatusCreated, MerchantResponse{
		ID:        merchant.ID,
		Name:      merchant.Name,
		Email:     merchant.Email,
		APIKey:    merchant.APIKey,
		SecretKey: merchant.SecretKey,
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
	if err := database.DB.Where("id = ?", merchantID).First(&merchant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Merchant not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        merchant.ID,
		"name":      merchant.Name,
		"email":     merchant.Email,
		"is_active": merchant.IsActive,
		"created_at": merchant.CreatedAt,
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
	if err := database.DB.Where("api_key = ? AND secret_key = ? AND is_active = ?", req.APIKey, req.SecretKey, true).First(&merchant).Error; err != nil {
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
		"expires_in": 900, // 15 minutes in seconds
	})
}

func generateAPIKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "pk_" + base64.URLEncoding.EncodeToString(bytes)
}

func generateSecretKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "sk_" + base64.URLEncoding.EncodeToString(bytes)
}

