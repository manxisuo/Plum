#!/bin/bash
# 银河麒麟V10 ARM64环境依赖安装脚本

set -e

echo "🚀 开始安装依赖到银河麒麟V10 ARM64环境..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# 检测系统
if [ "$(uname -m)" != "aarch64" ]; then
    echo "❌ 当前系统不是ARM64架构，请确认运行环境"
    exit 1
fi

# 1. 安装Go
echo "📦 安装Go 1.24.3..."
if ! command -v go &> /dev/null; then
    cd ../tools
    if [ -f "go1.24.3.linux-arm64.tar.gz" ]; then
        sudo tar -C /usr/local -xzf go1.24.3.linux-arm64.tar.gz
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        export PATH=$PATH:/usr/local/go/bin
        echo "✅ Go安装完成"
    else
        echo "❌ 未找到Go ARM64安装包，请检查tools目录"
        exit 1
    fi
    cd ../scripts
else
    echo "✅ Go已安装: $(go version)"
fi

# 2. 安装Node.js
echo "📦 安装Node.js 18.x..."
if ! command -v node &> /dev/null; then
    cd ../tools
    if [ -f "node-v18.20.4-linux-arm64.tar.xz" ]; then
        tar -xf node-v18.20.4-linux-arm64.tar.xz
        sudo mv node-v18.20.4-linux-arm64 /usr/local/nodejs18
        sudo ln -sf /usr/local/nodejs18/bin/node /usr/local/bin/node
        sudo ln -sf /usr/local/nodejs18/bin/npm /usr/local/bin/npm
        sudo ln -sf /usr/local/nodejs18/bin/npx /usr/local/bin/npx
        echo "✅ Node.js安装完成"
    else
        echo "❌ 未找到Node.js ARM64安装包，请检查tools目录"
        exit 1
    fi
    cd ../scripts
else
    echo "✅ Node.js已安装: $(node --version)"
fi

# 3. 安装系统依赖
echo "📦 安装系统依赖..."
# 银河麒麟V10基于Ubuntu/Debian，使用apt包管理器
if command -v apt &> /dev/null; then
    echo "🔌 离线模式：检查已安装的构建工具..."
    
    # 检查核心工具是否已安装
    MISSING_TOOLS=""
    
    if ! command -v curl &> /dev/null; then
        MISSING_TOOLS="$MISSING_TOOLS curl"
        echo "❌ curl: 未找到"
    else
        echo "✅ curl已安装: $(curl --version | head -1)"
    fi
    
    if ! command -v git &> /dev/null; then
        MISSING_TOOLS="$MISSING_TOOLS git"
        echo "❌ git: 未找到"
    else
        echo "✅ git已安装: $(git --version)"
    fi
    
    if ! command -v make &> /dev/null; then
        MISSING_TOOLS="$MISSING_TOOLS make"
        echo "❌ make: 未找到"
    else
        echo "✅ make已安装: $(make --version | head -1)"
    fi
    
    if ! command -v cmake &> /dev/null; then
        MISSING_TOOLS="$MISSING_TOOLS cmake"
        echo "❌ cmake: 未找到"
    else
        echo "✅ cmake已安装: $(cmake --version | head -1)"
    fi
    
    if ! command -v pkg-config &> /dev/null; then
        MISSING_TOOLS="$MISSING_TOOLS pkg-config"
        echo "❌ pkg-config: 未找到"
    else
        echo "✅ pkg-config已安装: $(pkg-config --version)"
    fi
    
    # 检查C++ SDK依赖 (plumclient现在使用httplib，不再需要libcurl)
    echo "✅ C++ SDK依赖: plumclient使用httplib，无需额外依赖"
    
    if ! pkg-config --exists pthread; then
        MISSING_TOOLS="$MISSING_TOOLS libpthread-stubs0-dev"
        echo "❌ pthread开发包: 未找到"
    else
        echo "✅ pthread开发包已安装"
    fi
    
    # 检查protobuf工具（先检查，稍后安装）
    if ! command -v protoc &> /dev/null; then
        MISSING_TOOLS="$MISSING_TOOLS protoc"
        echo "❌ protoc: 未找到（将在后续安装gRPC依赖时安装）"
    else
        echo "✅ protoc已安装: $(protoc --version)"
    fi
    
    # 报告结果
    if [ -n "$MISSING_TOOLS" ]; then
        echo "⚠️  以下工具缺失: $MISSING_TOOLS"
        echo "   尝试安装缺失的依赖..."
        
        # 离线模式：跳过网络安装，仅检查已安装的工具
        echo "⚠️  离线模式：以下工具缺失但无法自动安装："
        echo "   $MISSING_TOOLS"
        echo ""
        echo "💡 建议解决方案："
        echo "1. 在联网环境中预先安装这些依赖包"
        echo "2. 或者手动下载对应的.deb包并安装"
        echo "3. 对于libpthread-stubs0-dev，通常不是必需的，可以忽略"
        echo ""
        echo "🔧 如果必须安装，可以尝试："
        echo "   sudo dpkg -i <package.deb>"
        echo "   sudo apt-get install -f  # 修复依赖（需要网络）"
    fi
    echo "✅ 系统依赖检查完成"
