package limiter

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_AllScenarios(t *testing.T) {
	// 设置 Gin 测试模式
	gin.SetMode(gin.TestMode)

	// 定义限流器配置
	config := RateLimitConfig{
		MaxTokens:            1,
		RefillRate:           1,
		RefillInterval:       time.Second,
		KeyFunc:              func(c *gin.Context) string { return c.ClientIP() },
		BurstMultiplier:      1,
		Timeout:              time.Millisecond * 500,
		LimitExceededHandler: nil, // 使用默认的限流响应处理
		ExpirationDuration:   time.Minute * 5,
	}

	// 创建限流器中间件
	limiterMiddleware, err := NewRateLimiter(config)
	assert.NoError(t, err)

	router := gin.New()
	router.Use(limiterMiddleware)
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, world!")
	})

	clientIP := "192.168.1.1"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = clientIP

	// 第一次请求，应该成功
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 第二次请求，应该触发限流（因为 MaxTokens 是 1）
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// 等待足够的时间让令牌桶重新填充
	time.Sleep(1 * time.Second)

	// 第三次请求，应该成功
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_ValidateErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  RateLimitConfig
		wantErr bool
	}{
		{
			name: "MaxTokens is zero",
			config: RateLimitConfig{
				MaxTokens:          0,
				RefillRate:         1,
				RefillInterval:     time.Second,
				BurstMultiplier:    1,
				ExpirationDuration: time.Minute * 5,
			},
			wantErr: true,
		},
		{
			name: "RefillRate is zero",
			config: RateLimitConfig{
				MaxTokens:          1,
				RefillRate:         0,
				RefillInterval:     time.Second,
				BurstMultiplier:    1,
				ExpirationDuration: time.Minute * 5,
			},
			wantErr: true,
		},
		{
			name: "RefillInterval is zero",
			config: RateLimitConfig{
				MaxTokens:          1,
				RefillRate:         1,
				RefillInterval:     0,
				BurstMultiplier:    1,
				ExpirationDuration: time.Minute * 5,
			},
			wantErr: true,
		},
		{
			name: "BurstMultiplier is zero",
			config: RateLimitConfig{
				MaxTokens:          1,
				RefillRate:         1,
				RefillInterval:     time.Second,
				BurstMultiplier:    0,
				ExpirationDuration: time.Minute * 5,
			},
			wantErr: true,
		},
		{
			name: "ExpirationDuration less than RefillInterval",
			config: RateLimitConfig{
				MaxTokens:          1,
				RefillRate:         1,
				RefillInterval:     time.Second,
				BurstMultiplier:    1,
				ExpirationDuration: time.Millisecond * 500,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCleanupExpiredBuckets_AllScenarios(t *testing.T) {
	// 设置 Gin 测试模式
	gin.SetMode(gin.TestMode)

	config := RateLimitConfig{
		MaxTokens:            3,
		RefillRate:           1,
		RefillInterval:       time.Second,
		KeyFunc:              func(c *gin.Context) string { return c.ClientIP() },
		BurstMultiplier:      2,
		Timeout:              time.Second * 1,
		LimitExceededHandler: nil,                   // 使用默认的限流响应处理
		ExpirationDuration:   time.Millisecond * 10, // 设置为10毫秒以便快速过期
	}

	limiter := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		config:  config,
	}

	// 创建模拟请求
	clientIP := "192.168.1.2"
	bucket := &tokenBucket{
		tokens:         config.MaxTokens,
		lastRefill:     time.Now(),
		maxTokens:      config.MaxTokens * config.BurstMultiplier,
		refillRate:     config.RefillRate,
		refillInterval: config.RefillInterval,
	}
	limiter.buckets[clientIP] = bucket

	// 等待一段时间让令牌桶过期
	time.Sleep(time.Millisecond * 20)

	// 运行清理函数
	limiter.CleanupExpiredBuckets()

	// 确保令牌桶已被清理
	_, exists := limiter.buckets[clientIP]
	assert.False(t, exists, "Expected token bucket to be cleaned up")
}

func TestRateLimiterConfigValidationError(t *testing.T) {
	// 设置不合法的配置
	config := RateLimitConfig{
		MaxTokens:          0, // 非法的 MaxTokens 值
		RefillRate:         1,
		RefillInterval:     time.Second,
		BurstMultiplier:    2,
		ExpirationDuration: time.Minute * 5,
	}

	// 调用 NewRateLimiter，期望返回错误
	_, err := NewRateLimiter(config)
	assert.Error(t, err, "Expected validation error for MaxTokens = 0")
}

func TestRateLimiterDefaultLimitExceededHandler(t *testing.T) {
	// 设置 Gin 测试模式
	gin.SetMode(gin.TestMode)

	// 定义限流器配置
	config := RateLimitConfig{
		MaxTokens:            1,
		RefillRate:           1,
		RefillInterval:       time.Second,
		KeyFunc:              func(c *gin.Context) string { return c.ClientIP() },
		BurstMultiplier:      1,
		Timeout:              time.Millisecond * 500,
		LimitExceededHandler: nil, // 使用默认的限流响应处理
		ExpirationDuration:   time.Minute * 5,
	}

	// 创建限流器中间件
	limiterMiddleware, err := NewRateLimiter(config)
	assert.NoError(t, err)

	router := gin.New()
	router.Use(limiterMiddleware)
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, world!")
	})

	clientIP := "192.168.1.1"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = clientIP

	// 第一次请求，应该成功
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 第二次请求，应该触发限流并调用默认处理函数
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code) // 默认返回 429
}

func TestRateLimiterNoTimeout(t *testing.T) {
	// 设置 Gin 测试模式
	gin.SetMode(gin.TestMode)

	// 定义限流器配置
	config := RateLimitConfig{
		MaxTokens:            1,
		RefillRate:           1,
		RefillInterval:       time.Second,
		KeyFunc:              func(c *gin.Context) string { return c.ClientIP() },
		BurstMultiplier:      1,
		Timeout:              0,   // 设置 Timeout 为 0，确保直接进入 else 分支
		LimitExceededHandler: nil, // 使用默认的限流响应处理
		ExpirationDuration:   time.Minute * 5,
	}

	// 创建限流器中间件
	limiterMiddleware, err := NewRateLimiter(config)
	assert.NoError(t, err)

	router := gin.New()
	router.Use(limiterMiddleware)
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, world!")
	})

	clientIP := "192.168.1.1"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = clientIP

	// 第一次请求，应该成功
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 第二次请求，应该触发限流
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code) // 默认返回 429
}
