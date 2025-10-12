#pragma once

#include <functional>
#include <list>
#include <map>
#include <string>
#include <thread>
#include <atomic>
#include <memory>

// Forward declarations
namespace httplib {
    class Server;
}

namespace plumresource {

// 资源数据类型
enum class DataType {
    INT,
    DOUBLE,
    BOOL,
    ENUM,
    STRING
};

// 资源状态描述（用于注册资源）
struct ResourceStateDesc {
    DataType type;        // 数据项的类型
    std::string name;     // 数据项的名称
    std::string value;    // 数据项的取值
    std::string unit;     // 数据项的单位
    
    ResourceStateDesc() = default;  // 默认构造函数
    ResourceStateDesc(DataType t, const std::string& n, const std::string& v, const std::string& u = "")
        : type(t), name(n), value(v), unit(u) {}
};

// 资源操作描述（用于注册资源）
struct ResourceOpDesc {
    DataType type;        // 数据项的类型
    std::string name;     // 数据项的名称
    std::string value;    // 数据项的取值
    std::string unit;     // 数据项的单位
    std::string min;      // 当type是INT/DOUBLE/ENUM时有效
    std::string max;      // 当type是INT/DOUBLE/ENUM时有效
    
    ResourceOpDesc() = default;  // 默认构造函数
    ResourceOpDesc(DataType t, const std::string& n, const std::string& v, 
                   const std::string& u = "", const std::string& minVal = "", const std::string& maxVal = "")
        : type(t), name(n), value(v), unit(u), min(minVal), max(maxVal) {}
};

// 资源描述（用于注册资源）
struct ResourceDesc {
    std::string node;                              // 资源所在的节点的名称
    std::string deviceID;                          // 资源的ID（由注册者保证唯一性）
    std::string type;                              // 资源的类型（例如：Radar/Sonar/XXGun等）
    std::list<ResourceStateDesc> stateDescList;    // 状态描述列表
    std::list<ResourceOpDesc> opDescList;          // 操作描述列表
    
    ResourceDesc() = default;  // 默认构造函数
    ResourceDesc(const std::string& n, const std::string& id, const std::string& t)
        : node(n), deviceID(id), type(t) {}
};

// 资源状态（用于上报资源状态信息）
struct ResourceState {
    std::string name;     // 数据项的名称
    std::string value;    // 数据项的取值
    
    ResourceState(const std::string& n, const std::string& v) : name(n), value(v) {}
};

// 资源操作（用于下发资源操作）
struct ResourceOp {
    std::string name;     // 数据项的名称
    std::string value;    // 数据项的取值
    
    ResourceOp(const std::string& n, const std::string& v) : name(n), value(v) {}
};

// 资源操作回调函数类型
using ResourceOpCallback = std::function<void(const std::list<ResourceOp>&)>;

// 资源管理选项
struct ResourceOptions {
    std::string controllerBase;    // Controller基础URL
    std::string resourceId;        // 资源ID
    std::string nodeId;            // 节点ID
    int heartbeatSec{10};          // 心跳间隔（秒）
    // HTTP端口已移除，系统自动分配（避免冲突）
};

// 资源管理器类
class ResourceManager {
public:
    explicit ResourceManager(const ResourceOptions& opt);
    ~ResourceManager();

    // 注册资源
    bool registerResource(const ResourceDesc& resource);
    
    // 注销资源
    bool deleteResource(const std::string& resourceId);
    
    // 上报资源状态信息
    void submitResourceState(const std::list<ResourceState>& stateList);
    
    // 注册资源操作回调函数
    void setResourceOpCallback(ResourceOpCallback callback);
    
    // 启动资源管理器
    bool start();
    
    // 停止资源管理器
    void stop();

private:
    ResourceOptions options_;
    std::atomic<bool> stop_{false};
    std::thread hbThread_;
    std::string httpURL_;
    ResourceOpCallback opCallback_;
    std::map<std::string, ResourceDesc> registeredResources_;
    
    // HTTP服务器相关
    httplib::Server* httpServer_{nullptr};
    std::thread httpServerThread_;
    std::atomic<int> actualPort_{0};
    
    void heartbeatLoop();
    bool doRegister();
    bool doHeartbeat();
    bool startHttp();
    bool doRegisterResource(const ResourceDesc& resource);
    bool doDeleteResource(const std::string& resourceId);
    void doSubmitResourceState(const std::list<ResourceState>& stateList);
    
    // HTTP服务器回调
    void handleResourceOp(const std::list<ResourceOp>& opList);
    
    // 辅助函数
    std::string dataTypeToString(DataType type) const;
    DataType stringToDataType(const std::string& str) const;
};

} // namespace plumresource
