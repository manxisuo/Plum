# Plum - 分布式任务编排与调度系统

Plum 是一个现代化的分布式任务编排与调度系统，采用微服务架构设计，支持多种任务执行方式，提供完整的Web UI管理和监控功能。

## 🎯 项目概述

Plum 旨在解决分布式环境下的任务编排、调度和执行问题，支持从简单的脚本执行到复杂的业务流程编排。系统采用现代化的架构设计，提供高性能、高可靠性的任务调度服务。

### 核心特性

- 🚀 **多种执行器**：支持embedded、service、os_process三种任务执行方式
- 🔄 **工作流编排**：可视化DAG流程图，支持复杂业务流程编排
- 📊 **实时监控**：Web UI实时状态监控和历史记录管理
- 🌐 **分布式架构**：支持多节点部署和自动故障转移
- 🔧 **易于集成**：提供C++ SDK，支持多种编程语言
- 🌍 **国际化**：支持中英文界面切换
- 🎯 **智能管理**：工作器管理、资源监控、智能表单选择
- 🔌 **设备集成**：支持外部设备状态监控和操作控制

## 🏗️ 系统架构

### 核心组件

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Controller    │    │     Agent       │    │   Worker SDK    │
│   (调度中心)     │◄──►│   (节点代理)     │◄──►│   (应用集成)     │
│                 │    │                 │    │                 │
│ • 任务调度       │    │ • 节点管理       │    │ • 任务执行       │
│ • 工作流编排     │    │ • 服务发现       │    │ • 结果回传       │
│ • 状态管理       │    │ • 健康检查       │    │ • 心跳维护       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

**Controller（控制器）**
- 核心调度引擎，负责任务调度、工作流编排和状态管理
- 基于Go语言开发，使用SQLite作为数据存储
- 提供RESTful API接口和SSE实时通信

**Agent（代理）**
- 部署在各个节点上的轻量级代理
- 基于Go语言开发，与Controller技术栈统一
- 负责与Controller通信，报告节点状态
- 支持服务发现和健康检查

**Worker SDK**
- 提供C++ SDK，供应用程序集成
- 支持任务注册、执行和结果回传
- 实现嵌入式任务执行模式
- 支持HTTP和gRPC两种通信方式

## 🎮 任务执行引擎

### 三种执行器类型

#### 1. Embedded 执行器
- **内置任务**：builtin.echo、builtin.delay、builtin.sleep、builtin.fail
- **Worker回调**：应用程序集成SDK执行
- **特点**：低延迟、高可靠性、支持实时任务注册

#### 2. Service 执行器
- **HTTP/gRPC调用**：调用远程服务端点
- **服务发现**：自动发现和选择健康的服务实例
- **配置灵活**：支持版本、协议、端口、路径配置
- **负载均衡**：智能端点选择和健康检查

#### 3. OS Process 执行器
- **系统命令**：执行任意操作系统命令
- **进程管理**：完整的生命周期管理和资源清理
- **超时控制**：防止长时间运行的任务阻塞系统
- **输入输出**：支持标准输入输出和环境变量设置

### 执行器对比

| 执行器类型 | 适用场景 | 优势 | 劣势 |
|-----------|---------|------|------|
| embedded | 实时任务、低延迟要求 | 高性能、低延迟 | 需要集成SDK |
| service | 微服务调用、API集成 | 解耦、易扩展 | 网络依赖 |
| os_process | 脚本执行、系统操作 | 灵活、通用 | 资源消耗较大 |

## 🔄 工作流编排

### 工作流特性

- **可视化DAG**：直观的流程图展示，实时状态更新
- **顺序执行**：当前支持顺序执行，DAG并行执行已预留接口
- **状态管理**：完整的执行状态跟踪和历史记录
- **错误处理**：自动重试和故障转移机制

### 工作流管理

- **模板化设计**：基于TaskDefinition的模板化工作流定义
- **参数配置**：支持默认payload和运行时参数覆盖
- **历史管理**：完整的运行历史查看和清理功能
- **级联删除**：工作流删除时自动清理相关数据

### DAG可视化

```
[builtin.echo] → [my.service.task] → [os_process.script]
     ↓               ↓                   ↓
   成功🟢           运行中🟠            等待中🔵
```

