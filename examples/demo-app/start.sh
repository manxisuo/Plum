#!/bin/bash

# Plum Demo App启动脚本
# 此脚本会被Plum Agent调用

echo "Starting Plum Demo App..."
echo "Instance ID: $PLUM_INSTANCE_ID"
echo "App Name: $PLUM_APP_NAME"
echo "App Version: $PLUM_APP_VERSION"

# 启动应用
./demo-app

