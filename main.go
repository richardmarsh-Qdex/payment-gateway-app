package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-gateway-go/config"
	"payment-gateway-go/database"
	"payment-gateway-go/handlers"
	"payment-gateway-go/middleware"
	"payment-gateway-go/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	if err := database.Init(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize middleware
	middleware.InitAuth(cfg.JWT.SecretKey)
	middleware.InitRateLimit(cfg)

	// Initialize services
	paymentService := services.NewPaymentService(cfg.Security.EncryptionKey)

	// Initialize handlers
	paymentHandler := handlers.NewPaymentHandler(paymentService)
	merchantHandler := handlers.NewMerchantHandler()
	healthHandler := handlers.NewHealthHandler()

	// Setup router
	router := setupRouter(paymentHandler, merchantHandler, healthHandler, cfg)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRouter(
	paymentHandler *handlers.PaymentHandler,
	merchantHandler *handlers.MerchantHandler,
	healthHandler *handlers.HealthHandler,
	cfg *config.Config,
) *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS())

	// Health check endpoints (no auth required)
	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/ready", healthHandler.ReadinessCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes
		v1.POST("/merchants", merchantHandler.CreateMerchant)
		v1.POST("/merchants/token", merchantHandler.GenerateToken)

		// Protected routes (API Key authentication)
		protected := v1.Group("")
		protected.Use(middleware.RateLimit())
		protected.Use(middleware.AuthenticateAPIKey())
		{
			// Merchant routes
			protected.GET("/merchants/profile", merchantHandler.GetMerchantProfile)

			// Payment routes
			protected.POST("/payments", paymentHandler.CreatePayment)
			protected.GET("/payments", paymentHandler.ListPayments)
			protected.GET("/payments/:id", paymentHandler.GetPayment)
			protected.POST("/payments/:id/refund", paymentHandler.RefundPayment)
		}
	}

	return router
}

