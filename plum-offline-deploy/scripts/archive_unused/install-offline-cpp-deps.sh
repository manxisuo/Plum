#!/bin/bash
# 离线安装C++ SDK依赖脚本
# 用于银河麒麟V10 ARM64环境（无网络连接）

set -e

echo "🔧 离线安装C++ SDK依赖..."

# 检查是否有离线包
OFFLINE_DEPS_DIR="../tools/cpp-deps"
if [ -d "$OFFLINE_DEPS_DIR" ] && ls "$OFFLINE_DEPS_DIR"/*.deb 1> /dev/null 2>&1; then
    echo "📦 发现离线C++依赖包，开始安装..."
    cd "$OFFLINE_DEPS_DIR"
    
    # 安装所有.deb包
    echo "📥 安装离线依赖包..."
    sudo dpkg -i *.deb 2>/dev/null || {
        echo "⚠️  部分包安装失败，尝试修复依赖..."
        # 在离线环境下，我们无法使用apt-get install -f
        echo "   请检查包依赖关系"
    }
    
    cd - > /dev/null
    echo "✅ 离线依赖包安装完成"
else
    echo "❌ 未找到离线C++依赖包"
    echo "   需要准备以下ARM64 .deb包："
    echo "   - libpthread-stubs0-dev_*_arm64.deb"
    echo "   - build-essential_*_arm64.deb"
    echo "   注意: plumclient现在使用httplib，不再需要libcurl"
    echo "   - libc6-dev_*_arm64.deb"
    echo ""
    echo "   请将这些包放在 plum-offline-deploy/tools/cpp-deps/ 目录下"
    exit 1
fi

echo "🔍 验证安装结果..."

# 检查pthread (plumclient现在使用httplib，不再需要libcurl)
if pkg-config --exists libpthread-stubs; then
    echo "✅ pthread开发包已安装: $(pkg-config --modversion libpthread-stubs)"
else
    echo "❌ pthread开发包安装失败"
fi

# 检查pthread
if pkg-config --exists pthread; then
    echo "✅ pthread开发包已安装"
else
    echo "❌ pthread开发包安装失败"
fi

echo ""
echo "🎉 离线C++依赖安装完成！"
echo ""
echo "现在可以尝试构建C++ SDK："
echo "1. cd ../source/Plum"
echo "2. make sdk_cpp"
echo "3. make plumclient"
