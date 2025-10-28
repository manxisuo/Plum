# Plum Docker 部署问题解决指南

## 📋 概述

本文档记录了Plum项目Docker部署过程中遇到的常见问题及其解决方案，包括容器启动失败、权限问题、网络配置等。

## 🔧 问题1：容器启动失败 - exec format error

### 问题描述
```
standard_init_linux.go:228: exec user process caused: exec format error
```

### 问题原因
Docker镜像的架构（x86_64）与目标环境的CPU架构（ARM64）不匹配。

### 解决方案

#### 方案1：在目标ARM64环境重新构建镜像（推荐）
```bash
# 停止并清理旧服务和镜像
docker-compose -f docker-compose.offline.yml down
docker rmi plum-controller:latest plum-controller:offline plum-agent:latest plum-agent:offline

# 使用ARM64构建脚本
./docker/build-static-offline-fixed.sh

# 重新启动服务
docker-compose -f docker-compose.offline.yml up -d
```

#### 方案2：使用预构建的ARM64镜像
```bash
# 在联网环境准备ARM64镜像
docker pull --platform linux/arm64 nginx:alpine
docker save nginx:alpine | gzip > nginx-alpine-arm64.tar.gz

# 在目标环境加载镜像
docker load < nginx-alpine-arm64.tar.gz
```

### 预防措施
- 确保在目标架构环境中构建镜像
- 使用 `--platform linux/arm64` 参数强制构建ARM64镜像
- 验证镜像架构：`docker inspect <image> | grep -i Architecture`

---

## 🔧 问题2：数据库权限错误 - readonly database

### 问题描述
```
init db error: attempt to write a readonly database (1544)
```

### 问题原因
SQLite数据库文件或目录没有写入权限，通常是Docker数据卷权限问题。

### 解决方案

#### 方案1：重新创建数据卷
```bash
# 停止服务
docker-compose -f docker-compose.offline.yml down

# 删除有问题的数据卷
docker volume rm plum-offline_plum-controller-data

# 重新启动服务（会自动创建新的数据卷）
docker-compose -f docker-compose.offline.yml up -d
```

#### 方案2：手动设置数据卷权限
```bash
# 停止服务
docker-compose -f docker-compose.offline.yml down

# 创建数据卷并设置权限
docker volume create plum-offline_plum-controller-data

# 启动临时容器设置权限
docker run --rm -v plum-offline_plum-controller-data:/data alpine:3.18 sh -c "
  mkdir -p /data && 
  chown -R 1001:1001 /data && 
  chmod -R 755 /data
"

# 重新启动服务
docker-compose -f docker-compose.offline.yml up -d
```

#### 方案3：使用绑定挂载（临时解决）
```bash
# 创建本地目录
mkdir -p ./data/controller

# 修改docker-compose.offline.yml中的volumes配置
# 将 plum-controller-data:/app/data 改为 ./data/controller:/app/data
```

### 预防措施
- 确保Dockerfile中正确设置用户权限
- 使用非root用户运行容器
- 定期检查数据卷权限

---

## 🔧 问题3：文件上传失败 - HTTP 413

### 问题描述
```
413 Request Entity Too Large
```

### 问题原因
Nginx的 `client_max_body_size` 设置过小，限制了上传文件大小。

### 解决方案

#### 方案1：修改Nginx配置
编辑 `docker/nginx/nginx.conf` 文件，在 `http` 块中添加：

```nginx
http {
    # 设置客户端请求体最大大小为50MB
    client_max_body_size 50M;
    
    # 设置超时时间
    client_body_timeout 60s;
    client_header_timeout 60s;
    
    # 其他配置...
}
```

