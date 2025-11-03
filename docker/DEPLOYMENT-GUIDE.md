# Plum Docker 部署指南

## 📋 概述

本指南包含Plum项目的Docker部署方案：
- **在线部署**：使用Docker Compose快速启动
- **离线部署**：ARM64环境下的完整离线部署流程

## 🚀 在线部署

### Docker Compose 文件说明

| 文件 | 用途 | 服务 |
|------|------|------|
| `docker-compose.yml` | 测试环境 | Controller + 3个Agent + Nginx |
| `docker-compose.production.yml` | 生产环境 | 单节点Agent |
| `docker-compose.offline.yml` | 离线环境 | Controller + 3个Agent + Nginx |

### 启动命令

```bash
# 测试环境（完整系统）
docker-compose up -d

# 生产环境（单节点）
docker-compose -f docker-compose.production.yml up -d

# 仅启动Controller
docker-compose up -d plum-controller

# 带Nginx的测试环境
docker-compose --profile nginx up -d
```

### 自动化脚本

```bash
# 使用deploy.sh脚本
./docker/deploy.sh test start    # 测试环境
./docker/deploy.sh production start  # 生产环境
./docker/deploy.sh controller start  # 仅Controller
```

## 🏗️ 离线部署（ARM64）

### 联网环境准备

#### 1. 准备部署包
```bash
# 创建离线部署包
./plum-offline-deploy/scripts-prepare/prepare-offline-deploy.sh
# 选择不构建Docker镜像（输入N）
```

#### 2. 准备Docker镜像（可选方案）
```bash
# 方案A：仅准备基础镜像（推荐）
docker pull --platform linux/arm64 alpine:3.18
docker pull --platform linux/arm64 nginx:alpine
docker save alpine:3.18 | gzip > alpine-3.18-arm64.tar.gz
docker save nginx:alpine | gzip > nginx-alpine-arm64.tar.gz

# 方案B：准备完整镜像包（使用脚本）
./docker/generate-offline-images.sh
# 脚本会自动包含：
# - alpine:3.18（Plum容器基础镜像）
# - nginx:alpine（Nginx服务）

# 如果需要使用容器模式部署应用，需要单独准备应用基础镜像
# 详细步骤请参考：docs/OFFLINE_APP_BASE_IMAGES.md
# - ubuntu:22.04（应用容器基础镜像）
# - openEuler（可选）
# - kylin（可选，由官方提供）
```

#### 3. 打包传输
```bash
# 打包部署包
tar -czf plum-offline-deploy.tar.gz plum-offline-deploy/

# 传输文件到目标环境
# 必需文件：
# - plum-offline-deploy.tar.gz
# - alpine-3.18-arm64.tar.gz
# - nginx-alpine-arm64.tar.gz
# 
# 如果使用容器模式部署应用（可选），需要额外准备应用基础镜像：
# - ubuntu-22.04-arm64.tar.gz
# - openeuler-*.tar.gz
# - kylin-v10-*.tar（由官方提供）
# 
# 详细步骤请参考：docs/OFFLINE_APP_BASE_IMAGES.md
```

### 离线环境部署

#### 1. 解压部署包
```bash
tar -xzf plum-offline-deploy.tar.gz
cd plum-offline-deploy/source/Plum
```

#### 2. 加载Docker镜像
```bash
# 方案A：仅加载基础镜像（推荐）
gunzip -c alpine-3.18-arm64.tar.gz | docker load
gunzip -c nginx-alpine-arm64.tar.gz | docker load

# 如果使用容器模式部署应用（可选），加载应用基础镜像
gunzip -c ubuntu-22.04-arm64.tar.gz | docker load
# 或 openEuler
gunzip -c openeuler-latest-arm64.tar.gz | docker load
# 或 kylin（从官方提供的 tar 文件）
docker load < kylin-v10-Release-020.tar
docker tag <IMAGE_ID> kylin/kylin:v10-release-020  # 需要添加标签

# 方案B：加载完整镜像包（使用脚本）
./docker/load-offline-images.sh
```

**注意**：应用基础镜像需要单独准备，详细步骤请参考：[应用基础镜像准备指南](../docs/OFFLINE_APP_BASE_IMAGES.md)

#### 3. 构建Plum镜像（方案A需要）
```bash
# 如果使用方案A，需要构建Plum镜像
./docker/build-static-offline-fixed.sh
```

#### 4. 配置服务（可选：容器模式）
如果需要使用容器模式部署应用，需要配置Agent：

```bash
# 编辑agent-go/.env文件
cd plum-offline-deploy/source/Plum
vim agent-go/.env

# 添加或修改以下配置：
# AGENT_RUN_MODE=docker  # 启用容器模式
# PLUM_BASE_IMAGE=ubuntu:22.04  # 应用容器基础镜像
# PLUM_CONTAINER_MEMORY=512m  # 可选：容器内存限制
# PLUM_CONTAINER_CPUS=1.0  # 可选：容器CPU限制
```

