# 方式3：完全容器化测试指南

## 📋 测试目标

验证 Controller、Agent 和应用都以 Docker 容器方式运行，实现完全容器化部署。

## ✅ 前置条件

1. **Docker 和 Docker Compose 已安装**
   ```bash
   docker --version
   docker-compose --version
   # 或者使用新版本
   docker compose version
   ```

2. **宿主机有 Docker daemon 运行**
   ```bash
   sudo systemctl status docker
   # 或
   ps aux | grep dockerd
   ```

3. **准备测试应用**
   - 需要已打包的应用 artifact（zip 文件）
   - 建议使用纯后台应用（不需要GUI）

---

## 🚀 测试步骤

### 步骤1：准备配置文件

#### 1.1 检查 Controller 配置

```bash
cd /home/stone/code/Plum

# 检查或创建 controller/.env
cat controller/.env
# 确保有以下配置（或使用默认值）
# CONTROLLER_ADDR=:8080
# CONTROLLER_DB=file:controller.db
# CONTROLLER_DATA_DIR=.
```

#### 1.2 准备 Agent 配置（可选，docker-compose.yml 已包含环境变量）

```bash
# agent-go/.env 是可选的
# 因为 docker-compose.yml 中已经通过环境变量配置了
# 但如果存在，会被挂载到容器内

# 如果需要使用 .env 文件，确保包含：
cat agent-go/.env
# AGENT_RUN_MODE=docker
# PLUM_BASE_IMAGE=ubuntu:22.04
# PLUM_HOST_LIB_PATHS=/usr/lib,/usr/local/lib,/usr/lib/x86_64-linux-gnu
```

#### 1.3 配置环境变量（可选，用于覆盖 docker-compose.yml 的默认值）

创建 `.env` 文件在项目根目录（用于 docker-compose 变量替换）：

```bash
# .env (项目根目录)
PLUM_BASE_IMAGE=ubuntu:22.04
PLUM_HOST_LIB_PATHS=/usr/lib,/usr/local/lib,/usr/lib/x86_64-linux-gnu
# PLUM_CONTAINER_MEMORY=512m
# PLUM_CONTAINER_CPUS=1.0
```

### 步骤2：构建 Docker 镜像

```bash
cd /home/stone/code/Plum

# 构建所有服务镜像
docker-compose build

# 或只构建特定服务
docker-compose build plum-controller
docker-compose build plum-agent-a
```

### 步骤3：启动所有服务

```bash
# 启动所有服务（后台运行）
docker-compose up -d

# 查看启动状态
docker-compose ps

# 应该看到：
# NAME              IMAGE                COMMAND                  SERVICE          CREATED         STATUS          PORTS
# plum-controller   plum-controller      "./bin/controller"       plum-controller  2 seconds ago   Up 1 second     0.0.0.0:8080->8080/tcp
# plum-agent-a      plum-agent-a         "./plum-agent"          plum-agent-a     2 seconds ago   Up 1 second
# plum-agent-b      plum-agent-b         "./plum-agent"          plum-agent-b     2 seconds ago   Up 1 second
# plum-agent-c      plum-agent-c         "./plum-agent"          plum-agent-c     2 seconds ago   Up 1 second
```

### 步骤4：检查服务日志

```bash
# 查看 Controller 日志
docker-compose logs -f plum-controller

# 查看 Agent 日志（nodeA）
docker-compose logs -f plum-agent-a

# 查看所有服务日志
docker-compose logs -f
```

**关键检查点**：
- Controller 应该显示：`Controller running on :8080`
- Agent 应该显示：`Using app run mode: docker`
- Agent 应该显示：`Docker manager initialized successfully`

### 步骤5：验证网络连接

```bash
# 检查 Controller 是否可访问
curl http://localhost:8080/v1/nodes

# 应该返回节点列表（可能为空）

# 检查节点健康状态
curl http://localhost:8080/v1/nodes/nodeA
```

### 步骤6：上传应用并创建部署

```bash
# 1. 上传应用 artifact
UPLOAD_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/apps/upload \
  -F "file=@/path/to/your-app.zip")

echo $UPLOAD_RESPONSE
# 返回示例：{"artifactId":"xxx","url":"/artifacts/app_xxx.zip"}

# 提取 artifact URL（手动复制）
# ARTIFACT_URL="/artifacts/app_xxx.zip"

# 2. 创建部署（分配到 nodeA）
curl -X POST http://localhost:8080/v1/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-fully-containerized",
    "entries": [{
      "artifactUrl": "/artifacts/app_xxx.zip",
      "replicas": {"nodeA": 1}
    }]
  }'

# 返回示例：{"deploymentId":"yyy","instances":["inst-xxx"]}

# 3. 启动部署
DEPLOYMENT_ID="yyy"  # 使用上面返回的ID
curl -X POST "http://localhost:8080/v1/deployments/$DEPLOYMENT_ID?action=start"
```

