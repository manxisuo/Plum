package plum

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

// PlumClient Plum服务客户端
type PlumClient struct {
	controllerURL  string
	httpClient     *RetryableHTTPClient
	serviceCache   *SmartCache
	networkMonitor *NetworkMonitor
	config         *WeakNetworkConfig
	adaptiveMode   bool
}

// Endpoint 服务端点
type Endpoint struct {
	ServiceName string            `json:"serviceName"`
	InstanceID  string            `json:"instanceId"`
	NodeID      string            `json:"nodeId"`
	IP          string            `json:"ip"`
	Port        int               `json:"port"`
	Protocol    string            `json:"protocol"`
	Version     string            `json:"version"`
	Labels      map[string]string `json:"labels"`
	Healthy     bool              `json:"healthy"`
	LastSeen    int64             `json:"lastSeen"`
}

// ServiceCallResult 服务调用结果
type ServiceCallResult struct {
	StatusCode int
	Body       []byte
	Latency    time.Duration
	Error      error
}

// NewPlumClient 创建新的Plum客户端
func NewPlumClient(controllerURL string) *PlumClient {
	// 创建网络监控器
	monitor := NewNetworkMonitor(controllerURL)

	// 创建默认配置
	config := &WeakNetworkConfig{
		CacheTTL:          30 * time.Second,
		RetryMaxAttempts:  3,
		RetryBaseDelay:    100 * time.Millisecond,
		RetryMaxDelay:     5 * time.Second,
		RequestTimeout:    30 * time.Second,
		HeartbeatInterval: 5 * time.Second,
		EnableCompression: false,
		BatchSize:         1,
	}

	// 创建重试策略
	strategy := NewExponentialBackoffStrategy(
		config.RetryBaseDelay,
		config.RetryMaxDelay,
		config.RetryMaxAttempts,
	)

	// 创建HTTP客户端
	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
	}
	retryClient := NewRetryableHTTPClient(httpClient, strategy)

	// 创建智能缓存
	cache := NewSmartCache(config.CacheTTL)

	return &PlumClient{
		controllerURL:  controllerURL,
		httpClient:     retryClient,
		serviceCache:   cache,
		networkMonitor: monitor,
		config:         config,
		adaptiveMode:   true, // 默认启用自适应模式
	}
}

// NewPlumClientWithConfig 使用自定义配置创建Plum客户端
func NewPlumClientWithConfig(controllerURL string, config *WeakNetworkConfig) *PlumClient {
	monitor := NewNetworkMonitor(controllerURL)

	strategy := NewExponentialBackoffStrategy(
		config.RetryBaseDelay,
		config.RetryMaxDelay,
		config.RetryMaxAttempts,
	)

	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
	}
	retryClient := NewRetryableHTTPClient(httpClient, strategy)
	cache := NewSmartCache(config.CacheTTL)

	return &PlumClient{
		controllerURL:  controllerURL,
		httpClient:     retryClient,
		serviceCache:   cache,
		networkMonitor: monitor,
		config:         config,
		adaptiveMode:   true,
	}
}

// DiscoverService 发现服务端点
func (c *PlumClient) DiscoverService(serviceName string, version, protocol string) ([]Endpoint, error) {
	// 构建缓存键
	cacheKey := fmt.Sprintf("service:%s:%s:%s", serviceName, version, protocol)

	// 检查缓存
	if data, exists := c.serviceCache.Get(cacheKey); exists {
		if endpoints, ok := data.([]Endpoint); ok {
			return endpoints, nil
		}
	}

	// 构建查询参数
	url := fmt.Sprintf("%s/v1/discovery?service=%s", c.controllerURL, serviceName)
	if version != "" {
		url += "&version=" + version
	}
	if protocol != "" {
		url += "&protocol=" + protocol
	}

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 发送请求（带重试）
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discovery failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var endpoints []Endpoint
	if err := json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
		return nil, err
	}

	// 更新缓存
	c.serviceCache.Set(cacheKey, endpoints)

	return endpoints, nil
}

