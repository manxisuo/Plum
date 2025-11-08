#!/bin/bash
# 修复缺失的ARM64 Go工具文件

set -e

echo "🔧 修复ARM64 Go工具文件..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

DEPLOY_DIR="plum-offline-deploy"
TOOLS_DIR="$DEPLOY_DIR/tools"

echo "📁 项目根目录: $PROJECT_ROOT"
echo "📁 工具目录: $TOOLS_DIR"

# 确保目录存在
mkdir -p "$TOOLS_DIR/go-arm64-tools/bin"

# 检查Go文件是否存在
if [ ! -f "$TOOLS_DIR/go1.24.3.linux-arm64.tar.gz" ]; then
    echo "❌ 未找到Go ARM64文件: $TOOLS_DIR/go1.24.3.linux-arm64.tar.gz"
    exit 1
fi

echo "✅ 找到Go ARM64文件"

# 创建临时目录重新编译
TEMP_DIR="/tmp/go-arm64-build-fix"
rm -rf $TEMP_DIR
mkdir -p $TEMP_DIR

echo "📦 解压Go并重新编译工具..."

cd $TEMP_DIR
tar -xzf "$PROJECT_ROOT/$TOOLS_DIR/go1.24.3.linux-arm64.tar.gz"

# 设置交叉编译环境
export PATH="$TEMP_DIR/go/bin:$PATH"
export GOOS=linux
export GOARCH=arm64
export GOBIN="$TEMP_DIR/go-arm64-tools/bin"
mkdir -p $GOBIN

echo "🔧 重新编译ARM64工具..."

# 编译工具
GOOS=linux GOARCH=arm64 go install google.golang.org/protobuf/cmd/protoc-gen-go@latest || {
    echo "❌ protoc-gen-go编译失败"
    exit 1
}

GOOS=linux GOARCH=arm64 go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest || {
    echo "❌ protoc-gen-go-grpc编译失败"
    exit 1
}

# 验证编译结果
if [ -f "$GOBIN/protoc-gen-go" ] && [ -f "$GOBIN/protoc-gen-go-grpc" ]; then
    echo "✅ 重新编译成功"
    
    # 复制到正确位置
    cp "$GOBIN/protoc-gen-go" "$PROJECT_ROOT/$TOOLS_DIR/go-arm64-tools/bin/"
    cp "$GOBIN/protoc-gen-go-grpc" "$PROJECT_ROOT/$TOOLS_DIR/go-arm64-tools/bin/"
    
    chmod +x "$PROJECT_ROOT/$TOOLS_DIR/go-arm64-tools/bin/"*
    
    echo "✅ ARM64工具已修复并复制到部署包"
    echo "📋 目录结构："
    ls -la "$PROJECT_ROOT/$TOOLS_DIR/go-arm64-tools/bin/"
else
    echo "❌ 重新编译失败"
    exit 1
fi

# 清理临时目录
cd "$PROJECT_ROOT"
rm -rf $TEMP_DIR

echo "🎉 ARM64 Go工具修复完成!"
