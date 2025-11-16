#!/bin/bash
# 通用的 Python 项目 Docker 镜像构建脚本
# 使用方法: ./build-docker-python.sh <项目名> [主题]
# 示例: ./build-docker-python.sh FSL_MainControl
#       ./build-docker-python.sh FSL_MainControl blue
#       ./build-docker-python.sh Sim_Decision

set -e

if [ $# -lt 1 ]; then
    echo "用法: $0 <项目名> [主题]"
    echo "示例: $0 FSL_MainControl"
    echo "      $0 FSL_MainControl blue"
    echo "      $0 Sim_Decision"
    echo ""
    echo "可用的 Python 项目:"
    echo "  - FSL_MainControl"
    echo "  - Sim_Decision"
    echo ""
    echo "主题参数（可选）:"
    echo "  - blue: 使用蓝色主题样式（仅对 FSL_MainControl 有效）"
    exit 1
fi

APP_NAME="$1"
THEME="${2:-default}"
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

# 检查 requirements.txt 是否存在
if [ ! -f "$APP_DIR/requirements.txt" ]; then
    echo "错误: 找不到 requirements.txt: $APP_DIR/requirements.txt"
    exit 1
fi

# 检查 app.py 是否存在
if [ ! -f "$APP_DIR/app.py" ]; then
    echo "错误: 找不到 app.py: $APP_DIR/app.py"
    exit 1
fi

# 检查是否有通用的 copy-deps-python.sh
COPY_DEPS_SCRIPT="$SCRIPT_DIR/copy-deps-python.sh"
if [ ! -f "$COPY_DEPS_SCRIPT" ]; then
    echo "错误: 找不到通用的 copy-deps-python.sh 脚本"
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
DOCKERFILE="$APP_DIR/Dockerfile.python"
if [ ! -f "$DOCKERFILE" ]; then
    # 检查 Python 官方镜像是否可用（网络可能较慢，优先使用 Ubuntu 版本）
    # 如果网络正常，可以使用 Dockerfile.python.template（基于 python:3.11-slim）
    # 如果网络较慢，使用 Dockerfile.python.template.ubuntu（基于 ubuntu:24.04）
    USE_UBUNTU="${USE_UBUNTU_BASE:-true}"
    if [ "$USE_UBUNTU" = "true" ]; then
        DOCKERFILE="$SCRIPT_DIR/Dockerfile.python.template.ubuntu"
        echo "ℹ️  使用 Ubuntu 基础镜像版本（避免网络问题）"
    else
        DOCKERFILE="$SCRIPT_DIR/Dockerfile.python.template"
        echo "ℹ️  使用 Python 官方镜像版本"
    fi
    
    if [ ! -f "$DOCKERFILE" ]; then
        echo "错误: 找不到 Dockerfile"
        echo "请创建 $APP_DIR/Dockerfile.python 或 $SCRIPT_DIR/Dockerfile.python.template"
        exit 1
    fi
fi

# 构建镜像
# 注意：Docker 镜像名称必须是小写，所以需要转换
APP_NAME_LOWER=$(echo "$APP_NAME" | tr '[:upper:]' '[:lower:]')
IMAGE_TAG="${APP_NAME_LOWER}:1.0.0"

echo ""
if [ "$THEME" != "default" ]; then
    echo "🐳 构建 Docker 镜像（主题: ${THEME}）: $IMAGE_TAG"
else
    echo "🐳 构建 Docker 镜像: $IMAGE_TAG"
fi

# 构建参数
BUILD_ARGS=(
    --platform linux/arm64
    --load
    -f "$DOCKERFILE"
    --build-arg APP_NAME="$APP_NAME"
)

# 如果指定了主题，传递 THEME 构建参数
if [ "$THEME" != "default" ]; then
    BUILD_ARGS+=(--build-arg THEME="$THEME")
fi

BUILD_ARGS+=(-t "$IMAGE_TAG" "$DEPS_DIR")

# 使用 buildx 确保使用正确的架构
docker buildx build "${BUILD_ARGS[@]}"

# 清理临时目录
echo ""
echo "🧹 清理临时文件..."
rm -rf "$DEPS_DIR"

# 根据项目确定端口
if [[ "$APP_NAME" == "FSL_MainControl" ]]; then
    PORT=4000
elif [[ "$APP_NAME" == "Sim_Decision" ]]; then
    PORT=3000
else
    PORT=4000  # 默认端口
fi

echo ""
echo "✅ 镜像构建完成: $IMAGE_TAG"
echo ""
echo "镜像大小:"
docker images "$IMAGE_TAG" --format "  {{.Repository}}:{{.Tag}} - {{.Size}}"
echo ""
echo "测试镜像:"
echo "  docker run --rm -p ${PORT}:${PORT} $IMAGE_TAG"

