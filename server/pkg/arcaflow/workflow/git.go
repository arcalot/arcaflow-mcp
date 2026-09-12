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
	Sync(
		ctx context.Context,
		repoURL string,
		destDir string,
		ref string,
		progress ProgressReporter,
	) (string, error)
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
	progress ProgressReporter,
) (string, error) {
	if repoURL == "" {
		return "", fmt.Errorf("git repo url is required")
	}
	if destDir == "" {
		return "", fmt.Errorf("git destination directory is required")
	}
	defaultRef := ref
	if defaultRef == "" {
		defaultRef = "origin/HEAD"
	}
	fetchRef := ref
	if fetchRef == "" {
		fetchRef = "HEAD"
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

	fetchErr := client.withProgress(
		ctx,
		progress,
		ProgressStageFetch,
		func() error {
			return client.run(
				ctx,
				destDir,
				"git",
				"fetch",
				"--depth=1",
				"origin",
				fetchRef,
			)
		},
	)
	if fetchErr != nil {
		return "", fetchErr
	}

	_, err := client.withProgressCheckout(
		ctx,
		destDir,
		defaultRef,
		ref == "",
		progress,
	)
	if err != nil {
		return "", err
	}

	commit, err := client.output(ctx, destDir, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(commit), nil
}

func (client *execGitClient) withProgress(
	ctx context.Context,
	progress ProgressReporter,
	stage ProgressStage,
	action func() error,
) error {
	if progress != nil {
		progress.Start(stage)
	}
	err := action()
	if progress != nil {
		progress.Finish(stage, err)
	}
	return err
}

func (client *execGitClient) withProgressCheckout(
	ctx context.Context,
	destDir string,
	ref string,
	allowFallback bool,
	progress ProgressReporter,
) (string, error) {
	if progress != nil {
		progress.Start(ProgressStageCheckout)
	}
	selected, err := client.checkoutRef(ctx, destDir, ref, allowFallback)
	if progress != nil {
		progress.Finish(ProgressStageCheckout, err)
	}
	return selected, err
}

func (client *execGitClient) checkoutRef(
	ctx context.Context,
	workingDir string,
	ref string,
	allowFallback bool,
) (string, error) {
	if err := client.run(ctx, workingDir, "git", "checkout", ref); err == nil {
		return ref, nil
	} else if !allowFallback {
		return "", err
	}

	fallbacks := []string{"FETCH_HEAD", "origin/main", "origin/master"}
	var lastErr error
	for _, fallback := range fallbacks {
		if err := client.run(ctx, workingDir, "git", "checkout", fallback); err == nil {
			return fallback, nil
		} else {
			lastErr = err
		}
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("git checkout failed for %s and fallbacks", ref)
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
