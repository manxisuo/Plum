# Plum Docker 部署文档

## 📋 文档概览

本目录包含了Plum项目的完整Docker部署解决方案，包括：

- **详细部署指南**: `README-DEPLOYMENT.md`
- **快速启动指南**: `QUICK-START.md` 
- **使用示例**: `USAGE-EXAMPLES.md`
- **自动化脚本**: `deploy.sh`

## 🚀 快速开始

### 使用自动化脚本（推荐）

```bash
# 进入docker目录
cd docker

# 查看帮助
./deploy.sh help

# 启动测试环境
./deploy.sh test start

# 查看状态
./deploy.sh test status

# 停止服务
./deploy.sh test stop
```

### 手动使用Docker Compose

```bash
# 测试环境（Controller + 3个Agent）
docker-compose up -d

# 生产环境
docker-compose -f docker-compose.production.yml up -d

# 带nginx的测试环境
docker-compose --profile nginx up -d
```

## 📚 文档说明

### 1. README-DEPLOYMENT.md
**详细部署指南** - 包含完整的部署流程和最佳实践
- 环境准备
- 测试环境部署
- 生产环境部署
- 服务管理
- 故障排除
- 最佳实践

### 2. QUICK-START.md
**快速启动指南** - 常用命令和快速参考
- 常用启动命令
- 服务管理命令
- 常见问题解决
- 端口说明
- 部署方式选择

### 3. USAGE-EXAMPLES.md
**使用示例** - 详细的使用场景和操作示例
- 开发测试流程
- 集成测试流程
- UI测试流程
- 生产部署流程
- 分布式部署流程
- 维护操作

### 4. deploy.sh
**自动化部署脚本** - 简化日常操作
- 支持多种环境
- 自动化操作
- 健康检查
- 数据备份/恢复
- 资源清理

## 🎯 部署环境

### 测试环境
- **test**: 完整测试环境（Controller + 3个Agent）
- **test-simple**: 简单测试环境（仅Controller）
- **test-nginx**: 测试环境（包含nginx）

### 生产环境
- **production**: 生产环境配置
- **controller**: 仅启动Controller
- **agent**: 仅启动Agent

## 🔧 常用操作

### 启动服务
```bash
# 测试环境
./deploy.sh test start

# 生产环境
./deploy.sh production start

# 简单测试
./deploy.sh test-simple start
```

### 管理服务
```bash
# 查看状态
./deploy.sh test status

# 查看日志
./deploy.sh test logs

# 重启服务
./deploy.sh test restart

# 停止服务
./deploy.sh test stop
```

### 维护操作
```bash
# 健康检查
./deploy.sh test health

# 备份数据
./deploy.sh backup

# 清理资源
./deploy.sh clean
```

## 🌐 服务访问

### API接口
- **Controller API**: http://localhost:8080/v1/nodes
- **nginx代理**: http://localhost/v1/nodes

### Web界面
- **nginx服务**: http://localhost

## 📊 服务端口

| 服务 | 端口 | 用途 |
|------|------|------|
| Controller | 8080 | API接口 |
| nginx | 80/443 | Web UI和反向代理 |
| Agent | 内部 | 与Controller通信 |

## 🐛 故障排除

### 常见问题
1. **网络冲突**: 使用 `docker network prune` 清理
2. **端口冲突**: 检查端口占用，停止冲突服务
3. **内存不足**: 清理Docker资源或增加系统内存
4. **权限问题**: 检查文件权限设置

### 调试命令
```bash
# 查看详细日志
./deploy.sh test logs

# 检查服务状态
./deploy.sh test status

# 执行健康检查
./deploy.sh test health

# 查看Docker资源
docker stats
```

## 💡 最佳实践

### 1. 环境隔离
- 开发环境使用 `test-simple`
- 测试环境使用 `test`
- 生产环境使用 `production`

### 2. 数据管理
- 定期备份重要数据
- 使用命名卷存储数据
- 监控磁盘空间使用

### 3. 资源管理
- 设置合理的资源限制
- 定期清理无用资源
- 监控系统资源使用

### 4. 安全配置
- 使用非root用户运行
- 限制网络访问
- 定期更新镜像

## 📞 技术支持

### 获取帮助
```bash
# 查看脚本帮助
./deploy.sh help

# 查看Docker帮助
docker-compose --help

# 查看服务状态
./deploy.sh test status
```

### 调试模式
```bash
# 启用详细日志
export COMPOSE_LOG_LEVEL=DEBUG
./deploy.sh test start
```

---

## 📝 总结

本Docker部署解决方案提供了：

✅ **完整的部署文档** - 从测试到生产的完整指南  
✅ **自动化脚本** - 简化日常操作  
✅ **多种环境支持** - 适应不同使用场景  
✅ **故障排除指南** - 快速解决问题  
✅ **最佳实践** - 确保部署质量  

通过使用这些工具和文档，您可以轻松地部署和管理Plum系统，无论是用于开发测试还是生产环境。