# Nginx生产环境部署指南

## 📦 安装Nginx

### Ubuntu/Debian
```bash
# 更新包列表
sudo apt update

# 安装nginx
sudo apt install -y nginx

# 验证安装
nginx -v  # nginx version: nginx/1.18.0

# 启动nginx
sudo systemctl start nginx
sudo systemctl enable nginx  # 开机自启

# 检查状态
sudo systemctl status nginx
```

### 验证安装成功
```bash
# 访问默认页面
curl http://localhost

# 应该看到"Welcome to nginx!"
```

## 🔧 配置Plum

### 方案1：完整配置（推荐）

创建配置文件：
```bash
sudo nano /etc/nginx/sites-available/plum
```

配置内容：
```nginx
server {
    listen 80;
    server_name your-domain.com;  # 改成你的域名或IP

    # 前端静态文件
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
    }

    # SSE支持（实时更新）
    location /v1/stream {
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
}
```

启用配置：
```bash
# 创建软链接
sudo ln -s /etc/nginx/sites-available/plum /etc/nginx/sites-enabled/

# 测试配置
sudo nginx -t

# 重载配置
sudo systemctl reload nginx
```

### 方案2：简化配置（仅代理）

如果UI使用独立端口（如5173），只需代理API：

```bash
sudo nano /etc/nginx/sites-available/plum-api
```

```nginx
server {
    listen 80;
    server_name api.your-domain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 🚀 部署步骤

### 完整生产部署流程

```bash
# 1. 构建UI静态文件
cd /home/manxisuo/Plum
make ui-build

# 2. 验证构建产物
ls -la ui/dist/
# 应该看到: index.html, assets/等

# 3. 配置nginx（见上面）
sudo nano /etc/nginx/sites-available/plum

# 4. 启用配置
sudo ln -s /etc/nginx/sites-available/plum /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# 5. 启动Controller（后台）
cd /home/manxisuo/Plum
nohup ./controller/bin/controller > logs/controller.log 2>&1 &

# 6. 启动Agent（后台）
nohup ./agent-go/plum-agent > logs/agent.log 2>&1 &

# 7. 访问
# http://your-domain.com 或 http://your-ip
```

## 🔍 故障排查

### 检查nginx配置
```bash
# 测试配置文件语法
sudo nginx -t

# 查看nginx错误日志
sudo tail -f /var/log/nginx/error.log

# 查看nginx访问日志
sudo tail -f /var/log/nginx/access.log
```

### 检查服务状态
```bash
# nginx状态
sudo systemctl status nginx

# Controller是否运行
ps aux | grep controller

# 检查端口监听
sudo netstat -tlnp | grep -E "(80|8080|5173)"
```

### 权限问题
```bash
# nginx需要读取ui/dist/目录
# 确保权限正确
chmod -R 755 /home/manxisuo/Plum/ui/dist

# 或修改nginx用户
# /etc/nginx/nginx.conf
user manxisuo;  # 改成你的用户名
```

## 🔐 HTTPS配置（可选）

### 使用Let's Encrypt免费证书

```bash
# 安装certbot
sudo apt install -y certbot python3-certbot-nginx

# 自动配置HTTPS
sudo certbot --nginx -d your-domain.com

# 自动续期
sudo certbot renew --dry-run
```

配置后nginx会自动更新为：
```nginx
server {
    listen 443 ssl;
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ...
}
```

## 📝 常用nginx命令

```bash
# 启动/停止/重启
sudo systemctl start nginx
sudo systemctl stop nginx
sudo systemctl restart nginx
sudo systemctl reload nginx  # 重载配置（推荐，不中断连接）

# 查看状态
sudo systemctl status nginx

# 开机自启
sudo systemctl enable nginx
sudo systemctl disable nginx

# 测试配置
sudo nginx -t

# 查看版本
nginx -v
```

## 🎯 快速测试配置

最小化测试nginx是否正常工作：

```bash
# 1. 简单配置
sudo tee /etc/nginx/sites-available/test <<EOF
server {
    listen 8888;
    location / {
        return 200 'Nginx works!';
        add_header Content-Type text/plain;
    }
}
EOF

# 2. 启用
sudo ln -s /etc/nginx/sites-available/test /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# 3. 测试
curl http://localhost:8888
# 应该输出: Nginx works!

# 4. 确认后删除测试配置
sudo rm /etc/nginx/sites-enabled/test
sudo systemctl reload nginx
```

## 📊 性能优化（可选）

### nginx.conf优化

```nginx
# /etc/nginx/nginx.conf
worker_processes auto;
worker_connections 1024;

http {
    # 启用gzip压缩
    gzip on;
    gzip_types text/plain text/css application/json application/javascript;
    
    # 缓存静态文件
    location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

---

**提示**：生产环境建议使用nginx+静态文件，不要使用Vite开发服务器。

