#!/bin/bash
# 检查gRPC插件

echo "🔍 检查gRPC插件..."

# 检查当前系统
echo "📋 检查当前系统中的grpc_cpp_plugin："
if command -v grpc_cpp_plugin &> /dev/null; then
    echo "✅ 找到: $(which grpc_cpp_plugin)"
    grpc_cpp_plugin --help | head -3
else
    echo "❌ 未找到 grpc_cpp_plugin"
fi

echo ""
echo "📋 搜索系统中的grpc_cpp_plugin："
find /usr -name "grpc_cpp_plugin" 2>/dev/null | head -5

echo ""
echo "📋 检查已安装的gRPC包："
if command -v dpkg &> /dev/null; then
    dpkg -l | grep -i grpc || echo "未找到gRPC包"
fi

echo ""
echo "📋 检查gRPC依赖包内容..."
GRPC_DEPS_DIR="../tools/grpc-deps"
if [ -d "$GRPC_DEPS_DIR" ] && ls "$GRPC_DEPS_DIR"/*.deb 1> /dev/null 2>&1; then
    echo "发现gRPC依赖包："
    ls -la "$GRPC_DEPS_DIR"/*.deb
    
    echo ""
    echo "检查每个包的内容："
    for deb_file in "$GRPC_DEPS_DIR"/*.deb; do
        if [ -f "$deb_file" ]; then
            echo "📦 检查: $(basename "$deb_file")"
            if dpkg -c "$deb_file" | grep -q "grpc_cpp_plugin"; then
                echo "✅ 包含 grpc_cpp_plugin"
                dpkg -c "$deb_file" | grep "grpc_cpp_plugin"
            else
                echo "❌ 不包含 grpc_cpp_plugin"
            fi
        fi
    done
else
    echo "❌ 未找到gRPC依赖包目录"
fi
