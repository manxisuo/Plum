# Plum 环境变量配置完整指南

Plum支持通过`.env`文件统一管理所有配置，无需手动设置环境变量。

## 📋 配置优先级

```
环境变量 > .env文件 > 默认值
```

**说明**：
- 环境变量的优先级最高，可以临时覆盖 `.env` 文件
- `.env` 文件便于持久化配置，适合生产环境
- 默认值确保即使没有配置也能正常运行

## 🚀 快速开始

### 1. Controller

```bash
cd controller
cp env.example .env
vim .env  # 修改配置

# 启动时自动加载.env
./bin/controller
```

### 2. Agent

```bash
cd agent-go
cp env.example .env
vim .env  # 修改配置

# 启动时自动加载.env
./plum-agent
```

### 3. 应用（C++ SDK）

```bash
cd examples/kv-demo
cp ../../sdk/cpp/plumkv/env.example .env
vim .env  # 修改配置

# SDK自动加载.env
./kv-demo
```

---

## 📖 Controller 配置项

配置文件位置：`controller/.env`

### 服务配置

| 变量名 | 说明 | 默认值 | 示例 |
|--------|------|--------|------|
| `CONTROLLER_ADDR` | Controller监听地址 | `:8080` | `:8080`, `0.0.0.0:9090` |
| `CONTROLLER_DB` | 数据库路径 | `file:controller.db?_pragma=busy_timeout(5000)` | `file:/var/lib/plum/controller.db` |
| `CONTROLLER_DATA_DIR` | 数据目录（存放artifacts） | `.` | `/var/lib/plum/data` |

### 调度配置

| 变量名 | 说明 | 默认值 | 示例 |
|--------|------|--------|------|
| `TASK_SCHED_INTERVAL_SEC` | 任务调度器间隔（秒） | `1` | `1`, `5` |
| `TASK_EMBEDDED_TIMEOUT_MS` | 嵌入式任务默认超时（毫秒） | `30000` | `30000`, `60000` |

### 故障转移配置

| 变量名 | 说明 | 默认值 | 示例 |
|--------|------|--------|------|
| `HEARTBEAT_TTL_SEC` | 节点心跳超时时间（秒） | `3` | `3`, `30` |
| `FAILOVER_INTERVAL_SEC` | 故障转移检查间隔（秒） | `1` | `1`, `10` |
| `AUTO_MIGRATION_ENABLED` | 是否启用自动迁移（节点故障时迁移应用） | `false` | `true`, `false` |

### 性能监控配置

| 变量名 | 说明 | 默认值 | 示例 |
|--------|------|--------|------|
| `PERFORMANCE_MONITORING_ENABLED` | 是否启用性能监控 | `true` | `true`, `false` |
| `RESTART_TIME_THRESHOLD_MS` | 重启时间阈值（毫秒） | `2000` | `2000` |
| `MIGRATION_TIME_THRESHOLD_MS` | 迁移时间阈值（毫秒） | `2000` | `2000` |
| `DETECTION_TIME_THRESHOLD_MS` | 检测时间阈值（毫秒） | `1000` | `1000` |

### 弱网环境支持配置

| 变量名 | 说明 | 默认值 | 示例 |
|--------|------|--------|------|
| `WEAK_NETWORK_ENABLED` | 弱网支持启用 | `true` | `true`, `false` |
| `REQUEST_TIMEOUT` | 请求超时 | `30s` | `30s`, `60s` |
| `READ_TIMEOUT` | 读取超时 | `10s` | `10s` |
| `MAX_IDLE_CONNS` | 最大空闲连接数 | `100` | `100`, `200` |
| `RATE_LIMIT_RPS` | 限流每秒请求数 | `1000` | `1000`, `5000` |
| `CIRCUIT_BREAKER_ENABLED` | 熔断器启用 | `true` | `true`, `false` |
| `RETRY_MAX_ATTEMPTS` | 最大重试次数 | `3` | `3`, `5` |

**完整列表**：详见 `controller/env.example`

---

## 📖 Agent 配置项

配置文件位置：`agent-go/.env`

