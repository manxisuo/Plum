# 分布式KV存储 API文档

Plum的分布式KV存储提供集群级别的键值对存储能力，结合了持久化的可靠性和内存缓存的快速访问特性，专为分布式任务编排的状态管理设计。

## 🎯 核心概念

### 命名空间（Namespace）
- 数据隔离的基本单位
- 建议使用 `instanceId` 或 `appName`
- 每个命名空间独立，互不干扰

### 数据类型
- `string`：字符串
- `int`：64位整数
- `double`：双精度浮点数
- `bool`：布尔值
- `bytes`：二进制数据（Base64编码存储）

### 存储架构
```
┌───────────────────────────────────┐
│          Controller               │
│  ┌─────────────────────────────┐  │
│  │  SQLite (distributed_kv)   │  │ ← 持久化存储
│  └─────────────────────────────┘  │
│  ┌─────────────────────────────┐  │
│  │  SSE Notification           │  │ ← 实时通知
│  └─────────────────────────────┘  │
└────────┬──────────┬───────────────┘
         │          │
    ┌────▼────┐ ┌──▼──────┐
    │ NodeA   │ │ NodeB   │
    │ (缓存)  │ │ (缓存)  │
    └─────────┘ └─────────┘
```

## 📡 REST API

### PUT - 存储键值

**请求：**
```http
PUT /v1/kv/{namespace}/{key}
Content-Type: application/json

{
  "value": "100",
  "type": "int"
}
```

**响应：**
```json
{
  "namespace": "app-instance-1",
  "key": "counter",
  "value": "100",
  "type": "int"
}
```

**类型说明：**
- `type` 可选值：`string`、`int`、`double`、`bool`
- 默认为 `string`

### GET - 获取键值

**请求：**
```http
GET /v1/kv/{namespace}/{key}
```

**响应（存在）：**
```json
{
  "namespace": "app-instance-1",
  "key": "counter",
  "value": "100",
  "type": "int",
  "updatedAt": 1697000000
}
```

**响应（不存在）：**
```http
404 Not Found
```

### DELETE - 删除键值

**请求：**
```http
DELETE /v1/kv/{namespace}/{key}
```

**响应：**
```http
204 No Content
```

### GET - 列出所有键值

**请求：**
```http
GET /v1/kv/{namespace}
```

**响应：**
```json
[
  {
    "namespace": "app-instance-1",
    "key": "counter",
    "value": "100",
    "type": "int",
    "updatedAt": 1697000000
  },
  {
    "namespace": "app-instance-1",
    "key": "status",
    "value": "running",
    "type": "string",
    "updatedAt": 1697000001
  }
]
```

### GET - 前缀查询

**请求：**
```http
GET /v1/kv/{namespace}?prefix=task.
```

**响应：**
```json
[
  {
    "namespace": "app-instance-1",
    "key": "task.progress",
    "value": "75",
    "type": "int",
    "updatedAt": 1697000000
  },
  {
    "namespace": "app-instance-1",
    "key": "task.status",
    "value": "running",
    "type": "string",
    "updatedAt": 1697000001
  }
]
```

### POST - 批量存储

**请求：**
```http
POST /v1/kv/{namespace}/batch
Content-Type: application/json

{
  "items": [
    {"key": "k1", "value": "v1", "type": "string"},
    {"key": "k2", "value": "100", "type": "int"}
  ]
}
```

**响应：**
```json
{
  "namespace": "app-instance-1",
  "count": 2
}
```

### GET - 列出所有命名空间

**请求：**
```http
GET /v1/kv
```

**响应：**
```json
{
  "namespaces": [
    "app-instance-1",
    "app-instance-2",
    "kv-demo"
  ]
}
```

**说明：**
- 返回当前存储中所有存在数据的命名空间列表
- 按字母顺序排序
- 用于Web UI展示命名空间概览

### GET - 列出命名空间下的所有键

