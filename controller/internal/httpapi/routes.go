package httpapi

import (
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", withCORS(handleHealthz))
	// Swagger UI & OpenAPI
	mux.HandleFunc("/swagger", handleSwaggerUI)
	mux.HandleFunc("/swagger/", handleSwaggerUI)
	mux.HandleFunc("/swagger/openapi.json", withCORS(handleOpenAPI))
	// nodes & health
	mux.HandleFunc("/v1/stream", withCORS(handleSSEStream))
	mux.HandleFunc("/v1/nodes/heartbeat", withCORS(handleHeartbeat))
	mux.HandleFunc("/v1/nodes", withCORS(handleNodes))
	mux.HandleFunc("/v1/nodes/", withCORS(handleNodeByID))
	// services (register/discovery)
	mux.HandleFunc("/v1/services/register", withCORS(handleRegisterEndpoints))
	mux.HandleFunc("/v1/services/heartbeat", withCORS(handleHeartbeatEndpoints))
	mux.HandleFunc("/v1/services", withCORS(handleDeleteEndpoints)) // DELETE ?instanceId=
	mux.HandleFunc("/v1/services/list", withCORS(handleListServices))
	mux.HandleFunc("/v1/discovery", withCORS(handleDiscover))
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
	// task definitions
	mux.HandleFunc("/v1/task-defs", withCORS(handleTaskDefs))
	mux.HandleFunc("/v1/task-defs/", withCORS(handleTaskDefByID))
}
