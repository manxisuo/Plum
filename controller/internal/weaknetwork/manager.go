package weaknetwork

import (
	"context"
	"log"
	"net/http"
	"sync"
)

// WeakNetworkManager 弱网环境管理器
type WeakNetworkManager struct {
	config      *WeakNetworkConfig
	httpClient  *EnhancedHTTPClient
	limiter     RateLimiter
	breaker     *CircuitBreaker
	retry       *RetryExecutor
	healthCheck *HealthChecker
	metrics     *MetricsCollector
	adaptive    *AdaptiveManager

	mu       sync.RWMutex
	enabled  bool
	started  bool
	stopChan chan struct{}
}

// NewWeakNetworkManager 创建弱网环境管理器
func NewWeakNetworkManager(config *WeakNetworkConfig) *WeakNetworkManager {
	manager := &WeakNetworkManager{
		config:   config,
		enabled:  config.WeakNetworkEnabled, // 使用配置决定是否启用
		stopChan: make(chan struct{}),
	}

	// 创建自适应管理器
	if config.AdaptiveEnabled {
		manager.adaptive = NewAdaptiveManager(config)
	}

	// 初始化组件
	manager.initializeComponents()

	return manager
}

// initializeComponents 初始化组件
func (m *WeakNetworkManager) initializeComponents() {
	// 创建限流器
	if m.config.RateLimitEnabled {
		m.limiter = NewTokenBucketLimiter(m.config.RateLimitRPS, m.config.RateLimitBurst)
	}

	// 创建熔断器
	if m.config.CircuitBreakerEnabled {
		m.breaker = NewCircuitBreaker(Config{
			Name:        "controller",
			MaxRequests: uint32(m.config.CircuitBreakerMaxRequests),
			Interval:    m.config.CircuitBreakerInterval,
			Timeout:     m.config.CircuitBreakerTimeout,
		})
	}

	// 创建重试执行器
	if m.config.RetryEnabled {
		retryConfig := NewRetryConfig(
			m.config.RetryMaxAttempts,
			m.config.RetryBaseDelay,
			m.config.RetryMaxDelay,
			"exponential",
		)
		m.retry = NewRetryExecutor(retryConfig.CreateStrategy())
	}

	// 创建HTTP客户端
	factory := NewHTTPClientFactory(m.config)
	m.httpClient = factory.CreateClient()

	// 创建健康检查器
	if m.config.HealthCheckEnabled {
		m.healthCheck = NewHealthChecker(m.config.HealthCheckInterval, m.config.HealthCheckTimeout)
	}

	// 创建指标收集器
	if m.config.MetricsEnabled {
		m.metrics = NewMetricsCollector(m.config.MetricsInterval)
	}
}

// Start 启动管理器
func (m *WeakNetworkManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil
	}

	// 启动自适应管理
	if m.adaptive != nil {
		m.adaptive.Start()
	}

	// 启动健康检查
	if m.healthCheck != nil {
		if err := m.healthCheck.Start(); err != nil {
			log.Printf("Failed to start health checker: %v", err)
		}
	}

	// 启动指标收集
	if m.metrics != nil {
		if err := m.metrics.Start(); err != nil {
			log.Printf("Failed to start metrics collector: %v", err)
		}
	}

	m.started = true
	log.Println("Weak network manager started")

	return nil
}

// Stop 停止管理器
func (m *WeakNetworkManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return
	}

	// 停止自适应管理
	if m.adaptive != nil {
		m.adaptive.Stop()
	}

	// 停止健康检查
	if m.healthCheck != nil {
		m.healthCheck.Stop()
	}

	// 停止指标收集
	if m.metrics != nil {
		m.metrics.Stop()
	}

	close(m.stopChan)
	m.started = false
	log.Println("Weak network manager stopped")
}

// IsEnabled 检查是否启用
func (m *WeakNetworkManager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 如果启用了自适应管理，根据网络条件决定
	if m.adaptive != nil && m.adaptive.IsAdaptiveEnabled() {
		return m.adaptive.ShouldEnableWeakNetworkSupport()
	}

	// 否则使用手动设置
	return m.enabled
}

// Enable 启用弱网环境支持
func (m *WeakNetworkManager) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = true
	log.Println("Weak network support enabled")
}

// Disable 禁用弱网环境支持
func (m *WeakNetworkManager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
	log.Println("Weak network support disabled")
}

// GetHTTPClient 获取HTTP客户端
func (m *WeakNetworkManager) GetHTTPClient() *EnhancedHTTPClient {
	return m.httpClient
}

// GetLimiter 获取限流器
func (m *WeakNetworkManager) GetLimiter() RateLimiter {
	return m.limiter
}

// GetBreaker 获取熔断器
func (m *WeakNetworkManager) GetBreaker() *CircuitBreaker {
	return m.breaker
}

// GetRetry 获取重试执行器
func (m *WeakNetworkManager) GetRetry() *RetryExecutor {
	return m.retry
}

// GetHealthChecker 获取健康检查器
func (m *WeakNetworkManager) GetHealthChecker() *HealthChecker {
	return m.healthCheck
}

// GetMetrics 获取指标收集器
func (m *WeakNetworkManager) GetMetrics() *MetricsCollector {
	return m.metrics
}

// GetConfig 获取配置
func (m *WeakNetworkManager) GetConfig() *WeakNetworkConfig {
	return m.config
}

// UpdateConfig 更新配置
func (m *WeakNetworkManager) UpdateConfig(config *WeakNetworkConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = config
	m.initializeComponents()

	log.Println("Weak network configuration updated")
}

// WrapHandler 包装HTTP处理器
func (m *WeakNetworkManager) WrapHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 检查是否启用
		if !m.IsEnabled() {
			handler(w, r)
			return
		}

		// 限流检查
		if m.limiter != nil && !m.limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// 熔断器检查
		if m.breaker != nil {
			state := m.breaker.State()
			if state == StateOpen {
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
				return
			}
		}

		// 执行处理器
		handler(w, r)
	}
}

// WrapMiddleware 包装中间件
func (m *WeakNetworkManager) WrapMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查是否启用
		if !m.IsEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		// 限流检查
		if m.limiter != nil && !m.limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// 熔断器检查
		if m.breaker != nil {
			state := m.breaker.State()
			if state == StateOpen {
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
				return
			}
		}

		// 执行下一个处理器
		next.ServeHTTP(w, r)
	})
}

// ExecuteWithRetry 带重试执行
func (m *WeakNetworkManager) ExecuteWithRetry(ctx context.Context, fn func() error) error {
	if !m.IsEnabled() || m.retry == nil {
		return fn()
	}

	_, err := m.retry.ExecuteWithContext(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, fn()
	})

	return err
}

// GetStatus 获取状态
func (m *WeakNetworkManager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := map[string]interface{}{
		"enabled": m.enabled,
		"started": m.started,
	}

	if m.limiter != nil {
		status["rate_limit_enabled"] = true
	}

	if m.breaker != nil {
		status["circuit_breaker_enabled"] = true
		status["circuit_breaker_state"] = m.breaker.State().String()
	}

	if m.retry != nil {
		status["retry_enabled"] = true
	}

	if m.healthCheck != nil {
		status["health_check_enabled"] = true
		status["health_check_status"] = m.healthCheck.GetStatus()
	}

	if m.metrics != nil {
		status["metrics_enabled"] = true
		status["metrics"] = m.metrics.GetMetrics()
	}

	return status
}
