# Plum 项目完整总结

## 📋 项目概述

**Plum** 是一个现代化的分布式任务编排与调度系统，采用微服务架构设计，支持多种任务执行方式，提供完整的Web UI管理和监控功能。

### 核心定位
- 分布式任务调度和编排平台
- 支持多种执行器类型（embedded、service、os_process）
- 提供可视化工作流编排（DAG）
- 统一的资源和设备管理

---

## 🏗️ 系统架构

### 三大核心组件

#### 1. Controller (控制器)
- **语言**：Go
- **职责**：
  - 任务调度引擎
  - 工作流编排
  - 状态管理
  - RESTful API服务
- **数据库**：SQLite
- **端口**：8080 (HTTP API)
- **位置**：`controller/`

#### 2. Agent (节点代理)
- **语言**：C++
- **职责**：
  - 节点心跳和健康检查
  - 应用部署和生命周期管理
  - 服务发现和注册
  - 与Controller通信
- **位置**：`agent/`
- **特性**：
  - 启动应用时注入环境变量（PLUM_INSTANCE_ID, PLUM_APP_NAME, PLUM_APP_VERSION）
  - 支持SSE实时通信

#### 3. SDK (应用集成)
- **语言**：C++, Python
- **位置**：`sdk/cpp/`, `sdk/python/`
- **SDK类型**：
  - **plumworker**：嵌入式任务执行SDK（HTTP-based，旧版）
  - **plumresource**：资源管理SDK（设备集成）
  - **grpc worker**：新一代嵌入式工作器SDK（gRPC-based）

---

## 🎮 任务执行引擎

### 三种执行器类型

#### 1. Embedded 执行器
**特点**：任务代码嵌入在应用程序中执行

**两种实现方式**：
1. **HTTP Worker (旧版)**
   - Worker启动HTTP服务器监听端口
   - Controller通过HTTP POST调用Worker
   - 使用`plumworker` SDK
   
2. **gRPC Worker (新版，推荐)**
   - Worker启动gRPC服务器
   - Worker主动向Controller注册
   - 双向流通信，性能更好
   - 使用新的gRPC SDK

**目标类型**：
- `node`：在指定节点上执行
- `app`：在属于特定应用的Worker上执行

**Worker注册信息**：
- WorkerID, NodeID, InstanceID
- AppName, AppVersion
- 支持的任务列表（Tasks）
- 标签（Labels）：appName, deploymentId, version等

#### 2. Service 执行器
**特点**：通过HTTP调用远程服务端点

**配置参数**：
- `targetRef`：服务名称（必填）
- `serviceVersion`：服务版本（可选）
- `serviceProtocol`：http/https（可选）
- `servicePort`：端口号（可选）
- `servicePath`：API路径（可选）

**服务发现**：
- Controller从服务注册表中查找健康的服务实例
- 支持版本、协议、端口过滤
- 自动选择健康的端点

#### 3. OS Process 执行器
**特点**：在节点上执行操作系统命令

**配置参数**：
- `command`：要执行的命令（必填）
- `env`：环境变量（可选）
- `targetRef`：节点ID（可选，留空则在Controller本地执行）

---

## 🔄 工作流编排

### 工作流特性
- **顺序执行**：当前支持顺序执行步骤
- **DAG可视化**：使用Dagre布局算法，ECharts渲染
- **状态跟踪**：实时更新每个步骤的执行状态
- **参数传递**：支持步骤间的payload配置

### 工作流状态
- Pending（等待）
- Running（运行中）
- Succeeded（成功）
- Failed（失败）
- Canceled（已取消）

### DAG可视化颜色
- 🔵 蓝色：Pending
- 🟠 橙色：Running
- 🟢 绿色：Succeeded
- 🔴 红色：Failed
- ⚫ 灰色：Canceled

---

## 📊 核心数据模型

### 主要实体

#### Node（节点）
```go
type Node struct {
    NodeID   string
    IP       string
    Labels   map[string]string
    LastSeen time.Time
}
```

