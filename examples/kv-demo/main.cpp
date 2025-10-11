#include <iostream>
#include <thread>
#include <chrono>
#include <signal.h>
#include <atomic>
#include <cstdlib>
#include <plumkv/DistributedMemory.hpp>

using namespace plum::kv;

std::atomic<bool> g_running{true};
std::shared_ptr<DistributedMemory> g_dm;

void signal_handler(int sig) {
    std::cout << "\n[KV Demo] Received signal " << sig << ", shutting down gracefully..." << std::endl;
    
    // 优雅停止：保留进度，但标记为正常停止
    if (g_dm) {
        g_dm->remove("app.crashed");  // 清除崩溃标记
        g_dm->put("app.status", "stopped");  // 标记为正常停止（但保留进度）
        g_dm->put("app.stop_time", std::to_string(std::time(nullptr)));
        std::cout << "[KV Demo] Saved state before stopping (progress preserved)" << std::endl;
    }
    
    g_running = false;
}

std::string getEnvOr(const char* key, const char* defaultVal) {
    const char* val = std::getenv(key);
    return val ? std::string(val) : std::string(defaultVal);
}

int main() {
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    std::string appName = getEnvOr("PLUM_APP_NAME", "kv-demo");
    std::string instanceId = getEnvOr("PLUM_INSTANCE_ID", "kv-demo-001");
    std::string controllerBase = getEnvOr("CONTROLLER_BASE", "http://127.0.0.1:8080");

    std::cout << "========================================" << std::endl;
    std::cout << "  Plum KV Demo - 崩溃恢复演示" << std::endl;
    std::cout << "========================================" << std::endl;
    std::cout << "App Name:      " << appName << std::endl;
    std::cout << "Instance ID:   " << instanceId << std::endl;
    std::cout << "Controller:    " << controllerBase << std::endl;
    std::cout << "Namespace:     " << appName << " (使用appName共享)" << std::endl;
    std::cout << "========================================" << std::endl;

    // 创建分布式内存实例（使用appName作为namespace）
    // 同一应用的所有实例共享状态（适合主备切换场景）
    g_dm = DistributedMemory::create(appName, controllerBase);
    
    // ===== 状态恢复逻辑 =====
    
    // 检查是否有未完成的任务（崩溃或正常停止都会保留进度）
    int taskProgress = g_dm->getInt("task.progress", 0);
    int taskCounter = g_dm->getInt("task.counter", 0);
    bool wasCrashed = g_dm->exists("app.crashed");
    std::string lastStatus = g_dm->get("app.status", "");
    
    if (taskProgress > 0 && taskProgress < 100) {
        // 有未完成的任务，恢复执行
        std::string lastCheckpoint = g_dm->get("task.checkpoint", "");
        
        if (wasCrashed) {
            std::cout << "\n💥 检测到崩溃恢复..." << std::endl;
            std::string crashTime = g_dm->get("app.crash_time", "");
            std::cout << "  崩溃时间: " << crashTime << std::endl;
        } else {
            std::cout << "\n⏸️  检测到任务暂停，继续执行..." << std::endl;
            std::cout << "  上次状态: " << lastStatus << std::endl;
        }
        
        std::cout << "  上次进度: " << taskProgress << "%" << std::endl;
        std::cout << "  任务计数: " << taskCounter << std::endl;
        std::cout << "  检查点: " << lastCheckpoint << std::endl;
        
        // 清除崩溃标记（如果有）
        if (wasCrashed) {
            g_dm->remove("app.crashed");
        }
        
        std::cout << "✅ 状态恢复完成，从 " << taskProgress << "% 继续执行" << std::endl;
    } else if (taskProgress >= 100) {
        std::cout << "\n✨ 上次任务已完成，开始新任务" << std::endl;
        // 清除旧状态，重新开始
        taskProgress = 0;
        taskCounter = 0;
    } else {
        std::cout << "\n🆕 首次启动，开始新任务" << std::endl;
        taskProgress = 0;
        taskCounter = 0;
    }
    
    // 设置崩溃标记（异常退出时会保留此标记）
    g_dm->putBool("app.crashed", true);
    g_dm->put("app.crash_time", std::to_string(std::time(nullptr)));
    
    // ===== 模拟业务逻辑 =====
    
    std::cout << "\n🚀 开始执行任务..." << std::endl;
    std::cout << "提示：按Ctrl+C正常退出，或使用 kill -9 模拟崩溃\n" << std::endl;
    
    while (g_running && taskProgress < 100) {
        // 模拟任务执行
        std::this_thread::sleep_for(std::chrono::seconds(2));
        
        if (!g_running) break;
        
        taskProgress += 10;
        taskCounter++;
        
        // 定期保存状态到分布式内存
        g_dm->putInt("task.progress", taskProgress);
        g_dm->putInt("task.counter", taskCounter);
        g_dm->put("task.checkpoint", "step_" + std::to_string(taskCounter));
        g_dm->put("task.status", "running");
        
        std::cout << "📊 进度: " << taskProgress << "%"
                  << " | 计数: " << taskCounter
                  << " | 检查点: step_" << taskCounter
                  << std::endl;
        
        // 模拟不同阶段的操作
        if (taskCounter == 3) {
            std::cout << "💾 保存重要数据..." << std::endl;
            g_dm->put("important.data", "critical_value_" + std::to_string(taskCounter));
        }
        
        if (taskCounter == 5) {
            std::cout << "🔧 执行中间计算..." << std::endl;
            g_dm->putDouble("calculation.result", 3.14159 * taskCounter);
            
            // 测试二进制数据存储（模拟结构体）
            struct CheckpointData {
                int taskId;
                int progress;
                double timestamp;
                char description[32];
            };
            
            CheckpointData checkpoint;
            checkpoint.taskId = 12345;
            checkpoint.progress = taskProgress;
            checkpoint.timestamp = static_cast<double>(std::time(nullptr));
            strncpy(checkpoint.description, "Step5 checkpoint", sizeof(checkpoint.description) - 1);
            checkpoint.description[sizeof(checkpoint.description) - 1] = '\0';
            
            std::cout << "💾 保存二进制检查点数据..." << std::endl;
            g_dm->putBytes("binary.checkpoint", &checkpoint, sizeof(checkpoint));
        }
    }
    
    if (taskProgress >= 100) {
        std::cout << "\n✅ 任务完成！" << std::endl;
        
        // 任务完成，清除状态
        g_dm->remove("app.crashed");
        g_dm->put("task.status", "completed");
        g_dm->put("task.complete_time", std::to_string(std::time(nullptr)));
        
        std::cout << "\n📈 最终统计：" << std::endl;
        std::cout << "  总计数: " << taskCounter << std::endl;
        std::cout << "  完成进度: " << taskProgress << "%" << std::endl;
        
        // 验证二进制数据
        if (g_dm->exists("binary.checkpoint")) {
            auto binaryData = g_dm->getBytes("binary.checkpoint");
            if (binaryData.size() > 0) {
                struct CheckpointData {
                    int taskId;
                    int progress;
                    double timestamp;
                    char description[32];
                };
                
                if (binaryData.size() == sizeof(CheckpointData)) {
                    CheckpointData* checkpoint = reinterpret_cast<CheckpointData*>(binaryData.data());
                    std::cout << "\n🔬 二进制检查点数据验证：" << std::endl;
                    std::cout << "  TaskID: " << checkpoint->taskId << std::endl;
                    std::cout << "  Progress: " << checkpoint->progress << "%" << std::endl;
                    std::cout << "  Description: " << checkpoint->description << std::endl;
                }
            }
        }
        
        auto allData = g_dm->getAll();
        std::cout << "\n📦 分布式KV存储中的所有数据：" << std::endl;
        for (const auto& [k, v] : allData) {
            std::cout << "  " << k << " = " << v << std::endl;
        }
    }
    
    std::cout << "\n[KV Demo] Goodbye!" << std::endl;
    return 0;
}

