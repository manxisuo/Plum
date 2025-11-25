#!/usr/bin/env bash
# 下载 FSL_MainControl 所需 Python 依赖的 ARM64 离线安装包

set -euo pipefail

TARGET_DIR="offline-pip-packages"
# Ubuntu 22.04 默认 Python 3.10，通常也兼容 3.11 的包
PYTHON_VERSION="${PYTHON_VERSION:-310}"
PLATFORM="${PLATFORM:-manylinux2014_aarch64}"

echo "🚀 下载 FSL_MainControl 的离线包（目标平台: ${PLATFORM}, Python 版本: ${PYTHON_VERSION})"
mkdir -p "${TARGET_DIR}"

pip download \
  --platform "${PLATFORM}" \
  --python-version "${PYTHON_VERSION}" \
  --implementation cp \
  --only-binary=:all: \
  --dest "${TARGET_DIR}" \
  fastapi==0.115.0 \
  uvicorn[standard]==0.30.3 \
  requests==2.31.0

echo "✅ 下载完成，文件保存在 ${TARGET_DIR}/"

