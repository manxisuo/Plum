# Plum弱网环境支持

## 概述

Plum系统针对弱网环境提供了全面的支持，包括智能服务缓存、自适应重试策略、网络质量监控和动态配置调整等功能。

## 核心特性

### 1. 智能服务缓存系统

- **多级缓存**: 支持内存缓存，减少网络请求
- **TTL管理**: 可配置的缓存过期时间
- **自动清理**: 自动清理过期缓存条目
- **缓存统计**: 提供缓存命中率和大小统计

```go
// 创建智能缓存
cache := NewSmartCache(30 * time.Second)

// 设置缓存
cache.Set("service:user-service", endpoints, 60*time.Second)

// 获取缓存
if data, exists := cache.Get("service:user-service"); exists {
    endpoints := data.([]Endpoint)
    // 使用缓存的服务端点
}
```

### 2. 自适应重试策略

支持多种重试策略：

- **指数退避**: 延迟时间指数增长，避免网络拥塞
- **线性退避**: 延迟时间线性增长
- **固定延迟**: 固定延迟时间

```go
// 创建指数退避策略
strategy := NewExponentialBackoffStrategy(
    100*time.Millisecond, // 基础延迟
    5*time.Second,        // 最大延迟
    3,                    // 最大重试次数
)

// 创建支持重试的HTTP客户端
client := NewRetryableHTTPClient(httpClient, strategy)
```

### 3. 网络质量监控

实时监控网络质量，自动调整配置：

- **延迟监控**: 持续监控请求延迟
- **成功率统计**: 统计请求成功率
- **质量评估**: 自动评估网络质量等级
- **自适应配置**: 根据网络质量自动调整参数

```go
// 创建网络监控器
monitor := NewNetworkMonitor("http://localhost:8080")

// 开始监控
monitor.Start(5 * time.Second)

// 获取网络质量
quality := monitor.GetQuality()
isWeak := monitor.IsWeakNetwork()

// 获取推荐配置
config := monitor.GetRecommendedConfig()
```

### 4. 弱网环境配置

针对不同网络质量提供优化配置：

| 网络质量 | 缓存TTL | 重试次数 | 基础延迟 | 最大延迟 | 请求超时 | 心跳间隔 |
|---------|---------|----------|----------|----------|----------|----------|
| 优秀    | 10s     | 1        | 50ms     | 1s       | 10s      | 1s       |
| 良好    | 20s     | 2        | 100ms    | 2s       | 15s      | 2s       |
| 一般    | 30s     | 3        | 200ms    | 3s       | 20s      | 3s       |
| 差      | 60s     | 5        | 500ms    | 10s      | 30s      | 10s      |
| 很差    | 120s    | 10       | 1s       | 30s      | 60s      | 30s      |

## 使用方法

### 1. 基本使用

```go
// 创建Plum客户端（自动启用弱网支持）
client := plum.NewPlumClient("http://localhost:8080")

// 启用网络监控
client.StartNetworkMonitoring(5 * time.Second)

// 启用自适应模式
client.EnableAdaptiveMode()

// 正常使用服务发现
endpoints, err := client.DiscoverService("user-service", "", "")
```

### 2. 自定义配置

```go
// 创建弱网环境配置
config := &plum.WeakNetworkConfig{
    CacheTTL:           2 * time.Minute,
    RetryMaxAttempts:   10,
    RetryBaseDelay:     1 * time.Second,
    RetryMaxDelay:      30 * time.Second,
    RequestTimeout:     60 * time.Second,
    HeartbeatInterval:  30 * time.Second,
    EnableCompression:  true,
    BatchSize:          1,
}

// 使用自定义配置创建客户端
client := plum.NewPlumClientWithConfig("http://localhost:8080", config)
```

### 3. 网络状态监控

```go
// 获取网络质量
quality := client.GetNetworkQuality()
fmt.Printf("网络质量: %s\n", quality)

// 检查是否为弱网环境
if client.IsWeakNetwork() {
    fmt.Println("当前处于弱网环境")
}

// 获取网络统计
stats := client.GetNetworkStats()
fmt.Printf("平均延迟: %v\n", stats.Latency)
fmt.Printf("成功率: %.2f%%\n", stats.SuccessRate*100)
```

## 测试工具

### 1. 弱网环境测试

```bash
# 运行弱网环境测试
./tools/run_weak_network_test.sh
```

