package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/manxisuo/plum/controller/internal/store"
)

// ---- Resources ----

type RegisterResourceRequest struct {
	ResourceID string                    `json:"resourceId"`
	NodeID     string                    `json:"nodeId"`
	Type       string                    `json:"type"`
	URL        string                    `json:"url"`
	StateDesc  []store.ResourceStateDesc `json:"stateDesc"`
	OpDesc     []store.ResourceOpDesc    `json:"opDesc"`
}

type HeartbeatResourceRequest struct {
	ResourceID string `json:"resourceId"`
	NodeID     string `json:"nodeId"`
}

type SubmitResourceStateRequest struct {
	ResourceID string            `json:"resourceId"`
	Timestamp  int64             `json:"timestamp"`
	States     map[string]string `json:"states"`
}

type ResourceOpRequest struct {
	ResourceID string                    `json:"resourceId"`
	Operations []store.ResourceOperation `json:"operations"`
	Timestamp  int64                     `json:"timestamp"`
}

func handleRegisterResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RegisterResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.ResourceID == "" {
		http.Error(w, "resourceId required", http.StatusBadRequest)
		return
	}

	resource := store.Resource{
		ResourceID: req.ResourceID,
		NodeID:     req.NodeID,
		Type:       req.Type,
		URL:        req.URL,
		StateDesc:  req.StateDesc,
		OpDesc:     req.OpDesc,
		LastSeen:   time.Now().Unix(),
		CreatedAt:  time.Now().Unix(),
	}

	err := store.Current.RegisterResource(resource)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleHeartbeatResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req HeartbeatResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.ResourceID == "" {
		http.Error(w, "resourceId required", http.StatusBadRequest)
		return
	}

	if err := store.Current.HeartbeatResource(req.ResourceID, time.Now().Unix()); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleListResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := store.Current.ListResources()
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func handleGetResource(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/v1/resources/"):]
	if id == "" {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet {
		resource, ok, err := store.Current.GetResource(id)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, resource)
	} else if r.Method == http.MethodDelete {
		err := store.Current.DeleteResource(id)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	} else {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSubmitResourceState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req SubmitResourceStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.ResourceID == "" {
		http.Error(w, "resourceId required", http.StatusBadRequest)
		return
	}

	resourceState := store.ResourceState{
		ResourceID: req.ResourceID,
		Timestamp:  req.Timestamp,
		States:     req.States,
	}

	err := store.Current.SubmitResourceState(resourceState)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleListResourceStates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resourceID := r.URL.Query().Get("resourceId")
	if resourceID == "" {
		http.Error(w, "resourceId required", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	list, err := store.Current.ListResourceStates(resourceID, limit)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func handleResourceOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resourceID := r.URL.Query().Get("resourceId")
	if resourceID == "" {
		http.Error(w, "resourceId required", http.StatusBadRequest)
		return
	}

	// Get resource to find its callback URL
	resource, ok, err := store.Current.GetResource(resourceID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}

	var req ResourceOpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Forward operation to resource
	if resource.URL != "" {
		// Forward operation to resource via HTTP
		operationData := map[string]interface{}{
			"operations": req.Operations,
		}

		jsonData, err := json.Marshal(operationData)
		if err != nil {
			http.Error(w, "json marshal error", http.StatusInternalServerError)
			return
		}

		fmt.Printf("[Controller] Preparing to forward operation to %s with data: %s\n", resource.URL, string(jsonData))
		// Send HTTP request to resource (URL already contains the endpoint)
		client := &http.Client{
			Timeout: 5 * time.Second,
		}
		resp, err := client.Post(resource.URL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("[Controller] Failed to forward operation to %s: %v\n", resource.URL, err)
			http.Error(w, "failed to forward operation", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		fmt.Printf("[Controller] Received response from %s with status: %d\n", resource.URL, resp.StatusCode)

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("[Controller] Resource %s returned status %d for operation\n", resource.URL, resp.StatusCode)
			http.Error(w, "resource operation failed", http.StatusBadGateway)
			return
		}

		fmt.Printf("[Controller] Resource operation forwarded to %s: %+v\n", resource.URL, req.Operations)
	} else {
		fmt.Printf("Resource %s has no URL configured\n", resourceID)
	}

	w.WriteHeader(http.StatusNoContent)
}
