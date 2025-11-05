#include <iostream>
#include <string>
#include <cstdlib>
#include <thread>
#include <chrono>
#include <ctime>
#include <signal.h>
#include <atomic>
#include <mutex>
#include <vector>
#include <grpcpp/grpcpp.h>
#include "proto/task_service.grpc.pb.h"

using grpc::ClientContext;
using grpc::ClientReaderWriter;
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

// 执行任务
std::string executeTask(const std::string& taskName, const std::string& payload) {
    std::cout << "[Worker] Executing task: " << taskName << std::endl;
    std::cout << "[Worker] Payload: " << payload << std::endl;

    // 模拟任务处理
    if (taskName == "demo.echo") {
        std::this_thread::sleep_for(std::chrono::milliseconds(500));
        return "{\"status\":\"success\",\"echo\":\"" + payload + "\"}";
    } else if (taskName == "demo.delay") {
        std::this_thread::sleep_for(std::chrono::seconds(2));
        return "{\"status\":\"success\",\"message\":\"Delayed task completed\"}";
    } else {
        std::this_thread::sleep_for(std::chrono::milliseconds(500));
        return "{\"status\":\"success\",\"message\":\"Task completed\"}";
    }
}

// 连接到 Controller 并处理任务流
void runTaskStream(const std::string& controllerGrpcAddr,
                   const std::string& workerId,
                   const std::string& nodeId,
                   const std::string& instanceId,
                   const std::string& appName,
                   const std::string& appVersion,
                   const std::vector<std::string>& tasks) {
    // 创建 gRPC 客户端
    auto channel = grpc::CreateChannel(controllerGrpcAddr, grpc::InsecureChannelCredentials());
    auto stub = plum::task::TaskService::NewStub(channel);

    ClientContext context;
    std::shared_ptr<ClientReaderWriter<plum::task::TaskAck, plum::task::TaskRequest>> stream(
        stub->TaskStream(&context));

    // 发送注册信息
    plum::task::TaskAck registerAck;
    auto* reg = registerAck.mutable_register_();
    reg->set_worker_id(workerId);
    reg->set_node_id(nodeId);
    reg->set_instance_id(instanceId);
    reg->set_app_name(appName);
    reg->set_app_version(appVersion);
    for (const auto& task : tasks) {
        reg->add_tasks(task);
    }
    reg->mutable_labels()->insert({"type", "demo"});

    if (!stream->Write(registerAck)) {
        std::cerr << "[Worker] Failed to send registration" << std::endl;
        return;
    }

    std::cout << "[Worker] Connected to Controller and registered" << std::endl;
    std::cout << "[Worker] Supported tasks: ";
    for (size_t i = 0; i < tasks.size(); ++i) {
        std::cout << tasks[i];
        if (i < tasks.size() - 1) std::cout << ", ";
    }
    std::cout << std::endl;

    // Stream 写操作需要加锁（gRPC stream 不是线程安全的）
    std::mutex streamMutex;
    
    // 接收任务并处理
    std::thread receiveThread([stream, &streamMutex]() {
        plum::task::TaskRequest task;
        while (stream->Read(&task)) {
            // 在独立线程中处理任务
            std::thread([stream, &streamMutex, task]() {
                std::string result = executeTask(task.name(), task.payload());
                
                // 发送结果（需要加锁）
                std::lock_guard<std::mutex> lock(streamMutex);
                plum::task::TaskAck resultAck;
                auto* resp = resultAck.mutable_result();
                resp->set_task_id(task.task_id());
                resp->set_result(result);
                resp->set_error("");

                if (!stream->Write(resultAck)) {
                    std::cerr << "[Worker] Failed to send task result" << std::endl;
                } else {
                    std::cout << "[Worker] Task result sent: " << task.task_id() << std::endl;
                }
            }).detach();
        }
    });

    // 心跳循环
    std::thread heartbeatThread([stream, &streamMutex, workerId]() {
        while (g_running) {
            std::this_thread::sleep_for(std::chrono::seconds(30));
            if (!g_running) break;

            std::lock_guard<std::mutex> lock(streamMutex);
            plum::task::TaskAck heartbeatAck;
            auto* hb = heartbeatAck.mutable_heartbeat();
            hb->set_worker_id(workerId);
            if (!stream->Write(heartbeatAck)) {
                std::cerr << "[Worker] Failed to send heartbeat" << std::endl;
                break;
            }
        }
    });

    // 等待
    receiveThread.join();
    heartbeatThread.join();

    Status status = stream->Finish();
    if (!status.ok()) {
        std::cerr << "[Worker] Stream finished with error: " << status.error_message() << std::endl;
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
    std::string controllerGrpcAddr = getEnvOr("CONTROLLER_GRPC_ADDR", "127.0.0.1:9090");

    std::cout << "========================================" << std::endl;
    std::cout << "  Plum Worker Demo (Stream Mode)" << std::endl;
    std::cout << "========================================" << std::endl;
    std::cout << "Worker ID:         " << workerId << std::endl;
    std::cout << "Node ID:           " << nodeId << std::endl;
    std::cout << "Instance ID:       " << instanceId << std::endl;
    std::cout << "App Name:          " << appName << std::endl;
    std::cout << "App Version:       " << appVersion << std::endl;
    std::cout << "Controller gRPC:   " << controllerGrpcAddr << std::endl;
    std::cout << "Controller HTTP:   " << controllerBase << std::endl;
    std::cout << "========================================" << std::endl;

    std::vector<std::string> supportedTasks = {"demo.echo", "demo.delay"};

    // 连接到 Controller 并处理任务流
    while (g_running) {
        try {
            runTaskStream(controllerGrpcAddr, workerId, nodeId, instanceId, 
                         appName, appVersion, supportedTasks);
        } catch (const std::exception& e) {
            std::cerr << "[Worker] Error: " << e.what() << std::endl;
            if (g_running) {
                std::cout << "[Worker] Reconnecting in 5 seconds..." << std::endl;
                std::this_thread::sleep_for(std::chrono::seconds(5));
            }
        }
    }

    std::cout << "[Worker] Goodbye!" << std::endl;
    return 0;
}
