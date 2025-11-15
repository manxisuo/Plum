#!/bin/bash
# 通用的 Python 项目依赖复制脚本
# 使用方法: ./copy-deps-python.sh <项目名> <target_dir>
# 示例: ./copy-deps-python.sh FSL_MainControl /tmp/fsl-maincontrol-deps

set -e

if [ $# -lt 2 ]; then
    echo "用法: $0 <项目名> <target_dir>"
    echo "示例: $0 FSL_MainControl /tmp/fsl-maincontrol-deps"
    exit 1
fi

APP_NAME="$1"
TARGET_DIR="$2"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

APP_DIR="examples-local/$APP_NAME"

# 检查项目目录是否存在
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

echo "📦 复制 $APP_NAME Python 项目文件到 $TARGET_DIR..."

# 创建目标目录
mkdir -p "$TARGET_DIR/bin"

# 复制源代码文件
echo "复制源代码文件..."
cp "$APP_DIR/app.py" "$TARGET_DIR/"
cp "$APP_DIR/requirements.txt" "$TARGET_DIR/"

# 复制启动脚本和元数据（如果存在）
if [ -f "$APP_DIR/bin/start.sh" ]; then
    cp "$APP_DIR/bin/start.sh" "$TARGET_DIR/bin/"
    chmod +x "$TARGET_DIR/bin/start.sh"
fi
if [ -f "$APP_DIR/bin/meta.ini" ]; then
    cp "$APP_DIR/bin/meta.ini" "$TARGET_DIR/bin/"
fi

# 复制模板目录（如果存在，否则创建空目录）
if [ -d "$APP_DIR/templates" ] && [ "$(ls -A $APP_DIR/templates 2>/dev/null)" ]; then
    echo "复制模板文件..."
    cp -r "$APP_DIR/templates" "$TARGET_DIR/"
else
    mkdir -p "$TARGET_DIR/templates"
    touch "$TARGET_DIR/templates/.gitkeep"
fi

# 复制静态文件目录（如果存在，否则创建空目录）
# 注意：会递归复制所有子目录，包括 tiles/、leaflet/ 等
if [ -d "$APP_DIR/static" ] && [ "$(ls -A $APP_DIR/static 2>/dev/null)" ]; then
    echo "复制静态文件（包括 tiles 瓦片地图和 leaflet 库）..."
    cp -r "$APP_DIR/static" "$TARGET_DIR/"
    # 检查并提示 tiles 目录
    if [ -d "$APP_DIR/static/tiles" ] && [ "$(ls -A $APP_DIR/static/tiles 2>/dev/null)" ]; then
        TILE_COUNT=$(find "$APP_DIR/static/tiles" -name "*.png" 2>/dev/null | wc -l)
        echo "  ✓ 包含离线瓦片地图: $TILE_COUNT 张瓦片"
    else
        echo "  ⚠️  未找到瓦片地图目录，离线环境将使用空白占位图"
    fi
    # 检查并提示 leaflet 目录
    if [ -d "$APP_DIR/static/leaflet" ]; then
        echo "  ✓ 包含 Leaflet 库文件"
    fi
else
    mkdir -p "$TARGET_DIR/static"
    touch "$TARGET_DIR/static/.gitkeep"
fi

# 复制脚本目录（如果存在，否则创建空目录）
if [ -d "$APP_DIR/scripts" ] && [ "$(ls -A $APP_DIR/scripts 2>/dev/null)" ]; then
    echo "复制脚本文件..."
    cp -r "$APP_DIR/scripts" "$TARGET_DIR/"
else
    mkdir -p "$TARGET_DIR/scripts"
    touch "$TARGET_DIR/scripts/.gitkeep"
fi

# 复制离线 Python 包（如果存在）
if [ -d "$APP_DIR/offline-pip-packages" ] && [ "$(ls -A $APP_DIR/offline-pip-packages 2>/dev/null)" ]; then
    echo "复制离线 Python 包..."
    cp -r "$APP_DIR/offline-pip-packages" "$TARGET_DIR/offline-packages"
elif [ -d "$APP_DIR/offline-packages" ] && [ "$(ls -A $APP_DIR/offline-packages 2>/dev/null)" ]; then
    echo "复制离线 Python 包..."
    cp -r "$APP_DIR/offline-packages" "$TARGET_DIR/offline-packages"
else
    mkdir -p "$TARGET_DIR/offline-packages"
    touch "$TARGET_DIR/offline-packages/.gitkeep"
fi

echo "✅ Python 项目文件复制完成"
echo "   源代码: $TARGET_DIR/app.py"
echo "   依赖文件: $TARGET_DIR/requirements.txt"
echo "   启动脚本: $TARGET_DIR/bin/start.sh"
echo "   元数据: $TARGET_DIR/bin/meta.ini"

