package workflow

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecGitClientInputValidation(t *testing.T) {
	t.Parallel()
	client := &execGitClient{}
	if _, err := client.Sync(context.Background(), "", "dest", "", nil); err == nil {
		t.Fatalf("expected error for missing repo url")
	}
	if _, err := client.Sync(context.Background(), "repo", "", "", nil); err == nil {
		t.Fatalf("expected error for missing dest dir")
	}
}

func TestExecGitClientRunErrors(t *testing.T) {
	t.Parallel()
	client := &execGitClient{}
	dir := t.TempDir()
	err := client.run(context.Background(), dir, "false", "status")
	if err == nil {
		t.Fatalf("expected run error")
	}
	_, err = client.output(context.Background(), dir, "false", "rev-parse")
	if err == nil {
		t.Fatalf("expected output error")
	}
}

func TestExecGitClientSyncLocalRepo(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	ctx := context.Background()
	repoDir := t.TempDir()
	destDir := t.TempDir()

	runGit(t, repoDir, "init", "-b", "main")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "config", "user.name", "Test User")

	filePath := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "init")

	expected := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "HEAD"))
	client := &execGitClient{}
	commit, err := client.Sync(ctx, repoDir, destDir, "main", nil)
	if err != nil {
		t.Fatalf("sync repo: %v", err)
	}
	if commit != expected {
		t.Fatalf("expected commit %s, got %s", expected, commit)
	}
}

func TestExecGitClientSyncUsesDefaultRef(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	ctx := context.Background()
	repoDir := t.TempDir()
	destDir := t.TempDir()

	runGit(t, repoDir, "init", "-b", "main")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "config", "user.name", "Test User")

	filePath := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "init")

	expected := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "HEAD"))
	client := &execGitClient{}
	commit, err := client.Sync(ctx, repoDir, destDir, "", nil)
	if err != nil {
		t.Fatalf("sync repo: %v", err)
	}
	if commit != expected {
		t.Fatalf("expected commit %s, got %s", expected, commit)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
	}
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
	}
	return string(output)
}
