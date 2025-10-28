#!/bin/bash
# 准备包含必要包的alpine镜像脚本
# 在联网环境中运行

set -e

echo "🚀 准备包含必要包的alpine镜像..."

# 拉取基础alpine镜像
docker pull --platform linux/arm64 alpine:3.18

# 创建包含必要包的Dockerfile
cat > Dockerfile.alpine-with-packages << 'EOF'
FROM alpine:3.18

# 安装必要的包
RUN apk add --no-cache \
    ca-certificates \
    wget \
    tzdata \
    procps \
    curl

# 创建非root用户
RUN addgroup -g 1001 -S plum && \
    adduser -u 1001 -S plum -G plum

# 设置时区
RUN cp /usr/share/zoneinfo/UTC /etc/localtime && \
    echo "UTC" > /etc/timezone
EOF

# 构建包含包的镜像
echo "🐳 构建包含必要包的alpine镜像..."
docker build --platform linux/arm64 -f Dockerfile.alpine-with-packages -t alpine:3.18-with-packages .

# 保存镜像
echo "💾 保存镜像..."
docker save alpine:3.18-with-packages | gzip > alpine-3.18-with-packages-arm64.tar.gz

# 清理临时文件
rm -f Dockerfile.alpine-with-packages

echo "✅ alpine镜像准备完成: alpine-3.18-with-packages-arm64.tar.gz"
echo "📊 文件大小: $(ls -lh alpine-3.18-with-packages-arm64.tar.gz | awk '{print $5}')"
