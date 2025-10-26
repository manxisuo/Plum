# Plum Docker 快速启动指南

## 🚀 常用启动命令

### 测试环境
```bash
# 1. 单Controller测试
docker-compose up -d plum-controller

# 2. 完整测试环境（Controller + 3个Agent）
docker-compose up -d

# 3. 带Nginx的测试环境
docker-compose --profile nginx up -d
```

### 生产环境
```bash
# 1. 单节点生产部署
docker-compose -f docker-compose.production.yml up -d

# 2. 多节点部署（Controller节点）
docker-compose up -d plum-controller

# 3. 多节点部署（Controller + nginx节点）
docker-compose --profile nginx up -d plum-controller plum-nginx

# 4. 多节点部署（Agent节点）
export AGENT_NODE_ID=node1
export CONTROLLER_BASE=http://192.168.1.100:8080  # 替换为实际Controller IP
docker-compose -f docker-compose.production.yml up -d
```

## 🔧 服务管理

### 基本操作
```bash
# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f plum-controller

# 重启服务
docker-compose restart plum-controller

# 停止服务
docker-compose down
```

### 健康检查
```bash
# 检查Controller
curl http://localhost:8080/v1/nodes

# 检查nginx
curl http://localhost/health
```

## 🐛 常见问题解决

### 网络冲突
```bash
docker network prune
docker-compose down
docker-compose up -d
```

### 端口冲突
```bash
netstat -tulpn | grep :8080
docker-compose down
docker-compose up -d
```

### 内存不足
```bash
docker system prune
# 或增加系统内存
```

## 📊 服务端口

| 服务 | 端口 | 用途 |
|------|------|------|
| Controller | 8080 | API接口 |
| nginx | 80/443 | Web UI和反向代理 |
| Agent | 内部 | 与Controller通信 |

## 🎯 选择部署方式

| 场景 | 推荐命令 | 说明 |
|------|----------|------|
| 功能测试 | `docker-compose up -d plum-controller` | 只启动Controller |
| 集成测试 | `docker-compose up -d` | Controller + 3个Agent |
| UI测试 | `docker-compose --profile nginx up -d` | 包含Web界面 |
| 生产部署 | `docker-compose -f docker-compose.production.yml up -d` | 生产级配置 |
| Controller节点 | `docker-compose up -d plum-controller` | 只启动Controller |
| Controller+nginx节点 | `docker-compose --profile nginx up -d plum-controller plum-nginx` | Controller + nginx |
| Agent节点 | `docker-compose -f docker-compose.production.yml up -d` | 只启动Agent |

## 💡 小贴士

- 首次启动可能需要下载镜像，请耐心等待
- 使用 `docker-compose logs -f` 查看实时日志
- 生产环境建议设置资源限制
- 定期备份数据卷
- 使用 `docker system prune` 清理无用资源
