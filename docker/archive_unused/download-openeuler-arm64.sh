#!/bin/bash
# 下载 openEuler ARM64 镜像并导出脚本

set -e

IMAGE_NAME="openeuler/openeuler"
TAG="${1:-latest}"  # 默认使用 latest，可通过参数指定
OUTPUT_FILE="openeuler-${TAG}-arm64.tar.gz"

echo "🚀 开始下载 openEuler ARM64 镜像..."
echo "镜像: ${IMAGE_NAME}:${TAG}"
echo "平台: linux/arm64"
echo ""

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "❌ 错误: Docker 未安装"
    exit 1
fi

# 拉取镜像
echo "📥 拉取镜像..."
docker pull --platform linux/arm64 ${IMAGE_NAME}:${TAG}

# 验证架构
echo ""
echo "🔍 验证镜像架构..."
ARCH=$(docker inspect ${IMAGE_NAME}:${TAG} --format '{{.Architecture}}')
echo "镜像架构: ${ARCH}"

if [ "$ARCH" != "arm64" ] && [ "$ARCH" != "aarch64" ]; then
    echo "⚠️  警告: 镜像架构不是 ARM64 (实际: ${ARCH})"
fi

# 导出镜像
echo ""
echo "💾 导出镜像..."
docker save ${IMAGE_NAME}:${TAG} | gzip > "${OUTPUT_FILE}"

# 显示文件信息
echo ""
echo "✅ 导出完成!"
echo "📁 文件: ${OUTPUT_FILE}"
ls -lh "${OUTPUT_FILE}"
echo ""
echo "📊 文件大小: $(du -h ${OUTPUT_FILE} | cut -f1)"
echo ""
echo "🚚 在目标 ARM64 环境中导入:"
echo "   gunzip -c ${OUTPUT_FILE} | docker load"
echo ""
