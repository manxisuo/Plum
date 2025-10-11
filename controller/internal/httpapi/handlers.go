package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/manxisuo/plum/controller/internal/failover"
	"github.com/manxisuo/plum/controller/internal/notify"
	"github.com/manxisuo/plum/controller/internal/store"
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
	AppName      string `json:"appName"`    // 应用名称
	AppVersion   string `json:"appVersion"` // 应用版本
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
			AppName:      a.AppName,
			AppVersion:   a.AppVersion,
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
				// 从artifact URL中获取app信息
				var appName, appVersion string
				if artifact, ok, _ := store.Current.GetArtifactByPath(e.Artifact); ok {
					appName = artifact.AppName
					appVersion = artifact.Version
				}
				_ = store.Current.AddAssignment(nodeID, store.Assignment{
					InstanceID:   iid,
					DeploymentID: deploymentID,
					NodeID:       nodeID,
					Desired:      store.DesiredRunning,
					ArtifactURL:  e.Artifact,
					StartCmd:     e.StartCmd,
					AppName:      appName,
					AppVersion:   appVersion,
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

// ---- Workflows (sequential MVP) ----

type CreateWorkflowRequest struct {
	Name   string               `json:"name"`
	Labels map[string]string    `json:"labels"`
	Steps  []store.WorkflowStep `json:"steps"`
}

func handleWorkflows(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := store.Current.ListWorkflows()
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, list)
	case http.MethodPost:
		var req CreateWorkflowRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		// normalize step ids / ord
		for i := range req.Steps {
			if req.Steps[i].StepID == "" {
				req.Steps[i].StepID = strconv.FormatInt(time.Now().UnixNano()+int64(i), 36)
			}
			req.Steps[i].Ord = i
		}
		id, err := store.Current.CreateWorkflow(store.Workflow{WorkflowID: "", Name: req.Name, Labels: req.Labels, Steps: req.Steps})
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"workflowId": id})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleWorkflowByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/v1/workflows/"):]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		wf, ok, err := store.Current.GetWorkflow(id)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, wf)
	case http.MethodPost: // start a run
		if r.URL.Query().Get("action") == "run" {
			runID, err := store.Current.CreateWorkflowRun(id)
			if err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			// mark started
			_ = store.Current.UpdateWorkflowRunState(runID, "Running", time.Now().Unix())
			writeJSON(w, map[string]any{"runId": runID})
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
	case http.MethodDelete:
		err := store.Current.DeleteWorkflow(id)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		workflowID := r.URL.Query().Get("workflowId")
		var runs []store.WorkflowRun
		var err error
		if workflowID != "" {
			runs, err = store.Current.ListWorkflowRunsByWorkflow(workflowID)
		} else {
			runs, err = store.Current.ListWorkflowRuns()
		}
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, runs)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleWorkflowRunByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/v1/workflow-runs/"):]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		run, ok, err := store.Current.GetWorkflowRun(id)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		steps, _ := store.Current.ListWorkflowSteps(run.WorkflowID)
		srs, _ := store.Current.ListStepRuns(id)
		writeJSON(w, map[string]any{"run": run, "steps": steps, "stepRuns": srs})
	case http.MethodDelete:
		err := store.Current.DeleteWorkflowRun(id)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ---- TaskDefinitions ----
type CreateTaskDefRequest struct {
	Name           string            `json:"name"`
	Executor       string            `json:"executor"`
	TargetKind     string            `json:"targetKind"`
	TargetRef      string            `json:"targetRef"`
	Labels         map[string]string `json:"labels"`
	DefaultPayload map[string]any    `json:"defaultPayload"`
}

func handleTaskDefs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := store.Current.ListTaskDefs()
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, list)
	case http.MethodPost:
		var req CreateTaskDefRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		// 检查任务名称是否已存在
		if req.Name == "" {
			http.Error(w, "task name is required", http.StatusBadRequest)
			return
		}
		if _, exists, _ := store.Current.GetTaskDefByName(req.Name); exists {
			http.Error(w, "task name already exists", http.StatusConflict)
			return
		}
		payload := ""
		if req.DefaultPayload != nil {
			if bs, err := json.Marshal(req.DefaultPayload); err == nil {
				payload = string(bs)
			}
		}
		id, err := store.Current.CreateTaskDef(store.TaskDefinition{Name: req.Name, Executor: req.Executor, TargetKind: req.TargetKind, TargetRef: req.TargetRef, Labels: req.Labels, DefaultPayloadJSON: payload, CreatedAt: time.Now().Unix()})
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"defId": id})
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		// conflict if referenced by tasks
		n, err := store.Current.CountTasksByOrigin(id)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if n > 0 {
			w.WriteHeader(http.StatusConflict)
			writeJSON(w, map[string]any{"referenced": n})
			return
		}
		if err := store.Current.DeleteTaskDef(id); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleTaskDefByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/v1/task-defs/"):]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		td, ok, err := store.Current.GetTaskDef(id)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, td)
	case http.MethodPost: // action run
		if r.URL.Query().Get("action") == "run" {
			td, ok, err := store.Current.GetTaskDef(id)
			if err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			if !ok {
				http.NotFound(w, r)
				return
			}
			// accept optional payload and timeout from request body
			var rr struct {
				Payload    map[string]any `json:"payload"`
				TimeoutSec int            `json:"timeoutSec"`
				MaxRetries int            `json:"maxRetries"`
			}
			_ = json.NewDecoder(r.Body).Decode(&rr)
			payload := td.DefaultPayloadJSON
			if rr.Payload != nil { // override if provided
				if bs, err := json.Marshal(rr.Payload); err == nil {
					payload = string(bs)
				}
			}
			timeoutSec := rr.TimeoutSec
			if timeoutSec <= 0 {
				timeoutSec = 300 // default 5 minutes
			}
			maxRetries := rr.MaxRetries
			if maxRetries < 0 {
				maxRetries = 0
			}
			newID, err := store.Current.CreateTask(store.Task{Name: td.Name, Executor: td.Executor, TargetKind: td.TargetKind, TargetRef: td.TargetRef, State: "Pending", PayloadJSON: payload, TimeoutSec: timeoutSec, MaxRetries: maxRetries, CreatedAt: time.Now().Unix(), Labels: td.Labels, OriginTaskID: id})
			if err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			notify.PublishTasks()
			writeJSON(w, map[string]any{"taskId": newID})
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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
