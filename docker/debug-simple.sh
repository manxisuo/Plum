#!/bin/bash
# 简单调试脚本

binary_path="/data/usershare/应用包/HelloUI/bin/HelloUI"

echo "=== 调试 ldd 输出解析 ==="
echo "二进制文件: $binary_path"
echo ""

# 获取ldd输出
ldd_output=$(ldd "$binary_path" 2>/dev/null)
echo "ldd输出:"
echo "$ldd_output"
echo ""

# 逐行处理
echo "=== 逐行处理 ==="
while IFS= read -r line; do
    echo "处理行: '$line'"
    
    # 检查是否包含=>符号
    if echo "$line" | grep -q "=>"; then
        echo "  ✓ 包含=>符号"
        
        # 提取库文件路径
        lib_path=$(echo "$line" | sed -n 's/.*=>[[:space:]]*\([^[:space:]]*\).*/\1/p')
        echo "  提取的路径: '$lib_path'"
        
        # 检查路径是否有效
        if [[ "$lib_path" != "not found" && -n "$lib_path" && "$lib_path" != "("* ]]; then
            echo "  ✓ 有效路径: $lib_path"
            if [ -f "$lib_path" ]; then
                echo "  ✓ 文件存在"
            else
                echo "  ✗ 文件不存在"
            fi
        else
            echo "  ✗ 无效路径"
        fi
    else
        echo "  ✗ 不包含=>符号"
    fi
    echo ""
done <<< "$ldd_output"

echo "=== 测试完成 ==="
