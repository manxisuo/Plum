#!/bin/bash

# Plum KV Demo启动脚本

LOG_FILE="kv-demo.log"

echo "Starting Plum KV Demo..."
echo "Log file: $LOG_FILE"

# 环境变量由Agent注入
echo "Environment:"
echo "  PLUM_INSTANCE_ID: $PLUM_INSTANCE_ID"
echo "  PLUM_APP_NAME: $PLUM_APP_NAME"
echo "  CONTROLLER_BASE: $CONTROLLER_BASE"

# 启动应用，同时输出到日志文件和stdout
./kv-demo 2>&1 | tee "$LOG_FILE"

