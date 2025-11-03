# 离线部署应用基础镜像准备指南

## 📋 概述

在使用容器模式部署应用时，需要准备应用容器的基础镜像。本文档介绍如何手动准备常用的应用基础镜像，包括 ubuntu、openEuler 和 kylin。

## 🐧 Ubuntu 22.04 镜像

### 用途
Ubuntu 22.04 是推荐的应用容器基础镜像，兼容 glibc 应用（大多数 Linux 应用）。

### 下载和导出（x86 环境，下载 ARM64 镜像）

```bash
# 1. 拉取 ARM64 架构的镜像
docker pull --platform linux/arm64 ubuntu:22.04

# 2. 验证架构
docker inspect ubuntu:22.04 --format '{{.Architecture}}'
# 应该输出: arm64

# 3. 导出并压缩
docker save ubuntu:22.04 | gzip > ubuntu-22.04-arm64.tar.gz

# 4. 查看文件大小
ls -lh ubuntu-22.04-arm64.tar.gz
```

### 在目标环境加载

```bash
# 加载镜像
gunzip -c ubuntu-22.04-arm64.tar.gz | docker load

# 验证
docker images | grep ubuntu
```

## 🐉 openEuler 镜像

### 用途
openEuler 是华为推出的开源操作系统，适用于服务器场景。

### 下载和导出（x86 环境，下载 ARM64 镜像）

详细步骤请参考：[在 x86 环境下下载 openEuler ARM64 镜像](./OFFLINE_OPENEULER_IMAGE.md)

**快速操作**：

```bash
# 1. 拉取 ARM64 架构的镜像
docker pull --platform linux/arm64 openeuler/openeuler:latest
# 或指定版本
# docker pull --platform linux/arm64 openeuler/openeuler:22.03

# 2. 验证架构
docker inspect openeuler/openeuler:latest --format '{{.Architecture}}'

# 3. 导出并压缩
docker save openeuler/openeuler:latest | gzip > openeuler-latest-arm64.tar.gz

# 4. 查看可用标签
# 访问 https://hub.docker.com/r/openeuler/openeuler/tags
```

### 在目标环境加载

```bash
# 加载镜像
gunzip -c openeuler-latest-arm64.tar.gz | docker load

# 验证
docker images | grep openeuler
```

## 🏮 银河麒麟（kylin）镜像

### 用途
银河麒麟是国产操作系统，适用于政府、企业等对安全有要求的场景。

### 导入和标签设置

**注意**：kylin 镜像通常由官方提供压缩包，需要手动导入。

```bash
# 1. 加载镜像（从官方提供的 tar 文件）
docker load < kylin-v10-Release-020.tar

# 2. 检查镜像（可能显示 <none>）
docker images | grep "<none>"

# 3. 为镜像添加标签
# 找到镜像 ID（例如：9b0e4b0d9180）
docker tag <IMAGE_ID> kylin/kylin:v10-release-020

# 示例
docker tag 9b0e4b0d9180 kylin/kylin:v10-release-020

# 4. 验证标签
docker images | grep kylin
# 应该看到：
# kylin/kylin    v10-release-020    <image-id>    <size>
```

### 重新导出（带标签）

如果需要在其他环境使用，建议重新导出带标签的镜像：

```bash
# 使用 REPOSITORY:TAG 导出（保留标签信息）
docker save kylin/kylin:v10-release-020 | gzip > kylin-v10-Release-020-with-tag.tar.gz

# 在其他环境加载时会自动识别标签
gunzip -c kylin-v10-Release-020-with-tag.tar.gz | docker load
docker images | grep kylin
```

## 📝 在 Plum 中使用

### 配置 Agent 使用这些基础镜像

编辑 `agent-go/.env` 文件：

```bash
# 使用 Ubuntu
PLUM_BASE_IMAGE=ubuntu:22.04

# 或使用 openEuler
PLUM_BASE_IMAGE=openeuler/openeuler:22.03

# 或使用 kylin
PLUM_BASE_IMAGE=kylin/kylin:v10-release-020
```

### Docker Compose 配置

在 `docker-compose.yml` 或 `docker-compose.offline.yml` 中：

```yaml
environment:
  - PLUM_BASE_IMAGE=kylin/kylin:v10-release-020
```

## 🔍 镜像选择建议

| 镜像 | 适用场景 | 特点 |
|------|---------|------|
| **ubuntu:22.04** | 通用应用 | 兼容性好，支持 glibc 应用，社区活跃 |
| **openeuler/openeuler** | 服务器应用 | 华为开源，性能优化，适合企业级应用 |
| **kylin/kylin** | 国产化环境 | 符合国产化要求，适用于政府、国企等场景 |

## ⚠️ 注意事项

1. **架构匹配**：确保下载的镜像架构与目标环境一致（ARM64 或 AMD64）
2. **镜像标签**：导入后记得给镜像打标签，方便使用和管理
3. **导出格式**：使用 `docker save REPOSITORY:TAG` 而不是 `docker save IMAGE_ID`，以保留标签信息
4. **文件压缩**：使用 `gzip` 压缩可以显著减小文件大小（通常可减少 50-70%）

## 📚 相关文档

- [openEuler 镜像下载详细指南](./OFFLINE_OPENEULER_IMAGE.md)
- [容器应用管理文档](./CONTAINER_APP_MANAGEMENT.md)
- [离线部署指南](../docker/DEPLOYMENT-GUIDE.md)

---

*最后更新：2025年11月3日*

