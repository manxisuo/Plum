#!/bin/bash
# 准备ARM64版本的Go protobuf工具
# 支持在x86和ARM64系统上交叉编译ARM64工具

set -e

echo "🚀 准备ARM64版本的Go protobuf工具..."

# 确保在项目根目录下运行
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

DEPLOY_DIR="plum-offline-deploy"
TOOLS_DIR="$DEPLOY_DIR/tools"

# 检测当前系统架构
SYSTEM_ARCH=$(uname -m)
echo "📋 当前系统架构: $SYSTEM_ARCH"

# 确定需要的Go编译器架构
if [[ "$SYSTEM_ARCH" == "x86_64" ]] || [[ "$SYSTEM_ARCH" == "amd64" ]]; then
    # x86系统：需要使用x86版本的Go来交叉编译ARM64工具
    BUILD_SYSTEM="x86"
    echo "✅ 检测到x86系统，将使用x86 Go交叉编译ARM64工具"
elif [[ "$SYSTEM_ARCH" == "aarch64" ]] || [[ "$SYSTEM_ARCH" == "arm64" ]]; then
    # ARM64系统：可以直接使用ARM64版本的Go
    BUILD_SYSTEM="arm64"
    echo "✅ 检测到ARM64系统，将直接使用ARM64 Go编译"
else
    echo "⚠️  未知系统架构: $SYSTEM_ARCH，尝试使用ARM64 Go"
    BUILD_SYSTEM="unknown"
fi

# 查找可用的Go编译器
GO_COMPILER=""
GO_PATH=""

if [[ "$BUILD_SYSTEM" == "x86" ]]; then
    # x86系统：优先使用系统已安装的Go
    if command -v go &> /dev/null; then
        GO_COMPILER=$(which go)
        GO_PATH="$GO_COMPILER"
        echo "✅ 找到系统Go编译器: $GO_COMPILER"
        echo "   Go版本: $(go version)"
    else
        echo "❌ 系统未安装Go，请先安装Go"
        echo "   安装方法: sudo apt-get install golang-go"
        echo "   或下载: https://golang.google.cn/dl/"
        exit 1
    fi
else
    # ARM64系统：使用ARM64版本的Go
    ARM64_GO_FILE=""
    if [ -f "$TOOLS_DIR/go1.24.3.linux-arm64.tar.gz" ]; then
        ARM64_GO_FILE="$TOOLS_DIR/go1.24.3.linux-arm64.tar.gz"
        # 转换为绝对路径
        if command -v realpath &> /dev/null; then
            ARM64_GO_FILE="$(realpath "$ARM64_GO_FILE")"
        else
            ARM64_GO_FILE="$(cd "$(dirname "$ARM64_GO_FILE")" && pwd)/$(basename "$ARM64_GO_FILE")"
        fi
        echo "✅ 找到ARM64 Go: $ARM64_GO_FILE"
    else
        echo "❌ 未找到go1.24.3.linux-arm64.tar.gz文件"
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
    tar -xzf "$ARM64_GO_FILE"
    
    GO_COMPILER="$TEMP_DIR/go/bin/go"
    GO_PATH="$GO_COMPILER"
    cd "$PROJECT_ROOT"
    
    if [ ! -x "$GO_COMPILER" ]; then
        echo "❌ ARM64 Go解压后不可执行"
        exit 1
    fi
    
    echo "✅ 使用ARM64 Go: $GO_COMPILER"
    echo "   Go版本: $($GO_COMPILER version)"
fi

# 设置交叉编译环境
export GOOS=linux
export GOARCH=arm64
# 注意：不能设置GOBIN，因为Go不允许在设置GOBIN时交叉编译安装
# 我们将使用 go build -o 来指定输出路径
OUTPUT_DIR="/tmp/go-arm64-tools-output/bin"
mkdir -p $OUTPUT_DIR

echo ""
echo "🔧 开始交叉编译ARM64工具..."
echo "   使用Go编译器: $GO_PATH"
echo "   目标架构: $GOOS/$GOARCH"

# 验证Go版本
echo "Go版本: $($GO_PATH version)"

# 创建临时工作目录和临时GOPATH
TEMP_WORK_DIR="/tmp/go-cross-build-work"
TEMP_GOPATH="/tmp/go-cross-build-gopath"
# 注意：不清除旧目录，直接使用（如果存在的话）
mkdir -p $TEMP_WORK_DIR $TEMP_GOPATH/bin 2>/dev/null || true
cd $TEMP_WORK_DIR

# 初始化一个临时Go模块
cat > go.mod << 'EOF'
module temp-build

go 1.24.0
EOF

# 设置临时GOPATH和GOCACHE（但不设置GOBIN）
export GOPATH="$TEMP_GOPATH"
export GOCACHE="$TEMP_WORK_DIR/.cache"

