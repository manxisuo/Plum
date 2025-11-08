SHELL := /bin/bash

.PHONY: controller controller-run agent agent-run agent-run-multi agent-clean agent-help demo ui ui-dev ui-build proto proto-clean
.PHONY: sdk_cpp sdk_cpp_mirror sdk_cpp_echo_worker sdk_cpp_echo_worker-run
.PHONY: plumclient service_client_example service_client_example-run
.PHONY: examples_worker_demo examples_worker_demo-pkg
.PHONY: help stop-agent

controller:
	$(MAKE) -C controller build

controller-run:
	./controller/bin/controller

# ============ Agent 构建 ============
agent:
	@echo "Building Go Agent..."
	@cd agent-go && go build -o plum-agent
	@echo "✅ Go Agent built: agent-go/plum-agent"

agent-clean:
	@echo "Cleaning agent build artifacts..."
	@rm -f agent-go/plum-agent
	@rm -f agent-go/agent
	@echo "✅ Agent artifacts cleaned"

# ============ Agent 运行 ============
agent-run:
	@echo "Starting Go Agent..."
    #@AGENT_NODE_ID=nodeA ./agent-go/plum-agent
	@./agent-go/plum-agent

agent-run%:
	@num=$(patsubst agent-run%,%,$@); \
	echo "Starting Go Agent (node$$num)..."; \
	AGENT_NODE_ID=node$$num ./agent-go/plum-agent

# 运行多个Agent节点（后台）
agent-run-multi:
	@echo "Starting multiple Go Agents..."
	@mkdir -p logs
	@AGENT_NODE_ID=nodeA ./agent-go/plum-agent > logs/agent-nodeA.log 2>&1 & echo "Started nodeA (PID: $$!)"
	@sleep 1
	@AGENT_NODE_ID=nodeB ./agent-go/plum-agent > logs/agent-nodeB.log 2>&1 & echo "Started nodeB (PID: $$!)"
	@sleep 1
	@AGENT_NODE_ID=nodeC ./agent-go/plum-agent > logs/agent-nodeC.log 2>&1 & echo "Started nodeC (PID: $$!)"
	@echo "✅ 3 agents started. Logs in logs/agent-*.log"
	@echo "   To stop: pkill -f plum-agent"

# Agent帮助信息
agent-help:
	@echo "Plum Agent Commands:"
	@echo ""
	@echo "  构建："
	@echo "    make agent              - 构建Go Agent"
	@echo "    make agent-clean        - 清理编译产物"
	@echo ""
	@echo "  运行："
	@echo "    make agent-run          - 运行Go Agent (nodeA)"
	@echo "    make agent-runA         - 运行Go Agent (nodeA)"
	@echo "    make agent-runB         - 运行Go Agent (nodeB)"
	@echo "    make agent-runC         - 运行Go Agent (nodeC)"
	@echo "    make agent-run-multi    - 后台运行3个Go Agent (A/B/C)"
	@echo ""
	@echo "  环境变量："
	@echo "    AGENT_NODE_ID           - 节点ID（默认：nodeA）"
	@echo "    CONTROLLER_BASE         - Controller地址（默认：http://plum-controller:8080）"
	@echo "    AGENT_IP                - Agent对外通告的IP（默认：127.0.0.1）"
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
	@if [ ! -d "ui/node_modules" ]; then \
		echo "📦 node_modules 不存在，开始安装依赖..."; \
		cd ui && npm install --include=optional --silent; \
	else \
		echo "✅ node_modules 已存在，跳过安装"; \
		echo "   💡 可用选项:"; \
		echo "      make ui-update    - 增量更新依赖（推荐）"; \
		echo "      make ui-reinstall - 完全重新安装"; \
		echo "      make ui-clean     - 仅删除node_modules"; \
	fi

ui-update:
	@echo "📦 更新UI依赖（增量安装）..."
	cd ui && npm install --include=optional --silent

ui-reinstall:
	@echo "🔄 完全重新安装UI依赖..."
	@if [ -d "ui/node_modules" ]; then \
		echo "🗑️  删除现有 node_modules..."; \
		rm -rf ui/node_modules; \
	fi
	cd ui && npm install --include=optional --silent

ui-clean:
	@echo "🗑️  清理UI依赖..."
	@if [ -d "ui/node_modules" ]; then \
		rm -rf ui/node_modules; \
		echo "✅ node_modules 已删除"; \
	else \
		echo "⚠️  node_modules 目录不存在"; \
	fi

ui-dev:
	cd ui && npm run dev

ui-build:
	cd ui && npm run build


# SDK C++ (library and examples)
sdk_cpp:
	cmake -S sdk/cpp -B sdk/cpp/build -DCMAKE_BUILD_TYPE=Release
	cmake --build sdk/cpp/build --config Release -j

# SDK C++ (离线模式，不使用网络下载依赖)
sdk_cpp_offline:
	cmake -S sdk/cpp -B sdk/cpp/build -DCMAKE_BUILD_TYPE=Release -DUSE_OFFLINE_DEPS=ON
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
    #RESOURCE_ID=radar-001 RESOURCE_NODE_ID=nodeA ./sdk/cpp/build/examples/radar_sensor/radar_sensor
    #RESOURCE_ID=radar-001  ./sdk/cpp/build/examples/radar_sensor/radar_sensor
	./sdk/cpp/build/examples/radar_sensor/radar_sensor

