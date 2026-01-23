package workflow

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type stubPluginSchemaProvider struct {
	schema json.RawMessage
}

func (s stubPluginSchemaProvider) InputJSONSchema(
	_ context.Context,
	_ string,
	_ string,
) (json.RawMessage, error) {
	return s.schema, nil
}

func TestResolveInputJSONSchemaNamespaces(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	subworkflowPath := filepath.Join(root, "sub.yaml")
	if err := os.WriteFile(
		subworkflowPath,
		[]byte(
			"version: v0.2.0\n"+
				"input:\n"+
				"  root: SubInput\n"+
				"  objects:\n"+
				"    SubInput:\n"+
				"      id: SubInput\n"+
				"      properties:\n"+
				"        value:\n"+
				"          required: true\n"+
				"          type:\n"+
				"            type_id: string\n"+
				"outputs:\n"+
				"  success: {}\n",
		),
		0o644,
	); err != nil {
		t.Fatalf("write subworkflow: %v", err)
	}
	mainWorkflow := Workflow{
		ID:        "root",
		LocalPath: filepath.Join(root, "workflow.yaml"),
		Content: []byte(
			"version: v0.2.0\n" +
				"input:\n" +
				"  root: RootInput\n" +
				"  objects:\n" +
				"    RootInput:\n" +
				"      id: RootInput\n" +
				"      properties:\n" +
				"        plugin_params:\n" +
				"          required: true\n" +
				"          type:\n" +
				"            type_id: ref\n" +
				"            id: PluginInput\n" +
				"            namespace: $.steps.example.starting.inputs.input\n" +
				"        sub_params:\n" +
				"          required: true\n" +
				"          type:\n" +
				"            type_id: ref\n" +
				"            id: SubInput\n" +
				"            namespace: $.steps.subflow.execute.inputs.items.item\n" +
				"steps:\n" +
				"  example:\n" +
				"    plugin:\n" +
				"      deployment_type: image\n" +
				"      src: quay.io/example/plugin:1.0.0\n" +
				"    step: hello\n" +
				"    input: !expr $.input.plugin_params\n" +
				"  subflow:\n" +
				"    kind: foreach\n" +
				"    items: !expr $.input.sub_params\n" +
				"    workflow: sub.yaml\n" +
				"outputs:\n" +
				"  success: {}\n",
		),
	}
	pluginSchema := json.RawMessage(
		`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`,
	)
	resolver := NewInputSchemaResolver(
		WithPluginSchemaProvider(stubPluginSchemaProvider{schema: pluginSchema}),
	)
	raw, err := resolver.ResolveInputJSONSchema(context.Background(), mainWorkflow)
	if err != nil {
		t.Fatalf("resolve input schema: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	properties, _ := decoded["properties"].(map[string]interface{})
	if properties == nil {
		t.Fatalf("expected properties in resolved schema")
	}
	if _, ok := properties["plugin_params"]; !ok {
		t.Fatalf("expected plugin_params in schema")
	}
	if _, ok := properties["sub_params"]; !ok {
		t.Fatalf("expected sub_params in schema")
	}
}
