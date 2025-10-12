#include "plum_resource.hpp"
#include "../../env_loader.hpp"
#include <httplib.h>
#include <nlohmann/json.hpp>
#include <iostream>
#include <chrono>
#include <random>
#include <sstream>
#include <unistd.h>

using json = nlohmann::json;

namespace plumresource {

static std::string get_local_ip() {
    return "127.0.0.1"; // MVP
}

static std::string get_hostname() {
    char hostname[256];
    if (gethostname(hostname, sizeof(hostname)) == 0) {
        return std::string(hostname);
    }
    return "unknown";
}

static std::string generate_uuid() {
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, 15);
    std::uniform_int_distribution<> dis2(8, 11);
    
    std::stringstream ss;
    int i;
    ss << std::hex;
    for (i = 0; i < 8; i++) {
        ss << dis(gen);
    }
    ss << "-";
    for (i = 0; i < 4; i++) {
        ss << dis(gen);
    }
    ss << "-4";
    for (i = 0; i < 3; i++) {
        ss << dis(gen);
    }
    ss << "-";
    ss << dis2(gen);
    for (i = 0; i < 3; i++) {
        ss << dis(gen);
    }
    ss << "-";
    for (i = 0; i < 12; i++) {
        ss << dis(gen);
    }
    return ss.str();
}

void ResourceOptions::normalize() {
    // 如果nodeId未指定，自动使用主机名
    if (nodeId.empty()) {
        nodeId = get_hostname();
    }
    
    // 如果resourceId未指定，尝试从.env读取，否则生成并保存
    if (resourceId.empty()) {
        // 尝试从.env读取
        resourceId = plum::env::readValue("RESOURCE_ID");
        
        if (resourceId.empty()) {
            // 生成新ID
            std::string uuid = generate_uuid();
            resourceId = nodeId + "-" + uuid.substr(0, 8);
            
            // 保存到.env供下次启动使用
            if (plum::env::writeValue("RESOURCE_ID", resourceId)) {
                std::cout << "[plumresource] Generated resourceId: " << resourceId << " (saved to .env)" << std::endl;
            } else {
                std::cout << "[plumresource] Generated resourceId: " << resourceId << " (failed to save to .env)" << std::endl;
            }
        } else {
            std::cout << "[plumresource] Using resourceId from .env: " << resourceId << std::endl;
        }
    }
}

ResourceManager::ResourceManager(const ResourceOptions& opt) 
    : options_(opt), opCallback_(nullptr) {
}

ResourceManager::~ResourceManager() {
    stop();
}

bool ResourceManager::start() {
    if (!startHttp()) return false;
    hbThread_ = std::thread([this]{ heartbeatLoop(); });
    return true;
}

void ResourceManager::stop() {
    stop_.store(true);
    
    // Stop HTTP server
    if (httpServer_) {
        httpServer_->stop();
        delete httpServer_;
        httpServer_ = nullptr;
    }
    
    // Wait for threads to finish
    if (hbThread_.joinable()) hbThread_.join();
    if (httpServerThread_.joinable()) httpServerThread_.join();
}

