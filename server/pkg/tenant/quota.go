// Package tenant provides helpers for multi-tenant isolation.
package tenant

import "errors"

var (
	// ErrWorkspaceQuotaExceeded indicates workspace storage exceeded the limit.
	ErrWorkspaceQuotaExceeded = errors.New("tenant workspace quota exceeded")
	// ErrRequestQuotaExceeded indicates request quota exceeded the limit.
	ErrRequestQuotaExceeded = errors.New("tenant request quota exceeded")
)

// Quota defines per-tenant resource limits.
type Quota struct {
	MaxWorkspaceBytes int64
	MaxRequests       int64
}

// Check enforces quota limits against current usage values.
func (q Quota) Check(workspaceBytes int64, requestCount int64) error {
	if q.MaxWorkspaceBytes > 0 && workspaceBytes > q.MaxWorkspaceBytes {
		return ErrWorkspaceQuotaExceeded
	}
	if q.MaxRequests > 0 && requestCount >= q.MaxRequests {
		return ErrRequestQuotaExceeded
	}
	return nil
}
