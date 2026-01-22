package tenant

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceManager(t *testing.T) {
	root := t.TempDir()
	manager, err := NewWorkspaceManager(root)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	path, err := manager.Workspace("tenant-a")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("expected absolute path, got %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected workspace to exist: %v", err)
	}
}

func TestWorkspaceManagerEmptyRoot(t *testing.T) {
	if _, err := NewWorkspaceManager(" "); err == nil {
		t.Fatalf("expected error for empty root")
	}
}

func TestWorkspaceManagerInvalidTenant(t *testing.T) {
	manager, err := NewWorkspaceManager(t.TempDir())
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	if _, err := manager.Workspace(" "); err == nil {
		t.Fatalf("expected error for empty tenant id")
	}
}

func TestWorkspaceManagerNil(t *testing.T) {
	var manager *WorkspaceManager
	if _, err := manager.Workspace("tenant-a"); err == nil {
		t.Fatalf("expected error for nil manager")
	}
}

func TestWorkspaceManagerSanitizesTenantID(t *testing.T) {
	manager, err := NewWorkspaceManager(t.TempDir())
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	path, err := manager.Workspace("tenant/a")
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	if strings.Contains(path, "tenant/a") {
		t.Fatalf("expected sanitized path, got %q", path)
	}
	if filepath.Base(path) == "tenant/a" {
		t.Fatalf("expected sanitized base, got %q", filepath.Base(path))
	}
}

func TestLimiterAcquire(t *testing.T) {
	limiter := NewLimiter(1)
	release, ok := limiter.Acquire("tenant-a")
	if !ok {
		t.Fatalf("expected first acquire ok")
	}
	if _, ok := limiter.Acquire("tenant-a"); ok {
		t.Fatalf("expected second acquire blocked")
	}
	release()
	if _, ok := limiter.Acquire("tenant-a"); !ok {
		t.Fatalf("expected acquire after release")
	}
}
