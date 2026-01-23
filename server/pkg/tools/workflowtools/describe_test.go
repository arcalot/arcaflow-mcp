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

func TestWorkflowDescribeByPath(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "describe.yaml")
	content := []byte(
		"version: v0.2.0\n" +
			"description: Demo workflow\n" +
			"steps:\n" +
			"  step-a:\n" +
			"    plugin_schema_ref: plugin-schema.yaml\n",
	)
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewWorkflowDescribeTool(loader, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "describe.yaml",
		},
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload DescribeResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Description != "Demo workflow" {
		t.Fatalf("expected description")
	}
	if payload.StepCount != 1 {
		t.Fatalf("expected step count 1")
	}
	if len(payload.Steps) != 1 || payload.Steps[0].ID != "step-a" {
		t.Fatalf("expected step summary for step-a")
	}
}

func TestWorkflowDescribeMissingSource(t *testing.T) {
	loader := workflow.NewLoader()
	tool := NewWorkflowDescribeTool(loader, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
	if errObj == nil {
		t.Fatalf("expected missing source error")
	}
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}
