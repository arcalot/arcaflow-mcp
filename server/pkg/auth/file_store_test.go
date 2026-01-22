package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStorePersistsTokens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	manager, err := NewManager("admin-token", store)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	info, err := manager.CreateToken("tenant-a", &expiresAt)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	loadedStore, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	if _, ok := loadedStore.Get(info.Token); !ok {
		t.Fatalf("expected token to persist")
	}
}

func TestFileStoreRevokePersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	manager, err := NewManager("admin-token", store)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	info, err := manager.CreateToken("tenant-a", nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if !manager.RevokeToken(info.Token) {
		t.Fatalf("expected revoke to succeed")
	}

	loadedStore, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	if _, ok := loadedStore.Get(info.Token); ok {
		t.Fatalf("expected token to be removed")
	}
}

func TestFileStoreInvalidPath(t *testing.T) {
	if _, err := NewFileStore(""); err == nil {
		t.Fatalf("expected error for empty path")
	}
	if _, err := NewFileStore("tokens.json"); err == nil {
		t.Fatalf("expected error for path without directory")
	}
}

func TestFileStoreListByTenant(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	manager, err := NewManager("admin-token", store)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	if _, err := manager.CreateToken("tenant-a", nil); err != nil {
		t.Fatalf("create token: %v", err)
	}
	if _, err := manager.CreateToken("tenant-b", nil); err != nil {
		t.Fatalf("create token: %v", err)
	}
	tokens := store.ListByTenant("tenant-a")
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
	if tokens[0].TenantID != "tenant-a" {
		t.Fatalf("expected tenant-a, got %q", tokens[0].TenantID)
	}
}

func TestFileStoreInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := NewFileStore(path); err == nil {
		t.Fatalf("expected error for invalid json")
	}
}
