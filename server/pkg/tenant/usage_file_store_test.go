package tenant

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileUsageStorePersistsAndReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.json")
	store, err := NewFileUsageStore(path)
	if err != nil {
		t.Fatalf("expected store, got error %v", err)
	}
	store.RecordRequest("tenant-a")
	store.RecordRequest("tenant-a")
	store.RecordRateLimitViolation("tenant-a")
	store.RecordAuditEvent("tenant-a")
	store.SetWorkspaceBytes("tenant-a", 512)
	store.RecordRequest("tenant-b")

	reloaded, err := NewFileUsageStore(path)
	if err != nil {
		t.Fatalf("expected reload, got error %v", err)
	}
	usage := reloaded.Get("tenant-a")
	if usage.RequestCount != 2 {
		t.Fatalf("expected request_count 2, got %d", usage.RequestCount)
	}
	if usage.RateLimitViolations != 1 {
		t.Fatalf(
			"expected rate_limit_violations 1, got %d",
			usage.RateLimitViolations,
		)
	}
	if usage.AuditEvents != 1 {
		t.Fatalf("expected audit_events 1, got %d", usage.AuditEvents)
	}
	if usage.WorkspaceBytes != 512 {
		t.Fatalf(
			"expected workspace_bytes 512, got %d",
			usage.WorkspaceBytes,
		)
	}
	usages := reloaded.List()
	if len(usages) != 2 {
		t.Fatalf("expected 2 usages, got %d", len(usages))
	}
	if usages[0].TenantID != "tenant-a" || usages[1].TenantID != "tenant-b" {
		t.Fatalf("expected sorted usages, got %v", usages)
	}
}

func TestFileUsageStoreNilSafety(t *testing.T) {
	var store *FileUsageStore
	store.RecordRequest("tenant-a")
	store.RecordRateLimitViolation("tenant-a")
	store.RecordAuditEvent("tenant-a")
	store.SetWorkspaceBytes("tenant-a", 3)
	usage := store.Get("tenant-a")
	if usage.TenantID != "tenant-a" {
		t.Fatalf("expected tenant-a, got %q", usage.TenantID)
	}
	if len(store.List()) != 0 {
		t.Fatalf("expected nil list for nil store")
	}
}

func TestNewFileUsageStoreValidation(t *testing.T) {
	if _, err := NewFileUsageStore(""); err == nil {
		t.Fatalf("expected empty path error")
	}
	if _, err := NewFileUsageStore("usage.json"); err == nil {
		t.Fatalf("expected missing dir error")
	}
}

func TestFileUsageStoreInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write usage: %v", err)
	}
	if _, err := NewFileUsageStore(path); err == nil {
		t.Fatalf("expected parse error")
	}
}
