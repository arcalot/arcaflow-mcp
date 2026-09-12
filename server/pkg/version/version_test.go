package version

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentUsesEnv(t *testing.T) {
	t.Setenv("ARCAFLOW_MCP_VERSION", "env-version")
	if got := Current(); got != "env-version" {
		t.Fatalf("expected env version, got %q", got)
	}
}

func TestCurrentReadsVersionFile(t *testing.T) {
	t.Setenv("ARCAFLOW_MCP_VERSION", "")
	dir := t.TempDir()
	versionPath := filepath.Join(dir, "VERSION")
	if err := os.WriteFile(versionPath, []byte("file-version\n"), 0o644); err != nil {
		t.Fatalf("write version: %v", err)
	}
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(original)
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if got := Current(); got != "file-version" {
		t.Fatalf("expected file version, got %q", got)
	}
}

func TestCurrentFallsBack(t *testing.T) {
	t.Setenv("ARCAFLOW_MCP_VERSION", "")
	dir := t.TempDir()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(original)
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if got := Current(); got != fallbackVersion {
		t.Fatalf("expected fallback version %q, got %q", fallbackVersion, got)
	}
}
