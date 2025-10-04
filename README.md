# Plum

分布式服务框架（MVP/Walking Skeleton）。当前包含：
- 控制面（Go，HTTP API，SQLite 持久化）
- 节点 Agent（C++17，libcurl，下载/解压/运行进程，上报状态，按 desired 调谐）
- Web UI（Vite + Vue 3 + TypeScript，Element Plus，路由化页面：Home/Assignments/Nodes/Apps/Deployments/Services）

## 1. 环境准备

建议平台：Linux 或 WSL2（Ubuntu）。

### 1.1 系统依赖
```bash
sudo apt update
sudo apt install -y build-essential cmake libcurl4-openssl-dev jq tree curl unzip
```

### 1.2 安装 Go（>= 1.22）
可用发行版包或官方二进制。以下为官方二进制方式（推荐，可复制粘贴执行）。
```bash
# 选择版本
GO_VER=1.22.6
arch=$(uname -m); case "$arch" in
  x86_64) GO_ARCH=amd64 ;;
  aarch64) GO_ARCH=arm64 ;;
  *) echo "unsupported arch: $arch"; exit 1 ;;
esac

# 下载并安装
curl -LO https://dl.google.com/go/go${GO_VER}.linux-${GO_ARCH}.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go${GO_VER}.linux-${GO_ARCH}.tar.gz

# 加 PATH
if ! grep -q "/usr/local/go/bin" ~/.bashrc; then echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc; fi
source ~/.bashrc

# 可选（国内更快）
# echo 'export GOPROXY=https://goproxy.cn,direct' >> ~/.bashrc && source ~/.bashrc

go version
```

### 1.3 安装 Node.js（UI 开发）
需要 Node.js 18+（建议 20 LTS）与 npm。可用 nvm 安装：
```bash
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
source ~/.bashrc
nvm install --lts
node -v && npm -v
```

## 2. 构建与运行

项目根目录：`/home/stone/code/Plum`。

### 2.1 控制面（Go）
```bash
cd /home/stone/code/Plum
make controller                 # 构建二进制到 controller/bin/controller
./controller/bin/controller     # 启动（默认 :8080）
# 健康检查
curl -s http://127.0.0.1:8080/healthz
```
自定义监听地址/数据目录：
```bash
CONTROLLER_ADDR=:9090 CONTROLLER_DATA_DIR=/home/stone/code/Plum ./controller/bin/controller
```

### 2.1.1 Swagger UI（API 说明与测试）
控制面启动后，访问：
- 浏览器打开 http://127.0.0.1:8080/swagger
- OpenAPI JSON: http://127.0.0.1:8080/swagger/openapi.json

### 2.2 Agent（C++）
```bash
cd /home/stone/code/Plum
make agent                      # CMake 配置 + 编译（输出 agent/build/plum_agent）
make agent-run                  # 运行（默认 AGENT_NODE_ID=nodeA，CONTROLLER_BASE=http://127.0.0.1:8080）
```
自定义环境变量：
```bash
AGENT_NODE_ID=nodeB CONTROLLER_BASE=http://127.0.0.1:8080 AGENT_DATA_DIR=/tmp/plum-agent ./agent/build/plum_agent
```

### 2.3 创建一个演示部署（用于看到分配）
控制面运行后，新开终端执行：
```bash
curl -s -XPOST http://127.0.0.1:8080/v1/deployments \
  -H 'Content-Type: application/json' \
  -d '{
    "name":"demo-deploy",
    "entries": [
      {
        "artifactUrl":"/artifacts/your_app.zip",
        "startCmd":"",
        "replicas":{"nodeA":1, "nodeB":1}
      }
    ]
  }' | jq .
```

### 2.4 Web UI（Vite + Vue 3 + TS）
```bash
cd /home/stone/code/Plum
make ui         # 安装依赖（首次）
make ui-dev     # 开发模式（默认端口 5173，若被占用会自动递增）
# 浏览器访问 http://127.0.0.1:5173 （或终端提示的实际端口）
```
主要路由：
- /home：系统总览（KPI 卡片 + ECharts：节点健康、各服务端点数）
- /assignments：展示期望(Desired) + 实际(Phase/Healthy/LastReportAt)，行级 Start/Stop
- /nodes：节点列表（删除会做在用检查）
- /apps：应用工件上传/列表/删除（在用检查，可下载 ZIP）
- /deployments：部署列表（进入详情/配置/删除）
- /deployments/create：创建部署（多条目/多节点副本，startCmd 可选，默认跑包内 ./start.sh）
- /deployments/:id：部署详情（行级 Start/Stop/删除，支持“全部停止/按节点停止”批量操作）
- /services：服务与端点视图（左侧服务列表，右侧端点明细）

### 2.5 服务注册与发现（最小版）
- 注册：Agent 启动应用后读取包内 `meta.ini` 中的 `service=` 行并注册端点；随后周期心跳。
- 发现：`GET /v1/discovery?service=orders&version=&protocol=http&limit=20`
- 主动健康：应用也可主动调用 `/v1/services/heartbeat` 覆盖端点健康（当前不启用探针）。

