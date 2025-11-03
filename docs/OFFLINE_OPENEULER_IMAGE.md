# 在 x86 环境下下载 openEuler ARM64 镜像用于离线部署

## 📋 环境说明

- **操作环境**: Ubuntu 22.04 (WSL2, x86_64)
- **目标镜像**: openEuler ARM64
- **用途**: 离线部署到ARM64环境

## 🚀 操作步骤

### 步骤1：检查 Docker 多架构支持

```bash
# 检查 Docker 版本（需要 19.03+ 支持 --platform）
docker --version

# 检查 buildx 是否可用（现代 Docker 已内置）
docker buildx version

# 如果 buildx 不可用，启用实验性功能
# Docker 19.03+ 已默认支持 --platform，通常不需要额外配置
```

### 步骤2：拉取 openEuler ARM64 镜像

```bash
# 查看可用的 openEuler 镜像标签
# 访问 https://hub.docker.com/r/openeuler/openeuler/tags 查看所有可用标签

# 拉取 ARM64 架构的镜像（使用 --platform 参数）
docker pull --platform linux/arm64 openeuler/openeuler:latest

# 或指定特定版本（例如 22.03 LTS）
# docker pull --platform linux/arm64 openeuler/openeuler:22.03

# 验证镜像架构
docker inspect openeuler/openeuler:latest | grep -A 5 Architecture
```

### 步骤3：验证镜像架构

```bash
# 方法1：使用 docker inspect
docker inspect openeuler/openeuler:latest --format '{{.Architecture}}'

# 方法2：查看镜像详细信息
docker image inspect openeuler/openeuler:latest | grep -i architecture

# 应该显示: arm64 或 aarch64
```

### 步骤4：导出镜像

```bash
# 方法A：导出为 tar 文件（未压缩，速度快）
docker save -o openeuler-arm64.tar openeuler/openeuler:latest

# 方法B：导出为压缩的 tar.gz 文件（文件更小，推荐）
docker save openeuler/openeuler:latest | gzip > openeuler-arm64.tar.gz

# 查看文件大小
ls -lh openeuler-arm64.tar*
```

### 步骤5：验证导出的文件

```bash
# 验证 tar 文件完整性
file openeuler-arm64.tar.gz

# 查看文件大小
du -h openeuler-arm64.tar.gz
```

### 步骤6：传输到目标环境

```bash
# 使用 scp 传输（如果目标环境可访问）
scp openeuler-arm64.tar.gz user@target-host:/path/to/destination/

# 或使用 USB、网络共享等方式传输
```

### 步骤7：在目标 ARM64 环境导入

```bash
# 在目标 ARM64 环境中导入镜像

# 方法A：从 tar 文件导入
docker load < openeuler-arm64.tar

# 方法B：从 tar.gz 文件导入
gunzip -c openeuler-arm64.tar.gz | docker load
# 或
zcat openeuler-arm64.tar.gz | docker load

# 验证导入成功
docker images | grep openeuler

# 测试运行
docker run --rm openeuler/openeuler:latest uname -m
# 应该输出: aarch64
```

## 🔍 常见问题和解决方案

### Q1: 拉取时提示 "no matching manifest"

**原因**: 指定平台不存在该镜像

**解决**:
```bash
# 检查镜像是否支持 ARM64
docker manifest inspect openeuler/openeuler:latest

# 或访问 Docker Hub 查看支持的架构
# https://hub.docker.com/r/openeuler/openeuler/tags
```

### Q2: 在 x86 环境下无法运行 ARM64 镜像

**说明**: 这是正常的，x86 环境下只能拉取和导出 ARM64 镜像，不能运行

**解决**: ARM64 镜像只能在 ARM64 环境中运行，在 x86 环境下只需要完成拉取和导出即可

### Q3: 导出文件过大

**解决**: 使用压缩格式（tar.gz）：
```bash
docker save openeuler/openeuler:latest | gzip > openeuler-arm64.tar.gz
```

### Q4: 需要下载特定版本

```bash
# 查看可用标签（访问 Docker Hub）
# https://hub.docker.com/r/openeuler/openeuler/tags

# 拉取特定版本
docker pull --platform linux/arm64 openeuler/openeuler:22.03
docker pull --platform linux/arm64 openeuler/openeuler:23.09
```

## 📝 完整操作示例

```bash
# 1. 拉取 ARM64 镜像
docker pull --platform linux/arm64 openeuler/openeuler:latest

# 2. 验证架构
docker inspect openeuler/openeuler:latest --format '{{.Architecture}}'
# 输出: arm64

# 3. 导出并压缩
docker save openeuler/openeuler:latest | gzip > openeuler-arm64.tar.gz

# 4. 查看文件信息
ls -lh openeuler-arm64.tar.gz
# 输出类似: -rw-r--r-- 1 user user 150M Nov  3 10:00 openeuler-arm64.tar.gz

# 5. 在目标环境导入（示例）
# 传输文件到目标环境后：
gunzip -c openeuler-arm64.tar.gz | docker load

# 6. 验证导入
docker images | grep openeuler
```

## 💡 提示

1. **镜像版本选择**:
   - `latest`: 最新版本
   - `22.03`: 22.03 LTS 版本
   - `23.09`: 23.09 版本
   - 访问 [Docker Hub](https://hub.docker.com/r/openeuler/openeuler/tags) 查看所有可用标签

2. **压缩比**: 使用 `gzip` 压缩通常可以减少 50-70% 的文件大小

3. **批量操作**: 如需导出多个镜像，可以使用循环：
   ```bash
   for tag in latest 22.03 23.09; do
       docker pull --platform linux/arm64 openeuler/openeuler:$tag
       docker save openeuler/openeuler:$tag | gzip > openeuler-$tag-arm64.tar.gz
   done
   ```

4. **与 Plum 集成**: 可以将 openEuler 镜像作为应用容器基础镜像：
   ```bash
   # 在 agent-go/.env 中配置
   PLUM_BASE_IMAGE=openeuler/openeuler:22.03
   ```

---

*最后更新: 2025年11月3日*

