#!/bin/bash
# 加载离线Docker镜像脚本
# 用于在离线环境中加载预构建的Docker镜像

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 默认参数
IMAGES_DIR="offline-images"
TAG_SUFFIX=""

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--dir)
            IMAGES_DIR="$2"
            shift 2
            ;;
        -t|--tag)
            TAG_SUFFIX="$2"
            shift 2
            ;;
        -h|--help)
            echo "用法: $0 [选项]"
            echo "选项:"
            echo "  -d, --dir DIR     镜像文件目录 (默认: offline-images)"
            echo "  -t, --tag SUFFIX  镜像标签后缀 (默认: 无)"
            echo "  -h, --help        显示帮助信息"
            echo ""
            echo "示例:"
            echo "  $0                                    # 加载 offline-images/ 目录"
            echo "  $0 -d ./images -t offline             # 加载 ./images/ 目录，添加offline标签"
            echo "  $0 -d /path/to/images                 # 加载指定目录"
            exit 0
            ;;
        *)
            print_error "未知参数: $1"
            echo "使用 -h 或 --help 查看帮助信息"
            exit 1
            ;;
    esac
done

print_info "🚀 开始加载离线Docker镜像..."

# 检查目录是否存在
if [ ! -d "$IMAGES_DIR" ]; then
    print_error "镜像目录不存在: $IMAGES_DIR"
    exit 1
fi

# 检查是否有tar.gz文件
TAR_FILES=$(find "$IMAGES_DIR" -name "*.tar.gz" 2>/dev/null || echo "")
if [ -z "$TAR_FILES" ]; then
    print_error "在目录 $IMAGES_DIR 中未找到 .tar.gz 文件"
    exit 1
fi

print_info "📁 镜像目录: $IMAGES_DIR"
print_info "📋 找到的镜像文件:"
for file in $TAR_FILES; do
    echo "   - $(basename "$file")"
done

# 加载镜像
print_info "📥 开始加载镜像..."
for file in $TAR_FILES; do
    print_info "加载: $(basename "$file")"
    docker load < "$file"
    print_success "✅ $(basename "$file") 加载完成"
done

# 如果有标签后缀，重新标记镜像
if [ -n "$TAG_SUFFIX" ]; then
    print_info "🏷️  重新标记镜像 (添加后缀: $TAG_SUFFIX)..."
    
    # 获取当前镜像列表
    IMAGES=$(docker images --format "{{.Repository}}:{{.Tag}}" | grep -v "<none>")
    
    for image in $IMAGES; do
        if [[ "$image" != *"$TAG_SUFFIX" ]]; then
            new_tag="${image}:${TAG_SUFFIX}"
            docker tag "$image" "$new_tag"
            print_success "✅ 标记: $image -> $new_tag"
        fi
    done
fi

print_success "🎉 所有镜像加载完成！"
echo ""
print_info "📋 当前Docker镜像:"
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}" | head -10
