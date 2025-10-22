package weaknetwork

import (
	"log"
	"net"
	"sync"
	"time"
)

// NetworkCondition 网络条件
type NetworkCondition int

const (
	NetworkGood NetworkCondition = iota
	NetworkFair
	NetworkPoor
	NetworkVeryPoor
)

func (c NetworkCondition) String() string {
	switch c {
	case NetworkGood:
		return "good"
	case NetworkFair:
		return "fair"
	case NetworkPoor:
		return "poor"
	case NetworkVeryPoor:
		return "very_poor"
	default:
		return "unknown"
	}
}

// AdaptiveManager 自适应管理器
type AdaptiveManager struct {
	config           *WeakNetworkConfig
	networkCondition NetworkCondition
	lastCheck        time.Time
	checkInterval    time.Duration
	mu               sync.RWMutex

	// 网络质量指标
	avgLatency  time.Duration
	errorRate   float64
	successRate float64
	sampleCount int

	// 自适应配置
	adaptiveEnabled bool
	thresholds      AdaptiveThresholds
}

// AdaptiveThresholds 自适应阈值
type AdaptiveThresholds struct {
	GoodLatencyMax   time.Duration
	FairLatencyMax   time.Duration
	PoorLatencyMax   time.Duration
	GoodErrorRateMax float64
	FairErrorRateMax float64
	PoorErrorRateMax float64
}

// NewAdaptiveManager 创建自适应管理器
func NewAdaptiveManager(config *WeakNetworkConfig) *AdaptiveManager {
	return &AdaptiveManager{
		config:           config,
		networkCondition: NetworkGood,
		checkInterval:    30 * time.Second,
		adaptiveEnabled:  true,
		thresholds: AdaptiveThresholds{
			GoodLatencyMax:   50 * time.Millisecond,
			FairLatencyMax:   200 * time.Millisecond,
			PoorLatencyMax:   1000 * time.Millisecond,
			GoodErrorRateMax: 0.01, // 1%
			FairErrorRateMax: 0.05, // 5%
			PoorErrorRateMax: 0.20, // 20%
		},
	}
}

// Start 启动自适应管理
func (a *AdaptiveManager) Start() {
	if !a.adaptiveEnabled {
		return
	}

	go a.monitorLoop()
}

// Stop 停止自适应管理
func (a *AdaptiveManager) Stop() {
	a.adaptiveEnabled = false
}

// ShouldEnableWeakNetworkSupport 判断是否应该启用弱网支持
func (a *AdaptiveManager) ShouldEnableWeakNetworkSupport() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 如果网络条件为良好，可以禁用弱网支持
	if a.networkCondition == NetworkGood {
		return false
	}

	// 如果网络条件为一般、差或很差，启用弱网支持
	return true
}

// GetRecommendedConfig 获取推荐配置
func (a *AdaptiveManager) GetRecommendedConfig() *WeakNetworkConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()

	config := *a.config // 复制基础配置

	switch a.networkCondition {
	case NetworkGood:
		// 网络良好，使用轻量级配置
		config.RateLimitRPS = 2000
		config.RateLimitBurst = 4000
		config.CircuitBreakerTimeout = 30 * time.Second
		config.RetryMaxAttempts = 1
		config.CacheTTL = 10 * time.Second
		config.HealthCheckInterval = 60 * time.Second

	case NetworkFair:
		// 网络一般，使用中等配置
		config.RateLimitRPS = 1000
		config.RateLimitBurst = 2000
		config.CircuitBreakerTimeout = 60 * time.Second
		config.RetryMaxAttempts = 3
		config.CacheTTL = 30 * time.Second
		config.HealthCheckInterval = 30 * time.Second

	case NetworkPoor:
		// 网络差，使用强化配置
		config.RateLimitRPS = 500
		config.RateLimitBurst = 1000
		config.CircuitBreakerTimeout = 120 * time.Second
		config.RetryMaxAttempts = 5
		config.CacheTTL = 60 * time.Second
		config.HealthCheckInterval = 15 * time.Second

	case NetworkVeryPoor:
		// 网络很差，使用最强配置
		config.RateLimitRPS = 200
		config.RateLimitBurst = 500
		config.CircuitBreakerTimeout = 300 * time.Second
		config.RetryMaxAttempts = 10
		config.CacheTTL = 120 * time.Second
		config.HealthCheckInterval = 10 * time.Second
	}

	return &config
}