#### 方案2：完整的Nginx配置示例
```nginx
events {
    worker_connections 1024;
}

http {
    # 文件上传配置
    client_max_body_size 50M;
    client_body_timeout 60s;
    client_header_timeout 60s;
    
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    # 日志格式
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;
    error_log /var/log/nginx/error.log;

    # 基本配置
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    # Gzip压缩
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types
        text/plain
        text/css
        text/xml
        text/javascript
        application/json
        application/javascript
        application/xml+rss
        application/atom+xml
        image/svg+xml;

    # 上游服务器配置
    upstream plum_controller {
        server plum-controller:8080;
    }

    # 主服务器配置
    server {
        listen 80;
        server_name localhost;

        # 静态文件服务 (Web UI)
        location / {
            root /usr/share/nginx/html;
            index index.html;
            try_files $uri $uri/ /index.html;

            # 缓存配置
            location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
                expires 1y;
                add_header Cache-Control "public, immutable";
            }
        }

        # API代理到Controller
        location /v1/ {
            proxy_pass http://plum_controller;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            # 超时配置
            proxy_connect_timeout 30s;
            proxy_send_timeout 30s;
            proxy_read_timeout 30s;

            # 缓冲配置
            proxy_buffering on;
            proxy_buffer_size 4k;
            proxy_buffers 8 4k;
        }

        # 健康检查
        location /health {
            access_log off;
            return 200 "healthy\n";
            add_header Content-Type text/plain;
        }

        # 错误页面
        error_page 404 /404.html;
        error_page 500 502 503 504 /50x.html;

        location = /50x.html {
            root /usr/share/nginx/html;
        }
    }
}
```

#### 方案3：重启Nginx服务
```bash
# 重启nginx容器
docker-compose -f docker-compose.offline.yml restart plum-nginx

# 或者重新加载配置
docker exec -it plum-nginx nginx -s reload

# 验证配置
docker exec -it plum-nginx nginx -t
```

### 预防措施
- 根据实际需求设置合适的 `client_max_body_size`
- 考虑设置合理的超时时间
- 定期检查Nginx配置

---

## 🔧 问题4：离线环境Docker构建失败 - apk add网络错误

### 问题描述
```
ERROR: unable to select packages:
  ca-certificates (no such package):
    required by: world[ca-certificates]
```

### 问题原因
Docker构建过程中 `apk add` 命令试图联网下载包，但离线环境无法访问Alpine包仓库。

### 解决方案

#### 方案1：移除apk add命令（推荐）
修改 `docker/build-static-offline-fixed.sh`，移除网络依赖的包安装：

```bash
# Controller静态Dockerfile
cat > Dockerfile.controller.static << 'EOF'
FROM alpine:3.18
WORKDIR /app
# 注意：这里假设alpine:3.18已经包含了必要的包
COPY controller/bin/controller ./bin/controller
RUN addgroup -g 1001 -S plum && adduser -u 1001 -S plum -G plum
RUN mkdir -p /app/data && chown -R plum:plum /app
USER plum
EXPOSE 8080
CMD ["./bin/controller"]
EOF
```

#### 方案2：在联网环境准备完整镜像
```bash
# 在联网环境中运行
./docker/prepare-alpine-with-packages.sh

# 传输生成的镜像到目标环境
# alpine-3.18-with-packages-arm64.tar.gz
```

#### 方案3：使用scratch镜像
```bash
# 使用完全静态的scratch镜像
./docker/build-scratch-images.sh
```

### 预防措施
- 在联网环境预先准备包含必要包的镜像
- 使用静态编译的Go二进制文件
- 避免在离线环境中使用需要网络的操作

---

## 🔧 问题5：容器无法进入 - sh not found

### 问题描述
```
OCI runtime exec failed: exec failed: container_linux.go:380: starting container process caused: exec: "sh": executable file not found in $PATH
```

### 问题原因
使用了 `scratch` 基础镜像，没有shell环境。

### 解决方案

#### 方案1：使用alpine基础镜像
```bash
# 修改Dockerfile使用alpine而不是scratch
FROM alpine:3.18
# 而不是 FROM scratch
```

#### 方案2：准备包含shell的镜像
```bash
# 在联网环境准备包含必要工具的镜像
docker pull --platform linux/arm64 alpine:3.18
docker save alpine:3.18 | gzip > alpine-3.18-arm64.tar.gz
```

### 预防措施
- 根据需求选择合适的基础镜像
- 如果需要调试，使用包含shell的镜像
- 如果只需要运行服务，可以使用scratch镜像

---

## 🔧 问题6：Docker Compose版本兼容性

### 问题描述
```
ERROR: In file './docker-compose.offline.yml', service 'name' must be a mapping not a string.
ERROR: Unsupported config option for services.plum-nginx: 'profiles'
```

