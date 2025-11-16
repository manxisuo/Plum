#!/bin/bash
# 导入 Docker 镜像从 tar.gz 文件
# 使用方法: ./import-docker-images.sh [镜像目录]
# 默认目录: ./docker-images-export

set -u  # 只检查未定义变量

# 镜像目录
IMAGES_DIR="${1:-./docker-images-export}"

# 检查目录是否存在
if [ ! -d "$IMAGES_DIR" ]; then
    echo "❌ 错误: 目录不存在: $IMAGES_DIR"
    echo ""
    echo "使用方法: $0 [镜像目录]"
    echo "示例: $0 ~/plum_files/images/"
    exit 1
fi

echo "📦 开始导入 Docker 镜像..."
echo "镜像目录: $IMAGES_DIR"
echo ""

# 查找所有 .tar.gz 文件
mapfile -t IMAGE_FILES < <(find "$IMAGES_DIR" -maxdepth 1 -type f -name "*.tar.gz" | sort)

# 检查是否有文件
if [ ${#IMAGE_FILES[@]} -eq 0 ]; then
    echo "⚠️  未找到任何 .tar.gz 文件"
    exit 1
fi

TOTAL=${#IMAGE_FILES[@]}
SUCCESS=0
FAILED=0
SKIPPED=0

echo "找到 $TOTAL 个镜像文件"
echo ""

# 导入每个镜像
for IMAGE_FILE in "${IMAGE_FILES[@]}"; do
    FILENAME=$(basename "$IMAGE_FILE")
    
    echo "[$((SUCCESS + FAILED + SKIPPED + 1))/$TOTAL] 导入: $FILENAME"
    
    # 检查文件是否存在且可读
    if [ ! -f "$IMAGE_FILE" ]; then
        echo "  ⚠️  文件不存在，跳过: $FILENAME"
        SKIPPED=$((SKIPPED + 1))
        echo ""
        continue
    fi
    
    if [ ! -r "$IMAGE_FILE" ]; then
        echo "  ⚠️  文件不可读，跳过: $FILENAME"
        SKIPPED=$((SKIPPED + 1))
        echo ""
        continue
    fi
    
    # 获取文件大小
    FILE_SIZE=$(du -h "$IMAGE_FILE" 2>/dev/null | cut -f1 || echo "未知")
    echo "  📁 文件大小: $FILE_SIZE"
    
    # 导入镜像
    echo "  ⬆️  正在导入..."
    if docker load -i "$IMAGE_FILE" 2>&1; then
        echo "  ✅ 导入成功: $FILENAME"
        SUCCESS=$((SUCCESS + 1))
    else
        echo "  ❌ 导入失败: $FILENAME"
        FAILED=$((FAILED + 1))
    fi
    echo ""
done

# 输出统计信息
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 导入完成统计:"
echo "  总计: $TOTAL"
echo "  成功: $SUCCESS"
echo "  失败: $FAILED"
echo "  跳过: $SKIPPED"
echo ""

# 列出导入的镜像
if [ $SUCCESS -gt 0 ]; then
    echo "📦 已导入的镜像列表:"
    docker images --format "  {{.Repository}}:{{.Tag}} ({{.Size}})" | head -20
    echo ""
    
    # 计算总镜像数
    TOTAL_IMAGES=$(docker images -q | wc -l)
    echo "💾 当前系统中的镜像总数: $TOTAL_IMAGES"
fi

if [ $FAILED -gt 0 ]; then
    echo "⚠️  有 $FAILED 个镜像导入失败，请检查错误信息"
    exit 1
fi

if [ $SUCCESS -eq 0 ]; then
    echo "⚠️  没有成功导入任何镜像"
    exit 1
fi

echo "✅ 所有镜像导入完成！"

