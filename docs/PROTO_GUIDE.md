# Proto编译指南

## 🎯 Proto目录设计

### 为什么放在根目录？

Proto定义是**跨组件的接口契约**，放在根目录的原因：

1. **多组件共享**
   ```
   proto/task_service.proto (源文件)
       ↓
   ├── Controller (Go client)   使用
   ├── C++ Worker SDK (gRPC server)   使用
   ├── Agent-Go (未来可能)   使用
   └── Python SDK (未来)   使用
   ```

2. **单一数据源**
   - 一个proto文件，多处使用
   - 避免定义重复和不一致
   - 接口变更时只需修改一处

3. **符合微服务最佳实践**
   ```
   ✅ 推荐：
   project/
   ├── proto/           ← 共享接口定义
   ├── service-a/
   └── service-b/
   
   ❌ 不推荐：
   project/
   ├── service-a/proto/  ← 各自定义，易不一致
   └── service-b/proto/
   ```

## 🔨 编译方法

### 一键编译
```bash
make proto
```

这会：
1. 检查protoc是否安装
2. 自动安装Go插件（如果缺失）
3. 生成Go代码到controller/plum/proto/
4. 生成C++代码到sdk/cpp/grpc/proto/

### 详细输出
```
🔨 Generating protobuf code...
✓ protoc version: libprotoc 3.12.4
📦 Generating Go code...
✅ Go code generated
📦 Generating C++ code...
✅ C++ code generated
✅ All done!
```

## 📦 生成代码位置

### Go代码（Controller使用）
```
controller/plum/proto/
├── task_service.pb.go         # 消息类型
└── task_service_grpc.pb.go    # gRPC服务
```

**使用方式**：
```go
import pb "plum/controller/plum/proto"

client := pb.NewTaskServiceClient(conn)
```

### C++代码（Worker SDK使用）
```
sdk/cpp/grpc/proto/
├── task_service.pb.h          # 消息头文件
├── task_service.pb.cc         # 消息实现
├── task_service.grpc.pb.h     # gRPC服务头文件
└── task_service.grpc.pb.cc    # gRPC服务实现
```

**使用方式**：
```cpp
#include "proto/task_service.grpc.pb.h"

class TaskServiceImpl : public TaskService::Service { ... };
```

## 🔧 依赖安装

### Ubuntu/Debian
```bash
# protobuf编译器
sudo apt install protobuf-compiler

# C++ gRPC插件
sudo apt install libgrpc++-dev protobuf-compiler-grpc

# Go插件（脚本自动安装，也可手动）
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### 验证安装
```bash
protoc --version                # libprotoc 3.12.4+
grpc_cpp_plugin --version       # C++插件
protoc-gen-go --version         # Go插件
protoc-gen-go-grpc --version    # Go gRPC插件
```

## 📝 修改Proto的流程

### 完整工作流

1. **修改proto定义**
   ```bash
   vim proto/task_service.proto
   ```

2. **生成代码**
   ```bash
   make proto
   ```

3. **重新编译使用方**
   ```bash
   # Controller（Go）
   make controller
   
   # C++ Worker SDK
   make sdk_cpp
   
   # 重新编译example
   make sdk_cpp_grpc_echo_worker
   ```

4. **测试验证**
   ```bash
   # 启动Controller
   ./controller/bin/controller
   
   # 启动Worker
   ./sdk/cpp/build/examples/grpc_echo_worker/grpc_echo_worker
   ```

## 🎓 Proto版本管理

### 保持向后兼容

修改proto时遵循：
- ✅ 添加新字段：使用递增的字段编号
- ✅ 添加新消息类型：不影响现有代码
- ✅ 添加新RPC方法：老客户端不受影响
- ❌ 删除字段：破坏兼容性
- ❌ 修改字段编号：破坏兼容性
- ❌ 修改字段类型：破坏兼容性

### 示例：安全添加字段
```protobuf
message TaskRequest {
    string task_id = 1;
    string name = 2;
    string payload = 3;
    int32 timeout_sec = 4;    // ✅ 新增字段，使用新编号
}
```

## ⚙️ 高级配置

### 自定义生成路径

编辑`proto/generate.sh`：
```bash
# Go生成路径
protoc --go_out=./your-path ...

# C++生成路径
protoc --cpp_out=./your-path ...
```

### 添加新的proto文件

1. 创建proto文件：`proto/new_service.proto`
2. 在generate.sh中添加编译命令
3. 运行`make proto`

## 🐛 常见问题

### Q: protoc: command not found
```bash
sudo apt install protobuf-compiler
```

### Q: protoc-gen-go: program not found
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
# 确保$GOPATH/bin在PATH中
export PATH=$PATH:$(go env GOPATH)/bin
```

### Q: 生成代码提示版本不匹配
更新插件到最新版本：
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## 📊 当前Proto使用情况

### task_service.proto

**定义者**：根目录proto/  
**Go使用者**：controller/internal/grpc/client.go  
**C++使用者**：sdk/cpp/examples/grpc_echo_worker/

**通信模式**：
```
Controller (Go)
    ↓ gRPC调用
C++ Worker SDK
    ↓ 执行任务
应用程序
```

---

**提示**：修改proto后记得运行`make proto`并重新编译所有使用方！

