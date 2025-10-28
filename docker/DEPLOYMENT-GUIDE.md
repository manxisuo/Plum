# Plum Docker 部署指南

## 📋 概述

本指南包含Plum项目的Docker部署方案：
- **在线部署**：使用Docker Compose快速启动
- **离线部署**：ARM64环境下的完整离线部署流程

## 🚀 在线部署

### Docker Compose 文件说明

| 文件 | 用途 | 服务 |
|------|------|------|
| `docker-compose.yml` | 测试环境 | Controller + 3个Agent + Nginx |
| `docker-compose.production.yml` | 生产环境 | 单节点Agent |
| `docker-compose.offline.yml` | 离线环境 | Controller + 3个Agent + Nginx |

### 启动命令

```bash
# 测试环境（完整系统）
docker-compose up -d

# 生产环境（单节点）
docker-compose -f docker-compose.production.yml up -d

# 仅启动Controller
docker-compose up -d plum-controller

# 带Nginx的测试环境
docker-compose --profile nginx up -d
```

### 自动化脚本

```bash
# 使用deploy.sh脚本
./docker/deploy.sh test start    # 测试环境
./docker/deploy.sh production start  # 生产环境
./docker/deploy.sh controller start  # 仅Controller
```

## 🏗️ 离线部署（ARM64）

### 联网环境准备

#### 1. 准备部署包
```bash
# 创建离线部署包
./plum-offline-deploy/scripts-prepare/prepare-offline-deploy.sh
# 选择不构建Docker镜像（输入N）
```

#### 2. 准备Docker镜像（可选方案）
```bash
# 方案A：仅准备nginx（推荐）
docker pull --platform linux/arm64 alpine:3.18
docker pull --platform linux/arm64 nginx:alpine
docker save alpine:3.18 | gzip > alpine-3.18-arm64.tar.gz
docker save nginx:alpine | gzip > nginx-alpine-arm64.tar.gz

# 方案B：准备完整镜像包（使用脚本）
./docker/generate-offline-images.sh
```

#### 3. 打包传输
```bash
# 打包部署包
tar -czf plum-offline-deploy.tar.gz plum-offline-deploy/

# 传输文件到目标环境
# plum-offline-deploy.tar.gz
# alpine-3.18-arm64.tar.gz
# nginx-alpine-arm64.tar.gz
```

### 离线环境部署

#### 1. 解压部署包
```bash
tar -xzf plum-offline-deploy.tar.gz
cd plum-offline-deploy/source/Plum
```

#### 2. 加载Docker镜像
```bash
# 方案A：仅加载基础镜像（推荐）
docker load < alpine-3.18-arm64.tar.gz
docker load < nginx-alpine-arm64.tar.gz

# 方案B：加载完整镜像包（使用脚本）
./docker/load-offline-images.sh
```

#### 3. 构建Plum镜像（方案A需要）
```bash
# 如果使用方案A，需要构建Plum镜像
./docker/build-static-offline-fixed.sh
```

#### 4. 启动服务
```bash
# 启动所有服务
docker-compose -f docker-compose.offline.yml up -d

# 检查状态
docker-compose -f docker-compose.offline.yml ps
```

### 验证部署

```bash
# 测试Controller
curl http://localhost:8080/v1/nodes

# 测试Nginx
curl http://localhost/health

# 查看日志
docker-compose -f docker-compose.offline.yml logs -f
```

## 🔧 故障排除

### 常见问题

1. **架构不匹配**：镜像与目标机CPU不一致（amd64 vs arm64）。
   - 生成包时脚本会在文件名中加入架构后缀（如 `-amd64`、`-arm64`）。
   - 验证命令: `docker inspect <image:tag> | grep -i Architecture`
   - 选择与目标环境相同架构的 `.tar.gz` 加载；否则请在目标机上重建镜像。
2. **端口冲突**：检查8080、80端口占用
3. **配置文件缺失**：确保.env文件存在

### 日志查看
```bash
docker-compose -f docker-compose.offline.yml logs plum-controller
docker-compose -f docker-compose.offline.yml logs plum-agent-a
```

## 📞 获取帮助

如果遇到问题，请参考：
- **[问题解决指南](TROUBLESHOOTING-GUIDE.md)** - 常见问题及解决方案
- 查看服务日志：`docker-compose logs -f`
- 检查服务状态：`docker-compose ps`
- 参考Docker官方文档

---

*最后更新：2025年10月29日*
