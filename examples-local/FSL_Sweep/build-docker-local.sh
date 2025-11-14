#!/bin/bash
# 使用宿主机构建产物构建 FSL_Sweep Docker 镜像
# 此脚本会先准备依赖，然后构建镜像

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PROJECT_ROOT"

# 验证项目根目录
if [ ! -d "$PROJECT_ROOT/sdk" ] || [ ! -d "$PROJECT_ROOT/examples-local" ]; then
    echo "错误: 无法找到项目根目录"
    echo "当前目录: $PROJECT_ROOT"
    exit 1
fi

# 检查是否已编译
if [ ! -f "examples-local/FSL_Sweep/bin/FSL_Sweep" ]; then
    echo "⚠️  FSL_Sweep 未编译，正在编译..."
    make examples_FSL_Sweep
fi

# 准备依赖目录
DEPS_DIR="/tmp/fsl-sweep-deps-$(date +%s)"
echo "📦 准备依赖文件到: $DEPS_DIR"

# 执行依赖复制脚本
"$SCRIPT_DIR/copy-deps.sh" "$DEPS_DIR"

# 构建镜像
echo ""
echo "🐳 构建 Docker 镜像..."
docker build \
  -f examples-local/FSL_Sweep/Dockerfile.local \
  -t fsl-sweep:1.0.0 \
  "$DEPS_DIR"

# 清理临时目录
echo ""
echo "🧹 清理临时文件..."
rm -rf "$DEPS_DIR"

echo ""
echo "✅ 镜像构建完成: fsl-sweep:1.0.0"
echo ""
echo "测试镜像:"
echo "  docker run --rm fsl-sweep:1.0.0"

