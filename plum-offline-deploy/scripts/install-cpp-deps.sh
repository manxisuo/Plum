#!/bin/bash
# C++ SDK依赖安装脚本
# 用于银河麒麟V10 ARM64环境

set -e

echo "🚀 开始安装C++ SDK依赖..."

# 检测系统
if [ "$(uname -m)" != "aarch64" ]; then
    echo "❌ 当前系统不是ARM64架构，请确认运行环境"
    exit 1
fi

# 检查是否有apt包管理器
if ! command -v apt &> /dev/null; then
    echo "❌ 未检测到apt包管理器，请手动安装依赖"
    echo "   需要安装的包："
    echo "   - libpthread-stubs0-dev"
    echo "   - build-essential"
    echo "   注意: plumclient现在使用httplib，不再需要libcurl"
    echo "   - cmake"
    echo "   - pkg-config"
    exit 1
fi

echo "📦 更新包列表..."
sudo apt-get update

echo "📦 安装C++ SDK核心依赖..."
sudo apt-get install -y \
    build-essential \
    cmake \
    pkg-config \
    libpthread-stubs0-dev

echo "📦 安装其他有用的开发工具..."
sudo apt-get install -y \
    git \
    curl \
    wget \
    unzip \
    tar

echo "🔍 验证安装结果..."

# 检查CMake
if command -v cmake &> /dev/null; then
    echo "✅ CMake: $(cmake --version | head -1)"
else
    echo "❌ CMake安装失败"
    exit 1
fi

# 检查g++
if command -v g++ &> /dev/null; then
    echo "✅ g++: $(g++ --version | head -1)"
else
    echo "❌ g++安装失败"
    exit 1
fi

# 检查httplib (plumclient现在使用httplib，不再需要libcurl)
if [ -f "/usr/include/httplib.h" ] || [ -f "/usr/local/include/httplib.h" ]; then
    echo "✅ httplib头文件已找到"
else
    echo "ℹ️  httplib头文件未在系统路径找到，将使用项目内置版本"
fi

# 检查pthread
if pkg-config --exists pthread; then
    echo "✅ pthread: 已安装"
    echo "   链接库: $(pkg-config --libs pthread)"
else
    echo "❌ pthread开发包安装失败"
    exit 1
fi

# 检查pkg-config
if command -v pkg-config &> /dev/null; then
    echo "✅ pkg-config: $(pkg-config --version)"
else
    echo "❌ pkg-config安装失败"
    exit 1
fi

# 测试C++17支持
echo "🔧 测试C++17支持..."
if g++ -std=c++17 -x c++ -c /dev/null -o /dev/null 2>/dev/null; then
    echo "✅ C++17支持正常"
else
    echo "❌ C++17支持异常"
    exit 1
fi

# 测试线程支持
echo "🔧 测试线程支持..."
if g++ -pthread -x c++ -c /dev/null -o /dev/null 2>/dev/null; then
    echo "✅ 线程支持正常"
else
    echo "❌ 线程支持异常"
    exit 1
fi

# 测试curl支持
echo "🔧 测试curl支持..."
if g++ -lcurl -x c++ -c /dev/null -o /dev/null 2>/dev/null; then
    echo "✅ curl支持正常"
else
    echo "❌ curl支持异常"
    exit 1
fi

echo ""
echo "🎉 C++ SDK依赖安装完成！"
echo ""
echo "已安装的依赖："
echo "- 编译工具: gcc, g++, make"
echo "- 构建工具: cmake, pkg-config"
echo "- 开发库: libpthread-stubs0-dev (httplib为header-only库)"
echo "- 其他工具: git, curl, wget"
echo ""
echo "现在可以构建C++ SDK："
echo "1. cd ../source/Plum"
echo "2. make sdk_cpp"
echo "3. make plumclient"
echo "4. make service_client_example"
