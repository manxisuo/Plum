#pragma once

#include <functional>
#include <map>
#include <string>
#include <thread>
#include <atomic>

namespace plumworker {

using TaskHandler = std::function<std::string(const std::string& taskId, const std::string& name, const std::string& payloadJson)>;

struct WorkerOptions {
  std::string controllerBase;
  std::string workerId;
  std::string nodeId;
  std::map<std::string, std::string> labels; // 添加标签支持
  int capacity{1};
  int heartbeatSec{5};
  int httpPort{0}; // 0 means random
};

class Worker {
public:
  explicit Worker(const WorkerOptions& opt);
  ~Worker();

  void registerTask(const std::string& name, TaskHandler handler);
  bool start();
  void stop();

private:
  WorkerOptions options_;
  std::map<std::string, TaskHandler> handlers_;
  std::atomic<bool> stop_{false};
  std::thread hbThread_;
  std::string httpURL_;

  void heartbeatLoop();
  bool doRegister();
  bool doHeartbeat();
  bool startHttp();
};

}


