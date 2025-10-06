#include "grpc_worker.hpp"
#include <iostream>
#include <sstream>
#include <chrono>
#include <httplib.h>
#include <nlohmann/json.hpp>

using json = nlohmann::json;

namespace plum {

// TaskServiceImpl implementation
grpc::Status TaskServiceImpl::ExecuteTask(grpc::ServerContext* context, const task::TaskRequest* request,
                                         task::TaskResponse* response) {
    std::cout << "[GRPCWorker] Received task: " << request->name() << " (ID: " << request->task_id() << ")" << std::endl;
    
    auto it = handlers_.find(request->name());
    if (it == handlers_.end()) {
        response->set_error("Task not supported: " + request->name());
        return grpc::Status::OK;
    }

    try {
        std::string result = it->second(request->payload());
        response->set_result(result);
        std::cout << "[GRPCWorker] Task executed successfully: " << request->name() << std::endl;
    } catch (const std::exception& e) {
        response->set_error("Task execution failed: " + std::string(e.what()));
        std::cout << "[GRPCWorker] Task execution failed: " << e.what() << std::endl;
    }

    return grpc::Status::OK;
}

grpc::Status TaskServiceImpl::HealthCheck(grpc::ServerContext* context, const task::HealthRequest* request,
                                         task::HealthResponse* response) {
    if (request->worker_id() == workerId_) {
        response->set_healthy(true);
        response->set_message("OK");
    } else {
        response->set_healthy(false);
        response->set_message("Wrong worker ID");
    }
    return grpc::Status::OK;
}

// GRPCWorker implementation
GRPCWorker::GRPCWorker(const GRPCWorkerOptions& options) : options_(options) {
    // Get instance info from environment variables
    if (const char* instanceId = std::getenv("PLUM_INSTANCE_ID")) {
        options_.instanceId = instanceId;
    }
    if (const char* appName = std::getenv("PLUM_APP_NAME")) {
        options_.appName = appName;
    }
    if (const char* appVersion = std::getenv("PLUM_APP_VERSION")) {
        options_.appVersion = appVersion;
    }
}

GRPCWorker::~GRPCWorker() {
    stop();
}

void GRPCWorker::registerTask(const std::string& name, TaskHandler handler) {
    handlers_[name] = std::move(handler);
}

bool GRPCWorker::start() {
    if (running_.load()) {
        return false;
    }

    // Parse gRPC address
    std::string host = "0.0.0.0";
    int port = 18080;
    
    if (!options_.grpcAddress.empty()) {
        size_t colonPos = options_.grpcAddress.find(':');
        if (colonPos != std::string::npos) {
            host = options_.grpcAddress.substr(0, colonPos);
            port = std::stoi(options_.grpcAddress.substr(colonPos + 1));
        } else {
            port = std::stoi(options_.grpcAddress);
        }
    }

    // Create service and server
    service_ = std::make_unique<TaskServiceImpl>(handlers_, options_.workerId);
    
    grpc::ServerBuilder builder;
    builder.AddListeningPort(host + ":" + std::to_string(port), grpc::InsecureServerCredentials());
    builder.RegisterService(service_.get());
    
    server_ = builder.BuildAndStart();
    if (!server_) {
        std::cerr << "[GRPCWorker] Failed to start gRPC server" << std::endl;
        return false;
    }

    std::cout << "[GRPCWorker] gRPC server started on " << host << ":" << port << std::endl;

    // Start server thread
    serverThread_ = std::thread([this]() {
        server_->Wait();
    });

    // Register with controller
    if (!doRegister()) {
        std::cerr << "[GRPCWorker] Failed to register with controller" << std::endl;
        server_->Shutdown();
        if (serverThread_.joinable()) {
            serverThread_.join();
        }
        return false;
    }

    // Start heartbeat thread
    running_.store(true);
    hbThread_ = std::thread([this]() {
        heartbeatLoop();
    });

    std::cout << "[GRPCWorker] Worker started successfully" << std::endl;
    return true;
}

void GRPCWorker::stop() {
    if (!running_.load()) {
        return;
    }

    running_.store(false);
    stop_.store(true);

    // Shutdown server
    if (server_) {
        server_->Shutdown();
        if (serverThread_.joinable()) {
            serverThread_.join();
        }
    }

    // Stop heartbeat thread
    if (hbThread_.joinable()) {
        hbThread_.join();
    }

    std::cout << "[GRPCWorker] Worker stopped" << std::endl;
}

bool GRPCWorker::doRegister() {
    try {
        httplib::Client cli(options_.controllerBase.c_str());
        
        json j;
        j["workerId"] = options_.workerId;
        j["nodeId"] = options_.nodeId;
        j["instanceId"] = options_.instanceId;
        j["appName"] = options_.appName;
        j["appVersion"] = options_.appVersion;
        
        // Parse gRPC address for registration
        std::string grpcHost = "127.0.0.1";
        int grpcPort = 18080;
        
        if (!options_.grpcAddress.empty()) {
            size_t colonPos = options_.grpcAddress.find(':');
            if (colonPos != std::string::npos) {
                grpcHost = options_.grpcAddress.substr(0, colonPos);
                grpcPort = std::stoi(options_.grpcAddress.substr(colonPos + 1));
            } else {
                grpcPort = std::stoi(options_.grpcAddress);
            }
        }
        
        j["grpcAddress"] = grpcHost + ":" + std::to_string(grpcPort);
        
        // Convert task names to array
        json tasksArray = json::array();
        for (const auto& kv : handlers_) {
            tasksArray.push_back(kv.first);
        }
        j["tasks"] = tasksArray;
        
        // Convert labels to JSON object
        json labelsObj = json::object();
        for (const auto& kv : options_.labels) {
            labelsObj[kv.first] = kv.second;
        }
        j["labels"] = labelsObj;
        
        auto r = cli.Post("/v1/embedded-workers/register", j.dump(), "application/json");
        bool success = r && r->status >= 200 && r->status < 300;
        
        if (success) {
            std::cout << "[GRPCWorker] Registered with controller successfully" << std::endl;
        } else {
            std::cerr << "[GRPCWorker] Registration failed, status: " << (r ? r->status : -1) << std::endl;
        }
        
        return success;
    } catch (const std::exception& e) {
        std::cerr << "[GRPCWorker] Registration error: " << e.what() << std::endl;
        return false;
    }
}

void GRPCWorker::doHeartbeat() {
    try {
        httplib::Client cli(options_.controllerBase.c_str());
        
        json j;
        j["workerId"] = options_.workerId;
        
        auto r = cli.Post("/v1/embedded-workers/heartbeat", j.dump(), "application/json");
        bool success = r && r->status >= 200 && r->status < 300;
        
        if (!success) {
            std::cerr << "[GRPCWorker] Heartbeat failed, status: " << (r ? r->status : -1) << std::endl;
        }
    } catch (const std::exception& e) {
        std::cerr << "[GRPCWorker] Heartbeat error: " << e.what() << std::endl;
    }
}

void GRPCWorker::heartbeatLoop() {
    while (!stop_.load()) {
        doHeartbeat();
        
        // Sleep for heartbeat interval
        for (int i = 0; i < options_.heartbeatSec * 10 && !stop_.load(); ++i) {
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
        }
    }
}

} // namespace plum
