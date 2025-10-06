#pragma once

#include <string>
#include <map>
#include <functional>
#include <memory>
#include <thread>
#include <atomic>
#include <grpcpp/grpcpp.h>
#include "../grpc/proto/task_service.grpc.pb.h"

namespace plum {

using TaskHandler = std::function<std::string(const std::string&)>;

struct GRPCWorkerOptions {
    std::string controllerBase;
    std::string workerId;
    std::string nodeId;
    std::string instanceId;  // from environment PLUM_INSTANCE_ID
    std::string appName;     // from environment PLUM_APP_NAME
    std::string appVersion;  // from environment PLUM_APP_VERSION
    std::map<std::string, std::string> labels;
    std::string grpcAddress; // host:port for gRPC server
    int heartbeatSec{5};
};

class TaskServiceImpl final : public task::TaskService::Service {
public:
    TaskServiceImpl(const std::map<std::string, TaskHandler>& handlers, const std::string& workerId)
        : handlers_(handlers), workerId_(workerId) {}

    grpc::Status ExecuteTask(grpc::ServerContext* context, const task::TaskRequest* request,
                            task::TaskResponse* response) override;

    grpc::Status HealthCheck(grpc::ServerContext* context, const task::HealthRequest* request,
                            task::HealthResponse* response) override;

private:
    const std::map<std::string, TaskHandler>& handlers_;
    std::string workerId_;
};

class GRPCWorker {
public:
    explicit GRPCWorker(const GRPCWorkerOptions& options);
    ~GRPCWorker();

    // Register task handler
    void registerTask(const std::string& name, TaskHandler handler);

    // Start the worker (starts gRPC server and registers with controller)
    bool start();

    // Stop the worker
    void stop();

    // Check if worker is running
    bool isRunning() const { return running_.load(); }

private:
    bool doRegister();
    void doHeartbeat();
    void heartbeatLoop();

    GRPCWorkerOptions options_;
    std::map<std::string, TaskHandler> handlers_;
    std::unique_ptr<TaskServiceImpl> service_;
    std::unique_ptr<grpc::Server> server_;
    std::thread serverThread_;
    std::thread hbThread_;
    std::atomic<bool> running_{false};
    std::atomic<bool> stop_{false};
};

} // namespace plum
