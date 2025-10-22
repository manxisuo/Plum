package weaknetwork

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// RetryStrategy 重试策略接口
type RetryStrategy interface {
	ShouldRetry(attempt int, err error) bool
	GetDelay(attempt int) time.Duration
	GetMaxAttempts() int
}

// ExponentialBackoffStrategy 指数退避策略
type ExponentialBackoffStrategy struct {
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	MaxAttempts int
	Multiplier  float64
	Jitter      bool
}

// NewExponentialBackoffStrategy 创建指数退避策略
func NewExponentialBackoffStrategy(baseDelay, maxDelay time.Duration, maxAttempts int) *ExponentialBackoffStrategy {
	return &ExponentialBackoffStrategy{
		BaseDelay:   baseDelay,
		MaxDelay:    maxDelay,
		MaxAttempts: maxAttempts,
		Multiplier:  2.0,
		Jitter:      true,
	}
}

func (s *ExponentialBackoffStrategy) ShouldRetry(attempt int, err error) bool {
	return attempt < s.MaxAttempts && err != nil
}

func (s *ExponentialBackoffStrategy) GetDelay(attempt int) time.Duration {
	delay := float64(s.BaseDelay) * math.Pow(s.Multiplier, float64(attempt))

	if delay > float64(s.MaxDelay) {
		delay = float64(s.MaxDelay)
	}

	// 添加抖动
	if s.Jitter {
		jitter := rand.Float64() * 0.1 * delay
		delay += jitter
	}

	return time.Duration(delay)
}

func (s *ExponentialBackoffStrategy) GetMaxAttempts() int {
	return s.MaxAttempts
}

// LinearBackoffStrategy 线性退避策略
type LinearBackoffStrategy struct {
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	MaxAttempts int
	Increment   time.Duration
}

// NewLinearBackoffStrategy 创建线性退避策略
func NewLinearBackoffStrategy(baseDelay, maxDelay time.Duration, maxAttempts int) *LinearBackoffStrategy {
	return &LinearBackoffStrategy{
		BaseDelay:   baseDelay,
		MaxDelay:    maxDelay,
		MaxAttempts: maxAttempts,
		Increment:   baseDelay,
	}
}

func (s *LinearBackoffStrategy) ShouldRetry(attempt int, err error) bool {
	return attempt < s.MaxAttempts && err != nil
}

func (s *LinearBackoffStrategy) GetDelay(attempt int) time.Duration {
	delay := s.BaseDelay + time.Duration(attempt)*s.Increment
	if delay > s.MaxDelay {
		delay = s.MaxDelay
	}
	return delay
}

func (s *LinearBackoffStrategy) GetMaxAttempts() int {
	return s.MaxAttempts
}

// FixedDelayStrategy 固定延迟策略
type FixedDelayStrategy struct {
	Delay       time.Duration
	MaxAttempts int
}

// NewFixedDelayStrategy 创建固定延迟策略
func NewFixedDelayStrategy(delay time.Duration, maxAttempts int) *FixedDelayStrategy {
	return &FixedDelayStrategy{
		Delay:       delay,
		MaxAttempts: maxAttempts,
	}
}

func (s *FixedDelayStrategy) ShouldRetry(attempt int, err error) bool {
	return attempt < s.MaxAttempts && err != nil
}

func (s *FixedDelayStrategy) GetDelay(attempt int) time.Duration {
	return s.Delay
}

func (s *FixedDelayStrategy) GetMaxAttempts() int {
	return s.MaxAttempts
}

// RetryableFunc 可重试的函数类型
type RetryableFunc func() (interface{}, error)

// RetryableFuncWithContext 带上下文的可重试函数类型
type RetryableFuncWithContext func(ctx context.Context) (interface{}, error)

// RetryExecutor 重试执行器
type RetryExecutor struct {
	strategy RetryStrategy
}

// NewRetryExecutor 创建重试执行器
func NewRetryExecutor(strategy RetryStrategy) *RetryExecutor {
	return &RetryExecutor{
		strategy: strategy,
	}
}

// Execute 执行重试
func (r *RetryExecutor) Execute(fn RetryableFunc) (interface{}, error) {
	var lastErr error

	for attempt := 0; attempt <= r.strategy.GetMaxAttempts(); attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !r.strategy.ShouldRetry(attempt, err) {
			break
		}

		if attempt < r.strategy.GetMaxAttempts() {
			delay := r.strategy.GetDelay(attempt)
			time.Sleep(delay)
		}
	}

	return nil, lastErr
}

// ExecuteWithContext 带上下文执行重试
func (r *RetryExecutor) ExecuteWithContext(ctx context.Context, fn RetryableFuncWithContext) (interface{}, error) {
	var lastErr error

	for attempt := 0; attempt <= r.strategy.GetMaxAttempts(); attempt++ {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !r.strategy.ShouldRetry(attempt, err) {
			break
		}

		if attempt < r.strategy.GetMaxAttempts() {
			delay := r.strategy.GetDelay(attempt)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// 继续重试
			}
		}
	}

	return nil, lastErr
}

// HTTPRetryMiddleware HTTP重试中间件
type HTTPRetryMiddleware struct {
	executor *RetryExecutor
}

// NewHTTPRetryMiddleware 创建HTTP重试中间件
func NewHTTPRetryMiddleware(executor *RetryExecutor) *HTTPRetryMiddleware {
	return &HTTPRetryMiddleware{
		executor: executor,
	}
}

// Handler HTTP重试处理函数
func (m *HTTPRetryMiddleware) Handler(next func() error) func() {
	return func() {
		_, err := m.executor.Execute(func() (interface{}, error) {
			return nil, next()
		})

		if err != nil {
			// 重试失败，记录错误
			return
		}
	}
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Strategy    string // "exponential", "linear", "fixed"
}

// NewRetryConfig 创建重试配置
func NewRetryConfig(maxAttempts int, baseDelay, maxDelay time.Duration, strategy string) *RetryConfig {
	return &RetryConfig{
		MaxAttempts: maxAttempts,
		BaseDelay:   baseDelay,
		MaxDelay:    maxDelay,
		Strategy:    strategy,
	}
}

// CreateStrategy 根据配置创建重试策略
func (c *RetryConfig) CreateStrategy() RetryStrategy {
	switch c.Strategy {
	case "exponential":
		return NewExponentialBackoffStrategy(c.BaseDelay, c.MaxDelay, c.MaxAttempts)
	case "linear":
		return NewLinearBackoffStrategy(c.BaseDelay, c.MaxDelay, c.MaxAttempts)
	case "fixed":
		return NewFixedDelayStrategy(c.BaseDelay, c.MaxAttempts)
	default:
		return NewExponentialBackoffStrategy(c.BaseDelay, c.MaxDelay, c.MaxAttempts)
	}
}
