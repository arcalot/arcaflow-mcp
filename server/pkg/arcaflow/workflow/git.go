package workflow

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitClient syncs repositories for workflow discovery.
type GitClient interface {
	Sync(ctx context.Context, repoURL, destDir, ref string) (string, error)
}

type execGitClient struct{}

// NewGitClient returns a git client that shells out to the git binary.
func NewGitClient() GitClient {
	return &execGitClient{}
}

func (client *execGitClient) Sync(
	ctx context.Context,
	repoURL string,
	destDir string,
	ref string,
) (string, error) {
	if repoURL == "" {
		return "", fmt.Errorf("git repo url is required")
	}
	if destDir == "" {
		return "", fmt.Errorf("git destination directory is required")
	}
	if ref == "" {
		ref = "HEAD"
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create git cache dir: %w", err)
	}

	gitDir := filepath.Join(destDir, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		if err := client.run(ctx, destDir, "git", "init"); err != nil {
			return "", err
		}
		if err := client.run(
			ctx,
			destDir,
			"git",
			"remote",
			"add",
			"origin",
			repoURL,
		); err != nil {
			return "", err
		}
	}

	if err := client.run(ctx, destDir, "git", "fetch", "--all", "--tags"); err != nil {
		return "", err
	}
	if err := client.run(ctx, destDir, "git", "checkout", ref); err != nil {
		return "", err
	}

	commit, err := client.output(ctx, destDir, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(commit), nil
}

func (client *execGitClient) run(
	ctx context.Context,
	workingDir string,
	name string,
	args ...string,
) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workingDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (client *execGitClient) output(
	ctx context.Context,
	workingDir string,
	name string,
	args ...string,
) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workingDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(output)))
	}

	return string(output), nil
}