#### Deployment（部署）
```go
type Deployment struct {
    DeploymentID string
    Name         string
    Labels       map[string]string
    Replicas     map[string]int // nodeId -> replica count
}
```

#### Assignment（分配）
```go
type Assignment struct {
    InstanceID   string
    DeploymentID string
    NodeID       string
    Desired      DesiredState // Running/Stopped
    ArtifactURL  string
    StartCmd     string
    AppName      string
    AppVersion   string
}
```

#### TaskDefinition（任务定义）
```go
type TaskDef struct {
    DefID              string
    Name               string
    Executor           string // embedded/service/os_process
    TargetKind         string // node/app/service
    TargetRef          string
    Labels             map[string]string
    DefaultPayloadJSON string
    CreatedAt          int64
}
```

#### Task（任务运行）
```go
type Task struct {
    TaskID         string
    OriginTaskID   string // 关联到TaskDef
    Name           string
    Executor       string
    TargetKind     string
    TargetRef      string
    State          string
    PayloadJSON    string
    ResultJSON     string
    Error          string
    CreatedAt      int64
    StartedAt      int64
    FinishedAt     int64
    Attempt        int
}
```

#### Workflow（工作流）
```go
type Workflow struct {
    WorkflowID string
    Name       string
    Labels     map[string]string
    Steps      []WorkflowStep
}

type WorkflowStep struct {
    StepID       string
    Name         string
    Executor     string
    TargetKind   string
    TargetRef    string
    TimeoutSec   int
    MaxRetries   int
    Ord          int // 执行顺序
}
```

#### Worker（工作器）

**HTTP Worker (旧版)**：
```go
type Worker struct {
    WorkerID string
    NodeID   string
    URL      string
    Tasks    []string
    Labels   map[string]string
    Capacity int
    LastSeen int64
}
```

**Embedded Worker (新版gRPC)**：
```go
type EmbeddedWorker struct {
    WorkerID     string
    NodeID       string
    InstanceID   string
    AppName      string
    AppVersion   string
    GRPCAddress  string
    Tasks        []string
    Labels       map[string]string
    LastSeen     int64
}
```

#### Resource（资源）
```go
type Resource struct {
    ResourceID string
    NodeID     string
    Type       string
    URL        string
    StateDesc  []ResourceStateDesc
    OpDesc     []ResourceOpDesc
    LastSeen   int64
}

type ResourceStateDesc struct {
    Type  string // INT/DOUBLE/BOOL/ENUM/STRING
    Name  string
    Value string
    Unit  string
}

type ResourceOpDesc struct {
    Type  string
    Name  string
    Value string
    Unit  string
    Min   string
    Max   string
}
```

---

## 🛠️ 技术栈

### 后端
- **Go 1.19+**：Controller实现
- **C++17**：Agent和SDK实现
- **SQLite**：数据持久化
- **gRPC + Protocol Buffers**：高性能RPC通信
- **httplib**：C++ HTTP客户端/服务器
- **nlohmann/json**：JSON序列化

### 前端
- **Vue 3**：前端框架
- **TypeScript**：类型安全
- **Element Plus**：UI组件库
- **ECharts**：图表可视化
- **Dagre**：DAG布局算法
- **Vue I18n**：国际化（中英文）
- **Vite**：构建工具

### 构建系统
- **Makefile**：统一构建入口
- **CMake**：C++项目构建
- **npm**：前端依赖管理

---

## 🌐 API接口总览

### 节点管理
- `POST /v1/nodes/heartbeat` - 节点心跳
- `GET /v1/nodes` - 获取所有节点
- `GET /v1/nodes/{id}` - 获取特定节点
- `DELETE /v1/nodes/{id}` - 删除节点

