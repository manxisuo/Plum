package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/manxisuo/plum/controller/internal/store"
)

type EmbeddedWorkerRegisterRequest struct {
	WorkerID    string            `json:"workerId"`
	NodeID      string            `json:"nodeId"`
	InstanceID  string            `json:"instanceId"`
	AppName     string            `json:"appName"`
	AppVersion  string            `json:"appVersion"`
	GRPCAddress string            `json:"grpcAddress"`
	Tasks       []string          `json:"tasks"`
	Labels      map[string]string `json:"labels"`
}

type EmbeddedWorkerHeartbeatRequest struct {
	WorkerID string `json:"workerId"`
}

func handleRegisterEmbeddedWorker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EmbeddedWorkerRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.WorkerID == "" || req.NodeID == "" || req.GRPCAddress == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	worker := store.EmbeddedWorker{
		WorkerID:    req.WorkerID,
		NodeID:      req.NodeID,
		InstanceID:  req.InstanceID,
		AppName:     req.AppName,
		AppVersion:  req.AppVersion,
		GRPCAddress: req.GRPCAddress,
		Tasks:       req.Tasks,
		Labels:      req.Labels,
		LastSeen:    time.Now().Unix(),
	}

	if err := store.Current.RegisterEmbeddedWorker(worker); err != nil {
		fmt.Printf("Failed to register embedded worker %s: %v\n", req.WorkerID, err)
		http.Error(w, "registration failed", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Registered embedded worker %s at %s\n", req.WorkerID, req.GRPCAddress)
	w.WriteHeader(http.StatusCreated)
}

func handleHeartbeatEmbeddedWorker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EmbeddedWorkerHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.WorkerID == "" {
		http.Error(w, "missing workerId", http.StatusBadRequest)
		return
	}

	if err := store.Current.HeartbeatEmbeddedWorker(req.WorkerID, time.Now().Unix()); err != nil {
		fmt.Printf("Failed to heartbeat embedded worker %s: %v\n", req.WorkerID, err)
		http.Error(w, "heartbeat failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleListEmbeddedWorkers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workers, err := store.Current.ListEmbeddedWorkers()
	if err != nil {
		fmt.Printf("Failed to list embedded workers: %v\n", err)
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(workers); err != nil {
		fmt.Printf("Failed to encode embedded workers: %v\n", err)
	}
}

func handleGetEmbeddedWorker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workerID := r.URL.Path[len("/v1/embedded-workers/"):]
	if workerID == "" {
		http.Error(w, "missing workerId", http.StatusBadRequest)
		return
	}

	worker, exists, err := store.Current.GetEmbeddedWorker(workerID)
	if err != nil {
		fmt.Printf("Failed to get embedded worker %s: %v\n", workerID, err)
		http.Error(w, "get failed", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "worker not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(worker); err != nil {
		fmt.Printf("Failed to encode embedded worker: %v\n", err)
	}
}

func handleDeleteEmbeddedWorker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workerID := r.URL.Path[len("/v1/embedded-workers/"):]
	if workerID == "" {
		http.Error(w, "missing workerId", http.StatusBadRequest)
		return
	}

	if err := store.Current.DeleteEmbeddedWorker(workerID); err != nil {
		fmt.Printf("Failed to delete embedded worker %s: %v\n", workerID, err)
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Deleted embedded worker %s\n", workerID)
	w.WriteHeader(http.StatusNoContent)
}
