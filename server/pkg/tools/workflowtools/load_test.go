package workflowtools

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

func TestWorkflowLoadByPath(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "basic.yaml")
	content := []byte("version: v0.1\nsteps: {}\n")
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewWorkflowLoadTool(loader, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "basic.yaml",
		},
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload LoadResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Workflow.Name != "basic" {
		t.Fatalf("expected workflow name basic")
	}
	if payload.Workflow.Content == "" {
		t.Fatalf("expected workflow content")
	}
}

func TestWorkflowLoadFromURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("version: v0.1\nsteps: {}\n"))
	}))
	t.Cleanup(server.Close)

	loader := workflow.NewLoader(workflow.WithHTTPClient(server.Client()))
	tool := NewWorkflowLoadTool(loader, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "url",
			"location": server.URL,
		},
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload LoadResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Workflow.Content == "" {
		t.Fatalf("expected workflow content")
	}
}

func TestWorkflowLoadFromGit(t *testing.T) {
	fake := &fakeGitClient{
		commit: "abc123",
		files: map[string][]byte{
			filepath.Join("workflows", "sample.yaml"): []byte("version: v0.1\nsteps: {}\n"),
		},
	}
	loader := workflow.NewLoader(
		workflow.WithGitClient(fake),
		workflow.WithGitCacheDir(t.TempDir()),
	)
	tool := NewWorkflowLoadTool(loader, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "git",
			"location": "https://example.com/repo.git",
			"ref":      "main",
			"subdir":   "workflows",
		},
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload LoadResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Workflow.Content == "" {
		t.Fatalf("expected workflow content")
	}
}

func TestWorkflowLoadRequiresSelectorForMultiple(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "first.yaml")
	secondPath := filepath.Join(root, "second.yaml")
	content := []byte("version: v0.1\nsteps: {}\n")
	if err := os.WriteFile(firstPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(secondPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewWorkflowLoadTool(loader, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
	})
	if errObj != nil {
		t.Fatalf("expected discovery response, got %v", errObj)
	}

	var payload DiscoveryResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal discovery: %v", err)
	}
	if !payload.Selection.Required {
		t.Fatalf("expected selection to be required")
	}
}

type fakeGitClient struct {
	files  map[string][]byte
	commit string
}

func (client *fakeGitClient) Sync(
	ctx context.Context,
	repoURL string,
	destDir string,
	ref string,
	progress workflow.ProgressReporter,
) (string, error) {
	_ = ctx
	_ = repoURL
	_ = ref
	_ = progress
	for path, content := range client.files {
		fullPath := filepath.Join(destDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(fullPath, content, 0o644); err != nil {
			return "", err
		}
	}
	return client.commit, nil
}

func TestWorkflowLoadUnknownID(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "basic.yaml")
	content := []byte("version: v0.1\nsteps: {}\n")
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewWorkflowLoadTool(loader, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"id": "missing",
		},
	})
	if errObj == nil {
		t.Fatalf("expected error for missing id")
	}
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestWorkflowLoadInvalidArguments(t *testing.T) {
	loader := workflow.NewLoader()
	tool := NewWorkflowLoadTool(loader, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"bad": make(chan int),
	})
	if errObj == nil {
		t.Fatalf("expected invalid arguments error")
	}
}
