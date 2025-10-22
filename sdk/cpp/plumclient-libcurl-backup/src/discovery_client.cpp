#include "plum_client.hpp"
#include <curl/curl.h>
#include <nlohmann/json.hpp>
#include <sstream>
#include <random>

namespace plumclient {

// 静态回调函数
static size_t WriteCallback(void* contents, size_t size, size_t nmemb, void* userp) {
    size_t totalSize = size * nmemb;
    std::string* str = static_cast<std::string*>(userp);
    str->append(static_cast<char*>(contents), totalSize);
    return totalSize;
}

DiscoveryClient::DiscoveryClient(const std::string& controllerUrl,
                               std::shared_ptr<WeakNetworkSupport> weakNetworkSupport,
                               std::shared_ptr<Cache> cache)
    : controllerUrl_(controllerUrl),
      weakNetworkSupport_(weakNetworkSupport),
      cache_(cache) {
}

std::vector<Endpoint> DiscoveryClient::discoverService(const DiscoveryRequest& request) {
    // 构建缓存键
    std::string cacheKey = "discovery:" + request.service;
    if (!request.version.empty()) {
        cacheKey += ":" + request.version;
    }
    if (!request.protocol.empty()) {
        cacheKey += ":" + request.protocol;
    }
    
    // 尝试从缓存获取
    if (cache_) {
        auto cached = cache_->get(cacheKey);
        if (cached) {
            // 解析缓存的JSON
            try {
                nlohmann::json root = nlohmann::json::parse(*cached);
                return parseEndpointsFromJson(root);
            } catch (const std::exception& e) {
                // JSON解析失败，忽略缓存
            }
        }
    }
    
    // 构建查询参数
    std::string queryParams = "?service=" + request.service;
    if (!request.version.empty()) {
        queryParams += "&version=" + request.version;
    }
    if (!request.protocol.empty()) {
        queryParams += "&protocol=" + request.protocol;
    }
    
    // 发送请求
    std::string url = controllerUrl_ + "/v1/discovery" + queryParams;
    auto endpoints = makeDiscoveryRequest(url);
    
    // 缓存结果
    if (cache_ && !endpoints.empty()) {
        nlohmann::json root = nlohmann::json::array();
        for (const auto& endpoint : endpoints) {
            nlohmann::json ep;
            ep["serviceName"] = endpoint.serviceName;
            ep["instanceId"] = endpoint.instanceId;
            ep["nodeId"] = endpoint.nodeId;
            ep["ip"] = endpoint.ip;
            ep["port"] = endpoint.port;
            ep["protocol"] = endpoint.protocol;
            ep["version"] = endpoint.version;
            ep["healthy"] = endpoint.healthy;
            
            // 添加标签
            nlohmann::json labels = nlohmann::json::object();
            for (const auto& label : endpoint.labels) {
                labels[label.first] = label.second;
            }
            ep["labels"] = labels;
            
            root.push_back(ep);
        }
        
        // 序列化并缓存
        std::string jsonStr = root.dump();
        cache_->set(cacheKey, jsonStr, std::chrono::seconds(30));
    }
    
    return endpoints;
}

std::optional<Endpoint> DiscoveryClient::discoverRandomService(const DiscoveryRequest& request) {
    // 构建缓存键
    std::string cacheKey = "discovery_random:" + request.service;
    if (!request.version.empty()) {
        cacheKey += ":" + request.version;
    }
    if (!request.protocol.empty()) {
        cacheKey += ":" + request.protocol;
    }
    
    // 尝试从缓存获取
    if (cache_) {
        auto cached = cache_->get(cacheKey);
        if (cached) {
            // 解析缓存的JSON
            try {
                nlohmann::json root = nlohmann::json::parse(*cached);
                return parseEndpointFromJson(root);
            } catch (const std::exception& e) {
                // JSON解析失败，忽略缓存
            }
        }
    }
    
    // 构建查询参数
    std::string queryParams = "?service=" + request.service;
    if (!request.version.empty()) {
        queryParams += "&version=" + request.version;
    }
    if (!request.protocol.empty()) {
        queryParams += "&protocol=" + request.protocol;
    }
    
    // 发送请求
    std::string url = controllerUrl_ + "/v1/discovery/random" + queryParams;
    auto endpoint = makeRandomDiscoveryRequest(url);
    
    // 缓存结果
    if (cache_ && endpoint) {
        nlohmann::json root;
        root["serviceName"] = endpoint->serviceName;
        root["instanceId"] = endpoint->instanceId;
        root["nodeId"] = endpoint->nodeId;
        root["ip"] = endpoint->ip;
        root["port"] = endpoint->port;
        root["protocol"] = endpoint->protocol;
        root["version"] = endpoint->version;
        root["healthy"] = endpoint->healthy;
        
        // 添加标签
        nlohmann::json labels = nlohmann::json::object();
        for (const auto& label : endpoint->labels) {
            labels[label.first] = label.second;
        }
        root["labels"] = labels;
        
        // 序列化并缓存
        std::string jsonStr = root.dump();
        cache_->set(cacheKey, jsonStr, std::chrono::seconds(30));
    }
    
    return endpoint;
}

std::vector<Endpoint> DiscoveryClient::makeDiscoveryRequest(const std::string& path) {
    CURL* curl = curl_easy_init();
    if (!curl) {
        return {};
    }
    
    std::string responseBody;
    long httpCode = 0;
    
    // 设置URL
    curl_easy_setopt(curl, CURLOPT_URL, path.c_str());
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 30L);
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 10L);
    
    // 设置响应处理
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &responseBody);
    
    // 执行请求
    CURLcode res = curl_easy_perform(curl);
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &httpCode);
    
    curl_easy_cleanup(curl);
    
    // 检查结果
    if (res != CURLE_OK || httpCode != 200) {
        return {};
    }
    
    // 解析JSON响应
    try {
        nlohmann::json root = nlohmann::json::parse(responseBody);
        return parseEndpointsFromJson(root);
    } catch (const std::exception& e) {
        return {};
    }
}

