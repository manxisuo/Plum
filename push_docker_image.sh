#!/bin/bash

# push_docker_image.sh - 将本地 Docker 镜像推送到远程机器
# 用法: ./push_docker_image.sh <镜像名称:标签> <远程用户@远程地址>
# 示例: ./push_docker_image.sh fsl_maincontrol:1.0.0 user@192.168.1.192

set -euo pipefail

# 检查参数
if [ $# -lt 2 ]; then
    echo "用法: $0 <镜像名称:标签> <远程用户@远程地址>"
    echo "示例: $0 fsl_maincontrol:1.0.0 user@192.168.1.192"
    exit 1
fi

IMAGE_NAME="$1"
REMOTE_HOST="$2"

# 从远程地址中提取用户名和主机
REMOTE_USER=$(echo "$REMOTE_HOST" | cut -d'@' -f1)
REMOTE_IP=$(echo "$REMOTE_HOST" | cut -d'@' -f2)

# 检查镜像是否存在
if ! docker image inspect "$IMAGE_NAME" >/dev/null 2>&1; then
    echo "错误: 镜像 $IMAGE_NAME 不存在"
    exit 1
fi

# 生成临时文件名（使用镜像名称和标签，替换特殊字符）
SAFE_IMAGE_NAME=$(echo "$IMAGE_NAME" | tr '/:' '_')
TEMP_TAR="/tmp/${SAFE_IMAGE_NAME}.tar"

echo "=========================================="
echo "推送 Docker 镜像到远程机器"
echo "=========================================="
echo "镜像名称: $IMAGE_NAME"
echo "远程地址: $REMOTE_HOST"
echo "临时文件: $TEMP_TAR"
echo ""

# 保存镜像为 tar 文件
echo "[1/4] 正在保存镜像为 tar 文件..."
if docker save "$IMAGE_NAME" -o "$TEMP_TAR"; then
    TAR_SIZE=$(du -h "$TEMP_TAR" | cut -f1)
    echo "✓ 镜像已保存: $TEMP_TAR ($TAR_SIZE)"
else
    echo "✗ 保存镜像失败"
    exit 1
fi

# 传输到远程机器
echo ""
echo "[2/4] 正在传输镜像到远程机器 $REMOTE_HOST..."
if scp "$TEMP_TAR" "$REMOTE_HOST:/tmp/"; then
    echo "✓ 镜像已传输到远程机器"
else
    echo "✗ 传输失败"
    rm -f "$TEMP_TAR"
    exit 1
fi

# 在远程机器上加载镜像
echo ""
echo "[3/4] 正在在远程机器上加载镜像..."
REMOTE_TAR="/tmp/$(basename "$TEMP_TAR")"
if ssh "$REMOTE_HOST" "docker load -i $REMOTE_TAR"; then
    echo "✓ 镜像已在远程机器上加载"
else
    echo "✗ 加载镜像失败"
    # 清理远程临时文件
    ssh "$REMOTE_HOST" "rm -f $REMOTE_TAR" 2>/dev/null || true
    rm -f "$TEMP_TAR"
    exit 1
fi

# 清理临时文件
echo ""
echo "[4/4] 正在清理临时文件..."
if ssh "$REMOTE_HOST" "rm -f $REMOTE_TAR"; then
    echo "✓ 远程临时文件已删除"
else
    echo "⚠ 警告: 无法删除远程临时文件 $REMOTE_TAR"
fi

if rm -f "$TEMP_TAR"; then
    echo "✓ 本地临时文件已删除"
else
    echo "⚠ 警告: 无法删除本地临时文件 $TEMP_TAR"
fi

echo ""
echo "=========================================="
echo "✓ 完成！镜像 $IMAGE_NAME 已成功推送到 $REMOTE_HOST"
echo "=========================================="

