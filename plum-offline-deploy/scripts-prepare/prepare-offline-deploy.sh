#!/bin/bash
# 离线部署准备脚本 - 用于WSL2 x86环境准备ARM64部署包
# 使用方法：在项目根目录运行 ./plum-offline-deploy/scripts-prepare/prepare-offline-deploy.sh

set -e

# 确保在项目根目录运行
if [ ! -f "Makefile" ] || [ ! -d "controller" ] || [ ! -d "agent-go" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    echo "   当前目录: $(pwd)"
    echo "   期望找到: Makefile, controller/, agent-go/"
    exit 1
fi

# 检查必要的构建工具
echo "🔍 检查构建环境..."

# 检查Go
if ! command -v go &> /dev/null; then
    echo "❌ Go命令未找到，请确保："
    echo "   1. Go已正确安装"
    echo "   2. 不要使用sudo运行此脚本"
    echo "   3. PATH环境变量包含Go二进制目录"
    echo ""
    echo "   尝试运行: which go 或 go version"
    exit 1
else
    echo "✅ Go版本: $(go version)"
fi

# 检查npm
if ! command -v npm &> /dev/null; then
    echo "❌ npm命令未找到，请确保Node.js已正确安装"
    exit 1
else
    echo "✅ npm版本: $(npm --version)"
fi

# 检查是否使用sudo（通常不需要）
if [ "$EUID" -eq 0 ]; then
    echo "⚠️  检测到以root权限运行，这可能导致环境变量问题"
    echo "   建议以普通用户权限运行: $0"
    echo ""
fi

echo "🚀 开始准备离线部署包..."

# 创建目录结构
DEPLOY_DIR="plum-offline-deploy"
mkdir -p $DEPLOY_DIR/{source,tools,scripts,docs,scripts-prepare,go-vendor-backup}

echo "📝 注意：此脚本会清理并重新生成以下内容："
echo "   - 源代码目录 (source/Plum/)"
echo "   - Go依赖 (vendor/)"
echo "   - Node.js依赖 (node_modules/)"
echo "   - ARM64构建工具会被保留（如果已存在）"
echo ""

# 1. 清理并复制源代码（排除部署目录本身）
echo "📦 清理旧的源码并复制新版本..."

# 先清理旧的源码目录，避免文件混合
if [ -d "$DEPLOY_DIR/source/Plum" ]; then
    echo "清理旧的源码目录..."
    # 尝试先修改权限再删除
    chmod -R u+w "$DEPLOY_DIR/source/Plum" 2>/dev/null || true
    rm -rf "$DEPLOY_DIR/source/Plum" 2>/dev/null || {
        echo "⚠️  无法删除旧目录，可能是权限问题。"
        echo "   这可能是因为之前使用sudo运行过脚本。"
        echo ""
        # 检查是否在交互式环境中运行
        if [ -t 0 ] && [ -t 1 ]; then
            echo "请选择解决方案："
            echo "1) 使用sudo清理 (推荐)"
            echo "2) 手动清理后继续"
            echo "3) 退出脚本"
            echo ""
            read -p "请输入选择 (1/2/3): " choice
        else
            echo "非交互式环境，尝试使用sudo清理..."
            choice="1"
        fi
        
        case $choice in
            1)
                echo "使用sudo清理旧目录..."
                sudo rm -rf "$DEPLOY_DIR/source/Plum" || {
                    echo "❌ sudo清理也失败了"
                    exit 1
                }
                echo "✅ 清理完成"
                ;;
            2)
                echo "请手动清理后按回车继续..."
                echo "运行: rm -rf $DEPLOY_DIR/source/Plum"
                if [ -t 0 ] && [ -t 1 ]; then
                    read -p "按回车键继续..."
                else
                    echo "等待5秒后继续..."
                    sleep 5
                fi
                ;;
            3)
                echo "退出脚本"
                exit 1
                ;;
            *)
                echo "无效选择，退出脚本"
                exit 1
                ;;
        esac
    }
fi

mkdir -p $DEPLOY_DIR/source/Plum

# 使用rsync复制，排除构建文件和可能有权限问题的文件
echo "复制源代码（排除构建文件）..."

# 检查是否有rsync命令
if command -v rsync &> /dev/null; then
    echo "使用rsync复制..."
    # 复制目录
    for dir in controller agent-go ui proto sdk examples docs tools; do
        if [ -e "$dir" ]; then
            echo "复制: $dir"
            rsync -av --exclude='build/' \
                      --exclude='cmake-build-*/' \
                      --exclude='*.log' \
                      --exclude='*.db*' \
                      --exclude='node_modules/' \
                      --exclude='vendor/' \
                      "$dir/" "$DEPLOY_DIR/source/Plum/$dir/"
        fi
    done
    
    # 复制单个文件
    for file in Makefile README.md .gitignore; do
        if [ -e "$file" ]; then
            echo "复制: $file"
            cp "$file" "$DEPLOY_DIR/source/Plum/"
        fi
    done
