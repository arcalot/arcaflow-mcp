package workflow

import "sync"

// Cache stores workflow indices for reuse across calls.
type Cache interface {
	Get(key string) (WorkflowIndex, bool)
	Set(key string, index WorkflowIndex)
}

type memoryCache struct {
	mu     sync.RWMutex
	values map[string]WorkflowIndex
}

// NewMemoryCache returns a threadsafe in-memory cache.
func NewMemoryCache() Cache {
	return &memoryCache{
		values: make(map[string]WorkflowIndex),
	}
}

func (cache *memoryCache) Get(key string) (WorkflowIndex, bool) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	value, ok := cache.values[key]
	return value, ok
}

func (cache *memoryCache) Set(key string, index WorkflowIndex) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.values[key] = index
}
