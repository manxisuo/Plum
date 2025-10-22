#include "plum_client.hpp"
#include <curl/curl.h>
#include <nlohmann/json.hpp>
#include <sstream>
#include <random>
#include <algorithm>
#include <iostream>

namespace plumclient {

// 静态回调函数
static size_t WriteCallback(void* contents, size_t size, size_t nmemb, void* userp) {
    size_t totalSize = size * nmemb;
    std::string* str = static_cast<std::string*>(userp);
    str->append(static_cast<char*>(contents), totalSize);
    return totalSize;
}

// PlumClient 实现
PlumClient::PlumClient(const std::string& controllerUrl) 
    : controllerUrl_(controllerUrl) {
    initializeComponents();
}

PlumClient::PlumClient(const std::string& controllerUrl, const WeakNetworkConfig& config)
    : controllerUrl_(controllerUrl), config_(config) {
    initializeComponents();
}

PlumClient::~PlumClient() {
    stop();
}

void PlumClient::initializeComponents() {
    // 初始化curl
    curl_global_init(CURL_GLOBAL_DEFAULT);
    
    // 创建组件
    weakNetworkSupport_ = std::make_unique<WeakNetworkSupport>(config_);
    cache_ = std::make_unique<Cache>(config_);
    
    serviceClient_ = std::make_unique<ServiceClient>(controllerUrl_, 
        std::shared_ptr<WeakNetworkSupport>(weakNetworkSupport_.get(), [](WeakNetworkSupport*){}),
        std::shared_ptr<Cache>(cache_.get(), [](Cache*){}));
    discoveryClient_ = std::make_unique<DiscoveryClient>(controllerUrl_, 
        std::shared_ptr<WeakNetworkSupport>(weakNetworkSupport_.get(), [](WeakNetworkSupport*){}),
        std::shared_ptr<Cache>(cache_.get(), [](Cache*){}));
}

bool PlumClient::start() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (running_.load()) {
        return true;
    }
    
    // 启动弱网环境支持
    if (weakNetworkSupport_) {
        weakNetworkSupport_->start();
    }
    
    // 启动后台任务
    startBackgroundTasks();
    
    running_.store(true);
    return true;
}

void PlumClient::stop() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (!running_.load()) {
        return;
    }
    
    // 停止后台任务
    stopBackgroundTasks();
    
    // 停止弱网环境支持
    if (weakNetworkSupport_) {
        weakNetworkSupport_->stop();
    }
    
    running_.store(false);
    
    // 清理curl
    curl_global_cleanup();
}

bool PlumClient::isRunning() const {
    return running_.load();
}

bool PlumClient::registerService(const ServiceRegistration& registration) {
    if (!serviceClient_) {
        return false;
    }
    return serviceClient_->registerService(registration);
}

bool PlumClient::heartbeatService(const ServiceHeartbeat& heartbeat) {
    if (!serviceClient_) {
        return false;
    }
    return serviceClient_->heartbeatService(heartbeat);
}

bool PlumClient::unregisterService(const std::string& instanceId) {
    if (!serviceClient_) {
        return false;
    }
    return serviceClient_->unregisterService(instanceId);
}

std::vector<Endpoint> PlumClient::discoverService(const DiscoveryRequest& request) {
    if (!discoveryClient_) {
        return {};
    }
    return discoveryClient_->discoverService(request);
}

std::vector<Endpoint> PlumClient::discoverService(const std::string& service, 
                                                 const std::string& version, 
                                                 const std::string& protocol) {
    DiscoveryRequest request;
    request.service = service;
    request.version = version;
    request.protocol = protocol;
    return discoverService(request);
}

std::optional<Endpoint> PlumClient::discoverRandomService(const DiscoveryRequest& request) {
    if (!discoveryClient_) {
        return std::nullopt;
    }
    return discoveryClient_->discoverRandomService(request);
}

