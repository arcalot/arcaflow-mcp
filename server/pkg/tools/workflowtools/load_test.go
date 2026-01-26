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
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
	})
	if errObj == nil {
		t.Fatalf("expected selector error")
	}
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
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
