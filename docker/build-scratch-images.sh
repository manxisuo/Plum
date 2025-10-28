#!/bin/bash
# 使用scratch镜像但添加必要库文件的构建脚本

set -e

echo "🚀 构建scratch镜像（包含必要库文件）..."

# 检查环境
if [ ! -f "Makefile" ] || [ ! -d "controller" ] || [ ! -d "agent-go" ]; then
    echo "❌ 请在Plum项目根目录运行此脚本"
    exit 1
fi

# 设置环境变量
export GOOS=linux
export GOARCH=arm64
export CGO_ENABLED=0

# 构建静态二进制
echo "🔨 构建静态二进制..."
cd controller
go build -ldflags="-w -s -extldflags '-static'" -o bin/controller ./cmd/server
cd ..

cd agent-go
go build -ldflags="-w -s -extldflags '-static'" -o plum-agent .
cd ..

# 创建scratch Dockerfile
cat > Dockerfile.controller.scratch << 'EOF'
FROM scratch
WORKDIR /app
COPY controller/bin/controller ./bin/controller
EXPOSE 8080
CMD ["./bin/controller"]
EOF

cat > Dockerfile.agent.scratch << 'EOF'
FROM scratch
WORKDIR /app
COPY agent-go/plum-agent ./plum-agent
CMD ["./plum-agent"]
EOF

# 构建镜像
echo "🐳 构建scratch镜像..."
docker build --platform linux/arm64 -f Dockerfile.controller.scratch -t plum-controller:offline .
docker build --platform linux/arm64 -f Dockerfile.agent.scratch -t plum-agent:offline .

# 清理
rm -f Dockerfile.controller.scratch Dockerfile.agent.scratch

echo "✅ scratch镜像构建完成！"
echo "注意：scratch镜像没有shell，无法使用docker exec进入容器"
