#!/bin/bash
# 查找grpc_cpp_plugin位置

echo "🔍 查找grpc_cpp_plugin位置..."

# 检查PATH中是否有grpc_cpp_plugin
echo "📋 检查PATH中的grpc_cpp_plugin："
if command -v grpc_cpp_plugin &> /dev/null; then
    echo "✅ 找到: $(which grpc_cpp_plugin)"
    grpc_cpp_plugin --version 2>/dev/null || echo "⚠️  无法获取版本信息"
else
    echo "❌ PATH中未找到grpc_cpp_plugin"
fi

echo ""
echo "📋 查找系统中的所有grpc_cpp_plugin："
find /usr -name "grpc_cpp_plugin" -type f 2>/dev/null

echo ""
echo "📋 查找所有grpc相关可执行文件："
find /usr -name "*grpc*" -type f -executable 2>/dev/null | grep -v ".so" | head -10

echo ""
echo "📋 检查gRPC开发包安装："
if command -v dpkg &> /dev/null; then
    echo "已安装的gRPC包："
    dpkg -l | grep -i grpc
else
    echo "dpkg不可用"
fi

echo ""
echo "📋 检查pkg-config gRPC信息："
if pkg-config --exists grpc++; then
    echo "gRPC版本: $(pkg-config --modversion grpc++)"
    echo "gRPC包含目录: $(pkg-config --cflags grpc++)"
    echo "gRPC链接库: $(pkg-config --libs grpc++)"
else
    echo "❌ pkg-config grpc++不可用"
fi
