#!/bin/bash
# 银河麒麟V10 ARM64环境部署脚本

set -e

echo "🚀 开始部署Plum到银河麒麟V10..."

# 部署目录配置
DEPLOY_ROOT="/opt/plum"
SERVICE_USER="plum"
SERVICE_GROUP="plum"

# 1. 创建服务用户和目录
echo "📁 创建部署目录和服务用户..."
sudo mkdir -p $DEPLOY_ROOT/{bin,logs,data,ui}
sudo groupadd -f $SERVICE_GROUP
sudo useradd -r -g $SERVICE_GROUP -d $DEPLOY_ROOT -s /bin/false $SERVICE_USER || true
sudo chown -R $SERVICE_USER:$SERVICE_GROUP $DEPLOY_ROOT

# 2. 部署可执行文件
echo "📦 部署可执行文件..."
cd ../source/Plum

# 部署Controller
if [ -f "controller/bin/controller" ]; then
    sudo cp controller/bin/controller $DEPLOY_ROOT/bin/
    sudo chmod +x $DEPLOY_ROOT/bin/controller
    echo "✅ Controller已部署"
else
    echo "❌ Controller未找到，请先构建"
    exit 1
fi

# 部署Agent
if [ -f "agent-go/plum-agent" ]; then
    sudo cp agent-go/plum-agent $DEPLOY_ROOT/bin/
    sudo chmod +x $DEPLOY_ROOT/bin/plum-agent
    echo "✅ Agent已部署"
else
    echo "❌ Agent未找到，请先构建"
    exit 1
fi

# 3. 部署C++ SDK和Plum Client库
echo "📦 部署C++ SDK和Plum Client库..."

# 创建SDK目录
sudo mkdir -p $DEPLOY_ROOT/sdk/{lib,include,examples}

# 部署Plum Client库
if [ -f "sdk/cpp/build/plumclient/libplumclient.so" ]; then
    sudo cp sdk/cpp/build/plumclient/libplumclient.so $DEPLOY_ROOT/sdk/lib/
    sudo chmod 755 $DEPLOY_ROOT/sdk/lib/libplumclient.so
    echo "✅ Plum Client库已部署"
else
    echo "⚠️  Plum Client库未找到，跳过部署"
fi

# 部署Plum Client头文件
if [ -d "sdk/cpp/plumclient/include" ]; then
    sudo cp -r sdk/cpp/plumclient/include $DEPLOY_ROOT/sdk/
    sudo chown -R $SERVICE_USER:$SERVICE_GROUP $DEPLOY_ROOT/sdk/include
    echo "✅ Plum Client头文件已部署"
else
    echo "⚠️  Plum Client头文件未找到，跳过部署"
fi

# 部署Service Client示例
if [ -f "sdk/cpp/build/examples/service_client_example/service_client_example" ]; then
    sudo cp sdk/cpp/build/examples/service_client_example/service_client_example $DEPLOY_ROOT/sdk/examples/
    sudo chmod +x $DEPLOY_ROOT/sdk/examples/service_client_example
    echo "✅ Service Client示例已部署"
else
    echo "⚠️  Service Client示例未找到，跳过部署"
fi

# 部署其他C++示例
if [ -f "sdk/cpp/build/examples/echo_worker/echo_worker" ]; then
    sudo cp sdk/cpp/build/examples/echo_worker/echo_worker $DEPLOY_ROOT/sdk/examples/
    sudo chmod +x $DEPLOY_ROOT/sdk/examples/echo_worker
    echo "✅ Echo Worker示例已部署"
fi

if [ -f "sdk/cpp/build/examples/radar_sensor/radar_sensor" ]; then
    sudo cp sdk/cpp/build/examples/radar_sensor/radar_sensor $DEPLOY_ROOT/sdk/examples/
    sudo chmod +x $DEPLOY_ROOT/sdk/examples/radar_sensor
    echo "✅ Radar Sensor示例已部署"
fi

if [ -f "sdk/cpp/build/examples/grpc_echo_worker/grpc_echo_worker" ]; then
    sudo cp sdk/cpp/build/examples/grpc_echo_worker/grpc_echo_worker $DEPLOY_ROOT/sdk/examples/
    sudo chmod +x $DEPLOY_ROOT/sdk/examples/grpc_echo_worker
    echo "✅ gRPC Echo Worker示例已部署"
fi

