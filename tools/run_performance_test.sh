#!/bin/bash

# Plum性能测试脚本
# 用于测试50节点并发访问能力

echo "=== Plum性能测试 ==="
echo "测试目标: 50个并发节点，持续5分钟"
echo "开始时间: $(date)"
echo

# 检查Controller是否运行
echo "检查Controller状态..."
if ! curl -s http://localhost:8080/healthz > /dev/null; then
    echo "❌ Controller未运行，请先启动Controller"
    echo "运行: make controller-run"
    exit 1
fi
echo "✅ Controller运行正常"

# 编译性能测试工具
echo "编译性能测试工具..."
cd tools
go build -o performance_test performance.go
if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi
echo "✅ 编译成功"

# 运行性能测试
echo "开始性能测试..."
echo "注意: 测试将持续5分钟，请耐心等待"
echo

./performance_test

echo
echo "测试完成时间: $(date)"
echo "=== 性能测试结束 ==="