`meta.ini` 示例（可多行）：
```
name=demo-app
version=1.0.0
service=orders:http:8080
service=inventory:http:9090
```

## 3. 常用环境变量
- 控制面：
  - `CONTROLLER_ADDR`（默认 `:8080`）
  - `CONTROLLER_DATA_DIR`（默认 `.`，用于存放 artifacts 静态目录）
- Agent：
  - `AGENT_NODE_ID`（默认 `nodeA`）
  - `CONTROLLER_BASE`（默认 `http://127.0.0.1:8080`）
  - `AGENT_DATA_DIR`（默认 `/tmp/plum-agent`，实例工件与解压目录）
- UI：`VITE_API_BASE`（默认 `http://127.0.0.1:8080`）

## 4. 目录速览（关键路径）
```
Plum/
├─ controller/                 # 控制面（Go）
│  ├─ cmd/server/main.go       # 程序入口
│  ├─ internal/httpapi/        # 路由/处理器（nodes/apps/deployments/assignments/services/sse）
│  └─ internal/store/          # 存储接口与 SQLite 实现
├─ agent/                      # 节点 Agent（C++17）
│  ├─ CMakeLists.txt
│  └─ src/
│     ├─ main.cpp              # 心跳、SSE、拉取分配、调谐循环
│     ├─ http_client.hpp/.cpp  # libcurl 封装（GET/POST/DELETE）
│     ├─ reconciler.hpp/.cpp   # 下载/解压/进程监督/状态上报/服务注册
│     └─ fs_utils.hpp/.cpp     # 目录/解压辅助
├─ ui/                         # Web UI（Vite + Vue 3 + TS, Element Plus）
│  ├─ index.html
│  ├─ src/main.ts              # 应用入口
│  ├─ src/router.ts            # 路由
│  ├─ src/App.vue              # 布局+菜单
│  └─ src/views/*              # Home/Assignments/Nodes/Apps/Deployments/Services
├─ docs/assistant/POSTCARD.md  # AI 协作“项目明信片”
├─ tools/make_sync_packet.sh   # 生成 sync_packet.txt（上下文同步）
└─ Makefile                    # 顶层便捷命令
```

## 5. 当前进度与 Roadmap

### 5.1 已完成（阶段性）
- 控制面：
  - SQLite 存储（nodes/deployments/assignments/statuses/artifacts/endpoints）
  - Deployments：创建/列表/详情/配置（labels PATCH 不改名）/删除（级联清理 assignments/statuses）
  - Assignments：按节点获取（含 desired 与最新 Phase/Healthy/LastReportAt）、PATCH desired（Running/Stopped）、DELETE 单条
  - Services（注册与发现）：
    - 注册/替换端点：POST `/v1/services/register`
    - 端点心跳与健康覆盖：POST `/v1/services/heartbeat`
    - 删除实例端点：DELETE `/v1/services?instanceId=`
    - 发现：GET `/v1/discovery?service=&version=&protocol=&limit=`；服务列表：GET `/v1/services/list`
  - SSE：节点级事件流 `/v1/stream?nodeId=`（Agent/前端使用，Start/Stop 等操作秒级生效）
  - 故障迁移：节点不健康时，将该节点上 desired=Running 的实例随机迁移到健康节点
  - Swagger UI：`/swagger`（OpenAPI 文档统一 Deployment 术语）
- Agent：
  - 周期拉取 assignments + SSE 推送唤醒（更实时）；解析 `meta.ini` 中 `service=` 行并注册端点；定期心跳
  - 停止实例时，自动删除对应端点；SIGINT/SIGTERM 优雅退出并清理子进程
  - 修复 JSON 解析导致 `startCmd` 丢失 `/` 的问题
- Web UI：
  - 导航：Home/Assignments/Nodes/Apps/Deployments/Services（不再折叠 More）
  - Home：KPI 卡片 + ECharts（节点健康环图、各服务端点数柱状）
  - Deployments：列表/创建/详情/配置（统一 Deployment 文案与路由）
  - Assignments：实时刷新（SSE）与行级操作；列展示 Deployment
  - Apps：ZIP 上传/列表/删除；Artifact 下载链接修复（支持 `VITE_API_BASE`）

