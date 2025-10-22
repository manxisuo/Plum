#!/bin/bash
# 手动下载 rollup ARM64 二进制文件的脚本

set -e

echo "🚀 手动下载 rollup ARM64 二进制文件..."

# 检查当前目录
if [ ! -d "ui" ] || [ ! -f "ui/package.json" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

# 获取版本信息
ROLLUP_VERSION=$(grep -o '"@rollup/rollup-linux-arm64-gnu": "[^"]*"' ui/package.json | cut -d'"' -f4)
echo "📋 目标版本: $ROLLUP_VERSION"

# npm registry 信息
NPM_REGISTRY="https://registry.npmjs.org"
PACKAGE_NAME="@rollup/rollup-linux-arm64-gnu"

echo "🔍 获取包信息..."

# 创建临时目录
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

echo "📦 下载包信息..."
curl -s "$NPM_REGISTRY/$PACKAGE_NAME" > package-info.json

# 获取最新版本的真实版本号
if [ "$ROLLUP_VERSION" = "^4.52.5" ]; then
    # 对于 ^4.52.5，我们需要找到匹配的版本
    LATEST_VERSION=$(curl -s "$NPM_REGISTRY/$PACKAGE_NAME" | grep -o '"4\.52\.[0-9]*"' | head -1 | tr -d '"')
    if [ -z "$LATEST_VERSION" ]; then
        LATEST_VERSION="4.52.5"
    fi
else
    LATEST_VERSION="$ROLLUP_VERSION"
fi

echo "🎯 使用版本: $LATEST_VERSION"

# 下载 tarball
echo "📥 下载 tarball..."
TARBALL_URL="$NPM_REGISTRY/$PACKAGE_NAME/-/$PACKAGE_NAME-$LATEST_VERSION.tgz"
echo "URL: $TARBALL_URL"

# 尝试多种下载方法
echo "🔄 尝试下载..."
if wget --timeout=30 --tries=3 -O "$PACKAGE_NAME-$LATEST_VERSION.tgz" "$TARBALL_URL" 2>/dev/null; then
    echo "✅ wget 下载成功"
elif curl -L --connect-timeout 30 --max-time 300 -o "$PACKAGE_NAME-$LATEST_VERSION.tgz" "$TARBALL_URL" 2>/dev/null; then
    echo "✅ curl 下载成功"
else
    echo "❌ 下载失败，可能的原因："
    echo "   1. 网络连接问题"
    echo "   2. DNS 解析问题"
    echo "   3. npm registry 访问问题"
    echo ""
    echo "🔧 手动下载建议："
    echo "   wget '$TARBALL_URL'"
    echo "   或"
    echo "   curl -L -o rollup-linux-arm64-gnu-$LATEST_VERSION.tgz '$TARBALL_URL'"
    exit 1
fi

# 解包
echo "📁 解包..."
tar -tf "$PACKAGE_NAME-$LATEST_VERSION.tgz" > file-list.txt
tar -xzf "$PACKAGE_NAME-$LATEST_VERSION.tgz"

echo "📋 包内容:"
head -20 file-list.txt

# 查找二进制文件
BINARY_PATH=$(tar -tf "$PACKAGE_NAME-$LATEST_VERSION.tgz" | grep -E "(bin/|\.node$)" | head -5)
echo ""
echo "🔍 找到的文件:"
echo "$BINARY_PATH"

# 解压到正确位置
echo ""
echo "📍 解压到项目位置..."

# 回到项目目录
cd - > /dev/null

# 创建目标目录
mkdir -p "ui/node_modules/@rollup/rollup-linux-arm64-gnu"

# 解压到目标位置
cd "$TEMP_DIR"
tar -xzf "$PACKAGE_NAME-$LATEST_VERSION.tgz" --strip-components=1 -C "../../ui/node_modules/@rollup/rollup-linux-arm64-gnu/" 2>/dev/null || {
    echo "⚠️  直接解压失败，尝试手动复制文件..."
    
    # 手动提取文件
    tar -xzf "$PACKAGE_NAME-$LATEST_VERSION.tgz"
    
    # 复制 package 目录内容
    if [ -d "package" ]; then
        cp -r package/* "../../ui/node_modules/@rollup/rollup-linux-arm64-gnu/"
    fi
}

cd - > /dev/null

# 验证安装
echo ""
echo "🔍 验证安装..."
if [ -d "ui/node_modules/@rollup/rollup-linux-arm64-gnu" ]; then
    echo "✅ 目录已创建"
    ls -la "ui/node_modules/@rollup/rollup-linux-arm64-gnu/"
    
    # 查找二进制文件
    find "ui/node_modules/@rollup/rollup-linux-arm64-gnu/" -name "*.node" -o -name "rollup" | head -5
else
    echo "❌ 安装失败"
fi

# 清理临时文件
rm -rf "$TEMP_DIR"

echo ""
echo "🎉 下载完成！"
echo ""
echo "如果成功，你可以在目标机器上看到："
echo "ui/node_modules/@rollup/rollup-linux-arm64-gnu/"
