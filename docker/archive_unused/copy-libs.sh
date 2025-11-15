#!/bin/bash
# 动态库文件复制脚本
# 用于在离线ARM64环境中将必要的动态链接库复制到容器中

set -e

# 配置
CONTAINER_NAME="plum-agent-a"
TARGET_LIB_DIR="/lib"
TARGET_USR_LIB_DIR="/usr/lib"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 检查容器是否存在
check_container() {
    if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        print_error "容器 $CONTAINER_NAME 不存在或未运行"
        print_info "请先启动容器：docker-compose -f docker-compose.offline.yml up -d"
        exit 1
    fi
    print_success "容器 $CONTAINER_NAME 正在运行"
}

# 复制库文件
copy_lib() {
    local source_path="$1"
    local target_dir="$2"
    local lib_name=$(basename "$source_path")
    
    if [ -f "$source_path" ]; then
        print_info "复制 $lib_name 到 $target_dir"
        docker cp "$source_path" "$CONTAINER_NAME:$target_dir/"
        return 0
    else
        print_warning "未找到 $source_path"
        return 1
    fi
}

# 复制基础系统库
copy_basic_libs() {
    print_info "复制基础系统库..."
    
    local basic_libs=(
        "/lib/libpthread.so.0"
        "/lib/libc.so.6"
        "/lib/ld-linux-aarch64.so.1"
    )
    
    local copied=0
    for lib in "${basic_libs[@]}"; do
        if copy_lib "$lib" "$TARGET_LIB_DIR"; then
            ((copied++))
        fi
    done
    
    print_success "基础系统库复制完成 ($copied/3)"
}

# 复制扩展系统库
copy_extended_libs() {
    print_info "复制扩展系统库..."
    
    local extended_libs=(
        "/lib/libm.so.6"
        "/lib/libdl.so.2"
        "/lib/libgcc_s.so.1"
        "/lib/libstdc++.so.6"
    )
    
    local copied=0
    for lib in "${extended_libs[@]}"; do
        if copy_lib "$lib" "$TARGET_LIB_DIR"; then
            ((copied++))
        fi
    done
    
    print_success "扩展系统库复制完成 ($copied/4)"
}

# 复制网络和加密库
copy_network_libs() {
    print_info "复制网络和加密库..."
    
    local network_libs=(
        "/usr/lib/libssl.so.1.1"
        "/usr/lib/libcrypto.so.1.1"
        "/usr/lib/libz.so.1"
    )
    
    local copied=0
    for lib in "${network_libs[@]}"; do
        if copy_lib "$lib" "$TARGET_USR_LIB_DIR"; then
            ((copied++))
        fi
    done
    
    print_success "网络和加密库复制完成 ($copied/3)"
}

# 设置执行权限
set_permissions() {
    print_info "设置执行权限..."
    docker exec -it "$CONTAINER_NAME" chmod +x /lib/ld-linux-aarch64.so.1
    print_success "执行权限设置完成"
}

# 验证库文件
verify_libs() {
    print_info "验证库文件..."
    
    local basic_libs=(
        "/lib/libpthread.so.0"
        "/lib/libc.so.6"
        "/lib/ld-linux-aarch64.so.1"
    )
    
    local verified=0
    for lib in "${basic_libs[@]}"; do
        if docker exec -it "$CONTAINER_NAME" test -f "$lib"; then
            print_success "✓ $lib 存在"
            ((verified++))
        else
            print_warning "✗ $lib 不存在"
        fi
    done
    
    print_success "库文件验证完成 ($verified/3)"
}

# 显示使用说明
show_usage() {
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help     显示此帮助信息"
    echo "  -b, --basic    仅复制基础系统库"
    echo "  -e, --extended 复制基础+扩展系统库"
    echo "  -n, --network  复制基础+扩展+网络库"
    echo "  -a, --all      复制所有库文件（默认）"
    echo "  -c, --container 指定容器名称（默认: plum-agent-a）"
    echo ""
    echo "示例:"
    echo "  $0                    # 复制所有库文件"
    echo "  $0 -b                 # 仅复制基础库文件"
    echo "  $0 -c plum-agent-b   # 复制到指定容器"
}

# 主函数
main() {
    local mode="all"
    local container_name="$CONTAINER_NAME"
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -b|--basic)
                mode="basic"
                shift
                ;;
            -e|--extended)
                mode="extended"
                shift
                ;;
            -n|--network)
                mode="network"
                shift
                ;;
            -a|--all)
                mode="all"
                shift
                ;;
            -c|--container)
                container_name="$2"
                shift 2
                ;;
            *)
                print_error "未知选项: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    CONTAINER_NAME="$container_name"
    
    print_info "开始复制动态库文件到容器 $CONTAINER_NAME"
    print_info "模式: $mode"
    echo ""
    
    # 检查容器
    check_container
    
    # 根据模式复制库文件
    case $mode in
        "basic")
            copy_basic_libs
            ;;
        "extended")
            copy_basic_libs
            copy_extended_libs
            ;;
        "network")
            copy_basic_libs
            copy_extended_libs
            copy_network_libs
            ;;
        "all")
            copy_basic_libs
            copy_extended_libs
            copy_network_libs
            ;;
    esac
    
    # 设置权限和验证
    set_permissions
    verify_libs
    
    echo ""
    print_success "动态库文件复制完成！"
    print_info "现在可以测试应用运行："
    print_info "  docker exec -it $CONTAINER_NAME sh"
    print_info "  cd /app/data/nodeA/.../app"
    print_info "  ./HelloUI"
}

# 运行主函数
main "$@"