- **状态颜色**：
  - 🔵 蓝色：Pending（等待执行）
  - 🟠 橙色：Running（正在执行）
  - 🟢 绿色：Succeeded（成功完成）
  - 🔴 红色：Failed（执行失败）
  - ⚫ 灰色：Canceled（已取消）

## 📊 数据模型

### 核心实体

**Deployment（部署）**
- 长期运行的服务部署
- 支持实例数量管理和扩缩容
- 与Node关联，支持分布式部署

**TaskDefinition（任务定义）**
- 任务执行模板
- 支持默认payload配置
- 可被多个工作流引用

**TaskRun（任务运行）**
- 任务执行实例
- 记录执行状态、结果和时间信息
- 支持重试和超时控制

**Workflow（工作流）**
- 业务流程定义
- 包含多个步骤和执行顺序
- 支持参数传递和条件执行

**WorkflowRun（工作流运行）**
- 工作流执行实例
- 跟踪整体执行状态
- 包含所有步骤的执行记录

**Node（节点）**
- 计算节点信息
- 资源状态监控
- 健康检查和故障检测

**Service（服务）**
- 注册的服务信息
- 端点发现和管理
- 健康状态监控

**Worker（工作节点）**
- 任务执行能力注册
- 容量管理和负载均衡
- 心跳维护和状态同步

**Resource（资源）**
- 外部设备资源管理
- 状态监控和操作控制
- 支持传感器、执行器等设备类型

## 🛠️ 技术栈

### 后端技术
- **Go语言**：Controller和Agent统一使用Go开发
- **SQLite**：轻量级、零配置的嵌入式数据库
- **RESTful API**：标准化的HTTP接口设计
- **SSE (Server-Sent Events)**：实时通信和状态更新
- **gRPC**：高性能RPC框架（新版Worker通信）
- **微服务架构**：模块化、可扩展的系统设计

### 前端技术
- **Vue 3**：现代化的前端框架
- **TypeScript**：类型安全的JavaScript超集
- **Element Plus**：企业级UI组件库
- **ECharts + Dagre**：专业图表和布局算法
- **国际化**：多语言支持（中英文）

### 构建系统
- **Makefile**：统一构建脚本
- **CMake**：C++组件构建支持
- **Git**：版本控制和协作

## 🚀 快速开始

### 环境要求

- Go 1.19+
- Node.js 16+
- CMake 3.10+（用于C++ SDK）
- Git

### 安装和运行

1. **克隆项目**
```bash
git clone <repository-url>
cd Plum
```

2. **构建Controller**
```bash
make controller
```

3. **构建前端**
```bash
make ui-build
```

4. **运行Controller**
```bash
CONTROLLER_DATA_DIR=/path/to/data ./controller/bin/controller
```

5. **访问Web UI**
```
http://localhost:5173 (或 5174，取决于端口占用情况)
```

### 开发环境

1. **启动Controller**
```bash
make controller-run
```

2. **启动前端开发服务器**
```bash
cd ui && npm run dev
```

3. **构建C++ SDK**
```bash
make sdk_cpp
```

## 📖 使用指南

### 创建任务定义

```bash
# 创建embedded任务
curl -X POST "http://127.0.0.1:8080/v1/task-defs" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-task",
    "executor": "embedded",
    "defaultPayload": {"message": "hello"}
  }'

# 创建service任务
curl -X POST "http://127.0.0.1:8080/v1/task-defs" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-call",
    "executor": "service",
    "targetKind": "service",
    "targetRef": "my-service",
    "labels": {
      "serviceVersion": "v1",
      "serviceProtocol": "http",
      "servicePort": "8080",
      "servicePath": "/api/execute"
    }
  }'

# 创建os_process任务
curl -X POST "http://127.0.0.1:8080/v1/task-defs" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "system-script",
    "executor": "os_process",
    "defaultPayload": {
      "command": "python",
      "args": ["script.py"],
      "workingDir": "/path/to/scripts"
    }
  }'
```

### 创建工作流

