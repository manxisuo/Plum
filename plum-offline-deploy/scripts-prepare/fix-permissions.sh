#!/bin/bash
# 修复离线部署目录的权限问题

echo "🔧 修复离线部署目录权限..."

DEPLOY_DIR="plum-offline-deploy"

if [ ! -d "$DEPLOY_DIR" ]; then
    echo "❌ 未找到 $DEPLOY_DIR 目录"
    exit 1
fi

echo "修复 $DEPLOY_DIR 目录权限..."
if [ "$EUID" -eq 0 ]; then
    # 如果是root用户，修改为当前用户
    REAL_USER=$(who am i | awk '{print $1}')
    if [ -n "$REAL_USER" ]; then
        echo "修改目录所有者为: $REAL_USER"
        chown -R "$REAL_USER:$REAL_USER" "$DEPLOY_DIR"
    fi
else
    # 如果是普通用户，尝试修改权限
    echo "修改目录权限..."
    chmod -R u+w "$DEPLOY_DIR" 2>/dev/null || true
fi

# 特别处理可能有问题的构建文件
echo "清理构建文件..."
find "$DEPLOY_DIR" -type d -name "build" -exec chmod -R u+w {} \; 2>/dev/null || true
find "$DEPLOY_DIR" -type d -name "cmake-build-*" -exec chmod -R u+w {} \; 2>/dev/null || true

echo "✅ 权限修复完成"
echo ""
echo "现在可以重新运行: ./plum-offline-deploy/scripts-prepare/prepare-offline-deploy.sh"
