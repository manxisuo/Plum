package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"plum/controller/internal/notify"
	"plum/controller/internal/store"
)

func handleAssignmentByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/assignments/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodDelete:
		if a, ok, _ := store.Current.GetAssignment(id); ok {
			notify.Publish(a.NodeID)
		}
		_ = store.Current.DeleteEndpointsForInstance(id)
		_ = store.Current.DeleteAssignment(id)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodPatch:
		var body struct {
			Desired string `json:"desired"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var d store.DesiredState
		switch body.Desired {
		case string(store.DesiredRunning):
			d = store.DesiredRunning
		case string(store.DesiredStopped):
			d = store.DesiredStopped
		default:
			http.Error(w, "bad desired", http.StatusBadRequest)
			return
		}
		if a, ok, _ := store.Current.GetAssignment(id); ok {
			notify.Publish(a.NodeID)
		}
		if err := store.Current.UpdateAssignmentDesired(id, d); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if d == store.DesiredStopped {
			_ = store.Current.DeleteEndpointsForInstance(id)
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
