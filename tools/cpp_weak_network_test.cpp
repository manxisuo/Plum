#include <iostream>
#include <thread>
#include <chrono>
#include <vector>
#include <atomic>
#include <mutex>
#include <random>
#include <sstream>
#include <curl/curl.h>
#include <json/json.h>

// 简化的网络质量枚举
enum class NetworkQuality {
    Excellent, Good, Fair, Poor, VeryPoor
};

// 简化的网络统计
struct NetworkStats {
    std::chrono::milliseconds latency{0};
    double successRate{1.0};
    double errorRate{0.0};
    int sampleCount{0};
    std::chrono::system_clock::time_point lastUpdated;
};

// 简化的缓存
class SimpleCache {
public:
    void set(const std::string& key, const std::string& value, 
             std::chrono::seconds ttl = std::chrono::seconds(30)) {
        std::lock_guard<std::mutex> lock(mutex_);
        auto now = std::chrono::system_clock::now();
        entries_[key] = {value, now + ttl, now};
    }
    
    bool get(const std::string& key, std::string& value) {
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
        
        value = it->second.data;
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

private:
    struct CacheEntry {
        std::string data;
        std::chrono::system_clock::time_point expiresAt;
        std::chrono::system_clock::time_point createdAt;
    };
    
    std::map<std::string, CacheEntry> entries_;
    mutable std::mutex mutex_;
};

// 简化的重试策略
class RetryStrategy {
public:
    virtual ~RetryStrategy() = default;
    virtual bool shouldRetry(int attempt, int httpStatus, bool networkError) = 0;
    virtual std::chrono::milliseconds getDelay(int attempt) = 0;
    virtual int getMaxAttempts() = 0;
};

class ExponentialBackoffStrategy : public RetryStrategy {
public:
    ExponentialBackoffStrategy(std::chrono::milliseconds baseDelay,
                              std::chrono::milliseconds maxDelay,
                              int maxAttempts)
        : baseDelay_(baseDelay), maxDelay_(maxDelay), maxAttempts_(maxAttempts) {}
    
    bool shouldRetry(int attempt, int httpStatus, bool networkError) override {
        if (attempt >= maxAttempts_) return false;
        if (networkError) return true;
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
        static std::random_device rd;
        static std::mt19937 gen(rd());
        std::uniform_int_distribution<> dis(0, delay.count() / 10);
        auto jitter = dis(gen);
        
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

// 简化的HTTP客户端
class HttpClient {
public:
    HttpClient(const std::string& baseURL, std::unique_ptr<RetryStrategy> strategy)
        : baseURL_(baseURL), retryStrategy_(std::move(strategy)) {
        curl_global_init(CURL_GLOBAL_DEFAULT);
    }
    
    ~HttpClient() {
        curl_global_cleanup();
    }
    
    bool get(const std::string& path, std::string& response, int& httpStatus) {
        auto& strategy = *retryStrategy_;
        int maxAttempts = strategy.getMaxAttempts();
        
        for (int attempt = 0; attempt <= maxAttempts; attempt++) {
            bool success = performRequest(path, response, httpStatus);
            if (success) {
                return true;
            }
            
            if (attempt == maxAttempts) {
                break;
            }
            
            bool networkError = (httpStatus == 0);
            if (!strategy.shouldRetry(attempt, httpStatus, networkError)) {
                break;
            }
            
            auto delay = strategy.getDelay(attempt);
            std::this_thread::sleep_for(delay);
        }
        
        return false;
    }

private:
    bool performRequest(const std::string& path, std::string& response, int& httpStatus) {
        CURL* curl = curl_easy_init();
        if (!curl) return false;
        
        std::string url = baseURL_ + path;
        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);
        curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
        curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
        
        response.clear();
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, [](void* contents, size_t size, size_t nmemb, void* userp) -> size_t {
            size_t totalSize = size * nmemb;
            std::string* str = static_cast<std::string*>(userp);
            str->append(static_cast<char*>(contents), totalSize);
            return totalSize;
        });
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);
        
        CURLcode res = curl_easy_perform(curl);
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &httpStatus);
        curl_easy_cleanup(curl);
        
        return (res == CURLE_OK) && (httpStatus == 200);
    }
    
    std::string baseURL_;
    std::unique_ptr<RetryStrategy> retryStrategy_;
};

// 简化的网络监控器
class NetworkMonitor {
public:
    explicit NetworkMonitor(const std::string& controllerURL) 
        : controllerURL_(controllerURL) {
        curl_global_init(CURL_GLOBAL_DEFAULT);
    }
    
