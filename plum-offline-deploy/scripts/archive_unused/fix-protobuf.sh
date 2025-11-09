#!/bin/bash
# 修复protobuf开发包问题

set -e

echo "🔧 修复protobuf开发包问题..."

# 检查是否在正确的目录
if [ ! -d "sdk/cpp" ] || [ ! -f "Makefile" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

echo "📁 当前目录: $(pwd)"

# 1. 检查protobuf状态
echo ""
echo "🔍 检查当前protobuf状态..."
bash plum-offline-deploy/scripts/check-protobuf.sh

# 2. 尝试修复protobuf问题
echo ""
echo "🔧 尝试修复protobuf问题..."

# 检查是否有gRPC依赖包
GRPC_DEPS_DIR="../tools/grpc-deps"
if [ -d "$GRPC_DEPS_DIR" ] && ls "$GRPC_DEPS_DIR"/*.deb 1> /dev/null 2>&1; then
    echo "📦 发现gRPC依赖包，重新安装..."
    cd "$GRPC_DEPS_DIR"
    
    # 重新安装所有包
    echo "🔄 重新安装gRPC和protobuf包..."
    sudo dpkg -i *.deb 2>/dev/null || {
        echo "⚠️  部分包安装失败，尝试修复依赖..."
        sudo apt-get install -f -y 2>/dev/null || {
            echo "⚠️  无法自动修复依赖，请手动处理"
        }
    }
    
    cd - > /dev/null
else
    echo "📋 未找到gRPC依赖包，尝试系统安装..."
    echo "💡 建议手动安装以下包："
    echo "   sudo apt-get install libprotobuf-dev protobuf-compiler libgrpc++-dev libgrpc-dev"
fi

# 3. 验证修复结果
echo ""
echo "🔍 验证修复结果..."
if pkg-config --exists protobuf; then
    echo "✅ protobuf pkg-config可用"
    echo "   版本: $(pkg-config --modversion protobuf)"
else
    echo "❌ protobuf pkg-config仍然不可用"
fi

# 检查关键头文件
if [ -f "/usr/include/google/protobuf/port_def.inc" ]; then
    echo "✅ 找到关键头文件: /usr/include/google/protobuf/port_def.inc"
elif [ -f "/usr/local/include/google/protobuf/port_def.inc" ]; then
    echo "✅ 找到关键头文件: /usr/local/include/google/protobuf/port_def.inc"
else
    echo "❌ 仍然缺少关键头文件: google/protobuf/port_def.inc"
    echo "💡 可能需要手动安装完整的protobuf开发包"
fi

# 4. 测试构建
echo ""
echo "🧪 测试protobuf构建..."
if [ -f "sdk/cpp/grpc/proto/task_service.pb.h" ]; then
    echo "✅ 找到生成的protobuf头文件"
    
    # 检查头文件内容
    if grep -q "google/protobuf/port_def.inc" sdk/cpp/grpc/proto/task_service.pb.h; then
        echo "✅ 头文件包含正确的protobuf引用"
    else
        echo "⚠️  头文件可能不完整"
    fi
else
    echo "❌ 未找到生成的protobuf头文件"
    echo "💡 可能需要重新生成protobuf文件"
fi

echo ""
echo "🎯 修复完成！现在可以尝试："
echo "   make sdk_cpp_offline"
