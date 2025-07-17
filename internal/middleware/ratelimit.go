package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"

	"github.com/webpage-analyser-server/internal/constants"
	"github.com/webpage-analyser-server/internal/models"
)

// Rate limiting per IP address
type RateLimiter struct {
	ips   map[string]*rate.Limiter
	mu    *sync.RWMutex
	rate  rate.Limit
	burst int
}


func NewRateLimiter() *RateLimiter {
	requestsPerMinute := viper.GetFloat64("rate_limit.requests_per_minute")
	if requestsPerMinute == 0 {
		requestsPerMinute = constants.DefaultRequestsPerMinute
	}

	return &RateLimiter{
		ips:   make(map[string]*rate.Limiter),
		mu:    &sync.RWMutex{},
		rate:  rate.Limit(requestsPerMinute / 60.0), // Convert to requests per second
		burst: int(requestsPerMinute * constants.DefaultRateLimitBurstFactor),
	}
}

// getLimiter returns the rate limiter for an IP address
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.ips[ip] = limiter
	}

	return limiter
}

// RateLimit middleware implements rate limiting
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting if disabled
		if !viper.GetBool("rate_limit.enabled") {
			c.Next()
			return
		}

		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			c.JSON(constants.StatusTooManyRequests, models.ErrorResponse{
				Code:    constants.StatusTooManyRequests,
				Message: "Rate limit exceeded",
				Details: "Please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// cleanup removes old limiters periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(constants.DefaultRateLimitCleanupTimeout)
	for range ticker.C {
		rl.mu.Lock()
		rl.ips = make(map[string]*rate.Limiter)
		rl.mu.Unlock()
	}
} 