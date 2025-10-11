# Systemd服务部署指南

## 📋 服务文件说明

- `plum-controller.service` - Controller服务
- `plum-agent@.service` - Agent服务模板（支持多实例）

## 🚀 安装步骤

### 1. 准备工作

```bash
# 确保已构建
cd /home/manxisuo/Plum
make controller
make agent

# 创建数据目录
mkdir -p /home/manxisuo/Plum/data
mkdir -p /home/manxisuo/Plum/data/agents
```

### 2. 安装Controller服务

```bash
# 复制service文件
sudo cp deploy/systemd/plum-controller.service /etc/systemd/system/

# 重载systemd
sudo systemctl daemon-reload

# 启动Controller
sudo systemctl start plum-controller

# 开机自启
sudo systemctl enable plum-controller

# 查看状态
sudo systemctl status plum-controller
```

### 3. 安装Agent服务

```bash
# 复制service文件
sudo cp deploy/systemd/plum-agent@.service /etc/systemd/system/

# 重载systemd
sudo systemctl daemon-reload

# 启动多个Agent实例
sudo systemctl start plum-agent@nodeA
sudo systemctl start plum-agent@nodeB
sudo systemctl start plum-agent@nodeC

# 开机自启
sudo systemctl enable plum-agent@nodeA
sudo systemctl enable plum-agent@nodeB
sudo systemctl enable plum-agent@nodeC

# 查看状态
sudo systemctl status plum-agent@nodeA
```

## 🔧 常用命令

### Controller管理
```bash
# 启动/停止/重启
sudo systemctl start plum-controller
sudo systemctl stop plum-controller
sudo systemctl restart plum-controller

# 查看状态
sudo systemctl status plum-controller

# 查看日志
sudo journalctl -u plum-controller -f        # 实时日志
sudo journalctl -u plum-controller -n 100    # 最近100行
sudo journalctl -u plum-controller --since today  # 今天的日志
```

### Agent管理
```bash
# 启动特定节点
sudo systemctl start plum-agent@nodeA

# 停止特定节点
sudo systemctl stop plum-agent@nodeB

# 查看所有Agent状态
sudo systemctl status 'plum-agent@*'

# 查看特定Agent日志
sudo journalctl -u plum-agent@nodeA -f
```

### 批量管理Agent
```bash
# 启动所有Agent
sudo systemctl start plum-agent@nodeA plum-agent@nodeB plum-agent@nodeC

# 停止所有Agent
sudo systemctl stop 'plum-agent@*'

# 重启所有Agent
sudo systemctl restart 'plum-agent@*'
```

## ⚙️ 自定义配置

### 修改环境变量

```bash
# 编辑service文件
sudo systemctl edit plum-controller --full

# 修改Environment部分
Environment="CONTROLLER_ADDR=:9090"
Environment="CONTROLLER_DB=/var/lib/plum/plum.db"

# 重载并重启
sudo systemctl daemon-reload
sudo systemctl restart plum-controller
```

### 修改运行用户

```bash
sudo systemctl edit plum-controller --full

# 修改User和Group
User=plum
Group=plum

# 重载并重启
sudo systemctl daemon-reload
sudo systemctl restart plum-controller
```

## 📊 查看所有Plum服务

```bash
# 列出所有plum相关服务
systemctl list-units 'plum-*'

# 查看服务树
systemctl status plum-controller plum-agent@nodeA
```

## 🔍 故障排查

### Controller无法启动
```bash
# 查看详细错误
sudo journalctl -u plum-controller -n 50 --no-pager

# 检查可执行文件
ls -la /home/manxisuo/Plum/controller/bin/controller

# 手动测试启动
cd /home/manxisuo/Plum
./controller/bin/controller
```

### Agent无法连接Controller
```bash
# 查看Agent日志
sudo journalctl -u plum-agent@nodeA -n 50

# 检查Controller是否运行
sudo systemctl status plum-controller

# 测试连接
curl http://127.0.0.1:8080/healthz
```

## 🔄 更新程序

```bash
# 1. 重新构建
cd /home/manxisuo/Plum
git pull
make controller
make agent

# 2. 重启服务
sudo systemctl restart plum-controller
sudo systemctl restart 'plum-agent@*'

# 3. 验证
sudo systemctl status plum-controller
```

## 📝 服务依赖关系

```
plum-controller.service
    ↑ (Requires)
plum-agent@nodeA.service
plum-agent@nodeB.service
...
```

Agent依赖Controller，Controller先启动。

## 🎯 开机自启验证

```bash
# 检查是否已启用
sudo systemctl is-enabled plum-controller
sudo systemctl is-enabled plum-agent@nodeA

# 模拟重启测试
sudo systemctl reboot  # 慎用！
# 重启后检查服务
sudo systemctl status plum-controller plum-agent@nodeA
```

## 📊 监控服务

### 查看资源占用
```bash
# 实时监控
sudo systemctl status plum-controller
sudo systemctl status plum-agent@nodeA

# 详细信息
systemd-cgtop  # 类似top，显示systemd服务资源占用
```

### 设置资源限制
```bash
# 编辑service
sudo systemctl edit plum-controller --full

# 添加限制
[Service]
MemoryLimit=512M
CPUQuota=50%
```

---

**提示**：生产环境UI用nginx serve静态文件（ui/dist/），不需要单独的UI service。

