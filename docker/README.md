# Plum Docker 部署

## 📚 文档

- **[部署指南](DEPLOYMENT-GUIDE.md)** ⭐ **完整部署说明**
- **[问题解决指南](TROUBLESHOOTING-GUIDE.md)** 🔧 **常见问题解决方案**

## 🛠️ 工具脚本

- **[copy-libs.sh](copy-libs.sh)** 📦 **动态库文件复制脚本**
- **[smart-copy-libs.sh](smart-copy-libs.sh)** 🧠 **智能库文件复制脚本**

## 🚀 快速开始

### 在线部署
```bash
# 测试环境
docker-compose up -d

# 生产环境  
docker-compose -f docker-compose.production.yml up -d

# 使用脚本
./docker/deploy.sh test start
```

### 离线部署
```bash
# 1. 构建镜像
./docker/build-static-offline-fixed.sh

# 2. 启动服务
docker-compose -f docker-compose.offline.yml up -d
```

详细说明请参考：[部署指南](DEPLOYMENT-GUIDE.md)