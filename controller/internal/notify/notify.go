package notify

import (
	"sync"
	"time"
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

// KV channel (per namespace)
func SubscribeKV(namespace string) (chan struct{}, func()) {
	return Subscribe("__kv__:" + namespace)
}

// KV事件批处理器（节流）
type kvBatcher struct {
	mu            sync.Mutex
	pendingEvents map[string]time.Time // namespace -> last update time
	ticker        *time.Ticker
	stopCh        chan struct{}
}

var kvBatch = &kvBatcher{
	pendingEvents: make(map[string]time.Time),
	stopCh:        make(chan struct{}),
}

func init() {
	// 启动批处理协程
	kvBatch.ticker = time.NewTicker(100 * time.Millisecond)
	go kvBatch.run()
}

func (b *kvBatcher) run() {
	for {
		select {
		case <-b.ticker.C:
			b.flush()
		case <-b.stopCh:
			return
		}
	}
}

func (b *kvBatcher) flush() {
	b.mu.Lock()
	// 复制待发送的namespace列表
	toPublish := make([]string, 0, len(b.pendingEvents))
	for ns := range b.pendingEvents {
		toPublish = append(toPublish, ns)
	}
	// 清空待发送队列
	b.pendingEvents = make(map[string]time.Time)
	b.mu.Unlock()

	// 发送通知（不持有锁）
	for _, ns := range toPublish {
		Publish("__kv__:" + ns)
	}
}

func (b *kvBatcher) add(namespace string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pendingEvents[namespace] = time.Now()
}

func PublishKV(namespace, key, value, valueType string) {
	// 使用批处理器，100ms内的多次更新合并为一次通知
	kvBatch.add(namespace)
	// Also publish to global KV channel for monitoring (无节流)
	Publish("__kv__")
}
