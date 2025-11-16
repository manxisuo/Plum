#include "plumworker/stream_worker.hpp"
#include <iostream>
#include <cstdlib>
#include <thread>
#include <chrono>
#include <grpcpp/grpcpp.h>
#include "proto/task_service.grpc.pb.h"

using grpc::ClientContext;
using grpc::ClientReaderWriter;
using grpc::Status;
using plum::task::TaskService;
using plum::task::TaskRequest;
using plum::task::TaskAck;
using plum::task::WorkerRegister;
using plum::task::TaskResponse;
using plum::task::Heartbeat;

namespace plumworker {

StreamWorker::StreamWorker(const StreamWorkerOptions& options)
    : options_(options) {
    loadFromEnvironment();
}

StreamWorker::~StreamWorker() {
    stop();
}

void StreamWorker::loadFromEnvironment() {
    auto getEnv = [](const char* key, const char* defaultVal) -> std::string {
        const char* val = std::getenv(key);
        return val ? std::string(val) : std::string(defaultVal);
    };

    if (options_.workerId.empty()) {
        options_.workerId = getEnv("WORKER_ID", "");
    }
    if (options_.nodeId.empty()) {
        options_.nodeId = getEnv("WORKER_NODE_ID", "nodeA");
    }
    if (options_.instanceId.empty()) {
        options_.instanceId = getEnv("PLUM_INSTANCE_ID", "");
    }
    if (options_.appName.empty()) {
        options_.appName = getEnv("PLUM_APP_NAME", "");
    }
    if (options_.appVersion.empty()) {
        options_.appVersion = getEnv("PLUM_APP_VERSION", "1.0.0");
    }
    // 总是优先使用环境变量 CONTROLLER_GRPC_ADDR
    // 这样 Agent 注入的环境变量总是会被使用
    const char* addr = std::getenv("CONTROLLER_GRPC_ADDR");
    if (addr) {
        options_.controllerGrpcAddr = addr;
    }
}

void StreamWorker::registerTask(const std::string& taskName, TaskHandler handler) {
    handlers_[taskName] = handler;
    // 自动添加到支持的任务列表
    bool exists = false;
    for (const auto& task : options_.tasks) {
        if (task == taskName) {
            exists = true;
            break;
        }
    }
    if (!exists) {
        options_.tasks.push_back(taskName);
    }
}

bool StreamWorker::start() {
    if (running_.load()) {
        std::cerr << "[StreamWorker] Already running" << std::endl;
        return false;
    }

    if (handlers_.empty()) {
        std::cerr << "[StreamWorker] No tasks registered" << std::endl;
        return false;
    }

    if (options_.workerId.empty()) {
        std::cerr << "[StreamWorker] workerId is required" << std::endl;
        return false;
    }

    stop_.store(false);
    running_.store(true);

    std::cout << "========================================" << std::endl;
    std::cout << "  Plum Stream Worker" << std::endl;
    std::cout << "========================================" << std::endl;
    std::cout << "Worker ID:         " << options_.workerId << std::endl;
    std::cout << "Node ID:           " << options_.nodeId << std::endl;
    std::cout << "Instance ID:       " << options_.instanceId << std::endl;
    std::cout << "App Name:          " << options_.appName << std::endl;
    std::cout << "App Version:       " << options_.appVersion << std::endl;
    std::cout << "Controller gRPC:   " << options_.controllerGrpcAddr << std::endl;
    std::cout << "Supported Tasks:   ";
    for (size_t i = 0; i < options_.tasks.size(); ++i) {
        std::cout << options_.tasks[i];
        if (i < options_.tasks.size() - 1) std::cout << ", ";
    }
    std::cout << std::endl;
    std::cout << "========================================" << std::endl;

    // 主循环：连接和重连
    while (!stop_.load()) {
        try {
            if (runTaskStream()) {
                // 正常退出（可能是 stop() 调用）
                break;
            }
        } catch (const std::exception& e) {
            std::cerr << "[StreamWorker] Error: " << e.what() << std::endl;
        }

        if (stop_.load()) {
            break;
        }

        if (options_.autoReconnect) {
            std::cout << "[StreamWorker] Reconnecting in " << options_.reconnectIntervalSec 
                      << " seconds..." << std::endl;
            std::this_thread::sleep_for(std::chrono::seconds(options_.reconnectIntervalSec));
        } else {
            break;
        }
    }

    running_.store(false);
    return true;
}

void StreamWorker::stop() {
    if (!running_.load()) {
        return;
    }

    stop_.store(true);
    running_.store(false);

    // 等待主循环退出
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
}

bool StreamWorker::runTaskStream() {
    // 创建 gRPC 客户端
    auto channel = grpc::CreateChannel(options_.controllerGrpcAddr, 
                                       grpc::InsecureChannelCredentials());
    auto stub = TaskService::NewStub(channel);

    ClientContext context;
    auto stream = stub->TaskStream(&context);
    // TaskStream 返回 unique_ptr<ClientReaderWriterInterface>，需要转换为 shared_ptr 以便在多个线程间共享
    streamPtr_ = std::shared_ptr<grpc::ClientReaderWriterInterface<TaskAck, TaskRequest>>(stream.release());

    // 发送注册信息
    if (!sendRegistration(streamPtr_)) {
        std::cerr << "[StreamWorker] Failed to send registration" << std::endl;
        streamPtr_ = nullptr;
        return false;
    }

    std::cout << "[StreamWorker] Connected to Controller and registered" << std::endl;

    // 接收任务线程
    std::atomic<bool> receiveThreadRunning{true};
    std::thread receiveThread([this, &receiveThreadRunning]() {
        TaskRequest task;
        while (receiveThreadRunning.load()) {
            // Read() 应该在锁外调用，因为它是阻塞操作
            // 但是需要先检查 streamPtr_ 是否有效
            {
                std::lock_guard<std::mutex> lock(streamMutex_);
                if (!streamPtr_) {
                    break;
                }
            }
            
            // 在锁外进行阻塞的 Read 操作
            if (!streamPtr_->Read(&task)) {
                // Read 失败，可能是流关闭
                std::cerr << "[StreamWorker] Stream Read failed, exiting receive thread" << std::endl;
                break;
            }
            
            // 在独立线程中处理任务
            std::thread([this, task]() {
                handleTask(task.task_id(), task.name(), task.payload());
            }).detach();
        }
    });

    // 心跳线程
    std::thread heartbeatThread([this, &receiveThreadRunning]() {
        std::cout << "[StreamWorker] Heartbeat thread started, interval=" << options_.heartbeatIntervalSec << "s" << std::endl;
        while (!stop_.load() && receiveThreadRunning.load()) {
            std::this_thread::sleep_for(
                std::chrono::seconds(options_.heartbeatIntervalSec));
            
            if (stop_.load() || !receiveThreadRunning.load()) {
                break;
            }

            if (!sendHeartbeat(streamPtr_)) {
                std::cerr << "[StreamWorker] Failed to send heartbeat" << std::endl;
                receiveThreadRunning.store(false);
                break;
            }
        }
    });

    // 等待接收线程结束
    receiveThread.join();
    receiveThreadRunning.store(false);

    // 停止心跳线程
    heartbeatThread.join();

    // 完成流
    Status status;
    {
        std::lock_guard<std::mutex> lock(streamMutex_);
        if (streamPtr_) {
            status = streamPtr_->Finish();
            streamPtr_ = nullptr;
        }
    }
    if (!status.ok()) {
        std::cerr << "[StreamWorker] Stream finished with error: " 
                  << status.error_message() << std::endl;
    }

    return stop_.load();
}

bool StreamWorker::sendRegistration(std::shared_ptr<grpc::ClientReaderWriterInterface<TaskAck, TaskRequest>> stream) {
    TaskAck ack;
    auto* reg = ack.mutable_register_();
    reg->set_worker_id(options_.workerId);
    reg->set_node_id(options_.nodeId);
    reg->set_instance_id(options_.instanceId);
    reg->set_app_name(options_.appName);
    reg->set_app_version(options_.appVersion);
    
    for (const auto& task : options_.tasks) {
        reg->add_tasks(task);
    }
    
    for (const auto& [key, value] : options_.labels) {
        reg->mutable_labels()->insert({key, value});
    }

    std::lock_guard<std::mutex> lock(streamMutex_);
    return stream->Write(ack);
}

bool StreamWorker::sendHeartbeat(std::shared_ptr<grpc::ClientReaderWriterInterface<TaskAck, TaskRequest>> stream) {
    if (!stream) {
        return false;
    }
    TaskAck ack;
    auto* hb = ack.mutable_heartbeat();
    hb->set_worker_id(options_.workerId);

    std::lock_guard<std::mutex> lock(streamMutex_);
    bool ok = stream->Write(ack);
    if (ok) {
        std::cout << "[StreamWorker] Heartbeat sent" << std::endl;
    }
    return ok;
}

bool StreamWorker::sendTaskResult(std::shared_ptr<grpc::ClientReaderWriterInterface<TaskAck, TaskRequest>> stream,
                                  const std::string& taskId,
                                  const std::string& result,
                                  const std::string& error) {
    TaskAck ack;
    auto* resp = ack.mutable_result();
    resp->set_task_id(taskId);
    resp->set_result(result);
    resp->set_error(error);

    std::lock_guard<std::mutex> lock(streamMutex_);
    bool success = stream->Write(ack);
    if (success) {
        std::cout << "[StreamWorker] Task result sent: " << taskId << std::endl;
    } else {
        std::cerr << "[StreamWorker] Failed to send task result: " << taskId << std::endl;
    }
    return success;
}

void StreamWorker::handleTask(const std::string& taskId, const std::string& taskName,
                              const std::string& payload) {
    auto summarize = [](const std::string& text) -> std::string {
        static constexpr std::size_t kMaxLen = 2048;
        if (text.size() <= kMaxLen) {
            return text;
        }
        return text.substr(0, kMaxLen) + "...(truncated)";
    };

    std::cout << "[StreamWorker] Executing task: " << taskName 
              << " (taskId: " << taskId << ")" << std::endl;
    if (!payload.empty()) {
        std::cout << "[StreamWorker] Task payload: " << summarize(payload) << std::endl;
    } else {
        std::cout << "[StreamWorker] Task payload: <empty>" << std::endl;
    }

    // 查找任务处理函数
    auto it = handlers_.find(taskName);
    if (it == handlers_.end()) {
        std::cerr << "[StreamWorker] Unknown task: " << taskName << std::endl;
        sendTaskResult(streamPtr_, taskId, "", 
                      "Unknown task: " + taskName);
        return;
    }

    // 执行任务
    std::string result;
    std::string error;
    try {
        result = it->second(taskId, taskName, payload);
    } catch (const std::exception& e) {
        error = std::string("Task execution error: ") + e.what();
        std::cerr << "[StreamWorker] " << error << std::endl;
    }

    // 发送结果（需要加锁保护 stream）
    if (streamPtr_) {
        if (!result.empty()) {
            std::cout << "[StreamWorker] Task result for " << taskId << ": "
                      << summarize(result) << std::endl;
        }
        if (!error.empty()) {
            std::cerr << "[StreamWorker] Task error for " << taskId << ": "
                      << summarize(error) << std::endl;
        }
        sendTaskResult(streamPtr_, taskId, result, error);
    }
}

} // namespace plumworker

