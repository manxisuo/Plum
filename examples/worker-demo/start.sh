#!/bin/bash

# Plum Worker Demo启动脚本

echo "Starting Plum Worker Demo..."

# 设置环境变量（如果Agent未注入）
export WORKER_ID="${WORKER_ID:-worker-demo-${PLUM_INSTANCE_ID}}"
export WORKER_NODE_ID="${WORKER_NODE_ID:-nodeA}"
export CONTROLLER_GRPC_ADDR="${CONTROLLER_GRPC_ADDR:-127.0.0.1:9090}"

echo "Environment:"
echo "  PLUM_INSTANCE_ID: $PLUM_INSTANCE_ID"
echo "  PLUM_APP_NAME: $PLUM_APP_NAME"
echo "  PLUM_APP_VERSION: $PLUM_APP_VERSION"
echo "  WORKER_ID: $WORKER_ID"
echo "  CONTROLLER_GRPC_ADDR: $CONTROLLER_GRPC_ADDR"

# 启动Worker
exec ./worker-demo