# 安装protoc-gen-go ARM64版本
echo ""
echo "📦 编译protoc-gen-go ARM64版本..."

# 先获取模块到当前工作目录的依赖中
cd "$TEMP_WORK_DIR"
$GO_PATH get google.golang.org/protobuf/cmd/protoc-gen-go@latest || {
        echo "❌ 无法获取 protoc-gen-go 模块"
        cd "$PROJECT_ROOT"
        # 不清理临时目录，让用户手动清理或系统自动清理
        exit 1
}

# 获取模块的实际路径（从模块缓存）
PROTOC_GEN_GO_MODULE=$($GO_PATH list -m -f '{{.Path}}' google.golang.org/protobuf 2>/dev/null | head -1)
PROTOC_GEN_GO_VERSION=$($GO_PATH list -m -f '{{.Version}}' google.golang.org/protobuf 2>/dev/null | head -1)

if [ -z "$PROTOC_GEN_GO_MODULE" ]; then
    PROTOC_GEN_GO_MODULE="google.golang.org/protobuf"
fi

# 尝试从模块缓存构建
GOMODCACHE=$($GO_PATH env GOMODCACHE 2>/dev/null || echo "$HOME/go/pkg/mod")
MODULE_CACHE_PATH="$GOMODCACHE/${PROTOC_GEN_GO_MODULE}@${PROTOC_GEN_GO_VERSION}/cmd/protoc-gen-go"

if [ -d "$MODULE_CACHE_PATH" ]; then
    echo "   从模块缓存构建: $MODULE_CACHE_PATH"
    cd "$MODULE_CACHE_PATH"
    $GO_PATH build -o "$OUTPUT_DIR/protoc-gen-go" . || {
        echo "❌ 从模块缓存构建失败"
        cd "$PROJECT_ROOT"
        # 不清理临时目录
        exit 1
    }
else
    # 备用方法：在临时目录中创建软链接或直接构建
    echo "   模块缓存路径不存在，尝试直接构建..."
    cd "$TEMP_WORK_DIR"
    # 创建一个临时main.go来引用这个包
    mkdir -p cmd/protoc-gen-go
    cd cmd/protoc-gen-go
    # 尝试使用 go build 构建（Go 1.18+ 应该支持）
    $GO_PATH build -o "$OUTPUT_DIR/protoc-gen-go" google.golang.org/protobuf/cmd/protoc-gen-go@latest 2>&1 || {
        echo "❌ protoc-gen-go编译失败"
        cd "$PROJECT_ROOT"
        # 不清理临时目录
        exit 1
    }
fi

if [ -f "$OUTPUT_DIR/protoc-gen-go" ]; then
    chmod +x "$OUTPUT_DIR/protoc-gen-go"
    echo "✅ protoc-gen-go 编译成功"
else
    echo "❌ protoc-gen-go 编译失败：输出文件不存在"
    cd "$PROJECT_ROOT"
    # 不清理临时目录
    exit 1
fi

# 安装protoc-gen-go-grpc ARM64版本
echo ""
echo "📦 编译protoc-gen-go-grpc ARM64版本..."

# 先获取模块
cd "$TEMP_WORK_DIR"
$GO_PATH get google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest || {
    echo "❌ 无法获取 protoc-gen-go-grpc 模块"
    cd "$PROJECT_ROOT"
    # 不清理临时目录
    exit 1
}

# 获取protoc-gen-go-grpc模块的实际路径（注意：这是一个独立模块，不是grpc的一部分）
PROTOC_GO_GRPC_MODULE=$($GO_PATH list -m -f '{{.Path}}' google.golang.org/grpc/cmd/protoc-gen-go-grpc 2>/dev/null | head -1)
PROTOC_GO_GRPC_VERSION=$($GO_PATH list -m -f '{{.Version}}' google.golang.org/grpc/cmd/protoc-gen-go-grpc 2>/dev/null | head -1)

if [ -z "$PROTOC_GO_GRPC_MODULE" ]; then
    PROTOC_GO_GRPC_MODULE="google.golang.org/grpc/cmd/protoc-gen-go-grpc"
fi

# 尝试从模块缓存构建
GOMODCACHE=$($GO_PATH env GOMODCACHE 2>/dev/null || echo "$HOME/go/pkg/mod")
if [ -z "$GOMODCACHE" ] || [ "$GOMODCACHE" = "$HOME/go/pkg/mod" ]; then
    # 使用临时GOPATH的模块缓存
    GOMODCACHE="$TEMP_GOPATH/pkg/mod"
fi

echo "   模块缓存目录: $GOMODCACHE"
echo "   protoc-gen-go-grpc模块: $PROTOC_GO_GRPC_MODULE"
echo "   protoc-gen-go-grpc版本: $PROTOC_GO_GRPC_VERSION"

