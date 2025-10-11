package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/manxisuo/plum/controller/internal/notify"
	"github.com/manxisuo/plum/controller/internal/store"
)

// KV API Models
type KVPutRequest struct {
	Value string `json:"value"`
	Type  string `json:"type"` // string|int|double|bool
}

type KVPutBatchRequest struct {
	Items []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Type  string `json:"type"`
	} `json:"items"`
}

type KVDTO struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Type      string `json:"type"`
	UpdatedAt int64  `json:"updatedAt"`
}

// PUT /v1/kv/{namespace}/{key}
// GET /v1/kv/{namespace}/{key}
// DELETE /v1/kv/{namespace}/{key}
func handleKVByKey(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/kv/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	namespace := parts[0]
	key := parts[1]

	if namespace == "" || key == "" {
		http.Error(w, "namespace and key required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		handleKVPut(w, r, namespace, key)
	case http.MethodGet:
		handleKVGet(w, r, namespace, key)
	case http.MethodDelete:
		handleKVDelete(w, r, namespace, key)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleKVPut(w http.ResponseWriter, r *http.Request, namespace, key string) {
	var req KVPutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// 默认类型为string
	if req.Type == "" {
		req.Type = "string"
	}

	// 验证类型
	if req.Type != "string" && req.Type != "int" && req.Type != "double" && req.Type != "bool" {
		http.Error(w, "invalid type (must be string|int|double|bool)", http.StatusBadRequest)
		return
	}

	if err := store.Current.PutKV(namespace, key, req.Value, req.Type); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	// 发送SSE通知
	notify.PublishKV(namespace, key, req.Value, req.Type)

	writeJSON(w, map[string]any{
		"namespace": namespace,
		"key":       key,
		"value":     req.Value,
		"type":      req.Type,
	})
}

func handleKVGet(w http.ResponseWriter, r *http.Request, namespace, key string) {
	kv, ok, err := store.Current.GetKV(namespace, key)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, KVDTO{
		Namespace: kv.Namespace,
		Key:       kv.Key,
		Value:     kv.Value,
		Type:      kv.Type,
		UpdatedAt: kv.UpdatedAt,
	})
}

func handleKVDelete(w http.ResponseWriter, r *http.Request, namespace, key string) {
	if err := store.Current.DeleteKV(namespace, key); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	// 发送SSE删除通知
	notify.PublishKV(namespace, key, "", "deleted")

	w.WriteHeader(http.StatusNoContent)
}

// GET /v1/kv/{namespace}
func handleKVByNamespace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := strings.TrimPrefix(r.URL.Path, "/v1/kv/")
	if namespace == "" || strings.Contains(namespace, "/") {
		http.Error(w, "invalid namespace", http.StatusBadRequest)
		return
	}

	// 支持前缀查询
	prefix := r.URL.Query().Get("prefix")
	
	var kvs []store.DistributedKV
	var err error
	
	if prefix != "" {
		kvs, err = store.Current.ListKVByPrefix(namespace, prefix)
	} else {
		kvs, err = store.Current.ListKVByNamespace(namespace)
	}
	
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	out := make([]KVDTO, 0, len(kvs))
	for _, kv := range kvs {
		out = append(out, KVDTO{
			Namespace: kv.Namespace,
			Key:       kv.Key,
			Value:     kv.Value,
			Type:      kv.Type,
			UpdatedAt: kv.UpdatedAt,
		})
	}

	writeJSON(w, out)
}

// POST /v1/kv/{namespace}/batch
func handleKVBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/kv/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "batch" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	namespace := parts[0]

	var req KVPutBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, "items required", http.StatusBadRequest)
		return
	}

	kvs := make([]store.DistributedKV, 0, len(req.Items))
	for _, item := range req.Items {
		if item.Key == "" {
			http.Error(w, "key required", http.StatusBadRequest)
			return
		}
		t := item.Type
		if t == "" {
			t = "string"
		}
		kvs = append(kvs, store.DistributedKV{
			Key:   item.Key,
			Value: item.Value,
			Type:  t,
		})
	}

	if err := store.Current.PutKVBatch(namespace, kvs); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	// 批量通知
	for _, kv := range kvs {
		notify.PublishKV(namespace, kv.Key, kv.Value, kv.Type)
	}

	writeJSON(w, map[string]any{
		"namespace": namespace,
		"count":     len(kvs),
	})
}

// GET /v1/kv - List all namespaces
func handleKVListNamespaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespaces, err := store.Current.ListAllNamespaces()
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"namespaces": namespaces,
	})
}

// GET /v1/kv/{namespace}/keys - List keys in namespace (no values)
func handleKVListKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/kv/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "keys" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	namespace := parts[0]

	if namespace == "" {
		http.Error(w, "namespace required", http.StatusBadRequest)
		return
	}

	keys, err := store.Current.ListKeysByNamespace(namespace)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"namespace": namespace,
		"keys":      keys,
	})
}

// Helper functions for type conversion
func ParseInt(s string, defaultVal int64) int64 {
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v
	}
	return defaultVal
}

func ParseDouble(s string, defaultVal float64) float64 {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	return defaultVal
}

func ParseBool(s string, defaultVal bool) bool {
	if v, err := strconv.ParseBool(s); err == nil {
		return v
	}
	return defaultVal
}

