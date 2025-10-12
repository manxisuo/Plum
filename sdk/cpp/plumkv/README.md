# Plum KV SDK - C++分布式KV存储客户端

Plum的分布式KV存储C++ SDK，提供集群级别的键值对存储能力。
结合了持久化的可靠性和内存缓存的快速访问特性。

## 🎯 核心特性

- ✅ **持久化存储**：数据保存在Controller的SQLite中，不会丢失
- ✅ **快速访问**：本地缓存提供内存般的读取速度
- ✅ **命名空间隔离**：多应用/实例互不干扰
- ✅ **类型安全**：支持 string/int/double/bool/bytes
- ✅ **双模式同步**：支持轮询（稳定）和SSE（实时）
- ✅ **崩溃恢复**：支持应用崩溃后状态恢复
- ✅ **批量操作**：减少网络开销

## 📦 依赖

- C++17
- nlohmann/json (自动下载)
- cpp-httplib (自动下载)
- pthread

## 🔨 构建

```bash
cd sdk/cpp/plumkv
mkdir build && cd build
cmake ..
make
sudo make install  # 可选
```

### 使用GitHub镜像（中国网络）
```bash
cmake -DUSE_GITHUB_MIRROR=ON ..
```

## 📖 API参考

### 创建实例

```cpp
#include <plumkv/DistributedMemory.hpp>

using namespace plum::kv;

// 工厂方法（推荐）
auto dm = DistributedMemory::create("my-namespace");

// 指定Controller地址
auto dm = DistributedMemory::create("my-namespace", "http://192.168.1.100:8080");
```

### 环境变量配置

**同步模式（PLUM_KV_SYNC_MODE）：**

```bash
# 轮询模式（默认）- 5秒刷新一次，稳定可靠
export PLUM_KV_SYNC_MODE=polling

# SSE推送模式 - 实时更新，低延迟
export PLUM_KV_SYNC_MODE=sse

# 禁用同步 - 仅本地缓存，不从Controller同步
export PLUM_KV_SYNC_MODE=disabled
```

**模式对比：**

| 模式 | 延迟 | 带宽占用 | 适用场景 |
|------|------|---------|---------|
| **polling** | ~5秒 | 中等 | 默认推荐，弱网环境，移动网络 |
| **sse** | 毫秒级 | 低（idle时几乎为0） | 实时性要求高，稳定网络 |
| **disabled** | N/A | 极低 | 无跨节点同步需求，纯本地使用 |

### 基本操作

```cpp
// 字符串
dm->put("key", "value");
string val = dm->get("key", "default");

// 整数
dm->putInt("counter", 100);
int64_t count = dm->getInt("counter", 0);

// 浮点数
dm->putDouble("pi", 3.14159);
double pi = dm->getDouble("pi", 0.0);

// 布尔
dm->putBool("enabled", true);
bool enabled = dm->getBool("enabled", false);

// 二进制数据（Base64编码存储）
struct MyState {
    int counter;
    double value;
};
MyState state = {100, 3.14};
dm->putBytes("app.state", &state, sizeof(state));

// 读取二进制
auto bytes = dm->getBytes("app.state");
if (bytes.size() == sizeof(MyState)) {
    MyState* restored = reinterpret_cast<MyState*>(bytes.data());
    cout << "Counter: " << restored->counter << endl;
}

// vector版本
vector<uint8_t> data = {0x01, 0x02, 0xFF};
dm->putBytes("raw.data", data);
auto restored = dm->getBytes("raw.data");

// C风格接口（输出到预分配buffer）
char buffer[1024];
size_t bufferSize = sizeof(buffer);
if (dm->getBytes("app.state", buffer, bufferSize)) {
    cout << "成功读取 " << bufferSize << " 字节" << endl;
    MyState* state = reinterpret_cast<MyState*>(buffer);
} else {
    if (bufferSize > sizeof(buffer)) {
        cout << "缓冲区太小，需要 " << bufferSize << " 字节" << endl;
    } else {
        cout << "key不存在" << endl;
    }
}

// 检查存在
if (dm->exists("checkpoint")) {
    // ...
}

// 删除
dm->remove("temp");
```

### 批量操作

```cpp
// 批量保存（单次HTTP请求）
map<string, string> data = {
    {"progress", "75"},
    {"status", "running"},
    {"checkpoint", "step5"}
};
dm->putBatch(data);

// 获取所有
auto all = dm->getAll();
for (const auto& [k, v] : all) {
    cout << k << " = " << v << endl;
}
```

### 刷新缓存

```cpp
// 手动刷新（从Controller重新加载）
dm->refresh();
```

