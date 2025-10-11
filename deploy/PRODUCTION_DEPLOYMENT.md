# 生产环境部署指南

## 🎯 架构概览

```
┌─────────────────────────────────────┐
│  Nginx (80/443)                     │
│  - 服务UI静态文件 (ui/dist/)         │
│  - 反向代理API (/v1/ → :8080)       │
└─────────────────────────────────────┘
         ↓
┌─────────────────────────────────────┐
│  Plum Controller (8080)             │
│  - systemd管理                       │
│  - 自动重启                          │
└─────────────────────────────────────┘
         ↓
┌─────────────────────────────────────┐
│  Plum Agents (多节点)                │
│  - systemd模板服务                   │
│  - 每个节点独立实例                   │
└─────────────────────────────────────┘
```

## 📦 完整部署流程

### 步骤1：准备环境

```bash
# 以manxisuo用户登录服务器
ssh manxisuo@39.106.128.81

# 克隆项目
cd ~
git clone https://github.com/manxisuo/plum.git
cd plum

# 配置Go代理（中国网络）
go env -w GOPROXY=https://goproxy.cn,direct
```

### 步骤2：构建所有组件

```bash
cd /home/manxisuo/Plum

# 生成proto
make proto

# 构建Controller
make controller

# 构建Agent
make agent

# 构建UI静态文件
make ui
make ui-build

# 验证构建产物
ls -la controller/bin/controller
ls -la agent-go/plum-agent
ls -la ui/dist/index.html
```

### 步骤3：创建数据目录

```bash
mkdir -p /home/manxisuo/Plum/data
mkdir -p /home/manxisuo/Plum/data/agents
mkdir -p /home/manxisuo/Plum/data/artifacts
mkdir -p /home/manxisuo/Plum/logs
```

### 步骤4：安装systemd服务

```bash
# 安装Controller
sudo cp deploy/systemd/plum-controller.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl start plum-controller
sudo systemctl enable plum-controller

# 安装Agent
sudo cp deploy/systemd/plum-agent@.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl start plum-agent@nodeA
sudo systemctl enable plum-agent@nodeA

# 验证服务状态
sudo systemctl status plum-controller
sudo systemctl status plum-agent@nodeA
```

### 步骤5：配置Nginx

```bash
# 修复目录权限
chmod +x /home/manxisuo
chmod +x /home/manxisuo/Plum
chmod +x /home/manxisuo/Plum/ui
chmod -R 755 /home/manxisuo/Plum/ui/dist

# 创建nginx配置
sudo nano /etc/nginx/sites-available/plum
# 内容见下方"Nginx配置"

# 禁用默认站点
sudo rm /etc/nginx/sites-enabled/default

# 启用Plum
sudo ln -s /etc/nginx/sites-available/plum /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Nginx配置内容

```nginx
server {
    listen 80;
    server_name 39.106.128.81;  # 改成你的域名或IP

    # 访问日志
    access_log /var/log/nginx/plum-access.log;
    error_log /var/log/nginx/plum-error.log;

    # UI静态文件
    location / {
        root /home/manxisuo/Plum/ui/dist;
        try_files $uri $uri/ /index.html;
        index index.html;
    }

    # API代理
    location /v1/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # SSE实时更新
    location /v1/stream {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Connection '';
        proxy_buffering off;
        proxy_cache off;
        chunked_transfer_encoding off;
    }

    # 任务实时流
    location /v1/tasks/stream {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Connection '';
        proxy_buffering off;
        proxy_cache off;
        chunked_transfer_encoding off;
    }

    # Swagger UI
    location /swagger {
        proxy_pass http://127.0.0.1:8080;
    }

    # 健康检查
    location /healthz {
        proxy_pass http://127.0.0.1:8080;
    }

    # artifacts静态文件
    location /artifacts/ {
        proxy_pass http://127.0.0.1:8080;
    }
}
```

### 步骤6：验证部署

```bash
# 1. 检查所有服务
sudo systemctl status plum-controller
sudo systemctl status plum-agent@nodeA
sudo systemctl status nginx

# 2. 测试API
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/v1/nodes

# 3. 访问Web UI
# 浏览器打开: http://39.106.128.81
```

## 🔧 常用管理命令

### 查看日志
```bash
# Controller日志
sudo journalctl -u plum-controller -f

