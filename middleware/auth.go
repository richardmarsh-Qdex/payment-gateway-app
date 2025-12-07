package middleware

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"payment-gateway-go/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Claims struct {
	MerchantID uuid.UUID `json:"merchant_id"`
	Email      string    `json:"email"`
	jwt.RegisteredClaims
}

var jwtSecret []byte

// InitAuth initializes JWT secret
func InitAuth(secret string) {
	jwtSecret = []byte(secret)
}

// Authenticate supports both API Key and JWT token authentication
func Authenticate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try API Key first
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			var merchant models.Merchant
			if err := db.Where("api_key = ? AND is_active = ?", apiKey, true).First(&merchant).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
				} else {
					log.Printf("Database error during API key auth: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred"})
				}
				c.Abort()
				return
			}

			c.Set("merchant_id", merchant.ID)
			c.Set("merchant", merchant)
			c.Next()
			return
		}

		// Try JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key or authorization token required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("merchant_id", claims.MerchantID)
		c.Set("email", claims.Email)
		c.Next()
	}
}

// AuthenticateAPIKey authenticates requests using API key (deprecated, use Authenticate instead)
func AuthenticateAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "This endpoint requires authentication"})
		c.Abort()
	}
}

// AuthenticateJWT authenticates requests using JWT token
func AuthenticateJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("merchant_id", claims.MerchantID)
		c.Set("email", claims.Email)
		c.Next()
	}
}

// GenerateJWT generates a JWT token for a merchant
func GenerateJWT(merchantID uuid.UUID, email string, expiry time.Duration) (string, error) {
	claims := &Claims{
		MerchantID: merchantID,
		Email:      email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// GetMerchantID retrieves merchant ID from context
func GetMerchantID(c *gin.Context) (uuid.UUID, bool) {
	merchantID, exists := c.Get("merchant_id")
	if !exists {
		return uuid.Nil, false
	}
	if id, ok := merchantID.(uuid.UUID); ok {
		return id, true
	}
	return uuid.Nil, false
}

