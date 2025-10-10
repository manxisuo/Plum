SHELL := /bin/bash

.PHONY: controller controller-run agent agent-cpp agent-run agent-run-multi agent-clean agent-help demo ui ui-dev ui-build proto proto-clean
.PHONY: sdk_cpp sdk_cpp_mirror sdk_cpp_echo_worker sdk_cpp_echo_worker-run

controller:
	$(MAKE) -C controller build

controller-run:
	./controller/bin/controller

# ============ Agent 构建 ============
agent:
	@echo "Building Go Agent..."
	@cd agent-go && go build -o plum-agent
	@echo "✅ Go Agent built: agent-go/plum-agent"

agent-cpp:
	@echo "Building C++ Agent..."
	@cmake -S agent -B agent/build -DCMAKE_BUILD_TYPE=Release
	@cmake --build agent/build --config Release -j
	@echo "✅ C++ Agent built: agent/build/plum_agent"

agent-clean:
	@echo "Cleaning agent build artifacts..."
	@rm -f agent-go/plum-agent
	@rm -rf agent/build
	@echo "✅ Agent artifacts cleaned"

# ============ Agent 运行 ============
agent-run:
	@echo "Starting Go Agent (nodeA)..."
	@AGENT_NODE_ID=nodeA CONTROLLER_BASE=http://127.0.0.1:8080 ./agent-go/plum-agent

agent-run%:
	@num=$(patsubst agent-run%,%,$@); \
	echo "Starting Go Agent (node$$num)..."; \
	AGENT_NODE_ID=node$$num CONTROLLER_BASE=http://127.0.0.1:8080 ./agent-go/plum-agent

agent-cpp-run:
	@echo "Starting C++ Agent (nodeA)..."
	@AGENT_NODE_ID=nodeA CONTROLLER_BASE=http://127.0.0.1:8080 ./agent/build/plum_agent

agent-cpp-run%:
	@num=$(patsubst agent-cpp-run%,%,$@); \
	echo "Starting C++ Agent (node$$num)..."; \
	AGENT_NODE_ID=node$$num CONTROLLER_BASE=http://127.0.0.1:8080 ./agent/build/plum_agent

# 运行多个Agent节点（后台）
agent-run-multi:
	@echo "Starting multiple Go Agents..."
	@mkdir -p logs
	@AGENT_NODE_ID=nodeA CONTROLLER_BASE=http://127.0.0.1:8080 ./agent-go/plum-agent > logs/agent-nodeA.log 2>&1 & echo "Started nodeA (PID: $$!)"
	@sleep 1
	@AGENT_NODE_ID=nodeB CONTROLLER_BASE=http://127.0.0.1:8080 ./agent-go/plum-agent > logs/agent-nodeB.log 2>&1 & echo "Started nodeB (PID: $$!)"
	@sleep 1
	@AGENT_NODE_ID=nodeC CONTROLLER_BASE=http://127.0.0.1:8080 ./agent-go/plum-agent > logs/agent-nodeC.log 2>&1 & echo "Started nodeC (PID: $$!)"
	@echo "✅ 3 agents started. Logs in logs/agent-*.log"
	@echo "   To stop: pkill -f plum-agent"

# Agent帮助信息
agent-help:
	@echo "Plum Agent Commands:"
	@echo ""
	@echo "  构建："
	@echo "    make agent              - 构建Go Agent（推荐）"
	@echo "    make agent-cpp          - 构建C++ Agent（备份）"
	@echo "    make agent-clean        - 清理编译产物"
	@echo ""
	@echo "  运行："
	@echo "    make agent-run          - 运行Go Agent (nodeA)"
	@echo "    make agent-runA         - 运行Go Agent (nodeA)"
	@echo "    make agent-runB         - 运行Go Agent (nodeB)"
	@echo "    make agent-runC         - 运行Go Agent (nodeC)"
	@echo "    make agent-run-multi    - 后台运行3个Go Agent (A/B/C)"
	@echo ""
	@echo "  C++ Agent（旧版）："
	@echo "    make agent-cpp-run      - 运行C++ Agent (nodeA)"
	@echo "    make agent-cpp-runA     - 运行C++ Agent (nodeA)"
	@echo ""
	@echo "  环境变量："
	@echo "    AGENT_NODE_ID           - 节点ID（默认：nodeA）"
	@echo "    CONTROLLER_BASE         - Controller地址（默认：http://127.0.0.1:8080）"
	@echo "    AGENT_DATA_DIR          - 数据目录（默认：/tmp/plum-agent）"
	@echo ""

# ============ Proto编译 ============
proto:
	$(MAKE) -C proto all

proto-clean:
	$(MAKE) -C proto clean

demo:
	@echo "1) start controller: make controller && ./controller/bin/controller" 
	@echo "2) run agent: make agent && make agent-run"
	@echo "3) create task: curl -s -XPOST http://127.0.0.1:8080/v1/tasks -H 'Content-Type: application/json' -d '{"name":"app1","artifactUrl":"http://127.0.0.1:8000/app1.zip","startCmd":"echo hello","replicas":{"nodeA":1}}' | jq ."

ui:
	cd ui && npm i --silent

ui-dev:
	cd ui && npm run dev

ui-build:
	cd ui && npm run build


# SDK C++ (library and examples)
sdk_cpp:
	cmake -S sdk/cpp -B sdk/cpp/build -DCMAKE_BUILD_TYPE=Release
	cmake --build sdk/cpp/build --config Release -j

# SDK C++ (使用GitHub镜像，适合中国网络)
sdk_cpp_mirror:
	cmake -S sdk/cpp -B sdk/cpp/build -DCMAKE_BUILD_TYPE=Release -DUSE_GITHUB_MIRROR=ON
	cmake --build sdk/cpp/build --config Release -j

# SDK C++ echo_worker
sdk_cpp_echo_worker:
	cmake -S sdk/cpp -B sdk/cpp/build -DCMAKE_BUILD_TYPE=Release
	cmake --build sdk/cpp/build --target echo_worker -j

sdk_cpp_echo_worker-run:
	./sdk/cpp/build/examples/echo_worker/echo_worker

# SDK C++ radar_sensor
sdk_cpp_radar_sensor:
	cmake -S sdk/cpp -B sdk/cpp/build -DCMAKE_BUILD_TYPE=Release
	cmake --build sdk/cpp/build --target radar_sensor -j

sdk_cpp_radar_sensor-run:
	RESOURCE_ID=radar-001 RESOURCE_NODE_ID=nodeA CONTROLLER_BASE=http://127.0.0.1:8080 ./sdk/cpp/build/examples/radar_sensor/radar_sensor

# SDK C++ grpc_echo_worker
sdk_cpp_grpc_echo_worker:
	cmake -S sdk/cpp -B sdk/cpp/build -DCMAKE_BUILD_TYPE=Release
	cmake --build sdk/cpp/build --target grpc_echo_worker -j

sdk_cpp_grpc_echo_worker-run:
	PLUM_INSTANCE_ID=grpc-instance-001 PLUM_APP_NAME=grpc-echo-app PLUM_APP_VERSION=v2.0.0 WORKER_ID=grpc-echo-1 WORKER_NODE_ID=nodeA CONTROLLER_BASE=http://127.0.0.1:8080 GRPC_ADDRESS=0.0.0.0:18082 ./sdk/cpp/build/examples/grpc_echo_worker/grpc_echo_worker

# 优雅停止agent
stop-agent:
	@chmod +x tools/stop_agent.sh
	@tools/stop_agent.sh


