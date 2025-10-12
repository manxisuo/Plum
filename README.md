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
- 💾 **分布式KV存储**：集群级别的键值对存储，持久化可靠，支持状态共享和崩溃恢复

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

**核心组件**：
- Go 1.19+ （Controller和Agent）
- Node.js 16+ （Web UI）
- Git

**可选组件**：
- protobuf-compiler 3.12+ （修改proto文件时需要）
- CMake 3.10+ + pkg-config （构建C++ SDK）
- grpc++ + protobuf-dev （C++ gRPC Worker SDK）

### 安装依赖

#### Ubuntu/Debian

```bash
# 核心依赖
sudo apt update
sudo apt install -y git curl

# Go 1.19+（如果未安装）
wget https://golang.google.cn/dl/go1.22.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.6.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

# 配置Go代理（中国网络推荐）
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct

# Node.js 16+（使用nvm推荐）
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
source ~/.bashrc
nvm install 18
nvm use 18

# Node.js 16+（直接安装）
wget https://nodejs.org/dist/v18.20.4/node-v18.20.4-linux-x64.tar.xz
tar -xf node-v18.20.4-linux-x64.tar.xz
sudo mv node-v18.20.4-linux-x64 /usr/local/nodejs18
sudo ln -sf /usr/local/nodejs18/bin/node /usr/local/bin/node
sudo ln -sf /usr/local/nodejs18/bin/npm /usr/local/bin/npm
sudo ln -sf /usr/local/nodejs18/bin/npx /usr/local/bin/npx

# protobuf编译器（可选，修改proto时需要）
sudo apt install -y protobuf-compiler libgrpc++-dev protobuf-compiler-grpc

# Go protobuf插件（可选，修改proto时需要）
# 注意：需要先配置好GOPROXY（见上面Go安装部分）
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# CMake和C++开发环境（可选，构建C++ SDK时需要）
sudo apt install -y cmake build-essential pkg-config

# C++ gRPC依赖（可选，构建gRPC Worker时需要）
sudo apt install -y libgrpc++-dev libprotobuf-dev
```

#### 验证安装

```bash
go version                # go version go1.22.6 linux/amd64
node --version            # v18.x.x
npm --version             # 9.x.x
git --version             # git version 2.x.x

# 可选工具验证（构建C++ SDK时需要）
protoc --version          # libprotoc 3.12.4
protoc-gen-go --version   # protoc-gen-go v1.36.10
protoc-gen-go-grpc --version  # protoc-gen-go-grpc 1.5.1
cmake --version           # cmake version 3.x.x
pkg-config --version      # pkg-config 0.29.x
```

### 构建和运行

#### 方式1：生产模式（使用nginx）

```bash
# 1. 克隆项目
git clone https://github.com/manxisuo/plum.git
cd Plum

# 2. 生成proto代码（首次需要）
make proto

# 3. 构建所有组件
make controller       # 构建Controller
make agent           # 构建Agent
make ui               # 安装UI依赖
make ui-build         # 构建UI静态文件到ui/dist/

# 4. 配置nginx（示例）
# server {
#   listen 80;
#   location / {
#     root /path/to/Plum/ui/dist;
#     try_files $uri $uri/ /index.html;
#   }
#   location /v1/ {
#     proxy_pass http://localhost:8080;
#   }
# }

# 5. 启动服务
./controller/bin/controller &               # Controller (API服务)
make agent-run &                            # Agent
sudo systemctl restart nginx                # Nginx (UI服务)

# 6. 访问
# Web UI: http://your-server/
# API: http://your-server/v1/
```

#### 方式2：开发模式（推荐，无需nginx）

