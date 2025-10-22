#include <iostream>
#include <thread>
#include <chrono>
#include <random>
#include "plum_client.hpp"

using namespace plumclient;

int main() {
    std::cout << "=== Plum Client 示例程序 ===" << std::endl;
    
    // 创建客户端
    std::string controllerUrl = "http://localhost:8080";
    PlumClient client(controllerUrl);
    
    // 启动客户端
    if (!client.start()) {
        std::cerr << "启动客户端失败" << std::endl;
        return 1;
    }
    
    std::cout << "客户端已启动" << std::endl;
    
    // 启用弱网环境支持
    client.enableWeakNetworkSupport();
    std::cout << "弱网环境支持已启用" << std::endl;
    
    // 注册服务
    ServiceRegistration registration;
    registration.instanceId = "example-service-001";
    registration.serviceName = "example-service";
    registration.nodeId = "node-001";
    registration.ip = "127.0.0.1";
    registration.port = 9090;
    registration.protocol = "http";
    registration.version = "1.0.0";
    registration.labels["env"] = "test";
    registration.labels["region"] = "us-west";
    
    if (client.registerService(registration)) {
        std::cout << "服务注册成功" << std::endl;
    } else {
        std::cerr << "服务注册失败" << std::endl;
    }
    
    // 发送心跳
    ServiceHeartbeat heartbeat;
    heartbeat.instanceId = "example-service-001";
    
    Endpoint endpoint;
    endpoint.serviceName = "example-service";
    endpoint.instanceId = "example-service-001";
    endpoint.nodeId = "node-001";
    endpoint.ip = "127.0.0.1";
    endpoint.port = 9090;
    endpoint.protocol = "http";
    endpoint.version = "1.0.0";
    endpoint.healthy = true;
    endpoint.labels["env"] = "test";
    endpoint.labels["region"] = "us-west";
    
    heartbeat.endpoints.push_back(endpoint);
    
    if (client.heartbeatService(heartbeat)) {
        std::cout << "服务心跳发送成功" << std::endl;
    } else {
        std::cerr << "服务心跳发送失败" << std::endl;
    }
    
    // 等待一段时间让服务注册生效
    std::this_thread::sleep_for(std::chrono::seconds(2));
    
    // 服务发现
    std::cout << "\n=== 服务发现测试 ===" << std::endl;
    
    auto endpoints = client.discoverService("example-service");
    std::cout << "发现 " << endpoints.size() << " 个端点:" << std::endl;
    
    for (const auto& ep : endpoints) {
        std::cout << "  - " << ep.serviceName << " (" << ep.instanceId << ") "
                  << ep.protocol << "://" << ep.ip << ":" << ep.port
                  << " [" << (ep.healthy ? "健康" : "不健康") << "]" << std::endl;
    }
    
    // 随机服务发现
    auto randomEndpoint = client.discoverRandomService("example-service");
    if (randomEndpoint) {
        std::cout << "\n随机选择的端点: " << randomEndpoint->serviceName
                  << " (" << randomEndpoint->instanceId << ")" << std::endl;
    } else {
        std::cout << "\n未找到可用的端点" << std::endl;
    }
    
    // 服务调用测试
    std::cout << "\n=== 服务调用测试 ===" << std::endl;
    
    // 模拟服务调用
    auto result = client.callService("example-service", "GET", "/health");
    if (result.success) {
        std::cout << "服务调用成功: " << result.statusCode << " " << result.body << std::endl;
    } else {
        std::cout << "服务调用失败: " << result.error << std::endl;
    }
    
    // 带重试的服务调用
    auto retryResult = client.callServiceWithRetry("example-service", "GET", "/health", {}, "", 3);
    if (retryResult.success) {
        std::cout << "重试服务调用成功: " << retryResult.statusCode << std::endl;
    } else {
        std::cout << "重试服务调用失败: " << retryResult.error << std::endl;
    }
    
    // 负载均衡服务调用
    auto lbResult = client.callServiceWithLoadBalance("example-service", "GET", "/health");
    if (lbResult.success) {
        std::cout << "负载均衡服务调用成功: " << lbResult.statusCode << std::endl;
    } else {
        std::cout << "负载均衡服务调用失败: " << lbResult.error << std::endl;
    }
    
    // 网络质量监控
    std::cout << "\n=== 网络质量监控 ===" << std::endl;
    
    auto quality = client.getNetworkQuality();
    std::cout << "网络质量: " << static_cast<int>(quality) << std::endl;
    
    auto isWeak = client.isWeakNetwork();
    std::cout << "是否弱网: " << (isWeak ? "是" : "否") << std::endl;
    
    auto metrics = client.getNetworkMetrics();
    std::cout << "网络指标:" << std::endl;
    for (const auto& metric : metrics) {
        std::cout << "  " << metric.first << ": " << metric.second << std::endl;
    }
    
    // 缓存统计
    std::cout << "\n=== 缓存统计 ===" << std::endl;
    
    auto cacheSize = client.getCacheSize();
    std::cout << "缓存大小: " << cacheSize << std::endl;
    
    auto cacheStats = client.getCacheStats();
    std::cout << "缓存统计:" << std::endl;
    for (const auto& stat : cacheStats) {
        std::cout << "  " << stat.first << ": " << stat.second << std::endl;
    }
    
    // 客户端状态
    std::cout << "\n=== 客户端状态 ===" << std::endl;
    
    auto status = client.getStatus();
    std::cout << "客户端状态:" << std::endl;
    for (const auto& s : status) {
        std::cout << "  " << s.first << ": " << s.second << std::endl;
    }
    
    auto isHealthy = client.isHealthy();
    std::cout << "客户端健康状态: " << (isHealthy ? "健康" : "不健康") << std::endl;
    
    // 清理
    std::cout << "\n=== 清理 ===" << std::endl;
    
    if (client.unregisterService("example-service-001")) {
        std::cout << "服务注销成功" << std::endl;
    } else {
        std::cerr << "服务注销失败" << std::endl;
    }
    
    // 停止客户端
    client.stop();
    std::cout << "客户端已停止" << std::endl;
    
    return 0;
}
