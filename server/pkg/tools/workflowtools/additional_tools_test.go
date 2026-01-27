package workflowtools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"github.com/arcalot/arcaflow-mcp/server/pkg/state"
)

func TestWorkflowListInvalidKindAdditional(t *testing.T) {
	loader := workflow.NewLoader()
	tool := NewWorkflowListTool(loader, nil)
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "invalid",
			"location": "/tmp",
		},
	})
	if errObj == nil || errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestWorkflowLoadSelectorRequired(t *testing.T) {
	root := t.TempDir()
	content := []byte(`
version: v0.2.0
input:
  root: InputParams
  objects:
    InputParams:
      id: InputParams
      properties:
        name:
          required: true
          type:
            type_id: string
`)
	if err := os.WriteFile(filepath.Join(root, "one.yaml"), content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "two.yaml"), content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewWorkflowLoadTool(loader, nil)
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

func TestWorkflowSchemaGetContextCancelled(t *testing.T) {
	root := t.TempDir()
	content := []byte(`
version: v0.2.0
input:
  root: InputParams
  objects:
    InputParams:
      id: InputParams
      properties:
        name:
          required: true
          type:
            type_id: string
outputs:
  success: {}
`)
	path := filepath.Join(root, "schema.yaml")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	tool := NewWorkflowSchemaGetTool(loader, parser, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, errObj := tool.Handler(ctx, map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "schema.yaml",
		},
	})
	if errObj == nil {
		t.Fatalf("expected context error")
	}
}

func TestWorkflowInputValidateMissingDraft(t *testing.T) {
	root := t.TempDir()
	content := []byte(`
version: v0.2.0
input:
  root: InputParams
  objects:
    InputParams:
      id: InputParams
      properties:
        name:
          required: true
          type:
            type_id: string
`)
	if err := os.WriteFile(filepath.Join(root, "validate.yaml"), content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	manager := state.NewManager(0)
	tool := NewWorkflowInputValidateTool(loader, manager, nil)
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "validate.yaml",
		},
		"session_id": "missing-session",
	})
	if errObj == nil {
		t.Fatalf("expected missing draft error")
	}
}