### 5.2 待办（下一阶段）
- 强化健康：端点/实例健康探针（HTTP/TCP），心跳超时 Unknown/Unhealthy
- 故障恢复策略：自动回切/再均衡控制器（稳定期与限速）
- 单点解析接口：`/v1/resolve?service=...&strategy=random|round_robin&hashKey=...`
- 更细化的权限与用户体系（RBAC）
- 进程日志采集与查看（本地/集中式）
- 更完备的错误处理与前端提示（把 409 等转换成友好文案）
- 系统级 SSE（汇总事件），Home 实时增量刷新
- Swagger 注释生成：接入 `swaggo/swag`（注解 → OpenAPI）
 - 新“Task（短作业）/编排”技术路线（分阶段）：
   - 阶段A（MVP）：最小任务模型与API（/v1/tasks），Task/Attempt 状态机（Pending→Running→Succeeded/Failed），存储与查询，SSE 推送
   - 阶段B：执行器 Service/Process 双通道（服务端点调用或 Agent 进程执行），调度（健康+标签+随机），超时/取消/重试（退避），幂等键
   - 阶段C：编排（顺序/并行/条件/DAG），定时/事件触发，限流/优先级/配额，UI 可视化与审计
   - 阶段D：与工作流引擎对接（如 Temporal/Conductor）或引入代码式编排 SDK
  - Worker 选择策略（预研/规划，当前实现为“选择首个匹配的 Worker”）：
    - 随机 / 轮询（round-robin）选择可执行该任务的 Worker
    - 基于容量/负载（capacity-aware）选择：优先空闲度高或队列短的 Worker
    - 基于标签/约束/亲和/反亲和：匹配 `labels`、节点属性、地理就近
    - 粘性调度与一致性哈希（hashKey）：同源任务命中同一 Worker，提高缓存命中
    - 广播/并发执行模式（fan-out）：对所有或前 N 个 Worker 并发调用，最先成功即返回
    - 健康/超时/熔断/重试策略：失败退避、熔断降级、黑白名单
    - 优先级/配额/速率限制：不同租户/任务类别的公平性与隔离

### 5.3 设计原则
- 声明式：控制面描述“期望状态”，Agent 对齐“实际状态”。
- 可进化：先内存/SQLite，后迁移 etcd；先进程，后容器；先单机，后 HA。
- 可移植：抽象 `IRuntime`/`IProcessSupervisor`/`ISystemMetrics`，避免平台绑定。
- 可观测：默认指标/日志/事件可查询；问题可追踪、可告警。
- 安全默认开启：mTLS、RBAC、审计日志。

### 5.4 里程碑（摘要）
- M1 基础部署：副本调度、进程监督、健康检查、日志
- M2 服务注册与发现 + 基础监控/告警
- M3 持久化与 HA：etcd/Consul、多副本、幂等控制
- M4 调度与编排：亲和/反亲和、滚动/金丝雀/蓝绿发布

### 5.5 风险与约束
- 保持日志与大对象远离 etcd（仅存元数据/小配置）
- 所有控制面操作需可重放/幂等；明确超时/重试策略
- 弱网与边缘环境：优先 Pull/心跳、流量压缩与批量上报

## 6. API 速查（当前实现）
- Apps：
  - POST `/v1/apps/upload`（multipart: file=zip）
  - GET `/v1/apps`
  - DELETE `/v1/apps/{id}`（若在用，409）
- Nodes：
  - POST `/v1/nodes/heartbeat` → `{ ttlSec }`
  - GET `/v1/nodes`、GET `/v1/nodes/{id}`、DELETE `/v1/nodes/{id}`（若在用，409）
- Deployments：
  - POST `/v1/deployments`（支持 `entries[]`；`startCmd` 可选，缺省跑 `./start.sh`）
  - GET `/v1/deployments`、GET `/v1/deployments/{id}`（含 assignments）
  - PATCH `/v1/deployments/{id}`（更新 labels；名称不可改）
  - DELETE `/v1/deployments/{id}`（级联清理 assignments/statuses）
- Assignments：
  - GET `/v1/assignments?nodeId=`（返回 desired + 最新 Phase/Healthy/LastReportAt）
  - PATCH `/v1/assignments/{instanceId}` `{ desired: Running|Stopped }`
  - DELETE `/v1/assignments/{instanceId}`
- SSE：
  - GET `/v1/stream?nodeId=` → 事件：`update`
- Services（注册/发现）：
  - POST `/v1/services/register`、POST `/v1/services/heartbeat`、DELETE `/v1/services?instanceId=`
  - GET `/v1/discovery?service=&version=&protocol=&limit=`，GET `/v1/services/list`
\- Tasks（短作业）：
  - POST `/v1/tasks`、GET `/v1/tasks`、DELETE `/v1/tasks/{id}`、POST `/v1/tasks/start/{id}`、POST `/v1/tasks/rerun/{id}`、POST `/v1/tasks/cancel/{id}`
  - SSE `/v1/tasks/stream`
  - 执行器：
    - embedded：控制面调用 Worker URL（POST，Body: `{ taskId, name, payload }`）
    - service：从服务注册表选择健康端点并调用固定路径（默认 `/task`）
      - 可通过任务 `labels` 覆盖：
        - `serviceVersion`: 过滤服务版本（如 `1.0.0`）
        - `serviceProtocol`: 指定 `http|https`
        - `servicePort`: 覆盖端口号（如 `8080`）
        - `servicePath`: 指定调用路径（自动补 `/`）
    - os_process：由 Agent 启动外部进程（规划中）
