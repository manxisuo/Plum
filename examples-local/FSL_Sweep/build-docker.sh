#!/bin/bash
# FSL_Sweep Docker 镜像构建脚本
# 从项目根目录执行

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 脚本在 examples-local/FSL_Sweep/，需要上两级到项目根目录
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# 验证项目根目录是否正确（检查是否存在 sdk 和 examples-local 目录）
if [ ! -d "$PROJECT_ROOT/sdk" ] || [ ! -d "$PROJECT_ROOT/examples-local" ]; then
    echo "错误: 无法找到项目根目录"
    echo "当前计算的 PROJECT_ROOT: $PROJECT_ROOT"
    echo "请确保在项目根目录执行此脚本，或从项目根目录执行："
    echo "  cd /path/to/Plum"
    echo "  ./examples-local/FSL_Sweep/build-docker.sh"
    exit 1
fi

cd "$PROJECT_ROOT"

echo "构建 FSL_Sweep Docker 镜像..."
echo "构建上下文: $PROJECT_ROOT"
echo "Dockerfile: examples-local/FSL_Sweep/Dockerfile"

docker build \
  -f examples-local/FSL_Sweep/Dockerfile \
  -t fsl-sweep:1.0.0 \
  .

echo "✅ 镜像构建完成: fsl-sweep:1.0.0"
echo ""
echo "测试镜像:"
echo "  docker run --rm fsl-sweep:1.0.0"