#### 5. 启动服务
```bash
# 启动所有服务
docker-compose -f docker-compose.offline.yml up -d

# 检查状态
docker-compose -f docker-compose.offline.yml ps

# 注意：docker-compose.offline.yml已配置Docker socket挂载
# Agent容器已可以访问宿主机Docker来管理应用容器
```

### 验证部署

```bash
# 测试Controller
curl http://localhost:8080/v1/nodes

# 测试Nginx
curl http://localhost/health

# 查看日志
docker-compose -f docker-compose.offline.yml logs -f
```

## 🔧 故障排除

### 常见问题

1. **架构不匹配**：镜像与目标机CPU不一致（amd64 vs arm64）。
   - 生成包时脚本会在文件名中加入架构后缀（如 `-amd64`、`-arm64`）。
   - 验证命令: `docker inspect <image:tag> | grep -i Architecture`
   - 选择与目标环境相同架构的 `.tar.gz` 加载；否则请在目标机上重建镜像。
2. **端口冲突**：检查8080、80端口占用
3. **配置文件缺失**：确保.env文件存在
4. **容器模式无法启动应用容器**：
   - 确保已加载`ubuntu:22.04`镜像（或指定的`PLUM_BASE_IMAGE`）
   - 验证：`docker images | grep ubuntu`
   - 确保Agent容器已挂载Docker socket：`docker inspect plum-agent-a | grep docker.sock`
   - 检查Agent日志：`docker-compose -f docker-compose.offline.yml logs plum-agent-a`

### 日志查看
```bash
docker-compose -f docker-compose.offline.yml logs plum-controller
docker-compose -f docker-compose.offline.yml logs plum-agent-a
```

## 📞 获取帮助

如果遇到问题，请参考：
- **[问题解决指南](TROUBLESHOOTING-GUIDE.md)** - 常见问题及解决方案
- 查看服务日志：`docker-compose logs -f`
- 检查服务状态：`docker-compose ps`
- 参考Docker官方文档

---

## 🐳 容器模式离线部署说明

### 概述

Plum支持三种部署方式，离线环境下的容器模式部署需要额外准备应用基础镜像。

### 部署方式对比

| 部署方式 | Controller/Agent运行方式 | 应用运行方式 | 离线部署额外要求 |
|---------|-------------------------|------------|----------------|
| **方式1：裸应用模式** | 直接运行 | 进程方式 | 无 |
| **方式2：混合容器模式** | 直接运行 | 容器方式 | 需要应用基础镜像 |
| **方式3：完全容器化** | 容器运行 | 容器方式 | 需要应用基础镜像 |

### 容器模式离线部署步骤

1. **准备应用基础镜像**（需要手动准备）
   
   详细步骤请参考：[应用基础镜像准备指南](../docs/OFFLINE_APP_BASE_IMAGES.md)
   
   常用镜像：
   - **ubuntu:22.04**：通用应用，兼容 glibc
   - **openeuler/openeuler**：华为开源操作系统
   - **kylin/kylin**：银河麒麟，国产化环境

2. **传输镜像到离线环境**
   - 包含应用基础镜像的 tar/tar.gz 文件

3. **加载应用基础镜像**
   ```bash
   # Ubuntu 或 openEuler
   gunzip -c ubuntu-22.04-arm64.tar.gz | docker load
   
   # kylin（从官方 tar 文件）
   docker load < kylin-v10-Release-020.tar
   docker tag <IMAGE_ID> kylin/kylin:v10-release-020  # 添加标签
   ```

4. **配置Agent启用容器模式**
   ```bash
   # 编辑agent-go/.env或docker-compose.offline.yml环境变量
   AGENT_RUN_MODE=docker
   PLUM_BASE_IMAGE=kylin/kylin:v10-release-020  # 或其他基础镜像
   ```

5. **启动服务**
   ```bash
   docker-compose -f docker-compose.offline.yml up -d
   ```

### 注意事项

- **Docker Socket挂载**：`docker-compose.offline.yml`已配置Docker socket挂载，Agent容器可访问宿主机Docker
- **基础镜像选择**：
  - `ubuntu:22.04`：推荐，兼容glibc应用（大多数Linux应用）
  - `alpine:latest`：轻量级，但只支持musl libc应用
- **架构兼容性**：确保应用基础镜像与目标环境架构一致（ARM64或AMD64）
- **库路径映射**：如需共享宿主机库，配置`PLUM_HOST_LIB_PATHS`环境变量

详细说明请参考：
- [容器应用管理文档](../docs/CONTAINER_APP_MANAGEMENT.md)
- [应用基础镜像准备指南](../docs/OFFLINE_APP_BASE_IMAGES.md)
- [混合容器模式测试指南](../docs/TEST_CONTAINER_MODE.md)
- [完全容器化测试指南](../docs/TEST_FULLY_CONTAINERIZED.md)

---

*最后更新：2025年11月3日*