**请求：**
```http
GET /v1/kv/{namespace}/keys
```

**响应：**
```json
{
  "namespace": "app-instance-1",
  "keys": [
    "app.crashed",
    "app.status",
    "counter",
    "task.progress"
  ]
}
```

**说明：**
- 只返回键名列表，不包含值（节省带宽）
- 按字母顺序排序
- 用于Web UI快速浏览命名空间中的键

## 🔔 SSE 变更通知

### 订阅特定命名空间

```javascript
// 订阅特定命名空间的变更
const es = new EventSource('/v1/stream');
es.addEventListener('kv', (event) => {
  const data = JSON.parse(event.data);
  console.log('KV changed:', data);
  // { namespace, key, value, type }
});
```

### 事件格式

```json
{
  "event": "kv",
  "data": {
    "namespace": "app-instance-1",
    "key": "counter",
    "value": "101",
    "type": "int"
  }
}
```

## 💻 C++ SDK

### 创建实例

```cpp
#include <plumkv/DistributedMemory.hpp>

using namespace plum::kv;

// 使用instanceId作为命名空间（推荐）
string instanceId = getenv("PLUM_INSTANCE_ID");
auto dm = DistributedMemory::create(instanceId);

// 使用appName作为命名空间（全局共享）
string appName = getenv("PLUM_APP_NAME");
auto dm = DistributedMemory::create(appName);

// 自定义命名空间
auto dm = DistributedMemory::create("my-custom-namespace");
```

### API参考

#### 字符串操作
```cpp
// 存储
bool success = dm->put("status", "running");

// 获取
string status = dm->get("status", "unknown");

// 检查存在
if (dm->exists("checkpoint")) {
    // ...
}

// 删除
bool removed = dm->remove("temp_key");
```

#### 类型化操作
```cpp
// 整数
dm->putInt("counter", 100);
int64_t count = dm->getInt("counter", 0);

// 浮点数
dm->putDouble("progress", 75.5);
double prog = dm->getDouble("progress", 0.0);

// 布尔
dm->putBool("enabled", true);
bool enabled = dm->getBool("enabled", false);

// 二进制数据（Base64编码存储）
struct MyData {
    int id;
    double value;
    char name[32];
};

MyData data = {123, 3.14, "test"};
dm->putBytes("struct.data", &data, sizeof(data));

// 读取并恢复
auto bytes = dm->getBytes("struct.data");
if (bytes.size() == sizeof(MyData)) {
    MyData* restored = reinterpret_cast<MyData*>(bytes.data());
    cout << "ID: " << restored->id << endl;
}

// vector版本
vector<uint8_t> buffer = {0x01, 0x02, 0xFF};
dm->putBytes("raw.data", buffer);
auto restored = dm->getBytes("raw.data");
```

#### 批量操作
```cpp
// 批量存储
map<string, string> data = {
    {"k1", "v1"},
    {"k2", "v2"}
};
dm->putBatch(data);

// 获取所有
auto all = dm->getAll();
for (const auto& [key, value] : all) {
    cout << key << " = " << value << endl;
}

// 刷新缓存
dm->refresh();
```

#### 变更订阅
```cpp
// 订阅变更通知
dm->subscribe([](const string& key, const string& value) {
    cout << "Key " << key << " changed to " << value << endl;
});
```

## 🎓 使用场景

### 1. 崩溃恢复

```cpp
auto dm = DistributedMemory::create(instanceId);

// 启动时检查崩溃标记
if (dm->exists("app.crashed")) {
    cout << "检测到崩溃，正在恢复..." << endl;
    
    // 恢复状态
    int progress = dm->getInt("task.progress", 0);
    string checkpoint = dm->get("task.checkpoint", "");
    
    // 从检查点继续
    resumeFrom(checkpoint, progress);
    
    // 清除崩溃标记
    dm->remove("app.crashed");
} else {
    // 正常启动
    startNew();
}

// 设置崩溃标记（异常退出时会保留）
dm->putBool("app.crashed", true);

// 定期保存状态
dm->putInt("task.progress", currentProgress);
dm->putString("task.checkpoint", "step_" + to_string(step));

// 正常退出时清除标记
signal(SIGTERM, [](int) {
    g_dm->remove("app.crashed");
    exit(0);
});
```

