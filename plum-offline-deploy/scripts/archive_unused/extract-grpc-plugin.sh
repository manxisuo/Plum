#!/bin/bash
# 从已安装的gRPC包中提取grpc_cpp_plugin

set -e

echo "🔧 从已安装的gRPC包中提取grpc_cpp_plugin..."

# 检查是否在正确的目录
if [ ! -d "sdk/cpp" ] || [ ! -f "Makefile" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

echo "📁 当前目录: $(pwd)"

# 1. 检查已安装的gRPC包
echo ""
echo "📦 检查已安装的gRPC包..."
if command -v dpkg &> /dev/null; then
    echo "已安装的gRPC包："
    dpkg -l | grep -i grpc
else
    echo "❌ dpkg不可用"
    exit 1
fi

# 2. 查找gRPC包文件
echo ""
echo "🔍 查找gRPC包文件..."
GRPC_PACKAGES=(
    "libgrpc++-dev"
    "libgrpc-dev"
    "libgrpc6"
    "libgrpc++1"
)

for pkg in "${GRPC_PACKAGES[@]}"; do
    echo "📋 检查包: $pkg"
    if dpkg -l | grep -q "^ii.*$pkg"; then
        echo "✅ 已安装: $pkg"
        
        # 查找包文件
        PKG_FILES=$(dpkg -L "$pkg" 2>/dev/null | grep -E "(grpc_cpp_plugin|grpc.*plugin)" || true)
        if [ -n "$PKG_FILES" ]; then
            echo "✅ 找到相关文件:"
            echo "$PKG_FILES"
        else
            echo "❌ 未找到grpc_cpp_plugin相关文件"
        fi
    else
        echo "❌ 未安装: $pkg"
    fi
done

# 3. 查找所有可能的grpc插件
echo ""
echo "🔍 查找所有可能的grpc插件..."
find /usr -name "*grpc*plugin*" -type f 2>/dev/null | head -10
find /usr -name "*grpc*" -type f -executable 2>/dev/null | grep -v ".so" | head -10

# 4. 检查gRPC依赖包
echo ""
echo "🔍 检查gRPC依赖包..."
GRPC_DEPS_DIR="../tools/grpc-deps"
if [ -d "$GRPC_DEPS_DIR" ] && ls "$GRPC_DEPS_DIR"/*.deb 1> /dev/null 2>&1; then
    echo "📦 发现gRPC依赖包，检查内容..."
    cd "$GRPC_DEPS_DIR"
    
    for deb_file in *.deb; do
        if [ -f "$deb_file" ]; then
            echo "📋 检查包: $deb_file"
            if dpkg -c "$deb_file" | grep -q "grpc_cpp_plugin"; then
                echo "✅ 包含 grpc_cpp_plugin"
                dpkg -c "$deb_file" | grep "grpc_cpp_plugin"
                
                # 提取包内容
                TEMP_DIR="/tmp/grpc-extract"
                rm -rf "$TEMP_DIR"
                mkdir -p "$TEMP_DIR"
                
                echo "📦 提取包内容..."
                dpkg -x "$deb_file" "$TEMP_DIR"
                
                # 查找grpc_cpp_plugin
                PLUGIN_PATH=$(find "$TEMP_DIR" -name "grpc_cpp_plugin" -type f 2>/dev/null | head -1)
                if [ -n "$PLUGIN_PATH" ]; then
                    echo "✅ 找到插件: $PLUGIN_PATH"
                    
                    # 复制到系统路径
                    sudo cp "$PLUGIN_PATH" /usr/local/bin/grpc_cpp_plugin
                    sudo chmod +x /usr/local/bin/grpc_cpp_plugin
                    echo "✅ 已安装 grpc_cpp_plugin 到 /usr/local/bin/"
                    
                    # 验证安装
                    if [ -x "/usr/local/bin/grpc_cpp_plugin" ]; then
                        echo "✅ grpc_cpp_plugin 现在可用"
                        /usr/local/bin/grpc_cpp_plugin --help | head -3
                    fi
                fi
                
                # 清理临时目录
                rm -rf "$TEMP_DIR"
                break
            else
                echo "❌ 不包含 grpc_cpp_plugin"
            fi
        fi
    done
    
    cd - > /dev/null
else
    echo "📋 未找到gRPC依赖包"
fi

# 5. 如果仍然没有找到，尝试从源码编译
echo ""
echo "🔧 如果仍然没有找到grpc_cpp_plugin，尝试其他方法..."

# 检查是否有protobuf-compiler-grpc包
if command -v apt &> /dev/null; then
    echo "🔍 检查是否有protobuf-compiler-grpc包..."
    apt list --installed | grep -i grpc || echo "未找到grpc相关包"
    
    echo "💡 建议安装protobuf-compiler-grpc包："
    echo "   sudo apt install protobuf-compiler-grpc"
fi

# 6. 验证最终结果
echo ""
echo "🔍 验证最终结果..."
if command -v grpc_cpp_plugin &> /dev/null; then
    echo "✅ grpc_cpp_plugin 现在可用: $(which grpc_cpp_plugin)"
    grpc_cpp_plugin --help | head -3
else
    echo "❌ grpc_cpp_plugin 仍然不可用"
    echo "💡 可能需要安装额外的gRPC插件包"
fi

echo ""
echo "🎯 提取完成！"
