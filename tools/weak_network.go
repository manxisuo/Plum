package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	plum "github.com/manxisuo/plum/sdk/go"
)

// WeakNetworkSimulator 弱网环境模拟器
type WeakNetworkSimulator struct {
	clients []*plum.PlumClient
	config  *plum.WeakNetworkConfig
}

// NewWeakNetworkSimulator 创建弱网环境模拟器
func NewWeakNetworkSimulator(controllerURL string, clientCount int) *WeakNetworkSimulator {
	clients := make([]*plum.PlumClient, clientCount)

	// 创建弱网环境配置
	config := &plum.WeakNetworkConfig{
		CacheTTL:          2 * time.Minute,
		RetryMaxAttempts:  10,
		RetryBaseDelay:    1 * time.Second,
		RetryMaxDelay:     30 * time.Second,
		RequestTimeout:    60 * time.Second,
		HeartbeatInterval: 30 * time.Second,
		EnableCompression: true,
		BatchSize:         1,
	}

	for i := 0; i < clientCount; i++ {
		client := plum.NewPlumClientWithConfig(controllerURL, config)
		client.StartNetworkMonitoring(2 * time.Second)
		clients[i] = client
	}

	return &WeakNetworkSimulator{
		clients: clients,
		config:  config,
	}
}

// TestResult 测试结果
type TestResult struct {
	ClientID       int
	SuccessCount   int
	ErrorCount     int
	AvgLatency     time.Duration
	MaxLatency     time.Duration
	MinLatency     time.Duration
	NetworkQuality plum.NetworkQuality
	IsWeakNetwork  bool
	Errors         []string
}

// RunWeakNetworkTest 运行弱网环境测试
func (s *WeakNetworkSimulator) RunWeakNetworkTest(duration time.Duration) []TestResult {
	fmt.Printf("开始弱网环境测试：%d个客户端，持续%v\n", len(s.clients), duration)

	var wg sync.WaitGroup
	results := make([]TestResult, len(s.clients))

	startTime := time.Now()

	for i, client := range s.clients {
		wg.Add(1)
		go func(index int, c *plum.PlumClient) {
			defer wg.Done()
			results[index] = s.testClient(index, c, startTime, duration)
		}(i, client)
	}

	wg.Wait()

	return results
}

// testClient 测试单个客户端
func (s *WeakNetworkSimulator) testClient(clientID int, client *plum.PlumClient, startTime time.Time, duration time.Duration) TestResult {
	result := TestResult{
		ClientID:   clientID,
		MinLatency: time.Hour,
		Errors:     make([]string, 0),
	}

	testEndTime := startTime.Add(duration)

	for time.Now().Before(testEndTime) {
		// 模拟服务发现
		latency, err := s.simulateServiceDiscovery(client)
		if err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.SuccessCount++
			result.AvgLatency += latency

			if latency > result.MaxLatency {
				result.MaxLatency = latency
			}
			if latency < result.MinLatency {
				result.MinLatency = latency
			}
		}

		// 更新网络状态
		result.NetworkQuality = client.GetNetworkQuality()
		result.IsWeakNetwork = client.IsWeakNetwork()

		// 模拟网络延迟
		time.Sleep(500 * time.Millisecond)
	}

	if result.SuccessCount > 0 {
		result.AvgLatency = result.AvgLatency / time.Duration(result.SuccessCount)
	}

	return result
}

// simulateServiceDiscovery 模拟服务发现
func (s *WeakNetworkSimulator) simulateServiceDiscovery(client *plum.PlumClient) (time.Duration, error) {
	start := time.Now()

	// 尝试发现服务
	_, err := client.DiscoverService("test-service", "", "")
	if err != nil {
		return time.Since(start), err
	}

	return time.Since(start), nil
}

// simulateNetworkInstability 模拟网络不稳定
func (s *WeakNetworkSimulator) simulateNetworkInstability() {
	// 这里可以添加网络不稳定的模拟逻辑
	// 比如随机延迟、丢包等
}

