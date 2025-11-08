#include "plum_worker.hpp"
#include <nlohmann/json.hpp>
#include <iostream>
#include <cstdlib>

using namespace plumworker;
using json = nlohmann::json;

static std::string getenv_or(const char* k, const char* d) {
  const char* v = std::getenv(k); return v?std::string(v):std::string(d);
}

int main() {
  WorkerOptions opt;
  opt.controllerBase = getenv_or("CONTROLLER_BASE", "http://plum-controller:8080");
  opt.workerId = getenv_or("WORKER_ID", "cpp-echo-1");
  opt.nodeId = getenv_or("WORKER_NODE_ID", "nodeA");
  opt.capacity = 4;
  opt.heartbeatSec = 5;
  opt.httpPort = 18081; // fixed port for MVP
  
  // 设置Worker标签
  opt.labels["appName"] = "myApp";
  opt.labels["deploymentId"] = "deploy-123";
  opt.labels["version"] = "v1.2.0";

  Worker w(opt);
  w.registerTask("my.task.echo", [](const std::string& taskId, const std::string& name, const std::string& payload){
    std::cout << "my.task.echo: " << payload << std::endl;
    json in = json::parse(payload.empty()?"{}":payload);
    json out; out["taskId"] = taskId; out["name"] = name; out["echo"] = in;
    return out.dump();
  });
  w.registerTask("builtin.sleep", [](const std::string& taskId, const std::string& name, const std::string& payload){
    json in = json::parse(payload.empty()?"{}":payload);
    double seconds = in.value("seconds", 1.0);
    std::this_thread::sleep_for(std::chrono::milliseconds((int)(seconds*1000)));
    json out; out["ok"] = true; out["slept"] = seconds; return out.dump();
  });

  if (!w.start()) { std::cerr << "failed to start worker" << std::endl; return 1; }
  std::cout << "cpp echo worker started" << std::endl;
  // block
  while (true) std::this_thread::sleep_for(std::chrono::seconds(60));
  return 0;
}


