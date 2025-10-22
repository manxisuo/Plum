package plum

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"time"
)

// RetryStrategy 重试策略接口
type RetryStrategy interface {
	ShouldRetry(attempt int, err error, resp *http.Response) bool
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
func NewExponentialBackoffStrategy(baseDelay time.Duration, maxDelay time.Duration, maxAttempts int) *ExponentialBackoffStrategy {
	return &ExponentialBackoffStrategy{
		BaseDelay:   baseDelay,
		MaxDelay:    maxDelay,
		MaxAttempts: maxAttempts,
		Multiplier:  2.0,
		Jitter:      true,
	}
}

func (s *ExponentialBackoffStrategy) ShouldRetry(attempt int, err error, resp *http.Response) bool {
	if attempt >= s.MaxAttempts {
		return false
	}

	// 网络错误总是重试
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
			return true
		}
		// 超时错误
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return true
		}
		// 连接错误
		if _, ok := err.(*net.OpError); ok {
			return true
		}
		return false
	}

	// HTTP状态码重试策略
	if resp != nil {
		// 5xx服务器错误
		if resp.StatusCode >= 500 {
			return true
		}
		// 429 Too Many Requests
		if resp.StatusCode == 429 {
			return true
		}
		// 408 Request Timeout
		if resp.StatusCode == 408 {
			return true
		}
	}

	return false
}

func (s *ExponentialBackoffStrategy) GetDelay(attempt int) time.Duration {
	delay := float64(s.BaseDelay) * math.Pow(s.Multiplier, float64(attempt))

	if delay > float64(s.MaxDelay) {
		delay = float64(s.MaxDelay)
	}

	// 添加抖动避免惊群效应
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
func NewLinearBackoffStrategy(baseDelay time.Duration, maxDelay time.Duration, maxAttempts int) *LinearBackoffStrategy {
	return &LinearBackoffStrategy{
		BaseDelay:   baseDelay,
		MaxDelay:    maxDelay,
		MaxAttempts: maxAttempts,
		Increment:   baseDelay,
	}
}

func (s *LinearBackoffStrategy) ShouldRetry(attempt int, err error, resp *http.Response) bool {
	if attempt >= s.MaxAttempts {
		return false
	}

	// 网络错误重试
	if err != nil {
		if netErr, ok := err.(net.Error); ok && (netErr.Temporary() || netErr.Timeout()) {
			return true
		}
		if _, ok := err.(*net.OpError); ok {
			return true
		}
		return false
	}

	// HTTP状态码重试
	if resp != nil {
		return resp.StatusCode >= 500 || resp.StatusCode == 429 || resp.StatusCode == 408
	}

	return false
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

func (s *FixedDelayStrategy) ShouldRetry(attempt int, err error, resp *http.Response) bool {
	if attempt >= s.MaxAttempts {
		return false
	}

	if err != nil {
		if netErr, ok := err.(net.Error); ok && (netErr.Temporary() || netErr.Timeout()) {
			return true
		}
		if _, ok := err.(*net.OpError); ok {
			return true
		}
		return false
	}

	if resp != nil {
		return resp.StatusCode >= 500 || resp.StatusCode == 429 || resp.StatusCode == 408
	}

	return false
}

func (s *FixedDelayStrategy) GetDelay(attempt int) time.Duration {
	return s.Delay
}

func (s *FixedDelayStrategy) GetMaxAttempts() int {
	return s.MaxAttempts
}

// RetryableHTTPClient 支持重试的HTTP客户端
type RetryableHTTPClient struct {
	client   *http.Client
	strategy RetryStrategy
}

// NewRetryableHTTPClient 创建支持重试的HTTP客户端
func NewRetryableHTTPClient(client *http.Client, strategy RetryStrategy) *RetryableHTTPClient {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &RetryableHTTPClient{
		client:   client,
		strategy: strategy,
	}
}

// Do 执行HTTP请求（带重试）
func (c *RetryableHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.DoWithContext(req.Context(), req)
}

// DoWithContext 执行HTTP请求（带重试和上下文）
func (c *RetryableHTTPClient) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= c.strategy.GetMaxAttempts(); attempt++ {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 执行请求
		resp, err := c.client.Do(req)
		lastResp = resp
		lastErr = err

		// 检查是否需要重试
		if !c.strategy.ShouldRetry(attempt, err, resp) {
			if err != nil {
				return nil, err
			}
			return resp, nil
		}

		// 如果是最后一次尝试，直接返回
		if attempt == c.strategy.GetMaxAttempts() {
			break
		}

		// 计算延迟时间
		delay := c.strategy.GetDelay(attempt)

		// 等待延迟时间
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	// 返回最后一次的错误或响应
	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", c.strategy.GetMaxAttempts()+1, lastErr)
	}

	return lastResp, nil
}

// SetStrategy 设置重试策略
func (c *RetryableHTTPClient) SetStrategy(strategy RetryStrategy) {
	c.strategy = strategy
}

// GetStrategy 获取重试策略
func (c *RetryableHTTPClient) GetStrategy() RetryStrategy {
	return c.strategy
}