std::optional<Endpoint> PlumClient::discoverRandomService(const std::string& service,
                                                         const std::string& version,
                                                         const std::string& protocol) {
    DiscoveryRequest request;
    request.service = service;
    request.version = version;
    request.protocol = protocol;
    return discoverRandomService(request);
}

ServiceCallResult PlumClient::callService(const std::string& service,
                                        const std::string& method,
                                        const std::string& path,
                                        const std::map<std::string, std::string>& headers,
                                        const std::string& body) {
    // 先发现服务
    auto endpoints = discoverService(service);
    if (endpoints.empty()) {
        return {0, "", std::chrono::milliseconds(0), false, "No endpoints found"};
    }
    
    // 随机选择一个端点
    static std::random_device rd;
    static std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, endpoints.size() - 1);
    const auto& endpoint = endpoints[dis(gen)];
    
    // 构建URL
    std::string url = endpoint.protocol + "://" + endpoint.ip + ":" + std::to_string(endpoint.port) + path;
    
    // 执行HTTP请求
    return makeHttpRequest(method, url, headers, body);
}

ServiceCallResult PlumClient::callServiceWithRetry(const std::string& service,
                                                  const std::string& method,
                                                  const std::string& path,
                                                  const std::map<std::string, std::string>& headers,
                                                  const std::string& body,
                                                  int maxRetries) {
    int attempts = 0;
    while (attempts <= maxRetries) {
        auto result = callService(service, method, path, headers, body);
        
        if (result.success) {
            return result;
        }
        
        // 检查是否应该重试
        if (weakNetworkSupport_ && 
            !weakNetworkSupport_->shouldRetry(attempts, result.statusCode, !result.success)) {
            break;
        }
        
        if (attempts < maxRetries) {
            auto delay = weakNetworkSupport_ ? 
                weakNetworkSupport_->getRetryDelay(attempts) : 
                std::chrono::milliseconds(1000);
            std::this_thread::sleep_for(delay);
        }
        
        attempts++;
    }
    
    return {0, "", std::chrono::milliseconds(0), false, "Max retries exceeded"};
}

ServiceCallResult PlumClient::callServiceWithLoadBalance(const std::string& service,
                                                        const std::string& method,
                                                        const std::string& path,
                                                        const std::map<std::string, std::string>& headers,
                                                        const std::string& body) {
    // 实现负载均衡逻辑
    auto endpoints = discoverService(service);
    if (endpoints.empty()) {
        return {0, "", std::chrono::milliseconds(0), false, "No endpoints found"};
    }
    
    // 过滤健康的端点
    std::vector<Endpoint> healthyEndpoints;
    for (const auto& endpoint : endpoints) {
        if (endpoint.healthy) {
            healthyEndpoints.push_back(endpoint);
        }
    }
    
    if (healthyEndpoints.empty()) {
        return {0, "", std::chrono::milliseconds(0), false, "No healthy endpoints found"};
    }
    
    // 随机选择一个健康端点
    static std::random_device rd;
    static std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, healthyEndpoints.size() - 1);
    const auto& endpoint = healthyEndpoints[dis(gen)];
    
    // 构建URL并执行请求
    std::string url = endpoint.protocol + "://" + endpoint.ip + ":" + std::to_string(endpoint.port) + path;
    return makeHttpRequest(method, url, headers, body);
}

void PlumClient::enableWeakNetworkSupport() {
    weakNetworkEnabled_.store(true);
    if (weakNetworkSupport_) {
        weakNetworkSupport_->start();
    }
}

void PlumClient::disableWeakNetworkSupport() {
    weakNetworkEnabled_.store(false);
    if (weakNetworkSupport_) {
        weakNetworkSupport_->stop();
    }
}

bool PlumClient::isWeakNetworkSupportEnabled() const {
    return weakNetworkEnabled_.load();
}

NetworkQuality PlumClient::getNetworkQuality() const {
    if (!weakNetworkSupport_) {
        return NetworkQuality::Good;
    }
    return weakNetworkSupport_->getNetworkQuality();
}

