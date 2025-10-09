package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"plum/controller/internal/store"
)

type DeploymentDTO struct {
	DeploymentID string            `json:"deploymentId"`
	Name         string            `json:"name"`
	Labels       map[string]string `json:"labels"`
	Status       string            `json:"status"` // Stopped | Running
	Instances    int               `json:"instances"`
}

func handleDeployments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListDeployments(w, r)
	case http.MethodPost:
		handleCreateDeployment(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleListDeployments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	deployments, err := store.Current.ListDeployments()
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	out := make([]DeploymentDTO, 0, len(deployments))
	for _, t := range deployments {
		assigns, _ := store.Current.ListAssignmentsForDeployment(t.DeploymentID)
		out = append(out, DeploymentDTO{
			DeploymentID: t.DeploymentID,
			Name:         t.Name,
			Labels:       t.Labels,
			Status:       string(t.Status),
			Instances:    len(assigns),
		})
	}
	writeJSON(w, out)
}

func handleDeploymentByID(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	id := strings.TrimPrefix(path, "/v1/deployments/")
	if id == "" || id == path {
		http.NotFound(w, r)
		return
	}

	// 处理action参数（启动/停止部署）
	action := r.URL.Query().Get("action")
	if action == "start" || action == "stop" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed, use POST", http.StatusMethodNotAllowed)
			return
		}
		handleDeploymentAction(w, r, id, action)
		return
	}

	switch r.Method {
	case http.MethodGet:
		t, ok, _ := store.Current.GetDeployment(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		assigns, _ := store.Current.ListAssignmentsForDeployment(id)
		writeJSON(w, map[string]any{"deployment": t, "assignments": assigns})
	case http.MethodPatch:
		var body struct {
			Name   string            `json:"name"`
			Labels map[string]string `json:"labels"`
		}
		if err := jsonNewDecoder(w, r, &body); err != nil {
			return
		}
		// simple: get and recreate name/labels in store deployment (no dedicated update yet)
		t, ok, _ := store.Current.GetDeployment(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		// name is immutable
		if body.Name != "" && body.Name != t.Name {
			http.Error(w, "deployment name is immutable", http.StatusBadRequest)
			return
		}
		if body.Labels != nil {
			t.Labels = body.Labels
			// emulate labels update via delete+insert (keeping same name)
			_ = store.Current.DeleteDeployment(id)
			newID, _, err := store.Current.CreateDeployment(t.Name, t.Labels)
			if err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{"deploymentId": newID})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		// 级联删除：先删 assignments 与相关 statuses，再删 deployment
		assigns, _ := store.Current.ListAssignmentsForDeployment(id)
		for _, a := range assigns {
			_ = store.Current.DeleteStatusesForInstance(a.InstanceID)
		}
		_ = store.Current.DeleteAssignmentsForDeployment(id)
		_ = store.Current.DeleteDeployment(id)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDeploymentAction 处理部署的启动/停止操作
func handleDeploymentAction(w http.ResponseWriter, r *http.Request, id string, action string) {
	_, ok, _ := store.Current.GetDeployment(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	var newStatus store.DeploymentStatus
	if action == "start" {
		newStatus = store.DeploymentRunning
	} else {
		newStatus = store.DeploymentStopped
	}

	if err := store.Current.UpdateDeploymentStatus(id, newStatus); err != nil {
		http.Error(w, "failed to update status", http.StatusInternalServerError)
		return
	}

	// 如果是停止操作，将所有实例的Desired状态设为Stopped
	if action == "stop" {
		assigns, _ := store.Current.ListAssignmentsForDeployment(id)
		for _, a := range assigns {
			_ = store.Current.UpdateAssignmentDesired(a.InstanceID, store.DesiredStopped)
		}
	} else {
		// 如果是启动操作，将所有实例的Desired状态设为Running
		assigns, _ := store.Current.ListAssignmentsForDeployment(id)
		for _, a := range assigns {
			_ = store.Current.UpdateAssignmentDesired(a.InstanceID, store.DesiredRunning)
		}
	}

	writeJSON(w, map[string]any{
		"deploymentId": id,
		"status":       string(newStatus),
		"message":      "Deployment " + action + "ed successfully",
	})
}

// small helper to decode JSON with error handling
func jsonNewDecoder(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return err
	}
	return nil
}
