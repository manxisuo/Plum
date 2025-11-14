#!/usr/bin/env bash
# SimDecision 启动脚本

set -euo pipefail

# 在 Docker 容器中，start.sh 在 /app/start.sh，app.py 在 /app/app.py
# Dockerfile 已经设置了 WORKDIR /app，所以直接使用当前目录
cd /app

export PYTHONUNBUFFERED=1
export FLASK_APP=app.py
export FLASK_ENV=production

# 启动应用
exec python3 app.py