    ~NetworkMonitor() {
        curl_global_cleanup();
    }
    
    void start(std::chrono::seconds interval) {
        if (monitoring_.exchange(true)) return;
        
        monitorThread_ = std::thread([this, interval]() {
            while (monitoring_.load()) {
                performHealthCheck();
                std::this_thread::sleep_for(interval);
            }
        });
    }
    
    void stop() {
        if (monitoring_.exchange(false)) {
            if (monitorThread_.joinable()) {
                monitorThread_.join();
            }
        }
    }
    
    NetworkQuality getQuality() const {
        std::lock_guard<std::mutex> lock(statsMutex_);
        
        if (stats_.latency < std::chrono::milliseconds(50) && stats_.successRate > 0.99) {
            return NetworkQuality::Excellent;
        } else if (stats_.latency < std::chrono::milliseconds(100) && stats_.successRate > 0.95) {
            return NetworkQuality::Good;
        } else if (stats_.latency < std::chrono::milliseconds(500) && stats_.successRate > 0.90) {
            return NetworkQuality::Fair;
        } else if (stats_.latency < std::chrono::milliseconds(2000) && stats_.successRate > 0.80) {
            return NetworkQuality::Poor;
        } else {
            return NetworkQuality::VeryPoor;
        }
    }
    
    NetworkStats getStats() const {
        std::lock_guard<std::mutex> lock(statsMutex_);
        return stats_;
    }
    
    bool isWeakNetwork() const {
        auto quality = getQuality();
        return quality == NetworkQuality::Poor || quality == NetworkQuality::VeryPoor;
    }

private:
    void performHealthCheck() {
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
            return 0;
        });
        
        CURLcode res = curl_easy_perform(curl);
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &httpCode);
        curl_easy_cleanup(curl);
        
        auto end = std::chrono::high_resolution_clock::now();
        auto latency = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
        
        bool success = (res == CURLE_OK) && (httpCode == 200);
        updateStats(success, latency);
    }
    
    void updateStats(bool success, std::chrono::milliseconds latency) {
        std::lock_guard<std::mutex> lock(statsMutex_);
        
        stats_.sampleCount++;
        
        if (stats_.latency.count() == 0) {
            stats_.latency = latency;
        } else {
            double alpha = 0.1;
            stats_.latency = std::chrono::milliseconds(
                static_cast<long long>(stats_.latency.count() * (1 - alpha) + latency.count() * alpha)
            );
        }
        
        if (success) {
            stats_.successRate = (stats_.successRate * (stats_.sampleCount - 1) + 1.0) / stats_.sampleCount;
        } else {
            stats_.successRate = (stats_.successRate * (stats_.sampleCount - 1)) / stats_.sampleCount;
        }
        
        if (!success) {
            stats_.errorRate = (stats_.errorRate * (stats_.sampleCount - 1) + 1.0) / stats_.sampleCount;
        } else {
            stats_.errorRate = (stats_.errorRate * (stats_.sampleCount - 1)) / stats_.sampleCount;
        }
        
        stats_.lastUpdated = std::chrono::system_clock::now();
    }
    
    std::string controllerURL_;
    mutable std::mutex statsMutex_;
    NetworkStats stats_;
    std::atomic<bool> monitoring_{false};
    std::thread monitorThread_;
};

// 测试结果
struct TestResult {
    int clientId;
    int successCount{0};
    int errorCount{0};
    std::chrono::milliseconds avgLatency{0};
    std::chrono::milliseconds maxLatency{0};
    std::chrono::milliseconds minLatency{std::chrono::hours(1)};
    NetworkQuality networkQuality{NetworkQuality::Excellent};
    bool isWeakNetwork{false};
    std::vector<std::string> errors;
};

// 弱网环境测试器
class WeakNetworkTester {
public:
    WeakNetworkTester(const std::string& controllerURL, int clientCount) 
        : controllerURL_(controllerURL), clientCount_(clientCount) {
        
        // 创建重试策略
        auto strategy = std::make_unique<ExponentialBackoffStrategy>(
            std::chrono::milliseconds(100),
            std::chrono::milliseconds(5000),
            3
        );
        
        // 创建HTTP客户端
        httpClient_ = std::make_unique<HttpClient>(controllerURL, std::move(strategy));
        
        // 创建网络监控器
        networkMonitor_ = std::make_unique<NetworkMonitor>(controllerURL);
    }
    
    ~WeakNetworkTester() {
        if (networkMonitor_) {
            networkMonitor_->stop();
        }
    }
    
