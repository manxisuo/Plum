# Stream Worker SDK 使用指南

Stream Worker SDK 是 Plum 的新版 Worker SDK，基于 gRPC 双向流实现，提供了简洁的 API，让用户只需关注业务逻辑。

## 特性

- ✅ **简洁的 API**：只需注册任务处理函数，无需关心底层实现
- ✅ **自动管理**：自动处理连接、注册、心跳、重连等
- ✅ **环境变量支持**：自动从环境变量读取配置
- ✅ **线程安全**：内部处理了所有并发问题
- ✅ **自动重连**：连接断开后自动重连

## 快速开始

### 1. 包含头文件

```cpp
#include "plumworker/stream_worker.hpp"
using namespace plumworker;
```

### 2. 创建 Worker 并配置

```cpp
StreamWorkerOptions options;
// 大部分配置可以从环境变量自动读取
// 只需要设置必要的选项
options.labels["type"] = "my-worker";

StreamWorker worker(options);
```

### 3. 注册任务处理函数

```cpp
worker.registerTask("my.task.name", [](const std::string& taskId, 
                                       const std::string& taskName, 
                                       const std::string& payload) -> std::string {
    // 处理任务逻辑
    // taskId: 任务ID
    // taskName: 任务名称（与注册时相同）
    // payload: 任务负载（JSON字符串）
    
    // 返回任务结果（JSON字符串）
    return "{\"status\":\"success\",\"result\":\"...\"}";
});
```

### 4. 启动 Worker

```cpp
// 阻塞调用，直到 Worker 停止
worker.start();
```

## 完整示例

```cpp
#include "plumworker/stream_worker.hpp"
#include <signal.h>
#include <atomic>

std::atomic<bool> g_running{true};

void signal_handler(int sig) {
    g_running = false;
}

int main() {
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    // 配置 Worker
    StreamWorkerOptions options;
    options.labels["type"] = "demo";

    // 创建 Worker
    StreamWorker worker(options);

    // 注册任务处理函数
    worker.registerTask("demo.echo", [](const std::string& taskId, 
                                        const std::string& taskName, 
                                        const std::string& payload) -> std::string {
        return "{\"status\":\"success\",\"echo\":\"" + payload + "\"}";
    });

    worker.registerTask("demo.delay", [](const std::string& taskId, 
                                         const std::string& taskName, 
                                         const std::string& payload) -> std::string {
        std::this_thread::sleep_for(std::chrono::seconds(2));
        return "{\"status\":\"success\",\"message\":\"Delayed task completed\"}";
    });

    // 启动 Worker
    worker.start();

    return 0;
}
```

## 配置选项

`StreamWorkerOptions` 支持以下配置：

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `controllerGrpcAddr` | `std::string` | `"127.0.0.1:9090"` | Controller gRPC 地址 |
| `workerId` | `std::string` | 从 `WORKER_ID` 环境变量读取 | Worker ID |
| `nodeId` | `std::string` | 从 `WORKER_NODE_ID` 环境变量读取 | Node ID |
| `instanceId` | `std::string` | 从 `PLUM_INSTANCE_ID` 环境变量读取 | Instance ID |
| `appName` | `std::string` | 从 `PLUM_APP_NAME` 环境变量读取 | App Name |
| `appVersion` | `std::string` | 从 `PLUM_APP_VERSION` 环境变量读取 | App Version |
| `tasks` | `std::vector<std::string>` | 自动从注册的任务生成 | 支持的任务列表 |
| `labels` | `std::map<std::string, std::string>` | `{}` | 标签 |
| `heartbeatIntervalSec` | `int` | `30` | 心跳间隔（秒） |
| `reconnectIntervalSec` | `int` | `5` | 重连间隔（秒） |
| `autoReconnect` | `bool` | `true` | 是否自动重连 |

## 环境变量

SDK 会自动从以下环境变量读取配置：

- `CONTROLLER_GRPC_ADDR`: Controller gRPC 地址
- `WORKER_ID`: Worker ID
- `WORKER_NODE_ID`: Node ID
- `PLUM_INSTANCE_ID`: Instance ID
- `PLUM_APP_NAME`: App Name
- `PLUM_APP_VERSION`: App Version

## API 参考

### `StreamWorker::registerTask()`

注册任务处理函数。

```cpp
void registerTask(const std::string& taskName, TaskHandler handler);
```

- `taskName`: 任务名称（如 `"demo.echo"`）
- `handler`: 任务处理函数，类型为 `TaskHandler`：
  ```cpp
  using TaskHandler = std::function<std::string(
      const std::string& taskId,      // 任务ID
      const std::string& taskName,    // 任务名称
      const std::string& payload      // 任务负载（JSON字符串）
  )>;
  ```

### `StreamWorker::start()`

启动 Worker。这是阻塞调用，会一直运行直到 Worker 停止。

```cpp
bool start();
```

返回 `true` 表示启动成功，`false` 表示启动失败。

### `StreamWorker::stop()`

停止 Worker。

```cpp
void stop();
```

### `StreamWorker::isRunning()`

检查 Worker 是否正在运行。

```cpp
bool isRunning() const;
```

## 与旧版 Worker 的区别

| 特性 | 旧版 HTTP Worker | 新版 Stream Worker |
|------|------------------|-------------------|
| 协议 | HTTP | gRPC 双向流 |
| Worker 角色 | 服务端（监听端口） | 客户端（连接到 Controller） |
| 端口管理 | 需要管理端口 | 无需端口 |
| 网络环境 | 受 NAT/防火墙限制 | 不受限制 |
| API 复杂度 | 中等 | 简单 |
| 推荐使用 | 旧版应用 | **新应用推荐** |

## 注意事项

1. **任务处理函数应该是线程安全的**：多个任务可能并发执行
2. **任务处理函数不应该阻塞太久**：如果任务需要长时间运行，考虑异步处理
3. **返回值应该是有效的 JSON 字符串**：Controller 会解析返回的结果
4. **异常处理**：如果任务处理函数抛出异常，SDK 会自动捕获并发送错误结果