bool ResourceManager::startHttp() {
    auto svr = new httplib::Server();
    
    // 处理资源操作请求
    svr->Post("/resource/op", [this](const httplib::Request& req, httplib::Response& res){
        std::cout << "[plumresource] Received HTTP POST request to /resource/op" << std::endl;
        std::cout << "[plumresource] Request body: " << req.body << std::endl;
        try {
            auto j = json::parse(req.body);
            std::cout << "[plumresource] Parsed JSON: " << j.dump() << std::endl;
            std::list<ResourceOp> opList;
            
            if (j.contains("operations") && j["operations"].is_array()) {
                for (const auto& op : j["operations"]) {
                    if (op.contains("name") && op.contains("value")) {
                        std::cout << "[plumresource] Operation - Name: " << op["name"] << ", Value: " << op["value"] << std::endl;
                        opList.emplace_back(op["name"], op["value"]);
                    }
                }
            }
            
            if (!opList.empty() && opCallback_) {
                std::cout << "[plumresource] Calling callback with " << opList.size() << " operations" << std::endl;
                try {
                    handleResourceOp(opList);
                    res.status = 200;
                    res.set_content("{\"status\":\"success\"}", "application/json");
                    std::cout << "[plumresource] Operation handled successfully" << std::endl;
                } catch (const std::exception& e) {
                    std::cout << "[plumresource] Error in callback: " << e.what() << std::endl;
                    res.status = 500;
                    res.set_content("{\"status\":\"error\",\"message\":\"operation handler error\"}", "application/json");
                } catch (...) {
                    std::cout << "[plumresource] Unknown error in callback" << std::endl;
                    res.status = 500;
                    res.set_content("{\"status\":\"error\",\"message\":\"unknown error\"}", "application/json");
                }
            } else {
                std::cout << "[plumresource] Invalid request: opList.size()=" << opList.size() << ", opCallback_=" << (opCallback_ ? "true" : "false") << std::endl;
                res.status = 400;
                res.set_content("{\"status\":\"error\",\"message\":\"invalid request or no callback\"}", "application/json");
            }
        } catch (const json::exception& e) {
            std::cout << "[plumresource] JSON parse error: " << e.what() << std::endl;
            res.status = 400;
            res.set_content("{\"status\":\"error\",\"message\":\"json parse error\"}", "application/json");
        } catch (const std::exception& e) {
            std::cout << "[plumresource] Error: " << e.what() << std::endl;
            res.status = 500;
            res.set_content("{\"status\":\"error\",\"message\":\"internal error\"}", "application/json");
        } catch (...) {
            std::cout << "[plumresource] Unknown error" << std::endl;
            res.status = 500;
            res.set_content("{\"status\":\"error\",\"message\":\"unknown error\"}", "application/json");
        }
    });
    
    svr->set_exception_handler([](const httplib::Request&, httplib::Response& res, std::exception_ptr){
        res.status = 500;
        res.set_content("{\"status\":\"error\"}", "application/json");
    });
    
    // 使用系统自动分配端口（避免冲突）
    std::cout << "[plumresource] Starting HTTP server (auto-assign port)..." << std::endl;
    
    // Store server instance for cleanup
    httpServer_ = svr;
    
    // 使用bind_to_any_port自动分配端口
    std::atomic<bool> server_failed{false};
    std::string error_message;
    std::atomic<int> bound_port{0};
    
    std::thread server_thread([this, svr, &server_failed, &error_message, &bound_port]{
        try {
            if (!svr->is_valid()) {
                error_message = "Server configuration invalid";
                server_failed.store(true);
                return;
            }
            
            // 绑定到任意可用端口
            int p = svr->bind_to_any_port("0.0.0.0");
            if (p < 0) {
                error_message = "Failed to bind to any port";
                server_failed.store(true);
                return;
            }
            
            bound_port.store(p);
            std::cout << "[plumresource] HTTP server bound to port " << p << std::endl;
            
            // 开始监听
            if (!svr->listen_after_bind()) {
                error_message = "Failed to listen after bind";
                server_failed.store(true);
                return;
            }
            
            std::cout << "[plumresource] HTTP server stopped" << std::endl;
        } catch (const std::exception& e) {
            error_message = std::string("Exception: ") + e.what();
            server_failed.store(true);
        } catch (...) {
            error_message = "Unknown error";
            server_failed.store(true);
        }
    });
    
    // Move thread to member variable for proper lifecycle management
    httpServerThread_ = std::move(server_thread);
    
    // Wait for server to bind (up to 5 seconds)
    for (int i = 0; i < 50; ++i) {
        if (server_failed.load()) {
            std::cerr << "[plumresource] Failed to start HTTP server: " << error_message << std::endl;
            return false;
        }
        
        int p = bound_port.load();
        if (p > 0) {
            // Server bound successfully
            actualPort_.store(p);
            httpURL_ = "http://" + get_local_ip() + ":" + std::to_string(p);
            std::cout << "[plumresource] HTTP server started successfully on port " << p << std::endl;
            return true;
        }
        
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }
    
    std::cerr << "[plumresource] HTTP server startup timed out" << std::endl;
    return false;
}

bool ResourceManager::registerResource(const ResourceDesc& resource) {
    // Store resource description locally
    registeredResources_[resource.deviceID] = resource;
    
    // Register with controller
    return doRegisterResource(resource);
}

bool ResourceManager::deleteResource(const std::string& resourceId) {
    // Remove from local storage
    registeredResources_.erase(resourceId);
    
    // Delete from controller
    return doDeleteResource(resourceId);
}

void ResourceManager::submitResourceState(const std::list<ResourceState>& stateList) {
    doSubmitResourceState(stateList);
}

void ResourceManager::setResourceOpCallback(ResourceOpCallback callback) {
    opCallback_ = callback;
}

