#!/bin/bash

# C++弱网环境测试脚本

echo "=== Plum C++弱网环境测试 ==="
echo "测试目标: 验证C++ SDK在弱网环境下的服务发现能力"
echo "开始时间: $(date)"

# 检查Controller状态
echo "检查Controller状态..."
if ! curl -s http://localhost:8080/healthz > /dev/null; then
    echo "❌ Controller未运行"
    echo "请先启动Controller:"
    echo "运行: make controller-run"
    exit 1
fi
echo "✅ Controller运行正常"

# 检查依赖
echo "检查依赖..."
if ! pkg-config --exists libcurl; then
    echo "❌ libcurl未安装"
    echo "请安装: sudo apt-get install libcurl4-openssl-dev"
    exit 1
fi

if ! pkg-config --exists jsoncpp; then
    echo "❌ jsoncpp未安装"
    echo "请安装: sudo apt-get install libjsoncpp-dev"
    exit 1
fi
echo "✅ 依赖检查通过"

# 编译C++弱网环境测试工具
echo "编译C++弱网环境测试工具..."
cd tools

# 创建构建目录
mkdir -p build
cd build

# 配置CMake
cmake .. -DCMAKE_BUILD_TYPE=Release
if [ $? -ne 0 ]; then
    echo "❌ CMake配置失败"
    exit 1
fi

# 编译
make -j$(nproc)
if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi
echo "✅ 编译成功"

# 运行C++弱网环境测试
echo "运行C++弱网环境测试..."
./cpp_weak_network_test

echo "C++弱网环境测试完成"
