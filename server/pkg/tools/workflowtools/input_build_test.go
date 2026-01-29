package workflowtools

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
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
		"merge":    true,
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
		return
	}
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestInputBuildHelpers(t *testing.T) {
	t.Parallel()

	if shouldMerge(nil) != true {
		t.Fatalf("expected default merge to be true")
	}
	merge := false
	if shouldMerge(&merge) {
		t.Fatalf("expected merge override to be false")
	}
	if shouldValidate(nil) != true {
		t.Fatalf("expected default validate to be true")
	}
	validate := false
	if shouldValidate(&validate) {
		t.Fatalf("expected validate override to be false")
	}
	sessionID, err := newSessionID()
	if err != nil {
		t.Fatalf("expected session id, got %v", err)
	}
	if len(sessionID) != 32 {
		t.Fatalf("expected 32 char session id")
	}
}

func TestMergeObjectsNested(t *testing.T) {
	t.Parallel()

	base := map[string]interface{}{
		"root": map[string]interface{}{
			"count": 1,
		},
		"value": "base",
	}
	overlay := map[string]interface{}{
		"root": map[string]interface{}{
			"count": 2,
			"extra": "ok",
		},
	}
	merged := mergeObjects(base, overlay)
	root, _ := merged["root"].(map[string]interface{})
	if root["count"] != 2.0 && root["count"] != 2 {
		t.Fatalf("expected nested override")
	}
	if root["extra"] != "ok" {
		t.Fatalf("expected nested merge")
	}
	if merged["value"] != "base" {
		t.Fatalf("expected base value")
	}
}

func TestWorkflowInputBuildInvalidDraft(t *testing.T) {
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
`)
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	stateManager := state.NewManager(0)
	ctx := auth.WithTenantID(context.Background(), "local")
	if err := stateManager.Set(ctx, state.SessionData{
		SessionID:  "session-bad",
		WorkflowID: "input",
		DraftInput: []byte("{invalid"),
	}); err != nil {
		t.Fatalf("set session: %v", err)
	}
	tool := NewWorkflowInputBuildTool(loader, stateManager, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "input.yaml",
		},
		"session_id": "session-bad",
		"input": map[string]interface{}{
			"name": "arcaflow",
		},
	})
	if errObj == nil {
		t.Fatalf("expected invalid draft error")
	}
}

func TestWorkflowInputBuildInvalidSourceKind(t *testing.T) {
	loader := workflow.NewLoader()
	stateManager := state.NewManager(0)
	tool := NewWorkflowInputBuildTool(loader, stateManager, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "invalid",
			"location": "/tmp",
		},
		"input": map[string]interface{}{
			"name": "arcaflow",
		},
	})
	if errObj == nil {
		t.Fatalf("expected invalid source error")
	}
}

func TestWorkflowInputBuildSelectorError(t *testing.T) {
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
`)
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	stateManager := state.NewManager(0)
	tool := NewWorkflowInputBuildTool(loader, stateManager, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "missing.yaml",
		},
		"input": map[string]interface{}{
			"name": "arcaflow",
		},
	})
	if errObj == nil {
		t.Fatalf("expected selector error")
	}
}
