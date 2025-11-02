# Qt应用在容器中运行指南

## 问题说明

Qt应用（特别是Qt5/Qt6 UI程序）在容器中运行会遇到以下问题：

1. **缺少Qt运行时库**：alpine等精简镜像不包含Qt库
2. **缺少显示服务器**：Qt GUI程序需要X11或Wayland
3. **缺少图形驱动**：需要OpenGL等图形库

## 解决方案

### 方案1：使用包含Qt的基础镜像

使用Ubuntu或其他包含Qt的镜像：

```bash
# 在 agent-go/.env 中设置
PLUM_BASE_IMAGE=ubuntu:22.04
```

然后在Ubuntu镜像中安装Qt库（如果应用需要）：

```dockerfile
# 如果应用需要Qt运行时，可以在应用打包时包含Qt库
# 或者使用包含Qt的镜像（需要自己构建）
```

**优点**：简单，镜像包含基础库  
**缺点**：镜像较大，可能缺少Qt库  
**适用场景**：需要基础系统库的应用

---

### 方案2：在应用中包含 Qt 库（推荐，自包含）

在应用的artifact中包含Qt库：

```
app/
├── HeyUI           # 应用可执行文件
├── start.sh        # 启动脚本
├── lib/            # Qt库目录（重要！）
│   ├── libQt5Core.so.5
│   ├── libQt5Widgets.so.5
│   ├── libQt5Gui.so.5
│   └── ...
└── meta.ini
```

Agent会自动检测 `lib/` 目录并设置 `LD_LIBRARY_PATH=/app/lib`。

**打包示例**：

```bash
# 复制Qt库到lib目录
mkdir -p app/lib
cp /path/to/qt/lib/*.so* app/lib/

# 打包
cd app && zip -r app.zip .
```

**优点**：
- ✅ 完全自包含，不依赖宿主机
- ✅ 每个应用可以使用不同版本的库
- ✅ 便于分发和部署

**缺点**：
- ⚠️ 多个应用需要相同库时会重复存储
- ⚠️ 增加artifact大小

**适用场景**：单个应用或需要特定版本库

---

### 方案3：共享宿主机库路径（推荐，避免重复存储）⭐

**这是你提出的方案**，非常适合多个应用共享相同库的场景。

通过路径映射将宿主机的库路径挂载到容器内，多个应用可以共享相同的系统库。

#### 配置示例

```bash
# agent-go/.env
AGENT_RUN_MODE=docker
PLUM_BASE_IMAGE=ubuntu:22.04

# 挂载宿主机的库路径到容器
PLUM_HOST_LIB_PATHS=/usr/lib,/usr/local/lib,/opt/qt/lib
```

#### 工作原理

1. Agent检测到 `PLUM_HOST_LIB_PATHS` 环境变量
2. 将指定的宿主机路径**只读挂载**到容器的相同路径
3. 容器内的应用可以直接使用宿主机的库（如Qt库）

#### 使用场景示例

**场景**：宿主机安装了Qt库，多个Qt应用需要共享这些库

```bash
# 1. 宿主机有Qt库安装在 /opt/qt/lib
ls /opt/qt/lib/
# libQt5Core.so.5
# libQt5Widgets.so.5
# libQt5Gui.so.5

# 2. 配置Agent挂载此路径
# agent-go/.env
PLUM_HOST_LIB_PATHS=/opt/qt/lib

# 3. 容器内的应用可以直接使用
# 不需要在每个应用的artifact中包含Qt库
# 多个应用共享同一套库
```

**场景**：共享系统标准库

```bash
# 挂载多个库路径
PLUM_HOST_LIB_PATHS=/usr/lib,/usr/local/lib

# 容器内的应用可以使用宿主机的系统库
# 无需在每个应用中包含这些标准库
```

#### 优点

- ✅ **避免库的重复存储**：多个应用共享同一套库
- ✅ **减少artifact大小**：应用artifact不包含库文件
- ✅ **统一管理**：库更新只需在宿主机更新一次
- ✅ **节省磁盘空间**：特别适合多个应用共享相同库的场景

#### 缺点

- ⚠️ **依赖宿主机环境**：库必须存在于宿主机
- ⚠️ **架构兼容性要求**：宿主机和容器必须使用相同的CPU架构（都是x86_64或都是ARM64）
- ⚠️ **版本限制**：所有应用必须使用相同版本的库

#### 注意事项

1. **架构兼容性**（重要！）
   - 宿主机和容器必须使用相同的CPU架构
   - 都是 x86_64，或都是 ARM64
   - 否则库无法加载

2. **库版本兼容**
   - 确保应用与宿主机库版本兼容
   - 所有共享此库的应用必须兼容同一版本

3. **只读挂载**
   - 库路径是只读挂载，容器无法修改
   - 保护宿主机库不被意外修改

4. **路径检查**
   - Agent会自动检查路径是否存在
   - 路径不存在时会跳过并记录警告日志

5. **路径规范**
   - 使用绝对路径
   - 多个路径用逗号分隔
   - 路径会自动去除尾随斜杠