```bash
# 1. 克隆和初始化
git clone https://github.com/manxisuo/plum.git
cd Plum
make proto
make ui

# 2. 构建并启动Controller（终端1）
make controller && make controller-run
# API服务: http://localhost:8080

# 3. 构建并启动Agent（终端2）
make agent && make agent-run

# 4. 启动Web UI开发服务器（终端3）
make ui-dev
# UI服务: http://localhost:5173

# 5. 访问Web UI
# 浏览器打开 http://localhost:5173

# 提示：
# - 前端代码修改会自动热更新
# - Go代码修改后需重新构建：make controller 或 make agent
# - 然后重启对应进程
```

### 配置管理

Plum支持两种配置方式（优先级：**环境变量 > .env文件 > 默认值**）

#### 方式1：.env文件（推荐）

```bash
# Controller
cd controller
cp env.example .env
vim .env  # 修改配置

# Agent
cd agent-go
cp env.example .env
vim .env

# SDK应用（如kv-demo）
cd examples/kv-demo
cp ../../sdk/cpp/plumkv/env.example .env
vim .env
```

#### 方式2：环境变量

```bash
# Controller
CONTROLLER_ADDR=:9090 ./controller/bin/controller

# Agent
CONTROLLER_BASE=http://127.0.0.1:9090 make agent-run
```

#### 主要配置项

**Controller:**
- `CONTROLLER_ADDR` - 监听地址（默认`:8080`）
- `CONTROLLER_DB` - 数据库路径
- `CONTROLLER_DATA_DIR` - 数据目录
- `HEARTBEAT_TTL_SEC` - 心跳超时（默认30秒）
- `FAILOVER_ENABLED` - 故障转移开关（默认true）

**Agent:**
- `AGENT_NODE_ID` - 节点ID（默认`nodeA`）
- `CONTROLLER_BASE` - Controller地址
- `AGENT_DATA_DIR` - 数据目录

**SDK/应用:**
- `PLUM_INSTANCE_ID` - 实例ID（Agent注入）
- `PLUM_APP_NAME` - 应用名称（Agent注入）
- `PLUM_KV_SYNC_MODE` - KV同步模式：`polling`/`sse`/`disabled`
- `CONTROLLER_BASE` - Controller地址

**配置模板位置：**
- `controller/env.example` - Controller配置
- `agent-go/env.example` - Agent配置
- `sdk/cpp/plumkv/env.example` - KV SDK配置
- `sdk/cpp/plumresource/env.example` - Resource SDK配置
- `sdk/cpp/env_loader.hpp` - C++ SDK公共.env工具

### 常用命令速查

```bash
# 构建
make controller              # 构建Controller
make agent                   # 构建Agent
make proto                   # 生成proto代码
make sdk_cpp                 # 构建C++ SDK
make sdk_cpp_mirror          # 构建C++ SDK（使用GitHub镜像）
make agent-clean             # 清理Agent构建产物
make proto-clean             # 清理proto生成代码

# 运行
make controller-run          # 运行Controller
make agent-run               # 运行Agent (nodeA)
make agent-runB              # 运行Agent (nodeB)
make agent-run-multi         # 后台运行3个Agent (A/B/C)
make agent-help              # 查看Agent命令帮助

# Web UI
make ui                      # 安装依赖
make ui-dev                  # 开发模式
make ui-build                # 生产构建
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

### ✅ 分布式KV存储
- [x] 持久化存储（Controller SQLite）
- [x] 命名空间隔离
- [x] 类型安全接口（string/int/double/bool/bytes）
- [x] 二进制数据支持（Base64编码）
- [x] 本地缓存 + 快速访问
- [x] 双模式同步（polling/sse/disabled）
- [x] SSE实时推送（可选）
- [x] 批量操作支持
- [x] 崩溃恢复能力
- [x] 跨节点状态共享
- [x] C++ SDK封装
- [x] Web UI查看管理（查看/编辑/删除）

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
- [x] Go Agent（技术栈统一，维护成本降低）
- [x] Proto编译自动化（make proto一键生成）
- [x] 部署状态控制（Stopped/Running）
- [x] 进程监控和自动重启
- [x] C++ SDK构建支持
- [x] 开发环境配置
- [x] API文档和示例
- [x] 错误日志和调试信息

## 📦 示例应用

Plum提供了三个完整的示例应用，展示不同的使用场景：

### 1. demo-app - 基础HTTP服务
- 简单的HTTP服务器应用
- 演示基本的应用打包和部署流程
- 包含服务注册和健康检查
- 位置：`examples/demo-app/`

### 2. worker-demo - Embedded Worker集成
- 集成gRPC Worker SDK的应用
- 演示embedded执行器的使用
- 实现TaskService接口接受任务调度
- 支持自动注册和心跳
- 位置：`examples/worker-demo/`

### 3. kv-demo - 分布式KV存储崩溃恢复
- 演示分布式KV存储的崩溃恢复能力
- 定期保存任务进度到持久化KV存储
- 崩溃后自动恢复到上次检查点
- 支持跨节点迁移（主备切换）
- 展示状态共享和持久化特性
- 位置：`examples/kv-demo/`

**使用方式：**
```bash
# 构建示例
cd examples/demo-app  # 或 worker-demo / kv-demo
./build.sh

