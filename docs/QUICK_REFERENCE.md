# Plum 快速参考卡片

## 🚀 一键启动

```bash
# Terminal 1: Controller
cd /home/stone/code/Plum && ./controller/bin/controller

# Terminal 2: UI
cd /home/stone/code/Plum/ui && npm run dev

# Terminal 3: Agent (可选)
AGENT_NODE_ID=nodeA ./agent-go/plum-agent

# Terminal 4: gRPC Worker示例
cd /home/stone/code/Plum
PLUM_INSTANCE_ID=grpc-instance-001 \
PLUM_APP_NAME=grpc-echo-app \
PLUM_APP_VERSION=v2.0.0 \
WORKER_ID=grpc-echo-1 \
WORKER_NODE_ID=nodeA \
CONTROLLER_BASE=http://127.0.0.1:8080 \
GRPC_ADDRESS=0.0.0.0:18082 \
./sdk/cpp/build/examples/grpc_echo_worker/grpc_echo_worker
```

## 🌐 重要URL

- **UI**: http://localhost:5173 或 5174
- **API**: http://127.0.0.1:8080
- **Swagger**: http://127.0.0.1:8080/swagger
- **健康检查**: http://127.0.0.1:8080/healthz

## 🔑 核心概念速查

### 执行器类型
| 类型 | 用途 | TargetKind | TargetRef |
|------|------|-----------|-----------|
| embedded | 嵌入式任务 | node/app | 节点ID/应用名 |
| service | HTTP服务调用 | service | 服务名（必填） |
| os_process | 系统命令 | node | 节点ID（可选） |

### 任务状态
- **Pending** → **Running** → **Succeeded**
- **Failed** / **Timeout** / **Canceled**

### Worker类型
- **HTTP Worker**：旧版，启动HTTP服务器
- **gRPC Worker**：新版，主动注册，性能更好

## 📁 关键文件

### 后端
- `controller/internal/tasks/scheduler.go` - 调度核心
- `controller/internal/store/sqlite/sqlite.go` - 数据库
- `controller/internal/httpapi/routes.go` - API路由

### 前端
- `ui/src/views/` - 所有页面
- `ui/src/router.ts` - 路由配置
- `ui/src/i18n.ts` - 国际化
- `ui/src/components/IdDisplay.vue` - ID显示组件

### SDK
- `sdk/cpp/plumworker/` - HTTP Worker SDK
- `sdk/cpp/plumresource/` - Resource SDK
- `sdk/cpp/examples/grpc_echo_worker/` - gRPC Worker示例

## 🛠️ 常用命令

```bash
# 构建
make controller              # 构建Controller
make agent                   # 构建Go Agent
make agent-clean             # 清理Agent编译产物
make proto                   # 编译proto文件（Go+C++）
make sdk_cpp                 # 构建C++ SDK
make sdk_cpp_grpc_echo_worker # 构建gRPC Worker示例

# 运行
make controller-run          # 运行Controller
make agent-run               # 运行Go Agent (nodeA)
make agent-runA/B/C          # 运行指定节点的Go Agent
make agent-run-multi         # 后台运行3个Go Agent
make agent-help              # 显示Agent命令帮助

# 测试API
curl -s http://127.0.0.1:8080/v1/nodes | jq .
curl -s http://127.0.0.1:8080/v1/embedded-workers | jq .
curl -s http://127.0.0.1:8080/v1/tasks | jq .

# 清理进程
pkill -f controller
pkill -f "npm run dev"
pkill -f grpc_echo_worker
```

## 🎯 UI页面导航

1. **/** - 首页
2. **/nodes** - 节点管理
3. **/apps** - 应用包管理
4. **/deployments** - 部署管理
5. **/assignments** - 实例分配
6. **/services** - 服务发现
7. **/tasks** - 任务管理
8. **/workflows** - 工作流管理
9. **/resources** - 资源管理
10. **/workers** - 工作器管理

## 🔧 调试技巧

### 查看日志
```bash
# Controller日志
./controller/bin/controller 2>&1 | tee controller.log

# Worker日志
./sdk/cpp/build/examples/grpc_echo_worker/grpc_echo_worker 2>&1 | tee worker.log
```

### 检查进程
```bash
ps aux | grep controller
ps aux | grep agent
netstat -tlnp | grep :8080
```

### 数据库查询
```bash
sqlite3 data/plum.db "SELECT * FROM embedded_workers;"
sqlite3 data/plum.db "SELECT * FROM tasks ORDER BY created_at DESC LIMIT 5;"
```

## 💡 快速修复

### UI显示空白
```typescript
// 确保数组初始化
items.value = Array.isArray(data) ? data : []
```

### 字段名不匹配
```typescript
// 同时检查两种命名
row.field || row.Field
```

### 列宽过小
```vue
<!-- 带图标的tag至少需要120px -->
<el-table-column width="120">
```

### ID太长
```vue
<!-- 使用IdDisplay组件 -->
<IdDisplay :id="someId" :length="8" />
```

## 📊 数据流向

```
用户 → UI (Vue) → Controller API (Go) → SQLite
                      ↓
                  Scheduler → Worker/Service
                      ↓
                  更新状态 → SSE推送 → UI更新
```

## 🌍 国际化

### 添加新翻译
```typescript
// ui/src/i18n.ts
messages = {
  en: { 
    common: { newKey: 'New Text' }
  },
  zh: { 
    common: { newKey: '新文本' }
  }
}
```

### 使用翻译
```vue
{{ t('common.newKey') }}
```

## 📦 依赖管理

### Go
```bash
go mod tidy
go mod download
```

### Node.js
```bash
cd ui
npm install
npm update
```

### C++
- httplib (header-only)
- nlohmann/json (header-only)
- gRPC (系统安装)
- protobuf (系统安装)

---

**提示**：详细信息请查看 `docs/PROJECT_SUMMARY.md`

