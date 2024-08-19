# gin-ratelimiter

This repository contains a Gin middleware that implements rate limiting based on the token bucket algorithm. It allows you to control the request rate to your Gin-based web applications, supporting burst traffic and providing customizable configurations.

This README is also available in [Chinese/中文说明](#中文说明).

# English Version

## Features

- **Token Bucket Algorithm**: Controls request rate while allowing bursts.
- **Customizable**: Configure maximum tokens, refill rate, burst capacity, and more.
- **Concurrency Safe**: Uses `RWMutex` for efficient concurrent access.
- **Graceful Handling**: Supports custom handlers for rate limit exceed scenarios.
- **Expiration Management**: Automatically cleans up expired token buckets.

## Installation

Install the middleware via `go get`:

```bash
go get github.com/colommar/gin-ratelimiter
```

Then, import it in your Go code:

```go
import "github.com/colommar/gin-ratelimiter/limiter"
```

## Usage

### Basic Example

Here's a basic example of how to use the middleware in a Gin application:

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/colommar/gin-ratelimiter/limiter"
	"time"
)

func main() {
	config := limiter.RateLimitConfig{
		MaxTokens:            10,
		RefillRate:           1,
		RefillInterval:       time.Second,
		KeyFunc:              func(c *gin.Context) string { return c.ClientIP() },
		BurstMultiplier:      2,
		Timeout:              time.Second * 2,
		LimitExceededHandler: nil, // Optional: can set custom handler
		ExpirationDuration:   time.Minute * 5,
	}

	limiterMiddleware, err := limiter.NewRateLimiter(config)
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.Use(limiterMiddleware)

	r.GET("/", func(c *gin.Context) {
		c.String(200, "Hello, world!")
	})

	r.Run(":8080")
}
```

### Configuration

The `RateLimitConfig` struct allows you to customize the behavior of the rate limiter:

- **MaxTokens**: Maximum number of tokens in the bucket, controlling the maximum concurrency.
- **RefillRate**: Number of tokens added during each refill interval.
- **RefillInterval**: Duration between each refill of tokens.
- **KeyFunc**: Function to generate a unique key for each request (e.g., by IP, user ID).
- **BurstMultiplier**: Multiplier for burst capacity (actual burst capacity = `MaxTokens * BurstMultiplier`).
- **Timeout**: Maximum time to wait for a token if the bucket is empty.
- **LimitExceededHandler**: Optional custom handler to manage rate-limited responses.
- **ExpirationDuration**: Time after which inactive token buckets are cleaned up.

### Custom Limit Exceeded Handler

You can provide a custom handler when the rate limit is exceeded:

```go
config.LimitExceededHandler = func(c *gin.Context) {
    c.JSON(429, gin.H{"error": "Too many requests"})
    c.Abort()
}
```

### Expiration Management

The middleware automatically cleans up expired token buckets. You can set the `ExpirationDuration` in the configuration to control how long a bucket should be retained after its last use.

### Advanced Usage

For more advanced scenarios, you can modify the `RateLimitConfig` or even extend the middleware to suit your needs. Here's an example of setting a custom rate-limiting strategy based on a user's API key:

```go
config.KeyFunc = func(c *gin.Context) string {
    return c.GetHeader("X-API-Key")
}
```

## Testing

To run tests, use the following command:

```bash
go test -v ./...
```

Ensure that your tests cover all edge cases, including timeout scenarios, burst traffic, and token bucket expiration.

## Contributing

If you find a bug or have a feature request, feel free to open an issue or submit a pull request. Contributions are welcome!

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

# 中文说明

# gin-ratelimiter

该仓库包含一个基于令牌桶算法的 Gin 中间件，用于实现限流功能。它允许你控制基于 Gin 的 Web 应用程序的请求速率，支持突发流量并提供可定制的配置。

## 功能

- **令牌桶算法**：在允许突发的同时控制请求速率。
- **可定制**：可配置最大令牌数、填充速率、突发容量等。
- **并发安全**：使用 `RWMutex` 实现高效的并发访问。
- **优雅处理**：支持自定义限流超限处理函数。
- **过期管理**：自动清理过期的令牌桶。

## 安装

通过 `go get` 安装中间件：

```bash
go get github.com/colommar/gin-ratelimiter
```

然后在你的 Go 代码中导入：

```go
import "github.com/colommar/gin-ratelimiter/limiter"
```

## 用法

### 基本示例

以下是如何在 Gin 应用程序中使用该中间件的基本示例：

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/colommar/gin-ratelimiter/limiter"
	"time"
)

func main() {
	config := limiter.RateLimitConfig{
		MaxTokens:            10,
		RefillRate:           1,
		RefillInterval:       time.Second,
		KeyFunc:              func(c *gin.Context) string { return c.ClientIP() },
		BurstMultiplier:      2,
		Timeout:              time.Second * 2,
		LimitExceededHandler: nil, // 可选：可以设置自定义处理函数
		ExpirationDuration:   time.Minute * 5,
	}

	limiterMiddleware, err := limiter.NewRateLimiter(config)
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.Use(limiterMiddleware)

	r.GET("/", func(c *gin.Context) {
		c.String(200, "Hello, world!")
	})

	r.Run(":8080")
}
```

### 配置

`RateLimitConfig` 结构体允许你定制限流器的行为：

- **MaxTokens**：桶中的最大令牌数，控制最大并发量。
- **RefillRate**：每次填充时增加的令牌数量。
- **RefillInterval**：每次填充令牌的时间间隔。
- **KeyFunc**：生成每个请求唯一键值的函数（例如，按 IP 或用户 ID）。
- **BurstMultiplier**：突发容量倍数（实际突发容量 = `MaxTokens * BurstMultiplier`）。
- **Timeout**：当桶为空时等待令牌的最大时间。
- **LimitExceededHandler**：可选的自定义处理限流响应的函数。
- **ExpirationDuration**：不活跃的令牌桶被清理的时间。

### 自定义限流超限处理函数

你可以在超限时提供一个自定义处理函数：

```go
config.LimitExceededHandler = func(c *gin.Context) {
    c.JSON(429, gin.H{"error": "请求过多"})
    c.Abort()
}
```

### 过期管理

中间件会自动清理过期的令牌桶。你可以在配置中设置 `ExpirationDuration` 来控制令牌桶最后使用后的保留时间。

### 高级用法

对于更复杂的场景，你可以修改 `RateLimitConfig` 或扩展中间件以满足你的需求。以下是基于用户 API 密钥设置自定义限流策略的示例：

```go
config.KeyFunc = func(c *gin.Context) string {
    return c.GetHeader("X-API-Key")
}
```

## 测试

使用以下命令运行测试：

```bash
go test -v ./...
```

确保你的测试覆盖所有边界情况，包括超时场景、突发流量和令牌桶的过期。

## 贡献

如果你发现了 bug 或有功能需求，欢迎提出 issue 或提交 pull request。贡献是受欢迎的！

## 许可证

此项目使用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。