### 2. 分布式计数器

```cpp
auto dm = DistributedMemory::create("global-counters");

// 读取当前值
int count = dm->getInt("request.count", 0);

// 递增
dm->putInt("request.count", count + 1);

// 其他节点立即可见（通过SSE同步）
```

### 3. 配置共享

```cpp
auto dm = DistributedMemory::create("app-config");

// 中心配置管理
dm->put("log.level", "DEBUG");
dm->putInt("worker.max", 10);
dm->putBool("feature.enabled", true);

// 所有节点读取统一配置
string logLevel = dm->get("log.level", "INFO");
```

### 4. 任务协调

```cpp
auto dm = DistributedMemory::create("job-coordination");

// 分布式锁（简单版）
if (!dm->exists("task.lock")) {
    dm->put("task.lock", myNodeId);
    
    // 执行任务
    processTask();
    
    // 释放锁
    dm->remove("task.lock");
}

// 进度跟踪
dm->putInt("task.progress", currentProgress);

// 其他节点查询进度
int progress = dm->getInt("task.progress", 0);
```

### 5. 跨节点状态传递

```cpp
// NodeA: 步骤1完成，保存结果
dm->put("step1.result", resultData);
dm->putBool("step1.done", true);

// NodeB: 等待步骤1完成
while (!dm->getBool("step1.done", false)) {
    sleep(1);
}
string result = dm->get("step1.result");
// 执行步骤2...
```

## ⚙️ 工作原理

### 数据流

#### 写操作
```
Node1: dm->put("key", "value")
         ↓
    HTTP PUT /v1/kv/ns/key
         ↓
    Controller: SQLite INSERT
         ↓
    SSE notify: event=kv
         ↓
Node2/3/4: 更新本地缓存
```

#### 读操作
```
Node2: dm->get("key")
         ↓
    查本地缓存
         ↓
    命中 → 立即返回
    miss → HTTP GET /v1/kv/ns/key
         → 缓存结果
         → 返回
```

### 缓存策略

| 操作 | 缓存行为 |
|------|---------|
| put() | 写Controller + 更新本地缓存 |
| get() | 优先读缓存，miss时请求Controller |
| remove() | 删除Controller + 删除本地缓存 |
| refresh() | 重新加载所有数据到缓存 |

### 同步机制

1. **写同步**：
   - PUT请求成功 → 立即持久化到SQLite
   - 发送SSE通知 → 其他节点收到 → 更新缓存

2. **定期刷新**：
   - SDK每5秒调用 `refresh()` 同步最新数据
   - 防止SSE断线导致的数据不一致

3. **启动预加载**：
   - SDK初始化时调用 `GET /v1/kv/{namespace}`
   - 一次性加载所有数据到本地缓存

## 🔧 配置

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| CONTROLLER_BASE | http://127.0.0.1:8080 | Controller地址 |
| PLUM_INSTANCE_ID | - | Agent自动注入 |
| PLUM_APP_NAME | - | Agent自动注入 |

### 命名空间选择建议

| 场景 | 命名空间 | 说明 |
|------|---------|------|
| 崩溃恢复 | `PLUM_INSTANCE_ID` | 每个实例独立状态 |
| 全局配置 | `PLUM_APP_NAME` | 同应用所有实例共享 |
| 任务协调 | `job-{jobId}` | 同一任务的多个worker |
| 自定义 | 任意字符串 | 自定义隔离粒度 |

## 📊 性能特征

### 延迟
- **本地缓存读取**：~0.001ms
- **网络请求**：2-5ms（内网）
- **批量操作**：单次请求，节省往返

