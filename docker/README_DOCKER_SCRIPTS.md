# Docker 启动脚本使用指南

本目录提供了不使用 `docker-compose` 的启动脚本，适用于只有 Docker 的环境。

## 📋 脚本说明

### 1. `start-controller.sh` - 启动 Controller 和 Nginx

等价于 `docker-compose.main.yml`，启动以下服务：
- **plum-controller**: Plum Controller 服务
- **plum-nginx**: Nginx 反向代理和静态文件服务

### 2. `start-agent.sh` - 启动 Agent

等价于 `docker-compose.agent.yml`，启动以下服务：
- **plum-agent**: Plum Agent 服务

## 🚀 使用方法

### 启动 Controller 和 Nginx

```bash
cd /path/to/Plum
./docker/start-controller.sh
```

### 启动 Agent

```bash
cd /path/to/Plum
./docker/start-agent.sh
```

## 📝 前置条件

### 1. 构建镜像

在运行脚本之前，需要先构建 Docker 镜像：

```bash
# 构建 Controller 镜像
./docker/build-static-offline.sh controller

# 构建 Agent 镜像
./docker/build-static-offline.sh agent
```

### 2. 配置文件

- **Controller**: 需要 `controller/.env` 文件（可选，会使用默认值）
- **Agent**: 需要 `agent-go/.env` 文件（可选，会使用默认值）
- **Nginx**: 使用 `docker/nginx/nginx.conf.host`（适用于 host 网络模式）

### 3. UI 静态文件

确保已构建前端 UI：

```bash
cd ui
npm install
npm run build
```

生成的 `ui/dist` 目录将被挂载到 Nginx 容器中。

## 🔧 配置说明

### 网络模式

所有容器都使用 **host 网络模式**，这意味着：
- 容器直接使用宿主机的网络栈
- 容器内的端口就是宿主机的端口
- 容器之间可以通过 `localhost` 访问

### 数据卷

脚本会自动创建以下 Docker 数据卷：
- `plum-controller-data`: Controller 数据存储
- `plum-agent-data`: Agent 数据存储

### 端口

- **Controller**: `8080`
- **Nginx**: `80`
- **Agent**: 无对外端口（通过 Controller API 通信）

## 📊 管理命令

### 查看日志

```bash
# Controller 日志
docker logs -f plum-controller

# Nginx 日志
docker logs -f plum-nginx

# Agent 日志
docker logs -f plum-agent
```

### 停止服务

```bash
# 停止 Controller 和 Nginx
docker stop plum-controller plum-nginx
docker rm plum-controller plum-nginx

# 停止 Agent
docker stop plum-agent
docker rm plum-agent
```

### 重启服务

```bash
# 重启 Controller
docker restart plum-controller

# 重启 Nginx
docker restart plum-nginx

# 重启 Agent
docker restart plum-agent
```

### 查看容器状态

```bash
docker ps | grep plum
```

## 🔍 故障排查

### 1. 容器启动失败

检查日志：
```bash
docker logs plum-controller
docker logs plum-nginx
docker logs plum-agent
```

### 2. 端口冲突

如果端口被占用，需要：
- 停止占用端口的服务
- 或修改 `.env` 文件中的端口配置

### 3. 镜像不存在

确保已构建镜像：
```bash
docker images | grep plum
```

如果没有，运行构建脚本：
```bash
./docker/build-static-offline.sh controller
./docker/build-static-offline.sh agent
```

### 4. 权限问题

Agent 需要访问 Docker socket，确保：
- Docker socket 存在：`/var/run/docker.sock`
- 容器以 root 用户运行（`--user "0"`）

## 📚 与 docker-compose 的对应关系

| docker-compose 命令 | 等价脚本 |
|-------------------|---------|
| `docker-compose -f docker-compose.main.yml up -d` | `./docker/start-controller.sh` |
| `docker-compose -f docker-compose.agent.yml up -d` | `./docker/start-agent.sh` |
| `docker-compose -f docker-compose.main.yml down` | `docker stop plum-controller plum-nginx && docker rm plum-controller plum-nginx` |
| `docker-compose -f docker-compose.agent.yml down` | `docker stop plum-agent && docker rm plum-agent` |
| `docker-compose -f docker-compose.main.yml logs -f` | `docker logs -f plum-controller plum-nginx` |
| `docker-compose -f docker-compose.agent.yml logs -f` | `docker logs -f plum-agent` |

## ⚠️ 注意事项

1. **数据持久化**: 数据存储在 Docker 数据卷中，删除容器不会删除数据
2. **网络模式**: 使用 host 网络模式，容器端口不能冲突
3. **自动重启**: 容器配置了 `--restart unless-stopped`，系统重启后会自动启动
4. **健康检查**: 容器配置了健康检查，Docker 会自动监控容器状态

