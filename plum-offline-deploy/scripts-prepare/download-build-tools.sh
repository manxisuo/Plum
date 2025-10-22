#!/bin/bash
# 下载 build-essential 相关包的脚本 - 适用于飞腾ARM64
# 注意：如果目标机器已手动安装 build-essential，此脚本可能不再需要

set -e

echo "⚠️  注意：如果目标机器已手动安装 build-essential，此脚本可能不再需要"
echo "🚀 下载飞腾ARM64平台的 build-essential 相关包..."

# 确保在正确的目录运行
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOLS_DIR="$(dirname "$SCRIPT_DIR")/tools"

# 创建下载目录
BUILD_TOOLS_DIR="$TOOLS_DIR/build-tools-deps"
mkdir -p "$BUILD_TOOLS_DIR"
cd "$BUILD_TOOLS_DIR"

echo "📁 下载目录: $(pwd)"

echo "📋 目标平台: 飞腾 ARM64 (aarch64)"
echo "📋 注意: 银河麒麟官方源缺少标准ARM64 build-essential包"

# Ubuntu 20.04 基础URL (适用于大多数ARM64系统)
BASE_URL="http://archive.ubuntu.com/ubuntu/pool/main"
PORTS_URL="http://ports.ubuntu.com/pool/main"

echo "📦 下载基础包..."

# build-essential 主包 - 尝试多个源
echo "下载 build-essential for ARM64..."
DOWNLOAD_SUCCESS=false

# 尝试多个版本和源
for url_base in "$BASE_URL" "$PORTS_URL"; do
    for version in "12.4ubuntu1" "12.9ubuntu3" "12.6"; do
        echo "尝试: $url_base/e/eglibc/build-essential_${version}_arm64.deb"
        if wget -c "$url_base/e/eglibc/build-essential_${version}_arm64.deb" 2>/dev/null; then
            echo "✅ build-essential 下载成功: build-essential_${version}_arm64.deb"
            DOWNLOAD_SUCCESS=true
            break
        fi
    done
    if [ "$DOWNLOAD_SUCCESS" = true ]; then
        break
    fi
done

# 方案1: 优先下载银河麒麟的 crossbuild-essential-arm64 包进行测试
echo "📦 方案1: 下载银河麒麟 crossbuild-essential-arm64 包..."
if wget -c "https://archive.kylinos.cn/kylin/KYLIN-ALL/pool/build-essential/crossbuild-essential-arm64_12.6_all.deb"; then
    echo "✅ 银河麒麟 ARM64 交叉编译包下载成功"
    echo "⚠️  注意: 这是交叉编译包，将尝试在飞腾ARM64上使用"
else
    echo "❌ 银河麒麟包下载失败，尝试备用源..."
    if [ "$DOWNLOAD_SUCCESS" = false ]; then
        echo "⚠️  标准 build-essential 包也下载失败"
        exit 1
    fi
fi

# gcc 相关包
echo "下载 gcc 相关包..."
wget -c "$BASE_URL/g/gcc-defaults/gcc_9.4.0-1ubuntu1~20.04.1_arm64.deb" || echo "⚠️  gcc 包下载失败"
wget -c "$BASE_URL/g/gcc-defaults/g++_9.4.0-1ubuntu1~20.04.1_arm64.deb" || echo "⚠️  g++ 包下载失败"
wget -c "$BASE_URL/g/gcc-9/gcc-9_9.4.0-1ubuntu1~20.04.1_arm64.deb" || echo "⚠️  gcc-9 包下载失败"
wget -c "$BASE_URL/g/gcc-9/g++-9_9.4.0-1ubuntu1~20.04.1_arm64.deb" || echo "⚠️  g++-9 包下载失败"

# make
echo "下载 make..."
wget -c "$BASE_URL/m/make-dfsg/make_4.2.1-1.2_arm64.deb" || echo "⚠️  make 包下载失败"

# libc6-dev
echo "下载 libc6-dev..."
wget -c "http://archive.ubuntu.com/ubuntu/pool/main/g/glibc/libc6-dev_2.31-0ubuntu9.9_arm64.deb" || echo "⚠️  libc6-dev 包下载失败"

# dpkg-dev
echo "下载 dpkg-dev..."
wget -c "http://archive.ubuntu.com/ubuntu/pool/main/d/dpkg/dpkg-dev_1.19.7ubuntu3.2_all.deb" || echo "⚠️  dpkg-dev 包下载失败"

echo "✅ 下载完成！"
echo "下载的文件："
ls -la *.deb

echo ""
echo "在目标机器上安装："
echo "sudo dpkg -i *.deb"
echo "# 如果有依赖问题，运行："
echo "sudo apt-get install -f"
