# Agent Makefile 命令速查

## 📋 完整命令列表

### 🔨 构建命令

```bash
# 构建Go Agent（推荐）
make agent

# 构建C++ Agent（旧版备份）
make agent-cpp

# 清理所有Agent编译产物
make agent-clean
```

### 🚀 运行命令

#### 单节点运行
```bash
# 运行Go Agent，默认nodeA
make agent-run

# 运行指定节点
make agent-runA    # nodeA
make agent-runB    # nodeB
make agent-runC    # nodeC
make agent-runD    # nodeD
make agent-runE    # nodeE
# ... 支持任意节点ID
```

#### 多节点运行
```bash
# 后台运行3个Agent节点（nodeA/B/C）
make agent-run-multi

# 查看日志
tail -f logs/agent-nodeA.log
tail -f logs/agent-nodeB.log
tail -f logs/agent-nodeC.log

# 停止所有Agent
pkill -f plum-agent
```

#### C++ Agent（旧版）
```bash
# 运行C++ Agent
make agent-cpp-run      # nodeA
make agent-cpp-runA     # nodeA
make agent-cpp-runB     # nodeB
```

### ℹ️ 帮助命令
```bash
# 显示Agent命令帮助
make agent-help
```

## 🎯 常用场景

### 场景1：开发测试（单节点）
```bash
# 终端1：启动Controller
make controller-run

# 终端2：启动Agent
make agent && make agent-run
```

### 场景2：多节点测试
```bash
# 终端1：启动Controller
make controller-run

# 终端2：启动3个Agent节点（后台）
make agent && make agent-run-multi

# 查看节点状态
curl -s http://127.0.0.1:8080/v1/nodes | jq .

# 查看日志
tail -f logs/agent-*.log
```

### 场景3：快速重启
```bash
# 停止所有Agent
pkill -f plum-agent

# 重新编译并启动
make agent-clean && make agent && make agent-run
```

### 场景4：对比测试Go vs C++
```bash
# 终端1：Go Agent
make agent && make agent-runA

# 终端2：C++ Agent
make agent-cpp && make agent-cpp-runB
```

## 🔧 环境变量自定义

所有agent命令支持环境变量覆盖：

```bash
# 自定义节点ID
AGENT_NODE_ID=myNode make agent-run

# 自定义Controller地址
CONTROLLER_BASE=http://192.168.1.100:8080 make agent-run

# 自定义数据目录
AGENT_DATA_DIR=/var/plum-agent make agent-run

# 组合使用
AGENT_NODE_ID=edge01 \
CONTROLLER_BASE=http://192.168.1.100:8080 \
AGENT_DATA_DIR=/opt/plum \
make agent-run
```

## 📊 命令对比表

| 任务 | 旧命令 | 新命令 | 改进 |
|------|--------|--------|------|
| 构建Agent | `cd agent-go && go build` | `make agent` | ✅ 简化 |
| 运行nodeA | `AGENT_NODE_ID=nodeA ... ./agent-go/plum-agent` | `make agent-run` | ✅ 简化 |
| 运行nodeB | `AGENT_NODE_ID=nodeB ... ./agent-go/plum-agent` | `make agent-runB` | ✅ 简化 |
| 清理 | `rm agent-go/plum-agent` | `make agent-clean` | ✅ 统一 |
| 多节点 | 手动启动3次 | `make agent-run-multi` | ✅ 自动化 |
| 查看帮助 | 查文档 | `make agent-help` | ✅ 内置 |

## 🎨 输出示例

### agent-run 输出
```
$ make agent-run
Starting Go Agent (nodeA)...
2025/10/09 21:00:00 Starting Plum Agent
2025/10/09 21:00:00   NodeID: nodeA
2025/10/09 21:00:00   Controller: http://127.0.0.1:8080
2025/10/09 21:00:00   DataDir: /tmp/plum-agent
```

### agent-run-multi 输出
```
$ make agent-run-multi
Starting multiple Go Agents...
Started nodeA (PID: 12345)
Started nodeB (PID: 12346)
Started nodeC (PID: 12347)
✅ 3 agents started. Logs in logs/agent-*.log
   To stop: pkill -f plum-agent
```

### agent-clean 输出
```
$ make agent-clean
Cleaning agent build artifacts...
✅ Agent artifacts cleaned
```

## 🔍 故障排查

### 问题：make agent-run 提示文件不存在
```bash
# 解决：先构建
make agent
```

### 问题：多个Agent端口冲突
```bash
# Agent不监听端口，不会冲突
# 确保每个Agent的NODE_ID不同即可
```

### 问题：停止所有Agent
```bash
# 方法1：使用pkill
pkill -f plum-agent

# 方法2：找到PID后kill
ps aux | grep plum-agent
kill <PID>
```

### 问题：查看Agent日志
```bash
# 前台运行的Agent：直接在终端查看
# 后台运行的Agent：
tail -f logs/agent-nodeA.log
```

## 💡 高级技巧

### 技巧1：自定义运行多个节点
```bash
# 修改Makefile中的agent-run-multi
# 或者手动启动
make agent
AGENT_NODE_ID=node1 ./agent-go/plum-agent > logs/node1.log 2>&1 &
AGENT_NODE_ID=node2 ./agent-go/plum-agent > logs/node2.log 2>&1 &
AGENT_NODE_ID=node3 ./agent-go/plum-agent > logs/node3.log 2>&1 &
```

### 技巧2：使用systemd管理Agent
```bash
# 创建服务文件: /etc/systemd/system/plum-agent@.service
[Unit]
Description=Plum Agent %i
After=network.target

[Service]
Type=simple
User=plum
Environment="AGENT_NODE_ID=%i"
Environment="CONTROLLER_BASE=http://127.0.0.1:8080"
ExecStart=/opt/plum/agent-go/plum-agent
Restart=always

[Install]
WantedBy=multi-user.target

# 启动多个实例
systemctl start plum-agent@nodeA
systemctl start plum-agent@nodeB
```

## 📚 相关文档

- [agent-go/README.md](../agent-go/README.md) - Go Agent详细说明
- [AGENT_GO_MIGRATION.md](./AGENT_GO_MIGRATION.md) - 迁移文档
- [QUICK_REFERENCE.md](./QUICK_REFERENCE.md) - 快速参考

## 🎯 总结

新的Makefile命令让Agent管理变得简单：
- ✅ 一键构建：`make agent`
- ✅ 一键运行：`make agent-run`
- ✅ 多节点支持：`make agent-runA/B/C`
- ✅ 批量启动：`make agent-run-multi`
- ✅ 内置帮助：`make agent-help`

**推荐工作流**：
```bash
# 1. 首次使用
make agent && make agent-help

# 2. 日常开发
make agent-run  # 单节点测试

# 3. 集成测试
make agent-run-multi  # 多节点测试
```

