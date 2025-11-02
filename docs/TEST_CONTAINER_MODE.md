# 测试容器模式应用管理

本文档包含两种容器模式的测试指南：
- **方式2：混合容器模式**（Controller/Agent直接运行，App容器运行）
- **方式3：完全容器化**（Controller/Agent/App都容器运行）

---

## 方式2：混合容器模式测试指南

### 📋 测试目标

验证在 Controller 和 Agent 直接运行的情况下，应用能够以 Docker 容器方式运行。

### ✅ 前置条件

1. **Docker 已安装并运行**
   ```bash
   # 检查 Docker 是否安装
   docker --version
   # 应该显示类似：Docker version 24.x.x
   
   # 检查 Docker 服务是否运行
   sudo systemctl status docker
   # 或者
   ps aux | grep dockerd
   ```

2. **Agent 用户有权限访问 Docker**
   ```bash
   # 检查当前用户是否在 docker 组
   groups | grep docker
   
   # 如果不在，需要添加（需要重新登录生效）
   sudo usermod -aG docker $USER
   # 重新登录或使用：
   newgrp docker
   
   # 测试 Docker 权限
   docker ps
   # 应该能正常执行，不报权限错误
   ```

3. **准备测试应用**
   - 需要一个已打包的应用 artifact（zip 文件）
   - 或者使用现有的 demo-app

---

## 🚀 测试步骤

### 步骤1：构建 Agent

```bash
cd /home/stone/code/Plum

# 构建 Agent（确保包含最新的容器管理代码）
make agent

# 验证构建成功
ls -lh agent-go/plum-agent
```

### 步骤2：配置 Agent 使用容器模式

有两种方式配置：

#### 方式A：使用环境变量（推荐，临时测试）

```bash
# 在启动 Agent 时直接设置环境变量
AGENT_RUN_MODE=docker \
AGENT_NODE_ID=nodeA \
CONTROLLER_BASE=http://127.0.0.1:8080 \
AGENT_DATA_DIR=/tmp/plum-agent \
./agent-go/plum-agent
```

#### 方式B：使用 .env 文件（持久化配置）

```bash
cd /home/stone/code/Plum/agent-go

# 复制配置文件
cp env.example .env

# 编辑配置文件
vim .env
# 或
nano .env
```

在 `.env` 文件中设置：

```bash
# 节点配置
AGENT_NODE_ID=nodeA
CONTROLLER_BASE=http://127.0.0.1:8080
AGENT_DATA_DIR=/tmp/plum-agent

# 应用运行模式 - 关键配置！
AGENT_RUN_MODE=docker

# 容器模式配置（可选，有默认值）
PLUM_BASE_IMAGE=alpine:latest
# PLUM_CONTAINER_MEMORY=512m  # 可选：内存限制
# PLUM_CONTAINER_CPUS=1.0     # 可选：CPU限制
```

### 步骤3：确保基础镜像已拉取

```bash
# 拉取基础镜像（如果还没有）
docker pull alpine:latest

# 验证镜像存在
docker images | grep alpine
```

### 步骤4：启动 Controller

```bash
# 终端1：启动 Controller
cd /home/stone/code/Plum
make controller
make controller-run
```

应该看到类似输出：
```
Starting Controller...
Controller running on :8080
```

### 步骤5：启动 Agent（容器模式）

#### 如果使用环境变量方式（方式A）：

```bash
# 终端2：启动 Agent（容器模式）
cd /home/stone/code/Plum

AGENT_RUN_MODE=docker \
CONTROLLER_BASE=http://127.0.0.1:8080 \
./agent-go/plum-agent
```

#### 如果使用 .env 文件方式（方式B）：

```bash
# 终端2：启动 Agent
cd /home/stone/code/Plum
./agent-go/plum-agent
```

**关键检查点**：Agent 启动日志应该显示：
```
Using app run mode: docker
```

如果看到这个日志，说明容器模式已启用。

