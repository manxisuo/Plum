# Plum性能测试指南

## 测试目标

验证Plum系统是否满足以下性能需求：

1. ✅ 支持50个节点并发访问
2. ✅ 平均服务故障恢复时间不大于2秒
3. ✅ 响应延迟<500ms
4. ✅ 系统稳定性>95%

## 测试环境要求

### 硬件要求
- CPU: 4核及以上
- 内存: 8GB及以上
- 网络: 1Gbps网卡

### 软件要求
- Go 1.21+
- Linux/MacOS/Windows
- Plum Controller运行中

## 快速开始

### 1. 启动Controller

```bash
make controller-run
```

### 2. 运行性能测试

```bash
./tools/run_performance_test.sh
```

测试将持续5分钟，测试50个并发节点。

## 测试指标

### 1. 并发能力测试

**测试内容：** 50个节点同时发送心跳请求

**评价标准：**
- 优秀：≥45个节点成功 (90%)
- 良好：≥40个节点成功 (80%)
- 需要优化：<40个节点成功

### 2. 响应延迟测试

**测试内容：** 心跳请求的平均响应时间

**评价标准：**
- 优秀：<100ms
- 良好：<500ms
- 需要优化：≥500ms

### 3. 系统稳定性测试

**测试内容：** 请求成功率

**评价标准：**
- 优秀：>95%
- 良好：>90%
- 需要优化：≤90%

## 测试结果示例

```
=== 性能测试结果分析 ===
测试节点数: 50
成功节点数: 48
总成功请求: 14400
总错误请求: 120
成功率: 99.17%
平均延迟: 45ms
最大延迟: 230ms
最小延迟: 12ms

延迟分布:
  <100ms: 47个节点
  100-500ms: 1个节点
  500ms-1s: 0个节点
  1-2s: 0个节点
  >2s: 0个节点

性能评估:
✅ 节点并发能力: 优秀
✅ 响应延迟: 优秀
✅ 系统稳定性: 优秀
```

## 故障恢复时间测试

### 测试步骤

1. 启动Controller和2个Agent
2. 部署多副本应用（3个副本）
3. 模拟节点故障（kill agent进程）
4. 观察故障恢复时间

### 测试命令

```bash
# 终端1: 启动Controller
make controller-run

# 终端2: 启动Agent A
make agent-runA

# 终端3: 启动Agent B
make agent-runB

# 终端4: 部署应用
curl -X POST http://localhost:8080/v1/apps/upload -F "file=@myapp.zip"
curl -X POST http://localhost:8080/v1/deployments \
  -H "Content-Type: application/json" \
  -d '{"appId":"myapp-id","replicas":3}'

# 模拟故障
kill -9 <agent-pid>

# 观察日志，查看故障恢复时间
```

### 预期结果

- 故障检测时间：<1秒
- 实例迁移时间：<1秒
- 总恢复时间：<2秒

示例日志：
```
2025/10/22 10:25:23 节点 nodeA 离线检测
2025/10/22 10:25:24 开始迁移实例到健康节点
2025/10/22 10:25:24 性能监控: 实例迁移耗时 1.2秒
```

## 弱网环境测试

### 模拟弱网环境

使用tc命令模拟网络延迟和丢包：

```bash
# 添加100ms延迟和5%丢包率
sudo tc qdisc add dev eth0 root netem delay 100ms loss 5%

# 查看设置
sudo tc qdisc show dev eth0

# 删除设置
sudo tc qdisc del dev eth0 root
```

### 测试步骤

1. 设置网络延迟
2. 运行性能测试
3. 观察系统表现

### 预期结果

- 心跳重试机制正常工作
- 服务发现缓存生效
- 系统保持稳定运行

## 负载测试

### 压力测试工具

使用Go SDK编写压力测试：

```go
package main

import (
    "sync"
    "time"
    "github.com/manxisuo/plum/sdk/go"
)

func main() {
    client := plum.NewPlumClient("http://localhost:8080")
    
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ { // 100个并发请求
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < 1000; j++ { // 每个发送1000次请求
                client.DiscoverServiceRandom("my-service", "", "")
                time.Sleep(10 * time.Millisecond)
            }
        }()
    }
    wg.Wait()
}
```

## 性能优化建议

### 1. 数据库优化

如果并发量大，考虑：
- 使用PostgreSQL/MySQL替代SQLite
- 添加数据库连接池
- 优化索引

### 2. 缓存优化

- 启用服务发现缓存（SDK已支持）
- 使用Redis缓存热点数据
- 实现本地内存缓存

### 3. 网络优化

- 启用HTTP/2
- 使用连接池
- 实现请求批量处理

### 4. 配置优化

优化环境变量配置：
```bash
# 快速故障检测
HEARTBEAT_TTL_SEC=3
FAILOVER_INTERVAL_SEC=1
AGENT_HEARTBEAT_INTERVAL_SEC=1
AGENT_PROCESS_CHECK_INTERVAL_SEC=1

# 并发优化
GOMAXPROCS=8  # 设置为CPU核心数
```

## 监控和排查

### 查看性能指标

```bash
# Controller日志
tail -f controller/controller.log | grep "性能监控"

# Agent日志
tail -f agent-go/agent.log | grep "性能"

# 系统资源
top -p $(pgrep controller)
```

### 常见问题

**Q: 并发节点数少于50个成功**
A: 检查系统资源、数据库性能、网络连接数限制

**Q: 响应延迟过高**
A: 检查数据库查询、网络延迟、CPU使用率

**Q: 故障恢复时间超过2秒**
A: 优化心跳间隔、检查网络延迟、优化应用启动时间

## 持续监控

建议部署监控系统：
- Prometheus + Grafana（指标监控）
- ELK Stack（日志分析）
- Jaeger（链路追踪）

## 总结

通过以上测试，可以验证Plum系统是否满足：
- ✅ 50个节点并发访问
- ✅ 2秒内故障恢复
- ✅ 低延迟响应
- ✅ 高稳定性

定期进行性能测试，确保系统持续满足性能要求。

