// Package tenant provides helpers for multi-tenant isolation.
package tenant

import (
	"sort"
	"sync"
)

// UsageStats captures per-tenant usage metrics.
type UsageStats struct {
	TenantID            string `json:"tenant_id"`
	RequestCount        int64  `json:"request_count"`
	RateLimitViolations int64  `json:"rate_limit_violations"`
	AuditEvents         int64  `json:"audit_events"`
	ActiveSessions      int    `json:"active_sessions"`
	WorkspaceBytes      int64  `json:"workspace_bytes"`
}

// UsageStore records usage metrics per tenant.
type UsageStore interface {
	RecordRequest(tenantID string)
	RecordRateLimitViolation(tenantID string)
	RecordAuditEvent(tenantID string)
	Get(tenantID string) UsageStats
	List() []UsageStats
}

// InMemoryUsageStore stores usage metrics in memory.
type InMemoryUsageStore struct {
	mu     sync.Mutex
	usages map[string]UsageStats
}

// NewInMemoryUsageStore constructs an in-memory usage store.
func NewInMemoryUsageStore() *InMemoryUsageStore {
	return &InMemoryUsageStore{
		usages: make(map[string]UsageStats),
	}
}

// RecordRequest increments request counts for a tenant.
func (s *InMemoryUsageStore) RecordRequest(tenantID string) {
	if s == nil || tenantID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	usage := s.usages[tenantID]
	usage.TenantID = tenantID
	usage.RequestCount++
	s.usages[tenantID] = usage
}

// RecordRateLimitViolation increments rate limit violations for a tenant.
func (s *InMemoryUsageStore) RecordRateLimitViolation(tenantID string) {
	if s == nil || tenantID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	usage := s.usages[tenantID]
	usage.TenantID = tenantID
	usage.RateLimitViolations++
	s.usages[tenantID] = usage
}

// RecordAuditEvent increments audit totals for a tenant.
func (s *InMemoryUsageStore) RecordAuditEvent(tenantID string) {
	if s == nil || tenantID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	usage := s.usages[tenantID]
	usage.TenantID = tenantID
	usage.AuditEvents++
	s.usages[tenantID] = usage
}

// Get returns usage metrics for a tenant.
func (s *InMemoryUsageStore) Get(tenantID string) UsageStats {
	if s == nil || tenantID == "" {
		return UsageStats{TenantID: tenantID}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	usage := s.usages[tenantID]
	usage.TenantID = tenantID
	return usage
}

// List returns usage metrics for all tenants.
func (s *InMemoryUsageStore) List() []UsageStats {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	usages := make([]UsageStats, 0, len(s.usages))
	for _, usage := range s.usages {
		usages = append(usages, usage)
	}
	sort.Slice(usages, func(i, j int) bool {
		return usages[i].TenantID < usages[j].TenantID
	})
	return usages
}
