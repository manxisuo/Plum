package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/manxisuo/plum/controller/internal/store"
)

type EndpointDTO struct {
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

type RegisterRequest struct {
	InstanceID string        `json:"instanceId"`
	NodeID     string        `json:"nodeId"`
	IP         string        `json:"ip"`
	Endpoints  []EndpointDTO `json:"endpoints"`
}

type HeartbeatRequest struct {
	InstanceID string        `json:"instanceId"`
	Health     []EndpointDTO `json:"health"` // allow health override per endpoint
}

var discoveryCache = struct {
	sync.RWMutex
	entries map[string]store.Endpoint
}{
	entries: make(map[string]store.Endpoint),
}

func endpointToDTO(e store.Endpoint) EndpointDTO {
	return EndpointDTO{
		ServiceName: e.ServiceName,
		InstanceID:  e.InstanceID,
		NodeID:      e.NodeID,
		IP:          e.IP,
		Port:        e.Port,
		Protocol:    e.Protocol,
		Version:     e.Version,
		Labels:      e.Labels,
		Healthy:     e.Healthy,
		LastSeen:    e.LastSeen,
	}
}

func endpointsEqual(a, b store.Endpoint) bool {
	return a.ServiceName == b.ServiceName &&
		a.InstanceID == b.InstanceID &&
		a.IP == b.IP &&
		a.Port == b.Port &&
		a.Protocol == b.Protocol
}

func discoveryCacheKey(service, version, protocol string) string {
	return service + "|" + version + "|" + protocol
}

func selectRandomEndpoint(eps []store.Endpoint) store.Endpoint {
	rand.Seed(time.Now().UnixNano())
	return eps[rand.Intn(len(eps))]
}

func handleRegisterEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.InstanceID == "" {
		http.Error(w, "missing instanceId", http.StatusBadRequest)
		return
	}

	// nodeID可以为空（手动注册可以使用默认值）
	if req.NodeID == "" {
		req.NodeID = "manual"
	}

	// 统一使用增量注册模式：每个端点独立注册，互不影响
	// 这样可以：
	// 1. 支持在已有实例上手动注册额外服务，不会被Agent覆盖
	// 2. Agent只管理meta.ini中定义的服务，不影响其他服务
	// 3. 如果同一个端点（主键相同）重复注册，会自动更新（INSERT OR REPLACE）
	now := time.Now().Unix()

	// 判断是否是手动注册：
	// 手动注册时（没有 replace=true 参数）需要立即进行健康检查
	isManualRegistration := r.URL.Query().Get("replace") != "true"

	for _, e := range req.Endpoints {
		ep := store.Endpoint{
			ServiceName: e.ServiceName,
			InstanceID:  req.InstanceID,
			NodeID:      req.NodeID,
			IP:          req.IP,
			Port:        e.Port,
			Protocol:    e.Protocol,
			Version:     e.Version,
			Labels:      e.Labels,
			Healthy:     true, // 默认健康，如果是手动注册会在下面检查
			LastSeen:    now,
		}

		// 对于手动注册的端点，立即进行健康检查
		if isManualRegistration {
			isHealthy := checkEndpointHealth(ep.IP, ep.Port, ep.Protocol)
			ep.Healthy = isHealthy

			if !isHealthy {
				log.Printf("service register health check failed: service=%s, ip=%s, port=%d",
					ep.ServiceName, ep.IP, ep.Port)
			}
		}

		if err := store.Current.AddEndpoint(ep); err != nil {
			log.Printf("service register failed to add endpoint: %v", err)
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleHeartbeatEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.InstanceID == "" {
		http.Error(w, "missing instanceId", http.StatusBadRequest)
		return
	}
	// if health overrides provided, update
	if len(req.Health) > 0 {
		eps := make([]store.Endpoint, 0, len(req.Health))
		for _, e := range req.Health {
			eps = append(eps, store.Endpoint{ServiceName: e.ServiceName, InstanceID: req.InstanceID, IP: e.IP, Port: e.Port, Protocol: e.Protocol, Healthy: e.Healthy})
		}
		if err := store.Current.UpdateEndpointHealthForInstance(req.InstanceID, eps); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
	} else {
		// 如果没有提供健康覆盖，自动检查并更新健康状态
		// 这样可以处理注册时服务未启动的情况
		// 通过查询所有服务来找到该实例的端点（临时方案）
		allServices, err := store.Current.ListServices()
		if err == nil {
			updatedEps := make([]store.Endpoint, 0)
			for _, serviceName := range allServices {
				// 获取该服务的所有端点（包括不健康的）
				eps, err := store.Current.ListAllEndpointsByService(serviceName)
				if err == nil {
					for _, ep := range eps {
						if ep.InstanceID == req.InstanceID {
							// 对每个端点进行健康检查
							isHealthy := checkEndpointHealth(ep.IP, ep.Port, ep.Protocol)
							updatedEps = append(updatedEps, store.Endpoint{
								ServiceName: ep.ServiceName,
								InstanceID:  ep.InstanceID,
								IP:          ep.IP,
								Port:        ep.Port,
								Protocol:    ep.Protocol,
								Healthy:     isHealthy,
							})
						}
					}
				}
			}
			if len(updatedEps) > 0 {
				if err := store.Current.UpdateEndpointHealthForInstance(req.InstanceID, updatedEps); err != nil {
					// 如果更新失败，只更新 LastSeen
					if err := store.Current.TouchEndpointsForInstance(req.InstanceID, time.Now().Unix()); err != nil {
						http.Error(w, "db error", http.StatusInternalServerError)
						return
					}
				}
			} else {
				// 如果没有找到端点，只更新 LastSeen
				if err := store.Current.TouchEndpointsForInstance(req.InstanceID, time.Now().Unix()); err != nil {
					http.Error(w, "db error", http.StatusInternalServerError)
					return
				}
			}
		} else {
			// 如果无法获取服务列表，只更新 LastSeen
			if err := store.Current.TouchEndpointsForInstance(req.InstanceID, time.Now().Unix()); err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleDeleteEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	iid := r.URL.Query().Get("instanceId")
	if iid == "" {
		http.Error(w, "instanceId required", http.StatusBadRequest)
		return
	}
	_ = store.Current.DeleteEndpointsForInstance(iid)
	w.WriteHeader(http.StatusNoContent)
}

func handleDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	service := r.URL.Query().Get("service")
	if service == "" {
		http.Error(w, "service required", http.StatusBadRequest)
		return
	}

	// 如果请求参数包含 all=true，返回所有端点（包括不健康的）
	if r.URL.Query().Get("all") == "true" {
		eps, err := store.Current.ListAllEndpointsByService(service)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		out := make([]EndpointDTO, 0, len(eps))
		for _, e := range eps {
			out = append(out, EndpointDTO{ServiceName: e.ServiceName, InstanceID: e.InstanceID, NodeID: e.NodeID, IP: e.IP, Port: e.Port, Protocol: e.Protocol, Version: e.Version, Labels: e.Labels, Healthy: e.Healthy, LastSeen: e.LastSeen})
		}
		writeJSON(w, out)
		return
	}

	// 默认只返回健康的端点（服务发现场景）
	version := r.URL.Query().Get("version")
	protocol := r.URL.Query().Get("protocol")
	eps, err := store.Current.ListEndpointsByService(service, version, protocol)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	out := make([]EndpointDTO, 0, len(eps))
	for _, e := range eps {
		out = append(out, EndpointDTO{ServiceName: e.ServiceName, InstanceID: e.InstanceID, NodeID: e.NodeID, IP: e.IP, Port: e.Port, Protocol: e.Protocol, Version: e.Version, Labels: e.Labels, Healthy: e.Healthy, LastSeen: e.LastSeen})
	}
	// optional: max endpoints
	if lim := r.URL.Query().Get("limit"); lim != "" {
		if n, err := strconv.Atoi(lim); err == nil && n < len(out) {
			out = out[:n]
		}
	}
	writeJSON(w, out)
}

func handleDiscoverRandom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	service := r.URL.Query().Get("service")
	if service == "" {
		http.Error(w, "service required", http.StatusBadRequest)
		return
	}
	version := r.URL.Query().Get("version")
	protocol := r.URL.Query().Get("protocol")
	eps, err := store.Current.ListEndpointsByService(service, version, protocol)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if len(eps) == 0 {
		http.Error(w, "no endpoints found", http.StatusNotFound)
		return
	}
	selected := selectRandomEndpoint(eps)
	writeJSON(w, endpointToDTO(selected))
}

func handleDiscoverOne(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	service := r.URL.Query().Get("service")
	if service == "" {
		http.Error(w, "service required", http.StatusBadRequest)
		return
	}
	version := r.URL.Query().Get("version")
	protocol := r.URL.Query().Get("protocol")
	strategy := strings.ToLower(r.URL.Query().Get("strategy"))
	if strategy == "" {
		strategy = "random"
	}

	eps, err := store.Current.ListEndpointsByService(service, version, protocol)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if len(eps) == 0 {
		http.Error(w, "no endpoints found", http.StatusNotFound)
		return
	}

	var selected store.Endpoint

	switch strategy {
	case "random":
		selected = selectRandomEndpoint(eps)
	case "lazy":
		key := discoveryCacheKey(service, version, protocol)

		var cached store.Endpoint
		var ok bool
		discoveryCache.RLock()
		cached, ok = discoveryCache.entries[key]
		discoveryCache.RUnlock()

		found := false
		if ok {
			for _, e := range eps {
				if endpointsEqual(e, cached) {
					selected = e
					found = true
					break
				}
			}
		}
		if !found {
			selected = selectRandomEndpoint(eps)
		}
		discoveryCache.Lock()
		discoveryCache.entries[key] = selected
		discoveryCache.Unlock()
	default:
		http.Error(w, "invalid strategy", http.StatusBadRequest)
		return
	}

	writeJSON(w, endpointToDTO(selected))
}

func handleListServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	names, err := store.Current.ListServices()
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, names)
}

