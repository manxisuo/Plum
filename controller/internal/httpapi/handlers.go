package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"plum/controller/internal/failover"
	"plum/controller/internal/notify"
	"plum/controller/internal/store"
)

type NodeHello struct {
	NodeID string            `json:"nodeId"`
	IP     string            `json:"ip"`
	Labels map[string]string `json:"labels"`
}

type NodeDTO struct {
	NodeID   string            `json:"nodeId"`
	IP       string            `json:"ip"`
	Labels   map[string]string `json:"labels"`
	LastSeen int64             `json:"lastSeen"`
}

type LeaseAck struct {
	TTLSec int64 `json:"ttlSec"`
}

type Assignment struct {
	InstanceID   string `json:"instanceId"`
	DeploymentID string `json:"deploymentId"`
	Desired      string `json:"desired"`
	ArtifactURL  string `json:"artifactUrl"`
	StartCmd     string `json:"startCmd"`
	Phase        string `json:"phase"`
	Healthy      bool   `json:"healthy"`
	LastReport   int64  `json:"lastReportAt"`
}

type Assignments struct {
	Items []Assignment `json:"items"`
}

type StatusUpdate struct {
	InstanceID string `json:"instanceId"`
	Phase      string `json:"phase"`
	ExitCode   int32  `json:"exitCode"`
	Healthy    bool   `json:"healthy"`
	TsUnix     int64  `json:"tsUnix"`
}

type CreateDeploymentRequest struct {
	Name     string                  `json:"name"`
	Artifact string                  `json:"artifactUrl"` // legacy 单条
	StartCmd string                  `json:"startCmd"`    // legacy 单条
	Replicas map[string]int          `json:"replicas"`    // legacy: nodeId -> replica count
	Labels   map[string]string       `json:"labels"`
	Entries  []CreateDeploymentEntry `json:"entries"` // 新：多条目
}

type CreateDeploymentEntry struct {
	Artifact string         `json:"artifactUrl"`
	StartCmd string         `json:"startCmd"`
	Replicas map[string]int `json:"replicas"` // nodeId -> replica
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var hello NodeHello
	if err := json.NewDecoder(r.Body).Decode(&hello); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	now := time.Now()
	_ = store.Current.UpsertNode(hello.NodeID, store.Node{
		NodeID:   hello.NodeID,
		IP:       hello.IP,
		Labels:   hello.Labels,
		LastSeen: now,
	})
	// For walking skeleton, fixed TTL
	writeJSON(w, LeaseAck{TTLSec: 15})
}

func handleNodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		nodes, _ := store.Current.ListNodes()
		health := failover.ComputeHealth()
		out := make([]map[string]any, 0, len(nodes))
		for _, n := range nodes {
			out = append(out, map[string]any{
				"nodeId":   n.NodeID,
				"ip":       n.IP,
				"labels":   n.Labels,
				"lastSeen": n.LastSeen.Unix(),
				"health":   string(health[n.NodeID]),
			})
		}
		writeJSON(w, out)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleNodeByID(w http.ResponseWriter, r *http.Request) {
	// path: /v1/nodes/{id}
	id := r.URL.Path[len("/v1/nodes/"):]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		n, ok, _ := store.Current.GetNode(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, NodeDTO{NodeID: n.NodeID, IP: n.IP, Labels: n.Labels, LastSeen: n.LastSeen.Unix()})
	case http.MethodDelete:
		// 若有 assignments 引用该节点，拒绝删除
		if n, _ := store.Current.CountAssignmentsForNode(id); n > 0 {
			http.Error(w, "node in use", http.StatusConflict)
			return
		}
		_ = store.Current.DeleteNode(id)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	nodeID := r.URL.Query().Get("nodeId")
	if nodeID == "" {
		http.Error(w, "nodeId required", http.StatusBadRequest)
		return
	}
	assigns, _ := store.Current.ListAssignmentsForNode(nodeID)
	// Optional: throttle size with limit=
	if lim := r.URL.Query().Get("limit"); lim != "" {
		if n, err := strconv.Atoi(lim); err == nil && n < len(assigns) {
			assigns = assigns[:n]
		}
	}
	res := Assignments{Items: make([]Assignment, 0, len(assigns))}
	for _, a := range assigns {
		st, ok, _ := store.Current.LatestStatus(a.InstanceID)
		item := Assignment{
			InstanceID:   a.InstanceID,
			DeploymentID: a.DeploymentID,
			Desired:      string(a.Desired),
			ArtifactURL:  a.ArtifactURL,
			StartCmd:     a.StartCmd,
		}
		if ok {
			item.Phase = st.Phase
			item.Healthy = st.Healthy
			item.LastReport = st.TsUnix
		}
		res.Items = append(res.Items, item)
	}
	writeJSON(w, res)
}

func handleStatusUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var su StatusUpdate
	if err := json.NewDecoder(r.Body).Decode(&su); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	_ = store.Current.AppendStatus(su.InstanceID, store.InstanceStatus{
		InstanceID: su.InstanceID,
		Phase:      su.Phase,
		ExitCode:   int(su.ExitCode),
		Healthy:    su.Healthy,
		TsUnix:     su.TsUnix,
	})
	w.WriteHeader(http.StatusNoContent)
}

