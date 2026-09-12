package audit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStorePersistsRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.json")

	store, err := NewFileStore(path, 0)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	record := Record{
		Timestamp: time.Now().UTC(),
		TenantID:  "tenant-a",
		Action:    "mcp_request",
		Outcome:   "success",
		Status:    200,
		Method:    "POST",
		Path:      "/mcp",
	}
	if err := store.Add(record); err != nil {
		t.Fatalf("add record: %v", err)
	}

	reloaded, err := NewFileStore(path, 0)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	results := reloaded.Query(Filter{TenantID: "tenant-a"})
	if len(results) != 1 {
		t.Fatalf("expected 1 record, got %d", len(results))
	}
}

func TestFileStoreQueryFilters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.json")

	store, err := NewFileStore(path, 0)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	now := time.Now().UTC()
	records := []Record{
		{
			Timestamp: now.Add(-2 * time.Hour),
			TenantID:  "tenant-a",
			Action:    "mcp_request",
			Outcome:   "success",
			Status:    200,
			Method:    "POST",
			Path:      "/mcp",
		},
		{
			Timestamp: now.Add(-1 * time.Hour),
			TenantID:  "tenant-a",
			Action:    "admin_auth",
			Outcome:   "denied",
			Status:    401,
			Method:    "GET",
			Path:      "/admin",
		},
	}
	for _, record := range records {
		if err := store.Add(record); err != nil {
			t.Fatalf("add record: %v", err)
		}
	}

	status := 401
	results := store.Query(Filter{
		TenantID: "tenant-a",
		Status:   &status,
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 record, got %d", len(results))
	}
	if results[0].Status != 401 {
		t.Fatalf("expected status 401")
	}
}

func TestFileStoreRetentionPrunes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.json")

	store, err := NewFileStore(path, 1)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	oldRecord := Record{
		Timestamp: time.Now().UTC().Add(-48 * time.Hour),
		TenantID:  "tenant-a",
		Action:    "mcp_request",
		Outcome:   "success",
		Status:    200,
		Method:    "POST",
		Path:      "/mcp",
	}
	if err := store.Add(oldRecord); err != nil {
		t.Fatalf("add record: %v", err)
	}
	newRecord := Record{
		Timestamp: time.Now().UTC(),
		TenantID:  "tenant-a",
		Action:    "mcp_request",
		Outcome:   "success",
		Status:    200,
		Method:    "POST",
		Path:      "/mcp",
	}
	if err := store.Add(newRecord); err != nil {
		t.Fatalf("add record: %v", err)
	}

	results := store.Query(Filter{TenantID: "tenant-a"})
	if len(results) != 1 {
		t.Fatalf("expected 1 record after prune, got %d", len(results))
	}
	if !results[0].Timestamp.Equal(newRecord.Timestamp) {
		t.Fatalf("expected recent record")
	}
}

func TestNewFileStoreInvalidPath(t *testing.T) {
	if _, err := NewFileStore("", 0); err == nil {
		t.Fatalf("expected error for empty path")
	}
	if _, err := NewFileStore("audit.json", 0); err == nil {
		t.Fatalf("expected error for path without directory")
	}
	if _, err := NewFileStore("/tmp/audit.json", -1); err == nil {
		t.Fatalf("expected error for negative retention")
	}
}

func TestFileStoreQueryDefaultLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.json")

	store, err := NewFileStore(path, 0)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	for i := 0; i < defaultQueryLimit+10; i++ {
		if err := store.Add(Record{
			Timestamp: time.Now().UTC(),
			TenantID:  "tenant-a",
			Action:    "mcp_request",
			Outcome:   "success",
			Status:    200,
			Method:    "POST",
			Path:      "/mcp",
		}); err != nil {
			t.Fatalf("add record: %v", err)
		}
	}
	results := store.Query(Filter{TenantID: "tenant-a"})
	if len(results) != defaultQueryLimit {
		t.Fatalf("expected %d records, got %d", defaultQueryLimit, len(results))
	}
}

func TestFileStoreAddAssignsTimestamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.json")

	store, err := NewFileStore(path, 0)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	record := Record{
		TenantID: "tenant-a",
		Action:   "mcp_request",
		Outcome:  "success",
		Status:   200,
		Method:   "POST",
		Path:     "/mcp",
	}
	if err := store.Add(record); err != nil {
		t.Fatalf("add record: %v", err)
	}
	results := store.Query(Filter{TenantID: "tenant-a"})
	if len(results) != 1 {
		t.Fatalf("expected 1 record, got %d", len(results))
	}
	if results[0].Timestamp.IsZero() {
		t.Fatalf("expected timestamp to be set")
	}
}

