package tenant

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceUsageEmptyPath(t *testing.T) {
	if _, err := WorkspaceUsage(""); err == nil {
		t.Fatalf("expected error for empty path")
	}
}

func TestWorkspaceUsageSkipsSymlink(t *testing.T) {
	workspace := t.TempDir()
	externalDir := t.TempDir()

	workspaceFile := filepath.Join(workspace, "data.txt")
	if err := os.WriteFile(workspaceFile, []byte("1234"), 0o600); err != nil {
		t.Fatalf("write workspace file: %v", err)
	}
	externalFile := filepath.Join(externalDir, "external.txt")
	if err := os.WriteFile(externalFile, []byte("12345678"), 0o600); err != nil {
		t.Fatalf("write external file: %v", err)
	}
	symlinkPath := filepath.Join(workspace, "external-link")
	if err := os.Symlink(externalFile, symlinkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	total, err := WorkspaceUsage(workspace)
	if err != nil {
		t.Fatalf("workspace usage: %v", err)
	}
	if total != 4 {
		t.Fatalf("expected 4 bytes, got %d", total)
	}
}

func TestWorkspaceUsageMissingPath(t *testing.T) {
	if _, err := WorkspaceUsage("/path/does/not/exist"); err == nil {
		t.Fatalf("expected error for missing path")
	}
}
