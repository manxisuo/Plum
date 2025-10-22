package main

import (
	"fmt"
	"log"
	"time"

	plum "github.com/manxisuo/plum/sdk/go"
)

func main() {
	// 创建Plum客户端
	client := plum.NewPlumClient("http://localhost:8080")

	// 示例1: 服务发现
	fmt.Println("=== 服务发现示例 ===")
	endpoints, err := client.DiscoverService("my-service", "v1.0", "http")
	if err != nil {
		log.Printf("服务发现失败: %v", err)
	} else {
		fmt.Printf("发现 %d 个端点:\n", len(endpoints))
		for i, ep := range endpoints {
			fmt.Printf("  %d. %s://%s:%d (健康: %v)\n",
				i+1, ep.Protocol, ep.IP, ep.Port, ep.Healthy)
		}
	}

	// 示例2: 随机服务发现
	fmt.Println("\n=== 随机服务发现示例 ===")
	endpoint, err := client.DiscoverServiceRandom("my-service", "", "")
	if err != nil {
		log.Printf("随机服务发现失败: %v", err)
	} else {
		fmt.Printf("随机选择的端点: %s://%s:%d\n",
			endpoint.Protocol, endpoint.IP, endpoint.Port)
	}

	// 示例3: 服务调用
	fmt.Println("\n=== 服务调用示例 ===")
	result, err := client.CallService("my-service", "GET", "/api/health", nil, nil)
	if err != nil {
		log.Printf("服务调用失败: %v", err)
	} else {
		fmt.Printf("调用成功: HTTP %d, 延迟 %v\n",
			result.StatusCode, result.Latency)
		fmt.Printf("响应: %s\n", string(result.Body))
	}

	// 示例4: 带重试的服务调用
	fmt.Println("\n=== 带重试的服务调用示例 ===")
	result, err = client.CallServiceWithRetry("my-service", "GET", "/api/data", nil, nil, 3)
	if err != nil {
		log.Printf("重试调用失败: %v", err)
	} else {
		fmt.Printf("重试调用成功: HTTP %d, 延迟 %v\n",
			result.StatusCode, result.Latency)
	}

	// 示例5: 负载均衡调用
	fmt.Println("\n=== 负载均衡调用示例 ===")
	for i := 0; i < 5; i++ {
		result, err := client.LoadBalance("my-service", "GET", "/api/status", nil, nil, "random")
		if err != nil {
			log.Printf("负载均衡调用失败: %v", err)
		} else {
			fmt.Printf("调用 %d: HTTP %d, 延迟 %v\n",
				i+1, result.StatusCode, result.Latency)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 示例6: 服务注册
	fmt.Println("\n=== 服务注册示例 ===")
	err = client.RegisterService(
		"example-instance-1",
		"example-service",
		"v1.0",
		"http",
		"192.168.1.100",
		8080,
		map[string]string{
			"env":    "test",
			"region": "us-west",
		},
	)
	if err != nil {
		log.Printf("服务注册失败: %v", err)
	} else {
		fmt.Println("服务注册成功")
	}

	// 示例7: 心跳保持
	fmt.Println("\n=== 心跳保持示例 ===")
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			endpoints := []plum.Endpoint{
				{
					ServiceName: "example-service",
					InstanceID:  "example-instance-1",
					IP:          "192.168.1.100",
					Port:        8080,
					Protocol:    "http",
					Version:     "v1.0",
					Healthy:     true,
					LastSeen:    time.Now().Unix(),
				},
			}

			if err := client.Heartbeat("example-instance-1", endpoints); err != nil {
				log.Printf("心跳失败: %v", err)
			} else {
				fmt.Println("心跳发送成功")
			}
		}
	}()

	// 运行一段时间
	time.Sleep(30 * time.Second)
	fmt.Println("示例完成")
}
