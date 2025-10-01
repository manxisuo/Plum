#pragma once
#include <string>
#include <unordered_map>
#include <vector>

class HttpClient; // fwd

struct AssignmentItem {
	std::string instanceId;
	std::string artifactUrl;
	std::string startCmd;
};

class Reconciler {
public:
	Reconciler(const std::string& base_dir, HttpClient* http, const std::string& controller_base);
	void sync(const std::vector<AssignmentItem>& items);
    void stop_all_sync();
    void register_services(const std::string& instance_id, const std::string& node_id, const std::string& ip);
    void heartbeat_services(const std::string& instance_id);
	void delete_services(const std::string& instance_id);

private:
	std::string base_dir_;
	HttpClient* http_ {nullptr};
	std::string controller_;
    struct InstanceProcState { int pid; long stop_sent_ts; };
    std::unordered_map<std::string, InstanceProcState> instance_state_;

	void ensure_running(const AssignmentItem& it);
	void ensure_stopped_except(const std::unordered_map<std::string,bool>& keep);
	void reap_exited();
	void post_status(const std::string& instance_id, const std::string& phase, int exit_code, bool healthy);
};