#### 完整配置示例

```bash
# agent-go/.env
AGENT_RUN_MODE=docker
PLUM_BASE_IMAGE=ubuntu:22.04

# 共享宿主机Qt库（多个Qt应用可以共享）
PLUM_HOST_LIB_PATHS=/opt/qt/lib

# 如果还需要其他库路径
PLUM_HOST_LIB_PATHS=/opt/qt/lib,/usr/local/lib

# 虚拟显示（如果需要）
PLUM_CONTAINER_ENV=DISPLAY=:99
```

#### 验证

```bash
# 1. 检查容器内的库路径是否挂载成功
docker exec <container-name> ls -la /opt/qt/lib/

# 2. 检查库是否可以加载
docker exec <container-name> ldd /app/HeyUI | grep Qt

# 3. 查看Agent日志确认挂载
# 应该看到：Mounted host library path /opt/qt/lib to container
```

---

### 方案4：使用虚拟显示（无GUI需求时）

如果应用不需要真实显示，可以使用虚拟显示：

```bash
# 1. 在基础镜像中安装xvfb（虚拟X服务器）
# 使用ubuntu镜像并安装xvfb

# 2. 在启动脚本中启动xvfb
# start.sh:
#!/bin/sh
Xvfb :99 -screen 0 1024x768x24 &
export DISPLAY=:99
./HeyUI

# 3. 或者通过环境变量配置
PLUM_CONTAINER_ENV=DISPLAY=:99
```

---

## 📊 方案对比

| 方案 | 优点 | 缺点 | 适用场景 |
|------|------|------|----------|
| **方案1：Ubuntu镜像** | 简单，镜像包含基础库 | 镜像较大，可能缺少Qt库 | 需要基础系统库的应用 |
| **方案2：应用包含库** | 自包含，不依赖宿主机，支持不同版本 | 重复存储，artifact大 | 单个应用或需要特定版本库 |
| **方案3：共享宿主机库** ⭐ | **避免重复，统一管理，减少artifact大小** | 依赖宿主机，架构需兼容 | **多个应用共享相同库** |
| **方案4：虚拟显示** | 无需真实GUI环境 | 需要xvfb | 无GUI需求的应用 |

---

## 🎯 推荐组合

### 场景1：单个Qt应用 + 特定版本库

```bash
# 方案2：应用包含库
# 优点：完全自包含，可以使用特定版本
```

### 场景2：多个Qt应用 + 共享相同库 ⭐

```bash
# 方案3：共享宿主机库
# agent-go/.env
PLUM_HOST_LIB_PATHS=/opt/qt/lib
PLUM_CONTAINER_ENV=DISPLAY=:99
```

### 场景3：需要GUI显示

```bash
# 方案3（共享库） + 方案4（虚拟显示）
PLUM_HOST_LIB_PATHS=/opt/qt/lib
PLUM_CONTAINER_ENV=DISPLAY=:99
```

---

## 📝 快速开始示例

### 示例1：使用共享宿主机Qt库（推荐多应用场景）

```bash
# 1. 确保宿主机有Qt库
ls /opt/qt/lib/  # 或其他路径

# 2. 配置Agent
# agent-go/.env
AGENT_RUN_MODE=docker
PLUM_BASE_IMAGE=ubuntu:22.04
PLUM_HOST_LIB_PATHS=/opt/qt/lib

# 3. 应用artifact只需要可执行文件和启动脚本
# 不需要包含Qt库

# 4. 启动应用
# 容器会自动挂载宿主机库路径
```

### 示例2：应用自包含Qt库（推荐单应用场景）

```bash
# 1. 打包应用时包含Qt库
app/
├── HeyUI
├── start.sh
└── lib/
    └── libQt5*.so.5  # Qt库

# 2. 配置Agent
AGENT_RUN_MODE=docker
PLUM_BASE_IMAGE=ubuntu:22.04

# 3. Agent自动检测 lib/ 目录并设置 LD_LIBRARY_PATH
```

---

## ⚠️ 常见问题

### Q1：使用方案3时，容器内找不到库？

**检查**：
1. 宿主机路径是否存在：`ls /opt/qt/lib/`
2. Agent日志是否显示挂载成功：`Mounted host library path ...`
3. 容器和宿主机架构是否一致：`uname -m`
4. 容器内路径是否正确：`docker exec <container> ls /opt/qt/lib/`

### Q2：多个应用需要不同版本的库怎么办？

**方案**：
- 使用方案2（应用包含库），每个应用包含自己的库版本
- 或使用多个宿主机路径，分别挂载不同版本的库

### Q3：宿主机和容器架构不一致？

**解决**：
- 确保宿主机和容器使用相同的CPU架构
- 使用相同架构的基础镜像（如都是ubuntu:22.04 x86_64）

---

## 📚 相关文档

- [环境变量配置指南](./ENV_CONFIG.md) - 完整配置项说明
- [容器应用管理](./CONTAINER_APP_MANAGEMENT.md) - 容器模式详细说明
- [测试容器模式](./TEST_CONTAINER_MODE.md) - 测试步骤
