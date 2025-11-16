#!/usr/bin/env bash
# Plum Controller 与 Nginx 停止脚本
#
# 用法:
#   docker/stop-controller.sh [--purge]
#     --purge: 额外清理数据（删除数据卷 plum-controller-data）
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

log_info "停止容器 plum-controller（若存在）..."
docker stop plum-controller 2>/dev/null || true
log_info "停止容器 plum-nginx（若存在）..."
docker stop plum-nginx 2>/dev/null || true

log_info "删除容器 plum-controller（若存在）..."
docker rm plum-controller 2>/dev/null || true
log_info "删除容器 plum-nginx（若存在）..."
docker rm plum-nginx 2>/dev/null || true

if $PURGE; then
    log_warn "--purge 指定，开始清理数据..."
    if docker volume ls | grep -q "plum-controller-data"; then
        log_info "删除数据卷 plum-controller-data..."
        docker volume rm plum-controller-data 2>/dev/null || true
    else
        log_info "数据卷 plum-controller-data 不存在，跳过"
    fi
fi

log_info "✅ Controller 与 Nginx 已停止"
if $PURGE; then
    log_info "✅ 数据已清理（删除数据卷）"
fi