else
    echo "rsync不可用，使用cp并手动清理..."
    for dir in controller agent-go ui proto sdk examples docs tools; do
        if [ -e "$dir" ]; then
            echo "复制: $dir"
            cp -r "$dir" $DEPLOY_DIR/source/Plum/
        fi
    done
    
    # 复制单个文件
    for file in Makefile README.md .gitignore; do
        if [ -e "$file" ]; then
            echo "复制: $file"
            cp "$file" $DEPLOY_DIR/source/Plum/
        fi
    done
fi

# 清理可能有权限问题的构建文件
echo "清理构建文件..."
find "$DEPLOY_DIR/source/Plum" -type d -name "build" -exec rm -rf {} + 2>/dev/null || true
find "$DEPLOY_DIR/source/Plum" -type d -name "cmake-build-*" -exec rm -rf {} + 2>/dev/null || true
rm -rf "$DEPLOY_DIR/source/Plum"/*.log 2>/dev/null || true
rm -rf "$DEPLOY_DIR/source/Plum"/*.db* 2>/dev/null || true

# 2. 准备Go依赖（每次重新生成以确保最新）
echo "📦 生成最新的Go依赖..."

# Controller依赖 - 每次重新生成以确保最新
echo "生成Controller Go依赖..."
cd controller && go mod download && go mod vendor && cd ..
if [ -d "controller/vendor" ]; then
    cp -r controller/vendor $DEPLOY_DIR/source/Plum/controller/
    cp -r controller/vendor $DEPLOY_DIR/go-vendor-backup/controller-vendor
    echo "✅ Controller依赖生成成功"
else
    echo "❌ Controller vendor目录生成失败"
    exit 1
fi

# Agent依赖 - 每次重新生成以确保最新
echo "生成Agent Go依赖..."
cd agent-go && go mod download && go mod vendor && cd ..
if [ -d "agent-go/vendor" ]; then
    cp -r agent-go/vendor $DEPLOY_DIR/source/Plum/agent-go/
    cp -r agent-go/vendor $DEPLOY_DIR/go-vendor-backup/agent-vendor
    echo "✅ Agent依赖生成成功"
else
    echo "❌ Agent vendor目录生成失败"
    exit 1
fi

# 清理根目录的vendor（根据gitignore规则）
rm -rf controller/vendor agent-go/vendor

echo "✅ Go依赖已更新到最新版本"

# 3. 准备Node.js依赖（每次都重新生成以确保最新）
echo "📦 生成最新的Node.js依赖..."

# 先清理旧的node_modules
echo "更新UI依赖..."
cd ui

# 清理可能的问题文件
echo "清理可能的问题文件..."
rm -f package-lock.json
rm -rf node_modules

# 安装依赖，确保可选依赖也被安装
echo "安装Node.js依赖（包括ARM64可选依赖）..."
npm install --include=optional

# 验证关键依赖是否安装
echo "验证Rollup ARM64依赖..."
if [ -d "node_modules/@rollup/rollup-linux-arm64-gnu" ]; then
    echo "✅ Rollup ARM64 原生依赖已安装"
else
    echo "⚠️  Rollup ARM64 原生依赖未找到，尝试手动安装..."
    npm install @rollup/rollup-linux-arm64-gnu --save-optional || echo "无法安装ARM64依赖"
fi

cd ..

# 复制最新的依赖到部署包
echo "复制最新UI依赖到部署包..."
if [ -d "ui/node_modules" ]; then
    # 先删除旧的node_modules
    rm -rf $DEPLOY_DIR/source/Plum/ui/node_modules
    cp -r ui/node_modules $DEPLOY_DIR/source/Plum/ui/
    
    # 验证ARM64依赖是否被正确复制
    echo "验证ARM64依赖复制状态..."
    if [ -d "$DEPLOY_DIR/source/Plum/ui/node_modules/@rollup/rollup-linux-arm64-gnu" ]; then
        echo "✅ Rollup ARM64 依赖已正确复制到部署包"
    else
        echo "⚠️  Rollup ARM64 依赖未复制到部署包"
        echo "   检查源目录中的依赖状态..."
        if [ -d "ui/node_modules/@rollup/rollup-linux-arm64-gnu" ]; then
            echo "   源目录中有ARM64依赖，但复制失败"
        else
            echo "   源目录中也缺少ARM64依赖，需要在WSL2中重新安装"
        fi
    fi
    
    echo "✅ UI依赖已更新到最新版本"
else
    echo "❌ UI依赖生成失败"
    exit 1
fi

# 4. 下载ARM64版本的构建工具
echo "📦 下载ARM64构建工具..."

# 创建下载工具脚本
    cat > $DEPLOY_DIR/tools/download-tools.sh << 'EOF'
#!/bin/bash
echo "下载ARM64版本的构建工具..."

# Go 1.23.12 ARM64版本（与prepare-arm64-go-tools.sh保持一致）
if [ ! -f "go1.23.12.linux-arm64.tar.gz" ]; then
    echo "下载Go 1.23.12 ARM64版本..."
    wget https://golang.google.cn/dl/go1.23.12.linux-arm64.tar.gz || {
        echo "❌ 下载Go失败，尝试备用地址..."
        wget https://go.dev/dl/go1.23.12.linux-arm64.tar.gz || {
            echo "❌ 备用地址也失败，请检查网络连接或手动下载"
            exit 1
        }
    }
    echo "✅ Go下载完成"
else
    echo "✅ Go文件已存在: go1.23.12.linux-arm64.tar.gz"
fi

# Node.js 18.x ARM64版本  
if [ ! -f "node-v18.20.4-linux-arm64.tar.xz" ]; then
    echo "下载Node.js 18.20.4 ARM64版本..."
    wget https://nodejs.org/dist/v18.20.4/node-v18.20.4-linux-arm64.tar.xz || {
        echo "❌ 下载Node.js失败，请检查网络连接或手动下载"
        exit 1
    }
    echo "✅ Node.js下载完成"
else
    echo "✅ Node.js文件已存在: node-v18.20.4-linux-arm64.tar.xz"
fi

echo "✅ 所有工具下载完成"
EOF

chmod +x $DEPLOY_DIR/tools/download-tools.sh

# 5. 自动下载工具和准备ARM64工具
echo "🔧 准备ARM64构建工具..."

# 先下载ARM64版本的工具
cd $DEPLOY_DIR/tools
echo "📥 下载ARM64构建工具..."
if [ -f "./download-tools.sh" ]; then
    bash ./download-tools.sh || {
        echo "❌ 下载工具失败，无法继续准备ARM64工具"
        exit 1
    }
else
    echo "❌ 下载脚本不存在"
    exit 1
fi

# 检查Go文件是否下载成功
if [ ! -f "go1.23.12.linux-arm64.tar.gz" ]; then
    echo "❌ Go文件下载失败，请检查网络连接"
    echo "可以手动下载: wget https://golang.google.cn/dl/go1.23.12.linux-arm64.tar.gz"
    exit 1
fi

# 创建gRPC依赖目录（供手动下载的包使用）
echo "📁 创建gRPC依赖目录..."
mkdir -p grpc-deps
echo "📋 请手动下载以下ARM64包到 tools/grpc-deps/ 目录："
echo "   - libgrpc++-dev_*_arm64.deb"
echo "   - libgrpc-dev_*_arm64.deb"
echo "   - libprotobuf-dev_*_arm64.deb"
echo "   - protobuf-compiler_*_arm64.deb"

# 回到根目录并准备ARM64 protobuf工具
cd ../..

# 准备C++ SDK离线依赖
echo "📦 准备C++ SDK离线依赖..."

# 下载nlohmann/json
if [ -f "plum-offline-deploy/scripts-prepare/download-nlohmann-json.sh" ]; then
    echo "⬇️  下载nlohmann/json离线版本..."
    bash ./plum-offline-deploy/scripts-prepare/download-nlohmann-json.sh || {
        echo "⚠️  nlohmann/json下载失败，C++ SDK将无法在离线环境中构建"
    }
else
    echo "⚠️  未找到download-nlohmann-json.sh，跳过nlohmann/json下载"
fi

# 下载cpp-httplib
if [ -f "plum-offline-deploy/scripts-prepare/download-cpp-httplib.sh" ]; then
    echo "⬇️  下载cpp-httplib离线版本..."
    bash ./plum-offline-deploy/scripts-prepare/download-cpp-httplib.sh || {
        echo "⚠️  cpp-httplib下载失败，C++ SDK可能无法在离线环境中构建"
    }
else
    echo "⚠️  未找到download-cpp-httplib.sh，跳过cpp-httplib下载"
fi

# 注意：build-essential 已在目标机器手动安装，跳过相关下载步骤
echo "📋 build-essential 已在目标机器手动安装，跳过相关准备步骤"

if [ -f "plum-offline-deploy/scripts-prepare/prepare-arm64-go-tools.sh" ]; then
    echo "⚙️ 交叉编译ARM64 protobuf工具..."
    bash ./plum-offline-deploy/scripts-prepare/prepare-arm64-go-tools.sh || {
        echo "❌ ARM64工具编译失败"
        exit 1
    }
else
    echo "⚠️  未找到prepare-arm64-go-tools.sh，请手动运行"
fi

echo "✅ 准备完成！"
echo ""
echo "部署包已准备就绪，包含："
echo "✓ 源代码 (source/Plum/)"
echo "✓ Go依赖 (vendor/)"
echo "✓ Node.js依赖 (node_modules/)"
echo "✓ ARM64构建工具 (tools/)"
echo ""
echo "下一步：将整个 $DEPLOY_DIR 目录传输到目标ARM64环境"
