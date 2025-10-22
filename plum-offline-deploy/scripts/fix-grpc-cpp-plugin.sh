#!/bin/bash
# 修复grpc_cpp_plugin问题

set -e

echo "🔧 修复grpc_cpp_plugin问题..."

# 检查是否在正确的目录
if [ ! -d "proto" ] || [ ! -f "Makefile" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

echo "📁 当前目录: $(pwd)"

# 1. 查找grpc_cpp_plugin
echo ""
echo "🔍 查找grpc_cpp_plugin..."

# 检查常见位置
GRPC_PLUGIN_PATHS=(
    "/usr/bin/grpc_cpp_plugin"
    "/usr/local/bin/grpc_cpp_plugin"
    "/usr/lib/grpc/bin/grpc_cpp_plugin"
    "/usr/lib/x86_64-linux-gnu/grpc/bin/grpc_cpp_plugin"
    "/usr/lib/aarch64-linux-gnu/grpc/bin/grpc_cpp_plugin"
)

for path in "${GRPC_PLUGIN_PATHS[@]}"; do
    if [ -x "$path" ]; then
        echo "✅ 找到: $path"
    else
        echo "❌ 缺失: $path"
    fi
done

# 搜索系统中的grpc_cpp_plugin
echo ""
echo "🔍 搜索系统中的grpc_cpp_plugin..."
find /usr -name "grpc_cpp_plugin" 2>/dev/null | head -5

# 2. 检查已安装的gRPC包
echo ""
echo "📦 检查已安装的gRPC包..."
if command -v dpkg &> /dev/null; then
    echo "已安装的gRPC相关包："
    dpkg -l | grep -i grpc || echo "未找到gRPC包"
fi

# 3. 检查gRPC依赖包
echo ""
echo "🔍 检查gRPC依赖包..."
GRPC_DEPS_DIR="../tools/grpc-deps"
if [ -d "$GRPC_DEPS_DIR" ] && ls "$GRPC_DEPS_DIR"/*.deb 1> /dev/null 2>&1; then
    echo "📦 发现gRPC依赖包，检查内容..."
    cd "$GRPC_DEPS_DIR"
    
    # 检查每个包的内容
    for deb_file in *.deb; do
        if [ -f "$deb_file" ]; then
            echo "📋 检查包: $deb_file"
            dpkg -c "$deb_file" | grep -i grpc_cpp_plugin || echo "  未找到grpc_cpp_plugin"
        fi
    done
    
    cd - > /dev/null
else
    echo "📋 未找到gRPC依赖包"
fi

# 4. 尝试从包中提取grpc_cpp_plugin
echo ""
echo "🔧 尝试从包中提取grpc_cpp_plugin..."

if [ -d "$GRPC_DEPS_DIR" ] && ls "$GRPC_DEPS_DIR"/*.deb 1> /dev/null 2>&1; then
    cd "$GRPC_DEPS_DIR"
    
    # 查找包含grpc_cpp_plugin的包
    for deb_file in *.deb; do
        if [ -f "$deb_file" ]; then
            if dpkg -c "$deb_file" | grep -q "grpc_cpp_plugin"; then
                echo "✅ 在 $deb_file 中找到 grpc_cpp_plugin"
                
                # 提取包内容到临时目录
                TEMP_DIR="/tmp/grpc-extract"
                rm -rf "$TEMP_DIR"
                mkdir -p "$TEMP_DIR"
                
                # 提取包
                dpkg -x "$deb_file" "$TEMP_DIR"
                
                # 查找grpc_cpp_plugin
                PLUGIN_PATH=$(find "$TEMP_DIR" -name "grpc_cpp_plugin" -type f 2>/dev/null | head -1)
                if [ -n "$PLUGIN_PATH" ]; then
                    echo "✅ 找到插件: $PLUGIN_PATH"
                    
                    # 复制到系统路径
                    sudo cp "$PLUGIN_PATH" /usr/bin/grpc_cpp_plugin
                    sudo chmod +x /usr/bin/grpc_cpp_plugin
                    echo "✅ 已安装 grpc_cpp_plugin 到 /usr/bin/"
                    
                    # 验证安装
                    if [ -x "/usr/bin/grpc_cpp_plugin" ]; then
                        echo "✅ grpc_cpp_plugin 现在可用"
                        /usr/bin/grpc_cpp_plugin --help | head -3
                    fi
                fi
                
                # 清理临时目录
                rm -rf "$TEMP_DIR"
                break
            fi
        fi
    done
    
    cd - > /dev/null
else
    echo "📋 未找到gRPC依赖包，无法提取插件"
fi

# 5. 验证修复结果
echo ""
echo "🔍 验证修复结果..."
if command -v grpc_cpp_plugin &> /dev/null; then
    echo "✅ grpc_cpp_plugin 现在可用: $(which grpc_cpp_plugin)"
    grpc_cpp_plugin --help | head -3
else
    echo "❌ grpc_cpp_plugin 仍然不可用"
fi

# 6. 测试proto生成
echo ""
echo "🧪 测试proto生成..."
if [ -d "proto" ]; then
    echo "🔄 重新运行 make proto..."
    make proto
else
    echo "❌ proto目录不存在"
fi

echo ""
echo "🎯 修复完成！现在可以尝试："
echo "   make sdk_cpp_offline"
# 修复grpc_cpp_plugin问题

set -e

echo "🔧 修复grpc_cpp_plugin问题..."

# 检查是否在正确的目录
if [ ! -d "sdk/cpp" ] || [ ! -f "Makefile" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

echo "📁 当前目录: $(pwd)"

# 1. 查找grpc_cpp_plugin
echo ""
echo "🔍 查找grpc_cpp_plugin..."

# 检查是否在PATH中
if command -v grpc_cpp_plugin &> /dev/null; then
    echo "✅ grpc_cpp_plugin在PATH中: $(which grpc_cpp_plugin)"
    grpc_cpp_plugin --version 2>/dev/null || echo "⚠️  无法获取版本信息"
else
    echo "❌ grpc_cpp_plugin不在PATH中"
fi

# 查找所有可能的grpc_cpp_plugin位置
echo ""
echo "🔍 查找所有grpc_cpp_plugin位置..."
find /usr -name "grpc_cpp_plugin" -type f 2>/dev/null | head -10

# 2. 检查gRPC安装
echo ""
echo "📦 检查gRPC安装..."

# 检查已安装的gRPC包
if command -v dpkg &> /dev/null; then
    echo "已安装的gRPC相关包："
    dpkg -l | grep -i grpc || echo "未找到gRPC包"
fi

# 检查pkg-config
if pkg-config --exists grpc++; then
    echo "✅ gRPC pkg-config信息："
    echo "   版本: $(pkg-config --modversion grpc++)"
    echo "   包含目录: $(pkg-config --cflags grpc++)"
    echo "   链接库: $(pkg-config --libs grpc++)"
else
    echo "❌ pkg-config grpc++不可用"
fi

# 3. 尝试修复
echo ""
echo "🔧 尝试修复grpc_cpp_plugin问题..."

# 检查是否有gRPC依赖包
GRPC_DEPS_DIR="../tools/grpc-deps"
if [ -d "$GRPC_DEPS_DIR" ] && ls "$GRPC_DEPS_DIR"/*.deb 1> /dev/null 2>&1; then
    echo "📦 发现gRPC依赖包，重新安装..."
    cd "$GRPC_DEPS_DIR"
    
    # 重新安装gRPC开发包
    echo "🔄 重新安装gRPC开发包..."
    sudo dpkg -i libgrpc++-dev_*.deb 2>/dev/null || {
        echo "⚠️  libgrpc++-dev安装失败，尝试修复依赖..."
        sudo apt-get install -f -y 2>/dev/null || {
            echo "⚠️  无法自动修复依赖，请手动处理"
        }
    }
    
    cd - > /dev/null
else
    echo "📋 未找到gRPC依赖包，检查系统安装..."
fi

# 4. 查找并创建符号链接
echo ""
echo "🔍 查找grpc_cpp_plugin并创建符号链接..."

# 查找所有grpc_cpp_plugin
GRPC_PLUGIN_PATHS=$(find /usr -name "grpc_cpp_plugin" -type f 2>/dev/null)

if [ -n "$GRPC_PLUGIN_PATHS" ]; then
    echo "✅ 找到grpc_cpp_plugin："
    echo "$GRPC_PLUGIN_PATHS"
    
    # 选择第一个找到的插件
    FIRST_PLUGIN=$(echo "$GRPC_PLUGIN_PATHS" | head -1)
    echo "📋 使用插件: $FIRST_PLUGIN"
    
    # 检查插件是否可执行
    if [ -x "$FIRST_PLUGIN" ]; then
        echo "✅ 插件可执行"
        
        # 创建符号链接到/usr/local/bin
        if [ ! -L "/usr/local/bin/grpc_cpp_plugin" ]; then
            echo "🔗 创建符号链接..."
            sudo ln -sf "$FIRST_PLUGIN" /usr/local/bin/grpc_cpp_plugin
            echo "✅ 符号链接已创建: /usr/local/bin/grpc_cpp_plugin -> $FIRST_PLUGIN"
        else
            echo "✅ 符号链接已存在: /usr/local/bin/grpc_cpp_plugin"
        fi
    else
        echo "❌ 插件不可执行，尝试修复权限..."
        sudo chmod +x "$FIRST_PLUGIN"
        if [ -x "$FIRST_PLUGIN" ]; then
            echo "✅ 权限修复成功"
            sudo ln -sf "$FIRST_PLUGIN" /usr/local/bin/grpc_cpp_plugin
        else
            echo "❌ 权限修复失败"
        fi
    fi
else
    echo "❌ 未找到grpc_cpp_plugin"
    echo "💡 可能需要安装额外的gRPC包"
fi

# 5. 验证修复结果
echo ""
echo "🔍 验证修复结果..."

# 检查PATH中的grpc_cpp_plugin
if command -v grpc_cpp_plugin &> /dev/null; then
    echo "✅ grpc_cpp_plugin现在在PATH中: $(which grpc_cpp_plugin)"
    grpc_cpp_plugin --version 2>/dev/null || echo "⚠️  无法获取版本信息"
else
    echo "❌ grpc_cpp_plugin仍然不在PATH中"
fi

# 6. 测试protobuf生成
echo ""
echo "🧪 测试protobuf生成..."
if [ -d "proto" ]; then
    echo "🔄 重新运行protobuf生成..."
    make proto
else
    echo "❌ 未找到proto目录"
fi

echo ""
echo "🎯 修复完成！"
echo "如果grpc_cpp_plugin仍然不可用，请检查："
echo "1. 是否安装了完整的gRPC开发包"
echo "2. 插件文件是否存在于系统中"
echo "3. 是否需要额外的gRPC插件包"
