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
	"github.com/arcalot/arcaflow-mcp/server/pkg/state"
)

func TestWorkflowInputBuildMerge(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "input.yaml")
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
        count:
          type:
            type_id: integer
`)
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	stateManager := state.NewManager(0)
	tool := NewWorkflowInputBuildTool(loader, stateManager, slog.Default())

	first, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "input.yaml",
		},
		"session_id": "session-1",
		"input": map[string]interface{}{
			"name": "arcaflow",
		},
		"validate": false,
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var firstPayload InputBuildResult
	if err := json.Unmarshal([]byte(first.Content[0].Text), &firstPayload); err != nil {
		t.Fatalf("unmarshal first result: %v", err)
	}
	if firstPayload.Validation.Performed {
		t.Fatalf("expected validation to be skipped on first build")
	}

	second, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "input.yaml",
		},
		"session_id": "session-1",
		"input": map[string]interface{}{
			"count": 2,
		},
		"merge": true,
		"validate": true,
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var secondPayload InputBuildResult
	if err := json.Unmarshal([]byte(second.Content[0].Text), &secondPayload); err != nil {
		t.Fatalf("unmarshal second result: %v", err)
	}
	if !secondPayload.Validation.Performed || !secondPayload.Validation.Valid {
		t.Fatalf(
			"expected valid input after merge, performed=%t valid=%t issues=%v",
			secondPayload.Validation.Performed,
			secondPayload.Validation.Valid,
			secondPayload.Validation.Issues,
		)
	}
	var draft map[string]interface{}
	if err := json.Unmarshal(secondPayload.Draft, &draft); err != nil {
		t.Fatalf("parse draft: %v", err)
	}
	if draft["name"] != "arcaflow" {
		t.Fatalf("expected merged name field")
	}
	if draft["count"] != float64(2) {
		t.Fatalf("expected merged count field")
	}
}

func TestWorkflowInputBuildMissingInput(t *testing.T) {
	loader := workflow.NewLoader()
	stateManager := state.NewManager(0)
	tool := NewWorkflowInputBuildTool(loader, stateManager, slog.Default())
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
