#include "reconciler.hpp"
#include "fs_utils.hpp"
#include "http_client.hpp"
#include <cstdio>
#include <filesystem>
#include <sstream>
#include <unistd.h>
#include <sys/wait.h>
#include <sys/stat.h>
#include <ctime>
#include <iostream>
#include <signal.h>
#include <thread>
#include <chrono>

Reconciler::Reconciler(const std::string& base_dir, HttpClient* http, const std::string& controller_base)
    : base_dir_(base_dir), http_(http), controller_(controller_base) {
	ensure_dir(base_dir_);
}

void Reconciler::sync(const std::vector<AssignmentItem>& items) {
	std::unordered_map<std::string,bool> keep;
	for (const auto& it : items) keep[it.instanceId] = true;
	// Order: reap exited -> stop extras -> start missing
	reap_exited();
	ensure_stopped_except(keep);
	for (const auto& it : items) ensure_running(it);
}

static std::string join(const std::string& a, const std::string& b) { return (std::filesystem::path(a) / b).string(); }
static std::string ltrim_commas_spaces(std::string s) {
    size_t i = 0;
    while (i < s.size() && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\n' || s[i] == ',')) ++i;
    if (i > 0) s.erase(0, i);
    return s;
}

void Reconciler::ensure_running(const AssignmentItem& it) {
    auto sit = instance_state_.find(it.instanceId);
    if (sit != instance_state_.end()) {
        // If process still alive, nothing to do
        if (kill(sit->second.pid, 0) == 0) return;
        // Stale record, cleanup
        instance_state_.erase(sit);
    }
	std::string inst_dir = join(base_dir_, it.instanceId);
	ensure_dir(inst_dir);
	std::string zip_path = join(inst_dir, "pkg.zip");
	// download
    if (!file_exists(zip_path)) {
        auto resp = http_->get(it.artifactUrl);
        if (resp.status_code != 200) {
            std::cerr << "download failed status=" << resp.status_code << " url=" << it.artifactUrl << "\n";
            return;
        }
        if (resp.body.empty()) {
            std::cerr << "download empty body url=" << it.artifactUrl << "\n";
            return;
        }
        FILE* fp = std::fopen(zip_path.c_str(), "wb");
        if (!fp) { std::perror("fopen zip_path"); return; }
        size_t wn = fwrite(resp.body.data(), 1, resp.body.size(), fp);
        std::fclose(fp);
        if (wn != resp.body.size()) { std::cerr << "write zip truncated: " << wn << "/" << resp.body.size() << "\n"; return; }
        std::cerr << "saved artifact to " << zip_path << " size=" << wn << "\n";
    }
	// unzip
	std::string app_dir = join(inst_dir, "app");
	ensure_dir(app_dir);
    if (!file_exists(join(app_dir, "start.sh"))) {
        if (!unzip_zip(zip_path, app_dir)) { std::cerr << "unzip failed; ensure 'unzip' is installed. zip=" << zip_path << "\n"; return; }
    }
	// start
    std::string sh = join(app_dir, "start.sh");
    ::chmod(sh.c_str(), 0755);
    std::string cmdline = it.startCmd;
    cmdline = ltrim_commas_spaces(cmdline);
    if (cmdline.empty()) cmdline = "./start.sh";
    std::string full_cmd = "cd '" + app_dir + "' && " + cmdline;
    std::cerr << "exec: " << full_cmd << "\n";
    pid_t pid = fork();
	if (pid == 0) {
		// child
        // 新建会话/进程组，便于组信号终止
        (void)setsid();
        
        // 设置应用相关的环境变量
        if (!it.appName.empty()) {
            setenv("PLUM_APP_NAME", it.appName.c_str(), 1);
        }
        if (!it.appVersion.empty()) {
            setenv("PLUM_APP_VERSION", it.appVersion.c_str(), 1);
        }
        setenv("PLUM_INSTANCE_ID", it.instanceId.c_str(), 1);
        
        execl("/bin/sh", "sh", "-c", full_cmd.c_str(), (char*)nullptr);
		std::perror("exec");
		_Exit(127);
	} else if (pid > 0) {
		instance_state_[it.instanceId] = { pid, 0 };
        post_status(it.instanceId, "Running", 0, true);
	}
}

void Reconciler::ensure_stopped_except(const std::unordered_map<std::string,bool>& keep) {
    long now = (long)std::time(nullptr);
	for (auto it = instance_state_.begin(); it != instance_state_.end(); ) {
		if (!keep.count(it->first)) {
            if (it->second.stop_sent_ts == 0) {
                // 发送到进程组
                kill(-it->second.pid, SIGTERM);
				it->second.stop_sent_ts = now;
				++it;
			} else if (now - it->second.stop_sent_ts >= 5) {
                kill(-it->second.pid, SIGKILL);
				int status=0; (void)waitpid(it->second.pid, &status, WNOHANG);
                post_status(it->first, "Stopped", 0, true);
                // remove service endpoints for this instance
                delete_services(it->first);
				it = instance_state_.erase(it);
			} else {
				++it;
			}
		} else {
			++it;
		}
	}
}

void Reconciler::reap_exited() {
    for (auto it = instance_state_.begin(); it != instance_state_.end(); ) {
        int status = 0;
        pid_t r = waitpid(it->second.pid, &status, WNOHANG);
        if (r == it->second.pid) {
            if (it->second.stop_sent_ts > 0) {
                // 是我们请求的停止
                post_status(it->first, "Stopped", 0, true);
            } else {
                int code = WIFEXITED(status) ? WEXITSTATUS(status) : -1;
                bool ok = (code == 0);
                post_status(it->first, ok ? "Exited" : "Failed", code, ok);
            }
            it = instance_state_.erase(it);
        } else {
            ++it;
        }
    }
}

void Reconciler::post_status(const std::string& instance_id, const std::string& phase, int exit_code, bool healthy) {
    if (!http_) return;
    long ts = (long) (std::time(nullptr));
    std::string body = std::string("{\"instanceId\":\"") + instance_id + "\",\"phase\":\"" + phase + "\",\"exitCode\":" + std::to_string(exit_code) + ",\"healthy\":" + (healthy?"true":"false") + ",\"tsUnix\":" + std::to_string(ts) + "}";
    (void)http_->post_json(controller_ + "/v1/instances/status", body);
}

// Minimal service registration: read meta.ini if present under app_dir and register endpoints
void Reconciler::register_services(const std::string& instance_id, const std::string& node_id, const std::string& ip) {
    if (!http_) return;
    // meta.ini path: base_dir_/instance_id/app/meta.ini
    std::string meta = (std::filesystem::path(base_dir_) / instance_id / "app" / "meta.ini").string();
    if (!file_exists(meta)) return;
    // Very simple parse: look for lines like service=<name>:<protocol>:<port> or multiple lines
    FILE* fp = std::fopen(meta.c_str(), "r"); if (!fp) return;
    char buf[512];
    std::vector<std::tuple<std::string,std::string,int>> entries;
    while (std::fgets(buf, sizeof(buf), fp)) {
        std::string line(buf);
        if (line.rfind("service=", 0) == 0) {
            std::string v = line.substr(8);
            // trim
            while (!v.empty() && (v.back()=='\n' || v.back()=='\r' || v.back()==' ')) v.pop_back();
            size_t p1 = v.find(':'); size_t p2 = v.find(':', p1==std::string::npos?0:p1+1);
            if (p1!=std::string::npos && p2!=std::string::npos) {
                std::string name = v.substr(0, p1);
                std::string proto = v.substr(p1+1, p2-p1-1);
                int port = std::atoi(v.substr(p2+1).c_str());
                if (!name.empty() && port>0) entries.emplace_back(name, proto, port);
            }
        }
    }
    std::fclose(fp);
    if (entries.empty()) return;
    // build JSON
    std::string body = std::string("{") + "\"instanceId\":\"" + instance_id + "\",\"nodeId\":\"" + node_id + "\",\"ip\":\"" + ip + "\",\"endpoints\":[";
    bool first = true;
    for (auto& t : entries) {
        if (!first) body += ","; first=false;
        body += "{\"serviceName\":\"" + std::get<0>(t) + "\",\"protocol\":\"" + std::get<1>(t) + "\",\"port\":" + std::to_string(std::get<2>(t)) + "}";
    }
    body += "]}";
    (void)http_->post_json(controller_ + "/v1/services/register", body);
}

void Reconciler::heartbeat_services(const std::string& instance_id) {
    if (!http_) return;
    std::string body = std::string("{\"instanceId\":\"") + instance_id + "\"}";
    (void)http_->post_json(controller_ + "/v1/services/heartbeat", body);
}

void Reconciler::delete_services(const std::string& instance_id) {
    if (!http_) return;
    std::string url = controller_ + "/v1/services?instanceId=" + instance_id;
    (void)http_->del(url);
}

void Reconciler::stop_all_sync() {
    std::unordered_map<std::string,bool> empty;
    // 最长等待 ~7s：先发 TERM，5s 后升级为 KILL，再清理
    for (int i = 0; i < 70; ++i) {
        ensure_stopped_except(empty);
        reap_exited();
        if (instance_state_.empty()) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }
    // 最后一遍，确保残留强制清理
    ensure_stopped_except(empty);
    reap_exited();
}