### 服务发现
- `POST /v1/services/register` - 注册服务端点
- `POST /v1/services/heartbeat` - 服务心跳
- `GET /v1/services/list` - 获取服务列表
- `GET /v1/discovery?service={name}` - 服务发现
- `GET /v1/discovery/one?service={name}&strategy=lazy` - 获取单个服务端点（支持 random/lazy 策略）
- `DELETE /v1/services?instanceId={id}` - 删除服务

### 部署管理
- `GET /v1/deployments` - 获取所有部署
- `POST /v1/deployments` - 创建部署
- `GET /v1/deployments/{id}` - 获取部署详情
- `DELETE /v1/deployments/{id}` - 删除部署

### 分配管理
- `GET /v1/assignments?nodeId={id}` - 获取节点分配
- `GET /v1/assignments/{instanceId}` - 获取特定分配
- `PATCH /v1/assignments/{instanceId}` - 更新期望状态
- `DELETE /v1/assignments/{instanceId}` - 删除分配

### 任务定义
- `GET /v1/task-defs` - 获取所有任务定义
- `POST /v1/task-defs` - 创建任务定义
- `GET /v1/task-defs/{id}` - 获取任务定义详情
- `POST /v1/task-defs/{id}?action=run` - 运行任务定义
- `DELETE /v1/task-defs?id={id}` - 删除任务定义

### 任务运行
- `GET /v1/tasks` - 获取所有任务
- `GET /v1/tasks/{id}` - 获取任务详情
- `POST /v1/tasks/start/{id}` - 启动任务
- `POST /v1/tasks/cancel/{id}` - 取消任务
- `POST /v1/tasks/rerun/{id}` - 重新运行任务
- `DELETE /v1/tasks/{id}` - 删除任务
- `GET /v1/tasks/stream` - SSE任务状态流

### 工作流管理
- `GET /v1/workflows` - 获取所有工作流
- `POST /v1/workflows` - 创建工作流
- `GET /v1/workflows/{id}` - 获取工作流详情
- `POST /v1/workflows/{id}?action=run` - 运行工作流
- `DELETE /v1/workflows/{id}` - 删除工作流
- `GET /v1/workflows/{id}/runs` - 获取工作流运行记录

### 工作流运行
- `GET /v1/workflow-runs` - 获取所有运行
- `GET /v1/workflow-runs?workflowId={id}` - 获取特定工作流的运行
- `GET /v1/workflow-runs/{id}` - 获取运行详情
- `DELETE /v1/workflow-runs/{id}` - 删除运行记录

### 工作器管理
- `POST /v1/workers/register` - 注册HTTP Worker
- `POST /v1/workers/heartbeat` - Worker心跳
- `GET /v1/workers` - 获取所有HTTP Workers
- `POST /v1/embedded-workers/register` - 注册gRPC Worker
- `POST /v1/embedded-workers/heartbeat` - gRPC Worker心跳
- `GET /v1/embedded-workers` - 获取所有gRPC Workers
- `DELETE /v1/embedded-workers/{id}` - 删除gRPC Worker

### 资源管理
- `POST /v1/resources/register` - 注册资源
- `POST /v1/resources/heartbeat` - 资源心跳
- `GET /v1/resources` - 获取所有资源
- `GET /v1/resources/{id}` - 获取资源详情
- `POST /v1/resources/state` - 提交资源状态
- `GET /v1/resources/states?resourceId={id}` - 获取状态历史
- `POST /v1/resources/operation` - 执行资源操作
- `DELETE /v1/resources/{id}` - 删除资源

### 应用包管理
- `GET /v1/apps` - 获取所有应用包
- `POST /v1/apps/upload` - 上传应用包（ZIP）
- `DELETE /v1/apps/{id}` - 删除应用包

### 实时通信
- `GET /v1/stream?nodeId={id}` - SSE节点状态流
- `GET /v1/tasks/stream` - SSE任务状态流

---

## 💾 数据库设计

### SQLite表结构

#### 核心表
- `nodes` - 节点信息
- `deployments` - 部署定义
- `assignments` - 实例分配（包含app_name, app_version字段）
- `instance_status` - 实例状态
- `artifacts` - 应用包

