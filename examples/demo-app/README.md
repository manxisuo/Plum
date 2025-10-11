# Plum Demo应用

一个简单的C++示例应用，演示如何被Plum部署和管理。

## 📋 功能

- ✅ 读取Plum注入的环境变量
- ✅ 每10秒输出一次心跳日志
- ✅ 优雅处理SIGTERM信号
- ✅ 显示运行时间和状态
- ✅ 完整的构建和打包流程

## 🔨 构建

```bash
cd examples/demo-app

# 方式1：使用构建脚本（推荐）
./build.sh
# 生成: demo-app.zip

# 方式2：手动构建
mkdir -p build && cd build
cmake ..
make
cd ..
```

## 📦 打包

```bash
# 已包含在build.sh中
./build.sh

# 手动打包
mkdir -p package
cp build/demo-app package/
cp start.sh package/
cp meta.ini package/
cd package && zip -r ../demo-app.zip . && cd ..
```

## 🚀 部署到Plum

### 方法1：通过Web UI

1. **上传应用包**
   - 访问 http://your-plum-server/apps
   - 点击"上传应用包"
   - 选择 `demo-app.zip`

2. **创建部署**
   - 访问 http://your-plum-server/deployments
   - 点击"创建部署"
   - 选择刚上传的demo-app
   - 选择节点和副本数
   - 点击"创建"

3. **启动部署**
   - 在部署列表中找到刚创建的部署
   - 点击"启动"按钮
   - 实例会在对应节点上自动启动

### 方法2：通过API

```bash
# 1. 上传应用包
curl -X POST http://localhost:8080/v1/apps/upload \
  -F "file=@demo-app.zip"
# 返回: {"artifactId":"xxx","url":"/artifacts/demo-app_xxx.zip"}

# 2. 创建部署
curl -X POST http://localhost:8080/v1/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "name": "demo-deployment",
    "entries": [{
      "artifactUrl": "/artifacts/demo-app_xxx.zip",
      "replicas": {"nodeA": 1}
    }]
  }'
# 返回: {"deploymentId":"yyy"}

# 3. 启动部署
curl -X POST "http://localhost:8080/v1/deployments/yyy?action=start"
```

## 📊 查看运行状态

### Web UI
- 访问"分配"页面查看实例状态
- 查看实例日志（如果Agent配置了日志重定向）

### 命令行
```bash
# 查看部署状态
curl http://localhost:8080/v1/deployments

# 查看实例分配
curl http://localhost:8080/v1/assignments?nodeId=nodeA

# 在Agent节点上查看进程
ps aux | grep demo-app
```

## 🔧 环境变量

应用启动时，Agent会注入以下环境变量：

| 变量 | 示例值 | 说明 |
|------|--------|------|
| PLUM_INSTANCE_ID | abc123-def456 | 实例ID |
| PLUM_APP_NAME | demo-app | 应用名称 |
| PLUM_APP_VERSION | 1.0.0 | 应用版本 |

## 📝 文件说明

```
demo-app/
├── main.cpp              # 主程序源码
├── CMakeLists.txt        # CMake构建配置
├── start.sh              # 启动脚本（必须，Plum调用）
├── meta.ini              # 元数据（必须，包含name和version）
├── build.sh              # 构建和打包脚本
└── README.md             # 本文档
```

### meta.ini格式
```ini
# 必须字段
name=demo-app          # 应用名称
version=1.0.0          # 应用版本

# 可选字段（服务发现）
service=my-api:http:8080
```

## 🎯 预期行为

启动后应该看到：
```
========================================
  Plum Demo Application
========================================
App Name:    demo-app
App Version: 1.0.0
Instance ID: abc123-def456
PID:         12345
========================================

[1] Uptime: 0s | Time: Sat Oct 11 12:00:00 2025
[2] Uptime: 10s | Time: Sat Oct 11 12:00:10 2025
[3] Uptime: 20s | Time: Sat Oct 11 12:00:20 2025
...
```

## 🧪 测试场景

### 1. 正常启动和停止
```bash
# Web UI点击"启动" → 查看实例状态变为Running
# Web UI点击"停止" → 查看实例状态变为Stopped
```

### 2. 进程死亡自动重启
```bash
# 在Agent节点kill进程
kill -9 <demo-app-pid>

# 等待5秒，Agent会自动重启
# 状态: Running → Failed → Running
```

### 3. 多节点部署
```bash
# 创建部署时配置多个节点
# "replicas": {"nodeA": 2, "nodeB": 1}
# 会在nodeA启动2个实例，nodeB启动1个实例
```

## 🔄 修改和重新部署

```bash
# 1. 修改代码
vim main.cpp

# 2. 重新构建打包
./build.sh

# 3. 上传新版本到Plum
# 在UI中上传新的demo-app.zip

# 4. 创建新的部署或更新现有部署
```

## 🎓 扩展示例

基于此demo，你可以：
- 添加HTTP服务器（使用cpp-httplib）
- 集成Worker SDK注册任务
- 集成Resource SDK上报设备状态
- 添加数据库连接
- 实现业务逻辑

---

**提示**：这是最简单的demo，用于理解Plum的部署流程。实际应用可以更复杂。