### 步骤6：准备测试应用

如果你有现成的应用 artifact，可以跳过此步。否则创建一个简单的测试应用：

```bash
# 创建一个简单的测试应用
mkdir -p /tmp/test-app
cat > /tmp/test-app/start.sh << 'EOF'
#!/bin/sh
echo "Test app started with PLUM_INSTANCE_ID=$PLUM_INSTANCE_ID"
echo "Running in container mode"
while true; do
    echo "$(date): App is running..."
    sleep 10
done
EOF

chmod +x /tmp/test-app/start.sh

# 打包
cd /tmp/test-app
zip -r test-app.zip .
```

### 步骤7：上传应用并创建部署

#### 通过 Web UI（如果有）：
1. 访问 http://localhost:5173（或你的 UI 地址）
2. 上传应用 artifact
3. 创建部署并分配到 nodeA
4. 启动部署

#### 通过 API：

```bash
# 1. 上传应用 artifact
curl -X POST http://localhost:8080/v1/apps/upload \
  -F "file=@/tmp/test-app/test-app.zip"

# 返回示例：
# {"artifactId":"xxx","url":"/artifacts/test-app_xxx.zip"}

# 2. 创建部署
ARTIFACT_URL="/artifacts/test-app_xxx.zip"  # 使用上面返回的URL
curl -X POST http://localhost:8080/v1/deployments \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"test-container-app\",
    \"entries\": [{
      \"artifactUrl\": \"$ARTIFACT_URL\",
      \"replicas\": {\"nodeA\": 1}
    }]
  }"

# 返回示例：
# {"deploymentId":"yyy","instances":["inst-xxx"]}

# 3. 启动部署
DEPLOYMENT_ID="yyy"  # 使用上面返回的ID
curl -X POST "http://localhost:8080/v1/deployments/$DEPLOYMENT_ID?action=start"
```

### 步骤8：验证应用以容器方式运行

#### 检查容器是否创建：

```bash
# 查看运行中的容器（应该看到 plum-app- 开头的容器）
docker ps

# 应该看到类似输出：
# CONTAINER ID   IMAGE              COMMAND                  CREATED         STATUS         PORTS     NAMES
# abc123def456   alpine:latest      "./start.sh"            2 seconds ago   Up 1 second             plum-app-inst-xxx
```

#### 检查容器日志：

```bash
# 找到容器名称（从上面的输出获取）
CONTAINER_NAME="plum-app-inst-xxx"  # 替换为实际的容器名称

# 查看容器日志
docker logs $CONTAINER_NAME

# 实时跟踪日志
docker logs -f $CONTAINER_NAME
```

应该看到应用输出的日志。

#### 检查 Agent 日志：

Agent 的终端应该显示类似日志：
```
Started container abc123 for instance inst-xxx
Using base image: alpine:latest
```

#### 检查进程方式（对比验证）：

**验证这不是进程方式**：
```bash
# 检查是否有直接运行的进程（不应该有）
ps aux | grep "start.sh" | grep -v grep
# 如果有输出，说明可能还是进程模式，需要检查配置
```

**验证这是容器方式**：
```bash
# 容器内应该有应用进程
docker exec plum-app-inst-xxx ps aux
# 应该看到应用的进程
```

---

## 🧪 进一步测试

### 测试1：容器资源限制（如果配置了）

```bash
# 查看容器资源限制
docker inspect plum-app-inst-xxx | grep -A 10 "Resources"
```

### 测试2：容器环境变量

```bash
# 检查容器内的环境变量
docker exec plum-app-inst-xxx env | grep PLUM
# 应该看到：
# PLUM_INSTANCE_ID=inst-xxx
# PLUM_APP_NAME=...
# PLUM_APP_VERSION=...
```

### 测试3：容器故障恢复

```bash
# 停止容器（模拟故障）
docker stop plum-app-inst-xxx

# 观察 Agent 日志，应该检测到容器停止并自动重启
# 等待几秒后检查容器是否重新启动
docker ps | grep plum-app-inst-xxx
```