### 节点配置

| 变量名 | 说明 | 默认值 | 示例 |
|--------|------|--------|------|
| `AGENT_NODE_ID` | 节点ID（唯一标识） | `nodeA` | `nodeA`, `nodeB`, `worker-01` |
| `CONTROLLER_BASE` | Controller地址（建议在 `/etc/hosts` 中将 `plum-controller` 指向 Controller IP） | `http://plum-controller:8080` | `http://plum-controller:8080`, `http://controller.internal:8080` |
| `AGENT_IP` | Agent对外通告的IP（心跳与服务注册使用） | `127.0.0.1` | `192.168.1.101`, `10.0.0.5` |
| `AGENT_DATA_DIR` | Agent数据目录 | `/tmp/plum-agent` | `/var/lib/plum-agent`, `/app/data` |

### 应用运行模式配置

| 变量名 | 说明 | 可选值 | 默认值 |
|--------|------|--------|--------|
| `AGENT_RUN_MODE` | 应用运行模式 | `process`, `docker` | `process` |

**说明**：
- `process`：应用以进程方式运行（传统模式）
- `docker`：应用以Docker容器方式运行（需要Docker daemon运行）

### 容器模式配置（仅当 `AGENT_RUN_MODE=docker` 时生效）

| 变量名 | 说明 | 默认值 | 示例 | 备注 |
|--------|------|--------|------|------|
| `PLUM_BASE_IMAGE` | 容器基础镜像 | `alpine:latest` | `ubuntu:22.04`, `python:3.11` | 所有应用容器基于此镜像 |
| `PLUM_CONTAINER_MEMORY` | 容器内存限制 | 无限制 | `512m`, `1g`, `2048` | 格式：`512m`(MB), `1g`(GB), `2048`(字节) |
| `PLUM_CONTAINER_CPUS` | 容器CPU限制 | 无限制 | `1.0`, `2`, `0.5` | 格式：CPU核数 |
| `PLUM_CONTAINER_ENV` | 容器自定义环境变量 | 无 | `DISPLAY=:99`, `DISPLAY=:99,QT_QPA_PLATFORM=xcb` | 格式：`KEY1=value1,KEY2=value2` |
| `PLUM_HOST_LIB_PATHS` | 宿主机库路径映射 | 无 | `/usr/lib,/usr/local/lib`, `/opt/qt/lib` | 格式：`/path1,/path2,/path3`，只读挂载 |

**特殊说明**：
- `LD_LIBRARY_PATH`：如果应用目录有 `lib/` 子目录，Agent会自动添加 `LD_LIBRARY_PATH=/app/lib:/usr/lib:/lib`
  - 这对Qt应用很有用，可以将Qt库放在 `lib/` 目录中
- `PLUM_CONTAINER_ENV`：用于传递应用需要的特殊环境变量
  - 例如Qt应用可能需要：`DISPLAY=:99` 或 `QT_QPA_PLATFORM=xcb`
- `PLUM_HOST_LIB_PATHS`：将宿主机的库路径只读挂载到容器内
  - 适用于多个应用共享相同的系统库（如Qt库）
  - 避免每个应用都自包含库，减少重复存储
  - 注意：需要确保宿主机和容器架构兼容（都是x86_64或都是ARM64）

---

## 📖 SDK 配置项

### KV SDK 配置（`sdk/cpp/plumkv/env.example`）

| 变量名 | 说明 | 默认值 | 示例 |
|--------|------|--------|------|
| `CONTROLLER_BASE` | Controller地址 | `http://plum-controller:8080` | `http://plum-controller:8080`, `http://controller.internal:8080` |
| `PLUM_KV_SYNC_MODE` | KV同步模式 | `polling` | `polling`, `sse`, `disabled` |

**同步模式说明**：
- `polling`：轮询模式（默认），定期拉取更新
- `sse`：Server-Sent Events，实时推送更新
- `disabled`：禁用同步（仅本地缓存）

### Resource SDK 配置（`sdk/cpp/plumresource/env.example`）

