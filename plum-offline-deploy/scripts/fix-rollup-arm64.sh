#!/bin/bash
# 在目标ARM64机器上修复 rollup ARM64 依赖问题

set -e

echo "🚀 修复 rollup ARM64 依赖问题..."

# 检查是否在正确的目录
if [ ! -d "ui" ] || [ ! -f "ui/package.json" ]; then
    echo "❌ 请在包含 ui 目录的项目根目录运行此脚本"
    echo "   当前目录: $(pwd)"
    echo "   期望找到: ui/package.json"
    exit 1
fi

echo "📁 当前目录: $(pwd)"

# 检查是否已有 rollup ARM64 模块
if [ -d "ui/node_modules/@rollup/rollup-linux-arm64-gnu" ]; then
    echo "✅ @rollup/rollup-linux-arm64-gnu 目录已存在"
    echo "📋 目录内容:"
    ls -la ui/node_modules/@rollup/rollup-linux-arm64-gnu/
    
    if [ -f "ui/node_modules/@rollup/rollup-linux-arm64-gnu/rollup.linux-arm64-gnu.node" ]; then
        echo "✅ rollup.linux-arm64-gnu.node 文件存在"
        echo "📊 文件信息:"
        file ui/node_modules/@rollup/rollup-linux-arm64-gnu/rollup.linux-arm64-gnu.node
        echo ""
        echo "🎉 rollup ARM64 模块已存在，无需修复"
        exit 0
    else
        echo "⚠️  目录存在但缺少二进制文件，尝试重新安装..."
    fi
fi

echo "🔍 查找本地的 rollup ARM64 tarball..."

# 查找可能的 tarball 文件
POSSIBLE_FILES=(
    "rollup-linux-arm64-gnu-4.52.5.tgz"
    "ui/rollup-linux-arm64-gnu-4.52.5.tgz"
    "../tools/rollup-linux-arm64-gnu-4.52.5.tgz"
    "../../tools/rollup-linux-arm64-gnu-4.52.5.tgz"
    "../../rollup-linux-arm64-gnu-4.52.5.tgz"
    "/tmp/rollup-linux-arm64-gnu-4.52.5.tgz"
)

TARBALL_FILE=""
for file in "${POSSIBLE_FILES[@]}"; do
    if [ -f "$file" ]; then
        TARBALL_FILE="$file"
        echo "✅ 找到 tarball: $TARBALL_FILE"
        break
    fi
done

if [ -z "$TARBALL_FILE" ]; then
    echo "❌ 未找到 rollup-linux-arm64-gnu-4.52.5.tgz 文件"
    echo "📋 请将文件放到以下位置之一："
    for file in "${POSSIBLE_FILES[@]}"; do
        echo "   - $file"
    done
    echo ""
    echo "💡 可以通过以下方式获取："
    echo "   1. 在WSL2环境中运行: bash scripts-prepare/download-rollup-arm64.sh"
    echo "   2. 手动下载到项目根目录"
    exit 1
fi

echo "📦 使用 tarball: $TARBALL_FILE"

# 验证 tarball
if [ ! -s "$TARBALL_FILE" ]; then
    echo "❌ tarball 文件为空或损坏"
    exit 1
fi

echo "📊 文件大小: $(ls -lh "$TARBALL_FILE" | awk '{print $5}')"

# 创建临时目录进行解压
TEMP_DIR=$(mktemp -d)
echo "📁 临时目录: $TEMP_DIR"

# 保存当前目录
OLDPWD=$(pwd)

# 解压 tarball
echo "📦 解压 tarball..."
cd "$TEMP_DIR"
tar -xzf "$OLDPWD/$TARBALL_FILE"

echo "📋 tarball 内容:"
ls -la

if [ ! -f "package/rollup.linux-arm64-gnu.node" ]; then
    echo "❌ tarball 中缺少 rollup.linux-arm64-gnu.node 文件"
    echo "📋 package 目录内容:"
    ls -la package/ 2>/dev/null || echo "package 目录不存在"
    exit 1
fi

echo ""
echo "📁 安装到 node_modules..."
cd "$OLDPWD"

# 确保目标目录存在
mkdir -p ui/node_modules/@rollup/rollup-linux-arm64-gnu

# 复制文件
cp -r "$TEMP_DIR/package/"* ui/node_modules/@rollup/rollup-linux-arm64-gnu/

# 清理临时目录
rm -rf "$TEMP_DIR"

echo ""
echo "🔍 验证安装..."
if [ -d "ui/node_modules/@rollup/rollup-linux-arm64-gnu" ]; then
    echo "✅ @rollup/rollup-linux-arm64-gnu 目录已创建"
    ls -la ui/node_modules/@rollup/rollup-linux-arm64-gnu/
    
    if [ -f "ui/node_modules/@rollup/rollup-linux-arm64-gnu/rollup.linux-arm64-gnu.node" ]; then
        echo "✅ rollup.linux-arm64-gnu.node 已安装"
        echo "📊 文件信息:"
        file ui/node_modules/@rollup/rollup-linux-arm64-gnu/rollup.linux-arm64-gnu.node
        
        echo ""
        echo "🎉 rollup ARM64 模块安装成功！"
        echo "   现在可以尝试运行: make ui-dev"
    else
        echo "❌ rollup.linux-arm64-gnu.node 文件安装失败"
        exit 1
    fi
else
    echo "❌ 目标目录创建失败"
    exit 1
fi
