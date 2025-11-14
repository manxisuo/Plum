# Python 基础镜像离线准备指南

## 问题

树莓派网络较慢，无法直接从 Docker Hub 拉取 `python:3.11-slim` 镜像。

## 解决方案

### 方案 1：在 WSL2 中下载并传输（推荐）

#### 在 WSL2 中操作：

```bash
# 1. 拉取 ARM64 架构的 Python 镜像
docker pull --platform linux/arm64 python:3.11-slim

# 2. 验证架构
docker inspect python:3.11-slim --format '{{.Architecture}}'
# 应该输出: arm64

# 3. 导出并压缩
docker save python:3.11-slim | gzip > python-3.11-slim-arm64.tar.gz

# 4. 查看文件大小
ls -lh python-3.11-slim-arm64.tar.gz
```

#### 传输到树莓派：

```bash
# 使用 scp 或其他方式传输
scp python-3.11-slim-arm64.tar.gz pi@<树莓派IP>:/tmp/
```

#### 在树莓派上加载：

```bash
# 加载镜像
gunzip -c /tmp/python-3.11-slim-arm64.tar.gz | docker load

# 验证
docker images | grep python
```

### 方案 2：使用 Ubuntu 基础镜像 + 手动安装 Python

如果无法下载 Python 镜像，可以修改 Dockerfile 使用 Ubuntu 基础镜像，然后手动安装 Python。