### 问题原因
Docker Compose版本过低（如1.25.0），不支持某些新特性。

### 解决方案

#### 方案1：使用兼容的配置格式
```yaml
# 使用 version: '3.3' 而不是 name: plum-offline
version: '3.3'

services:
  plum-controller:
    # 移除 profiles 字段
    # profiles: - nginx  # 删除这行
```

#### 方案2：移除不支持的字段
- 移除 `profiles` 字段
- 移除 `start_period` 字段
- 使用 `depends_on` 替代 `profiles`

### 预防措施
- 检查目标环境的Docker Compose版本
- 使用兼容的配置格式
- 测试配置文件语法：`docker-compose config`

---

## 🔧 问题7：应用执行失败 - not found

### 问题描述
```
./start.sh: exec: line 8: /app/data/nodeA/.../HelloUI: not found
sh: ./HelloUI: not found
```

### 问题原因
1. **架构不匹配**：应用可执行文件的架构（x86_64）与目标环境的CPU架构（ARM64）不匹配
2. **缺少动态链接库**：应用是动态链接的，但容器内缺少必要的系统库文件

### 解决方案

#### 方案1：检查文件架构（确认问题）
```bash
# 在容器中检查文件架构
file ./HelloUI

# 检查系统架构
uname -m

# 检查动态链接库依赖
ldd ./HelloUI 2>/dev/null || echo "静态链接或架构不匹配"
```

#### 方案1.1：检查容器内库文件（架构匹配但缺少库）
```bash
# 检查必要的动态链接库
ls -la /lib/ld-linux-aarch64.so.1
ls -la /lib/libpthread.so.0
ls -la /lib/libc.so.6

# 查找库文件
find /lib -name "libpthread.so*" 2>/dev/null
find /lib -name "libc.so*" 2>/dev/null
```

#### 方案2：重新构建Agent镜像（推荐）
```bash
# 确保使用alpine:3.18基础镜像（包含动态链接库）
./docker/build-static-offline-fixed.sh

# 重新启动Agent服务
docker-compose -f docker-compose.offline.yml restart plum-agent-a
```

#### 方案3：手动复制库文件（推荐）
```bash
# 1. 查找必要的系统库文件
echo "🔍 查找系统库文件..."
find /lib -name "libpthread.so*" -exec ls -la {} \;
find /lib -name "libc.so*" -exec ls -la {} \;
find /lib -name "ld-linux-aarch64.so*" -exec ls -la {} \;

# 2. 复制基础库文件到容器中
echo "📦 复制库文件到容器..."
docker cp /lib/libpthread.so.0 plum-agent-a:/lib/
docker cp /lib/libc.so.6 plum-agent-a:/lib/
docker cp /lib/ld-linux-aarch64.so.1 plum-agent-a:/lib/

# 3. 设置执行权限
echo "🔧 设置执行权限..."
docker exec -it plum-agent-a chmod +x /lib/ld-linux-aarch64.so.1

# 4. 验证库文件
echo "✅ 验证库文件..."
docker exec -it plum-agent-a ls -la /lib/libpthread.so.0 /lib/libc.so.6 /lib/ld-linux-aarch64.so.1
```

#### 方案3.1：复制其他常用库文件
```bash
# 复制其他常用的系统库（根据需要）
docker cp /lib/libm.so.6 plum-agent-a:/lib/          # 数学库
docker cp /lib/libdl.so.2 plum-agent-a:/lib/          # 动态链接库
docker cp /lib/libgcc_s.so.1 plum-agent-a:/lib/       # GCC运行时库
docker cp /lib/libstdc++.so.6 plum-agent-a:/lib/      # C++标准库

# 复制到/usr/lib（如果应用需要）
docker cp /usr/lib/libssl.so.1.1 plum-agent-a:/usr/lib/    # OpenSSL
docker cp /usr/lib/libcrypto.so.1.1 plum-agent-a:/usr/lib/ # OpenSSL加密库
docker cp /usr/lib/libz.so.1 plum-agent-a:/usr/lib/        # 压缩库
```

