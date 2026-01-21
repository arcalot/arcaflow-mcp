package tenant

import (
	"os"
	"path/filepath"
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
