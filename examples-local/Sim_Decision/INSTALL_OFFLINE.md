# SimDecision 离线安装指南

本文档说明如何在 ARM64 架构的银河麒麟V10 或 OpenEuler 系统上离线安装 SimDecision。

## 前置要求

### 在联网环境（下载机器）

1. **Python 环境**：需要安装 Python 3.x 和 pip
2. **架构**：下载机器可以是 x86_64 或 ARM64（推荐 x86_64，因为下载速度可能更快）

### 在目标环境（离线机器）

1. **操作系统**：银河麒麟V10 或 OpenEuler（ARM64）
2. **Python 环境**：Python 3.x（建议 3.7 或更高版本）
3. **编译工具**（如果下载的是源码包）：
   - gcc
   - g++
   - python3-dev / python3-devel
   - 其他编译依赖

## 步骤 1：下载离线安装包

在联网环境下执行：

```bash
cd examples-local/SimDecision
./download_dependencies.sh
```

脚本会自动下载 Flask/Requests 及其依赖到 `offline-pip-packages/` 目录（默认针对 Python 3.11，可通过设置 `PYTHON_VERSION` 覆盖）。

### 下载选项说明

- 默认会下载 manylinux2014_aarch64 平台、Python 3.11 的二进制 wheel。
- 可通过环境变量调整：
  ```bash
  PYTHON_VERSION=310 PLATFORM=manylinux2014_aarch64 ./download_dependencies.sh
  ```
- 如果某些包没有对应 wheel，脚本会提示，可改用源码包或手动下载。

## 步骤 2：传输到目标环境

将以下内容复制到目标环境：

1. `offline-pip-packages/` 目录（包含所有下载的包）
2. `app.py` 文件
3. `templates/` 目录（如果存在）
4. `static/` 目录（如果存在）
5. `bin/` 目录（包含启动脚本等）

可以使用以下方式传输：

```bash
# 使用 tar 打包
tar -czf simdecision-offline.tar.gz SimDecision/

# 或使用 zip
zip -r simdecision-offline.zip SimDecision/
```

## 步骤 3：在目标环境安装

### 3.1 解压文件

```bash
tar -xzf simdecision-offline.tar.gz
cd SimDecision
```

### 3.2 安装依赖

```bash
./install_dependencies.sh
```

脚本默认从 `offline-pip-packages/` 目录安装 Flask 3.0.0 与 Requests 2.31.0；如包目录不在默认路径，可传入参数：

```bash
./install_dependencies.sh /path/to/offline-pip-packages
```

### 3.3 如果遇到编译错误

如果某些包是源码包，需要编译，但遇到编译错误，可能需要：

1. **安装编译工具**：
   ```bash
   # 银河麒麟V10 / OpenEuler
   sudo yum install gcc g++ python3-devel
   # 或
   sudo dnf install gcc g++ python3-devel
   ```

2. **安装系统依赖**（某些 Python 包需要系统库）：
   ```bash
   # 例如：Flask 可能需要
   sudo yum install openssl-devel
   ```

3. **重新尝试安装**：
   ```bash
   pip3 install --no-index --find-links=offline-pip-packages flask==3.0.0 requests==2.31.0
   ```

## 步骤 4：验证安装

```bash
python3 -c "import flask; import requests; print('Flask version:', flask.__version__); print('Requests version:', requests.__version__)"
```

如果输出版本号，说明安装成功。

## 步骤 5：运行 SimDecision

```bash
# 设置环境变量（可选）
export CONTROLLER_URL=http://localhost:8080
export PORT=3000

# 运行应用
python3 app.py
```

## 常见问题

### Q1: 下载的包是源码包，目标环境没有编译工具怎么办？

**A**: 有两个选择：
1. 在目标环境安装编译工具（见步骤 3.3）
2. 在相同架构的其他机器上编译 wheel 包，然后复制到目标环境

### Q2: 某些包下载失败怎么办？

**A**: 可以手动下载：
```bash
pip3 download --platform manylinux2014_aarch64 --python-version 311 --implementation cp --only-binary=:all: 包名 -d offline-pip-packages
```

### Q3: 如何确认下载的包是 ARM64 兼容的？

**A**: 检查文件名：
- wheel 包：文件名包含 `linux_aarch64` 或 `manylinux2014_aarch64`
- 源码包：文件名通常以 `.tar.gz` 结尾，不包含平台信息，可以在任何平台编译

### Q4: 目标环境的 Python 版本与下载时不同怎么办？

**A**: 可以尝试：
1. 使用相同版本的 Python 下载
2. 或者下载兼容的包（Python 3.x 之间通常兼容）

## 依赖说明

SimDecision 主要依赖：

- **Flask**：Web 框架
- **requests**：HTTP 客户端库

这些包的依赖关系会自动处理，`pip download` 会下载所有必需的依赖。

## 手动下载特定包

如果需要单独下载某个包：

```bash
# 下载 Flask
pip3 download --platform manylinux2014_aarch64 --python-version 311 --implementation cp --only-binary=:all: Flask -d offline-pip-packages

# 下载 requests
pip3 download --platform manylinux2014_aarch64 --python-version 311 --implementation cp --only-binary=:all: requests -d offline-pip-packages
```

## 注意事项

1. **架构匹配**：确保下载的包是 ARM64 架构的
2. **Python 版本**：确保目标环境的 Python 版本与下载时兼容
3. **编译工具**：如果下载的是源码包，需要编译工具
4. **系统库**：某些 Python 包可能依赖系统库，需要提前安装

