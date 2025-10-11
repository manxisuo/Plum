# Plum Worker Demo

一个集成gRPC Worker SDK的示例应用，演示如何在Plum中执行embedded任务。

## 📋 功能

- ✅ 启动gRPC服务器监听任务请求
- ✅ 自动注册到Controller
- ✅ 支持任务执行（demo.echo、demo.delay）
- ✅ 定期发送心跳
- ✅ 优雅处理SIGTERM信号
- ✅ 完整的构建和打包流程

## 🎯 与普通应用的区别

| 特性 | 普通应用 | Worker应用 |
|------|---------|-----------|
| 功能 | 独立运行 | 接受任务调度 |
| gRPC | 不需要 | 必须启动gRPC服务 |
| 注册 | 不需要 | 向Controller注册能力 |
| 任务 | 无 | 执行embedded任务 |

## 🔨 构建

### 前置条件
```bash
# 必须先生成proto代码
cd /home/stone/code/Plum
make proto
```

### 构建Worker Demo
```bash
cd examples/worker-demo

# 使用构建脚本
./build.sh
# 生成: worker-demo.zip
```

## 📦 部署到Plum

### 1. 上传应用
```bash
# Web UI上传worker-demo.zip
# 或API:
curl -X POST http://localhost:8080/v1/apps/upload \
  -F "file=@worker-demo.zip"
```

### 2. 创建部署
```bash
# 在Web UI创建部署
# - 选择worker-demo
# - 选择节点（如nodeA）
# - 副本数：1
```

### 3. 启动部署
```bash
# 点击"启动"按钮
# Worker会自动：
# 1. 启动gRPC服务（18090端口）
# 2. 注册到Controller
# 3. 开始接受任务
```

### 4. 验证Worker已注册
```bash
# Web UI查看"工作器"页面
# 或API:
curl http://localhost:8080/v1/embedded-workers
# 应该看到worker-demo
```

## 🎮 使用Worker执行任务

### 创建任务定义
```bash
# 在Web UI的"任务定义"页面：
# - 名称: demo-task
# - 执行器: embedded
# - 目标类型: app
# - 目标引用: worker-demo
# - Payload: {"message": "hello"}
```

### 运行任务
```bash
# 在任务定义列表点击"运行"
# Controller会：
# 1. 找到worker-demo的Worker
# 2. 通过gRPC调用Worker执行任务
# 3. 获取结果并显示
```

## 📊 查看运行状态

### Web UI
- "工作器"页面：查看Worker状态和支持的任务
- "任务运行"页面：查看任务执行历史和结果

### 日志
```bash
# Worker日志（Agent节点）
# 会看到:
# [Worker] Executing task: demo.echo (ID: xxx)
# [Worker] Payload: {"message":"hello"}
# [Worker] Task completed: demo.echo
```

## 🔧 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| PLUM_INSTANCE_ID | - | Agent注入 |
| PLUM_APP_NAME | worker-demo | Agent注入 |
| PLUM_APP_VERSION | 1.0.0 | Agent注入 |
| WORKER_ID | worker-demo-{instanceId} | Worker唯一ID |
| WORKER_NODE_ID | nodeA | 所在节点 |
| CONTROLLER_BASE | http://127.0.0.1:8080 | Controller地址 |
| GRPC_ADDRESS | 0.0.0.0:18090 | gRPC监听地址 |

## 📝 文件说明

```
worker-demo/
├── main.cpp              # Worker实现（gRPC服务）
├── CMakeLists.txt        # 构建配置（链接proto库）
├── start.sh              # 启动脚本
├── meta.ini              # 元数据（name、version、service）
├── build.sh              # 构建打包脚本
└── README.md             # 本文档
```

## 🎯 支持的任务

此Worker支持两个任务：

### 1. demo.echo
模拟echo任务，2秒后返回成功。

### 2. demo.delay
模拟延迟任务，可用于测试超时等场景。

## 🧪 完整测试流程

```bash
# 1. 构建并上传
cd examples/worker-demo
./build.sh
# 上传worker-demo.zip到Plum

# 2. 创建部署并启动
# Web UI操作

# 3. 创建任务定义
# - executor: embedded
# - targetKind: app
# - targetRef: worker-demo

# 4. 运行任务
# 点击任务定义的"运行"按钮

# 5. 查看结果
# 在"任务运行"页面查看执行结果
```

## 🔄 扩展功能

基于此demo，可以实现：
- 添加更多任务类型
- 实现实际业务逻辑
- 集成数据库操作
- 调用外部API
- 数据处理和转换

## ⚠️ 注意事项

1. **端口冲突**：默认18090端口，如果多实例需要不同端口
2. **proto依赖**：必须先`make proto`生成proto代码
3. **gRPC依赖**：需要安装libgrpc++-dev
4. **网络访问**：Worker需要能访问Controller

---

**提示**：这是一个完整的Worker示例，展示了Plum的核心能力：动态任务调度和执行。

