#!/bin/bash
# 通用的 Controller 和 Agent Docker 镜像构建脚本
# 使用方法: ./docker/build-docker.sh [controller|agent|all] [--local]
# 示例: ./docker/build-docker.sh controller
#       ./docker/build-docker.sh agent
#       ./docker/build-docker.sh all
#       ./docker/build-docker.sh all --local  # 使用本地 Go 环境构建（推荐，适合网络慢的环境）

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# 检查是否在项目根目录
if [ ! -f "Makefile" ] || [ ! -d "controller" ] || [ ! -d "agent-go" ]; then
    echo "❌ 请在 Plum 项目根目录运行此脚本"
    exit 1
fi

# 解析参数
BUILD_TARGET="${1:-all}"
USE_LOCAL="${2:-}"

# 检查是否使用本地构建模式
if [ "$USE_LOCAL" = "--local" ]; then
    USE_LOCAL_BUILD=true
else
    USE_LOCAL_BUILD=false
fi

# 本地构建函数（使用主机 Go 环境）
build_controller_local() {
    echo "🔨 使用本地 Go 环境构建 Controller..."
    
    # 检查 Go 环境
    if ! command -v go &> /dev/null; then
        echo "❌ Go 环境未找到，请先安装 Go"
        exit 1
    fi
    
    # 设置环境变量
    export GOOS=linux
    export GOARCH=arm64
    export CGO_ENABLED=0
    
    # 构建二进制
    cd controller
    if [ ! -f "cmd/server/main.go" ]; then
        echo "❌ 未找到 Controller 主文件"
        exit 1
    fi
    go build -ldflags="-w -s -extldflags '-static'" -o bin/controller ./cmd/server
    cd ..
    
    # 创建临时 Dockerfile
    cat > /tmp/Dockerfile.controller.local << 'EOF'
FROM alpine:3.18
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1001 -S plum && \
    adduser -u 1001 -S plum -G plum && \
    mkdir -p /app/data && \
    chown -R plum:plum /app
COPY controller/bin/controller ./bin/controller
COPY controller/static ./controller/static 2>/dev/null || true
COPY controller/env.example ./.env
USER plum
EXPOSE 8080
ENV CONTROLLER_ADDR=:8080
ENV CONTROLLER_DB=file:/app/data/controller.db?_pragma=busy_timeout(5000)
ENV CONTROLLER_DATA_DIR=/app/data
CMD ["./bin/controller"]
EOF
    
    # 构建镜像
    echo "🐳 构建 Controller Docker 镜像..."
    docker buildx build \
        --platform linux/arm64 \
        --load \
        -f /tmp/Dockerfile.controller.local \
        -t plum-controller:latest \
        .
    
    rm -f /tmp/Dockerfile.controller.local
    echo "✅ Controller 镜像构建完成: plum-controller:latest"
    docker images plum-controller:latest --format "  镜像大小: {{.Size}}"
}

build_agent_local() {
    echo "🔨 使用本地 Go 环境构建 Agent..."
    
    # 检查 Go 环境
    if ! command -v go &> /dev/null; then
        echo "❌ Go 环境未找到，请先安装 Go"
        exit 1
    fi
    
    # 设置环境变量
    export GOOS=linux
    export GOARCH=arm64
    export CGO_ENABLED=0
    
    # 构建二进制
    cd agent-go
    if [ ! -f "main.go" ]; then
        echo "❌ 未找到 Agent 主文件"
        exit 1
    fi
    go build -ldflags="-w -s -extldflags '-static'" -o plum-agent .
    cd ..
    
    # 创建临时 Dockerfile
    cat > /tmp/Dockerfile.agent.local << 'EOF'
FROM alpine:3.18
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -g 1001 -S plum && \
    adduser -u 1001 -S plum -G plum && \
    mkdir -p /app/data && \
    chown -R plum:plum /app
COPY agent-go/plum-agent ./plum-agent
COPY agent-go/env.example ./.env
USER plum
ENV AGENT_NODE_ID=nodeA
ENV CONTROLLER_BASE=http://plum-controller:8080
ENV AGENT_DATA_DIR=/app/data
CMD ["./plum-agent"]
EOF
    
    # 构建镜像
    echo "🐳 构建 Agent Docker 镜像..."
    docker buildx build \
        --platform linux/arm64 \
        --load \
        -f /tmp/Dockerfile.agent.local \
        -t plum-agent:latest \
        .
    
    rm -f /tmp/Dockerfile.agent.local
    echo "✅ Agent 镜像构建完成: plum-agent:latest"
    docker images plum-agent:latest --format "  镜像大小: {{.Size}}"
}

