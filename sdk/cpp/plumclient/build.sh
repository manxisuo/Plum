#!/bin/bash

# 构建plumclient库
set -e

echo "=== 构建 Plum Client 库 ==="

# 创建构建目录
mkdir -p build
cd build

# 配置CMake
echo "配置CMake..."
cmake .. -DCMAKE_BUILD_TYPE=Release

# 编译
echo "编译库..."
make -j$(nproc)

# 安装
echo "安装库..."
sudo make install

echo "=== 构建完成 ==="
echo "库已安装到系统目录"
echo "可以使用以下命令运行示例程序:"
echo "  ./service_client_example"