# Agent日志
sudo journalctl -u plum-agent@nodeA -f

# 所有Plum服务日志
sudo journalctl -u 'plum-*' -f

# 查看最近错误
sudo journalctl -u plum-controller -p err -n 50
```

### 服务管理
```bash
# 重启所有服务
sudo systemctl restart plum-controller
sudo systemctl restart 'plum-agent@*'
sudo systemctl reload nginx

# 查看所有Plum服务
systemctl list-units 'plum-*'

# 开机自启管理
sudo systemctl enable plum-controller
sudo systemctl enable plum-agent@nodeA
```

## 🔄 更新部署

### 更新代码
```bash
# 1. 停止服务
sudo systemctl stop 'plum-agent@*'
sudo systemctl stop plum-controller

# 2. 更新代码
cd /home/manxisuo/Plum
git pull
make proto
make controller
make agent

# 3. 更新UI（如果有变化）
make ui-build

# 4. 重启服务
sudo systemctl start plum-controller
sudo systemctl start 'plum-agent@*'

# 5. 重载nginx（如果配置有变）
sudo systemctl reload nginx
```

### 快速重启
```bash
# 只重启服务，不重新构建
sudo systemctl restart plum-controller
sudo systemctl restart 'plum-agent@*'
```

## 🔐 安全加固

### 1. 使用专用用户
```bash
# 创建plum用户
sudo useradd -r -s /bin/bash -m plum

# 修改文件所有者
sudo chown -R plum:plum /home/manxisuo/Plum

# 修改service文件的User和Group
sudo systemctl edit plum-controller --full
# User=plum
# Group=plum
```

### 2. 配置防火墙
```bash
# 只开放必要端口
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 22/tcp   # SSH

# 关闭其他端口
sudo ufw deny 8080/tcp  # Controller只在内部访问
sudo ufw deny 5173/tcp  # Vite dev不用于生产

# 启用防火墙
sudo ufw enable
```

### 3. 配置HTTPS
```bash
# 安装certbot
sudo apt install -y certbot python3-certbot-nginx

# 自动配置HTTPS
sudo certbot --nginx -d your-domain.com

# 自动续期
sudo certbot renew --dry-run
```

## 📊 监控和维护

### 系统资源监控
```bash
# 查看服务资源占用
systemctl status plum-controller
systemd-cgtop

# 磁盘使用
du -sh /home/manxisuo/Plum/data/*
```

### 日志管理
```bash
# 限制日志大小
sudo journalctl --vacuum-size=100M
sudo journalctl --vacuum-time=7d

# 配置日志保留策略
sudo nano /etc/systemd/journald.conf
# SystemMaxUse=100M
# MaxRetentionSec=7day
```

### 数据库备份
```bash
# 创建备份脚本
cat > /home/manxisuo/Plum/backup.sh <<'EOF'
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR=/home/manxisuo/Plum/backups
mkdir -p $BACKUP_DIR
cp /home/manxisuo/Plum/data/plum.db $BACKUP_DIR/plum_$DATE.db
# 保留最近7天备份
find $BACKUP_DIR -name "plum_*.db" -mtime +7 -delete
EOF

chmod +x /home/manxisuo/Plum/backup.sh

# 添加到crontab（每天凌晨2点备份）
crontab -e
# 0 2 * * * /home/manxisuo/Plum/backup.sh
```

## 🚨 应急处理

### 服务无响应
```bash
# 强制重启
sudo systemctl kill -s KILL plum-controller
sudo systemctl start plum-controller
```

### 回滚版本
```bash
# 停止服务
sudo systemctl stop plum-controller plum-agent@nodeA

# 回滚代码
cd /home/manxisuo/Plum
git checkout <commit-id>
make controller
make agent

# 恢复数据库（如果需要）
cp backups/plum_20251010.db data/plum.db

# 重启服务
sudo systemctl start plum-controller plum-agent@nodeA
```

## ✅ 部署检查清单

- [ ] Controller服务运行中
- [ ] Agent服务运行中
- [ ] Nginx配置正确
- [ ] 数据目录权限正确
- [ ] 防火墙规则配置
- [ ] 开机自启已启用
- [ ] 日志正常输出
- [ ] Web UI可访问
- [ ] API可访问
- [ ] 备份脚本配置

---

**完成部署后，访问 http://39.106.128.81 即可使用Plum！**

