package workflowtools

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
)

func TestWorkflowDiscoverFilesystem(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	content := []byte("version: v0.1\nsteps: {}\n")
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewWorkflowDiscoverTool(loader, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload DiscoveryResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal discovery: %v", err)
	}
	if payload.Source.Kind != "filesystem" {
		t.Fatalf("expected filesystem source, got %s", payload.Source.Kind)
	}
	if payload.Selection.Required {
		t.Fatalf("expected selection to be optional for single workflow")
	}
	if payload.Selection.SuggestedSelector == nil ||
		payload.Selection.SuggestedSelector.Path != "workflow.yaml" {
		t.Fatalf("expected suggested selector for workflow.yaml")
	}
	if payload.Cache.Status != string(workflow.CacheStatusMiss) {
		t.Fatalf("expected cache miss, got %s", payload.Cache.Status)
	}
}
