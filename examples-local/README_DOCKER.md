# FSL 项目 Docker 镜像构建指南（通用版）

## 概述

本项目提供了通用的 Docker 镜像构建脚本，可以用于所有 FSL 项目（FSL_Sweep、FSL_Destroy、FSL_Investigate、FSL_Evaluate 等）。

## 快速开始

### 构建单个项目

```bash
# 从项目根目录执行
cd /path/to/Plum

# 构建 FSL_Sweep
./examples-local/build-docker-local.sh FSL_Sweep

# 构建 FSL_Destroy
./examples-local/build-docker-local.sh FSL_Destroy

# 构建 FSL_Investigate
./examples-local/build-docker-local.sh FSL_Investigate

# 构建 FSL_Evaluate
./examples-local/build-docker-local.sh FSL_Evaluate

# 构建 Sim 项目（统一使用 make）
./examples-local/build-docker-local.sh SimRoutePlan
./examples-local/build-docker-local.sh SimNaviControl
```

### 构建所有 FSL 项目

```bash
# 先编译所有项目
make examples_FSL_All

# 然后依次构建镜像
for app in FSL_Sweep FSL_Destroy FSL_Investigate FSL_Evaluate FSL_Plan FSL_Statistics; do
    ./examples-local/build-docker-local.sh $app
done
```

## 工作原理

通用构建脚本会自动：

1. **检查项目是否已编译**：如果未编译，会自动执行 `make examples_<项目名>`
   - 所有项目（FSL 和 Sim）都使用统一的 Makefile 规则
   - 所有项目的可执行文件都输出到 `examples-local/<项目名>/bin/<项目名>`
2. **复制依赖库**：使用 `copy-deps.sh` 脚本复制所有必需的库文件
3. **构建 Docker 镜像**：使用通用的 `Dockerfile.local.template` 或项目特定的 `Dockerfile.local`

**注意**：FSL 和 Sim 项目现在使用完全相同的构建流程，无需区别对待。

## 文件说明

### 通用脚本

- **`build-docker-local.sh`**：通用的构建脚本，接受项目名作为参数
- **`copy-deps.sh`**：通用的依赖复制脚本，自动识别并复制所有必需的库
- **`Dockerfile.local.template`**：通用的 Dockerfile 模板

### 项目特定文件（可选）

如果某个项目需要特殊的配置，可以在项目目录下创建：

- **`<项目名>/Dockerfile.local`**：项目特定的 Dockerfile（会优先使用）
- **`<项目名>/bin/start.sh`**：启动脚本（如果不存在，会使用默认的）
- **`<项目名>/bin/meta.ini`**：应用元数据（如果不存在，会跳过）

## 手动构建（高级用法）

如果需要更多控制，可以手动执行：

```bash
# 1. 编译项目
make examples_FSL_Sweep

# 2. 准备依赖
./examples-local/copy-deps.sh FSL_Sweep /tmp/fsl-sweep-deps

# 3. 构建镜像
docker buildx build \
  --platform linux/arm64 \
  --load \
  -f examples-local/Dockerfile.local.template \
  --build-arg APP_NAME=FSL_Sweep \
  -t fsl-sweep:1.0.0 \
  /tmp/fsl-sweep-deps

# 4. 清理
rm -rf /tmp/fsl-sweep-deps
```

## 测试镜像

```bash
# 测试 FSL_Sweep
docker run --rm fsl-sweep:1.0.0

# 测试 FSL_Destroy
docker run --rm fsl-destroy:1.0.0
```

## 在 Plum 中使用

1. 在 Plum Web UI 的 `/apps` 页面，点击"使用镜像创建"
2. 填写信息：
   - 应用名称：`FSL_Sweep`（或其他项目名）
   - 版本：`1.0.0`
   - 镜像仓库：`fsl-sweep`（镜像名会自动转换为小写）
   - 镜像标签：`1.0.0`
   - 启动命令：留空（使用镜像默认的 `./start.sh`）

## 支持的项目

### FSL 项目（使用 Makefile 构建）
- ✅ FSL_Sweep
- ✅ FSL_Destroy
- ✅ FSL_Investigate
- ✅ FSL_Evaluate
- ✅ FSL_Plan
- ✅ FSL_Statistics

### Sim 项目（使用 qmake 构建）
- ✅ SimRoutePlan
- ✅ SimNaviControl
- ✅ SimSonar
- ✅ SimTargetHit
- ✅ SimTargetRecognize
- ✅ SimDecision

## 注意事项

1. **架构要求**：脚本会自动使用 `--platform linux/arm64`，确保在树莓派上构建正确的镜像
2. **网络要求**：首次构建需要下载 Ubuntu 24.04 基础镜像，后续构建会使用缓存
3. **依赖库**：脚本会自动识别并复制所有必需的库文件，包括 gRPC、protobuf 等
4. **GLIBC 版本**：使用 Ubuntu 24.04 作为基础镜像，确保与宿主机（Ubuntu 24.04）的 GLIBC 版本匹配。Agent 会自动添加 `--security-opt seccomp=unconfined` 以支持旧版 Docker 环境

## 故障排除

### 问题：找不到项目

**错误**：`错误: 项目目录不存在: examples-local/FSL_XXX`

**解决**：确保项目名拼写正确，项目目录存在于 `examples-local/` 下

### 问题：项目未编译

**错误**：`错误: FSL_XXX 未编译`

**解决**：运行 `make examples_<项目名>` 先编译项目

### 问题：镜像构建失败

**错误**：`ERROR: failed to build`

**解决**：
1. 检查网络连接（首次构建需要下载基础镜像）
2. 确保有足够的磁盘空间
3. 检查 Docker 是否正常运行：`docker info`

## 与项目特定脚本的兼容性

如果某个项目已经有自己的 `build-docker-local.sh` 脚本（如 FSL_Sweep），可以继续使用项目特定的脚本，或者迁移到通用脚本。

通用脚本会优先查找项目特定的 `Dockerfile.local`，如果不存在则使用通用模板。