std::optional<Endpoint> DiscoveryClient::makeRandomDiscoveryRequest(const std::string& path) {
    CURL* curl = curl_easy_init();
    if (!curl) {
        return std::nullopt;
    }
    
    std::string responseBody;
    long httpCode = 0;
    
    // 设置URL
    curl_easy_setopt(curl, CURLOPT_URL, path.c_str());
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 30L);
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 10L);
    
    // 设置响应处理
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &responseBody);
    
    // 执行请求
    CURLcode res = curl_easy_perform(curl);
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &httpCode);
    
    curl_easy_cleanup(curl);
    
    // 检查结果
    if (res != CURLE_OK || httpCode != 200) {
        return std::nullopt;
    }
    
    // 解析JSON响应
    try {
        nlohmann::json root = nlohmann::json::parse(responseBody);
        return parseEndpointFromJson(root);
    } catch (const std::exception& e) {
        return std::nullopt;
    }
}

std::vector<Endpoint> DiscoveryClient::parseEndpointsFromJson(const nlohmann::json& root) {
    std::vector<Endpoint> endpoints;
    
    if (!root.is_array()) {
        return endpoints;
    }
    
    for (const auto& item : root) {
        Endpoint endpoint;
        endpoint.serviceName = item.value("serviceName", "");
        endpoint.instanceId = item.value("instanceId", "");
        endpoint.nodeId = item.value("nodeId", "");
        endpoint.ip = item.value("ip", "");
        endpoint.port = item.value("port", 0);
        endpoint.protocol = item.value("protocol", "");
        endpoint.version = item.value("version", "");
        endpoint.healthy = item.value("healthy", true);
        
        // 解析标签
        if (item.contains("labels") && item["labels"].is_object()) {
            const auto& labels = item["labels"];
            for (auto it = labels.begin(); it != labels.end(); ++it) {
                endpoint.labels[it.key()] = it.value().get<std::string>();
            }
        }
        
        endpoints.push_back(endpoint);
    }
    
    return endpoints;
}

std::optional<Endpoint> DiscoveryClient::parseEndpointFromJson(const nlohmann::json& root) {
    if (!root.is_object()) {
        return std::nullopt;
    }
    
    Endpoint endpoint;
    endpoint.serviceName = root.value("serviceName", "");
    endpoint.instanceId = root.value("instanceId", "");
    endpoint.nodeId = root.value("nodeId", "");
    endpoint.ip = root.value("ip", "");
    endpoint.port = root.value("port", 0);
    endpoint.protocol = root.value("protocol", "");
    endpoint.version = root.value("version", "");
    endpoint.healthy = root.value("healthy", true);
    
    // 解析标签
    if (root.contains("labels") && root["labels"].is_object()) {
        const auto& labels = root["labels"];
        for (auto it = labels.begin(); it != labels.end(); ++it) {
            endpoint.labels[it.key()] = it.value().get<std::string>();
        }
    }
    
    return endpoint;
}

} // namespace plumclient