测试内容：
- 20个并发客户端
- 2分钟测试时间
- 网络质量监控
- 自适应配置验证
- 性能指标统计

### 2. 弱网环境示例

```bash
# 运行弱网环境示例
go run sdk/go/examples/weak_network_example.go
```

示例功能：
- 网络监控演示
- 自适应配置调整
- 服务发现测试
- 配置对比展示

## 性能指标

### 1. 缓存性能

- **缓存命中率**: >80% (弱网环境)
- **缓存延迟**: <1ms
- **内存使用**: 可配置，默认30秒TTL

### 2. 重试性能

- **重试成功率**: >90% (网络恢复后)
- **重试延迟**: 指数退避，避免网络拥塞
- **最大重试次数**: 可配置，默认3-10次

### 3. 网络适应性

- **检测延迟**: <5秒
- **配置调整**: 实时生效
- **弱网识别**: 准确率>95%

## 最佳实践

### 1. 配置建议

- **生产环境**: 启用自适应模式，使用默认配置
- **弱网环境**: 增加缓存TTL和重试次数
- **测试环境**: 使用固定配置，便于调试

### 2. 监控建议

- **定期检查**: 监控网络质量变化
- **日志记录**: 记录重试和缓存统计
- **告警设置**: 设置弱网环境告警阈值

### 3. 故障处理

- **网络中断**: 依赖缓存和重试机制
- **服务不可用**: 自动切换到其他端点
- **配置错误**: 回退到默认配置

## 故障排除

### 1. 常见问题

**Q: 缓存不生效？**
A: 检查缓存TTL配置，确保服务名称正确

**Q: 重试次数过多？**
A: 检查网络质量，调整重试策略参数

**Q: 自适应模式不工作？**
A: 确保网络监控已启动，检查配置参数

### 2. 调试方法

```go
// 启用详细日志
client.EnableAdaptiveMode()

// 检查当前配置
config := client.GetConfig()
fmt.Printf("当前配置: %+v\n", config)

// 检查网络状态
stats := client.GetNetworkStats()
fmt.Printf("网络统计: %+v\n", stats)
```

## 弱网支持开启模式

### 配置选项

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `WEAK_NETWORK_ENABLED` | `true` | 是否启用弱网环境支持 |
| `ADAPTIVE_ENABLED` | `true` | 是否启用自适应管理 |

### 三种开启模式

#### 1. 手动模式 (`WEAK_NETWORK_ENABLED=true`, `ADAPTIVE_ENABLED=false`)

**特点：**
- 弱网支持始终开启
- 不进行网络质量检测
- 使用固定配置

**适用场景：**
- 开发环境
- 测试环境
- 网络环境已知且稳定的生产环境

**配置示例：**
```bash
WEAK_NETWORK_ENABLED=true
ADAPTIVE_ENABLED=false
```

#### 2. 自适应模式 (`WEAK_NETWORK_ENABLED=true`, `ADAPTIVE_ENABLED=true`) **推荐**

**特点：**
- 根据网络质量自动开启/关闭
- 网络良好时：关闭弱网支持，减少开销
- 网络差时：开启弱网支持，提高稳定性

**适用场景：**
- 生产环境（推荐）
- 网络环境变化较大的环境
- 需要平衡性能和稳定性的环境

**配置示例：**
```bash
WEAK_NETWORK_ENABLED=true
ADAPTIVE_ENABLED=true
```

#### 3. 禁用模式 (`WEAK_NETWORK_ENABLED=false`)

**特点：**
- 完全禁用弱网环境支持
- 最小化系统开销
- 适用于网络环境非常稳定的场景

**适用场景：**
- 高性能环境
- 内网环境
- 网络环境极其稳定的场景

**配置示例：**
```bash
WEAK_NETWORK_ENABLED=false
```

### 自适应逻辑

| 网络条件 | 延迟 | 错误率 | 弱网支持 | 配置级别 | 说明 |
|----------|------|--------|----------|----------|------|
| 良好 | <50ms | <1% | ❌ 关闭 | 轻量级 | 网络质量优秀，无需额外保护 |
| 一般 | <200ms | <5% | ✅ 开启 | 中等 | 网络质量一般，启用基础保护 |
| 差 | <1000ms | <20% | ✅ 开启 | 强化 | 网络质量较差，启用强化保护 |
| 很差 | >1000ms | >20% | ✅ 开启 | 最强 | 网络质量很差，启用最强保护 |

