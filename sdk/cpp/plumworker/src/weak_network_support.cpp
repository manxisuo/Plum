#include "weak_network_support.hpp"
#include <iostream>
#include <random>
#include <sstream>
#include <curl/curl.h>
#include <json/json.h>

namespace plumworker {

// 网络监控器实现
NetworkMonitor::NetworkMonitor(const std::string& controllerURL) 
    : controllerURL_(controllerURL) {
    // 初始化curl
    curl_global_init(CURL_GLOBAL_DEFAULT);
}

NetworkMonitor::~NetworkMonitor() {
    stop();
    curl_global_cleanup();
}

void NetworkMonitor::start(std::chrono::seconds interval) {
    if (monitoring_.exchange(true)) {
        return; // 已经在监控
    }
    
    monitorThread_ = std::thread([this, interval]() {
        this->monitorLoop(interval);
    });
}

void NetworkMonitor::stop() {
    if (monitoring_.exchange(false)) {
        if (monitorThread_.joinable()) {
            monitorThread_.join();
        }
    }
}

void NetworkMonitor::monitorLoop(std::chrono::seconds interval) {
    while (monitoring_.load()) {
        performHealthCheck();
        std::this_thread::sleep_for(interval);
    }
}

void NetworkMonitor::performHealthCheck() {
    auto start = std::chrono::high_resolution_clock::now();
    
    CURL* curl = curl_easy_init();
    if (!curl) return;
    
    std::string url = controllerURL_ + "/healthz";
    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 5L);
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
    
    long httpCode = 0;
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, [](void*, size_t, size_t, void*) -> size_t {
        return 0; // 忽略响应内容
    });
    
    CURLcode res = curl_easy_perform(curl);
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &httpCode);
    curl_easy_cleanup(curl);
    
    auto end = std::chrono::high_resolution_clock::now();
    auto latency = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
    
    bool success = (res == CURLE_OK) && (httpCode == 200);
    bool timeout = (res == CURLE_OPERATION_TIMEDOUT);
    
    updateStats(success, latency, timeout);
}

void NetworkMonitor::updateStats(bool success, std::chrono::milliseconds latency, bool timeout) {
    std::lock_guard<std::mutex> lock(statsMutex_);
    
    stats_.sampleCount++;
    
    // 更新延迟（指数移动平均）
    if (stats_.latency.count() == 0) {
        stats_.latency = latency;
    } else {
        double alpha = 0.1; // 平滑因子
        stats_.latency = std::chrono::milliseconds(
            static_cast<long long>(stats_.latency.count() * (1 - alpha) + latency.count() * alpha)
        );
    }
    
    // 更新成功率
    if (success) {
        stats_.successRate = (stats_.successRate * (stats_.sampleCount - 1) + 1.0) / stats_.sampleCount;
    } else {
        stats_.successRate = (stats_.successRate * (stats_.sampleCount - 1)) / stats_.sampleCount;
    }
    
    // 更新错误率
    if (!success) {
        stats_.errorRate = (stats_.errorRate * (stats_.sampleCount - 1) + 1.0) / stats_.sampleCount;
    } else {
        stats_.errorRate = (stats_.errorRate * (stats_.sampleCount - 1)) / stats_.sampleCount;
    }
    
    // 更新超时率
    if (timeout) {
        stats_.timeoutRate = (stats_.timeoutRate * (stats_.sampleCount - 1) + 1.0) / stats_.sampleCount;
    } else {
        stats_.timeoutRate = (stats_.timeoutRate * (stats_.sampleCount - 1)) / stats_.sampleCount;
    }
    
    stats_.lastUpdated = std::chrono::system_clock::now();
}

NetworkQuality NetworkMonitor::getQuality() const {
    std::lock_guard<std::mutex> lock(statsMutex_);
    
    const auto& stats = stats_;
    
    // 基于延迟和成功率判断网络质量
    if (stats.latency < std::chrono::milliseconds(50) && stats.successRate > 0.99) {
        return NetworkQuality::Excellent;
    } else if (stats.latency < std::chrono::milliseconds(100) && stats.successRate > 0.95) {
        return NetworkQuality::Good;
    } else if (stats.latency < std::chrono::milliseconds(500) && stats.successRate > 0.90) {
        return NetworkQuality::Fair;
    } else if (stats.latency < std::chrono::milliseconds(2000) && stats.successRate > 0.80) {
        return NetworkQuality::Poor;
    } else {
        return NetworkQuality::VeryPoor;
    }
}

NetworkStats NetworkMonitor::getStats() const {
    std::lock_guard<std::mutex> lock(statsMutex_);
    return stats_;
}

bool NetworkMonitor::isWeakNetwork() const {
    auto quality = getQuality();
    return quality == NetworkQuality::Poor || quality == NetworkQuality::VeryPoor;
}

