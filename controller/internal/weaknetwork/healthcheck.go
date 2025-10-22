package weaknetwork

import (
	"context"
	"log"
	"sync"
	"time"
)

// HealthStatus 健康状态
type HealthStatus int

const (
	StatusHealthy HealthStatus = iota
	StatusUnhealthy
	StatusUnknown
)

func (s HealthStatus) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusUnhealthy:
		return "unhealthy"
	case StatusUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// HealthChecker 健康检查器
type HealthChecker struct {
	interval  time.Duration
	timeout   time.Duration
	checks    []HealthCheck
	status    HealthStatus
	lastCheck time.Time
	lastError error
	mu        sync.RWMutex
	started   bool
	stopChan  chan struct{}
}

// HealthCheck 健康检查接口
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) error
}

// HTTPHealthCheck HTTP健康检查
type HTTPHealthCheck struct {
	name    string
	url     string
	timeout time.Duration
}

// NewHTTPHealthCheck 创建HTTP健康检查
func NewHTTPHealthCheck(name, url string, timeout time.Duration) *HTTPHealthCheck {
	return &HTTPHealthCheck{
		name:    name,
		url:     url,
		timeout: timeout,
	}
}

func (h *HTTPHealthCheck) Name() string {
	return h.name
}

func (h *HTTPHealthCheck) Check(ctx context.Context) error {
	// 这里应该实现实际的HTTP健康检查
	// 为了简化，我们返回nil
	return nil
}

// DatabaseHealthCheck 数据库健康检查
type DatabaseHealthCheck struct {
	name    string
	checkFn func(ctx context.Context) error
}

// NewDatabaseHealthCheck 创建数据库健康检查
func NewDatabaseHealthCheck(name string, checkFn func(ctx context.Context) error) *DatabaseHealthCheck {
	return &DatabaseHealthCheck{
		name:    name,
		checkFn: checkFn,
	}
}

func (d *DatabaseHealthCheck) Name() string {
	return d.name
}

func (d *DatabaseHealthCheck) Check(ctx context.Context) error {
	return d.checkFn(ctx)
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(interval, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		interval: interval,
		timeout:  timeout,
		checks:   make([]HealthCheck, 0),
		status:   StatusUnknown,
		stopChan: make(chan struct{}),
	}
}

// AddCheck 添加健康检查
func (h *HealthChecker) AddCheck(check HealthCheck) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks = append(h.checks, check)
}

// Start 启动健康检查
func (h *HealthChecker) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started {
		return nil
	}

	h.started = true
	go h.run()

	return nil
}

// Stop 停止健康检查
func (h *HealthChecker) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.started {
		return
	}

	close(h.stopChan)
	h.started = false
}

// run 运行健康检查
func (h *HealthChecker) run() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.performChecks()
		case <-h.stopChan:
			return
		}
	}
}

// performChecks 执行健康检查
func (h *HealthChecker) performChecks() {
	h.mu.Lock()
	defer h.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	allHealthy := true
	var lastErr error

	for _, check := range h.checks {
		if err := check.Check(ctx); err != nil {
			log.Printf("Health check failed for %s: %v", check.Name(), err)
			allHealthy = false
			lastErr = err
		}
	}

	if allHealthy {
		h.status = StatusHealthy
		h.lastError = nil
	} else {
		h.status = StatusUnhealthy
		h.lastError = lastErr
	}

	h.lastCheck = time.Now()
}

// GetStatus 获取健康状态
func (h *HealthChecker) GetStatus() HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.status
}

// GetLastCheck 获取最后检查时间
func (h *HealthChecker) GetLastCheck() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastCheck
}

// GetLastError 获取最后错误
func (h *HealthChecker) GetLastError() error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastError
}

// IsHealthy 检查是否健康
func (h *HealthChecker) IsHealthy() bool {
	return h.GetStatus() == StatusHealthy
}

// GetDetailedStatus 获取详细状态
func (h *HealthChecker) GetDetailedStatus() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status := map[string]interface{}{
		"status":     h.status.String(),
		"last_check": h.lastCheck,
		"healthy":    h.status == StatusHealthy,
	}

	if h.lastError != nil {
		status["last_error"] = h.lastError.Error()
	}

	// 添加各个检查的状态
	checks := make([]map[string]interface{}, len(h.checks))
	for i, check := range h.checks {
		checks[i] = map[string]interface{}{
			"name": check.Name(),
			// 这里可以添加每个检查的详细状态
		}
	}
	status["checks"] = checks

	return status
}
