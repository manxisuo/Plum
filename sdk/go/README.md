# Plum Go SDK

Plum分布式任务调度系统的Go客户端SDK，提供服务发现、服务调用、负载均衡等功能。

## 功能特性

- ✅ 服务发现（支持随机选择）
- ✅ 服务调用（支持重试）
- ✅ 负载均衡（随机、轮询）
- ✅ 服务注册
- ✅ 心跳保持
- ✅ 本地缓存
- ✅ 弱网环境支持

## 安装

```bash
go get github.com/manxisuo/plum/sdk/go
```

## 快速开始

### 1. 创建客户端

```go
import "github.com/manxisuo/plum/sdk/go"

client := plum.NewPlumClient("http://localhost:8080")
```

### 2. 服务发现

```go
// 发现所有端点
endpoints, err := client.DiscoverService("my-service", "v1.0", "http")
if err != nil {
    log.Fatal(err)
}

// 随机选择一个端点
endpoint, err := client.DiscoverServiceRandom("my-service", "", "")
if err != nil {
    log.Fatal(err)
}
```

### 3. 服务调用

```go
// 简单调用
result, err := client.CallService("my-service", "GET", "/api/data", nil, nil)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Response: %s\n", string(result.Body))

// 带重试的调用
result, err = client.CallServiceWithRetry("my-service", "POST", "/api/process", 
    map[string]string{"Authorization": "Bearer token"}, 
    []byte(`{"key":"value"}`), 
    3) // 最多重试3次
```

### 4. 负载均衡

```go
// 随机负载均衡
result, err := client.LoadBalance("my-service", "GET", "/api/status", nil, nil, "random")

// 轮询负载均衡
result, err := client.LoadBalance("my-service", "GET", "/api/status", nil, nil, "round_robin")
```

### 5. 服务注册

```go
err := client.RegisterService(
    "my-instance-1",
    "my-service",
    "v1.0",
    "http",
    "192.168.1.100",
    8080,
    map[string]string{
        "env": "production",
        "region": "us-west",
    },
)
```

### 6. 心跳保持

```go
go func() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        endpoints := []plum.Endpoint{{
            ServiceName: "my-service",
            InstanceID:  "my-instance-1",
            IP:          "192.168.1.100",
            Port:        8080,
            Protocol:    "http",
            Version:     "v1.0",
            Healthy:     true,
            LastSeen:    time.Now().Unix(),
        }}
        
        if err := client.Heartbeat("my-instance-1", endpoints); err != nil {
            log.Printf("Heartbeat failed: %v", err)
        }
    }
}()
```

## API文档

### PlumClient

#### NewPlumClient(controllerURL string) *PlumClient
创建新的Plum客户端。

#### DiscoverService(serviceName, version, protocol string) ([]Endpoint, error)
发现服务的所有端点。

#### DiscoverServiceRandom(serviceName, version, protocol string) (*Endpoint, error)
随机选择一个服务端点。

#### CallService(serviceName, method, path string, headers map[string]string, body []byte) (*ServiceCallResult, error)
调用服务（自动服务发现）。

#### CallServiceWithRetry(serviceName, method, path string, headers map[string]string, body []byte, maxRetries int) (*ServiceCallResult, error)
带重试的服务调用。

#### LoadBalance(serviceName, method, path string, headers map[string]string, body []byte, strategy string) (*ServiceCallResult, error)
负载均衡调用。支持策略：random、round_robin。

#### RegisterService(instanceID, serviceName, version, protocol, host string, port int, labels map[string]string) error
注册服务。

#### Heartbeat(instanceID string, endpoints []Endpoint) error
发送心跳。

#### ClearCache()
清除本地缓存。

## 完整示例

参考 [examples/service_client_example.go](examples/service_client_example.go)

## 性能优化

- 使用本地缓存减少服务发现请求
- 支持连接复用
- 自动重试机制
- 弱网环境支持

## 许可证

MIT License

