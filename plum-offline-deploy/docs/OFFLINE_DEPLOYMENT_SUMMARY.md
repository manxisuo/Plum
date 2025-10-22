# Plum 离线部署总结

## 部署环境
- **开发环境**: Windows 11 WSL2 (x86_64)
- **目标环境**: 银河麒麟V10 (aarch64/ARM64)
- **部署模式**: 离线部署（无法联网）

## 1. 需要准备的文件清单

### 1.1 源代码和项目文件
```
plum-offline-deploy/
├── source/Plum/                    # 完整项目源码
│   ├── controller/                 # Controller源码
│   │   ├── cmd/                   # 主程序入口
│   │   ├── internal/              # 内部模块
│   │   ├── go.mod                 # Go模块定义
│   │   ├── go.sum                 # 依赖锁定
│   │   └── vendor/                # Go依赖包（离线模式）
│   ├── agent-go/                   # Agent源码
│   │   ├── *.go                   # Go源文件
│   │   ├── go.mod                 # Go模块定义
│   │   ├── go.sum                 # 依赖锁定
│   │   └── vendor/                # Go依赖包（离线模式）
│   ├── ui/                        # Web UI前端
│   │   ├── src/                   # Vue源码
│   │   ├── package.json           # Node.js依赖定义
│   │   ├── package-lock.json      # 依赖锁定
│   │   └── node_modules/          # Node.js依赖包
│   ├── proto/                     # Protocol Buffers定义
│   ├── sdk/                       # C++ SDK（可选）
│   ├── examples/                  # 示例应用
│   ├── Makefile                   # 构建脚本
│   └── README.md                  # 文档
```

### 1.2 ARM64构建工具
```
tools/
├── go1.23.0.linux-arm64.tar.gz    # Go 1.23.0 ARM64版本
├── node-v18.20.4-linux-arm64.tar.xz # Node.js 18.x ARM64版本
└── download-tools.sh              # 工具下载脚本
```

### 1.3 部署脚本
```
scripts/
├── install-deps.sh                # 依赖安装脚本
├── build-all.sh                   # 构建脚本  
├── deploy.sh                      # 部署脚本
└── OFFLINE_DEPLOYMENT_GUIDE.md   # 详细部署指南
```

## 2. 准备步骤（在WSL2中执行）

### 步骤1：准备依赖
```bash
cd /home/stone/code/Plum

# Controller Go依赖
cd controller && go mod vendor && cd ..

# Agent Go依赖
cd agent-go && go mod vendor && cd ..

# UI Node.js依赖
cd ui && npm install && cd ..
```

### 步骤2：下载ARM64工具
```bash
# 下载Go ARM64版本
wget https://golang.google.cn/dl/go1.23.0.linux-arm64.tar.gz -O plum-offline-deploy/tools/

# 下载Node.js ARM64版本
wget https://nodejs.org/dist/v18.20.4/node-v18.20.4-linux-arm64.tar.xz -O plum-offline-deploy/tools/
```

### 步骤3：打包项目
```bash
# 复制完整项目（包含vendor和node_modules）
cp -r . plum-offline-deploy/source/Plum

# 设置脚本权限
chmod +x plum-offline-deploy/scripts/*.sh
```

## 3. 目标环境部署步骤

### 步骤1：安装依赖
```bash
cd plum-offline-deploy/scripts
./install-deps.sh
```

### 步骤2：构建项目
```bash
./build-all.sh
```

### 步骤3：部署服务
```bash
./deploy.sh
```

## 4. 关键配置说明

### 4.1 Go离线构建
- 使用`go mod vendor`下载所有依赖到vendor目录
- 构建时使用`-mod=vendor`参数避免网络请求
- 确保Go版本为1.23.0，支持ARM64架构

### 4.2 Node.js离线构建
- 使用`npm install`下载所有依赖到node_modules目录
- 构建时直接从本地node_modules读取依赖
- 确保Node.js版本为18.x，支持ARM64架构

### 4.3 系统服务配置
- 使用systemd管理Controller和Agent服务
- 配置文件位于`/opt/plum/.env.*`
- 服务用户：`plum`，权限控制安全
- Web UI通过nginx代理到Controller API

## 5. 验证清单

### 构建验证
- [ ] Controller可执行文件：`/opt/plum/bin/controller`
- [ ] Agent可执行文件：`/opt/plum/bin/plum-agent`
- [ ] Web UI静态文件：`/opt/plum/ui/`

### 服务验证
- [ ] Controller服务状态：`systemctl status plum-controller`
- [ ] Agent服务状态：`systemctl status plum-agent`
- [ ] API访问测试：`curl http://localhost:8080/v1/nodes`
- [ ] Web UI访问：浏览器打开 `http://localhost`

### 功能验证
- [ ] DAG工作流创建
- [ ] 任务执行
- [ ] 状态监控
- [ ] 数据库操作

## 6. 常见问题解决

### 架构不匹配
确保所有工具都是ARM64版本：
```bash
file /opt/plum/bin/controller  # 应显示ARM64
file /opt/plum/bin/plum-agent  # 应显示ARM64
```

### 权限问题
```bash
sudo chown -R plum:plum /opt/plum/
sudo chmod +x /opt/plum/bin/*
```

### 依赖缺失
检查vendor和node_modules目录完整性：
```bash
ls -la controller/vendor/
ls -la agent-go/vendor/
ls -la ui/node_modules/
```

## 7. 文件大小预估

| 组件 | 预估大小 | 备注 |
|------|----------|------|
| 源代码 | ~50MB | 包含.git历史 |
| Go依赖(vendor) | ~100MB | Controller + Agent |
| Node.js依赖 | ~500MB | UI dependencies |
| Go工具 | ~150MB | ARM64版本 |
| Node.js工具 | ~50MB | ARM64版本 |
| **总计** | **~850MB** | 适合2GB+存储介质 |

## 8. 传输建议

1. **USB 3.0 U盘**：推荐使用16GB+ U盘
2. **网络传输**：如果网络可通（但需要离线构建）
3. **光盘刻录**：适合一次性部署
4. **压缩打包**：使用tar.gz减少传输时间

---

## 下一步操作

1. 在WSL2中运行准备脚本
2. 下载ARM64构建工具
3. 测试本地构建（验证脚本正确性）
4. 打包传输到目标环境
5. 按照部署指南执行部署