func handleCreateDeployment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req CreateDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// 规范化为 entries（兼容旧格式），startCmd 可选
	entries := req.Entries
	if len(entries) == 0 {
		if req.Name == "" || req.Artifact == "" || len(req.Replicas) == 0 {
			http.Error(w, "missing fields", http.StatusBadRequest)
			return
		}
		entries = []CreateDeploymentEntry{{Artifact: req.Artifact, StartCmd: req.StartCmd, Replicas: req.Replicas}}
	}
	if req.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	deploymentID, instances, _ := store.Current.CreateDeployment(req.Name, req.Labels)
	for _, e := range entries {
		if e.Artifact == "" || len(e.Replicas) == 0 {
			continue
		}
		for nodeID, replicas := range e.Replicas {
			for i := 0; i < replicas; i++ {
				iid := store.Current.NewInstanceID(deploymentID)
				_ = store.Current.AddAssignment(nodeID, store.Assignment{
					InstanceID:   iid,
					DeploymentID: deploymentID,
					NodeID:       nodeID,
					Desired:      store.DesiredRunning,
					ArtifactURL:  e.Artifact,
					StartCmd:     e.StartCmd,
				})
				notify.Publish(nodeID)
				instances = append(instances, iid)
			}
		}
	}
	writeJSON(w, map[string]any{
		"deploymentId": deploymentID,
		"instances":    instances,
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json error: %v", err)
	}
}

// ---- Tasks (Phase A minimal) ----

type CreateTaskRequest struct {
	Name       string            `json:"name"`
	Executor   string            `json:"executor"`   // service|embedded|os_process
	TargetKind string            `json:"targetKind"` // service|deployment|node
	TargetRef  string            `json:"targetRef"`
	Payload    map[string]any    `json:"payload"`
	Labels     map[string]string `json:"labels"`
	TimeoutSec int               `json:"timeoutSec"`
	MaxRetries int               `json:"maxRetries"`
	AutoStart  bool              `json:"autoStart"`
}

func handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := store.Current.ListTasks()
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, list)
	case http.MethodPost:
		var req CreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		payloadJSON, _ := json.Marshal(req.Payload)
		initialState := "Pending"
		if !req.AutoStart {
			initialState = "Queued"
		}
		id, err := store.Current.CreateTask(store.Task{
			Name:        req.Name,
			Executor:    req.Executor,
			TargetKind:  req.TargetKind,
			TargetRef:   req.TargetRef,
			State:       initialState,
			PayloadJSON: string(payloadJSON),
			TimeoutSec:  req.TimeoutSec,
			MaxRetries:  req.MaxRetries,
			CreatedAt:   time.Now().Unix(),
			Labels:      req.Labels,
		})
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if req.AutoStart {
			notify.PublishTasks()
		}
		writeJSON(w, map[string]any{"taskId": id})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleTaskByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/v1/tasks/"):]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		t, ok, err := store.Current.GetTask(id)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, t)
	case http.MethodDelete:
		_ = store.Current.DeleteTask(id)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// POST /v1/tasks/start/{id}
func handleTaskStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Path[len("/v1/tasks/start/"):]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	// set state from Queued->Pending and notify
	if err := store.Current.UpdateTaskState(id, "Pending"); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	notify.PublishTasks()
	w.WriteHeader(http.StatusNoContent)
}

// POST /v1/tasks/rerun/{id}  -> create a new task with same fields
func handleTaskRerun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Path[len("/v1/tasks/rerun/"):]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	t, ok, err := store.Current.GetTask(id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	origin := t.TaskID
	if t.OriginTaskID != "" {
		origin = t.OriginTaskID
	}
	newID, err := store.Current.CreateTask(store.Task{
		Name: t.Name, Executor: t.Executor, TargetKind: t.TargetKind, TargetRef: t.TargetRef,
		State: "Pending", PayloadJSON: t.PayloadJSON, TimeoutSec: t.TimeoutSec, MaxRetries: t.MaxRetries,
		CreatedAt: time.Now().Unix(), Labels: t.Labels, OriginTaskID: origin,
	})
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	notify.PublishTasks()
	writeJSON(w, map[string]any{"taskId": newID})
}

// POST /v1/tasks/cancel/{id}
func handleTaskCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Path[len("/v1/tasks/cancel/"):]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if err := store.Current.UpdateTaskState(id, "Canceled"); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	notify.PublishTasks()
	w.WriteHeader(http.StatusNoContent)
}

// ---- Workers (embedded) ----

type RegisterWorkerRequest struct {
	WorkerID string            `json:"workerId"`
	NodeID   string            `json:"nodeId"`
	URL      string            `json:"url"`
	Tasks    []string          `json:"tasks"`
	Labels   map[string]string `json:"labels"`
	Capacity int               `json:"capacity"`
}

func handleRegisterWorker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RegisterWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.WorkerID == "" {
		http.Error(w, "workerId required", http.StatusBadRequest)
		return
	}
	err := store.Current.RegisterWorker(store.Worker{
		WorkerID: req.WorkerID,
		NodeID:   req.NodeID,
		URL:      req.URL,
		Tasks:    req.Tasks,
		Labels:   req.Labels,
		Capacity: req.Capacity,
		LastSeen: time.Now().Unix(),
	})
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type HeartbeatWorkerRequest struct {
	WorkerID string `json:"workerId"`
	Capacity int    `json:"capacity"`
}

func handleHeartbeatWorker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req HeartbeatWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.WorkerID == "" {
		http.Error(w, "workerId required", http.StatusBadRequest)
		return
	}
	if err := store.Current.HeartbeatWorker(req.WorkerID, req.Capacity, time.Now().Unix()); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleListWorkers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := store.Current.ListWorkers()
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}
