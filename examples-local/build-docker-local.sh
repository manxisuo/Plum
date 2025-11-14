#!/bin/bash
# 通用的 FSL 项目 Docker 镜像构建脚本
# 使用方法: ./build-docker-local.sh <项目名>
# 示例: ./build-docker-local.sh FSL_Sweep
#       ./build-docker-local.sh FSL_Destroy

set -e

if [ $# -lt 1 ]; then
    echo "用法: $0 <项目名>"
    echo "示例: $0 FSL_Sweep"
    echo "      $0 FSL_Destroy"
    echo "      $0 FSL_Investigate"
    echo "      $0 FSL_Evaluate"
    echo ""
    echo "可用的项目:"
    echo ""
    echo "FSL 项目:"
    echo "  - FSL_Sweep"
    echo "  - FSL_Destroy"
    echo "  - FSL_Investigate"
    echo "  - FSL_Evaluate"
    echo "  - FSL_Plan"
    echo "  - FSL_Statistics"
    echo ""
    echo "Sim 项目:"
    echo "  - SimRoutePlan"
    echo "  - SimNaviControl"
    echo "  - SimSonar"
    echo "  - SimTargetHit"
    echo "  - SimTargetRecognize"
    exit 1
fi

APP_NAME="$1"
APP_DIR="examples-local/$APP_NAME"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# 验证项目根目录
if [ ! -d "$PROJECT_ROOT/sdk" ] || [ ! -d "$PROJECT_ROOT/examples-local" ]; then
    echo "错误: 无法找到项目根目录"
    echo "当前目录: $PROJECT_ROOT"
    exit 1
fi

# 验证项目目录存在
if [ ! -d "$APP_DIR" ]; then
    echo "错误: 项目目录不存在: $APP_DIR"
    exit 1
fi

# 检查是否已编译（所有项目的可执行文件都在项目目录的 bin/ 目录）
# 统一路径：examples-local/<项目名>/bin/<项目名>
BIN_FILE="$APP_DIR/bin/$APP_NAME"

if [ ! -f "$BIN_FILE" ]; then
    echo "⚠️  $APP_NAME 未编译，正在编译..."
    # 统一使用 Makefile 中的规则
    make "examples_$APP_NAME"
fi

# 检查是否有通用的 copy-deps.sh
COPY_DEPS_SCRIPT="$SCRIPT_DIR/copy-deps.sh"
if [ ! -f "$COPY_DEPS_SCRIPT" ]; then
    echo "错误: 找不到通用的 copy-deps.sh 脚本"
    echo "请确保 $COPY_DEPS_SCRIPT 存在"
    exit 1
fi

# 准备依赖目录（使用小写作为临时目录名，避免路径问题）
APP_NAME_LOWER=$(echo "$APP_NAME" | tr '[:upper:]' '[:lower:]')
DEPS_DIR="/tmp/${APP_NAME_LOWER}-deps-$(date +%s)"
echo "📦 准备依赖文件到: $DEPS_DIR"

# 执行依赖复制脚本
"$COPY_DEPS_SCRIPT" "$APP_NAME" "$DEPS_DIR"

# 检查是否有项目特定的 Dockerfile，否则使用通用模板
DOCKERFILE="$APP_DIR/Dockerfile.local"
if [ ! -f "$DOCKERFILE" ]; then
    # 使用通用的 Dockerfile 模板
    DOCKERFILE="$SCRIPT_DIR/Dockerfile.local.template"
    if [ ! -f "$DOCKERFILE" ]; then
        echo "错误: 找不到 Dockerfile"
        echo "请创建 $APP_DIR/Dockerfile.local 或 $SCRIPT_DIR/Dockerfile.local.template"
        exit 1
    fi
fi

# 构建镜像
# 注意：Docker 镜像名称必须是小写，所以需要转换
APP_NAME_LOWER=$(echo "$APP_NAME" | tr '[:upper:]' '[:lower:]')
echo ""
echo "🐳 构建 Docker 镜像: $APP_NAME_LOWER:1.0.0"
# 使用 buildx 确保使用正确的架构
docker buildx build \
  --platform linux/arm64 \
  --load \
  -f "$DOCKERFILE" \
  --build-arg APP_NAME="$APP_NAME" \
  -t "${APP_NAME_LOWER}:1.0.0" \
  "$DEPS_DIR"

# 清理临时目录
echo ""
echo "🧹 清理临时文件..."
rm -rf "$DEPS_DIR"

echo ""
echo "✅ 镜像构建完成: ${APP_NAME_LOWER}:1.0.0"
echo ""
echo "镜像大小:"
docker images "${APP_NAME_LOWER}:1.0.0" --format "  {{.Repository}}:{{.Tag}} - {{.Size}}"
echo ""
echo "测试镜像:"
echo "  docker run --rm ${APP_NAME_LOWER}:1.0.0"