// DiscoverServiceRandom 随机发现一个服务端点
func (c *PlumClient) DiscoverServiceRandom(serviceName string, version, protocol string) (*Endpoint, error) {
	// 构建查询参数
	url := fmt.Sprintf("%s/v1/discovery/random?service=%s", c.controllerURL, serviceName)
	if version != "" {
		url += "&version=" + version
	}
	if protocol != "" {
		url += "&protocol=" + protocol
	}

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 发送请求（带重试）
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no endpoints found for service: %s", serviceName)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discovery failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var endpoint Endpoint
	if err := json.NewDecoder(resp.Body).Decode(&endpoint); err != nil {
		return nil, err
	}

	return &endpoint, nil
}

// StartNetworkMonitoring 开始网络监控
func (c *PlumClient) StartNetworkMonitoring(interval time.Duration) {
	c.networkMonitor.Start(interval)
}

// StopNetworkMonitoring 停止网络监控
func (c *PlumClient) StopNetworkMonitoring() {
	c.networkMonitor.Stop()
}

// GetNetworkQuality 获取网络质量
func (c *PlumClient) GetNetworkQuality() NetworkQuality {
	return c.networkMonitor.GetQuality()
}

// IsWeakNetwork 判断是否为弱网环境
func (c *PlumClient) IsWeakNetwork() bool {
	return c.networkMonitor.IsWeakNetwork()
}

// GetNetworkStats 获取网络统计
func (c *PlumClient) GetNetworkStats() NetworkStats {
	return c.networkMonitor.GetStats()
}

// EnableAdaptiveMode 启用自适应模式
func (c *PlumClient) EnableAdaptiveMode() {
	c.adaptiveMode = true
	c.adaptToNetworkConditions()
}

// DisableAdaptiveMode 禁用自适应模式
func (c *PlumClient) DisableAdaptiveMode() {
	c.adaptiveMode = false
}

// adaptToNetworkConditions 根据网络条件自适应调整配置
func (c *PlumClient) adaptToNetworkConditions() {
	if !c.adaptiveMode {
		return
	}

	recommendedConfig := c.networkMonitor.GetRecommendedConfig()

	// 更新配置
	c.config = recommendedConfig

	// 更新缓存TTL
	c.serviceCache = NewSmartCache(recommendedConfig.CacheTTL)

	// 更新重试策略
	strategy := NewExponentialBackoffStrategy(
		recommendedConfig.RetryBaseDelay,
		recommendedConfig.RetryMaxDelay,
		recommendedConfig.RetryMaxAttempts,
	)
	c.httpClient.SetStrategy(strategy)

	// 更新HTTP客户端超时
	if c.httpClient.client != nil {
		c.httpClient.client.Timeout = recommendedConfig.RequestTimeout
	}
}

// GetConfig 获取当前配置
func (c *PlumClient) GetConfig() *WeakNetworkConfig {
	return c.config
}

// SetConfig 设置配置
func (c *PlumClient) SetConfig(config *WeakNetworkConfig) {
	c.config = config

	// 更新缓存
	c.serviceCache = NewSmartCache(config.CacheTTL)

	// 更新重试策略
	strategy := NewExponentialBackoffStrategy(
		config.RetryBaseDelay,
		config.RetryMaxDelay,
		config.RetryMaxAttempts,
	)
	c.httpClient.SetStrategy(strategy)

	// 更新HTTP客户端超时
	if c.httpClient.client != nil {
		c.httpClient.client.Timeout = config.RequestTimeout
	}
}

