#!/bin/bash
# SimDecision 启动脚本

cd "$(dirname "$0")"

# 检查 Python 环境
if ! command -v python3 &> /dev/null; then
    echo "错误: 未找到 python3，请先安装 Python 3"
    exit 1
fi

# 检查依赖
if ! python3 -c "import flask" 2>/dev/null; then
    echo "警告: Flask 未安装，正在尝试安装..."
    pip3 install -r ../requirements.txt || {
        echo "错误: 无法安装依赖，请手动执行: pip3 install -r requirements.txt"
        exit 1
    }
fi

# 启动应用
export FLASK_APP=../app.py
export FLASK_ENV=production
python3 ../app.py

