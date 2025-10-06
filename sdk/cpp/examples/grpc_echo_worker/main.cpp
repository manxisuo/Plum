#include <iostream>
#include <string>
#include <thread>
#include <chrono>
#include "grpc_worker.hpp"

using namespace plum;

std::string getenv_or(const char* name, const char* default_value) {
    const char* value = std::getenv(name);
    return value ? value : default_value;
}

std::string echoTask(const std::string& payload) {
    std::cout << "[GRPC Echo Worker] Executing echo task with payload: " << payload << std::endl;
    
    // Simulate some work
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    
    // Return the payload as result
    return "{\"echo\": \"" + payload + "\", \"timestamp\": " + std::to_string(std::time(nullptr)) + "}";
}

std::string helloTask(const std::string& payload) {
    std::cout << "[GRPC Echo Worker] Executing hello task with payload: " << payload << std::endl;
    
    // Simulate some work
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    
    return "{\"message\": \"Hello from gRPC worker!\", \"payload\": \"" + payload + "\"}";
}

int main() {
    std::cout << "[GRPC Echo Worker] Starting gRPC-based worker..." << std::endl;
    
    // Print environment variables
    std::cout << "PLUM_INSTANCE_ID: " << getenv_or("PLUM_INSTANCE_ID", "not set") << std::endl;
    std::cout << "PLUM_APP_NAME: " << getenv_or("PLUM_APP_NAME", "not set") << std::endl;
    std::cout << "PLUM_APP_VERSION: " << getenv_or("PLUM_APP_VERSION", "not set") << std::endl;
    
    GRPCWorkerOptions opt;
    opt.controllerBase = getenv_or("CONTROLLER_BASE", "http://127.0.0.1:8080");
    opt.workerId = getenv_or("WORKER_ID", "grpc-echo-1");
    opt.nodeId = getenv_or("WORKER_NODE_ID", "nodeA");
    opt.grpcAddress = getenv_or("GRPC_ADDRESS", "0.0.0.0:18082");
    opt.heartbeatSec = 5;
    
    // Set additional labels
    opt.labels["appName"] = getenv_or("PLUM_APP_NAME", "grpc-echo-app");
    opt.labels["deploymentId"] = "grpc-deploy-123";
    opt.labels["version"] = getenv_or("PLUM_APP_VERSION", "v2.0.0");

    GRPCWorker worker(opt);
    
    // Register task handlers
    worker.registerTask("grpc.echo", echoTask);
    worker.registerTask("grpc.hello", helloTask);
    
    // Start the worker
    if (!worker.start()) {
        std::cerr << "[GRPC Echo Worker] Failed to start worker" << std::endl;
        return 1;
    }
    
    std::cout << "[GRPC Echo Worker] Worker started successfully. Press Ctrl+C to stop." << std::endl;
    
    // Keep the worker running
    while (worker.isRunning()) {
        std::this_thread::sleep_for(std::chrono::seconds(1));
    }
    
    std::cout << "[GRPC Echo Worker] Worker stopped." << std::endl;
    return 0;
}
