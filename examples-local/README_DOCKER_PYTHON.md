# Python 项目 Docker 镜像构建指南（通用版）

## 概述

本项目提供了通用的 Python 项目 Docker 镜像构建脚本，可以用于所有 Python 项目（FSL_MainControl、Sim_Decision 等）。

## 快速开始

### 构建单个项目

```bash
# 从项目根目录执行
cd /path/to/Plum

# 构建 FSL_MainControl
./examples-local/build-docker-python.sh FSL_MainControl

# 构建 Sim_Decision
./examples-local/build-docker-python.sh Sim_Decision
```

## 工作原理

通用构建脚本会自动：

1. **检查项目文件**：验证 `requirements.txt` 和 `app.py` 是否存在
2. **复制项目文件**：使用 `copy-deps-python.sh` 脚本复制所有必需的文件
   - 源代码文件（`app.py`）
   - 依赖文件（`requirements.txt`）
   - 启动脚本（`bin/start.sh`）
   - 元数据（`bin/meta.ini`）
   - 模板文件（`templates/`，如果存在）
   - 静态文件（`static/`，如果存在）
   - 脚本文件（`scripts/`，如果存在）
   - 离线 Python 包（`offline-pip-packages/` 或 `offline-packages/`，如果存在）
3. **构建 Docker 镜像**：使用通用的 `Dockerfile.python.template` 或项目特定的 `Dockerfile.python`

## 文件说明

### 通用脚本

- **`build-docker-python.sh`**：通用的 Python 项目构建脚本，接受项目名作为参数
- **`copy-deps-python.sh`**：通用的 Python 项目文件复制脚本
- **`Dockerfile.python.template`**：通用的 Python 项目 Dockerfile 模板

### 项目特定文件（可选）

如果某个项目需要特殊的配置，可以在项目目录下创建：

- **`<项目名>/Dockerfile.python`**：项目特定的 Dockerfile（会优先使用）
- **`<项目名>/bin/start.sh`**：启动脚本（必需）
- **`<项目名>/bin/meta.ini`**：应用元数据（必需）

## 手动构建（高级用法）

如果需要更多控制，可以手动执行：

```bash
# 1. 准备依赖
./examples-local/copy-deps-python.sh FSL_MainControl /tmp/fsl-maincontrol-deps

# 2. 构建镜像
docker buildx build \
  --platform linux/arm64 \
  --load \
  -f examples-local/Dockerfile.python.template \
  --build-arg APP_NAME=FSL_MainControl \
  -t fsl_maincontrol:1.0.0 \
  /tmp/fsl-maincontrol-deps

# 3. 清理
rm -rf /tmp/fsl-maincontrol-deps
```

## 测试镜像

```bash
# 测试 FSL_MainControl（FastAPI，端口 4000）
docker run --rm -p 4000:4000 fsl_maincontrol:1.0.0

# 测试 Sim_Decision（Flask，端口 3000）
docker run --rm -p 3000:3000 sim_decision:1.0.0
```

## 支持的项目

- ✅ FSL_MainControl（FastAPI + uvicorn）
- ✅ Sim_Decision（Flask）

## 离线安装依赖

如果项目有离线 Python 包，脚本会自动检测并使用：

1. **检查目录**：`offline-pip-packages/` 或 `offline-packages/`
2. **自动使用**：如果存在且不为空，Dockerfile 会自动使用离线包安装依赖
3. **回退到 PyPI**：如果离线包不存在或为空，会从 PyPI 安装

## 注意事项

1. **架构要求**：脚本会自动使用 `--platform linux/arm64`，确保在树莓派上构建正确的镜像
2. **网络要求**：首次构建需要下载 Python 基础镜像，后续构建会使用缓存
3. **Python 版本**：使用 `python:3.11-slim` 作为基础镜像，如果需要其他版本可以修改模板
4. **依赖安装**：如果网络慢，建议预先下载离线包

## 故障排除

### 问题：找不到 requirements.txt

**错误**：`错误: 找不到 requirements.txt`

**解决**：确保项目目录下有 `requirements.txt` 文件

### 问题：找不到 app.py

**错误**：`错误: 找不到 app.py`

**解决**：确保项目目录下有 `app.py` 文件

### 问题：镜像构建失败

**错误**：`ERROR: failed to build`

**解决**：
1. 检查网络连接（首次构建需要下载基础镜像）
2. 确保有足够的磁盘空间
3. 检查 Docker 是否正常运行：`docker info`

### 问题：依赖安装失败

**错误**：`pip install` 失败

**解决**：
1. 检查 `requirements.txt` 格式是否正确
2. 如果有离线包，确保目录结构正确
3. 检查 Python 版本是否兼容

## 与 C++ 项目的区别

| 特性 | C++ 项目 | Python 项目 |
|------|---------|------------|
| 构建脚本 | `build-docker-local.sh` | `build-docker-python.sh` |
| 依赖复制 | `copy-deps.sh` | `copy-deps-python.sh` |
| Dockerfile 模板 | `Dockerfile.local.template` | `Dockerfile.python.template` |
| 基础镜像 | `ubuntu:24.04` | `python:3.11-slim` |
| 需要编译 | 是（需要先编译） | 否（直接复制源代码） |
| 依赖管理 | 系统库（.so 文件） | Python 包（requirements.txt） |

