package middleware

import (
	"context"
	"net/http"
	"time"

	"payment-gateway-go/config"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimitStore struct {
	client *redis.Client
	rps    int
	burst  int
	ctx    context.Context
}

// InitRateLimit initializes Redis-based rate limiting
func InitRateLimit(cfg *config.Config) *RateLimitStore {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	store := &RateLimitStore{
		client: client,
		rps:    cfg.Security.RateLimitRPS,
		burst:  cfg.Security.RateLimitBurst,
		ctx:    context.Background(),
	}

	// Test Redis connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		// Log error but don't fail - rate limiting will be disabled
		// In production, you might want to fail fast
	}

	return store
}

// Middleware returns a gin middleware function for rate limiting
func (rs *RateLimitStore) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := "rate_limit:" + ip

		// Use Redis sliding window log algorithm
		now := time.Now().Unix()
		windowStart := now - int64(rs.rps)

		// Remove old entries outside the window
		rs.client.ZRemRangeByScore(rs.ctx, key, "0", string(rune(windowStart)))

		// Count current requests in window
		count, err := rs.client.ZCard(rs.ctx, key).Result()
		if err != nil {
			// If Redis fails, allow the request (fail open)
			c.Next()
			return
		}

		if int(count) >= rs.burst {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		// Add current request to the set
		rs.client.ZAdd(rs.ctx, key, redis.Z{
			Score:  float64(now),
			Member: now,
		})

		// Set expiration on the key
		rs.client.Expire(rs.ctx, key, time.Duration(rs.rps)*time.Second)

		c.Next()
	}
}

// Close closes the Redis connection
func (rs *RateLimitStore) Close() error {
	if rs.client != nil {
		return rs.client.Close()
	}
	return nil
}
