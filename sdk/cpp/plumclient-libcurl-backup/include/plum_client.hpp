#pragma once

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <functional>
#include <chrono>
#include <mutex>
#include <atomic>
#include <thread>
#include <optional>
#include <nlohmann/json.hpp>

namespace plumclient {

// 前向声明
class ServiceClient;
class DiscoveryClient;
class WeakNetworkSupport;
class Cache;

// 服务端点信息
struct Endpoint {
    std::string serviceName;
    std::string instanceId;
    std::string nodeId;
    std::string ip;
    int port;
    std::string protocol;
    std::string version;
    std::map<std::string, std::string> labels;
    bool healthy;
    std::chrono::system_clock::time_point lastSeen;
};

// 服务注册请求
struct ServiceRegistration {
    std::string instanceId;
    std::string serviceName;
    std::string nodeId;
    std::string ip;
    int port;
    std::string protocol;
    std::string version;
    std::map<std::string, std::string> labels;
};

// 服务心跳请求
struct ServiceHeartbeat {
    std::string instanceId;
    std::vector<Endpoint> endpoints;
};

// 服务发现请求
struct DiscoveryRequest {
    std::string service;
    std::string version;
    std::string protocol;
};

// 服务调用结果
struct ServiceCallResult {
    int statusCode;
    std::string body;
    std::chrono::milliseconds latency;
    bool success;
    std::string error;
};

// 网络质量
enum class NetworkQuality {
    Excellent,
    Good,
    Fair,
    Poor,
    VeryPoor
};

// 弱网环境配置
struct WeakNetworkConfig {
    // 缓存配置
    std::chrono::seconds cacheTTL{30};
    int cacheMaxSize{1000};
    
    // 重试配置
    int retryMaxAttempts{3};
    std::chrono::milliseconds retryBaseDelay{100};
    std::chrono::milliseconds retryMaxDelay{5000};
    
    // 超时配置
    std::chrono::seconds requestTimeout{30};
    std::chrono::seconds connectTimeout{10};
    
    // 限流配置
    int rateLimitRPS{1000};
    int rateLimitBurst{2000};
    
    // 健康检查配置
    std::chrono::seconds healthCheckInterval{30};
    std::chrono::seconds healthCheckTimeout{5};
};

// 主客户端类
class PlumClient {
public:
    // 构造函数
    explicit PlumClient(const std::string& controllerUrl);
    explicit PlumClient(const std::string& controllerUrl, const WeakNetworkConfig& config);
    
    // 析构函数
    ~PlumClient();
    
    // 禁用拷贝构造和赋值
    PlumClient(const PlumClient&) = delete;
    PlumClient& operator=(const PlumClient&) = delete;
    
    // 启动和停止
    bool start();
    void stop();
    bool isRunning() const;
    
    // 服务注册
    bool registerService(const ServiceRegistration& registration);
    bool heartbeatService(const ServiceHeartbeat& heartbeat);
    bool unregisterService(const std::string& instanceId);
    
    // 服务发现
    std::vector<Endpoint> discoverService(const DiscoveryRequest& request);
    std::vector<Endpoint> discoverService(const std::string& service, 
                                        const std::string& version = "", 
                                        const std::string& protocol = "");
    
    // 随机服务发现
    std::optional<Endpoint> discoverRandomService(const DiscoveryRequest& request);
    std::optional<Endpoint> discoverRandomService(const std::string& service,
                                                 const std::string& version = "",
                                                 const std::string& protocol = "");
    
    // 服务调用
    ServiceCallResult callService(const std::string& service,
                                const std::string& method,
                                const std::string& path,
                                const std::map<std::string, std::string>& headers = {},
                                const std::string& body = "");
    
    // 带重试的服务调用
    ServiceCallResult callServiceWithRetry(const std::string& service,
                                         const std::string& method,
                                         const std::string& path,
                                         const std::map<std::string, std::string>& headers = {},
                                         const std::string& body = "",
                                         int maxRetries = 3);
    
    // 负载均衡服务调用
    ServiceCallResult callServiceWithLoadBalance(const std::string& service,
                                               const std::string& method,
                                               const std::string& path,
                                               const std::map<std::string, std::string>& headers = {},
                                               const std::string& body = "");
    
    // 弱网环境支持
    void enableWeakNetworkSupport();
    void disableWeakNetworkSupport();
    bool isWeakNetworkSupportEnabled() const;
    
    // 网络质量监控
    NetworkQuality getNetworkQuality() const;
    bool isWeakNetwork() const;
    std::map<std::string, std::string> getNetworkMetrics() const;
    
    // 缓存管理
    void clearCache();
    size_t getCacheSize() const;
    std::map<std::string, std::string> getCacheStats() const;
    
    // 配置管理
    void updateConfig(const WeakNetworkConfig& config);
    WeakNetworkConfig getConfig() const;
    
    // 状态查询
    std::map<std::string, std::string> getStatus() const;
    bool isHealthy() const;

private:
    std::string controllerUrl_;
    WeakNetworkConfig config_;
    
