#include "plum_client.hpp"
#include <algorithm>
#include <cmath>
#include <thread>
#include <chrono>

namespace plumclient {

WeakNetworkSupport::WeakNetworkSupport(const WeakNetworkConfig& config)
    : config_(config), lastCheck_(std::chrono::system_clock::now()) {
}

WeakNetworkSupport::~WeakNetworkSupport() {
    stop();
}

void WeakNetworkSupport::start() {
    if (enabled_.load()) {
        return;
    }
    
    enabled_.store(true);
    stopMonitoring_.store(false);
    
    // 启动网络质量监控线程
    monitorThread_ = std::thread(&WeakNetworkSupport::monitorNetworkQuality, this);
}

void WeakNetworkSupport::stop() {
    if (!enabled_.load()) {
        return;
    }
    
    enabled_.store(false);
    stopMonitoring_.store(true);
    
    // 等待监控线程结束
    if (monitorThread_.joinable()) {
        monitorThread_.join();
    }
}

bool WeakNetworkSupport::isEnabled() const {
    return enabled_.load();
}

NetworkQuality WeakNetworkSupport::getNetworkQuality() const {
    return networkQuality_.load();
}

bool WeakNetworkSupport::isWeakNetwork() const {
    auto quality = networkQuality_.load();
    return quality == NetworkQuality::Poor || quality == NetworkQuality::VeryPoor;
}

std::map<std::string, std::string> WeakNetworkSupport::getNetworkMetrics() const {
    std::lock_guard<std::mutex> lock(metricsMutex_);
    
    std::map<std::string, std::string> metrics;
    metrics["network_quality"] = std::to_string(static_cast<int>(networkQuality_.load()));
    metrics["avg_latency_ms"] = std::to_string(avgLatency_.load().count());
    metrics["error_rate"] = std::to_string(errorRate_.load());
    metrics["request_count"] = std::to_string(requestCount_.load());
    metrics["error_count"] = std::to_string(errorCount_.load());
    
    return metrics;
}

bool WeakNetworkSupport::shouldRetry(int attempt, int httpStatus, bool networkError) const {
    if (!enabled_.load()) {
        return false;
    }
    
    if (attempt >= config_.retryMaxAttempts) {
        return false;
    }
    
    // 网络错误总是重试
    if (networkError) {
        return true;
    }
    
    // HTTP状态码重试策略
    if (httpStatus >= 500) {
        return true; // 服务器错误
    }
    
    if (httpStatus == 429) {
        return true; // 限流
    }
    
    if (httpStatus == 408) {
        return true; // 超时
    }
    
    // 根据网络质量决定是否重试
    auto quality = networkQuality_.load();
    if (quality == NetworkQuality::Poor || quality == NetworkQuality::VeryPoor) {
        return httpStatus >= 400; // 弱网环境下，4xx错误也重试
    }
    
    return false;
}

std::chrono::milliseconds WeakNetworkSupport::getRetryDelay(int attempt) const {
    if (attempt <= 0) {
        return std::chrono::milliseconds(0);
    }
    
    // 指数退避
    auto delay = config_.retryBaseDelay * (1 << (attempt - 1));
    
    // 限制最大延迟
    if (delay > config_.retryMaxDelay) {
        delay = config_.retryMaxDelay;
    }
    
    // 根据网络质量调整延迟
    auto quality = networkQuality_.load();
    if (quality == NetworkQuality::VeryPoor) {
        delay *= 2; // 网络很差时，延迟更长
    } else if (quality == NetworkQuality::Poor) {
        delay = delay * 3 / 2; // 网络差时，延迟稍长
    }
    
    return delay;
}

int WeakNetworkSupport::getMaxRetries() const {
    return config_.retryMaxAttempts;
}

bool WeakNetworkSupport::shouldRateLimit() const {
    if (!enabled_.load()) {
        return false;
    }
    
    // 简单的令牌桶算法实现
    static std::chrono::system_clock::time_point lastRefill = std::chrono::system_clock::now();
    static int tokens = config_.rateLimitBurst;
    
    auto now = std::chrono::system_clock::now();
    auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(now - lastRefill);
    
    // 补充令牌
    int tokensToAdd = elapsed.count() * config_.rateLimitRPS / 1000;
    if (tokensToAdd > 0) {
        tokens = std::min(config_.rateLimitBurst, tokens + tokensToAdd);
        lastRefill = now;
    }
    
    // 检查是否有可用令牌
    if (tokens > 0) {
        tokens--;
        return false; // 不需要限流
    }
    
    return true; // 需要限流
}

void WeakNetworkSupport::recordRequest() {
    requestCount_.fetch_add(1);
    
    // 更新网络质量
    updateNetworkQuality();
}

void WeakNetworkSupport::monitorNetworkQuality() {
    while (!stopMonitoring_.load()) {
        updateNetworkQuality();
        
        // 每5秒检查一次
        std::this_thread::sleep_for(std::chrono::seconds(5));
    }
}

void WeakNetworkSupport::updateNetworkQuality() {
    std::lock_guard<std::mutex> lock(metricsMutex_);
    
    auto now = std::chrono::system_clock::now();
    auto elapsed = std::chrono::duration_cast<std::chrono::seconds>(now - lastCheck_);
    
    if (elapsed.count() >= 10) { // 每10秒更新一次
        // 计算错误率
        int totalRequests = requestCount_.load();
        int errors = errorCount_.load();
        
        if (totalRequests > 0) {
            errorRate_.store(static_cast<double>(errors) / totalRequests);
        }
        
        // 更新网络质量
        networkQuality_.store(determineNetworkQuality());
        
        // 重置计数器
        requestCount_.store(0);
        errorCount_.store(0);
        lastCheck_ = now;
    }
}

NetworkQuality WeakNetworkSupport::determineNetworkQuality() const {
    double errorRate = errorRate_.load();
    auto latency = avgLatency_.load();
    
    // 根据错误率和延迟判断网络质量
    if (errorRate < 0.01 && latency < std::chrono::milliseconds(100)) {
        return NetworkQuality::Excellent;
    } else if (errorRate < 0.05 && latency < std::chrono::milliseconds(200)) {
        return NetworkQuality::Good;
    } else if (errorRate < 0.1 && latency < std::chrono::milliseconds(500)) {
        return NetworkQuality::Fair;
    } else if (errorRate < 0.2 && latency < std::chrono::milliseconds(1000)) {
        return NetworkQuality::Poor;
    } else {
        return NetworkQuality::VeryPoor;
    }
}

} // namespace plumclient
