#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

export LD_LIBRARY_PATH="${APP_ROOT}/lib:${LD_LIBRARY_PATH:-}"
export QT_QPA_PLATFORM=offscreen
export WORKER_ID="${WORKER_ID:-fsl-plan-dev}"
export WORKER_NODE_ID="${WORKER_NODE_ID:-nodeA}"
export PLUM_INSTANCE_ID="${PLUM_INSTANCE_ID:-fsl-plan-001}"
export PLUM_APP_NAME="${PLUM_APP_NAME:-FSL_Plan}"
export PLUM_APP_VERSION="${PLUM_APP_VERSION:-1.0.0}"

exec "${SCRIPT_DIR}/FSL_Plan"