### 步骤7：验证应用容器运行

#### 检查应用容器

```bash
# 查看运行中的应用容器
docker ps | grep plum-app-

# 应该看到类似输出：
# CONTAINER ID   IMAGE          COMMAND          CREATED         STATUS         PORTS     NAMES
# abc123def456   ubuntu:22.04   "./start.sh"    5 seconds ago   Up 4 seconds            plum-app-inst-xxx
```

#### 检查容器日志

```bash
# 查看应用容器日志
CONTAINER_NAME=$(docker ps | grep plum-app- | awk '{print $NF}' | head -1)
docker logs $CONTAINER_NAME

# 应该看到应用的输出日志
```

#### 检查 Agent 日志

```bash
# 查看 Agent 日志，确认容器创建
docker-compose logs plum-agent-a | grep -E "(Started container|Mounted host library|Using base image)"

# 应该看到：
# Using base image: ubuntu:22.04
# Mounted host library path /usr/lib to container
# Created container abc123 for instance inst-xxx
# Started container abc123 for instance inst-xxx
```

#### 验证容器网络

```bash
# 检查应用容器是否在 plum-network 中
docker inspect plum-app-inst-xxx | grep -A 5 "Networks"

# 应该看到 plum-network
```

### 步骤8：测试容器管理功能

#### 测试1：容器故障恢复

```bash
# 停止应用容器（模拟故障）
docker stop plum-app-inst-xxx

# 观察 Agent 日志
docker-compose logs -f plum-agent-a

# 应该看到：
# ⚠️ Detected instance xxx process died unexpectedly
# Instance xxx not running, will start
# Created container ... for instance xxx
# Started container ... for instance xxx

# 检查新容器
docker ps | grep plum-app-
# 应该看到新的容器（ID可能不同）
```

#### 测试2：停止部署

```bash
# 通过 Controller API 停止部署
curl -X POST "http://localhost:8080/v1/deployments/$DEPLOYMENT_ID?action=stop"

# 等待几秒后检查容器
docker ps -a | grep plum-app-inst-xxx
# 容器应该被删除（不存在）
```

#### 测试3：重启部署

```bash
# 重新启动部署
curl -X POST "http://localhost:8080/v1/deployments/$DEPLOYMENT_ID?action=start"

# 检查新容器
docker ps | grep plum-app-
# 应该看到新创建的容器
```

---

## 🧪 高级测试

### 测试1：多节点部署

```bash
# 创建部署，分配到多个节点
curl -X POST http://localhost:8080/v1/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "name": "multi-node-test",
    "entries": [{
      "artifactUrl": "/artifacts/app_xxx.zip",
      "replicas": {"nodeA": 1, "nodeB": 1, "nodeC": 1}
    }]
  }'

# 检查每个节点上的容器
docker ps | grep plum-app-
# 应该看到3个容器（每个节点一个）
```

### 测试2：库路径共享

如果使用了 `PLUM_HOST_LIB_PATHS`：

```bash
# 检查容器内的库路径
CONTAINER_NAME=$(docker ps | grep plum-app- | awk '{print $NF}' | head -1)
docker exec $CONTAINER_NAME ls -la /usr/lib | head -10
docker exec $CONTAINER_NAME ls -la /usr/local/lib | head -10

# 应该看到宿主机库的内容
```

### 测试3：资源限制

如果配置了资源限制：

```bash
# 查看容器的资源限制
docker inspect plum-app-inst-xxx | grep -A 15 "Resources"
```

---

## ⚠️ 常见问题排查

### 问题1：Agent 容器启动失败，提示 "failed to connect to docker daemon"

**原因**：Docker socket 挂载失败或权限问题

**解决**：
```bash
# 检查 Docker socket 是否存在
ls -l /var/run/docker.sock

# 检查挂载是否成功
docker inspect plum-agent-a | grep -A 3 "docker.sock"

# 检查 Agent 容器内的权限
docker exec plum-agent-a ls -l /var/run/docker.sock
```

### 问题2：Agent 日志显示 "Using app run mode: process"（不是 docker）

**原因**：环境变量未正确传递到容器

