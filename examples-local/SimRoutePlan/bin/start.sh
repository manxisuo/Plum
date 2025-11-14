#!/bin/sh
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# 如果担心环境变量没有，可以再设置一次（虽然通常不需要）
# export DISPLAY=:99

echo "应用名称: $PLUM_APP_NAME" >> log
echo "应用版本: $PLUM_APP_VERSION" >> log
echo "实例ID: $PLUM_INSTANCE_ID" >> log

exec "$SCRIPT_DIR/SimRoutePlan" "$@"
