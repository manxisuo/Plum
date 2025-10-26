# Controller IP 配置指南

## 🔍 **什么是 CONTROLLER_BASE？**

`CONTROLLER_BASE` 是Agent节点用来连接Controller的环境变量，需要设置为Controller节点的实际地址。

## 📋 **配置方式**

### **方式1：使用IP地址（推荐）**
```bash
# 获取Controller节点IP
# 在Controller节点上执行：
ip addr show | grep inet
# 或者
hostname -I

# 在Agent节点上设置
export CONTROLLER_BASE=http://192.168.1.100:8080
```

### **方式2：使用域名**
```bash
# 如果Controller有域名
export CONTROLLER_BASE=http://plum-controller.company.com:8080
```

### **方式3：使用Docker服务名（同网络）**
```bash
# 如果Controller和Agent在同一个Docker网络中
export CONTROLLER_BASE=http://plum-controller:8080
```

## 🛠️ **实际部署示例**

### **场景1：单机部署（测试）**
```bash
# Controller和Agent在同一台机器
export CONTROLLER_BASE=http://localhost:8080
docker-compose -f docker-compose.production.yml up -d
```

### **场景2：分布式部署**
```bash
# Controller节点：192.168.1.100
# Agent节点：192.168.1.101

# 在Agent节点上执行
export AGENT_NODE_ID=worker-001
export CONTROLLER_BASE=http://192.168.1.100:8080
docker-compose -f docker-compose.production.yml up -d
```

### **场景3：云环境部署**
```bash
# Controller在云服务器上
export AGENT_NODE_ID=cloud-worker-001
export CONTROLLER_BASE=http://controller.example.com:8080
docker-compose -f docker-compose.production.yml up -d
```

## 🔧 **获取Controller IP的方法**

### **方法1：查看网络接口**
```bash
# 查看所有网络接口
ip addr show

# 查看特定接口（如eth0）
ip addr show eth0
```

### **方法2：查看Docker容器IP**
```bash
# 查看Controller容器IP
docker inspect plum-controller | grep IPAddress

# 或者使用docker-compose
docker-compose exec plum-controller hostname -i
```

### **方法3：使用ping测试**
```bash
# 从Agent节点ping Controller节点
ping controller-node-ip
```

## 🚨 **常见问题**

### **问题1：连接超时**
```bash
# 检查网络连通性
curl -v http://controller-ip:8080/v1/health

# 检查防火墙
sudo ufw status
```

### **问题2：DNS解析失败**
```bash
# 使用IP地址而不是域名
export CONTROLLER_BASE=http://192.168.1.100:8080

# 或者添加DNS记录
echo "192.168.1.100 controller" >> /etc/hosts
```

### **问题3：端口不可达**
```bash
# 检查Controller是否启动
docker-compose ps plum-controller

# 检查端口是否开放
netstat -tulpn | grep :8080
```

## 💡 **最佳实践**

### **1. 使用环境变量文件**
```bash
# 创建 .env 文件
echo "CONTROLLER_BASE=http://192.168.1.100:8080" > .env
echo "AGENT_NODE_ID=worker-001" >> .env

# 使用环境变量文件
docker-compose -f docker-compose.production.yml --env-file .env up -d
```

### **2. 使用部署脚本**
```bash
# 设置环境变量后使用脚本
export AGENT_NODE_ID=worker-001
export CONTROLLER_BASE=http://192.168.1.100:8080
./deploy.sh agent start
```

### **3. 验证连接**
```bash
# 启动后验证连接
docker-compose -f docker-compose.production.yml logs plum-agent

# 查看Agent是否成功注册到Controller
curl http://controller-ip:8080/v1/nodes
```

## 📊 **配置检查清单**

- [ ] Controller节点IP地址已确认
- [ ] 网络连通性已测试
- [ ] 端口8080已开放
- [ ] 防火墙规则已配置
- [ ] DNS解析正常（如果使用域名）
- [ ] Agent节点环境变量已设置
- [ ] 启动后连接状态已验证

---

## 🎯 **总结**

`controller-ip` 是一个占位符，需要替换为实际的Controller节点IP地址。根据部署环境选择合适的配置方式，并确保网络连通性。
