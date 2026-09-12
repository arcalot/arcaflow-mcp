// Package ratelimit provides per-tenant fixed-window rate limiting.
package ratelimit

import (
	"errors"
	"sync"
	"time"
)

// Clock returns the current time for rate limiting decisions.
type Clock func() time.Time

// Config controls fixed-window rate limiting.
type Config struct {
	Limit          int
	Window         time.Duration
	BackoffEnabled bool
	BackoffBase    time.Duration
	BackoffMax     time.Duration
	Clock          Clock
}

// Limiter enforces per-tenant request limits.
type Limiter struct {
	limit          int
	window         time.Duration
	backoffEnabled bool
	backoffBase    time.Duration
	backoffMax     time.Duration
	clock          Clock
	mu             sync.Mutex
	tenants        map[string]*tenantState
}

// Decision captures the rate limit decision for a request.
type Decision struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
}

type tenantState struct {
	windowStart   time.Time
	count         int
	penaltyLevel  int
	cooldownUntil time.Time
}

// NewLimiter constructs a limiter with the provided config.
func NewLimiter(cfg Config) (*Limiter, error) {
	if cfg.Limit <= 0 {
		return nil, errors.New("rate limit must be greater than zero")
	}
	if cfg.Window <= 0 {
		return nil, errors.New("rate limit window must be greater than zero")
	}
	if cfg.BackoffEnabled {
		if cfg.BackoffBase <= 0 {
			return nil, errors.New("backoff base must be greater than zero")
		}
		if cfg.BackoffMax <= 0 {
			return nil, errors.New("backoff max must be greater than zero")
		}
		if cfg.BackoffMax < cfg.BackoffBase {
			return nil, errors.New("backoff max must be >= backoff base")
		}
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	return &Limiter{
		limit:          cfg.Limit,
		window:         cfg.Window,
		backoffEnabled: cfg.BackoffEnabled,
		backoffBase:    cfg.BackoffBase,
		backoffMax:     cfg.BackoffMax,
		clock:          clock,
		tenants:        make(map[string]*tenantState),
	}, nil
}

// Limit returns the configured request limit for the window.
func (l *Limiter) Limit() int {
	return l.limit
}

// Window returns the configured window duration.
func (l *Limiter) Window() time.Duration {
	return l.window
}

// Allow applies the fixed-window limit and returns a decision.
func (l *Limiter) Allow(tenantID string) Decision {
	now := l.clock().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()

	state, ok := l.tenants[tenantID]
	if !ok {
		state = &tenantState{windowStart: now}
		l.tenants[tenantID] = state
	}

	if now.Sub(state.windowStart) >= l.window {
		state.windowStart = now
		state.count = 0
		state.penaltyLevel = 0
		state.cooldownUntil = time.Time{}
	}

	if l.backoffEnabled && !state.cooldownUntil.IsZero() &&
		!now.Before(state.cooldownUntil) {
		state.windowStart = now
		state.count = 0
		state.penaltyLevel = 0
		state.cooldownUntil = time.Time{}
	}

	if l.backoffEnabled && now.Before(state.cooldownUntil) {
		state.penaltyLevel++
		backoff := l.backoffDuration(state.penaltyLevel)
		state.cooldownUntil = now.Add(backoff)
		return Decision{
			Allowed:   false,
			Remaining: 0,
			ResetAt:   state.cooldownUntil,
		}
	}

	resetAt := state.windowStart.Add(l.window)
	if state.count >= l.limit {
		decisionReset := resetAt
		if l.backoffEnabled {
			state.penaltyLevel++
			backoff := l.backoffDuration(state.penaltyLevel)
			state.cooldownUntil = now.Add(backoff)
			decisionReset = state.cooldownUntil
		}
		return Decision{
			Allowed:   false,
			Remaining: 0,
			ResetAt:   decisionReset,
		}
	}

	state.count++
	state.penaltyLevel = 0
	state.cooldownUntil = time.Time{}
	remaining := l.limit - state.count
	if remaining < 0 {
		remaining = 0
	}
	return Decision{
		Allowed:   true,
		Remaining: remaining,
		ResetAt:   resetAt,
	}
}

func (l *Limiter) backoffDuration(level int) time.Duration {
	if !l.backoffEnabled || level <= 0 {
		return 0
	}
	backoff := l.backoffBase << (level - 1)
	if backoff > l.backoffMax {
		return l.backoffMax
	}
	return backoff
}
