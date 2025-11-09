#!/bin/bash
# 修复离线环境下的dpkg配置问题

set -e

echo "🔧 修复离线环境下的dpkg配置问题..."

# 检查是否在正确的目录
if [ ! -d "sdk/cpp" ] || [ ! -f "Makefile" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

echo "📁 当前目录: $(pwd)"

# 1. 检查dpkg状态
echo ""
echo "🔍 检查dpkg状态..."
if dpkg --audit 2>/dev/null | grep -q "broken"; then
    echo "❌ 发现损坏的包"
    dpkg --audit
else
    echo "✅ 没有发现损坏的包"
fi

# 2. 检查未配置的包
echo ""
echo "🔍 检查未配置的包..."
if dpkg -l | grep -q "^iU"; then
    echo "❌ 发现未配置的包"
    dpkg -l | grep "^iU"
else
    echo "✅ 没有发现未配置的包"
fi

# 3. 尝试修复dpkg配置
echo ""
echo "🔧 尝试修复dpkg配置..."
echo "📋 运行: sudo dpkg --configure -a"
sudo dpkg --configure -a || {
    echo "⚠️  dpkg配置失败，可能需要手动处理"
}

# 4. 检查gRPC依赖包
echo ""
echo "🔍 检查gRPC依赖包..."
GRPC_DEPS_DIR="../tools/grpc-deps"
if [ -d "$GRPC_DEPS_DIR" ] && ls "$GRPC_DEPS_DIR"/*.deb 1> /dev/null 2>&1; then
    echo "📦 发现gRPC依赖包，尝试重新安装..."
    cd "$GRPC_DEPS_DIR"
    
    # 按依赖顺序安装
    echo "🔄 按依赖顺序重新安装包..."
    
    # 先安装基础库
    for pkg in libc-ares2 libprotobuf17 libgrpc6 libgrpc++1; do
        if ls ${pkg}_*.deb 1> /dev/null 2>&1; then
            echo "📥 安装 $pkg..."
            sudo dpkg -i ${pkg}_*.deb 2>/dev/null || {
                echo "⚠️  $pkg 安装失败，继续安装其他包..."
            }
        fi
    done
    
    # 再安装开发包
    for pkg in libprotobuf-dev protobuf-compiler libgrpc-dev libgrpc++-dev; do
        if ls ${pkg}_*.deb 1> /dev/null 2>&1; then
            echo "📥 安装 $pkg..."
            sudo dpkg -i ${pkg}_*.deb 2>/dev/null || {
                echo "⚠️  $pkg 安装失败，继续安装其他包..."
            }
        fi
    done
    
    cd - > /dev/null
else
    echo "📋 未找到gRPC依赖包"
fi

# 5. 验证修复结果
echo ""
echo "🔍 验证修复结果..."

# 检查protoc
if command -v protoc &> /dev/null; then
    echo "✅ protoc可用: $(protoc --version)"
else
    echo "❌ protoc不可用"
fi

# 检查pkg-config
if pkg-config --exists protobuf; then
    echo "✅ protobuf pkg-config可用"
    echo "   版本: $(pkg-config --modversion protobuf)"
else
    echo "❌ protobuf pkg-config不可用"
fi

if pkg-config --exists grpc++; then
    echo "✅ gRPC pkg-config可用"
    echo "   版本: $(pkg-config --modversion grpc++)"
else
    echo "❌ gRPC pkg-config不可用"
fi

# 6. 最终检查
echo ""
echo "🔍 最终检查..."
if dpkg --audit 2>/dev/null | grep -q "broken"; then
    echo "❌ 仍有损坏的包，需要手动处理"
    dpkg --audit
else
    echo "✅ 所有包状态正常"
fi

echo ""
echo "🎯 修复完成！现在可以尝试："
echo "   make sdk_cpp_offline"
