#!/bin/bash
# 银河麒麟V10 ARM64环境构建脚本

set -e

echo "🚀 开始构建Plum项目..."

# 设置Go环境
export PATH=$PATH:/usr/local/go/bin

# 配置离线模式，防止Go下载工具链
export GOTOOLCHAIN=local
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=arm64

echo "🔧 Go环境配置:"
echo "   PATH: $PATH"
echo "   GOTOOLCHAIN: $GOTOOLCHAIN"
echo "   Go版本: $(go version)"

# 检查Go protobuf工具（离线模式）
echo "📦 检查Go protobuf工具..."
if [ ! -f "$HOME/go/bin/protoc-gen-go" ] || [ ! -f "$HOME/go/bin/protoc-gen-go-grpc" ]; then
    if [ -d "../tools/go-arm64-tools/bin" ]; then
        echo "🔌 离线模式：使用预编译的ARM64工具..."
        export GOPATH=$HOME/go
        mkdir -p $GOPATH/bin
        cp ../tools/go-arm64-tools/bin/* $GOPATH/bin/
        chmod +x $GOPATH/bin/*
        echo "✅ 使用预编译的ARM64工具"
    else
        echo "🔌 离线模式：预编译工具未找到，检查现有工具..."
        export GOPATH=$HOME/go
        mkdir -p $GOPATH/bin
        
        # 检查是否已经有工具
        if [ -f "$GOPATH/bin/protoc-gen-go" ] && [ -f "$GOPATH/bin/protoc-gen-go-grpc" ]; then
            echo "✅ protobuf工具已存在"
        else
            echo "⚠️  缺少protobuf工具，proto生成可能失败"
            echo "   建议在WSL2环境中重新运行prepare-offline-deploy.sh"
        fi
    fi
else
    echo "✅ Go protobuf工具已存在"
fi

# 进入项目目录
cd ../source/Plum

# 设置环境变量，传递给make命令
export GOTOOLCHAIN=local
export CGO_ENABLED=0

# 1. 生成proto代码
echo "📦 生成protobuf代码..."
if [ -f "Makefile" ]; then
    make proto
    echo "✅ Proto代码生成完成"
else
    echo "❌ 未找到Makefile"
    exit 1
fi

# 2. 构建Controller
echo "📦 构建Controller..."
echo "🔧 环境变量: GOTOOLCHAIN=$GOTOOLCHAIN, CGO_ENABLED=$CGO_ENABLED"

# 方法1: 直接构建（推荐，避免make传递环境变量的问题）
cd controller

# 使用vendor模式构建，避免网络依赖
echo "🔧 构建配置：使用vendor模式 + 离线工具链"
if [ -d "vendor" ]; then
    echo "使用vendor目录构建..."
    CGO_ENABLED=0 GOTOOLCHAIN=local go build -mod=vendor -trimpath -ldflags "-s -w" -o bin/controller ./cmd/server
    echo "✅ Controller构建完成（使用vendor模式）"
else
    echo "❌ 未找到vendor目录，这可能导致构建失败"
    echo "尝试使用模块模式，但可能因网络问题失败..."
    CGO_ENABLED=0 GOTOOLCHAIN=local go build -trimpath -ldflags "-s -w" -o bin/controller ./cmd/server || {
        echo "❌ Controller构建失败，请检查依赖和网络配置"
        exit 1
    }
    echo "✅ Controller构建完成（使用模块模式）"
fi

if [ -f "bin/controller" ]; then
    echo "✅ Controller构建完成: bin/controller"
    # 验证构建结果
    file bin/controller
    echo "Controller大小: $(du -h bin/controller | cut -f1)"
else
    echo "❌ Controller构建失败"
    exit 1
fi
cd ..

# 3. 构建Agent
echo "📦 构建Agent..."
cd agent-go

# 使用vendor模式构建
if [ -d "vendor" ]; then
    go build -mod=vendor -o plum-agent
    echo "✅ Agent构建完成（使用vendor模式）"
else
    go build -o plum-agent
    echo "✅ Agent构建完成（使用模块模式）"
fi

if [ -f "plum-agent" ]; then
    echo "✅ Agent构建完成: plum-agent"
    # 验证构建结果
    file plum-agent
    echo "Agent大小: $(du -h plum-agent | cut -f1)"
else
    echo "❌ Agent构建失败"
    exit 1
fi
cd ..

# 4. 构建C++ SDK和Plum Client库
echo "📦 构建C++ SDK和Plum Client库..."

# 检查CMake是否可用
if ! command -v cmake &> /dev/null; then
    echo "❌ CMake未安装，跳过C++ SDK构建"
    echo "   如需构建C++ SDK，请安装CMake: sudo apt-get install cmake"
else
    echo "🔧 检查C++依赖..."
    
    # 检查httplib (plumclient现在使用httplib，不再需要libcurl)
    if [ -f "/usr/include/httplib.h" ] || [ -f "/usr/local/include/httplib.h" ]; then
        echo "✅ httplib头文件已找到"
    else
        echo "ℹ️  httplib头文件未在系统路径找到，将使用项目内置版本"
    fi
    
    # 检查pthread
    if ! pkg-config --exists pthread; then
        echo "⚠️  pthread未找到，C++ SDK构建可能失败"
        echo "   请安装: sudo apt-get install libpthread-stubs0-dev"
    else
        echo "✅ pthread已安装"
    fi
    
    # 构建C++ SDK
    echo "🚀 开始构建C++ SDK..."
    if make sdk_cpp_offline; then
        echo "✅ C++ SDK构建完成"
        
        # 构建Plum Client库
        echo "🚀 开始构建Plum Client库..."
        if make plumclient; then
            echo "✅ Plum Client库构建完成"
            
            # 构建Service Client示例
            echo "🚀 开始构建Service Client示例..."
            if make service_client_example; then
                echo "✅ Service Client示例构建完成"
            else
                echo "⚠️  Service Client示例构建失败，但库构建成功"
            fi
        else
            echo "⚠️  Plum Client库构建失败"
        fi
    else
        echo "⚠️  C++ SDK构建失败，跳过Plum Client库构建"
    fi
fi

# 5. 构建Web UI
echo "📦 构建Web UI..."
cd ui

# 检查node_modules是否已存在
if [ ! -d "node_modules" ]; then
    echo "❌ 未找到node_modules，请先运行依赖准备脚本"
    exit 1
fi

# 检查并修复 rollup ARM64 依赖
echo "🔍 检查 rollup ARM64 依赖..."
if [ ! -d "node_modules/@rollup/rollup-linux-arm64-gnu" ]; then
    echo "⚠️  rollup ARM64 依赖缺失，尝试修复..."
    cd ..
    
    # 尝试使用修复脚本
    if [ -f "../scripts/fix-rollup-arm64.sh" ]; then
        echo "🔧 运行 rollup ARM64 修复脚本..."
        bash ../scripts/fix-rollup-arm64.sh || {
            echo "⚠️  修复脚本运行失败，继续尝试构建..."
        }
    else
        echo "⚠️  修复脚本不存在，跳过 rollup 修复"
    fi
    
    cd ui
fi

# 构建UI
echo "🚀 开始构建UI..."
npm run build

if [ -d "dist" ]; then
    echo "✅ Web UI构建完成: dist/"
    echo "UI构建产物大小: $(du -sh dist | cut -f1)"
else
    echo "❌ Web UI构建失败"
    exit 1
fi
cd ..

echo "🎉 所有组件构建完成！"
echo ""
echo "构建结果:"
echo "- Controller: controller/bin/controller"
echo "- Agent: agent-go/plum-agent"  
echo "- Web UI: ui/dist/"

# 检查C++ SDK构建结果
if [ -f "sdk/cpp/build/plumclient/libplumclient.so" ]; then
    echo "- Plum Client库: sdk/cpp/build/plumclient/libplumclient.so"
    echo "  库大小: $(du -h sdk/cpp/build/plumclient/libplumclient.so | cut -f1)"
fi

if [ -f "sdk/cpp/build/examples/service_client_example/service_client_example" ]; then
    echo "- Service Client示例: sdk/cpp/build/examples/service_client_example/service_client_example"
    echo "  示例大小: $(du -h sdk/cpp/build/examples/service_client_example/service_client_example | cut -f1)"
fi

echo ""
echo "下一步: 运行部署脚本"
