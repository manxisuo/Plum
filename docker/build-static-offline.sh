#!/bin/bash
# 完全离线Docker镜像构建脚本
# 使用方法: ./docker/build-static-offline.sh [controller|agent|all]
# 示例: ./docker/build-static-offline.sh controller
#       ./docker/build-static-offline.sh agent
#       ./docker/build-static-offline.sh all  # 默认，构建两个

set -e

# 解析参数
BUILD_TARGET="${1:-all}"

echo "🚀 完全离线Docker镜像构建"
echo "=================================="
echo "构建目标: $BUILD_TARGET"
echo ""

# 检查环境
if [ ! -f "Makefile" ] || [ ! -d "controller" ] || [ ! -d "agent-go" ]; then
    echo "❌ 请在Plum项目根目录运行此脚本"
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ Go环境未找到"
    exit 1
fi

echo "✅ Go版本: $(go version)"

# 设置环境变量
export GOOS=linux
export GOARCH=arm64
export CGO_ENABLED=0

echo "✅ Go环境变量设置完成"

# 检查依赖
if [ "$BUILD_TARGET" = "controller" ] || [ "$BUILD_TARGET" = "all" ]; then
    if [ ! -d "controller/vendor" ]; then
        echo "📦 生成Controller依赖..."
        cd controller && go mod vendor && cd ..
    fi
fi

if [ "$BUILD_TARGET" = "agent" ] || [ "$BUILD_TARGET" = "all" ]; then
    if [ ! -d "agent-go/vendor" ]; then
        echo "📦 生成Agent依赖..."
        cd agent-go && go mod vendor && cd ..
    fi
fi

# 构建函数
build_controller() {
    echo "🔨 构建Controller..."
    cd controller
    if [ -f "cmd/server/main.go" ]; then
        echo "✅ 找到Controller主文件: cmd/server/main.go"
        go build -ldflags="-w -s -extldflags '-static'" -o bin/controller ./cmd/server
    else
        echo "❌ 未找到Controller主文件"
        exit 1
    fi
    cd ..
    
    # 验证构建结果
    if [ -f "controller/bin/controller" ]; then
        echo "✅ Controller构建成功"
        ls -lh controller/bin/controller
    else
        echo "❌ Controller构建失败"
        exit 1
    fi
}

build_agent() {
    echo "🔨 构建Agent..."
    cd agent-go
    if [ -f "main.go" ]; then
        echo "✅ 找到Agent主文件: main.go"
        go build -ldflags="-w -s -extldflags '-static'" -o plum-agent .
    else
        echo "❌ 未找到Agent主文件"
        exit 1
    fi
    cd ..
    
    # 验证构建结果
    if [ -f "agent-go/plum-agent" ]; then
        echo "✅ Agent构建成功"
        ls -lh agent-go/plum-agent
    else
        echo "❌ Agent构建失败"
        exit 1
    fi
}

# 根据参数构建
case "$BUILD_TARGET" in
    controller)
        build_controller
        ;;
    agent)
        build_agent
        ;;
    all)
        build_controller
        echo ""
        build_agent
        ;;
    *)
        echo "用法: $0 [controller|agent|all]"
        echo ""
        echo "示例:"
        echo "  $0 controller    # 只构建 Controller"
        echo "  $0 agent         # 只构建 Agent"
        echo "  $0 all           # 构建 Controller 和 Agent（默认）"
        exit 1
        ;;
esac

# 创建静态Dockerfile
echo "📝 创建静态Dockerfile..."

if [ "$BUILD_TARGET" = "controller" ] || [ "$BUILD_TARGET" = "all" ]; then
    # Controller静态Dockerfile
    cat > Dockerfile.controller.static << 'EOF'
FROM alpine:3.18
WORKDIR /app
# 注意：这里假设alpine:3.18已经包含了必要的包
# 如果alpine镜像中没有这些包，需要预先准备一个包含这些包的镜像
COPY controller/bin/controller ./bin/controller
COPY controller/static ./controller/static
COPY controller/env.example ./.env
RUN addgroup -g 1001 -S plum && adduser -u 1001 -S plum -G plum
RUN mkdir -p /app/data && chown -R plum:plum /app
USER plum
EXPOSE 8080
CMD ["./bin/controller"]
EOF
fi

if [ "$BUILD_TARGET" = "agent" ] || [ "$BUILD_TARGET" = "all" ]; then
    # Agent静态Dockerfile
    cat > Dockerfile.agent.static << 'EOF'
FROM alpine:3.18
WORKDIR /app
# 注意：这里假设alpine:3.18已经包含了必要的包
COPY agent-go/plum-agent ./plum-agent
COPY agent-go/env.example ./.env
RUN addgroup -g 1001 -S plum && adduser -u 1001 -S plum -G plum
RUN mkdir -p /app/data && chown -R plum:plum /app
USER plum
CMD ["./plum-agent"]
EOF
fi

# 构建静态镜像函数
build_controller_image() {
    echo "🐳 构建Controller静态镜像..."
    docker build --platform linux/arm64 -f Dockerfile.controller.static -t plum-controller:offline .
}

build_agent_image() {
    echo "🐳 构建Agent静态镜像..."
    docker build --platform linux/arm64 -f Dockerfile.agent.static -t plum-agent:offline .
}

# 根据参数构建镜像
case "$BUILD_TARGET" in
    controller)
        build_controller_image
        ;;
    agent)
        build_agent_image
        ;;
    all)
        build_controller_image
        echo ""
        build_agent_image
        ;;
esac

# 清理临时文件
echo ""
echo "🧹 清理临时文件..."
if [ "$BUILD_TARGET" = "controller" ] || [ "$BUILD_TARGET" = "all" ]; then
    rm -f Dockerfile.controller.static
fi
if [ "$BUILD_TARGET" = "agent" ] || [ "$BUILD_TARGET" = "all" ]; then
    rm -f Dockerfile.agent.static
fi

# 验证镜像
echo ""
echo "✅ 验证镜像..."
if [ "$BUILD_TARGET" = "controller" ] || [ "$BUILD_TARGET" = "all" ]; then
    docker images | grep "plum-controller" | grep offline || echo "⚠️  Controller镜像未找到"
fi
if [ "$BUILD_TARGET" = "agent" ] || [ "$BUILD_TARGET" = "all" ]; then
    docker images | grep "plum-agent" | grep offline || echo "⚠️  Agent镜像未找到"
fi

echo ""
echo "🎉 静态Docker镜像构建完成！"
if [ "$BUILD_TARGET" = "all" ]; then
    echo "现在可以启动服务:"
    echo "  docker-compose -f docker-compose.offline.yml up -d"
fi
