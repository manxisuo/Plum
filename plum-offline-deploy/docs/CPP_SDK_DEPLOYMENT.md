# C++ SDK 离线部署指南

## 概述

本文档详细说明了如何在银河麒麟V10 ARM64环境中离线部署Plum Client C++ SDK。

## 前置条件

### 系统要求
- 银河麒麟V10 ARM64系统
- 已安装基础开发工具

### 依赖检查
在开始部署前，请运行依赖检查脚本：

```bash
cd plum-offline-deploy/scripts
./check-cpp-deps.sh
```

### 必需依赖
- **CMake**: 3.16或更高版本
- **g++**: 支持C++17的编译器
- **httplib**: HTTP客户端库 (header-only，项目内置)
- **pthread**: POSIX线程库
- **pkg-config**: 包配置工具

### 安装依赖
如果缺少依赖，请运行：

```bash
sudo apt-get update
sudo apt-get install cmake g++ libpthread-stubs0-dev pkg-config
```

## 部署步骤

### 1. 检查依赖
```bash
cd plum-offline-deploy/scripts
./check-cpp-deps.sh
```

### 2. 构建C++ SDK
```bash
# 方法1: 使用专门的C++ SDK构建脚本
./build-cpp-sdk.sh

# 方法2: 使用完整构建脚本（包含所有组件）
./build-all.sh
```

### 3. 部署C++ SDK
```bash
# 部署到项目目录
./deploy.sh

# 或者安装到系统目录（可选）
sudo ./install-cpp-sdk.sh
```

## 构建结果

### 库文件
- **Plum Client库**: `sdk/cpp/build/plumclient/libplumclient.so`
- **头文件**: `sdk/cpp/plumclient/include/`
- **示例程序**: `sdk/cpp/build/examples/service_client_example/service_client_example`

### 其他C++示例
- **Echo Worker**: `sdk/cpp/build/examples/echo_worker/echo_worker`
- **Radar Sensor**: `sdk/cpp/build/examples/radar_sensor/radar_sensor`
- **gRPC Echo Worker**: `sdk/cpp/build/examples/grpc_echo_worker/grpc_echo_worker`

## 使用方法

### 1. 基本使用
```cpp
#include <plum_client.hpp>

int main() {
    plumclient::PlumClient client("http://localhost:8080");
    client.start();
    
    // 使用客户端...
    
    client.stop();
    return 0;
}
```

### 2. 编译命令
```bash
# 使用pkg-config（如果安装了系统SDK）
g++ -std=c++17 -o myapp myapp.cpp $(pkg-config --cflags --libs plumclient)

# 手动指定路径
g++ -std=c++17 -o myapp myapp.cpp \
    -I/opt/plum/sdk/include \
    -L/opt/plum/sdk/lib \
    -lplumclient -lcurl -lpthread
```

### 3. 运行示例
```bash
# 运行Service Client示例
/opt/plum/sdk/examples/service_client_example

# 运行其他示例
/opt/plum/sdk/examples/echo_worker
/opt/plum/sdk/examples/radar_sensor
/opt/plum/sdk/examples/grpc_echo_worker
```

## 部署目录结构

### 项目部署目录
```
/opt/plum/
├── bin/                    # 可执行文件
│   ├── controller
│   └── plum-agent
├── sdk/                    # C++ SDK
│   ├── lib/
│   │   └── libplumclient.so
│   ├── include/
│   │   └── plum_client.hpp
│   └── examples/
│       ├── service_client_example
│       ├── echo_worker
│       ├── radar_sensor
│       └── grpc_echo_worker
├── ui/                     # Web UI
├── data/                   # 数据目录
└── logs/                   # 日志目录
```

### 系统安装目录（可选）
```
/usr/local/
├── lib/
│   └── libplumclient.so
├── include/plumclient/
│   └── plum_client.hpp
└── lib/pkgconfig/
    └── plumclient.pc
```

## 功能特性

### 1. 服务发现和调用
- 服务注册、心跳、注销
- 服务发现和随机选择
- 服务调用和负载均衡

### 2. 弱网环境支持
- 智能缓存系统
- 自适应重试策略
- 网络质量监控
- 限流和熔断保护

### 3. 配置管理
- 灵活的配置选项
- 运行时配置更新
- 环境变量支持

## 故障排除

### 1. 构建失败
**问题**: CMake配置失败
**解决**: 检查CMake版本和依赖库

**问题**: 编译错误
**解决**: 检查g++版本和C++17支持

**问题**: 链接错误
**解决**: 检查httplib和pthread库

### 2. 运行时错误
**问题**: 库文件未找到
**解决**: 检查LD_LIBRARY_PATH或使用绝对路径

**问题**: 头文件未找到
**解决**: 检查包含路径设置

**问题**: 网络连接失败
**解决**: 检查Controller是否运行

### 3. 性能问题
**问题**: 编译速度慢
**解决**: 使用多核编译 `make -j$(nproc)`

**问题**: 运行时性能差
**解决**: 启用弱网环境支持，调整配置

## 最佳实践

### 1. 开发环境
- 使用IDE支持CMake项目
- 配置代码补全和调试
- 设置断点和单步调试

### 2. 生产环境
- 启用弱网环境支持
- 配置适当的超时和重试
- 监控网络质量和性能

### 3. 集成测试
- 使用示例程序验证功能
- 测试弱网环境下的表现
- 验证错误处理和恢复

## 更新和维护

### 1. 更新SDK
```bash
# 重新构建
./build-cpp-sdk.sh

# 重新部署
./deploy.sh
```

### 2. 清理构建
```bash
# 清理构建目录
rm -rf sdk/cpp/build

# 重新构建
./build-cpp-sdk.sh
```

### 3. 日志查看
```bash
# 查看构建日志
./build-cpp-sdk.sh 2>&1 | tee build.log

# 查看运行日志
journalctl -u plum-controller -f
```

## 支持信息

### 1. 文档资源
- `sdk/cpp/plumclient/README.md` - SDK使用文档
- `docs/PLUMCLIENT_SUMMARY.md` - 功能总结
- `docs/WEAK_NETWORK_SUPPORT.md` - 弱网环境支持

### 2. 示例程序
- `sdk/cpp/examples/service_client_example/` - 基本使用示例
- `tools/cpp_weak_network_test.cpp` - 弱网环境测试

### 3. 测试工具
- `tools/test_plumclient.sh` - 功能测试脚本
- `tools/run_cpp_weak_network_test.sh` - 弱网环境测试

## 总结

Plum Client C++ SDK提供了完整的服务发现、调用和弱网环境支持功能。通过离线部署包，可以在银河麒麟V10 ARM64环境中轻松部署和使用该SDK。

主要优势：
- **完全离线**: 无需网络连接即可部署
- **功能完整**: 支持所有核心功能
- **性能优化**: 支持弱网环境下的性能优化
- **易于使用**: 提供简洁的API和丰富的文档
- **稳定可靠**: 经过充分测试和验证
