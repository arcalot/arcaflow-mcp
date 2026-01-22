package tenant

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStorePersistsTenants(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tenants.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	created, err := store.Create(Record{ID: "tenant-a", DisplayName: "Tenant A"})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if created.ID != "tenant-a" {
		t.Fatalf("expected tenant-a, got %q", created.ID)
	}

	loaded, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	if _, ok := loaded.Get("tenant-a"); !ok {
		t.Fatalf("expected tenant to persist")
	}
}

func TestFileStoreDeletePersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tenants.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	if _, err := store.Create(Record{ID: "tenant-a"}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if !store.Delete("tenant-a") {
		t.Fatalf("expected delete to succeed")
	}

	loaded, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	if _, ok := loaded.Get("tenant-a"); ok {
		t.Fatalf("expected tenant to be removed")
	}
}

func TestFileStoreInvalidPath(t *testing.T) {
	if _, err := NewFileStore(""); err == nil {
		t.Fatalf("expected error for empty path")
	}
	if _, err := NewFileStore("tenants.json"); err == nil {
		t.Fatalf("expected error for path without directory")
	}
}

func TestFileStoreUpdateTrimmed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tenants.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	if _, err := store.Create(Record{ID: "tenant-a"}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	name := "  Tenant A  "
	updated, err := store.Update("tenant-a", Update{DisplayName: &name})
	if err != nil {
		t.Fatalf("update tenant: %v", err)
	}
	if updated.DisplayName != "Tenant A" {
		t.Fatalf("expected trimmed name, got %q", updated.DisplayName)
	}
}

func TestFileStoreUpdateMissingFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tenants.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	if _, err := store.Create(Record{ID: "tenant-a"}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if _, err := store.Update("tenant-a", Update{}); err != ErrTenantUpdateRequired {
		t.Fatalf("expected update required error, got %v", err)
	}
}

func TestFileStoreListIDs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tenants.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	if _, err := store.Create(Record{ID: "tenant-b"}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if _, err := store.Create(Record{ID: "tenant-a"}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	ids := store.ListIDs()
	if len(ids) != 2 || ids[0] != "tenant-a" || ids[1] != "tenant-b" {
		t.Fatalf("unexpected ids: %v", ids)
	}
}

func TestFileStoreInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tenants.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := NewFileStore(path); err == nil {
		t.Fatalf("expected error for invalid json")
	}
}

func TestFileStoreNilStore(t *testing.T) {
	var store *FileStore
	if _, err := store.Create(Record{ID: "tenant-a"}); err != ErrTenantNotFound {
		t.Fatalf("expected tenant not found error, got %v", err)
	}
	if _, ok := store.Get("tenant-a"); ok {
		t.Fatalf("expected get to fail for nil store")
	}
	if store.Delete("tenant-a") {
		t.Fatalf("expected delete to be false for nil store")
	}
}

func TestFileStoreUpdateMissingTenant(t *testing.T) {
	store, err := NewFileStore(filepath.Join(t.TempDir(), "tenants.json"))
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	name := "Tenant A"
	if _, err := store.Update("tenant-a", Update{DisplayName: &name}); err != ErrTenantNotFound {
		t.Fatalf("expected tenant not found error, got %v", err)
	}
}

func TestFileStoreCreateInvalidID(t *testing.T) {
	store, err := NewFileStore(filepath.Join(t.TempDir(), "tenants.json"))
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	if _, err := store.Create(Record{ID: "bad id"}); err != ErrInvalidTenantID {
		t.Fatalf("expected invalid tenant id error, got %v", err)
	}
}

func TestFileStoreDeleteMissing(t *testing.T) {
	store, err := NewFileStore(filepath.Join(t.TempDir(), "tenants.json"))
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	if store.Delete("tenant-a") {
		t.Fatalf("expected delete to be false for missing tenant")
	}
}

func TestFileStoreEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tenants.json")
	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	if len(store.List()) != 0 {
		t.Fatalf("expected empty store")
	}
}
