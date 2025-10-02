package notify

import (
	"sync"
)

type subscriber struct {
	ch chan struct{}
}

type Notifier struct {
	mu   sync.Mutex
	subs map[string]map[*subscriber]struct{} // key -> subs (nodeId 或 task:*)
}

var global = &Notifier{subs: make(map[string]map[*subscriber]struct{})}

func Subscribe(nodeID string) (chan struct{}, func()) {
	s := &subscriber{ch: make(chan struct{}, 1)}
	global.mu.Lock()
	m := global.subs[nodeID]
	if m == nil {
		m = make(map[*subscriber]struct{})
		global.subs[nodeID] = m
	}
	m[s] = struct{}{}
	global.mu.Unlock()
	cancel := func() {
		global.mu.Lock()
		if mm, ok := global.subs[nodeID]; ok {
			delete(mm, s)
			if len(mm) == 0 {
				delete(global.subs, nodeID)
			}
		}
		close(s.ch)
		global.mu.Unlock()
	}
	return s.ch, cancel
}

func Publish(nodeID string) {
	global.mu.Lock()
	defer global.mu.Unlock()
	if mm, ok := global.subs[nodeID]; ok {
		for s := range mm {
			select {
			case s.ch <- struct{}{}:
			default:
			}
		}
	}
}

// Tasks channel (global)
func SubscribeTasks() (chan struct{}, func()) {
	return Subscribe("__tasks__")
}

func PublishTasks() {
	Publish("__tasks__")
}
