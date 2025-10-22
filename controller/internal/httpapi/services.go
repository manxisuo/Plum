package httpapi

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
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
	if req.InstanceID == "" || req.NodeID == "" {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}
	eps := make([]store.Endpoint, 0, len(req.Endpoints))
	now := time.Now().Unix()
	for _, e := range req.Endpoints {
		eps = append(eps, store.Endpoint{
			ServiceName: e.ServiceName,
			InstanceID:  req.InstanceID,
			NodeID:      req.NodeID,
			IP:          req.IP,
			Port:        e.Port,
			Protocol:    e.Protocol,
			Version:     e.Version,
			Labels:      e.Labels,
			Healthy:     true, // default inherit instance (registered = healthy)
			LastSeen:    now,
		})
	}
	if err := store.Current.ReplaceEndpointsForInstance(req.NodeID, req.InstanceID, eps); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
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
	// 随机选择一个端点
	rand.Seed(time.Now().UnixNano())
	selected := eps[rand.Intn(len(eps))]
	out := EndpointDTO{
		ServiceName: selected.ServiceName,
		InstanceID:  selected.InstanceID,
		NodeID:      selected.NodeID,
		IP:          selected.IP,
		Port:        selected.Port,
		Protocol:    selected.Protocol,
		Version:     selected.Version,
		Labels:      selected.Labels,
		Healthy:     selected.Healthy,
		LastSeen:    selected.LastSeen,
	}
	writeJSON(w, out)
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
