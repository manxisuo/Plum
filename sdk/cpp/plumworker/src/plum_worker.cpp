#include "plum_worker.hpp"
#include <httplib.h>
#include <nlohmann/json.hpp>
#include <iostream>
#include <chrono>

using json = nlohmann::json;

namespace plumworker {

static std::string get_local_ip() {
  return "127.0.0.1"; // MVP
}

Worker::Worker(const WorkerOptions& opt) : options_(opt) {}
Worker::~Worker() { stop(); }

void Worker::registerTask(const std::string& name, TaskHandler handler) {
  handlers_[name] = std::move(handler);
}

bool Worker::start() {
  if (!startHttp()) return false;
  if (!doRegister()) return false;
  hbThread_ = std::thread([this]{ heartbeatLoop(); });
  return true;
}

void Worker::stop() {
  stop_.store(true);
  if (hbThread_.joinable()) hbThread_.join();
}

bool Worker::startHttp() {
  // Pick port or use provided
  int port = options_.httpPort > 0 ? options_.httpPort : 0;
  auto svr = new httplib::Server();
  svr->Post("/run", [this](const httplib::Request& req, httplib::Response& res){
    try {
      auto j = json::parse(req.body);
      std::string name = j.value("name", "");
      std::string taskId = j.value("taskId", "");
      json payload = j.value("payload", json::object());
      auto it = handlers_.find(name);
      if (it == handlers_.end()) { res.status = 404; res.set_content("{}", "application/json"); return; }
      std::string out = it->second(taskId, name, payload.dump());
      res.status = 200; res.set_content(out.empty()?"{}":out, "application/json");
    } catch (...) { res.status = 400; res.set_content("{}", "application/json"); }
  });
  svr->set_exception_handler([](const httplib::Request&, httplib::Response& res, std::exception_ptr){ res.status=500; res.set_content("{}","application/json"); });
  // Run http in background
  std::thread([this, svr, port]{
    int p = port;
    if (p == 0) { p = 0; }
    svr->listen("0.0.0.0", p);
  }).detach();
  // construct URL (assume chosen port)
  // NOTE: cpp-httplib doesn't expose actual port when 0; MVP要求显式指定端口
  if (options_.httpPort == 0) {
    std::cerr << "[plumworker] please set WorkerOptions.httpPort to a fixed port" << std::endl;
    return false;
  }
  httpURL_ = std::string("http://") + get_local_ip() + ":" + std::to_string(options_.httpPort);
  return true;
}

bool Worker::doRegister() {
  try {
    httplib::Client cli(options_.controllerBase.c_str());
    json j;
    j["workerId"] = options_.workerId;
    j["nodeId"] = options_.nodeId;
    j["url"] = httpURL_ + "/run";
    json arr = json::array();
    for (auto& kv : handlers_) arr.push_back(kv.first);
    j["tasks"] = arr;
    
    // 使用配置的标签，而不是空对象
    json labelsJson = json::object();
    for (const auto& kv : options_.labels) {
      labelsJson[kv.first] = kv.second;
    }
    j["labels"] = labelsJson;
    
    j["capacity"] = options_.capacity;
    auto r = cli.Post("/v1/workers/register", j.dump(), "application/json");
    return r && r->status >= 200 && r->status < 300;
  } catch (...) { return false; }
}

bool Worker::doHeartbeat() {
  try {
    httplib::Client cli(options_.controllerBase.c_str());
    json j; j["workerId"] = options_.workerId; j["capacity"] = options_.capacity;
    auto r = cli.Post("/v1/workers/heartbeat", j.dump(), "application/json");
    return r && r->status >= 200 && r->status < 300;
  } catch (...) { return false; }
}

void Worker::heartbeatLoop() {
  using namespace std::chrono_literals;
  while (!stop_.load()) {
    (void)doHeartbeat();
    std::this_thread::sleep_for(std::chrono::seconds(options_.heartbeatSec));
  }
}

}


