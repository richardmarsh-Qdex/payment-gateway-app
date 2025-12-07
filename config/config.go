package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Security SecurityConfig
	Payment  PaymentConfig
}

type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	Environment  string
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	SecretKey     string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type SecurityConfig struct {
	EncryptionKey  string
	RateLimitRPS   int
	RateLimitBurst int
	PBKDF2Iterations int
}

type PaymentConfig struct {
	DefaultCurrency string
	SupportedCurrencies []string
	MinAmount       float64
	MaxAmount       float64
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		if cfg := os.Getenv("ENVIRONMENT"); cfg != "production" {
			log.Println("Warning: .env file not found or could not be loaded")
		}
	}

	environment := getEnv("ENVIRONMENT", "development")

	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
			Environment:  environment,
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			DBName:          getEnv("DB_NAME", "payment_gateway"),
			SSLMode:         getEnv("DB_SSLMODE", "require"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 15*time.Minute),
			ConnMaxIdleTime: getDurationEnv("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		Security: SecurityConfig{
			RateLimitRPS:    getIntEnv("RATE_LIMIT_RPS", 100),
			RateLimitBurst:  getIntEnv("RATE_LIMIT_BURST", 200),
			PBKDF2Iterations: getIntEnv("PBKDF2_ITERATIONS", 600000),
		},
		Payment: PaymentConfig{
			DefaultCurrency:    getEnv("DEFAULT_CURRENCY", "USD"),
			SupportedCurrencies: getEnvStringSlice("SUPPORTED_CURRENCIES", []string{"USD", "EUR", "GBP", "JPY", "CAD", "AUD"}),
			MinAmount:          getFloatEnv("MIN_PAYMENT_AMOUNT", 0.01),
			MaxAmount:          getFloatEnv("MAX_PAYMENT_AMOUNT", 100000.00),
		},
	}

	// Require JWT_SECRET_KEY - no default allowed
	jwtSecret, err := getRequiredEnv("JWT_SECRET_KEY")
	if err != nil {
		return nil, fmt.Errorf("missing required environment variable: JWT_SECRET_KEY")
	}
	config.JWT.SecretKey = jwtSecret
	config.JWT.AccessExpiry = getDurationEnv("JWT_ACCESS_EXPIRY", 15*time.Minute)
	config.JWT.RefreshExpiry = getDurationEnv("JWT_REFRESH_EXPIRY", 7*24*time.Hour)

	// Require ENCRYPTION_KEY - no default allowed
	encryptionKey, err := getRequiredEnv("ENCRYPTION_KEY")
	if err != nil {
		return nil, fmt.Errorf("missing required environment variable: ENCRYPTION_KEY")
	}
	if len(encryptionKey) < 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be at least 32 characters")
	}
	config.Security.EncryptionKey = encryptionKey

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		} else {
			log.Printf("Warning: could not parse env var %s, using default value %d. Error: %v", key, defaultValue, err)
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		} else {
			log.Printf("Warning: could not parse env var %s, using default value %f. Error: %v", key, defaultValue, err)
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		} else {
			log.Printf("Warning: could not parse env var %s, using default value %v. Error: %v", key, defaultValue, err)
		}
	}
	return defaultValue
}

func getRequiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("missing required environment variable: %s", key)
	}
	return value, nil
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	result := []string{}
	for _, item := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}

