#include "http_client.hpp"
#include "reconciler.hpp"
#include <chrono>
#include <cstdlib>
#include <iostream>
#include <string>
#include <thread>
#include <vector>
#include <atomic>
#include <csignal>

static std::string getenv_or(const char* key, const char* defv) {
	const char* v = std::getenv(key);
	return v ? std::string(v) : std::string(defv);
}

static std::atomic<bool> g_stop{false};

static void handle_sigint(int) {
    g_stop.store(true);
}

int main() {
	std::string node_id = getenv_or("AGENT_NODE_ID", "nodeA");
	std::string controller = getenv_or("CONTROLLER_BASE", "http://127.0.0.1:8080");
	HttpClient http;
    Reconciler reconciler(std::string(getenv_or("AGENT_DATA_DIR", "/tmp/plum-agent")) + "/" + node_id, &http, controller);

    std::signal(SIGINT, handle_sigint);
    std::signal(SIGTERM, handle_sigint);

    while (!g_stop.load()) {
		// Heartbeat (Register)
		std::string hb = std::string("{\"nodeId\":\"") + node_id + "\",\"ip\":\"127.0.0.1\"}";
		auto resp = http.post_json(controller + "/v1/nodes/heartbeat", hb);
		if (resp.status_code != 200) {
			std::cerr << "heartbeat failed" << std::endl;
		}

		// Fetch assignments
		auto asg = http.get(controller + "/v1/assignments?nodeId=" + node_id);
		if (asg.status_code == 200 && !asg.body.empty()) {
			// very simple parse: not introducing json lib, search tokens
            std::vector<AssignmentItem> items;
			std::string s = asg.body;
			// naive parsing: find instanceId, artifactUrl, startCmd occurrences
			size_t pos = 0;
            while (true) {
                size_t i1 = s.find("\"instanceId\":\"", pos); if (i1 == std::string::npos) break; i1 += 14; size_t e1 = s.find("\"", i1);
                size_t i2 = s.find("\"artifactUrl\":\"", e1); if (i2 == std::string::npos) break; i2 += 15; size_t e2 = s.find("\"", i2);
                // desired 位于 instanceId 之后
                size_t ides = s.find("\"desired\":\"", e1); if (ides == std::string::npos) break; ides += 11; size_t edes = s.find("\"", ides);
                std::string inst = s.substr(i1, e1-i1);
                std::string desired = s.substr(ides, edes-ides);
                std::string art = s.substr(i2, e2-i2);
                std::string cmd;
                // startCmd 可选
                size_t i3 = s.find("\"startCmd\":\"", e2);
                if (i3 != std::string::npos) { i3 += 12; size_t e3 = s.find("\"", i3); cmd = s.substr(i3, e3-i3); pos = e3 + 1; }
                else { cmd.clear(); pos = e2 + 1; }
            // Normalize artifact URL: support absolute, /relative, and bare paths
            if (!(art.rfind("http://", 0) == 0 || art.rfind("https://", 0) == 0)) {
                if (!art.empty() && art[0] == '/') art = controller + art;
                else art = controller + "/" + art;
            }
                if (desired == "Running") {
                    AssignmentItem it{ inst, art, cmd };
                    items.push_back(it);
                }
			}
			reconciler.sync(items);
		}

        std::this_thread::sleep_for(std::chrono::seconds(5));
	}
    // graceful stop: stop all child instances we started
    reconciler.sync({});
    reconciler.stop_all_sync();
	return 0;
}


