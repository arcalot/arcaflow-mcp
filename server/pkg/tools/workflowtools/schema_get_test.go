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

func TestWorkflowSchemaGetByPath(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "schema.yaml")
	content := []byte(
		"version: v0.1\n" +
			"input:\n" +
			"  root: InputParams\n" +
			"  objects:\n" +
			"    InputParams:\n" +
			"      id: InputParams\n" +
			"      properties:\n" +
			"        foo:\n" +
			"          required: true\n" +
			"          type:\n" +
			"            type_id: string\n" +
			"outputSchema:\n" +
			"  type: object\n" +
			"  properties:\n" +
			"    result:\n" +
			"      type: string\n",
	)
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	tool := NewWorkflowSchemaGetTool(loader, parser, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "schema.yaml",
		},
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload SchemaGetResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(payload.InputJSONSchema) == 0 || len(payload.OutputJSONSchema) == 0 {
		t.Fatalf("expected input and output schemas")
	}
	if payload.Workflow.Name != "schema" {
		t.Fatalf("expected workflow name schema")
	}
}

func TestWorkflowSchemaGetMissingSource(t *testing.T) {
	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	tool := NewWorkflowSchemaGetTool(loader, parser, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
	if errObj == nil {
		t.Fatalf("expected missing source error")
	}
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}
