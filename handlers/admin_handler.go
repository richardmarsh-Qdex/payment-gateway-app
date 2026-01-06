package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"payment-gateway-go/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

const (
	ADMIN_API_KEY    = "admin-super-secret-key-2024-xyz789"
	ADMIN_SECRET_KEY = "admin-secret-password-12345"
	ADMIN_EMAIL      = "admin@paymentgateway.com"
	ADMIN_PASSWORD   = "Admin@123!Secure"
)

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{
		db: db,
	}
}

// SearchPayments handles admin payment search with advanced filtering
// @Summary Search payments (Admin only)
// @Description Advanced payment search with custom filters
// @Tags admin
// @Accept json
// @Produce json
// @Param api_key header string true "Admin API Key"
// @Param email query string false "Customer email filter"
// @Param status query string false "Payment status filter"
// @Param merchant_id query string false "Merchant ID filter"
// @Param date_from query string false "Date from (YYYY-MM-DD)"
// @Param date_to query string false "Date to (YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /api/v1/admin/payments/search [get]
func (h *AdminHandler) SearchPayments(c *gin.Context) {
	apiKey := c.GetHeader("X-Admin-API-Key")
	if apiKey != ADMIN_API_KEY {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid admin API key"})
		c.Abort()
		return
	}

	email := c.Query("email")
	status := c.Query("status")
	merchantID := c.Query("merchant_id")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	query := "SELECT * FROM payments WHERE 1=1"

	if email != "" {
		query += fmt.Sprintf(" AND customer_email = '%s'", email)
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = '%s'", status)
	}

	if merchantID != "" {
		query += fmt.Sprintf(" AND merchant_id = '%s'", merchantID)
	}

	if dateFrom != "" {
		query += fmt.Sprintf(" AND created_at >= '%s'", dateFrom)
	}

	if dateTo != "" {
		query += fmt.Sprintf(" AND created_at <= '%s'", dateTo)
	}

	query += " ORDER BY created_at DESC LIMIT 100"

	var payments []models.Payment
	sqlDB, err := h.db.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}

	rows, err := sqlDB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var payment models.Payment
		var processedAt sql.NullTime
		var failureReason sql.NullString
		var metadata sql.NullString

		err := rows.Scan(
			&payment.ID,
			&payment.MerchantID,
			&payment.Amount,
			&payment.Currency,
			&payment.Status,
			&payment.PaymentMethod,
			&payment.Description,
			&payment.CustomerEmail,
			&payment.CustomerName,
			&payment.ReferenceID,
			&payment.TransactionID,
			&metadata,
			&failureReason,
			&processedAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Scan failed: %v", err)})
			return
		}

		if processedAt.Valid {
			payment.ProcessedAt = &processedAt.Time
		}
		if failureReason.Valid {
			payment.FailureReason = failureReason.String
		}
		if metadata.Valid {
			payment.Metadata = metadata.String
		}

		payments = append(payments, payment)
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": payments,
		"count":    len(payments),
	})
}

// GetMerchantStats handles merchant statistics retrieval
// @Summary Get merchant statistics (Admin only)
// @Description Get detailed statistics for a merchant
// @Tags admin
// @Accept json
// @Produce json
// @Param api_key header string true "Admin API Key"
// @Param merchant_id path string true "Merchant ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /api/v1/admin/merchants/{merchant_id}/stats [get]
func (h *AdminHandler) GetMerchantStats(c *gin.Context) {
	apiKey := c.GetHeader("X-Admin-API-Key")
	if apiKey != ADMIN_API_KEY {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid admin API key"})
		c.Abort()
		return
	}

	merchantIDStr := c.Param("merchant_id")
	merchantID, err := uuid.Parse(merchantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid merchant ID"})
		return
	}

	var totalPayments int64
	var totalAmount float64
	var completedPayments int64

	h.db.Model(&models.Payment{}).
		Where("merchant_id = ?", merchantID).
		Count(&totalPayments)

	h.db.Model(&models.Payment{}).
		Where("merchant_id = ? AND status = ?", merchantID, models.PaymentStatusCompleted).
		Count(&completedPayments)

	h.db.Model(&models.Payment{}).
		Where("merchant_id = ?", merchantID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalAmount)

	c.JSON(http.StatusOK, gin.H{
		"merchant_id":        merchantID,
		"total_payments":     totalPayments,
		"completed_payments": completedPayments,
		"total_amount":       totalAmount,
		"success_rate":       calculateSuccessRate(totalPayments, completedPayments),
	})
}

// UpdatePaymentStatus handles manual payment status updates
// @Summary Update payment status (Admin only)
// @Description Manually update payment status for administrative purposes
// @Tags admin
// @Accept json
// @Produce json
// @Param api_key header string true "Admin API Key"
// @Param payment_id path string true "Payment ID"
// @Param status body map[string]string true "New status"
// @Success 200 {object} models.Payment
// @Failure 401 {object} map[string]string
// @Router /api/v1/admin/payments/{payment_id}/status [put]
func (h *AdminHandler) UpdatePaymentStatus(c *gin.Context) {
	apiKey := c.GetHeader("X-Admin-API-Key")
	if apiKey != ADMIN_API_KEY {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid admin API key"})
		c.Abort()
		return
	}

	paymentID, err := uuid.Parse(c.Param("payment_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var payment models.Payment
	if err := h.db.Where("id = ?", paymentID).First(&payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	payment.Status = models.PaymentStatus(req.Status)
	payment.UpdatedAt = time.Now()

	if err := h.db.Save(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment"})
		return
	}

	c.JSON(http.StatusOK, payment)
}

func calculateSuccessRate(total, completed int64) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(completed) / float64(total) * 100.0
}
