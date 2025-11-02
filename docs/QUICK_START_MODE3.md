# 方式3：完全容器化快速开始

## 🚀 快速启动（5分钟）

### 步骤1：构建镜像

```bash
cd /home/stone/code/Plum

# 构建所有服务镜像
docker-compose build
```

### 步骤2：启动服务

```bash
# 启动所有服务（后台运行）
docker-compose up -d

# 查看服务状态
docker-compose ps
```

### 步骤3：检查日志

```bash
# 查看 Controller 日志
docker-compose logs -f plum-controller

# 查看 Agent 日志（nodeA）
docker-compose logs -f plum-agent-a
```

**关键检查**：
- Controller 应显示：`Controller running on :8080`
- Agent 应显示：`Using app run mode: docker`

### 步骤4：测试 API

```bash
# 检查 Controller 是否可访问
curl http://localhost:8080/v1/nodes

# 应该返回节点列表
```

### 步骤5：部署应用

```bash
# 1. 上传应用
curl -X POST http://localhost:8080/v1/apps/upload \
  -F "file=@/path/to/your-app.zip"

# 2. 创建部署（替换 ARTIFACT_URL）
curl -X POST http://localhost:8080/v1/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-app",
    "entries": [{
      "artifactUrl": "/artifacts/app_xxx.zip",
      "replicas": {"nodeA": 1}
    }]
  }'

# 3. 启动部署（替换 DEPLOYMENT_ID）
curl -X POST "http://localhost:8080/v1/deployments/DEPLOYMENT_ID?action=start"
```

### 步骤6：验证

```bash
# 检查应用容器
docker ps | grep plum-app-

# 查看容器日志
CONTAINER_NAME=$(docker ps | grep plum-app- | awk '{print $NF}' | head -1)
docker logs $CONTAINER_NAME
```

## ⚙️ 配置说明

### 环境变量配置

可以通过 `.env` 文件（项目根目录）覆盖默认配置：

```bash
# .env (项目根目录)
PLUM_BASE_IMAGE=ubuntu:22.04
PLUM_HOST_LIB_PATHS=/usr/lib,/usr/local/lib,/usr/lib/x86_64-linux-gnu
PLUM_CONTAINER_MEMORY=512m
PLUM_CONTAINER_CPUS=1.0
```

### docker-compose.yml 中的配置

所有配置已包含在 `docker-compose.yml` 中：
- ✅ Docker socket 挂载
- ✅ `AGENT_RUN_MODE=docker`
- ✅ 基础镜像配置（默认 `ubuntu:22.04`）
- ✅ 库路径映射支持
- ✅ 容器环境变量支持

## 📚 详细文档

- [完整测试指南](./TEST_FULLY_CONTAINERIZED.md) - 详细的测试步骤和故障排查
- [容器应用管理](./CONTAINER_APP_MANAGEMENT.md) - 架构和配置说明
- [环境变量配置](./ENV_CONFIG.md) - 所有配置项说明

## 🛑 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止并删除数据卷（谨慎使用）
docker-compose down -v
```

