#!/bin/bash
# 修复银河麒麟系统上的protobuf问题

set -e

echo "🔧 修复银河麒麟系统上的protobuf问题..."

# 检查是否在正确的目录
if [ ! -d "sdk/cpp" ] || [ ! -f "Makefile" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

echo "📁 当前目录: $(pwd)"

# 1. 检查protobuf安装状态
echo ""
echo "🔍 检查protobuf安装状态..."

# 检查protoc
if command -v protoc &> /dev/null; then
    echo "✅ protoc版本: $(protoc --version)"
else
    echo "❌ protoc不可用"
fi

# 检查pkg-config
if pkg-config --exists protobuf; then
    echo "✅ protobuf pkg-config信息："
    echo "   版本: $(pkg-config --modversion protobuf)"
    echo "   包含目录: $(pkg-config --cflags protobuf)"
    echo "   链接库: $(pkg-config --libs protobuf)"
else
    echo "❌ pkg-config protobuf不可用"
fi

# 2. 查找protobuf头文件
echo ""
echo "🔍 查找protobuf头文件..."

# 检查标准位置
PROTOBUF_HEADER_PATHS=(
    "/usr/include/google/protobuf"
    "/usr/local/include/google/protobuf"
    "/usr/include/google/protobuf/port_def.inc"
    "/usr/local/include/google/protobuf/port_def.inc"
)

for path in "${PROTOBUF_HEADER_PATHS[@]}"; do
    if [ -e "$path" ]; then
        echo "✅ 找到: $path"
    else
        echo "❌ 缺失: $path"
    fi
done

# 3. 查找所有protobuf相关文件
echo ""
echo "🔍 查找所有protobuf相关文件..."
find /usr/include -name "*protobuf*" -type d 2>/dev/null | head -10
find /usr/include -name "port_def.inc" 2>/dev/null

# 4. 检查已安装的包
echo ""
echo "📦 检查已安装的protobuf包..."
if command -v dpkg &> /dev/null; then
    echo "已安装的protobuf相关包："
    dpkg -l | grep -i protobuf || echo "未找到protobuf包"
    echo ""
    echo "已安装的grpc相关包："
    dpkg -l | grep -i grpc || echo "未找到grpc包"
fi

# 5. 尝试修复
echo ""
echo "🔧 尝试修复protobuf问题..."

# 检查是否有gRPC依赖包
GRPC_DEPS_DIR="../tools/grpc-deps"
if [ -d "$GRPC_DEPS_DIR" ] && ls "$GRPC_DEPS_DIR"/*.deb 1> /dev/null 2>&1; then
    echo "📦 发现gRPC依赖包，重新安装..."
    cd "$GRPC_DEPS_DIR"
    
    # 按依赖顺序安装
    echo "🔄 按依赖顺序重新安装包..."
    
    # 先安装基础库
    sudo dpkg -i libc-ares2_*.deb 2>/dev/null || true
    sudo dpkg -i libprotobuf17_*.deb 2>/dev/null || true
    sudo dpkg -i libgrpc6_*.deb 2>/dev/null || true
    sudo dpkg -i libgrpc++1_*.deb 2>/dev/null || true
    
    # 再安装开发包
    sudo dpkg -i libprotobuf-dev_*.deb 2>/dev/null || true
    sudo dpkg -i protobuf-compiler_*.deb 2>/dev/null || true
    sudo dpkg -i libgrpc-dev_*.deb 2>/dev/null || true
    sudo dpkg -i libgrpc++-dev_*.deb 2>/dev/null || true
    
    # 修复依赖
    sudo apt-get install -f -y 2>/dev/null || {
        echo "⚠️  无法自动修复依赖，请手动处理"
    }
    
    cd - > /dev/null
else
    echo "📋 未找到gRPC依赖包，检查系统安装..."
fi

# 6. 验证修复结果
echo ""
echo "🔍 验证修复结果..."

# 重新检查protoc
if command -v protoc &> /dev/null; then
    echo "✅ protoc现在可用: $(protoc --version)"
else
    echo "❌ protoc仍然不可用"
fi

# 重新检查pkg-config
if pkg-config --exists protobuf; then
    echo "✅ protobuf pkg-config现在可用"
    echo "   版本: $(pkg-config --modversion protobuf)"
    echo "   包含目录: $(pkg-config --cflags protobuf)"
else
    echo "❌ protobuf pkg-config仍然不可用"
fi

# 检查关键头文件
echo ""
echo "🔍 检查关键头文件..."
if [ -f "/usr/include/google/protobuf/port_def.inc" ]; then
    echo "✅ 找到关键头文件: /usr/include/google/protobuf/port_def.inc"
elif [ -f "/usr/local/include/google/protobuf/port_def.inc" ]; then
    echo "✅ 找到关键头文件: /usr/local/include/google/protobuf/port_def.inc"
else
    echo "❌ 仍然缺少关键头文件: google/protobuf/port_def.inc"
    echo "💡 尝试查找其他位置..."
    find /usr -name "port_def.inc" 2>/dev/null | head -5
fi

# 7. 测试构建
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
