#!/usr/bin/env bash
# 下载 get-pip.py 用于离线安装 pip

set -euo pipefail

TARGET_DIR="plum-offline-deploy/tools"

mkdir -p "${TARGET_DIR}"
echo "🚀 下载 get-pip.py 到 ${TARGET_DIR}"
curl -fsSL https://bootstrap.pypa.io/get-pip.py -o "${TARGET_DIR}/get-pip.py"
echo "✅ 已下载: ${TARGET_DIR}/get-pip.py"