// CallService 调用服务
func (c *PlumClient) CallService(serviceName, method, path string, headers map[string]string, body []byte) (*ServiceCallResult, error) {
	// 随机选择一个端点
	endpoint, err := c.DiscoverServiceRandom(serviceName, "", "")
	if err != nil {
		return nil, err
	}

	// 构建URL
	url := fmt.Sprintf("%s://%s:%d%s", endpoint.Protocol, endpoint.IP, endpoint.Port, path)

	// 创建请求
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	// 设置默认headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Plum-Go-SDK/1.0")

	// 设置自定义headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 记录开始时间
	start := time.Now()

	// 发送请求
	resp, err := c.httpClient.Do(req)
	latency := time.Since(start)

	result := &ServiceCallResult{
		Latency: latency,
		Error:   err,
	}

	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = err
		return result, err
	}

	result.StatusCode = resp.StatusCode
	result.Body = responseBody

	return result, nil
}

// CallServiceWithRetry 带重试的服务调用
func (c *PlumClient) CallServiceWithRetry(serviceName, method, path string, headers map[string]string, body []byte, maxRetries int) (*ServiceCallResult, error) {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		result, err := c.CallService(serviceName, method, path, headers, body)
		if err == nil && result.StatusCode < 500 {
			return result, nil
		}

		lastErr = err
		if result != nil {
			lastErr = fmt.Errorf("HTTP %d: %s", result.StatusCode, string(result.Body))
		}

		// 指数退避
		if i < maxRetries {
			backoff := time.Duration(1<<uint(i)) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("service call failed after %d retries: %v", maxRetries, lastErr)
}

// LoadBalance 负载均衡调用
func (c *PlumClient) LoadBalance(serviceName, method, path string, headers map[string]string, body []byte, strategy string) (*ServiceCallResult, error) {
	endpoints, err := c.DiscoverService(serviceName, "", "")
	if err != nil {
		return nil, err
	}

	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints available for service: %s", serviceName)
	}

	// 选择端点
	var selected *Endpoint
	switch strategy {
	case "random":
		selected = &endpoints[rand.Intn(len(endpoints))]
	case "round_robin":
		// 简单的轮询实现
		selected = &endpoints[time.Now().UnixNano()%int64(len(endpoints))]
	default:
		selected = &endpoints[0] // 默认选择第一个
	}

	// 构建URL
	url := fmt.Sprintf("%s://%s:%d%s", selected.Protocol, selected.IP, selected.Port, path)

	// 创建请求
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	// 设置headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Plum-Go-SDK/1.0")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	start := time.Now()
	resp, err := c.httpClient.Do(req)
	latency := time.Since(start)

	result := &ServiceCallResult{
		Latency: latency,
		Error:   err,
	}

	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = err
		return result, err
	}

	result.StatusCode = resp.StatusCode
	result.Body = responseBody

	return result, nil
}

// RegisterService 注册服务
func (c *PlumClient) RegisterService(instanceID string, serviceName, version, protocol, host string, port int, labels map[string]string) error {
	endpoint := Endpoint{
		ServiceName: serviceName,
		InstanceID:  instanceID,
		IP:          host,
		Port:        port,
		Protocol:    protocol,
		Version:     version,
		Labels:      labels,
		Healthy:     true,
		LastSeen:    time.Now().Unix(),
	}

	registerReq := map[string]interface{}{
		"instanceId": instanceID,
		"endpoints":  []Endpoint{endpoint},
	}

	jsonData, err := json.Marshal(registerReq)
	if err != nil {
		return err
	}

	url := c.controllerURL + "/v1/services/register"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("service registration failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// Heartbeat 发送心跳
func (c *PlumClient) Heartbeat(instanceID string, endpoints []Endpoint) error {
	heartbeatReq := map[string]interface{}{
		"instanceId": instanceID,
		"endpoints":  endpoints,
	}

	jsonData, err := json.Marshal(heartbeatReq)
	if err != nil {
		return err
	}

	url := c.controllerURL + "/v1/services/heartbeat"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// ClearCache 清除服务缓存
func (c *PlumClient) ClearCache() {
	c.serviceCache.Clear()
}
