#include "plum_client.hpp"
#include <curl/curl.h>
#include <nlohmann/json.hpp>
#include <sstream>

namespace plumclient {

// 静态回调函数
static size_t WriteCallback(void* contents, size_t size, size_t nmemb, void* userp) {
    size_t totalSize = size * nmemb;
    std::string* str = static_cast<std::string*>(userp);
    str->append(static_cast<char*>(contents), totalSize);
    return totalSize;
}

ServiceClient::ServiceClient(const std::string& controllerUrl,
                           std::shared_ptr<WeakNetworkSupport> weakNetworkSupport,
                           std::shared_ptr<Cache> cache)
    : controllerUrl_(controllerUrl),
      weakNetworkSupport_(weakNetworkSupport),
      cache_(cache) {
}

bool ServiceClient::registerService(const ServiceRegistration& registration) {
    // 构建请求体
    nlohmann::json request;
    request["instanceId"] = registration.instanceId;
    request["serviceName"] = registration.serviceName;
    request["nodeId"] = registration.nodeId;
    request["ip"] = registration.ip;
    request["port"] = registration.port;
    request["protocol"] = registration.protocol;
    request["version"] = registration.version;
    
    // 添加标签
    nlohmann::json labels = nlohmann::json::object();
    for (const auto& label : registration.labels) {
        labels[label.first] = label.second;
    }
    request["labels"] = labels;
    
    // 序列化JSON
    std::string jsonStr = request.dump();
    
    // 发送请求
    std::string url = controllerUrl_ + "/v1/services/register";
    return makeRequest("POST", url, jsonStr, {{"Content-Type", "application/json"}});
}

bool ServiceClient::heartbeatService(const ServiceHeartbeat& heartbeat) {
    // 构建请求体
    nlohmann::json request;
    request["instanceId"] = heartbeat.instanceId;
    
    // 添加端点信息
    nlohmann::json endpoints = nlohmann::json::array();
    for (const auto& endpoint : heartbeat.endpoints) {
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
        
        endpoints.push_back(ep);
    }
    request["endpoints"] = endpoints;
    
    // 序列化JSON
    std::string jsonStr = request.dump();
    
    // 发送请求
    std::string url = controllerUrl_ + "/v1/services/heartbeat";
    return makeRequest("POST", url, jsonStr, {{"Content-Type", "application/json"}});
}

bool ServiceClient::unregisterService(const std::string& instanceId) {
    std::string url = controllerUrl_ + "/v1/services?instanceId=" + instanceId;
    return makeRequest("DELETE", url);
}

bool ServiceClient::makeRequest(const std::string& method,
                               const std::string& path,
                               const std::string& body,
                               const std::map<std::string, std::string>& headers) {
    CURL* curl = curl_easy_init();
    if (!curl) {
        return false;
    }
    
    // 设置URL
    curl_easy_setopt(curl, CURLOPT_URL, path.c_str());
    
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
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 30L);
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 10L);
    
    // 设置头部
    struct curl_slist* headerList = nullptr;
    for (const auto& header : headers) {
        std::string headerStr = header.first + ": " + header.second;
        headerList = curl_slist_append(headerList, headerStr.c_str());
    }
    if (headerList) {
        curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headerList);
    }
    
    // 设置响应处理
    std::string responseBody;
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &responseBody);
    
    // 执行请求
    CURLcode res = curl_easy_perform(curl);
    long httpCode = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &httpCode);
    
    // 清理
    if (headerList) {
        curl_slist_free_all(headerList);
    }
    curl_easy_cleanup(curl);
    
    // 检查结果
    bool success = (res == CURLE_OK) && (httpCode >= 200 && httpCode < 300);
    
    // 记录请求（用于弱网环境支持）
    if (weakNetworkSupport_) {
        weakNetworkSupport_->recordRequest();
        if (!success) {
            // 记录错误
        }
    }
    
    return success;
}

} // namespace plumclient
