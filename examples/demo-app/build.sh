#!/bin/bash
# Demo App构建和打包脚本

set -e

echo "Building Plum Demo App..."

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
cp build/demo-app package/
cp start.sh package/
cp meta.ini package/
chmod +x package/start.sh
chmod +x package/demo-app

# 创建ZIP包
cd package
zip -r ../demo-app.zip .
cd ..

echo "✅ Package created: demo-app.zip"
ls -lh demo-app.zip

echo ""
echo "上传到Plum："
echo "  1. 在Plum UI中进入'应用'页面"
echo "  2. 点击'上传应用包'"
echo "  3. 选择 demo-app.zip"
echo "  4. 创建部署，选择此应用"
echo "  5. 点击'启动'按钮"

