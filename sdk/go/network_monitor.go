package plum

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// NetworkQuality 网络质量
type NetworkQuality int

const (
	NetworkQualityExcellent NetworkQuality = iota // 优秀
	NetworkQualityGood                            // 良好
	NetworkQualityFair                            // 一般
	NetworkQualityPoor                            // 差
	NetworkQualityVeryPoor                        // 很差
)

func (q NetworkQuality) String() string {
	switch q {
	case NetworkQualityExcellent:
		return "excellent"
	case NetworkQualityGood:
		return "good"
	case NetworkQualityFair:
		return "fair"
	case NetworkQualityPoor:
		return "poor"
	case NetworkQualityVeryPoor:
		return "very_poor"
	default:
		return "unknown"
	}
}

// NetworkStats 网络统计
type NetworkStats struct {
	Latency     time.Duration
	SuccessRate float64
	ErrorRate   float64
	TimeoutRate float64
	LastUpdated time.Time
	SampleCount int
}

// NetworkMonitor 网络监控器
type NetworkMonitor struct {
	controllerURL string
	stats         *NetworkStats
	mutex         sync.RWMutex
	httpClient    *http.Client
	monitoring    bool
	stopChan      chan struct{}
}

// NewNetworkMonitor 创建网络监控器
func NewNetworkMonitor(controllerURL string) *NetworkMonitor {
	return &NetworkMonitor{
		controllerURL: controllerURL,
		stats: &NetworkStats{
			Latency:     0,
			SuccessRate: 1.0,
			ErrorRate:   0.0,
			TimeoutRate: 0.0,
			LastUpdated: time.Now(),
			SampleCount: 0,
		},
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		stopChan: make(chan struct{}),
	}
}

// Start 开始监控
func (nm *NetworkMonitor) Start(interval time.Duration) {
	if nm.monitoring {
		return
	}

	nm.monitoring = true
	go nm.monitor(interval)
}

// Stop 停止监控
func (nm *NetworkMonitor) Stop() {
	if !nm.monitoring {
		return
	}

	nm.monitoring = false
	close(nm.stopChan)
}

// GetQuality 获取网络质量
func (nm *NetworkMonitor) GetQuality() NetworkQuality {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()

	stats := nm.stats

	// 基于延迟和成功率判断网络质量
	if stats.Latency < 50*time.Millisecond && stats.SuccessRate > 0.99 {
		return NetworkQualityExcellent
	} else if stats.Latency < 100*time.Millisecond && stats.SuccessRate > 0.95 {
		return NetworkQualityGood
	} else if stats.Latency < 500*time.Millisecond && stats.SuccessRate > 0.90 {
		return NetworkQualityFair
	} else if stats.Latency < 2*time.Second && stats.SuccessRate > 0.80 {
		return NetworkQualityPoor
	} else {
		return NetworkQualityVeryPoor
	}
}

// GetStats 获取网络统计
func (nm *NetworkMonitor) GetStats() NetworkStats {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()
	return *nm.stats
}

// IsWeakNetwork 判断是否为弱网环境
func (nm *NetworkMonitor) IsWeakNetwork() bool {
	quality := nm.GetQuality()
	return quality == NetworkQualityPoor || quality == NetworkQualityVeryPoor
}

// GetRecommendedConfig 获取推荐配置
func (nm *NetworkMonitor) GetRecommendedConfig() *WeakNetworkConfig {
	quality := nm.GetQuality()

	config := &WeakNetworkConfig{
		CacheTTL:          30 * time.Second,
		RetryMaxAttempts:  3,
		RetryBaseDelay:    100 * time.Millisecond,
		RetryMaxDelay:     5 * time.Second,
		RequestTimeout:    30 * time.Second,
		HeartbeatInterval: 5 * time.Second,
		EnableCompression: false,
		BatchSize:         1,
	}

	switch quality {
	case NetworkQualityExcellent:
		config.CacheTTL = 10 * time.Second
		config.RetryMaxAttempts = 1
		config.RetryBaseDelay = 50 * time.Millisecond
		config.RetryMaxDelay = 1 * time.Second
		config.RequestTimeout = 10 * time.Second
		config.HeartbeatInterval = 1 * time.Second
		config.EnableCompression = false
		config.BatchSize = 10

	case NetworkQualityGood:
		config.CacheTTL = 20 * time.Second
		config.RetryMaxAttempts = 2
		config.RetryBaseDelay = 100 * time.Millisecond
		config.RetryMaxDelay = 2 * time.Second
		config.RequestTimeout = 15 * time.Second
		config.HeartbeatInterval = 2 * time.Second
		config.EnableCompression = false
		config.BatchSize = 5

	case NetworkQualityFair:
		config.CacheTTL = 30 * time.Second
		config.RetryMaxAttempts = 3
		config.RetryBaseDelay = 200 * time.Millisecond
		config.RetryMaxDelay = 3 * time.Second
		config.RequestTimeout = 20 * time.Second
		config.HeartbeatInterval = 3 * time.Second
		config.EnableCompression = true
		config.BatchSize = 3

	case NetworkQualityPoor:
		config.CacheTTL = 60 * time.Second
		config.RetryMaxAttempts = 5
		config.RetryBaseDelay = 500 * time.Millisecond
		config.RetryMaxDelay = 10 * time.Second
		config.RequestTimeout = 30 * time.Second
		config.HeartbeatInterval = 10 * time.Second
		config.EnableCompression = true
		config.BatchSize = 2

	case NetworkQualityVeryPoor:
		config.CacheTTL = 120 * time.Second
		config.RetryMaxAttempts = 10
		config.RetryBaseDelay = 1 * time.Second
		config.RetryMaxDelay = 30 * time.Second
		config.RequestTimeout = 60 * time.Second
		config.HeartbeatInterval = 30 * time.Second
		config.EnableCompression = true
		config.BatchSize = 1
	}

	return config
}

