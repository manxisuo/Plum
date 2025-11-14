# FSL_Sweep Docker 镜像构建指南

## 前置条件

1. 确保已安装 Docker
2. 确保项目已编译（`make examples_FSL_Sweep`）

## 构建方式

### 方式 1：使用宿主机构建产物（推荐，适合离线环境）

此方式使用宿主机上已编译好的可执行文件和库，不需要在 Docker 中下载和编译依赖。

```bash
# 从项目根目录执行
cd /path/to/Plum

# 使用自动化脚本（推荐）
./examples-local/FSL_Sweep/build-docker-local.sh
```

或者手动执行：

```bash
# 1. 准备依赖文件
./examples-local/FSL_Sweep/copy-deps.sh /tmp/fsl-sweep-deps

# 2. 构建镜像
docker build -f examples-local/FSL_Sweep/Dockerfile.local -t fsl-sweep:1.0.0 /tmp/fsl-sweep-deps

# 3. 清理临时文件
rm -rf /tmp/fsl-sweep-deps
```

**优点**：
- ✅ 不需要网络连接（完全离线）
- ✅ 构建速度快（不需要编译）
- ✅ 使用宿主机的编译环境，避免版本冲突
- ✅ 适合跨架构构建（在树莓派上构建，在 Kylin 上使用）

### 方式 2：在 Docker 中完整构建（需要网络）

此方式在 Docker 容器中从源码编译，需要下载所有依赖。

```bash
# 从项目根目录执行
cd /path/to/Plum
docker build -f examples-local/FSL_Sweep/Dockerfile -t fsl-sweep:1.0.0 .
```

**注意**：此方式需要较长时间（可能需要 30 分钟以上），因为需要从源码构建 gRPC。

## 测试镜像

```bash
# 运行容器（需要设置必要的环境变量）
docker run --rm \
  -e WORKER_ID=fsl-sweep-test \
  -e WORKER_NODE_ID=nodeA \
  -e PLUM_INSTANCE_ID=test-001 \
  -e PLUM_APP_NAME=FSL_Sweep \
  -e PLUM_APP_VERSION=1.0.0 \
  fsl-sweep:1.0.0
```

## 在 Plum 中使用

1. 在 Plum Web UI 的 `/apps` 页面，点击"使用镜像创建"
2. 填写信息：
   - 应用名称：`FSL_Sweep`
   - 版本：`1.0.0`
   - 镜像仓库：`fsl-sweep`（或你的镜像仓库地址）
   - 镜像标签：`1.0.0`
   - 启动命令：留空（使用镜像默认的 `./start.sh`）

## 跨架构构建（树莓派 → Kylin）

如果需要在树莓派（ARM）上构建镜像，然后在 Kylin（ARM64）上使用：

1. **架构兼容性**：
   - 树莓派通常是 ARM32 或 ARM64
   - Kylin V10 通常是 ARM64
   - 如果架构相同（都是 ARM64），可以直接使用
   - 如果架构不同，需要交叉编译

2. **使用方式 1（推荐）**：
   ```bash
   # 在树莓派上
   cd /path/to/Plum
   make examples_FSL_Sweep  # 编译 ARM 版本
   ./examples-local/FSL_Sweep/build-docker-local.sh
   
   # 导出镜像
   docker save fsl-sweep:1.0.0 | gzip > fsl-sweep-arm64.tar.gz
   
   # 传输到 Kylin 并导入
   # scp fsl-sweep-arm64.tar.gz user@kylin:/tmp/
   # 在 Kylin 上: docker load < fsl-sweep-arm64.tar.gz
   ```

3. **使用方式 2**：
   - 在 Kylin 上直接构建（如果 Kylin 有编译环境）
   - 或者使用 Docker buildx 进行交叉编译

## 注意事项

1. **依赖库路径**：
   - 使用 `Dockerfile.local` 时，依赖库由 `copy-deps.sh` 自动收集
   - 确保宿主机上的库文件完整（特别是 gRPC 和 protobuf 相关库）

2. **gRPC 连接**：确保容器能够访问 Controller 的 gRPC 服务（默认端口 9090）

3. **环境变量**：Agent 会自动设置以下环境变量：
   - `PLUM_INSTANCE_ID`
   - `PLUM_APP_NAME`
   - `PLUM_APP_VERSION`
   - `WORKER_ID`（如果 Agent 设置了）
   - `WORKER_NODE_ID`（如果 Agent 设置了）

4. **架构匹配**：
   - 确保镜像的架构与运行环境匹配
   - 使用 `docker inspect fsl-sweep:1.0.0 | grep Architecture` 检查镜像架构

## 优化建议

1. **使用更小的基础镜像**：可以考虑使用 `alpine` 或 `distroless` 镜像来减小镜像大小
2. **静态链接**：如果可能，使用静态链接来减少运行时依赖
3. **多架构支持**：如果需要支持 ARM64，可以使用 `docker buildx` 构建多架构镜像

