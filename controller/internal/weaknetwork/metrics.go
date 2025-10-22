package weaknetwork

import (
	"sync"
	"time"
)

// MetricType 指标类型
type MetricType int

const (
	Counter MetricType = iota
	Gauge
	Histogram
)

// Metric 指标
type Metric struct {
	Name      string
	Type      MetricType
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	interval   time.Duration
	metrics    map[string]*Metric
	mu         sync.RWMutex
	started    bool
	stopChan   chan struct{}
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		interval: interval,
		metrics:  make(map[string]*Metric),
		stopChan: make(chan struct{}),
	}
}

// Start 启动指标收集
func (m *MetricsCollector) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.started {
		return nil
	}
	
	m.started = true
	go m.collect()
	
	return nil
}

// Stop 停止指标收集
func (m *MetricsCollector) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.started {
		return
	}
	
	close(m.stopChan)
	m.started = false
}

// collect 收集指标
func (m *MetricsCollector) collect() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.updateMetrics()
		case <-m.stopChan:
			return
		}
	}
}

// updateMetrics 更新指标
func (m *MetricsCollector) updateMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	
	// 更新系统指标
	m.updateSystemMetrics(now)
	
	// 更新网络指标
	m.updateNetworkMetrics(now)
	
	// 更新业务指标
	m.updateBusinessMetrics(now)
}

// updateSystemMetrics 更新系统指标
func (m *MetricsCollector) updateSystemMetrics(now time.Time) {
	// CPU使用率
	m.setMetric("system.cpu.usage", Gauge, 0.0, map[string]string{"type": "cpu"}, now)
	
	// 内存使用率
	m.setMetric("system.memory.usage", Gauge, 0.0, map[string]string{"type": "memory"}, now)
	
	// 磁盘使用率
	m.setMetric("system.disk.usage", Gauge, 0.0, map[string]string{"type": "disk"}, now)
}

// updateNetworkMetrics 更新网络指标
func (m *MetricsCollector) updateNetworkMetrics(now time.Time) {
	// 网络延迟
	m.setMetric("network.latency", Histogram, 0.0, map[string]string{"type": "latency"}, now)
	
	// 网络错误率
	m.setMetric("network.error_rate", Gauge, 0.0, map[string]string{"type": "error_rate"}, now)
	
	// 网络吞吐量
	m.setMetric("network.throughput", Gauge, 0.0, map[string]string{"type": "throughput"}, now)
}

// updateBusinessMetrics 更新业务指标
func (m *MetricsCollector) updateBusinessMetrics(now time.Time) {
	// 请求总数
	m.setMetric("business.requests.total", Counter, 0.0, map[string]string{"type": "total"}, now)
	
	// 成功请求数
	m.setMetric("business.requests.success", Counter, 0.0, map[string]string{"type": "success"}, now)
	
	// 失败请求数
	m.setMetric("business.requests.failure", Counter, 0.0, map[string]string{"type": "failure"}, now)
	
	// 响应时间
	m.setMetric("business.response_time", Histogram, 0.0, map[string]string{"type": "response_time"}, now)
}

// setMetric 设置指标
func (m *MetricsCollector) setMetric(name string, metricType MetricType, value float64, labels map[string]string, timestamp time.Time) {
	metric := &Metric{
		Name:      name,
		Type:      metricType,
		Value:     value,
		Labels:    labels,
		Timestamp: timestamp,
	}
	m.metrics[name] = metric
}

// IncrementCounter 增加计数器
func (m *MetricsCollector) IncrementCounter(name string, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := m.getMetricKey(name, labels)
	if metric, exists := m.metrics[key]; exists {
		metric.Value++
		metric.Timestamp = time.Now()
	} else {
		m.setMetric(key, Counter, 1.0, labels, time.Now())
	}
}

// SetGauge 设置仪表
func (m *MetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := m.getMetricKey(name, labels)
	m.setMetric(key, Gauge, value, labels, time.Now())
}

// RecordHistogram 记录直方图
func (m *MetricsCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := m.getMetricKey(name, labels)
	if metric, exists := m.metrics[key]; exists {
		// 更新直方图值（这里简化处理）
		metric.Value = value
		metric.Timestamp = time.Now()
	} else {
		m.setMetric(key, Histogram, value, labels, time.Now())
	}
}

// getMetricKey 获取指标键
func (m *MetricsCollector) getMetricKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += ":" + k + "=" + v
	}
	return key
}

// GetMetrics 获取所有指标
func (m *MetricsCollector) GetMetrics() map[string]*Metric {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 返回副本
	metrics := make(map[string]*Metric)
	for k, v := range m.metrics {
		metrics[k] = v
	}
	return metrics
}

// GetMetric 获取指定指标
func (m *MetricsCollector) GetMetric(name string, labels map[string]string) *Metric {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	key := m.getMetricKey(name, labels)
	return m.metrics[key]
}

// GetCounterValue 获取计数器值
func (m *MetricsCollector) GetCounterValue(name string, labels map[string]string) float64 {
	metric := m.GetMetric(name, labels)
	if metric != nil && metric.Type == Counter {
		return metric.Value
	}
	return 0.0
}

// GetGaugeValue 获取仪表值
func (m *MetricsCollector) GetGaugeValue(name string, labels map[string]string) float64 {
	metric := m.GetMetric(name, labels)
	if metric != nil && metric.Type == Gauge {
		return metric.Value
	}
	return 0.0
}

// GetHistogramValue 获取直方图值
func (m *MetricsCollector) GetHistogramValue(name string, labels map[string]string) float64 {
	metric := m.GetMetric(name, labels)
	if metric != nil && metric.Type == Histogram {
		return metric.Value
	}
	return 0.0
}

// ClearMetrics 清空指标
func (m *MetricsCollector) ClearMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.metrics = make(map[string]*Metric)
}

// GetMetricsSummary 获取指标摘要
func (m *MetricsCollector) GetMetricsSummary() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	summary := map[string]interface{}{
		"total_metrics": len(m.metrics),
		"timestamp":     time.Now(),
	}
	
	// 按类型统计
	counters := 0
	gauges := 0
	histograms := 0
	
	for _, metric := range m.metrics {
		switch metric.Type {
		case Counter:
			counters++
		case Gauge:
			gauges++
		case Histogram:
			histograms++
		}
	}
	
	summary["counters"] = counters
	summary["gauges"] = gauges
	summary["histograms"] = histograms
	
	return summary
}
