# 环境变量配置指南

Plum支持通过`.env`文件统一管理所有配置，无需手动设置环境变量。

## 配置优先级

```
环境变量 > .env文件 > 默认值
```

## 快速开始

### 1. Controller

```bash
cd controller
cp env.example .env
vim .env  # 修改配置

# 启动时自动加载.env
./bin/controller
```

### 2. Agent

```bash
cd agent-go
cp env.example .env
vim .env  # 修改配置

# 启动时自动加载.env
./plum-agent
```

### 3. 应用（C++ SDK）

```bash
cd examples/kv-demo
cp env.example .env
vim .env  # 修改配置

# SDK自动加载.env
./kv-demo
```

## 配置文件示例

### Controller (`controller/.env`)

```bash
CONTROLLER_ADDR=:8080
CONTROLLER_DB=file:controller.db
CONTROLLER_DATA_DIR=.
HEARTBEAT_TTL_SEC=30
FAILOVER_ENABLED=true
```

### Agent (`agent-go/.env`)

```bash
AGENT_NODE_ID=nodeA
CONTROLLER_BASE=http://127.0.0.1:8080
AGENT_DATA_DIR=/tmp/plum-agent
```

### 应用 (`app/.env`)

```bash
CONTROLLER_BASE=http://127.0.0.1:8080
PLUM_KV_SYNC_MODE=polling
```

## 实现细节

### Go组件（Controller/Agent）

使用 `github.com/joho/godotenv`：

```go
import "github.com/joho/godotenv"

func main() {
    godotenv.Load()  // 加载.env
    // ...
}
```

### C++ SDK（plumkv）

内置简单.env解析器：

```cpp
// 自动在create()时加载
auto kv = DistributedMemory::create("ns");
```

## 注意事项

1. **.env文件不会被提交到Git**（已在.gitignore中排除）
2. **env.example是模板**（会被提交，用于参考）
3. **环境变量优先级更高**（可用于临时覆盖）
4. **支持注释**（以#开头的行）
5. **支持引号**（`KEY="value"` 或 `KEY='value'`）

## 生产部署

生产环境建议：
- 使用systemd的`EnvironmentFile=`指令
- 或直接在.env中配置
- 或使用配置管理工具（如Ansible）

详见：`deploy/PRODUCTION_DEPLOYMENT.md`

