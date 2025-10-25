#!/bin/bash

# Plum Docker 部署脚本
# 用法: ./deploy.sh [环境] [操作]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    cat << EOF
Plum Docker 部署脚本

用法:
    $0 [环境] [操作]

环境:
    test        - 测试环境（Controller + 3个Agent）
    test-simple - 简单测试环境（仅Controller）
    test-nginx  - 测试环境（包含nginx）
    production  - 生产环境
    controller  - 仅启动Controller
    agent       - 仅启动Agent

操作:
    start       - 启动服务（默认）
    stop        - 停止服务
    restart     - 重启服务
    status      - 查看状态
    logs        - 查看日志
    clean       - 清理资源
    backup      - 备份数据
    restore     - 恢复数据

示例:
    $0 test start          # 启动测试环境
    $0 production stop     # 停止生产环境
    $0 test-simple status  # 查看简单测试环境状态
    $0 test logs           # 查看测试环境日志
    $0 clean               # 清理所有资源

EOF
}

# 检查Docker环境
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose 未安装"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        print_error "Docker 服务未运行"
        exit 1
    fi
}

# 启动服务
start_services() {
    local env=$1
    print_info "启动 $env 环境..."

    case $env in
        "test")
            docker-compose up -d
            ;;
        "test-simple")
            docker-compose -f docker-compose.controller-test-simple.yml up -d
            ;;
        "test-nginx")
            docker-compose --profile nginx up -d
            ;;
        "production")
            docker-compose -f docker-compose.production.yml up -d
            ;;
        "controller")
            docker-compose up -d plum-controller
            ;;
        "agent")
            docker-compose up -d plum-agent-a plum-agent-b plum-agent-c
            ;;
        *)
            print_error "未知环境: $env"
            exit 1
            ;;
    esac

    print_success "$env 环境启动完成"
}

# 停止服务
stop_services() {
    local env=$1
    print_info "停止 $env 环境..."

    case $env in
        "test")
            docker-compose down
            ;;
        "test-simple")
            docker-compose -f docker-compose.controller-test-simple.yml down
            ;;
        "test-nginx")
            docker-compose --profile nginx down
            ;;
        "production")
            docker-compose -f docker-compose.production.yml down
            ;;
        "controller")
            docker-compose stop plum-controller
            ;;
        "agent")
            docker-compose stop plum-agent-a plum-agent-b plum-agent-c
            ;;
        *)
            print_error "未知环境: $env"
            exit 1
            ;;
    esac

    print_success "$env 环境停止完成"
}

# 重启服务
restart_services() {
    local env=$1
    print_info "重启 $env 环境..."
    stop_services $env
    sleep 2
    start_services $env
}

# 查看状态
show_status() {
    local env=$1
    print_info "查看 $env 环境状态..."

    case $env in
        "test")
            docker-compose ps
            ;;
        "test-simple")
            docker-compose -f docker-compose.controller-test-simple.yml ps
            ;;
        "test-nginx")
            docker-compose --profile nginx ps
            ;;
        "production")
            docker-compose -f docker-compose.production.yml ps
            ;;
        "controller")
            docker-compose ps plum-controller
            ;;
        "agent")
            docker-compose ps plum-agent-a plum-agent-b plum-agent-c
            ;;
        *)
            print_error "未知环境: $env"
            exit 1
            ;;
    esac
}

# 查看日志
show_logs() {
    local env=$1
    print_info "查看 $env 环境日志..."

    case $env in
        "test")
            docker-compose logs -f
            ;;
        "test-simple")
            docker-compose -f docker-compose.controller-test-simple.yml logs -f
            ;;
        "test-nginx")
            docker-compose --profile nginx logs -f
            ;;
        "production")
            docker-compose -f docker-compose.production.yml logs -f
            ;;
        "controller")
            docker-compose logs -f plum-controller
            ;;
        "agent")
            docker-compose logs -f plum-agent-a plum-agent-b plum-agent-c
            ;;
        *)
            print_error "未知环境: $env"
            exit 1
            ;;
    esac
}

# 清理资源
clean_resources() {
    print_info "清理Docker资源..."

    # 停止所有相关容器
    docker-compose down 2>/dev/null || true
    docker-compose -f docker-compose.controller-test-simple.yml down 2>/dev/null || true
    docker-compose --profile nginx down 2>/dev/null || true
    docker-compose -f docker-compose.production.yml down 2>/dev/null || true

    # 清理未使用的资源
    docker system prune -f
    docker volume prune -f
    docker network prune -f

    print_success "资源清理完成"
}

# 备份数据
backup_data() {
    local backup_dir="./backups"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="plum_backup_${timestamp}.tar.gz"

    print_info "备份数据到 $backup_dir/$backup_file..."

    mkdir -p $backup_dir

    # 备份Controller数据
    if docker volume ls | grep -q plum_plum-data; then
        docker run --rm \
            -v plum_plum-data:/data \
            -v $(pwd)/$backup_dir:/backup \
            alpine tar czf /backup/$backup_file -C /data .
        print_success "数据备份完成: $backup_dir/$backup_file"
    else
        print_warning "未找到数据卷，跳过备份"
    fi
}

# 恢复数据
restore_data() {
    local backup_file=$1

    if [ -z "$backup_file" ]; then
        print_error "请指定备份文件"
        exit 1
    fi

    if [ ! -f "$backup_file" ]; then
        print_error "备份文件不存在: $backup_file"
        exit 1
    fi

    print_info "从 $backup_file 恢复数据..."

    # 停止服务
    docker-compose down 2>/dev/null || true

    # 恢复数据
    docker run --rm \
        -v plum_plum-data:/data \
        -v $(pwd):/backup \
        alpine tar xzf /backup/$backup_file -C /data

    print_success "数据恢复完成"
}

# 健康检查
health_check() {
    print_info "执行健康检查..."

    # 检查Controller
    if curl -s http://localhost:8080/v1/nodes > /dev/null; then
        print_success "Controller 健康检查通过"
    else
        print_error "Controller 健康检查失败"
    fi

    # 检查nginx（如果运行）
    if curl -s http://localhost/health > /dev/null; then
        print_success "nginx 健康检查通过"
    else
        print_warning "nginx 未运行或健康检查失败"
    fi
}

# 主函数
main() {
    # 检查是否是帮助请求
    if [ "$1" = "help" ] || [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
        show_help
        exit 0
    fi

    local env=${1:-"test"}
    local action=${2:-"start"}

    # 检查Docker环境
    check_docker

    case $action in
        "start")
            start_services $env
            sleep 5
            health_check
            ;;
        "stop")
            stop_services $env
            ;;
        "restart")
            restart_services $env
            sleep 5
            health_check
            ;;
        "status")
            show_status $env
            ;;
        "logs")
            show_logs $env
            ;;
        "clean")
            clean_resources
            ;;
        "backup")
            backup_data
            ;;
        "restore")
            restore_data $3
            ;;
        "health")
            health_check
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_error "未知操作: $action"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"