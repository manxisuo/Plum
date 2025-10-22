package weaknetwork

import (
	"context"
	"sync"
	"time"
)

// CircuitState 熔断器状态
type CircuitState int

const (
	StateClosed   CircuitState = iota // 关闭状态
	StateOpen                         // 开启状态
	StateHalfOpen                     // 半开状态
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name          string
	maxRequests   int                                      // 半开状态最大请求数
	interval      time.Duration                            // 统计间隔
	timeout       time.Duration                            // 熔断超时时间
	readyToTrip   func(counts Counts) bool                 // 熔断条件
	onStateChange func(name string, from, to CircuitState) // 状态变化回调

	mutex      sync.Mutex
	state      CircuitState
	generation uint64
	counts     Counts
	expiry     time.Time
}

// Counts 计数器
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// RequestResult 请求结果
type RequestResult int

const (
	Success RequestResult = iota
	Failure
	Timeout
	Rejected
)

// Config 熔断器配置
type Config struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange func(name string, from, to CircuitState)
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(cfg Config) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          cfg.Name,
		maxRequests:   int(cfg.MaxRequests),
		interval:      cfg.Interval,
		timeout:       cfg.Timeout,
		readyToTrip:   cfg.ReadyToTrip,
		onStateChange: cfg.OnStateChange,
	}

	// 设置默认熔断条件
	if cb.readyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 5
		}
	}

	cb.toNewGeneration(time.Now())
	return cb
}

// Execute 执行请求
func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, err == nil)
	return result, err
}

// ExecuteWithContext 带上下文的执行请求
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		cb.afterRequest(generation, false)
		return nil, ctx.Err()
	default:
	}

	result, err := req()
	cb.afterRequest(generation, err == nil)
	return result, err
}

// State 获取当前状态
func (cb *CircuitBreaker) State() CircuitState {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts 获取计数器
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}

// beforeRequest 请求前检查
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, ErrOpenState
	}

	if state == StateHalfOpen && cb.counts.Requests >= uint32(cb.maxRequests) {
		return generation, ErrTooManyRequests
	}

	cb.counts.Requests++
	return generation, nil
}

// afterRequest 请求后处理
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// currentState 获取当前状态
func (cb *CircuitBreaker) currentState(now time.Time) (CircuitState, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// onSuccess 成功处理
func (cb *CircuitBreaker) onSuccess(state CircuitState, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0
		cb.counts.TotalSuccesses++
	case StateHalfOpen:
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0
		cb.counts.TotalSuccesses++
		if cb.counts.ConsecutiveSuccesses >= uint32(cb.maxRequests) {
			cb.setState(StateClosed, now)
		}
	}
}

// onFailure 失败处理
func (cb *CircuitBreaker) onFailure(state CircuitState, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.ConsecutiveFailures++
		cb.counts.ConsecutiveSuccesses = 0
		cb.counts.TotalFailures++
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

// setState 设置状态
func (cb *CircuitBreaker) setState(state CircuitState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

// toNewGeneration 创建新代
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	var zero Counts
	cb.counts = zero

	var expiry time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval > 0 {
			expiry = now.Add(cb.interval)
		}
	case StateOpen:
		expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		expiry = time.Time{}
	}
	cb.expiry = expiry
}

// 错误定义
var (
	ErrOpenState       = &CircuitBreakerError{Message: "circuit breaker is open"}
	ErrTooManyRequests = &CircuitBreakerError{Message: "too many requests"}
)

// CircuitBreakerError 熔断器错误
type CircuitBreakerError struct {
	Message string
}

func (e *CircuitBreakerError) Error() string {
	return e.Message
}

// HTTPCircuitBreakerMiddleware HTTP熔断器中间件
type HTTPCircuitBreakerMiddleware struct {
	breaker *CircuitBreaker
}

// NewHTTPCircuitBreakerMiddleware 创建HTTP熔断器中间件
func NewHTTPCircuitBreakerMiddleware(breaker *CircuitBreaker) *HTTPCircuitBreakerMiddleware {
	return &HTTPCircuitBreakerMiddleware{
		breaker: breaker,
	}
}

// Handler HTTP熔断器处理函数
func (m *HTTPCircuitBreakerMiddleware) Handler(next func()) func() {
	return func() {
		_, err := m.breaker.Execute(func() (interface{}, error) {
			next()
			return nil, nil
		})

		if err != nil {
			// 熔断器开启，返回503 Service Unavailable
			return
		}
	}
}
