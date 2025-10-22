#!/bin/bash
# 安装gRPC插件

set -e

echo "🔧 安装gRPC插件..."

# 检查是否在正确的目录
if [ ! -d "sdk/cpp" ] || [ ! -f "Makefile" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

echo "📁 当前目录: $(pwd)"

# 1. 检查当前状态
echo ""
echo "🔍 检查当前状态..."
if command -v grpc_cpp_plugin &> /dev/null; then
    echo "✅ grpc_cpp_plugin 已可用: $(which grpc_cpp_plugin)"
    exit 0
else
    echo "❌ grpc_cpp_plugin 不可用"
fi

# 2. 检查是否有protobuf-compiler-grpc包
echo ""
echo "🔍 检查protobuf-compiler-grpc包..."
if command -v apt &> /dev/null; then
    echo "📦 检查可用的gRPC相关包..."
    apt list --installed | grep -i grpc || echo "未找到gRPC包"
    
    echo ""
    echo "📦 检查可用的protobuf-compiler-grpc包..."
    apt list --available | grep -i "protobuf-compiler-grpc" || echo "未找到protobuf-compiler-grpc包"
    
    echo ""
    echo "📦 检查可用的gRPC插件包..."
    apt list --available | grep -i "grpc.*plugin" || echo "未找到gRPC插件包"
fi

# 3. 尝试安装protobuf-compiler-grpc
echo ""
echo "🔧 尝试安装protobuf-compiler-grpc..."
if command -v apt &> /dev/null; then
    echo "📥 尝试安装protobuf-compiler-grpc..."
    sudo apt update 2>/dev/null || echo "⚠️  apt update失败，可能是离线环境"
    
    if sudo apt install -y protobuf-compiler-grpc 2>/dev/null; then
        echo "✅ protobuf-compiler-grpc 安装成功"
    else
        echo "❌ protobuf-compiler-grpc 安装失败"
        echo "💡 可能是离线环境，需要手动安装"
    fi
else
    echo "❌ apt不可用，无法安装包"
fi

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

# 5. 验证最终结果
echo ""
echo "🔍 验证最终结果..."
if command -v grpc_cpp_plugin &> /dev/null; then
    echo "✅ grpc_cpp_plugin 现在可用: $(which grpc_cpp_plugin)"
    grpc_cpp_plugin --help | head -3
else
    echo "❌ grpc_cpp_plugin 仍然不可用"
    echo "💡 可能需要手动安装protobuf-compiler-grpc包"
fi

echo ""
echo "🎯 安装完成！"