#### 服务发现表
- `service_endpoints` - 服务端点注册

#### 任务相关表
- `task_defs` - 任务定义
- `tasks` - 任务运行记录
- `workflows` - 工作流定义
- `workflow_steps` - 工作流步骤
- `workflow_runs` - 工作流运行
- `workflow_step_runs` - 步骤运行记录

#### Worker表
- `workers` - HTTP Worker注册（旧版）
- `embedded_workers` - gRPC Worker注册（新版）

#### 资源表
- `resources` - 资源注册
- `resource_state_desc` - 资源状态描述
- `resource_op_desc` - 资源操作描述
- `resource_states` - 资源状态历史

---

## 🖥️ Web UI页面

### 页面列表

1. **Home (/)** - 首页概览
2. **Nodes (/nodes)** - 节点管理
3. **Apps (/apps)** - 应用包管理（上传、删除）
4. **Deployments (/deployments)** - 部署列表和管理
5. **Assignments (/assignments)** - 实例分配管理
6. **Services (/services)** - 服务发现和端点管理
7. **Tasks (/tasks)** - 任务定义和运行管理
8. **Workflows (/workflows)** - 工作流管理
9. **Resources (/resources)** - 资源管理（设备监控）
10. **Workers (/workers)** - 工作器管理（新增）

### UI组件库

#### 自定义组件
- `IdDisplay.vue` - ID缩短显示组件（支持悬停、复制）
- `WorkflowDAG.vue` - 工作流DAG可视化
- `AppsPanel.vue` - 应用包管理面板
- `DeploymentsPanel.vue` - 部署管理面板

#### 工具函数
- `ui/src/utils/formatters.ts` - 格式化工具（ID、时间、文件大小等）

### UI特性
- ✅ 响应式设计
- ✅ 中英文国际化
- ✅ 实时状态更新（SSE）
- ✅ 分页支持
- ✅ 搜索和过滤
- ✅ 智能表单（下拉框自动加载数据）
- ✅ ID缩短显示（悬停查看完整ID，点击复制）

---

## 🔧 开发和构建

### 常用命令

#### Controller
```bash
make controller          # 构建Controller
make controller-run      # 运行Controller
./bin/controller         # 直接运行
```

#### Agent
```bash
make agent              # 构建Agent
make agent-run          # 运行Agent
```

#### UI
```bash
cd ui && npm install    # 安装依赖
npm run dev             # 开发模式（端口5173或5174）
npm run build           # 生产构建
```

#### C++ SDK
```bash
make sdk_cpp                    # 构建所有C++ SDK
make sdk_cpp_echo_worker        # 构建echo worker示例
make sdk_cpp_radar_sensor       # 构建radar sensor示例
make sdk_cpp_grpc_echo_worker   # 构建gRPC worker示例
```

### 环境变量

#### Controller
- `CONTROLLER_DATA_DIR` - 数据目录（默认./data）
- `PORT` - HTTP端口（默认8080）

#### Agent
- `CONTROLLER_BASE` - Controller地址（如 http://127.0.0.1:8080）
- `NODE_ID` - 节点ID

#### Worker SDK
- `PLUM_INSTANCE_ID` - 实例ID（由Agent注入）
- `PLUM_APP_NAME` - 应用名称（由Agent注入）
- `PLUM_APP_VERSION` - 应用版本（由Agent注入）
- `WORKER_ID` - Worker ID
- `WORKER_NODE_ID` - 节点ID
- `CONTROLLER_BASE` - Controller地址
- `GRPC_ADDRESS` - gRPC监听地址（如 0.0.0.0:18082）

#### Resource SDK
- `RESOURCE_ID` - 资源ID
- `RESOURCE_NODE_ID` - 节点ID
- `CONTROLLER_BASE` - Controller地址

---

## 🎯 重要设计决策

### 1. Executor和TargetKind的关系

