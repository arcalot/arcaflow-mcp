package workflowtools

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

func TestWorkflowListFilesystem(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "basic.yaml")
	content := []byte("version: v0.1\nsteps: {}\n")
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewWorkflowListTool(loader, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected single content entry, got %d", len(result.Content))
	}

	var payload ListResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Source.Kind != "filesystem" {
		t.Fatalf("expected filesystem source kind")
	}
	if len(payload.Workflows) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(payload.Workflows))
	}
	if payload.Workflows[0].Name != "basic" {
		t.Fatalf("expected workflow name basic")
	}
}

func TestWorkflowListInvalidKind(t *testing.T) {
	loader := workflow.NewLoader()
	tool := NewWorkflowListTool(loader, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "unknown",
			"location": "/tmp",
		},
	})
	if errObj == nil {
		t.Fatalf("expected error for invalid kind")
	}
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestWorkflowListMissingSource(t *testing.T) {
	loader := workflow.NewLoader()
	tool := NewWorkflowListTool(loader, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
	if errObj == nil {
		t.Fatalf("expected error for missing source")
	}
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}