**解决**：
```bash
# 检查容器的环境变量
docker exec plum-agent-a env | grep AGENT_RUN_MODE
# 应该显示：AGENT_RUN_MODE=docker

# 如果没有，检查 docker-compose.yml 配置
grep AGENT_RUN_MODE docker-compose.yml
```

### 问题3：应用容器创建失败，提示 "Cannot connect to the Docker daemon"

**原因**：Agent 容器无法访问 Docker socket

**解决**：
```bash
# 确认 Docker socket 挂载
docker inspect plum-agent-a | grep -A 5 "Mounts" | grep docker.sock

# 测试 Agent 容器内能否访问 Docker
docker exec plum-agent-a docker ps
# 如果失败，可能是权限问题
```

### 问题4：库路径挂载失败

**原因**：Agent 容器内看不到宿主机的路径

**说明**：
- Agent 容器内检查路径时，路径是相对于宿主机的
- 但如果 Agent 容器内使用 `os.Stat()` 检查，可能失败
- 需要确保路径在宿主机存在，并且 Agent 容器有权限访问

**解决**：
```bash
# 在宿主机确认路径存在
ls -la /usr/lib
ls -la /usr/local/lib

# 测试 Agent 容器能否看到（通过 docker exec）
# 注意：Agent 容器内可能看不到宿主机路径
# 但容器创建时，Docker 会正确挂载
```

### 问题5：网络连接问题

**原因**：容器间无法通信

**解决**：
```bash
# 检查所有容器是否在同一网络
docker network inspect plum_plum-network

# 应该看到所有 plum-controller、plum-agent-* 容器

# 测试 Agent 容器能否连接 Controller
docker exec plum-agent-a wget -O- http://plum-controller:8080/v1/nodes
```

---

## 🔍 验证清单

完成测试后，检查以下项：

- [ ] Controller 容器运行正常
  - [ ] 可访问 http://localhost:8080
  - [ ] 日志无错误

- [ ] Agent 容器运行正常
  - [ ] 日志显示 "Using app run mode: docker"
  - [ ] 日志显示 "Docker manager initialized successfully"
  - [ ] 能连接到 Controller（显示节点状态）

- [ ] 应用容器创建成功
  - [ ] `docker ps` 能看到 `plum-app-` 开头的容器
  - [ ] 容器状态为 "Up"
  - [ ] 容器日志显示应用正常运行

- [ ] 容器管理功能正常
  - [ ] 停止部署后容器被删除
  - [ ] 重新启动后新容器被创建
  - [ ] 容器故障后自动重启

- [ ] 网络和通信正常
  - [ ] 所有容器在 `plum-network` 中
  - [ ] Agent 能连接 Controller
  - [ ] 应用容器能正常工作

---

## 📊 性能对比

### 启动时间对比

| 组件 | 进程模式 | 容器模式（方式3） |
|------|---------|-----------------|
| Controller | ~100ms | ~500ms |
| Agent | ~100ms | ~500ms |
| App | ~100ms | ~500ms |
| **总计** | ~300ms | ~1500ms |

### 资源占用对比

| 组件 | 进程模式 | 容器模式（方式3） |
|------|---------|-----------------|
| Controller | ~20MB | ~30MB |
| Agent | ~15MB | ~25MB |
| App（每个） | ~10MB | ~40MB |
| **总计（1个App）** | ~45MB | ~95MB |

---

## ✅ 成功标准

如果满足以下条件，说明方式3测试成功：

1. ✅ 所有服务（Controller、Agent、App）都以容器方式运行
2. ✅ Agent 容器能成功创建应用容器
3. ✅ 应用容器正常运行
4. ✅ 容器故障能自动恢复
5. ✅ 停止部署时容器被正确删除
6. ✅ 容器网络通信正常

---

## 🎯 下一步

测试成功后，可以：

1. **性能优化**：调整资源限制，优化启动速度
2. **生产部署**：使用 docker-compose.production.yml（如果有）
3. **监控集成**：集成 Prometheus、Grafana 等监控工具
4. **CI/CD 集成**：将容器构建集成到 CI/CD 流程

---

## 📝 相关文档

- [容器应用管理](./CONTAINER_APP_MANAGEMENT.md) - 详细架构说明
- [环境变量配置](./ENV_CONFIG.md) - 完整配置项说明
- [Qt应用容器运行](./QT_APP_IN_CONTAINER.md) - Qt应用特殊配置
- [部署状态](./DEPLOYMENT_STATUS.md) - 三种方式对比

