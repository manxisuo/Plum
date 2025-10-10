package httpapi

import (
	"net/http"

	"github.com/manxisuo/plum/controller/internal/notify"
)

// Server-Sent Events: /v1/stream?nodeId=
func handleSSEStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}
	nodeID := r.URL.Query().Get("nodeId")
	if nodeID == "" {
		http.Error(w, "nodeId required", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch, cancel := notify.Subscribe(nodeID)
	defer cancel()
	// initial ping
	_, _ = w.Write([]byte("event: ping\n"))
	_, _ = w.Write([]byte("data: init\n\n"))
	flusher.Flush()
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
			_, _ = w.Write([]byte("event: update\n"))
			_, _ = w.Write([]byte("data: assignments\n\n"))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// /v1/tasks/stream (global)
func handleTasksSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch, cancel := notify.SubscribeTasks()
	defer cancel()
	// initial ping
	_, _ = w.Write([]byte("event: ping\n"))
	_, _ = w.Write([]byte("data: init\n\n"))
	flusher.Flush()
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
			_, _ = w.Write([]byte("event: update\n"))
			_, _ = w.Write([]byte("data: tasks\n\n"))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
