#include <iostream>
#include <string>
#include <cstdlib>
#include <ctime>
#include <thread>
#include <chrono>
#include <signal.h>
#include <atomic>

std::atomic<bool> g_running{true};

void signal_handler(int sig) {
    std::cout << "\n[Demo App] Received signal " << sig << ", shutting down..." << std::endl;
    g_running = false;
}

std::string getEnvOr(const char* key, const char* defaultVal) {
    const char* val = std::getenv(key);
    return val ? std::string(val) : std::string(defaultVal);
}

int main() {
    // 注册信号处理
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    // 从环境变量读取Plum注入的信息
    std::string instanceId = getEnvOr("PLUM_INSTANCE_ID", "unknown");
    std::string appName = getEnvOr("PLUM_APP_NAME", "demo-app");
    std::string appVersion = getEnvOr("PLUM_APP_VERSION", "1.0.0");
    
    std::cout << "========================================" << std::endl;
    std::cout << "  Plum Demo Application" << std::endl;
    std::cout << "========================================" << std::endl;
    std::cout << "App Name:    " << appName << std::endl;
    std::cout << "App Version: " << appVersion << std::endl;
    std::cout << "Instance ID: " << instanceId << std::endl;
    std::cout << "PID:         " << getpid() << std::endl;
    std::cout << "========================================" << std::endl;
    std::cout << std::endl;

    int counter = 0;
    time_t startTime = time(nullptr);

    while (g_running) {
        counter++;
        time_t now = time(nullptr);
        int uptime = (int)(now - startTime);
        
        std::cout << "[" << counter << "] "
                  << "Uptime: " << uptime << "s | "
                  << "Time: " << ctime(&now);
        
        // 每10秒输出一次状态
        std::this_thread::sleep_for(std::chrono::seconds(10));
    }

    std::cout << std::endl;
    std::cout << "[Demo App] Shutting down gracefully..." << std::endl;
    std::cout << "[Demo App] Total uptime: " << (time(nullptr) - startTime) << " seconds" << std::endl;
    std::cout << "[Demo App] Goodbye!" << std::endl;
    
    return 0;
}

