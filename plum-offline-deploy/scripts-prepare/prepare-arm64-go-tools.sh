#!/bin/bash
# 准备ARM64版本的Go protobuf工具

set -e

echo "🚀 准备ARM64版本的Go protobuf工具..."

# 确保在项目根目录下运行
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

DEPLOY_DIR="plum-offline-deploy"
TOOLS_DIR="$DEPLOY_DIR/tools"

# 检查是否已有ARM64 Go
ARM64_GO_FILE=""
if [ -f "$TOOLS_DIR/go1.23.12.linux-arm64.tar.gz" ]; then
    ARM64_GO_FILE="$TOOLS_DIR/go1.23.12.linux-arm64.tar.gz"
    # 转换为绝对路径，避免切换目录后找不到文件
    if command -v realpath &> /dev/null; then
        ARM64_GO_FILE="$(realpath "$ARM64_GO_FILE")"
    else
        # 如果realpath不可用，使用cd和pwd的方式
        ARM64_GO_FILE="$(cd "$(dirname "$ARM64_GO_FILE")" && pwd)/$(basename "$ARM64_GO_FILE")"
    fi
    echo "✅ 找到ARM64 Go: $ARM64_GO_FILE"
else
    echo "❌ 未找到go1.23.12.linux-arm64.tar.gz文件"
    echo "请将文件放到: $TOOLS_DIR/"
    exit 1
fi

# 创建临时目录
TEMP_DIR="/tmp/go-arm64-build"
rm -rf $TEMP_DIR
mkdir -p $TEMP_DIR

# 解压ARM64 Go
echo "📦 解压ARM64 Go..."
cd $TEMP_DIR
echo "从 $ARM64_GO_FILE 解压到 $TEMP_DIR"
tar -xzf "$ARM64_GO_FILE"

# 设置交叉编译环境
export PATH="$TEMP_DIR/go/bin:$PATH"
export GOOS=linux
export GOARCH=arm64
export GOBIN="$TEMP_DIR/go-arm64-tools/bin"
mkdir -p $GOBIN

echo "🔧 开始交叉编译ARM64工具..."

# 验证Go版本
echo "Go版本: $(go version)"

# 安装protoc-gen-go ARM64版本
echo "📦 编译protoc-gen-go ARM64版本..."
GOOS=linux GOARCH=arm64 go install google.golang.org/protobuf/cmd/protoc-gen-go@latest || {
    echo "❌ protoc-gen-go编译失败"
    exit 1
}

# 安装protoc-gen-go-grpc ARM64版本
echo "📦 编译protoc-gen-go-grpc ARM64版本..."
GOOS=linux GOARCH=arm64 go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest || {
    echo "❌ protoc-gen-go-grpc编译失败"
    exit 1
}

# 验证编译结果
echo "🔍 验证编译结果..."
echo "检查文件: $GOBIN/protoc-gen-go"
ls -la "$GOBIN/protoc-gen-go" 2>/dev/null || echo "protoc-gen-go文件不存在"
echo "检查文件: $GOBIN/protoc-gen-go-grpc"
ls -la "$GOBIN/protoc-gen-go-grpc" 2>/dev/null || echo "protoc-gen-go-grpc文件不存在"

