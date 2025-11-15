# Docker 镜像标签使用指南

## 📋 镜像标签说明

Plum 项目中有两种构建方式，生成不同标签的镜像：

### 1. `build-docker.sh --local` 构建
- 生成镜像：`plum-controller:latest`、`plum-agent:latest`
- 使用本地 Go 环境编译，不需要下载大型 golang 镜像
- 适合网络慢的环境

### 2. `build-static-offline.sh` 构建
- 生成镜像：`plum-controller:offline`、`plum-agent:offline`
- 使用本地 Go 环境编译
- 适合完全离线环境

## 🔄 Docker Compose 文件与镜像标签的关系

### 关键概念

**镜像标签只是一个标识符**，任何 docker-compose 文件都可以使用任何标签的镜像，只要：
1. 镜像已经预先构建好
2. yml 文件中的 `image:` 标签与已构建的镜像标签匹配

### 两种使用方式

#### 使用 `image:` 指令（使用预先构建的镜像）

```yaml
# docker-compose.main.yml
services:
  plum-controller:
    image: plum-controller:offline
```

- **特点**：使用预先构建好的镜像，启动更快
- **要求**：必须先运行构建脚本生成对应标签的镜像

## 📝 实际使用示例

### 场景 1：使用 `latest` 标签的镜像

```bash
# 1. 构建镜像（使用本地 Go 环境）
./docker/build-docker.sh all --local

# 2. 修改 docker-compose.main.yml 和 docker-compose.agent.yml，将：
#    image: plum-controller:offline 改为 image: plum-controller:latest
#    image: plum-agent:offline 改为 image: plum-agent:latest

# 3. 启动服务
docker-compose -f docker-compose.main.yml -f docker-compose.agent.yml up -d
```

### 场景 2：使用 `offline` 标签的镜像

```bash
# 1. 构建镜像
./docker/build-static-offline.sh

# 2. 直接使用现有的 yml 文件（使用 offline 标签）
docker-compose -f docker-compose.main.yml -f docker-compose.agent.yml up -d
```

### 场景 3：同时启动 Controller 和 Agent

```bash
# 1. 构建镜像
./docker/build-docker.sh all --local

# 2. 修改 yml 文件中的镜像标签为 latest（如果需要）

# 3. 同时启动 Controller 和 Agent
docker-compose -f docker-compose.main.yml -f docker-compose.agent.yml up -d
```

## 🎯 推荐方案

### 网络慢的环境（推荐）

```bash
# 1. 使用本地构建（不需要下载 golang 镜像）
./docker/build-docker.sh all --local

# 2. 修改需要的 yml 文件，将 offline 标签改为 latest
#    例如：docker-compose.main.yml, docker-compose.agent.yml

# 3. 启动服务
docker-compose -f docker-compose.main.yml -f docker-compose.agent.yml up -d
```

### 完全离线环境

```bash
# 1. 构建 offline 标签的镜像
./docker/build-static-offline.sh

# 2. 直接使用现有的 yml 文件（使用 offline 标签）
docker-compose -f docker-compose.main.yml -f docker-compose.agent.yml up -d
```

## ⚠️ 常见问题

### Q: `build-docker.sh --local` 构建的镜像必须使用 `docker-compose.yml` 启动吗？

**A: 不是！** 镜像标签只是标识符，可以在任何 docker-compose 文件中使用。

只需要：
1. 确保镜像已经构建好（标签为 `latest`）
2. 在 yml 文件中使用 `image: plum-controller:latest`（而不是 `build:` 或 `image: plum-controller:offline`）

### Q: 可以在 `docker-compose.agent.yml` 中使用 `latest` 标签的镜像吗？

**A: 可以！** 只需修改 yml 文件：

```yaml
# 修改前
image: plum-agent:offline

# 修改后
image: plum-agent:latest
```

### Q: 如何同时启动 Controller 和 Agent？

**A: 使用多个 yml 文件组合启动：**

```bash
# 同时启动 Controller 和 Agent
docker-compose -f docker-compose.main.yml -f docker-compose.agent.yml up -d

# 或者分别启动
docker-compose -f docker-compose.main.yml up -d  # 启动 Controller
docker-compose -f docker-compose.agent.yml up -d  # 启动 Agent
```

## 📚 相关文件

- `docker-compose.main.yml` - Controller 服务配置，使用 `image: plum-controller:offline`
- `docker-compose.agent.yml` - Agent 服务配置，使用 `image: plum-agent:offline`

**💡 提示**：如果需要启动多个 Agent 节点，可以多次运行 `docker-compose.agent.yml`，或者创建多个 Agent 配置文件。