else
    echo "⚠️  未检测到apt包管理器，请手动安装必要依赖"
fi

# 4. 安装Go protobuf工具
echo "📦 安装Go protobuf工具..."
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
mkdir -p $GOPATH/bin

# 设置Go代理和工具链配置（防止离线环境下载工具链）
go env -w GO111MODULE=on
go env -w GOPROXY=direct
go env -w GOTOOLCHAIN=local
go env -w GOSUMDB=off

echo "🔧 Go环境配置："
echo "   GO111MODULE=$(go env GO111MODULE)"
echo "   GOPROXY=$(go env GOPROXY)"  
echo "   GOTOOLCHAIN=$(go env GOTOOLCHAIN)"
echo "   GOSUMDB=$(go env GOSUMDB)"

# 优先使用预编译的ARM64工具（离线模式）
if [ -d "../tools/go-arm64-tools/bin" ]; then
    echo "🔌 离线模式：使用预编译的ARM64 protobuf工具..."
    cp ../tools/go-arm64-tools/bin/* $GOPATH/bin/
    chmod +x $GOPATH/bin/*
    echo "✅ 预编译工具安装完成"
else
    echo "🔌 离线模式：预编译工具未找到"
    echo "   检查现有protobuf工具..."
    
    # 检查是否已经有protobuf工具
    if [ -f "$GOPATH/bin/protoc-gen-go" ] && [ -f "$GOPATH/bin/protoc-gen-go-grpc" ]; then
        echo "✅ protobuf工具已存在"
    else
        echo "⚠️  缺少protobuf工具，但无法在线安装"
        echo "   建议在WSL2环境中重新运行prepare-offline-deploy.sh"
    fi
fi

# 5. 修复UI依赖（解决Rollup ARM64可选依赖问题）
echo "🔧 检查和修复UI依赖..."
if [ -d "../source/Plum/ui" ]; then
    cd ../source/Plum/ui
    
    # 离线模式：检查UI依赖状态
    if [ ! -d "node_modules/@rollup/rollup-linux-arm64-gnu" ] && [ -d "node_modules" ]; then
        echo "⚠️  检测到Rollup ARM64依赖缺失"
        echo "🔌 离线模式：无法自动修复，需要手动处理"
        echo "   如果UI构建失败，可以尝试："
        echo "   1. 在WSL2环境中重新运行prepare-offline-deploy.sh"
        echo "   2. 或者手动处理UI依赖问题"
    else
        echo "✅ UI依赖检查正常"
    fi
    
    # 显示关键UI依赖状态
    echo "📋 UI依赖状态："
    if [ -d "node_modules" ]; then
        echo "✅ node_modules目录存在"
        if [ -d "node_modules/@rollup" ]; then
            echo "✅ rollup包存在"
            if [ -d "node_modules/@rollup/rollup-linux-arm64-gnu" ]; then
                echo "✅ rollup ARM64原生依赖存在"
            else
                echo "❌ rollup ARM64原生依赖缺失"
            fi
        else
            echo "❌ rollup包缺失"
        fi
    else
        echo "❌ node_modules目录不存在"
    fi
    
    cd ../../../scripts
fi

# 5.5. 安装gRPC开发包（如果存在）
echo "🔧 检查和安装gRPC开发包..."
GRPC_DEPS_DIR="../tools/grpc-deps"
if [ -d "$GRPC_DEPS_DIR" ] && ls "$GRPC_DEPS_DIR"/*.deb 1> /dev/null 2>&1; then
    echo "📦 发现gRPC依赖包，开始安装..."
    cd "$GRPC_DEPS_DIR"
    
    # 检查关键包
    key_packages=("libgrpc++-dev" "libgrpc-dev" "libprotobuf-dev" "protobuf-compiler")
    for pkg in "${key_packages[@]}"; do
        if ls ${pkg}_*.deb 1> /dev/null 2>&1; then
            echo "✅ 找到 $pkg 包"
        fi
    done
    
    # 安装所有包
    echo "📥 安装gRPC开发包..."
    sudo dpkg -i *.deb 2>/dev/null || {
        echo "⚠️  部分包安装失败，离线模式无法自动修复依赖"
        echo "💡 如果安装失败，请检查："
        echo "1. 包是否已损坏"
        echo "2. 依赖关系是否正确"
        echo "3. 系统是否缺少基础库"
        echo ""
        echo "🔧 手动修复建议："
        echo "   sudo dpkg --configure -a  # 配置未完成的包"
        echo "   sudo apt-get install -f  # 修复依赖（需要网络）"
    }
    
    # 验证安装
    if pkg-config --exists grpc++; then
        echo "✅ gRPC开发包安装成功"
    else
        echo "⚠️  gRPC开发包可能未完全安装"
    fi
    
    # 验证protoc是否可用
    if command -v protoc &> /dev/null; then
        echo "✅ protoc编译器安装成功: $(protoc --version)"
    else
        echo "⚠️  protoc编译器可能未正确安装"
        echo "   尝试刷新PATH..."
        export PATH="/usr/bin:/usr/local/bin:$PATH"
        hash -r
        if command -v protoc &> /dev/null; then
            echo "✅ protoc现在可用: $(protoc --version)"
        else
            echo "❌ protoc仍然不可用，请检查protobuf-compiler包是否正确安装"
        fi
    fi
    
    cd - > /dev/null
else
    echo "📋 未找到gRPC依赖包，请手动下载并安装以下ARM64包："
    echo "   - libgrpc++-dev_*_arm64.deb"
    echo "   - libgrpc-dev_*_arm64.deb"  
    echo "   - libprotobuf-dev_*_arm64.deb"
    echo "   - protobuf-compiler_*_arm64.deb"
    echo ""
    echo "安装命令：sudo dpkg -i *.deb && sudo apt-get install -f"
    
    # 检查protoc是否在其他位置可用
    echo "🔍 检查系统是否有protoc..."
    for path in /usr/bin/protoc /usr/local/bin/protoc; do
        if [ -x "$path" ]; then
            echo "✅ 找到protoc: $path"
            export PATH="$(dirname $path):$PATH"
            break
        fi
    done
fi

# 6. 安装esbuild ARM64依赖
echo "🔧 安装esbuild ARM64依赖..."
if [ -f "$ROOT_DIR/tools/esbuild-linux-arm64-0.21.5.tgz" ]; then
    echo "📦 发现esbuild ARM64包，开始安装..."
    if [ -f "$ROOT_DIR/scripts/install-esbuild-arm64-0.21.5.sh" ]; then
        ( cd "$ROOT_DIR/source/Plum" && bash "$ROOT_DIR/scripts/install-esbuild-arm64-0.21.5.sh" ) || {
            echo "⚠️  esbuild ARM64安装失败"
            echo "💡 请检查esbuild包是否完整"
        }
    else
        echo "⚠️  未找到install-esbuild-arm64-0.21.5.sh脚本"
        echo "💡 请确保脚本已正确复制到部署包中"
    fi
else
    echo "📋 未找到esbuild ARM64包，跳过安装"
    echo "💡 如需安装，请将esbuild-linux-arm64-0.21.5.tgz放到tools目录"
fi

# 7. 安装rollup ARM64依赖
echo "🔧 安装rollup ARM64依赖..."
if [ -f "$ROOT_DIR/tools/rollup-linux-arm64-gnu-4.52.5.tgz" ]; then
    echo "📦 发现rollup ARM64包，开始安装..."
    if [ -f "$ROOT_DIR/scripts/fix-rollup-arm64.sh" ]; then
        ( cd "$ROOT_DIR/source/Plum" && bash "$ROOT_DIR/scripts/fix-rollup-arm64.sh" ) || {
            echo "⚠️  rollup ARM64安装失败"
            echo "💡 请检查rollup包是否完整"
        }
    else
        echo "⚠️  未找到fix-rollup-arm64.sh脚本"
        echo "💡 请确保脚本已正确复制到部署包中"
    fi
else
    echo "📋 未找到rollup ARM64包，跳过安装"
    echo "💡 如需安装，请将rollup-linux-arm64-gnu-4.52.5.tgz放到tools目录"
fi

# 8. 验证安装
echo "🔍 验证安装结果..."
echo "Go版本: $(go version)"
echo "Node.js版本: $(node --version)"
echo "npm版本: $(npm --version)"
if command -v protoc &> /dev/null; then
    echo "protoc版本: $(protoc --version)"
fi

# 验证核心构建工具
echo ""
echo "🔧 验证核心构建工具..."
BUILD_TOOLS_OK=true

for tool in gcc g++ make cmake; do
    if command -v $tool &> /dev/null; then
        echo "✅ $tool: $($tool --version | head -1)"
    else
        echo "❌ $tool: 未找到"
        BUILD_TOOLS_OK=false
    fi
done

if [ "$BUILD_TOOLS_OK" = true ]; then
    echo "✅ 核心构建工具检查通过"
    echo ""
    echo "🎉 离线环境依赖检查完成！"
    echo ""
    echo "🔌 离线模式构建说明："
    echo "   1. cd ../source/Plum"
    echo "   2. export GOTOOLCHAIN=local CGO_ENABLED=0    # 设置离线Go环境"
    echo "   3. make controller    # 构建Controller"
    echo "   4. make agent         # 构建Agent"
    echo "   5. make ui-build      # 构建UI（如果依赖正常）"
    echo ""
    echo "   如果遇到UI构建问题，运行相关修复脚本"
else
    echo "⚠️  部分构建工具缺失"
    echo "🔌 离线模式：无法自动安装缺失工具"
    echo "   建议：确保系统已预装build-essential等基础工具"
fi