// AnalyzeResults 分析测试结果
func (s *WeakNetworkSimulator) AnalyzeResults(results []TestResult) {
	fmt.Println("\n=== 弱网环境测试结果分析 ===")

	totalSuccess := 0
	totalErrors := 0
	totalLatency := time.Duration(0)
	maxLatency := time.Duration(0)
	minLatency := time.Hour

	weakNetworkClients := 0
	excellentQualityClients := 0
	goodQualityClients := 0
	fairQualityClients := 0
	poorQualityClients := 0
	veryPoorQualityClients := 0

	for _, result := range results {
		totalSuccess += result.SuccessCount
		totalErrors += result.ErrorCount

		if result.SuccessCount > 0 {
			totalLatency += result.AvgLatency * time.Duration(result.SuccessCount)

			if result.MaxLatency > maxLatency {
				maxLatency = result.MaxLatency
			}
			if result.MinLatency < minLatency {
				minLatency = result.MinLatency
			}
		}

		// 统计网络质量分布
		switch result.NetworkQuality {
		case plum.NetworkQualityExcellent:
			excellentQualityClients++
		case plum.NetworkQualityGood:
			goodQualityClients++
		case plum.NetworkQualityFair:
			fairQualityClients++
		case plum.NetworkQualityPoor:
			poorQualityClients++
		case plum.NetworkQualityVeryPoor:
			veryPoorQualityClients++
		}

		if result.IsWeakNetwork {
			weakNetworkClients++
		}
	}

	avgLatency := time.Duration(0)
	if totalSuccess > 0 {
		avgLatency = totalLatency / time.Duration(totalSuccess)
	}

	successRate := float64(totalSuccess) / float64(totalSuccess+totalErrors) * 100

	fmt.Printf("测试客户端数: %d\n", len(results))
	fmt.Printf("总成功请求: %d\n", totalSuccess)
	fmt.Printf("总错误请求: %d\n", totalErrors)
	fmt.Printf("成功率: %.2f%%\n", successRate)
	fmt.Printf("平均延迟: %v\n", avgLatency)
	fmt.Printf("最大延迟: %v\n", maxLatency)
	fmt.Printf("最小延迟: %v\n", minLatency)

	fmt.Println("\n网络质量分布:")
	fmt.Printf("  优秀: %d个客户端\n", excellentQualityClients)
	fmt.Printf("  良好: %d个客户端\n", goodQualityClients)
	fmt.Printf("  一般: %d个客户端\n", fairQualityClients)
	fmt.Printf("  差: %d个客户端\n", poorQualityClients)
	fmt.Printf("  很差: %d个客户端\n", veryPoorQualityClients)
	fmt.Printf("  弱网环境: %d个客户端\n", weakNetworkClients)

	// 性能评估
	fmt.Println("\n弱网环境适应性评估:")

	if successRate > 90 {
		fmt.Println("✅ 弱网环境适应性: 优秀")
	} else if successRate > 80 {
		fmt.Println("⚠️  弱网环境适应性: 良好")
	} else if successRate > 70 {
		fmt.Println("⚠️  弱网环境适应性: 一般")
	} else {
		fmt.Println("❌ 弱网环境适应性: 需要优化")
	}

	if avgLatency < 2*time.Second {
		fmt.Println("✅ 弱网环境延迟: 优秀")
	} else if avgLatency < 5*time.Second {
		fmt.Println("⚠️  弱网环境延迟: 良好")
	} else if avgLatency < 10*time.Second {
		fmt.Println("⚠️  弱网环境延迟: 一般")
	} else {
		fmt.Println("❌ 弱网环境延迟: 需要优化")
	}

	if weakNetworkClients == 0 {
		fmt.Println("✅ 网络质量: 所有客户端网络质量良好")
	} else {
		fmt.Printf("⚠️  网络质量: %d个客户端处于弱网环境\n", weakNetworkClients)
	}
}

// Stop 停止模拟器
func (s *WeakNetworkSimulator) Stop() {
	for _, client := range s.clients {
		client.StopNetworkMonitoring()
	}
}

func main() {
	// 检查Controller是否运行
	controllerURL := "http://localhost:8080"
	resp, err := http.Get(controllerURL + "/healthz")
	if err != nil || resp.StatusCode != 200 {
		fmt.Println("❌ Controller未运行，请先启动Controller")
		fmt.Println("运行: make controller-run")
		return
	}
	resp.Body.Close()
	fmt.Println("✅ Controller运行正常")

	// 创建弱网环境模拟器
	simulator := NewWeakNetworkSimulator(controllerURL, 20) // 20个客户端
	defer simulator.Stop()

	// 运行测试
	results := simulator.RunWeakNetworkTest(2 * time.Minute) // 测试2分钟

	// 分析结果
	simulator.AnalyzeResults(results)

	fmt.Println("\n弱网环境测试完成")
}
