#include "DistributedMemory.hpp"
#include <iostream>
#include <sstream>
#include <cstdlib>
#include <cstring>
#include <nlohmann/json.hpp>
#include <httplib.h>

using json = nlohmann::json;

namespace plum {
namespace kv {

// ===== Base64 编解码 =====

static const std::string base64_chars = 
    "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    "abcdefghijklmnopqrstuvwxyz"
    "0123456789+/";

static std::string base64Encode(const void* data, size_t size) {
    const unsigned char* bytes = static_cast<const unsigned char*>(data);
    std::string ret;
    int i = 0;
    unsigned char char_array_3[3];
    unsigned char char_array_4[4];

    while (size--) {
        char_array_3[i++] = *(bytes++);
        if (i == 3) {
            char_array_4[0] = (char_array_3[0] & 0xfc) >> 2;
            char_array_4[1] = ((char_array_3[0] & 0x03) << 4) + ((char_array_3[1] & 0xf0) >> 4);
            char_array_4[2] = ((char_array_3[1] & 0x0f) << 2) + ((char_array_3[2] & 0xc0) >> 6);
            char_array_4[3] = char_array_3[2] & 0x3f;

            for(i = 0; i < 4; i++)
                ret += base64_chars[char_array_4[i]];
            i = 0;
        }
    }

    if (i) {
        for(int j = i; j < 3; j++)
            char_array_3[j] = '\0';

        char_array_4[0] = (char_array_3[0] & 0xfc) >> 2;
        char_array_4[1] = ((char_array_3[0] & 0x03) << 4) + ((char_array_3[1] & 0xf0) >> 4);
        char_array_4[2] = ((char_array_3[1] & 0x0f) << 2) + ((char_array_3[2] & 0xc0) >> 6);

        for (int j = 0; j < i + 1; j++)
            ret += base64_chars[char_array_4[j]];

        while(i++ < 3)
            ret += '=';
    }

    return ret;
}

static bool isBase64(unsigned char c) {
    return (isalnum(c) || (c == '+') || (c == '/'));
}

static std::vector<uint8_t> base64Decode(const std::string& encoded_string) {
    size_t in_len = encoded_string.size();
    int i = 0;
    int j = 0;
    int in_ = 0;
    unsigned char char_array_4[4], char_array_3[3];
    std::vector<uint8_t> ret;

    while (in_len-- && (encoded_string[in_] != '=') && isBase64(encoded_string[in_])) {
        char_array_4[i++] = encoded_string[in_]; in_++;
        if (i == 4) {
            for (i = 0; i < 4; i++)
                char_array_4[i] = base64_chars.find(char_array_4[i]);

            char_array_3[0] = (char_array_4[0] << 2) + ((char_array_4[1] & 0x30) >> 4);
            char_array_3[1] = ((char_array_4[1] & 0xf) << 4) + ((char_array_4[2] & 0x3c) >> 2);
            char_array_3[2] = ((char_array_4[2] & 0x3) << 6) + char_array_4[3];

            for (i = 0; i < 3; i++)
                ret.push_back(char_array_3[i]);
            i = 0;
        }
    }

    if (i) {
        for (j = 0; j < i; j++)
            char_array_4[j] = base64_chars.find(char_array_4[j]);

        char_array_3[0] = (char_array_4[0] << 2) + ((char_array_4[1] & 0x30) >> 4);
        char_array_3[1] = ((char_array_4[1] & 0xf) << 4) + ((char_array_4[2] & 0x3c) >> 2);

        for (j = 0; j < i - 1; j++)
            ret.push_back(char_array_3[j]);
    }

    return ret;
}

static std::string getEnvOr(const char* key, const char* defaultVal) {
    const char* val = std::getenv(key);
    return val ? std::string(val) : std::string(defaultVal);
}

static std::string parseHost(const std::string& url) {
    // 简单解析 http://host:port
    auto pos = url.find("://");
    if (pos == std::string::npos) return url;
    auto hostPort = url.substr(pos + 3);
    auto slashPos = hostPort.find('/');
    if (slashPos != std::string::npos) {
        hostPort = hostPort.substr(0, slashPos);
    }
    return hostPort;
}

static int parsePort(const std::string& url) {
    auto host = parseHost(url);
    auto colonPos = host.find(':');
    if (colonPos != std::string::npos) {
        return std::stoi(host.substr(colonPos + 1));
    }
    return url.find("https://") == 0 ? 443 : 80;
}

static std::string parseHostOnly(const std::string& url) {
    auto host = parseHost(url);
    auto colonPos = host.find(':');
    if (colonPos != std::string::npos) {
        return host.substr(0, colonPos);
    }
    return host;
}

// Factory method
std::shared_ptr<DistributedMemory> DistributedMemory::create(
    const std::string& ns,
    const std::string& controllerURL
) {
    std::string url = controllerURL;
    if (url.empty()) {
        url = getEnvOr("CONTROLLER_BASE", "http://127.0.0.1:8080");
    }
    
    auto dm = std::shared_ptr<DistributedMemory>(new DistributedMemory(ns, url));
    dm->preloadCache();
    dm->startSSE();
    return dm;
}

SyncMode DistributedMemory::parseSyncMode() {
    const char* mode = std::getenv("PLUM_KV_SYNC_MODE");
    if (!mode) return SyncMode::Polling; // 默认轮询
    
    std::string modeStr(mode);
    if (modeStr == "sse" || modeStr == "SSE") return SyncMode::SSE;
    if (modeStr == "disabled" || modeStr == "DISABLED") return SyncMode::Disabled;
    return SyncMode::Polling;
}

DistributedMemory::DistributedMemory(const std::string& ns, const std::string& controllerURL)
    : namespace_(ns), controllerURL_(controllerURL), sseRunning_(false) {
    syncMode_ = parseSyncMode();
    
    const char* modeNames[] = {"Polling", "SSE", "Disabled"};
    std::cout << "[KVStore] Initialized for namespace: " << ns 
              << ", sync mode: " << modeNames[static_cast<int>(syncMode_)] << std::endl;
}

DistributedMemory::~DistributedMemory() {
    stopSSE();
}

void DistributedMemory::preloadCache() {
    try {
        auto host = parseHostOnly(controllerURL_);
        auto port = parsePort(controllerURL_);
        
        httplib::Client cli(host, port);
        cli.set_connection_timeout(3, 0);
        cli.set_read_timeout(5, 0);
        
        auto res = cli.Get(buildURL("/v1/kv/" + namespace_).c_str());
        if (res && res->status == 200) {
            auto j = json::parse(res->body);
            if (j.is_array()) {
                std::lock_guard<std::mutex> lock(cacheMutex_);
                for (auto& item : j) {
                    std::string key = item.value("key", "");
                    std::string value = item.value("value", "");
                    std::string type = item.value("type", "string");
                    if (!key.empty()) {
                        cache_[key] = value;
                        types_[key] = type;
                    }
                }
                std::cout << "[KVStore] Preloaded " << cache_.size() << " keys" << std::endl;
            }
        }
    } catch (const std::exception& e) {
        std::cerr << "[KVStore] Preload failed: " << e.what() << std::endl;
    }
}

void DistributedMemory::startSSE() {
    if (syncMode_ == SyncMode::Disabled) {
        std::cout << "[KVStore] Sync disabled, using local cache only" << std::endl;
        return;
    }
    
    sseRunning_ = true;
    
    if (syncMode_ == SyncMode::SSE) {
        std::cout << "[KVStore] Starting SSE mode" << std::endl;
        sseThread_ = std::thread([this]() { sseLoop(); });
    } else {
        std::cout << "[KVStore] Starting polling mode (5s interval)" << std::endl;
        sseThread_ = std::thread([this]() { pollingLoop(); });
    }
}

void DistributedMemory::stopSSE() {
    sseRunning_ = false;
    if (sseThread_.joinable()) {
        sseThread_.join();
    }
}

void DistributedMemory::pollingLoop() {
    while (sseRunning_) {
        std::this_thread::sleep_for(std::chrono::seconds(5));
        if (!sseRunning_) break;
        refresh();
    }
}

void DistributedMemory::sseLoop() {
    while (sseRunning_) {
        try {
            auto host = parseHostOnly(controllerURL_);
            auto port = parsePort(controllerURL_);
            
            httplib::Client cli(host, port);
            cli.set_read_timeout(300, 0); // 5分钟超时
            
            std::cout << "[KVStore] Connecting to SSE stream..." << std::endl;
            
            auto res = cli.Get(
                buildURL("/v1/stream?namespace=" + namespace_).c_str(),
                [this](const char* data, size_t len) {
                    if (len > 0) {
                        parseSSEStream(std::string(data, len));
                    }
                    return sseRunning_.load();
                }
            );
            
            if (!sseRunning_) break;
            
            std::cout << "[KVStore] SSE disconnected, reconnecting in 3s..." << std::endl;
            std::this_thread::sleep_for(std::chrono::seconds(3));
        } catch (const std::exception& e) {
            std::cerr << "[KVStore] SSE error: " << e.what() << ", retrying in 5s..." << std::endl;
            std::this_thread::sleep_for(std::chrono::seconds(5));
        }
    }
}

void DistributedMemory::parseSSEStream(const std::string& chunk) {
    sseBuffer_ += chunk;
    
    // SSE协议：事件以 \n\n 结尾
    size_t pos;
    while ((pos = sseBuffer_.find("\n\n")) != std::string::npos) {
        std::string event = sseBuffer_.substr(0, pos);
        sseBuffer_ = sseBuffer_.substr(pos + 2);
        
        // 解析事件
        std::string eventType;
        std::string eventData;
        
        std::istringstream iss(event);
        std::string line;
        while (std::getline(iss, line)) {
            if (line.empty() || line[0] == ':') continue; // 注释或空行
            
            size_t colonPos = line.find(':');
            if (colonPos == std::string::npos) continue;
            
            std::string field = line.substr(0, colonPos);
            std::string value = colonPos + 1 < line.size() ? line.substr(colonPos + 1) : "";
            
            // 去掉开头的空格
            if (!value.empty() && value[0] == ' ') value = value.substr(1);
            
            if (field == "event") {
                eventType = value;
            } else if (field == "data") {
                eventData = value;
            }
        }
        
        // 处理kv事件
        if (eventType == "kv" && !eventData.empty()) {
            try {
                auto j = json::parse(eventData);
                std::string key = j.value("key", "");
                std::string value = j.value("value", "");
                std::string type = j.value("type", "string");
                
                if (!key.empty()) {
                    bool shouldUpdate = false;
                    {
                        std::lock_guard<std::mutex> lock(cacheMutex_);
                        
                        // 客户端去重：如果值没变化，跳过更新和回调
                        auto it = cache_.find(key);
                        if (it == cache_.end() || it->second != value || types_[key] != type) {
                            cache_[key] = value;
                            types_[key] = type;
                            shouldUpdate = true;
                        }
                    }
                    
                    if (shouldUpdate) {
                        std::cout << "[KVStore] SSE update: " << key << " = " << value << std::endl;
                        // 触发回调
                        onSSEEvent(key, value);
                    }
                }
            } catch (const std::exception& e) {
                std::cerr << "[KVStore] Parse SSE event failed: " << e.what() << std::endl;
            }
        }
    }
}

void DistributedMemory::refresh() {
    preloadCache();
}

std::string DistributedMemory::buildURL(const std::string& path) const {
    return path;
}

bool DistributedMemory::httpPut(const std::string& key, const std::string& value, const std::string& type) {
    try {
        auto host = parseHostOnly(controllerURL_);
        auto port = parsePort(controllerURL_);
        
        httplib::Client cli(host, port);
        cli.set_connection_timeout(3, 0);
        cli.set_write_timeout(5, 0);
        
        json body = {
            {"value", value},
            {"type", type}
        };
        
        auto res = cli.Put(buildURL("/v1/kv/" + namespace_ + "/" + key).c_str(),
                          body.dump(), "application/json");
        
        if (res && res->status == 200) {
            // 更新本地缓存
            std::lock_guard<std::mutex> lock(cacheMutex_);
            cache_[key] = value;
            types_[key] = type;
            return true;
        }
        return false;
    } catch (const std::exception& e) {
        std::cerr << "[KVStore] Put error: " << e.what() << std::endl;
        return false;
    }
}

std::string DistributedMemory::httpGet(const std::string& key, bool& found) {
    try {
        auto host = parseHostOnly(controllerURL_);
        auto port = parsePort(controllerURL_);
        
        httplib::Client cli(host, port);
        cli.set_connection_timeout(3, 0);
        cli.set_read_timeout(5, 0);
        
        auto res = cli.Get(buildURL("/v1/kv/" + namespace_ + "/" + key).c_str());
        if (res && res->status == 200) {
            auto j = json::parse(res->body);
            std::string value = j.value("value", "");
            std::string type = j.value("type", "string");
            
            // 更新本地缓存
            std::lock_guard<std::mutex> lock(cacheMutex_);
            cache_[key] = value;
            types_[key] = type;
            
            found = true;
            return value;
        }
        found = false;
        return "";
    } catch (const std::exception& e) {
        found = false;
        return "";
    }
}

bool DistributedMemory::httpDelete(const std::string& key) {
    try {
        auto host = parseHostOnly(controllerURL_);
        auto port = parsePort(controllerURL_);
        
        httplib::Client cli(host, port);
        cli.set_connection_timeout(3, 0);
        
        auto res = cli.Delete(buildURL("/v1/kv/" + namespace_ + "/" + key).c_str());
        if (res && (res->status == 204 || res->status == 200)) {
            // 从本地缓存删除
            std::lock_guard<std::mutex> lock(cacheMutex_);
            cache_.erase(key);
            types_.erase(key);
            return true;
        }
        return false;
    } catch (const std::exception& e) {
        return false;
    }
}

// ===== Public API实现 =====

bool DistributedMemory::put(const std::string& key, const std::string& value) {
    return httpPut(key, value, "string");
}

std::string DistributedMemory::get(const std::string& key, const std::string& defaultValue) {
    // 优先读缓存
    {
        std::lock_guard<std::mutex> lock(cacheMutex_);
        auto it = cache_.find(key);
        if (it != cache_.end()) {
            return it->second;
        }
    }
    
    // 缓存miss，请求Controller
    bool found;
    std::string value = httpGet(key, found);
    return found ? value : defaultValue;
}

bool DistributedMemory::exists(const std::string& key) {
    // 先查缓存
    {
        std::lock_guard<std::mutex> lock(cacheMutex_);
        if (cache_.find(key) != cache_.end()) {
            return true;
        }
    }
    
    // 请求Controller确认
    bool found;
    httpGet(key, found);
    return found;
}

bool DistributedMemory::remove(const std::string& key) {
    return httpDelete(key);
}

bool DistributedMemory::putInt(const std::string& key, int64_t value) {
    return httpPut(key, std::to_string(value), "int");
}

int64_t DistributedMemory::getInt(const std::string& key, int64_t defaultValue) {
    std::string val = get(key, "");
    if (val.empty()) return defaultValue;
    try {
        return std::stoll(val);
    } catch (...) {
        return defaultValue;
    }
}

bool DistributedMemory::putDouble(const std::string& key, double value) {
    return httpPut(key, std::to_string(value), "double");
}

double DistributedMemory::getDouble(const std::string& key, double defaultValue) {
    std::string val = get(key, "");
    if (val.empty()) return defaultValue;
    try {
        return std::stod(val);
    } catch (...) {
        return defaultValue;
    }
}

bool DistributedMemory::putBool(const std::string& key, bool value) {
    return httpPut(key, value ? "true" : "false", "bool");
}

bool DistributedMemory::getBool(const std::string& key, bool defaultValue) {
    std::string val = get(key, "");
    if (val.empty()) return defaultValue;
    return val == "true" || val == "1";
}

bool DistributedMemory::putBytes(const std::string& key, const void* data, size_t size) {
    if (!data || size == 0) {
        return httpPut(key, "", "bytes");
    }
    std::string encoded = base64Encode(data, size);
    return httpPut(key, encoded, "bytes");
}

bool DistributedMemory::putBytes(const std::string& key, const std::vector<uint8_t>& data) {
    if (data.empty()) {
        return httpPut(key, "", "bytes");
    }
    return putBytes(key, data.data(), data.size());
}

std::vector<uint8_t> DistributedMemory::getBytes(const std::string& key, const std::vector<uint8_t>& defaultValue) {
    std::string encoded = get(key, "");
    if (encoded.empty()) {
        return defaultValue;
    }
    
    try {
        return base64Decode(encoded);
    } catch (...) {
        return defaultValue;
    }
}

bool DistributedMemory::getBytes(const std::string& key, void* buffer, size_t& size) {
    if (!buffer || size == 0) {
        return false;
    }
    
    std::string encoded = get(key, "");
    if (encoded.empty()) {
        size = 0;
        return false;
    }
    
    try {
        auto data = base64Decode(encoded);
        if (data.size() > size) {
            // 缓冲区太小
            size = data.size();  // 返回实际需要的大小
            return false;
        }
        
        // 复制到用户buffer
        std::memcpy(buffer, data.data(), data.size());
        size = data.size();
        return true;
    } catch (...) {
        size = 0;
        return false;
    }
}

std::map<std::string, std::string> DistributedMemory::getAll() {
    std::lock_guard<std::mutex> lock(cacheMutex_);
    return cache_;
}

bool DistributedMemory::putBatch(const std::map<std::string, std::string>& kvs) {
    try {
        auto host = parseHostOnly(controllerURL_);
        auto port = parsePort(controllerURL_);
        
        httplib::Client cli(host, port);
        cli.set_connection_timeout(3, 0);
        cli.set_write_timeout(5, 0);
        
        json::array_t items;
        for (const auto& [k, v] : kvs) {
            items.push_back({
                {"key", k},
                {"value", v},
                {"type", "string"}
            });
        }
        
        json body = {{"items", items}};
        
        auto res = cli.Post(buildURL("/v1/kv/" + namespace_ + "/batch").c_str(),
                           body.dump(), "application/json");
        
        if (res && res->status == 200) {
            // 更新本地缓存
            std::lock_guard<std::mutex> lock(cacheMutex_);
            for (const auto& [k, v] : kvs) {
                cache_[k] = v;
                types_[k] = "string";
            }
            return true;
        }
        return false;
    } catch (const std::exception& e) {
        std::cerr << "[KVStore] Batch put error: " << e.what() << std::endl;
        return false;
    }
}

void DistributedMemory::subscribe(std::function<void(const std::string&, const std::string&)> callback) {
    std::lock_guard<std::mutex> lock(callbackMutex_);
    callbacks_.push_back(callback);
}

void DistributedMemory::onSSEEvent(const std::string& event, const std::string& data) {
    // SSE事件处理（当实现真正的SSE时使用）
    // 触发回调
    std::lock_guard<std::mutex> lock(callbackMutex_);
    for (auto& cb : callbacks_) {
        cb(event, data);
    }
}

} // namespace kv
} // namespace plum