func TestFileStoreQueryOutcomeFilter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.json")

	store, err := NewFileStore(path, 0)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	if err := store.Add(Record{
		Timestamp: time.Now().UTC(),
		TenantID:  "tenant-a",
		Action:    "mcp_request",
		Outcome:   "success",
		Status:    200,
		Method:    "POST",
		Path:      "/mcp",
	}); err != nil {
		t.Fatalf("add record: %v", err)
	}
	if err := store.Add(Record{
		Timestamp: time.Now().UTC(),
		TenantID:  "tenant-a",
		Action:    "mcp_request",
		Outcome:   "denied",
		Status:    401,
		Method:    "POST",
		Path:      "/mcp",
	}); err != nil {
		t.Fatalf("add record: %v", err)
	}
	results := store.Query(Filter{Outcome: "denied"})
	if len(results) != 1 {
		t.Fatalf("expected 1 record, got %d", len(results))
	}
	if results[0].Outcome != "denied" {
		t.Fatalf("expected denied outcome")
	}
}

func TestFileStoreQueryActionFilter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.json")

	store, err := NewFileStore(path, 0)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	if err := store.Add(Record{
		Timestamp: time.Now().UTC(),
		TenantID:  "tenant-a",
		Action:    "mcp_request",
		Outcome:   "success",
		Status:    200,
		Method:    "POST",
		Path:      "/mcp",
	}); err != nil {
		t.Fatalf("add record: %v", err)
	}
	if err := store.Add(Record{
		Timestamp: time.Now().UTC(),
		TenantID:  "tenant-a",
		Action:    "admin_auth",
		Outcome:   "denied",
		Status:    401,
		Method:    "GET",
		Path:      "/admin",
	}); err != nil {
		t.Fatalf("add record: %v", err)
	}
	results := store.Query(Filter{Action: "admin_auth"})
	if len(results) != 1 {
		t.Fatalf("expected 1 record, got %d", len(results))
	}
	if results[0].Action != "admin_auth" {
		t.Fatalf("expected admin_auth action")
	}
}

func TestFileStoreQueryStatusFilter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.json")

	store, err := NewFileStore(path, 0)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	if err := store.Add(Record{
		Timestamp: time.Now().UTC(),
		TenantID:  "tenant-a",
		Action:    "mcp_request",
		Outcome:   "success",
		Status:    200,
		Method:    "POST",
		Path:      "/mcp",
	}); err != nil {
		t.Fatalf("add record: %v", err)
	}
	status := 200
	results := store.Query(Filter{Status: &status})
	if len(results) != 1 {
		t.Fatalf("expected 1 record, got %d", len(results))
	}
}

func TestFileStoreQueryTimeRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.json")

	store, err := NewFileStore(path, 0)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	now := time.Now().UTC()
	oldRecord := Record{
		Timestamp: now.Add(-2 * time.Hour),
		TenantID:  "tenant-a",
		Action:    "mcp_request",
		Outcome:   "success",
		Status:    200,
		Method:    "POST",
		Path:      "/mcp",
	}
	newRecord := Record{
		Timestamp: now.Add(-30 * time.Minute),
		TenantID:  "tenant-a",
		Action:    "mcp_request",
		Outcome:   "success",
		Status:    200,
		Method:    "POST",
		Path:      "/mcp",
	}
	if err := store.Add(oldRecord); err != nil {
		t.Fatalf("add record: %v", err)
	}
	if err := store.Add(newRecord); err != nil {
		t.Fatalf("add record: %v", err)
	}
	from := now.Add(-1 * time.Hour)
	to := now.Add(-10 * time.Minute)
	results := store.Query(Filter{From: &from, To: &to})
	if len(results) != 1 {
		t.Fatalf("expected 1 record, got %d", len(results))
	}
	if !results[0].Timestamp.Equal(newRecord.Timestamp) {
		t.Fatalf("expected newer record in range")
	}
}

func TestFileStoreInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := NewFileStore(path, 0); err == nil {
		t.Fatalf("expected error for invalid json")
	}
}

func TestFileStoreQueryLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	store, err := NewFileStore(path, 0)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		if err := store.Add(Record{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			TenantID:  "tenant-a",
			Action:    "mcp_request",
			Outcome:   "success",
			Status:    200,
			Method:    "POST",
			Path:      "/mcp",
		}); err != nil {
			t.Fatalf("add record: %v", err)
		}
	}
	results := store.Query(Filter{Limit: 1})
	if len(results) != 1 {
		t.Fatalf("expected 1 record, got %d", len(results))
	}
	if !results[0].Timestamp.Equal(now.Add(2 * time.Minute)) {
		t.Fatalf("expected most recent record")
	}
}

func TestFileStoreNilStore(t *testing.T) {
	var store *FileStore
	if err := store.Add(Record{Action: "test"}); err == nil {
		t.Fatalf("expected error for nil store")
	}
	if records := store.Query(Filter{}); records != nil {
		t.Fatalf("expected nil records for nil store")
	}
}
