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

	// 启用网络监控
	client.StartNetworkMonitoring(5 * time.Second)
	defer client.StopNetworkMonitoring()

	// 启用自适应模式
	client.EnableAdaptiveMode()

	fmt.Println("=== 弱网环境支持示例 ===")

	// 显示初始网络状态
	fmt.Printf("初始网络质量: %s\n", client.GetNetworkQuality())
	fmt.Printf("是否为弱网环境: %v\n", client.IsWeakNetwork())

	// 显示当前配置
	config := client.GetConfig()
	fmt.Printf("当前配置: %s\n", config.String())

	// 模拟服务发现
	fmt.Println("\n=== 服务发现测试 ===")

	// 尝试发现服务
	endpoints, err := client.DiscoverService("example-service", "", "")
	if err != nil {
		log.Printf("服务发现失败: %v", err)
	} else {
		fmt.Printf("发现 %d 个端点\n", len(endpoints))
	}

	// 随机服务发现
	endpoint, err := client.DiscoverServiceRandom("example-service", "", "")
	if err != nil {
		log.Printf("随机服务发现失败: %v", err)
	} else {
		fmt.Printf("随机选择的端点: %s://%s:%d\n",
			endpoint.Protocol, endpoint.IP, endpoint.Port)
	}

	// 监控网络状态变化
	fmt.Println("\n=== 网络状态监控 ===")

	for i := 0; i < 10; i++ {
		time.Sleep(2 * time.Second)

		quality := client.GetNetworkQuality()
		stats := client.GetNetworkStats()
		isWeak := client.IsWeakNetwork()

		fmt.Printf("监控周期 %d: 质量=%s, 弱网=%v, 延迟=%v, 成功率=%.2f%%\n",
			i+1, quality, isWeak, stats.Latency, stats.SuccessRate*100)

		// 如果网络质量发生变化，显示新配置
		if i > 0 {
			newConfig := client.GetConfig()
			fmt.Printf("  当前配置: CacheTTL=%v, RetryMaxAttempts=%d, RequestTimeout=%v\n",
				newConfig.CacheTTL, newConfig.RetryMaxAttempts, newConfig.RequestTimeout)
		}
	}

	// 测试自定义配置
	fmt.Println("\n=== 自定义配置测试 ===")

	// 创建弱网环境配置
	weakConfig := &plum.WeakNetworkConfig{
		CacheTTL:          2 * time.Minute,  // 更长的缓存时间
		RetryMaxAttempts:  10,               // 更多重试次数
		RetryBaseDelay:    1 * time.Second,  // 更长的重试延迟
		RetryMaxDelay:     30 * time.Second, // 最大重试延迟
		RequestTimeout:    60 * time.Second, // 更长的请求超时
		HeartbeatInterval: 30 * time.Second, // 更长的心跳间隔
		EnableCompression: true,             // 启用压缩
		BatchSize:         1,                // 减少批处理大小
	}

	client.SetConfig(weakConfig)
	fmt.Printf("应用弱网配置: %s\n", weakConfig.String())

	// 测试服务调用
	fmt.Println("\n=== 服务调用测试 ===")

	// 模拟服务调用
	result, err := client.CallService("example-service", "GET", "/health", nil, nil)
	if err != nil {
		log.Printf("服务调用失败: %v", err)
	} else {
		fmt.Printf("服务调用成功: HTTP %d, 延迟 %v\n",
			result.StatusCode, result.Latency)
	}

	// 显示最终统计
	fmt.Println("\n=== 最终网络统计 ===")
	stats := client.GetNetworkStats()
	fmt.Printf("总样本数: %d\n", stats.SampleCount)
	fmt.Printf("平均延迟: %v\n", stats.Latency)
	fmt.Printf("成功率: %.2f%%\n", stats.SuccessRate*100)
	fmt.Printf("错误率: %.2f%%\n", stats.ErrorRate*100)
	fmt.Printf("超时率: %.2f%%\n", stats.TimeoutRate*100)
	fmt.Printf("最后更新: %v\n", stats.LastUpdated)

	fmt.Println("\n弱网环境支持示例完成")
}
