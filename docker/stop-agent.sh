#!/usr/bin/env bash
# Plum Agent 停止脚本
#
# 用法:
#   docker/stop-agent.sh [--purge]
#     --purge: 额外清理数据（删除数据卷 plum-agent-data 与临时目录 /tmp/plum-agent）
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

PURGE=false
if [[ "${1:-}" == "--purge" ]]; then
    PURGE=true
fi

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    log_error "Docker 未运行"
    exit 1
fi

log_info "停止容器 plum-agent（若存在）..."
docker stop plum-agent 2>/dev/null || true

log_info "删除容器 plum-agent（若存在）..."
docker rm plum-agent 2>/dev/null || true

if $PURGE; then
    log_warn "--purge 指定，开始清理数据..."
    if docker volume ls | grep -q "plum-agent-data"; then
        log_info "删除数据卷 plum-agent-data..."
        docker volume rm plum-agent-data 2>/dev/null || true
    else
        log_info "数据卷 plum-agent-data 不存在，跳过"
    fi
    if [ -d "/tmp/plum-agent" ]; then
        log_info "删除临时目录 /tmp/plum-agent..."
        rm -rf /tmp/plum-agent || true
    else
        log_info "临时目录 /tmp/plum-agent 不存在，跳过"
    fi
fi

log_info "✅ Agent 已停止"
if $PURGE; then
    log_info "✅ 数据已清理（包括数据卷与临时目录）"
fi


