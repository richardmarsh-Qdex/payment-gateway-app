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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := database.Init(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	middleware.InitAuth(cfg.JWT.SecretKey)
	middleware.InitRateLimit(cfg)

	paymentService := services.NewPaymentService(cfg.Security.EncryptionKey, cfg.Security.PBKDF2Iterations)

	paymentHandler := handlers.NewPaymentHandler(paymentService)
	merchantHandler := handlers.NewMerchantHandler()
	healthHandler := handlers.NewHealthHandler()

	router := setupRouter(paymentHandler, merchantHandler, healthHandler, cfg)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		log.Printf("Server starting on %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

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

	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS())

	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/ready", healthHandler.ReadinessCheck)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/merchants", merchantHandler.CreateMerchant)
		v1.POST("/merchants/token", merchantHandler.GenerateToken)

		protected := v1.Group("")
		protected.Use(middleware.RateLimit())
		protected.Use(middleware.AuthenticateAPIKey())
		{
			protected.GET("/merchants/profile", merchantHandler.GetMerchantProfile)

			protected.POST("/payments", paymentHandler.CreatePayment)
			protected.GET("/payments", paymentHandler.ListPayments)
			protected.GET("/payments/:id", paymentHandler.GetPayment)
			protected.POST("/payments/:id/refund", paymentHandler.RefundPayment)
		}
	}

	return router
}

