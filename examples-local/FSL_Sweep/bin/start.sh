#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

export LD_LIBRARY_PATH="${APP_ROOT}/lib:${LD_LIBRARY_PATH:-}"
export QT_QPA_PLATFORM=offscreen
# WORKER_ID 应该使用 PLUM_INSTANCE_ID，确保每个实例都有唯一的 Worker ID
# 如果未设置，使用 PLUM_INSTANCE_ID（Agent 会自动注入）
export WORKER_ID="${WORKER_ID:-${PLUM_INSTANCE_ID:-fsl-sweep-dev}}"
# WORKER_NODE_ID 由 Agent 自动注入，不需要默认值
# 如果未设置，应该报错而不是使用错误的默认值
if [ -z "${WORKER_NODE_ID:-}" ]; then
    echo "错误: WORKER_NODE_ID 未设置，Agent 应该自动注入此环境变量" >&2
    exit 1
fi
export WORKER_NODE_ID
export PLUM_INSTANCE_ID="${PLUM_INSTANCE_ID:-fsl-sweep-001}"
export PLUM_APP_NAME="${PLUM_APP_NAME:-FSL_Sweep}"
export PLUM_APP_VERSION="${PLUM_APP_VERSION:-1.0.0}"

exec "${SCRIPT_DIR}/FSL_Sweep"