WeakNetworkConfig NetworkMonitor::getRecommendedConfig() const {
    auto quality = getQuality();
    
    WeakNetworkConfig config;
    
    switch (quality) {
        case NetworkQuality::Excellent:
            config.cacheTTL = std::chrono::seconds(10);
            config.retryMaxAttempts = 1;
            config.retryBaseDelay = std::chrono::milliseconds(50);
            config.retryMaxDelay = std::chrono::milliseconds(1000);
            config.requestTimeout = std::chrono::seconds(10);
            config.heartbeatInterval = std::chrono::seconds(1);
            config.enableCompression = false;
            config.batchSize = 10;
            break;
            
        case NetworkQuality::Good:
            config.cacheTTL = std::chrono::seconds(20);
            config.retryMaxAttempts = 2;
            config.retryBaseDelay = std::chrono::milliseconds(100);
            config.retryMaxDelay = std::chrono::milliseconds(2000);
            config.requestTimeout = std::chrono::seconds(15);
            config.heartbeatInterval = std::chrono::seconds(2);
            config.enableCompression = false;
            config.batchSize = 5;
            break;
            
        case NetworkQuality::Fair:
            config.cacheTTL = std::chrono::seconds(30);
            config.retryMaxAttempts = 3;
            config.retryBaseDelay = std::chrono::milliseconds(200);
            config.retryMaxDelay = std::chrono::milliseconds(3000);
            config.requestTimeout = std::chrono::seconds(20);
            config.heartbeatInterval = std::chrono::seconds(3);
            config.enableCompression = true;
            config.batchSize = 3;
            break;
            
        case NetworkQuality::Poor:
            config.cacheTTL = std::chrono::seconds(60);
            config.retryMaxAttempts = 5;
            config.retryBaseDelay = std::chrono::milliseconds(500);
            config.retryMaxDelay = std::chrono::milliseconds(10000);
            config.requestTimeout = std::chrono::seconds(30);
            config.heartbeatInterval = std::chrono::seconds(10);
            config.enableCompression = true;
            config.batchSize = 2;
            break;
            
        case NetworkQuality::VeryPoor:
            config.cacheTTL = std::chrono::seconds(120);
            config.retryMaxAttempts = 10;
            config.retryBaseDelay = std::chrono::milliseconds(1000);
            config.retryMaxDelay = std::chrono::milliseconds(30000);
            config.requestTimeout = std::chrono::seconds(60);
            config.heartbeatInterval = std::chrono::seconds(30);
            config.enableCompression = true;
            config.batchSize = 1;
            break;
    }
    
    return config;
}

// 弱网环境支持的Worker实现
WeakNetworkWorker::WeakNetworkWorker(const WorkerOptions& opt) 
    : Worker(opt), serviceCache_(std::chrono::seconds(30)) {
    networkMonitor_ = std::make_unique<NetworkMonitor>(opt.controllerBase);
}

WeakNetworkWorker::~WeakNetworkWorker() {
    stop();
}

void WeakNetworkWorker::enableWeakNetworkSupport() {
    weakNetworkEnabled_ = true;
    networkMonitor_->start(std::chrono::seconds(5));
    adaptToNetworkConditions();
}

void WeakNetworkWorker::disableWeakNetworkSupport() {
    weakNetworkEnabled_ = false;
    networkMonitor_->stop();
}

void WeakNetworkWorker::setWeakNetworkConfig(const WeakNetworkConfig& config) {
    weakNetworkConfig_ = config;
    serviceCache_ = SmartCache<std::string>(config.cacheTTL);
    
    // 更新HTTP配置
    httpConfig_.timeout = config.requestTimeout;
    httpConfig_.retryStrategy = std::make_unique<ExponentialBackoffStrategy>(
        config.retryBaseDelay, config.retryMaxDelay, config.retryMaxAttempts
    );
}

WeakNetworkConfig WeakNetworkWorker::getWeakNetworkConfig() const {
    return weakNetworkConfig_;
}

NetworkQuality WeakNetworkWorker::getNetworkQuality() const {
    return networkMonitor_->getQuality();
}

bool WeakNetworkWorker::isWeakNetwork() const {
    return networkMonitor_->isWeakNetwork();
}

NetworkStats WeakNetworkWorker::getNetworkStats() const {
    return networkMonitor_->getStats();
}

bool WeakNetworkWorker::start() {
    if (weakNetworkEnabled_) {
        adaptToNetworkConditions();
    }
    return Worker::start();
}

void WeakNetworkWorker::stop() {
    if (weakNetworkEnabled_) {
        networkMonitor_->stop();
    }
    Worker::stop();
}

void WeakNetworkWorker::adaptToNetworkConditions() {
    if (!weakNetworkEnabled_) return;
    
    auto recommendedConfig = networkMonitor_->getRecommendedConfig();
    setWeakNetworkConfig(recommendedConfig);
}

bool WeakNetworkWorker::doRegisterWithRetry() {
    if (!weakNetworkEnabled_) {
        return doRegister();
    }
    
    auto& retryStrategy = *httpConfig_.retryStrategy;
    int maxAttempts = retryStrategy.getMaxAttempts();
    
    for (int attempt = 0; attempt <= maxAttempts; attempt++) {
        bool success = doRegister();
        if (success) {
            return true;
        }
        
        if (attempt == maxAttempts) {
            break;
        }
        
        auto delay = retryStrategy.getDelay(attempt);
        std::this_thread::sleep_for(delay);
    }
    
    return false;
}

bool WeakNetworkWorker::doHeartbeatWithRetry() {
    if (!weakNetworkEnabled_) {
        return doHeartbeat();
    }
    
    auto& retryStrategy = *httpConfig_.retryStrategy;
    int maxAttempts = retryStrategy.getMaxAttempts();
    
    for (int attempt = 0; attempt <= maxAttempts; attempt++) {
        bool success = doHeartbeat();
        if (success) {
            return true;
        }
        
        if (attempt == maxAttempts) {
            break;
        }
        
        auto delay = retryStrategy.getDelay(attempt);
        std::this_thread::sleep_for(delay);
    }
    
    return false;
}

} // namespace plumworker
