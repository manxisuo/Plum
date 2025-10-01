package httpapi

import (
    "net/http"
)

func RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/healthz", withCORS(handleHealthz))
    mux.HandleFunc("/v1/nodes/heartbeat", withCORS(handleHeartbeat))
    mux.HandleFunc("/v1/nodes", withCORS(handleNodes))
    mux.HandleFunc("/v1/nodes/", withCORS(handleNodeByID))
    mux.HandleFunc("/v1/apps", withCORS(handleListApps))
    mux.HandleFunc("/v1/apps/upload", withCORS(handleAppUpload))
    mux.HandleFunc("/v1/apps/", withCORS(handleDeleteApp))
    mux.HandleFunc("/v1/assignments", withCORS(handleGetAssignments))
    mux.HandleFunc("/v1/assignments/", withCORS(handleAssignmentByID))
    mux.HandleFunc("/v1/instances/status", withCORS(handleStatusUpdate))
    mux.HandleFunc("/v1/tasks", withCORS(handleTasks))
    mux.HandleFunc("/v1/tasks/", withCORS(handleTaskByID))
}


