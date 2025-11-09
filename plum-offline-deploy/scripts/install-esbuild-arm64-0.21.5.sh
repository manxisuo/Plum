#!/bin/bash
# 安装 esbuild ARM64 0.21.5 版本脚本

set -e

echo "🚀 安装 esbuild ARM64 版本 0.21.5..."

# 检查是否在正确的目录
if [ ! -d "ui" ] || [ ! -f "ui/package.json" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    echo "   当前目录: $(pwd)"
    echo "   期望找到: ui/package.json"
    exit 1
fi

cd ui

echo "📁 当前目录: $(pwd)"

# 检查当前 esbuild 版本
if [ -f "node_modules/esbuild/package.json" ]; then
    CURRENT_VERSION=$(grep '"version"' node_modules/esbuild/package.json | cut -d'"' -f4)
    echo "📋 当前 esbuild 版本: $CURRENT_VERSION"
    if [ "$CURRENT_VERSION" != "0.21.5" ]; then
        echo "⚠️  版本不匹配，但继续安装 ARM64 版本..."
    fi
else
    echo "❌ 未找到 esbuild package.json 文件"
    exit 1
fi

# 强制检查并删除 x64 包和相关文件
echo "🔍 检查并清理 x64 相关包..."

# 删除所有可能的x64包
rm -rf node_modules/@esbuild/esbuild-linux-x64 2>/dev/null || true
rm -rf node_modules/@esbuild/linux-x64 2>/dev/null || true

# 检查是否存在x64包
if [ -d "node_modules/@esbuild/esbuild-linux-x64" ] || [ -d "node_modules/@esbuild/linux-x64" ]; then
    echo "⚠️  仍然存在 x64 包，强制删除..."
    rm -rf node_modules/@esbuild/esbuild-linux-x64 node_modules/@esbuild/linux-x64
    echo "✅ 已强制删除 x64 包"
else
    echo "✅ 没有发现 x64 包"
fi

# 检查是否已有完整的 ARM64 包
ARM64_READY=false
if [ -d "node_modules/@esbuild/linux-arm64" ]; then
    # 检查ARM64包是否完整
    if [ -f "node_modules/@esbuild/linux-arm64/package.json" ] && \
       [ -f "node_modules/@esbuild/linux-arm64/esbuild" ]; then
        echo "✅ ARM64 包已存在且完整"
        ARM64_READY=true
    else
        echo "⚠️  ARM64 包存在但不完整，重新安装..."
        rm -rf node_modules/@esbuild/linux-arm64
    fi
fi

if [ "$ARM64_READY" = false ]; then
    # 查找可能的 tarball 文件
    POSSIBLE_FILES=(
        "esbuild-linux-arm64-0.21.5.tgz"
        "linux-arm64-0.21.5.tgz"
        "../esbuild-linux-arm64-0.21.5.tgz"
        "../../esbuild-linux-arm64-0.21.5.tgz"
        "../tools/esbuild-linux-arm64-0.21.5.tgz"
        "../../tools/esbuild-linux-arm64-0.21.5.tgz"
        "../../../tools/esbuild-linux-arm64-0.21.5.tgz"
        "~/esbuild-linux-arm64-0.21.5.tgz"
        "$HOME/esbuild-linux-arm64-0.21.5.tgz"
        "/tmp/esbuild-linux-arm64-0.21.5.tgz"
    )
    
    TARBALL_FILE=""
    for file in "${POSSIBLE_FILES[@]}"; do
        if [ -f "$file" ]; then
            TARBALL_FILE="$file"
            echo "✅ 找到已下载的文件: $TARBALL_FILE"
            break
        fi
    done
    
    if [ -n "$TARBALL_FILE" ]; then
        echo "📦 使用本地文件: $TARBALL_FILE"
    else
        echo "📥 未找到本地文件，尝试下载 esbuild ARM64 0.21.5..."
        
        DOWNLOAD_URL="https://registry.npmjs.org/@esbuild/linux-arm64/-/linux-arm64-0.21.5.tgz"
        TARBALL_FILE="linux-arm64-0.21.5.tgz"
        
        echo "🔗 下载链接: $DOWNLOAD_URL"
        
        # 尝试下载（如果网络可用）
        if command -v wget &> /dev/null; then
            echo "⬇️  使用 wget 下载..."
            if wget -O "$TARBALL_FILE" "$DOWNLOAD_URL" 2>/dev/null; then
                echo "✅ 下载成功"
            else
                echo "❌ wget 下载失败，可能是离线环境"
                echo "💡 请将 esbuild-linux-arm64-0.21.5.tgz 文件放到以下位置之一："
                for file in "${POSSIBLE_FILES[@]}"; do
                    echo "   - $file"
                done
                exit 1
            fi
        elif command -v curl &> /dev/null; then
            echo "⬇️  使用 curl 下载..."
            if curl -L -o "$TARBALL_FILE" "$DOWNLOAD_URL" 2>/dev/null; then
                echo "✅ 下载成功"
            else
                echo "❌ curl 下载失败，可能是离线环境"
                echo "💡 请将 esbuild-linux-arm64-0.21.5.tgz 文件放到以下位置之一："
                for file in "${POSSIBLE_FILES[@]}"; do
                    echo "   - $file"
                done
                exit 1
            fi
        else
            echo "❌ 没有找到 wget 或 curl 下载工具"
            echo "💡 离线环境，请将 esbuild-linux-arm64-0.21.5.tgz 文件放到以下位置之一："
            for file in "${POSSIBLE_FILES[@]}"; do
                echo "   - $file"
            done
            exit 1
        fi
    fi
    
    # 验证文件存在
    if [ ! -f "$TARBALL_FILE" ]; then
        echo "❌ 文件不存在: $TARBALL_FILE"
        exit 1
    fi
    
    echo "📁 解压并安装 ARM64 esbuild..."
    echo "📦 使用文件: $TARBALL_FILE"
    
    # 先验证文件
    echo "🔍 验证文件完整性..."
    if [ ! -s "$TARBALL_FILE" ]; then
        echo "❌ 文件为空或不存在"
        exit 1
    fi
    
    echo "📊 文件大小: $(ls -lh "$TARBALL_FILE" | awk '{print $5}')"
    
    # 测试文件是否是有效的 tar.gz
    if ! gzip -t "$TARBALL_FILE" 2>/dev/null; then
        echo "⚠️  文件可能损坏，但仍尝试解压..."
    fi
    
    # 解压文件 - 使用更详细的错误信息
    echo "🔧 开始解压..."
    if tar -xzf "$TARBALL_FILE" 2>&1; then
        echo "✅ 解压成功"
    else
        TAR_ERROR=$?
        echo "❌ 解压失败，错误码: $TAR_ERROR"
        echo "🔍 尝试其他解压方法..."
        
        # 尝试先解压 gz 再解压 tar
        if command -v gunzip &> /dev/null; then
            echo "🔄 尝试分步解压..."
            cp "$TARBALL_FILE" temp_file.tgz
            if gunzip temp_file.tgz 2>/dev/null && tar -xf temp_file.tar 2>/dev/null; then
                echo "✅ 分步解压成功"
                rm -f temp_file.tar
            else
                echo "❌ 分步解压也失败"
                rm -f temp_file.tar temp_file.tgz 2>/dev/null
                exit 1
            fi
        else
            echo "❌ 无法恢复，请检查文件是否完整"
            exit 1
        fi
    fi
    
    if [ ! -d "package" ]; then
        echo "❌ 解压失败，package 目录不存在"
        echo "📋 当前目录内容:"
        ls -la
        exit 1
    fi
    
    # 安装到正确位置 - esbuild期望的目录名是 linux-arm64，不是 esbuild-linux-arm64
    mkdir -p node_modules/@esbuild/linux-arm64
    cp -r package/* node_modules/@esbuild/linux-arm64/
    
    # 清理临时文件（只清理解压出来的 package 目录）
    rm -rf package
    
    # 如果是从网络下载的临时文件才删除，本地文件保留
    if [[ "$TARBALL_FILE" == "linux-arm64-0.21.5.tgz" ]]; then
        rm -f "$TARBALL_FILE"
        echo "🗑️  清理临时下载文件"
    else
        echo "📁 保留本地文件: $TARBALL_FILE"
    fi
    
    echo "✅ ARM64 esbuild 安装完成"
fi

# 最终验证和强制清理
echo ""
echo "🔍 最终验证和清理..."

# 再次强制清理任何残留的x64包
echo "🗑️  最终清理 x64 包..."
rm -rf node_modules/@esbuild/esbuild-linux-x64 2>/dev/null || true
rm -rf node_modules/@esbuild/linux-x64 2>/dev/null || true

if [ -d "node_modules/@esbuild" ]; then
    echo "📋 @esbuild 目录内容："
    ls -la node_modules/@esbuild/
    
    # 检查是否还有x64相关文件
    echo ""
    echo "🔍 检查是否还有 x64 相关文件..."
    find node_modules/@esbuild/ -name "*x64*" 2>/dev/null && {
        echo "⚠️  发现残留的 x64 文件，删除中..."
        find node_modules/@esbuild/ -name "*x64*" -exec rm -rf {} + 2>/dev/null || true
    }
    
    if [ -d "node_modules/@esbuild/linux-arm64" ]; then
        echo "✅ ARM64 esbuild 包已安装"
        
        # 检查二进制文件
        if [ -f "node_modules/@esbuild/linux-arm64/esbuild" ]; then
            echo "✅ ARM64 二进制文件存在"
            echo "📊 文件信息："
            file node_modules/@esbuild/linux-arm64/esbuild
        else
            echo "⚠️  二进制文件缺失，检查包内容："
            ls -la node_modules/@esbuild/linux-arm64/
        fi
        
        # 最终检查
        if find node_modules/@esbuild/ -name "*x64*" 2>/dev/null | grep -q .; then
            echo "❌ 仍然存在 x64 相关文件!"
            find node_modules/@esbuild/ -name "*x64*"
            exit 1
        else
            echo "✅ 确认没有任何 x64 包冲突"
        fi
    else
        echo "❌ ARM64 esbuild 包安装失败"
        exit 1
    fi
else
    echo "❌ @esbuild 目录不存在"
    exit 1
fi

cd ..

echo ""
echo "🎉 esbuild ARM64 0.21.5 安装完成！"
echo "   现在可以尝试运行: make ui-dev"
