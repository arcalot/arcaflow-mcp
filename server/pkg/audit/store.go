// Package audit provides audit logging persistence and querying.
package audit

import "time"

// Record captures an auditable event.
type Record struct {
	Timestamp time.Time `json:"timestamp"`
	TenantID  string    `json:"tenant_id,omitempty"`
	Action    string    `json:"action"`
	Outcome   string    `json:"outcome"`
	Status    int       `json:"status"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Error     string    `json:"error,omitempty"`
}

// Filter describes query constraints for audit records.
type Filter struct {
	TenantID string
	Action   string
	Outcome  string
	Status   *int
	From     *time.Time
	To       *time.Time
	Limit    int
}

// Store persists and queries audit records.
type Store interface {
	Add(record Record) error
	Query(filter Filter) []Record
}
