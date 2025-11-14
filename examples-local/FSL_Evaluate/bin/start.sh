#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export WORKER_ID="${WORKER_ID:-fsl-evaluate-dev}"
export WORKER_NODE_ID="${WORKER_NODE_ID:-nodeA}"
export PLUM_INSTANCE_ID="${PLUM_INSTANCE_ID:-fsl-evaluate-001}"
export PLUM_APP_NAME="${PLUM_APP_NAME:-FSL_Evaluate}"
export PLUM_APP_VERSION="${PLUM_APP_VERSION:-1.0.0}"

exec "${SCRIPT_DIR}/FSL_Evaluate"