| 变量名 | 说明 | 默认值 | 示例 |
|--------|------|--------|------|
| `CONTROLLER_BASE` | Controller地址 | `http://plum-controller:8080` | `http://plum-controller:8080`, `http://controller.internal:8080` |
| `RESOURCE_ID` | 资源ID | 自动生成 | `sensor-001` |
| `RESOURCE_NODE_ID` | 节点ID | 主机名 | `nodeA` |

---

## 📖 Agent 注入的环境变量（应用运行时）

Agent会自动向应用注入以下环境变量（无需在`.env`中配置）：

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `PLUM_INSTANCE_ID` | 实例ID（每个实例唯一） | `3504f3c73a6aa13a14547f078799a9ec-5ffb69d9` |
| `PLUM_APP_NAME` | 应用名称 | `demo-app`, `my-service` |
| `PLUM_APP_VERSION` | 应用版本 | `1.0.0`, `v2.1.3` |

**说明**：这些变量由Agent在启动应用时自动注入，应用可以直接使用。

---

## 📝 完整配置示例

### Controller 配置示例

```bash
# controller/.env
# ========== 服务配置 ==========
CONTROLLER_ADDR=:8080
CONTROLLER_DB=file:controller.db?_pragma=busy_timeout(5000)
CONTROLLER_DATA_DIR=.

# ========== 故障转移配置 ==========
HEARTBEAT_TTL_SEC=3
FAILOVER_INTERVAL_SEC=1
AUTO_MIGRATION_ENABLED=false

# ========== 调度配置 ==========
TASK_SCHED_INTERVAL_SEC=1
TASK_EMBEDDED_TIMEOUT_MS=30000
```

### Agent 配置示例（进程模式）

```bash
# agent-go/.env
# ========== 节点配置 ==========
AGENT_NODE_ID=nodeA
CONTROLLER_BASE=http://plum-controller:8080
AGENT_IP=192.168.1.10
AGENT_DATA_DIR=/tmp/plum-agent

# ========== 应用运行模式 ==========
AGENT_RUN_MODE=process  # 进程模式（默认）
```

### Agent 配置示例（容器模式）

```bash
# agent-go/.env
# ========== 节点配置 ==========
AGENT_NODE_ID=nodeA
CONTROLLER_BASE=http://plum-controller:8080
AGENT_IP=192.168.1.10
AGENT_DATA_DIR=/tmp/plum-agent

# ========== 应用运行模式 ==========
AGENT_RUN_MODE=docker  # 容器模式

# ========== 容器模式配置 ==========
PLUM_BASE_IMAGE=ubuntu:22.04  # 使用Ubuntu镜像（适合Qt应用）
PLUM_CONTAINER_MEMORY=512m     # 内存限制
PLUM_CONTAINER_CPUS=1.0        # CPU限制
PLUM_CONTAINER_ENV=DISPLAY=:99  # 自定义环境变量（Qt应用需要）
```

### Agent 配置示例（Qt应用专用）

```bash
# agent-go/.env
AGENT_RUN_MODE=docker
PLUM_BASE_IMAGE=ubuntu:22.04  # Qt应用需要更完整的系统库
PLUM_CONTAINER_ENV=DISPLAY=:99,QT_QPA_PLATFORM=xcb
```

### SDK 配置示例（KV存储）

```bash
# examples/kv-demo/.env
CONTROLLER_BASE=http://plum-controller:8080
PLUM_KV_SYNC_MODE=polling  # 或 sse（实时推送）
```

---

## ⚙️ 实现细节

### Go组件（Controller/Agent）

使用 `github.com/joho/godotenv` 加载配置：

```go
import "github.com/joho/godotenv"

func main() {
    // 自动查找并加载 .env 文件
    // 查找顺序：程序目录 → 程序上级目录 → 当前工作目录
    godotenv.Load()
    
    // 读取配置
    value := os.Getenv("CONFIG_KEY")
}
```

**查找顺序**：
1. `{程序目录}/.env`（如 `agent-go/.env`）
2. `{程序上级目录}/.env`（如项目根目录 `.env`）
3. `{当前工作目录}/.env`

