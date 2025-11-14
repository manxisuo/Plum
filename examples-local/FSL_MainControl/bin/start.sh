#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

export PYTHONUNBUFFERED=1
export PLAN_SERVICE_BASE="${PLAN_SERVICE_BASE:-http://127.0.0.1:4100}"

exec uvicorn app:app --host 0.0.0.0 --port 4000 --app-dir "${APP_ROOT}"

