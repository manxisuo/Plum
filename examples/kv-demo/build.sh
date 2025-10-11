#!/bin/bash
# KV Demo构建和打包脚本

set -e

echo "Building Plum KV Demo..."

# 创建构建目录
mkdir -p build
cd build

# CMake配置和编译
cmake ..
make

# 返回主目录
cd ..

# 打包
echo "Creating package..."
mkdir -p package
cp build/kv-demo package/
cp start.sh package/
cp meta.ini package/
chmod +x package/start.sh
chmod +x package/kv-demo

# 创建ZIP包
cd package
zip -r ../kv-demo.zip .
cd ..

echo "✅ Package created: kv-demo.zip"
ls -lh kv-demo.zip

echo ""
echo "部署到Plum："
echo "  1. 上传 kv-demo.zip"
echo "  2. 创建部署并启动"
echo "  3. 观察应用正常运行"
echo "  4. kill -9 杀掉进程（模拟崩溃）"
echo "  5. Agent自动重启应用"
echo "  6. 应用从崩溃点恢复继续执行！"

