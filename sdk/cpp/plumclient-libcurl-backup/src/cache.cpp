#include "plum_client.hpp"
#include <algorithm>
#include <chrono>
#include <thread>
#include <optional>

namespace plumclient {

Cache::Cache(const WeakNetworkConfig& config) : config_(config) {
    // 启动清理线程
    cleanupThread_ = std::thread(&Cache::cleanup, this);
}

Cache::~Cache() {
    stopCleanup_.store(true);
    if (cleanupThread_.joinable()) {
        cleanupThread_.join();
    }
}

void Cache::set(const std::string& key, const std::string& value, std::chrono::seconds ttl) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    // 如果缓存已满，删除最旧的条目
    if (entries_.size() >= static_cast<size_t>(config_.cacheMaxSize)) {
        auto oldest = std::min_element(entries_.begin(), entries_.end(),
            [](const auto& a, const auto& b) {
                return a.second.createdAt < b.second.createdAt;
            });
        if (oldest != entries_.end()) {
            entries_.erase(oldest);
        }
    }
    
    // 设置TTL
    auto ttlToUse = ttl.count() > 0 ? ttl : config_.cacheTTL;
    
    // 创建缓存条目
    CacheEntry entry;
    entry.value = value;
    entry.createdAt = std::chrono::system_clock::now();
    entry.expiresAt = entry.createdAt + ttlToUse;
    
    entries_[key] = entry;
}

std::optional<std::string> Cache::get(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = entries_.find(key);
    if (it == entries_.end()) {
        missCount_.fetch_add(1);
        return std::nullopt;
    }
    
    // 检查是否过期
    if (isExpired(it->second)) {
        entries_.erase(it);
        missCount_.fetch_add(1);
        return std::nullopt;
    }
    
    hitCount_.fetch_add(1);
    return it->second.value;
}

void Cache::remove(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);
    entries_.erase(key);
}

void Cache::clear() {
    std::lock_guard<std::mutex> lock(mutex_);
    entries_.clear();
    hitCount_.store(0);
    missCount_.store(0);
}

size_t Cache::size() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return entries_.size();
}

std::map<std::string, std::string> Cache::getStats() const {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::map<std::string, std::string> stats;
    stats["size"] = std::to_string(entries_.size());
    stats["max_size"] = std::to_string(config_.cacheMaxSize);
    stats["hit_count"] = std::to_string(hitCount_.load());
    stats["miss_count"] = std::to_string(missCount_.load());
    
    int totalRequests = hitCount_.load() + missCount_.load();
    if (totalRequests > 0) {
        double hitRate = static_cast<double>(hitCount_.load()) / totalRequests;
        stats["hit_rate"] = std::to_string(hitRate);
    } else {
        stats["hit_rate"] = "0.0";
    }
    
    return stats;
}

void Cache::cleanup() {
    while (!stopCleanup_.load()) {
        {
            std::lock_guard<std::mutex> lock(mutex_);
            
            // 删除过期的条目
            auto it = entries_.begin();
            while (it != entries_.end()) {
                if (isExpired(it->second)) {
                    it = entries_.erase(it);
                } else {
                    ++it;
                }
            }
        }
        
        // 每30秒清理一次
        std::this_thread::sleep_for(std::chrono::seconds(30));
    }
}

bool Cache::isExpired(const CacheEntry& entry) const {
    return std::chrono::system_clock::now() > entry.expiresAt;
}

} // namespace plumclient