#### 方案4：重新编译ARM64版本
```bash
# 在目标ARM64环境中重新编译
# C++应用
g++ -o HelloUI-arm64 HelloUI.cpp

# Go应用
GOOS=linux GOARCH=arm64 go build -o HelloUI-arm64 main.go
```

#### 方案3：使用交叉编译（在WSL2中）
```bash
# 在WSL2中交叉编译ARM64版本
GOOS=linux GOARCH=arm64 go build -o HelloUI-arm64 main.go

# C++交叉编译
aarch64-linux-gnu-g++ -o HelloUI-arm64 HelloUI.cpp
```

#### 方案4：替换文件
```bash
# 将ARM64版本复制到容器中
docker cp HelloUI-arm64 plum-agent-a:/app/data/nodeA/.../HelloUI

# 设置执行权限
docker exec -it plum-agent-a chmod +x /app/data/nodeA/.../HelloUI
```

### 预防措施
- 确保应用在目标架构下编译
- 使用交叉编译工具链
- 验证可执行文件的架构：`file <executable>`

---

## 🔧 问题8：动态库文件复制指南

### 问题描述
在离线ARM64环境中，使用 `alpine:3.18` 基础镜像的容器缺少必要的动态链接库，导致动态链接的应用无法运行。

### 解决方案

#### 方案1：基础库文件复制
```bash
# 1. 查找基础系统库文件
echo "🔍 查找基础系统库文件..."
find /lib -name "libpthread.so*" -exec ls -la {} \;
find /lib -name "libc.so*" -exec ls -la {} \;
find /lib -name "ld-linux-aarch64.so*" -exec ls -la {} \;

# 2. 复制基础库文件到容器中
echo "📦 复制基础库文件到容器..."
docker cp /lib/libpthread.so.0 plum-agent-a:/lib/
docker cp /lib/libc.so.6 plum-agent-a:/lib/
docker cp /lib/ld-linux-aarch64.so.1 plum-agent-a:/lib/

# 3. 设置执行权限
echo "🔧 设置执行权限..."
docker exec -it plum-agent-a chmod +x /lib/ld-linux-aarch64.so.1

# 4. 验证库文件
echo "✅ 验证库文件..."
docker exec -it plum-agent-a ls -la /lib/libpthread.so.0 /lib/libc.so.6 /lib/ld-linux-aarch64.so.1
```

#### 方案2：常用库文件复制
```bash
# 复制其他常用的系统库（根据需要）
docker cp /lib/libm.so.6 plum-agent-a:/lib/          # 数学库
docker cp /lib/libdl.so.2 plum-agent-a:/lib/          # 动态链接库
docker cp /lib/libgcc_s.so.1 plum-agent-a:/lib/       # GCC运行时库
docker cp /lib/libstdc++.so.6 plum-agent-a:/lib/      # C++标准库

# 复制到/usr/lib（如果应用需要）
docker cp /usr/lib/libssl.so.1.1 plum-agent-a:/usr/lib/    # OpenSSL
docker cp /usr/lib/libcrypto.so.1.1 plum-agent-a:/usr/lib/ # OpenSSL加密库
docker cp /usr/lib/libz.so.1 plum-agent-a:/usr/lib/        # 压缩库
```

#### 方案3：智能库文件复制脚本（推荐）
```bash
# 使用智能脚本自动分析二进制文件依赖
./docker/smart-copy-libs.sh ./HelloUI

# 或者指定完整路径
./docker/smart-copy-libs.sh /app/data/nodeA/e34a5f89d14a74695f6b2a20d132ebff-16baf3cc/app/HelloUI

# 仅显示需要复制的库文件（不实际复制）
./docker/smart-copy-libs.sh -d ./HelloUI

# 复制到指定容器
./docker/smart-copy-libs.sh -c plum-agent-b ./HelloUI
```

