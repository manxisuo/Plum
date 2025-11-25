# Python 离线包准备指南

## 问题

树莓派网络较慢，无法从 PyPI 下载 Python 依赖包，导致 Docker 构建失败。

## 解决方案

### 方案 1：使用项目自带的离线包下载脚本（推荐）

#### FSL_MainControl

```bash
# 在 WSL2 或网络较好的环境中执行
cd examples-local/FSL_MainControl
./download_dependencies.sh

# 这会创建 offline-pip-packages/ 目录，包含所有依赖
```

#### Sim_Decision

```bash
# 在 WSL2 或网络较好的环境中执行
cd examples-local/Sim_Decision
./download_dependencies.sh

# 这会创建 offline-pip-packages/ 目录，包含所有依赖
```

### 方案 2：手动下载离线包

#### 在 WSL2 中操作：

```bash
# 1. 进入项目目录
cd examples-local/FSL_MainControl  # 或 Sim_Decision

# 2. 创建离线包目录
mkdir -p offline-pip-packages

# 3. 下载依赖包（ARM64 架构）
pip download \
  --platform manylinux2014_aarch64 \
  --python-version 311 \
  --implementation cp \
  --only-binary=:all: \
  --dest offline-pip-packages \
  -r requirements.txt

# 4. 查看下载的包
ls -lh offline-pip-packages/
```

#### 传输到树莓派：

```bash
# 使用 scp 或其他方式传输整个目录
scp -r offline-pip-packages pi@<树莓派IP>:/home/pi/Plum/examples-local/FSL_MainControl/
```

### 方案 3：使用国内 PyPI 镜像源（已集成到 Dockerfile）

Dockerfile 已经配置了国内镜像源作为备选方案：
1. 优先使用清华大学镜像源
2. 如果失败，尝试阿里云镜像源
3. 最后回退到官方 PyPI

但网络太慢时仍可能超时，建议使用离线包。

## 使用离线包构建

一旦有了离线包，构建脚本会自动检测并使用：

```bash
# 构建时会自动检测 offline-pip-packages/ 或 offline-packages/ 目录
./examples-local/build-docker-python.sh FSL_MainControl
```

## 注意事项

1. **架构匹配**：确保下载的包是 ARM64 架构（`manylinux2014_aarch64`）
2. **Python 版本**：
   - Ubuntu 24.04 默认是 Python 3.12
   - 下载脚本默认使用 Python 3.12（可通过 `PYTHON_VERSION` 覆盖）
   - 如果某些包没有 3.12 版本，可以尝试 3.11（通常兼容）
3. **目录位置**：离线包目录应该在项目根目录下（`examples-local/<项目名>/offline-pip-packages/`）
4. **使用项目脚本**：推荐使用项目自带的 `download_dependencies.sh` 脚本，它会自动处理版本和平台

## 检查离线包

```bash
# 检查离线包是否存在且不为空
ls -lh examples-local/FSL_MainControl/offline-pip-packages/

# 应该看到很多 .whl 文件
```