| Executor | 允许的TargetKind | TargetRef含义 |
|----------|-----------------|--------------|
| embedded | node, app | node: 节点ID; app: 应用名称 |
| service | service | 服务名称（必填） |
| os_process | node | 节点ID（可选） |

### 2. Worker标签设计

**标签用途**：用于Worker选择和路由

**常用标签**：
- `appName` - 应用名称（推荐使用）
- `serviceName` - 服务名称（旧版，向后兼容）
- `deploymentId` - 部署ID
- `version` - 版本号

**选择逻辑**：
- `targetKind=node` + `targetRef=nodeA`：选择nodeA上的Worker
- `targetKind=app` + `targetRef=myApp`：选择appName=myApp的Worker
- 留空：选择任意可用Worker

### 3. 状态管理

**任务状态流转**：
```
Pending → Running → Succeeded
                 ↘ Failed
                 ↘ Timeout
                 ↘ Canceled
```

**部署期望状态**：
- `Running` - 期望运行
- `Stopped` - 期望停止

**实例实际状态**：
- `phase` - 当前阶段（如 running, stopped）
- `healthy` - 健康状态（true/false）

### 4. ID生成规则

所有ID都是32字符的MD5哈希值：
- TaskID
- DefID
- WorkflowID
- RunID
- DeploymentID
- InstanceID

**UI优化**：使用IdDisplay组件显示前8个字符，悬停查看完整ID

---

## 📁 项目结构

```
Plum/
├── controller/          # Go控制器
│   ├── cmd/            # 主程序入口
│   └── internal/       # 内部实现
│       ├── store/      # 数据存储接口和SQLite实现
│       ├── httpapi/    # HTTP API处理器
│       ├── tasks/      # 任务调度器
│       ├── failover/   # 故障转移
│       └── grpc/       # gRPC客户端
├── agent/              # C++节点代理
├── sdk/
│   ├── cpp/           # C++ SDK
│   │   ├── plumworker/      # HTTP Worker SDK
│   │   ├── plumresource/    # Resource SDK
│   │   └── examples/        # 示例程序
│   └── python/        # Python SDK
├── ui/                # Vue前端
│   ├── src/
│   │   ├── views/     # 页面组件
│   │   ├── components/# 可复用组件
│   │   ├── utils/     # 工具函数
│   │   ├── router.ts  # 路由配置
│   │   └── i18n.ts    # 国际化配置
│   └── public/        # 静态资源
├── proto/             # gRPC协议定义
├── docs/              # 文档
├── Makefile           # 构建脚本
└── README.md          # 项目说明
```

---

## 🚀 快速启动

### 1. 启动Controller
```bash
cd /home/stone/code/Plum
make controller
./controller/bin/controller
```

### 2. 启动Agent
```bash
AGENT_NODE_ID=nodeA ./agent-go/plum-agent
```

### 3. 启动UI
```bash
cd ui
npm run dev
# 访问 http://localhost:5173 或 5174
```

### 4. 运行示例Worker
```bash
# gRPC Worker示例
PLUM_INSTANCE_ID=grpc-instance-001 \
PLUM_APP_NAME=grpc-echo-app \
PLUM_APP_VERSION=v2.0.0 \
WORKER_ID=grpc-echo-1 \
WORKER_NODE_ID=nodeA \
CONTROLLER_BASE=http://127.0.0.1:8080 \
GRPC_ADDRESS=0.0.0.0:18082 \
./sdk/cpp/build/examples/grpc_echo_worker/grpc_echo_worker
```

---

## 🐛 常见问题和解决方案

### 1. WSL2网络问题
- **问题**：localhost和127.0.0.1行为不一致
- **解决**：统一使用127.0.0.1

### 2. UI显示空数据
- **原因**：API返回null时未处理
- **解决**：确保所有数组都初始化为`[]`，使用`Array.isArray(data) ? data : []`

### 3. 端口冲突
- **问题**：多个进程监听同一端口
- **解决**：检查进程并kill，或使用不同端口

