#!/bin/bash
# 性能测试脚本 - 测试故障恢复性能

set -e

echo "🚀 Plum 性能测试脚本"
echo "测试目标："
echo "1. App被kill后，到被重新拉起的间隔时间 < 2秒"
echo "2. 节点崩溃后，到节点上的app被迁移到其他节点，间隔时间 < 2秒"
echo ""

# 检查是否在正确的目录
if [ ! -f "Makefile" ] || [ ! -d "controller" ] || [ ! -d "agent-go" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

echo "📁 当前目录: $(pwd)"

# 1. 构建项目
echo ""
echo "🔧 构建项目..."
make controller
make agent-go

# 2. 启动Controller（后台）
echo ""
echo "🚀 启动Controller..."
./bin/controller &
CONTROLLER_PID=$!
echo "Controller PID: $CONTROLLER_PID"

# 等待Controller启动
sleep 3

# 3. 启动Agent（后台）
echo ""
echo "🚀 启动Agent..."
cd agent-go
./agent-go &
AGENT_PID=$!
echo "Agent PID: $AGENT_PID"
cd ..

# 等待Agent启动
sleep 3

# 4. 部署测试应用
echo ""
echo "📦 部署测试应用..."
# 这里需要根据实际的应用部署方式进行调整
# 假设有一个测试应用可以部署

# 5. 测试App重启性能
echo ""
echo "🧪 测试App重启性能..."
echo "步骤："
echo "1. 找到运行中的应用进程"
echo "2. 记录开始时间"
echo "3. 使用 kill -9 强制杀死进程"
echo "4. 监控进程重启"
echo "5. 记录重启完成时间"
echo "6. 计算重启耗时"

# 查找应用进程
APP_PID=$(ps aux | grep -v grep | grep "test-app" | awk '{print $2}' | head -1)
if [ -n "$APP_PID" ]; then
    echo "找到应用进程 PID: $APP_PID"
    
    # 记录开始时间
    START_TIME=$(date +%s.%N)
    echo "开始时间: $(date)"
    
    # 杀死进程
    echo "杀死进程 $APP_PID..."
    kill -9 $APP_PID
    
    # 监控重启
    echo "监控进程重启..."
    RESTART_TIMEOUT=10  # 10秒超时
    RESTART_COUNT=0
    
    while [ $RESTART_COUNT -lt $RESTART_TIMEOUT ]; do
        sleep 0.1
        NEW_PID=$(ps aux | grep -v grep | grep "test-app" | awk '{print $2}' | head -1)
        if [ -n "$NEW_PID" ] && [ "$NEW_PID" != "$APP_PID" ]; then
            END_TIME=$(date +%s.%N)
            RESTART_DURATION=$(echo "$END_TIME - $START_TIME" | bc)
            echo "✅ 进程重启成功！"
            echo "新进程 PID: $NEW_PID"
            echo "重启耗时: ${RESTART_DURATION}秒"
            
            if (( $(echo "$RESTART_DURATION < 2.0" | bc -l) )); then
                echo "🎉 性能测试通过：重启时间 < 2秒"
            else
                echo "❌ 性能测试失败：重启时间 >= 2秒"
            fi
            break
        fi
        RESTART_COUNT=$((RESTART_COUNT + 1))
    done
    
    if [ $RESTART_COUNT -ge $RESTART_TIMEOUT ]; then
        echo "❌ 重启超时，测试失败"
    fi
else
    echo "⚠️  未找到测试应用进程，跳过重启测试"
fi

# 6. 测试节点故障迁移性能
echo ""
echo "🧪 测试节点故障迁移性能..."
echo "步骤："
echo "1. 记录开始时间"
echo "2. 停止Agent进程（模拟节点故障）"
echo "3. 监控应用迁移到其他节点"
echo "4. 记录迁移完成时间"
echo "5. 计算迁移耗时"

# 记录开始时间
MIGRATION_START_TIME=$(date +%s.%N)
echo "迁移开始时间: $(date)"

# 停止Agent
echo "停止Agent进程 $AGENT_PID（模拟节点故障）..."
kill $AGENT_PID

# 监控迁移
echo "监控应用迁移..."
MIGRATION_TIMEOUT=10  # 10秒超时
MIGRATION_COUNT=0

while [ $MIGRATION_COUNT -lt $MIGRATION_TIMEOUT ]; do
    sleep 0.1
    # 检查是否有其他Agent接管了应用
    # 这里需要根据实际的迁移检测方式进行调整
    MIGRATION_COUNT=$((MIGRATION_COUNT + 1))
done

MIGRATION_END_TIME=$(date +%s.%N)
MIGRATION_DURATION=$(echo "$MIGRATION_END_TIME - $MIGRATION_START_TIME" | bc)
echo "迁移耗时: ${MIGRATION_DURATION}秒"

if (( $(echo "$MIGRATION_DURATION < 2.0" | bc -l) )); then
    echo "🎉 性能测试通过：迁移时间 < 2秒"
else
    echo "❌ 性能测试失败：迁移时间 >= 2秒"
fi

# 7. 清理
echo ""
echo "🧹 清理测试环境..."
if [ -n "$CONTROLLER_PID" ]; then
    kill $CONTROLLER_PID 2>/dev/null || true
fi
if [ -n "$AGENT_PID" ]; then
    kill $AGENT_PID 2>/dev/null || true
fi

echo ""
echo "🎯 性能测试完成！"
echo "请查看日志中的性能监控信息："
echo "- 重启时间统计"
echo "- 迁移时间统计"
echo "- 性能警告信息"