    std::unique_ptr<ServiceClient> serviceClient_;
    std::unique_ptr<DiscoveryClient> discoveryClient_;
    std::unique_ptr<WeakNetworkSupport> weakNetworkSupport_;
    std::unique_ptr<Cache> cache_;
    
    std::atomic<bool> running_{false};
    std::atomic<bool> weakNetworkEnabled_{false};
    mutable std::mutex mutex_;
    
    // 内部方法
    void initializeComponents();
    void startBackgroundTasks();
    void stopBackgroundTasks();
    ServiceCallResult makeHttpRequest(const std::string& method,
                                    const std::string& url,
                                    const std::map<std::string, std::string>& headers,
                                    const std::string& body);
};

// 服务客户端类
class ServiceClient {
public:
    explicit ServiceClient(const std::string& controllerUrl, 
                          std::shared_ptr<WeakNetworkSupport> weakNetworkSupport,
                          std::shared_ptr<Cache> cache);
    
    bool registerService(const ServiceRegistration& registration);
    bool heartbeatService(const ServiceHeartbeat& heartbeat);
    bool unregisterService(const std::string& instanceId);
    
private:
    std::string controllerUrl_;
    std::shared_ptr<WeakNetworkSupport> weakNetworkSupport_;
    std::shared_ptr<Cache> cache_;
    
    bool makeRequest(const std::string& method, 
                    const std::string& path, 
                    const std::string& body = "",
                    const std::map<std::string, std::string>& headers = {});
};

// 服务发现客户端类
class DiscoveryClient {
public:
    explicit DiscoveryClient(const std::string& controllerUrl,
                           std::shared_ptr<WeakNetworkSupport> weakNetworkSupport,
                           std::shared_ptr<Cache> cache);
    
    std::vector<Endpoint> discoverService(const DiscoveryRequest& request);
    std::optional<Endpoint> discoverRandomService(const DiscoveryRequest& request);
    
private:
    std::string controllerUrl_;
    std::shared_ptr<WeakNetworkSupport> weakNetworkSupport_;
    std::shared_ptr<Cache> cache_;
    
    std::vector<Endpoint> makeDiscoveryRequest(const std::string& path);
    std::optional<Endpoint> makeRandomDiscoveryRequest(const std::string& path);
    std::vector<Endpoint> parseEndpointsFromJson(const nlohmann::json& root);
    std::optional<Endpoint> parseEndpointFromJson(const nlohmann::json& root);
};

// 弱网环境支持类
class WeakNetworkSupport {
public:
    explicit WeakNetworkSupport(const WeakNetworkConfig& config);
    ~WeakNetworkSupport();
    
    void start();
    void stop();
    bool isEnabled() const;
    
    NetworkQuality getNetworkQuality() const;
    bool isWeakNetwork() const;
    std::map<std::string, std::string> getNetworkMetrics() const;
    
    bool shouldRetry(int attempt, int httpStatus, bool networkError) const;
    std::chrono::milliseconds getRetryDelay(int attempt) const;
    int getMaxRetries() const;
    
    bool shouldRateLimit() const;
    void recordRequest();
    
private:
    WeakNetworkConfig config_;
    std::atomic<bool> enabled_{false};
    std::atomic<NetworkQuality> networkQuality_{NetworkQuality::Good};
    
    // 网络质量监控
    std::atomic<std::chrono::milliseconds> avgLatency_{std::chrono::milliseconds(0)};
    std::atomic<double> errorRate_{0.0};
    std::atomic<int> requestCount_{0};
    std::atomic<int> errorCount_{0};
    std::chrono::system_clock::time_point lastCheck_;
    
    mutable std::mutex metricsMutex_;
    std::thread monitorThread_;
    std::atomic<bool> stopMonitoring_{false};
    
    void monitorNetworkQuality();
    void updateNetworkQuality();
    NetworkQuality determineNetworkQuality() const;
};

// 缓存类
class Cache {
public:
    explicit Cache(const WeakNetworkConfig& config);
    ~Cache();
    
    void set(const std::string& key, const std::string& value, 
             std::chrono::seconds ttl = std::chrono::seconds(0));
    std::optional<std::string> get(const std::string& key);
    void remove(const std::string& key);
    void clear();
    
    size_t size() const;
    std::map<std::string, std::string> getStats() const;
    
private:
    struct CacheEntry {
        std::string value;
        std::chrono::system_clock::time_point expiresAt;
        std::chrono::system_clock::time_point createdAt;
    };
    
    WeakNetworkConfig config_;
    std::map<std::string, CacheEntry> entries_;
    mutable std::mutex mutex_;
    
    std::atomic<int> hitCount_{0};
    std::atomic<int> missCount_{0};
    std::thread cleanupThread_;
    std::atomic<bool> stopCleanup_{false};
    
    void cleanup();
    bool isExpired(const CacheEntry& entry) const;
};

} // namespace plumclient
