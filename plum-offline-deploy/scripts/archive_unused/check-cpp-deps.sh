#!/bin/bash
# C++ SDK依赖检查脚本
# 用于银河麒麟V10 ARM64环境

echo "🔍 检查C++ SDK依赖..."

# 检查CMake
echo "📦 检查CMake..."
if command -v cmake &> /dev/null; then
    echo "✅ CMake已安装: $(cmake --version | head -n1)"
else
    echo "❌ CMake未安装"
    echo "   安装命令: sudo apt-get install cmake"
    exit 1
fi

# 检查g++
echo "📦 检查g++..."
if command -v g++ &> /dev/null; then
    echo "✅ g++已安装: $(g++ --version | head -n1)"
else
    echo "❌ g++未安装"
    echo "   安装命令: sudo apt-get install g++"
    exit 1
fi

# 检查httplib (plumclient现在使用httplib，不再需要libcurl)
echo "📦 检查httplib..."
if [ -f "/usr/include/httplib.h" ] || [ -f "/usr/local/include/httplib.h" ]; then
    echo "✅ httplib头文件已找到"
else
    echo "ℹ️  httplib头文件未在系统路径找到，将使用项目内置版本"
fi

# 检查pthread
echo "📦 检查pthread..."
if pkg-config --exists pthread; then
    echo "✅ pthread已安装"
    echo "   链接库: $(pkg-config --libs pthread)"
else
    echo "❌ pthread未找到"
    echo "   安装命令: sudo apt-get install libpthread-stubs0-dev"
    exit 1
fi

# 检查make
echo "📦 检查make..."
if command -v make &> /dev/null; then
    echo "✅ make已安装: $(make --version | head -n1)"
else
    echo "❌ make未安装"
    echo "   安装命令: sudo apt-get install make"
    exit 1
fi

# 检查pkg-config
echo "📦 检查pkg-config..."
if command -v pkg-config &> /dev/null; then
    echo "✅ pkg-config已安装: $(pkg-config --version)"
else
    echo "❌ pkg-config未安装"
    echo "   安装命令: sudo apt-get install pkg-config"
    exit 1
fi

# 检查nlohmann/json（如果已安装）
echo "📦 检查nlohmann/json..."
if pkg-config --exists nlohmann_json; then
    echo "✅ nlohmann/json已安装: $(pkg-config --modversion nlohmann_json)"
    echo "   包含目录: $(pkg-config --cflags nlohmann_json)"
else
    echo "⚠️  nlohmann/json未通过pkg-config找到"
    echo "   将在构建时自动下载（离线模式）"
fi

# 检查C++标准库
echo "📦 检查C++标准库..."
if g++ -std=c++17 -x c++ -c /dev/null -o /dev/null 2>/dev/null; then
    echo "✅ C++17支持正常"
else
    echo "❌ C++17支持异常"
    echo "   请检查g++版本是否支持C++17"
    exit 1
fi

# 检查线程支持
echo "📦 检查线程支持..."
if g++ -pthread -x c++ -c /dev/null -o /dev/null 2>/dev/null; then
    echo "✅ 线程支持正常"
else
    echo "❌ 线程支持异常"
    echo "   请检查pthread库是否正确安装"
    exit 1
fi

# 检查curl支持
echo "📦 检查curl支持..."
if g++ -lcurl -x c++ -c /dev/null -o /dev/null 2>/dev/null; then
    echo "✅ curl支持正常"
else
    echo "❌ httplib支持异常"
    echo "   请检查httplib头文件是否正确安装"
    exit 1
fi

echo ""
echo "🎉 所有C++依赖检查通过！"
echo ""
echo "可以开始构建C++ SDK和Plum Client库"
echo "运行命令: ./build-cpp-sdk.sh"
