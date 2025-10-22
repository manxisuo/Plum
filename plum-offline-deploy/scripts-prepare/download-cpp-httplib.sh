#!/bin/bash
# 下载 cpp-httplib 离线版本脚本

set -e

echo "🚀 下载 cpp-httplib 离线版本..."

# 确保在项目根目录下运行
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

# 目标目录应该是部署包的源目录
DEPLOY_DIR="plum-offline-deploy"
if [ -d "$DEPLOY_DIR/source/Plum" ]; then
    cd "$DEPLOY_DIR/source/Plum"
    echo "📁 切换到部署包源目录: $(pwd)"
fi

# 保存当前工作目录（部署包源目录）
TARGET_DIR="$(pwd)"
echo "📁 目标项目目录: $TARGET_DIR"

# 创建第三方依赖目录
THIRD_PARTY_DIR="$TARGET_DIR/sdk/cpp/third_party"
mkdir -p "$THIRD_PARTY_DIR"

echo "📁 目标目录: $THIRD_PARTY_DIR"

# GitHub仓库信息
REPO_URL="https://github.com/yhirose/cpp-httplib"
VERSION="v0.15.3"
TEMP_DIR="/tmp/cpp-httplib-download"

echo "📦 下载版本: $VERSION"

# 清理临时目录
rm -rf "$TEMP_DIR"
mkdir -p "$TEMP_DIR"

# 下载仓库（只下载指定版本）
echo "⬇️  下载仓库..."
cd "$TEMP_DIR"

if command -v git &> /dev/null; then
    echo "使用git下载..."
    git clone --depth 1 --branch "$VERSION" "$REPO_URL.git" cpp-httplib
else
    echo "未找到git，尝试使用wget下载zip包..."
    ZIP_URL="$REPO_URL/archive/refs/tags/$VERSION.zip"
    ZIP_FILE="cpp-httplib-$VERSION.zip"
    wget -O "$ZIP_FILE" "$ZIP_URL" || {
        echo "❌ 下载zip包失败"
        exit 1
    }
    unzip -q "$ZIP_FILE"
    
    # 重命名目录
    mv "cpp-httplib-${VERSION#v}" cpp-httplib
fi

echo "📋 下载内容："
ls -la cpp-httplib/httplib.h 2>/dev/null || echo "⚠️  未找到 httplib.h"

# 复制到项目目录
echo "📁 复制到项目目录..."
if [ -f "cpp-httplib/httplib.h" ]; then
    mkdir -p "$THIRD_PARTY_DIR/cpp-httplib"
    cp cpp-httplib/httplib.h "$THIRD_PARTY_DIR/cpp-httplib/"
    echo "✅ cpp-httplib 离线版本已准备完成"
    echo "📂 位置: $THIRD_PARTY_DIR/cpp-httplib/"
    ls -la "$THIRD_PARTY_DIR/cpp-httplib/"
else
    echo "❌ 下载的文件结构不正确"
    exit 1
fi

# 清理临时目录
rm -rf "$TEMP_DIR"

echo ""
echo "🎉 cpp-httplib 离线版本准备完成！"
echo "现在可以在离线环境中使用: make sdk_cpp_offline"
