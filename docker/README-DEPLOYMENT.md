# Plum Docker 部署指南

## 📋 目录
- [概述](#概述)
- [环境准备](#环境准备)
- [测试环境部署](#测试环境部署)
- [生产环境部署](#生产环境部署)
- [服务管理](#服务管理)
- [故障排除](#故障排除)
- [最佳实践](#最佳实践)

## 🎯 概述

Plum支持多种Docker部署方式，适用于不同的使用场景：

- **测试环境**: 单机部署，快速验证功能
- **生产环境**: 分布式部署，高可用性
- **开发环境**: 本地开发调试

## 🛠 环境准备

### 系统要求
- Docker Engine 20.10+
- Docker Compose 2.0+
- 内存: 至少2GB
- 磁盘: 至少5GB可用空间

### 检查环境
```bash
# 检查Docker版本
docker --version
docker-compose --version

# 检查系统资源
docker system df
docker system info
```

## 🧪 测试环境部署

### 1. 单Controller测试

**用途**: 快速验证Controller功能

```bash
# 启动Controller
docker-compose -f docker-compose.controller-test-simple.yml up -d

# 检查状态
docker-compose -f docker-compose.controller-test-simple.yml ps

# 测试API
curl http://localhost:8080/v1/nodes

# 停止服务
docker-compose -f docker-compose.controller-test-simple.yml down
```

**特点**:
- 只启动Controller
- 使用命名卷存储数据
- 端口映射: 8080
- 适合功能验证

### 2. 完整测试环境

**用途**: 测试完整系统（Controller + 3个Agent）

```bash
# 启动所有服务
docker-compose up -d

# 检查服务状态
docker-compose ps

# 测试系统
curl http://localhost:8080/v1/nodes
curl http://localhost:8080/v1/services

# 停止服务
docker-compose down
```

**特点**:
- Controller + 3个Agent
- 自动健康检查
- 资源限制配置
- 适合集成测试

### 3. 带Nginx的测试环境

**用途**: 测试Web UI和反向代理

```bash
# 启动包含nginx的完整环境
docker-compose --profile nginx up -d

# 检查所有服务
docker-compose ps

# 测试Web访问
curl http://localhost/health
curl http://localhost/v1/nodes

# 停止服务
docker-compose --profile nginx down
```

**特点**:
- 包含nginx反向代理
- 端口80/443映射
- 静态文件服务
- 适合UI测试

## 🏭 生产环境部署

### 1. 单节点生产部署

**用途**: 单机生产环境

```bash
# 使用生产配置启动
docker-compose -f docker-compose.production.yml up -d

# 检查服务状态
docker-compose -f docker-compose.production.yml ps

# 查看日志
docker-compose -f docker-compose.production.yml logs -f
```

**特点**:
- 单Agent配置
- 生产级资源限制
- 持久化数据存储
- 自动重启策略

### 2. 多节点分布式部署

**用途**: 大规模分布式环境

#### Controller节点部署
```bash
# 在Controller节点执行
docker-compose -f docker-compose.yml up -d plum-controller

# 检查Controller状态
docker-compose ps plum-controller

# 测试Controller API
curl http://localhost:8080/v1/nodes
```

#### Agent节点部署
```bash
# 在Agent节点执行（修改node_id）
export AGENT_NODE_ID=node1
docker-compose -f docker-compose.yml up -d plum-agent-a

# 检查Agent状态
docker-compose ps plum-agent-a

# 查看Agent日志
docker-compose logs plum-agent-a
```

**特点**:
- 分布式部署
- 节点间通信
- 故障转移支持
- 负载均衡

### 3. 高可用部署

**用途**: 企业级高可用环境

```bash
# 使用Docker Swarm模式
docker swarm init
docker stack deploy -c docker-compose.swarm.yml plum

# 检查服务状态
docker service ls
docker service ps plum_controller
```

**特点**:
- 服务自动重启
- 负载均衡
- 滚动更新
- 故障恢复

## 🔧 服务管理

### 常用命令

#### 启动服务
```bash
# 启动所有服务
docker-compose up -d

# 启动特定服务
docker-compose up -d plum-controller

# 启动带profile的服务
docker-compose --profile nginx up -d
```

#### 停止服务
```bash
# 停止所有服务
docker-compose down

# 停止特定服务
docker-compose stop plum-controller

# 强制停止并删除卷
docker-compose down -v
```

#### 重启服务
```bash
# 重启所有服务
docker-compose restart

# 重启特定服务
docker-compose restart plum-controller

# 重新构建并启动
docker-compose up -d --build
```

#### 查看状态
```bash
# 查看服务状态
docker-compose ps

# 查看服务日志
docker-compose logs -f plum-controller

# 查看资源使用
docker stats
```

### 数据管理

#### 备份数据
```bash
# 备份Controller数据
docker run --rm -v plum_plum-data:/data -v $(pwd):/backup alpine tar czf /backup/plum-data-backup.tar.gz -C /data .

# 备份特定服务数据
docker cp plum-controller:/app/data ./controller-backup
```

#### 恢复数据
```bash
# 恢复Controller数据
docker run --rm -v plum_plum-data:/data -v $(pwd):/backup alpine tar xzf /backup/plum-data-backup.tar.gz -C /data
```

## 🔍 故障排除

### 常见问题

#### 1. 网络冲突
```bash
# 错误: Pool overlaps with other one on this address space
# 解决: 检查并清理网络
docker network ls
docker network prune
docker-compose down
docker-compose up -d
```

#### 2. 端口冲突
```bash
# 错误: Port already in use
# 解决: 检查端口占用
netstat -tulpn | grep :8080
docker-compose down
docker-compose up -d
```

#### 3. 权限问题
```bash
# 错误: Permission denied
# 解决: 检查文件权限
ls -la ./controller/.env
chmod 644 ./controller/.env
```

#### 4. 内存不足
```bash
# 错误: Out of memory
# 解决: 检查系统资源
docker system df
docker system prune
# 或增加系统内存
```

### 日志分析

#### 查看详细日志
```bash
# 查看所有服务日志
docker-compose logs

# 查看特定服务日志
docker-compose logs plum-controller

# 实时查看日志
docker-compose logs -f plum-controller

# 查看最近100行日志
docker-compose logs --tail=100 plum-controller
```

#### 健康检查
```bash
# 检查Controller健康状态
curl http://localhost:8080/v1/nodes

# 检查Agent状态
docker-compose exec plum-agent-a pgrep plum-agent

# 检查nginx状态
curl http://localhost/health
```

## 🎯 最佳实践

### 1. 环境配置

#### 开发环境
```bash
# 使用测试配置
docker-compose -f docker-compose.controller-test-simple.yml up -d
```

#### 测试环境
```bash
# 使用完整测试配置
docker-compose up -d
```

#### 生产环境
```bash
# 使用生产配置
docker-compose -f docker-compose.production.yml up -d
```

### 2. 资源管理

#### 设置资源限制
```yaml
# 在docker-compose.yml中配置
deploy:
  resources:
    limits:
      cpus: '1.0'
      memory: 512M
    reservations:
      cpus: '0.5'
      memory: 256M
```

#### 监控资源使用
```bash
# 实时监控
docker stats

# 查看资源使用历史
docker system df
```

### 3. 安全配置

#### 使用非root用户
```dockerfile
# 在Dockerfile中配置
RUN addgroup -g 1001 -S plum && \
    adduser -u 1001 -S plum -G plum
USER plum
```

#### 网络安全
```yaml
# 在docker-compose.yml中配置
networks:
  plum-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.25.0.0/16
```

### 4. 数据持久化

#### 使用命名卷
```yaml
# 推荐配置
volumes:
  plum-data:
    driver: local
```

#### 定期备份
```bash
# 创建备份脚本
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
docker run --rm -v plum_plum-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/plum-backup-$DATE.tar.gz -C /data .
```

## 📞 技术支持

### 获取帮助
```bash
# 查看Docker版本信息
docker version

# 查看Compose版本信息
docker-compose version

# 查看系统信息
docker system info
```

### 常用调试命令
```bash
# 进入容器调试
docker-compose exec plum-controller sh

# 查看容器详细信息
docker inspect plum-controller

# 查看网络配置
docker network inspect plum_plum-network
```

---

## 📝 总结

本指南涵盖了Plum Docker部署的各个方面，从简单的测试环境到复杂的生产环境。根据您的具体需求选择合适的部署方式，并遵循最佳实践确保系统的稳定性和安全性。

如有问题，请参考故障排除部分或联系技术支持。
