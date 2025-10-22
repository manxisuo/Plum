#!/bin/bash
# C++ SDK和Plum Client库构建脚本
# 用于银河麒麟V10 ARM64环境

set -e

echo "🚀 开始构建C++ SDK和Plum Client库..."

# 检查CMake是否可用
if ! command -v cmake &> /dev/null; then
    echo "❌ CMake未安装，请先安装CMake:"
    echo "   sudo apt-get update"
    echo "   sudo apt-get install cmake"
    exit 1
fi

echo "✅ CMake已安装: $(cmake --version | head -n1)"

# 检查C++依赖
echo "🔧 检查C++依赖..."

# 检查httplib (plumclient现在使用httplib，不再需要libcurl)
if [ -f "/usr/include/httplib.h" ] || [ -f "/usr/local/include/httplib.h" ]; then
    echo "✅ httplib头文件已找到"
else
    echo "ℹ️  httplib头文件未在系统路径找到，将使用项目内置版本"
fi

# 检查pthread
if ! pkg-config --exists pthread; then
    echo "❌ pthread未找到，请安装:"
    echo "   sudo apt-get install libpthread-stubs0-dev"
    exit 1
else
    echo "✅ pthread已安装"
fi

# 检查g++
if ! command -v g++ &> /dev/null; then
    echo "❌ g++未找到，请安装:"
    echo "   sudo apt-get install g++"
    exit 1
else
    echo "✅ g++已安装: $(g++ --version | head -n1)"
fi

# 进入项目目录
cd ../source/Plum

# 1. 构建C++ SDK（离线模式）
echo "📦 构建C++ SDK（离线模式）..."
if make sdk_cpp_offline; then
    echo "✅ C++ SDK构建完成"
else
    echo "❌ C++ SDK构建失败"
    exit 1
fi

# 2. 构建Plum Client库
echo "📦 构建Plum Client库..."
if make plumclient; then
    echo "✅ Plum Client库构建完成"
else
    echo "❌ Plum Client库构建失败"
    exit 1
fi

# 3. 构建Service Client示例
echo "📦 构建Service Client示例..."
if make service_client_example; then
    echo "✅ Service Client示例构建完成"
else
    echo "⚠️  Service Client示例构建失败，但库构建成功"
fi

# 验证构建结果
echo "🔍 验证构建结果..."

if [ -f "sdk/cpp/build/plumclient/libplumclient.so" ]; then
    echo "✅ Plum Client库: sdk/cpp/build/plumclient/libplumclient.so"
    echo "  库大小: $(du -h sdk/cpp/build/plumclient/libplumclient.so | cut -f1)"
    echo "  架构: $(file sdk/cpp/build/plumclient/libplumclient.so | grep -o 'ARM64\|aarch64\|arm64' || echo '未知')"
else
    echo "❌ Plum Client库未找到"
fi

if [ -f "sdk/cpp/build/examples/service_client_example/service_client_example" ]; then
    echo "✅ Service Client示例: sdk/cpp/build/examples/service_client_example/service_client_example"
    echo "  示例大小: $(du -h sdk/cpp/build/examples/service_client_example/service_client_example | cut -f1)"
    echo "  架构: $(file sdk/cpp/build/examples/service_client_example/service_client_example | grep -o 'ARM64\|aarch64\|arm64' || echo '未知')"
else
    echo "❌ Service Client示例未找到"
fi

# 检查其他C++示例
echo "🔍 检查其他C++示例..."

if [ -f "sdk/cpp/build/examples/echo_worker/echo_worker" ]; then
    echo "✅ Echo Worker示例: sdk/cpp/build/examples/echo_worker/echo_worker"
    echo "  大小: $(du -h sdk/cpp/build/examples/echo_worker/echo_worker | cut -f1)"
fi

if [ -f "sdk/cpp/build/examples/radar_sensor/radar_sensor" ]; then
    echo "✅ Radar Sensor示例: sdk/cpp/build/examples/radar_sensor/radar_sensor"
    echo "  大小: $(du -h sdk/cpp/build/examples/radar_sensor/radar_sensor | cut -f1)"
fi

if [ -f "sdk/cpp/build/examples/grpc_echo_worker/grpc_echo_worker" ]; then
    echo "✅ gRPC Echo Worker示例: sdk/cpp/build/examples/grpc_echo_worker/grpc_echo_worker"
    echo "  大小: $(du -h sdk/cpp/build/examples/grpc_echo_worker/grpc_echo_worker | cut -f1)"
fi

echo ""
echo "🎉 C++ SDK构建完成！"
echo ""
echo "构建结果:"
echo "- Plum Client库: sdk/cpp/build/plumclient/libplumclient.so"
echo "- Service Client示例: sdk/cpp/build/examples/service_client_example/service_client_example"
echo "- 其他C++示例: sdk/cpp/build/examples/*/"
echo ""
echo "下一步: 可以运行示例程序测试功能"
