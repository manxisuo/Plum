# Go相关工具完整需求清单

## 📋 你已经准备的文件
✅ `go1.23.12.linux-arm64.tar.gz` - Go ARM64版本

## 🔴 还需要准备的Go工具

### 问题分析
你的项目使用了protobuf，需要以下Go工具：
1. `protoc-gen-go` - protobuf Go代码生成器
2. `protoc-gen-go-grpc` - gRPC Go代码生成器

这些工具是**架构相关的二进制文件**，不能直接复用x86版本的。

### 当前工具状态检查
```bash
# 你的WSL2环境中的工具（x86_64）
$GOPATH/bin/protoc-gen-go     # ELF 64-bit x86-64
$GOPATH/bin/protoc-gen-go-grpc # ELF 64-bit x86-64
```

### 解决方案

#### 方案1：在WSL2中准备ARM64版本的工具（推荐）

在WSL2环境中执行以下步骤：

```bash
# 1. 安装Go ARM64版本（临时）
wget https://go.dev/dl/go1.23.12.linux-arm64.tar.gz
sudo tar -C /tmp -xzf go1.23.12.linux-arm64.tar.gz
export PATH="/tmp/go/bin:$PATH"

# 2. 设置Go环境
export GOOS=linux
export GOARCH=arm64
export GOBIN=/tmp/go-arm64-tools/bin
mkdir -p $GOBIN

# 3. 交叉编译ARM64版本的工具
GOOS=linux GOARCH=arm64 go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
GOOS=linux GOARCH=arm64 go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

然后将这些ARM64版本的工具打包。

#### 方案2：在目标环境联网安装（如果允许临时联网）

修改`install-deps.sh`脚本，在安装Go后添加：

```bash
# 安装Go protobuf工具
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
mkdir -p $GOPATH/bin

# 这些命令需要联网
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

#### 方案3：预先构建并包含在离线包中（最佳）

在WSL2环境中做准备：

```bash
# 创建ARM64工具目录
mkdir -p plum-offline-deploy/tools/go-arm64-tools/bin

# 使用你已有的ARM64 Go编译工具
cd /tmp
tar -xzf go1.23.12.linux-arm64.tar.gz
export PATH="/tmp/go/bin:$PATH"

# 交叉编译ARM64版本
GOOS=linux GOARCH=arm64 go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
GOOS=linux GOARCH=arm64 go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 复制到部署包
cp /tmp/go-arm64-tools/bin/* plum-offline-deploy/tools/go-arm64-tools/bin/
```

### 最终文件结构

你需要准备的Go相关文件：
```
tools/
├── go1.23.12.linux-arm64.tar.gz     # ✅ 你已有
├── go-arm64-tools/                  # 新增
│   └── bin/
│       ├── protoc-gen-go            # ARM64版本
│       └── protoc-gen-go-grpc       # ARM64版本
└── install-go-tools.sh              # 安装脚本
```

### 验证命令

在目标环境验证：
```bash
go version                           # go1.23.12 linux/arm64
protoc-gen-go --version             # protoc-gen-go v1.x.x
protoc-gen-go-grpc --version        # protoc-gen-go-grpc v1.x.x
```

## 🎯 总结

**你需要额外准备的Go工具**：
1. `protoc-gen-go` 的ARM64版本
2. `protoc-gen-go-grpc` 的ARM64版本

**推荐方案**：在WSL2中使用交叉编译准备ARM64版本的工具，这样可以实现完全离线部署。
