#!/bin/bash
# Qt5应用程序库文件复制脚本
# 专门用于复制Qt5应用程序所需的库文件到容器中

set -e

# 配置
CONTAINER_NAME="plum-agent-a"

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

# 显示使用说明
show_usage() {
    echo "用法: $0 [选项] <二进制文件路径>"
    echo ""
    echo "参数:"
    echo "  <二进制文件路径>    要分析的二进制文件路径"
    echo ""
    echo "选项:"
    echo "  -h, --help         显示此帮助信息"
    echo "  -c, --container    指定容器名称（默认: plum-agent-a）"
    echo "  -d, --dry-run      仅显示需要复制的库文件，不实际复制"
    echo ""
    echo "示例:"
    echo "  $0 ./HelloUI                           # 分析当前目录的HelloUI"
    echo "  $0 /data/usershare/应用包/HelloUI/bin/HelloUI  # 分析指定路径的HelloUI"
    echo "  $0 -c plum-agent-b ./HelloUI          # 复制到指定容器"
    echo "  $0 -d ./HelloUI                       # 仅显示需要复制的库文件"
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

# 检查二进制文件是否存在
check_binary() {
    local binary_path="$1"
    
    if [ ! -f "$binary_path" ]; then
        print_error "二进制文件不存在: $binary_path"
        exit 1
    fi
    
    if [ ! -x "$binary_path" ]; then
        print_warning "二进制文件不可执行: $binary_path"
    fi
    
    print_success "找到二进制文件: $binary_path"
}

# 分析库依赖
analyze_dependencies() {
    local binary_path="$1"
    
    print_info "分析二进制文件库依赖: $binary_path"
    
    # 使用ldd分析依赖
    local ldd_output
    if ! ldd_output=$(ldd "$binary_path" 2>/dev/null); then
        print_error "无法分析二进制文件依赖，可能不是动态链接的可执行文件"
        return 1
    fi
    
    # 解析ldd输出，提取库文件路径
    local libs=()
    while IFS= read -r line; do
        # 检查是否包含=>符号（表示库文件映射）
        if echo "$line" | grep -q "=>"; then
            # 提取库文件路径
            local lib_path
            lib_path=$(echo "$line" | sed -n 's/.*=>[[:space:]]*\([^[:space:]]*\).*/\1/p')
            
            # 跳过"not found"和空路径
            if [[ "$lib_path" != "not found" && -n "$lib_path" && "$lib_path" != "("* ]]; then
                libs+=("$lib_path")
            fi
        fi
    done <<< "$ldd_output"
    
    if [ ${#libs[@]} -eq 0 ]; then
        print_warning "未找到动态库依赖"
        return 1
    fi
    
    print_success "找到 ${#libs[@]} 个库依赖"
    
    # 返回库文件数组
    printf '%s\n' "${libs[@]}"
}

# 复制库文件到容器
copy_libs_to_container() {
    local dry_run="$1"
    shift
    local libs=("$@")
    
    print_info "开始复制库文件到容器 $CONTAINER_NAME"
    
    local copied=0
    local failed=0
    
    for lib_path in "${libs[@]}"; do
        if [ ! -f "$lib_path" ]; then
            print_warning "库文件不存在: $lib_path"
            ((failed++))
            continue
        fi
        
        local lib_name=$(basename "$lib_path")
        local target_dir="/lib"
        
        # 如果是/usr/lib下的库，复制到容器的/usr/lib
        if [[ "$lib_path" == "/usr/lib"* ]]; then
            target_dir="/usr/lib"
        fi
        
        if [ "$dry_run" = "true" ]; then
            print_info "[DRY-RUN] 将复制 $lib_name 到 $target_dir"
        else
            print_info "复制 $lib_name 到 $target_dir"
            if docker cp "$lib_path" "$CONTAINER_NAME:$target_dir/"; then
                print_success "✓ $lib_name 复制成功"
                ((copied++))
            else
                print_error "✗ $lib_name 复制失败"
                ((failed++))
            fi
        fi
    done
    
    if [ "$dry_run" = "true" ]; then
        print_info "[DRY-RUN] 将复制 $copied 个库文件"
    else
        print_success "库文件复制完成: 成功 $copied 个，失败 $failed 个"
    fi
}

# 设置执行权限
set_permissions() {
    local dry_run="$1"
    
    if [ "$dry_run" = "true" ]; then
        print_info "[DRY-RUN] 将设置 /lib/ld-linux-aarch64.so.1 执行权限"
        return
    fi
    
    print_info "设置执行权限..."
    if docker exec -it "$CONTAINER_NAME" chmod +x /lib/ld-linux-aarch64.so.1 2>/dev/null; then
        print_success "执行权限设置完成"
    else
        print_warning "无法设置执行权限，可能文件不存在"
    fi
}

# 验证库文件
verify_libs() {
    local dry_run="$1"
    shift
    local libs=("$@")
    
    if [ "$dry_run" = "true" ]; then
        print_info "[DRY-RUN] 将验证 ${#libs[@]} 个库文件"
        return
    fi
    
    print_info "验证库文件..."
    
    local verified=0
    for lib_path in "${libs[@]}"; do
        local lib_name=$(basename "$lib_path")
        local target_dir="/lib"
        
        if [[ "$lib_path" == "/usr/lib"* ]]; then
            target_dir="/usr/lib"
        fi
        
        if docker exec -it "$CONTAINER_NAME" test -f "$target_dir/$lib_name" 2>/dev/null; then
            print_success "✓ $lib_name 存在"
            ((verified++))
        else
            print_warning "✗ $lib_name 不存在"
        fi
    done
    
    print_success "库文件验证完成: $verified/${#libs[@]} 个存在"
}

# 主函数
main() {
    local binary_path=""
    local container_name="$CONTAINER_NAME"
    local dry_run="false"
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -c|--container)
                container_name="$2"
                shift 2
                ;;
            -d|--dry-run)
                dry_run="true"
                shift
                ;;
            -*)
                print_error "未知选项: $1"
                show_usage
                exit 1
                ;;
            *)
                binary_path="$1"
                shift
                ;;
        esac
    done
    
    if [ -z "$binary_path" ]; then
        print_error "请指定二进制文件路径"
        show_usage
        exit 1
    fi
    
    CONTAINER_NAME="$container_name"
    
    print_info "Qt5应用程序库文件复制脚本"
    print_info "容器: $CONTAINER_NAME"
    print_info "二进制文件: $binary_path"
    print_info "模式: $([ "$dry_run" = "true" ] && echo "DRY-RUN" || echo "实际复制")"
    echo ""
    
    # 检查容器
    check_container
    
    # 检查二进制文件
    check_binary "$binary_path"
    
    # 分析库依赖
    print_info "开始分析库依赖..."
    local libs=()
    
    # 直接使用ldd分析依赖
    local ldd_output
    if ! ldd_output=$(ldd "$binary_path" 2>/dev/null); then
        print_error "无法分析二进制文件依赖，可能不是动态链接的可执行文件"
        exit 1
    fi
    
    # 解析ldd输出，提取库文件路径
    while IFS= read -r line; do
        # 检查是否包含=>符号（表示库文件映射）
        if echo "$line" | grep -q "=>"; then
            # 提取库文件路径
            local lib_path
            lib_path=$(echo "$line" | sed -n 's/.*=>[[:space:]]*\([^[:space:]]*\).*/\1/p')
            
            # 跳过"not found"和空路径
            if [[ "$lib_path" != "not found" && -n "$lib_path" && "$lib_path" != "("* ]]; then
                libs+=("$lib_path")
                print_info "找到库文件: $lib_path"
            fi
        fi
    done <<< "$ldd_output"
    
    print_success "总共找到 ${#libs[@]} 个库文件"
    
    if [ ${#libs[@]} -eq 0 ]; then
        print_warning "未找到需要复制的库文件"
        exit 0
    fi
    
    # 复制库文件
    copy_libs_to_container "$dry_run" "${libs[@]}"
    
    # 设置权限
    set_permissions "$dry_run"
    
    # 验证库文件
    verify_libs "$dry_run" "${libs[@]}"
    
    echo ""
    if [ "$dry_run" = "true" ]; then
        print_info "DRY-RUN 完成！使用 -d 选项查看需要复制的库文件"
    else
        print_success "Qt5应用程序库文件复制完成！"
        print_info "现在可以测试应用运行："
        print_info "  docker exec -it $CONTAINER_NAME sh"
        print_info "  cd $(dirname "$binary_path")"
        print_info "  ./$(basename "$binary_path")"
    fi
}

# 运行主函数
main "$@"
