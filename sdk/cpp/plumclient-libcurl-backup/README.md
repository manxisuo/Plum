# Plum Client C++ SDK

Plum Client C++ SDK 是一个用于与 Plum Controller 交互的客户端库，提供完整的服务发现、服务调用和弱网环境支持功能。

## 功能特性

- **服务注册与发现**: 支持服务注册、心跳、注销和发现
- **服务调用**: 支持直接调用、重试调用和负载均衡调用
- **弱网环境支持**: 智能缓存、自适应重试、网络质量监控
- **配置管理**: 灵活的配置选项和运行时配置更新
- **监控与统计**: 详细的性能指标和缓存统计

## 构建

### 依赖项

- C++17 或更高版本
- CMake 3.16 或更高版本
- libcurl
- nlohmann/json
- pthread

### 构建步骤

```bash
# 在项目根目录
make plumclient
```

## 使用方法

### 基本使用

```cpp
#include "plum_client.hpp"

using namespace plumclient;

// 创建客户端
PlumClient client("http://localhost:8080");

// 启动客户端
if (!client.start()) {
    std::cerr << "Failed to start client" << std::endl;
    return -1;
}

// 注册服务
ServiceRegistration registration;
registration.instanceId = "service-001";
registration.serviceName = "my-service";
registration.nodeId = "node-001";
registration.ip = "192.168.1.100";
registration.port = 8080;
registration.protocol = "http";
registration.version = "1.0.0";
registration.labels["env"] = "production";

if (client.registerService(registration)) {
    std::cout << "Service registered successfully" << std::endl;
}

// 发现服务
auto endpoints = client.discoverService("my-service");
for (const auto& endpoint : endpoints) {
    std::cout << "Found endpoint: " << endpoint.ip << ":" << endpoint.port << std::endl;
}

// 随机发现服务
auto randomEndpoint = client.discoverRandomService("my-service");
if (randomEndpoint) {
    std::cout << "Random endpoint: " << randomEndpoint->ip << ":" << randomEndpoint->port << std::endl;
}

// 调用服务
auto result = client.callService("my-service", "GET", "/api/health");
if (result.success) {
    std::cout << "Service call successful: " << result.body << std::endl;
}

// 停止客户端
client.stop();
```

### 弱网环境支持

```cpp
// 配置弱网环境支持
WeakNetworkConfig config;
config.cacheTTL = std::chrono::seconds(60);
config.cacheMaxSize = 1000;
config.retryMaxAttempts = 5;
config.retryBaseDelay = std::chrono::milliseconds(200);
config.retryMaxDelay = std::chrono::milliseconds(5000);
config.requestTimeout = std::chrono::seconds(30);
config.rateLimitRPS = 100;
config.rateLimitBurst = 200;

// 创建带配置的客户端
PlumClient client("http://localhost:8080", config);

// 启用弱网环境支持
client.enableWeakNetworkSupport();

// 使用带重试的服务调用
auto result = client.callServiceWithRetry("my-service", "GET", "/api/data", {}, "", 3);

// 使用负载均衡的服务调用
auto result2 = client.callServiceWithLoadBalance("my-service", "POST", "/api/update", 
    {{"Content-Type", "application/json"}}, "{\"data\": \"value\"}");

// 监控网络质量
if (client.isWeakNetwork()) {
    std::cout << "Weak network detected" << std::endl;
}

auto metrics = client.getNetworkMetrics();
for (const auto& metric : metrics) {
    std::cout << metric.first << ": " << metric.second << std::endl;
}
```

### 配置管理

```cpp
// 获取当前配置
auto currentConfig = client.getConfig();
std::cout << "Cache TTL: " << currentConfig.cacheTTL.count() << " seconds" << std::endl;

// 更新配置
WeakNetworkConfig newConfig = currentConfig;
newConfig.cacheTTL = std::chrono::seconds(120);
newConfig.retryMaxAttempts = 10;
client.updateConfig(newConfig);

// 获取客户端状态
auto status = client.getStatus();
for (const auto& stat : status) {
    std::cout << stat.first << ": " << stat.second << std::endl;
}
```

## API 参考

### PlumClient 类

