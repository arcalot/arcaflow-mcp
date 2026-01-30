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

func TestWorkflowInputTemplate(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "example.yaml")
	content := []byte(`
version: v0.2.0
steps: {}
input:
  root: RootObject
  objects:
    RootObject:
      id: RootObject
      properties:
        nickname:
          required: true
          type:
            type_id: string
outputs:
  success:
    status: ok
`)
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	tool := NewWorkflowInputTemplateTool(loader, parser, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "example.yaml",
		},
		"goal": "max performance",
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload InputTemplateResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Goal != "max performance" {
		t.Fatalf("expected goal to be preserved")
	}
	if len(payload.InputJSONSchema) == 0 {
		t.Fatalf("expected input schema")
	}
	if len(payload.ExampleInput) == 0 {
		t.Fatalf("expected example input")
	}
	if payload.InputKey != "input" {
		t.Fatalf("expected input key to be input")
	}
}

func TestWorkflowInputTemplateMissingSource(t *testing.T) {
	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	tool := NewWorkflowInputTemplateTool(loader, parser, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
	if errObj == nil {
		t.Fatalf("expected missing source error")
		return
	}
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestWorkflowInputTemplateInvalidArguments(t *testing.T) {
	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	tool := NewWorkflowInputTemplateTool(loader, parser, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"bad": make(chan int),
	})
	if errObj == nil || errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestWorkflowInputTemplateSelectorRequired(t *testing.T) {
	root := t.TempDir()
	content := []byte(`
version: v0.2.0
steps: {}
input:
  root: RootObject
  objects:
    RootObject:
      id: RootObject
      properties:
        nickname:
          required: true
          type:
            type_id: string
outputs:
  success:
    status: ok
`)
	if err := os.WriteFile(filepath.Join(root, "one.yaml"), content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "two.yaml"), content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	tool := NewWorkflowInputTemplateTool(loader, parser, slog.Default())
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

func TestWorkflowInputTemplateCanceledContext(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "example.yaml")
	content := []byte(`
version: v0.2.0
steps: {}
input:
  root: RootObject
  objects:
    RootObject:
      id: RootObject
      properties:
        nickname:
          required: true
          type:
            type_id: string
outputs:
  success:
    status: ok
`)
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	tool := NewWorkflowInputTemplateTool(loader, parser, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, errObj := tool.Handler(ctx, map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "example.yaml",
		},
	})
	if errObj == nil {
		t.Fatalf("expected context error")
	}
}

func TestWorkflowInputTemplateInvalidSourceKind(t *testing.T) {
	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	tool := NewWorkflowInputTemplateTool(loader, parser, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "invalid",
			"location": "/tmp",
		},
	})
	if errObj == nil {
		t.Fatalf("expected invalid source error")
	}
}
