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

func TestPluginSchemaGetByPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	schemaPath := filepath.Join(root, "plugin-schema.json")
	workflowContent := []byte(
		"version: v0.1\n" +
			"steps:\n" +
			"  sample:\n" +
			"    plugin_schema_ref: plugin-schema.json\n",
	)
	schemaContent := []byte(
		`{"type":"object","properties":{"name":{"type":"string"}}}`,
	)
	if err := os.WriteFile(workflowPath, workflowContent, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(schemaPath, schemaContent, 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewPluginSchemaGetTool(loader, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "workflow.yaml",
		},
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload PluginSchemaGetResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(payload.Schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(payload.Schemas))
	}
	if payload.Schemas[0].StepID != "sample" {
		t.Fatalf("expected step sample")
	}
	if !json.Valid(payload.Schemas[0].Schema) {
		t.Fatalf("expected valid json schema")
	}
}

func TestPluginSchemaGetMissingStep(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	workflowContent := []byte(
		"version: v0.1\n" +
			"steps:\n" +
			"  sample:\n" +
			"    plugin_schema_ref: plugin-schema.json\n",
	)
	if err := os.WriteFile(workflowPath, workflowContent, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewPluginSchemaGetTool(loader, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "workflow.yaml",
		},
		"step_id": "missing",
	})
	if errObj == nil {
		t.Fatalf("expected missing step error")
		return
	}
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestPluginSchemaGetMissingSchemaFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	workflowContent := []byte(
		"version: v0.1\n" +
			"steps:\n" +
			"  sample:\n" +
			"    plugin_schema_ref: plugin-schema.json\n",
	)
	if err := os.WriteFile(workflowPath, workflowContent, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewPluginSchemaGetTool(loader, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "workflow.yaml",
		},
	})
	if errObj == nil {
		t.Fatalf("expected missing schema error")
	}
}

func TestPluginSchemaGetContextCancelled(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	workflowContent := []byte(
		"version: v0.1\n" +
			"steps:\n" +
			"  sample:\n" +
			"    plugin_schema_ref: plugin-schema.json\n",
	)
	if err := os.WriteFile(workflowPath, workflowContent, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewPluginSchemaGetTool(loader, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, errObj := tool.Handler(ctx, map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "workflow.yaml",
		},
	})
	if errObj == nil {
		t.Fatalf("expected context error")
	}
}

func TestPluginSchemaGetNoSchemasHint(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	workflowContent := []byte(
		"version: v0.1\n" +
			"steps:\n" +
			"  sample:\n" +
			"    plugin: {}\n",
	)
	if err := os.WriteFile(workflowPath, workflowContent, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	tool := NewPluginSchemaGetTool(loader, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": root,
		},
		"selector": map[string]interface{}{
			"path": "workflow.yaml",
		},
		"step_id": "sample",
	})
	if errObj == nil {
		t.Fatalf("expected no schema error")
	}
	if errObj.Data == nil {
		t.Fatalf("expected hint details")
	}
	if details, ok := errObj.Data.(map[string]string); ok {
		if details["hint"] == "" {
			t.Fatalf("expected hint when no schemas present")
		}
	} else if details, ok := errObj.Data.(map[string]interface{}); ok {
		if details["hint"] == "" {
			t.Fatalf("expected hint when no schemas present")
		}
	} else {
		t.Fatalf("unexpected details type")
	}
}
