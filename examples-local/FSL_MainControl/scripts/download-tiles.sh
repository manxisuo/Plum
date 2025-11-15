#!/bin/bash
# 下载离线地图瓦片的便捷脚本
# 使用方法: ./download-tiles.sh [选项]
# 
# 示例:
#   ./download-tiles.sh                    # 使用默认参数
#   ./download-tiles.sh --min-zoom 11 --max-zoom 15
#   ./download-tiles.sh --lat 30.664554 --lon 122.510268 --radius 0.05

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PYTHON_SCRIPT="$SCRIPT_DIR/download_tiles.py"

# 检查 Python 脚本是否存在
if [ ! -f "$PYTHON_SCRIPT" ]; then
    echo "❌ 错误: 找不到 download_tiles.py"
    exit 1
fi

# 检查 Python 3 是否可用
if ! command -v python3 &> /dev/null; then
    echo "❌ 错误: 未找到 python3"
    exit 1
fi

# 检查 requests 库是否安装
if ! python3 -c "import requests" 2>/dev/null; then
    echo "⚠️  警告: requests 库未安装，正在安装..."
    pip3 install requests || {
        echo "❌ 错误: 无法安装 requests 库"
        echo "   请手动运行: pip3 install requests"
        exit 1
    }
fi

echo "📥 开始下载离线地图瓦片..."
echo ""

# 执行 Python 脚本，传递所有参数
python3 "$PYTHON_SCRIPT" "$@"

echo ""
echo "✅ 瓦片下载完成！"
echo ""
echo "📁 瓦片文件保存在: $SCRIPT_DIR/../static/tiles/"
echo ""
echo "💡 提示:"
echo "   - 这些瓦片文件会被自动包含在 Docker 镜像中"
echo "   - 离线环境会优先使用本地瓦片，缺失时显示空白占位图"