### 吞吐量
- **写操作**：1000+ ops/秒（单Controller）
- **读操作**：10000+ ops/秒（缓存命中）
- **并发连接**：100+ 节点

### 容量
- 受SQLite限制：TB级别
- 建议单namespace：< 10000个key
- 单个value：< 1MB

## ⚠️ 注意事项

### 1. 网络依赖
- 写操作需要访问Controller
- 网络断开时只能读取缓存（旧数据）

### 2. 一致性
- 强一致性（Controller单点写入）
- 最终一致性（SSE异步通知）

### 3. 并发写入
- 无内置分布式锁
- 需要应用层协调（如使用CAS模式）

### 4. 命名空间隔离
- 不同namespace完全隔离
- 选择合适的namespace策略

### 5. 数据持久化
- 数据存储在Controller的SQLite
- Controller重启不丢数据
- 建议定期备份SQLite文件

## 🧪 测试示例

### cURL测试

```bash
# 1. 存储键值
curl -X PUT http://localhost:8080/v1/kv/test-ns/counter \
  -H "Content-Type: application/json" \
  -d '{"value": "100", "type": "int"}'

# 2. 获取键值
curl http://localhost:8080/v1/kv/test-ns/counter

# 3. 列出所有
curl http://localhost:8080/v1/kv/test-ns

# 4. 前缀查询
curl http://localhost:8080/v1/kv/test-ns?prefix=task.

# 5. 批量存储
curl -X POST http://localhost:8080/v1/kv/test-ns/batch \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"key": "k1", "value": "v1", "type": "string"},
      {"key": "k2", "value": "200", "type": "int"}
    ]
  }'

# 6. 删除键值
curl -X DELETE http://localhost:8080/v1/kv/test-ns/counter
```

### C++ SDK测试

详见 [sdk/cpp/plumkv/README.md](../sdk/cpp/plumkv/README.md) 和 [examples/kv-demo](../examples/kv-demo/README.md)

## 🚧 未来规划

以下功能已规划但尚未实现：

### ⏳ TTL过期时间
```cpp
// API设计
dm->put("session.token", "abc123", 3600);  // 1小时后过期
dm->putWithTTL("cache.data", "value", chrono::seconds(300));

// 数据库添加expires_at字段
// Controller定期清理过期数据
```

**应用场景：**
- 临时会话数据
- 缓存管理
- 定时任务触发

### ⏳ 监听特定key的变更
```cpp
// API设计
dm->watch("task.status", [](const string& oldVal, const string& newVal) {
    cout << "Status changed from " << oldVal << " to " << newVal << endl;
});

// 只接收感兴趣的key的通知，减少无效处理
```

**应用场景：**
- 状态机触发
- 事件驱动架构
- 条件等待

### ⏳ 支持JSON对象和数组
```cpp
// API设计
dm->putJSON("config", R"({"host": "localhost", "port": 8080})");
json config = dm->getJSON("config");

dm->putArray("tasks", {"task1", "task2", "task3"});
vector<string> tasks = dm->getArray("tasks");
```

**应用场景：**
- 复杂配置管理
- 结构化数据存储
- 列表和集合操作

### ⏳ CAS原子操作
```cpp
// API设计（Compare-And-Swap）
bool success = dm->compareAndSwap("counter", "99", "100");
// 只有当前值是99时才更新为100

int newVal = dm->increment("counter", 1);  // 原子递增
```

**应用场景：**
- 分布式计数器
- 分布式锁
- 并发控制

### ⏳ 真正的SSE EventSource（替代定期轮询）
```cpp
// 当前实现：SDK每5秒定期调用refresh()
// 计划改进：使用cpp-httplib的SSE客户端实时接收

class SSEClient {
    void connect(const string& url);
    void onMessage(function<void(const string& event, const string& data)> handler);
};

// 在DistributedMemory中
sseClient_.connect(controllerURL_ + "/v1/stream?namespace=" + namespace_);
sseClient_.onMessage([this](const string& event, const string& data) {
    if (event == "kv") {
        updateCacheFromSSE(data);
    }
});
```

