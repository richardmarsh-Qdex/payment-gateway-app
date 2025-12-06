package middleware

import (
	"net/http"
	"sync"
	"time"

	"payment-gateway-go/config"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type rateLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimitStore struct {
	visitors map[string]*rateLimiter
	mu       sync.RWMutex
	rps      int
	burst    int
}

var store *RateLimitStore

// InitRateLimit initializes rate limiting
func InitRateLimit(cfg *config.Config) {
	store = &RateLimitStore{
		visitors: make(map[string]*rateLimiter),
		rps:      cfg.Security.RateLimitRPS,
		burst:    cfg.Security.RateLimitBurst,
	}

	go func() {
		for {
			time.Sleep(5 * time.Minute)
			store.cleanup()
		}
	}()
}

func (rs *RateLimitStore) getVisitor(ip string) *rateLimiter {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	visitor, exists := rs.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(rs.rps), rs.burst)
		visitor = &rateLimiter{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		rs.visitors[ip] = visitor
	} else {
		visitor.lastSeen = time.Now()
	}

	return visitor
}

func (rs *RateLimitStore) cleanup() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	for ip, visitor := range rs.visitors {
		if time.Since(visitor.lastSeen) > 10*time.Minute {
			delete(rs.visitors, ip)
		}
	}
}

// RateLimit middleware limits requests per IP
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		visitor := store.getVisitor(ip)

		if !visitor.limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