# protoc-gen-go-grpc是独立模块，路径就是模块本身
MODULE_CACHE_PATH="$GOMODCACHE/${PROTOC_GO_GRPC_MODULE}@${PROTOC_GO_GRPC_VERSION}"
echo "   尝试的模块路径: $MODULE_CACHE_PATH"

if [ -d "$MODULE_CACHE_PATH" ]; then
    echo "   从模块缓存构建: $MODULE_CACHE_PATH"
    cd "$MODULE_CACHE_PATH"
    $GO_PATH build -o "$OUTPUT_DIR/protoc-gen-go-grpc" . || {
        echo "❌ 从模块缓存构建失败"
        echo "   调试信息："
        echo "   - 当前目录: $(pwd)"
        echo "   - Go版本: $($GO_PATH version)"
        echo "   - 目标架构: $GOOS/$GOARCH"
        cd "$PROJECT_ROOT"
        # 不清理临时目录
        exit 1
    }
else
    # 备用方法：使用go list获取实际目录
    echo "   模块缓存路径不存在，使用go list查找实际路径..."
    cd "$TEMP_WORK_DIR"
    
    # 使用go list获取实际解压的模块目录
    GRPC_CMD_DIR=$($GO_PATH list -m -f '{{.Dir}}' "${PROTOC_GO_GRPC_MODULE}@${PROTOC_GO_GRPC_VERSION}" 2>/dev/null)
    
    if [ -n "$GRPC_CMD_DIR" ] && [ -d "$GRPC_CMD_DIR" ]; then
        echo "   找到模块目录: $GRPC_CMD_DIR"
        cd "$GRPC_CMD_DIR"
        $GO_PATH build -o "$OUTPUT_DIR/protoc-gen-go-grpc" . || {
            echo "❌ 从实际模块目录构建失败"
            cd "$PROJECT_ROOT"
            exit 1
        }
    else
        # 最后的备用方法：直接从模块路径构建（需要Go 1.18+）
        echo "   使用go list查找失败，尝试从下载缓存查找..."
        
        # 查找所有可能包含protoc-gen-go-grpc的目录
        POSSIBLE_PATHS=(
            "$GOMODCACHE/${PROTOC_GO_GRPC_MODULE}@${PROTOC_GO_GRPC_VERSION}"
            "$TEMP_GOPATH/pkg/mod/${PROTOC_GO_GRPC_MODULE}@${PROTOC_GO_GRPC_VERSION}"
            "$HOME/go/pkg/mod/${PROTOC_GO_GRPC_MODULE}@${PROTOC_GO_GRPC_VERSION}"
        )
        
        FOUND_PATH=""
        for path in "${POSSIBLE_PATHS[@]}"; do
            if [ -d "$path" ] && [ -f "$path/main.go" ]; then
                FOUND_PATH="$path"
                break
            fi
        done
        
        if [ -n "$FOUND_PATH" ]; then
            echo "   找到备用路径: $FOUND_PATH"
            cd "$FOUND_PATH"
            $GO_PATH build -o "$OUTPUT_DIR/protoc-gen-go-grpc" . || {
                echo "❌ 从备用路径构建失败"
                cd "$PROJECT_ROOT"
                exit 1
            }
        else
            echo "❌ 无法找到protoc-gen-go-grpc源码路径"
            echo "   已尝试的路径："
            for path in "${POSSIBLE_PATHS[@]}"; do
                echo "     - $path"
            done
            cd "$PROJECT_ROOT"
            exit 1
        fi
    fi
fi

if [ -f "$OUTPUT_DIR/protoc-gen-go-grpc" ]; then
    chmod +x "$OUTPUT_DIR/protoc-gen-go-grpc"
    echo "✅ protoc-gen-go-grpc 编译成功"
else
    echo "❌ protoc-gen-go-grpc 编译失败：输出文件不存在"
    cd "$PROJECT_ROOT"
    # 不清理临时目录，让用户手动清理或系统自动清理
    exit 1
fi

cd "$PROJECT_ROOT"

# 先验证编译结果是否存在（如果存在，说明编译成功）
echo ""
echo "🔍 验证编译结果..."
if [ -f "$OUTPUT_DIR/protoc-gen-go" ] && [ -f "$OUTPUT_DIR/protoc-gen-go-grpc" ]; then
    echo "✅ 编译成功，两个工具文件都已生成"
elif [ -f "$OUTPUT_DIR/protoc-gen-go" ]; then
    echo "⚠️  警告: protoc-gen-go 已生成，但 protoc-gen-go-grpc 缺失"
    echo "   检查编译过程中的错误信息..."
    exit 1