### 4. ID显示问题
- **问题**：32字符ID占用过多空间
- **解决**：使用IdDisplay组件显示缩短版

### 5. 列宽过小导致小黑点
- **问题**：el-tag内容溢出显示为小黑点
- **解决**：适当增加列宽（如120px、130px）

### 6. 字段名不匹配
- **问题**：API返回PascalCase，前端使用camelCase
- **解决**：同时检查两种命名（`row.field || row.Field`）

---

## 📝 代码规范

### Git提交规范
遵循Conventional Commits：
- `feat:` - 新功能
- `fix:` - 修复bug
- `docs:` - 文档更新
- `refactor:` - 代码重构
- `style:` - 代码格式
- `test:` - 测试相关
- `chore:` - 构建/工具相关

### 代码风格
- **Go**：遵循Go官方规范
- **TypeScript**：使用ESLint和Prettier
- **C++**：遵循Google C++风格指南

---

## 🔄 最近完成的功能

### 1. Workers管理页面（新增）
- 展示嵌入式工作器和HTTP工作器
- 显示应用信息、支持的任务、健康状态
- 支持搜索、过滤、详情查看、删除

### 2. 资源管理功能
- 外部设备资源注册和管理
- 实时状态监控和历史记录
- 操作命令下发
- C++ Resource SDK实现

### 3. gRPC Worker架构（新版）
- Worker主动注册到Controller
- 使用gRPC双向流通信
- 自动从环境变量获取实例信息
- 不再需要Worker启动HTTP服务器

### 4. UI优化
- ID显示优化（IdDisplay组件）
- 智能表单下拉框（节点、应用、服务）
- 状态统一（删除Completed，统一使用Succeeded）
- 列宽优化和小黑点修复

### 5. Agent环境变量注入
- 启动应用时自动注入PLUM_*环境变量
- Worker SDK可自动获取实例信息

---

## 🚧 待实现功能

### 高优先级
- [ ] DAG并行执行（当前只支持顺序执行）
- [ ] 任务优先级和队列管理
- [ ] 更多内置任务类型
- [ ] 工作流条件分支

### 中优先级
- [ ] 用户认证和权限管理
- [ ] 性能指标收集（Prometheus）
- [ ] 告警和通知系统
- [ ] CLI命令行工具

### 低优先级
- [ ] 多Controller集群
- [ ] 分布式锁和协调
- [ ] 插件系统

---

## 📚 重要文件说明

### 配置文件
- `Makefile` - 统一构建脚本
- `ui/vite.config.ts` - Vite配置
- `sdk/cpp/CMakeLists.txt` - C++ SDK构建配置

### 核心实现
- `controller/internal/tasks/scheduler.go` - 任务调度核心逻辑
- `controller/internal/store/sqlite/sqlite.go` - 数据库实现
- `controller/internal/httpapi/routes.go` - API路由注册
- `ui/src/router.ts` - 前端路由
- `ui/src/i18n.ts` - 国际化配置

### SDK实现
- `sdk/cpp/plumworker/` - HTTP Worker SDK
- `sdk/cpp/plumresource/` - Resource SDK
- `sdk/cpp/examples/grpc_echo_worker/` - gRPC Worker示例

---

## 🎨 UI设计规范

### 布局风格
参考Tasks.vue的标准布局：
```vue
<!-- 操作按钮和统计信息 -->
<div style="display:flex; justify-content:space-between;">
  <!-- 左侧：操作按钮 -->
  <div style="display:flex; gap:8px;">
    <el-button>刷新</el-button>
    <el-button>创建</el-button>
  </div>
  
  <!-- 中间：统计信息 -->
  <div style="display:flex; gap:20px; justify-content:center;">
    <!-- 20px图标 + 数字 + 标签 -->
  </div>
  
  <!-- 右侧：占位 -->
  <div style="width:120px;"></div>
</div>

<!-- 主内容卡片 -->
<el-card>
  <template #header>
    <span>标题</span>
    <span>{{ count }} 项</span>
  </template>
  <el-table>...</el-table>
  <el-pagination>...</el-pagination>
</el-card>
```

