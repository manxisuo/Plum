#!/bin/bash

# 弱网环境测试脚本

echo "=== Plum弱网环境测试 ==="
echo "测试目标: 验证弱网环境下的服务发现和调用能力"
echo "开始时间: $(date)"

# 检查Controller状态
echo "检查Controller状态..."
if ! curl -s http://localhost:8080/healthz > /dev/null; then
    echo "❌ Controller未运行"
    echo "请先启动Controller:"
    echo "运行: make controller-run"
    exit 1
fi
echo "✅ Controller运行正常"

# 编译弱网环境测试工具
echo "编译弱网环境测试工具..."
cd tools
go build -o weak_network_test weak_network.go
if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi
echo "✅ 编译成功"

# 运行弱网环境测试
echo "运行弱网环境测试..."
./weak_network_test

echo "弱网环境测试完成"
