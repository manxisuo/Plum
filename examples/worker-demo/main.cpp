// 使用 SDK 的简化版本示例
#include <iostream>
#include <thread>
#include <chrono>
#include <signal.h>
#include "plumworker/stream_worker.hpp"

using namespace plumworker;

// 全局 Worker 指针，用于信号处理
StreamWorker* g_worker = nullptr;

void signal_handler(int sig) {
    std::cout << "\n[Worker Demo] Received signal " << sig << ", shutting down..." << std::endl;
    if (g_worker) {
        g_worker->stop();
    }
}

int main() {
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    // 配置 Worker
    StreamWorkerOptions options;
    // 大部分配置可以从环境变量自动读取（PLUM_INSTANCE_ID, PLUM_APP_NAME 等）
    // 只需要设置必要的选项
    options.labels["type"] = "demo";

    // 创建 Worker
    StreamWorker worker(options);
    g_worker = &worker;  // 设置全局指针，用于信号处理

    // 注册任务处理函数
    worker.registerTask("demo.echo", [](const std::string& taskId, 
                                        const std::string& taskName, 
                                        const std::string& payload) -> std::string {
        std::cout << "[Task Handler] Executing " << taskName << std::endl;
        std::cout << "[Task Handler] Payload: " << payload << std::endl;
        
        // 模拟任务处理
        std::this_thread::sleep_for(std::chrono::milliseconds(500));
        
        return "{\"status\":\"success\",\"echo\":\"" + payload + "\"}";
    });

    worker.registerTask("demo.delay", [](const std::string& taskId, 
                                         const std::string& taskName, 
                                         const std::string& payload) -> std::string {
        std::cout << "[Task Handler] Executing " << taskName << std::endl;
        
        // 模拟延迟任务
        std::this_thread::sleep_for(std::chrono::seconds(2));
        
        return "{\"status\":\"success\",\"message\":\"Delayed task completed\"}";
    });

    // 启动 Worker（阻塞调用）
    // SDK 会自动处理：连接、注册、心跳、任务接收、结果发送、重连等
    worker.start();

    std::cout << "[Worker Demo] Goodbye!" << std::endl;
    return 0;
}