### 统计图标规范
- 尺寸：20px × 20px
- 图标：12px
- 圆角：4px
- 渐变背景

### 列宽建议
- ID列：100-120px（使用IdDisplay）
- 状态列（带图标）：120-130px
- 时间列：160-220px
- 名称列：160-200px
- 操作列：180-280px

---

## 🔐 安全和权限

### 当前状态
- ❌ 无认证机制
- ❌ 无权限控制
- ✅ CORS支持

### 未来计划
- [ ] JWT认证
- [ ] RBAC权限模型
- [ ] API密钥管理

---

## 📊 性能和扩展性

### 当前性能
- SQLite单机数据库
- 单Controller实例
- 支持多节点Agent
- 支持多Worker并发

### 扩展性考虑
- Controller可水平扩展（需要分布式锁）
- Agent轻量级，支持大量节点
- Worker按需扩展

---

## 🧪 测试和调试

### 日志位置
- Controller：标准输出
- Agent：标准输出
- Worker：标准输出

### 调试技巧
1. 使用`curl`测试API
2. 检查浏览器控制台（前端错误）
3. 查看Controller日志（后端错误）
4. 使用`jq`格式化JSON输出

### 常用调试命令
```bash
# 查看任务状态
curl -s http://127.0.0.1:8080/v1/tasks/{id} | jq .

# 查看Worker列表
curl -s http://127.0.0.1:8080/v1/embedded-workers | jq .

# 查看节点列表
curl -s http://127.0.0.1:8080/v1/nodes | jq .

# 查看进程
ps aux | grep controller
ps aux | grep agent
```

---

## 💡 开发建议

### 添加新页面
1. 在`ui/src/views/`创建Vue组件
2. 在`ui/src/router.ts`添加路由
3. 在`ui/src/App.vue`添加导航菜单
4. 在`ui/src/i18n.ts`添加国际化文本

### 添加新API
1. 在`controller/internal/store/store.go`定义接口
2. 在`controller/internal/store/sqlite/`实现
3. 在`controller/internal/httpapi/`添加处理器
4. 在`controller/internal/httpapi/routes.go`注册路由

### 添加新执行器
1. 在`controller/internal/tasks/scheduler.go`添加执行逻辑
2. 更新UI的executor下拉框选项
3. 添加相应的表单字段和验证

---

## 🎓 学习资源

### 项目文档
- `README.md` - 项目概述和快速开始
- `docs/PROJECT_SUMMARY.md` - 本文档
- `ui/src/components/README.md` - UI组件使用说明

### 代码示例
- `sdk/cpp/examples/echo_worker/` - HTTP Worker示例
- `sdk/cpp/examples/grpc_echo_worker/` - gRPC Worker示例
- `sdk/cpp/examples/radar_sensor/` - Resource SDK示例

---

## 📞 重要提示

### 开发环境
- OS: WSL2 (Linux 5.15)
- Shell: /bin/bash
- 工作目录: /home/stone/code/Plum

### 后台进程
可能正在运行的后台进程：
- Controller (端口8080)
- UI开发服务器 (端口5173/5174)
- gRPC Worker示例 (端口18082)

### 清理命令
```bash
# 停止所有相关进程
pkill -f controller
pkill -f "npm run dev"
pkill -f grpc_echo_worker
```

---

## 🎯 下一步工作建议

1. **完成ID优化**：继续优化剩余页面的ID显示
2. **Worker管理完善**：添加Worker详细信息和操作历史
3. **资源管理增强**：添加更多设备类型支持
4. **性能优化**：添加缓存和索引
5. **文档完善**：添加API文档和开发指南

---

**最后更新时间**：2025-10-09
**项目状态**：活跃开发中
**当前版本**：开发版

