package plum

import (
	"sync"
	"time"
)

// CacheEntry 缓存条目
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
	CreatedAt time.Time
}

// SmartCache 智能缓存系统
type SmartCache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
	ttl     time.Duration
}

// NewSmartCache 创建智能缓存
func NewSmartCache(defaultTTL time.Duration) *SmartCache {
	return &SmartCache{
		entries: make(map[string]*CacheEntry),
		ttl:     defaultTTL,
	}
}

// Set 设置缓存
func (c *SmartCache) Set(key string, data interface{}, customTTL ...time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	ttl := c.ttl
	if len(customTTL) > 0 {
		ttl = customTTL[0]
	}

	c.entries[key] = &CacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
}

// Get 获取缓存
func (c *SmartCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(entry.ExpiresAt) {
		// 异步清理过期条目
		go c.cleanup()
		return nil, false
	}

	return entry.Data, true
}

// GetWithTTL 获取缓存并返回剩余TTL
func (c *SmartCache) GetWithTTL(key string) (interface{}, time.Duration, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, 0, false
	}

	// 检查是否过期
	if time.Now().After(entry.ExpiresAt) {
		go c.cleanup()
		return nil, 0, false
	}

	remaining := time.Until(entry.ExpiresAt)
	return entry.Data, remaining, true
}

// Delete 删除缓存
func (c *SmartCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.entries, key)
}

// Clear 清空缓存
func (c *SmartCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.entries = make(map[string]*CacheEntry)
}

// cleanup 清理过期条目
func (c *SmartCache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

// Size 返回缓存大小
func (c *SmartCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.entries)
}

// Keys 返回所有键
func (c *SmartCache) Keys() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	keys := make([]string, 0, len(c.entries))
	for key := range c.entries {
		keys = append(keys, key)
	}
	return keys
}

// IsExpired 检查键是否过期
func (c *SmartCache) IsExpired(key string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return true
	}

	return time.Now().After(entry.ExpiresAt)
}

// Refresh 刷新缓存TTL
func (c *SmartCache) Refresh(key string, newTTL time.Duration) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		return false
	}

	entry.ExpiresAt = time.Now().Add(newTTL)
	return true
}
