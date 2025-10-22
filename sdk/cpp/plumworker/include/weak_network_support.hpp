#pragma once

#include <chrono>
#include <map>
#include <memory>
#include <mutex>
#include <string>
#include <vector>
#include <atomic>
#include <thread>
#include <functional>

namespace plumworker {

// 网络质量枚举
enum class NetworkQuality {
    Excellent,
    Good,
    Fair,
    Poor,
    VeryPoor
};

// 网络统计信息
struct NetworkStats {
    std::chrono::milliseconds latency{0};
    double successRate{1.0};
    double errorRate{0.0};
    double timeoutRate{0.0};
    std::chrono::system_clock::time_point lastUpdated;
    int sampleCount{0};
};

// 弱网环境配置
struct WeakNetworkConfig {
    std::chrono::seconds cacheTTL{30};
    int retryMaxAttempts{3};
    std::chrono::milliseconds retryBaseDelay{100};
    std::chrono::milliseconds retryMaxDelay{5000};
    std::chrono::seconds requestTimeout{30};
    std::chrono::seconds heartbeatInterval{5};
    bool enableCompression{false};
    int batchSize{1};
};

// 智能缓存条目
template<typename T>
struct CacheEntry {
    T data;
    std::chrono::system_clock::time_point expiresAt;
    std::chrono::system_clock::time_point createdAt;
};

// 智能缓存
template<typename T>
class SmartCache {
public:
    explicit SmartCache(std::chrono::seconds defaultTTL) 
        : defaultTTL_(defaultTTL) {}
    
    void set(const std::string& key, const T& data, 
             std::chrono::seconds customTTL = std::chrono::seconds(0)) {
        std::lock_guard<std::mutex> lock(mutex_);
        auto ttl = customTTL.count() > 0 ? customTTL : defaultTTL_;
        auto now = std::chrono::system_clock::now();
        entries_[key] = CacheEntry<T>{
            data,
            now + ttl,
            now
        };
    }
    
    bool get(const std::string& key, T& data) {
        std::lock_guard<std::mutex> lock(mutex_);
        auto it = entries_.find(key);
        if (it == entries_.end()) {
            return false;
        }
        
        auto now = std::chrono::system_clock::now();
        if (now > it->second.expiresAt) {
            entries_.erase(it);
            return false;
        }
        
        data = it->second.data;
        return true;
    }
    
    void clear() {
        std::lock_guard<std::mutex> lock(mutex_);
        entries_.clear();
    }
    
    size_t size() const {
        std::lock_guard<std::mutex> lock(mutex_);
        return entries_.size();
    }
    
    void cleanup() {
        std::lock_guard<std::mutex> lock(mutex_);
        auto now = std::chrono::system_clock::now();
        for (auto it = entries_.begin(); it != entries_.end();) {
            if (now > it->second.expiresAt) {
                it = entries_.erase(it);
            } else {
                ++it;
            }
        }
    }

private:
    std::map<std::string, CacheEntry<T>> entries_;
    mutable std::mutex mutex_;
    std::chrono::seconds defaultTTL_;
};

// 重试策略接口
class RetryStrategy {
public:
    virtual ~RetryStrategy() = default;
    virtual bool shouldRetry(int attempt, int httpStatus, bool networkError) = 0;
    virtual std::chrono::milliseconds getDelay(int attempt) = 0;
    virtual int getMaxAttempts() = 0;
};

// 指数退避重试策略
class ExponentialBackoffStrategy : public RetryStrategy {
public:
    ExponentialBackoffStrategy(std::chrono::milliseconds baseDelay,
                              std::chrono::milliseconds maxDelay,
                              int maxAttempts)
        : baseDelay_(baseDelay), maxDelay_(maxDelay), maxAttempts_(maxAttempts) {}
    
    bool shouldRetry(int attempt, int httpStatus, bool networkError) override {
        if (attempt >= maxAttempts_) return false;
        
        // 网络错误总是重试
        if (networkError) return true;
        
        // HTTP状态码重试策略
        return httpStatus >= 500 || httpStatus == 429 || httpStatus == 408;
    }
    
    std::chrono::milliseconds getDelay(int attempt) override {
        auto delay = std::chrono::milliseconds(
            static_cast<long long>(baseDelay_.count() * std::pow(2.0, attempt))
        );
        
        if (delay > maxDelay_) {
            delay = maxDelay_;
        }
        
        // 添加抖动
        auto jitter = std::rand() % (delay.count() / 10);
        return delay + std::chrono::milliseconds(jitter);
    }
    
    int getMaxAttempts() override {
        return maxAttempts_;
    }

private:
    std::chrono::milliseconds baseDelay_;
    std::chrono::milliseconds maxDelay_;
    int maxAttempts_;
};

// 网络监控器
class NetworkMonitor {
public:
    explicit NetworkMonitor(const std::string& controllerURL);
    ~NetworkMonitor();
    
    void start(std::chrono::seconds interval);
    void stop();
    
    NetworkQuality getQuality() const;
    NetworkStats getStats() const;
    bool isWeakNetwork() const;
    WeakNetworkConfig getRecommendedConfig() const;

private:
    std::string controllerURL_;
    mutable std::mutex statsMutex_;
    NetworkStats stats_;
    std::atomic<bool> monitoring_{false};
    std::thread monitorThread_;
    
    void monitorLoop(std::chrono::seconds interval);
    void performHealthCheck();
    void updateStats(bool success, std::chrono::milliseconds latency, bool timeout);
};

// HTTP客户端配置
struct HttpClientConfig {
    std::chrono::seconds timeout{30};
    std::unique_ptr<RetryStrategy> retryStrategy;
    bool enableCompression{false};
};

// 弱网环境支持的Worker
class WeakNetworkWorker : public Worker {
public:
    explicit WeakNetworkWorker(const WorkerOptions& opt);
    ~WeakNetworkWorker();
    
    void enableWeakNetworkSupport();
    void disableWeakNetworkSupport();
    void setWeakNetworkConfig(const WeakNetworkConfig& config);
    WeakNetworkConfig getWeakNetworkConfig() const;
    
    NetworkQuality getNetworkQuality() const;
    bool isWeakNetwork() const;
    NetworkStats getNetworkStats() const;
    
    // 重写基类方法以支持弱网环境
    bool start() override;
    void stop() override;

private:
    std::unique_ptr<NetworkMonitor> networkMonitor_;
    WeakNetworkConfig weakNetworkConfig_;
    std::atomic<bool> weakNetworkEnabled_{false};
    SmartCache<std::string> serviceCache_;
    HttpClientConfig httpConfig_;
    
    void adaptToNetworkConditions();
    bool doRegisterWithRetry();
    bool doHeartbeatWithRetry();
};

} // namespace plumworker
