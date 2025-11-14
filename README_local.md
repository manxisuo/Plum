
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

##  树莓派24.04

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

### 安装docker：

```bash
sudo usermod -aG docker $USER
newgrp docker
```

### 安装Qt

```bash
sudo apt update
sudo apt install -y qtbase5-dev qtbase5-dev-tools qtchooser \
qtdeclarative5-dev qt5-qmake qt5-qmltooling-plugins
```

### 安装Go

```bash
sudo tar -C /usr/local -xzf tools/go1.24.9.linux-arm64.tar.gz

# 加入.bashrc
export PATH=$PATH:/usr/local/go/bin
```

Go国内源
```bash
echo 'export GOPROXY=https://goproxy.cn,direct' >> ~/.bashrc
echo 'export GOSUMDB=sum.golang.org' >> ~/.bashrc
```

### protoc和protoc-gen-go和protoc-gen-go-grpc

```bash
sudo apt install -y protobuf-compiler
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3
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

### gRPC C++ 开发包

方法1：系统自带的

```bash
sudo apt install -y libgrpc++-dev protobuf-compiler-grpc
```

### protobuf 开发包

方法1：系统自带的（失败）

```bash
sudo apt install -y libprotobuf-dev protobuf-compiler
```

方法2：最新的

```bash
wget https://github.com/protocolbuffers/protobuf/releases/download/v24.0/protoc-24.0-linux-aarch_64.zip
unzip protoc-24.0-linux-aarch_64.zip -d $HOME/protoc
echo 'export PATH=$HOME/protoc/bin:$PATH' >> ~/.bashrc
echo 'export LD_LIBRARY_PATH=$HOME/protoc/lib:$LD_LIBRARY_PATH' >> ~/.bashrc
source ~/.bashrc
```

统一使用新版 Protobuf C++

```bash
git clone -b v24.0 https://github.com/protocolbuffers/protobuf.git
cd protobuf
git submodule update --init --recursive
mkdir build
cd build
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -Dprotobuf_BUILD_TESTS=OFF \
  -DCMAKE_POSITION_INDEPENDENT_CODE=ON
make -j$(nproc)
sudo make install
sudo ldconfig
protoc --version
```