bool ResourceManager::doRegisterResource(const ResourceDesc& resource) {
    try {
        httplib::Client cli(options_.controllerBase.c_str());
        
        json j;
        j["resourceId"] = resource.deviceID;
        j["nodeId"] = resource.node;
        j["type"] = resource.type;
        j["url"] = httpURL_ + "/resource/op";
        
        // Convert state descriptions
        json stateArray = json::array();
        for (const auto& state : resource.stateDescList) {
            json stateJson;
            stateJson["type"] = dataTypeToString(state.type);
            stateJson["name"] = state.name;
            stateJson["value"] = state.value;
            stateJson["unit"] = state.unit;
            stateArray.push_back(stateJson);
        }
        j["stateDesc"] = stateArray;
        
        // Convert operation descriptions
        json opArray = json::array();
        for (const auto& op : resource.opDescList) {
            json opJson;
            opJson["type"] = dataTypeToString(op.type);
            opJson["name"] = op.name;
            opJson["value"] = op.value;
            opJson["unit"] = op.unit;
            opJson["min"] = op.min;
            opJson["max"] = op.max;
            opArray.push_back(opJson);
        }
        j["opDesc"] = opArray;
        
        auto r = cli.Post("/v1/resources/register", j.dump(), "application/json");
        return r && r->status >= 200 && r->status < 300;
    } catch (const std::exception& e) {
        std::cerr << "[plumresource] register error: " << e.what() << std::endl;
        return false;
    }
}

bool ResourceManager::doDeleteResource(const std::string& resourceId) {
    try {
        httplib::Client cli(options_.controllerBase.c_str());
        
        json j;
        j["resourceId"] = resourceId;
        
        auto r = cli.Post("/v1/resources/delete", j.dump(), "application/json");
        return r && r->status >= 200 && r->status < 300;
    } catch (const std::exception& e) {
        std::cerr << "[plumresource] delete error: " << e.what() << std::endl;
        return false;
    }
}

void ResourceManager::doSubmitResourceState(const std::list<ResourceState>& stateList) {
    try {
        httplib::Client cli(options_.controllerBase.c_str());
        
        json j;
        j["resourceId"] = options_.resourceId;
        j["timestamp"] = std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now().time_since_epoch()).count();
        
        // Convert states list to object format
        json statesObject = json::object();
        for (const auto& state : stateList) {
            statesObject[state.name] = state.value;
        }
        j["states"] = statesObject;
        
        auto r = cli.Post("/v1/resources/state", j.dump(), "application/json");
        if (!r || r->status < 200 || r->status >= 300) {
            std::cerr << "[plumresource] submit state failed, status: " << (r ? r->status : -1) << std::endl;
        } else {
            std::cout << "[plumresource] submit state success" << std::endl;
        }
    } catch (const std::exception& e) {
        std::cerr << "[plumresource] submit state error: " << e.what() << std::endl;
    }
}

void ResourceManager::handleResourceOp(const std::list<ResourceOp>& opList) {
    if (opCallback_) {
        opCallback_(opList);
    }
}

bool ResourceManager::doRegister() {
    // Register all resources with controller
    bool allSuccess = true;
    for (const auto& kv : registeredResources_) {
        if (!doRegisterResource(kv.second)) {
            allSuccess = false;
        }
    }
    return allSuccess;
}

bool ResourceManager::doHeartbeat() {
    try {
        httplib::Client cli(options_.controllerBase.c_str());
        
        json j;
        j["resourceId"] = options_.resourceId;
        j["nodeId"] = options_.nodeId;
        
        auto r = cli.Post("/v1/resources/heartbeat", j.dump(), "application/json");
        return r && r->status >= 200 && r->status < 300;
    } catch (const std::exception& e) {
        std::cerr << "[plumresource] heartbeat error: " << e.what() << std::endl;
        return false;
    }
}

void ResourceManager::heartbeatLoop() {
    using namespace std::chrono_literals;
    
    // Initial registration
    std::this_thread::sleep_for(1s);
    doRegister();
    
    // Heartbeat loop
    while (!stop_.load()) {
        doHeartbeat();
        std::this_thread::sleep_for(std::chrono::seconds(options_.heartbeatSec));
    }
}

std::string ResourceManager::dataTypeToString(DataType type) const {
    switch (type) {
        case DataType::INT: return "INT";
        case DataType::DOUBLE: return "DOUBLE";
        case DataType::BOOL: return "BOOL";
        case DataType::ENUM: return "ENUM";
        case DataType::STRING: return "STRING";
        default: return "UNKNOWN";
    }
}

DataType ResourceManager::stringToDataType(const std::string& str) const {
    if (str == "INT") return DataType::INT;
    if (str == "DOUBLE") return DataType::DOUBLE;
    if (str == "BOOL") return DataType::BOOL;
    if (str == "ENUM") return DataType::ENUM;
    if (str == "STRING") return DataType::STRING;
    return DataType::STRING; // Default
}

} // namespace plumresource
