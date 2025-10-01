SHELL := /bin/bash

.PHONY: controller agent agent-build agent-run demo ui ui-dev ui-build

controller:
	$(MAKE) -C controller build

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


