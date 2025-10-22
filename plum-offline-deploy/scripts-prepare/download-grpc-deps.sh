#!/bin/bash
# 下载 gRPC 和 protobuf 开发包的脚本 - 适用于银河麒麟V10 ARM64

set -e

echo "🚀 下载 gRPC 和 protobuf 开发包 (ARM64)..."

# 确保在正确的目录运行
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOLS_DIR="$(dirname "$SCRIPT_DIR")/tools"

# 创建下载目录
GRPC_DEPS_DIR="$TOOLS_DIR/grpc-deps"
mkdir -p "$GRPC_DEPS_DIR"
cd "$GRPC_DEPS_DIR"

echo "📁 下载目录: $(pwd)"
echo "📋 目标平台: 银河麒麟V10 ARM64 (aarch64)"

# 定义下载来源和版本
# 使用Ubuntu 20.04 LTS的包，通常与银河麒麟V10兼容
BASE_URL="http://archive.ubuntu.com/ubuntu/pool/main"
PORTS_URL="http://ports.ubuntu.com/pool/main"

echo "📦 下载 gRPC 相关包..."

# 定义要下载的包列表和可能的版本
declare -A PACKAGES=(
    # grpc包
    ["libgrpc++-dev"]="1.27.0-0ubuntu1_arm64.deb"
    ["libgrpc-dev"]="1.27.0-0ubuntu1_arm64.deb"
    ["libgrpc11"]="1.27.0-0ubuntu1_arm64.deb"
    ["libgrpc++1"]="1.27.0-0ubuntu1_arm64.deb"
    ["grpc-devtools"]="1.27.0-0ubuntu1_arm64.deb"
    
    # protobuf包
    ["libprotobuf-dev"]="3.12.4-1ubuntu7_arm64.deb"
    ["libprotobuf23"]="3.12.4-1ubuntu7_arm64.deb"
    ["libprotoc23"]="3.12.4-1ubuntu7_arm64.deb"
    ["protobuf-compiler"]="3.12.4-1ubuntu7_arm64.deb"
    
    # 依赖包
    ["libc-ares2"]="1.16.1-1_arm64.deb"
    ["libc-ares-dev"]="1.16.1-1_arm64.deb"
    ["libssl-dev"]="1.1.1f-1ubuntu2.20_arm64.deb"
    ["libz-dev"]="1:1.2.11.dfsg-2ubuntu1.5_arm64.deb"
)

# 尝试多个版本号的下载函数
download_package() {
    local package_name="$1"
    local primary_version="$2"
    local downloaded=false
    
    # 尝试不同的URL基础路径
    local urls=(
        "$BASE_URL"
        "$PORTS_URL"
    )
    
    # 尝试不同的版本号
    local versions=(
        "$primary_version"
        "${primary_version%.*}.$((${primary_version##*.}-1))_arm64.deb"
        "${primary_version%.*}.$((${primary_version##*.}+1))_arm64.deb"
    )
    
    for url_base in "${urls[@]}"; do
        for version in "${versions[@]}"; do
            local filename="${package_name}_${version}"
            local url=""
            
            # 根据不同包类型选择正确的URL路径
            case "$package_name" in
                libgrpc*|grpc*)
                    url="${url_base}/g/grpc/${filename}"
                    ;;
                libprotobuf*|protobuf*)
                    url="${url_base}/p/protobuf/${filename}"
                    ;;
                libc-ares*)
                    url="${url_base}/c/c-ares/${filename}"
                    ;;
                libssl*)
                    url="${url_base}/o/openssl/${filename}"
                    ;;
                libz*)
                    url="${url_base}/z/zlib/${filename}"
                    ;;
            esac
            
            echo "  尝试下载: $url"
            if wget -c "$url" 2>/dev/null; then
                echo "  ✅ $filename 下载成功"
                downloaded=true
                break
            fi
        done
        
        if [ "$downloaded" = true ]; then
            break
        fi
    done
    
    if [ "$downloaded" = false ]; then
        echo "  ❌ $package_name 下载失败"
        return 1
    fi
}

# 下载所有包
echo "开始下载包..."
for package in "${!PACKAGES[@]}"; do
    echo "📦 下载 $package..."
    download_package "$package" "${PACKAGES[$package]}" || {
        echo "⚠️  $package 下载失败，继续下载其他包..."
    }
done

echo ""
echo "✅ 下载完成！"
echo "📋 下载的文件："
ls -la *.deb

echo ""
echo "🔍 验证关键包："
required_packages=("libgrpc++-dev" "libgrpc-dev" "libprotobuf-dev" "protobuf-compiler")
for pkg in "${required_packages[@]}"; do
    if ls ${pkg}_*.deb 1> /dev/null 2>&1; then
        echo "✅ $pkg: $(ls ${pkg}_*.deb | head -1)"
    else
        echo "❌ $pkg: 未找到"
    fi
done

echo ""
echo "📋 在目标机器上安装命令："
echo "cd /path/to/grpc-deps/"
echo "sudo dpkg -i *.deb"
echo "# 如果有依赖问题，运行："
echo "sudo apt-get install -f"
echo ""
echo "📋 或者逐个安装关键包："
required_packages=("libgrpc++-dev" "libgrpc-dev" "libprotobuf-dev" "protobuf-compiler")
for pkg in "${required_packages[@]}"; do
    if ls ${pkg}_*.deb 1> /dev/null 2>&1; then
        echo "sudo dpkg -i ${pkg}_*.deb"
    fi
done
