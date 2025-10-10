SHELL := /bin/bash

# Go工具链路径
GOBIN := $(shell go env GOPATH)/bin

.PHONY: all clean generate-go generate-cpp

all: generate-go generate-cpp
	@echo "✅ Proto generation complete"

# 生成Go代码
generate-go:
	@echo "📦 Generating Go code..."
	@command -v protoc > /dev/null 2>&1 || { echo "❌ protoc not found"; exit 1; }
	@test -f $(GOBIN)/protoc-gen-go || { echo "❌ protoc-gen-go not found. Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1; }
	@test -f $(GOBIN)/protoc-gen-go-grpc || { echo "❌ protoc-gen-go-grpc not found. Run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; exit 1; }
	@cd .. && protoc -I=proto \
		--go_out=. --go_opt=module=github.com/manxisuo/plum \
		--go-grpc_out=. --go-grpc_opt=module=github.com/manxisuo/plum \
		--plugin=protoc-gen-go=$(GOBIN)/protoc-gen-go \
		--plugin=protoc-gen-go-grpc=$(GOBIN)/protoc-gen-go-grpc \
		proto/task_service.proto
	@echo "✅ Go: controller/proto/"

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
	@rm -f ../controller/proto/*.pb.go
	@rm -f ../controller/plum/proto/*.pb.go 2>/dev/null || true
	@rm -f ../controller/*.pb.go 2>/dev/null || true
	@rm -rf ../plum 2>/dev/null || true
	@rm -f ../sdk/cpp/grpc/proto/*.pb.{h,cc}
	@rm -rf ../sdk/cpp/grpc/proto/proto 2>/dev/null || true
	@echo "✅ Cleaned"

