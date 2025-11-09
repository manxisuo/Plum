#!/usr/bin/env bash
# 在目标环境离线安装 pip（兼容 Python 3.8）

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

GET_PIP="${ROOT_DIR}/tools/get-pip.py"
WHEEL_DIR="${ROOT_DIR}/tools/pip-packages"

if [ ! -f "$GET_PIP" ]; then
    echo "❌ 未找到 get-pip.py: $GET_PIP"
    echo "   请先在联网环境运行 scripts-prepare/download-pip.sh"
    exit 1
fi

if [ ! -d "$WHEEL_DIR" ]; then
    echo "❌ 未找到 pip 离线包目录: $WHEEL_DIR"
    echo "   请确认已运行 pip wheel 下载步骤（tools/pip-packages）"
    exit 1
fi

echo "🚀 使用离线包安装 pip ..."
python3 "$GET_PIP" --no-index --find-links "$WHEEL_DIR"
echo "✅ pip 离线安装完成"

