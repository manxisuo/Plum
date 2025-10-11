#include <iostream>
#include <string>
#include <cstdlib>
#include <thread>
#include <chrono>
#include <signal.h>
#include <atomic>
#include <grpcpp/grpcpp.h>
#include "proto/task_service.grpc.pb.h"

using grpc::Server;
using grpc::ServerBuilder;
using grpc::ServerContext;
using grpc::Status;

std::atomic<bool> g_running{true};

void signal_handler(int sig) {
    std::cout << "\n[Worker Demo] Received signal " << sig << ", shutting down..." << std::endl;
    g_running = false;
}

std::string getEnvOr(const char* key, const char* defaultVal) {
    const char* val = std::getenv(key);
    return val ? std::string(val) : std::string(defaultVal);
}

// 实现TaskService
class TaskServiceImpl final : public plum::task::TaskService::Service {
public:
    Status ExecuteTask(ServerContext* context, 
                      const plum::task::TaskRequest* request,
                      plum::task::TaskResponse* response) override {
        std::cout << "[Worker] Executing task: " << request->name() 
                  << " (ID: " << request->task_id() << ")" << std::endl;
        std::cout << "[Worker] Payload: " << request->payload() << std::endl;

        // 模拟任务处理
        std::this_thread::sleep_for(std::chrono::seconds(2));

        // 返回结果
        std::string result = "{\"status\":\"success\",\"message\":\"Task completed\",\"processed_at\":\"" 
                           + std::to_string(time(nullptr)) + "\"}";
        response->set_result(result);
        
        std::cout << "[Worker] Task completed: " << request->name() << std::endl;
        return Status::OK;
    }

    Status HealthCheck(ServerContext* context,
                      const plum::task::HealthRequest* request,
                      plum::task::HealthResponse* response) override {
        response->set_healthy(true);
        response->set_message("Worker is running");
        return Status::OK;
    }
};

// 注册Worker到Controller
bool registerWorker(const std::string& controllerBase, 
                   const std::string& workerId,
                   const std::string& nodeId,
                   const std::string& instanceId,
                   const std::string& appName,
                   const std::string& appVersion,
                   const std::string& grpcAddress) {
    // 构造注册请求
    std::string url = controllerBase + "/v1/embedded-workers/register";
    std::string json = "{";
    json += "\"workerId\":\"" + workerId + "\",";
    json += "\"nodeId\":\"" + nodeId + "\",";
    json += "\"instanceId\":\"" + instanceId + "\",";
    json += "\"appName\":\"" + appName + "\",";
    json += "\"appVersion\":\"" + appVersion + "\",";
    json += "\"grpcAddress\":\"" + grpcAddress + "\",";
    json += "\"tasks\":[\"demo.echo\",\"demo.delay\"],";
    json += "\"labels\":{\"type\":\"demo\"}";
    json += "}";

    // 简单的HTTP POST（实际应用应使用httplib）
    std::string cmd = "curl -s -X POST \"" + url + "\" "
                     "-H \"Content-Type: application/json\" "
                     "-d '" + json + "'";
    int ret = system(cmd.c_str());
    
    if (ret == 0) {
        std::cout << "[Worker] Registered to Controller successfully" << std::endl;
        return true;
    } else {
        std::cerr << "[Worker] Failed to register to Controller" << std::endl;
        return false;
    }
}

// 心跳
void heartbeatLoop(const std::string& controllerBase, const std::string& workerId) {
    while (g_running) {
        std::this_thread::sleep_for(std::chrono::seconds(30));
        if (!g_running) break;

        std::string url = controllerBase + "/v1/embedded-workers/heartbeat";
        std::string json = "{\"workerId\":\"" + workerId + "\"}";
        std::string cmd = "curl -s -X POST \"" + url + "\" "
                         "-H \"Content-Type: application/json\" "
                         "-d '" + json + "' > /dev/null 2>&1";
        system(cmd.c_str());
    }
}

int main() {
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    // 读取环境变量
    std::string instanceId = getEnvOr("PLUM_INSTANCE_ID", "worker-demo-001");
    std::string appName = getEnvOr("PLUM_APP_NAME", "worker-demo");
    std::string appVersion = getEnvOr("PLUM_APP_VERSION", "1.0.0");
    std::string workerId = getEnvOr("WORKER_ID", "worker-demo-1");
    std::string nodeId = getEnvOr("WORKER_NODE_ID", "nodeA");
    std::string controllerBase = getEnvOr("CONTROLLER_BASE", "http://127.0.0.1:8080");
    std::string grpcAddress = getEnvOr("GRPC_ADDRESS", "0.0.0.0:18090");

    std::cout << "========================================" << std::endl;
    std::cout << "  Plum Worker Demo" << std::endl;
    std::cout << "========================================" << std::endl;
    std::cout << "Worker ID:    " << workerId << std::endl;
    std::cout << "Node ID:      " << nodeId << std::endl;
    std::cout << "Instance ID:  " << instanceId << std::endl;
    std::cout << "App Name:     " << appName << std::endl;
    std::cout << "App Version:  " << appVersion << std::endl;
    std::cout << "gRPC Address: " << grpcAddress << std::endl;
    std::cout << "Controller:   " << controllerBase << std::endl;
    std::cout << "========================================" << std::endl;

    // 启动gRPC服务器
    TaskServiceImpl service;
    ServerBuilder builder;
    builder.AddListeningPort(grpcAddress, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);
    std::unique_ptr<Server> server(builder.BuildAndStart());
    
    std::cout << "[Worker] gRPC server listening on " << grpcAddress << std::endl;

    // 注册到Controller
    if (!registerWorker(controllerBase, workerId, nodeId, instanceId, 
                       appName, appVersion, grpcAddress)) {
        std::cerr << "[Worker] Registration failed, but continuing..." << std::endl;
    }

    // 启动心跳线程
    std::thread heartbeatThread(heartbeatLoop, controllerBase, workerId);

    std::cout << "[Worker] Ready to accept tasks!" << std::endl;
    std::cout << "[Worker] Supported tasks: demo.echo, demo.delay" << std::endl;
    std::cout << std::endl;

    // 主循环
    while (g_running) {
        std::this_thread::sleep_for(std::chrono::seconds(5));
    }

    // 清理
    std::cout << "[Worker] Shutting down..." << std::endl;
    server->Shutdown();
    if (heartbeatThread.joinable()) {
        heartbeatThread.join();
    }
    
    std::cout << "[Worker] Goodbye!" << std::endl;
    return 0;
}