# Docker 构建函数（使用 Dockerfile 多阶段构建）
build_controller_docker() {
    echo "🐳 构建 Controller Docker 镜像（Dockerfile 多阶段构建）..."
    docker buildx build \
        --platform linux/arm64 \
        --load \
        -f docker/controller/Dockerfile \
        -t plum-controller:latest \
        .
    echo "✅ Controller 镜像构建完成: plum-controller:latest"
    docker images plum-controller:latest --format "  镜像大小: {{.Size}}"
}

build_agent_docker() {
    echo "🐳 构建 Agent Docker 镜像（Dockerfile 多阶段构建）..."
    docker buildx build \
        --platform linux/arm64 \
        --load \
        -f docker/agent/Dockerfile \
        -t plum-agent:latest \
        .
    echo "✅ Agent 镜像构建完成: plum-agent:latest"
    docker images plum-agent:latest --format "  镜像大小: {{.Size}}"
}

# 根据参数构建
if [ "$USE_LOCAL_BUILD" = true ]; then
    # 使用本地构建模式
    case "$BUILD_TARGET" in
        controller)
            build_controller_local
            ;;
        agent)
            build_agent_local
            ;;
        all)
            build_controller_local
            echo ""
            build_agent_local
            ;;
        *)
            echo "用法: $0 [controller|agent|all] [--local]"
            echo ""
            echo "示例:"
            echo "  $0 controller --local    # 使用本地 Go 环境构建 Controller（推荐，适合网络慢）"
            echo "  $0 agent --local         # 使用本地 Go 环境构建 Agent（推荐，适合网络慢）"
            echo "  $0 all --local            # 使用本地 Go 环境构建 Controller 和 Agent（推荐）"
            exit 1
            ;;
    esac
else
    # 使用 Docker 构建模式
    case "$BUILD_TARGET" in
        controller)
            build_controller_docker
            ;;
        agent)
            build_agent_docker
            ;;
        all)
            build_controller_docker
            echo ""
            build_agent_docker
            ;;
        *)
            echo "用法: $0 [controller|agent|all] [--local]"
            echo ""
            echo "构建模式:"
            echo "  默认模式（Docker 多阶段构建）:"
            echo "    $0 controller    # 只构建 Controller（需要下载 golang 镜像）"
            echo "    $0 agent         # 只构建 Agent（需要下载 golang 镜像）"
            echo "    $0 all           # 构建 Controller 和 Agent（默认）"
            echo ""
            echo "  本地构建模式（推荐，适合网络慢的环境）:"
            echo "    $0 controller --local    # 使用本地 Go 环境构建 Controller"
            echo "    $0 agent --local         # 使用本地 Go 环境构建 Agent"
            echo "    $0 all --local           # 使用本地 Go 环境构建 Controller 和 Agent"
            echo ""
            echo "💡 提示: 如果网络慢导致构建失败，请使用 --local 模式"
            exit 1
            ;;
    esac
fi

echo ""
echo "🎉 构建完成！"
echo ""
if [ "$USE_LOCAL_BUILD" = true ]; then
    echo "📝 注意: 使用了本地构建模式，镜像标签为 plum-controller:latest 和 plum-agent:latest"
    echo "   这些镜像可以在任何 docker-compose 文件中使用，只需将 yml 文件中的镜像标签改为 latest"
    echo "   例如：将 'image: plum-controller:offline' 改为 'image: plum-controller:latest'"
    echo ""
fi
echo "测试镜像:"
if [ "$BUILD_TARGET" = "controller" ] || [ "$BUILD_TARGET" = "all" ]; then
    echo "  Controller: docker run --rm -p 8080:8080 plum-controller:latest"
fi
if [ "$BUILD_TARGET" = "agent" ] || [ "$BUILD_TARGET" = "all" ]; then
    echo "  Agent: docker run --rm plum-agent:latest"
fi

