#include "DistributedMemory.hpp"
#include <iostream>
#include <sstream>
#include <cstdlib>
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

DistributedMemory::DistributedMemory(const std::string& ns, const std::string& controllerURL)
    : namespace_(ns), controllerURL_(controllerURL), sseRunning_(false) {
    std::cout << "[KVStore] Initialized for namespace: " << ns << std::endl;
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
    // SSE实现较复杂，暂时简化为定期轮询
    // 后续可以改进为真正的SSE EventSource
    sseRunning_ = true;
    sseThread_ = std::thread([this]() {
        while (sseRunning_) {
            std::this_thread::sleep_for(std::chrono::seconds(5));
            if (!sseRunning_) break;
            // 定期刷新缓存
            refresh();
        }
    });
}

void DistributedMemory::stopSSE() {
    sseRunning_ = false;
    if (sseThread_.joinable()) {
        sseThread_.join();
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