```bash
curl -X POST "http://127.0.0.1:8080/v1/workflows" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-workflow",
    "steps": [
      {
        "name": "step1",
        "executor": "embedded",
        "definitionId": "task-def-id-1",
        "timeoutSec": 30,
        "maxRetries": 3,
        "ord": 0
      },
      {
        "name": "step2",
        "executor": "service",
        "definitionId": "task-def-id-2",
        "timeoutSec": 60,
        "maxRetries": 1,
        "ord": 1
      }
    ]
  }'
```

### 运行任务和工作流

```bash
# 运行任务定义
curl -X POST "http://127.0.0.1:8080/v1/task-defs/{id}?action=run"

# 运行工作流
curl -X POST "http://127.0.0.1:8080/v1/workflows/{id}?action=run"
```

## 🔧 API 速查

### 任务定义管理
- `GET /v1/task-defs` - 获取所有任务定义
- `POST /v1/task-defs` - 创建任务定义
- `GET /v1/task-defs/{id}` - 获取特定任务定义
- `POST /v1/task-defs/{id}?action=run` - 运行任务定义

### 任务运行管理
- `GET /v1/tasks` - 获取所有任务运行
- `GET /v1/tasks/{id}` - 获取特定任务运行
- `POST /v1/tasks/{id}?action=start` - 启动任务
- `POST /v1/tasks/{id}?action=cancel` - 取消任务
- `DELETE /v1/tasks/{id}` - 删除任务

### 工作流管理
- `GET /v1/workflows` - 获取所有工作流
- `POST /v1/workflows` - 创建工作流
- `GET /v1/workflows/{id}` - 获取特定工作流
- `POST /v1/workflows/{id}?action=run` - 运行工作流
- `DELETE /v1/workflows/{id}` - 删除工作流

### 工作流运行管理
- `GET /v1/workflow-runs` - 获取所有工作流运行
- `GET /v1/workflow-runs/{id}` - 获取特定工作流运行
- `GET /v1/workflow-runs?workflowId={id}` - 获取特定工作流的所有运行
- `DELETE /v1/workflow-runs/{id}` - 删除工作流运行

### 节点和服务管理
- `GET /v1/nodes` - 获取所有节点
- `GET /v1/services` - 获取所有服务
- `GET /v1/discovery` - 获取服务端点发现信息
- `GET /v1/workers` - 获取所有工作节点

### 工作器管理
- `GET /v1/embedded-workers` - 获取所有嵌入式工作器
- `POST /v1/embedded-workers/register` - 注册嵌入式工作器
- `POST /v1/embedded-workers/heartbeat` - 工作器心跳
- `DELETE /v1/embedded-workers/{id}` - 删除嵌入式工作器

### 资源管理
- `GET /v1/resources` - 获取所有资源
- `POST /v1/resources/register` - 注册资源
- `POST /v1/resources/state` - 提交资源状态
- `POST /v1/resources/operation` - 执行资源操作
- `GET /v1/resources/states` - 获取资源状态历史

## 📋 已完成功能

### ✅ 核心功能
- [x] 分布式任务调度和执行
- [x] 三种任务执行器（embedded、service、os_process）
- [x] 工作流编排和顺序执行
- [x] DAG可视化流程图
- [x] 实时状态监控和更新
- [x] Web UI管理和监控界面
- [x] 工作器管理（嵌入式工作器和HTTP工作器）
- [x] 资源管理（设备状态监控和操作控制）
- [x] 智能表单选择（节点、应用、服务下拉框）

### ✅ 任务管理
- [x] TaskDefinition模板化设计
- [x] 任务运行实例管理
- [x] 任务状态跟踪（Pending、Running、Succeeded、Failed、Timeout、Canceled）
- [x] 任务重试和超时控制
- [x] 任务结果记录和错误处理
- [x] 内置任务支持（echo、delay、sleep、fail）

### ✅ 工作流功能
- [x] 工作流定义和管理
- [x] 工作流运行实例跟踪
- [x] 工作流历史记录管理
- [x] 工作流删除和清理
- [x] DAG可视化显示
- [x] 实时状态更新和自动刷新
- [x] 工作流步骤参数配置

### ✅ 执行器实现
- [x] **embedded执行器**
  - [x] 内置任务执行
  - [x] Worker回调执行
  - [x] 实时任务注册
