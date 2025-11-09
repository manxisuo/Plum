#!/bin/bash
# 检查和修复protobuf开发包问题

set -e

echo "🔍 检查protobuf开发包状态..."

# 检查protoc是否可用
echo "📋 检查protoc编译器："
if command -v protoc &> /dev/null; then
    echo "✅ protoc版本: $(protoc --version)"
else
    echo "❌ protoc不可用"
fi

# 检查protobuf头文件
echo ""
echo "📋 检查protobuf头文件："
PROTOBUF_HEADER_PATHS=(
    "/usr/include/google/protobuf"
    "/usr/local/include/google/protobuf"
    "/usr/include/google/protobuf/port_def.inc"
    "/usr/local/include/google/protobuf/port_def.inc"
)

for path in "${PROTOBUF_HEADER_PATHS[@]}"; do
    if [ -e "$path" ]; then
        echo "✅ 找到: $path"
    else
        echo "❌ 缺失: $path"
    fi
done

# 检查pkg-config
echo ""
echo "📋 检查pkg-config protobuf："
if pkg-config --exists protobuf; then
    echo "✅ protobuf pkg-config信息："
    echo "   版本: $(pkg-config --modversion protobuf)"
    echo "   包含目录: $(pkg-config --cflags protobuf)"
    echo "   链接库: $(pkg-config --libs protobuf)"
else
    echo "❌ pkg-config protobuf不可用"
fi

# 检查已安装的protobuf包
echo ""
echo "📋 检查已安装的protobuf包："
if command -v dpkg &> /dev/null; then
    echo "已安装的protobuf相关包："
    dpkg -l | grep -i protobuf || echo "未找到protobuf包"
else
    echo "dpkg不可用，无法检查包状态"
fi

# 建议修复方案
echo ""
echo "🔧 建议修复方案："
echo "1. 如果protobuf-dev包缺失，请安装："
echo "   sudo apt-get install libprotobuf-dev protobuf-compiler"
echo ""
echo "2. 如果包已安装但头文件缺失，请重新安装："
echo "   sudo apt-get install --reinstall libprotobuf-dev"
echo ""
echo "3. 检查gRPC依赖包是否完整："
echo "   sudo apt-get install libgrpc++-dev libgrpc-dev"
