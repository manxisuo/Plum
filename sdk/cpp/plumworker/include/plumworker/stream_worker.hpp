#pragma once

#include <functional>
#include <map>
#include <string>
#include <thread>
#include <atomic>
#include <memory>
#include <mutex>
#include <vector>
#include <grpcpp/grpcpp.h>
#include "proto/task_service.grpc.pb.h"

using grpc::ClientReaderWriter;
using plum::task::TaskRequest;
using plum::task::TaskAck;

namespace plumworker {

// 任务处理函数类型
// 参数：taskId (任务ID), taskName (任务名称), payload (任务负载JSON字符串)
// 返回：任务结果JSON字符串
using TaskHandler = std::function<std::string(const std::string& taskId, 
                                               const std::string& taskName, 
                                               const std::string& payload)>;

// Worker 配置选项
struct StreamWorkerOptions {
    std::string controllerGrpcAddr = "127.0.0.1:9090";  // Controller gRPC 地址
    std::string workerId;           // Worker ID（如果为空，从环境变量 WORKER_ID 读取）
    std::string nodeId;             // Node ID（如果为空，从环境变量 WORKER_NODE_ID 读取）
    std::string instanceId;         // Instance ID（如果为空，从环境变量 PLUM_INSTANCE_ID 读取）
    std::string appName;            // App Name（如果为空，从环境变量 PLUM_APP_NAME 读取）
    std::string appVersion;         // App Version（如果为空，从环境变量 PLUM_APP_VERSION 读取）
    std::vector<std::string> tasks; // 支持的任务列表（从注册的任务自动生成）
    std::map<std::string, std::string> labels;  // 标签
    int heartbeatIntervalSec = 30;  // 心跳间隔（秒）
    int reconnectIntervalSec = 5;   // 重连间隔（秒）
    bool autoReconnect = true;      // 是否自动重连
};

// 流式 gRPC Worker（新版）
class StreamWorker {
public:
    explicit StreamWorker(const StreamWorkerOptions& options);
    ~StreamWorker();

    // 注册任务处理函数
    void registerTask(const std::string& taskName, TaskHandler handler);

    // 启动 Worker（阻塞调用）
    bool start();

    // 停止 Worker
    void stop();

    // 检查 Worker 是否正在运行
    bool isRunning() const { return running_.load(); }

private:
    StreamWorkerOptions options_;
    std::map<std::string, TaskHandler> handlers_;
    std::atomic<bool> running_{false};
    std::atomic<bool> stop_{false};
    std::mutex streamMutex_;

    // 从环境变量读取配置
    void loadFromEnvironment();

    // 连接到 Controller 并处理任务流（内部方法）
    bool runTaskStream();

    // 发送注册信息
    bool sendRegistration(std::shared_ptr<grpc::ClientReaderWriterInterface<TaskAck, TaskRequest>> stream);

    // 发送心跳
    bool sendHeartbeat(std::shared_ptr<grpc::ClientReaderWriterInterface<TaskAck, TaskRequest>> stream);

    // 发送任务结果
    bool sendTaskResult(std::shared_ptr<grpc::ClientReaderWriterInterface<TaskAck, TaskRequest>> stream,
                       const std::string& taskId, 
                       const std::string& result, const std::string& error = "");

    // 处理接收到的任务
    void handleTask(const std::string& taskId, const std::string& taskName, 
                   const std::string& payload);

    // Stream 指针（用于在任务处理线程中发送结果）
    std::shared_ptr<grpc::ClientReaderWriterInterface<TaskAck, TaskRequest>> streamPtr_;
};

} // namespace plumworker

