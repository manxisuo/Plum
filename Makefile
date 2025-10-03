SHELL := /bin/bash

.PHONY: controller controller-run agent agent-build agent-run demo ui ui-dev ui-build sdk_cpp sdk_cpp_echo_worker sdk_cpp_echo_worker-run

controller:
	$(MAKE) -C controller build

controller-run:
	./controller/bin/controller

agent:
	cmake -S agent -B agent/build -DCMAKE_BUILD_TYPE=Release
	cmake --build agent/build --config Release -j

agent-run:
	AGENT_NODE_ID=nodeA CONTROLLER_BASE=http://127.0.0.1:8080 ./agent/build/plum_agent

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

# SDK C++ echo_worker
sdk_cpp_echo_worker:
	cmake -S sdk/cpp -B sdk/cpp/build -DCMAKE_BUILD_TYPE=Release
	cmake --build sdk/cpp/build --target echo_worker -j

sdk_cpp_echo_worker-run:
	./sdk/cpp/build/examples/echo_worker/echo_worker


