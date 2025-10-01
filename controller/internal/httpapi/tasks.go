package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"plum/controller/internal/store"
)

type TaskDTO struct {
	TaskID    string            `json:"taskId"`
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	Instances int               `json:"instances"`
}

func handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListTasks(w, r)
	case http.MethodPost:
		handleCreateTask(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleListTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tasks, err := store.Current.ListTasks()
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	out := make([]TaskDTO, 0, len(tasks))
	for _, t := range tasks {
		assigns, _ := store.Current.ListAssignmentsForTask(t.TaskID)
		out = append(out, TaskDTO{TaskID: t.TaskID, Name: t.Name, Labels: t.Labels, Instances: len(assigns)})
	}
	writeJSON(w, out)
}

func handleTaskByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/tasks/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		t, ok, _ := store.Current.GetTask(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		assigns, _ := store.Current.ListAssignmentsForTask(id)
		writeJSON(w, map[string]any{"task": t, "assignments": assigns})
	case http.MethodPatch:
		var body struct {
			Name   string            `json:"name"`
			Labels map[string]string `json:"labels"`
		}
		if err := jsonNewDecoder(w, r, &body); err != nil {
			return
		}
		// simple: get and recreate name/labels in store task (no dedicated update yet)
		t, ok, _ := store.Current.GetTask(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		// name is immutable
		if body.Name != "" && body.Name != t.Name {
			http.Error(w, "task name is immutable", http.StatusBadRequest)
			return
		}
		if body.Labels != nil {
			t.Labels = body.Labels
			// emulate labels update via delete+insert (keeping same name)
			_ = store.Current.DeleteTask(id)
			newID, _, err := store.Current.CreateTask(t.Name, t.Labels)
			if err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{"taskId": newID})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		// 级联删除：先删 assignments 与相关 statuses，再删 task
		assigns, _ := store.Current.ListAssignmentsForTask(id)
		for _, a := range assigns {
			_ = store.Current.DeleteEndpointsForInstance(a.InstanceID)
			_ = store.Current.DeleteStatusesForInstance(a.InstanceID)
		}
		_ = store.Current.DeleteAssignmentsForTask(id)
		_ = store.Current.DeleteTask(id)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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