#### 构造函数
- `PlumClient(const std::string& controllerUrl)`: 使用默认配置创建客户端
- `PlumClient(const std::string& controllerUrl, const WeakNetworkConfig& config)`: 使用自定义配置创建客户端

#### 生命周期管理
- `bool start()`: 启动客户端
- `void stop()`: 停止客户端
- `bool isRunning() const`: 检查客户端是否运行中

#### 服务管理
- `bool registerService(const ServiceRegistration& registration)`: 注册服务
- `bool heartbeatService(const ServiceHeartbeat& heartbeat)`: 发送服务心跳
- `bool unregisterService(const std::string& instanceId)`: 注销服务

#### 服务发现
- `std::vector<Endpoint> discoverService(const DiscoveryRequest& request)`: 发现服务
- `std::vector<Endpoint> discoverService(const std::string& service, const std::string& version = "", const std::string& protocol = "")`: 发现服务（简化版本）
- `std::optional<Endpoint> discoverRandomService(const DiscoveryRequest& request)`: 随机发现服务
- `std::optional<Endpoint> discoverRandomService(const std::string& service, const std::string& version = "", const std::string& protocol = "")`: 随机发现服务（简化版本）

#### 服务调用
- `ServiceCallResult callService(...)`: 直接调用服务
- `ServiceCallResult callServiceWithRetry(...)`: 带重试的服务调用
- `ServiceCallResult callServiceWithLoadBalance(...)`: 负载均衡服务调用

#### 弱网环境支持
- `void enableWeakNetworkSupport()`: 启用弱网环境支持
- `void disableWeakNetworkSupport()`: 禁用弱网环境支持
- `bool isWeakNetworkSupportEnabled() const`: 检查弱网环境支持是否启用

#### 网络质量监控
- `NetworkQuality getNetworkQuality() const`: 获取网络质量
- `bool isWeakNetwork() const`: 检查是否为弱网环境
- `std::map<std::string, std::string> getNetworkMetrics() const`: 获取网络指标

#### 缓存管理
- `void clearCache()`: 清空缓存
- `size_t getCacheSize() const`: 获取缓存大小
- `std::map<std::string, std::string> getCacheStats() const`: 获取缓存统计

#### 配置管理
- `void updateConfig(const WeakNetworkConfig& config)`: 更新配置
- `WeakNetworkConfig getConfig() const`: 获取当前配置

#### 状态查询
- `std::map<std::string, std::string> getStatus() const`: 获取客户端状态
- `bool isHealthy() const`: 检查客户端健康状态

## 示例程序

项目包含一个完整的示例程序 `service_client_example`，演示了所有主要功能的使用方法。

```bash
# 构建示例程序
make service_client_example

# 运行示例程序
./sdk/cpp/build/examples/service_client_example/service_client_example
```

## 注意事项

1. **线程安全**: 客户端是线程安全的，可以在多线程环境中使用
2. **资源管理**: 客户端会自动管理内部资源，包括线程和缓存
3. **错误处理**: 所有方法都有适当的错误处理，建议检查返回值
4. **配置**: 弱网环境支持需要适当的配置才能发挥最佳效果
5. **网络质量**: 网络质量监控需要一定的时间才能产生准确的指标

## 故障排除

### 常见问题

1. **编译错误**: 确保安装了所有依赖项，特别是 libcurl 和 nlohmann/json
2. **链接错误**: 确保链接了 pthread 库
3. **运行时错误**: 检查 Controller 是否运行，网络连接是否正常
4. **性能问题**: 调整弱网环境配置，特别是缓存和重试设置

### 调试

启用详细日志输出可以帮助调试问题：

```cpp
// 获取详细的客户端状态
auto status = client.getStatus();
for (const auto& stat : status) {
    std::cout << stat.first << ": " << stat.second << std::endl;
}

// 获取网络指标
auto metrics = client.getNetworkMetrics();
for (const auto& metric : metrics) {
    std::cout << metric.first << ": " << metric.second << std::endl;
}

// 获取缓存统计
auto cacheStats = client.getCacheStats();
for (const auto& stat : cacheStats) {
    std::cout << stat.first << ": " << stat.second << std::endl;
}
```

## 许可证

本项目使用与 Plum 项目相同的许可证。