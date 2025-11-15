#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 在容器中，start.sh 在 /app/start.sh，app.py 在 /app/app.py
# 在本地，start.sh 在 bin/start.sh，app.py 在项目根目录
# 所以 APP_ROOT 应该是 start.sh 所在目录（容器中）或父目录（本地）
if [ -f "${SCRIPT_DIR}/../app.py" ]; then
    # 本地环境：start.sh 在 bin/，app.py 在项目根目录
    APP_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
else
    # 容器环境：start.sh 和 app.py 都在 /app
    APP_ROOT="${SCRIPT_DIR}"
fi
VENV_DIR="${APP_ROOT}/venv"

export PYTHONUNBUFFERED=1
export PLAN_SERVICE_BASE="${PLAN_SERVICE_BASE:-http://127.0.0.1:4100}"

# 如果存在虚拟环境，使用虚拟环境中的 Python
if [ -d "${VENV_DIR}" ] && [ -f "${VENV_DIR}/bin/uvicorn" ]; then
    exec "${VENV_DIR}/bin/uvicorn" app:app --host 0.0.0.0 --port 4000 --app-dir "${APP_ROOT}"
else
    # 回退到系统 Python（容器中依赖已通过 Dockerfile 安装）
    exec uvicorn app:app --host 0.0.0.0 --port 4000 --app-dir "${APP_ROOT}"
fi

