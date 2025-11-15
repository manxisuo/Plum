
## 压缩复制离线包

```bash
./plum-offline-deploy/scripts-prepare/prepare-offline-deploy.sh
tar -cvf plum-offline-deploy-1114.tar ./plum-offline-deploy
scp plum-offline-deploy-1114.tar jari@192.168.1.192:/data/usershare
scp plum-offline-deploy-1114.tar pi@192.168.1.2:~/plum
```

## FSL_MainControl

```bash
find ./examples-local -name build -exec rm -rf {} \;
make examples_FSL_All
make examples_FSL_All_Pkg
```

## 树莓派24.04配置

### 网络配置

sudo vi /etc/netplan/50-cloud-init.yaml

```yml
[sudo] password for pi:
network:
  version: 2
  renderer: networkd
  ethernets:
    eth0:
      dhcp4: false
      addresses:
        - 192.168.1.2/24
      nameservers:
        addresses:
          - 8.8.8.8
  wifis:
    wlan0:
      dhcp4: true
      optional: true
      access-points:
        "ZhuangZhuang":
          password: "13912159154"
```

### 软件源

sudo mv /etc/apt/sources.list.d/ubuntu.sources /etc/apt/sources.list.d/ubuntu.sources.bak
sudo vi /etc/apt/sources.list.d/ubuntu-official.sources

```ini
# Ubuntu 24.04 LTS ARM64 official ports repository
Types: deb
URIs: http://ports.ubuntu.com/ubuntu-ports/
Suites: noble
Components: main restricted universe multiverse
Architectures: arm64
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg

Types: deb
URIs: http://ports.ubuntu.com/ubuntu-ports/
Suites: noble-updates
Components: main restricted universe multiverse
Architectures: arm64
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg

Types: deb
URIs: http://ports.ubuntu.com/ubuntu-ports/
Suites: noble-security
Components: main restricted universe multiverse
Architectures: arm64
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg

Types: deb
URIs: http://ports.ubuntu.com/ubuntu-ports/
Suites: noble-backports
Components: main restricted universe multiverse
Architectures: arm64
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg
```

### 安装docker：

```bash
sudo apt install -y ca-certificates curl gnupg lsb-release

# 添加 Docker 官方 GPG Key
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# 添加 Docker 仓库
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# 安装 Docker Engine
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 启动并验证
sudo systemctl enable docker
sudo systemctl start docker
sudo systemctl status docker

# 如果你不想每次都用 sudo：
sudo usermod -aG docker $USER

# 然后退出当前 shell 再重新登录，或者执行：
newgrp docker
```

### 安装Qt

```bash
sudo apt update
sudo apt install -y qtbase5-dev qtbase5-dev-tools qtchooser qtdeclarative5-dev qt5-qmake qt5-qmltooling-plugins
```

### 安装Go

```bash
sudo tar -C /usr/local -xzf tools/go1.24.9.linux-arm64.tar.gz

# 加入.bashrc
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
```

Go国内源
```bash
echo 'export GOPROXY=https://goproxy.cn,direct' >> ~/.bashrc
echo 'export GOSUMDB=sum.golang.org' >> ~/.bashrc
```

### 安装Node

```bash
curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
sudo apt install -y nodejs
```

### make/gcc/g++等

```bash
sudo apt install -y build-essential
```

### cmake等

```bash
sudo apt install -y cmake g++ make git pkg-config
```

```bash
sudo apt install net-tools
```

## protoc / protoc-gen-go / protoc-gen-go-grpc / gRPC C++开发包 / Protobuf C++开发包

### 1️⃣ 更新系统 & 安装基础工具

```bash
sudo apt update
sudo apt upgrade -y
sudo apt install -y build-essential git curl wget pkg-config autoconf automake libtool
```

### 2️⃣ 安装 Protobuf C++ 开发包和 protoc

Ubuntu 官方仓库提供：

```bash
sudo apt install -y protobuf-compiler libprotobuf-dev
```

验证安装：

```bash
protoc --version
# 可能输出类似 libprotoc 3.x
```

这一步安装了：

* `protoc` 编译器
* C++ 库和头文件（libprotobuf-dev）

---

### 3️⃣ 安装 gRPC C++ 开发包

Ubuntu 24.04 提供 gRPC C++ 库：

```bash
sudo apt install -y libgrpc++-dev
```

> 如果需要 gRPC C++ 的工具链（比如 `grpc_cpp_plugin`），通常随 libgrpc++-dev 一起提供。

验证：

```bash
which grpc_cpp_plugin
# 输出路径说明已安装
```

### 4️⃣ 安装 Go 和 Go 插件

<!-- Ubuntu 自带 Go 版本可能偏旧，建议先安装较新 Go：

```bash
sudo apt install -y golang
go version
# 确认版本，如果太低可以用官方二进制安装最新 Go
``` -->

然后安装 protoc 的 Go 插件：

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

确保 Go bin 目录在 PATH 中：

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

验证：

```bash
protoc-gen-go --version
protoc-gen-go-grpc --version
```

⚠️ 注意

1. **版本偏旧**：Ubuntu apt 仓库的 `protoc`、`libprotobuf-dev` 版本可能比官方最新的低，可能不支持最新 proto 特性。
2. **Go 插件是最新的**：`protoc-gen-go` 和 `protoc-gen-go-grpc` 通过 `go install` 获取最新版本，可与较旧的 protoc 兼容。
3. **ARM64**：Ubuntu 官方仓库提供的包都支持 ARM64，无需特殊处理。

### 编译 grpc_cpp_plugin

```bash
git clone --branch v1.51.1 --depth 1 https://github.com/grpc/grpc
cd grpc
git submodule update --init --recursive --depth 1

mkdir build && cd build
cmake -GNinja -DCMAKE_BUILD_TYPE=Release -DgRPC_BUILD_TESTS=OFF ..
ninja grpc_cpp_plugin
sudo cp grpc_cpp_plugin /usr/local/bin/
```

## Plum

```bash
make proto
make controller
make agent
make sdk_cpp_offline
make ui-update
make ui-dev
make ui-build

#docker pull --platform linux/arm64 ubuntu:22.04
docker load < ubuntu-22.04-arm64.tar
```

