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

func TestWorkflowInputValidatePayload(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "validate.yaml")
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
	tool := NewWorkflowInputValidateTool(loader, stateManager, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "validate.yaml",
		},
		"input": map[string]interface{}{
			"name": "arcaflow",
		},
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload InputValidateResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if !payload.Validation.Valid {
		t.Fatalf("expected input to be valid")
	}
}

func TestWorkflowInputValidateFromSession(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "validate.yaml")
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
	if err := stateManager.Set(
		ctx,
		state.SessionData{
			SessionID:  "session-1",
			WorkflowID: "validate",
			DraftInput: json.RawMessage(`{"name":"arcaflow"}`),
		},
	); err != nil {
		t.Fatalf("set session data: %v", err)
	}
	tool := NewWorkflowInputValidateTool(loader, stateManager, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "validate.yaml",
		},
		"session_id": "session-1",
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload InputValidateResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.SessionID != "session-1" {
		t.Fatalf("expected session id to be returned")
	}
	if !payload.Validation.Valid {
		t.Fatalf("expected input to be valid")
	}
}

func TestWorkflowInputValidateMissingInput(t *testing.T) {
	loader := workflow.NewLoader()
	stateManager := state.NewManager(0)
	tool := NewWorkflowInputValidateTool(loader, stateManager, slog.Default())
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

func TestResolveValidationInputErrors(t *testing.T) {
	t.Parallel()

	manager := state.NewManager(0)
	_, _, err := resolveValidationInput(context.Background(), manager, "", nil)
	if err == nil {
		t.Fatalf("expected error when input and session missing")
	}
	if err.Error() == "" {
		t.Fatalf("expected error message")
	}

	_, _, err = resolveValidationInput(
		context.Background(),
		manager,
		"session-missing",
		nil,
	)
	if err == nil {
		t.Fatalf("expected error for missing session")
	}
}

func TestWorkflowInputValidateInvalidArguments(t *testing.T) {
	loader := workflow.NewLoader()
	stateManager := state.NewManager(0)
	tool := NewWorkflowInputValidateTool(loader, stateManager, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"bad": make(chan int),
	})
	if errObj == nil {
		t.Fatalf("expected invalid arguments error")
	}
}

func TestWorkflowInputValidateInvalidSourceKind(t *testing.T) {
	loader := workflow.NewLoader()
	stateManager := state.NewManager(0)
	tool := NewWorkflowInputValidateTool(loader, stateManager, slog.Default())
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
