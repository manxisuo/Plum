#!/usr/bin/env bash
# 在离线环境安装 SimDecision 所需的 Python 依赖

set -euo pipefail

PACKAGE_DIR="${1:-offline-pip-packages}"

if [ ! -d "${PACKAGE_DIR}" ]; then
  echo "❌ 未找到包目录: ${PACKAGE_DIR}"
  echo "请先在联网环境运行 download_dependencies.sh"
  exit 1
fi

echo "🚀 使用 pip 安装离线包（来源: ${PACKAGE_DIR}）"
pip install --no-index --find-links="${PACKAGE_DIR}" importlib-metadata==6.6.0 flask==3.0.0 requests==2.31.0
echo "✅ 安装完成"

