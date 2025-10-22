#!/bin/bash
# 检查grpc依赖的脚本

set -e

echo "🔍 检查 gRPC 依赖..."

# 检查pkg-config是否能找到grpc++
echo "📋 检查 grpc++ 包："
if pkg-config --exists grpc++; then
    echo "✅ grpc++ 包已安装"
    pkg-config --modversion grpc++
    pkg-config --cflags grpc++
    pkg-config --libs grpc++
else
    echo "❌ grpc++ 包未找到"
    
    echo ""
    echo "🔍 检查相关包："
    for pkg in grpc protobuf; do
        if pkg-config --exists $pkg; then
            echo "✅ $pkg: $(pkg-config --modversion $pkg)"
        else
            echo "❌ $pkg: 未找到"
        fi
    done
    
    echo ""
    echo "💡 需要的包："
    echo "   sudo apt install libgrpc++-dev libgrpc-dev libprotobuf-dev protobuf-compiler"
    echo ""
    echo "🔌 离线环境解决方案："
    echo "   1. 在联网环境中下载ARM64版本的.deb包"
    echo "   2. 或跳过C++ SDK构建（如果不需要）"
fi

echo ""
echo "🔍 检查系统已安装的相关包："
dpkg -l | grep -E "(grpc|protobuf)" || echo "未找到相关包"
