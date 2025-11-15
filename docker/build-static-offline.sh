#!/bin/bash
# 完全离线Docker镜像构建脚本

set -e

echo "🚀 完全离线Docker镜像构建"
echo "=================================="

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
if [ ! -d "controller/vendor" ]; then
    echo "📦 生成Controller依赖..."
    cd controller && go mod vendor && cd ..
fi

if [ ! -d "agent-go/vendor" ]; then
    echo "📦 生成Agent依赖..."
    cd agent-go && go mod vendor && cd ..
fi

# 构建Controller（修复路径）
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

# 构建Agent
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
echo "✅ 验证构建结果..."
if [ -f "controller/bin/controller" ]; then
    echo "✅ Controller构建成功"
    ls -lh controller/bin/controller
else
    echo "❌ Controller构建失败"
    exit 1
fi

if [ -f "agent-go/plum-agent" ]; then
    echo "✅ Agent构建成功"
    ls -lh agent-go/plum-agent
else
    echo "❌ Agent构建失败"
    exit 1
fi

# 创建静态Dockerfile
echo "📝 创建静态Dockerfile..."

# Controller静态Dockerfile
cat > Dockerfile.controller.static << 'EOF'
FROM alpine:3.18
WORKDIR /app
# 注意：这里假设alpine:3.18已经包含了必要的包
# 如果alpine镜像中没有这些包，需要预先准备一个包含这些包的镜像
COPY controller/bin/controller ./bin/controller
RUN addgroup -g 1001 -S plum && adduser -u 1001 -S plum -G plum
RUN mkdir -p /app/data && chown -R plum:plum /app
USER plum
EXPOSE 8080
CMD ["./bin/controller"]
EOF

# Agent静态Dockerfile
cat > Dockerfile.agent.static << 'EOF'
FROM alpine:3.18
WORKDIR /app
# 注意：这里假设alpine:3.18已经包含了必要的包
COPY agent-go/plum-agent ./plum-agent
RUN addgroup -g 1001 -S plum && adduser -u 1001 -S plum -G plum
RUN mkdir -p /app/data && chown -R plum:plum /app
USER plum
CMD ["./plum-agent"]
EOF

# 构建静态镜像
echo "🐳 构建Controller静态镜像..."
docker build --platform linux/arm64 -f Dockerfile.controller.static -t plum-controller:offline .

echo "🐳 构建Agent静态镜像..."
docker build --platform linux/arm64 -f Dockerfile.agent.static -t plum-agent:offline .

# 清理临时文件
rm -f Dockerfile.controller.static Dockerfile.agent.static

# 验证镜像
echo "✅ 验证镜像..."
docker images | grep -E "(plum-controller|plum-agent)" | grep offline

echo ""
echo "🎉 静态Docker镜像构建完成！"
echo "现在可以启动服务:"
echo "  docker-compose -f docker-compose.offline.yml up -d"
