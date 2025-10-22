package weaknetwork

import (
	"context"
	"sync"
	"time"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow() bool
	AllowN(n int) bool
	Wait(ctx context.Context) error
	WaitN(ctx context.Context, n int) error
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	capacity   int        // 桶容量
	tokens     int        // 当前令牌数
	refillRate int        // 每秒补充令牌数
	lastRefill time.Time  // 上次补充时间
	mutex      sync.Mutex // 互斥锁
}

// NewTokenBucketLimiter 创建令牌桶限流器
func NewTokenBucketLimiter(rps, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		capacity:   burst,
		tokens:     burst,
		refillRate: rps,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许请求
func (r *TokenBucketLimiter) Allow() bool {
	return r.AllowN(1)
}

// AllowN 检查是否允许n个请求
func (r *TokenBucketLimiter) AllowN(n int) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill)

	// 补充令牌
	tokensToAdd := int(elapsed.Seconds()) * r.refillRate
	if tokensToAdd > 0 {
		r.tokens += tokensToAdd
		if r.tokens > r.capacity {
			r.tokens = r.capacity
		}
		r.lastRefill = now
	}

	// 检查是否有足够的令牌
	if r.tokens >= n {
		r.tokens -= n
		return true
	}

	return false
}

// Wait 等待直到允许请求
func (r *TokenBucketLimiter) Wait(ctx context.Context) error {
	return r.WaitN(ctx, 1)
}

// WaitN 等待直到允许n个请求
func (r *TokenBucketLimiter) WaitN(ctx context.Context, n int) error {
	for {
		if r.AllowN(n) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 10):
			// 短暂等待后重试
		}
	}
}

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	windowSize  time.Duration // 窗口大小
	requests    []time.Time   // 请求时间记录
	mutex       sync.Mutex    // 互斥锁
	maxRequests int           // 最大请求数
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(windowSize time.Duration, maxRequests int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windowSize:  windowSize,
		requests:    make([]time.Time, 0),
		maxRequests: maxRequests,
	}
}

// Allow 检查是否允许请求
func (s *SlidingWindowLimiter) Allow() bool {
	return s.AllowN(1)
}

// AllowN 检查是否允许n个请求
func (s *SlidingWindowLimiter) AllowN(n int) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-s.windowSize)

	// 清理过期请求
	var validRequests []time.Time
	for _, reqTime := range s.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	s.requests = validRequests

	// 检查是否超过限制
	if len(s.requests)+n > s.maxRequests {
		return false
	}

	// 记录新请求
	for i := 0; i < n; i++ {
		s.requests = append(s.requests, now)
	}

	return true
}

// Wait 等待直到允许请求
func (s *SlidingWindowLimiter) Wait(ctx context.Context) error {
	return s.WaitN(ctx, 1)
}

// WaitN 等待直到允许n个请求
func (s *SlidingWindowLimiter) WaitN(ctx context.Context, n int) error {
	for {
		if s.AllowN(n) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 10):
			// 短暂等待后重试
		}
	}
}

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	limiter RateLimiter
}

// NewRateLimitMiddleware 创建限流中间件
func NewRateLimitMiddleware(limiter RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
	}
}

// Handler 限流处理函数
func (m *RateLimitMiddleware) Handler(next func()) func() {
	return func() {
		if m.limiter.Allow() {
			next()
		}
		// 如果限流，直接丢弃请求
	}
}

// HTTPRateLimitMiddleware HTTP限流中间件
type HTTPRateLimitMiddleware struct {
	limiter RateLimiter
}

// NewHTTPRateLimitMiddleware 创建HTTP限流中间件
func NewHTTPRateLimitMiddleware(limiter RateLimiter) *HTTPRateLimitMiddleware {
	return &HTTPRateLimitMiddleware{
		limiter: limiter,
	}
}

// Handler HTTP限流处理函数
func (m *HTTPRateLimitMiddleware) Handler(next func()) func() {
	return func() {
		if !m.limiter.Allow() {
			// 返回429 Too Many Requests
			return
		}
		next()
	}
}
