package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
)

const autoPerfEnvVar = "ARCAFLOW_AUTO_PERF_PATH"

type autoPerfPluginSchemaStub struct{}

func (autoPerfPluginSchemaStub) InputJSONSchema(
	_ context.Context,
	_ string,
	_ string,
) (json.RawMessage, error) {
	return json.RawMessage(
		`{"type":"object","properties":{"placeholder":{"type":"string"}}}`,
	), nil
}

func TestAutoPerfInputSchemaResolution(t *testing.T) {
	t.Parallel()

	workflowDoc := loadAutoPerfWorkflow(t)
	resolver := workflow.NewInputSchemaResolver(
		workflow.WithPluginSchemaProvider(autoPerfPluginSchemaStub{}),
	)
	raw, err := resolver.ResolveInputJSONSchema(context.Background(), workflowDoc)
	if err != nil {
		t.Fatalf("resolve auto-perf input schema: %v", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("unmarshal resolved schema: %v", err)
	}

	properties, _ := schema["properties"].(map[string]interface{})
	if properties == nil {
		t.Fatalf("expected properties in resolved schema")
	}
	for _, key := range []string{
		"enable_horreum",
		"pcp_params",
		"stressng_tests",
		"sysbench_cpu_tests",
		"fio_tests",
	} {
		if _, ok := properties[key]; !ok {
			t.Fatalf("expected property %q in resolved schema", key)
		}
	}
}

func TestAutoPerfInputValidationAndExport(t *testing.T) {
	t.Setenv("ARCAFLOW_MCP_PLUGIN_SCHEMA_MODE", "stub")

	workflowDoc := loadAutoPerfWorkflow(t)
	parser := workflow.NewParser()
	parsed, err := parser.Parse(context.Background(), workflowDoc)
	if err != nil {
		t.Fatalf("parse auto-perf workflow: %v", err)
	}

	exported, err := workflow.GenerateInputFile(
		context.Background(),
		parsed,
		[]byte(`{
  "enable_horreum": false,
  "horreum_params": {
    "horreum_url": "https://example.com",
    "horreum_api_key": "placeholder",
    "test_name": "example",
    "test_owner": "example",
    "test_access_rights": "PUBLIC"
  },
  "pcp_params": {},
  "stressng_tests": [],
  "sysbench_cpu_tests": [],
  "sysbench_memory_tests": [],
  "fio_tests": [],
  "coremark_pro_tests": [],
  "autobench_tests": []
}`),
		workflow.ExportFormatJSON,
	)
	if err != nil {
		t.Fatalf("export auto-perf input: %v", err)
	}
	if !json.Valid(exported.Payload) {
		t.Fatalf("expected exported auto-perf input to be valid json")
	}
}

func loadAutoPerfWorkflow(t *testing.T) workflow.Workflow {
	t.Helper()

	workflowPath := os.Getenv(autoPerfEnvVar)
	if workflowPath == "" {
		t.Skipf("%s not set; skipping auto-perf integration tests", autoPerfEnvVar)
	}
	stat, err := os.Stat(workflowPath)
	if err != nil {
		t.Skipf("auto-perf workflow path unavailable: %v", err)
	}
	if stat.IsDir() {
		workflowPath = filepath.Join(workflowPath, "workflow.yaml")
	}
	if _, err := os.Stat(workflowPath); err != nil {
		t.Skipf("auto-perf workflow file unavailable: %v", err)
	}

	loader := workflow.NewLoader()
	index, err := loader.LoadFromFilesystem(context.Background(), workflowPath)
	if err != nil {
		t.Fatalf("load auto-perf workflow: %v", err)
	}
	if len(index.Workflows) == 0 {
		t.Fatalf("expected auto-perf workflow to load")
	}
	return index.Workflows[0]
}

