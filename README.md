# Plum

分布式服务框架（MVP/Walking Skeleton）。当前包含：
- 控制面（Go，HTTP API，SQLite 持久化）
- 节点 Agent（C++17，libcurl，下载/解压/运行进程，上报状态，按 desired 调谐）
- Web UI（Vite + Vue 3 + TypeScript，Element Plus，路由化页面：Assignments/Nodes/Apps/Tasks）

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
AGENT_NODE_ID=nodeA CONTROLLER_BASE=http://127.0.0.1:8080 AGENT_DATA_DIR=/tmp/plum-agent ./agent/build/plum_agent
```

### 2.3 创建一个演示任务（用于看到分配）
在控制面运行后，新开终端执行：
```bash
curl -s -XPOST http://127.0.0.1:8080/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "name":"demo-task",
    "entries": [
      {
        "artifactUrl":"/artifacts/your_app.zip",
        "startCmd":"",
        "replicas":{"nodeA":1}
      }
    ]
  }' | jq .
```

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

### 2.4 Web UI（Vite + Vue 3 + TS）
```bash
cd /home/stone/code/Plum
make ui         # 安装依赖（首次）
make ui-dev     # 开发模式（默认端口 5173）
# 浏览器访问 http://127.0.0.1:5173
```
主要路由：
- /assignments：展示期望(Desired) + 实际(Phase/Healthy/LastReportAt)，行级 Start/Stop
- /nodes：节点列表（删除会做在用检查）
- /apps：应用工件上传/列表/删除（在用检查）
- /tasks：任务列表（进入详情/配置/删除）
- /tasks/create：创建任务（多条目，多节点副本，startCmd 可选，默认跑包内 ./start.sh）
- /tasks/:id：任务详情（行级 Start/Stop/删除，支持“全部停止/按节点停止”批量操作）

如果控制面不在默认地址，可在 UI 启动时指定：
```bash
cd ui
VITE_API_BASE=http://127.0.0.1:8080 npm run dev
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
│  ├─ internal/httpapi/        # 路由/处理器（nodes/apps/tasks/assignments）
│  └─ internal/store/          # 存储接口与 SQLite 实现
├─ agent/                      # 节点 Agent（C++17）
│  ├─ CMakeLists.txt
│  └─ src/
│     ├─ main.cpp              # 心跳、拉取分配、调谐循环
│     ├─ http_client.hpp/.cpp  # libcurl 封装
│     ├─ reconciler.hpp/.cpp   # 下载/解压/进程监督/状态上报
│     └─ fs_utils.hpp/.cpp     # 目录/解压辅助
├─ ui/                         # Web UI（Vite + Vue 3 + TS, Element Plus）
│  ├─ index.html
│  ├─ src/main.ts              # 应用入口
│  ├─ src/router.ts            # 路由
│  ├─ src/App.vue              # 布局+菜单
│  └─ src/views/*              # Assignments/Nodes/Apps/TaskList/TaskDetail/TaskCreate/TaskConfig
├─ docs/assistant/POSTCARD.md  # AI 协作“项目明信片”
├─ tools/make_sync_packet.sh   # 生成 sync_packet.txt（上下文同步）
└─ Makefile                    # 顶层便捷命令
```

## 5. 当前进度与 Roadmap

### 5.1 已完成（阶段性）
- 控制面：
  - SQLite 存储（nodes/tasks/assignments/statuses/artifacts）
  - Apps：上传/列表/删除（删除前检查是否被 assignments 引用，409）
  - Nodes：列表/详情/删除（删除前检查是否被 assignments 引用，409）；心跳注册
  - Tasks：创建（支持 entries[] 多条目，startCmd 可选，默认 ./start.sh）、列表、详情、删除（级联清理 assignments/statuses）、限制名称不可修改、PATCH 仅改 labels
  - Assignments：按节点获取（含 desired 与最新 Phase/Healthy/LastReportAt）、PATCH desired（Running/Stopped）、DELETE 单条
- Agent：
  - 周期拉取 assignments（只拉起 desired=Running 的实例）
  - 下载 zip（支持 http/https 与相对路径自动拼接）、解压、执行 startCmd 或默认 ./start.sh
  - 上报状态：Running/Stopped/Exited/Failed，Failed 包含退出码
  - 调谐：先回收→停止（TERM，超时 5s KILL）→启动；以会话/进程组方式启动并组信号终止，避免残留
- Web UI：
  - 路由化页面（Assignments/Nodes/Apps/Tasks）
  - Assignments：Desired/Phase/Healthy/LastReportAt，行级 Start/Stop
  - Tasks：列表（按钮间距修正）、详情（行级 Start/Stop/删除；“全部停止/按节点停止”批量操作）、创建（多条目/多节点副本，startCmd 可选）、配置（查看条目推导与编辑 labels）

### 5.2 待办（下一阶段）
- 健康探针与心跳健康：超时标记 Unknown/Unhealthy
- 任务“强制删除”选项（先下发 Stopped 再清理）
- 更细化的权限与用户体系（RBAC）
- 进程日志采集与查看（本地/集中式）
- 更完备的错误处理与前端提示（把 409 等转换成友好文案）
- 接入 `swaggo/swag` 注释生成 OpenAPI，并用 `http-swagger` 提供 Swagger UI
- 故障节点恢复后处理策略：
  - 保持现状（默认）：不回切，最稳。
  - 手动回切：提供“迁回到原节点/指定节点”的管理操作。
  - 自动回切/再均衡：增加稳定期与限速
    - 配置项示例：FAILBACK_ENABLED、FAILBACK_STABLE_SEC、FAILBACK_MAX_RATE
    - 策略示例：原节点健康稳定一段时间后，将部分实例按限速迁回（自动回切/再均衡控制器：恢复后按稳定期与限速迁回）；或引入“首选节点/亲和”标签进行温和再均衡。
- 提供单点解析接口：`/v1/resolve?service=...&strategy=random|round_robin&hashKey=...`（返回一个可用端点）

### 5.3 设计原则
- 声明式：控制面描述“期望状态”，Agent 对齐“实际状态”。
- 可进化：先内存/SQLite，后迁移 etcd；先进程，后容器；先单机，后 HA。
- 可移植：抽象 `IRuntime`/`IProcessSupervisor`/`ISystemMetrics`，避免平台绑定。
- 可观测：默认指标/日志/事件可查询；问题可追踪、可告警。
- 安全默认开启：mTLS、RBAC、审计日志。

### 5.4 里程碑
- M0（已完成）Walking Skeleton：
  - 控制面最小 HTTP：心跳、分配、任务创建、状态上报（内存态）。
  - Agent 心跳 + 拉取分配；UI 查询节点分配。
- M1 基础部署：
  - 多节点/副本的直接调度（按 `replicas`）
  - Agent 进程监督：启动/停止/重启（指数退避）
  - 健康检查（HTTP/TCP 探针）
  - 本地日志（按实例滚动）与基本状态机
- M2 服务注册与发现 + 基础监控/告警：
  - Register/Heartbeat/Discover 接口与租约（TTL）
  - Prometheus 指标导出，基础规则告警（CPU、崩溃、健康异常）
- M3 持久化与 HA：
  - 元数据从内存/SQLite 迁移至 etcd（或 Consul）
  - 控制面多副本，Leader 选举，租约与幂等控制
- M4 调度与编排：
  - 亲和/反亲和、节点标签与约束，滚动/金丝雀/蓝绿发布
  - 失败回滚，容量与配额（配合 M7）
- M5 运行时抽象与容器化（可选）：
  - `process` → `containerd/Docker` 插拔式运行时
  - 资源限制（CPU/内存）与隔离（cgroups/Job Objects）
- M6 网络与服务网格（可选）：
  - 端到端 mTLS，Sidecar/无 Sidecar 模式
  - 服务层路由、慢启动、故障注入（实验性）
- M7 多租户与安全：
  - 命名空间/项目，RBAC，限额与配额，审计日志
- M8 可观测性完善：
  - 指标/日志/追踪三栈（Prometheus+Loki/ELK+OTel）
  - SLO/错误预算与告警联动
- M9 扩展与生态：
  - gRPC/OpenAPI 开放接口，Webhook/Operator，SDK（Go/C++/TS）
  - 插件机制（调度器/运行时/探针/日志采集）
- M10 Windows 路线（Phase W）：
  - W0 编译通过 → W1 进程监督 → W2 Windows Service → W3 性能计数器 → W4 试点 → W5 容器
- M11 交付与升级：
  - Helm/Compose 清单、离线包、灰度升级、配置与密钥管理

### 5.5 子系统目标
- 控制面：
  - API Server、控制器（任务/实例/节点/服务）、调度器、事件总线（NATS 可选）
  - 幂等（operation_id）、版本化（resource_version）、一致性边界（强一致元数据、最终一致状态）
- 数据面（Agent）：
  - 归档下载与校验、解压缓存、进程监督、健康探针、回传状态
  - 断网可恢复、离线缓存、限速与重试
- 注册与发现：
  - 实例注册/心跳、租约与过期、客户端发现与缓存策略
- 日志与指标：
  - 实例本地日志滚动，集中式采集可选（Loki/ELK）；Prometheus 指标
- 告警：
  - 规则引擎（阈值、速率、组合），通知渠道（邮件/飞书/钉钉/Slack）
- 安全：
  - mTLS（CA/证书下发与轮换）、RBAC、审计日志、密钥与配置
- 存储：
  - etcd 作为源数据存储；SQLite 仅用于轻量/单机场景
- UI/UX：
  - 仪表盘、任务/实例/节点/服务的视图与动作、日志/事件/告警查询

### 5.6 非目标（短期暂缓）
- Serverless FaaS 平台化能力
- 跨云/多集群联邦与全局一致流量治理
- 完整 Service Mesh 全家桶（保留集成接口）

### 5.7 风险与约束
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
- Tasks：
  - POST `/v1/tasks`（支持 `entries[]`；`startCmd` 可选，缺省跑 `./start.sh`）
  - GET `/v1/tasks`、GET `/v1/tasks/{id}`（含 assignments）
  - PATCH `/v1/tasks/{id}`（更新 labels；名称不可改）
  - DELETE `/v1/tasks/{id}`（级联清理 assignments/statuses）
- Assignments：
  - GET `/v1/assignments?nodeId=`（返回 desired + 最新 Phase/Healthy/LastReportAt）
  - PATCH `/v1/assignments/{instanceId}` `{ desired: Running|Stopped }`
  - DELETE `/v1/assignments/{instanceId}`