# 4. 部署Web UI
echo "📦 部署Web UI..."
if [ -d "ui/dist" ]; then
    sudo cp -r ui/dist/* $DEPLOY_ROOT/ui/
    sudo chown -R $SERVICE_USER:$SERVICE_GROUP $DEPLOY_ROOT/ui
    echo "✅ Web UI已部署"
else
    echo "❌ Web UI未找到，请先构建"
    exit 1
fi

# 4. 创建配置文件
echo "📝 创建配置文件..."

# Controller环境配置
sudo tee $DEPLOY_ROOT/.env.controller > /dev/null << EOF
CONTROLLER_ADDR=:8080
CONTROLLER_DB=$DEPLOY_ROOT/data/controller.db
CONTROLLER_DATA_DIR=$DEPLOY_ROOT/data
HEARTBEAT_TTL_SEC=30
FAILOVER_ENABLED=true
EOF

# Agent环境配置
sudo tee $DEPLOY_ROOT/.env.agent > /dev/null << EOF
AGENT_NODE_ID=nodeA
CONTROLLER_BASE=http://127.0.0.1:8080
AGENT_DATA_DIR=$DEPLOY_ROOT/data/agent
EOF

sudo chown $SERVICE_USER:$SERVICE_GROUP $DEPLOY_ROOT/.env.*

# 5. 创建systemd服务文件
echo "🔧 创建systemd服务..."

# Controller服务
sudo tee /etc/systemd/system/plum-controller.service > /dev/null << EOF
[Unit]
Description=Plum Controller
After=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_GROUP
WorkingDirectory=$DEPLOY_ROOT
EnvironmentFile=$DEPLOY_ROOT/.env.controller
ExecStart=$DEPLOY_ROOT/bin/controller
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Agent服务
sudo tee /etc/systemd/system/plum-agent.service > /dev/null << EOF
[Unit]
Description=Plum Agent
After=network.target plum-controller.service
Requires=plum-controller.service

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_GROUP
WorkingDirectory=$DEPLOY_ROOT
EnvironmentFile=$DEPLOY_ROOT/.env.agent
ExecStart=$DEPLOY_ROOT/bin/plum-agent
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 6. 配置nginx（如果需要）
echo "🌐 配置nginx..."
if command -v nginx &> /dev/null; then
    sudo tee /etc/nginx/sites-available/plum > /dev/null << EOF
server {
    listen 80;
    server_name localhost;
    
    location / {
        root $DEPLOY_ROOT/ui;
        try_files \$uri \$uri/ /index.html;
    }
    
    location /v1/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }
}
EOF
    
    sudo ln -sf /etc/nginx/sites-available/plum /etc/nginx/sites-enabled/
    sudo nginx -t && sudo systemctl reload nginx
    echo "✅ nginx配置完成"
else
    echo "⚠️  nginx未安装，Web UI需要手动配置Web服务器"
fi

# 7. 设置权限和启动服务
echo "🔧 设置权限..."
sudo chown -R $SERVICE_USER:$SERVICE_GROUP $DEPLOY_ROOT
sudo chmod -R 755 $DEPLOY_ROOT/bin
sudo chmod -R 644 $DEPLOY_ROOT/.env.*

# 重载systemd配置
sudo systemctl daemon-reload

# 8. 启动服务
echo "🚀 启动服务..."
sudo systemctl enable plum-controller plum-agent
sudo systemctl start plum-controller
sleep 3
sudo systemctl start plum-agent

# 检查服务状态
echo "🔍 检查服务状态..."
sudo systemctl status plum-controller --no-pager -l
sudo systemctl status plum-agent --no-pager -l

echo ""
echo "🎉 部署完成！"
echo ""
echo "服务状态："
echo "- Controller: sudo systemctl status plum-controller"
echo "- Agent: sudo systemctl status plum-agent"
echo ""
echo "访问地址："
echo "- Web UI: http://localhost (如果配置了nginx)"
echo "- API: http://localhost:8080/v1/"
echo ""
echo "C++ SDK部署："
echo "- Plum Client库: $DEPLOY_ROOT/sdk/lib/libplumclient.so"
echo "- 头文件: $DEPLOY_ROOT/sdk/include/"
echo "- 示例程序: $DEPLOY_ROOT/sdk/examples/"
echo ""
echo "C++ SDK使用："
echo "- 编译时链接: -L$DEPLOY_ROOT/sdk/lib -lplumclient"
echo "- 包含头文件: -I$DEPLOY_ROOT/sdk/include"
echo "- 运行示例: $DEPLOY_ROOT/sdk/examples/service_client_example"
echo ""
echo "日志查看："
echo "- Controller: sudo journalctl -u plum-controller -f"
echo "- Agent: sudo journalctl -u plum-agent -f"
echo ""
echo "配置文件："
echo "- Controller: $DEPLOY_ROOT/.env.controller"
echo "- Agent: $DEPLOY_ROOT/.env.agent"
