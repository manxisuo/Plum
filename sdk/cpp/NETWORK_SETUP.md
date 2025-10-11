# C++ SDK网络配置指南

## 🌍 网络环境问题

C++ SDK使用CMake FetchContent从GitHub下载依赖：
- nlohmann/json
- cpp-httplib

如果无法访问GitHub，有以下解决方案。

## ✅ 方案1：使用GitHub镜像（推荐）

### 一键构建
```bash
make sdk_cpp_mirror
```

### 原理
使用[ghproxy.link](https://ghproxy.link)作为GitHub加速代理：
```
原地址: https://github.com/nlohmann/json.git
镜像:   https://ghproxy.link/https://github.com/nlohmann/json.git
```

### 手动配置
```bash
cmake -S sdk/cpp -B sdk/cpp/build -DUSE_GITHUB_MIRROR=ON
cmake --build sdk/cpp/build -j
```

## 🔧 方案2：配置Git全局代理

### 使用ghproxy
```bash
git config --global url."https://ghproxy.link/https://github.com/".insteadOf "https://github.com/"
```

之后正常构建：
```bash
make sdk_cpp
```

### 取消配置
```bash
git config --global --unset url."https://ghproxy.com/https://github.com/".insteadOf
```

## 📦 方案3：手动下载依赖

### 下载并放置
```bash
# 1. 创建目录
mkdir -p sdk/cpp/build/_deps

# 2. 下载nlohmann/json
cd sdk/cpp/build/_deps
git clone https://ghproxy.link/https://github.com/nlohmann/json.git json-src
cd json-src && git checkout v3.11.3 && cd ../..

# 3. 下载cpp-httplib
git clone https://ghproxy.link/https://github.com/yhirose/cpp-httplib.git httplib-src
cd httplib-src && git checkout v0.15.3 && cd ../..

# 4. 返回项目根目录构建
cd /home/stone/code/Plum
make sdk_cpp
```

## 🌐 方案4：使用系统包（不推荐）

某些系统有这些库的包，但版本可能不对：

```bash
# Ubuntu/Debian
sudo apt install nlohmann-json3-dev

# 修改CMakeLists.txt注释掉FetchContent
# 改用find_package(nlohmann_json REQUIRED)
```

## 🎯 推荐方案

### 个人开发
```bash
make sdk_cpp_mirror   # 简单直接
```

### 团队/CI环境
```bash
# 配置一次，全局生效
git config --global url."https://ghproxy.link/https://github.com/".insteadOf "https://github.com/"
make sdk_cpp
```

### 离线环境
手动下载依赖（方案3），打包整个`sdk/cpp/build/_deps/`目录。

## 📝 其他可用镜像

### ghproxy.link（推荐）
```bash
https://ghproxy.link/https://github.com/...
```
- 速度快
- 稳定性好
- 注意：ghproxy.com 会重定向到 ghproxy.link

### gitclone.com
```bash
https://gitclone.com/github.com/...
```

### fastgit.org（已停止服务）
~~不再推荐~~

## 🔍 验证依赖是否下载成功

```bash
ls -la sdk/cpp/build/_deps/
# 应该看到:
# json-src/
# httplib-src/
```

## ⚠️ 注意事项

1. **首次构建慢**：需要下载依赖（header-only库）
2. **后续构建快**：依赖已缓存在build/_deps/
3. **清理构建**：`rm -rf sdk/cpp/build` 会删除依赖缓存
4. **网络要求**：只有首次需要网络，之后可离线构建

---

**提示**：如果使用镜像仍然失败，可以在有网络的机器上构建，然后打包`sdk/cpp/build/_deps/`目录。