// monitor 监控网络质量
func (nm *NetworkMonitor) monitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nm.performHealthCheck()
		case <-nm.stopChan:
			return
		}
	}
}

// performHealthCheck 执行健康检查
func (nm *NetworkMonitor) performHealthCheck() {
	start := time.Now()

	// 创建请求
	req, err := http.NewRequest("GET", nm.controllerURL+"/healthz", nil)
	if err != nil {
		nm.updateStats(false, 0, true)
		return
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	// 发送请求
	resp, err := nm.httpClient.Do(req)
	latency := time.Since(start)

	success := err == nil && resp != nil && resp.StatusCode == 200
	timeout := err != nil && isTimeoutError(err)

	if resp != nil {
		resp.Body.Close()
	}

	nm.updateStats(success, latency, timeout)
}

// updateStats 更新统计信息
func (nm *NetworkMonitor) updateStats(success bool, latency time.Duration, timeout bool) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	stats := nm.stats
	stats.SampleCount++

	// 更新延迟（使用指数移动平均）
	if stats.Latency == 0 {
		stats.Latency = latency
	} else {
		alpha := 0.1 // 平滑因子
		stats.Latency = time.Duration(float64(stats.Latency)*(1-alpha) + float64(latency)*alpha)
	}

	// 更新成功率
	if success {
		stats.SuccessRate = (stats.SuccessRate*float64(stats.SampleCount-1) + 1.0) / float64(stats.SampleCount)
	} else {
		stats.SuccessRate = (stats.SuccessRate * float64(stats.SampleCount-1)) / float64(stats.SampleCount)
	}

	// 更新错误率
	if !success {
		stats.ErrorRate = (stats.ErrorRate*float64(stats.SampleCount-1) + 1.0) / float64(stats.SampleCount)
	} else {
		stats.ErrorRate = (stats.ErrorRate * float64(stats.SampleCount-1)) / float64(stats.SampleCount)
	}

	// 更新超时率
	if timeout {
		stats.TimeoutRate = (stats.TimeoutRate*float64(stats.SampleCount-1) + 1.0) / float64(stats.SampleCount)
	} else {
		stats.TimeoutRate = (stats.TimeoutRate * float64(stats.SampleCount-1)) / float64(stats.SampleCount)
	}

	stats.LastUpdated = time.Now()
}

// isTimeoutError 检查是否为超时错误
func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}

// WeakNetworkConfig 弱网环境配置
type WeakNetworkConfig struct {
	CacheTTL          time.Duration
	RetryMaxAttempts  int
	RetryBaseDelay    time.Duration
	RetryMaxDelay     time.Duration
	RequestTimeout    time.Duration
	HeartbeatInterval time.Duration
	EnableCompression bool
	BatchSize         int
}

// String 返回配置的字符串表示
func (c *WeakNetworkConfig) String() string {
	return fmt.Sprintf("WeakNetworkConfig{CacheTTL: %v, RetryMaxAttempts: %d, RetryBaseDelay: %v, RetryMaxDelay: %v, RequestTimeout: %v, HeartbeatInterval: %v, EnableCompression: %v, BatchSize: %d}",
		c.CacheTTL, c.RetryMaxAttempts, c.RetryBaseDelay, c.RetryMaxDelay, c.RequestTimeout, c.HeartbeatInterval, c.EnableCompression, c.BatchSize)
}
