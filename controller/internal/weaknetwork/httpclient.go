package weaknetwork

import (
	"context"
	"net"
	"net/http"
	"time"
)

// HTTPClientConfig HTTP客户端配置
type HTTPClientConfig struct {
	Timeout             time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
	DisableKeepAlives   bool
}

// EnhancedHTTPClient 增强的HTTP客户端
type EnhancedHTTPClient struct {
	client  *http.Client
	config  *HTTPClientConfig
	limiter RateLimiter
	breaker *CircuitBreaker
	retry   *RetryExecutor
}

// NewEnhancedHTTPClient 创建增强的HTTP客户端
func NewEnhancedHTTPClient(config *HTTPClientConfig, limiter RateLimiter, breaker *CircuitBreaker, retry *RetryExecutor) *EnhancedHTTPClient {
	// 创建传输配置
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     config.DisableKeepAlives,
	}

	// 创建HTTP客户端
	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	return &EnhancedHTTPClient{
		client:  client,
		config:  config,
		limiter: limiter,
		breaker: breaker,
		retry:   retry,
	}
}

// Do 执行HTTP请求
func (c *EnhancedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.DoWithContext(req.Context(), req)
}

// DoWithContext 带上下文执行HTTP请求
func (c *EnhancedHTTPClient) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	// 限流检查
	if c.limiter != nil && !c.limiter.Allow() {
		return nil, &HTTPError{Code: 429, Message: "rate limit exceeded"}
	}

	// 熔断器检查
	if c.breaker != nil {
		state := c.breaker.State()
		if state == StateOpen {
			return nil, &HTTPError{Code: 503, Message: "circuit breaker is open"}
		}
	}

	// 重试执行
	if c.retry != nil {
		result, err := c.retry.ExecuteWithContext(ctx, func(ctx context.Context) (interface{}, error) {
			return c.performRequest(ctx, req)
		})

		if err != nil {
			return nil, err
		}

		return result.(*http.Response), nil
	}

	// 直接执行
	result, err := c.performRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return result.(*http.Response), nil
}

// performRequest 执行实际请求
func (c *EnhancedHTTPClient) performRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	// 设置超时
	if c.config.ReadTimeout > 0 || c.config.WriteTimeout > 0 {
		// 创建带超时的上下文
		timeout := c.config.ReadTimeout + c.config.WriteTimeout
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
	}

	// 执行请求
	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		// 网络错误，可以重试
		return nil, &HTTPError{Code: 0, Message: err.Error(), Retryable: true}
	}

	// 检查HTTP状态码
	if resp.StatusCode >= 500 {
		// 服务器错误，可以重试
		return resp, &HTTPError{Code: resp.StatusCode, Message: "server error", Retryable: true}
	}

	if resp.StatusCode == 429 {
		// 限流错误，可以重试
		return resp, &HTTPError{Code: resp.StatusCode, Message: "rate limit exceeded", Retryable: true}
	}

	if resp.StatusCode == 408 {
		// 超时错误，可以重试
		return resp, &HTTPError{Code: resp.StatusCode, Message: "request timeout", Retryable: true}
	}

	return resp, nil
}

// Get 执行GET请求
func (c *EnhancedHTTPClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post 执行POST请求
func (c *EnhancedHTTPClient) Post(url, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

// PostWithContext 带上下文执行POST请求
func (c *EnhancedHTTPClient) PostWithContext(ctx context.Context, url, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.DoWithContext(ctx, req)
}

// SetLimiter 设置限流器
func (c *EnhancedHTTPClient) SetLimiter(limiter RateLimiter) {
	c.limiter = limiter
}

// SetBreaker 设置熔断器
func (c *EnhancedHTTPClient) SetBreaker(breaker *CircuitBreaker) {
	c.breaker = breaker
}

// SetRetry 设置重试执行器
func (c *EnhancedHTTPClient) SetRetry(retry *RetryExecutor) {
	c.retry = retry
}

// GetConfig 获取配置
func (c *EnhancedHTTPClient) GetConfig() *HTTPClientConfig {
	return c.config
}

// HTTPError HTTP错误
type HTTPError struct {
	Code      int
	Message   string
	Retryable bool
}

func (e *HTTPError) Error() string {
	return e.Message
}

// IsRetryable 检查是否可重试
func (e *HTTPError) IsRetryable() bool {
	return e.Retryable
}

// HTTPClientFactory HTTP客户端工厂
type HTTPClientFactory struct {
	config *WeakNetworkConfig
}

// NewHTTPClientFactory 创建HTTP客户端工厂
func NewHTTPClientFactory(config *WeakNetworkConfig) *HTTPClientFactory {
	return &HTTPClientFactory{
		config: config,
	}
}

// CreateClient 创建HTTP客户端
func (f *HTTPClientFactory) CreateClient() *EnhancedHTTPClient {
	// 创建HTTP客户端配置
	httpConfig := &HTTPClientConfig{
		Timeout:             f.config.RequestTimeout,
		ReadTimeout:         f.config.ReadTimeout,
		WriteTimeout:        f.config.WriteTimeout,
		IdleTimeout:         f.config.IdleTimeout,
		MaxIdleConns:        f.config.MaxIdleConns,
		MaxIdleConnsPerHost: f.config.MaxIdleConnsPerHost,
		MaxConnsPerHost:     f.config.MaxConnsPerHost,
		DisableKeepAlives:   f.config.DisableKeepAlives,
	}

	// 创建限流器
	var limiter RateLimiter
	if f.config.RateLimitEnabled {
		limiter = NewTokenBucketLimiter(f.config.RateLimitRPS, f.config.RateLimitBurst)
	}

	// 创建熔断器
	var breaker *CircuitBreaker
	if f.config.CircuitBreakerEnabled {
		breaker = NewCircuitBreaker(Config{
			Name:        "http-client",
			MaxRequests: uint32(f.config.CircuitBreakerMaxRequests),
			Interval:    f.config.CircuitBreakerInterval,
			Timeout:     f.config.CircuitBreakerTimeout,
		})
	}

	// 创建重试执行器
	var retry *RetryExecutor
	if f.config.RetryEnabled {
		retryConfig := NewRetryConfig(
			f.config.RetryMaxAttempts,
			f.config.RetryBaseDelay,
			f.config.RetryMaxDelay,
			"exponential",
		)
		retry = NewRetryExecutor(retryConfig.CreateStrategy())
	}

	return NewEnhancedHTTPClient(httpConfig, limiter, breaker, retry)
}
