#!/usr/bin/env bash
# Plum Controller 和 Nginx 启动脚本
# 等价于 docker-compose.main.yml

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

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

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    log_error "Docker 未运行，请先启动 Docker"
    exit 1
fi

# 检查镜像是否存在
if ! docker images | grep -q "plum-controller.*offline"; then
    log_error "镜像 plum-controller:offline 不存在，请先构建镜像"
    exit 1
fi

if ! docker images | grep -q "nginx.*alpine"; then
    log_warn "镜像 nginx:alpine 不存在，正在拉取..."
    docker pull nginx:alpine || {
        log_error "拉取 nginx:alpine 失败"
        exit 1
    }
fi

# 创建数据卷（如果不存在）
log_info "检查数据卷..."
if ! docker volume ls | grep -q "plum-controller-data"; then
    log_info "创建数据卷 plum-controller-data..."
    docker volume create plum-controller-data
fi

# 检查 .env 文件
ENV_FILE="${PROJECT_ROOT}/controller/.env"
if [ ! -f "${ENV_FILE}" ]; then
    log_warn ".env 文件不存在: ${ENV_FILE}"
    log_warn "将使用默认环境变量"
    ENV_FILE=""
fi

# 停止并删除已存在的容器
log_info "清理已存在的容器..."
docker stop plum-controller plum-nginx 2>/dev/null || true
docker rm plum-controller plum-nginx 2>/dev/null || true

# 查找 docker 可执行文件路径（用于挂载到容器内）
DOCKER_BIN=$(command -v docker 2>/dev/null || echo "")
if [ -z "$DOCKER_BIN" ]; then
    # 尝试常见路径
    for path in /usr/bin/docker /usr/local/bin/docker /opt/docker/bin/docker; do
        if [ -f "$path" ]; then
            DOCKER_BIN="$path"
            break
        fi
    done
fi

if [ -z "$DOCKER_BIN" ] || [ ! -f "$DOCKER_BIN" ]; then
    log_warn "未找到 docker 可执行文件，Controller 将无法列出 Docker 镜像"
    log_warn "如果需要在界面中查看镜像列表，请确保 docker 命令可用"
    DOCKER_BIN=""
fi

# 启动 Controller
log_info "启动 Controller..."
CONTROLLER_CMD=(
    docker run -d
    --name plum-controller
    --network host
    --restart unless-stopped
    --health-cmd "wget --no-verbose --tries=1 --spider http://localhost:8080/v1/nodes || exit 1"
    --health-interval 30s
    --health-timeout 10s
    --health-retries 3
    -v plum-controller-data:/app/data
    -e CONTROLLER_ADDR=:8080
    -e CONTROLLER_DB=file:/app/data/controller.db?_pragma=busy_timeout\(5000\)
    -e CONTROLLER_DATA_DIR=/app/data
    -e HEARTBEAT_TTL_SEC=3
    -e AUTO_MIGRATION_ENABLED=false
)

# 挂载 docker 可执行文件和 socket（如果找到）
if [ -n "$DOCKER_BIN" ]; then
    CONTROLLER_CMD+=(-v "${DOCKER_BIN}:/usr/bin/docker:ro")
    log_info "挂载 docker 可执行文件: $DOCKER_BIN"
    
    # 挂载 docker socket（如果存在）
    if [ -S /var/run/docker.sock ]; then
        CONTROLLER_CMD+=(-v /var/run/docker.sock:/var/run/docker.sock)
        log_info "挂载 docker socket: /var/run/docker.sock"
    fi
fi

if [ -n "${ENV_FILE}" ]; then
    CONTROLLER_CMD+=(-v "${ENV_FILE}:/app/.env:ro")
fi

CONTROLLER_CMD+=(plum-controller:offline)

"${CONTROLLER_CMD[@]}"

# 等待 Controller 启动
log_info "等待 Controller 启动..."
sleep 3

# 检查 Controller 是否运行
if ! docker ps | grep -q "plum-controller"; then
    log_error "Controller 启动失败"
    docker logs plum-controller
    exit 1
fi

# 检查 nginx 配置文件（优先使用 host 模式配置）
NGINX_CONF="${PROJECT_ROOT}/docker/nginx/nginx.conf.host"
if [ ! -f "${NGINX_CONF}" ]; then
    # 回退到标准配置
    NGINX_CONF="${PROJECT_ROOT}/docker/nginx/nginx.conf"
    if [ ! -f "${NGINX_CONF}" ]; then
        log_error "Nginx 配置文件不存在: ${NGINX_CONF}"
        exit 1
    fi
    log_warn "使用标准 nginx.conf，建议使用 nginx.conf.host（适用于 host 网络模式）"
fi

# 检查 UI 静态文件目录
UI_DIST="${PROJECT_ROOT}/ui/dist"
if [ ! -d "${UI_DIST}" ]; then
    log_warn "UI 静态文件目录不存在: ${UI_DIST}"
    log_warn "Nginx 将无法提供前端页面"
fi

# 启动 Nginx
log_info "启动 Nginx..."
NGINX_CMD=(
    docker run -d
    --name plum-nginx
    --network host
    --restart unless-stopped
    -v "${NGINX_CONF}:/etc/nginx/nginx.conf:ro"
)

if [ -d "${UI_DIST}" ]; then
    NGINX_CMD+=(-v "${UI_DIST}:/usr/share/nginx/html:ro")
fi

NGINX_CMD+=(nginx:alpine)

"${NGINX_CMD[@]}"

# 等待 Nginx 启动
log_info "等待 Nginx 启动..."
sleep 2

# 检查 Nginx 是否运行
if ! docker ps | grep -q "plum-nginx"; then
    log_error "Nginx 启动失败"
    docker logs plum-nginx
    exit 1
fi

log_info "✅ Controller 和 Nginx 启动成功！"
log_info "Controller: http://localhost:8080"
log_info "Web UI: http://localhost:80"
log_info ""
log_info "查看日志:"
log_info "  Controller: docker logs -f plum-controller"
log_info "  Nginx: docker logs -f plum-nginx"
log_info ""
log_info "停止服务:"
log_info "  docker stop plum-controller plum-nginx"
log_info "  docker rm plum-controller plum-nginx"

