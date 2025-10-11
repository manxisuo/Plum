#!/bin/bash
# Worker Demo构建和打包脚本

set -e

echo "Building Plum Worker Demo..."

# 检查proto文件是否存在
PROTO_DIR="../../sdk/cpp/grpc/proto"
if [ ! -f "$PROTO_DIR/task_service.pb.cc" ]; then
    echo "❌ Proto文件不存在，请先运行: make proto"
    exit 1
fi

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
cp build/worker-demo package/
cp start.sh package/
cp meta.ini package/
chmod +x package/start.sh
chmod +x package/worker-demo

# 创建ZIP包
cd package
zip -r ../worker-demo.zip .
cd ..

echo "✅ Package created: worker-demo.zip"
ls -lh worker-demo.zip

echo ""
echo "部署到Plum："
echo "  1. 上传 worker-demo.zip"
echo "  2. 创建部署"
echo "  3. 启动后，Worker会自动注册到Controller"
echo "  4. 在'任务定义'中可以创建executor=embedded的任务"
echo "  5. targetKind=app, targetRef=worker-demo"

