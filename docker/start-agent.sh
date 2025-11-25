#!/usr/bin/env bash
# Plum Agent 启动脚本
# 等价于 docker-compose.agent.yml

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

# 检查镜像是否存在（使用 docker image inspect 更可靠）
if ! docker image inspect plum-agent:offline > /dev/null 2>&1; then
    log_error "镜像 plum-agent:offline 不存在，请先构建镜像"
    exit 1
fi

# 创建数据卷（如果不存在）
log_info "检查数据卷..."
if ! docker volume ls | grep -q "plum-agent-data"; then
    log_info "创建数据卷 plum-agent-data..."
    docker volume create plum-agent-data
fi

# 检查 .env 文件
ENV_FILE="${PROJECT_ROOT}/agent-go/.env"
if [ ! -f "${ENV_FILE}" ]; then
    log_warn ".env 文件不存在: ${ENV_FILE}"
    log_warn "将使用默认环境变量"
    ENV_FILE=""
fi

# 检查 Docker socket
if [ ! -S /var/run/docker.sock ]; then
    log_error "Docker socket 不存在: /var/run/docker.sock"
    log_error "Agent 需要访问 Docker socket 来管理应用容器"
    exit 1
fi

# 停止并删除已存在的容器
log_info "清理已存在的容器..."
docker stop plum-agent 2>/dev/null || true
docker rm plum-agent 2>/dev/null || true

# 检测系统架构，确定库路径
ARCH=$(uname -m)
log_info "检测到系统架构: ${ARCH}"

# 根据架构设置库路径
case "${ARCH}" in
    x86_64)
        LIB_PATHS=(
            "/usr/lib:/usr/lib:ro"
            "/usr/local/lib:/usr/local/lib:ro"
            "/usr/lib/x86_64-linux-gnu:/usr/lib/x86_64-linux-gnu:ro"
        )
        ;;
    aarch64|arm64)
        # 检查是否是银河麒麟系统
        if [ -d "/usr/lib/aarch64-linux-gnu" ]; then
            LIB_PATHS=(
                "/usr/lib:/usr/lib:ro"
                "/usr/local/lib:/usr/local/lib:ro"
                "/usr/lib/aarch64-linux-gnu:/usr/lib/aarch64-linux-gnu:ro"
            )
        # 检查是否是欧拉系统
        elif [ -d "/usr/lib64" ]; then
            LIB_PATHS=(
                "/usr/lib:/usr/lib:ro"
                "/usr/lib64:/usr/lib64:ro"
            )
        else
            LIB_PATHS=(
                "/usr/lib:/usr/lib:ro"
                "/usr/local/lib:/usr/local/lib:ro"
            )
        fi
        ;;
    *)
        log_warn "未知架构 ${ARCH}，使用默认库路径"
        LIB_PATHS=(
            "/usr/lib:/usr/lib:ro"
            "/usr/local/lib:/usr/local/lib:ro"
        )
        ;;
esac

# 启动 Agent
log_info "启动 Agent..."
AGENT_CMD=(
    docker run -d
    --name plum-agent
    --network host
    --user "0"
    --restart unless-stopped
    --health-cmd "pgrep plum-agent || exit 1"
    --health-interval 30s
    --health-timeout 10s
    --health-retries 3
    -v plum-agent-data:/app/data
    -v /var/run/docker.sock:/var/run/docker.sock
    -v /tmp/plum-agent:/tmp/plum-agent
)

# 添加库路径挂载
for lib_path in "${LIB_PATHS[@]}"; do
    AGENT_CMD+=(-v "${lib_path}")
done

# 添加 .env 文件挂载
if [ -n "${ENV_FILE}" ]; then
    AGENT_CMD+=(-v "${ENV_FILE}:/app/.env:ro")
fi

AGENT_CMD+=(plum-agent:offline)

"${AGENT_CMD[@]}"

# 等待 Agent 启动
log_info "等待 Agent 启动..."
sleep 2

# 检查 Agent 是否运行
if ! docker ps | grep -q "plum-agent"; then
    log_error "Agent 启动失败"
    docker logs plum-agent
    exit 1
fi

log_info "✅ Agent 启动成功！"
log_info ""
log_info "查看日志:"
log_info "  docker logs -f plum-agent"
log_info ""
log_info "停止服务:"
log_info "  docker stop plum-agent"
log_info "  docker rm plum-agent"

