package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type PerformanceTest struct {
	ControllerURL string
	Concurrency   int
	Duration      time.Duration
	Results       []TestResult
}

type TestResult struct {
	NodeID       string
	StartTime    time.Time
	EndTime      time.Time
	SuccessCount int
	ErrorCount   int
	AvgLatency   time.Duration
	MaxLatency   time.Duration
	MinLatency   time.Duration
	Errors       []string
}

type NodeHello struct {
	NodeID string            `json:"nodeId"`
	Addr   string            `json:"addr"`
	Labels map[string]string `json:"labels"`
}

type LeaseAck struct {
	TTL int `json:"ttl"`
}

func main() {
	test := &PerformanceTest{
		ControllerURL: "http://localhost:8080",
		Concurrency:   50,              // 50个并发节点
		Duration:      5 * time.Minute, // 测试5分钟
	}

	fmt.Printf("开始性能测试：%d个并发节点，持续%v\n", test.Concurrency, test.Duration)

	// 启动测试
	test.RunPerformanceTest()

	// 分析结果
	test.AnalyzeResults()
}

func (pt *PerformanceTest) RunPerformanceTest() {
	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < pt.Concurrency; i++ {
		wg.Add(1)
		go func(nodeID int) {
			defer wg.Done()
			pt.runNodeTest(nodeID, startTime)
		}(i)
	}

	wg.Wait()
}

func (pt *PerformanceTest) runNodeTest(nodeID int, startTime time.Time) {
	nodeIDStr := fmt.Sprintf("test-node-%d", nodeID)
	result := TestResult{
		NodeID:     nodeIDStr,
		StartTime:  time.Now(),
		MinLatency: time.Hour, // 初始化为很大的值
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 模拟节点心跳
	hello := NodeHello{
		NodeID: nodeIDStr,
		Addr:   fmt.Sprintf("192.168.1.%d:8080", 100+nodeID),
		Labels: map[string]string{
			"test": "true",
			"node": fmt.Sprintf("%d", nodeID),
		},
	}

	testEndTime := startTime.Add(pt.Duration)

	for time.Now().Before(testEndTime) {
		latency, err := pt.sendHeartbeat(client, hello)
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

		// 模拟心跳间隔
		time.Sleep(1 * time.Second)
	}

	result.EndTime = time.Now()
	if result.SuccessCount > 0 {
		result.AvgLatency = result.AvgLatency / time.Duration(result.SuccessCount)
	}

	pt.Results = append(pt.Results, result)
}

func (pt *PerformanceTest) sendHeartbeat(client *http.Client, hello NodeHello) (time.Duration, error) {
	start := time.Now()

	jsonData, err := json.Marshal(hello)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", pt.ControllerURL+"/v1/nodes/heartbeat", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return latency, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var ack LeaseAck
	if err := json.NewDecoder(resp.Body).Decode(&ack); err != nil {
		return latency, err
	}

	return latency, nil
}

func (pt *PerformanceTest) AnalyzeResults() {
	fmt.Println("\n=== 性能测试结果分析 ===")

	totalSuccess := 0
	totalErrors := 0
	totalLatency := time.Duration(0)
	maxLatency := time.Duration(0)
	minLatency := time.Hour

	successfulNodes := 0

	for _, result := range pt.Results {
		totalSuccess += result.SuccessCount
		totalErrors += result.ErrorCount

		if result.SuccessCount > 0 {
			successfulNodes++
			totalLatency += result.AvgLatency * time.Duration(result.SuccessCount)

			if result.MaxLatency > maxLatency {
				maxLatency = result.MaxLatency
			}
			if result.MinLatency < minLatency {
				minLatency = result.MinLatency
			}
		}
	}

	avgLatency := time.Duration(0)
	if totalSuccess > 0 {
		avgLatency = totalLatency / time.Duration(totalSuccess)
	}

	fmt.Printf("测试节点数: %d\n", len(pt.Results))
	fmt.Printf("成功节点数: %d\n", successfulNodes)
	fmt.Printf("总成功请求: %d\n", totalSuccess)
	fmt.Printf("总错误请求: %d\n", totalErrors)
	fmt.Printf("成功率: %.2f%%\n", float64(totalSuccess)/float64(totalSuccess+totalErrors)*100)
	fmt.Printf("平均延迟: %v\n", avgLatency)
	fmt.Printf("最大延迟: %v\n", maxLatency)
	fmt.Printf("最小延迟: %v\n", minLatency)

	// 延迟分布分析
	latencyBuckets := map[string]int{
		"<100ms":    0,
		"100-500ms": 0,
		"500ms-1s":  0,
		"1-2s":      0,
		">2s":       0,
	}

	for _, result := range pt.Results {
		if result.SuccessCount > 0 {
			avg := result.AvgLatency
			switch {
			case avg < 100*time.Millisecond:
				latencyBuckets["<100ms"]++
			case avg < 500*time.Millisecond:
				latencyBuckets["100-500ms"]++
			case avg < 1*time.Second:
				latencyBuckets["500ms-1s"]++
			case avg < 2*time.Second:
				latencyBuckets["1-2s"]++
			default:
				latencyBuckets[">2s"]++
			}
		}
	}

	fmt.Println("\n延迟分布:")
	for bucket, count := range latencyBuckets {
		fmt.Printf("  %s: %d个节点\n", bucket, count)
	}

	// 错误分析
	if totalErrors > 0 {
		fmt.Println("\n错误分析:")
		errorTypes := make(map[string]int)
		for _, result := range pt.Results {
			for _, err := range result.Errors {
				errorTypes[err]++
			}
		}
		for err, count := range errorTypes {
			fmt.Printf("  %s: %d次\n", err, count)
		}
	}

	// 性能评估
	fmt.Println("\n性能评估:")
	if successfulNodes >= 45 { // 90%的节点成功
		fmt.Println("✅ 节点并发能力: 优秀")
	} else if successfulNodes >= 40 { // 80%的节点成功
		fmt.Println("⚠️  节点并发能力: 良好")
	} else {
		fmt.Println("❌ 节点并发能力: 需要优化")
	}

	if avgLatency < 100*time.Millisecond {
		fmt.Println("✅ 响应延迟: 优秀")
	} else if avgLatency < 500*time.Millisecond {
		fmt.Println("⚠️  响应延迟: 良好")
	} else {
		fmt.Println("❌ 响应延迟: 需要优化")
	}

	if float64(totalSuccess)/float64(totalSuccess+totalErrors) > 0.95 {
		fmt.Println("✅ 系统稳定性: 优秀")
	} else if float64(totalSuccess)/float64(totalSuccess+totalErrors) > 0.90 {
		fmt.Println("⚠️  系统稳定性: 良好")
	} else {
		fmt.Println("❌ 系统稳定性: 需要优化")
	}
}
