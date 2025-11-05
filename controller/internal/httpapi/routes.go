package httpapi

import (
	"net/http"
	"strings"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", withCORS(handleHealthz))
	// Swagger UI & OpenAPI
	mux.HandleFunc("/swagger", handleSwaggerUI)
	mux.HandleFunc("/swagger/", handleSwaggerUI)
	mux.HandleFunc("/swagger/openapi.json", withCORS(handleOpenAPI))
	// Swagger UI static files - 必须在通配符路由之前
	mux.HandleFunc("/swagger/swagger-ui.css", handleSwaggerCSS)
	mux.HandleFunc("/swagger/swagger-ui-bundle.js", handleSwaggerJS)
	// nodes & health
	mux.HandleFunc("/v1/stream", withCORS(handleSSEStream))
	mux.HandleFunc("/v1/nodes/heartbeat", withCORS(handleHeartbeat))
	mux.HandleFunc("/v1/nodes", withCORS(handleNodes))
	mux.HandleFunc("/v1/nodes/", withCORS(handleNodeByID))
	// services (register/discovery)
	mux.HandleFunc("/v1/services/register", withCORS(handleRegisterEndpoints))
	mux.HandleFunc("/v1/services/heartbeat", withCORS(handleHeartbeatEndpoints))
	mux.HandleFunc("/v1/services", withCORS(handleDeleteEndpoints))       // DELETE ?instanceId=
	mux.HandleFunc("/v1/services/endpoint", withCORS(handleEndpointCRUD)) // DELETE/PATCH单个端点
	mux.HandleFunc("/v1/services/list", withCORS(handleListServices))
	mux.HandleFunc("/v1/discovery", withCORS(handleDiscover))
	mux.HandleFunc("/v1/discovery/random", withCORS(handleDiscoverRandom))
	mux.HandleFunc("/v1/apps", withCORS(handleListApps))
	mux.HandleFunc("/v1/apps/upload", withCORS(handleAppUpload))
	mux.HandleFunc("/v1/apps/", withCORS(handleDeleteApp))
	mux.HandleFunc("/v1/assignments", withCORS(handleGetAssignments))
	mux.HandleFunc("/v1/assignments/", withCORS(handleAssignmentByID))
	mux.HandleFunc("/v1/instances/status", withCORS(handleStatusUpdate))
	// embedded workers
	mux.HandleFunc("/v1/workers/register", withCORS(handleRegisterWorker))
	mux.HandleFunc("/v1/workers/heartbeat", withCORS(handleHeartbeatWorker))
	mux.HandleFunc("/v1/workers", withCORS(handleListWorkers))
	// resources
	mux.HandleFunc("/v1/resources/register", withCORS(handleRegisterResource))
	mux.HandleFunc("/v1/resources/heartbeat", withCORS(handleHeartbeatResource))
	mux.HandleFunc("/v1/resources", withCORS(handleListResources))
	mux.HandleFunc("/v1/resources/", withCORS(handleGetResource))
	mux.HandleFunc("/v1/resources/state", withCORS(handleSubmitResourceState))
	mux.HandleFunc("/v1/resources/states", withCORS(handleListResourceStates))
	mux.HandleFunc("/v1/resources/operation", withCORS(handleResourceOperation))

	// Embedded Workers (new gRPC-based)
	mux.HandleFunc("/v1/embedded-workers/register", withCORS(handleRegisterEmbeddedWorker))
	mux.HandleFunc("/v1/embedded-workers/heartbeat", withCORS(handleHeartbeatEmbeddedWorker))
	mux.HandleFunc("/v1/embedded-workers", withCORS(handleListEmbeddedWorkers))
	mux.HandleFunc("/v1/embedded-workers/", withCORS(handleGetEmbeddedWorker))
	mux.HandleFunc("/v1/embedded-workers/delete/", withCORS(handleDeleteEmbeddedWorker))

	// deployments
	mux.HandleFunc("/v1/deployments", withCORS(handleDeployments))
	mux.HandleFunc("/v1/deployments/", withCORS(handleDeploymentByID))
	// tasks (Phase A minimal)
	mux.HandleFunc("/v1/tasks", withCORS(handleTasks))
	mux.HandleFunc("/v1/tasks/", withCORS(handleTaskByID))
	mux.HandleFunc("/v1/tasks/stream", withCORS(handleTasksSSE))
	mux.HandleFunc("/v1/tasks/start/", withCORS(handleTaskStart))
	mux.HandleFunc("/v1/tasks/rerun/", withCORS(handleTaskRerun))
	mux.HandleFunc("/v1/tasks/cancel/", withCORS(handleTaskCancel))
	// workflows (sequential MVP)
	mux.HandleFunc("/v1/workflows", withCORS(handleWorkflows))
	mux.HandleFunc("/v1/workflows/", withCORS(handleWorkflowByID))
	mux.HandleFunc("/v1/workflow-runs", withCORS(handleWorkflowRuns))
	mux.HandleFunc("/v1/workflow-runs/", withCORS(handleWorkflowRunByID))
	// DAG workflows (v2)
	mux.HandleFunc("/v1/dag/workflows", withCORS(handleDAGWorkflows))
	mux.HandleFunc("/v1/dag/workflows/", withCORS(handleDAGWorkflowByID))
	mux.HandleFunc("/v1/dag/runs/", withCORS(handleDAGRunStatus))
	// task definitions
	mux.HandleFunc("/v1/task-defs", withCORS(handleTaskDefs))
	mux.HandleFunc("/v1/task-defs/", withCORS(handleTaskDefByID))
	// distributed KV (both with and without trailing slash)
	mux.HandleFunc("/v1/kv/", withCORS(handleKV))
	mux.HandleFunc("/v1/kv", withCORS(handleKV))
}

// handleKV routes KV requests based on path pattern
func handleKV(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/kv")
	path = strings.TrimPrefix(path, "/") // Remove leading slash if present

	// GET /v1/kv or /v1/kv/ - list all namespaces
	if path == "" {
		handleKVListNamespaces(w, r)
		return
	}

	// /v1/kv/{namespace}/batch
	if strings.HasSuffix(path, "/batch") {
		handleKVBatch(w, r)
		return
	}

	// GET /v1/kv/{namespace}/keys - list keys in namespace
	if strings.HasSuffix(path, "/keys") {
		handleKVListKeys(w, r)
		return
	}

	// /v1/kv/{namespace}/{key}
	if strings.Count(path, "/") >= 1 {
		handleKVByKey(w, r)
		return
	}

	// /v1/kv/{namespace}
	if path != "" && !strings.Contains(path, "/") {
		handleKVByNamespace(w, r)
		return
	}

	http.Error(w, "invalid path", http.StatusBadRequest)
}