**优势：**
- 延迟更低（毫秒级 vs 秒级）
- 更节省资源（事件驱动 vs 轮询）
- 更实时（立即推送 vs 等待下次轮询）

### ⏳ gRPC协议支持（替代HTTP+Base64）

**当前实现**：HTTP REST + JSON + Base64编码二进制
**计划改进**：可选的gRPC接口

```protobuf
// proto定义
message KVPutRequest {
  string namespace = 1;
  string key = 2;
  bytes value = 3;      // 原生二进制支持
  string type = 4;
}

service KVService {
  rpc Put(KVPutRequest) returns (KVPutResponse);
  rpc Get(KVGetRequest) returns (KVGetResponse);
  rpc Watch(KVWatchRequest) returns (stream KVEvent);  // 双向流
}
```

**优势：**
- 二进制数据无需Base64（节省33%空间和编解码时间）
- 双向流支持（实时推送更高效）
- 类型安全（protobuf强类型）
- 性能更高（二进制协议）

**为什么现在不用？**
- HTTP REST已经够用（对于小型二进制数据）
- 简单性和一致性优先
- 避免协议混用（REST+gRPC）

**实施条件：**
- 当二进制数据成为性能瓶颈时
- 当需要大量二进制数据传输时（> 100KB频繁操作）
- 当需要双向流实时推送时

### ⏳ 其他可能的增强

- **事务支持**：批量操作的原子性保证
- **数据版本控制**：支持历史版本查询和回滚
- **权限控制**：基于namespace的访问控制
- **数据压缩**：大value自动压缩存储（gzip/lz4）
- **数据统计**：namespace使用情况、热点key分析

## 🎨 Web UI 查看

Plum提供了Web界面用于查看和管理KV存储。

### 访问方式

1. 启动Controller和UI
```bash
make controller-run    # 终端1
make ui-dev           # 终端2
```

2. 浏览器访问 `http://localhost:5173/kv-store`

### 功能特性

**命名空间列表**
- 展示所有存在数据的命名空间
- 显示每个命名空间中的键数量
- 点击展开查看详细键列表

**键名浏览**
- 列出指定命名空间下的所有键
- 按字母顺序排序
- 快速浏览数据结构

**统计信息**
- 命名空间总数
- 键总数

**使用场景**
- 🔍 调试应用状态存储
- 📊 监控KV数据分布
- 🧹 定位需要清理的数据

**限制**
- 当前版本**仅支持查看**，不支持编辑/删除
- 不显示键值内容（节省带宽）
- 未来版本将支持完整CRUD操作

### 截图示例

**命名空间列表视图：**
```
┌─────────────────────────────────────┐
│ Distributed KV Store          刷新  │
├─────────────────────────────────────┤
│ 命名空间: 3    总键数: 15           │
├─────────────────────────────────────┤
│ ▼ kv-demo                    5 个键 │
│   • app.crashed                      │
│   • app.status                       │
│   • counter                          │
│   • last_update                      │
│   • task.progress                    │
├─────────────────────────────────────┤
│ ▶ app-instance-1            8 个键  │
│ ▶ app-instance-2            2 个键  │
└─────────────────────────────────────┘
```

## 🔗 相关文档

- [C++ SDK文档](../sdk/cpp/plumkv/README.md)
- [KV Demo示例](../examples/kv-demo/README.md)
- [API总览](./API.md)

---

**设计理念**：简单、可靠、实用 - 为分布式任务编排提供持久化的状态管理能力。

**技术定位**：持久化的分布式KV存储（类似etcd/Consul），而非临时缓存  
**当前版本**：v1.0 - 核心KV功能完整可用  
**规划版本**：v2.0 - 增强特性逐步实现