# 上传zip包到Plum并创建部署
# 详见各示例的README.md
```

## ❓ 常见问题

### 构建问题

**Q: make proto报错"protoc: command not found"**
```bash
sudo apt install protobuf-compiler
```

**Q: make agent报错"go: command not found"**
```bash
# 检查Go是否安装
which go

# 如果已安装但找不到，添加到PATH
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**Q: proto生成代码位置不对**
```bash
# 清理后重新生成
make proto-clean
make proto

# 验证位置
ls controller/proto/*.pb.go
```

### 运行问题

**Q: Agent无法检测到进程死亡**
- 确保使用Go版本Agent（agent-go/），不是C++版本
- Go Agent已修复僵尸进程检测问题

**Q: 部署创建后实例立即启动**
- 新版本已修复：默认状态为Stopped
- 需要手动点击"启动"按钮

**Q: UI端口5173被占用**
```bash
# Vite会自动尝试5174、5175等端口
# 或手动指定端口
cd ui && npm run dev -- --port 5180
```

**Q: 生产环境如何部署UI？**
```bash
# 1. 构建静态文件
make ui-build

# 2. 使用nginx serve ui/dist/目录
# 详细步骤见: docs/NGINX_DEPLOYMENT.md

# 3. 或使用任何静态文件服务器
cd ui/dist && python3 -m http.server 8000
```

**Q: 如何查看日志？**
```bash
# Controller日志（前台运行时直接显示）
./controller/bin/controller

# 或重定向到文件
./controller/bin/controller > controller.log 2>&1 &

# Agent日志
make agent-run > agent.log 2>&1 &

# 多Agent日志（自动保存）
make agent-run-multi
tail -f logs/agent-nodeA.log
```

**Q: 数据库文件在哪里？**
- 默认位置：`./controller.db`（Controller运行目录）
- 自定义：`CONTROLLER_DB=/path/to/db.db ./controller/bin/controller`
- 备份：`cp controller.db controller.db.backup`

### 依赖问题

**Q: go install后找不到protoc-gen-go**
```bash
# 添加GOPATH/bin到PATH
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**Q: go install卡住或超时**
```bash
# 配置国内Go代理
go env -w GOPROXY=https://goproxy.cn,direct

# 重新安装
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

**Q: C++ SDK编译失败**
```bash
# 安装完整的C++开发环境
sudo apt install -y cmake build-essential pkg-config libgrpc++-dev libprotobuf-dev

# 验证安装
pkg-config --version
cmake --version
```

**Q: pkg-config not found**
```bash
sudo apt install pkg-config
```

**Q: C++ SDK下载依赖失败（无法访问GitHub）**
```bash
# 推荐：使用GitHub镜像构建
make sdk_cpp_mirror

# 详细方案见: sdk/cpp/NETWORK_SETUP.md
```

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