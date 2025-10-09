SHELL := /bin/bash

# Go工具链路径
GOBIN := $(shell go env GOPATH)/bin
PROTOC_GEN_GO := $(GOBIN)/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(GOBIN)/protoc-gen-go-grpc

.PHONY: all clean check-tools

all: check-tools generate-go generate-cpp
	@echo "✅ Proto generation complete"

# 检查protoc是否安装
check-tools:
	@command -v protoc > /dev/null 2>&1 || { echo "❌ protoc not found. Install: sudo apt install protobuf-compiler"; exit 1; }
	@echo "✓ protoc version: $$(protoc --version)"

# 检查并安装Go插件
$(PROTOC_GEN_GO):
	@echo "→ Installing protoc-gen-go..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

$(PROTOC_GEN_GO_GRPC):
	@echo "→ Installing protoc-gen-go-grpc..."
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 生成Go代码
generate-go: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
	@echo "📦 Generating Go code..."
	@cd .. && protoc -I=proto \
		--go_out=. --go_opt=module=github.com/manxisuo/Plum/controller \
		--go-grpc_out=. --go-grpc_opt=module=github.com/manxisuo/Plum/controller \
		--plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
		--plugin=protoc-gen-go-grpc=$(PROTOC_GEN_GO_GRPC) \
		proto/task_service.proto
	@echo "✅ Go: controller/plum/proto/"

# 生成C++代码（在父目录执行）
generate-cpp:
	@if command -v grpc_cpp_plugin > /dev/null 2>&1; then \
		echo "📦 Generating C++ code..."; \
		mkdir -p ../sdk/cpp/grpc/proto; \
		cd .. && protoc -I=proto \
			--cpp_out=./sdk/cpp/grpc/proto \
			--grpc_out=./sdk/cpp/grpc/proto \
			--plugin=protoc-gen-grpc=$$(which grpc_cpp_plugin) \
			proto/task_service.proto; \
		echo "✅ C++: sdk/cpp/grpc/proto/"; \
	else \
		echo "⚠️  grpc_cpp_plugin not found, C++ skipped"; \
		echo "   Install: sudo apt install libgrpc++-dev protobuf-compiler-grpc"; \
	fi

# 清理生成的代码
clean:
	@echo "Cleaning generated proto files..."
	@rm -f ../controller/plum/proto/*.pb.go
	@rm -f ../controller/*.pb.go 2>/dev/null || true
	@rm -f ../sdk/cpp/grpc/proto/*.pb.{h,cc}
	@rm -rf ../sdk/cpp/grpc/proto/proto 2>/dev/null || true
	@echo "✅ Cleaned"

