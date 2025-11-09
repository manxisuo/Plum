#!/usr/bin/env bash
# 下载 get-pip.py 用于离线安装 pip

set -euo pipefail

TARGET_DIR="plum-offline-deploy/tools"

mkdir -p "${TARGET_DIR}"
echo "🚀 下载适用于 Python 3.8 的 get-pip.py 到 ${TARGET_DIR}"
curl -fsSL https://bootstrap.pypa.io/pip/3.8/get-pip.py -o "${TARGET_DIR}/get-pip.py"
echo "✅ 已下载: ${TARGET_DIR}/get-pip.py (Python 3.8 兼容版)"