    std::vector<TestResult> runTest(std::chrono::seconds duration) {
        std::cout << "开始C++弱网环境测试：" << clientCount_ << "个客户端，持续" 
                  << duration.count() << "秒" << std::endl;
        
        // 启动网络监控
        networkMonitor_->start(std::chrono::seconds(2));
        
        std::vector<TestResult> results(clientCount_);
        std::vector<std::thread> threads;
        
        auto startTime = std::chrono::steady_clock::now();
        
        for (int i = 0; i < clientCount_; i++) {
            threads.emplace_back([this, i, &results, startTime, duration]() {
                results[i] = testClient(i, startTime, duration);
            });
        }
        
        for (auto& thread : threads) {
            thread.join();
        }
        
        return results;
    }
    
    void analyzeResults(const std::vector<TestResult>& results) {
        std::cout << "\n=== C++弱网环境测试结果分析 ===" << std::endl;
        
        int totalSuccess = 0;
        int totalErrors = 0;
        std::chrono::milliseconds totalLatency{0};
        std::chrono::milliseconds maxLatency{0};
        std::chrono::milliseconds minLatency{std::chrono::hours(1)};
        
        int weakNetworkClients = 0;
        int excellentQualityClients = 0;
        int goodQualityClients = 0;
        int fairQualityClients = 0;
        int poorQualityClients = 0;
        int veryPoorQualityClients = 0;
        
        for (const auto& result : results) {
            totalSuccess += result.successCount;
            totalErrors += result.errorCount;
            
            if (result.successCount > 0) {
                totalLatency += result.avgLatency * result.successCount;
                
                if (result.maxLatency > maxLatency) {
                    maxLatency = result.maxLatency;
                }
                if (result.minLatency < minLatency) {
                    minLatency = result.minLatency;
                }
            }
            
            switch (result.networkQuality) {
                case NetworkQuality::Excellent: excellentQualityClients++; break;
                case NetworkQuality::Good: goodQualityClients++; break;
                case NetworkQuality::Fair: fairQualityClients++; break;
                case NetworkQuality::Poor: poorQualityClients++; break;
                case NetworkQuality::VeryPoor: veryPoorQualityClients++; break;
            }
            
            if (result.isWeakNetwork) {
                weakNetworkClients++;
            }
        }
        
        std::chrono::milliseconds avgLatency{0};
        if (totalSuccess > 0) {
            avgLatency = totalLatency / totalSuccess;
        }
        
        double successRate = totalSuccess > 0 ? (double)totalSuccess / (totalSuccess + totalErrors) * 100.0 : 0.0;
        
        std::cout << "测试客户端数: " << results.size() << std::endl;
        std::cout << "总成功请求: " << totalSuccess << std::endl;
        std::cout << "总错误请求: " << totalErrors << std::endl;
        std::cout << "成功率: " << std::fixed << std::setprecision(2) << successRate << "%" << std::endl;
        std::cout << "平均延迟: " << avgLatency.count() << "ms" << std::endl;
        std::cout << "最大延迟: " << maxLatency.count() << "ms" << std::endl;
        std::cout << "最小延迟: " << minLatency.count() << "ms" << std::endl;
        
        std::cout << "\n网络质量分布:" << std::endl;
        std::cout << "  优秀: " << excellentQualityClients << "个客户端" << std::endl;
        std::cout << "  良好: " << goodQualityClients << "个客户端" << std::endl;
        std::cout << "  一般: " << fairQualityClients << "个客户端" << std::endl;
        std::cout << "  差: " << poorQualityClients << "个客户端" << std::endl;
        std::cout << "  很差: " << veryPoorQualityClients << "个客户端" << std::endl;
        std::cout << "  弱网环境: " << weakNetworkClients << "个客户端" << std::endl;
        
        std::cout << "\nC++弱网环境适应性评估:" << std::endl;
        
        if (successRate > 90) {
            std::cout << "✅ C++弱网环境适应性: 优秀" << std::endl;
        } else if (successRate > 80) {
            std::cout << "⚠️  C++弱网环境适应性: 良好" << std::endl;
        } else if (successRate > 70) {
            std::cout << "⚠️  C++弱网环境适应性: 一般" << std::endl;
        } else {
            std::cout << "❌ C++弱网环境适应性: 需要优化" << std::endl;
        }
        
        if (avgLatency < std::chrono::milliseconds(2000)) {
            std::cout << "✅ C++弱网环境延迟: 优秀" << std::endl;
        } else if (avgLatency < std::chrono::milliseconds(5000)) {
            std::cout << "⚠️  C++弱网环境延迟: 良好" << std::endl;
        } else if (avgLatency < std::chrono::milliseconds(10000)) {
            std::cout << "⚠️  C++弱网环境延迟: 一般" << std::endl;
        } else {
            std::cout << "❌ C++弱网环境延迟: 需要优化" << std::endl;
        }
        
        if (weakNetworkClients == 0) {
            std::cout << "✅ C++网络质量: 所有客户端网络质量良好" << std::endl;
        } else {
            std::cout << "⚠️  C++网络质量: " << weakNetworkClients << "个客户端处于弱网环境" << std::endl;
        }
    }

private:
    TestResult testClient(int clientId, std::chrono::steady_clock::time_point startTime, 
                         std::chrono::seconds duration) {
        TestResult result;
        result.clientId = clientId;
        result.minLatency = std::chrono::hours(1);
        
        auto testEndTime = startTime + duration;
        
        while (std::chrono::steady_clock::now() < testEndTime) {
            // 模拟服务发现
            auto latency = simulateServiceDiscovery();
            if (latency.count() > 0) {
                result.successCount++;
                result.avgLatency += latency;
                
                if (latency > result.maxLatency) {
                    result.maxLatency = latency;
                }
                if (latency < result.minLatency) {
                    result.minLatency = latency;
                }
            } else {
                result.errorCount++;
                result.errors.push_back("服务发现失败");
            }
            
            // 更新网络状态
            result.networkQuality = networkMonitor_->getQuality();
            result.isWeakNetwork = networkMonitor_->isWeakNetwork();
            
            // 模拟网络延迟
            std::this_thread::sleep_for(std::chrono::milliseconds(500));
        }
        
        if (result.successCount > 0) {
            result.avgLatency = result.avgLatency / result.successCount;
        }
        
        return result;
    }
    
