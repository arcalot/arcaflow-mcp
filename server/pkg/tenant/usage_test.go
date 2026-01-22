package tenant

import "testing"

func TestInMemoryUsageStoreRecords(t *testing.T) {
	store := NewInMemoryUsageStore()
	store.RecordRequest("tenant-a")
	store.RecordRequest("tenant-a")
	store.RecordRateLimitViolation("tenant-a")
	store.RecordAuditEvent("tenant-a")

	usage := store.Get("tenant-a")
	if usage.TenantID != "tenant-a" {
		t.Fatalf("expected tenant-a, got %q", usage.TenantID)
	}
	if usage.RequestCount != 2 {
		t.Fatalf("expected request_count 2, got %d", usage.RequestCount)
	}
	if usage.RateLimitViolations != 1 {
		t.Fatalf("expected rate_limit_violations 1, got %d", usage.RateLimitViolations)
	}
	if usage.AuditEvents != 1 {
		t.Fatalf("expected audit_events 1, got %d", usage.AuditEvents)
	}
}

func TestInMemoryUsageStoreListSorted(t *testing.T) {
	store := NewInMemoryUsageStore()
	store.RecordRequest("tenant-b")
	store.RecordRequest("tenant-a")

	usages := store.List()
	if len(usages) != 2 {
		t.Fatalf("expected 2 usages, got %d", len(usages))
	}
	if usages[0].TenantID != "tenant-a" || usages[1].TenantID != "tenant-b" {
		t.Fatalf("unexpected order: %v", usages)
	}
}

func TestInMemoryUsageStoreEmptyTenant(t *testing.T) {
	store := NewInMemoryUsageStore()
	store.RecordRequest("")
	usage := store.Get("")
	if usage.TenantID != "" {
		t.Fatalf("expected empty tenant id, got %q", usage.TenantID)
	}
	if usage.RequestCount != 0 {
		t.Fatalf("expected zero request count, got %d", usage.RequestCount)
	}
}

func TestInMemoryUsageStoreNilSafety(t *testing.T) {
	var store *InMemoryUsageStore
	store.RecordRequest("tenant-a")
	store.RecordRateLimitViolation("tenant-a")
	store.RecordAuditEvent("tenant-a")
	usage := store.Get("tenant-a")
	if usage.TenantID != "tenant-a" {
		t.Fatalf("expected tenant-a, got %q", usage.TenantID)
	}
	if len(store.List()) != 0 {
		t.Fatalf("expected nil list for nil store")
	}
}