- [x] **service执行器**
  - [x] HTTP/gRPC服务调用
  - [x] 服务发现和端点选择
  - [x] 服务标签配置（版本、协议、端口、路径）
  - [x] 健康检查和负载均衡
- [x] **os_process执行器**
  - [x] 外部进程执行
  - [x] 进程生命周期管理
  - [x] 超时控制和资源清理
  - [x] 输入输出捕获和环境变量设置

### ✅ 数据管理
- [x] SQLite数据库存储
- [x] 在线数据库迁移
- [x] 数据一致性保证
- [x] 级联删除支持
- [x] 完整的CRUD操作

### ✅ Web UI功能
- [x] 响应式设计
- [x] 国际化支持（中英文）
- [x] 实时状态更新
- [x] DAG可视化图表
- [x] 任务和工作流管理界面
- [x] 运行历史查看
- [x] 错误处理和用户反馈
- [x] 智能表单下拉框（节点、应用、服务选择）
- [x] 工作器管理界面（嵌入式工作器和HTTP工作器）
- [x] 资源管理功能（设备状态监控和操作控制）

### ✅ 工作器管理
- [x] 嵌入式工作器注册和管理
- [x] HTTP工作器注册和管理
- [x] 工作器健康状态监控
- [x] 工作器能力展示（支持的任务列表）
- [x] 工作器详情查看和删除

### ✅ 资源管理
- [x] 外部设备资源注册
- [x] 资源状态实时监控
- [x] 资源操作命令下发
- [x] 资源状态历史记录
- [x] 资源类型管理（传感器、执行器等）
- [x] C++资源管理SDK

### ✅ 开发和运维
- [x] 统一构建系统（Makefile）
- [x] C++ SDK构建支持
- [x] 开发环境配置
- [x] API文档和示例
- [x] 错误日志和调试信息

## 🚧 待办功能

### 🔄 工作流增强
- [ ] DAG并行执行支持
- [ ] 条件分支和循环控制
- [ ] 工作流模板和参数化
- [ ] 工作流版本管理
- [ ] 工作流调度（定时执行）
- [ ] 工作流依赖管理

### 🎯 执行器优化
- [ ] **embedded执行器增强**
  - [ ] 更多内置任务类型
  - [ ] 动态任务注册
  - [ ] 任务优先级管理
- [ ] **service执行器增强**
  - [ ] 更多负载均衡策略（随机、轮询、容量感知）
  - [ ] 服务熔断和降级
  - [ ] 请求重试和超时配置
  - [ ] 服务健康检查增强
- [ ] **os_process执行器增强**
  - [ ] 进程组管理
  - [ ] 资源限制（CPU、内存）
  - [ ] 进程监控和统计
  - [ ] 批量命令执行

### 📊 监控和运维
- [ ] 性能指标收集（Prometheus集成）
- [ ] 分布式链路追踪
- [ ] 告警和通知系统
- [ ] 系统健康检查
- [ ] 自动故障恢复
- [ ] 容量规划和资源优化

### 🔐 安全和权限
- [ ] 用户认证和授权
- [ ] 角色权限管理
- [ ] API访问控制
- [ ] 敏感信息加密
- [ ] 审计日志记录

### 🌐 分布式功能
- [ ] 多Controller集群
- [ ] 数据同步和一致性
- [ ] 跨数据中心部署
- [ ] 网络分区容错
- [ ] 分布式锁和协调

### 📈 扩展性增强
- [ ] 插件系统支持
- [ ] 自定义执行器开发
- [ ] 第三方系统集成
- [ ] 消息队列支持
- [ ] 事件驱动架构

### 🛠️ 开发工具
- [ ] CLI命令行工具
- [ ] 配置管理工具
- [ ] 调试和诊断工具
- [ ] 性能分析工具
- [ ] 自动化测试套件

## 🤝 贡献指南

我们欢迎社区贡献！请遵循以下步骤：

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

### 开发规范

- 使用 Conventional Commits 规范提交信息
- 代码需要经过测试验证
- 新功能需要添加相应的文档
- 遵循项目的代码风格和架构设计

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 📞 联系我们

- 项目主页：[GitHub Repository]
- 问题反馈：[GitHub Issues]
- 文档：[Project Documentation]

---

**Plum** - 让任务编排更简单，让分布式调度更可靠！