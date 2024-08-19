package limiter

import (
	"errors"
	"github.com/gin-gonic/gin"
	"sync"
	"time"
)

type RateLimitConfig struct {
	MaxTokens            int
	RefillRate           int
	RefillInterval       time.Duration
	KeyFunc              func(*gin.Context) string
	BurstMultiplier      int
	Timeout              time.Duration
	LimitExceededHandler gin.HandlerFunc
	ExpirationDuration   time.Duration
}

type tokenBucket struct {
	tokens         int
	lastRefill     time.Time
	maxTokens      int
	refillRate     int
	refillInterval time.Duration
	mutex          sync.Mutex
}

type RateLimiter struct {
	buckets map[string]*tokenBucket
	config  RateLimitConfig
	mutex   sync.RWMutex
}

func NewRateLimiter(config RateLimitConfig) (gin.HandlerFunc, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	limiter := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		config:  config,
	}

	return limiter.RateLimitMiddleware(), nil
}

func (rl *RateLimiter) getBucket(key string) *tokenBucket {
	rl.mutex.RLock()
	bucket, exists := rl.buckets[key]
	rl.mutex.RUnlock()

	if !exists {
		rl.mutex.Lock()
		defer rl.mutex.Unlock()

		if bucket, exists = rl.buckets[key]; !exists {
			bucket = &tokenBucket{
				tokens:         rl.config.MaxTokens,
				lastRefill:     time.Now(),
				maxTokens:      rl.config.MaxTokens * rl.config.BurstMultiplier,
				refillRate:     rl.config.RefillRate,
				refillInterval: rl.config.RefillInterval,
			}
			rl.buckets[key] = bucket
		}
	}

	return bucket
}

func (rl *RateLimiter) CleanupExpiredBuckets() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	for key, bucket := range rl.buckets {
		bucket.mutex.Lock()
		if now.Sub(bucket.lastRefill) > rl.config.ExpirationDuration {
			delete(rl.buckets, key)
		}
		bucket.mutex.Unlock()
	}
}

func defaultLimitExceededHandler(c *gin.Context) {
	c.AbortWithStatus(429)
}

func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := rl.config.KeyFunc(c)
		bucket := rl.getBucket(key)

		bucket.mutex.Lock()
		defer bucket.mutex.Unlock()

		now := time.Now()
		elapsed := now.Sub(bucket.lastRefill)
		refillTokens := int(elapsed.Seconds()/rl.config.RefillInterval.Seconds()) * bucket.refillRate

		if refillTokens > 0 {
			bucket.tokens = minInt(bucket.tokens+refillTokens, bucket.maxTokens)
			bucket.lastRefill = now
		}

		if bucket.tokens >= 1 {
			bucket.tokens--
			c.Next()
		} else {
			if rl.config.Timeout > 0 {
				timer := time.NewTimer(rl.config.Timeout)
				select {
				case <-timer.C:
					handler := rl.config.LimitExceededHandler
					if handler == nil {
						handler = defaultLimitExceededHandler
					}
					handler(c)
					c.Abort()
				case <-c.Done():
					timer.Stop()
				}
			} else {
				handler := rl.config.LimitExceededHandler
				if handler == nil {
					handler = defaultLimitExceededHandler
				}
				handler(c)
				c.Abort()
			}
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (r *RateLimitConfig) Validate() error {
	if r.MaxTokens <= 0 {
		return errors.New("MaxTokens must be greater than 0")
	}
	if r.RefillRate <= 0 {
		return errors.New("RefillRate must be greater than 0")
	}
	if r.RefillInterval <= 0 {
		return errors.New("RefillInterval must be greater than 0")
	}
	if r.BurstMultiplier <= 0 {
		return errors.New("BurstMultiplier must be greater than 0")
	}
	if r.ExpirationDuration <= r.RefillInterval {
		return errors.New("ExpirationDuration must be greater than RefillInterval")
	}
	return nil
}