// monitorLoop 监控循环
func (a *AdaptiveManager) monitorLoop() {
	ticker := time.NewTicker(a.checkInterval)
	defer ticker.Stop()

	for a.adaptiveEnabled {
		select {
		case <-ticker.C:
			a.performNetworkCheck()
		}
	}
}

// performNetworkCheck 执行网络检查
func (a *AdaptiveManager) performNetworkCheck() {
	// 这里可以添加实际的网络质量检测逻辑
	// 比如ping测试、HTTP请求测试等

	// 模拟网络质量检测
	latency, errorRate := a.simulateNetworkCheck()

	a.mu.Lock()
	defer a.mu.Unlock()

	// 更新指标
	a.avgLatency = latency
	a.errorRate = errorRate
	a.successRate = 1.0 - errorRate
	a.sampleCount++
	a.lastCheck = time.Now()

	// 更新网络条件
	oldCondition := a.networkCondition
	a.networkCondition = a.determineNetworkCondition(latency, errorRate)

	if oldCondition != a.networkCondition {
		log.Printf("Network condition changed from %s to %s",
			oldCondition.String(), a.networkCondition.String())
	}
}

// simulateNetworkCheck 模拟网络检查
func (a *AdaptiveManager) simulateNetworkCheck() (time.Duration, float64) {
	// 这里应该实现真实的网络检查
	// 为了演示，我们返回模拟值

	// 模拟延迟检测
	start := time.Now()
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 5*time.Second)
	if err != nil {
		return 5 * time.Second, 1.0 // 连接失败，100%错误率
	}
	conn.Close()
	latency := time.Since(start)

	// 模拟错误率检测
	errorRate := 0.0
	if latency > 1*time.Second {
		errorRate = 0.1 // 10%错误率
	}

	return latency, errorRate
}

// determineNetworkCondition 确定网络条件
func (a *AdaptiveManager) determineNetworkCondition(latency time.Duration, errorRate float64) NetworkCondition {
	if latency <= a.thresholds.GoodLatencyMax && errorRate <= a.thresholds.GoodErrorRateMax {
		return NetworkGood
	} else if latency <= a.thresholds.FairLatencyMax && errorRate <= a.thresholds.FairErrorRateMax {
		return NetworkFair
	} else if latency <= a.thresholds.PoorLatencyMax && errorRate <= a.thresholds.PoorErrorRateMax {
		return NetworkPoor
	} else {
		return NetworkVeryPoor
	}
}

// GetNetworkCondition 获取当前网络条件
func (a *AdaptiveManager) GetNetworkCondition() NetworkCondition {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.networkCondition
}

// GetNetworkMetrics 获取网络指标
func (a *AdaptiveManager) GetNetworkMetrics() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return map[string]interface{}{
		"condition":    a.networkCondition.String(),
		"avg_latency":  a.avgLatency.String(),
		"error_rate":   a.errorRate,
		"success_rate": a.successRate,
		"sample_count": a.sampleCount,
		"last_check":   a.lastCheck,
	}
}

// SetAdaptiveEnabled 设置自适应启用状态
func (a *AdaptiveManager) SetAdaptiveEnabled(enabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.adaptiveEnabled = enabled
}

// IsAdaptiveEnabled 检查是否启用自适应
func (a *AdaptiveManager) IsAdaptiveEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.adaptiveEnabled
}