if [ -f "$GOBIN/protoc-gen-go" ] && [ -f "$GOBIN/protoc-gen-go-grpc" ]; then
    echo "✅ ARM64工具编译成功!"
    
    # 检查架构
    echo "protoc-gen-go架构: $(file $GOBIN/protoc-gen-go | grep -o 'aarch64\|ARM64\|ARM' || echo 'ARM64')"
    echo "protoc-gen-go-grpc架构: $(file $GOBIN/protoc-gen-go-grpc | grep -o 'aarch64\|ARM64\|ARM' || echo 'ARM64')"
    
    # 创建部署目录
    echo "🔧 创建部署目录: $TOOLS_DIR/go-arm64-tools/bin"
    mkdir -p "$TOOLS_DIR/go-arm64-tools/bin"
    
    # 验证源文件存在
    echo "🔍 验证源文件："
    ls -la "$GOBIN/protoc-gen-go" || echo "❌ protoc-gen-go源文件不存在"
    ls -la "$GOBIN/protoc-gen-go-grpc" || echo "❌ protoc-gen-go-grpc源文件不存在"
    
    # 复制到部署包
    echo "📦 复制文件到部署包..."
    if cp "$GOBIN/protoc-gen-go" "$TOOLS_DIR/go-arm64-tools/bin/"; then
        echo "✅ protoc-gen-go 复制成功"
    else
        echo "❌ protoc-gen-go 复制失败"
    fi
    
    if cp "$GOBIN/protoc-gen-go-grpc" "$TOOLS_DIR/go-arm64-tools/bin/"; then
        echo "✅ protoc-gen-go-grpc 复制成功"
    else
        echo "❌ protoc-gen-go-grpc 复制失败"
    fi
    
    chmod +x "$TOOLS_DIR/go-arm64-tools/bin/"*
    
    echo "✅ ARM64工具已复制到部署包"
    echo "📋 最终目录结构："
    ls -la "$TOOLS_DIR/go-arm64-tools/bin/"
    
else
    echo "❌ ARM64工具编译失败"
    echo "临时目录内容:"
    ls -la "$TEMP_DIR/go-arm64-tools/bin/" 2>/dev/null || echo "临时bin目录不存在"
    ls -la "$GOBIN/" 2>/dev/null || echo "GOBIN目录不存在"
    cd "$PROJECT_ROOT"
    rm -rf $TEMP_DIR
    exit 1
fi

# 清理临时目录
cd "$PROJECT_ROOT"
rm -rf $TEMP_DIR

# 确保目录存在，然后创建安装脚本（不要覆盖已有的bin目录）
echo "创建安装脚本..."
mkdir -p "$TOOLS_DIR/go-arm64-tools"
cat > "$TOOLS_DIR/go-arm64-tools/install.sh" << 'EOF'
#!/bin/bash
# 在目标ARM64环境安装Go protobuf工具

set -e

echo "🚀 安装Go protobuf工具到目标环境..."

# 检查当前目录
if [ ! -f "bin/protoc-gen-go" ] || [ ! -f "bin/protoc-gen-go-grpc" ]; then
    echo "❌ 未找到ARM64工具文件"
    exit 1
fi

# 设置GOPATH
export GOPATH=$HOME/go
mkdir -p $GOPATH/bin

# 复制工具
cp bin/protoc-gen-go $GOPATH/bin/
cp bin/protoc-gen-go-grpc $GOPATH/bin/

# 设置权限
chmod +x $GOPATH/bin/protoc-gen-go
chmod +x $GOPATH/bin/protoc-gen-go-grpc

# 验证
echo "✅ 工具安装完成!"
echo "protoc-gen-go: $($GOPATH/bin/protoc-gen-go --version 2>/dev/null || echo '已安装')"
echo "protoc-gen-go-grpc: $($GOPATH/bin/protoc-gen-go-grpc --version 2>/dev/null || echo '已安装')"
EOF

chmod +x "$TOOLS_DIR/go-arm64-tools/install.sh"

echo ""
echo "🎉 ARM64 Go工具准备完成!"
echo ""
echo "文件结构:"
echo "$TOOLS_DIR/"
echo "├── go1.23.12.linux-arm64.tar.gz     # Go ARM64版本"
echo "└── go-arm64-tools/"
echo "    ├── bin/"
echo "    │   ├── protoc-gen-go            # ARM64版本"
echo "    │   └── protoc-gen-go-grpc       # ARM64版本"
echo "    └── install.sh                   # 安装脚本"
echo ""
echo "在目标环境使用: cd go-arm64-tools && ./install.sh"
