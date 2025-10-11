#!/bin/bash

# Plum Worker Demo启动脚本

echo "Starting Plum Worker Demo..."

# 设置环境变量（如果Agent未注入）
export WORKER_ID="${WORKER_ID:-worker-demo-${PLUM_INSTANCE_ID}}"
export WORKER_NODE_ID="${WORKER_NODE_ID:-nodeA}"
export CONTROLLER_BASE="${CONTROLLER_BASE:-http://127.0.0.1:8080}"
export GRPC_ADDRESS="${GRPC_ADDRESS:-0.0.0.0:18090}"

echo "Environment:"
echo "  PLUM_INSTANCE_ID: $PLUM_INSTANCE_ID"
echo "  PLUM_APP_NAME: $PLUM_APP_NAME"
echo "  PLUM_APP_VERSION: $PLUM_APP_VERSION"
echo "  WORKER_ID: $WORKER_ID"
echo "  GRPC_ADDRESS: $GRPC_ADDRESS"

# 启动Worker
./worker-demo

