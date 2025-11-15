#!/bin/bash
# 生成离线Docker镜像包脚本
# 用于在联网环境准备完整的Docker镜像包

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 确保在项目根目录
if [ ! -f "Makefile" ] || [ ! -d "controller" ] || [ ! -d "agent-go" ]; then
    print_error "请在项目根目录运行此脚本"
    exit 1
fi

OUTPUT_DIR="offline-images"
TARGET_PLATFORM="linux/arm64" # 默认为arm64，可通过环境变量覆盖，如: TARGET_PLATFORM=linux/amd64
print_info "🚀 开始生成离线Docker镜像包..."

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 1. 拉取基础镜像
print_info "📥 拉取基础镜像 (平台: ${TARGET_PLATFORM})..."
docker pull --platform "${TARGET_PLATFORM}" alpine:3.18
docker pull --platform "${TARGET_PLATFORM}" nginx:alpine

# 2. 检查是否有预构建的plum镜像
HAS_PLUM_CONTROLLER=false
HAS_PLUM_AGENT=false

if docker images plum-controller:offline --format "{{.Repository}}:{{.Tag}}" | grep -q "plum-controller:offline"; then
    HAS_PLUM_CONTROLLER=true
    print_info "✅ 发现预构建的plum-controller:offline镜像"
fi

if docker images plum-agent:offline --format "{{.Repository}}:{{.Tag}}" | grep -q "plum-agent:offline"; then
    HAS_PLUM_AGENT=true
    print_info "✅ 发现预构建的plum-agent:offline镜像"
fi

# 3. 导出镜像
print_info "💾 导出Docker镜像..."

# 计算各镜像架构
ALPINE_ARCH=$(docker inspect alpine:3.18 --format '{{.Architecture}}' || echo unknown)
NGINX_ARCH=$(docker inspect nginx:alpine --format '{{.Architecture}}' || echo unknown)

# 导出基础镜像
ALPINE_OUT="$OUTPUT_DIR/alpine-3.18-${ALPINE_ARCH}.tar.gz"
docker save alpine:3.18 | gzip > "$ALPINE_OUT"
print_success "alpine镜像已导出: $ALPINE_OUT"

NGINX_OUT="$OUTPUT_DIR/nginx-alpine-${NGINX_ARCH}.tar.gz"
docker save nginx:alpine | gzip > "$NGINX_OUT"
print_success "nginx镜像已导出: $NGINX_OUT"

# 导出plum镜像（如果存在）
if [ "$HAS_PLUM_CONTROLLER" = true ]; then
    CTRL_ARCH=$(docker inspect plum-controller:offline --format '{{.Architecture}}' || echo unknown)
    CTRL_OUT="$OUTPUT_DIR/plum-controller-offline-${CTRL_ARCH}.tar.gz"
    docker save plum-controller:offline | gzip > "$CTRL_OUT"
    print_success "plum-controller镜像已导出: $CTRL_OUT"
fi

if [ "$HAS_PLUM_AGENT" = true ]; then
    AGENT_ARCH=$(docker inspect plum-agent:offline --format '{{.Architecture}}' || echo unknown)
    AGENT_OUT="$OUTPUT_DIR/plum-agent-offline-${AGENT_ARCH}.tar.gz"
    docker save plum-agent:offline | gzip > "$AGENT_OUT"
    print_success "plum-agent镜像已导出: $AGENT_OUT"
fi

# 4. 生成镜像清单
print_info "📋 生成镜像清单..."
cat > "$OUTPUT_DIR/images.txt" << EOF
alpine:3.18 arch=${ALPINE_ARCH}
nginx:alpine arch=${NGINX_ARCH}
EOF

if [ "$HAS_PLUM_CONTROLLER" = true ]; then
    echo "plum-controller:offline arch=${CTRL_ARCH}" >> "$OUTPUT_DIR/images.txt"
fi

if [ "$HAS_PLUM_AGENT" = true ]; then
    echo "plum-agent:offline arch=${AGENT_ARCH}" >> "$OUTPUT_DIR/images.txt"
fi

print_success "镜像清单已生成: $OUTPUT_DIR/images.txt"

# 5. 显示结果
print_info "📊 生成完成！"
echo ""
echo "📁 输出目录: $OUTPUT_DIR/"
echo "📋 文件列表:"
ls -lh "$OUTPUT_DIR/"
echo ""
echo "📋 镜像清单:"
cat "$OUTPUT_DIR/images.txt"
echo ""
echo "🚚 在离线环境中加载镜像:"
echo "   for f in $OUTPUT_DIR/*.tar.gz; do docker load < \"\$f\"; done"
echo ""
echo "ℹ️  提示: 文件名已包含架构后缀 (例如: -amd64 / -arm64)。请在与镜像架构匹配的目标环境中使用。"
echo "   验证镜像架构: docker inspect <image:tag> | grep -i Architecture"
if [ "$HAS_PLUM_CONTROLLER" = false ] || [ "$HAS_PLUM_AGENT" = false ]; then
    echo "⚠️  注意: 缺少plum镜像，需要在离线环境中运行:"
    echo "   ./docker/build-static-offline.sh"
fi