### 配置级别说明

#### 轻量级配置（网络良好）
```bash
RATE_LIMIT_RPS=2000
RATE_LIMIT_BURST=4000
CIRCUIT_BREAKER_TIMEOUT=30s
RETRY_MAX_ATTEMPTS=1
CACHE_TTL=10s
HEALTH_CHECK_INTERVAL=60s
```

#### 中等配置（网络一般）
```bash
RATE_LIMIT_RPS=1000
RATE_LIMIT_BURST=2000
CIRCUIT_BREAKER_TIMEOUT=60s
RETRY_MAX_ATTEMPTS=3
CACHE_TTL=30s
HEALTH_CHECK_INTERVAL=30s
```

#### 强化配置（网络差）
```bash
RATE_LIMIT_RPS=500
RATE_LIMIT_BURST=1000
CIRCUIT_BREAKER_TIMEOUT=120s
RETRY_MAX_ATTEMPTS=5
CACHE_TTL=60s
HEALTH_CHECK_INTERVAL=15s
```

#### 最强配置（网络很差）
```bash
RATE_LIMIT_RPS=200
RATE_LIMIT_BURST=500
CIRCUIT_BREAKER_TIMEOUT=300s
RETRY_MAX_ATTEMPTS=10
CACHE_TTL=120s
HEALTH_CHECK_INTERVAL=10s
```

### 监控弱网支持状态

#### 通过代码监控
```go
// 获取弱网支持状态
status := weakNetworkManager.GetStatus()
fmt.Printf("弱网支持启用: %v\n", status["enabled"])
fmt.Printf("自适应模式: %v\n", status["adaptive_enabled"])

// 获取网络条件
if adaptive := weakNetworkManager.GetAdaptive(); adaptive != nil {
    condition := adaptive.GetNetworkCondition()
    fmt.Printf("网络条件: %s\n", condition.String())
    
    metrics := adaptive.GetNetworkMetrics()
    fmt.Printf("平均延迟: %v\n", metrics["avg_latency"])
    fmt.Printf("错误率: %.2f%%\n", metrics["error_rate"].(float64)*100)
}
```

#### 通过API监控
```bash
# 获取弱网支持状态
curl http://localhost:8080/v1/weak-network/status

# 获取网络质量指标
curl http://localhost:8080/v1/weak-network/metrics
```

### 最佳实践

#### 1. 生产环境配置
```bash
# 推荐配置
WEAK_NETWORK_ENABLED=true
ADAPTIVE_ENABLED=true
RATE_LIMIT_RPS=1000
CIRCUIT_BREAKER_ENABLED=true
RETRY_ENABLED=true
HEALTH_CHECK_ENABLED=true
```

#### 2. 开发环境配置
```bash
# 开发环境配置
WEAK_NETWORK_ENABLED=true
ADAPTIVE_ENABLED=false
RATE_LIMIT_RPS=5000
CIRCUIT_BREAKER_ENABLED=false
RETRY_ENABLED=true
```

#### 3. 高性能环境配置
```bash
# 高性能环境配置
WEAK_NETWORK_ENABLED=false
# 其他组件保持默认值
```

#### 4. 弱网环境配置
```bash
# 弱网环境配置
WEAK_NETWORK_ENABLED=true
ADAPTIVE_ENABLED=true
RATE_LIMIT_RPS=200
CIRCUIT_BREAKER_TIMEOUT=300s
RETRY_MAX_ATTEMPTS=10
CACHE_TTL=120s
```

### 故障排除

#### 问题1：弱网支持没有自动开启
**原因：** 自适应模式检测到网络质量良好
**解决：** 检查网络质量指标，或切换到手动模式

#### 问题2：系统响应变慢
**原因：** 弱网支持配置过于保守
**解决：** 调整限流和重试参数，或检查网络质量

#### 问题3：自适应模式频繁切换
**原因：** 网络质量不稳定
**解决：** 调整自适应检测间隔和阈值

## 更新日志

- **v1.0.0**: 初始版本，支持基本弱网环境功能
- **v1.1.0**: 添加网络质量监控和自适应配置
- **v1.2.0**: 优化缓存策略和重试机制
- **v1.3.0**: 添加自适应开启模式，支持根据网络质量自动调整
