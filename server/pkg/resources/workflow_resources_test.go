package resources

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
)

func TestWorkflowSchemaResourceReadCachesList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	content := []byte(
		"version: v0.1\n" +
		"steps: {}\n" +
			"input:\n" +
			"  root: InputParams\n" +
			"  objects:\n" +
			"    InputParams:\n" +
			"      id: InputParams\n" +
			"      properties:\n" +
			"        name:\n" +
			"          required: true\n" +
			"          type:\n" +
			"            type_id: string\n" +
			"outputs:\n" +
			"  success: {}\n",
	)
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	provider := NewWorkflowResourceProvider(loader, parser, nil)
	uri := "workflow-schema://filesystem?location=" +
		url.QueryEscape(root) +
		"&path=workflow.yaml"
	ctx := auth.WithTenantID(context.Background(), "tenant-a")
	contentEntry, handled, errObj := provider.Read(ctx, uri)
	if errObj != nil {
		t.Fatalf("read resource: %v", errObj)
	}
	if !handled || contentEntry == nil {
		t.Fatalf("expected resource content")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(contentEntry.Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if _, ok := payload["input_json_schema"]; !ok {
		t.Fatalf("expected input_json_schema")
	}

	items, errObj := provider.List(ctx)
	if errObj != nil {
		t.Fatalf("list resources: %v", errObj)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 cached resource, got %d", len(items))
	}
}

func TestWorkflowResourceRead(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	content := []byte("version: v0.1\nsteps: {}\n")
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	provider := NewWorkflowResourceProvider(loader, parser, nil)
	uri := "workflow://filesystem?location=" +
		url.QueryEscape(root) +
		"&path=workflow.yaml"
	ctx := auth.WithTenantID(context.Background(), "tenant-b")
	contentEntry, handled, errObj := provider.Read(ctx, uri)
	if errObj != nil {
		t.Fatalf("read resource: %v", errObj)
	}
	if !handled || contentEntry == nil {
		t.Fatalf("expected resource content")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(contentEntry.Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["content"] == "" {
		t.Fatalf("expected workflow content")
	}
}

func TestPluginSchemaResourceRead(t *testing.T) {
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
	parser := workflow.NewParser()
	provider := NewWorkflowResourceProvider(loader, parser, nil)
	uri := "plugin-schema://filesystem?location=" +
		url.QueryEscape(root) +
		"&path=workflow.yaml"
	ctx := auth.WithTenantID(context.Background(), "tenant-c")
	contentEntry, handled, errObj := provider.Read(ctx, uri)
	if errObj != nil {
		t.Fatalf("read resource: %v", errObj)
	}
	if !handled || contentEntry == nil {
		t.Fatalf("expected resource content")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(contentEntry.Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	schemas, _ := payload["schemas"].([]interface{})
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema")
	}
}

func TestWorkflowExampleResourceReadGeneratesExample(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	content := []byte(
		"version: v0.1\n" +
		"steps: {}\n" +
			"input:\n" +
			"  root: InputParams\n" +
			"  objects:\n" +
			"    InputParams:\n" +
			"      id: InputParams\n" +
			"      properties:\n" +
			"        name:\n" +
			"          required: true\n" +
			"          type:\n" +
			"            type_id: string\n" +
			"outputs:\n" +
			"  success: {}\n",
	)
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	provider := NewWorkflowResourceProvider(loader, parser, nil)
	uri := "workflow-example://filesystem?location=" +
		url.QueryEscape(root) +
		"&path=workflow.yaml"
	ctx := auth.WithTenantID(context.Background(), "tenant-d")
	contentEntry, handled, errObj := provider.Read(ctx, uri)
	if errObj != nil {
		t.Fatalf("read resource: %v", errObj)
	}
	if !handled || contentEntry == nil {
		t.Fatalf("expected resource content")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(contentEntry.Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["example_input"] == nil {
		t.Fatalf("expected example input")
	}
}

func TestWorkflowResourceInvalidURI(t *testing.T) {
	t.Parallel()

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	provider := NewWorkflowResourceProvider(loader, parser, nil)
	uri := "workflow-schema://filesystem?path=workflow.yaml"
	_, handled, errObj := provider.Read(context.Background(), uri)
	if !handled {
		t.Fatalf("expected workflow uri to be handled")
	}
	if errObj == nil {
		t.Fatalf("expected missing location error")
	}
}

func TestWorkflowResourceUnsupportedScheme(t *testing.T) {
	t.Parallel()

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	provider := NewWorkflowResourceProvider(loader, parser, nil)
	_, handled, errObj := provider.Read(context.Background(), "unknown://value")
	if handled {
		t.Fatalf("expected unsupported scheme to be ignored")
	}
	if errObj != nil {
		t.Fatalf("expected no error for unsupported scheme")
	}
}

func TestPluginSchemaResourceMissingStep(t *testing.T) {
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
	if err := os.WriteFile(workflowPath, workflowContent, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(schemaPath, []byte(`{"type":"object"}`), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	provider := NewWorkflowResourceProvider(loader, parser, nil)
	uri := "plugin-schema://filesystem?location=" +
		url.QueryEscape(root) +
		"&path=workflow.yaml&step_id=missing"
	_, handled, errObj := provider.Read(context.Background(), uri)
	if !handled {
		t.Fatalf("expected plugin schema uri to be handled")
	}
	if errObj == nil {
		t.Fatalf("expected missing step error")
	}
}

func TestWorkflowResourceLoadIndexUnknownKind(t *testing.T) {
	t.Parallel()

	loader := workflow.NewLoader()
	_, err := loadIndex(context.Background(), loader, sourceParams{
		Kind:     "unknown",
		Location: "/tmp",
	})
	if err == nil {
		t.Fatalf("expected unknown kind error")
	}
}

func TestWorkflowResourceSelectWorkflowErrors(t *testing.T) {
	t.Parallel()

	workflowItem := workflow.Workflow{ID: "id-1", Path: "path.yaml", Name: "name"}
	_, err := selectWorkflow([]workflow.Workflow{workflowItem}, selectorParams{
		ID: "missing",
	})
	if err == nil {
		t.Fatalf("expected missing id error")
	}
	_, err = selectWorkflow([]workflow.Workflow{workflowItem}, selectorParams{
		Path: "missing.yaml",
	})
	if err == nil {
		t.Fatalf("expected missing path error")
	}
}

func TestWorkflowResourceRenderError(t *testing.T) {
	t.Parallel()

	provider := NewWorkflowResourceProvider(nil, nil, nil)
	_, errObj := provider.renderResource("workflow://x", map[string]interface{}{
		"bad": make(chan int),
	})
	if errObj == nil {
		t.Fatalf("expected render error")
	}
}

func TestWorkflowSchemaResourceMissingInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	if err := os.WriteFile(
		workflowPath,
		[]byte("version: v0.1\noutputs:\n  success: {}\n"),
		0o644,
	); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	provider := NewWorkflowResourceProvider(loader, parser, nil)
	uri := "workflow-schema://filesystem?location=" +
		url.QueryEscape(root) +
		"&path=workflow.yaml"
	_, handled, errObj := provider.Read(context.Background(), uri)
	if !handled {
		t.Fatalf("expected schema resource to be handled")
	}
	if errObj == nil {
		t.Fatalf("expected schema parse error")
	}
}

func TestWorkflowExampleResourceMissingInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	if err := os.WriteFile(
		workflowPath,
		[]byte("version: v0.1\noutputs:\n  success: {}\n"),
		0o644,
	); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	provider := NewWorkflowResourceProvider(loader, parser, nil)
	uri := "workflow-example://filesystem?location=" +
		url.QueryEscape(root) +
		"&path=workflow.yaml"
	_, handled, errObj := provider.Read(context.Background(), uri)
	if !handled {
		t.Fatalf("expected example resource to be handled")
	}
	if errObj == nil {
		t.Fatalf("expected example parse error")
	}
}

func TestParseWorkflowResourceURIValidation(t *testing.T) {
	t.Parallel()

	if _, _, handled, err := parseWorkflowResourceURI(
		"workflow://?location=/tmp",
	); !handled || err == nil {
		t.Fatalf("expected missing kind error")
	}

	if _, _, handled, err := parseWorkflowResourceURI(
		"workflow://filesystem",
	); !handled || err == nil {
		t.Fatalf("expected missing location error")
	}
}

func TestWorkflowSchemaResourceCanceledContext(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	content := []byte(
		"version: v0.1\n" +
			"input:\n" +
			"  root: InputParams\n" +
			"  objects:\n" +
			"    InputParams:\n" +
			"      id: InputParams\n" +
			"      properties:\n" +
			"        name:\n" +
			"          required: true\n" +
			"          type:\n" +
			"            type_id: string\n",
	)
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	provider := NewWorkflowResourceProvider(loader, parser, nil)
	uri := "workflow-schema://filesystem?location=" +
		url.QueryEscape(root) +
		"&path=workflow.yaml"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, handled, errObj := provider.Read(ctx, uri)
	if !handled {
		t.Fatalf("expected schema resource to be handled")
	}
	if errObj == nil {
		t.Fatalf("expected context error")
	}
}

func TestWorkflowExampleResourceCanceledContext(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	content := []byte(
		"version: v0.1\n" +
			"input:\n" +
			"  root: InputParams\n" +
			"  objects:\n" +
			"    InputParams:\n" +
			"      id: InputParams\n" +
			"      properties:\n" +
			"        name:\n" +
			"          required: true\n" +
			"          type:\n" +
			"            type_id: string\n",
	)
	if err := os.WriteFile(workflowPath, content, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	provider := NewWorkflowResourceProvider(loader, parser, nil)
	uri := "workflow-example://filesystem?location=" +
		url.QueryEscape(root) +
		"&path=workflow.yaml"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, handled, errObj := provider.Read(ctx, uri)
	if !handled {
		t.Fatalf("expected example resource to be handled")
	}
	if errObj == nil {
		t.Fatalf("expected context error")
	}
}