bool PlumClient::isWeakNetwork() const {
    if (!weakNetworkSupport_) {
        return false;
    }
    return weakNetworkSupport_->isWeakNetwork();
}

std::map<std::string, std::string> PlumClient::getNetworkMetrics() const {
    if (!weakNetworkSupport_) {
        return {};
    }
    return weakNetworkSupport_->getNetworkMetrics();
}

void PlumClient::clearCache() {
    if (cache_) {
        cache_->clear();
    }
}

size_t PlumClient::getCacheSize() const {
    if (!cache_) {
        return 0;
    }
    return cache_->size();
}

std::map<std::string, std::string> PlumClient::getCacheStats() const {
    if (!cache_) {
        return {};
    }
    return cache_->getStats();
}

void PlumClient::updateConfig(const WeakNetworkConfig& config) {
    std::lock_guard<std::mutex> lock(mutex_);
    config_ = config;
    
    // 重新初始化组件
    initializeComponents();
}

WeakNetworkConfig PlumClient::getConfig() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return config_;
}

std::map<std::string, std::string> PlumClient::getStatus() const {
    std::map<std::string, std::string> status;
    status["running"] = running_.load() ? "true" : "false";
    status["weak_network_enabled"] = weakNetworkEnabled_.load() ? "true" : "false";
    status["cache_size"] = std::to_string(getCacheSize());
    
    if (weakNetworkSupport_) {
        auto metrics = weakNetworkSupport_->getNetworkMetrics();
        status.insert(metrics.begin(), metrics.end());
    }
    
    return status;
}

bool PlumClient::isHealthy() const {
    if (!running_.load()) {
        return false;
    }
    
    if (weakNetworkSupport_) {
        return !weakNetworkSupport_->isWeakNetwork();
    }
    
    return true;
}

void PlumClient::startBackgroundTasks() {
    // 启动后台任务，如缓存清理、健康检查等
}

void PlumClient::stopBackgroundTasks() {
    // 停止后台任务
}

ServiceCallResult PlumClient::makeHttpRequest(const std::string& method,
                                            const std::string& url,
                                            const std::map<std::string, std::string>& headers,
                                            const std::string& body) {
    ServiceCallResult result;
    auto start = std::chrono::high_resolution_clock::now();
    
    CURL* curl = curl_easy_init();
    if (!curl) {
        result.error = "Failed to initialize curl";
        return result;
    }
    
    std::string responseBody;
    long httpCode = 0;
    
    // 设置URL
    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    
    // 设置HTTP方法
    if (method == "POST") {
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body.c_str());
    } else if (method == "PUT") {
        curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "PUT");
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body.c_str());
    } else if (method == "DELETE") {
        curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "DELETE");
    }
    
    // 设置超时
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, static_cast<long>(config_.requestTimeout.count()));
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, static_cast<long>(config_.connectTimeout.count()));
    
    // 设置回调函数
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &responseBody);
    
    // 设置头部
    struct curl_slist* headerList = nullptr;
    for (const auto& header : headers) {
        std::string headerStr = header.first + ": " + header.second;
        headerList = curl_slist_append(headerList, headerStr.c_str());
    }
    if (headerList) {
        curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headerList);
    }
    
    // 执行请求
    CURLcode res = curl_easy_perform(curl);
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &httpCode);
    
    auto end = std::chrono::high_resolution_clock::now();
    result.latency = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
    
    // 清理
    if (headerList) {
        curl_slist_free_all(headerList);
    }
    curl_easy_cleanup(curl);
    
    // 设置结果
    result.statusCode = static_cast<int>(httpCode);
    result.body = responseBody;
    result.success = (res == CURLE_OK) && (httpCode >= 200 && httpCode < 300);
    
    if (!result.success) {
        result.error = "HTTP request failed: " + std::to_string(httpCode);
    }
    
    return result;
}

} // namespace plumclient