### 测试4：通过 Controller 停止应用

```bash
# 停止部署
curl -X POST "http://localhost:8080/v1/deployments/$DEPLOYMENT_ID?action=stop"

# 检查容器是否被删除
docker ps -a | grep plum-app-inst-xxx
# 容器应该被删除（不存在或状态为 Exited）
```

### 测试5：重新启动应用

```bash
# 重新启动部署
curl -X POST "http://localhost:8080/v1/deployments/$DEPLOYMENT_ID?action=start"

# 检查新容器是否创建
docker ps | grep plum-app-inst-
# 应该看到新的容器（可能有不同的ID）
```

---

## ⚠️ 常见问题排查

### 问题1：Agent 启动失败，提示 "failed to connect to docker daemon"

**原因**：Docker daemon 未运行或 Agent 无权限访问

**解决**：
```bash
# 启动 Docker
sudo systemctl start docker

# 检查权限
groups | grep docker
# 如果不在 docker 组，添加：
sudo usermod -aG docker $USER
newgrp docker  # 或在新的终端重新登录
```

### 问题2：Agent 日志显示 "Using app run mode: process"（不是 docker）

**原因**：环境变量未正确设置

**解决**：
```bash
# 检查环境变量
echo $AGENT_RUN_MODE

# 检查 .env 文件
cat agent-go/.env | grep AGENT_RUN_MODE

# 确保设置为 docker
export AGENT_RUN_MODE=docker
# 或修改 .env 文件
```

### 问题3：应用启动失败，提示 "failed to create container"

**原因**：
- 基础镜像不存在
- Docker socket 权限问题
- 应用目录不存在

**解决**：
```bash
# 拉取基础镜像
docker pull alpine:latest

# 检查 Docker socket 权限
ls -l /var/run/docker.sock
# 应该是 docker 组可访问

# 检查 Agent 日志获取详细错误信息
```

### 问题4：容器创建成功但立即退出

**原因**：
- 应用启动命令错误
- 应用脚本有问题

**解决**：
```bash
# 查看容器日志
docker logs plum-app-inst-xxx

# 查看容器退出码
docker inspect plum-app-inst-xxx | grep "ExitCode"

# 手动运行容器测试
docker run --rm -it alpine:latest /bin/sh
# 在容器内测试启动命令
```

### 问题5：无法访问容器内的应用服务

**原因**：容器网络隔离，端口未映射

**说明**：这是正常的，容器模式的应用默认使用 bridge 网络，如果需要外部访问，需要在 Docker 配置中添加端口映射（当前实现中暂未支持）。

---

## ✅ 成功标准

如果满足以下条件，说明测试成功：

1. ✅ Agent 日志显示 "Using app run mode: docker"
2. ✅ `docker ps` 能看到 `plum-app-` 开头的容器
3. ✅ 容器状态为 "Up" 且持续运行
4. ✅ `docker logs` 能看到应用输出
5. ✅ 容器内有 PLUM_* 环境变量
6. ✅ 通过 Controller 停止应用后，容器被删除
7. ✅ 重新启动应用后，新容器被创建

---

## 📝 测试结果记录

建议记录以下信息：

- [ ] Docker 版本：`docker --version`
- [ ] Agent 运行模式确认：日志中显示 "docker"
- [ ] 容器创建成功：容器ID/名称
- [ ] 容器运行状态：正常/异常
- [ ] 环境变量：是否包含 PLUM_*
- [ ] 故障恢复：kill 容器后是否自动重启
- [ ] 停止/启动：是否正常删除/创建容器

---

## 🎯 下一步

测试成功后，可以：

1. **测试方式3（完全容器化）**：使用 docker-compose.yml
2. **性能对比**：比较进程模式和容器模式的资源使用
3. **容器配置调优**：测试不同的基础镜像和资源限制

