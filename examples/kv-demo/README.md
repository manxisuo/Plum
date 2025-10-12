# Plum KV Demo - 分布式KV存储崩溃恢复演示

演示如何使用Plum分布式KV存储实现应用崩溃后的状态恢复。

## 🎯 功能演示

### 核心特性
- ✅ 使用appName作为命名空间（所有实例共享状态）
- ✅ 定期保存任务进度到分布式KV存储
- ✅ 崩溃或停止后自动恢复到上次保存的状态
- ✅ 正常退出时清除崩溃标记
- ✅ 异常退出时保留崩溃标记供恢复
- ✅ 支持跨节点迁移（主备切换）

### 崩溃恢复流程

```
启动 → 检查崩溃标记 → 是 → 恢复状态 → 继续执行
                  ↓
                  否 → 正常启动 → 从0%开始
```

## 🔨 构建

```bash
cd examples/kv-demo
./build.sh
# 生成: kv-demo.zip
```

## ⚙️ 配置

本应用使用 Plum KV SDK，配置示例见：`../../sdk/cpp/plumkv/env.example`

```bash
# 复制配置模板
cp ../../sdk/cpp/plumkv/env.example .env
vim .env  # 修改配置

# 主要配置项：
# CONTROLLER_BASE=http://127.0.0.1:8080
# PLUM_KV_SYNC_MODE=polling  # 或 sse/disabled
```

**注意：** Agent部署时会自动注入 `PLUM_INSTANCE_ID`、`PLUM_APP_NAME` 等环境变量。

## 📦 部署测试

### 1. 上传并部署
```bash
# Web UI上传 kv-demo.zip
# 创建部署 → 选择节点 → 启动
```

### 2. 观察正常运行
```bash
# 查看日志，应该看到：
# 🆕 正常启动（无崩溃记录）
# 🚀 开始执行任务...
# 📊 进度: 10% | 计数: 1 | 检查点: step_1
# 📊 进度: 20% | 计数: 2 | 检查点: step_2
# ...
```

### 3. 模拟崩溃
```bash
# 在应用运行到50%左右时
# SSH到Agent节点
ps aux | grep kv-demo
kill -9 <PID>  # 强制杀掉进程（模拟崩溃）
```

### 4. 观察自动恢复
```bash
# Agent会自动重启应用
# 查看日志，应该看到：
# 🔄 检测到崩溃标记，正在恢复...
#   上次进度: 50%
#   任务计数: 5
#   检查点: step_5
# ✅ 状态恢复完成，从 50% 继续执行
# 📊 进度: 60% | 计数: 6 | 检查点: step_6
# ...
```

### 5. 正常退出测试
```bash
# 等待任务完成或按Ctrl+C
# 应该看到：
# [KV Demo] Received signal 2, shutting down gracefully...
# [KV Demo] Cleared crash flag
# ✅ 任务完成！
```

### 6. 验证清除
```bash
# 再次启动应用
# 应该看到：
# 🆕 正常启动（无崩溃记录）
# 说明正常退出时崩溃标记已清除
```

## 🔍 分布式KV存储API演示

### 基本操作
```cpp
// 创建实例（使用instanceId隔离）
auto dm = DistributedMemory::create(instanceId);

// 字符串操作
dm->put("status", "running");
string status = dm->get("status", "unknown");

// 整数操作
dm->putInt("counter", 100);
int count = dm->getInt("counter", 0);

// 浮点数操作
dm->putDouble("progress", 75.5);
double prog = dm->getDouble("progress", 0.0);

// 布尔操作
dm->putBool("crashed", true);
bool isCrashed = dm->getBool("crashed", false);

// 检查存在
if (dm->exists("checkpoint")) {
    string cp = dm->get("checkpoint");
}

// 删除
dm->remove("temp_data");
```

### 批量操作
```cpp
// 批量保存
map<string, string> checkpoint = {
    {"progress", "75"},
    {"status", "step5"},
    {"timestamp", to_string(time(nullptr))}
};
dm->putBatch(checkpoint);

// 获取所有数据
auto all = dm->getAll();
for (const auto& [k, v] : all) {
    cout << k << " = " << v << endl;
}
```

### 变更订阅
```cpp
// 订阅变更通知（高级功能）
dm->subscribe([](const string& key, const string& value) {
    cout << "KV changed: " << key << " = " << value << endl;
});
```

## 🎓 崩溃恢复模式

### 模式1：检查点恢复（本示例）
```cpp
// 定期保存检查点
dm->putInt("progress", currentProgress);
dm->putString("checkpoint", currentStep);

// 重启时恢复
if (dm->exists("crashed")) {
    int progress = dm->getInt("progress", 0);
    string step = dm->get("checkpoint");
    resumeFrom(step, progress);
}
```

### 模式2：事务日志
```cpp
// 记录操作日志
dm->putInt("operation.counter", opCount);
dm->putString("operation.last", "UPDATE_USER_123");

// 重启时重放
if (dm->exists("crashed")) {
    int lastOp = dm->getInt("operation.counter", 0);
    replayFrom(lastOp);
}
```

### 模式3：状态机恢复
```cpp
// 保存状态机状态
dm->putString("fsm.state", "STATE_PROCESSING");
dm->putString("fsm.data", serializeData());

// 重启时恢复状态机
if (dm->exists("crashed")) {
    string state = dm->get("fsm.state", "STATE_INIT");
    string data = dm->get("fsm.data");
    restoreFSM(state, deserialize(data));
}
```

## 📊 数据隔离

### 命名空间策略（本demo使用appName）

```cpp
// 本demo采用方案A: 按应用名（所有实例共享状态）
string appName = getenv("PLUM_APP_NAME");
auto dm = DistributedMemory::create(appName);
// 所有kv-demo实例共享进度和状态

// 适用场景：
// - 主备切换：instance1崩溃，instance2接管继续
// - 分布式任务：多个实例协作完成同一任务
// - 全局状态：需要跨实例共享的配置和状态
```

**其他可选方案：**
```cpp
// 方案B: 按实例ID（每个实例独立）
string instanceId = getenv("PLUM_INSTANCE_ID");
auto dm = DistributedMemory::create(instanceId);
// 每个实例有独立的状态空间，互不干扰

// 方案C: 自定义命名空间
auto dm = DistributedMemory::create("myapp-global");
```

## ⚠️ 注意事项

1. **网络依赖**：需要能访问Controller
2. **命名空间选择**：多实例应用建议用instanceId隔离
3. **数据持久化**：数据存储在Controller的SQLite中
4. **崩溃标记**：异常退出时不清除，重启时检测
5. **正常退出**：需要显式清除崩溃标记

## 🔧 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| PLUM_INSTANCE_ID | kv-demo-001 | Agent注入 |
| PLUM_APP_NAME | kv-demo | Agent注入 |
| CONTROLLER_BASE | http://127.0.0.1:8080 | Controller地址 |

## 📝 文件说明

```
kv-demo/
├── main.cpp           # 崩溃恢复演示
├── CMakeLists.txt     # 构建配置
├── start.sh           # 启动脚本
├── meta.ini           # 应用元数据
├── build.sh           # 构建打包脚本
└── README.md          # 本文档
```

---

**提示**：这个demo展示了Plum分布式KV存储的核心价值 - 让应用具备崩溃恢复能力！

