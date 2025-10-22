#!/bin/bash
# 查找protobuf头文件位置

echo "🔍 查找protobuf头文件位置..."

# 查找所有protobuf相关目录
echo "📁 查找protobuf相关目录："
find /usr -name "*protobuf*" -type d 2>/dev/null | head -10

echo ""
echo "📁 查找port_def.inc文件："
find /usr -name "port_def.inc" 2>/dev/null

echo ""
echo "📁 查找google/protobuf目录："
find /usr -path "*/google/protobuf" -type d 2>/dev/null

echo ""
echo "📁 检查/usr/include/google/protobuf内容："
if [ -d "/usr/include/google/protobuf" ]; then
    ls -la /usr/include/google/protobuf/ | head -10
else
    echo "❌ /usr/include/google/protobuf 不存在"
fi

echo ""
echo "📁 检查/usr/local/include/google/protobuf内容："
if [ -d "/usr/local/include/google/protobuf" ]; then
    ls -la /usr/local/include/google/protobuf/ | head -10
else
    echo "❌ /usr/local/include/google/protobuf 不存在"
fi

echo ""
echo "📁 查找所有google目录："
find /usr -path "*/google" -type d 2>/dev/null | head -5

echo ""
echo "📁 检查pkg-config信息："
if pkg-config --exists protobuf; then
    echo "protobuf包含目录: $(pkg-config --cflags protobuf)"
    echo "protobuf链接库: $(pkg-config --libs protobuf)"
else
    echo "❌ pkg-config protobuf不可用"
fi
