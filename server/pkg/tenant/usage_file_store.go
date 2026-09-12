// Package tenant provides helpers for multi-tenant isolation.
package tenant

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// FileUsageStore persists usage statistics to a JSON file.
type FileUsageStore struct {
	path   string
	mu     sync.Mutex
	usages map[string]UsageStats
}

type usageFilePayload struct {
	Usages []UsageStats `json:"usages"`
}

// NewFileUsageStore initializes a file-backed usage store.
func NewFileUsageStore(path string) (*FileUsageStore, error) {
	if path == "" {
		return nil, errors.New("usage store path must not be empty")
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil, errors.New("usage store path must include a directory")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	store := &FileUsageStore{
		path:   path,
		usages: make(map[string]UsageStats),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

// RecordRequest increments request counts for a tenant.
func (s *FileUsageStore) RecordRequest(tenantID string) {
	s.updateUsage(tenantID, func(usage UsageStats) UsageStats {
		usage.RequestCount++
		return usage
	})
}

// RecordRateLimitViolation increments rate limit violations for a tenant.
func (s *FileUsageStore) RecordRateLimitViolation(tenantID string) {
	s.updateUsage(tenantID, func(usage UsageStats) UsageStats {
		usage.RateLimitViolations++
		return usage
	})
}

// RecordAuditEvent increments audit totals for a tenant.
func (s *FileUsageStore) RecordAuditEvent(tenantID string) {
	s.updateUsage(tenantID, func(usage UsageStats) UsageStats {
		usage.AuditEvents++
		return usage
	})
}

// SetWorkspaceBytes records the latest workspace size for a tenant.
func (s *FileUsageStore) SetWorkspaceBytes(tenantID string, bytes int64) {
	if bytes < 0 {
		return
	}
	s.updateUsage(tenantID, func(usage UsageStats) UsageStats {
		usage.WorkspaceBytes = bytes
		return usage
	})
}

// Get returns usage metrics for a tenant.
func (s *FileUsageStore) Get(tenantID string) UsageStats {
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
func (s *FileUsageStore) List() []UsageStats {
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

func (s *FileUsageStore) updateUsage(
	tenantID string,
	mutate func(UsageStats) UsageStats,
) {
	if s == nil || tenantID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	usage, ok := s.usages[tenantID]
	if !ok {
		usage = UsageStats{TenantID: tenantID}
	}
	usage = mutate(usage)
	s.usages[tenantID] = usage
	// Best-effort persistence to keep usage available after restarts.
	_ = s.persistLocked()
}

func (s *FileUsageStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s.persistLocked()
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	var payload usageFilePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	for _, usage := range payload.Usages {
		if usage.TenantID == "" {
			continue
		}
		s.usages[usage.TenantID] = usage
	}
	return nil
}

func (s *FileUsageStore) persistLocked() error {
	payload := usageFilePayload{
		Usages: make([]UsageStats, 0, len(s.usages)),
	}
	for _, usage := range s.usages {
		payload.Usages = append(payload.Usages, usage)
	}
	sort.Slice(payload.Usages, func(i, j int) bool {
		return payload.Usages[i].TenantID < payload.Usages[j].TenantID
	})
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(s.path), "usage-*.json")
	if err != nil {
		return err
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	if err := tempFile.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tempFile.Write(data); err != nil {
		return err
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(tempFile.Name(), s.path)
}
