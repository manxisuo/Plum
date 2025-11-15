# Plum Docker 部署

## 📚 文档

- **[部署指南](DEPLOYMENT-GUIDE.md)** ⭐ **完整部署说明**
- **[问题解决指南](TROUBLESHOOTING-GUIDE.md)** 🔧 **常见问题解决方案**

## 🛠️ 工具脚本

- **[build-docker.sh](build-docker.sh)** 🐳 **构建 Controller 和 Agent Docker 镜像（推荐）**
- **[build-static-offline.sh](build-static-offline.sh)** 🔧 **离线静态构建脚本**
- **[deploy.sh](deploy.sh)** 🚀 **部署脚本**
- **[generate-offline-images.sh](generate-offline-images.sh)** 📦 **生成离线镜像包**
- **[load-offline-images.sh](load-offline-images.sh)** 📥 **加载离线镜像包**
- **[prepare-alpine-with-packages.sh](prepare-alpine-with-packages.sh)** 🔨 **准备包含必要包的 Alpine 镜像**

**注意**：库文件复制功能已统一使用 `examples-local/copy-deps.sh`，旧的库复制脚本已归档到 `archive_unused/` 目录。

## 🚀 快速开始

### 构建 Docker 镜像

#### 方式一：使用 build-docker.sh（推荐）

**适合网络慢的环境（推荐）**：
```bash
# 使用本地 Go 环境构建（不需要下载 golang 镜像）
./docker/build-docker.sh all --local

# 只构建 Controller
./docker/build-docker.sh controller --local

# 只构建 Agent
./docker/build-docker.sh agent --local
```

**适合网络好的环境**：
```bash
# 使用 Docker 多阶段构建（需要下载 golang:1.23-alpine 镜像）
./docker/build-docker.sh all

# 只构建 Controller
./docker/build-docker.sh controller

# 只构建 Agent
./docker/build-docker.sh agent
```

#### 方式二：使用 build-static-offline.sh（离线构建）

```bash
# 完全离线构建（使用本地 Go 环境，生成 offline 标签的镜像）
./docker/build-static-offline.sh
# 生成的镜像: plum-controller:offline, plum-agent:offline
```

#### 两种方式的区别

| 特性 | `build-docker.sh --local` | `build-static-offline.sh` |
|------|---------------------------|--------------------------------|
| **网络要求** | 需要下载 `alpine:3.18`（约 5MB，可能已缓存） | 需要下载 `alpine:3.18`（约 5MB，可能已缓存） |
| **Go 环境** | 使用本地 Go 环境 | 使用本地 Go 环境 |
| **镜像标签** | `plum-controller:latest`<br>`plum-agent:latest` | `plum-controller:offline`<br>`plum-agent:offline` |
| **使用方式** | 在任何 docker-compose 文件中使用 `image: plum-controller:latest` | 在 docker-compose 文件中使用 `image: plum-controller:offline` |

**💡 重要说明**：
- **镜像标签只是标识符**，任何 docker-compose 文件都可以使用任何标签的镜像
- 只需确保 yml 文件中的 `image:` 标签与已构建的镜像标签匹配
- 所有 yml 文件（如 `docker-compose.main.yml`、`docker-compose.agent.yml`）都使用 `image:` 指令，需要预先构建对应标签的镜像
- 如果使用 `build-docker.sh --local` 构建了 `latest` 标签的镜像，可以在任何 yml 文件中使用，只需将 `image: plum-controller:offline` 改为 `image: plum-controller:latest`

**💡 建议**：如果网络慢导致 `build-docker.sh` 失败，使用 `./docker/build-docker.sh all --local` 即可。

### 部署服务

```bash
# 启动 Controller（主服务）
docker-compose -f docker-compose.main.yml up -d

# 启动 Agent（工作节点）
docker-compose -f docker-compose.agent.yml up -d

# 同时启动 Controller 和 Agent
docker-compose -f docker-compose.main.yml -f docker-compose.agent.yml up -d
```

### 离线部署
```bash
# 1. 构建镜像（生成 offline 标签）
./docker/build-static-offline.sh

# 2. 启动服务（使用 offline 标签的镜像）
docker-compose -f docker-compose.main.yml up -d  # 启动 Controller
docker-compose -f docker-compose.agent.yml up -d  # 启动 Agent

# 或者同时启动
docker-compose -f docker-compose.main.yml -f docker-compose.agent.yml up -d
```

详细说明请参考：[部署指南](DEPLOYMENT-GUIDE.md)