    std::chrono::milliseconds simulateServiceDiscovery() {
        auto start = std::chrono::high_resolution_clock::now();
        
        std::string response;
        int httpStatus = 0;
        
        // 检查缓存
        std::string cacheKey = "service:test-service";
        std::string cachedResponse;
        if (cache_.get(cacheKey, cachedResponse)) {
            return std::chrono::milliseconds(1); // 缓存命中，返回1ms延迟
        }
        
        // 发送请求
        bool success = httpClient_->get("/v1/discovery?service=test-service", response, httpStatus);
        
        auto end = std::chrono::high_resolution_clock::now();
        auto latency = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
        
        if (success) {
            // 缓存响应
            cache_.set(cacheKey, response, std::chrono::seconds(30));
            return latency;
        }
        
        return std::chrono::milliseconds(0); // 失败
    }
    
    std::string controllerURL_;
    int clientCount_;
    std::unique_ptr<HttpClient> httpClient_;
    std::unique_ptr<NetworkMonitor> networkMonitor_;
    SimpleCache cache_;
};

// 检查Controller状态
bool checkControllerStatus(const std::string& url) {
    CURL* curl = curl_easy_init();
    if (!curl) return false;
    
    curl_easy_setopt(curl, CURLOPT_URL, (url + "/healthz").c_str());
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 5L);
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
    
    long httpCode = 0;
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, [](void*, size_t, size_t, void*) -> size_t {
        return 0;
    });
    
    CURLcode res = curl_easy_perform(curl);
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &httpCode);
    curl_easy_cleanup(curl);
    
    return (res == CURLE_OK) && (httpCode == 200);
}

int main() {
    std::cout << "=== Plum C++弱网环境测试 ===" << std::endl;
    std::cout << "测试目标: 验证C++ SDK在弱网环境下的服务发现能力" << std::endl;
    
    std::string controllerURL = "http://localhost:8080";
    
    // 检查Controller状态
    std::cout << "检查Controller状态..." << std::endl;
    if (!checkControllerStatus(controllerURL)) {
        std::cout << "❌ Controller未运行" << std::endl;
        std::cout << "请先启动Controller:" << std::endl;
        std::cout << "运行: make controller-run" << std::endl;
        return 1;
    }
    std::cout << "✅ Controller运行正常" << std::endl;
    
    // 创建测试器
    WeakNetworkTester tester(controllerURL, 15); // 15个客户端
    
    // 运行测试
    auto results = tester.runTest(std::chrono::seconds(90)); // 测试90秒
    
    // 分析结果
    tester.analyzeResults(results);
    
    std::cout << "\nC++弱网环境测试完成" << std::endl;
    
    return 0;
}
