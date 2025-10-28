#!/bin/bash
# 调试版本的智能库文件复制脚本

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
    
    echo "ldd输出:"
    echo "$ldd_output"
    echo ""
    
    # 解析ldd输出，提取库文件路径
    local libs=()
    while IFS= read -r line; do
        echo "处理行: '$line'"
        # 检查是否包含=>符号（表示库文件映射）
        if echo "$line" | grep -q "=>"; then
            echo "  包含=>符号"
            # 提取库文件路径
            local lib_path
            lib_path=$(echo "$line" | sed -n 's/.*=>[[:space:]]*\([^[:space:]]*\).*/\1/p')
            echo "  提取的路径: '$lib_path'"
            
            # 跳过"not found"和空路径
            if [[ "$lib_path" != "not found" && -n "$lib_path" && "$lib_path" != "("* ]]; then
                echo "  添加到列表: '$lib_path'"
                libs+=("$lib_path")
            else
                echo "  跳过: '$lib_path'"
            fi
        else
            echo "  不包含=>符号"
        fi
    done <<< "$ldd_output"
    
    echo ""
    print_success "找到 ${#libs[@]} 个库依赖"
    
    for lib in "${libs[@]}"; do
        echo "  - $lib"
    done
    
    # 返回库文件数组
    printf '%s\n' "${libs[@]}"
}

# 主函数
main() {
    local binary_path="$1"
    
    if [ -z "$binary_path" ]; then
        print_error "请指定二进制文件路径"
        exit 1
    fi
    
    print_info "调试模式 - 分析二进制文件: $binary_path"
    
    # 分析库依赖
    local libs
    if ! libs=($(analyze_dependencies "$binary_path")); then
        print_error "无法分析库依赖"
        exit 1
    fi
    
    if [ ${#libs[@]} -eq 0 ]; then
        print_warning "未找到需要复制的库文件"
        exit 0
    fi
    
    print_success "调试完成！"
}

# 运行主函数
main "$@"
