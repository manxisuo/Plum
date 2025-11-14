#!/usr/bin/env bash
# 下载 SimDecision 所需 Python 依赖的 ARM64 离线安装包

set -euo pipefail

TARGET_DIR="offline-pip-packages"
# Ubuntu 24.04 使用 Python 3.12，但也可以使用 3.11 的包（兼容）
PYTHON_VERSION="${PYTHON_VERSION:-312}"
PLATFORM="${PLATFORM:-manylinux2014_aarch64}"

echo "🚀 下载 Flask 与 Requests 的离线包（目标平台: ${PLATFORM}, Python 版本: ${PYTHON_VERSION})"
mkdir -p "${TARGET_DIR}"

pip download \
  --platform "${PLATFORM}" \
  --python-version "${PYTHON_VERSION}" \
  --implementation cp \
  --only-binary=:all: \
  --dest "${TARGET_DIR}" \
  flask==3.0.0 \
  requests==2.31.0 \
  importlib-metadata==6.6.0 \
  charset-normalizer==3.3.2

echo "✅ 下载完成，文件保存在 ${TARGET_DIR}/"

