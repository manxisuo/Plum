#!/bin/bash
# 下载 nlohmann/json 离线版本

set -e

echo "🚀 下载 nlohmann/json 离线版本..."

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
REPO_URL="https://github.com/nlohmann/json"
VERSION="v3.11.3"
TEMP_DIR="/tmp/nlohmann-json-download"

echo "📦 下载版本: $VERSION"

# 清理临时目录
rm -rf "$TEMP_DIR"
mkdir -p "$TEMP_DIR"

# 下载仓库（只下载指定版本）
echo "⬇️  下载仓库..."
cd "$TEMP_DIR"

if command -v git &> /dev/null; then
    echo "使用git下载..."
    git clone --depth 1 --branch "$VERSION" "$REPO_URL.git" json
else
    echo "❌ git命令不可用，尝试直接下载zip文件..."
    
    # 尝试下载zip文件
    ZIP_URL="https://github.com/nlohmann/json/archive/refs/tags/${VERSION}.zip"
    ZIP_FILE="json-${VERSION}.zip"
    
    if command -v wget &> /dev/null; then
        wget -O "$ZIP_FILE" "$ZIP_URL"
    elif command -v curl &> /dev/null; then
        curl -L -o "$ZIP_FILE" "$ZIP_URL"
    else
        echo "❌ 既没有git也没有wget/curl，无法下载"
        exit 1
    fi
    
    echo "📦 解压zip文件..."
    unzip -q "$ZIP_FILE"
    
    # 重命名目录
    mv "json-${VERSION#v}" json
fi

echo "📋 下载内容："
ls -la json/include/

# 复制到项目目录
echo "📁 复制到项目目录..."
if [ -d "json/include/nlohmann" ]; then
    cp -r json/include/nlohmann "$THIRD_PARTY_DIR/"
    echo "✅ nlohmann/json 离线版本已准备完成"
    echo "📂 位置: $THIRD_PARTY_DIR/nlohmann/"
    ls -la "$THIRD_PARTY_DIR/nlohmann/" | head -5
else
    echo "❌ 下载的文件结构不正确"
    exit 1
fi

# 清理临时目录
rm -rf "$TEMP_DIR"

echo ""
echo "🎉 nlohmann/json 离线版本准备完成！"
echo "现在可以在离线环境中使用: make sdk_cpp_offline"
