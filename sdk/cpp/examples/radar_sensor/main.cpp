#include "plum_resource.hpp"
#include <iostream>
#include <cstdlib>
#include <chrono>
#include <thread>
#include <random>

using namespace plumresource;

static std::string getenv_or(const char* k, const char* d) {
    const char* v = std::getenv(k);
    return v ? std::string(v) : std::string(d);
}

// 模拟雷达传感器类
class RadarSensor {
private:
    std::string deviceId_;
    double currentRange_;
    double currentAngle_;
    bool isActive_;
    std::random_device rd_;
    std::mt19937 gen_;
    std::uniform_real_distribution<double> rangeDist_;
    std::uniform_real_distribution<double> angleDist_;
    
public:
    RadarSensor(const std::string& deviceId) 
        : deviceId_(deviceId), currentRange_(0.0), currentAngle_(0.0), isActive_(false),
          gen_(rd_()), rangeDist_(100.0, 5000.0), angleDist_(0.0, 360.0) {
    }
    
    // 获取当前状态
    std::list<ResourceState> getCurrentState() {
        std::list<ResourceState> states;
        
        if (isActive_) {
            // 模拟雷达数据更新
            currentRange_ = rangeDist_(gen_);
            currentAngle_ = angleDist_(gen_);
        }
        
        states.emplace_back("范围", std::to_string(currentRange_));
        states.emplace_back("角度", std::to_string(currentAngle_));
        states.emplace_back("激活", isActive_ ? "true" : "false");
        states.emplace_back("能量", isActive_ ? "100" : "0");
        
        return states;
    }
    
    // 处理操作命令
    void handleOperations(const std::list<ResourceOp>& operations) {
        std::cout << "[RadarSensor] Received " << operations.size() << " operations:" << std::endl;
        for (const auto& op : operations) {
            std::cout << "[RadarSensor] Processing operation: " << op.name << " = " << op.value << std::endl;
            if (op.name == "能量") {
                if (op.value == "on" || op.value == "true" || op.value == "1") {
                    isActive_ = true;
                    std::cout << "[RadarSensor] Power ON" << std::endl;
                } else if (op.value == "off" || op.value == "false" || op.value == "0") {
                    isActive_ = false;
                    std::cout << "[RadarSensor] Power OFF" << std::endl;
                }
            } else if (op.name == "范围") {
                try {
                    double range = std::stod(op.value);
                    if (range >= 100.0 && range <= 5000.0) {
                        currentRange_ = range;
                        std::cout << "[RadarSensor] Range set to: " << range << std::endl;
                    } else {
                        std::cout << "[RadarSensor] Invalid range: " << range << " (expected 100.0 to 5000.0)" << std::endl;
                    }
                } catch (const std::exception& e) {
                    std::cout << "[RadarSensor] Invalid range value: " << op.value << " (exception: " << e.what() << ")" << std::endl;
                }
            } else if (op.name == "角度") {
                std::cout << "[RadarSensor] Received angle operation with value: " << op.value << std::endl;
                try {
                    double angle = std::stod(op.value);
                    std::cout << "[RadarSensor] Parsed angle value: " << angle << std::endl;
                    if (angle >= 0.0 && angle <= 360.0) {
                        currentAngle_ = angle;
                        std::cout << "[RadarSensor] Angle set to: " << angle << std::endl;
                    } else {
                        std::cout << "[RadarSensor] Invalid angle: " << angle << " (expected 0.0 to 360.0)" << std::endl;
                    }
                } catch (const std::exception& e) {
                    std::cout << "[RadarSensor] Invalid angle value: " << op.value << " (exception: " << e.what() << ")" << std::endl;
                }
            } else {
                std::cout << "[RadarSensor] Unknown operation: " << op.name << " = " << op.value << std::endl;
            }
        }
    }
};

int main() {
    // 配置选项
    ResourceOptions opt;
    opt.controllerBase = getenv_or("CONTROLLER_BASE", "http://127.0.0.1:8080");
    opt.resourceId = getenv_or("RESOURCE_ID", "radar-001");
    opt.nodeId = getenv_or("RESOURCE_NODE_ID", "nodeA");
    opt.heartbeatSec = 10;
    opt.httpPort = 18081; // 固定端口
    
    // 创建资源管理器
    ResourceManager resourceManager(opt);
    
    // 创建雷达传感器
    RadarSensor radar(opt.resourceId);
    
    // 设置操作回调
    resourceManager.setResourceOpCallback([&radar](const std::list<ResourceOp>& operations) {
        radar.handleOperations(operations);
    });
    
    // 定义资源描述
    ResourceDesc radarDesc(opt.nodeId, opt.resourceId, "Radar");
    
    // 添加状态描述
    radarDesc.stateDescList.emplace_back(DataType::DOUBLE, "范围", "0.0", "米");
    radarDesc.stateDescList.emplace_back(DataType::DOUBLE, "角度", "0.0", "度");
    radarDesc.stateDescList.emplace_back(DataType::BOOL, "激活", "false");
    radarDesc.stateDescList.emplace_back(DataType::INT, "能量", "0", "%");
    
    // 添加操作描述
    radarDesc.opDescList.emplace_back(DataType::BOOL, "能量", "false", "", "false", "true");
    radarDesc.opDescList.emplace_back(DataType::DOUBLE, "范围", "1000.0", "米", "100.0", "5000.0");
    radarDesc.opDescList.emplace_back(DataType::DOUBLE, "角度", "0.0", "度", "0.0", "360.0");
    
    // 注册资源
    if (!resourceManager.registerResource(radarDesc)) {
        std::cerr << "Failed to register radar resource" << std::endl;
        return 1;
    }
    
    // 启动资源管理器
    if (!resourceManager.start()) {
        std::cerr << "Failed to start resource manager" << std::endl;
        return 1;
    }
    
    std::cout << "Radar sensor resource manager started successfully" << std::endl;
    std::cout << "Device ID: " << opt.resourceId << std::endl;
    std::cout << "Node ID: " << opt.nodeId << std::endl;
    std::cout << "Controller: " << opt.controllerBase << std::endl;
    std::cout << "HTTP Port: " << opt.httpPort << std::endl;
    
    // 模拟运行，定期上报状态
    while (true) {
        std::this_thread::sleep_for(std::chrono::seconds(5));
        
        // 获取当前状态并上报
        auto states = radar.getCurrentState();
        resourceManager.submitResourceState(states);
        
        std::cout << "[RadarSensor] State updated: ";
        for (const auto& state : states) {
            std::cout << state.name << "=" << state.value << " ";
        }
        std::cout << std::endl;
    }
    
    return 0;
}
