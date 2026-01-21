package tenant

import (
	"sync"
)

// Limiter enforces a per-tenant concurrent usage limit.
type Limiter struct {
	limit int
	mu    sync.Mutex
	items map[string]chan struct{}
}

// NewLimiter builds a limiter with a fixed per-tenant limit.
func NewLimiter(limit int) *Limiter {
	if limit <= 0 {
		return nil
	}
	return &Limiter{
		limit: limit,
		items: make(map[string]chan struct{}),
	}
}

// Acquire reserves a slot for the tenant and returns a release function.
func (l *Limiter) Acquire(tenantID string) (func(), bool) {
	if l == nil {
		return func() {}, true
	}
	l.mu.Lock()
	channel, ok := l.items[tenantID]
	if !ok {
		channel = make(chan struct{}, l.limit)
		l.items[tenantID] = channel
	}
	l.mu.Unlock()

	select {
	case channel <- struct{}{}:
		return func() {
			select {
			case <-channel:
			default:
			}
		}, true
	default:
		return nil, false
	}
}