elif [ -f "$OUTPUT_DIR/protoc-gen-go-grpc" ]; then
    echo "⚠️  警告: protoc-gen-go-grpc 已生成，但 protoc-gen-go 缺失"
    echo "   检查编译过程中的错误信息..."
    exit 1
else
    echo "❌ 编译失败：两个工具文件都不存在"
    exit 1
fi

# 跳过清理临时目录
# 注意：这些临时文件在 /tmp 目录下，系统会在重启时自动清理
# 删除这些文件是可选的，不是必须的，所以直接跳过删除操作
# 如果确实需要清理，可以手动执行：
#   sudo rm -rf /tmp/go-cross-build-gopath /tmp/go-cross-build-work

# 显示编译结果信息
echo ""
echo "📋 编译结果详情:"
echo "检查文件: $OUTPUT_DIR/protoc-gen-go"
ls -lh "$OUTPUT_DIR/protoc-gen-go" 2>/dev/null || echo "⚠️  protoc-gen-go文件不存在"
echo "检查文件: $OUTPUT_DIR/protoc-gen-go-grpc"
ls -lh "$OUTPUT_DIR/protoc-gen-go-grpc" 2>/dev/null || echo "⚠️  protoc-gen-go-grpc文件不存在"

if [ -f "$OUTPUT_DIR/protoc-gen-go" ] && [ -f "$OUTPUT_DIR/protoc-gen-go-grpc" ]; then
    echo "✅ ARM64工具编译成功!"
    
    # 检查架构（如果file命令可用）
    if command -v file &> /dev/null; then
        echo ""
        echo "📋 验证工具架构:"
        PROTOC_GO_ARCH=$(file "$OUTPUT_DIR/protoc-gen-go" 2>/dev/null | grep -oE '(aarch64|ARM64|ARM|x86-64|x86_64)' || echo "未知")
        PROTOC_GO_GRPC_ARCH=$(file "$OUTPUT_DIR/protoc-gen-go-grpc" 2>/dev/null | grep -oE '(aarch64|ARM64|ARM|x86-64|x86_64)' || echo "未知")
        echo "  protoc-gen-go架构: $PROTOC_GO_ARCH"
        echo "  protoc-gen-go-grpc架构: $PROTOC_GO_GRPC_ARCH"
        
        if [[ ! "$PROTOC_GO_ARCH" =~ (aarch64|ARM64|ARM) ]]; then
            echo "⚠️  警告: protoc-gen-go架构可能不正确（期望ARM64）"
        fi
        if [[ ! "$PROTOC_GO_GRPC_ARCH" =~ (aarch64|ARM64|ARM) ]]; then
            echo "⚠️  警告: protoc-gen-go-grpc架构可能不正确（期望ARM64）"
        fi
    fi
    
    # 创建部署目录
    echo "🔧 创建部署目录: $TOOLS_DIR/go-arm64-tools/bin"
    mkdir -p "$TOOLS_DIR/go-arm64-tools/bin"
    
    # 验证源文件存在
    echo "🔍 验证源文件："
    ls -la "$OUTPUT_DIR/protoc-gen-go" || echo "❌ protoc-gen-go源文件不存在"
    ls -la "$OUTPUT_DIR/protoc-gen-go-grpc" || echo "❌ protoc-gen-go-grpc源文件不存在"
    
    # 复制到部署包
    echo "📦 复制文件到部署包..."
    if cp "$OUTPUT_DIR/protoc-gen-go" "$TOOLS_DIR/go-arm64-tools/bin/"; then
        echo "✅ protoc-gen-go 复制成功"
    else
        echo "❌ protoc-gen-go 复制失败"
    fi
    
    if cp "$OUTPUT_DIR/protoc-gen-go-grpc" "$TOOLS_DIR/go-arm64-tools/bin/"; then
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
    ls -la "$OUTPUT_DIR/" 2>/dev/null || echo "输出目录不存在"
    cd "$PROJECT_ROOT"
    exit 1
fi

# 注意：不清理临时目录
# 这些临时文件在 /tmp 下，系统会在重启时自动清理
# 如果需要手动清理，可以执行：
#   sudo rm -rf /tmp/go-arm64-build /tmp/go-arm64-tools-output
cd "$PROJECT_ROOT"

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
if [[ "$BUILD_SYSTEM" != "arm64" ]]; then
    echo "├── go1.24.3.linux-arm64.tar.gz     # Go ARM64版本（目标环境用）"
else
    echo "├── go1.24.3.linux-arm64.tar.gz     # Go ARM64版本"
fi
echo "└── go-arm64-tools/"
echo "    ├── bin/"
echo "    │   ├── protoc-gen-go            # ARM64版本"
echo "    │   └── protoc-gen-go-grpc       # ARM64版本"
echo "    └── install.sh                   # 安装脚本"
echo ""
echo "在目标环境使用: cd go-arm64-tools && ./install.sh"
