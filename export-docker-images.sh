#!/bin/bash
# 导出 Docker 镜像为 tar.gz 文件
# 使用方法: ./export-docker-images.sh [输出目录]
# 默认输出目录: ./docker-images-export

# 不使用 set -e，手动处理错误
set -u  # 只检查未定义变量

# 输出目录
OUTPUT_DIR="${1:-./docker-images-export}"

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

echo "📦 开始导出 Docker 镜像..."
echo "输出目录: $OUTPUT_DIR"
echo ""

# 定义要导出的镜像列表
IMAGES=(
    "alpine:3.18"
    "fsl_destroy:1.0.0"
    "fsl_evaluate:1.0.0"
    "fsl_investigate:1.0.0"
    "fsl_maincontrol:1.0.0"
    "fsl_plan:1.0.0"
    "fsl_statistics:1.0.0"
    "fsl_sweep:1.0.0"
    "kylin/kylin:v10-release-020"
    "nginx:alpine"
    "python:3.11-slim"
    "sim_decision:1.0.0"
    "sim_navicontrol:1.0.0"
    "sim_routeplan:1.0.0"
    "sim_sonar:1.0.0"
    "sim_targethit:1.0.0"
    "sim_targetrecognize:1.0.0"
    "ubuntu:22.04"
    "ubuntu:24.04"
    "plum-agent:offline"
    "plum-controller:offline"
)

# 计数器
TOTAL=${#IMAGES[@]}
SUCCESS=0
FAILED=0
SKIPPED=0

# 导出每个镜像
for IMAGE in "${IMAGES[@]}"; do
    # 将冒号和斜杠替换为短横线，生成文件名
    # 例如: kylin/kylin:v10-release-020 -> kylin-kylin-v10-release-020.tar.gz
    FILENAME=$(echo "$IMAGE" | sed 's|/|-|g' | sed 's|:|-|g').tar.gz
    OUTPUT_FILE="$OUTPUT_DIR/$FILENAME"
    
    echo "[$((SUCCESS + FAILED + SKIPPED + 1))/$TOTAL] 导出: $IMAGE"
    
    # 先检查文件是否已存在（优先检查，避免不必要的镜像检查）
    if [ -f "$OUTPUT_FILE" ]; then
        FILE_SIZE=$(du -h "$OUTPUT_FILE" 2>/dev/null | cut -f1 || echo "未知")
        echo "  ℹ️  文件已存在，跳过: $FILENAME ($FILE_SIZE)"
        SKIPPED=$((SKIPPED + 1))
        echo ""
        continue
    fi
    
    # 检查镜像是否存在
    if ! docker image inspect "$IMAGE" > /dev/null 2>&1; then
        echo "  ⚠️  镜像不存在，跳过: $IMAGE"
        SKIPPED=$((SKIPPED + 1))
        echo ""
        continue
    fi
    
    # 导出镜像
    echo "  ⬇️  正在导出..."
    docker save "$IMAGE" 2>&1 | gzip > "$OUTPUT_FILE"
    SAVE_EXIT_CODE=${PIPESTATUS[0]}
    
    if [ $SAVE_EXIT_CODE -eq 0 ] && [ -f "$OUTPUT_FILE" ] && [ -s "$OUTPUT_FILE" ]; then
        # 获取文件大小
        FILE_SIZE=$(du -h "$OUTPUT_FILE" 2>/dev/null | cut -f1 || echo "未知")
        echo "  ✅ 导出成功: $FILENAME ($FILE_SIZE)"
        SUCCESS=$((SUCCESS + 1))
    else
        echo "  ❌ 导出失败: $IMAGE (退出码: $SAVE_EXIT_CODE)"
        # 删除可能创建的不完整文件
        rm -f "$OUTPUT_FILE"
        FAILED=$((FAILED + 1))
    fi
    echo ""
done

# 输出统计信息
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 导出完成统计:"
echo "  总计: $TOTAL"
echo "  成功: $SUCCESS"
echo "  失败: $FAILED"
echo "  跳过: $SKIPPED"
echo ""
echo "📁 输出目录: $OUTPUT_DIR"
echo ""

# 列出所有导出的文件
if [ $SUCCESS -gt 0 ]; then
    echo "📦 导出的文件列表:"
    ls -lh "$OUTPUT_DIR"/*.tar.gz 2>/dev/null | awk '{print "  " $9 " (" $5 ")"}'
    echo ""
    
    # 计算总大小
    TOTAL_SIZE=$(du -sh "$OUTPUT_DIR" 2>/dev/null | cut -f1)
    echo "💾 总大小: $TOTAL_SIZE"
fi

if [ $FAILED -gt 0 ]; then
    echo "⚠️  有 $FAILED 个镜像导出失败，请检查错误信息"
    exit 1
fi

echo "✅ 所有镜像导出完成！"