### 变更订阅

```cpp
// 订阅变更通知
dm->subscribe([](const string& key, const string& value) {
    cout << "Key changed: " << key << " -> " << value << endl;
});
```

## 🎮 使用场景

### 场景1：崩溃恢复

```cpp
auto dm = DistributedMemory::create(instanceId);

// 检查是否从崩溃中恢复
if (dm->exists("app.crashed")) {
    cout << "恢复中..." << endl;
    int progress = dm->getInt("task.progress", 0);
    string checkpoint = dm->get("task.checkpoint", "");
    resumeFrom(checkpoint, progress);
    dm->remove("app.crashed");
} else {
    cout << "正常启动" << endl;
    startNew();
}

// 设置崩溃标记（异常退出时保留）
dm->putBool("app.crashed", true);

// 定期保存状态
dm->putInt("task.progress", currentProgress);
dm->putString("task.checkpoint", currentStep);

// 正常退出时清除标记
signal(SIGTERM, [](int) {
    dm->remove("app.crashed");
    exit(0);
});
```

### 场景2：分布式计数器

```cpp
auto dm = DistributedMemory::create("global");

// 原子递增（需要CAS，暂未实现）
int count = dm->getInt("counter", 0);
dm->putInt("counter", count + 1);
```

### 场景3：配置共享

```cpp
auto dm = DistributedMemory::create("app-config");

// NodeA: 写配置
dm->put("log_level", "DEBUG");
dm->putInt("max_workers", 10);

// NodeB/C/D: 读配置
string logLevel = dm->get("log_level", "INFO");
int maxWorkers = dm->getInt("max_workers", 4);
```

### 场景4：任务协调

```cpp
auto dm = DistributedMemory::create("job-coordination");

// Worker1: 获取任务
if (!dm->exists("task.lock")) {
    dm->put("task.lock", "worker1");
    processTask();
    dm->remove("task.lock");
}

// Worker2: 检查进度
int progress = dm->getInt("task.progress", 0);
cout << "任务进度: " << progress << "%" << endl;
```

## 🔧 配置

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| CONTROLLER_BASE | http://127.0.0.1:8080 | Controller地址 |
| PLUM_INSTANCE_ID | - | Agent注入的实例ID |

### 命名空间建议

| 使用场景 | 命名空间 | 隔离级别 |
|---------|---------|---------|
| 崩溃恢复 | instanceId | 每个实例独立 |
| 全局配置 | appName | 同一应用共享 |
| 自定义 | 任意字符串 | 自定义隔离 |

## 📊 性能特征

### 操作延迟（本地缓存命中）
- get(): ~0.001ms（内存读取）
- exists(): ~0.001ms
- getAll(): ~0.01ms

### 操作延迟（网络请求）
- put(): 2-5ms（写Controller + 更新缓存）
- get() (miss): 2-5ms（请求Controller + 缓存）
- remove(): 2-5ms

### 同步延迟
- 写入后通知其他节点：< 100ms（SSE推送）
- 定期刷新：5秒（可配置）

## ⚙️ 工作原理

### 三层架构
```
┌──────────────┐
│  应用代码    │
│ dm->put(...) │
└──────┬───────┘
       │
┌──────▼───────┐
│  本地缓存    │ ← 读取优先
│ map<k,v>     │
└──────┬───────┘
       │
┌──────▼───────┐
│  HTTP Client │ ← 写入/miss时
│ → Controller │
└──────────────┘
```

### 数据流

**写流程：**
```
put(k,v) → HTTP PUT → Controller SQLite
                    → SSE notify
                    → 其他节点更新缓存
```

**读流程：**
```
get(k) → 查本地缓存
       → 命中: 立即返回
       → miss: HTTP GET → 缓存并返回
```

## 🧪 集成到应用

### CMakeLists.txt
```cmake
# 添加plumkv SDK
add_subdirectory(../../sdk/cpp/plumkv plumkv)

add_executable(myapp main.cpp)
target_link_libraries(myapp PRIVATE plumkv)
```

### 代码示例
```cpp
#include <plumkv/DistributedMemory.hpp>

int main() {
    string instanceId = getenv("PLUM_INSTANCE_ID");
    auto dm = plum::kv::DistributedMemory::create(instanceId);
    
    // 使用分布式内存
    dm->putInt("counter", 1);
    
    return 0;
}
```

## 🔗 相关文档

- [KV Demo示例](../../examples/kv-demo/README.md)
- [Plum文档](../../../README.md)

---

**提示**：分布式内存让您的应用具备集群级别的状态共享和崩溃恢复能力！

