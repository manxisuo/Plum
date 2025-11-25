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
export CONTROLLER_BASE="${CONTROLLER_BASE:-http://plum-controller:8080}"

# 检查 app.py 是否存在
if [ ! -f "${APP_ROOT}/app.py" ]; then
    echo "错误: 找不到 app.py 文件 (APP_ROOT=${APP_ROOT})" >&2
    exit 1
fi

# 检查 uvicorn 命令是否存在
UVICORN_CMD=""
if [ -d "${VENV_DIR}" ] && [ -f "${VENV_DIR}/bin/uvicorn" ]; then
    UVICORN_CMD="${VENV_DIR}/bin/uvicorn"
elif command -v uvicorn >/dev/null 2>&1; then
    UVICORN_CMD="uvicorn"
else
    echo "错误: 找不到 uvicorn 命令" >&2
    echo "请确保已安装 uvicorn: pip install uvicorn" >&2
    exit 1
fi

# 启动 uvicorn（使用 exec 替换当前进程，确保信号正确处理）
echo "启动 FSL_MainControl..."
echo "APP_ROOT=${APP_ROOT}"
echo "UVICORN_CMD=${UVICORN_CMD}"
exec "${UVICORN_CMD}" app:app --host 0.0.0.0 --port 4000 --app-dir "${APP_ROOT}"

