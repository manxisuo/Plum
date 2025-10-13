package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/manxisuo/plum/controller/internal/dagengine"
	"github.com/manxisuo/plum/controller/internal/store"
)

var dagOrch *dagengine.DAGOrchestrator

// InitDAGOrchestrator - 初始化DAG编排器
func InitDAGOrchestrator(s store.Store) {
	dagOrch = dagengine.NewDAGOrchestrator(s)
	dagOrch.Start()
}

// StopDAGOrchestrator - 停止DAG编排器
func StopDAGOrchestrator() {
	if dagOrch != nil {
		dagOrch.Stop()
	}
}

// handleDAGWorkflows - /v1/dag/workflows
func handleDAGWorkflows(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListDAGWorkflows(w, r)
	case http.MethodPost:
		handleCreateDAGWorkflow(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDAGWorkflowByID - /v1/dag/workflows/{id}
func handleDAGWorkflowByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/dag/workflows/")
	parts := strings.Split(path, "/")
	id := parts[0]

	if id == "" {
		http.Error(w, "workflow id required", http.StatusBadRequest)
		return
	}

	// 如果有 /run 后缀，执行工作流
	if len(parts) > 1 && parts[1] == "run" && r.Method == http.MethodPost {
		handleRunDAGWorkflow(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetDAGWorkflow(w, r, id)
	case http.MethodDelete:
		handleDeleteDAGWorkflow(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// 列出所有DAG工作流
func handleListDAGWorkflows(w http.ResponseWriter, r *http.Request) {
	dags, err := store.Current.ListWorkflowDAGs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, dags)
}

// 创建DAG工作流
func handleCreateDAGWorkflow(w http.ResponseWriter, r *http.Request) {
	var dag store.WorkflowDAG
	if err := json.NewDecoder(r.Body).Decode(&dag); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// 基本验证
	if dag.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if len(dag.Nodes) == 0 {
		http.Error(w, "nodes required", http.StatusBadRequest)
		return
	}

	id, err := store.Current.CreateWorkflowDAG(dag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"workflowId": id,
	})
}

// 获取DAG工作流
func handleGetDAGWorkflow(w http.ResponseWriter, r *http.Request, id string) {
	dag, ok, err := store.Current.GetWorkflowDAG(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, dag)
}

// 删除DAG工作流
func handleDeleteDAGWorkflow(w http.ResponseWriter, r *http.Request, id string) {
	if err := store.Current.DeleteWorkflowDAG(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// 运行DAG工作流
func handleRunDAGWorkflow(w http.ResponseWriter, r *http.Request, id string) {
	if dagOrch == nil {
		http.Error(w, "dag orchestrator not initialized", http.StatusInternalServerError)
		return
	}

	runID, err := dagOrch.StartDAGRun(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"runId": runID,
	})
}

// handleDAGRunStatus - /v1/dag/runs/{runId}/status
func handleDAGRunStatus(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/dag/runs/")
	parts := strings.Split(path, "/")
	runID := parts[0]

	if dagOrch == nil {
		http.Error(w, "dag orchestrator not initialized", http.StatusInternalServerError)
		return
	}

	// 获取节点状态
	nodeStates := dagOrch.GetRunStatus(runID)

	// 从数据库获取Run信息
	run, ok, err := store.Current.GetWorkflowRun(runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	// 返回状态（节点状态可能为空或从Task重建）
	if nodeStates == nil {
		nodeStates = make(map[string]string)
	}

	writeJSON(w, map[string]any{
		"runId": runID,
		"state": run.State,
		"nodes": nodeStates,
	})
}
