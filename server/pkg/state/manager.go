package state

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
)

var (
	// ErrTenantRequired indicates the tenant ID is missing.
	ErrTenantRequired = errors.New("tenant id is required")
	// ErrSessionRequired indicates the session ID is missing.
	ErrSessionRequired = errors.New("session id is required")
)

// SessionData captures per-tenant state for input construction and analysis.
type SessionData struct {
	SessionID  string                 `json:"session_id"`
	WorkflowID string                 `json:"workflow_id,omitempty"`
	DraftInput json.RawMessage        `json:"draft_input,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	UpdatedAt  time.Time              `json:"updated_at"`
	ExpiresAt  time.Time              `json:"expires_at"`
}

// Manager stores per-tenant session state in memory.
type Manager struct {
	mu        sync.RWMutex
	sessions  map[string]map[string]SessionData
	ttl       time.Duration
	now       func() time.Time
	clockLock sync.Mutex
}

// NewManager constructs a state manager with a default TTL.
func NewManager(ttl time.Duration) *Manager {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &Manager{
		sessions: make(map[string]map[string]SessionData),
		ttl:      ttl,
		now:      time.Now,
	}
}

// WithClock overrides the time provider for tests.
func (m *Manager) WithClock(now func() time.Time) {
	if now == nil {
		return
	}
	m.clockLock.Lock()
	defer m.clockLock.Unlock()
	m.now = now
}

// Set stores session data for the tenant in the context.
func (m *Manager) Set(ctx context.Context, data SessionData) error {
	tenantID, ok := auth.TenantIDFromContext(ctx)
	if !ok || tenantID == "" {
		return ErrTenantRequired
	}
	if data.SessionID == "" {
		return ErrSessionRequired
	}
	now := m.now()
	if data.UpdatedAt.IsZero() {
		data.UpdatedAt = now
	}
	if data.ExpiresAt.IsZero() {
		data.ExpiresAt = now.Add(m.ttl)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[tenantID]; !ok {
		m.sessions[tenantID] = make(map[string]SessionData)
	}
	m.sessions[tenantID][data.SessionID] = data
	return nil
}

// Get retrieves session data for the tenant in the context.
func (m *Manager) Get(ctx context.Context, sessionID string) (SessionData, bool, error) {
	tenantID, ok := auth.TenantIDFromContext(ctx)
	if !ok || tenantID == "" {
		return SessionData{}, false, ErrTenantRequired
	}
	if sessionID == "" {
		return SessionData{}, false, ErrSessionRequired
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	sessionMap, ok := m.sessions[tenantID]
	if !ok {
		return SessionData{}, false, nil
	}
	data, ok := sessionMap[sessionID]
	if !ok {
		return SessionData{}, false, nil
	}
	if !data.ExpiresAt.IsZero() && data.ExpiresAt.Before(m.now()) {
		return SessionData{}, false, nil
	}
	return data, true, nil
}

// Delete removes a session for the tenant in the context.
func (m *Manager) Delete(ctx context.Context, sessionID string) error {
	tenantID, ok := auth.TenantIDFromContext(ctx)
	if !ok || tenantID == "" {
		return ErrTenantRequired
	}
	if sessionID == "" {
		return ErrSessionRequired
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	sessionMap, ok := m.sessions[tenantID]
	if !ok {
		return nil
	}
	delete(sessionMap, sessionID)
	if len(sessionMap) == 0 {
		delete(m.sessions, tenantID)
	}
	return nil
}

// CleanupExpired removes expired sessions across all tenants.
func (m *Manager) CleanupExpired() int {
	removed := 0
	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()
	for tenantID, sessionMap := range m.sessions {
		for sessionID, data := range sessionMap {
			if !data.ExpiresAt.IsZero() && data.ExpiresAt.Before(now) {
				delete(sessionMap, sessionID)
				removed++
			}
		}
		if len(sessionMap) == 0 {
			delete(m.sessions, tenantID)
		}
	}
	return removed
}
