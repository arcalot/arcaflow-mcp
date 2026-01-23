package workflowtools

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"github.com/arcalot/arcaflow-mcp/server/pkg/state"
)

func TestWorkflowInputExportJSON(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "export.yaml")
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
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	stateManager := state.NewManager(0)
	tool := NewWorkflowInputExportTool(loader, parser, stateManager, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "export.yaml",
		},
		"input": map[string]interface{}{
			"name": "arcaflow",
		},
		"format": "json",
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload InputExportResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Format != "json" {
		t.Fatalf("expected json format")
	}
	if !strings.Contains(payload.Payload, `"name":"arcaflow"`) {
		t.Fatalf("expected payload to include input")
	}
	if !payload.Metadata.Validated {
		t.Fatalf("expected validation metadata")
	}
}

func TestWorkflowInputExportFromSessionYAML(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "export.yaml")
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
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	stateManager := state.NewManager(0)
	ctx := auth.WithTenantID(context.Background(), "local")
	if err := stateManager.Set(ctx, state.SessionData{
		SessionID:  "session-1",
		WorkflowID: "export",
		DraftInput: json.RawMessage(`{"name":"arcaflow"}`),
	}); err != nil {
		t.Fatalf("set session: %v", err)
	}

	tool := NewWorkflowInputExportTool(loader, parser, stateManager, slog.Default())
	result, errObj := tool.Handler(ctx, map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "export.yaml",
		},
		"session_id": "session-1",
		"format":     "yaml",
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload InputExportResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Format != "yaml" {
		t.Fatalf("expected yaml format")
	}
	if !strings.Contains(payload.Payload, "name: arcaflow") {
		t.Fatalf("expected yaml payload")
	}
}

func TestWorkflowInputExportMissingInput(t *testing.T) {
	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	stateManager := state.NewManager(0)
	tool := NewWorkflowInputExportTool(loader, parser, stateManager, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": "/tmp",
		},
	})
	if errObj == nil {
		t.Fatalf("expected missing input error")
	}
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}
