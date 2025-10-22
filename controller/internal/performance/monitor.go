package performance

import (
	"log"
	"sync"
	"time"
)

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	mu sync.RWMutex

	// 重启时间统计
	restartTimes map[string]time.Duration

	// 迁移时间统计
	migrationTimes map[string]time.Duration

	// 故障检测时间统计
	detectionTimes map[string]time.Duration
}

var monitor *PerformanceMonitor
var once sync.Once

// GetMonitor 获取全局性能监控器实例
func GetMonitor() *PerformanceMonitor {
	once.Do(func() {
		monitor = &PerformanceMonitor{
			restartTimes:   make(map[string]time.Duration),
			migrationTimes: make(map[string]time.Duration),
			detectionTimes: make(map[string]time.Duration),
		}
	})
	return monitor
}

// RecordRestartTime 记录应用重启时间
func (m *PerformanceMonitor) RecordRestartTime(instanceID string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.restartTimes[instanceID] = duration
	log.Printf("性能监控: 实例 %s 重启耗时 %v", instanceID, duration)

	// 检查是否超过2秒阈值
	if duration > 2*time.Second {
		log.Printf("⚠️  性能警告: 实例 %s 重启时间 %v 超过2秒阈值", instanceID, duration)
	}
}

// RecordMigrationTime 记录应用迁移时间
func (m *PerformanceMonitor) RecordMigrationTime(instanceID string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.migrationTimes[instanceID] = duration
	log.Printf("性能监控: 实例 %s 迁移耗时 %v", instanceID, duration)

	// 检查是否超过2秒阈值
	if duration > 2*time.Second {
		log.Printf("⚠️  性能警告: 实例 %s 迁移时间 %v 超过2秒阈值", instanceID, duration)
	}
}

// RecordDetectionTime 记录故障检测时间
func (m *PerformanceMonitor) RecordDetectionTime(nodeID string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.detectionTimes[nodeID] = duration
	log.Printf("性能监控: 节点 %s 故障检测耗时 %v", nodeID, duration)
}

// GetRestartStats 获取重启时间统计
func (m *PerformanceMonitor) GetRestartStats() map[string]time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]time.Duration)
	for k, v := range m.restartTimes {
		stats[k] = v
	}
	return stats
}

// GetMigrationStats 获取迁移时间统计
func (m *PerformanceMonitor) GetMigrationStats() map[string]time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]time.Duration)
	for k, v := range m.migrationTimes {
		stats[k] = v
	}
	return stats
}

// GetDetectionStats 获取检测时间统计
func (m *PerformanceMonitor) GetDetectionStats() map[string]time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]time.Duration)
	for k, v := range m.detectionTimes {
		stats[k] = v
	}
	return stats
}

// GetAverageRestartTime 获取平均重启时间
func (m *PerformanceMonitor) GetAverageRestartTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.restartTimes) == 0 {
		return 0
	}

	var total time.Duration
	for _, duration := range m.restartTimes {
		total += duration
	}
	return total / time.Duration(len(m.restartTimes))
}

// GetAverageMigrationTime 获取平均迁移时间
func (m *PerformanceMonitor) GetAverageMigrationTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.migrationTimes) == 0 {
		return 0
	}

	var total time.Duration
	for _, duration := range m.migrationTimes {
		total += duration
	}
	return total / time.Duration(len(m.migrationTimes))
}

// ClearStats 清理统计数据
func (m *PerformanceMonitor) ClearStats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.restartTimes = make(map[string]time.Duration)
	m.migrationTimes = make(map[string]time.Duration)
	m.detectionTimes = make(map[string]time.Duration)

	log.Printf("性能监控: 统计数据已清理")
}
