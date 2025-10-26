# Plum Docker 使用示例

## 🚀 快速开始

### 使用部署脚本（推荐）

```bash
# 进入docker目录
cd docker

# 查看帮助
./deploy.sh help

# 启动测试环境
./deploy.sh test start

# 查看状态
./deploy.sh test status

# 查看日志
./deploy.sh test logs

# 停止服务
./deploy.sh test stop
```

### 手动使用Docker Compose

```bash
# 测试环境
docker-compose up -d

# 生产环境
docker-compose -f docker-compose.production.yml up -d

# 带nginx的测试环境
docker-compose --profile nginx up -d
```

## 📋 详细使用示例

### 1. 开发测试流程

```bash
# 1. 启动简单测试环境（仅Controller）
./deploy.sh test-simple start

# 2. 测试Controller API
curl http://localhost:8080/v1/nodes

# 3. 停止测试环境
./deploy.sh test-simple stop
```

### 2. 集成测试流程

```bash
# 1. 启动完整测试环境
./deploy.sh test start

# 2. 等待服务启动
sleep 10

# 3. 检查所有服务状态
./deploy.sh test status

# 4. 测试系统功能
curl http://localhost:8080/v1/nodes
curl http://localhost:8080/v1/services

# 5. 查看日志
./deploy.sh test logs
```

### 3. UI测试流程

```bash
# 1. 启动带nginx的测试环境
./deploy.sh test-nginx start

# 2. 测试Web访问
curl http://localhost/health
curl http://localhost/v1/nodes

# 3. 在浏览器中访问
# http://localhost
```

### 4. 生产部署流程

```bash
# 1. 启动生产环境
./deploy.sh production start

# 2. 检查服务状态
./deploy.sh production status

# 3. 执行健康检查
./deploy.sh production health

# 4. 监控日志
./deploy.sh production logs
```

### 5. 分布式部署流程

#### Controller节点
```bash
# 在Controller节点执行
./deploy.sh controller start

# 检查Controller状态
./deploy.sh controller status

# 测试Controller API
curl http://localhost:8080/v1/nodes
```

#### Agent节点
```bash
# 在Agent节点执行
export AGENT_NODE_ID=node1
export CONTROLLER_BASE=http://192.168.1.100:8080  # 替换为实际Controller IP
./deploy.sh agent start

# 检查Agent状态
./deploy.sh agent status

# 查看Agent日志
./deploy.sh agent logs
```

## 🔧 维护操作

### 数据备份
```bash
# 备份数据
./deploy.sh backup

# 查看备份文件
ls -la backups/
```

### 数据恢复
```bash
# 恢复数据
./deploy.sh restore backups/plum_backup_20240101_120000.tar.gz
```

### 资源清理
```bash
# 清理Docker资源
./deploy.sh clean
```

### 服务重启
```bash
# 重启测试环境
./deploy.sh test restart

# 重启生产环境
./deploy.sh production restart
```

## 🐛 故障排除

### 常见问题解决

#### 1. 网络冲突
```bash
# 清理网络
docker network prune

# 重新启动
./deploy.sh test restart
```

#### 2. 端口冲突
```bash
# 检查端口占用
netstat -tulpn | grep :8080

# 停止冲突服务
sudo systemctl stop apache2  # 或其他占用端口的服务

# 重新启动
./deploy.sh test start
```

#### 3. 内存不足
```bash
# 清理Docker资源
./deploy.sh clean

# 检查系统内存
free -h

# 增加swap空间（如果需要）
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

#### 4. 权限问题
```bash
# 检查文件权限
ls -la controller/.env

# 修复权限
chmod 644 controller/.env
chmod 644 agent-go/.env
```

### 日志分析

#### 查看详细日志
```bash
# 查看所有服务日志
./deploy.sh test logs

# 查看特定服务日志
docker-compose logs plum-controller
docker-compose logs plum-agent-a
```

#### 日志过滤
```bash
# 查看错误日志
docker-compose logs plum-controller | grep ERROR

# 查看最近100行日志
docker-compose logs --tail=100 plum-controller
```

## 📊 监控和调试

### 系统监控
```bash
# 查看Docker资源使用
docker stats

# 查看系统资源
htop
free -h
df -h
```

### 服务监控
```bash
# 健康检查
./deploy.sh test health

# 检查服务状态
./deploy.sh test status

# 查看服务详细信息
docker inspect plum-controller
```

### 网络调试
```bash
# 查看网络配置
docker network ls
docker network inspect plum_plum-network

# 测试网络连接
docker-compose exec plum-controller ping plum-agent-a
```

## 🎯 最佳实践

### 1. 环境隔离
```bash
# 开发环境
./deploy.sh test-simple start

# 测试环境
./deploy.sh test start

# 生产环境
./deploy.sh production start
```

### 2. 数据管理
```bash
# 定期备份
./deploy.sh backup

# 清理旧备份
find backups/ -name "*.tar.gz" -mtime +7 -delete
```

### 3. 资源管理
```bash
# 定期清理
./deploy.sh clean

# 监控资源使用
docker system df
```

### 4. 安全配置
```bash
# 使用非root用户运行
# 在Dockerfile中已配置

# 限制资源使用
# 在docker-compose.yml中已配置

# 定期更新镜像
docker-compose pull
docker-compose up -d
```

## 📞 获取帮助

### 查看帮助信息
```bash
# 查看脚本帮助
./deploy.sh help

# 查看Docker帮助
docker-compose --help

# 查看服务状态
./deploy.sh test status
```

### 调试模式
```bash
# 启用详细日志
export COMPOSE_LOG_LEVEL=DEBUG
./deploy.sh test start

# 查看Docker日志
journalctl -u docker.service
```

---

## 📝 总结

本指南提供了Plum Docker部署的完整使用示例，从简单的测试环境到复杂的生产环境。通过使用部署脚本，可以大大简化日常操作，提高工作效率。

记住：
- 测试环境用于功能验证
- 生产环境用于实际部署
- 定期备份重要数据
- 监控系统资源使用
- 遵循安全最佳实践