### C++ SDK

内置简单`.env`解析器（`sdk/cpp/env_loader.hpp`）：

```cpp
// SDK在初始化时自动加载 .env 文件
auto kv = DistributedMemory::create("namespace");
// 自动读取 CONTROLLER_BASE, PLUM_KV_SYNC_MODE 等配置
```

---

## 💡 配置技巧

### 1. 使用注释组织配置

```bash
# ========== 节点配置 ==========
AGENT_NODE_ID=nodeA
CONTROLLER_BASE=http://plum-controller:8080

# ========== 容器模式配置 ==========
AGENT_RUN_MODE=docker
PLUM_BASE_IMAGE=ubuntu:22.04
```

### 2. 使用引号处理特殊字符

```bash
# 值中包含空格或特殊字符时使用引号
CONTROLLER_ADDR=":8080"
PLUM_CONTAINER_ENV="DISPLAY=:99,QT_QPA_PLATFORM=xcb"
```

### 3. 临时覆盖配置（使用环境变量）

```bash
# 临时使用不同的配置，不修改.env文件
AGENT_RUN_MODE=docker PLUM_BASE_IMAGE=alpine:latest ./agent-go/plum-agent
```

### 4. 不同环境使用不同配置

```bash
# 开发环境
cp env.example .env.dev
vim .env.dev

# 生产环境
cp env.example .env.prod
vim .env.prod

# 运行时指定
ENV_FILE=.env.prod ./agent-go/plum-agent  # 需要代码支持
```

---

## ⚠️ 注意事项

1. **`.env` 文件不会被提交到Git**
   - 已在 `.gitignore` 中排除
   - 保护敏感信息

2. **`env.example` 是模板**
   - 会被提交到Git
   - 包含所有配置项的示例和说明

3. **环境变量优先级更高**
   - 可用于临时覆盖 `.env` 配置
   - 适合测试和调试

4. **支持注释**
   - 以 `#` 开头的行会被忽略
   - 用于说明和分组

5. **支持引号**
   - `KEY="value"` 或 `KEY='value'`
   - 值中包含空格时必须使用

6. **容器模式前置条件**
   - `AGENT_RUN_MODE=docker` 时，需要：
     - Docker daemon 运行
     - Agent 有权限访问 Docker socket
     - 基础镜像已拉取（如 `docker pull ubuntu:22.04`）

---

## 🔍 配置验证

### 检查配置是否加载

```bash
# Agent 启动时会输出加载的配置
# 查看日志中的 "Loaded configuration from ..."
./agent-go/plum-agent
# 应该看到：Loaded configuration from /path/to/.env
```

### 检查实际使用的配置值

```bash
# Agent 启动日志会显示关键配置
./agent-go/plum-agent
# 应该看到：
# Starting Plum Agent
#   NodeID: nodeA
#   Controller: http://plum-controller:8080
#   DataDir: /tmp/plum-agent
# Using app run mode: docker  # 如果是容器模式
```

---

## 📚 相关文档

- [容器应用管理](./CONTAINER_APP_MANAGEMENT.md) - 容器模式详细说明
- [Qt应用容器运行指南](./QT_APP_IN_CONTAINER.md) - Qt应用配置
- [部署状态](./DEPLOYMENT_STATUS.md) - 部署方式说明
- [生产环境部署](../deploy/PRODUCTION_DEPLOYMENT.md) - 生产配置建议

---

## 📋 配置项快速索引

### Controller
- [服务配置](#服务配置)
- [调度配置](#调度配置)
- [故障转移配置](#故障转移配置)
- [性能监控配置](#性能监控配置)
- [弱网环境支持配置](#弱网环境支持配置)

### Agent
- [节点配置](#节点配置)
- [应用运行模式配置](#应用运行模式配置)
- [容器模式配置](#容器模式配置仅当-agent_run_modedocker-时生效)

### SDK
- [KV SDK配置](#kv-sdk-配置)
- [Resource SDK配置](#resource-sdk-配置)

---

**最后更新**：2025-11-02（添加容器模式配置）