#### 方案4：批量复制脚本
```bash
# 创建批量复制脚本
cat > copy-libs.sh << 'EOF'
#!/bin/bash

CONTAINER_NAME="plum-agent-a"

echo "🔍 查找并复制基础库文件..."

# 基础库文件列表
BASIC_LIBS=(
    "/lib/libpthread.so.0"
    "/lib/libc.so.6"
    "/lib/ld-linux-aarch64.so.1"
    "/lib/libm.so.6"
    "/lib/libdl.so.2"
    "/lib/libgcc_s.so.1"
    "/lib/libstdc++.so.6"
)

# 复制基础库文件
for lib in "${BASIC_LIBS[@]}"; do
    if [ -f "$lib" ]; then
        echo "📦 复制 $lib"
        docker cp "$lib" "$CONTAINER_NAME:/lib/"
    else
        echo "⚠️  未找到 $lib"
    fi
done

# 设置执行权限
echo "🔧 设置执行权限..."
docker exec -it "$CONTAINER_NAME" chmod +x /lib/ld-linux-aarch64.so.1

echo "✅ 基础库文件复制完成！"
EOF

chmod +x copy-libs.sh
./copy-libs.sh
```

### 常用库文件说明

#### 基础系统库
- **`libc.so.6`** - C标准库，几乎所有程序都需要
- **`libpthread.so.0`** - POSIX线程库，多线程程序需要
- **`ld-linux-aarch64.so.1`** - 动态链接器，动态链接程序的入口点

#### 数学和运行时库
- **`libm.so.6`** - 数学库，包含数学函数
- **`libdl.so.2`** - 动态链接库，用于动态加载库
- **`libgcc_s.so.1`** - GCC运行时库，C++程序需要
- **`libstdc++.so.6`** - C++标准库

#### 网络和加密库
- **`libssl.so.1.1`** - OpenSSL SSL库，HTTPS连接需要
- **`libcrypto.so.1.1`** - OpenSSL加密库，加密操作需要
- **`libz.so.1`** - 压缩库，压缩/解压缩需要

#### 图形和多媒体库
- **`libX11.so.6`** - X11图形库，GUI程序需要
- **`libGL.so.1`** - OpenGL库，3D图形需要
- **`libasound.so.2`** - ALSA音频库，音频处理需要

### 预防措施
- 根据应用需求复制相应的库文件
- 使用 `ldd` 命令检查应用的库依赖
- 创建库文件复制脚本，便于重复使用
- 定期更新库文件版本

---

## 📋 故障排除检查清单

### 1. 环境检查
- [ ] 确认目标环境架构（ARM64）
- [ ] 检查Docker和Docker Compose版本
- [ ] 验证网络连接状态

### 2. 镜像检查
- [ ] 确认镜像架构匹配：`docker inspect <image> | grep Architecture`
- [ ] 检查镜像是否存在：`docker images`
- [ ] 验证镜像完整性

### 3. 容器检查
- [ ] 查看容器状态：`docker-compose ps`
- [ ] 检查容器日志：`docker-compose logs <service>`
- [ ] 验证容器健康状态

### 4. 权限检查
- [ ] 检查数据卷权限：`docker volume inspect <volume>`
- [ ] 验证文件系统权限
- [ ] 确认用户权限设置

### 5. 网络检查
- [ ] 测试服务连通性：`curl http://localhost:8080/v1/nodes`
- [ ] 检查端口占用：`netstat -tulpn | grep :8080`
- [ ] 验证防火墙设置

---

## 🚀 快速修复命令

### 重置所有服务
```bash
# 停止所有服务
docker-compose -f docker-compose.offline.yml down

# 清理数据卷（谨慎使用）
docker volume prune -f

# 重新构建镜像
./docker/build-static-offline-fixed.sh

# 重新启动服务
docker-compose -f docker-compose.offline.yml up -d
```

### 查看服务状态
```bash
# 查看所有容器状态
docker-compose -f docker-compose.offline.yml ps

# 查看日志
docker-compose -f docker-compose.offline.yml logs -f

# 测试API
curl http://localhost:8080/v1/nodes
curl http://localhost/health
```

### 进入容器调试
```bash
# 进入Controller容器
docker exec -it plum-controller sh

# 进入Agent容器
docker exec -it plum-agent-a sh

# 进入Nginx容器
docker exec -it plum-nginx sh
```

---

## 📞 获取帮助

如果遇到本文档未涵盖的问题：

1. **查看日志**：`docker-compose logs <service>`
2. **检查状态**：`docker-compose ps`
3. **验证配置**：`docker-compose config`
4. **参考文档**：`docker/DEPLOYMENT-GUIDE.md`

---

*最后更新：2025年10月29日*
