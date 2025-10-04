#!/bin/bash

# 停止agent的优雅脚本

echo "正在停止agent..."

# 查找agent进程
AGENT_PID=$(ps aux | grep -v grep | grep "./agent/build/plum_agent" | awk '{print $2}')

if [ -z "$AGENT_PID" ]; then
    echo "没有找到运行中的agent进程"
else
    echo "找到agent进程 PID: $AGENT_PID"
    
    # 优雅停止
    echo "发送SIGTERM信号..."
    kill -TERM $AGENT_PID
    
    # 等待清理
    echo "等待5秒让agent清理子进程..."
    sleep 5
    
    # 检查是否还在运行
    if kill -0 $AGENT_PID 2>/dev/null; then
        echo "agent仍在运行，发送SIGKILL信号..."
        kill -9 $AGENT_PID
    else
        echo "agent已优雅停止"
    fi
fi

# 清理可能的孤儿进程
echo "清理HelloUI进程..."
pkill -f HelloUI

echo "停止完成"
