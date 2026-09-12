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
		return
	}
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestExtractStepSummaries(t *testing.T) {
	t.Parallel()

	root := map[string]interface{}{
		"steps": map[string]interface{}{
			"step-a": map[string]interface{}{
				"plugin_schema":     "schema.yaml",
				"plugin_schema_ref": "schema-ref.yaml",
				"plugin": map[string]interface{}{
					"name":       "plugin-a",
					"image":      "image-a",
					"schema":     "schema-inline",
					"schema_ref": "schema-ref-inline",
				},
			},
			"step-b": "invalid",
		},
	}
	summaries := extractStepSummaries(root)
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries")
	}
	if summaries[0].ID == "" {
		t.Fatalf("expected summary id")
	}
	if stringValue(nil, "version") != "" {
		t.Fatalf("expected empty value for nil root")
	}
}

func TestDescriptionValueFallbacks(t *testing.T) {
	t.Parallel()

	root := map[string]interface{}{
		"summary": "Summary text",
	}
	if descriptionValue(root) != "Summary text" {
		t.Fatalf("expected summary fallback")
	}
}

func TestDescriptionValueTitleFallback(t *testing.T) {
	t.Parallel()

	root := map[string]interface{}{
		"title": "Title text",
	}
	if descriptionValue(root) != "Title text" {
		t.Fatalf("expected title fallback")
	}
}

func TestStringFieldNonString(t *testing.T) {
	t.Parallel()

	root := map[string]interface{}{
		"version": 123,
	}
	if stringField(root, "version") != "" {
		t.Fatalf("expected empty string for non-string value")
	}
}

func TestWorkflowDescribeInvalidArguments(t *testing.T) {
	loader := workflow.NewLoader()
	tool := NewWorkflowDescribeTool(loader, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"bad": make(chan int),
	})
	if errObj == nil {
		t.Fatalf("expected invalid arguments error")
	}
}

func TestWorkflowDescribeCanceledContext(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "describe.yaml")
	content := []byte(
		"version: v0.2.0\n" +
			"description: Demo workflow\n" +
			"steps: {}\n",
	)
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewWorkflowDescribeTool(loader, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, errObj := tool.Handler(ctx, map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "describe.yaml",
		},
	})
	if errObj == nil {
		t.Fatalf("expected context error")
	}
}

func TestWorkflowDescribeInvalidWorkflowContent(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "describe.yaml")
	if err := os.WriteFile(workflowPath, []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewWorkflowDescribeTool(loader, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "describe.yaml",
		},
	})
	if errObj == nil {
		t.Fatalf("expected parse error")
	}
}
