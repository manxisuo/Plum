#!/bin/bash
# C++ SDK安装脚本
# 用于将Plum Client库安装到系统目录

set -e

echo "🚀 开始安装C++ SDK到系统..."

# 检查是否以root权限运行
if [ "$EUID" -ne 0 ]; then
    echo "❌ 请使用sudo运行此脚本"
    echo "   运行命令: sudo ./install-cpp-sdk.sh"
    exit 1
fi

# 安装目录配置
INSTALL_PREFIX="/usr/local"
LIB_DIR="$INSTALL_PREFIX/lib"
INCLUDE_DIR="$INSTALL_PREFIX/include/plumclient"
PKG_CONFIG_DIR="$INSTALL_PREFIX/lib/pkgconfig"

echo "📁 创建安装目录..."
mkdir -p $LIB_DIR
mkdir -p $INCLUDE_DIR
mkdir -p $PKG_CONFIG_DIR

# 进入项目目录
cd ../source/Plum

# 1. 安装Plum Client库
echo "📦 安装Plum Client库..."
if [ -f "sdk/cpp/build/plumclient/libplumclient.so" ]; then
    cp sdk/cpp/build/plumclient/libplumclient.so $LIB_DIR/
    chmod 755 $LIB_DIR/libplumclient.so
    echo "✅ Plum Client库已安装到 $LIB_DIR/libplumclient.so"
else
    echo "❌ Plum Client库未找到，请先构建"
    exit 1
fi

# 2. 安装头文件
echo "📦 安装头文件..."
if [ -d "sdk/cpp/plumclient/include" ]; then
    cp -r sdk/cpp/plumclient/include/* $INCLUDE_DIR/
    chmod -R 644 $INCLUDE_DIR/*
    echo "✅ 头文件已安装到 $INCLUDE_DIR/"
else
    echo "❌ 头文件未找到，请先构建"
    exit 1
fi

# 3. 创建pkg-config文件
echo "📦 创建pkg-config文件..."
cat > $PKG_CONFIG_DIR/plumclient.pc << EOF
prefix=$INSTALL_PREFIX
exec_prefix=\${prefix}
libdir=\${exec_prefix}/lib
includedir=\${prefix}/include

Name: plumclient
Description: Plum Client C++ SDK
Version: 1.0.0
Libs: -L\${libdir} -lplumclient -lcurl -lpthread
Cflags: -I\${includedir}
EOF

chmod 644 $PKG_CONFIG_DIR/plumclient.pc
echo "✅ pkg-config文件已创建"

# 4. 更新动态链接库缓存
echo "📦 更新动态链接库缓存..."
ldconfig
echo "✅ 动态链接库缓存已更新"

# 5. 验证安装
echo "🔍 验证安装..."

# 检查库文件
if [ -f "$LIB_DIR/libplumclient.so" ]; then
    echo "✅ 库文件: $LIB_DIR/libplumclient.so"
    echo "   大小: $(du -h $LIB_DIR/libplumclient.so | cut -f1)"
    echo "   架构: $(file $LIB_DIR/libplumclient.so | grep -o 'ARM64\|aarch64\|arm64' || echo '未知')"
else
    echo "❌ 库文件未找到"
fi

# 检查头文件
if [ -f "$INCLUDE_DIR/plum_client.hpp" ]; then
    echo "✅ 头文件: $INCLUDE_DIR/plum_client.hpp"
else
    echo "❌ 头文件未找到"
fi

# 检查pkg-config
if pkg-config --exists plumclient; then
    echo "✅ pkg-config配置正常"
    echo "   包含目录: $(pkg-config --cflags plumclient)"
    echo "   链接库: $(pkg-config --libs plumclient)"
else
    echo "❌ pkg-config配置异常"
fi

# 6. 创建使用示例
echo "📝 创建使用示例..."
cat > /tmp/plumclient_test.cpp << 'EOF'
#include <plum_client.hpp>
#include <iostream>

int main() {
    try {
        plumclient::PlumClient client("http://localhost:8080");
        std::cout << "Plum Client库测试成功！" << std::endl;
        return 0;
    } catch (const std::exception& e) {
        std::cerr << "错误: " << e.what() << std::endl;
        return 1;
    }
}
EOF

# 编译测试
echo "🔧 编译测试程序..."
if g++ -std=c++17 -o /tmp/plumclient_test /tmp/plumclient_test.cpp $(pkg-config --cflags --libs plumclient) 2>/dev/null; then
    echo "✅ 编译测试成功"
    rm -f /tmp/plumclient_test /tmp/plumclient_test.cpp
else
    echo "⚠️  编译测试失败，请检查依赖"
    rm -f /tmp/plumclient_test /tmp/plumclient_test.cpp
fi

echo ""
echo "🎉 C++ SDK安装完成！"
echo ""
echo "安装位置："
echo "- 库文件: $LIB_DIR/libplumclient.so"
echo "- 头文件: $INCLUDE_DIR/"
echo "- pkg-config: $PKG_CONFIG_DIR/plumclient.pc"
echo ""
echo "使用方法："
echo "- 编译时链接: $(pkg-config --cflags --libs plumclient)"
echo "- 或者手动指定: -I$INCLUDE_DIR -L$LIB_DIR -lplumclient -lcurl -lpthread"
echo ""
echo "示例编译："
echo "g++ -std=c++17 -o myapp myapp.cpp \$(pkg-config --cflags --libs plumclient)"