# ============ Plum Client 库 ============
plumclient:
	@echo "Building Plum Client library..."
	@cmake -S sdk/cpp -B sdk/cpp/build -DCMAKE_BUILD_TYPE=Release
	@cmake --build sdk/cpp/build --target plumclient -j
	@echo "✅ Plum Client library built: sdk/cpp/build/plumclient/libplumclient.so"

# ============ Service Client Example ============
service_client_example:
	@echo "Building Service Client Example..."
	@cmake -S sdk/cpp -B sdk/cpp/build -DCMAKE_BUILD_TYPE=Release
	@cmake --build sdk/cpp/build --target service_client_example -j
	@echo "✅ Service Client Example built: sdk/cpp/build/examples/service_client_example/service_client_example"

service_client_example-run:
	@echo "Running Service Client Example..."
	@echo "⚠️  确保Controller正在运行: make controller && make controller-run"
	@echo "⚠️  确保至少有一个Agent正在运行: make agent && make agent-run"
	@echo ""
	@echo "启动Service Client Example..."
	@./sdk/cpp/build/examples/service_client_example/service_client_example

# 优雅停止agent
stop-agent:
	@chmod +x tools/stop_agent.sh
	@tools/stop_agent.sh

# ============ Examples ============
# Worker Demo 构建
examples_worker_demo:
	@echo "Building Worker Demo..."
	@if [ ! -f "sdk/cpp/grpc/proto/task_service.pb.cc" ]; then \
		echo "❌ Proto文件不存在，请先运行: make proto"; \
		exit 1; \
	fi
	@cd examples/worker-demo && \
		mkdir -p build && \
		cd build && \
		cmake .. && \
		make
	@echo "✅ Worker Demo built: examples/worker-demo/build/worker-demo"

# Worker Demo 打包
examples_worker_demo-pkg: examples_worker_demo
	@echo "Packaging Worker Demo..."
	@cd examples/worker-demo && \
		VERSION=$$(grep "^version=" meta.ini | cut -d'=' -f2 | tr -d ' ' || echo "unknown"); \
		echo "Version: $$VERSION"; \
		mkdir -p package && \
		cp build/worker-demo package/ && \
		cp start.sh package/ && \
		cp meta.ini package/ && \
		chmod +x package/start.sh && \
		chmod +x package/worker-demo && \
		cd package && \
		zip -q -r ../worker-demo-$$VERSION.zip . && \
		cd .. && \
		rm -rf package
	@echo "✅ Package created: examples/worker-demo/worker-demo-$$(grep '^version=' examples/worker-demo/meta.ini | cut -d'=' -f2 | tr -d ' ').zip"
	@ls -lh examples/worker-demo/worker-demo-*.zip | tail -1

# ============ 帮助信息 ============
help:
	@echo "Plum 项目构建和运行命令:"
	@echo ""
	@echo "  🎯 核心组件:"
	@echo "    make controller              - 构建Controller"
	@echo "    make controller-run          - 运行Controller"
	@echo "    make agent                   - 构建Go Agent"
	@echo "    make agent-run               - 运行Go Agent (nodeA)"
	@echo "    make agent-runA/B/C         - 运行指定节点Agent"
	@echo "    make agent-run-multi         - 后台运行3个Agent"
	@echo ""
	@echo "  📚 C++ SDK:"
	@echo "    make sdk_cpp                 - 构建所有C++ SDK"
	@echo "    make sdk_cpp_offline         - 离线模式构建C++ SDK"
	@echo "    make sdk_cpp_mirror          - 使用镜像构建C++ SDK"
	@echo ""
	@echo "  🔧 C++ 示例程序:"
	@echo "    make sdk_cpp_echo_worker     - 构建echo_worker示例"
	@echo "    make sdk_cpp_echo_worker-run - 运行echo_worker示例"
	@echo "    make sdk_cpp_radar_sensor    - 构建radar_sensor示例"
	@echo "    make sdk_cpp_radar_sensor-run- 运行radar_sensor示例"
	@echo ""
	@echo "  📦 示例应用:"
	@echo "    make examples_worker_demo    - 构建worker-demo"
	@echo "    make examples_worker_demo-pkg - 打包worker-demo（包含meta.ini和start.sh）"
	@echo ""
	@echo "  🌐 Plum Client 库:"
	@echo "    make plumclient              - 构建Plum Client库"
	@echo "    make plumclient-run          - 安装Plum Client库到系统"
	@echo "    make service_client_example  - 构建服务客户端示例"
	@echo "    make service_client_example-run - 运行服务客户端示例"
	@echo ""
	@echo "  🎨 UI:"
	@echo "    make ui                      - 安装UI依赖"
	@echo "    make ui-dev                  - 开发模式运行UI"
	@echo "    make ui-build                - 构建UI"
	@echo ""
	@echo "  🧹 清理:"
	@echo "    make agent-clean             - 清理Agent构建产物"
	@echo "    make proto-clean             - 清理Proto构建产物"
	@echo "    make ui-clean                - 清理UI依赖"
	@echo ""
	@echo "  📖 其他:"
	@echo "    make demo                    - 显示演示步骤"
	@echo "    make agent-help              - 显示Agent详细帮助"
	@echo "    make stop-agent              - 停止所有Agent"
	@echo ""
