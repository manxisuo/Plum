# Plum 离线部署包

这个目录包含了Plum项目离线部署到银河麒麟V10 ARM64环境的所有必要文件。

> **📚 详细部署指南**: 请参考 `source/Plum/docker/DEPLOYMENT-GUIDE.md` 获取完整的部署说明，包括Docker和传统两种部署方式。

## 📁 目录结构

```
plum-offline-deploy/
├── README.md                          # 本文件 - 部署包说明
├── docs/                              # 部署相关文档
│   ├── OFFLINE_DEPLOYMENT_SUMMARY.md  # 离线部署总结
│   ├── ENVIRONMENT_COMPARISON.md      # 环境对比分析
│   ├── FILE_COMPARISON_TABLE.md       # 文件对比表格
│   └── GO_TOOLS_REQUIREMENT.md        # Go工具需求说明
├── scripts-prepare/                   # WSL2环境准备脚本
│   ├── prepare-offline-deploy.sh      # 主准备脚本
│   ├── prepare-arm64-go-tools.sh     # ARM64工具准备脚本
│   └── fix-permissions.sh            # 权限修复脚本
├── scripts/                           # 目标环境部署脚本
│   ├── install-deps.sh               # 依赖安装脚本
│   ├── build-all.sh                  # 构建脚本
│   ├── build-cpp-sdk.sh              # C++ SDK构建脚本
│   ├── check-cpp-deps.sh             # C++依赖检查脚本
│   ├── install-cpp-sdk.sh            # C++ SDK安装脚本
│   └── deploy.sh                     # 部署脚本
├── tools/                            # 构建工具（ARM64版本）
│   ├── go1.24.3.linux-arm64.tar.gz # Go ARM64版本
│   ├── node-v18.20.4-linux-arm64.tar.xz # Node.js ARM64版本
│   └── go-arm64-tools/               # Go protobuf工具（ARM64）
├── source/                           # 项目源码（包含依赖）
│   └── Plum/                        # 完整项目源码
│       ├── controller/               # Controller源码+vendor依赖
│       ├── agent-go/                 # Agent源码+vendor依赖
│       ├── ui/                       # Web UI源码+node_modules
│       └── ...                       # 其他源码文件
└── go-vendor-backup/                 # Go依赖包备份
    ├── controller-vendor/            # Controller依赖备份
    └── agent-vendor/                 # Agent依赖备份
```

## 🚀 使用说明

### 在WSL2环境中准备（已完成）
```bash
# 在项目根目录运行
./plum-offline-deploy/scripts-prepare/prepare-offline-deploy.sh
```

### 在目标环境部署

#### 方案1：Docker容器部署（推荐）
```bash
cd plum-offline-deploy/source/Plum

# 方案A：使用预构建镜像包
./docker/load-offline-images.sh
docker-compose -f docker-compose.offline.yml up -d

# 方案B：在目标环境构建镜像
./docker/build-static-offline.sh
docker-compose -f docker-compose.offline.yml up -d
```

#### 方案2：传统源码部署
```bash
cd plum-offline-deploy/scripts

# 1. 安装依赖
./install-deps.sh

# 2. 构建项目（包含C++ SDK）
./build-all.sh

# 3. 部署服务
./deploy.sh
```

**详细部署指南请参考**: `source/Plum/docker/DEPLOYMENT-GUIDE.md`

### 单独构建C++ SDK
```bash
cd plum-offline-deploy/scripts

# 1. 检查C++依赖
./check-cpp-deps.sh

# 2. 构建C++ SDK
./build-cpp-sdk.sh

# 3. 安装C++ SDK到系统（可选）
sudo ./install-cpp-sdk.sh
```

### C++ SDK依赖问题
如果遇到C++ SDK依赖问题，可以运行：
```bash
cd plum-offline-deploy/scripts

# 检查C++ SDK依赖
./check-cpp-deps.sh

# 或者安装完整的C++ SDK依赖
./install-cpp-deps.sh
```

## 📋 文件说明

### 准备脚本（scripts-prepare/）
- **prepare-offline-deploy.sh**: 主要准备脚本，复制源码和依赖
- **prepare-arm64-go-tools.sh**: ARM64工具交叉编译脚本
- **fix-permissions.sh**: 权限修复脚本

### 部署脚本（scripts/）
- **install-deps.sh**: 在目标环境安装Go、Node.js和系统依赖
- **build-all.sh**: 构建Controller、Agent、Web UI和C++ SDK
- **build-cpp-sdk.sh**: 专门构建C++ SDK和Plum Client库
- **check-cpp-deps.sh**: 检查C++ SDK依赖（CMake、httplib、pthread等）
- **install-cpp-deps.sh**: 安装C++ SDK依赖（httplib、pthread等）
- **install-cpp-sdk.sh**: 将C++ SDK安装到系统目录
- **deploy.sh**: 部署为systemd服务并配置nginx

### 工具文件（tools/）
- **go1.24.3.linux-arm64.tar.gz**: Go 1.24.3 ARM64版本
- **node-v18.20.4-linux-arm64.tar.xz**: Node.js 18.x ARM64版本
- **go-arm64-tools/**: 预编译的protobuf工具（ARM64）

### 源码（source/）
- **Plum/**: 完整的项目源码，包含所有vendor和node_modules依赖

## 🎯 关键特性

1. **完全离线**: 所有依赖都已预下载，无需网络连接
2. **架构匹配**: 所有工具和构建产物都是ARM64版本
3. **依赖完整**: 包含Go vendor和Node.js node_modules
4. **C++ SDK支持**: 包含Plum Client库和示例程序
5. **文档齐全**: 详细的部署文档和说明

## 🔧 故障排除

1. **Go工具问题**: 确保使用了ARM64版本的Go和protobuf工具
2. **依赖缺失**: 检查vendor和node_modules目录是否存在
3. **C++ SDK问题**: 确保安装了CMake、httplib和pthread开发包
4. **权限问题**: 确保脚本有执行权限，服务用户有适当权限
5. **网络问题**: 如果遇到网络依赖，使用预下载的工具文件

## 📞 支持

如遇问题，请参考：
- `docs/OFFLINE_DEPLOYMENT_SUMMARY.md` - 详细部署指南
- `docs/ENVIRONMENT_COMPARISON.md` - 环境对比说明
- `docs/GO_TOOLS_REQUIREMENT.md` - Go工具需求说明
