# Plum Controller 测试环境

## 🎯 概述

这个测试环境专门用于单独测试Plum Controller的功能，不包含Agent和其他组件。

## 📁 文件说明

- `docker-compose.controller-test.yml` - Controller测试环境的Docker Compose配置
- `docker/test-controller.sh` - Controller测试脚本
- `docker/README-controller-test.md` - 本文档

## 🚀 快速开始

### 1. 启动Controller测试服务

```bash
# 使用测试脚本（推荐）
./docker/test-controller.sh start

# 或使用Docker Compose
docker-compose -f docker-compose.controller-test.yml up -d
```

### 2. 检查服务状态

```bash
# 查看服务状态
./docker/test-controller.sh status

# 或使用Docker Compose
docker-compose -f docker-compose.controller-test.yml ps
```

### 3. 运行健康检查

```bash
# 运行完整健康检查
./docker/test-controller.sh test

# 手动检查健康接口
curl http://localhost:8080/v1/health
```

### 4. 查看日志

```bash
# 查看实时日志
./docker/test-controller.sh logs

# 或使用Docker Compose
docker-compose -f docker-compose.controller-test.yml logs -f
```

## 🔧 测试脚本使用

### 基本命令

```bash
# 启动服务
./docker/test-controller.sh start

# 停止服务
./docker/test-controller.sh stop

# 重启服务
./docker/test-controller.sh restart

# 查看状态
./docker/test-controller.sh status

# 查看日志
./docker/test-controller.sh logs

# 运行健康检查
./docker/test-controller.sh test

# 进入容器shell
./docker/test-controller.sh shell

# 清理测试数据
./docker/test-controller.sh clean

# 显示帮助
./docker/test-controller.sh -h
```

## 📊 服务配置

### 端口映射
- **宿主机端口**: 8080
- **容器端口**: 8080
- **访问地址**: http://localhost:8080

### 数据持久化
- **测试数据目录**: `./test-data/`
- **数据库文件**: `./test-data/controller-test.db`
- **配置文件**: `./controller/.env`

### 环境变量
```bash
CONTROLLER_ADDR=:8080
CONTROLLER_DB=file:/app/data/controller-test.db
CONTROLLER_DATA_DIR=/app/data
HEARTBEAT_TTL_SEC=30
FAILOVER_ENABLED=true
```

## 🏥 健康检查

### 自动健康检查
- **检查间隔**: 30秒
- **超时时间**: 10秒
- **重试次数**: 3次
- **启动延迟**: 10秒

### 手动健康检查
```bash
# 检查健康接口
curl http://localhost:8080/v1/health

# 检查节点接口
curl http://localhost:8080/v1/nodes

# 检查服务接口
curl http://localhost:8080/v1/services
```

## 🔍 测试场景

### 1. 基础功能测试
```bash
# 启动服务
./docker/test-controller.sh start

# 等待服务启动
sleep 10

# 运行健康检查
./docker/test-controller.sh test
```

### 2. API接口测试
```bash
# 测试节点管理
curl -X GET http://localhost:8080/v1/nodes

# 测试服务发现
curl -X GET http://localhost:8080/v1/services

# 测试任务定义
curl -X GET http://localhost:8080/v1/task-defs
```

### 3. 数据库测试
```bash
# 进入容器
./docker/test-controller.sh shell

# 查看数据库文件
ls -la /app/data/

# 检查数据库内容
sqlite3 /app/data/controller-test.db ".tables"
```

## 🐛 故障排除

### 常见问题

#### 1. 端口冲突
```bash
# 检查端口占用
netstat -tlnp | grep :8080

# 修改端口映射
# 编辑 docker-compose.controller-test.yml
# 将 "8080:8080" 改为 "8081:8080"
```

#### 2. 服务启动失败
```bash
# 查看详细日志
./docker/test-controller.sh logs

# 检查容器状态
docker ps -a | grep plum-controller-test

# 检查镜像是否存在
docker images | grep plum-controller
```

#### 3. 健康检查失败
```bash
# 检查服务是否完全启动
sleep 30

# 手动测试健康接口
curl -v http://localhost:8080/v1/health

# 检查容器内部
./docker/test-controller.sh shell
```

### 调试模式

```bash
# 进入容器调试
./docker/test-controller.sh shell

# 查看进程
ps aux

# 查看端口监听
netstat -tlnp

# 查看环境变量
env | grep CONTROLLER
```

## 📈 性能测试

### 资源监控
```bash
# 查看容器资源使用
docker stats plum-controller-test

# 查看系统资源
top
htop
```

### 压力测试
```bash
# 使用curl进行简单压力测试
for i in {1..100}; do
  curl -s http://localhost:8080/v1/health > /dev/null
done
```

## 🧹 清理和维护

### 清理测试数据
```bash
# 清理所有测试数据
./docker/test-controller.sh clean

# 手动清理
docker-compose -f docker-compose.controller-test.yml down
rm -rf ./test-data/
docker system prune -f
```

### 更新镜像
```bash
# 重新构建镜像
docker build -f docker/controller/Dockerfile -t plum-controller:latest .

# 重启服务
./docker/test-controller.sh restart
```

## 🎯 最佳实践

### 1. 测试流程
1. 启动服务
2. 等待服务完全启动
3. 运行健康检查
4. 执行功能测试
5. 清理测试数据

### 2. 日志管理
- 定期查看日志
- 保存重要日志
- 监控错误信息

### 3. 数据管理
- 定期备份测试数据
- 清理过期数据
- 监控磁盘使用

---

**Plum Controller测试环境** - 让测试更简单，让开发更高效！ 🧪