// 单个端点的CRUD操作（路由分发）
func handleEndpointCRUD(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		handleDeleteEndpoint(w, r)
	case http.MethodPatch, http.MethodPut:
		handleUpdateEndpoint(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// 删除单个端点
func handleDeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	serviceName := r.URL.Query().Get("serviceName")
	instanceID := r.URL.Query().Get("instanceId")
	ip := r.URL.Query().Get("ip")
	portStr := r.URL.Query().Get("port")
	protocol := r.URL.Query().Get("protocol")

	if serviceName == "" || instanceID == "" || ip == "" || portStr == "" || protocol == "" {
		http.Error(w, "missing required parameters: serviceName, instanceId, ip, port, protocol", http.StatusBadRequest)
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		http.Error(w, "invalid port", http.StatusBadRequest)
		return
	}

	if err := store.Current.DeleteEndpoint(serviceName, instanceID, ip, port, protocol); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// 更新单个端点
func handleUpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取旧端点的标识（用于定位要更新的记录）
	serviceName := r.URL.Query().Get("serviceName")
	instanceID := r.URL.Query().Get("instanceId")
	oldIP := r.URL.Query().Get("ip")
	oldPortStr := r.URL.Query().Get("port")
	oldProtocol := r.URL.Query().Get("protocol")

	if serviceName == "" || instanceID == "" || oldIP == "" || oldPortStr == "" || oldProtocol == "" {
		http.Error(w, "missing required query parameters: serviceName, instanceId, ip, port, protocol", http.StatusBadRequest)
		return
	}

	oldPort, err := strconv.Atoi(oldPortStr)
	if err != nil || oldPort <= 0 || oldPort > 65535 {
		http.Error(w, "invalid port", http.StatusBadRequest)
		return
	}

	// 解析请求体（新的端点信息）
	var epDTO EndpointDTO
	if err := json.NewDecoder(r.Body).Decode(&epDTO); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if epDTO.ServiceName == "" || epDTO.InstanceID == "" || epDTO.IP == "" || epDTO.Port <= 0 || epDTO.Port > 65535 || epDTO.Protocol == "" {
		http.Error(w, "missing or invalid fields in request body", http.StatusBadRequest)
		return
	}

	// 转换为store.Endpoint
	ep := store.Endpoint{
		ServiceName: epDTO.ServiceName,
		InstanceID:  epDTO.InstanceID,
		NodeID:      epDTO.NodeID,
		IP:          epDTO.IP,
		Port:        epDTO.Port,
		Protocol:    epDTO.Protocol,
		Version:     epDTO.Version,
		Labels:      epDTO.Labels,
		Healthy:     epDTO.Healthy,
		LastSeen:    time.Now().Unix(),
	}

	if err := store.Current.UpdateEndpoint(serviceName, instanceID, oldIP, oldPort, oldProtocol, ep); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// checkEndpointHealth 检查端点健康状态
// 对于HTTP/HTTPS协议，发送HEAD请求验证；对于其他协议，进行TCP连接检查
func checkEndpointHealth(ip string, port int, protocol string) bool {
	address := net.JoinHostPort(ip, strconv.Itoa(port))

	// 对于HTTP协议，发送HEAD请求进行更严格的检查
	if protocol == "http" || protocol == "https" {
		return checkHTTPHealth(ip, port, protocol)
	}

	// 对于其他协议（gRPC、TCP等），进行TCP连接检查
	return checkTCPHealth(address)
}

// checkHTTPHealth 通过HTTP请求检查端点健康状态
// 支持HEAD、GET请求，兼容POST-only服务（返回405也算健康）
func checkHTTPHealth(ip string, port int, protocol string) bool {
	address := net.JoinHostPort(ip, strconv.Itoa(port))
	url := protocol + "://" + address

	// 创建HTTP客户端，设置超时
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	// 先尝试HEAD请求（最轻量）
	if healthy, _ := tryHTTPRequest(client, "HEAD", url, address); healthy {
		return true
	}

	// 如果HEAD失败（比如返回405 Method Not Allowed），尝试GET请求
	// 这样可以兼容只接受POST的服务（它们通常会拒绝HEAD/GET，但至少证明HTTP服务存在）
	if healthy, _ := tryHTTPRequest(client, "GET", url, address); healthy {
		return true
	}

	// 两个请求都失败，认为服务不存在或不可用
	log.Printf("service health check failed for %s: both HEAD and GET requests rejected", address)
	return false
}

// tryHTTPRequest 尝试发送HTTP请求，返回是否健康及状态码
func tryHTTPRequest(client *http.Client, method, url, address string) (bool, int) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Printf("service health check: failed to create %s request for %s: %v", method, address, err)
		return false, 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) // 缩短单次请求超时
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		// 连接超时、拒绝连接等错误
		log.Printf("service health check: %s request to %s failed: %v", method, address, err)
		return false, 0
	}
	defer resp.Body.Close()

	// 任何HTTP响应都认为服务存在（包括错误码）
	// 常见的状态码：
	// - 200 OK: 服务正常
	// - 404 Not Found: 路径不存在，但服务存在
	// - 405 Method Not Allowed: 方法不支持，但服务存在（比如POST-only服务拒绝GET/HEAD）
	// - 500 Internal Server Error: 服务错误，但服务存在
	// - 其他2xx, 4xx, 5xx: 都说明HTTP服务存在

	// 特殊情况：某些服务可能返回非HTTP响应，这种情况resp.StatusCode可能是0
	// 但如果有响应，至少说明TCP连接成功且可能有某种服务

	return true, resp.StatusCode
}

// checkTCPHealth 通过TCP连接检查端点健康状态
func checkTCPHealth(address string) bool {
	// 设置超时（3秒）
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		log.Printf("service health check failed for %s: %v", address, err)
		return false
	}
	defer conn.Close()

	// TCP连接成功，认为基本健康
	return true
}
